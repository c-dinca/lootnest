package app

import (
	"cmp"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/plat"
)

// Controllers' batteries (core/padbattery.go): read every 30 seconds, and
// at once when Settings, Controllers asks; a warning when one runs low or
// is charged, as a note in the window or over the game; and the battery
// saver, sent to a PlayStation pad when it should change. Never in safe
// mode.

const (
	batteryEvery = 30 * time.Second
	batteryRead  = 1500 * time.Millisecond
)

// padPower is one controller's battery as the loop keeps it.
type padPower struct {
	id, name                       string
	path                           string // a PlayStation pad's HID path
	dualsense, bluetooth, ps       bool
	bat                            core.PadBattery
	seen, switched, saver, saverOK bool
	warned                         int
}

// PadPowerView is a controller's battery for the page.
type PadPowerView struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Saver bool   `json:"saver"` // the battery saver is on for it now
	core.PadBattery
}

var (
	batteryPoke = make(chan struct{}, 1)
	batteryMu   sync.Mutex
	batteryNow  []PadPowerView
	saverOn     = map[string]bool{}
)

// batteryAsk reads the batteries again soon (Settings, Controllers opened,
// a pad connected, a setting changed).
func batteryAsk() {
	select {
	case batteryPoke <- struct{}{}:
	default:
	}
}

// batteries is the latest reading, for apiControllers.
func batteries() []PadPowerView {
	batteryMu.Lock()
	defer batteryMu.Unlock()
	return append([]PadPowerView{}, batteryNow...)
}

// saverHolds says whether the battery saver is on for a pad, so Lootnest's
// own Vibrate waits.
func saverHolds(id string) bool {
	batteryMu.Lock()
	defer batteryMu.Unlock()
	return saverOn[id]
}

func (a *App) batteryLoop() {
	if a.safe {
		return
	}
	pads := map[string]*padPower{}
	t := time.NewTimer(5 * time.Second)
	for {
		select {
		case <-t.C:
		case <-batteryPoke:
			if !t.Stop() {
				select {
				case <-t.C:
				default:
				}
			}
		}
		a.batteryLook(pads)
		t.Reset(batteryEvery)
	}
}

// batteryLook reads every controller's battery once.
func (a *App) batteryLook(pads map[string]*padPower) {
	a.mu.Lock()
	off, mode := a.cfg == nil || a.cfg.Settings.PadBatteryOff, ""
	if a.cfg != nil {
		mode = a.cfg.Settings.PadSaver
	}
	a.mu.Unlock()
	if off {
		for id, p := range pads {
			if p.saver {
				a.sendSaver(p, false)
			}
			delete(pads, id)
		}
		a.batteryPublish(pads)
		return
	}
	devs := plat.PadDevices()
	names := a.padNames()
	here := map[string]bool{}
	// PlayStation pads, from their own reports.
	for _, ps := range psPadsPresent() {
		inst := strings.ToLower(plat.HIDInstance(ps.path))
		id := ""
		for _, d := range devs {
			if len(d.Chain) > 0 && strings.ToLower(d.Chain[0]) == inst {
				id = core.PadID(d.Chain)
				break
			}
		}
		if id == "" || here[id] {
			continue
		}
		p := pads[id]
		if p == nil {
			p = &padPower{id: id, ps: true}
			pads[id] = p
		}
		p.path, p.dualsense, p.bluetooth = ps.path, ps.dualsense, ps.bluetooth
		p.name = cmp.Or(names[id], map[bool]string{true: "DualSense", false: "DualShock 4"}[ps.dualsense])
		bat, ok, short := readPSBattery(ps)
		if !ok && short && ps.bluetooth && !p.switched {
			// A Bluetooth pad sends its battery only in its full reports,
			// which it starts sending once asked for its calibration.
			p.switched = true
			if w, err := plat.OpenPadWrite(ps.path); err == nil {
				if err := w.Feature(0x05); err != nil {
					a.logf("Controller battery: %s didn't switch to its full reports: %v", p.name, err)
				}
				w.Close()
				time.Sleep(200 * time.Millisecond)
				bat, ok, _ = readPSBattery(ps)
			}
		}
		if !ok {
			continue
		}
		here[id] = true
		a.batteryNote(p, bat, mode)
	}
	// Xbox-style pads, from XInput's rough level. A pad on its cable (and
	// every virtual one, Lootnest's own included) has nothing to say.
	var cdevs []core.PadDevice
	for _, d := range devs {
		cdevs = append(cdevs, core.PadDevice(d))
	}
	var slots []core.XInputSlot
	xpads := plat.XInputPads()
	for _, x := range xpads {
		slots = append(slots, core.XInputSlot{Slot: x.Index + 1})
	}
	known := map[int]string{}
	for _, p := range core.Pads(cdevs, slots, nil, false) {
		if p.Slot > 0 && p.Virtual == "" {
			known[p.Slot] = p.ID
		}
	}
	for _, x := range xpads {
		typ, level, ok := plat.XInputBattery(x.Index)
		if !ok {
			continue
		}
		bat, ok := core.XInputBattery(typ, level)
		if !ok || bat.Wired {
			continue
		}
		id := cmp.Or(known[x.Index+1], fmt.Sprintf("xinput:%d", x.Index+1))
		p := pads[id]
		if p == nil {
			p = &padPower{id: id}
			pads[id] = p
		}
		p.name = cmp.Or(names[id], fmt.Sprintf("Xbox controller %d", x.Index+1))
		here[id] = true
		a.batteryNote(p, bat, mode)
	}
	for id := range pads {
		if !here[id] {
			delete(pads, id)
		}
	}
	a.batteryPublish(pads)
}

// batteryNote takes one reading: the warning, then the saver.
func (a *App) batteryNote(p *padPower, bat core.PadBattery, mode string) {
	say, warned := core.BatteryWarn(p.bat, bat, p.seen, p.warned)
	if !p.seen || p.bat != bat {
		a.logf("Controller battery: %s %s", p.name, batteryWords(bat))
	}
	p.bat, p.seen, p.warned = bat, true, warned
	if say != "" {
		a.batterySay(p, say, mode)
	}
	if p.ps {
		if want := core.SaverWanted(mode, bat, p.saver); want != p.saver || (want && !p.saverOK) {
			a.sendSaver(p, want)
		}
	}
}

// sendSaver turns a PlayStation pad's battery saver on or off.
func (a *App) sendSaver(p *padPower, on bool) {
	w, err := plat.OpenPadWrite(p.path)
	if err == nil {
		var r []byte
		padBuzzMu.Lock()
		padBuzzSeq++
		seq := padBuzzSeq
		padBuzzMu.Unlock()
		if r, err = core.PSSaver(p.dualsense, p.bluetooth, w.Size(), seq, on); err == nil {
			err = w.Write(r)
		}
		w.Close()
	}
	if err != nil {
		a.logf("Controller battery: the battery saver couldn't change on %s: %v", p.name, err)
		p.saver, p.saverOK = on, false // tried again at the next look
		return
	}
	p.saver, p.saverOK = on, true
	a.logf("Controller battery: battery saver %s on %s", map[bool]string{true: "on", false: "off"}[on], p.name)
}

// batteryWords is a reading for the log.
func batteryWords(b core.PadBattery) string {
	switch {
	case b.Full:
		return "is charged"
	case b.Level != "":
		return "is " + b.Level
	case b.Charging:
		return fmt.Sprintf("is at %d%%, charging", b.Percent)
	}
	return fmt.Sprintf("is at %d%%", b.Percent)
}

// batterySay warns about a pad: a note in the window when it shows, else
// over whatever is in front (a game).
func (a *App) batterySay(p *padPower, say, mode string) {
	var title, line string
	pct := fmt.Sprintf("%d%%", p.bat.Percent)
	if p.bat.Level != "" {
		pct = ""
	}
	switch say {
	case "low":
		title, line = p.name+" is running low", "Charge it soon."
		if pct != "" {
			title = p.name + " is at " + pct
		}
		if mode == "" && p.ps {
			line = "Charge it soon, or turn on Battery saver in Settings, Controllers."
		}
	case "empty":
		title, line = p.name+" is almost empty", "Plug it in, or it turns off soon."
		if pct != "" {
			line = pct + " left. Plug it in, or it turns off soon."
		}
	case "charged":
		title, line = p.name+" is charged", "You can unplug it."
	default:
		return
	}
	a.logf("Controller battery: %s. %s", title, line)
	if a.hwnd != 0 && plat.IsVisible(a.hwnd) {
		a.emit("padBatterySay", map[string]any{"id": p.id, "title": title, "line": line, "kind": say})
		return
	}
	plat.ShowToast(plat.Toast{Kicker: "Controller battery", Title: title, Line: line})
}

// batteryPublish keeps the reading for the page and sends it when it changed.
func (a *App) batteryPublish(pads map[string]*padPower) {
	view := []PadPowerView{}
	on := map[string]bool{}
	for _, p := range pads {
		view = append(view, PadPowerView{ID: p.id, Name: p.name, Saver: p.saver, PadBattery: p.bat})
		on[p.id] = p.saver
	}
	batteryMu.Lock()
	was, _ := json.Marshal(sortedPower(batteryNow))
	batteryNow, saverOn = view, on
	now, _ := json.Marshal(sortedPower(view))
	batteryMu.Unlock()
	if string(was) != string(now) {
		a.emit("padBattery", map[string]any{"pads": json.RawMessage(now)})
	}
}

func sortedPower(v []PadPowerView) []PadPowerView {
	out := append([]PadPowerView{}, v...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// readPSBattery reads a PlayStation pad's reports for up to 1.5 seconds,
// until one carries its battery. short says a Bluetooth pad sent only its
// short reports.
func readPSBattery(ps psPad) (bat core.PadBattery, ok, short bool) {
	r, err := plat.OpenPad(ps.path)
	if err != nil {
		return bat, false, false
	}
	var once sync.Once
	shut := func() { once.Do(r.Close) }
	done := make(chan struct{})
	go func() {
		select {
		case <-time.After(batteryRead):
			shut()
		case <-done:
		}
	}()
	defer func() { close(done); shut() }()
	buf := make([]byte, 512)
	for i := 0; i < 400; i++ {
		n, err := r.Read(buf)
		if err != nil {
			return bat, false, short
		}
		if b, good := core.PSBattery(ps.dualsense, ps.bluetooth, buf[:n]); good {
			return b, true, false
		}
		if n > 0 && buf[0] == 0x01 && ps.bluetooth {
			short = true
		}
	}
	return bat, false, short
}
