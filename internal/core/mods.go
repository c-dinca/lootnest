package core

import (
	"cmp"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Mods, phases 1 and 2 (docs/ROADMAP.md, section 2): sets of mods per game
// and per profile, put into its folder by copy. The user brings each mod, an archive or a folder;
// Lootnest never downloads one, never ships a loader, and never uses links
// or a virtual file system.
//
// A game's mods live in Lootnest's data folder, under mods\<key hash>:
//
//	<mod id>\mod.json, <mod id>\files\  each mod, laid out as it goes into the game's folder
//	set.json         the sets by owner: the mods in order, on or off; a later mod wins on the same file
//	deployed.json    what's in the game's folder now: whose set, each file, its mod and hash
//	originals\       every game file a mod replaced, kept before it was touched
//	changed\<time>\  files the game or the user changed while it was modded
//	journal.json     a run that started, finished on the next start if cut short
//
// Only Lootnest's own copies are ever deleted, and the files it added only
// while they still hash as it put them in.
// ---------------------------------------------------------------------------

// ModMax is the most a mod may hold once extracted.
const ModMax int64 = 2 << 30

// ModFile is one file of a mod, where it goes in the game's folder.
type ModFile struct {
	Rel    string `json:"rel"` // with "/"
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// Mod is one mod Lootnest keeps for a game.
type Mod struct {
	ID     string    `json:"id"`
	Name   string    `json:"name"`
	Source string    `json:"source"` // the archive's or folder's name
	Added  time.Time `json:"added"`
	Into   string    `json:"into,omitempty"`   // the game's folder it goes into, "" for its root
	AtRoot bool      `json:"atRoot,omitempty"` // nothing in the game matched it, so it goes at the root
	Files  []ModFile `json:"files"`
}

// ModEntry is a mod's place in a set. The mods are kept once for the game
// and shared; each set only says which are on and in what order.
type ModEntry struct {
	ID string `json:"id"`
	On bool   `json:"on"`
}

// DeployedFile is one file Lootnest put into a game's folder.
type DeployedFile struct {
	Rel      string `json:"rel"`
	Mod      string `json:"mod"`
	SHA256   string `json:"sha256"`
	Replaces bool   `json:"replaces,omitempty"`
	Orig     string `json:"orig,omitempty"` // the kept original's sha256
}

// ModsDeployed is what's in a game's folder from its mods.
type ModsDeployed struct {
	At    time.Time      `json:"at"`
	Owner string         `json:"owner,omitempty"` // whose set: "" the game's, else a profile id
	Mods  []string       `json:"mods"`
	Files []DeployedFile `json:"files"`
	Dirs  []string       `json:"dirs,omitempty"` // folders it made, deepest first
}

// ModPlan is what putting mods in does to the game's folder.
type ModPlan struct {
	Added    int      `json:"added"`
	Replaced int      `json:"replaced"`
	Bytes    int64    `json:"bytes"`
	Replaces []string `json:"replaces"` // the game's files it replaces
	files    []DeployedFile
	dirs     []string
}

// ModsResult says how a run went.
type ModsResult struct {
	Changed    int    `json:"changed"`              // files changed since they went in, kept aside
	ChangedDir string `json:"changedDir,omitempty"` // where
}

// modJournal is a run that started: "undeploy" (then the mods in Then go
// in) or "deploy" (Plan goes in).
type modJournal struct {
	Op    string        `json:"op"`
	Game  string        `json:"game"`
	Owner string        `json:"owner,omitempty"`
	Exe   string        `json:"exe,omitempty"`
	At    time.Time     `json:"at"`
	Then  []string      `json:"then,omitempty"`
	Plan  *ModsDeployed `json:"plan,omitempty"`
}

// ModsDir is where one game's mods are kept, by its Library key.
func ModsDir(dataDir, key string) string {
	h := sha1.Sum([]byte(strings.ToLower(key)))
	return filepath.Join(dataDir, "mods", hex.EncodeToString(h[:6]))
}

// modBlocked are the files a store, its copy protection or a Steam
// emulator reads to decide whether the game starts and whose it is. A mod
// bringing one is refused: Lootnest never ships or "repairs" anything that
// makes a game start. Lowercase names, matched with path.Match; anything in
// a steam_settings folder is refused as well.
var modBlocked = []string{
	"steam_api.dll", "steam_api64.dll", "steamclient*.dll", "steam_appid.txt", "steam_interfaces.txt", "steam_emu.ini",
	"eossdk-*.dll", "galaxy*.dll", "uplay_r1*.dll", "uplay_r2*.dll", "upc_r*.dll",
	"*.cdx", "coldclientloader.ini", "onlinefix*.ini", "onlinefix*.dll", "cream_api.ini",
	"codex.ini", "rune.ini", "rld.ini", "skidrow.ini", "3dmgame.ini", "ali213.ini",
}

func modBlockedFile(rel string) bool {
	parts := strings.Split(strings.ToLower(rel), "/")
	for _, p := range parts[:len(parts)-1] {
		if p == "steam_settings" {
			return true
		}
	}
	for _, pat := range modBlocked {
		if ok, _ := path.Match(pat, parts[len(parts)-1]); ok {
			return true
		}
	}
	return false
}

// modDataDirs are folders games keep their content in, where a mod's files
// go when they match what's inside.
var modDataDirs = []string{"Data", "Content", "Mods", "BepInEx/plugins", "BepInEx"}

// ModsRefused says why Lootnest won't put mods into the game in gameDir,
// or nil.
func ModsRefused(gameDir string) error {
	for _, p := range strings.FieldsFunc(strings.ToLower(filepath.Clean(gameDir)), func(r rune) bool { return r == '\\' || r == '/' }) {
		if p == "program files" || p == "program files (x86)" || p == "windowsapps" {
			return errors.New("Windows protects the games under Program Files and WindowsApps, so Lootnest doesn't put mods there. A game in a folder outside them can have mods")
		}
	}
	if ac := AntiCheat(gameDir); ac != "" {
		return fmt.Errorf("this game ships with %s, which can ban an account over changed game files, so Lootnest doesn't put mods into it", ac)
	}
	return nil
}

// AddMod keeps a copy of a mod (a .zip, .7z or .rar, or a folder) for the
// game in gameDir, laid out as it goes into the game's folder. Nothing in
// the game's folder changes. Mods kept earlier that aren't in the set (an
// add that wasn't finished) are deleted first.
func AddMod(root, gameDir, src string, now time.Time) (*Mod, error) {
	st, err := os.Stat(src)
	if err != nil {
		return nil, errors.New("Lootnest can't find that mod")
	}
	if isWithin(src, gameDir) || isWithin(gameDir, src) {
		return nil, errors.New("choose the mod, not the game's own folder")
	}
	pruneMods(root)
	m := &Mod{ID: NewID("m"), Source: filepath.Base(src), Added: now.UTC(), Name: filepath.Base(src)}
	dir := filepath.Join(root, m.ID)
	fail := func(err error) (*Mod, error) {
		_ = removeAll(dir)
		return nil, err
	}
	from, staged := src, false
	if !st.IsDir() {
		m.Name = rarPartRe.ReplaceAllString(strings.TrimSuffix(m.Name, filepath.Ext(m.Name)), "")
		from, staged = filepath.Join(dir, "stage"), true
		if _, err := StageUpdateMax(src, from, ModMax); err != nil {
			return fail(err)
		}
	}
	files, err := modSourceFiles(from)
	if err != nil {
		return fail(err)
	}
	var prefix string
	prefix, m.Into, m.AtRoot = modPlace(gameDir, files)
	seen := map[string]bool{}
	for _, f := range files {
		rel, ok := strings.CutPrefix(f.Rel, prefix)
		if !ok || rel == "" || readmeName(rel) {
			continue // notes beside the mod's files
		}
		if m.Into != "" {
			rel = m.Into + "/" + rel
		}
		if seen[strings.ToLower(rel)] {
			continue
		}
		seen[strings.ToLower(rel)] = true
		dst := filepath.Join(dir, "files", filepath.FromSlash(rel))
		if staged {
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err == nil {
				err = os.Rename(f.src, dst)
			}
		} else if info, ierr := os.Stat(f.src); ierr != nil {
			err = ierr
		} else {
			err = copyFile(f.src, dst, info)
		}
		if err != nil {
			return fail(fmt.Errorf("%s: %w", f.Rel, err))
		}
		sum, err := fileSHA256(dst)
		if err != nil {
			return fail(err)
		}
		m.Files = append(m.Files, ModFile{Rel: rel, Size: f.Size, SHA256: sum})
	}
	_ = removeAll(filepath.Join(dir, "stage"))
	if len(m.Files) == 0 {
		return fail(errors.New("the mod has no files, only notes"))
	}
	sort.Slice(m.Files, func(i, j int) bool { return m.Files[i].Rel < m.Files[j].Rel })
	if err := writeJSONSynced(filepath.Join(dir, "mod.json"), m); err != nil {
		return fail(err)
	}
	return m, nil
}

// modSourceFiles lists a mod's files, refusing paths that leave it and
// mods over ModMax. Links are never followed.
func modSourceFiles(dir string) ([]UpdateFile, error) {
	var out []UpdateFile
	var total int64
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&fs.ModeSymlink != 0 || d.Type()&fs.ModeIrregular != 0 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		r, err := updateRel(rel)
		if err != nil {
			return errors.New("the mod has a file outside its own folder")
		}
		if total += info.Size(); total > ModMax {
			return ErrTooBig
		}
		out = append(out, UpdateFile{Rel: r, Size: info.Size(), src: p})
		return nil
	})
	return out, err
}

// modPlace works out where a mod's files go: the folder of the mod to
// skip (an archive often wraps its files in one), and the game's folder
// they go into. A mod whose top level matches the game's folder goes at
// its root; one matching what's inside a data folder the game has (Data,
// Content, Mods, BepInEx) goes there. With no match it goes at the root,
// as it is, and atRoot says so.
func modPlace(gameDir string, files []UpdateFile) (prefix, into string, atRoot bool) {
	entries := func(rel string) map[string]bool {
		have := map[string]bool{}
		es, _ := os.ReadDir(filepath.Join(gameDir, filepath.FromSlash(rel)))
		for _, e := range es {
			have[strings.ToLower(e.Name())] = true
		}
		return have
	}
	tops := func(prefix string) (map[string]bool, string) {
		out, only := map[string]bool{}, ""
		for _, f := range files {
			rel, ok := strings.CutPrefix(f.Rel, prefix)
			if !ok || readmeName(rel) {
				continue
			}
			top, _, dir := strings.Cut(rel, "/")
			out[strings.ToLower(top)] = true
			if !dir || (only != "" && !strings.EqualFold(only, top)) {
				only = "\x00"
			} else if only == "" {
				only = top
			}
		}
		if only == "\x00" {
			only = ""
		}
		return out, only
	}
	matches := func(have, want map[string]bool) bool {
		for t := range want {
			if have[t] {
				return true
			}
		}
		return false
	}
	var data []string // the data folders the game has, as it spells them
	for _, d := range modDataDirs {
		if r, ok := realRel(gameDir, d); ok {
			data = append(data, r)
		}
	}
	game := entries("")
	for p, depth := "", 0; depth <= 2; depth++ {
		want, only := tops(p)
		if matches(game, want) {
			return p, "", false
		}
		for _, d := range data {
			if matches(entries(d), want) {
				return p, d, false
			}
		}
		if only == "" {
			break
		}
		p += only + "/"
		if strings.EqualFold(only, filepath.Base(gameDir)) {
			return p, "", false // the mod's folder is named like the game's
		}
	}
	return "", "", true
}

// realRel is rel inside dir as the folder spells it, when it's a folder.
func realRel(dir, rel string) (string, bool) {
	var out []string
	for _, part := range strings.Split(rel, "/") {
		es, err := os.ReadDir(dir)
		if err != nil {
			return "", false
		}
		found := false
		for _, e := range es {
			if e.IsDir() && strings.EqualFold(e.Name(), part) {
				out, dir, found = append(out, e.Name()), filepath.Join(dir, e.Name()), true
				break
			}
		}
		if !found {
			return "", false
		}
	}
	return strings.Join(out, "/"), true
}

// PlanMods works out what putting mods (in order, a later one winning on
// the same file) into the game in gameDir does, or refuses with the
// reason. mainExe is the game's program, "" to find it; deployed is what's
// in its folder from mods now, so the plan is against the game as it was.
func PlanMods(gameDir, mainExe string, mods []*Mod, deployed *ModsDeployed) (*ModPlan, error) {
	if st, err := os.Stat(gameDir); err != nil || !st.IsDir() {
		return nil, errors.New("the game's folder isn't there")
	}
	if err := ModsRefused(gameDir); err != nil {
		return nil, err
	}
	if mainExe == "" {
		if exes := GameExes(gameDir, ""); len(exes) > 0 {
			mainExe = exes[0]
		}
	}
	mainRel := ""
	if r, err := filepath.Rel(gameDir, mainExe); err == nil && mainExe != "" {
		mainRel = filepath.ToSlash(r)
	}
	ours := map[string]DeployedFile{}
	if deployed != nil {
		for _, f := range deployed.Files {
			ours[strings.ToLower(f.Rel)] = f
		}
	}
	at := map[string]int{}
	var files []DeployedFile
	var sizes []int64
	var blocked []string
	for _, m := range mods {
		for _, f := range m.Files {
			rel, err := updateRel(f.Rel)
			if err != nil || rel != f.Rel {
				return nil, fmt.Errorf("%s has a file outside the game's folder (%s)", m.Name, f.Rel)
			}
			if modBlockedFile(rel) {
				blocked = append(blocked, path.Base(rel))
				continue
			}
			if strings.EqualFold(rel, mainRel) {
				return nil, fmt.Errorf("%s would replace the game's own program, %s, so Lootnest doesn't put it in", m.Name, path.Base(rel))
			}
			d := DeployedFile{Rel: rel, Mod: m.ID, SHA256: f.SHA256}
			k := strings.ToLower(rel)
			if i, ok := at[k]; ok {
				files[i], sizes[i] = d, f.Size
			} else {
				at[k] = len(files)
				files, sizes = append(files, d), append(sizes, f.Size)
			}
		}
	}
	if len(blocked) > 0 {
		sort.Strings(blocked)
		blocked = slices.Compact(blocked)
		if len(blocked) > 3 {
			blocked = append(blocked[:3], "others")
		}
		return nil, fmt.Errorf("the mod brings %s, which a store, its copy protection or a Steam emulator reads to start the game, so Lootnest doesn't put it in", joinAnd(blocked))
	}
	p := &ModPlan{Replaces: []string{}}
	dirSeen := map[string]bool{}
	for i, f := range files {
		target := filepath.Join(gameDir, filepath.FromSlash(f.Rel))
		if o, ok := ours[strings.ToLower(f.Rel)]; ok {
			f.Replaces = o.Replaces
		} else if st, err := os.Stat(target); err == nil {
			if st.IsDir() {
				return nil, fmt.Errorf("the mod has a file, %s, where the game has a folder", f.Rel)
			}
			f.Replaces = true
		}
		for d := path.Dir(f.Rel); d != "." && !dirSeen[d]; d = path.Dir(d) {
			dirSeen[d] = true
			st, err := os.Stat(filepath.Join(gameDir, filepath.FromSlash(d)))
			if err == nil && !st.IsDir() {
				return nil, fmt.Errorf("the mod has a folder, %s, where the game has a file", d)
			}
			if err != nil {
				p.dirs = append(p.dirs, d)
			}
		}
		if f.Replaces {
			p.Replaced++
			p.Replaces = append(p.Replaces, f.Rel)
		} else {
			p.Added++
		}
		p.Bytes += sizes[i]
		p.files = append(p.files, f)
	}
	sort.Strings(p.Replaces)
	sort.Sort(sort.Reverse(sort.StringSlice(p.dirs))) // deepest first
	return p, nil
}

// ApplyMods makes the game's folder hold exactly mods, in order, as owner's
// set: what's in from mods now comes out first, then mods go in. With no mods, it only puts
// the game back as it was. Every step is in the journal first, so a run
// cut short is finished by FinishModJournal. step, if not nil, hears how
// far it is.
func ApplyMods(root, gameDir, mainExe, owner string, mods []*Mod, now time.Time, step func(done, total int)) (ModsResult, error) {
	last, err := readModJournal(root)
	if err != nil {
		return ModsResult{}, err
	}
	if last != nil && len(mods) == 0 {
		// Putting the game back takes out what the last run put in, even
		// when that run can't finish (a mod file Lootnest kept is gone).
		j := &modJournal{Op: "undeploy", Game: last.Game, Exe: last.Exe, At: now.UTC(), Plan: last.Plan}
		if err := writeJSONSynced(modJournalPath(root), j); err != nil {
			return ModsResult{}, err
		}
		return runModJournal(root, j, step)
	}
	if last != nil {
		if _, err := runModJournal(root, last, nil); err != nil {
			return ModsResult{}, err // the last run first, so its files are known
		}
	}
	if len(mods) > 0 {
		if _, err := PlanMods(gameDir, mainExe, mods, LoadDeployed(root)); err != nil {
			return ModsResult{}, err // refused before anything changes
		}
	}
	j := &modJournal{Op: "undeploy", Game: gameDir, Exe: mainExe, Owner: owner, At: now.UTC()}
	for _, m := range mods {
		j.Then = append(j.Then, m.ID)
	}
	if err := writeJSONSynced(modJournalPath(root), j); err != nil {
		return ModsResult{}, err
	}
	return runModJournal(root, j, step)
}

// FinishModJournal finishes a run that was cut short (Lootnest closed, the
// PC turned off, or a file was in use), and says whether there was one.
func FinishModJournal(root string) (bool, ModsResult, error) {
	j, err := readModJournal(root)
	if j == nil || err != nil {
		return err != nil, ModsResult{}, err
	}
	res, err := runModJournal(root, j, nil)
	return true, res, err
}

// readModJournal is the run that didn't finish, or nil.
func readModJournal(root string) (*modJournal, error) {
	b, err := os.ReadFile(modJournalPath(root))
	if err != nil {
		return nil, nil
	}
	var j modJournal
	if err := json.Unmarshal(b, &j); err != nil {
		return nil, fmt.Errorf("the record of the last mods run can't be read: %w", err)
	}
	return &j, nil
}

// ModJournalGame is the game folder a run that didn't finish was changing,
// or "".
func ModJournalGame(root string) string {
	if j, _ := readModJournal(root); j != nil {
		return j.Game
	}
	return ""
}

// ModJournals lists the games' mod folders with a run that didn't finish.
func ModJournals(dataDir string) []string {
	ms, _ := filepath.Glob(filepath.Join(dataDir, "mods", "*", "journal.json"))
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		out = append(out, filepath.Dir(m))
	}
	return out
}

// ModJournalPending says whether a run for this game didn't finish.
func ModJournalPending(root string) bool { return exists(modJournalPath(root)) }

func modJournalPath(root string) string { return filepath.Join(root, "journal.json") }

func runModJournal(root string, j *modJournal, step func(done, total int)) (ModsResult, error) {
	var res ModsResult
	if j.Op == "undeploy" {
		var err error
		if res, err = undeployMods(root, j, step); err != nil {
			return res, err
		}
		if len(j.Then) == 0 {
			return res, os.Remove(modJournalPath(root))
		}
		mods, err := LoadMods(root, j.Then)
		var p *ModPlan
		if err == nil {
			p, err = PlanMods(j.Game, j.Exe, mods, nil)
		}
		if err != nil {
			_ = os.Remove(modJournalPath(root)) // the game is as it was, so there's nothing to finish
			return res, err
		}
		j = &modJournal{Op: "deploy", Game: j.Game, Exe: j.Exe, Owner: j.Owner, At: j.At, Plan: &ModsDeployed{At: j.At, Owner: j.Owner, Mods: j.Then, Files: p.files, Dirs: p.dirs}}
		if err := writeJSONSynced(modJournalPath(root), j); err != nil {
			return res, err
		}
	}
	if err := deployMods(root, j, &res, step); err != nil {
		return res, err
	}
	return res, os.Remove(modJournalPath(root))
}

// modStep lets tests stop a run part way, as if the PC turned off.
var modStep func(i int) error

// deployMods puts a plan's files in. Each game file a mod replaces is kept
// in originals first and read back (a copy already there is the true
// original and is never overwritten); each mod file goes in through a
// temporary name beside it and is read back before it's moved in. Files
// already in as planned are left, so a run cut short goes on from there; a
// file the game changed after its original was kept is kept in changed
// first.
func deployMods(root string, j *modJournal, res *ModsResult, step func(done, total int)) error {
	d := j.Plan
	for i := range d.Files {
		if modStep != nil {
			if err := modStep(i); err != nil {
				return err
			}
		}
		f := &d.Files[i]
		target := filepath.Join(j.Game, filepath.FromSlash(f.Rel))
		orig := filepath.Join(root, "originals", filepath.FromSlash(f.Rel))
		cur, err := fileSHA256(target)
		if err != nil {
			return modErr(f.Rel, err)
		}
		kept := exists(orig)
		switch {
		case kept:
			f.Replaces = true
		case cur != "" && (f.Replaces || cur != f.SHA256): // equal to the mod's and not in the plan: placed earlier in this run
			info, err := os.Stat(target)
			if err == nil {
				err = copyFile(target, orig+tmpSuffix, info)
			}
			if err == nil {
				err = os.Rename(orig+tmpSuffix, orig)
			}
			if err != nil {
				_ = os.Remove(orig + tmpSuffix)
				return fmt.Errorf("couldn't keep a copy of %s first: %w", f.Rel, err)
			}
			f.Replaces = true
		default:
			f.Replaces = false
		}
		if f.Replaces {
			if f.Orig, err = fileSHA256(orig); err != nil {
				return modErr(f.Rel, err)
			}
		}
		if kept && cur != "" && cur != f.SHA256 && cur != f.Orig {
			if err := res.keep(changedDir(root, j.At), target, f.Rel, false); err != nil {
				return err
			}
		}
		if cur != f.SHA256 {
			if err := placeChecked(filepath.Join(root, f.Mod, "files", filepath.FromSlash(f.Rel)), target, f.SHA256); err != nil {
				return modErr(f.Rel, err)
			}
		}
		if step != nil {
			step(i+1, len(d.Files))
		}
	}
	return writeJSONSynced(filepath.Join(root, "deployed.json"), d)
}

// undeployMods takes what's deployed out of the game's folder: the kept
// originals go back, and the files mods added go. A file changed since it
// went in (by the game or the user) is kept in changed\<time> first; an
// added one is moved there rather than deleted. With nothing recorded as
// deployed, the journal's plan says what a deploy cut short may have put in.
func undeployMods(root string, j *modJournal, step func(done, total int)) (ModsResult, error) {
	var res ModsResult
	d := LoadDeployed(root)
	if d == nil {
		d = j.Plan
	}
	if d == nil {
		return res, nil
	}
	changed := changedDir(root, j.At)
	for i, f := range d.Files {
		if modStep != nil {
			if err := modStep(i); err != nil {
				return res, err
			}
		}
		p := filepath.Join(j.Game, filepath.FromSlash(f.Rel))
		_ = os.Remove(p + tmpSuffix)
		cur, err := fileSHA256(p)
		if err != nil {
			return res, modErr(f.Rel, err)
		}
		orig := filepath.Join(root, "originals", filepath.FromSlash(f.Rel))
		kept, err := fileSHA256(orig)
		if err != nil {
			return res, modErr(f.Rel, err)
		}
		want := cmp.Or(f.Orig, kept)
		switch {
		case kept != "" && cur == want:
			// back already, in a run cut short
		case kept != "":
			if cur != "" && cur != f.SHA256 {
				if err := res.keep(changed, p, f.Rel, false); err != nil {
					return res, err
				}
			}
			if err := placeChecked(orig, p, want); err != nil {
				return res, modErr(f.Rel, err)
			}
		case f.Replaces, cur == "":
			// the game's own file never left, or is back and its copy gone
		case cur == f.SHA256:
			if err := removeFile(p); err != nil {
				return res, modErr(f.Rel, err)
			}
		default:
			if err := res.keep(changed, p, f.Rel, true); err != nil {
				return res, modErr(f.Rel, err)
			}
		}
		if step != nil {
			step(i+1, len(d.Files))
		}
	}
	for _, dir := range d.Dirs {
		_ = os.Remove(filepath.Join(j.Game, filepath.FromSlash(dir))) // only when empty
	}
	// The originals are back in the game's folder, so Lootnest's copies of
	// them go before the record of what's deployed: a deploy never meets
	// copies left from an earlier one.
	if err := removeAll(filepath.Join(root, "originals")); err != nil {
		return res, err
	}
	if err := os.Remove(filepath.Join(root, "deployed.json")); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return res, err
	}
	return res, nil
}

func changedDir(root string, at time.Time) string {
	return filepath.Join(root, "changed", at.Local().Format("20060102-150405"))
}

// keep copies p, a file the game or the user changed while it was modded,
// into dir before Lootnest puts another file in its place; move takes it
// out of the game's folder.
func (res *ModsResult) keep(dir, p, rel string, move bool) error {
	info, err := os.Stat(p)
	if err != nil {
		return err
	}
	if err := copyFile(p, filepath.Join(dir, filepath.FromSlash(rel)), info); err != nil {
		return fmt.Errorf("couldn't keep a copy of %s, which changed since it went in: %w", rel, err)
	}
	res.Changed++
	res.ChangedDir = dir
	if move {
		return removeFile(p)
	}
	return nil
}

func modErr(rel string, err error) error {
	if s := err.Error(); errors.Is(err, fs.ErrPermission) || strings.Contains(s, "in use") || strings.Contains(s, "being used") {
		return fmt.Errorf("%s is in use, so Lootnest stopped part way. Close the game and try again; Lootnest also finishes it the next time it starts", rel)
	}
	return fmt.Errorf("%s: %w", rel, err)
}

// placeChecked copies src over dst through a temporary name beside it, and
// moves it in only once the copy reads back as want.
func placeChecked(src, dst, want string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	tmp := dst + tmpSuffix
	if err := copyFile(src, tmp, info); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if got, err := fileSHA256(tmp); err != nil || got != want {
		_ = os.Remove(tmp)
		return fmt.Errorf("%s: %w", dst, ErrCopyMismatch)
	}
	return moveInto(tmp, dst)
}

// removeFile deletes a file, clearing read-only first if it must.
func removeFile(p string) error {
	if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
		_ = os.Chmod(p, 0o666)
		if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return nil
}

// fileSHA256 is a file's sha256, or "" when it isn't there.
func fileSHA256(p string) (string, error) {
	f, err := os.Open(p)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func writeJSONSynced(p string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(p, b)
}

// LoadDeployed is what's in a game's folder from mods, or nil.
func LoadDeployed(root string) *ModsDeployed {
	b, err := os.ReadFile(filepath.Join(root, "deployed.json"))
	if err != nil {
		return nil
	}
	var d ModsDeployed
	if json.Unmarshal(b, &d) != nil {
		return nil
	}
	return &d
}

// LoadModSets is a game's sets by owner: "" is the game's own, used when
// profiles are off or a profile has no set of its own, else a profile id.
// The one list 1.86 kept is the game's set.
func LoadModSets(root string) map[string][]ModEntry {
	out := map[string][]ModEntry{}
	b, err := os.ReadFile(filepath.Join(root, "set.json"))
	if err != nil {
		return out
	}
	var old []ModEntry
	if json.Unmarshal(b, &old) == nil {
		out[""] = old
		return out
	}
	_ = json.Unmarshal(b, &out)
	return out
}

// LoadModSet is owner's own set in order.
func LoadModSet(root, owner string) []ModEntry { return LoadModSets(root)[owner] }

// SaveModSet writes owner's own set.
func SaveModSet(root, owner string, set []ModEntry) error {
	sets := LoadModSets(root)
	sets[owner] = set
	return writeJSONSynced(filepath.Join(root, "set.json"), sets)
}

// ModSetFor is the set a profile plays with and whose it is: its own, else
// its person's when it's a run (of), else the game's. Every mod the game
// keeps is in it; those only other sets have are at the end, off.
func ModSetFor(root, profileID, of string) (string, []ModEntry) {
	return ModSetIn(LoadModSets(root), profileID, of)
}

// ModSetIn is ModSetFor over sets already loaded.
func ModSetIn(sets map[string][]ModEntry, profileID, of string) (string, []ModEntry) {
	owner := ""
	for _, o := range []string{profileID, of} {
		if _, ok := sets[o]; ok && o != "" {
			owner = o
			break
		}
	}
	out := slices.Clone(sets[owner])
	seen := map[string]bool{}
	for _, e := range out {
		seen[e.ID] = true
	}
	for _, o := range slices.Sorted(maps.Keys(sets)) {
		for _, e := range sets[o] {
			if !seen[e.ID] {
				seen[e.ID] = true
				out = append(out, ModEntry{ID: e.ID})
			}
		}
	}
	return owner, out
}

// LoadMods reads mods by id.
func LoadMods(root string, ids []string) ([]*Mod, error) {
	out := make([]*Mod, 0, len(ids))
	for _, id := range ids {
		m, err := LoadMod(root, id)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

// LoadMod reads one mod.
func LoadMod(root, id string) (*Mod, error) {
	if !validModID(id) {
		return nil, errors.New("unknown mod")
	}
	b, err := os.ReadFile(filepath.Join(root, id, "mod.json"))
	if err != nil {
		return nil, errors.New("that mod isn't kept any more")
	}
	var m Mod
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func validModID(id string) bool {
	return id != "" && id == filepath.Base(id) && strings.HasPrefix(id, "m")
}

// RemoveMod deletes Lootnest's copy of a mod and takes it out of every set.
// A mod with files in the game's folder must come out of it first.
func RemoveMod(root, id string) error {
	if !validModID(id) {
		return errors.New("unknown mod")
	}
	if d := LoadDeployed(root); d != nil {
		for _, f := range d.Files {
			if f.Mod == id {
				return errors.New("that mod is in the game's folder; put the game back as it was first")
			}
		}
	}
	sets := LoadModSets(root)
	for o, set := range sets {
		sets[o] = slices.DeleteFunc(set, func(e ModEntry) bool { return e.ID == id })
	}
	if err := writeJSONSynced(filepath.Join(root, "set.json"), sets); err != nil {
		return err
	}
	return removeAll(filepath.Join(root, id))
}

// pruneMods deletes the mods kept for a game that are neither in its set
// nor in its folder: adds that weren't finished.
func pruneMods(root string) {
	keep := map[string]bool{}
	for _, set := range LoadModSets(root) {
		for _, e := range set {
			keep[e.ID] = true
		}
	}
	if d := LoadDeployed(root); d != nil {
		for _, f := range d.Files {
			keep[f.Mod] = true
		}
	}
	es, _ := os.ReadDir(root)
	for _, e := range es {
		if e.IsDir() && validModID(e.Name()) && !keep[e.Name()] {
			_ = removeAll(filepath.Join(root, e.Name()))
		}
	}
}
