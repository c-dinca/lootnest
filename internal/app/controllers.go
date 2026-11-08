//go:build windows

package app

import (
	"os"
	"path/filepath"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// Controllers is what Settings, Controllers shows: the pads Windows lists,
// the XInput slots in use, the translators installed or running, and notes
// on how they combine.
type Controllers struct {
	Pads        []core.Pad            `json:"pads"`
	XInput      []core.XInputSlot     `json:"xinput"`
	Translators []core.ControllerTool `json:"translators"`
	Notes       []string              `json:"notes"`
}

// apiControllers only reads: the device list, XInput's slots, the running
// processes, two folders and two services. It changes nothing, so it runs
// in safe mode too.
func (a *App) apiControllers() Controllers {
	procs, _ := win.ProcessNames()
	dirs, svcs := core.ControllerProbes()
	appData := map[string]bool{}
	for _, d := range dirs {
		if fi, err := os.Stat(filepath.Join(os.Getenv("APPDATA"), d)); err == nil && fi.IsDir() {
			appData[d] = true
		}
	}
	services := map[string]bool{}
	for _, s := range svcs {
		if ok, run := win.ServiceState(s); ok {
			services[s] = run
		}
	}
	tools := core.FindControllerTools(procs, appData, services)
	slots := []core.XInputSlot{}
	for _, x := range win.XInputPads() {
		slots = append(slots, core.XInputSlot{Slot: x.Index + 1, Kind: core.XInputKind(x.SubType), Wireless: x.Wireless})
	}
	var devs []core.PadDevice
	for _, d := range win.PadDevices() {
		devs = append(devs, core.PadDevice(d))
	}
	_, vigem := services["ViGEmBus"]
	return Controllers{Pads: core.Pads(devs, slots, tools, vigem), XInput: slots, Translators: tools, Notes: core.ControllerNotes(tools)}
}
