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
	if len(IconPNG) < 1000 || string(IconPNG[1:4]) != "PNG" {
		t.Error("the icon")
	}
}
