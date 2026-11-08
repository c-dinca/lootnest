//go:build windows

package app

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// Session tracks one game being played.
type Session struct {
	GameID   string    `json:"gameId"`
	PlayerID string    `json:"playerId"`
	Phase    string    `json:"phase"` // loading, launching, playing, saving
	Outside  bool      `json:"outside"`
	Since    time.Time `json:"since"`
	deadline time.Time
	missed   int
	viaPlay  bool // started from Lootnest's own Play button
	front    bool // the game's window was in front at the last look
}

func (a *App) sessionsSnapshot() []Session {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]Session, 0, len(a.sessions))
	for _, s := range a.sessions {
		out = append(out, *s)
	}
	return out
}

func (a *App) emitSessions() {
	a.emit("sessions", map[string]any{"sessions": a.sessionsSnapshot(), "active": a.store.LoadState()})
	a.updateTrayTip()
}

func (a *App) updateTrayTip() {
	if a.tray == nil {
		return
	}
	a.mu.Lock()
	tip := "Lootnest"
	for _, s := range a.sessions {
		if s.Phase == "playing" {
			tip = fmt.Sprintf("Lootnest: %s is playing %s", a.cfg.PlayerName(s.PlayerID), a.gameName(s.GameID))
			break
		}
	}
	a.mu.Unlock()
	a.ui(func() { a.tray.SetTip(tip) })
}

func (a *App) gameName(id string) string {
	if g := a.cfg.Game(id); g != nil {
		return g.Name
	}
	return id
}

func procName(g *core.Game) string {
	return strings.ToLower(strings.TrimSuffix(g.ProcessName, ".exe"))
}

func isRunning(g *core.Game) bool {
	procs, err := win.ProcessNames()
	return err == nil && procs[procName(g)]
}

// Need is returned to the UI when a choice is still required.
type Need struct {
	Need  string         `json:"need,omitempty"` // "owner" | "newMode" | "cloud"
	Cloud *CloudConflict `json:"cloud,omitempty"`
}

type swapRequest struct {
	GameID     string `json:"gameId"`
	PlayerID   string `json:"playerId"`
	FirstOwner string `json:"firstOwner"`
	NewProfile string `json:"newProfile"`
	CopyFrom   string `json:"copyFrom"` // with NewProfile "copy": whose profile to copy
	Launch     bool   `json:"launch"`
	ReqID      string `json:"reqId"` // Steam request waiting for this swap
	// Cloud answers a "cloud" need: "here" keeps this PC's saves, "cloud"
	// takes the cloud's.
	Cloud string `json:"cloud"`
}

// startSwap validates a play/load request and runs it in the background.
func (a *App) startSwap(req swapRequest) (Need, error) {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	g := cfg.Game(req.GameID)
	if g == nil {
		a.mu.Unlock()
		return Need{}, errors.New("That game no longer exists.")
	}
	if cfg.Player(req.PlayerID) == nil {
		a.mu.Unlock()
		return Need{}, errors.New("That profile no longer exists.")
	}
	if s := a.sessions[g.ID]; s != nil {
		a.mu.Unlock()
		if s.Phase == "playing" {
			return Need{}, fmt.Errorf("%s is already being played by %s.", g.Name, cfg.PlayerName(s.PlayerID))
		}
		return Need{}, fmt.Errorf("%s is busy right now; try again in a moment.", g.Name)
	}
	a.mu.Unlock()

	if g.Launch.Type != "steam" {
		if err := a.errMoving(g.Launch.Exe); err != nil {
			return Need{}, err
		}
	}
	if isRunning(g) {
		return Need{}, fmt.Errorf("%s is already running. Close it first, then pick who's playing.", g.Name)
	}
	if err := a.eng.CheckPaths(g); err != nil {
		return Need{}, err
	}
	active := a.store.LoadState()[g.ID]
	if active != "" && cfg.Player(active) == nil {
		active = ""
	}
	if active == "" && req.FirstOwner == "" {
		return Need{Need: "owner"}, nil
	}
	// Cloud saves: newer saves from another PC come down first; saves that
	// changed on both need the player's choice.
	plan, conflict := a.beforePlay(cfg, g, req.PlayerID, req.Cloud)
	if conflict != nil {
		return Need{Need: "cloud", Cloud: conflict}, nil
	}
	fromCloud := plan != nil && (plan.Action == core.SyncDown || plan.Action == core.SyncConflict && req.Cloud == "cloud")
	if active != req.PlayerID && req.FirstOwner != req.PlayerID && !a.eng.HasProfile(req.PlayerID, g.ID) && req.NewProfile == "" && !fromCloud {
		return Need{Need: "newMode"}, nil
	}

	a.mu.Lock()
	if a.sessions[g.ID] != nil {
		a.mu.Unlock()
		return Need{}, fmt.Errorf("%s is busy right now; try again in a moment.", g.Name)
	}
	a.sessions[g.ID] = &Session{GameID: g.ID, PlayerID: req.PlayerID, Phase: "loading", Since: time.Now(), viaPlay: req.Launch}
	a.mu.Unlock()
	a.emitSessions()
	if req.Launch {
		_ = a.store.RecordPlay("game:"+g.ID, req.PlayerID, time.Now())
	}

	go a.runSwap(cfg, g, req, plan)
	return Need{}, nil
}

func (a *App) runSwap(cfg *core.Config, g *core.Game, req swapRequest, plan *syncPlan) {
	who := cfg.PlayerName(req.PlayerID)
	fail := func(err error) {
		a.mu.Lock()
		delete(a.sessions, g.ID)
		a.mu.Unlock()
		a.logf("ERROR %s / %s: %v", g.Name, who, err)
		a.emit("toast", map[string]any{"kind": "error", "text": err.Error()})
		a.emitSessions()
	}
	if err := a.applyBeforePlay(cfg, g, req.PlayerID, req.Cloud, plan); err != nil {
		a.answerRequest(req.ReqID, "cancel", "")
		fail(fmt.Errorf("Couldn't bring %s's saves for %s from the cloud, so nothing was changed: %v", who, g.Name, err))
		return
	}
	prev := a.store.LoadState()[g.ID]
	err := a.eng.Activate(cfg, g.ID, req.PlayerID, core.ActivateOptions{
		FirstOwner: req.FirstOwner,
		NewProfile: req.NewProfile,
		CopyFrom:   req.CopyFrom,
		Keep:       cfg.Settings.BackupsToKeep,
	})
	if err != nil {
		a.answerRequest(req.ReqID, "cancel", "")
		fail(err)
		return
	}
	if prev != "" && prev != req.PlayerID {
		go a.syncUp(cfg, g, prev) // their files went back to their profile
	}
	if req.ReqID != "" || req.Launch {
		// The profile's saves are in place now: if the game updated since
		// it last started, they're kept as a point before it opens them.
		if a.updatedSince("game:"+g.ID, a.profileBuild(g)) {
			a.pointBeforeUpdate("game:" + g.ID)
		}
		a.armCloudWatch(g)
		a.stepsBefore("game:"+g.ID, g.Name)
	}
	if req.ReqID != "" {
		// Steam's wrapper starts the game; the watcher follows it from here.
		a.mu.Lock()
		if s := a.sessions[g.ID]; s != nil {
			s.Phase = "launching"
			s.deadline = time.Now().Add(time.Duration(cfg.Settings.StartTimeoutSeconds) * time.Second)
		}
		a.mu.Unlock()
		a.answerRequest(req.ReqID, "run", "")
		a.logf("Steam is starting %s for %s", g.Name, who)
		a.emitSessions()
		return
	}
	if !req.Launch {
		a.mu.Lock()
		delete(a.sessions, g.ID)
		a.mu.Unlock()
		a.emit("toast", map[string]any{"kind": "ok", "text": fmt.Sprintf("%s's files are now loaded for %s.", who, g.Name)})
		a.emitSessions()
		return
	}
	if err := a.launchGame(cfg, g, req.PlayerID); err != nil {
		// The swap succeeded; files stay loaded for this player.
		a.stepsAfter("game:"+g.ID, false)
		fail(fmt.Errorf("%s's files are loaded, but %s didn't launch: %v", who, g.Name, err))
		return
	}
	a.logf("Launched %s for %s", g.Name, who)
	a.mu.Lock()
	if s := a.sessions[g.ID]; s != nil {
		s.Phase = "launching"
		s.deadline = time.Now().Add(time.Duration(cfg.Settings.StartTimeoutSeconds) * time.Second)
	}
	a.mu.Unlock()
	a.emitSessions()
}

func (a *App) launchGame(cfg *core.Config, g *core.Game, playerID string) error {
	args := strings.TrimSpace(g.Launch.Args + " " + g.PlayerArgs[playerID])
	switch g.Launch.Type {
	case "steam":
		if a.env.SteamPath == "" {
			return errors.New("Steam wasn't found on this PC")
		}
		line := fmt.Sprintf("-applaunch %d", g.Launch.AppID)
		if args != "" {
			line += " " + args
		}
		return win.StartDetached(a.env.SteamPath+`\steam.exe`, line, a.env.SteamPath)
	case "exe":
		exe, err := a.env.ResolveExe(g.Launch.Exe)
		if err != nil {
			return err
		}
		return a.startExe("game:"+g.ID, exe, args)
	case "uri":
		if !core.SafeLaunchURI(g.Launch.URI) {
			return errors.New("Lootnest can't start this game through its launcher; choose its .exe in Edit game")
		}
		return win.ShellOpen(g.Launch.URI)
	}
	return fmt.Errorf("unknown launch type %q", g.Launch.Type)
}

// watch runs forever: it follows every managed game's process, finishes
// sessions when games close, and picks up games launched outside Lootnest.
func (a *App) watch() {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	n := 0
	for range t.C {
		a.tick()
		n++
		if n%3 == 0 && !a.safe { // every ~6 s: achievements unlocked in a running game, and the hotkeys
			if procs, err := win.ProcessNames(); err == nil {
				a.achTick(procs)
			}
		}
		if n%60 == 0 && !a.safe { // every ~2 min: updates that landed in Downloads
			go a.downloadsCheck()
		}
		if n%15 == 0 { // every ~30 s: finish Steam set-up once Steam is closed
			a.mu.Lock()
			pending := a.hooksPending
			a.mu.Unlock()
			if pending && !steamRunning() {
				a.autoHooksAlways()
			}
		}
	}
}

func (a *App) tick() {
	procs, err := win.ProcessNames()
	if err != nil {
		return
	}
	a.foundTick(procs)
	state := a.store.LoadState()
	changed := false
	type finish struct {
		cfg  *core.Config
		game *core.Game
		pid  string
	}
	var finishes []finish
	var notes [][2]string
	hide := ""    // a game Lootnest started is now up: the window steps aside
	back := false // the game Big Picture hid for closed while in front

	a.mu.Lock()
	cfg := a.cfg
	for i := range cfg.Games {
		g := &cfg.Games[i]
		running := procs[procName(g)]
		s := a.sessions[g.ID]
		switch {
		case s == nil && running:
			if !cfg.Settings.WatchOutsideLaunches {
				continue
			}
			owner := state[g.ID]
			if owner == "" || cfg.Player(owner) == nil {
				continue
			}
			a.sessions[g.ID] = &Session{GameID: g.ID, PlayerID: owner, Phase: "playing", Outside: true, Since: time.Now()}
			changed = true
			go a.store.RecordPlay("game:"+g.ID, owner, time.Now())
			a.logf("%s started outside Lootnest; %s's files are loaded", g.Name, cfg.PlayerName(owner))
			who := cfg.PlayerName(owner)
			if g.Launch.Type == "steam" && g.SteamDirect {
				notes = append(notes, [2]string{fmt.Sprintf("%s started with %s's settings", g.Name, who),
					fmt.Sprintf("Not %s? Quit the game and start it from the %s (Lootnest) shortcut to choose who's playing.", who, g.Name)})
			} else {
				notes = append(notes, [2]string{g.Name + " started",
					fmt.Sprintf("It's using %s's settings and saves. Progress will be saved to %s when the game closes.", who, who)})
			}
		case s != nil && s.Phase == "launching":
			if running {
				s.Phase = "playing"
				s.Since = time.Now()
				changed = true
				// Hidden only now, once the game is running, so the window
				// doesn't vanish to the desktop while the game is loading.
				if cfg.Settings.HideWhilePlaying {
					hide = g.ID
				}
				c := cfg.Clone()
				go a.checkCloudWatch(c, c.Game(g.ID), s.PlayerID)
			} else if time.Now().After(s.deadline) {
				delete(a.sessions, g.ID)
				changed = true
				go a.stepsAfter("game:"+g.ID, false)
				notes = append(notes, [2]string{g.Name + " didn't start",
					fmt.Sprintf("%s's files stay loaded. If it starts later, Lootnest will still save it.", cfg.PlayerName(s.PlayerID))})
			}
		case s != nil && s.Phase == "playing":
			if running {
				s.missed = 0
				if g.ID == a.hidFor {
					s.front = win.ForegroundIs(procName(g))
				}
				continue
			}
			s.missed++
			if s.missed < 2 { // ignore quick restarts (launcher -> game)
				continue
			}
			if g.ID == a.hidFor {
				a.hidFor, back = "", s.front
			}
			go func(g core.Game, player string, since time.Time) {
				now, key := time.Now(), "game:"+g.ID
				_ = a.store.AddPlaytime(key, player, now.Sub(since))
				_ = a.store.AddActivity(key, player, since, now)
				a.logSession(core.SessionRecord{Game: key, Profile: player, Start: since, End: now, Source: "profile", Build: a.profileBuild(&g)})
			}(*g, s.PlayerID, s.Since)
			s.Phase = "saving"
			changed = true
			finishes = append(finishes, finish{cfg.Clone(), cfg.Game(g.ID), s.PlayerID})
		}
	}
	a.mu.Unlock()

	if changed {
		a.emitSessions()
	}
	if hide != "" {
		a.ui(func() {
			a.mu.Lock()
			if a.bigPicture && win.IsVisible(a.hwnd) {
				a.hidFor = hide
			}
			a.mu.Unlock()
			win.ShowWindow(a.hwnd, win.SW_HIDE)
		})
	}
	if back {
		// Back to Big Picture, where the game was started, unless the window
		// was opened or Big Picture left while it ran.
		a.ui(func() {
			a.mu.Lock()
			bp := a.bigPicture
			a.mu.Unlock()
			if bp && !win.IsVisible(a.hwnd) {
				a.show()
				a.emit("bpReturn", nil)
			}
		})
	}
	for _, n := range notes {
		a.notify(n[0], n[1])
	}
	for _, f := range finishes {
		go a.finishSession(f.cfg, f.game, f.pid)
	}
}

func (a *App) finishSession(cfg *core.Config, g *core.Game, playerID string) {
	defer a.stepsAfter("game:"+g.ID, true)
	time.Sleep(3 * time.Second) // let the game and Steam finish writing
	// The build that ran, if it updated as it started, so the next start
	// doesn't take it for a new update.
	_, _ = a.store.NoteBuild("game:"+g.ID, a.profileBuild(g))
	who := cfg.PlayerName(playerID)
	if takeCloudReplaced(g.ID) {
		// Steam Cloud's copy, not this player's files: keep it as a backup and
		// put the player's own profile back, instead of saving over it.
		dir, err := a.eng.RevertSession(cfg, g.ID, playerID, cfg.Settings.BackupsToKeep)
		a.mu.Lock()
		delete(a.sessions, g.ID)
		a.mu.Unlock()
		if err != nil {
			a.logf("ERROR putting %s's files back for %s: %v", who, g.Name, err)
			a.notify("Couldn't put "+who+"'s files back", err.Error())
		} else {
			a.logf("Kept %s's profile for %s; this session's files (from Steam Cloud) are in %s", who, g.Name, dir)
			a.notify("Kept "+who+"'s "+g.Name+" profile", "This session used Steam Cloud's copy, so it was saved as a backup instead. "+who+"'s own files are loaded again.")
		}
		a.emitSessions()
		return
	}
	err := a.eng.FinishSession(cfg, g.ID, playerID)
	a.mu.Lock()
	delete(a.sessions, g.ID)
	a.mu.Unlock()
	if err != nil {
		a.logf("ERROR saving %s for %s: %v", g.Name, who, err)
		a.notify("Couldn't save "+g.Name, err.Error())
		a.emit("toast", map[string]any{"kind": "error", "text": err.Error()})
	} else {
		a.notify("Saved "+who+"'s "+g.Name, "Settings and saves are stored in "+who+"'s profile.")
		a.queueBackup()
		go a.syncUp(cfg, g, playerID)
	}
	a.emitSessions()
}

// apiCloseGame asks a running game to quit, as if its window was closed, so
// it saves and exits the way it normally does. The watcher then finishes
// the session as usual: the player's progress goes back to their profile.
func (a *App) apiCloseGame(gameID string) error {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	g := cfg.Game(gameID)
	if g == nil {
		return errors.New("That game no longer exists.")
	}
	if !isRunning(g) {
		return fmt.Errorf("%s isn't running. Lootnest finishes the session in a few seconds.", g.Name)
	}
	if win.CloseProcessWindows(procName(g)) == 0 {
		return fmt.Errorf("Lootnest couldn't close %s. Quit it from the game's own menu; the progress is saved when it closes.", g.Name)
	}
	a.logf("Asked %s to close", g.Name)
	return nil
}

// apiProfileInfo reads, off the UI thread, when each player's profile for
// a game was last saved and how big it is, for the game page and picker.
// It arrives as a "profileInfo" event.
func (a *App) apiProfileInfo(gameID string) {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	go func() {
		out := map[string]core.ProfileInfo{}
		for _, p := range cfg.Players {
			if info, ok := a.eng.ProfileInfo(p.ID, gameID); ok {
				out[p.ID] = info
			}
		}
		a.emit("profileInfo", map[string]any{"gameId": gameID, "info": out})
	}()
}
