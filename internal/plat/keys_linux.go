package plat

import (
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"lootnest/internal/desk"
)

// Keys for the whole desktop and pictures of games on Linux. On X11
// Lootnest holds its keys and sees other programs' windows itself, as on
// Windows. On Wayland only the desktop portal can: GlobalShortcuts for the
// keys (the desktop may ask once whether Lootnest may have them, and may
// give other keys) and Screenshot for a picture of the screen.

// HotKey is a key the app may hold while a game runs, as the desktop
// portal is told about it.
type HotKey struct {
	ID   int
	Mods uint32 // ModControl, ModShift, ModAlt
	VK   uint32
	Says string // what it does
}

var (
	hkMu    sync.Mutex
	hkKeys  []HotKey
	hkFire  func(id int)
	hkSay   func(msg string)
	hkHeld  = map[int]bool{}
	hkBound bool // the portal's session was asked for
)

// SetHotKeys says which keys the app may hold, what runs when one is
// pressed (on the GTK thread) and where to say what the desktop answered.
func SetHotKeys(keys []HotKey, fire func(id int), say func(msg string)) {
	hkMu.Lock()
	hkKeys, hkFire, hkSay = keys, fire, say
	hkMu.Unlock()
}

func hotKey(id int) (HotKey, bool) {
	for _, k := range hkKeys {
		if k.ID == id {
			return k, true
		}
	}
	return HotKey{}, false
}

// keysym is X11's name for a Windows virtual key: F1 to F24, letters and
// digits.
func keysym(vk uint32) uint32 {
	switch {
	case vk >= 0x70 && vk <= 0x87: // VK_F1 to VK_F24
		return 0xffbe + vk - 0x70
	case vk >= 'A' && vk <= 'Z':
		return vk + 32
	case vk >= '0' && vk <= '9':
		return vk
	}
	return 0
}

func xMods(mods uint32) uint32 {
	var m uint32
	if mods&ModControl != 0 {
		m |= desk.ControlMask
	}
	if mods&ModShift != 0 {
		m |= desk.ShiftMask
	}
	if mods&ModAlt != 0 {
		m |= desk.AltMask
	}
	return m
}

// trigger spells a key as the portal does: "CTRL+SHIFT+F5".
func trigger(mods, vk uint32) string {
	var parts []string
	if mods&ModControl != 0 {
		parts = append(parts, "CTRL")
	}
	if mods&ModAlt != 0 {
		parts = append(parts, "ALT")
	}
	if mods&ModShift != 0 {
		parts = append(parts, "SHIFT")
	}
	switch {
	case vk >= 0x70 && vk <= 0x87:
		parts = append(parts, "F"+strconv.Itoa(int(vk-0x6f)))
	default:
		parts = append(parts, string(rune(vk)))
	}
	return strings.Join(parts, "+")
}

// RegisterHotKey holds a key the app declared with SetHotKeys. On X11 it
// answers false when another program holds it. On Wayland the first one
// asks the portal for every declared key at once; until the desktop
// answers, and after, a press reaches the app only while the key is held.
func RegisterHotKey(hwnd uintptr, id int, mods, vk uint32) bool {
	hkMu.Lock()
	k, ok := hotKey(id)
	fire := hkFire
	hkMu.Unlock()
	if !ok || fire == nil {
		return false
	}
	if desk.X11() {
		sym := keysym(vk)
		if sym == 0 || !desk.GrabKey(sym, xMods(mods), func() { fire(id) }) {
			return false
		}
	} else {
		bindPortalKeys()
	}
	hkMu.Lock()
	hkHeld[k.ID] = true
	hkMu.Unlock()
	return true
}

// UnregisterHotKey lets go of a key. A portal key stays the desktop's
// until Lootnest quits; its presses are dropped.
func UnregisterHotKey(hwnd uintptr, id int) {
	hkMu.Lock()
	k, ok := hotKey(id)
	delete(hkHeld, id)
	hkMu.Unlock()
	if ok && desk.X11() {
		if sym := keysym(k.VK); sym != 0 {
			desk.UngrabKey(sym, xMods(k.Mods))
		}
	}
}

func bindPortalKeys() {
	hkMu.Lock()
	if hkBound {
		hkMu.Unlock()
		return
	}
	hkBound = true
	var asks []desk.Shortcut
	for _, k := range hkKeys {
		asks = append(asks, desk.Shortcut{ID: "lootnest-" + strconv.Itoa(k.ID), Says: k.Says, Trigger: trigger(k.Mods, k.VK)})
	}
	say := hkSay
	hkMu.Unlock()
	desk.BindShortcuts(asks, func(sid string) {
		id, err := strconv.Atoi(strings.TrimPrefix(sid, "lootnest-"))
		if err != nil {
			return
		}
		hkMu.Lock()
		held, fire := hkHeld[id], hkFire
		hkMu.Unlock()
		if held && fire != nil {
			fire(id)
		}
	}, func(got map[string]string, err error) {
		if say == nil {
			return
		}
		if err != nil {
			say(fmt.Sprintf("The desktop gave Lootnest no keys while you play (%v); save points and screenshots still work from a game's page", err))
			return
		}
		var keys []string
		for _, a := range asks {
			if t := got[a.ID]; t != "" {
				keys = append(keys, a.Says+": "+t)
			} else if _, ok := got[a.ID]; ok {
				keys = append(keys, a.Says+": no key yet (set one in the desktop's keyboard settings)")
			}
		}
		say("Keys while you play, from the desktop: " + strings.Join(keys, "; "))
	})
}

// GameWindow is the front-most shown window of a program named proc (as
// ProcessNames names it), on X11; 0 on Wayland or when there's none.
// Small windows (a launcher's tray, a splash's shadow) are passed over.
func GameWindow(proc string) uintptr {
	proc = strings.ToLower(proc)
	wins := desk.X11Windows()
	for i := len(wins) - 1; i >= 0; i-- {
		w := wins[i]
		if w.PID <= 0 || w.W < 200 || w.H < 150 {
			continue
		}
		if names, err := pidNames(filepath.Join("/proc", strconv.Itoa(w.PID))); err == nil && names[proc] {
			return w.ID
		}
	}
	return 0
}

// pidNames are one process's names, as ProcessNames gives them.
func pidNames(dir string) (map[string]bool, error) {
	out := map[string]bool{}
	if _, err := os.Stat(dir); err != nil {
		return nil, err
	}
	addProcess(out, dir)
	return out, nil
}

// CaptureWindow copies a window's picture on X11.
func CaptureWindow(hwnd uintptr) (*image.RGBA, error) {
	if hwnd == 0 {
		return nil, errors.New("no window")
	}
	return desk.X11Capture(hwnd, false)
}

// CaptureScreen copies what the screen shows where a window is, on X11.
// With no window it's the whole screen, through the desktop portal on
// Wayland (the file the portal saves is removed once read).
func CaptureScreen(hwnd uintptr) (*image.RGBA, error) {
	if hwnd != 0 {
		return desk.X11Capture(hwnd, true)
	}
	path, err := desk.Screenshot()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	img, err := png.Decode(f)
	f.Close()
	if err != nil {
		return nil, fmt.Errorf("the portal's picture: %w", err)
	}
	os.Remove(path)
	rgba, ok := img.(*image.RGBA)
	if !ok {
		rgba = image.NewRGBA(img.Bounds())
		draw.Draw(rgba, rgba.Bounds(), img, img.Bounds().Min, draw.Src)
	}
	return rgba, nil
}
