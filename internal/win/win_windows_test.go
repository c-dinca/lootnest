//go:build windows

package win

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"

	"lootnest/internal/core"
)

// These run on Windows itself (CI's windows job) and check the parts of
// Lootnest that only Windows can: the note over a game, hot keys, running
// as administrator and Windows Firewall.

func TestToastShows(t *testing.T) {
	shown := make(chan uintptr, 1)
	failed := make(chan error, 1)
	toastShown = func(h uintptr) { shown <- h }
	ToastFailed = func(_ Toast, err error) { failed <- err }
	defer func() { toastShown, ToastFailed = nil, nil }()
	ShowToast(Toast{Kicker: "Achievement unlocked", Title: "A test", Line: "Rare, 4.2% of players have it", Accent: "#6C9BD2", Badge: "#6FA8F5"})
	select {
	case h := <-shown:
		if h == 0 {
			t.Fatal("no window")
		}
		if r, _, _ := procIsWindowVisible.Call(h); r == 0 {
			t.Fatal("the toast's window isn't visible")
		}
	case err := <-failed:
		t.Fatalf("the toast couldn't be shown: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("no toast after 5 seconds")
	}
}

func TestProcessNames(t *testing.T) {
	procs, err := ProcessNames()
	if err != nil {
		t.Fatal(err)
	}
	self := strings.ToLower(strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe"))
	if !procs[self] {
		t.Fatalf("this test (%s) isn't among the %d processes", self, len(procs))
	}
}

func TestKnownFoldersAndDisk(t *testing.T) {
	for name, p := range map[string]string{"Documents": Documents(), "Saved Games": SavedGames(), "LocalLow": LocalLow(), "Program Files": ProgramFiles()} {
		if p == "" {
			t.Errorf("%s: empty", name)
		}
	}
	if DiskFree(os.TempDir()) <= 0 {
		t.Error("no free space reported for the temp folder")
	}
}

func TestHotKey(t *testing.T) {
	// A key no one uses, on this thread (hwnd 0), taken and given back.
	const id = 0x7A11
	if !RegisterHotKey(0, id, ModControl|ModShift|ModAlt|ModNoRepeat, 0x87 /*F24*/) {
		t.Fatal("couldn't register Ctrl+Shift+Alt+F24")
	}
	UnregisterHotKey(0, id)
	if !RegisterHotKey(0, id, ModControl|ModShift|ModAlt|ModNoRepeat, 0x87) {
		t.Fatal("the key wasn't given back")
	}
	UnregisterHotKey(0, id)
}

func TestRunAndWaitExitCode(t *testing.T) {
	cmd := filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
	code, err := RunElevatedAndWait(0, cmd, "/c exit 3")
	if err != nil {
		t.Skipf("can't run as administrator here: %v", err)
	}
	if code != 3 {
		t.Fatalf("exit code %d, want 3", code)
	}
}

func elevated() bool { return windows.GetCurrentProcessToken().IsElevated() }

// Keep a game offline: the rules go in, are found by their tag, and come out.
func TestFirewallRules(t *testing.T) {
	if !elevated() {
		t.Skip("needs an administrator, as Windows' prompt gives the real app")
	}
	exe := filepath.Join(t.TempDir(), "Lootnest Test Game.exe")
	if err := os.WriteFile(exe, []byte("MZ"), 0o644); err != nil {
		t.Fatal(err)
	}
	exe = LongPath(exe) // as the app does: the temp folder has a short name here
	if strings.Contains(exe, "~") {
		t.Fatalf("still a short name: %s", exe)
	}
	tag := core.OfflineTag("dir:test:" + exe)
	ps := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	run := func(script string) string {
		c := exec.Command(ps, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-EncodedCommand", core.EncodePowerShell("$ProgressPreference = 'SilentlyContinue'\n"+script))
		var stderr strings.Builder
		c.Stderr = &stderr
		out, err := c.Output()
		if err != nil {
			t.Fatalf("%v: %s %s", err, out, stderr.String())
		}
		return strings.TrimSpace(string(out))
	}
	count := "@(Get-NetFirewallRule -Group 'Lootnest' -ErrorAction SilentlyContinue | Where-Object { $_.Description -eq '" + tag + "' }).Count"
	defer run(core.FirewallScript(tag, "Lootnest Test Game", nil, false))
	run(core.FirewallScript(tag, "Lootnest Test Game", []string{exe}, true))
	run(core.FirewallScript(tag, "Lootnest Test Game", []string{exe}, true)) // twice: still one rule
	if got := run(count); got != "1" {
		t.Fatalf("rules after blocking twice: %s", got)
	}
	run(core.FirewallScript(tag, "Lootnest Test Game", nil, false))
	if got := run(count); got != "0" {
		t.Fatalf("rules after letting it online: %s", got)
	}
}
