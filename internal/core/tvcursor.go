package core

import "slices"

// Keeping the PC's mouse off the TV in TV mode. A streaming host's isolated
// virtual display (Apollo's isolated_virtual_display_option) is made the
// primary display at (0,0), with the PC's own displays moved up and left
// so they touch it only at its top-left corner. Windows puts the cursor on
// the primary display, and a single corner point can't be crossed, so the
// mouse was trapped on the TV. TV mode is driven by the controller, so the
// mouse is kept on the PC's displays instead.

// Rect is a display's area in desktop coordinates; R and B are exclusive.
type Rect struct{ L, T, R, B int }

// Has says whether the point is inside.
func (r Rect) Has(x, y int) bool { return x >= r.L && x < r.R && y >= r.T && y < r.B }

func (r Rect) area() int { return (r.R - r.L) * (r.B - r.T) }

// overlap is how far two ranges share, negative when they're apart.
func overlap(a0, a1, b0, b1 int) int { return min(a1, b1) - max(a0, b0) }

// sharesEdge says whether two displays share a stretch of an edge, which
// the cursor can cross, rather than at most a corner.
func sharesEdge(a, b Rect) bool {
	if (a.R == b.L || b.R == a.L) && overlap(a.T, a.B, b.T, b.B) > 0 {
		return true
	}
	return (a.B == b.T || b.B == a.T) && overlap(a.L, a.R, b.L, b.R) > 0
}

// CursorKeep says where the mouse belongs while the TV shows the display
// tv: clip is the PC's displays' bounds, which never reach into tv, and
// home the middle of the largest of them, for a cursor found on the TV.
// ok is false unless tv is the isolated display: other displays exist,
// none overlaps it or shares an edge with it, and their bounds stay out
// of it. Otherwise the cursor is left alone.
func CursorKeep(tv Rect, displays []Rect) (clip Rect, homeX, homeY int, ok bool) {
	var others []Rect
	for _, d := range displays {
		if d == tv || d.area() <= 0 {
			continue
		}
		if overlap(d.L, d.R, tv.L, tv.R) > 0 && overlap(d.T, d.B, tv.T, tv.B) > 0 || sharesEdge(d, tv) {
			return Rect{}, 0, 0, false
		}
		others = append(others, d)
	}
	if len(others) == 0 || tv.area() <= 0 {
		return Rect{}, 0, 0, false
	}
	clip, big := others[0], others[0]
	for _, d := range others[1:] {
		clip = Rect{min(clip.L, d.L), min(clip.T, d.T), max(clip.R, d.R), max(clip.B, d.B)}
		if d.area() > big.area() {
			big = d
		}
	}
	if overlap(clip.L, clip.R, tv.L, tv.R) > 0 && overlap(clip.T, clip.B, tv.T, tv.B) > 0 {
		return Rect{}, 0, 0, false
	}
	return clip, (big.L + big.R) / 2, (big.T + big.B) / 2, true
}

// PCDisplay is the PC's own display while the TV shows tv: the largest
// other display that doesn't overlap it, where TV mode's bar on the PC goes.
// ok is false when the TV's is the only one.
func PCDisplay(tv Rect, displays []Rect) (Rect, bool) {
	var best Rect
	for _, d := range displays {
		if d == tv || d.area() <= 0 || overlap(d.L, d.R, tv.L, tv.R) > 0 && overlap(d.T, d.B, tv.T, tv.B) > 0 {
			continue
		}
		if d.area() > best.area() {
			best = d
		}
	}
	return best, best.area() > 0
}

// PCDisplays is every display of the PC's own while the TV shows tv,
// the largest first: each gets TV mode's bar, and windows that open on the
// TV go to one of them.
func PCDisplays(tv Rect, displays []Rect) []Rect {
	var out []Rect
	for _, d := range displays {
		if d == tv || d.area() <= 0 || overlap(d.L, d.R, tv.L, tv.R) > 0 && overlap(d.T, d.B, tv.T, tv.B) > 0 {
			continue
		}
		out = append(out, d)
	}
	slices.SortStableFunc(out, func(a, b Rect) int { return b.area() - a.area() })
	return out
}

// Centre is the middle of an area.
func (r Rect) Centre() (x, y int) { return (r.L + r.R) / 2, (r.T + r.B) / 2 }

// OnTV says whether a window is on the TV's display: its middle is there.
func OnTV(win, tv Rect) bool {
	x, y := win.Centre()
	return win.area() > 0 && tv.Has(x, y)
}

// FillsTV says whether a window covers the whole of the TV's display, as a
// game in fullscreen does.
func FillsTV(win, tv Rect) bool {
	return win.L <= tv.L && win.T <= tv.T && win.R >= tv.R && win.B >= tv.B
}

// PlaceIn is where a window of win's size goes in a display's work area:
// in its middle, no larger than it.
func PlaceIn(win, work Rect) Rect {
	w, h := min(win.R-win.L, work.R-work.L), min(win.B-win.T, work.B-work.T)
	l, t := work.L+(work.R-work.L-w)/2, work.T+(work.B-work.T-h)/2
	return Rect{l, t, l + w, t + h}
}
