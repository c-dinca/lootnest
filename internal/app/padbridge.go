package app

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/plat"
)

// A PlayStation controller as an Xbox one, for games that only know Xbox
// pads (the owner, 16 Nov 2026: as DS4Windows does, inside Lootnest, with
// ViGEmBus). While a game Lootnest follows runs and Settings.PadAsXbox is
// on, Lootnest reads each PlayStation pad's input reports and feeds a
// virtual Xbox 360 pad of its own through the ViGEmBus driver, when that
// driver is already on the PC: one Xbox pad for each PlayStation pad, so
// two people with two DualSenses are two players (the owner, 23 Nov 2026).
// A game that knows PlayStation pads itself (Steam's DualShock and
// DualSense categories, its store details, Sony's pad library in its
// folder: core/padknows.go) gets them as they are, since it would see each
// one twice and one pad would play for two people. A game's launch options
// can say otherwise both ways (KeepPad, XboxPad). Lootnest ships no
// driver, hides nothing, and sends nothing to the pads.

type padBridge struct {
	stop chan struct{}
	done chan struct{}
	// Under App.mu: which pads games want as Xbox ones now, the game that
	// keeps them as they are (for the note), and since when no game wants
	// the bridge (it stays a little, so a game's process seen missing for
	// one look doesn't unplug and plug the pads again).
	ds, ds4 bool
	kept    string
	idle    time.Time
}

// padHold is how long the pads stay after the last game that wanted them.
const padHold = 20 * time.Second

// padKnowsKept is what each running game knows of PlayStation pads, read
// once per play (appinfo.vdf is large); forgotten when no game runs.
var padKnowsKept = struct {
	sync.Mutex
	m map[string]core.PSKnows
}{m: map[string]core.PSKnows{}}

// padTick starts the bridge when a game that wants it runs and the setting
// is on, tells it which pads to turn into Xbox ones, and takes it away when
// no game wants it. It runs with the game watch.
func (a *App) padTick(running []achRun) {
	a.mu.Lock()
	on := a.cfg != nil && a.cfg.Settings.PadAsXbox && !a.safe
	var opts map[string]core.LaunchOptions
	if on {
		opts = a.cfg.LaunchOpts
	}
	behind := a.conGame != ""
	a.mu.Unlock()
	if len(running) == 0 {
		padKnowsKept.Lock()
		clear(padKnowsKept.m)
		padKnowsKept.Unlock()
	}
	ds, ds4, kept := false, false, ""
	if on {
		for _, r := range running {
			o := opts[r.key]
			var k core.PSKnows
			switch {
			case o.KeepPad:
				k = core.PSKnows{DualShock: true, DualSense: true, Why: "you"}
			case !o.XboxPad:
				k = a.psKnows(r)
			}
			ds = ds || !k.DualSense
			ds4 = ds4 || !k.DualShock
			if (k.DualSense || k.DualShock) && kept == "" {
				kept = r.name
			}
		}
		if len(running) == 0 && behind { // a game Lootnest stands behind but can't follow by its process
			ds, ds4 = true, true
		}
	}
	want := ds || ds4
	a.mu.Lock()
	b := a.pad
	switch {
	case want && b == nil:
		b = &padBridge{stop: make(chan struct{}), done: make(chan struct{}), ds: ds, ds4: ds4, kept: kept}
		a.pad = b
		a.mu.Unlock()
		go a.runPad(b)
		return
	case want:
		b.ds, b.ds4, b.kept, b.idle = ds, ds4, kept, time.Time{}
	case b != nil && on && len(running) == 0 && (b.idle.IsZero() || time.Since(b.idle) < padHold):
		if b.idle.IsZero() {
			b.idle = time.Now()
		}
	case b != nil:
		a.pad = nil
		a.padNote = ""
		a.mu.Unlock()
		close(b.stop)
		return
	default:
		a.padNote = ""
		if on && kept != "" {
			a.padNote = kept + " knows PlayStation controllers itself, so they're left as they are"
		}
	}
	a.mu.Unlock()
}

// psKnows is what a running game knows of PlayStation pads, read once.
func (a *App) psKnows(r achRun) core.PSKnows {
	padKnowsKept.Lock()
	k, ok := padKnowsKept.m[r.key]
	padKnowsKept.Unlock()
	if ok {
		return k
	}
	k = a.readPSKnows(r)
	padKnowsKept.Lock()
	padKnowsKept.m[r.key] = k
	padKnowsKept.Unlock()
	switch {
	case k.DualSense && k.DualShock:
		a.logf("Controller as Xbox: %s knows PlayStation controllers itself (%s), so they're left as they are", r.name, k.Why)
	case k.DualSense || k.DualShock:
		a.logf("Controller as Xbox: %s knows only the %s itself (%s)", r.name, map[bool]string{true: "DualSense", false: "DualShock 4"}[k.DualSense], k.Why)
	default:
		a.logf("Controller as Xbox: nothing says %s knows PlayStation controllers, so they're shown to it as Xbox ones", r.name)
	}
	return k
}

// readPSKnows asks Steam's appinfo.vdf on this PC, the game's store details
// already kept, and its folder. Nothing goes online.
func (a *App) readPSKnows(r achRun) core.PSKnows {
	id := r.appID
	if id <= 0 {
		id = core.FolderAppID(r.dir)
	}
	if id <= 0 {
		id = knownStoreAppID(r.name)
	}
	var k core.PSKnows
	add := func(n core.PSKnows) {
		k.DualShock = k.DualShock || n.DualShock
		k.DualSense = k.DualSense || n.DualSense
		if k.Why == "" {
			k.Why = n.Why
		}
	}
	if id > 0 {
		add(core.SteamPSPads(a.env.SteamPath, map[int]bool{id: true})[id])
		if d, ok := core.LoadDetails(DataDir(), id); ok {
			add(core.DetailsPS(d))
		}
	}
	add(core.FolderPS(r.dir))
	switch k.Why {
	case "steam":
		k.Why = "Steam says so"
	case "details":
		k.Why = "its store details say so"
	case "files":
		k.Why = "Sony's controller library is in its folder"
	}
	return k
}

// padSay logs a state of the pads once, and keeps it for Settings.
func (a *App) padSay(note string) {
	a.mu.Lock()
	same := a.padNote == note
	a.padNote = note
	a.mu.Unlock()
	if !same {
		a.logf("Controller as Xbox: %s", note)
	}
}

// padState is what the pads are doing now, for Settings, Controllers.
func (a *App) padState() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.padNote
}

// psPad is a PlayStation pad present, by its HID path.
type psPad struct {
	path                 string
	dualsense, bluetooth bool
}

// psPadsPresent are the PlayStation pads present, in the order of their
// paths, so the same pads plug in in the same order each time.
func psPadsPresent() []psPad {
	var out []psPad
	seen := map[string]bool{}
	for _, p := range plat.HIDPaths() {
		low := strings.ToLower(p)
		if seen[low] {
			continue
		}
		seen[low] = true
		if pid, bt, is := core.PSPadPath(p); is {
			ds, _ := core.PSPad(pid)
			out = append(out, psPad{p, ds, bt})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].path < out[j].path })
	return out
}

// padBack is a feeder's virtual pad coming back: its pad went, it was
// told to stop, or the pad couldn't be opened (err).
type padBack struct {
	path string
	x    *plat.ViGEmPad
	err  error
}

// padFeeder feeds one virtual pad from one PlayStation pad.
type padFeeder struct {
	pad      psPad
	stop     chan struct{}
	stopping bool
}

// padResting is a virtual pad whose PlayStation pad went: kept plugged a
// while for the same pad, so a Bluetooth pad that drops and comes back
// keeps its player in the game.
type padResting struct {
	x     *plat.ViGEmPad
	since time.Time
}

const (
	padRest     = 30 * time.Second
	padRetry    = 5 * time.Second
	padScan     = 2 * time.Second
	padsMax     = 8 // XInput has four slots; DirectInput and Windows.Gaming.Input games take more
	padBackSize = padsMax * 2
)

// runPad looks for PlayStation pads every two seconds and gives each one
// games want as an Xbox pad a virtual Xbox pad of its own, until told to
// stop. The virtual pads go with the bridge.
func (a *App) runPad(b *padBridge) {
	defer close(b.done)
	if !plat.ViGEmPresent() {
		a.padSay("the ViGEmBus driver isn't on this PC")
		<-b.stop
		return
	}
	feeding := map[string]*padFeeder{}
	resting := map[string]padResting{}
	failed := map[string]time.Time{}
	back := make(chan padBack, padBackSize)
	wanted := func(p psPad) bool {
		a.mu.Lock()
		defer a.mu.Unlock()
		if p.dualsense {
			return b.ds
		}
		return b.ds4
	}
	takeBack := func(m padBack) {
		f := feeding[m.path]
		delete(feeding, m.path)
		switch {
		case m.err != nil:
			m.x.Close()
			failed[m.path] = time.Now()
			a.padSay(sentence(m.err))
		case f != nil && !wanted(f.pad):
			m.x.Close()
		default:
			resting[m.path] = padResting{m.x, time.Now()}
		}
	}
	scan := func() {
		present := psPadsPresent()
		here := map[string]bool{}
		for _, p := range present {
			here[p.path] = true
		}
		for path, f := range feeding {
			if !f.stopping && (!here[path] || !wanted(f.pad)) {
				f.stopping = true
				close(f.stop) // its pad comes back on the channel
			}
		}
		for path, r := range resting {
			if time.Since(r.since) > padRest {
				r.x.Close()
				delete(resting, path)
			}
		}
		for _, p := range present {
			if feeding[p.path] != nil || !wanted(p) || time.Since(failed[p.path]) < padRetry || len(feeding) >= padsMax {
				continue
			}
			x := resting[p.path].x
			delete(resting, p.path)
			if x == nil {
				var err error
				if x, err = plat.NewViGEmPad(); err != nil {
					a.padSay(sentence(err))
					failed[p.path] = time.Now()
					continue
				}
			}
			f := &padFeeder{pad: p, stop: make(chan struct{})}
			feeding[p.path] = f
			go a.feedPad(f, x, back)
		}
		a.padSay(padNoteOf(feeding, present, b, a))
	}
	scan()
	tick := time.NewTicker(padScan)
	defer tick.Stop()
	for {
		select {
		case <-b.stop:
			for _, f := range feeding {
				if !f.stopping {
					close(f.stop)
				}
			}
			for len(feeding) > 0 {
				m := <-back
				delete(feeding, m.path)
				m.x.Close()
			}
			for _, r := range resting {
				r.x.Close()
			}
			return
		case m := <-back:
			takeBack(m)
		case <-tick.C:
			scan()
		}
	}
}

// padNoteOf says what the pads are doing, for the log and Settings.
func padNoteOf(feeding map[string]*padFeeder, present []psPad, b *padBridge, a *App) string {
	if len(present) == 0 {
		return "no PlayStation controller is connected"
	}
	ds, ds4 := 0, 0
	for _, f := range feeding {
		if f.pad.dualsense {
			ds++
		} else {
			ds4++
		}
	}
	a.mu.Lock()
	kept := b.kept
	a.mu.Unlock()
	if ds+ds4 == 0 {
		if kept != "" {
			return kept + " knows PlayStation controllers itself, so they're left as they are"
		}
		return "your PlayStation controllers are left as they are"
	}
	var parts []string
	for _, c := range []struct {
		n    int
		name string
	}{{ds, "DualSense"}, {ds4, "DualShock 4"}} {
		switch {
		case c.n == 1:
			parts = append(parts, "your "+c.name)
		case c.n > 1:
			parts = append(parts, fmt.Sprintf("your %d %ss", c.n, c.name))
		}
	}
	if ds+ds4 == 1 {
		return "games see " + strings.Join(parts, " and ") + " as an Xbox controller"
	}
	return "games see " + strings.Join(parts, " and ") + " as Xbox controllers, one each"
}

// feedPad reads one PlayStation pad and feeds its virtual pad until the pad
// goes or it's told to stop, then hands the virtual pad back.
func (a *App) feedPad(f *padFeeder, x *plat.ViGEmPad, back chan<- padBack) {
	r, err := plat.OpenPad(f.pad.path)
	if err != nil {
		back <- padBack{f.pad.path, x, err}
		return
	}
	var once sync.Once
	shut := func() { once.Do(r.Close) }
	gone := make(chan struct{})
	go func() { // a Read that's waiting ends when the feeder is stopped
		select {
		case <-f.stop:
			shut()
		case <-gone:
		}
	}()
	buf := make([]byte, 512)
	var last core.PadState
	sent := false
	for {
		n, err := r.Read(buf)
		if err != nil {
			break
		}
		if s, ok := core.PSToXbox(f.pad.dualsense, f.pad.bluetooth, buf[:n]); ok && (s != last || !sent) {
			last, sent = s, true
			if err := x.Submit(s.XUSBReport()); err != nil {
				a.padSay(sentence(err))
			}
		}
	}
	close(gone)
	shut()
	_ = x.Submit(core.PadState{}.XUSBReport()) // nothing held on a pad nobody feeds
	back <- padBack{f.pad.path, x, nil}
}
