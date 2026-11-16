package core

// TopWindow is a window on show, as the app lists them: who made it and
// how large it is.
type TopWindow struct {
	ID   uint64
	PID  uint32
	W, H int
}

// NewGameWindow picks the window a game just opened, when Lootnest doesn't
// know the game's process: the front-most window that wasn't there before
// the game was started, isn't Lootnest's own, and takes at least a quarter
// of the screen (a launcher's small "starting" note doesn't count). now is
// front first. It returns 0 when there's none yet.
func NewGameWindow(seen map[uint64]bool, now []TopWindow, self uint32, screenW, screenH int) uint64 {
	for _, w := range now {
		if seen[w.ID] || w.PID == self || w.PID == 0 {
			continue
		}
		if w.W < 480 || w.H < 300 || screenW <= 0 || screenH <= 0 {
			continue
		}
		if int64(w.W)*int64(w.H)*4 < int64(screenW)*int64(screenH) {
			continue
		}
		return w.ID
	}
	return 0
}
