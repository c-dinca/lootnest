package core

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// big is the 7z fixtures' data/big.pak (testdata/delta-*.7z, made with py7zr
// from the same rule).
func big() []byte {
	b := make([]byte, 200000)
	for i := range b {
		b[i] = byte((i*31 + i/251) % 256)
	}
	return b
}

// v2 is the new version, as the fixtures hold it under "Wrap/".
func v2() map[string][]byte {
	return map[string][]byte{"Game.exe": []byte("exe v2"), "data/big.pak": big(), "data/level1.pak": []byte("level one v2"),
		"data/new.pak": []byte("new in v2"), "readme.txt": []byte("hello")}
}

// gameV1 makes the version installed, with saves and a mod of the user's.
func gameV1(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "Games", "Night Tale")
	for name, body := range map[string][]byte{"Game.exe": []byte("exe v1"), "data/big.pak": big(), "data/level1.pak": []byte("level one v1"),
		"data/old.pak": []byte("gone in v2"), "Saves/slot.sav": []byte("my save"), "Mods/m.pak": []byte("my mod")} {
		write(t, filepath.Join(dir, filepath.FromSlash(name)), string(body))
	}
	return dir
}

func v1Intact(t *testing.T, dir string) {
	t.Helper()
	for name, want := range map[string]string{"Game.exe": "exe v1", "data/level1.pak": "level one v1", "data/old.pak": "gone in v2", "Saves/slot.sav": "my save", "Mods/m.pak": "my mod"} {
		if got := readFile(t, filepath.Join(dir, filepath.FromSlash(name))); got != want {
			t.Errorf("%s is %q, want %q", name, got, want)
		}
	}
	for _, gone := range []string{"data/new.pak", "readme.txt"} {
		if isFile(filepath.Join(dir, filepath.FromSlash(gone))) {
			t.Errorf("%s is still there", gone)
		}
	}
}

func deltaZip(t *testing.T, files map[string][]byte, prefix string, store bool) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		m := zip.Deflate
		if store {
			m = zip.Store
		}
		w, err := zw.CreateHeader(&zip.FileHeader{Name: prefix + n, Method: m})
		if err != nil {
			t.Fatal(err)
		}
		w.Write(files[n])
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// checkV2Staged: only what changed was written.
func checkPlan(t *testing.T, p *DeltaPlan, stage string) {
	t.Helper()
	got := map[string]bool{}
	for _, f := range p.Changed {
		got[f.Rel] = f.New
	}
	want := map[string]bool{"Game.exe": false, "data/level1.pak": false, "data/new.pak": true, "readme.txt": true}
	if fmt.Sprint(got) != fmt.Sprint(want) || p.Same != 1 || p.SameBytes != 200000 || p.Prefix != "Wrap/" || p.NewExe != "Game.exe" || !p.Full || p.Files != 5 {
		t.Fatalf("plan: %+v", p)
	}
	if isFile(DeltaStaged(stage, "data/big.pak")) {
		t.Error("an unchanged file was written")
	}
	if readFile(t, DeltaStaged(stage, "data/level1.pak")) != "level one v2" {
		t.Error("a changed file wasn't written right")
	}
}

// applyAndRevert applies a staged plan, checks the new version and the
// user's files, then rolls back and checks the old version is whole.
func applyAndRevert(t *testing.T, game, stage string, p *DeltaPlan) {
	t.Helper()
	extras, err := DeltaExtras(game, stage, p)
	if err != nil {
		t.Fatal(err)
	}
	var ex []string
	for _, e := range extras {
		ex = append(ex, e.Rel)
	}
	sort.Strings(ex)
	if strings.Join(ex, ",") != "Mods,Saves" {
		t.Errorf("extras: %q", ex)
	}
	remove, err := DeltaRemovals(game, stage, p, []string{"Saves", "Mods"})
	if err != nil || strings.Join(remove, ",") != "data/old.pak" {
		t.Fatalf("removals: %q %v", remove, err)
	}
	rec := filepath.Join(t.TempDir(), "rec")
	save := func(a *DeltaApplied) error { return SaveDeltaApplied(rec, a) }
	keep := game + ".lootnest-old"
	a, err := ApplyDelta(game, stage, keep, p.NewExe, p, remove, save)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range v2() {
		if got := readFile(t, filepath.Join(game, filepath.FromSlash(name))); got != string(want) {
			t.Errorf("after: %s differs", name)
		}
	}
	if readFile(t, filepath.Join(game, "Saves", "slot.sav")) != "my save" || readFile(t, filepath.Join(game, "Mods", "m.pak")) != "my mod" || isFile(filepath.Join(game, "data", "old.pak")) {
		t.Error("after: the user's files or a removal are wrong")
	}
	if readFile(t, filepath.Join(keep, "data", "old.pak")) != "gone in v2" || readFile(t, filepath.Join(keep, "Game.exe")) != "exe v1" || isFile(filepath.Join(keep, "data", "big.pak")) {
		t.Error("the kept old files are wrong")
	}
	if l, err := LoadDeltaApplied(rec); err != nil || l.State != "applied" || len(l.Replaced) != 2 || len(l.Added) != 2 || len(l.Removed) != 1 {
		t.Fatalf("record: %+v %v", l, err)
	}
	if err := RevertDelta(a, save); err != nil {
		t.Fatal(err)
	}
	v1Intact(t, game)
	if _, err := os.Lstat(keep); err == nil {
		t.Error("the keep folder is left")
	}
	if readFile(t, DeltaStaged(stage, "Game.exe")) != "exe v2" {
		t.Error("rolling back didn't put the new files back in the staging folder")
	}
	// Again: nothing changes.
	if err := RevertDelta(a, save); err != nil {
		t.Fatal(err)
	}
	v1Intact(t, game)
}

func TestRootFromNames(t *testing.T) {
	for _, c := range []struct {
		names         []string
		exe           string
		prefix, found string
		ok            bool
	}{
		{[]string{"Game.exe", "data/a.pak"}, "Game.exe", "", "Game.exe", true},
		{[]string{"Wrap/bin/Game.exe", "Wrap/data/a.pak"}, "bin/Game.exe", "Wrap/", "bin/Game.exe", true},
		{[]string{"Wrap/bin/game.EXE", "Wrap/data/a.pak"}, "bin/Game.exe", "Wrap/", "bin/game.EXE", true},
		{[]string{"A/B/Binaries/Win64/Game.exe", "A/B/x.pak"}, "Game.exe", "A/B/", "Binaries/Win64/Game.exe", true},
		{[]string{"data/a.pak"}, "Game.exe", "", "", false},
	} {
		p, e, ok := rootFromNames(c.names, c.exe)
		if p != c.prefix || e != c.found || ok != c.ok {
			t.Errorf("%v: %q %q %v", c.names, p, e, ok)
		}
	}
}

func TestDeltaZipLocal(t *testing.T) {
	game := gameV1(t)
	zpath := filepath.Join(t.TempDir(), "v2.zip")
	write(t, zpath, string(deltaZip(t, v2(), "Wrap/", false)))
	f, _ := os.Open(zpath)
	defer f.Close()
	st, _ := f.Stat()
	stage := game + ".lootnest-new"
	var steps []string
	p, err := StageDelta(context.Background(), DeltaSource{Name: "v2.zip", At: f, Size: st.Size()}, game, stage,
		DeltaOptions{ExeRel: "Game.exe", Progress: func(d DeltaProgress) { steps = append(steps, d.Step) }})
	if err != nil {
		t.Fatal(err)
	}
	checkPlan(t, p, stage)
	if steps[0] != "compare" || steps[len(steps)-1] != "fetch" {
		t.Errorf("progress: %q", steps)
	}
	if l, err := LoadDeltaPlan(stage); err != nil || len(l.Changed) != 4 {
		t.Fatalf("plan.json: %+v %v", l, err)
	}
	v1Intact(t, game) // staging changed nothing
	applyAndRevert(t, game, stage, p)

	// A second run with the files already staged writes nothing again.
	before, _ := os.Stat(DeltaStaged(stage, "data/level1.pak"))
	time.Sleep(20 * time.Millisecond)
	if _, err := StageDelta(context.Background(), DeltaSource{Name: "v2.zip", At: f, Size: st.Size()}, game, stage, DeltaOptions{ExeRel: "Game.exe"}); err != nil {
		t.Fatal(err)
	}
	if after, _ := os.Stat(DeltaStaged(stage, "data/level1.pak")); !after.ModTime().Equal(before.ModTime()) {
		t.Error("a staged file was written again")
	}

	// No room: refused before anything is written.
	stage2 := filepath.Join(t.TempDir(), "s2")
	_, err = StageDelta(context.Background(), DeltaSource{Name: "v2.zip", At: f, Size: st.Size()}, game, stage2,
		DeltaOptions{ExeRel: "Game.exe", Room: func(n int64) error { return fmt.Errorf("needs %d", n) }})
	if err == nil || !strings.Contains(err.Error(), "needs") {
		t.Fatalf("room: %v", err)
	}
	if ents, _ := os.ReadDir(filepath.Join(stage2, deltaFiles)); len(ents) != 0 {
		t.Error("files were written without room")
	}
}

// A patch (no exe, a few files): what it has goes in, nothing is removed.
func TestDeltaPatch(t *testing.T) {
	game := gameV1(t)
	zpath := filepath.Join(t.TempDir(), "patch.zip")
	write(t, zpath, string(deltaZip(t, map[string][]byte{"data/level1.pak": []byte("level one v2")}, "Night Tale Patch/", false)))
	f, _ := os.Open(zpath)
	defer f.Close()
	st, _ := f.Stat()
	stage := game + ".lootnest-new"
	p, err := StageDelta(context.Background(), DeltaSource{Name: "patch.zip", At: f, Size: st.Size()}, game, stage, DeltaOptions{ExeRel: "Game.exe"})
	if err != nil {
		t.Fatal(err)
	}
	if p.Full || p.NewExe != "" || p.Prefix != "Night Tale Patch/" || len(p.Changed) != 1 {
		t.Fatalf("plan: %+v", p)
	}
	if r, err := DeltaRemovals(game, stage, p, nil); err != nil || len(r) != 0 {
		t.Fatalf("a patch removed: %q %v", r, err)
	}
	a, err := ApplyDelta(game, stage, game+".lootnest-old", "Game.exe", p, nil, func(*DeltaApplied) error { return nil })
	if err != nil || readFile(t, filepath.Join(game, "data", "level1.pak")) != "level one v2" || readFile(t, filepath.Join(game, "data", "old.pak")) != "gone in v2" {
		t.Fatalf("apply: %v", err)
	}
	if err := RevertDelta(a, func(*DeltaApplied) error { return nil }); err != nil {
		t.Fatal(err)
	}
	v1Intact(t, game)
}

func Test7zDelta(t *testing.T) {
	for _, kind := range []string{"solid", "nonsolid"} {
		src := filepath.Join("testdata", "delta-"+kind+".7z")
		f, err := os.Open(src)
		if err != nil {
			t.Fatal(err)
		}
		st, _ := f.Stat()
		game := gameV1(t)
		stage := game + ".lootnest-new"
		p, err := StageDelta(context.Background(), DeltaSource{Name: "x.7z", At: f, Size: st.Size()}, game, stage, DeltaOptions{ExeRel: "Game.exe"})
		f.Close()
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		checkPlan(t, p, stage)
		applyAndRevert(t, game, stage, p)

		// The same over http, read in parts.
		body, _ := os.ReadFile(src)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeContent(w, r, "x.7z", time.Time{}, bytes.NewReader(body))
		}))
		rf, err := OpenRemote(context.Background(), srv.Client(), srv.URL+"/x.7z")
		if err != nil {
			t.Fatal(err)
		}
		game2 := gameV1(t)
		p2, err := StageDelta(context.Background(), DeltaSource{Name: "x.7z", At: rf, Size: rf.Size()}, game2, game2+".lootnest-new", DeltaOptions{ExeRel: "Game.exe"})
		srv.Close()
		if err != nil {
			t.Fatalf("%s remote: %v", kind, err)
		}
		checkPlan(t, p2, game2+".lootnest-new")
	}
}

// Over http, a .zip is read for its list and the changed files only.
func TestDeltaRemoteZipFetchesOnlyChanges(t *testing.T) {
	old := remoteBlock
	remoteBlock = 64 << 10
	defer func() { remoteBlock = old }()
	rng := rand.New(rand.NewSource(1))
	files := map[string][]byte{}
	for i := 0; i < 4; i++ {
		b := make([]byte, 1<<20)
		rng.Read(b)
		files[fmt.Sprintf("data/chunk%d.pak", i)] = b
	}
	files["Game.exe"] = []byte("exe v1")
	game := filepath.Join(t.TempDir(), "G")
	for n, b := range files {
		write(t, filepath.Join(game, filepath.FromSlash(n)), string(b))
	}
	newFiles := map[string][]byte{}
	for n, b := range files {
		newFiles[n] = b
	}
	newFiles["Game.exe"] = []byte("exe v2")
	archive := deltaZip(t, newFiles, "", true)
	var served atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cw := &countWriter{ResponseWriter: w, n: &served}
		http.ServeContent(cw, r, "v2.zip", time.Time{}, bytes.NewReader(archive))
	}))
	defer srv.Close()
	rf, err := OpenRemote(context.Background(), srv.Client(), srv.URL+"/v2.zip")
	if err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(t.TempDir(), "stage")
	p, err := StageDelta(context.Background(), DeltaSource{Name: "v2.zip", At: rf, Size: rf.Size()}, game, stage, DeltaOptions{ExeRel: "Game.exe"})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Changed) != 1 || p.Changed[0].Rel != "Game.exe" || p.Same != 4 {
		t.Fatalf("plan: %+v", p)
	}
	if got := served.Load(); got > int64(len(archive))/4 || rf.Fetched() > int64(len(archive))/4 {
		t.Errorf("fetched %d of %d bytes (served %d)", rf.Fetched(), len(archive), got)
	}
	// A server that sends only whole files is said so.
	whole := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(archive) }))
	defer whole.Close()
	if _, err := OpenRemote(context.Background(), whole.Client(), whole.URL+"/v2.zip"); !errors.Is(err, ErrNoRanges) {
		t.Errorf("no ranges: %v", err)
	}
}

type countWriter struct {
	http.ResponseWriter
	n *atomic.Int64
}

func (c *countWriter) Write(b []byte) (int, error) {
	n, err := c.ResponseWriter.Write(b)
	c.n.Add(int64(n))
	return n, err
}

// ---- a minimal RAR5 writer (stored files) for the .rar path ----

func rarVint(v uint64) []byte {
	var b []byte
	for {
		c := byte(v & 0x7f)
		v >>= 7
		if v != 0 {
			b = append(b, c|0x80)
			continue
		}
		return append(b, c)
	}
}

// rarHeader is a block: CRC32, size, then body (type, flags and the rest).
func rarHeader(body []byte) []byte {
	h := append(rarVint(uint64(len(body))), body...)
	out := make([]byte, 4, 4+len(h))
	binary.LittleEndian.PutUint32(out, crc32.ChecksumIEEE(h))
	return append(out, h...)
}

type rarPart struct {
	name       string
	data       []byte
	fileSize   int64
	crc        uint32
	from, more bool // continues from the volume before, into the next
}

func rarVolume(parts []rarPart, volume, last bool, num int) []byte {
	out := []byte{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07, 0x01, 0x00}
	var arch []byte
	arch = append(arch, rarVint(1)...) // main header
	arch = append(arch, rarVint(0)...) // header flags
	af := uint64(0)
	if volume {
		af |= 0x0001
		if num > 0 {
			af |= 0x0002
		}
	}
	arch = append(arch, rarVint(af)...)
	if af&0x0002 != 0 {
		arch = append(arch, rarVint(uint64(num))...)
	}
	out = append(out, rarHeader(arch)...)
	for _, p := range parts {
		hf := uint64(0x0002) // data area
		if p.from {
			hf |= 0x0008
		}
		if p.more {
			hf |= 0x0010
		}
		var b []byte
		b = append(b, rarVint(2)...)
		b = append(b, rarVint(hf)...)
		b = append(b, rarVint(uint64(len(p.data)))...)
		b = append(b, rarVint(0x0004)...) // CRC32 present
		b = append(b, rarVint(uint64(p.fileSize))...)
		b = append(b, rarVint(0x20)...) // attributes
		crc := make([]byte, 4)
		binary.LittleEndian.PutUint32(crc, p.crc)
		b = append(b, crc...)
		b = append(b, rarVint(0)...) // version 0, stored
		b = append(b, rarVint(0)...) // Windows
		b = append(b, rarVint(uint64(len(p.name)))...)
		b = append(b, p.name...)
		out = append(out, rarHeader(b)...)
		out = append(out, p.data...)
	}
	end := append(rarVint(5), rarVint(0)...)
	ef := uint64(0)
	if volume && !last {
		ef = 1
	}
	out = append(out, rarHeader(append(end, rarVint(ef)...))...)
	return out
}

// makeRar writes files (in name order, under prefix) into one volume, or
// two with the last file split between them.
func makeRar(files map[string][]byte, prefix string, volumes int) [][]byte {
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	var parts []rarPart
	for _, n := range names {
		b := files[n]
		parts = append(parts, rarPart{name: prefix + n, data: b, fileSize: int64(len(b)), crc: crc32.ChecksumIEEE(b)})
	}
	if volumes == 1 {
		return [][]byte{rarVolume(parts, false, true, 0)}
	}
	lastF := parts[len(parts)-1]
	half := len(lastF.data) / 2
	a := rarPart{name: lastF.name, data: lastF.data[:half], fileSize: lastF.fileSize, crc: crc32.ChecksumIEEE(lastF.data[:half]), more: true}
	b := rarPart{name: lastF.name, data: lastF.data[half:], fileSize: lastF.fileSize, crc: lastF.crc, from: true}
	return [][]byte{rarVolume(append(parts[:len(parts)-1:len(parts)-1], a), true, false, 0), rarVolume([]rarPart{b}, true, true, 1)}
}

func TestDeltaRarStreamed(t *testing.T) {
	files := v2()
	for _, vols := range []int{1, 2} {
		archive := makeRar(files, "Wrap/", vols)
		names := []string{"NightTale.rar"}
		if vols == 2 {
			names = []string{"NightTale.part1.rar", "NightTale.part2.rar"}
		}
		var mu sync.Mutex
		dropped := false
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for i, n := range names {
				if r.URL.Path != "/"+n {
					continue
				}
				body := archive[i]
				mu.Lock()
				drop := !dropped && r.Header.Get("Range") == "" && i == 0
				dropped = dropped || drop
				mu.Unlock()
				if drop {
					// Half of it, then the connection goes.
					w.Header().Set("Content-Length", fmt.Sprint(len(body)))
					w.Write(body[:len(body)/2])
					w.(http.Flusher).Flush()
					conn, _, _ := w.(http.Hijacker).Hijack()
					conn.Close()
					return
				}
				http.ServeContent(w, r, n, time.Time{}, bytes.NewReader(body))
				return
			}
			http.NotFound(w, r)
		}))
		urls := map[string]string{}
		for _, n := range names {
			urls[n] = srv.URL + "/" + n
		}
		rv := &RemoteVolumes{Ctx: context.Background(), Client: srv.Client(), URLs: urls}
		game := gameV1(t)
		stage := game + ".lootnest-new"
		p, err := StageDelta(context.Background(), DeltaSource{Name: names[0], FS: rv}, game, stage, DeltaOptions{ExeRel: "Game.exe"})
		srv.Close()
		if err != nil {
			t.Fatalf("%d volume(s): %v", vols, err)
		}
		if !dropped {
			t.Error("the dropped connection wasn't tried")
		}
		checkPlan(t, p, stage)
		if rv.Read() < int64(len(archive[0])) {
			t.Errorf("read %d bytes", rv.Read())
		}
		applyAndRevert(t, game, stage, p)
	}
}

// A failure after any move puts the game back as it was; so does a crash,
// from the record alone.
func TestDeltaApplyFailures(t *testing.T) {
	defer func() { deltaAfterMove = nil }()
	for stop := 1; stop <= 7; stop++ {
		game := gameV1(t)
		zpath := filepath.Join(t.TempDir(), "v2.zip")
		write(t, zpath, string(deltaZip(t, v2(), "Wrap/", false)))
		f, _ := os.Open(zpath)
		st, _ := f.Stat()
		stage := game + ".lootnest-new"
		p, err := StageDelta(context.Background(), DeltaSource{Name: "v2.zip", At: f, Size: st.Size()}, game, stage, DeltaOptions{ExeRel: "Game.exe"})
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		remove, _ := DeltaRemovals(game, stage, p, []string{"Saves", "Mods"})
		rec := filepath.Join(t.TempDir(), "rec")
		save := func(a *DeltaApplied) error { return SaveDeltaApplied(rec, a) }
		deltaAfterMove = func(n int) error {
			if n == stop {
				return errors.New("the disk said no")
			}
			return nil
		}
		if _, err := ApplyDelta(game, stage, game+".lootnest-old", "Game.exe", p, remove, save); err == nil || !strings.Contains(err.Error(), "the disk said no") {
			t.Fatalf("stop %d: %v", stop, err)
		}
		v1Intact(t, game)
		if l, _ := LoadDeltaApplied(rec); l == nil || l.State != "undone" {
			t.Errorf("stop %d: record %+v", stop, l)
		}

		// A crash: the moves stop, nothing runs after; the next start reads
		// the record and puts it back.
		type crash struct{}
		deltaAfterMove = func(n int) error {
			if n == stop {
				panic(crash{})
			}
			return nil
		}
		func() {
			defer func() { recover() }()
			ApplyDelta(game, stage, game+".lootnest-old", "Game.exe", p, remove, save)
		}()
		deltaAfterMove = nil
		l, err := LoadDeltaApplied(rec)
		if err != nil || l.State != "applying" {
			t.Fatalf("stop %d: after a crash: %+v %v", stop, l, err)
		}
		if err := RevertDelta(l, save); err != nil {
			t.Fatal(err)
		}
		v1Intact(t, game)
	}
}
