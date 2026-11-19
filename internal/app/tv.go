package app

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/plat"
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
	tvPatience  = 20 // seconds without an answer before the holder gives up
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
		if _, err := plat.StartOutsideJob(exe, "--tray", filepath.Dir(exe)); err != nil {
			return 1
		}
		if h = findLootnest(30 * time.Second); h == 0 {
			return 1
		}
	}
	if !plat.SendCommand(h, "tv:"+strconv.FormatUint(uint64(pid), 10)) {
		return 1
	}
	ask := "tv?:" + strconv.FormatUint(uint64(pid), 10)
	// TV mode ended (Lootnest says so), or Lootnest is gone: the stream
	// ends here. A Lootnest busy for a few seconds (saving a profile's
	// files after a game) doesn't answer at once; that isn't an end.
	for unsure := 0; unsure < tvPatience; {
		time.Sleep(time.Second)
		h = plat.FindWindow(WindowClass)
		if h == 0 {
			unsure++
			continue
		}
		switch plat.AskCommand(h, ask) {
		case tvAnswerOn:
			unsure = 0
		case tvAnswerOff:
			return 0
		default:
			unsure++
		}
	}
	return 0
}

func findLootnest(wait time.Duration) uintptr {
	for end := time.Now().Add(wait); ; time.Sleep(100 * time.Millisecond) {
		if h := plat.FindWindow(WindowClass); h != 0 || time.Now().After(end) {
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
		a.tvWasHidden = !plat.IsVisible(a.hwnd)
	}
	ready := a.ready
	if !ready {
		a.openTV = true
	}
	a.mu.Unlock()
	a.logf("TV mode on (asked by process %d)", holder)
	if plat.InJob() {
		a.logf("WARNING Lootnest runs inside another program's job: if that program ends it (a streaming host ending the stream), Lootnest closes with it")
	}
	a.show()
	plat.FullscreenPrimary(a.hwnd)
	plat.ForceForeground(a.hwnd)
	tvTaskbar(false)
	plat.WatchPadRawInput(a.hwnd, true)
	held := plat.RegisterHotKey(a.hwnd, hotkeyTV, plat.ModControl|plat.ModShift|plat.ModNoRepeat, plat.VK_F11)
	if !held && !was {
		a.logf("Ctrl+Shift+F11 is taken by another program; TV mode is left from the bar on the PC's screen")
	}
	a.mu.Lock()
	a.tvKeyHeld = held || a.tvKeyHeld && was
	a.mu.Unlock()
	if !was {
		a.startGuide()
		go a.keepCursor()
	}
	if ready {
		a.emit("tv", map[string]any{"on": true})
	}
	go func() {
		plat.WaitProcess(holder)
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
	a.hidFor, a.tvMenuOn, a.tvPlaying, a.tvKeyHeld = "", false, "", false
	a.conGame, a.conWin = "", 0
	a.conGen++
	a.mu.Unlock()
	plat.UnregisterHotKey(a.hwnd, hotkeyTV)
	// A newer Lootnest for TV goes on the TV once the app there is closed.
	go func() {
		// The TV app writes a stream's numbers as it ends.
		time.Sleep(8 * time.Second)
		a.noteStreams()
		time.Sleep(2 * time.Minute)
		a.updateTVApps()
	}()
	plat.HideTVBar()
	tvTaskbar(true)
	plat.WatchPadRawInput(a.hwnd, false)
	plat.SetFullscreen(a.hwnd, false)
	if hidden || playing {
		plat.ShowWindow(a.hwnd, plat.SW_HIDE)
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

// tvBack brings TV mode or Big Picture back to the front, with the
// keyboard and the controller, once the game it stood behind closed.
func (a *App) tvBack() {
	a.show()
	a.mu.Lock()
	tv := a.tv
	a.mu.Unlock()
	if tv {
		plat.FullscreenPrimary(a.hwnd)
	}
	plat.ForceForeground(a.hwnd)
}

// tvTaskbarMark is kept while TV mode has the taskbar hidden, so a
// Lootnest that ended without leaving TV mode shows it again at its next
// start (tvTaskbarAtStart).
func tvTaskbarMark() string { return filepath.Join(DataDir(), "taskbar-hidden") }

// tvTaskbar hides Windows' main taskbar for TV mode (it's on the TV's
// display while a stream runs) and shows it again after.
func tvTaskbar(show bool) {
	if show {
		plat.ShowTaskbar(true)
		_ = os.Remove(tvTaskbarMark())
		return
	}
	_ = os.WriteFile(tvTaskbarMark(), nil, 0o600)
	plat.ShowTaskbar(false)
}

// tvTaskbarAtStart shows the taskbar a TV mode that didn't end left hidden.
func tvTaskbarAtStart() {
	if _, err := os.Stat(tvTaskbarMark()); err == nil {
		tvTaskbar(true)
	}
}

// procOf is the process name of a game by its profile game id or Library
// key, or "" when Lootnest doesn't know it.
func (a *App) procOf(key string) string {
	a.mu.Lock()
	if a.cfg != nil {
		if g := a.cfg.Game(key); g != nil {
			p := procName(g)
			a.mu.Unlock()
			return p
		}
	}
	a.mu.Unlock()
	foundRunsMu.Lock()
	defer foundRunsMu.Unlock()
	if r := foundRuns[key]; r != nil {
		return r.proc
	}
	return ""
}

// keepCursor keeps the PC's mouse on the PC's own displays while TV mode is
// on (core.CursorKeep): TV mode is driven by the controller, and the
// isolated display the TV shows touches the PC's only at a corner, so a
// cursor Windows put on it couldn't come back. A cursor found on the TV is
// moved to the PC's largest display, and it's held there again whenever
// Windows or a game lets it go. It's let go when TV mode ends.
func (a *App) keepCursor() {
	var (
		clip   [4]int
		tv     core.Rect
		keep   bool
		held   bool
		hx, hy int
		looked time.Time
	)
	box := func(b [4]int) core.Rect { return core.Rect{L: b[0], T: b[1], R: b[2], B: b[3]} }
	for {
		time.Sleep(100 * time.Millisecond)
		a.mu.Lock()
		on := a.tv
		a.mu.Unlock()
		if !on {
			if held {
				plat.ClipCursor(nil)
			}
			plat.HideTVBar()
			return
		}
		if time.Since(looked) >= time.Second {
			looked = time.Now()
			var ds []core.Rect
			for _, d := range plat.Displays() {
				ds = append(ds, box(d))
			}
			tv = box(plat.PrimaryDisplay())
			c, x, y, ok := core.CursorKeep(tv, ds)
			if ok != keep {
				if ok {
					a.logf("TV mode: the mouse stays on the PC's displays (%d,%d to %d,%d); the TV's is %d,%d to %d,%d", c.L, c.T, c.R, c.B, tv.L, tv.T, tv.R, tv.B)
				} else {
					a.logf("TV mode: the mouse is free (%d displays, the TV's isn't on its own)", len(ds))
				}
			}
			if !ok && held {
				plat.ClipCursor(nil)
				held = false
			}
			keep, hx, hy, clip = ok, x, y, [4]int{c.L, c.T, c.R, c.B}
			if pc, ok := core.PCDisplay(tv, ds); ok {
				plat.ShowTVBar([4]int{pc.L, pc.T, pc.R, pc.B}, a.tvBarShows(), a.tvLeaveFromPC)
			} else {
				plat.HideTVBar()
			}
		}
		if !keep {
			continue
		}
		if x, y, ok := plat.CursorPos(); ok && tv.Has(x, y) {
			plat.MoveCursor(hx, hy)
		}
		if plat.CursorClip() != clip {
			plat.ClipCursor(&clip)
			held = true
		}
	}
}

// hotkeyTV is Ctrl+Shift+F11, held while TV mode is on: it leaves TV mode
// from the PC's keyboard, wherever the focus is.
const hotkeyTV = 0x4C54 // "LT"

// tvBarShows is what TV mode's bar on the PC says.
func (a *App) tvBarShows() plat.TVBar {
	a.mu.Lock()
	playing, keys := a.tvPlaying, a.tvKeyHeld
	a.mu.Unlock()
	line := "The TV's stream ends when you leave"
	if keys {
		line = "Ctrl+Shift+F11 leaves it too"
	}
	title := "Lootnest is on the TV"
	if playing != "" {
		title = playing + " is on the TV"
	}
	return plat.TVBar{Title: title, Line: line, Key: "Leave TV mode", Busy: "Leaving"}
}

// tvLeaveFromPC is the bar's key and Ctrl+Shift+F11: TV mode ends, the
// stream with it, and the PC is as it was.
func (a *App) tvLeaveFromPC() {
	a.ui(func() {
		a.logf("TV mode off: left from the PC")
		a.leaveTV()
	})
}

// pidOf is the process an app is followed by, or 0.
func (a *App) pidOf(key string) uint32 {
	foundRunsMu.Lock()
	defer foundRunsMu.Unlock()
	if r := foundRuns[key]; r != nil {
		return r.pid
	}
	return 0
}

func (a *App) mediaApps() []core.MediaApp {
	exists := func(p string) bool { _, err := os.Stat(p); return err == nil }
	self, _ := os.Executable()
	return core.FindMediaApps(core.MediaPaths{
		AppData: os.Getenv("APPDATA"), LocalAppData: os.Getenv("LOCALAPPDATA"),
		ProgramFiles: os.Getenv("ProgramFiles"), ProgramFilesX86: os.Getenv("ProgramFiles(x86)"),
		Self: self,
	}, exists)
}

// apiMediaApps lists the apps on this PC that TV mode can start.
func (a *App) apiMediaApps() []core.MediaApp { return a.mediaApps() }

// apiLaunchApp starts one of TV mode's apps, or brings it back to the
// front when it already runs, and follows it as a game: Lootnest stays
// behind it, Home opens the menu over it, and closing it comes back.
func (a *App) apiLaunchApp(id string) error {
	var app *core.MediaApp
	for _, m := range a.mediaApps() {
		if m.ID == id {
			app = &m
			break
		}
	}
	if app == nil {
		return errors.New("That app isn't on this PC")
	}
	key := core.MediaAppKey(app.ID)
	r := &foundRun{proc: app.Proc, until: time.Now().Add(2 * time.Minute)}
	plat.AllowAnyForeground()
	if app.ByPID {
		if old := a.pidOf(key); old != 0 && plat.ProcessAlive(old) {
			r.pid = old
		} else {
			args := app.Args
			if app.Own { // on the monitor Lootnest is on
				args += " --on " + strconv.FormatUint(uint64(a.hwnd), 10)
			}
			pid, err := plat.StartTracked(app.Exe, args, filepath.Dir(app.Exe))
			if err != nil {
				return err
			}
			r.pid = pid
		}
	} else if procs, err := plat.ProcessNames(); err != nil || !procs[app.Proc] {
		if err := plat.StartDetached(app.Exe, app.Args, filepath.Dir(app.Exe)); err != nil {
			return err
		}
	}
	foundRunsMu.Lock()
	foundRuns[key] = r
	foundRunsMu.Unlock()
	a.logf("Starting %s from TV mode", app.Name)
	a.tvGameStarting(key, app.Name)
	// A program made for a mouse gets the controller as one; a site in
	// Lootnest's own window is driven by the controller already.
	a.pointerSet(conKey(key), !app.Own)
	return nil
}
