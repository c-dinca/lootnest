package core

import (
	"strings"
	"testing"
)

func TestFrameCounterRate(t *testing.T) {
	const f = 1000 // ticks a second
	c := NewFrameCounter(f)
	// 60 frames a second for 3 seconds, heard by DXGI and by the kernel.
	for i := 0; i < 180; i++ {
		at := int64(i * f / 60)
		c.Add(42, FrameDXGI, at)
		c.Add(42, FrameKernel, at)
	}
	last := int64(179 * f / 60)
	if fps, ok := c.Rate(42, last+500); !ok || fps != 60 {
		t.Fatalf("60 fps from two sources: got %d %v, want 60 (the largest, not the sum)", fps, ok)
	}
	// Another program's frames don't count.
	if _, ok := c.Rate(7, last); ok {
		t.Fatal("a process that drew nothing has a rate")
	}
	// Gone quiet for more than 3 seconds: no rate.
	if _, ok := c.Rate(42, last+3*f+1); ok {
		t.Fatal("a rate after 3 quiet seconds")
	}
	// A slower source alone (Direct3D 9 at 30) beside nothing else.
	for i := 0; i < 90; i++ {
		c.Add(9, FrameD3D9, int64(i*f/30))
	}
	if fps, ok := c.Rate(9, int64(89*f/30)); !ok || fps != 30 {
		t.Fatalf("30 fps: got %d %v", fps, ok)
	}
}

func TestFrameCounterKeepsTwoSeconds(t *testing.T) {
	c := NewFrameCounter(100)
	for i := 0; i < 1000; i++ {
		c.Add(1, FrameDXGI, int64(i))
	}
	if n := len(c.procs[1].at[FrameDXGI]); n > 201 {
		t.Fatalf("kept %d frames, want about two seconds' worth", n)
	}
	if fps, _ := c.Rate(1, 999); fps != 100 {
		t.Fatalf("got %d, want 100", fps)
	}
	// Out of order by a little, as events from two processors can be.
	c.Add(1, FrameDXGI, 995)
	if fps, _ := c.Rate(1, 999); fps != 101 {
		t.Fatalf("a late frame inside the second: got %d, want 101", fps)
	}
	c.Add(0, FrameDXGI, 1000)
	c.Add(1, 9, 1000)
	if len(c.procs) != 1 {
		t.Fatal("pid 0 or an unknown source was counted")
	}
	c.Forget()
	if _, ok := c.Rate(1, 999); ok {
		t.Fatal("Forget kept frames")
	}
}

func TestCleanFPSCorner(t *testing.T) {
	for in, want := range map[string]string{"": "tl", "br": "br", "tr": "tr", "middle": "tl"} {
		if got := CleanFPSCorner(in); got != want {
			t.Errorf("CleanFPSCorner(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPerfLogUsersScript(t *testing.T) {
	s, err := PerfLogUsersScript("S-1-5-21-1004336348-1177238915-682003330-1001")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"S-1-5-32-559", "'S-1-5-21-1004336348-1177238915-682003330-1001'", "Add-LocalGroupMember", "MemberExists", "exit 0"} {
		if !strings.Contains(s, want) {
			t.Errorf("script lacks %q", want)
		}
	}
	for _, bad := range []string{"", "S-1-5-21-1'; Remove-Item C:\\ -Recurse; '", "Administrator", "S-1-"} {
		if _, err := PerfLogUsersScript(bad); err == nil {
			t.Errorf("PerfLogUsersScript(%q) took it", bad)
		}
	}
}
