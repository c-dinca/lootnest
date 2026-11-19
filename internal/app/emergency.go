package app

import (
	"os"
	"sync/atomic"
	"time"

	"lootnest/internal/plat"
)

// The emergency key (plat.EmergencyKey, Ctrl+Shift+Q), the owner's ask of
// 19 Nov 2026: when the TV or Lootnest stops answering in TV mode, the PC
// must still get out of it from its keyboard. It's held while TV mode or
// Big Picture is on, on a thread of its own, and pressing it never waits
// on Lootnest's window: the mouse is let go, the taskbar comes back, the
// TV's stream ends (the holder is ended, so the engine takes the TV's
// display away and the PC's displays come back), then Lootnest quits, or
// is ended when its window doesn't answer within emergencyWait.

const emergencyWait = 5 * time.Second

var (
	consoleArmed atomic.Bool   // TV mode or Big Picture is on
	tvHolderPID  atomic.Uint32 // the "--tv" holder while TV mode is on
	emergencyKey atomic.Bool   // the key could be held
	emergencyRan atomic.Bool
)

// armConsole notes whether TV mode or Big Picture is on, and the holder.
func armConsole(on bool, holder uint32) {
	consoleArmed.Store(on)
	tvHolderPID.Store(holder)
}

// watchEmergency starts holding the key whenever the console is armed.
func (a *App) watchEmergency() {
	plat.WatchEmergencyKey(consoleArmed.Load, a.emergency, func(held bool) {
		emergencyKey.Store(held)
		if !held {
			a.logf("%s is taken by another program; TV mode is left from the bar on the PC's screen or Ctrl+Shift+F11", plat.EmergencyKey)
		}
	})
}

// emergency is the key pressed. It runs off every Lootnest thread but its
// own, so a window that doesn't answer can't hold it up.
func (a *App) emergency() {
	if emergencyRan.Swap(true) {
		return
	}
	a.logf("%s pressed: leaving TV mode and quitting", plat.EmergencyKey)
	plat.ClipCursor(nil)
	tvTaskbar(true)
	plat.HideTVBar()
	if pid := tvHolderPID.Load(); pid != 0 {
		if plat.KillPID(pid) {
			a.logf("Ended the TV's stream (process %d)", pid)
		}
	}
	done := make(chan struct{})
	a.ui(func() {
		a.leaveTV()
		a.quit(true)
		close(done)
	})
	select {
	case <-done:
		// The window closes and Lootnest ends as it always does; if it
		// still hasn't a while later, it's ended below.
		time.Sleep(3 * emergencyWait)
	case <-time.After(emergencyWait):
	}
	a.logf("Lootnest didn't close after %s; ending it", plat.EmergencyKey)
	os.Exit(3)
}
