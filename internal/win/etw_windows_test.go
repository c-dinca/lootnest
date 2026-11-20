//go:build windows

package win

import (
	"errors"
	"sync"
	"testing"
	"time"
	"unsafe"
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
