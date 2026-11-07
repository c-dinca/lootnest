package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectTool(t *testing.T) {
	d := t.TempDir()
	if DetectTool(d) != nil {
		t.Fatal("none expected")
	}
	bin := filepath.Join(d, "Game", "Binaries", "Win64")
	_ = os.MkdirAll(bin, 0o755)
	_ = os.WriteFile(filepath.Join(bin, "steam_emu.ini"), []byte("[Settings]\n###CODEX###\nAppId=123\n"), 0o644)
	if tl := DetectTool(d); tl == nil || tl.Name != "CODEX" || !tl.Reads {
		t.Fatalf("got %+v", tl)
	}
	d2 := t.TempDir()
	_ = os.MkdirAll(filepath.Join(d2, "steam_settings"), 0o755)
	if tl := DetectTool(d2); tl == nil || tl.Name != "Goldberg (GSE)" || tl.Settings == "" {
		t.Fatalf("got %+v", tl)
	}
}

func TestAchievementFiles(t *testing.T) {
	appdata := t.TempDir()
	e := &Env{Getenv: func(k string) string {
		if k == "APPDATA" {
			return appdata
		}
		return ""
	}}
	looked, there := e.AchievementFiles(42)
	if len(looked) == 0 || len(there) != 0 {
		t.Fatalf("looked %d, there %d", len(looked), len(there))
	}
	p := filepath.Join(appdata, "GSE Saves", "42", "achievements.json")
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	_ = os.WriteFile(p, []byte(`{"A":{"earned":true,"earned_time":100},"B":{"earned":false}}`), 0o644)
	_, there = e.AchievementFiles(42)
	if len(there) != 1 || there[0].Unlocked != 1 {
		t.Fatalf("got %+v", there)
	}
}

func TestToolAccountName(t *testing.T) {
	s := filepath.Join(t.TempDir(), "steam_settings")
	_ = os.MkdirAll(s, 0o755)
	keep := filepath.Join(t.TempDir(), "keep")
	if err := SetToolAccountName(s, "Cristian", keep); err != nil {
		t.Fatal(err)
	}
	if ToolAccountName(s) != "Cristian" {
		t.Fatalf("got %q", ToolAccountName(s))
	}
	// An existing file keeps its other lines; the old one is kept first.
	ini := filepath.Join(s, "configs.user.ini")
	_ = os.WriteFile(ini, []byte("[user::general]\naccount_name=Old\nlanguage=english\n"), 0o644)
	_ = os.WriteFile(filepath.Join(s, "force_account_name.txt"), []byte("Old"), 0o644)
	if err := SetToolAccountName(s, " Mi[h]ai\n$1 ", keep); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(ini)
	if !strings.Contains(string(b), "account_name=Mihai$1") || !strings.Contains(string(b), "language=english") {
		t.Fatalf("got %q", b)
	}
	if l, _ := os.ReadFile(filepath.Join(s, "force_account_name.txt")); string(l) != "Mihai$1" {
		t.Fatalf("legacy %q", l)
	}
	if k, _ := os.ReadFile(filepath.Join(keep, "configs.user.ini")); !strings.Contains(string(k), "Old") {
		t.Fatalf("kept %q", k)
	}
	// A section without the line gets it.
	_ = os.WriteFile(ini, []byte("[user::general]\nlanguage=english\n"), 0o644)
	_ = SetToolAccountName(s, "Ana", keep)
	if ToolAccountName(s) != "Ana" {
		t.Fatal("section insert")
	}
}
