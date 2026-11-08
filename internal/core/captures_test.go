package core

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeCapture(t *testing.T, p string, at time.Time) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("capture"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, at, at); err != nil {
		t.Fatal(err)
	}
}

// capturePC is a PC with captures from every app, as Windows lays them out.
func capturePC(t *testing.T) (CaptureEnv, map[string]string) {
	root := t.TempDir()
	env := CaptureEnv{Videos: filepath.Join(root, "Videos"), Pictures: filepath.Join(root, "Pictures"), SteamPath: filepath.Join(root, "Steam"),
		Games: []string{"Baldur's Gate 3", "Hades II", "Celeste"}}
	at := time.Date(2026, 11, 8, 21, 4, 13, 0, time.UTC)
	files := map[string]string{
		"steam":    filepath.Join(env.SteamPath, "userdata", "123", "760", "remote", "1086940", "screenshots", "20261108210413_1.jpg"),
		"steamThb": filepath.Join(env.SteamPath, "userdata", "123", "760", "remote", "1086940", "screenshots", "thumbnails", "20261108210413_1.jpg"),
		"gamebar":  filepath.Join(env.Videos, "Captures", "Hades II 2026-11-08 21-04-13.png"),
		"gamebar2": filepath.Join(env.Videos, "Captures", "Celeste 11_8_2026 9_04_13 PM.mp4"),
		"nvidia":   filepath.Join(env.Videos, "Celeste", "Celeste 2026.11.08 - 21.04.13.01.mp4"),
		"amd":      filepath.Join(env.Videos, "Radeon ReLive", "Hades II", "Hades II_replay_2026.11.08-21.04.mp4"),
		"obs":      filepath.Join(env.Videos, "2026-11-08 21-04-13.mkv"),
		"other":    filepath.Join(env.Videos, "Holiday.mp4"),
		"notes":    filepath.Join(env.Videos, "Captures", "notes.txt"),
		"own":      filepath.Join(env.Pictures, "Lootnest", "Screenshots", "Baldur's Gate 3", "2026-11-08 21-04-13.png"),
		"unknown":  filepath.Join(env.Videos, "Captures", "Desktop 2026-11-01 10-00-00.png"),
	}
	for _, p := range files {
		writeCapture(t, p, at)
	}
	writeCapture(t, files["unknown"], at.AddDate(0, 0, -7)) // in no session
	return env, files
}

func TestGameBarGame(t *testing.T) {
	for in, want := range map[string]string{
		"Hades II 2026-11-08 21-04-13.png":        "Hades II",
		"Baldur's Gate 3 2026-11-08 21-04-13.mp4": "Baldur's Gate 3",
		"Celeste 11_8_2026 9_04_13 PM.mp4":        "Celeste",
		"Hades II 2026-11-08 21-04-13 (2).png":    "Hades II",
		"2026-11-08 21-04-13.mkv":                 "",
		"Holiday.mp4":                             "",
	} {
		if got := GameBarGame(in); got != want {
			t.Errorf("GameBarGame(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCaptureSourcesAndScan(t *testing.T) {
	env, files := capturePC(t)
	src := map[string]int{}
	for _, s := range CaptureSources(env) {
		src[s.Name]++
	}
	// NVIDIA's folder is told by the game's name; Captures and Radeon ReLive aren't games.
	if src[CapGameBar] != 1 || src[CapSteam] != 1 || src[CapNVIDIA] != 1 || src[CapAMD] != 1 || src[CapOBS] != 1 || src[CapLootnest] != 1 {
		t.Fatalf("sources %v", src)
	}
	got := map[string]Capture{}
	scanned := ScanCaptures(CaptureSources(env), nil)
	for _, c := range scanned {
		got[c.Path] = c
	}
	if len(got) != 8 {
		t.Fatalf("scanned %d: %v", len(got), got)
	}
	for _, k := range []string{"steamThb", "other", "notes"} {
		if _, ok := got[files[k]]; ok {
			t.Errorf("%s shouldn't be a capture", k)
		}
	}
	if c := got[files["steam"]]; c.AppID != 1086940 || c.Source != CapSteam || c.Kind != "picture" || c.Size != 7 {
		t.Errorf("steam %+v", c)
	}
	if c := got[files["gamebar"]]; c.Hint != "Hades II" || c.Source != CapGameBar {
		t.Errorf("game bar %+v", c)
	}
	if c := got[files["nvidia"]]; c.Hint != "Celeste" || c.Kind != "clip" || c.Source != CapNVIDIA {
		t.Errorf("nvidia %+v", c)
	}
	if c := got[files["obs"]]; c.Source != CapOBS || c.Hint != "" {
		t.Errorf("obs %+v", c)
	}
	// What's in the index as it was isn't read again, so it doesn't count
	// against the 20,000; a file written again is.
	writeCapture(t, files["obs"], time.Now())
	if again := ScanCaptures(CaptureSources(env), scanned); len(again) != 1 || again[0].Path != files["obs"] {
		t.Errorf("indexed scanned again: %v", again)
	}
}

func TestMatchCapture(t *testing.T) {
	env, files := capturePC(t)
	games := []CaptureGame{{Key: "game:bg3", Name: "Baldur's Gate 3", AppID: 1086940}, {Key: "found:hades2", Name: "Hades™ II"}, {Key: "found:celeste", Name: "Celeste"}}
	at := time.Date(2026, 11, 8, 21, 4, 13, 0, time.UTC)
	games = append(games, CaptureGame{Key: "found:obsgame", Name: "Untitled"})
	sessions := []SessionRecord{{Game: "found:obsgame", Start: at.Add(-time.Hour), End: at.Add(-30 * time.Second)}}
	want := map[string]string{"steam": "game:bg3", "gamebar": "found:hades2", "gamebar2": "found:celeste", "nvidia": "found:celeste",
		"amd": "found:hades2", "obs": "found:obsgame", "own": "game:bg3", "unknown": ""}
	for _, c := range ScanCaptures(CaptureSources(env), nil) {
		for k, p := range files {
			if p == c.Path {
				if got := MatchCapture(c, games, sessions); got != want[k] {
					t.Errorf("%s matched %q, want %q", k, got, want[k])
				}
			}
		}
	}
	// Outside the session and its minute either side: unmatched.
	late := Capture{Path: "x.mkv", Source: CapOBS, At: at.Add(2 * time.Minute)}
	if got := MatchCapture(late, games, sessions); got != "" {
		t.Errorf("late clip matched %q", got)
	}
	// A session of a game no longer in the Library matches nothing.
	if got := MatchCapture(Capture{Path: "y.mkv", At: at}, games[:3], sessions); got != "" {
		t.Errorf("clip matched a game gone from the Library: %q", got)
	}
}

// tree is every file and folder under root, with what each is, for checking
// nothing there changed.
func tree(t *testing.T, root string) map[string]string {
	out := map[string]string{}
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// A folder counts by being there: NTFS updates a folder's time and
		// size when a file in it is rewritten, as this test does itself.
		if d.IsDir() {
			out[p] = "dir"
			return nil
		}
		fi, _ := d.Info()
		out[p] = fmt.Sprint(fi.Mode(), fi.ModTime(), fi.Size())
		return nil
	})
	return out
}

func TestUpdateCapturesIndex(t *testing.T) {
	env, files := capturePC(t)
	store := &Store{Dir: t.TempDir()}
	root := filepath.Dir(env.Videos)
	games := []CaptureGame{{Key: "game:bg3", Name: "Baldur's Gate 3", AppID: 1086940}}
	list, err := store.UpdateCaptures(ScanCaptures(CaptureSources(env), nil), games, nil)
	if err != nil || len(list) != 8 {
		t.Fatalf("%d captures, %v", len(list), err)
	}
	if err := store.AssignCapture(files["unknown"], "game:bg3"); err != nil {
		t.Fatal(err)
	}
	if err := store.AssignCapture(filepath.Join(root, "nothing.png"), "game:bg3"); err == nil {
		t.Error("assigned a capture that isn't in the index")
	}
	// A scan that misses a file keeps it while it's there, as it was; a
	// deleted file drops out; a match by hand stays.
	if err := os.Remove(files["obs"]); err != nil {
		t.Fatal(err)
	}
	before := tree(t, root)
	list, err = store.UpdateCaptures(nil, games, nil)
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string]Capture{}
	for _, c := range store.LoadCaptures() {
		byPath[c.Path] = c
	}
	if len(list) != 7 || len(byPath) != 7 {
		t.Fatalf("index has %d, %d", len(list), len(byPath))
	}
	if _, ok := byPath[files["obs"]]; ok {
		t.Error("a deleted capture stayed in the index")
	}
	if c := byPath[files["unknown"]]; c.Key != "game:bg3" || !c.Manual {
		t.Errorf("assigned capture %+v", c)
	}
	if c := byPath[files["steam"]]; c.Key != "game:bg3" {
		t.Errorf("steam capture %+v", c)
	}
	// A file written again is a new capture: matched afresh.
	writeCapture(t, files["unknown"], time.Now())
	list, _ = store.UpdateCaptures(ScanCaptures(CaptureSources(env), store.LoadCaptures()), games, nil)
	for _, c := range list {
		if c.Path == files["unknown"] && (c.Manual || c.Key != "") {
			t.Errorf("changed capture kept its match: %+v", c)
		}
	}
	// A match by hand to a game gone from the Library is matched again;
	// "not a game" stays.
	if err := store.AssignCapture(files["unknown"], CaptureNoGame); err != nil {
		t.Fatal(err)
	}
	if err := store.AssignCapture(files["gamebar"], "found:gone"); err != nil {
		t.Fatal(err)
	}
	for _, c := range mustUpdate(t, store, games) {
		if c.Path == files["unknown"] && (c.Key != CaptureNoGame || !c.Manual) || c.Path == files["gamebar"] && (c.Key != "" || c.Manual) {
			t.Errorf("after a scan: %+v", c)
		}
	}
	// The scan and the index never wrote among the captures.
	before[files["unknown"]] = tree(t, root)[files["unknown"]]
	after := tree(t, root)
	if len(after) != len(before) {
		t.Fatalf("files under the captures changed: %d, was %d", len(after), len(before))
	}
	for p, v := range before {
		if after[p] != v {
			t.Errorf("%s changed", p)
		}
	}
	if es, _ := os.ReadDir(store.Dir); len(es) != 1 || es[0].Name() != "captures.json" {
		t.Errorf("data folder has %v", es)
	}
}

func mustUpdate(t *testing.T, s *Store, games []CaptureGame) []Capture {
	t.Helper()
	list, err := s.UpdateCaptures(nil, games, nil)
	if err != nil {
		t.Fatal(err)
	}
	return list
}
