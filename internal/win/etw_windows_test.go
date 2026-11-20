//go:build windows

package win

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	"lootnest/internal/core"
)

// The structures ETW reads and writes, as Windows' headers lay them out on
// 64-bit Windows.
func TestETWLayouts(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("64-bit layouts")
	}
	for _, c := range []struct {
		name      string
		got, want uintptr
	}{
		{"EVENT_TRACE_PROPERTIES", unsafe.Sizeof(eventTraceProperties{}), 120},
		{"EVENT_TRACE_PROPERTIES.BufferSize(KB)", unsafe.Offsetof(eventTraceProperties{}.BufferSizeKB), 48},
		{"EVENT_TRACE_PROPERTIES.LoggerThreadId", unsafe.Offsetof(eventTraceProperties{}.LoggerThreadID), 104},
		{"EVENT_TRACE_LOGFILEW", unsafe.Sizeof(eventTraceLogfile{}), 448},
		{"EVENT_TRACE_LOGFILEW.ProcessTraceMode", unsafe.Offsetof(eventTraceLogfile{}.ProcessTraceMode), 28},
		{"EVENT_TRACE_LOGFILEW.BufferCallback", unsafe.Offsetof(eventTraceLogfile{}.BufferCallback), 400},
		{"EVENT_TRACE_LOGFILEW.EventRecordCallback", unsafe.Offsetof(eventTraceLogfile{}.EventRecordCallback), 424},
		{"EVENT_TRACE_LOGFILEW.Context", unsafe.Offsetof(eventTraceLogfile{}.Context), 440},
		{"EVENT_HEADER", unsafe.Sizeof(eventHeader{}), 80},
		{"EVENT_HEADER.ProcessId", unsafe.Offsetof(eventHeader{}.ProcessID), 12},
		{"EVENT_HEADER.ProviderId", unsafe.Offsetof(eventHeader{}.ProviderID), 24},
		{"EVENT_HEADER.EventDescriptor.Id", unsafe.Offsetof(eventHeader{}.ID), 40},
		{"ENABLE_TRACE_PARAMETERS", unsafe.Sizeof(enableTraceParameters{}), 48},
		{"ENABLE_TRACE_PARAMETERS.EnableFilterDesc", unsafe.Offsetof(enableTraceParameters{}.EnableFilterDesc), 32},
		{"EVENT_FILTER_DESCRIPTOR", unsafe.Sizeof(eventFilterDescriptor{}), 16},
	} {
		if c.got != c.want {
			t.Errorf("%s: %d, want %d", c.name, c.got, c.want)
		}
	}
}

// The session starts, hears what the PC draws for a few seconds (CI's
// runner is an administrator; frames may be none there), and stops, twice.
func TestFrameTraceStartsAndStops(t *testing.T) {
	for round := 0; round < 2; round++ {
		var mu sync.Mutex
		seen := map[[2]uint32]int{}
		tr, err := StartFrameTrace(func(pid uint32, src int, at int64) {
			mu.Lock()
			seen[[2]uint32{pid, uint32(src)}]++
			mu.Unlock()
		})
		if errors.Is(err, ErrFramesNotAllowed) {
			if FramesAllowed() {
				t.Fatal("FramesAllowed says yes, the session says no")
			}
			t.Skip("this account may not start the session")
		}
		if err != nil {
			t.Fatalf("round %d: %v", round, err)
		}
		if _, err := StartFrameTrace(func(uint32, int, int64) {}); err == nil {
			t.Fatal("a second session started beside the first")
		}
		time.Sleep(2500 * time.Millisecond)
		stopped := make(chan struct{})
		go func() { tr.Stop(); tr.Stop(); close(stopped) }()
		select {
		case <-stopped:
		case <-time.After(10 * time.Second):
			t.Fatal("Stop didn't return")
		}
		mu.Lock()
		t.Logf("round %d: frames heard (pid, source): %v", round, seen)
		mu.Unlock()
	}
	if PerfFreq() <= 0 || PerfNow() <= 0 {
		t.Fatal("no performance counter")
	}
	if !FramesAllowed() {
		t.Error("the session started but FramesAllowed says no")
	}
	if sid := UserSID(); len(sid) < 8 {
		t.Errorf("UserSID: %q", sid)
	}
}

// The frame reader's pipe: one writer, one reader, a second server refused.
func TestOutboundPipe(t *testing.T) {
	name := `\\.\pipe\lootnest-test-` + time.Now().Format("150405.000000")
	f, err := ServeOutboundPipe(name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if g, err := ServeOutboundPipe(name); err == nil {
		g.Close()
		t.Fatal("a second server took the same name")
	}
	got := make(chan string, 1)
	go func() {
		var r *os.File
		for i := 0; i < 50 && r == nil; i++ {
			r, _ = os.OpenFile(name, os.O_RDONLY, 0)
			if r == nil {
				time.Sleep(20 * time.Millisecond)
			}
		}
		if r == nil {
			got <- "no reader"
			return
		}
		defer r.Close()
		b := make([]byte, 64)
		n, _ := r.Read(b)
		got <- string(b[:n])
	}()
	if err := WaitPipeReader(f); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("{\"v\":1}\n")); err != nil {
		t.Fatal(err)
	}
	if s := <-got; s != "{\"v\":1}\n" {
		t.Fatalf("read %q", s)
	}
}

// The frame reader's task: the script registers it for this account with
// the highest rights, puts the program in Program Files, and the untask
// script takes both away. CI's runner is an administrator, as Windows'
// prompt makes the app.
func TestFrameTask(t *testing.T) {
	if !elevated() {
		t.Skip("needs an administrator, as Windows' prompt gives the real app")
	}
	src := filepath.Join(t.TempDir(), "Lootnest.exe")
	if err := os.WriteFile(src, []byte("MZ test"), 0o644); err != nil {
		t.Fatal(err)
	}
	ps := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	run := func(script string) {
		c := exec.Command(ps, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-EncodedCommand", core.EncodePowerShell("$ProgressPreference = 'SilentlyContinue'\n"+script))
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
	script, err := core.FrameTaskScript(src, UserSID())
	if err != nil {
		t.Fatal(err)
	}
	defer run(core.FrameUntaskScript())
	run(script)
	run(script) // twice: replaced, not refused
	reader := filepath.Join(os.Getenv("ProgramFiles"), core.FrameReaderDir, core.FrameReaderExe)
	if b, err := os.ReadFile(reader); err != nil || string(b) != "MZ test" {
		t.Fatalf("reader in Program Files: %q %v", b, err)
	}
	schtasks := filepath.Join(os.Getenv("SystemRoot"), "System32", "schtasks.exe")
	out, err := RunHiddenOutput(schtasks, `/query /xml /tn "`+core.FrameTaskPath+core.FrameTaskName+`"`, 20*time.Second)
	if err != nil {
		t.Fatalf("the task isn't there: %v %s", err, out)
	}
	out = strings.ReplaceAll(out, "\x00", "") // in case it answers in UTF-16
	for _, want := range []string{"HighestAvailable", "InteractiveToken", "--frames", core.FrameReaderExe} {
		if !strings.Contains(out, want) {
			t.Errorf("task lacks %q:\n%s", want, out)
		}
	}
	run(core.FrameUntaskScript())
	if _, err := RunHiddenOutput(schtasks, `/query /tn "`+core.FrameTaskPath+core.FrameTaskName+`"`, 20*time.Second); err == nil {
		t.Error("the task is still there after the untask script")
	}
	if _, err := os.Stat(reader); err == nil {
		t.Error("the reader is still in Program Files")
	}
}
