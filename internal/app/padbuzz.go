package app

import (
	"errors"
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/plat"
)

// Naming controllers and telling them apart (core/padbuzz.go): Settings,
// Controllers has a Vibrate key on each pad Lootnest can make buzz, and a
// name for each real one. Nothing is sent to a pad but from that key.

const padBuzzFor = 450 * time.Millisecond

var (
	padBuzzMu  sync.Mutex
	padBuzzing = map[string]bool{}
	padBuzzSeq uint8
)

// apiPadBuzz makes one controller vibrate briefly. It answers at once; a
// "padBuzz" event says when it's done, with an error when it couldn't.
func (a *App) apiPadBuzz(id string) error {
	id = strings.ToLower(strings.TrimSpace(id))
	padBuzzMu.Lock()
	if padBuzzing[id] {
		padBuzzMu.Unlock()
		return nil
	}
	padBuzzing[id] = true
	padBuzzMu.Unlock()
	go func() {
		err := a.padBuzz(id)
		padBuzzMu.Lock()
		delete(padBuzzing, id)
		padBuzzMu.Unlock()
		ev := map[string]any{"id": id}
		if err != nil {
			a.logf("Controllers: couldn't make a controller vibrate: %v", err)
			ev["error"] = sentence(err)
		}
		a.emit("padBuzz", ev)
	}()
	return nil
}

var errPadGone = errors.New("That controller isn't connected any more.")

func (a *App) padBuzz(id string) error {
	devs := plat.PadDevices()
	var slots []core.XInputSlot
	for _, x := range plat.XInputPads() {
		slots = append(slots, core.XInputSlot{Slot: x.Index + 1, Kind: core.XInputKind(x.SubType), Wireless: x.Wireless})
	}
	var cdevs []core.PadDevice
	for _, d := range devs {
		cdevs = append(cdevs, core.PadDevice(d))
	}
	var pad *core.Pad
	for _, p := range core.Pads(cdevs, slots, nil, false) {
		if p.ID == id {
			pad = &p
			break
		}
	}
	if pad == nil {
		return errPadGone
	}
	if !pad.Buzz {
		return errors.New("Lootnest can't make that controller vibrate.")
	}
	if pad.VID != core.SonyVendor {
		if err := plat.XInputRumble(pad.Slot-1, 40000, 40000); err != nil {
			return err
		}
		time.Sleep(padBuzzFor)
		return plat.XInputRumble(pad.Slot-1, 0, 0)
	}
	dualsense, _ := core.PSPad(pad.PID)
	bluetooth := pad.Bus == "bluetooth"
	// The pad's HID devices, by their instance ids, then their paths.
	hid := map[string]bool{}
	for _, d := range devs {
		if !d.XUSB && len(d.Chain) > 0 && core.PadID(d.Chain) == id {
			hid[strings.ToLower(d.Chain[0])] = true
		}
	}
	err := errPadGone
	for _, path := range plat.HIDPaths() {
		if !hid[strings.ToLower(plat.HIDInstance(path))] {
			continue
		}
		if err = psBuzz(path, dualsense, bluetooth); err == nil {
			return nil
		}
	}
	return err
}

// psBuzz starts a PlayStation pad's motors and stops them again.
func psBuzz(path string, dualsense, bluetooth bool) error {
	w, err := plat.OpenPadWrite(path)
	if err != nil {
		return err
	}
	defer w.Close()
	send := func(strong, weak uint8) error {
		padBuzzMu.Lock()
		padBuzzSeq++
		seq := padBuzzSeq
		padBuzzMu.Unlock()
		r, err := core.PSRumble(dualsense, bluetooth, w.Size(), seq, strong, weak)
		if err != nil {
			return err
		}
		return w.Write(r)
	}
	if err := send(200, 200); err != nil {
		return err
	}
	time.Sleep(padBuzzFor)
	return send(0, 0)
}

// apiPadName names a controller by its id, or takes its name away.
func (a *App) apiPadName(id, name string) error {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	names, err := core.SetPadName(cfg.PadNames, id, name)
	if err != nil {
		return err
	}
	cfg.PadNames = names
	return a.saveConfig(cfg)
}

// padNames is a copy of the names given to controllers.
func (a *App) padNames() map[string]string {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := map[string]string{}
	if a.cfg != nil {
		for k, v := range a.cfg.PadNames {
			out[k] = v
		}
	}
	return out
}

// padNameOfPath is the name given to the pad a HID path belongs to, for the
// Xbox bridge's note; "" when it has none.
func padNameOfPath(names map[string]string, devs []plat.PadDevice, path string) string {
	if len(names) == 0 {
		return ""
	}
	inst := strings.ToLower(plat.HIDInstance(path))
	for _, d := range devs {
		if len(d.Chain) > 0 && strings.ToLower(d.Chain[0]) == inst {
			return names[core.PadID(d.Chain)]
		}
	}
	return ""
}
