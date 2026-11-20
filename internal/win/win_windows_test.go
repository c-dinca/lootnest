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

// A CI machine has no pads; the calls must still answer, and a service
// every Windows has must read as installed and running.
func TestControllers(t *testing.T) {
	for _, d := range PadDevices() {
		if len(d.Chain) == 0 {
			t.Fatalf("a device without an instance id: %+v", d)
		}
	}
	if n := len(XInputPads()); n > 4 {
		t.Fatalf("%d XInput slots", n)
	}
	if ok, run := ServiceState("EventLog"); !ok || !run {
		t.Fatalf("EventLog: installed %v, running %v", ok, run)
	}
	if ok, _ := ServiceState("NoSuchServiceLootnest"); ok {
		t.Fatal("a service that doesn't exist reads as installed")
	}
}

func TestKnownFoldersAndDisk(t *testing.T) {
	for name, p := range map[string]string{"Documents": Documents(), "Saved Games": SavedGames(), "LocalLow": LocalLow(), "Program Files": ProgramFiles(), "Downloads": Downloads()} {
		if p == "" {
			t.Errorf("%s: empty", name)
		}
	}
	if DiskFree(os.TempDir()) <= 0 {
		t.Error("no free space reported for the temp folder")
	}
}

func TestRecycle(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "Game.v1.3.part1.rar"), filepath.Join(dir, "Game.v1.3.part2.rar")
	for _, p := range []string{a, b} {
		if err := os.WriteFile(p, []byte("lootnest test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := Recycle(0, a, b); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{a, b} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%s is still there", p)
		}
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
		c := exec.Command(ps, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-EncodedCommand", core.EncodePowerShell("$ProgressPreference = 'SilentlyContinue'\ntry {\n"+script+"\n} catch { Write-Output ('ERROR: ' + $_.Exception.Message + ' AT ' + $_.InvocationInfo.Line); exit 1 }"))
		var stderr strings.Builder
		c.Stderr = &stderr
		out, err := c.Output()
		if err != nil {
			t.Fatalf("%v running %q: %s %s", err, strings.SplitN(script, "\n", 4)[len(strings.SplitN(script, "\n", 4))-1], out, stderr.String())
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
	// Letting it online again when its rules are already gone (taken out in
	// Windows' own settings) succeeds too.
	run(core.FirewallScript(tag, "Lootnest Test Game", nil, false))
}

// TV mode's holder waits on Lootnest's process, and Lootnest on the
// holder's: WaitProcess returns once the process ended, and at once for
// one that's gone. AskCommand answers 0 from no window.
func TestWaitProcessAndAsk(t *testing.T) {
	cmd := exec.Command("cmd", "/c", "ping -n 2 127.0.0.1 >nul")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := uint32(cmd.Process.Pid)
	done := make(chan struct{})
	go func() { WaitProcess(pid); close(done) }()
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("WaitProcess didn't return after the process ended")
	}
	_ = cmd.Wait()
	gone := make(chan struct{})
	go func() { WaitProcess(pid); close(gone) }()
	select {
	case <-gone:
	case <-time.After(5 * time.Second):
		t.Fatal("WaitProcess waited on a process that's gone")
	}
	if r := AskCommand(0, "tv?:1"); r != 0 {
		t.Fatalf("AskCommand to no window answered %d", r)
	}
	_ = InJob() // only that it answers
}

func TestPSButtonIn(t *testing.T) {
	usbDS := make([]byte, 64)
	usbDS[0], usbDS[10] = 0x01, 0x01
	if !PSButtonIn(0x0ce6, usbDS) {
		t.Fatal("DualSense over USB: PS not read")
	}
	usbDS[10] = 0
	if PSButtonIn(0x0ce6, usbDS) {
		t.Fatal("DualSense over USB: PS read when up")
	}
	bt := make([]byte, 78)
	bt[0], bt[11] = 0x31, 0x01
	if !PSButtonIn(0x0ce6, bt) {
		t.Fatal("DualSense over Bluetooth: PS not read")
	}
	simple := []byte{0x01, 128, 128, 128, 128, 0x08, 0, 0x01, 0, 0}
	if !PSButtonIn(0x0ce6, simple) || !PSButtonIn(0x09cc, simple) {
		t.Fatal("simple report: PS not read")
	}
	ds4 := make([]byte, 64)
	ds4[0], ds4[7] = 0x01, 0x01
	if !PSButtonIn(0x09cc, ds4) {
		t.Fatal("DualShock 4 over USB: PS not read")
	}
	if PSButtonIn(0x0ce6, nil) {
		t.Fatal("an empty report read as PS")
	}
}

func TestTaskbarSettingsRoundTrip(t *testing.T) {
	all, mode := TaskbarSettings()
	t.Cleanup(func() {
		if err := SetTaskbarSettings(all, mode); err != nil {
			t.Errorf("putting back %d, %d: %v", all, mode, err)
		}
		if a, m := TaskbarSettings(); a != all || m != mode {
			t.Errorf("put back as %d, %d, want %d, %d", a, m, all, mode)
		}
	})
	if err := SetTaskbarSettings(1, 0); err != nil {
		t.Fatal(err)
	}
	if a, m := TaskbarSettings(); a != 1 || m != 0 {
		t.Errorf("got %d, %d, want 1, 0", a, m)
	}
	if err := SetTaskbarSettings(-1, -1); err != nil {
		t.Fatal(err)
	}
	if a, m := TaskbarSettings(); a != -1 || m != -1 {
		t.Errorf("removed values read %d, %d", a, m)
	}
}

func TestProcessNameOfSelf(t *testing.T) {
	if got := ProcessName(uint32(os.Getpid())); got == "" || strings.HasSuffix(got, ".exe") || got != strings.ToLower(got) {
		t.Errorf("ProcessName of this test: %q", got)
	}
}
