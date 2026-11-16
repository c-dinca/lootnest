package core

import "testing"

func TestNewGameWindow(t *testing.T) {
	seen := map[uint64]bool{1: true, 2: true}
	const me = 100
	cases := []struct {
		name string
		now  []TopWindow
		want uint64
	}{
		{"nothing new", []TopWindow{{1, 7, 1920, 1080}, {2, 8, 800, 600}}, 0},
		{"the game's window", []TopWindow{{9, 30, 1920, 1080}, {1, 7, 1920, 1080}}, 9},
		{"Lootnest's own new window isn't it", []TopWindow{{9, me, 1920, 1080}}, 0},
		{"a launcher's small note isn't it", []TopWindow{{9, 30, 420, 160}, {1, 7, 1920, 1080}}, 0},
		{"a window under a quarter of the screen isn't it", []TopWindow{{9, 30, 900, 500}}, 0},
		{"a windowed game is", []TopWindow{{9, 30, 1280, 720}}, 9},
		{"the front one of two new windows", []TopWindow{{12, 31, 1920, 1080}, {9, 30, 1280, 720}}, 12},
		{"a window with no process is skipped", []TopWindow{{9, 0, 1920, 1080}}, 0},
	}
	for _, c := range cases {
		if got := NewGameWindow(seen, c.now, me, 1920, 1080); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
	if got := NewGameWindow(seen, []TopWindow{{9, 30, 1920, 1080}}, me, 0, 0); got != 0 {
		t.Errorf("no screen size: got %d, want 0", got)
	}
}
