//go:build windows

package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"playswap/internal/core"
	"playswap/internal/win"
)

// ---------------------------------------------------------------------------
// Steam launch options hand-off.
//
// A Steam game's launch options are set to
//     "…\PlaySwap.exe" --steam <appid> -- %command%
// so pressing Play in Steam starts a short-lived PlaySwap process (the
// "wrapper") with the game's real command line. The wrapper asks the main
// PlaySwap window to show the picker and swap files, waits for the answer
// in requests\<id>.json, then starts the game itself (so it inherits Steam's
// environment) and stays alive until the game closes.
// ---------------------------------------------------------------------------

type requestResult struct {
	Action  string `json:"action"` // "run" | "cancel"
	Message string `json:"message,omitempty"`
}

func requestsDir() string { return filepath.Join(DataDir(), "requests") }

func (a *App) answerRequest(reqID, action, msg string) {
	if reqID == "" {
		return
	}
	a.mu.Lock()
	delete(a.requests, reqID)
	a.mu.Unlock()
	_ = os.MkdirAll(requestsDir(), 0o755)
	b, _ := json.Marshal(requestResult{Action: action, Message: msg})
	p := filepath.Join(requestsDir(), filepath.Base(reqID)+".json")
	_ = os.WriteFile(p+".tmp", b, 0o644)
	_ = os.Rename(p+".tmp", p)
}

// cancelRequests answers every open Steam request with "cancel" (used when the
// window is closed with the picker still up).
func (a *App) cancelRequests() {
	a.mu.Lock()
	ids := make([]string, 0, len(a.requests))
	for id := range a.requests {
		ids = append(ids, id)
	}
	a.mu.Unlock()
	for _, id := range ids {
		a.answerRequest(id, "cancel", "")
	}
	if len(ids) > 0 {
		a.emit("pickerClosed", nil)
	}
}

// handleSteamRequest runs on the UI thread when a wrapper asks who's playing.
func (a *App) handleSteamRequest(appID int, reqID string) {
	a.mu.Lock()
	var game *core.Game
	for i := range a.cfg.Games {
		g := &a.cfg.Games[i]
		if g.Launch.Type == "steam" && g.Launch.AppID == appID {
			game = g
			break
		}
	}
	var s *Session
	gameID := ""
	if game != nil {
		gameID = game.ID
		s = a.sessions[gameID]
	}
	a.mu.Unlock()

	switch {
	case game == nil:
		a.logf("Steam started app %d, which PlaySwap doesn't manage; letting it run", appID)
		a.answerRequest(reqID, "run", "")
	case s != nil && (s.Phase == "launching" || s.Phase == "playing" || s.viaPlay):
		// PlaySwap's own Play button already swapped the files.
		a.answerRequest(reqID, "run", "")
	case s != nil:
		a.answerRequest(reqID, "cancel", "PlaySwap is still busy with this game.")
	default:
		a.mu.Lock()
		a.requests[reqID] = gameID
		a.mu.Unlock()
		a.requestLaunch(gameID, reqID)
	}
}

func (a *App) apiCancelRequest(reqID string) {
	a.answerRequest(reqID, "cancel", "")
}

func (a *App) apiCopy(text string) error {
	return win.SetClipboardText(a.hwnd, text)
}

type SteamHook struct {
	GameID   string `json:"gameId"`
	Name     string `json:"name"`
	AppID    int    `json:"appId"`
	Option   string `json:"option"`
	Current  string `json:"current"`
	Detected bool   `json:"detected"`
	Direct   bool   `json:"direct"` // Steam starts the game itself (Game.SteamDirect)
}

type SteamSetup struct {
	Games        []SteamHook `json:"games"`
	Error        string      `json:"error,omitempty"`
	SteamRunning bool        `json:"steamRunning"`
	Auto         bool        `json:"auto"`
}

func (a *App) apiSteamSetup() SteamSetup {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	out := SteamSetup{Games: []SteamHook{}, SteamRunning: steamRunning(), Auto: cfg.Settings.AutoSteamHooks}
	userDir, err := a.env.SteamUser()
	if err != nil {
		out.Error = err.Error()
	}
	for _, g := range cfg.Games {
		if g.Launch.Type != "steam" {
			continue
		}
		h := SteamHook{GameID: g.ID, Name: g.Name, AppID: g.Launch.AppID, Direct: g.SteamDirect}
		if userDir != "" {
			h.Current, _ = core.SteamLaunchOptions(userDir, g.Launch.AppID)
		}
		h.Detected = core.HasWrapper(h.Current, g.Launch.AppID)
		if h.Detected {
			h.Option = h.Current
		} else {
			h.Option = core.WrapperLaunchOption(a.exePath, g.Launch.AppID, h.Current)
		}
		out.Games = append(out.Games, h)
	}
	return out
}

// ------------------------------------------------------------- wrapper ----

// SteamWrapper is what runs when Steam starts a game through the launch
// options. It returns the exit code to use.
func SteamWrapper(appID int, command []string) int {
	if len(command) == 0 {
		win.MessageBox(0, "Steam didn't pass the game's command to PlaySwap.\n\nIn the game's launch options, make sure the PlaySwap line ends with %command%.",
			Name, win.MB_ICONWARNING)
		return 1
	}
	exe, _ := os.Executable()
	logf := func(format string, args ...any) {
		f, err := os.OpenFile(filepath.Join(DataDir(), "playswap.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err == nil {
			fmt.Fprintf(f, "[%s] %s\r\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, args...))
			f.Close()
		}
	}
	run := func() int {
		code, err := runSteamCommand(appID, command, logf)
		if err == nil {
			return code
		}
		logf("Steam wrapper: couldn't start %s: %v", command[0], err)
		if errors.Is(err, os.ErrPermission) {
			// Anti-cheats such as FACEIT AC only let Steam start the game.
			// The PlaySwap window offers to fix that in one click.
			if h := win.FindWindow(WindowClass); h != 0 && win.SendCommand(h, fmt.Sprintf("blocked:%d", appID)) {
				return 1
			}
			win.MessageBox(0, "An anti-cheat, such as FACEIT AC, stopped PlaySwap from starting the game. It only lets Steam start it.\n\n"+
				"Open PlaySwap, choose Edit game for this game and turn on \"Let Steam start this game\".", Name, win.MB_ICONWARNING)
			return 1
		}
		win.MessageBox(0, "Couldn't start the game:\n"+err.Error(), Name, win.MB_ICONERROR)
		return 1
	}

	h := win.FindWindow(WindowClass)
	if h == 0 {
		if err := win.StartDetached(exe, "--tray", filepath.Dir(exe)); err != nil {
			logf("Steam wrapper: couldn't start PlaySwap (%v); starting the game without asking", err)
			return run()
		}
		for i := 0; i < 200 && h == 0; i++ {
			time.Sleep(100 * time.Millisecond)
			h = win.FindWindow(WindowClass)
		}
		if h == 0 {
			logf("Steam wrapper: PlaySwap didn't open; starting the game without asking")
			return run()
		}
	}

	reqID := core.NewID("r")
	resPath := filepath.Join(requestsDir(), reqID+".json")
	if !win.SendCommand(h, fmt.Sprintf("steam:%d:%s", appID, reqID)) {
		logf("Steam wrapper: PlaySwap didn't answer; starting the game without asking")
		return run()
	}
	deadline := time.Now().Add(30 * time.Minute)
	for time.Now().Before(deadline) {
		time.Sleep(150 * time.Millisecond)
		b, err := os.ReadFile(resPath)
		if err != nil {
			if win.FindWindow(WindowClass) == 0 {
				logf("Steam wrapper: PlaySwap closed while waiting; starting the game without asking")
				return run()
			}
			continue
		}
		_ = os.Remove(resPath)
		var res requestResult
		_ = json.Unmarshal(b, &res)
		if res.Action == "cancel" {
			if res.Message != "" {
				win.MessageBox(0, res.Message, Name, win.MB_ICONWARNING)
			}
			return 0
		}
		return run()
	}
	return 0
}

// runSteamCommand starts the game exactly as Steam asked and waits until it
// (and the game's main process, if it relaunches itself) has closed, so
// Steam keeps showing the game as running. err is set when the game
// couldn't be started at all.
func runSteamCommand(appID int, command []string, logf func(string, ...any)) (int, error) {
	cmd := exec.Command(command[0], command[1:]...)
	if wd, err := os.Getwd(); err == nil {
		cmd.Dir = wd
	}
	if err := cmd.Start(); err != nil {
		return 1, err
	}
	_ = cmd.Wait()
	code := 0
	if cmd.ProcessState != nil {
		code = cmd.ProcessState.ExitCode()
	}

	// Some games relaunch themselves; keep waiting for the configured process.
	store := &core.Store{Dir: DataDir()}
	if cfg, _, err := store.LoadConfig(); err == nil {
		for _, g := range cfg.Games {
			if g.Launch.Type == "steam" && g.Launch.AppID == appID && g.ProcessName != "" {
				name := strings.ToLower(g.ProcessName)
				for {
					procs, err := win.ProcessNames()
					if err != nil || !procs[name] {
						break
					}
					time.Sleep(2 * time.Second)
				}
			}
		}
	}
	return code, nil
}

// handleSteamBlocked runs when an anti-cheat stopped the Steam wrapper from
// starting a game: end the stuck launch and offer the one-click fix.
func (a *App) handleSteamBlocked(appID int) {
	a.mu.Lock()
	gameID := findSteamGameID(a.cfg, appID)
	playerID := ""
	if s := a.sessions[gameID]; gameID != "" && s != nil && s.Phase == "launching" {
		playerID = s.PlayerID
		delete(a.sessions, gameID) // the player's files stay loaded
	}
	a.mu.Unlock()
	if gameID == "" {
		return
	}
	a.logf("An anti-cheat blocked %s from starting through PlaySwap", a.gameName(gameID))
	a.emitSessions()
	a.show()
	a.emit("blocked", map[string]any{"gameId": gameID, "playerId": playerID})
}

// apiFixBlocked runs fixBlocked in the background (closing Steam can take a
// while, and bound calls run on the UI thread). The result arrives as a
// "blockedDone" event.
func (a *App) apiFixBlocked(gameID, playerID string) error {
	go func() {
		need, err := a.fixBlocked(gameID, playerID)
		ev := map[string]any{"gameId": gameID, "playerId": playerID, "need": need.Need}
		if err != nil {
			a.logf("ERROR letting Steam start %s: %v", a.gameName(gameID), err)
			ev["error"] = err.Error()
		}
		a.emit("blockedDone", ev)
	}()
	return nil
}

// fixBlocked lets Steam start the game from now on (taking PlaySwap out of
// its launch options, restarting Steam if needed), then starts it for the
// player. Need "picker" means the UI should ask who's playing.
func (a *App) fixBlocked(gameID, playerID string) (Need, error) {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	g := cfg.Game(gameID)
	if g == nil || g.Launch.Type != "steam" {
		return Need{}, errors.New("That game isn't in PlaySwap anymore.")
	}
	if !g.SteamDirect {
		g.SteamDirect = true
		if _, err := a.apiSaveGameQuiet(*g); err != nil {
			return Need{}, err
		}
	}
	a.mu.Lock()
	restoresOnly := !a.cfg.Settings.AutoSteamHooks
	a.mu.Unlock()
	res, err := a.applyHooks(true, restoresOnly)
	if err != nil {
		return Need{}, err
	}
	if res.Restarted {
		// Give Steam a moment to come back before asking it to start the game.
		for i := 0; i < 60 && !steamRunning(); i++ {
			time.Sleep(500 * time.Millisecond)
		}
		time.Sleep(5 * time.Second)
	}
	if playerID == "" {
		return Need{Need: "picker"}, nil
	}
	need, err := a.apiPlay(swapRequest{GameID: gameID, PlayerID: playerID})
	if err == nil && need.Need != "" {
		return Need{Need: "picker"}, nil
	}
	return need, err
}

// ParseSteamArgs pulls "--steam <appid> -- <command...>" out of the arguments.
func ParseSteamArgs(args []string) (appID int, command []string, ok bool) {
	for i := 0; i < len(args); i++ {
		if strings.EqualFold(args[i], "--steam") && i+1 < len(args) {
			id, err := strconv.Atoi(args[i+1])
			if err != nil {
				return 0, nil, false
			}
			rest := args[i+2:]
			for j, a := range rest {
				if a == "--" {
					return id, rest[j+1:], true
				}
			}
			return id, rest, true
		}
	}
	return 0, nil, false
}
