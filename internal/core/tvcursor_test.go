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

func TestPCDisplay(t *testing.T) {
	tv := Rect{0, 0, 1920, 1080}
	small, big := Rect{-1280, -720, 0, 0}, Rect{-4480, -1440, -1920, 0}
	if d, ok := PCDisplay(tv, []Rect{tv, small, big}); !ok || d != big {
		t.Fatalf("%v %v", d, ok)
	}
	if d, ok := PCDisplay(tv, []Rect{tv, {1920, 0, 3840, 1080}}); !ok || d.L != 1920 {
		t.Fatalf("beside: %v %v", d, ok)
	}
	if _, ok := PCDisplay(tv, []Rect{tv}); ok {
		t.Fatal("only the TV")
	}
	if _, ok := PCDisplay(tv, []Rect{tv, {100, 100, 500, 500}}); ok {
		t.Fatal("inside the TV")
	}
}

func TestPCDisplays(t *testing.T) {
	// Apollo's layout: the TV at (0,0), the PC's two monitors up and left,
	// touching it only at its corner.
	tv := Rect{0, 0, 1920, 1080}
	small := Rect{-3840, -1080, -1920, 0}
	big := Rect{-1920, -1440, 640, 0}
	got := PCDisplays(tv, []Rect{tv, small, big})
	if len(got) != 2 || got[0] != big || got[1] != small {
		t.Errorf("got %v, want the larger first", got)
	}
	if got := PCDisplays(tv, []Rect{tv}); len(got) != 0 {
		t.Errorf("the TV's alone: %v", got)
	}
	if got := PCDisplays(tv, []Rect{tv, {100, 100, 900, 700}}); len(got) != 0 {
		t.Errorf("a display over the TV's isn't the PC's: %v", got)
	}
}

func TestOnTVAndPlaceIn(t *testing.T) {
	tv := Rect{0, 0, 1920, 1080}
	if !OnTV(Rect{100, 100, 900, 700}, tv) || OnTV(Rect{-1000, -800, -200, -100}, tv) {
		t.Error("a window's middle decides where it is")
	}
	if OnTV(Rect{-32000, -32000, -31840, -31972}, tv) {
		t.Error("a minimised window isn't on the TV")
	}
	if !FillsTV(Rect{0, 0, 1920, 1080}, tv) || !FillsTV(Rect{-8, -8, 1928, 1088}, tv) || FillsTV(Rect{0, 0, 1920, 1040}, tv) {
		t.Error("only a window over the whole TV fills it")
	}
	work := Rect{-1920, -1080, 0, -40}
	if got := PlaceIn(Rect{100, 100, 900, 700}, work); got != (Rect{-1360, -860, -560, -260}) {
		t.Errorf("a window keeps its size, in the middle: %v", got)
	}
	if got := PlaceIn(Rect{0, 0, 2560, 1440}, work); got != work {
		t.Errorf("a window larger than the display fits it: %v", got)
	}
}
