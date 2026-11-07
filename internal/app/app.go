//go:build windows

// Package app is the Lootnest desktop application: window, tray, watcher
// and the API the interface talks to.
package app

import (
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	webview2 "github.com/jchv/go-webview2"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

const (
	Name        = "Lootnest"
	WindowClass = "LootnestMainWindow"
	MutexName   = `Local\Lootnest.SingleInstance.v1`
	UninstallID = "Lootnest"
	bgColor     = 0x0E1117 // --ink-2 in ui/app.css: the title bar frames the window like the sidebar
	borderColor = 0x2A3142 // --line
)

// Version is set at build time with -ldflags "-X lootnest/internal/app.Version=..."
var Version = "1.0.0"

//go:embed ui/index.html
var uiHTML string

//go:embed ui/system.css
var uiSystemCSS string

//go:embed ui/app.css
var uiCSS string

//go:embed ui/app.js
var uiJS string

// Bricolage Grotesque and Figtree (both SIL Open Font License 1.1), embedded
// so the window looks the same on every PC without going online.
//
//go:embed ui/fonts/*.woff2
var uiFonts embed.FS

// uiFontNames are the files in ui/fonts that system.css asks for.
var uiFontNames = []string{"bricolage-latin", "bricolage-latin-ext", "figtree-latin", "figtree-latin-ext"}

// pageHTML is the UI with its stylesheet, script and fonts inlined. Its
// Content-Security-Policy allows only this one script (by a nonce made for
// each start), images and fonts from data: URLs and Steam's image servers,
// and nothing else: no other scripts, no inline handlers, no fetches.
func pageHTML() string {
	nonce := make([]byte, 18)
	_, _ = rand.Read(nonce)
	n := base64.RawStdEncoding.EncodeToString(nonce)
	css := uiSystemCSS + "\n" + uiCSS
	for _, name := range uiFontNames {
		if b, err := uiFonts.ReadFile("ui/fonts/" + name + ".woff2"); err == nil {
			css = strings.Replace(css, "@FONT:"+name+"@", base64.StdEncoding.EncodeToString(b), 1)
		}
	}
	h := strings.Replace(uiHTML, "/*@CSS*/", css, 1)
	h = strings.ReplaceAll(h, "@NONCE@", n)
	return strings.Replace(h, "/*@JS*/", uiJS, 1)
}

type App struct {
	w    webview2.WebView
	hwnd uintptr

	store *core.Store
	eng   *core.Engine
	env   *core.Env

	mu       sync.Mutex
	cfg      *core.Config
	firstRun bool
	loadErr  string
	sessions map[string]*Session
	pending  string
	pendReq  string
	requests map[string]string // open Steam requests: reqID -> gameID
	ready    bool

	hookMu       sync.Mutex
	hooksPending bool
	found        []core.FoundGame
	scanning     bool

	tray           *win.Tray
	quitting       bool
	trayHintShown  bool
	taskbarCreated uint32
	exePath        string

	logMu sync.Mutex
}

type Options struct {
	StartHidden bool
	LaunchGame  string
	MigrateErr  error // moving PlaySwap's data folder failed; it's used where it is
}

func newEnv(a *App) *core.Env {
	steamPath, _ := win.SteamInfo()
	home := os.Getenv("USERPROFILE")
	extra := []string{win.ProgramFiles(), win.ProgramFilesX86(), win.WindowsDir(), win.Desktop()}
	if sd := os.Getenv("SystemDrive"); sd != "" {
		extra = append(extra, sd+`\Users`, sd+`\ProgramData`)
	}
	return &core.Env{
		Getenv:     os.Getenv,
		Home:       home,
		Documents:  win.Documents(),
		SavedGames: win.SavedGames(),
		LocalLow:   win.LocalLow(),
		SteamPath:  steamPath,
		SteamUser: func() (string, error) {
			_, active := win.SteamInfo()
			configured := ""
			if a != nil {
				a.mu.Lock()
				if a.cfg != nil {
					configured = a.cfg.Settings.SteamAccountID
				}
				a.mu.Unlock()
			}
			return core.SteamUserDir(steamPath, configured, active)
		},
		SaveDB: func() *core.SaveDB {
			if a == nil {
				return nil
			}
			a.mu.Lock()
			on := a.cfg != nil && a.cfg.Settings.SaveDB
			a.mu.Unlock()
			if !on {
				return nil
			}
			return loadSaveDB()
		},
		DataDir:        DataDir(),
		ExtraProtected: extra,
	}
}

// Run starts the app and blocks until it quits.
func Run(opt Options) error {
	a := &App{sessions: map[string]*Session{}, requests: map[string]string{}}
	a.exePath = AppExe()
	a.store = &core.Store{Dir: DataDir()}
	if err := os.MkdirAll(a.store.Dir, 0o755); err != nil {
		return err
	}
	a.env = newEnv(a)
	a.eng = &core.Engine{Store: a.store, Env: a.env, Log: a.logf, Name: func(id string) string {
		a.mu.Lock()
		defer a.mu.Unlock()
		return a.cfg.PlayerName(id)
	}, Warn: func(title, text string) { a.notify(title, text) }}

	cfg, first, err := a.store.LoadConfig()
	if err != nil {
		a.loadErr = err.Error()
		a.logf("ERROR %v", err)
	}
	a.cfg, a.firstRun = cfg, first
	a.pending = opt.LaunchGame
	a.logf("Lootnest %s started (steam: %q)", Version, a.env.SteamPath)
	if opt.MigrateErr != nil {
		a.logf("Couldn't move PlaySwap's data to the Lootnest folder yet (%v); using %s for now", opt.MigrateErr, DataDir())
	}
	// A swap cut short when Lootnest closed or the PC turned off is done
	// again first, before anything saves the files in place.
	var recovered []string
	if err == nil && !first {
		var rerr error
		recovered, rerr = a.eng.RecoverSwaps(a.cfg, a.cfg.Settings.BackupsToKeep)
		if rerr != nil {
			a.logf("ERROR %v", rerr)
		}
	}
	// Games set up by older versions to swap Steam's whole folder for them
	// (with Steam Cloud's records): swap the folders inside it instead.
	if err == nil && !first {
		fixed := false
		for i := range a.cfg.Games {
			changed, serr := a.eng.SplitSteamAppFolder(a.cfg, &a.cfg.Games[i])
			if serr != nil {
				a.logf("ERROR updating the folders of %s: %v", a.cfg.Games[i].Name, serr)
			}
			fixed = fixed || changed
		}
		// A key saved in plain text by an earlier version gets encrypted.
		if k := a.cfg.Settings.SteamAPIKey; k != "" && !strings.HasPrefix(k, secretPrefix) {
			if enc, perr := protectSecret(k); perr == nil {
				a.cfg.Settings.SteamAPIKey, fixed = enc, true
			}
		}
		if k := a.cfg.Settings.GridDBKey; k != "" && !strings.HasPrefix(k, secretPrefix) {
			if enc, perr := protectSecret(k); perr == nil {
				a.cfg.Settings.GridDBKey, fixed = enc, true
			}
		}
		if fixed {
			if serr := a.store.SaveConfig(a.cfg); serr != nil {
				a.logf("ERROR %v", serr)
			}
		}
	}

	win.EnableDPIAwareness()
	win.AllowDarkMenus()
	dpi := win.SystemDPI()
	scale := func(v int) int { return v * dpi / 96 }

	a.w = webview2.NewWithOptions(webview2.WebViewOptions{
		AutoFocus: true,
		DataPath:  filepath.Join(DataDir(), "WebView2"),
		// The window only ever shows Lootnest's own page: nothing else can be
		// loaded or navigated to, so the page's access to Lootnest can't leak.
		AllowRequest: core.AllowedPageRequest,
		WindowOptions: webview2.WindowOptions{
			Title:       Name,
			Width:       uint(scale(1100)),
			Height:      uint(scale(740)),
			IconId:      1,
			Center:      true,
			ClassName:   WindowClass,
			Hidden:      opt.StartHidden && opt.LaunchGame == "",
			Background:  bgColor,
			OnCreate:    func(hwnd uintptr) { win.StyleTitleBar(hwnd, bgColor, borderColor); win.SetWindowIcons(hwnd, dpi) },
			MessageHook: a.hook,
		},
	})
	if a.w == nil {
		r := win.MessageBox(0, "Lootnest needs the Microsoft Edge WebView2 Runtime, which isn't installed on this PC.\n\nOpen the download page now?",
			Name, win.MB_YESNO|win.MB_ICONWARNING)
		if r == win.IDYES {
			_ = win.ShellOpen("https://developer.microsoft.com/microsoft-edge/webview2/")
		}
		return errors.New("WebView2 runtime missing")
	}
	a.hwnd = uintptr(a.w.Window())
	a.w.SetSize(scale(860), scale(580), webview2.HintMin)
	a.taskbarCreated = win.RegisterWindowMessage("TaskbarCreated")

	a.bind()
	a.w.SetHtml(pageHTML())

	a.tray = win.NewTray(a.hwnd, dpi)
	a.tray.Add(Name)
	if len(recovered) > 0 {
		names := make([]string, len(recovered))
		a.mu.Lock()
		for i, id := range recovered {
			names[i] = a.gameName(id)
		}
		a.mu.Unlock()
		a.notify("Finished an interrupted swap", "Lootnest closed in the middle of loading a profile's files for "+strings.Join(names, ", ")+". It's finished now, and nobody's saves were changed.")
	}

	if !opt.StartHidden || opt.LaunchGame != "" {
		a.show()
	}
	go a.watch()
	go func() {
		time.Sleep(2 * time.Second)
		a.startScan()
		a.autoHooks()
		go a.watchUpdates()
		go a.cleanupLegacy()
		cleanRestoreLeftovers()
		go a.watchGameSaves()
		time.Sleep(time.Minute)
		a.backupIfDue()
	}()
	go func() {
		time.Sleep(20 * time.Second)
		a.mu.Lock()
		ready := a.ready
		a.mu.Unlock()
		if !ready {
			a.logf("WARNING the interface hasn't loaded after 20s (WebView2 problem?)")
		}
	}()
	a.w.Run()
	a.tray.Remove()
	a.logf("Lootnest quit")
	return nil
}

// ----------------------------------------------------------- window glue --

func (a *App) hook(hwnd, msg, wp, lp uintptr) (uintptr, bool) {
	switch {
	case msg == win.WM_CLOSE:
		if a.quitting {
			return 0, false
		}
		a.mu.Lock()
		toTray := a.cfg.Settings.CloseToTray
		a.mu.Unlock()
		if toTray {
			a.cancelRequests()
			win.ShowWindow(hwnd, win.SW_HIDE)
			if !a.trayHintShown && a.tray != nil {
				a.trayHintShown = true
				a.tray.Notify("Lootnest is still running", "It keeps watching your games from the tray. Right-click the icon to quit.")
			}
			return 0, true
		}
		a.quit(false)
		return 0, true
	case msg == win.WM_COPYDATA:
		if cmd, ok := win.ReadCommand(lp); ok {
			a.handleCommand(cmd)
			return 1, true
		}
	case msg == win.WM_TRAY:
		switch lp & 0xFFFF {
		case win.NIN_SELECT, win.NIN_KEYSELECT, win.WM_LBUTTONUP, win.NIN_BALLOONUSERCLICK:
			a.show()
		case win.WM_CONTEXTMENU, win.WM_RBUTTONUP:
			a.trayMenu()
		}
		return 0, true
	case msg == win.WM_DPICHANGED:
		win.ApplyDPIRect(hwnd, lp)
		return 0, true
	case a.taskbarCreated != 0 && uint32(msg) == a.taskbarCreated:
		if a.tray != nil {
			a.tray.Add(Name)
			a.updateTrayTip()
		}
		return 0, true
	}
	return 0, false
}

func (a *App) trayMenu() {
	a.mu.Lock()
	items := []win.MenuItem{}
	for _, s := range a.sessions {
		if s.Phase == "playing" {
			items = append(items, win.MenuItem{Text: fmt.Sprintf("%s is playing %s", a.cfg.PlayerName(s.PlayerID), a.gameName(s.GameID)), Disabled: true, ID: 9999})
		}
	}
	items = append(items, win.MenuItem{ID: 1, Text: "Open Lootnest"}, win.MenuItem{Sep: true})
	games := a.cfg.Games
	for i, g := range games {
		if i >= 15 {
			break
		}
		items = append(items, win.MenuItem{ID: 100 + i, Text: "Play " + strings.ReplaceAll(g.Name, "&", "&&") + "…"})
	}
	if len(games) > 0 {
		items = append(items, win.MenuItem{Sep: true})
	}
	items = append(items, win.MenuItem{ID: 2, Text: "Quit Lootnest"})
	ids := make([]string, len(games))
	for i, g := range games {
		ids[i] = g.ID
	}
	a.mu.Unlock()

	switch id := win.ShowMenu(a.hwnd, items); {
	case id == 1:
		a.show()
	case id == 2:
		a.quit(false)
	case id >= 100 && id-100 < len(ids):
		a.requestLaunch(ids[id-100], "")
	}
}

func (a *App) handleCommand(cmd string) {
	switch {
	case cmd == "show":
		a.show()
	case strings.HasPrefix(cmd, "launch:"):
		a.requestLaunch(strings.TrimPrefix(cmd, "launch:"), "")
	case strings.HasPrefix(cmd, "steam:"):
		parts := strings.SplitN(strings.TrimPrefix(cmd, "steam:"), ":", 2)
		if len(parts) == 2 {
			if id, err := strconv.Atoi(parts[0]); err == nil {
				a.handleSteamRequest(id, parts[1])
			}
		}
	case strings.HasPrefix(cmd, "blocked:"):
		if id, err := strconv.Atoi(strings.TrimPrefix(cmd, "blocked:")); err == nil {
			a.handleSteamBlocked(id)
		}
	case cmd == "quit-now":
		a.quit(true)
	}
}

// requestLaunch brings the window up with the "who's playing?" picker.
// reqID is set when Steam is waiting for the answer.
func (a *App) requestLaunch(gameID, reqID string) {
	a.show()
	a.mu.Lock()
	ready := a.ready
	if !ready {
		a.pending, a.pendReq = gameID, reqID
	}
	a.mu.Unlock()
	if ready {
		a.emit("launch", map[string]any{"gameId": gameID, "reqId": reqID})
	}
}

func (a *App) quit(force bool) {
	if !force {
		a.mu.Lock()
		var playing []string
		for _, s := range a.sessions {
			if s.Phase == "playing" || s.Phase == "launching" {
				playing = append(playing, fmt.Sprintf("%s (%s)", a.gameName(s.GameID), a.cfg.PlayerName(s.PlayerID)))
			}
		}
		a.mu.Unlock()
		if name := movingName(); name != "" {
			win.MessageBox(a.hwnd, "Lootnest is moving "+name+" to another drive. Quit once it's done; Lootnest tells you when.", Name, win.MB_OK|win.MB_ICONINFORMATION)
			return
		}
		if len(playing) > 0 {
			msg := "Still playing: " + strings.Join(playing, ", ") + ".\n\nIf Lootnest quits now, that progress is saved the next time you pick a profile for the game in Lootnest. Quit anyway?"
			if win.MessageBox(a.hwnd, msg, Name, win.MB_YESNO|win.MB_ICONQUESTION|win.MB_DEFBUTTON2) != win.IDYES {
				return
			}
		}
	}
	a.quitting = true
	a.cancelRequests()
	if a.tray != nil {
		a.tray.Remove()
	}
	win.PostMessage(a.hwnd, win.WM_CLOSE, 0, 0)
}

// show brings the window up and makes sure the web view inside it is
// visible and sized (it can stay blank if the window was created hidden).
func (a *App) show() {
	a.checkIfStale()
	win.Present(a.hwnd)
	if a.w != nil {
		a.w.RefreshView()
	}
}

// ui runs f on the UI thread.
func (a *App) ui(f func()) {
	if a.w != nil {
		a.w.Dispatch(f)
	}
}

func (a *App) emit(typ string, payload map[string]any) {
	if payload == nil {
		payload = map[string]any{}
	}
	payload["type"] = typ
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	a.ui(func() { a.w.Eval("window.__ps&&window.__ps(" + string(b) + ")") })
}

func (a *App) notify(title, text string) {
	a.mu.Lock()
	on := a.cfg.Settings.Notifications
	a.mu.Unlock()
	if on && a.tray != nil {
		a.ui(func() { a.tray.Notify(title, text) })
	}
}

// ------------------------------------------------------------------- log --

func (a *App) logf(format string, args ...any) {
	a.logMu.Lock()
	defer a.logMu.Unlock()
	p := a.store.LogPath()
	if st, err := os.Stat(p); err == nil && st.Size() > 1<<20 {
		_ = os.Rename(p, p+".1")
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "[%s] %s\r\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, args...))
}

func dirOf(p string) string { return filepath.Dir(p) }
