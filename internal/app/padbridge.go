package app

import (
	"time"

	"lootnest/internal/core"
	"lootnest/internal/plat"
)

// A PlayStation controller as an Xbox one, for games that only know Xbox
// pads (the owner, 16 Nov 2026: as DS4Windows does, inside Lootnest, with
// ViGEmBus). While a game Lootnest follows runs and Settings.PadAsXbox is
// on, Lootnest reads the pad's input reports and feeds a virtual Xbox 360
// pad through the ViGEmBus driver, when that driver is already on the PC.
// Lootnest ships no driver, hides nothing, and sends nothing to the pad.
// A game that knows the PlayStation pad itself would see two pads, so a
// game's launch options can leave the pad as it is (KeepPad).

type padBridge struct {
	stop chan struct{}
	done chan struct{}
}

// padTick starts the pad when a game is running and the setting is on,
// and takes it away when no game is. It runs with the game watch.
func (a *App) padTick(running []achRun) {
	a.mu.Lock()
	want := false
	if a.cfg != nil && a.cfg.Settings.PadAsXbox && !a.safe {
		for _, r := range running {
			if !a.cfg.LaunchOpts[r.key].KeepPad {
				want = true
			}
		}
		if len(running) == 0 && a.conGame != "" { // a game Lootnest stands behind but can't follow by its process
			want = true
		}
	}
	b := a.pad
	switch {
	case want && b == nil:
		b = &padBridge{stop: make(chan struct{}), done: make(chan struct{})}
		a.pad = b
		a.mu.Unlock()
		go a.runPad(b)
		return
	case !want && b != nil:
		a.pad = nil
		a.padNote = ""
		a.mu.Unlock()
		close(b.stop)
		return
	case !want:
		a.padNote = ""
	}
	a.mu.Unlock()
}

// padSay logs a state of the pad once, and keeps it for Settings.
func (a *App) padSay(note string) {
	a.mu.Lock()
	same := a.padNote == note
	a.padNote = note
	a.mu.Unlock()
	if !same {
		a.logf("Controller as Xbox: %s", note)
	}
}

// padState is what the pad is doing now, for Settings, Controllers.
func (a *App) padState() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.padNote
}

// psPadPath is the first PlayStation pad present.
func psPadPath() (path string, dualsense, bluetooth, ok bool) {
	for _, p := range plat.HIDPaths() {
		if pid, bt, is := core.PSPadPath(p); is {
			ds, _ := core.PSPad(pid)
			return p, ds, bt, true
		}
	}
	return "", false, false, false
}

// runPad plugs the virtual pad in and feeds it from the PlayStation pad
// until told to stop. A pad that isn't there, or goes, is looked for
// again every two seconds; the virtual pad rests in the meantime.
func (a *App) runPad(b *padBridge) {
	defer close(b.done)
	x, err := plat.NewViGEmPad()
	if err != nil {
		a.padSay(sentence(err))
		<-b.stop
		return
	}
	defer x.Close()
	wait := func(d time.Duration) bool { // false once it's told to stop
		select {
		case <-b.stop:
			return false
		case <-time.After(d):
			return true
		}
	}
	buf := make([]byte, 512)
	for {
		path, dualsense, bluetooth, ok := psPadPath()
		if !ok {
			a.padSay("no PlayStation controller is connected")
			if !wait(2 * time.Second) {
				return
			}
			continue
		}
		r, err := plat.OpenPad(path)
		if err != nil {
			a.padSay(sentence(err))
			if !wait(2 * time.Second) {
				return
			}
			continue
		}
		name := "DualShock 4"
		if dualsense {
			name = "DualSense"
		}
		a.padSay("games see your " + name + " as an Xbox controller")
		gone := make(chan struct{})
		go func() { // a Read that's waiting ends when the bridge stops
			select {
			case <-b.stop:
				r.Close()
			case <-gone:
			}
		}()
		var last core.PadState
		for {
			n, err := r.Read(buf)
			if err != nil {
				break
			}
			if s, ok := core.PSToXbox(dualsense, bluetooth, buf[:n]); ok && s != last {
				last = s
				if err := x.Submit(s.XUSBReport()); err != nil {
					a.padSay(sentence(err))
				}
			}
		}
		close(gone)
		_ = x.Submit(core.PadState{}.XUSBReport()) // nothing held on a pad nobody feeds
		select {
		case <-b.stop:
			return
		default:
			r.Close()
		}
		if !wait(time.Second) {
			return
		}
	}
}
