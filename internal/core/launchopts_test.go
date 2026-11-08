package core

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCleanLaunchOptions(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "OBS.exe")
	if err := os.WriteFile(exe, []byte("MZ"), 0o644); err != nil {
		t.Fatal(err)
	}
	o, err := CleanLaunchOptions(LaunchOptions{
		WorkDir:     " " + dir + " ",
		CloseApps:   []CloseApp{{Name: " Discord "}, {Name: "discord.exe"}, {Name: "chrome.exe", Force: true}},
		StartBefore: []Program{{Path: exe, Args: " --startreplaybuffer "}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if o.WorkDir != dir || !reflect.DeepEqual(o.CloseApps, []CloseApp{{Name: "Discord.exe"}, {Name: "chrome.exe", Force: true}}) ||
		o.StartBefore[0].Args != "--startreplaybuffer" || !o.HasSteps() || o.Empty() {
		t.Fatalf("cleaned = %+v", o)
	}
	if !(LaunchOptions{}).Empty() || (LaunchOptions{RunAsAdmin: true}).HasSteps() {
		t.Fatal("Empty and HasSteps")
	}

	bad := map[string]LaunchOptions{
		"can't find the folder":  {WorkDir: filepath.Join(dir, "gone")},
		"with Browse":            {WorkDir: "games"},
		"can't find the folder ": {WorkDir: exe}, // a file, not a folder
		"isn't a program's name": {CloseApps: []CloseApp{{Name: `C:\Apps\Discord.exe`}}},
		"part of Windows":        {CloseApps: []CloseApp{{Name: "Explorer.EXE"}}},
		"or Lootnest":            {CloseApps: []CloseApp{{Name: "lootnest"}}},
		"Choose a program":       {StartAfter: []Program{{Path: filepath.Join(dir, "notes.txt")}}},
		"can't find":             {StartBefore: []Program{{Path: filepath.Join(dir, "Gone.exe")}}},
		"fit on one line":        {StartBefore: []Program{{Path: exe, Args: "a\nb"}}},
		"up to 5":                {StartAfter: make([]Program, 6)},
	}
	for want, in := range bad {
		if _, err := CleanLaunchOptions(in); err == nil || !strings.Contains(err.Error(), strings.TrimSpace(want)) {
			t.Errorf("%+v: err = %v, want %q", in, err, want)
		}
	}
}

func TestLaunchOptionsMoved(t *testing.T) {
	old, nw := t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(nw, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(filepath.Dir(old), "Tools")
	for in, want := range map[string]string{
		filepath.Join(old, "bin"):  filepath.Join(nw, "bin"),
		old:                        nw,
		filepath.Join(old, "gone"): "", // not in the new copy: the game's own folder
		other:                      other,
		"":                         "",
	} {
		if got := (LaunchOptions{WorkDir: in, RunAsAdmin: true}).Moved(old, nw); got.WorkDir != want || !got.RunAsAdmin {
			t.Errorf("Moved(%q) = %+v, want %q", in, got, want)
		}
	}
}

func TestLaunchOptionsInConfig(t *testing.T) {
	s := &Store{Dir: t.TempDir()}
	cfg, _, _ := s.LoadConfig()
	o := LaunchOptions{WorkDir: `D:\Games\X\bin`, RunAsAdmin: true, CloseApps: []CloseApp{{Name: "Discord.exe", Force: true}},
		StartBefore: []Program{{Path: `C:\OBS\obs64.exe`, Args: "--minimize-to-tray"}}, StartAfter: []Program{{Path: `C:\Tools\backup.exe`}}, ReopenClosed: true}
	cfg.LaunchOpts = map[string]LaunchOptions{"dir:d:\\games\\x": o}
	if err := s.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	back, _, err := s.LoadConfig()
	if err != nil || !reflect.DeepEqual(back.LaunchOpts["dir:d:\\games\\x"], o) {
		t.Fatalf("round trip: %v %+v", err, back.LaunchOpts)
	}

	if !RekeyConfig(back, "dir:d:\\games\\x", "dir:e:\\x") || !reflect.DeepEqual(back.LaunchOpts["dir:e:\\x"], o) || len(back.LaunchOpts) != 1 {
		t.Fatalf("rekeyed = %+v", back.LaunchOpts)
	}
	// The new copy's own options stay; the old copy's go.
	back.LaunchOpts["old"] = LaunchOptions{RunAsAdmin: true}
	RekeyConfig(back, "old", "dir:e:\\x")
	if !reflect.DeepEqual(back.LaunchOpts["dir:e:\\x"], o) || len(back.LaunchOpts) != 1 {
		t.Fatalf("kept = %+v", back.LaunchOpts)
	}
}
