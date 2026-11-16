package core

import (
	"testing"
	"time"
)

func TestPointerStep(t *testing.T) {
	const frame = 8 * time.Millisecond
	run := func(s PadSticks, n int) (x, y, w int) {
		var p Pointer
		for i := 0; i < n; i++ {
			dx, dy, dw := p.Step(s, frame, 1080)
			x, y, w = x+dx, y+dy, w+dw
		}
		return
	}
	// At rest, and inside the dead zone: nothing moves.
	if x, y, w := run(PadSticks{LX: 0.2, LY: -0.1, RY: 0.2}, 125); x != 0 || y != 0 || w != 0 {
		t.Fatalf("a resting stick moved %d,%d and scrolled %d", x, y, w)
	}
	// Fully right for a second: about 1.15 screen heights, and not up or down.
	if x, y, _ := run(PadSticks{LX: 1}, 125); x < 1150 || x > 1250 || y != 0 {
		t.Fatalf("full right for a second moved %d,%d", x, y)
	}
	// Up on the stick is up on the screen.
	if _, y, _ := run(PadSticks{LY: 1}, 125); y > -1150 {
		t.Fatalf("full up moved %d", y)
	}
	// A small push is slow, and still moves (the leftover pixels add up).
	if x, _, _ := run(PadSticks{LX: 0.4}, 125); x < 20 || x > 120 {
		t.Fatalf("a small push for a second moved %d", x)
	}
	// A diagonal isn't faster than straight.
	dx, dy, _ := run(PadSticks{LX: 0.7071, LY: -0.7071}, 125)
	if d := dx*dx + dy*dy; d > 1260*1260 || d < 1100*1100 {
		t.Fatalf("a full diagonal for a second moved %d,%d", dx, dy)
	}
	// The right stick scrolls: up is up, about 15 notches a second.
	if _, _, w := run(PadSticks{RY: 1}, 125); w < 1700 || w > 1850 {
		t.Fatalf("scrolling up for a second turned %d", w)
	}
	if _, _, w := run(PadSticks{RY: -1}, 125); w > -1700 {
		t.Fatalf("scrolling down for a second turned %d", w)
	}
	// A stalled loop counts as a tenth of a second at most.
	var p Pointer
	if dx, _, _ := p.Step(PadSticks{LX: 1}, 3*time.Second, 1080); dx > 130 {
		t.Fatalf("a stalled step moved %d", dx)
	}
	if StickUnit(32767) != 1 || StickUnit(-32768) != -1 || StickUnit(0) != 0 {
		t.Fatal("stick units")
	}
}
