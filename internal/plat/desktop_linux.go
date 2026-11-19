package plat

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"lootnest/internal/core"
	"lootnest/internal/desk"
)

// What Lootnest asks of a Linux desktop besides its window, over
// internal/desk (GTK, the Secret Service, AppIndicator, notifications).

// MessageBox shows a modal dialog with Windows' MB_ flags and answers with
// Windows' ID numbers.
func MessageBox(hwnd uintptr, text, caption string, flags uint32) int {
	kind := desk.Info
	switch flags & 0xF0 {
	case MB_ICONERROR:
		kind = desk.Error
	case MB_ICONQUESTION:
		kind = desk.Question
	case MB_ICONWARNING:
		kind = desk.Warning
	}
	buttons := desk.ButtonsOK
	switch flags & 0x0F {
	case MB_OKCANCEL:
		buttons = desk.ButtonsOKCancel
	case MB_YESNO:
		buttons = desk.ButtonsYesNo
	case MB_YESNOCANCEL:
		buttons = desk.ButtonsYesNoCancel
	}
	return desk.Message(caption, text, kind, buttons, flags&MB_DEFBUTTON2 != 0)
}

// Browse asks for a file or a folder with the desktop's own picker.
func Browse(o BrowseOptions) (string, error) {
	var fs []desk.Filter
	for _, f := range o.Filters {
		fs = append(fs, desk.Filter{Name: f.Name, Patterns: f.Spec})
	}
	p, err := desk.Browse(o.Title, o.Folder, o.Start, fs)
	if errors.Is(err, desk.ErrCancelled) {
		return "", ErrCancelled
	}
	return p, err
}

func SetClipboardText(hwnd uintptr, text string) error {
	desk.SetClipboardText(text)
	return nil
}

func SetClipboardImage(hwnd uintptr, pngData []byte) error { return desk.SetClipboardPNG(pngData) }

// IconPath is Lootnest's icon on disk, for notifications, the tray and
// the window.
func IconPath() string { return iconPath() }

func iconPath() string {
	return filepath.Join(core.XDGDataHome(home(), os.Getenv), "lootnest", "icons", "lootnest.png")
}

// ShowToast: a Linux desktop doesn't let a window float over a game, so a
// note over the game is a notification.
func ShowToast(t Toast) {
	body := strings.TrimSpace(strings.Trim(t.Kicker+". "+t.Line, ". "))
	desk.Notify(t.Title, body, iconPath())
}

// Tray is Lootnest's tray icon on Linux; it does nothing on a desktop
// without AppIndicator, where notifications still show.
type Tray struct {
	t   *desk.Tray
	tip string
}

func NewTray(hwnd uintptr, dpi int) *Tray {
	return &Tray{t: desk.NewTray(filepath.Dir(iconPath()))}
}

// Shown tells whether the desktop shows the icon (so closing the window
// can leave Lootnest there).
func (t *Tray) Shown() bool { return t != nil && t.t != nil }

func (t *Tray) Add(tip string) {
	if t != nil {
		t.tip = tip
		t.t.Show(true, tip)
	}
}

func (t *Tray) SetTip(tip string) {
	if t != nil && tip != t.tip {
		t.tip = tip
		t.t.Show(true, tip)
	}
}

func (t *Tray) Notify(title, text string) { desk.Notify(title, text, iconPath()) }

func (t *Tray) Remove() {
	if t != nil {
		t.t.Show(false, t.tip)
	}
}

// SetMenu replaces the tray's menu (Linux only: on Windows the menu is
// made when the icon is clicked).
func (t *Tray) SetMenu(items []MenuItem, pick func(id int)) {
	if t == nil {
		return
	}
	ds := make([]desk.MenuItem, len(items))
	for i, it := range items {
		ds[i] = desk.MenuItem{ID: it.ID, Text: strings.ReplaceAll(it.Text, "&&", "&"), Disabled: it.Disabled, Sep: it.Sep}
	}
	t.t.SetMenu(ds, pick)
}
