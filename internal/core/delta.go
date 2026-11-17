package core

import (
	"archive/zip"
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bodgit/sevenzip"
	"github.com/nwaples/rardecode/v2"
)

// ---------------------------------------------------------------------------
// An update that takes only what changed (the owner, 17 Nov 2026: a 100 GB
// game shouldn't need another 100 GB to update).
//
// StageDelta reads an archive of the game's new version and compares each
// of its files with the one in the game's folder, by size and the CRC-32
// the archive keeps for it (computed for the game's file). Only files that
// differ, or are new, are written, into a staging folder beside the game.
// A .zip or .7z read through a RemoteFile is fetched only where those files
// are (and its list at the end); a .rar is read through as it streams, and
// what's the same is dropped as it goes. The archive itself is never saved.
//
// ApplyDelta then moves files on the same drive: each old file the update
// replaces, or that the new version no longer has, goes into
// <folder>.lootnest-old, and each new one comes in from the staging folder.
// Moves on one drive take no room and no time. Every move is undone by
// RevertDelta, which works from the record alone, so a crash at any point is
// put right at the next start, and Roll back is the same call.
// ---------------------------------------------------------------------------

// DeltaSource is an archive to read: a .zip or .7z through At (and Size),
// or a .rar whose parts FS opens by name, Name being its first part.
type DeltaSource struct {
	Name string
	At   io.ReaderAt
	Size int64
	FS   fs.FS
}

// DeltaFile is a file the update writes.
type DeltaFile struct {
	Rel  string `json:"rel"`
	Size int64  `json:"size"`
	New  bool   `json:"new,omitempty"`
}

// DeltaPlan is what the archive holds against the game.
type DeltaPlan struct {
	Format    string      `json:"format"`
	Prefix    string      `json:"prefix,omitempty"` // the archive's own top folders, skipped
	NewExe    string      `json:"newExe,omitempty"` // the game's exe in the new version, when the archive has it
	Full      bool        `json:"full"`             // the whole game: files it lacks are old ones
	Changed   []DeltaFile `json:"changed"`
	Same      int         `json:"same"`
	SameBytes int64       `json:"sameBytes"`
	Bytes     int64       `json:"bytes"` // of the changed and new files
	Files     int         `json:"files"` // in the new version
}

// DeltaProgress says how far StageDelta is.
type DeltaProgress struct {
	Step  string `json:"step"` // "compare", "fetch"
	Done  int    `json:"done"`
	Of    int    `json:"of"`
	Bytes int64  `json:"bytes"` // written so far
}

// DeltaOptions steer StageDelta.
type DeltaOptions struct {
	ExeRel   string                 // the game's exe, relative to its folder
	Room     func(need int64) error // refuses when there isn't room for need more bytes
	Progress func(DeltaProgress)
}

const deltaFiles = "files"

// DeltaStaged is where a staged file is.
func DeltaStaged(stage, rel string) string {
	return filepath.Join(stage, deltaFiles, filepath.FromSlash(rel))
}

type deltaEntry struct {
	name   string // in the archive
	rel    string // in the game's folder
	size   int64
	crc    uint32
	hasCRC bool
	order  int64
	open   func() (io.ReadCloser, error)
}

// StageDelta compares the archive with the game in gameDir and writes what
// differs into stage. A file already there from a try cut short, with the
// right size, isn't written again.
func StageDelta(ctx context.Context, src DeltaSource, gameDir, stage string, o DeltaOptions) (*DeltaPlan, error) {
	if err := os.MkdirAll(filepath.Join(stage, deltaFiles), 0o755); err != nil {
		return nil, err
	}
	var plan *DeltaPlan
	var err error
	switch ext := strings.ToLower(filepath.Ext(src.Name)); ext {
	case ".zip", ".7z":
		plan, err = stageListed(ctx, src, ext[1:], gameDir, stage, o)
	case ".rar":
		plan, err = stageStreamed(ctx, src, gameDir, stage, o)
	default:
		return nil, errors.New("only a .zip, .7z or .rar can be compared with the game")
	}
	if err != nil {
		return nil, err
	}
	b, _ := json.MarshalIndent(plan, "", "  ")
	return plan, writeFileAtomic(filepath.Join(stage, "plan.json"), b)
}

// LoadDeltaPlan reads a staged plan.
func LoadDeltaPlan(stage string) (*DeltaPlan, error) {
	b, err := os.ReadFile(filepath.Join(stage, "plan.json"))
	if err != nil {
		return nil, err
	}
	var p DeltaPlan
	return &p, json.Unmarshal(b, &p)
}

// listEntries reads a .zip's or .7z's list of files.
func listEntries(src DeltaSource, format string) ([]deltaEntry, error) {
	var out []deltaEntry
	switch format {
	case "zip":
		zr, err := zip.NewReader(src.At, src.Size)
		if err != nil {
			return nil, errors.New("that .zip can't be read; it may be damaged or incomplete")
		}
		for _, f := range zr.File {
			if f.FileInfo().IsDir() || !f.Mode().IsRegular() {
				continue
			}
			f := f
			off, _ := f.DataOffset()
			out = append(out, deltaEntry{name: f.Name, size: int64(f.UncompressedSize64), crc: f.CRC32, hasCRC: true, order: off, open: f.Open})
		}
	case "7z":
		r, err := sevenzip.NewReader(src.At, src.Size)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "password") {
				return nil, errors.New("that .7z has a password; Lootnest can't open it")
			}
			return nil, errors.New("that .7z can't be read; it may be damaged or incomplete")
		}
		for i, f := range r.File {
			if f.FileInfo().IsDir() {
				continue
			}
			f := f
			out = append(out, deltaEntry{name: f.Name, size: int64(f.UncompressedSize), crc: f.CRC32, hasCRC: f.CRC32 != 0 || f.UncompressedSize == 0, order: int64(i), open: f.Open})
		}
	}
	return out, nil
}

// rootFromNames finds where the game's folder is inside an archive: the
// top folders to skip (prefix) and the exe's path under them. ok is false
// when the archive doesn't hold the exe.
func rootFromNames(names []string, exeRel string) (prefix, newExe string, ok bool) {
	lr := strings.ToLower(exeRel)
	base := path.Base(lr)
	var cands []string
	for _, n := range names {
		if strings.ToLower(path.Base(n)) == base {
			cands = append(cands, n)
		}
	}
	if len(cands) == 0 {
		return "", "", false
	}
	sort.Slice(cands, func(i, j int) bool {
		if a, b := strings.Count(cands[i], "/"), strings.Count(cands[j], "/"); a != b {
			return a < b
		}
		return cands[i] < cands[j]
	})
	// Every file must be under the folders skipped: an exe deeper in the
	// archive than in the game is the game's folder moved, not wrapped.
	allUnder := func(p string) bool {
		for _, n := range names {
			if !strings.HasPrefix(n, p) && !readmeName(n) {
				return false
			}
		}
		return true
	}
	for _, c := range cands {
		lc := strings.ToLower(c)
		if lc == lr {
			return "", c, true
		}
		if strings.HasSuffix(lc, "/"+lr) {
			if p := c[:len(c)-len(exeRel)]; allUnder(p) {
				return p, c[len(p):], true
			}
		}
	}
	// The exe moved: the game's folder is under the archive's wrapping
	// folders, those every file is in.
	p := ""
	for {
		top := ""
		for _, n := range names {
			rest, _ := strings.CutPrefix(n, p)
			i := strings.Index(rest, "/")
			if i <= 0 || (top != "" && !strings.EqualFold(top, rest[:i])) {
				top = ""
				break
			}
			top = rest[:i]
		}
		if top == "" {
			break
		}
		p += top + "/"
	}
	if c := cands[0]; strings.HasPrefix(c, p) {
		return p, c[len(p):], true
	}
	return "", cands[0], true
}

// fileCRC is a file's CRC-32.
func fileCRC(p string) (uint32, error) {
	f, err := os.Open(p)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	h := crc32.NewIEEE()
	if _, err := io.Copy(h, bufio.NewReaderSize(f, 1<<20)); err != nil {
		return 0, err
	}
	return h.Sum32(), nil
}

// stageListed handles a .zip or .7z, whose list comes first.
func stageListed(ctx context.Context, src DeltaSource, format, gameDir, stage string, o DeltaOptions) (*DeltaPlan, error) {
	all, err := listEntries(src, format)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(all))
	for i := range all {
		rel, err := updateRel(all[i].name)
		if err != nil {
			return nil, err
		}
		all[i].name = rel
		names = append(names, rel)
	}
	plan := &DeltaPlan{Format: format}
	var ok bool
	if plan.Prefix, plan.NewExe, ok = rootFromNames(names, o.ExeRel); !ok {
		files := make([]UpdateFile, len(names))
		for i, n := range names {
			files[i] = UpdateFile{Rel: n}
		}
		plan.Prefix = bestPrefix(gameDir, files)
	}
	var build []deltaEntry
	for _, e := range all {
		if rel, in := strings.CutPrefix(e.name, plan.Prefix); in && rel != "" && !(plan.Prefix == "" && plan.NewExe == "" && readmeName(rel)) {
			e.rel = rel
			build = append(build, e)
		}
	}
	plan.Files = len(build)

	// Which differ: by size, then by CRC-32.
	var todo []deltaEntry
	unsure := map[string]bool{} // no CRC kept: compared once written
	for i, e := range build {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if o.Progress != nil {
			o.Progress(DeltaProgress{Step: "compare", Done: i, Of: len(build)})
		}
		local := filepath.Join(gameDir, filepath.FromSlash(e.rel))
		st, err := os.Stat(local)
		switch {
		case err != nil || !st.Mode().IsRegular():
			todo = append(todo, e)
		case st.Size() != e.size:
			todo = append(todo, e)
		case !e.hasCRC:
			todo = append(todo, e)
			unsure[e.rel] = true
		default:
			c, err := fileCRC(local)
			if err != nil {
				return nil, fmt.Errorf("couldn't read %s: %w", e.rel, err)
			}
			if c == e.crc {
				plan.Same++
				plan.SameBytes += e.size
			} else {
				todo = append(todo, e)
			}
		}
	}
	var need int64
	for _, e := range todo {
		if st, err := os.Stat(DeltaStaged(stage, e.rel)); err != nil || st.Size() != e.size {
			need += e.size
		}
	}
	if o.Room != nil {
		if err := o.Room(need); err != nil {
			return nil, err
		}
	}
	// In the archive's order, so a .7z's solid blocks are read once.
	sort.SliceStable(todo, func(i, j int) bool { return todo[i].order < todo[j].order })
	var written int64
	for i, e := range todo {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if o.Progress != nil {
			o.Progress(DeltaProgress{Step: "fetch", Done: i, Of: len(todo), Bytes: written})
		}
		dst := DeltaStaged(stage, e.rel)
		if st, err := os.Stat(dst); err != nil || st.Size() != e.size {
			rc, err := e.open()
			if err != nil {
				return nil, fmt.Errorf("%s: %w", e.rel, err)
			}
			c, err := writeStaged(dst, rc, e.size)
			rc.Close()
			if err != nil {
				return nil, fmt.Errorf("%s: %w", e.rel, err)
			}
			if e.hasCRC && c != e.crc {
				_ = os.Remove(dst)
				return nil, fmt.Errorf("%s doesn't match the archive's checksum", e.rel)
			}
		}
		written += e.size
		if unsure[e.rel] && sameAsLocal(dst, filepath.Join(gameDir, filepath.FromSlash(e.rel))) {
			_ = os.Remove(dst)
			plan.Same++
			plan.SameBytes += e.size
			continue
		}
		plan.add(gameDir, e.rel, e.size)
	}
	if err := writeNames(stage, build); err != nil {
		return nil, err
	}
	plan.Full = deltaFull(plan, gameDir, build)
	return plan, nil
}

func (p *DeltaPlan) add(gameDir, rel string, size int64) {
	_, err := os.Lstat(filepath.Join(gameDir, filepath.FromSlash(rel)))
	p.Changed = append(p.Changed, DeltaFile{Rel: rel, Size: size, New: err != nil})
	p.Bytes += size
}

// sameAsLocal says whether two files have the same size and CRC-32.
func sameAsLocal(a, b string) bool {
	sa, err1 := os.Stat(a)
	sb, err2 := os.Stat(b)
	if err1 != nil || err2 != nil || sa.Size() != sb.Size() {
		return false
	}
	ca, err1 := fileCRC(a)
	cb, err2 := fileCRC(b)
	return err1 == nil && err2 == nil && ca == cb
}

// writeStaged writes r into dst through a temp file and returns its CRC-32.
func writeStaged(dst string, r io.Reader, size int64) (uint32, error) {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return 0, err
	}
	tmp := dst + tmpSuffix
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	h := crc32.NewIEEE()
	n, err := io.Copy(f, io.TeeReader(r, h))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && size >= 0 && n != size {
		err = errors.New("it came out shorter than the archive says")
	}
	if err != nil {
		_ = os.Remove(tmp)
		return 0, err
	}
	return h.Sum32(), os.Rename(tmp, dst)
}

// writeNames keeps the list of the new version's files, for what's gone.
func writeNames(stage string, build []deltaEntry) error {
	var b strings.Builder
	for _, e := range build {
		b.WriteString(e.rel)
		b.WriteByte('\n')
	}
	return writeFileAtomic(filepath.Join(stage, "names.txt"), []byte(b.String()))
}

// deltaFull: the archive is the whole game when it has the exe and at least
// half of the bytes the game's folder has.
func deltaFull(p *DeltaPlan, gameDir string, build []deltaEntry) bool {
	if p.NewExe == "" {
		return false
	}
	var have int64
	for _, e := range build {
		have += e.size
	}
	return have*2 >= Size(gameDir)
}

// stageStreamed handles a .rar, read once from start to end: its list
// isn't known until then, so each file is written, then compared.
func stageStreamed(ctx context.Context, src DeltaSource, gameDir, stage string, o DeltaOptions) (*DeltaPlan, error) {
	r, err := rardecode.OpenReader(src.Name, rardecode.FileSystem(src.FS))
	if err != nil {
		return nil, errors.New("that .rar can't be opened; it may be damaged, or a part of it missing (" + shortErr(err) + ")")
	}
	defer r.Close()
	plan := &DeltaPlan{Format: "rar"}
	decided := false
	var build []deltaEntry
	var written int64
	lexe := strings.ToLower(o.ExeRel)
	for n := 0; ; n++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		h, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			if errors.Is(err, rardecode.ErrArchivedFileEncrypted) || strings.Contains(strings.ToLower(err.Error()), "password") {
				return nil, errors.New("that .rar has a password; Lootnest can't open it")
			}
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, fmt.Errorf("reading the .rar: %w", err)
		}
		if h.IsDir || h.LinkType != 0 {
			continue
		}
		name, err := updateRel(h.Name)
		if err != nil {
			return nil, err
		}
		if !decided && !readmeName(name) {
			// The archive's own top folders, from its first file: the
			// rest is read as it streams.
			plan.Prefix = bestPrefix(gameDir, []UpdateFile{{Rel: name}})
			decided = true
		}
		rel, in := strings.CutPrefix(name, plan.Prefix)
		if !in || rel == "" || !decided {
			continue
		}
		if strings.ToLower(rel) == lexe || (plan.NewExe == "" && strings.EqualFold(path.Base(rel), path.Base(o.ExeRel))) {
			plan.NewExe = rel
		}
		size := h.UnPackedSize
		build = append(build, deltaEntry{rel: rel, size: size})
		if o.Progress != nil {
			o.Progress(DeltaProgress{Step: "fetch", Done: n, Bytes: written})
		}
		dst := DeltaStaged(stage, rel)
		if st, err := os.Stat(dst); err == nil && !h.UnKnownSize && st.Size() == size {
			plan.add(gameDir, rel, size) // written by a try cut short
			continue
		}
		if o.Room != nil && !h.UnKnownSize {
			if err := o.Room(size); err != nil {
				return nil, err
			}
		}
		want := size
		if h.UnKnownSize {
			want = -1
		}
		if _, err := writeStaged(dst, r, want); err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, fmt.Errorf("%s: %w", rel, err)
		}
		if st, err := os.Stat(dst); err == nil {
			size = st.Size()
			build[len(build)-1].size = size
		}
		written += size
		if sameAsLocal(dst, filepath.Join(gameDir, filepath.FromSlash(rel))) {
			_ = os.Remove(dst)
			plan.Same++
			plan.SameBytes += size
			continue
		}
		plan.add(gameDir, rel, size)
	}
	plan.Files = len(build)
	if err := writeNames(stage, build); err != nil {
		return nil, err
	}
	plan.Full = deltaFull(plan, gameDir, build)
	return plan, nil
}

// deltaNames reads the new version's files, by lowercase path.
func deltaNames(stage string) (map[string]bool, map[string]bool, error) {
	b, err := os.ReadFile(filepath.Join(stage, "names.txt"))
	if err != nil {
		return nil, nil, err
	}
	all, tops := map[string]bool{}, map[string]bool{}
	for _, l := range strings.Split(string(b), "\n") {
		if l == "" {
			continue
		}
		l = strings.ToLower(l)
		all[l] = true
		for d := l; ; d = path.Dir(d) {
			all[d] = true // its folders too
			if path.Dir(d) == "." {
				tops[d] = true
				break
			}
		}
	}
	return all, tops, nil
}

// DeltaExtras are the things at the top of the game's folder that the new
// version doesn't have (a mods folder, a settings file): the user chooses
// which to keep. A store's files and a stand-in tool's aren't offered:
// they follow the new version.
func DeltaExtras(gameDir, stage string, plan *DeltaPlan) ([]KeepItem, error) {
	out := []KeepItem{}
	if !plan.Full {
		return out, nil
	}
	_, tops, err := deltaNames(stage)
	if err != nil {
		return nil, err
	}
	ents, _ := os.ReadDir(gameDir)
	for _, e := range ents {
		n := e.Name()
		if tops[strings.ToLower(n)] || keepBlocked(n) || e.Type()&fs.ModeSymlink != 0 || strings.HasSuffix(n, tmpSuffix) {
			continue
		}
		out = append(out, KeepItem{Rel: n, Why: "yours", Size: Size(filepath.Join(gameDir, n))})
	}
	return out, nil
}

// DeltaRemovals lists the game's files the new version doesn't have, for a
// whole new version only: they're moved aside with the old ones. Nothing
// under protect (save locations and what the user keeps, relative to the
// game's folder) is listed.
func DeltaRemovals(gameDir, stage string, plan *DeltaPlan, protect []string) ([]string, error) {
	if !plan.Full {
		return nil, nil
	}
	all, _, err := deltaNames(stage)
	if err != nil {
		return nil, err
	}
	var guard []string
	for _, p := range protect {
		if r, err := updateRel(p); err == nil {
			guard = append(guard, strings.ToLower(r))
		}
	}
	var out []string
	err = filepath.WalkDir(gameDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(gameDir, p)
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		lr := strings.ToLower(rel)
		for _, g := range guard {
			if lr == g || strings.HasPrefix(lr, g+"/") {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if d.IsDir() || all[lr] || strings.HasSuffix(lr, tmpSuffix) {
			return nil
		}
		out = append(out, rel)
		return nil
	})
	sort.Strings(out)
	return out, err
}

// DeltaApplied is the record of a delta update, enough to undo it.
type DeltaApplied struct {
	At       time.Time `json:"at"`
	Game     string    `json:"game"`  // the game's folder
	Keep     string    `json:"keep"`  // where the old files went
	Stage    string    `json:"stage"` // where the new ones came from
	Replaced []string  `json:"replaced"`
	Added    []string  `json:"added"`
	Removed  []string  `json:"removed"`
	Dirs     []string  `json:"dirs,omitempty"` // folders it made, deepest first
	State    string    `json:"state"`          // applying, applied, undone
}

// deltaAfterMove lets tests stop an apply after n moves.
var deltaAfterMove func(n int) error

// ApplyDelta moves the staged files into the game's folder, and the ones
// they replace or that are gone from the new version into keep. exe is the
// game's exe afterwards, relative to its folder, checked last. On any
// failure everything moved is moved back.
func ApplyDelta(gameDir, stage, keep, exe string, plan *DeltaPlan, remove []string, save func(*DeltaApplied) error) (*DeltaApplied, error) {
	if _, err := os.Lstat(keep); err == nil {
		return nil, errors.New("the files kept from the update before are still there. Roll that update back, or let Lootnest remove them, first")
	}
	a := &DeltaApplied{At: time.Now().UTC(), Game: gameDir, Keep: keep, Stage: stage, Replaced: []string{}, Added: []string{}, Removed: []string{}, State: "applying"}
	dirSeen := map[string]bool{}
	for _, f := range plan.Changed {
		rel, err := updateRel(f.Rel)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(DeltaStaged(stage, rel)); err != nil {
			return nil, errors.New("a downloaded file is missing (" + rel + "). Get the update again")
		}
		if _, err := os.Lstat(filepath.Join(gameDir, filepath.FromSlash(rel))); err == nil {
			a.Replaced = append(a.Replaced, rel)
			continue
		}
		a.Added = append(a.Added, rel)
		for d := path.Dir(rel); d != "." && !dirSeen[d]; d = path.Dir(d) {
			dirSeen[d] = true
			if _, err := os.Lstat(filepath.Join(gameDir, filepath.FromSlash(d))); err != nil {
				a.Dirs = append(a.Dirs, d)
			}
		}
	}
	for _, r := range remove {
		rel, err := updateRel(r)
		if err != nil {
			return nil, err
		}
		a.Removed = append(a.Removed, rel)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(a.Dirs)))
	if err := save(a); err != nil {
		return nil, err
	}
	moves := 0
	step := func(from, to string) error {
		if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
			return err
		}
		if err := os.Rename(from, to); err != nil {
			return fmt.Errorf("couldn't move %s (a file in it may be open): %w", filepath.Base(from), err)
		}
		moves++
		if deltaAfterMove != nil {
			return deltaAfterMove(moves)
		}
		return nil
	}
	in := func(rel string) string { return filepath.Join(gameDir, filepath.FromSlash(rel)) }
	kept := func(rel string) string { return filepath.Join(keep, filepath.FromSlash(rel)) }
	fail := func(err error) (*DeltaApplied, error) {
		if rerr := RevertDelta(a, save); rerr != nil {
			return a, fmt.Errorf("%v; putting the files back failed too (%v): Lootnest tries again at its next start", err, rerr)
		}
		return a, err
	}
	for _, rel := range a.Replaced {
		if err := step(in(rel), kept(rel)); err != nil {
			return fail(err)
		}
		if err := step(DeltaStaged(stage, rel), in(rel)); err != nil {
			return fail(err)
		}
	}
	for _, rel := range a.Added {
		if err := step(DeltaStaged(stage, rel), in(rel)); err != nil {
			return fail(err)
		}
	}
	for _, rel := range a.Removed {
		if _, err := os.Lstat(in(rel)); err != nil {
			continue
		}
		if err := step(in(rel), kept(rel)); err != nil {
			return fail(err)
		}
	}
	if exe != "" {
		if st, err := os.Stat(in(exe)); err != nil || !st.Mode().IsRegular() {
			return fail(errors.New("the game's exe, " + exe + ", isn't there after the update"))
		}
	}
	a.State = "applied"
	return a, save(a)
}

// RevertDelta moves everything a delta update moved back where it was: the
// new files back into the staging folder, the old ones back into the
// game's folder. It works from the record alone and can run again.
func RevertDelta(a *DeltaApplied, save func(*DeltaApplied) error) error {
	in := func(rel string) string { return filepath.Join(a.Game, filepath.FromSlash(rel)) }
	kept := func(rel string) string { return filepath.Join(a.Keep, filepath.FromSlash(rel)) }
	exists := func(p string) bool { _, err := os.Lstat(p); return err == nil }
	back := func(rel string) error { // a new file, back to the staging folder
		p := in(rel)
		if !exists(p) {
			return nil
		}
		s := DeltaStaged(a.Stage, rel)
		if !exists(s) {
			if err := os.MkdirAll(filepath.Dir(s), 0o755); err == nil && os.Rename(p, s) == nil {
				return nil
			}
		}
		return os.Remove(p) // it's the new version's, which can be got again
	}
	var first error
	note := func(err error) {
		if err != nil && first == nil {
			first = err
		}
	}
	for _, rel := range a.Removed {
		if exists(kept(rel)) && !exists(in(rel)) {
			note(os.MkdirAll(filepath.Dir(in(rel)), 0o755))
			note(os.Rename(kept(rel), in(rel)))
		}
	}
	for _, rel := range a.Replaced {
		if !exists(kept(rel)) {
			continue // never moved, or already back
		}
		note(back(rel))
		note(os.Rename(kept(rel), in(rel)))
	}
	for _, rel := range a.Added {
		note(back(rel))
	}
	for _, d := range a.Dirs {
		_ = os.Remove(in(d)) // only when empty
	}
	if first != nil {
		return first
	}
	removeEmptyDirs(a.Keep)
	a.State = "undone"
	return save(a)
}

// removeEmptyDirs removes dir and the folders in it that hold nothing.
func removeEmptyDirs(dir string) {
	var dirs []string
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			dirs = append(dirs, p)
		}
		return nil
	})
	for i := len(dirs) - 1; i >= 0; i-- {
		_ = os.Remove(dirs[i])
	}
}

// SaveDeltaApplied and LoadDeltaApplied keep the record in dir.
func SaveDeltaApplied(dir string, a *DeltaApplied) error {
	b, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return writeFileAtomic(filepath.Join(dir, "delta.json"), b)
}

func LoadDeltaApplied(dir string) (*DeltaApplied, error) {
	b, err := os.ReadFile(filepath.Join(dir, "delta.json"))
	if err != nil {
		return nil, err
	}
	var a DeltaApplied
	return &a, json.Unmarshal(b, &a)
}
