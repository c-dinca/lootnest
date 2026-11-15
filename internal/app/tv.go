//go:build windows

package app

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"lootnest/internal/win"
)

// TV mode (Lootnest Link, milestone M1): Lootnest as a console's screen on
// a TV, for a streaming host such as Apollo or Sunshine that starts
// "lootnest.exe --tv" as its app and ends the stream when that program
// ends.
//
// The program the host starts is never the Lootnest that swaps files and
// starts games. A host ends its app by ending the app's whole job, so that
// Lootnest would be cut off in the middle of loading a profile's files,
// with the game it started. So "--tv" is a second Lootnest.exe that only
// holds the stream (TVHolder): it asks the running Lootnest for TV mode,
// starting it first, outside the host's job when the job allows it, and
// stays until TV mode ends.

const (
	tvAnswerOn  = 2 // "tv?" while TV mode is on for that holder
	tvAnswerOff = 1
)

// TVHolder is the whole of "lootnest.exe --tv". It returns the exit code.
func TVHolder() int {
	pid := uint32(os.Getpid())
	h := findLootnest(2 * time.Second)
	if h == 0 {
		exe, err := os.Executable()
		if err != nil {
			return 1
		}
		// Started hidden: TV mode shows it, on the TV, a moment later.
		if _, err := win.StartOutsideJob(exe, "--tray", filepath.Dir(exe)); err != nil {
			return 1
		}
		if h = findLootnest(30 * time.Second); h == 0 {
			return 1
		}
	}
	if !win.SendCommand(h, "tv:"+strconv.FormatUint(uint64(pid), 10)) {
		return 1
	}
	ask := "tv?:" + strconv.FormatUint(uint64(pid), 10)
	for {
		time.Sleep(time.Second)
		if h = win.FindWindow(WindowClass); h == 0 || win.AskCommand(h, ask) != tvAnswerOn {
			return 0 // TV mode ended, or Lootnest quit: the stream ends here
		}
	}
}

func findLootnest(wait time.Duration) uintptr {
	for end := time.Now().Add(wait); ; time.Sleep(100 * time.Millisecond) {
		if h := win.FindWindow(WindowClass); h != 0 || time.Now().After(end) {
			return h
		}
	}
}

// tvCommand handles "tv:<pid>" (a holder asks for TV mode) and
// "tv?:<pid>" (is it still on for that holder). It runs on the window's
// thread.
func (a *App) tvCommand(cmd string) uintptr {
	if rest, ok := strings.CutPrefix(cmd, "tv?:"); ok {
		pid, _ := strconv.ParseUint(rest, 10, 32)
		a.mu.Lock()
		on := a.tv && a.tvHolder == uint32(pid)
		a.mu.Unlock()
		if on {
			return tvAnswerOn
		}
		return tvAnswerOff
	}
	pid, err := strconv.ParseUint(strings.TrimPrefix(cmd, "tv:"), 10, 32)
	if err != nil || pid == 0 {
		return tvAnswerOff
	}
	a.enterTV(uint32(pid))
	return tvAnswerOn
}

// enterTV fills the primary display with Big Picture in TV mode, for the
// holder with this process id, and leaves TV mode when that holder ends
// (the host ended the stream).
func (a *App) enterTV(holder uint32) {
	a.mu.Lock()
	was := a.tv
	a.tv, a.tvHolder, a.bigPicture = true, holder, true
	if !was {
		a.tvWasHidden = !win.IsVisible(a.hwnd)
	}
	ready := a.ready
	if !ready {
		a.openTV = true
	}
	a.mu.Unlock()
	a.logf("TV mode on (asked by process %d)", holder)
	if win.InJob() {
		a.logf("WARNING Lootnest runs inside another program's job: if that program ends it (a streaming host ending the stream), Lootnest closes with it")
	}
	a.show()
	win.FullscreenPrimary(a.hwnd)
	win.ForceForeground(a.hwnd)
	if ready {
		a.emit("tv", map[string]any{"on": true})
	}
	go func() {
		win.WaitProcess(holder)
		a.ui(func() {
			a.mu.Lock()
			mine := a.tv && a.tvHolder == holder
			a.mu.Unlock()
			if mine {
				a.logf("TV mode off: the stream ended")
				a.leaveTV()
			}
		})
	}()
}

// leaveTV ends TV mode: Big Picture closes, the window goes back to where
// it was, and to the tray if it was there. The holder sees it and ends,
// and the stream with it. Call it on the window's thread.
func (a *App) leaveTV() {
	a.mu.Lock()
	if !a.tv {
		a.mu.Unlock()
		return
	}
	hidden, playing := a.tvWasHidden, a.hidFor != ""
	a.tv, a.tvHolder, a.openTV, a.bigPicture = false, 0, false, false
	a.mu.Unlock()
	win.SetFullscreen(a.hwnd, false)
	if hidden || playing {
		win.ShowWindow(a.hwnd, win.SW_HIDE)
	}
	a.emit("tv", map[string]any{"on": false})
}

// apiTVExit is Exit TV mode in TV mode's menu.
func (a *App) apiTVExit() {
	a.ui(func() {
		a.logf("TV mode off: left from its menu")
		a.leaveTV()
	})
}

// tvBack brings TV mode back to the front, with the keyboard and the
// controller, once the game it hid for closed.
func (a *App) tvBack() {
	a.show()
	win.FullscreenPrimary(a.hwnd)
	win.ForceForeground(a.hwnd)
}
