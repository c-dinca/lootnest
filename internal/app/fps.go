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
// administrator or a member of Performance Log Users; apiFPSAllow adds the
// account once, with Windows' prompt, and it counts from the next sign-in.
// ---------------------------------------------------------------------------

const hotkeyFPS = 0x4C46 // "LF"

var fpsRun struct {
	mu        sync.Mutex
	playing   bool      // a game runs (the watch loop says)
	loop      bool      // fpsLoop runs
	since     time.Time // when it was last turned on, to show "--" a moment
	exclusive string    // the last game seen in exclusive fullscreen
	denied    bool      // Windows refused the session since Lootnest started
}

// FPSState is what the page shows about the counter.
type FPSState struct {
	On        bool   `json:"on"`
	Corner    string `json:"corner"`
	Access    string `json:"access"` // "ok", "ask", "signout" or "unsupported"
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
	case plat.FramesAllowed():
		st.Access = "ok"
	case s.FPSGroupAdded:
		st.Access = "signout"
	default:
		st.Access = "ask"
	}
	fpsRun.mu.Lock()
	st.Exclusive = fpsRun.exclusive
	if fpsRun.denied && st.Access == "ok" { // the token says yes, Windows said no
		st.Access = "ask"
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

// apiFPSAllow puts this account into Performance Log Users, with Windows'
// prompt. It counts from the next time the user signs in to Windows.
func (a *App) apiFPSAllow() (FPSState, error) {
	if runtime.GOOS != "windows" {
		return a.fpsState(), errors.New("The FPS counter is only on Windows for now.")
	}
	script, err := core.PerfLogUsersScript(plat.UserSID())
	if err != nil {
		return a.fpsState(), err
	}
	ps := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	args := "-NoProfile -NonInteractive -ExecutionPolicy Bypass -WindowStyle Hidden -EncodedCommand " + core.EncodePowerShell(script)
	code, err := plat.RunElevatedAndWait(a.hwnd, ps, args)
	switch {
	case errors.Is(err, plat.ErrCancelledByUser):
		return a.fpsState(), errors.New("Nothing changed: Windows' prompt was cancelled.")
	case err != nil:
		return a.fpsState(), errors.New("Windows couldn't be asked: " + err.Error())
	case code != 0:
		a.logf("FPS counter: adding this account to Performance Log Users ended with code %d", code)
		return a.fpsState(), errors.New("Windows didn't add your account to Performance Log Users, so nothing changed.")
	}
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	cfg.Settings.FPSGroupAdded = true
	if err := a.saveConfig(cfg); err != nil {
		return a.fpsState(), err
	}
	a.logf("FPS counter: this account is in Performance Log Users; it counts from the next sign-in to Windows")
	st := a.fpsState()
	a.emit("fps", map[string]any{"state": st})
	return st, nil
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
	count := core.NewFrameCounter(plat.PerfFreq())
	tr, err := plat.StartFrameTrace(count.Add)
	if err != nil {
		if errors.Is(err, plat.ErrFramesNotAllowed) {
			fpsRun.mu.Lock()
			told := fpsRun.denied
			fpsRun.denied = true
			fpsRun.mu.Unlock()
			if !told {
				st := a.fpsState()
				a.logf("FPS counter: Windows doesn't let this account read frame timings (%s)", st.Access)
				line := "Allow it once in Lootnest: Settings, Games and swaps."
				if st.Access == "signout" {
					line = "Sign out of Windows and in again to finish allowing it."
				}
				plat.ShowToast(plat.Toast{Kicker: "FPS counter", Title: "Needs your permission first", Line: line})
				a.emit("fps", map[string]any{"state": st})
			}
		} else {
			a.logf("FPS counter couldn't listen to frames: %v", err)
		}
		// Off again, so the hotkey's next press tries afresh.
		a.mu.Lock()
		cfg := a.cfg.Clone()
		a.mu.Unlock()
		cfg.Settings.FPSCounter = false
		if a.saveConfig(cfg) == nil {
			a.emit("fps", map[string]any{"state": a.fpsState()})
		}
		return
	}
	a.logf("FPS counter: listening to frames")
	defer func() {
		tr.Stop()
		plat.HideFPS()
		a.logf("FPS counter: stopped listening")
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
		if pid == self { // Lootnest itself, its Home menu over a game included
			plat.HideFPS()
			continue
		}
		if rate, ok := count.Rate(pid, plat.PerfNow()); ok {
			plat.ShowFPS(strconv.Itoa(rate), corner)
		} else if fresh {
			plat.ShowFPS("", corner) // "--" for a moment, so the key is seen to work
		} else {
			plat.HideFPS()
		}
		if !noted && plat.ExclusiveFullscreen() {
			noted = true
			name := "A game"
			if r, _, ok := a.runningNow(); ok {
				name = r.name
			}
			a.logf("FPS counter: %s runs in exclusive fullscreen, where no other window shows; borderless or windowed shows the counter", name)
			fpsRun.mu.Lock()
			fpsRun.exclusive = name
			fpsRun.mu.Unlock()
			a.emit("fps", map[string]any{"state": a.fpsState()})
		}
	}
}
