//go:build windows

package app

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"lootnest/internal/core"
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
	// TV mode ended (Lootnest says so), or Lootnest is gone: the stream
	// ends here. A Lootnest busy for a few seconds (saving a profile's
	// files after a game) doesn't answer at once; that isn't an end.
	for unsure := 0; unsure < tvPatience; {
		time.Sleep(time.Second)
		h = win.FindWindow(WindowClass)
		if h == 0 {
			unsure++
			continue
		}
		switch win.AskCommand(h, ask) {
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
	tvTaskbar(false)
	win.WatchPadRawInput(a.hwnd, true)
	if !was {
		go a.watchGuide()
	}
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
	a.hidFor, a.tvMenuOn = "", false
	a.mu.Unlock()
	tvTaskbar(true)
	win.WatchPadRawInput(a.hwnd, false)
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

// tvTaskbarMark is kept while TV mode has the taskbar hidden, so a
// Lootnest that ended without leaving TV mode shows it again at its next
// start (tvTaskbarAtStart).
func tvTaskbarMark() string { return filepath.Join(DataDir(), "taskbar-hidden") }

// tvTaskbar hides Windows' main taskbar for TV mode (it's on the TV's
// display while a stream runs) and shows it again after.
func tvTaskbar(show bool) {
	if show {
		win.ShowTaskbar(true)
		_ = os.Remove(tvTaskbarMark())
		return
	}
	_ = os.WriteFile(tvTaskbarMark(), nil, 0o600)
	win.ShowTaskbar(false)
}

// tvTaskbarAtStart shows the taskbar a TV mode that didn't end left hidden.
func tvTaskbarAtStart() {
	if _, err := os.Stat(tvTaskbarMark()); err == nil {
		tvTaskbar(true)
	}
}

// tvGameStarting runs right after a game was asked to start. In TV mode,
// Big Picture shows the game starting, and Lootnest lets the game come to
// the front over it and steps behind every other window a moment later,
// still covering the desktop, rather than hiding.
func (a *App) tvGameStarting(key, name string) {
	a.mu.Lock()
	tv := a.tv
	a.mu.Unlock()
	if !tv {
		return
	}
	a.emit("tvGame", map[string]any{"phase": "starting", "key": key, "name": name})
	go func() {
		time.Sleep(1500 * time.Millisecond)
		a.ui(func() {
			a.mu.Lock()
			tv := a.tv
			a.mu.Unlock()
			if tv {
				win.SendToBack(a.hwnd)
			}
		})
	}()
}

// tvGameUp brings the game TV mode started to the front once its process
// runs, as its window may open behind Lootnest. Call it on the window's
// thread.
func (a *App) tvGameUp(key string) {
	proc, pid := a.procOf(key), a.pidOf(key)
	a.emit("tvGame", map[string]any{"phase": "playing", "key": key})
	win.SendToBack(a.hwnd)
	if proc == "" && pid == 0 {
		return
	}
	go func() {
		// Its window can take a while after its process.
		for i := 0; i < 20; i++ {
			gw := uintptr(0)
			if pid != 0 {
				gw = win.WindowOfPID(pid)
			} else {
				gw = win.GameWindow(proc)
			}
			if gw != 0 {
				a.ui(func() { win.ForceForeground(gw) })
				return
			}
			time.Sleep(time.Second)
		}
	}()
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

// watchGuide follows XInput pads' Guide button while TV mode is on: the
// TV's controller reaches the PC as one, through the streaming host.
func (a *App) watchGuide() {
	was := false
	for {
		time.Sleep(60 * time.Millisecond)
		a.mu.Lock()
		tv := a.tv
		a.mu.Unlock()
		if !tv {
			return
		}
		down := win.XInputGuideDown()
		if down && !was {
			a.ui(a.tvHome)
		}
		was = down
	}
}

// tvHome is the controller's Home button in TV mode. Over a game it opens
// TV mode's menu, as a console's does, and closes it again; with no game it
// opens Big Picture's own menu. Call it on the window's thread.
func (a *App) tvHome() {
	a.mu.Lock()
	tv, playing, on := a.tv, a.hidFor, a.tvMenuOn
	a.mu.Unlock()
	if !tv {
		return
	}
	switch {
	case playing == "":
		a.emit("tvHome", map[string]any{})
	case on:
		a.tvResume()
	default:
		a.mu.Lock()
		a.tvMenuOn = true
		a.mu.Unlock()
		a.tvBack()
		a.emit("tvMenu", map[string]any{"on": true, "key": playing})
	}
}

// tvResume closes TV mode's menu and brings the game back to the front.
func (a *App) tvResume() {
	a.mu.Lock()
	playing := a.hidFor
	a.tvMenuOn = false
	a.mu.Unlock()
	a.emit("tvMenu", map[string]any{"on": false})
	if playing != "" {
		a.tvGameUp(playing)
	}
}

// apiTVResume is Resume in TV mode's menu, and Play on the game running.
func (a *App) apiTVResume() { a.ui(a.tvResume) }

// apiTVStay is Lootnest in TV mode's menu: the menu closes, Lootnest stays
// in front, and the game keeps running behind it.
func (a *App) apiTVStay() {
	a.mu.Lock()
	a.tvMenuOn = false
	a.mu.Unlock()
}

// apiTVCloseGame asks the game TV mode started to quit, as closing its
// window would; Lootnest comes back when it's gone (tick).
func (a *App) apiTVCloseGame() error {
	a.mu.Lock()
	playing := a.hidFor
	a.tvMenuOn = false
	a.mu.Unlock()
	if playing == "" {
		return nil
	}
	proc, pid := a.procOf(playing), a.pidOf(playing)
	closed := 0
	if pid != 0 {
		closed = win.CloseWindowsOfPID(pid)
	} else if proc != "" {
		closed = win.CloseProcessWindows(proc)
	}
	if closed == 0 {
		return errors.New("Lootnest couldn't close the game. Quit it from its own menu")
	}
	a.logf("Asked %s to close from TV mode's menu", proc)
	return nil
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
	return core.FindMediaApps(core.MediaPaths{
		AppData: os.Getenv("APPDATA"), LocalAppData: os.Getenv("LOCALAPPDATA"),
		ProgramFiles: os.Getenv("ProgramFiles"), ProgramFilesX86: os.Getenv("ProgramFiles(x86)"),
		DataDir: DataDir(),
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
	win.AllowAnyForeground()
	if app.ByPID {
		if old := a.pidOf(key); old != 0 && win.ProcessAlive(old) {
			r.pid = old
		} else {
			pid, err := win.StartTracked(app.Exe, app.Args, filepath.Dir(app.Exe))
			if err != nil {
				return err
			}
			r.pid = pid
		}
	} else if procs, err := win.ProcessNames(); err != nil || !procs[app.Proc] {
		if err := win.StartDetached(app.Exe, app.Args, filepath.Dir(app.Exe)); err != nil {
			return err
		}
	}
	foundRunsMu.Lock()
	foundRuns[key] = r
	foundRunsMu.Unlock()
	a.logf("Starting %s from TV mode", app.Name)
	a.tvGameStarting(key, app.Name)
	return nil
}
