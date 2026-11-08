package core

import (
	"regexp"
	"strconv"
	"strings"
)

// Controllers on this PC, from what Windows says about each device: its ids,
// its parents' ids and its HID usage. Nothing here reads or writes a pad's
// reports, which can switch a Bluetooth DualSense into another mode.

// PadDevice is one HID or Xbox 360 (XUSB) device as Windows lists it.
type PadDevice struct {
	Chain       []string // its instance id, then its parent's and so on, up to four
	HardwareIDs []string
	CompatIDs   []string
	Name        string // the name its bus reports, or the one Windows shows
	XUSB        bool   // found by the Xbox 360 pad interface rather than HID
}

// Pad is one controller as Settings shows it.
type Pad struct {
	Name    string `json:"name"`
	Family  string `json:"family"` // xbox, playstation, nintendo or generic, for the glyphs
	VID     uint16 `json:"vid"`
	PID     uint16 `json:"pid"`
	Bus     string `json:"bus"` // usb, bluetooth, or "" when Windows doesn't say
	XInput  bool   `json:"xinput"`
	Slot    int    `json:"slot"`    // its XInput slot, 1 to 4, when it can be told
	Virtual string `json:"virtual"` // vigem, vjoy, steam or root; "" for a real pad
	MadeBy  string `json:"madeBy"`  // the app that made a virtual pad, when known
}

// XInputSlot is one of Windows' four Xbox-style slots that has a pad in it.
type XInputSlot struct {
	Slot     int    `json:"slot"` // 1 to 4
	Kind     string `json:"kind"`
	Wireless bool   `json:"wireless"`
}

type padModel struct{ name, family string }

var padModels = map[uint32]padModel{
	0x045E028E: {"Xbox 360 controller", "xbox"},
	0x045E028F: {"Xbox 360 wireless controller", "xbox"},
	0x045E0719: {"Xbox 360 wireless controller", "xbox"},
	0x045E02D1: {"Xbox One controller", "xbox"},
	0x045E02DD: {"Xbox One controller", "xbox"},
	0x045E02E0: {"Xbox One S controller", "xbox"},
	0x045E02EA: {"Xbox One S controller", "xbox"},
	0x045E02FD: {"Xbox One S controller", "xbox"},
	0x045E02FF: {"Xbox controller", "xbox"},
	0x045E02E3: {"Xbox Elite controller", "xbox"},
	0x045E0B00: {"Xbox Elite Series 2 controller", "xbox"},
	0x045E0B05: {"Xbox Elite Series 2 controller", "xbox"},
	0x045E0B22: {"Xbox Elite Series 2 controller", "xbox"},
	0x045E0B0A: {"Xbox Adaptive Controller", "xbox"},
	0x045E0B0C: {"Xbox Adaptive Controller", "xbox"},
	0x045E0B12: {"Xbox Series controller", "xbox"},
	0x045E0B13: {"Xbox Series controller", "xbox"},
	0x045E0B20: {"Xbox Series controller", "xbox"},
	0x054C0268: {"DualShock 3", "playstation"},
	0x054C05C4: {"DualShock 4 (first model)", "playstation"},
	0x054C09CC: {"DualShock 4 (second model)", "playstation"},
	0x054C0BA0: {"DualShock 4 wireless adaptor", "playstation"},
	0x054C0CE6: {"DualSense", "playstation"},
	0x054C0DF2: {"DualSense Edge", "playstation"},
	0x057E2006: {"Joy-Con (L)", "nintendo"},
	0x057E2007: {"Joy-Con (R)", "nintendo"},
	0x057E2008: {"Joy-Con pair", "nintendo"},
	0x057E200E: {"Joy-Con pair", "nintendo"},
	0x057E2009: {"Switch Pro Controller", "nintendo"},
	0x046DC216: {"Logitech F310 (DirectInput mode)", "generic"},
	0x046DC21D: {"Logitech F310", "xbox"},
	0x046DC219: {"Logitech F710 (DirectInput mode)", "generic"},
	0x046DC21F: {"Logitech F710", "xbox"},
	0x28DE1102: {"Steam Controller", "xbox"},
	0x28DE1142: {"Steam Controller", "xbox"},
	0x28DE1205: {"Steam Deck controls", "xbox"},
	0x28DE11FF: {"Steam virtual pad", "xbox"},
	0x1234BEAD: {"vJoy virtual pad", "generic"},
}

// By maker, for pads the table doesn't name.
var padVendors = map[uint16]padModel{
	0x045E: {"", "xbox"},
	0x054C: {"", "playstation"},
	0x057E: {"", "nintendo"},
	0x2DC8: {"8BitDo controller", "generic"},
}

// modelOf names a controller by its USB vendor and product ids, and gives
// the family of glyphs to draw for it. An unknown pad has no name and the
// generic family.
func modelOf(vid, pid uint16) (name, family string) {
	if m, ok := padModels[uint32(vid)<<16|uint32(pid)]; ok {
		return m.name, m.family
	}
	if m, ok := padVendors[vid]; ok {
		return m.name, m.family
	}
	return "", "generic"
}

// "VID_054C&PID_0CE6" over USB, "VID&0002054c_PID&0ce6" over Bluetooth,
// "VID&02045e_PID&0b13" over Bluetooth LE.
var vidPidRe = regexp.MustCompile(`(?i)VID[_&](?:[0-9A-F]{4}|[0-9A-F]{2})?([0-9A-F]{4})[_&]PID[_&]([0-9A-F]{4})`)

// parseVIDPID reads the vendor and product ids from a hardware or instance id.
func parseVIDPID(id string) (vid, pid uint16, ok bool) {
	m := vidPidRe.FindStringSubmatch(id)
	if m == nil {
		return 0, 0, false
	}
	v, _ := strconv.ParseUint(m[1], 16, 16)
	p, _ := strconv.ParseUint(m[2], 16, 16)
	return uint16(v), uint16(p), true
}

// gameUsage says whether a HID device's compatible ids name it a joystick
// or a gamepad (generic desktop usages 4 and 5).
func gameUsage(compat []string) bool {
	for _, c := range compat {
		switch strings.ToUpper(c) {
		case "HID_DEVICE_SYSTEM_GAME", "HID_DEVICE_UP:0001_U:0004", "HID_DEVICE_UP:0001_U:0005":
			return true
		}
	}
	return false
}

// padBus says how a pad is connected, from its ids and its parents'.
func padBus(chain []string) string {
	for _, id := range chain {
		u := strings.ToUpper(id)
		switch {
		case strings.Contains(u, "{00001124-") || strings.Contains(u, "{00001812-"), // the Bluetooth HID services
			strings.HasPrefix(u, "BTHENUM\\"), strings.HasPrefix(u, "BTHLEDEVICE\\"):
			return "bluetooth"
		case strings.HasPrefix(u, "USB\\"):
			return "usb"
		}
	}
	return ""
}

// padVirtual says which kind of virtual pad a device is, or "" for a real one.
func padVirtual(vid, pid uint16, ids []string, vigem bool) string {
	switch uint32(vid)<<16 | uint32(pid) {
	case 0x1234BEAD:
		return "vjoy"
	case 0x28DE11FF:
		return "steam"
	}
	root := false
	for _, id := range ids {
		u := strings.ToUpper(id)
		if strings.Contains(u, "VIGEM") {
			return "vigem"
		}
		root = root || strings.HasPrefix(u, "ROOT\\") || strings.HasPrefix(u, "SWD\\")
	}
	if !root {
		return ""
	}
	if vigem && (vid == 0x045E && pid == 0x028E || vid == 0x054C && pid == 0x05C4) {
		return "vigem" // the Xbox 360 and DualShock 4 pads ViGEmBus makes
	}
	return "root"
}

// One physical pad can have several HID devices (one per collection, or per
// USB interface): they share the first id up its chain that isn't HID's
// own or a USB interface's.
func padKey(chain []string) string {
	for _, id := range chain {
		u := strings.ToUpper(id)
		if !strings.HasPrefix(u, "HID\\") && !strings.Contains(u, "&MI_") {
			return u
		}
	}
	if len(chain) > 0 {
		return strings.ToUpper(chain[0])
	}
	return ""
}

// Pads turns the devices Windows lists into the controllers Settings shows:
// joysticks, gamepads and known pads whatever their usage, one row per
// physical pad. vigem says the ViGEmBus driver is installed.
func Pads(devs []PadDevice, slots []XInputSlot, tools []ControllerTool, vigem bool) []Pad {
	out := []Pad{}
	at := map[string]int{}
	for _, d := range devs {
		var vid, pid uint16
		for _, id := range append(append([]string{}, d.HardwareIDs...), d.Chain...) {
			if v, p, ok := parseVIDPID(id); ok {
				vid, pid = v, p
				break
			}
		}
		_, known := padModels[uint32(vid)<<16|uint32(pid)]
		if !d.XUSB && !known && !gameUsage(d.CompatIDs) {
			continue
		}
		xin := d.XUSB || len(d.Chain) > 0 && strings.Contains(strings.ToUpper(d.Chain[0]), "&IG_")
		key := padKey(d.Chain)
		if i, ok := at[key]; ok && key != "" {
			out[i].XInput = out[i].XInput || xin
			continue
		}
		name, fam := modelOf(vid, pid)
		if xin && fam == "generic" {
			fam = "xbox" // an Xbox-style pad from a maker the tables don't know
		}
		if name == "" {
			name = strings.TrimSpace(d.Name)
		}
		if name == "" {
			name = "Game controller"
		}
		kind := padVirtual(vid, pid, append(append([]string{}, d.Chain...), d.HardwareIDs...), vigem)
		bus := ""
		if kind == "" {
			bus = padBus(d.Chain)
		}
		at[key] = len(out)
		out = append(out, Pad{Name: name, Family: fam, VID: vid, PID: pid, Bus: bus, XInput: xin, Virtual: kind, MadeBy: padMaker(kind, tools)})
	}
	// Windows doesn't say which slot is whose; with one of each, it's that one.
	xi := -1
	for i, p := range out {
		if p.XInput {
			if xi >= 0 {
				xi = -2
				break
			}
			xi = i
		}
	}
	if xi >= 0 && len(slots) == 1 {
		out[xi].Slot = slots[0].Slot
	}
	return out
}

// XInputKind names an XInput subtype.
func XInputKind(sub uint8) string {
	switch sub {
	case 1:
		return "gamepad"
	case 2:
		return "wheel"
	case 3:
		return "arcade stick"
	case 4:
		return "flight stick"
	case 5:
		return "dance pad"
	case 6, 7, 11:
		return "guitar"
	case 8:
		return "drum kit"
	case 0x13:
		return "arcade pad"
	}
	return "controller"
}

// ControllerTool is an app or driver that changes how games see pads.
type ControllerTool struct {
	Name      string `json:"name"`
	Does      string `json:"does"`
	Running   bool   `json:"running"`
	Installed bool   `json:"installed"`
	procs     []string
	appData   string // a folder under %APPDATA% that means it's installed
	service   string
	pads      bool // games see a pad it makes
	vigem     bool // that pad comes from ViGEmBus, beside the real one
	keys      bool // it turns presses into keys
}

const steamInput = "Steam Input"

var controllerTools = []ControllerTool{
	{Name: steamInput, Does: "Can translate your pad for the games Steam starts, shortcuts included, so they see an Xbox pad.", procs: []string{"steam"}, pads: true},
	{Name: "DS4Windows", Does: "Makes your PlayStation pad look like an Xbox pad to games.", procs: []string{"ds4windows"}, appData: "DS4Windows", pads: true, vigem: true},
	{Name: "DSX", Does: "Makes your DualSense look like an Xbox or DualShock 4 pad to games, and can set its triggers and lights.", procs: []string{"dsx"}, pads: true, vigem: true},
	{Name: "DualSenseX", Does: "Makes your DualSense look like an Xbox pad to games. DSX is its newer version.", procs: []string{"dualsensex"}, pads: true, vigem: true},
	{Name: "reWASD", Does: "Remaps buttons, and can make one pad look like another to games.", procs: []string{"rewasd", "rewasdservice", "rewasdengine"}, pads: true},
	{Name: "x360ce", Does: "Makes a pad that isn't an Xbox pad look like one to games.", procs: []string{"x360ce", "x360ce_x64"}, pads: true, vigem: true},
	{Name: "BetterJoy", Does: "Makes Switch Pro Controllers and Joy-Cons look like an Xbox pad to games.", procs: []string{"betterjoy", "betterjoyforcemu"}, pads: true, vigem: true},
	{Name: "InputMapper", Does: "Makes PlayStation pads look like an Xbox pad to games.", procs: []string{"inputmapper"}, pads: true, vigem: true},
	{Name: "JoyToKey", Does: "Turns your pad's buttons into key presses and mouse moves.", procs: []string{"joytokey"}, keys: true},
	{Name: "AntiMicroX", Does: "Turns your pad's buttons into key presses and mouse moves.", procs: []string{"antimicrox"}, keys: true},
	{Name: "Xpadder", Does: "Turns your pad's buttons into key presses and mouse moves.", procs: []string{"xpadder"}, keys: true},
	{Name: "HidHide", Does: "Hides the pads you choose from games, so they see only the translated one.", service: "HidHide"},
	{Name: "ViGEmBus", Does: "The driver that apps such as DS4Windows use to add a virtual Xbox or PlayStation pad.", service: "ViGEmBus"},
}

// ControllerProbes are the folders under %APPDATA% and the services whose
// presence FindControllerTools needs.
func ControllerProbes() (appData, services []string) {
	for _, t := range controllerTools {
		if t.appData != "" {
			appData = append(appData, t.appData)
		}
		if t.service != "" {
			services = append(services, t.service)
		}
	}
	return appData, services
}

// FindControllerTools lists the tools that are installed or running, from
// the running processes (lower case, without .exe), the %APPDATA% folders
// present, and the installed services (true when running).
func FindControllerTools(procs, appData, services map[string]bool) []ControllerTool {
	out := []ControllerTool{}
	for _, t := range controllerTools {
		for _, p := range t.procs {
			t.Running = t.Running || procs[p]
		}
		run, ok := services[t.service]
		if t.service != "" && ok {
			t.Installed, t.Running = true, run
		}
		t.Installed = t.Installed || t.Running || t.appData != "" && appData[t.appData]
		if t.Installed {
			out = append(out, t)
		}
	}
	return out
}

func toolOn(tools []ControllerTool, name string) bool {
	for _, t := range tools {
		if t.Name == name && t.Installed {
			return true
		}
	}
	return false
}

// padMaker names the app that made a virtual pad of this kind, or "".
func padMaker(kind string, tools []ControllerTool) string {
	switch kind {
	case "steam":
		return steamInput
	case "vjoy":
		return "vJoy"
	case "vigem":
		for _, t := range tools {
			if t.vigem && t.Running {
				return t.Name
			}
		}
		return "an app that uses ViGEmBus"
	}
	return ""
}

// ControllerNotes are the plain lines Settings shows under the tools, for
// combinations that make games behave oddly.
func ControllerNotes(tools []ControllerTool) []string {
	notes := []string{}
	var both []string
	steam := false
	for _, t := range tools {
		if t.Running && t.pads {
			both = append(both, t.Name)
			steam = steam || t.Name == steamInput
		}
	}
	if len(both) > 1 {
		n := joinAnd(both) + " both translating can make games see double input."
		if len(both) > 2 {
			n = joinAnd(both) + " all translating can make games see double input."
		}
		if steam {
			n += " For a game Steam starts, you can turn Steam Input off in the game's Properties, under Controller."
		}
		notes = append(notes, n)
	}
	var copies []string
	for _, t := range tools {
		if t.Running && t.vigem && !toolOn(tools, "HidHide") {
			copies = append(copies, t.Name)
		}
	}
	switch {
	case len(copies) == 1:
		notes = append(notes, "Games may see two pads, your own and the copy "+copies[0]+" makes, unless HidHide hides your own.")
	case len(copies) > 1:
		notes = append(notes, "Games may see more than one pad, your own and the copies "+joinAnd(copies)+" make, unless HidHide hides your own.")
	}
	for _, t := range tools {
		if t.Running && t.keys {
			notes = append(notes, "While "+t.Name+" runs, a game that reads the pad as well gets each press twice: as the button and as a key.")
		}
	}
	return notes
}

func joinAnd(s []string) string {
	if len(s) < 2 {
		return strings.Join(s, "")
	}
	return strings.Join(s[:len(s)-1], ", ") + " and " + s[len(s)-1]
}
