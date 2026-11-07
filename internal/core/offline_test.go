package core

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestGameExes(t *testing.T) {
	d := t.TempDir()
	mk := func(rel string) {
		p := filepath.Join(d, filepath.FromSlash(rel))
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		_ = os.WriteFile(p, []byte("MZ"), 0o644)
	}
	mk("Game.exe")
	mk("Game/Binaries/Win64/Game-Win64-Shipping.exe")
	mk("unins000.exe")
	mk("_CommonRedist/vcredist_x64.exe")
	mk("Engine/Binaries/Win64/CrashReportClient.exe")
	mk("readme.txt")
	got := GameExes(d, filepath.Join(d, "Game", "Binaries", "Win64", "Game-Win64-Shipping.exe"))
	if len(got) != 3 {
		t.Fatalf("got %v", got)
	}
	if !strings.HasSuffix(got[0], "Game-Win64-Shipping.exe") || !strings.HasSuffix(got[1], "Game.exe") {
		t.Fatalf("main first, then shallow: %v", got)
	}
	for _, p := range got {
		if strings.Contains(strings.ToLower(p), "unins") || strings.Contains(strings.ToLower(p), "redist") {
			t.Fatalf("should skip %s", p)
		}
	}
}

func TestFirewallScript(t *testing.T) {
	tag := OfflineTag("dir:d:\\games\\x")
	if !strings.HasPrefix(tag, "lootnest:") || tag != OfflineTag("DIR:D:\\Games\\X") {
		t.Fatalf("tag %q", tag)
	}
	s := FirewallScript(tag, "Bob's Game", []string{`D:\Games\Bob's\Bob.exe`}, true)
	if !strings.Contains(s, `-Program 'D:\Games\Bob''s\Bob.exe'`) || !strings.Contains(s, "'Lootnest keeps Bob''s Game offline (Bob.exe)'") {
		t.Fatalf("quoting: %s", s)
	}
	if !strings.Contains(s, "Remove-NetFirewallRule") || !strings.Contains(s, "-Direction Outbound -Action Block") {
		t.Fatal(s)
	}
	un := FirewallScript(tag, "x", []string{"a.exe"}, false)
	if strings.Contains(un, "New-NetFirewallRule") {
		t.Fatal("unblock must only remove")
	}
	if !strings.HasSuffix(un, "exit 0\n") || !strings.HasSuffix(s, "exit 0\n") {
		t.Fatal("the script must end in success when there's nothing to take away")
	}
	enc := EncodePowerShell("hé")
	raw, _ := base64.StdEncoding.DecodeString(enc)
	u := make([]uint16, len(raw)/2)
	for i := range u {
		u[i] = uint16(raw[2*i]) | uint16(raw[2*i+1])<<8
	}
	if string(utf16.Decode(u)) != "hé" {
		t.Fatal("encoding")
	}
}
