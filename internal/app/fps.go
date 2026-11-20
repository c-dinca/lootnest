package app

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/plat"
)

// ---------------------------------------------------------------------------
// The FPS counter (core/fps.go, win/etw.go, win/fpsbox.go): while a game
// runs and the counter is on, Lootnest listens to the frames Windows reports
// and shows how many the program in front draws a second, in a corner of
// its screen. Ctrl+Shift+F10 (held only while a game runs), the Home menu
// over a game and Settings turn it on and off. Listening takes an
// administrator's full rights, so unless this sign-in has them, the frames
// come from Lootnest's frame reader (app/framereader.go), which the user
// allows once with Windows' prompt and which works at once. In games
// without an anti-cheat the counter is drawn inside the game instead, as
// Steam's is, and needs nothing allowed (app/overlay.go).
// ---------------------------------------------------------------------------

const hotkeyFPS = 0x4C46 // "LF"

var fpsRun struct {
	mu        sync.Mutex
	playing   bool      // a game runs (the watch loop says)
	loop      bool      // fpsLoop runs
	since     time.Time // when it was last turned on, to show "--" a moment
	exclusive string    // the last game seen in exclusive fullscreen
	denied    bool      // Windows refused the session since Lootnest started
	task      bool      // the frame reader's task is there (frameTaskThere)
	taskAt    time.Time // when that was asked
	old       bool      // the reader in Program Files speaks another protocol
}

// FPSState is what the page shows about the counter.
type FPSState struct {
	On        bool   `json:"on"`
	Corner    string `json:"corner"`
	Access    string `json:"access"`           // "ok", "ask", "update" or "unsupported"
	Reader    bool   `json:"reader,omitempty"` // the frame reader is allowed (it can be taken away)
	Exclusive string `json:"exclusive,omitempty"`
}

func (a *App) fpsState() FPSState {
	a.mu.Lock()
	s := a.cfg.Settings
	a.mu.Unlock()
	st := FPSState{On: s.FPSCounter, Corner: core.CleanFPSCorner(s.FPSCorner)}
	switch {
	case runtime.GOOS != "windows":
		st.Access = "unsupported"
	default:
		st.Reader = a.frameTaskThere(false)
		switch {
		case plat.FramesAllowed() || st.Reader:
			st.Access = "ok"
		default:
			st.Access = "ask"
		}
	}
	fpsRun.mu.Lock()
	st.Exclusive = fpsRun.exclusive
	if fpsRun.denied && st.Access == "ok" && !st.Reader { // the token says yes, Windows said no
		st.Access = "ask"
	}
	if fpsRun.old && st.Reader {
		st.Access = "update"
	}
	fpsRun.mu.Unlock()
	return st
}

func (a *App) apiFPSState() FPSState { return a.fpsState() }

// apiFPSSet turns the counter on or off, from Settings or the Home menu.
func (a *App) apiFPSSet(on bool) (FPSState, error) {
	if runtime.GOOS != "windows" {
		return a.fpsState(), errors.New("The FPS counter is only on Windows for now.")
	}
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	cfg.Settings.FPSCounter = on
	if err := a.saveConfig(cfg); err != nil {
		return a.fpsState(), err
	}
	if on {
		fpsRun.mu.Lock()
		fpsRun.since = time.Now()
		fpsRun.denied = false
		fpsRun.mu.Unlock()
	}
	a.fpsFollow()
	st := a.fpsState()
	a.emit("fps", map[string]any{"state": st})
	return st, nil
}

// fpsToggle is Ctrl+Shift+F10.
func (a *App) fpsToggle() {
	a.mu.Lock()
	on := !a.cfg.Settings.FPSCounter
	a.mu.Unlock()
	if _, err := a.apiFPSSet(on); err != nil {
		a.logf("FPS counter: %v", err)
		return
	}
	a.logf("FPS counter %s: Ctrl+Shift+F10", map[bool]string{true: "on", false: "off"}[on])
}

// apiFPSAllow puts Lootnest's frame reader in place, with Windows' prompt
// once: a copy of lootnest.exe in Program Files and a task that starts it
// with the account's highest rights. It works at once, with no sign-out.
func (a *App) apiFPSAllow() (FPSState, error) {
	if runtime.GOOS != "windows" {
		return a.fpsState(), errors.New("The FPS counter is only on Windows for now.")
	}
	script, err := core.FrameTaskScript(a.exePath, plat.UserSID())
	if err != nil {
		return a.fpsState(), err
	}
	if err := a.runFrameScript(script); err != nil {
		return a.fpsState(), err
	}
	fpsRun.mu.Lock()
	fpsRun.denied, fpsRun.old = false, false
	fpsRun.mu.Unlock()
	if !a.frameTaskThere(true) {
		a.logf("FPS counter: the frame reader's task isn't in Task Scheduler after allowing it")
		return a.fpsState(), errors.New("Windows didn't keep Lootnest's frame reader, so nothing changed.")
	}
	a.logf("FPS counter: the frame reader is allowed (Task Scheduler, %s)", frameReaderPath())
	st := a.fpsState()
	a.emit("fps", map[string]any{"state": st})
	a.fpsFollow()
	return st, nil
}

// apiFPSRemove takes the frame reader and its task away (Windows asks).
func (a *App) apiFPSRemove() (FPSState, error) {
	if runtime.GOOS != "windows" {
		return a.fpsState(), nil
	}
	if err := a.runFrameScript(core.FrameUntaskScript()); err != nil {
		return a.fpsState(), err
	}
	a.frameTaskThere(true)
	a.logf("FPS counter: the frame reader was taken away")
	st := a.fpsState()
	a.emit("fps", map[string]any{"state": st})
	return st, nil
}

func (a *App) runFrameScript(script string) error {
	ps := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	args := "-NoProfile -NonInteractive -ExecutionPolicy Bypass -WindowStyle Hidden -EncodedCommand " + core.EncodePowerShell(script)
	code, err := plat.RunElevatedAndWait(a.hwnd, ps, args)
	switch {
	case errors.Is(err, plat.ErrCancelledByUser):
		return errors.New("Nothing changed: Windows' prompt was cancelled.")
	case err != nil:
		return errors.New("Windows couldn't be asked: " + err.Error())
	case code != 0:
		a.logf("FPS counter: the frame reader's script ended with code %d", code)
		return errors.New("Windows didn't take the change, so nothing changed.")
	}
	return nil
}

// fpsTick runs with the hotkeys, every few seconds: Ctrl+Shift+F10 is held
// only while a game runs, and the counter listens only then.
func (a *App) fpsTick(running []achRun) {
	a.mu.Lock()
	playing := len(running) > 0 || a.conGame != ""
	a.mu.Unlock()
	if runtime.GOOS != "windows" {
		return
	}
	a.holdHotkey(hotkeyFPS, plat.VK_F10, playing, "Ctrl+Shift+F10 is taken by another program; the FPS counter still turns on from Settings or the Home menu over a game")
	fpsRun.mu.Lock()
	fpsRun.playing = playing
	fpsRun.mu.Unlock()
	a.fpsFollow()
}

// fpsWanted: on, a game runs, and not in safe mode.
func (a *App) fpsWanted() bool {
	a.mu.Lock()
	on := a.cfg.Settings.FPSCounter
	a.mu.Unlock()
	fpsRun.mu.Lock()
	playing := fpsRun.playing
	fpsRun.mu.Unlock()
	return on && playing && !a.safe
}

// fpsFollow starts the loop when the counter is wanted; the loop ends by
// itself when it isn't.
func (a *App) fpsFollow() {
	if !a.fpsWanted() {
		return
	}
	fpsRun.mu.Lock()
	start := !fpsRun.loop
	fpsRun.loop = true
	fpsRun.mu.Unlock()
	if start {
		go a.fpsLoop()
	}
}

func (a *App) fpsLoop() {
	defer func() {
		fpsRun.mu.Lock()
		fpsRun.loop = false
		fpsRun.mu.Unlock()
	}()
	// Frames come from the DLL inside story games (app/overlay.go) and,
	// for every other game, from what Windows reports, which is started the
	// first time a game needs it.
	var src frameSource
	tried := false
	outside := func() frameSource {
		if tried {
			return src
		}
		tried = true
		var err error
		if src, err = a.startFrames(); err == nil {
			a.logf("FPS counter: listening to frames through %s", src.Kind())
			return src
		}
		if errors.Is(err, plat.ErrFramesNotAllowed) {
			fpsRun.mu.Lock()
			told := fpsRun.denied
			fpsRun.denied = true
			fpsRun.mu.Unlock()
			if !told {
				st := a.fpsState()
				a.logf("FPS counter: Windows doesn't let this account read frame timings (%s); only games without an anti-cheat show it", st.Access)
				plat.ShowToast(plat.Toast{Kicker: "FPS counter", Title: "Needs your permission for this game",
					Line: "Games with an anti-cheat need it allowed once: Lootnest's Settings, Games and swaps."})
				a.emit("fps", map[string]any{"state": st})
			}
		} else {
			a.logf("FPS counter couldn't listen to frames: %v", err)
		}
		return nil
	}
	defer func() {
		overlaysOff()
		if src != nil {
			src.Stop()
			a.logf("FPS counter: stopped listening")
		}
		plat.HideFPS()
	}()
	self := uint32(os.Getpid())
	noted := false
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
	for range t.C {
		if !a.fpsWanted() {
			return
		}
		a.mu.Lock()
		corner := core.CleanFPSCorner(a.cfg.Settings.FPSCorner)
		a.mu.Unlock()
		fpsRun.mu.Lock()
		fresh := time.Since(fpsRun.since) < 3*time.Second
		fpsRun.mu.Unlock()
		pid := plat.ForegroundPID()
		a.overlayConsider(pid)
		inGame, dllFPS, counted := a.overlayTick(pid, pid != self)
		switch {
		case pid == self: // Lootnest itself, its Home menu over a game included
			plat.HideFPS()
			continue
		case inGame: // drawn inside the game
			plat.HideFPS()
			continue
		case counted:
			plat.ShowFPS(strconv.Itoa(dllFPS), corner)
			continue
		}
		if s := outside(); s != nil {
			if rate, ok := s.Rate(pid); ok {
				plat.ShowFPS(strconv.Itoa(rate), corner)
			} else if fresh {
				plat.ShowFPS("", corner) // "--" for a moment, so the key is seen to work
			} else {
				plat.HideFPS()
			}
		} else if fresh {
			plat.ShowFPS("", corner)
		} else {
			plat.HideFPS()
		}
		if !noted && plat.ExclusiveFullscreen() {
			noted = true
			name := "A game"
			if r, _, ok := a.runningNow(); ok {
				name = r.name
			}
			a.logf("FPS counter: %s runs in exclusive fullscreen and has the counter outside the game, where no other window shows; borderless or windowed shows it", name)
			fpsRun.mu.Lock()
			fpsRun.exclusive = name
			fpsRun.mu.Unlock()
			a.emit("fps", map[string]any{"state": a.fpsState()})
		}
	}
}
