package core

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// The whole way, with the real plugin and Lootnest's own pieces wired as the
// app wires them: a feed on a local server lists a newer build of a
// made-up game; the plugin offers it; Update downloads the archive into a
// staging folder, checks it, opens it, copies the saves, replaces the
// game's folder and keeps the old one; the saves are intact; Roll back puts
// the old folder back with the saves intact again.
func TestSourcesUpdateEndToEnd(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "Games", "Night Tale")
	outside := filepath.Join(root, "AppData", "NightTale")
	for name, body := range map[string]string{
		"bin/NightTale.exe": "exe 1.2.0", "data/pak0.pak": "data 1.2.0", "data/old-only.pak": "gone in 1.3.0",
		"Saves/slot1.sav": "in-folder save", "Saves/sub/slot2.sav": "another in-folder save",
	} {
		write(t, filepath.Join(dir, filepath.FromSlash(name)), body)
	}
	write(t, filepath.Join(outside, "profile.dat"), "a save outside the folder")
	write(t, filepath.Join(outside, "settings.cfg"), "settings outside the folder")

	// The new build, as a .zip with a wrapping folder, and a traversal entry
	// that must never land anywhere.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"Night.Tale.v1.3.0/bin/NightTale.exe": "exe 1.3.0", "Night.Tale.v1.3.0/data/pak0.pak": "data 1.3.0", "Night.Tale.v1.3.0/data/pak1.pak": "new in 1.3.0",
		"Night.Tale.v1.3.0/Saves/slot1.sav": "a save the build ships, which must not replace the user's",
	} {
		w, _ := zw.Create(name)
		w.Write([]byte(body))
	}
	zw.Close()
	archive := buf.Bytes()
	sum := sha256.Sum256(archive)

	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/feed.json":
			fmt.Fprintf(w, `{"lootnest": 1, "name": "Fixture feed", "downloads": [
				{"title": "Night Tale v1.3.0", "uris": ["%s/files/nt130.zip"], "uploadDate": %q, "fileSize": %d, "version": "1.3.0", "sha256": %q}]}`,
				"http://"+r.Host, time.Now().Add(-time.Hour).UTC().Format(time.RFC3339), len(archive), hex.EncodeToString(sum[:]))
		case "/files/nt130.zip":
			http.ServeContent(w, r, "nt130.zip", time.Time{}, bytes.NewReader(archive))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// Lootnest's side, as app/fullupdate.go does it.
	const key = "dir:night tale"
	game := ScriptGame{Key: key, Name: "Night Tale", Source: "hydra", Folder: "Night Tale", Version: "1.2.0"}
	saves := func() []SaveLocation {
		return []SaveLocation{{Path: "<base>/Saves", Resolved: filepath.Join(dir, "Saves")}, {Path: "<home>/AppData/NightTale", Resolved: outside}}
	}
	resolve := func(p string) (string, error) {
		p = strings.Replace(p, "<base>", dir, 1)
		return filepath.FromSlash(strings.Replace(p, "<home>", root, 1)), nil
	}
	backups := &Cloud{Root: filepath.Join(root, "Lootnest", "saves-before-update"), Computer: "TEST-PC"}
	stateDir := filepath.Join(root, "Lootnest", "fullupdates", "nt")
	staging := dir + ".lootnest-new"
	q := NewDownloadQueue(filepath.Join(root, "Lootnest", "downloads.json"), GuardedClient(true), nil)
	t.Cleanup(q.Close)
	var logs []string
	var offers []ScriptOffer
	var started map[string]any
	var h *ScriptHost
	hooks := FullUpdateHooks{
		Save: func(u *FullUpdate) error { return SaveFullUpdate(stateDir, u) },
		Log:  func(s string) { logs = append(logs, s) },
		Backup: func() (string, error) {
			hash, err := SavesHash(saves())
			if err != nil {
				return "", err
			}
			v, err := backups.SaveGameVersion("night-tale", "Before the update", saves(), hash, 5)
			if err != nil {
				return "", err
			}
			return v.ID, nil
		},
		RestoreSaves: func(id string) error {
			_, err := backups.RestoreGameVersion("night-tale", id, saves(), resolve, filepath.Join(root, "restore-tmp"))
			return err
		},
	}
	h, _ = sourcesPlugin(t, ScriptServices{
		Games:  func() []ScriptGame { return []ScriptGame{game} },
		Offers: func(l []ScriptOffer) { mu.Lock(); offers = l; mu.Unlock() },
		Notify: func(string) {},
		Update: func(op string, raw json.RawMessage) (any, error) {
			if op != "start" {
				return nil, fmt.Errorf("unexpected %s", op)
			}
			_ = json.Unmarshal(raw, &started)
			var list []Download
			for i, f := range started["files"].([]any) {
				f := f.(map[string]any)
				list = append(list, Download{ID: fmt.Sprint("d", i), Job: "j1", URL: f["url"].(string), Dst: filepath.Join(staging, "dl", SafeDownloadName(f["name"].(string))),
					Size: int64(f["size"].(float64)), SHA256: f["sha256"].(string)})
			}
			q.Add(list...)
			return map[string]any{"job": "j1"}, nil
		},
	})
	act := func(arg map[string]any) {
		t.Helper()
		var r actResult
		out, err := h.Call("action", arg)
		if err != nil {
			t.Fatal(err)
		}
		if _ = json.Unmarshal(out, &r); r.Error != "" {
			t.Fatal(r.Error)
		}
	}
	firstOffer := func() *ScriptOffer {
		mu.Lock()
		defer mu.Unlock()
		if len(offers) == 0 {
			return nil
		}
		return &offers[0]
	}
	waitFor := func(what string, ok func() bool) {
		t.Helper()
		for end := time.Now().Add(15 * time.Second); !ok(); time.Sleep(20 * time.Millisecond) {
			if time.Now().After(end) {
				t.Fatalf("never happened: %s", what)
			}
		}
	}

	// 1. The feed is read and the update is offered.
	act(map[string]any{"id": "notice"})
	act(map[string]any{"id": "add", "values": map[string]any{"url": srv.URL + "/feed.json"}})
	waitFor("the offer", func() bool { return firstOffer() != nil })
	o := firstOffer()
	if o.Game != key || o.Version != "1.3.0" || o.Have != "1.2.0" || o.Source != "Fixture feed" || o.Size != int64(len(archive)) {
		t.Fatalf("offer: %+v", o)
	}

	// 2. Update: the plugin hands Lootnest the file; it's downloaded into the
	// staging folder, never over the game, and checked against the feed's hash.
	out, err := h.Call("offer", map[string]any{"id": o.ID, "action": "update"})
	if err != nil || strings.Contains(string(out), `"error"`) {
		t.Fatalf("update: %s %v", out, err)
	}
	waitFor("the download", func() bool { return q.Job("j1").Ready == 1 })
	got, err := os.ReadFile(filepath.Join(staging, "dl", "nt130.zip"))
	if err != nil || !bytes.Equal(got, archive) {
		t.Fatalf("the downloaded file: %v", err)
	}
	if readFile(t, filepath.Join(dir, "bin", "NightTale.exe")) != "exe 1.2.0" {
		t.Fatal("the game changed while only downloading")
	}

	// 3. Opened in staging; it's a whole build; the plan lists what's kept.
	if _, err := StageUpdateMax(filepath.Join(staging, "dl", "nt130.zip"), filepath.Join(staging, "build"), 1<<30); err != nil {
		t.Fatal(err)
	}
	build, newExe, err := FindBuild(filepath.Join(staging, "build"), "bin/NightTale.exe")
	if err != nil || newExe != "bin/NightTale.exe" {
		t.Fatalf("FindBuild: %q %v", newExe, err)
	}
	plan, err := PlanFullUpdate(dir, build, []string{"Saves"}, nil)
	if err != nil || len(plan.Keep) != 1 || plan.Keep[0].Rel != "Saves" {
		t.Fatalf("plan: %+v %v", plan, err)
	}

	// 4. The user says yes: saves copied, folder swapped, checked.
	u := &FullUpdate{ID: "j1", Key: key, Name: "Night Tale", Dir: dir, Exe: "bin/NightTale.exe", Title: o.Title, Version: o.Version, Staging: staging, Build: build, NewExe: newExe,
		Keep: []string{"Saves"}, State: FUStaged}
	if err := RunFullUpdate(u, hooks); err != nil {
		t.Fatalf("the update: %v\n%s", err, strings.Join(logs, "\n"))
	}
	savesIntact := func(where string) {
		t.Helper()
		for p, want := range map[string]string{
			filepath.Join(dir, "Saves", "slot1.sav"):        "in-folder save",
			filepath.Join(dir, "Saves", "sub", "slot2.sav"): "another in-folder save",
			filepath.Join(outside, "profile.dat"):           "a save outside the folder",
			filepath.Join(outside, "settings.cfg"):          "settings outside the folder",
		} {
			if got := readFile(t, p); got != want {
				t.Errorf("%s: %s is %q, want %q", where, p, got, want)
			}
		}
	}
	savesIntact("after the update")
	for name, want := range map[string]string{"bin/NightTale.exe": "exe 1.3.0", "data/pak0.pak": "data 1.3.0", "data/pak1.pak": "new in 1.3.0"} {
		if got := readFile(t, filepath.Join(dir, filepath.FromSlash(name))); got != want {
			t.Errorf("%s is %q, want %q", name, got, want)
		}
	}
	if isFile(filepath.Join(dir, "data", "old-only.pak")) {
		t.Error("a file the new build dropped is still there")
	}
	// The old folder is kept whole, and the copy of the saves has every location.
	if readFile(t, filepath.Join(u.Old, "bin", "NightTale.exe")) != "exe 1.2.0" || readFile(t, filepath.Join(u.Old, "data", "old-only.pak")) != "gone in 1.3.0" {
		t.Error("the old folder isn't whole")
	}
	vs := backups.GameVersions("night-tale")
	if len(vs) != 1 || vs[0].ID != u.Saves || len(vs[0].Paths) != 2 || vs[0].Size == 0 {
		t.Fatalf("the saves' copy: %+v", vs)
	}
	if st, _ := LoadFullUpdate(stateDir); st == nil || st.State != FUKept {
		t.Fatalf("state on disk: %+v", st)
	}
	if len(logs) < 5 {
		t.Errorf("the log has %d lines", len(logs))
	}

	// The plugin hears it's in place: the game isn't offered that build again.
	game.Version = "" // as the next scan would find: the folder's name says nothing
	if _, err := h.Call("update", map[string]any{"game": key, "offer": o.ID, "state": "kept"}); err != nil {
		t.Fatal(err)
	}
	var d detected
	pluginTest(t, h, map[string]any{"what": "detect"}, &d)
	if len(d.Offers) != 0 {
		t.Errorf("offered again: %+v", d.Offers)
	}

	// 5. Played a little, then Roll back: the saves are copied again first,
	// the old folder is back, and every save is still what the user had.
	time.Sleep(1100 * time.Millisecond) // a copy is named to the second
	if err := RollbackFullUpdate(u, hooks); err != nil {
		t.Fatal(err)
	}
	savesIntact("after the roll back")
	if readFile(t, filepath.Join(dir, "bin", "NightTale.exe")) != "exe 1.2.0" || readFile(t, filepath.Join(dir, "data", "old-only.pak")) != "gone in 1.3.0" || isFile(filepath.Join(dir, "data", "pak1.pak")) {
		t.Error("the old build isn't back as it was")
	}
	if len(backups.GameVersions("night-tale")) != 2 {
		t.Error("the saves weren't copied before the roll back")
	}
	if !isFile(filepath.Join(u.Aside, "bin", "NightTale.exe")) {
		t.Error("the new build wasn't kept aside for the Recycle Bin")
	}
	game.Version = "1.2.0"
	if _, err := h.Call("update", map[string]any{"game": key, "offer": o.ID, "state": "rolledback"}); err != nil {
		t.Fatal(err)
	}
	pluginTest(t, h, map[string]any{"what": "detect"}, &d)
	if len(d.Offers) != 1 || d.Offers[0].Version != "1.3.0" {
		t.Errorf("after the roll back the update should be offered again: %+v", d.Offers)
	}

	// 6. Again, and this time it fails after the swap: the game and every
	// save are as they were, and the copy of the saves is put back.
	_ = os.RemoveAll(u.Aside)
	if _, err := StageUpdateMax(filepath.Join(staging, "dl", "nt130.zip"), filepath.Join(staging, "build2"), 1<<30); err != nil {
		t.Fatal(err)
	}
	build, newExe, _ = FindBuild(filepath.Join(staging, "build2"), "bin/NightTale.exe")
	u2 := &FullUpdate{ID: "j2", Key: key, Name: "Night Tale", Dir: dir, Exe: "bin/NightTale.exe", Staging: staging, Build: build, NewExe: newExe, Keep: []string{"Saves"}, State: FUStaged}
	failing := hooks
	failing.After = func(state string) error {
		if state == FURestore {
			write(t, filepath.Join(outside, "profile.dat"), "damaged on the way")
			return fmt.Errorf("the disk is full")
		}
		return nil
	}
	if err := RunFullUpdate(u2, failing); err == nil || u2.State != FUFailed {
		t.Fatalf("a failing update: %v %s", err, u2.State)
	}
	savesIntact("after a failed update")
	if readFile(t, filepath.Join(dir, "bin", "NightTale.exe")) != "exe 1.2.0" || isDir(dir+".lootnest-old") {
		t.Error("a failed update didn't leave the game as it was")
	}
}
