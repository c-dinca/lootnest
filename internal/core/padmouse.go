package core

import (
	"math"
	"time"
)

// The controller as a mouse: the left stick moves the pointer and the
// right stick scrolls, for a program on the TV that only knows a mouse
// (a store's launcher, Spotify's or Stremio's own window). Lootnest turns
// it on from the Home menu, or by itself for an app that needs it, and
// only while that program is in front (app/pointer.go).

// PadSticks are a controller's sticks, each from -1 to 1, up and right
// positive, as XInput gives them.
type PadSticks struct{ LX, LY, RX, RY float64 }

// StickUnit is a stick's raw value from -1 to 1.
func StickUnit(v int16) float64 {
	if v < 0 {
		return float64(v) / 32768
	}
	return float64(v) / 32767
}

// Pointer turns sticks into whole pixels and wheel steps, keeping what's
// left over for the next step so slow movement isn't lost.
type Pointer struct{ x, y, w float64 }

const (
	pointerDead  = 0.24 // a stick at rest isn't quite at zero
	pointerSpeed = 1.15 // screen heights a second with the stick fully over
	pointerWheel = 1800 // wheel units a second (120 is one notch)
)

// stickCurve is how far over a stick counts for: nothing inside the dead
// zone, slow and fine for small pushes, fast only near the edge.
func stickCurve(m float64) float64 {
	if m <= pointerDead {
		return 0
	}
	m = math.Min(1, (m-pointerDead)/(1-pointerDead))
	return 0.18*m + 0.82*m*m*m
}

// Step is how far the pointer moves (right and down positive, in pixels)
// and how much the wheel turns (up positive) in dt, on a screen screenH
// pixels high.
func (p *Pointer) Step(s PadSticks, dt time.Duration, screenH int) (dx, dy, wheel int) {
	if screenH <= 0 {
		screenH = 1080
	}
	secs := dt.Seconds()
	if secs <= 0 {
		return 0, 0, 0
	}
	if secs > 0.1 {
		secs = 0.1 // a stalled loop doesn't throw the pointer across the screen
	}
	if m := math.Hypot(s.LX, s.LY); m > pointerDead {
		v := stickCurve(m) * pointerSpeed * float64(screenH) * secs
		p.x += s.LX / m * v
		p.y -= s.LY / m * v
	} else {
		p.x, p.y = 0, 0
	}
	if m := math.Abs(s.RY); m > pointerDead {
		p.w += math.Copysign(stickCurve(m), s.RY) * pointerWheel * secs
	} else {
		p.w = 0
	}
	dx, dy, wheel = int(p.x), int(p.y), int(p.w)
	p.x -= float64(dx)
	p.y -= float64(dy)
	p.w -= float64(wheel)
	return dx, dy, wheel
}
