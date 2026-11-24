//go:build linux

package desk

import (
	"os"
	"testing"
	"time"
)

// What the check saw: it runs in TestMain, on the main thread GTK needs.
var seen struct {
	ran        bool
	fromOther  bool
	clip       string
	tray       *Tray
	menuPicked int

	x11             bool
	grabbed, regrab bool
	pressed         int
	shortcutsAnswer bool
	shotErr         error
}

func TestMain(m *testing.M) {
	if (os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != "") && Init() {
		seen.ran = true
		dir, _ := os.MkdirTemp("", "desk-test")
		go func() {
			defer Quit()
			// From another goroutine: through the GTK loop.
			OnMain(func() { seen.fromOther = true })
			SetClipboardText("Lootnest on Linux")
			seen.clip = ClipboardText()
			seen.tray = NewTray(dir)
			seen.tray.Show(true, "Lootnest")
			seen.tray.SetMenu([]MenuItem{{ID: 1, Text: "Open Lootnest"}, {Sep: true}, {ID: 2, Text: "Quit Lootnest"}},
				func(id int) { seen.menuPicked = id })
			Notify("Lootnest", "A test notification", "") // without a notification service it goes nowhere
			_, _ = KeyLookup("lootnest-test")             // without a keyring it's an error, never a crash
			// Keys: on X11 a key is held and let go; a press reaches its
			// function only through the grab (the filter is checked by
			// calling it as X would).
			if seen.x11 = X11(); seen.x11 {
				seen.grabbed = GrabKey(0xffc2, ControlMask|ShiftMask, func() { seen.pressed++ }) // Ctrl+Shift+F5
				keysMu.Lock()
				var code uint64
				for k := range grabbed {
					code = k >> 32
				}
				keysMu.Unlock()
				OnMain(func() { deskKey(cUint(code), cUint(ControlMask|ShiftMask)) })
				UngrabKey(0xffc2, ControlMask|ShiftMask)
				OnMain(func() { deskKey(cUint(code), cUint(ControlMask|ShiftMask)) })
				seen.regrab = GrabKey(0xffc2, ControlMask|ShiftMask, func() {})
				UngrabKey(0xffc2, ControlMask|ShiftMask)
			}
			// The portal: whatever this session bus has, an answer comes
			// (an error without a portal), and nothing hangs.
			done := make(chan bool, 1)
			BindShortcuts([]Shortcut{{ID: "t", Says: "A test", Trigger: "CTRL+SHIFT+F5"}}, func(string) {},
				func(map[string]string, error) { done <- true })
			select {
			case seen.shortcutsAnswer = <-done:
			case <-time.After(35 * time.Second):
			}
			if !seen.x11 {
				_, seen.shotErr = Screenshot()
			}
			time.Sleep(200 * time.Millisecond)
		}()
		Main()
		os.RemoveAll(dir)
	}
	os.Exit(m.Run())
}

func TestDesktop(t *testing.T) {
	if !seen.ran {
		t.Skip("no display")
	}
	if !seen.fromOther {
		t.Error("OnMain from another goroutine")
	}
	if seen.clip != "Lootnest on Linux" {
		t.Errorf("the clipboard: %q", seen.clip)
	}
	if seen.tray == nil {
		t.Log("no AppIndicator library here: no tray, as on such a desktop")
	}
	if seen.x11 && (!seen.grabbed || seen.pressed != 1 || !seen.regrab) {
		t.Errorf("X11 keys: grabbed %v, pressed %d times (want 1), held again after letting go %v", seen.grabbed, seen.pressed, seen.regrab)
	}
	if !seen.shortcutsAnswer {
		t.Error("BindShortcuts never answered")
	}
	if len(IconPNG) < 1000 || string(IconPNG[1:4]) != "PNG" {
		t.Error("the icon")
	}
}
