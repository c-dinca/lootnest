package app

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/plat"
)

// Windows that open on the TV in TV mode. While a stream runs, the TV's
// display is Windows' primary one (so games start there), and Windows opens
// most new windows on the primary display: a browser or a folder opened on
// the PC's own monitor showed up on the TV, where the mouse can't go. While
// TV mode is on, a window that comes onto the TV's display is moved to the
// PC's display the mouse is on, unless it's the TV's:
//
//   - Lootnest's own (its windows for YouTube, Spotify, Discord and
//     Stremio are Lootnest's program too);
//   - a game or app started from Big Picture, by its process, and anything
//     that opens in the first 45 seconds after a start (launchers, then
//     the game);
//   - one that fills the TV, as a game in fullscreen does;
//   - one that opened while the TV's side had the keyboard (Lootnest, or
//     the game it follows), as a game's own dialogs do;
//   - Windows' own desktop and taskbars.
//
// Windows already on the TV when TV mode starts are moved too, unless
// they're the TV's. A window is judged once, when it comes onto the TV;
// moved back there by its program, it's judged again.

// tvLaunchGrace is how long after a start new windows count as the TV's.
const tvLaunchGrace = 45 * time.Second

// shellClasses are Windows' own windows, which stay where they are.
var shellClasses = map[string]bool{
	"Progman": true, "WorkerW": true, "Shell_TrayWnd": true, "Shell_SecondaryTrayWnd": true,
	"Windows.UI.Core.CoreWindow": true, "XamlExplorerHostIslandWindow": true, "NotifyIconOverflowWindow": true,
	"TopLevelWindowForOverflowXamlIsland": true,
}

// tvSide says whether a process is the TV's: Lootnest's own program, a
// game or app it started and follows, or the game it stands behind.
func (a *App) tvSide(pid uint32, self string) bool {
	if pid == uint32(os.Getpid()) {
		return true
	}
	name := plat.ProcessName(pid)
	if name == "" {
		return false
	}
	if name == self {
		return true
	}
	a.mu.Lock()
	game := a.conGame
	a.mu.Unlock()
	if game != "" && strings.EqualFold(a.procOf(game), name) {
		return true
	}
	foundRunsMu.Lock()
	defer foundRunsMu.Unlock()
	for _, r := range foundRuns {
		if r.pid == pid || r.proc != "" && strings.EqualFold(r.proc, name) {
			return true
		}
	}
	return false
}

func (a *App) tvWindows() {
	self := plat.ProcessName(uint32(os.Getpid()))
	if exe, err := os.Executable(); self == "" && err == nil {
		self = strings.TrimSuffix(strings.ToLower(filepath.Base(exe)), ".exe")
	}
	box := func(b [4]int) core.Rect { return core.Rect{L: b[0], T: b[1], R: b[2], B: b[3]} }
	onTV := map[uintptr]bool{} // where each window was last seen
	first := true
	fgTV := true // the TV's side had the keyboard at the last look
	for {
		time.Sleep(300 * time.Millisecond)
		a.mu.Lock()
		on, conWin, started := a.tv, a.conWin, a.conStart
		a.mu.Unlock()
		if !on {
			return
		}
		var ds []core.Rect
		for _, d := range plat.Displays() {
			ds = append(ds, box(d))
		}
		tv := box(plat.PrimaryDisplay())
		// Only with the TV's display on its own beside the PC's (as for the
		// mouse): otherwise the primary display may be the PC's own monitor.
		if _, _, _, ok := core.CursorKeep(tv, ds); !ok {
			clear(onTV)
			first = true
			continue
		}
		pcs := core.PCDisplays(tv, ds)
		launching := time.Since(started) < tvLaunchGrace
		seen := map[uintptr]bool{}
		for _, w := range plat.TopWindows() {
			seen[w.HWND] = true
			r := box(plat.WindowBox(w.HWND))
			in := core.OnTV(r, tv)
			was, known := onTV[w.HWND]
			onTV[w.HWND] = in
			if !in || known && was {
				continue
			}
			// It has just come onto the TV (or was there when TV mode began).
			if w.HWND == a.hwnd || w.HWND == conWin || launching || core.FillsTV(r, tv) ||
				!first && fgTV || shellClasses[plat.WindowClass(w.HWND)] || a.tvSide(w.PID, self) {
				continue
			}
			to := pcs[0]
			if x, y, ok := plat.CursorPos(); ok {
				for _, pc := range pcs {
					if pc.Has(x, y) {
						to = pc
						break
					}
				}
			}
			work := box(plat.WorkAreaOf([4]int{to.L, to.T, to.R, to.B}))
			at := core.PlaceIn(r, work)
			plat.MoveWindowTo(w.HWND, [4]int{at.L, at.T, at.R, at.B})
			onTV[w.HWND] = false
			a.logf("TV mode: a window of %q opened on the TV; moved to the PC's display at %d,%d", plat.ProcessName(w.PID), to.L, to.T)
		}
		for h := range onTV {
			if !seen[h] {
				delete(onTV, h)
			}
		}
		first = false
		fg := plat.Foreground()
		fgTV = fg != 0 && (fg == a.hwnd || fg == conWin || core.OnTV(box(plat.WindowBox(fg)), tv) && a.tvSide(plat.ForegroundPID(), self))
	}
}
