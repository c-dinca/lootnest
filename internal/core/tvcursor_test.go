package core

import "testing"

func TestCursorKeep(t *testing.T) {
	tv := Rect{0, 0, 1920, 1080}
	// Apollo's isolated layout: the PC's monitor ends where the TV starts,
	// touching it only at the corner.
	pc := Rect{-2560, -1440, 0, 0}
	clip, x, y, ok := CursorKeep(tv, []Rect{tv, pc})
	if !ok || clip != pc || x != -1280 || y != -720 || tv.Has(x, y) || !clip.Has(x, y) {
		t.Fatalf("isolated: %v %d,%d %v", clip, x, y, ok)
	}
	// Two PC monitors side by side, both up and left of the TV.
	left := Rect{-3840, -1080, -1920, 0}
	clip, x, y, ok = CursorKeep(tv, []Rect{left, tv, {-1920, -1080, 0, 0}})
	if !ok || clip != (Rect{-3840, -1080, 0, 0}) || !left.Has(x, y) && !clip.Has(x, y) {
		t.Fatalf("two: %v %d,%d %v", clip, x, y, ok)
	}
	for name, ds := range map[string][]Rect{
		"only the TV":       {tv},
		"side by side":      {tv, {1920, 0, 3840, 1080}},
		"above":             {tv, {0, -1080, 1920, 0}},
		"overlapping":       {tv, {100, 100, 2000, 1200}},
		"bounds reach in":   {tv, {-1920, -1080, 0, 0}, {1920, 1080, 3840, 2160}},
		"no displays found": nil,
	} {
		if _, _, _, ok := CursorKeep(tv, ds); ok {
			t.Fatalf("%s: kept the cursor", name)
		}
	}
}
