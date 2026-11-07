// Package core holds Lootnest's platform-neutral logic: configuration,
// path resolution, profile storage and file swapping.
package core

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
)

const ConfigVersion = 1

type Player struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	// Profile page (see ValidatePlayers for the limits):
	// - Tagline: a line under the name; Bio: a few lines about them.
	// - Avatar: a picture is saved as avatars/<id>.png; without one,
	//   AvatarGame (a Library key) shows that game's cover, else the
	//   initial.
	// - Banner: "" (their colour), "image" (banners/<id>.jpg) or a Library
	//   key (that game's art). BannerY: which part of the picture shows,
	//   0-100 from the top.
	// - Now / NowNote: the game they're into right now, and a word on it.
	// - Favorites: up to six Library keys, in order.
	// - Sections: which parts of the page show, in order (ProfileSections).
	Tagline    string   `json:"tagline,omitempty"`
	Bio        string   `json:"bio,omitempty"`
	Avatar     bool     `json:"avatar,omitempty"`
	AvatarGame string   `json:"avatarGame,omitempty"`
	Banner     string   `json:"banner,omitempty"`
	BannerY    int      `json:"bannerY,omitempty"`
	Now        string   `json:"now,omitempty"`
	NowNote    string   `json:"nowNote,omitempty"`
	Favorites  []string `json:"favorites,omitempty"`
	Sections   []string `json:"sections,omitempty"`
}

// ProfileSections are the parts a profile page can show, in their default
// order.
var ProfileSections = []string{"now", "stats", "favorites", "activity", "shelf", "collection", "recent"}

type PathEntry struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

type Launch struct {
	// Type is "steam", "exe" or "uri" (started through its launcher's link,
	// see SafeLaunchURI; Exe is then the game's own exe, for its icon).
	Type  string `json:"type"`
	AppID int    `json:"appId,omitempty"`
	Exe   string `json:"exe,omitempty"`
	URI   string `json:"uri,omitempty"`
	Args  string `json:"args,omitempty"`
}

type Game struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Launch      Launch            `json:"launch"`
	ProcessName string            `json:"processName"`
	PlayerArgs  map[string]string `json:"playerArgs,omitempty"`
	Paths       []PathEntry       `json:"paths"`
	// SteamPrevOptions are the Steam launch options the game had before
	// Lootnest added itself; they're put back when Lootnest lets go.
	SteamPrevOptions string `json:"steamPrevOptions,omitempty"`
	// SteamDirect keeps Lootnest out of the game's Steam launch options, so
	// Steam starts the game itself. Needed when an anti-cheat (FACEIT AC)
	// refuses to let any other program start the game.
	SteamDirect bool `json:"steamDirect,omitempty"`
}

// UsesSteamWrapper reports whether Steam's Play button should go through
// Lootnest for this game.
func (g *Game) UsesSteamWrapper() bool {
	return g.Launch.Type == "steam" && !g.SteamDirect
}

type Settings struct {
	StartMinimized       bool   `json:"startMinimized"`
	CloseToTray          bool   `json:"closeToTray"`
	HideWhilePlaying     bool   `json:"hideWhilePlaying"`
	WatchOutsideLaunches bool   `json:"watchOutsideLaunches"`
	Notifications        bool   `json:"notifications"`
	BackupsToKeep        int    `json:"backupsToKeep"`
	StartTimeoutSeconds  int    `json:"startTimeoutSeconds"`
	SteamAccountID       string `json:"steamAccountId"`
	AutoSteamHooks       bool   `json:"autoSteamHooks"`
	// DownloadCovers lets Lootnest look games outside Steam up on the Steam
	// store by name and save their cover. Off by default: the only network
	// access Lootnest has, and only when the user turns it on.
	DownloadCovers bool `json:"downloadCovers"`
	// CheckUpdates asks GitHub once a day whether a newer Lootnest exists.
	// Off by default; "Check now" in Settings works either way.
	CheckUpdates bool `json:"checkUpdates"`
	// GameNews shows each game's latest updates and announcements from
	// Steam's news feed on its page. Off by default.
	GameNews bool `json:"gameNews"`
	// AutoInstallUpdates installs a newer Lootnest found by the automatic
	// check by itself, while the window is closed to the tray and no game
	// is running. UpdatesAsked: the user answered "Keep Lootnest up to date?".
	AutoInstallUpdates bool `json:"autoInstallUpdates"`
	UpdatesAsked       bool `json:"updatesAsked"`
	// GridDBKey is the user's own SteamGridDB API key (optional): with
	// DownloadCovers on, art for games the Steam store doesn't know comes
	// from SteamGridDB.
	GridDBKey string `json:"gridDbKey,omitempty"`
	// Backups to a folder the user picks (usually one a cloud app syncs):
	// BackupDir is that folder (backups go in <dir>\Lootnest), BackupAuto
	// makes one after each session, BackupKeep is how many this PC keeps.
	// With a password, BackupSalt (base64) and BackupKey (the key made from
	// the password, DPAPI-encrypted) encrypt them; the password isn't kept.
	BackupDir  string `json:"backupDir,omitempty"`
	BackupAuto bool   `json:"backupAuto,omitempty"`
	BackupKeep int    `json:"backupKeep,omitempty"`
	BackupSalt string `json:"backupSalt,omitempty"`
	BackupKey  string `json:"backupKey,omitempty"`
	// CloudSync keeps each player's saves in BackupDir up to date and brings
	// newer ones from another PC before a game starts (cloudsync.go).
	CloudSync bool `json:"cloudSync,omitempty"`
	// GameSaves keeps versions of the saves of Library games outside Steam
	// in BackupDir (gamesaves.go); SaveDB downloads the list of where games
	// keep their saves (Ludusavi's, from PCGamingWiki) to find them.
	GameSaves bool `json:"gameSaves,omitempty"`
	SaveDB    bool `json:"saveDb,omitempty"`
}

type Config struct {
	Version   int      `json:"version"`
	Players   []Player `json:"players"`
	Games     []Game   `json:"games"`
	Settings  Settings `json:"settings"`
	Shortcuts []string `json:"shortcuts,omitempty"` // desktop shortcuts Lootnest created, removed on uninstall
	// SteamRestores: launch options to put back for Steam games removed from
	// Lootnest (app id -> original options), applied when Steam is closed.
	SteamRestores map[string]string `json:"steamRestores,omitempty"`
	// IgnoredFound: scan results the user chose to hide.
	IgnoredFound []string `json:"ignoredFound,omitempty"`
	// LibraryPrefs: settings for Library games without profiles, by scan key.
	LibraryPrefs map[string]LibraryPref `json:"libraryPrefs,omitempty"`
	// AddedGames: games the user added to the Library by hand ("Add game"),
	// kept even when a scan doesn't find them.
	AddedGames []FoundGame `json:"addedGames,omitempty"`
}

// LibraryPref is what the user changed about a Library game that has no
// profiles: the name it's shown with and options added when it starts.
type LibraryPref struct {
	Name string `json:"name,omitempty"`
	Args string `json:"args,omitempty"`
}

// CleanLibraryPref trims a game's settings and checks they're sensible.
func CleanLibraryPref(p LibraryPref) (LibraryPref, error) {
	p.Name = strings.TrimSpace(p.Name)
	p.Args = strings.TrimSpace(p.Args)
	if len(p.Name) > 120 {
		return p, errors.New("That name is too long.")
	}
	if len(p.Args) > 1000 {
		return p, errors.New("Those launch options are too long.")
	}
	if strings.ContainsAny(p.Name+p.Args, "\r\n\t\x00") {
		return p, errors.New("Names and launch options must fit on one line.")
	}
	return p, nil
}

// PlayerColors is the palette offered in the UI. The first two are the defaults.
var PlayerColors = []string{"#6C9BD2", "#D9738E", "#5FB49C", "#C9A35F", "#9C88D0", "#D98A5F"}

// oldPlayerColors are the brighter colours versions up to 1.3 offered; each
// maps to the calmer colour at the same place in PlayerColors.
var oldPlayerColors = []string{"#4F8CFF", "#FF5D8F", "#2EC4A0", "#FFB020", "#A77BFF", "#FF7A45"}

func DefaultConfig() *Config {
	return &Config{
		Version: ConfigVersion,
		Players: []Player{
			{ID: "p1", Name: "Player 1", Color: PlayerColors[0]},
			{ID: "p2", Name: "Player 2", Color: PlayerColors[1]},
		},
		Games: []Game{},
		Settings: Settings{
			StartMinimized:       true,
			CloseToTray:          true,
			HideWhilePlaying:     true,
			WatchOutsideLaunches: true,
			Notifications:        true,
			BackupsToKeep:        10,
			StartTimeoutSeconds:  240,
			AutoSteamHooks:       true,
		},
	}
}

func (c *Config) normalize() {
	for i := range c.Players {
		for j, old := range oldPlayerColors {
			if strings.EqualFold(c.Players[i].Color, old) {
				c.Players[i].Color = PlayerColors[j]
			}
		}
	}
	if c.Version == 0 {
		c.Version = ConfigVersion
	}
	if c.Players == nil {
		c.Players = []Player{}
	}
	if c.Games == nil {
		c.Games = []Game{}
	}
	if c.Settings.BackupsToKeep <= 0 {
		c.Settings.BackupsToKeep = 10
	}
	if c.Settings.StartTimeoutSeconds <= 0 {
		c.Settings.StartTimeoutSeconds = 240
	}
	for i := range c.Games {
		if c.Games[i].Paths == nil {
			c.Games[i].Paths = []PathEntry{}
		}
	}
}

func (c *Config) Game(id string) *Game {
	for i := range c.Games {
		if c.Games[i].ID == id {
			return &c.Games[i]
		}
	}
	return nil
}

func (c *Config) Player(id string) *Player {
	for i := range c.Players {
		if c.Players[i].ID == id {
			return &c.Players[i]
		}
	}
	return nil
}

func (c *Config) PlayerName(id string) string {
	if p := c.Player(id); p != nil {
		return p.Name
	}
	return id
}

func (c *Config) Clone() *Config {
	b, _ := json.Marshal(c)
	var out Config
	_ = json.Unmarshal(b, &out)
	return &out
}

func NewID(prefix string) string {
	b := make([]byte, 5)
	_, _ = rand.Read(b)
	return prefix + hex.EncodeToString(b)
}

var idRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,40}$`)
var colorRe = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// ValidateGame checks a game definition. It does not touch the filesystem.
func ValidateGame(g *Game) error {
	g.Name = strings.TrimSpace(g.Name)
	g.ProcessName = strings.TrimSpace(g.ProcessName)
	g.ProcessName = strings.TrimSuffix(strings.TrimSuffix(g.ProcessName, ".exe"), ".EXE")
	if g.Name == "" {
		return errors.New("Give the game a name.")
	}
	if !idRe.MatchString(g.ID) {
		return fmt.Errorf("invalid game id %q", g.ID)
	}
	switch g.Launch.Type {
	case "steam":
		if g.Launch.AppID <= 0 {
			return errors.New("Pick the Steam game to launch.")
		}
	case "exe":
		if strings.TrimSpace(g.Launch.Exe) == "" {
			return errors.New("Choose the game's .exe file.")
		}
	case "uri":
		if !SafeLaunchURI(g.Launch.URI) {
			return errors.New("Lootnest can't start this game through its launcher; choose its .exe instead.")
		}
	default:
		return errors.New("Choose how the game is launched.")
	}
	if g.ProcessName == "" || strings.ContainsAny(g.ProcessName, `\/:*?"<>|`) {
		return errors.New("Set the game's process name (the .exe that runs while you play).")
	}
	if len(g.Paths) == 0 {
		return errors.New("Add at least one folder or file that holds the game's settings or saves.")
	}
	seen := map[string]bool{}
	for i := range g.Paths {
		p := &g.Paths[i]
		p.Path = strings.TrimSpace(p.Path)
		if p.ID == "" {
			p.ID = NewID("d")
		}
		if !idRe.MatchString(p.ID) {
			return fmt.Errorf("invalid path id %q", p.ID)
		}
		if p.Path == "" {
			return errors.New("One of the save locations is empty.")
		}
		key := strings.ToLower(p.Path)
		if seen[key] {
			return fmt.Errorf("%s is listed twice.", p.Path)
		}
		seen[key] = true
	}
	if g.PlayerArgs == nil {
		g.PlayerArgs = map[string]string{}
	}
	return nil
}

func ValidatePlayers(ps []Player) error {
	if len(ps) < 1 {
		return errors.New("Keep at least one player.")
	}
	if len(ps) > 8 {
		return errors.New("Lootnest supports up to 8 players.")
	}
	names := map[string]bool{}
	ids := map[string]bool{}
	for i := range ps {
		p := &ps[i]
		p.Name = strings.TrimSpace(p.Name)
		if p.Name == "" {
			return errors.New("Every player needs a name.")
		}
		if len([]rune(p.Name)) > 24 {
			return fmt.Errorf("%q is too long; keep names under 24 characters.", p.Name)
		}
		if names[strings.ToLower(p.Name)] {
			return fmt.Errorf("Two players are called %s.", p.Name)
		}
		names[strings.ToLower(p.Name)] = true
		if p.ID == "" {
			p.ID = NewID("p")
		}
		if !idRe.MatchString(p.ID) || ids[p.ID] {
			return fmt.Errorf("invalid player id %q", p.ID)
		}
		ids[p.ID] = true
		p.Tagline = strings.TrimSpace(p.Tagline)
		if len([]rune(p.Tagline)) > 80 || strings.ContainsAny(p.Tagline, "\r\n") {
			return fmt.Errorf("%s's line is too long; keep it to one line under 80 characters.", p.Name)
		}
		p.Bio = strings.TrimSpace(strings.ReplaceAll(p.Bio, "\r\n", "\n"))
		if len([]rune(p.Bio)) > 300 || strings.Count(p.Bio, "\n") > 5 {
			return fmt.Errorf("%s's about is too long; keep it under 300 characters and 6 lines.", p.Name)
		}
		p.NowNote = strings.TrimSpace(p.NowNote)
		if len([]rune(p.NowNote)) > 80 || strings.ContainsAny(p.NowNote, "\r\n") {
			return fmt.Errorf("%s's note on the game they're into is too long; keep it to one line under 80 characters.", p.Name)
		}
		if len(p.Favorites) > 6 {
			return fmt.Errorf("%s can show up to 6 favourite games.", p.Name)
		}
		for _, k := range append([]string{p.Banner, p.AvatarGame, p.Now}, p.Favorites...) {
			if len(k) > 400 || strings.ContainsAny(k, "\r\n\x00") {
				return errors.New("invalid game in a profile")
			}
		}
		p.BannerY = min(max(p.BannerY, 0), 100)
		seen := map[string]bool{}
		secs := p.Sections[:0:0]
		for _, sec := range p.Sections {
			if slices.Contains(ProfileSections, sec) && !seen[sec] {
				seen[sec] = true
				secs = append(secs, sec)
			}
		}
		p.Sections = secs
		if !colorRe.MatchString(p.Color) {
			p.Color = PlayerColors[i%len(PlayerColors)]
		}
	}
	return nil
}

// ---------------------------------------------------------------- storage ---

// Store owns the on-disk layout:
//
//	<dir>/config.json
//	<dir>/state.json            gameID -> playerID whose files are live
//	<dir>/profiles/<player>/<game>/<pathID>
//	<dir>/backups/<game>/<timestamp>/<pathID>
//	<dir>/lootnest.log
type Store struct {
	Dir string
	mu  sync.Mutex
}

func (s *Store) ConfigPath() string  { return filepath.Join(s.Dir, "config.json") }
func (s *Store) StatePath() string   { return filepath.Join(s.Dir, "state.json") }
func (s *Store) ProfilesDir() string { return filepath.Join(s.Dir, "profiles") }
func (s *Store) BackupsDir() string  { return filepath.Join(s.Dir, "backups") }
func (s *Store) LogPath() string     { return filepath.Join(s.Dir, "lootnest.log") }
func (s *Store) ProfileDir(playerID, gameID string) string {
	return filepath.Join(s.ProfilesDir(), playerID, gameID)
}

// LoadConfig returns the saved config, or defaults (and firstRun=true) if none exists.
func (s *Store) LoadConfig() (cfg *Config, firstRun bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := os.ReadFile(s.ConfigPath())
	if errors.Is(err, os.ErrNotExist) {
		return DefaultConfig(), true, nil
	}
	if err != nil {
		return nil, false, err
	}
	cfg = DefaultConfig()
	cfg.Games = nil
	cfg.Players = nil
	if err := json.Unmarshal(b, cfg); err != nil {
		// Keep the broken file for the user, start from defaults.
		_ = os.Rename(s.ConfigPath(), s.ConfigPath()+".broken")
		return DefaultConfig(), true, fmt.Errorf("config.json was unreadable and was renamed to config.json.broken: %w", err)
	}
	cfg.normalize()
	return cfg, false, nil
}

func (s *Store) SaveConfig(cfg *Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg.normalize()
	return writeJSONAtomic(s.ConfigPath(), cfg)
}

func (s *Store) LoadState() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := map[string]string{}
	b, err := os.ReadFile(s.StatePath())
	if err == nil {
		_ = json.Unmarshal(b, &st)
	}
	return st
}

func (s *Store) SetActive(gameID, playerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := map[string]string{}
	if b, err := os.ReadFile(s.StatePath()); err == nil {
		_ = json.Unmarshal(b, &st)
	}
	if playerID == "" {
		delete(st, gameID)
	} else {
		st[gameID] = playerID
	}
	return writeJSONAtomic(s.StatePath(), st)
}

func writeJSONAtomic(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Played is when a game was last started from Lootnest, and by whom.
type Played struct {
	At     time.Time `json:"at"`
	Player string    `json:"player,omitempty"`
}

func (s *Store) RecentPath() string { return filepath.Join(s.Dir, "recent.json") }

// LoadRecent returns when each game was last played, by Library key
// ("game:<id>" for games with profiles, the scan key for the others).
func (s *Store) LoadRecent() map[string]Played {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]Played{}
	if b, err := os.ReadFile(s.RecentPath()); err == nil {
		_ = json.Unmarshal(b, &out)
	}
	return out
}

// RecordPlay notes that a game was just started. Only the 200 most recent
// games are kept.
func (s *Store) RecordPlay(key, playerID string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := map[string]Played{}
	if b, err := os.ReadFile(s.RecentPath()); err == nil {
		_ = json.Unmarshal(b, &rec)
	}
	rec[key] = Played{At: at, Player: playerID}
	if len(rec) > 200 {
		oldest := ""
		for k, p := range rec {
			if oldest == "" || p.At.Before(rec[oldest].At) {
				oldest = k
			}
		}
		delete(rec, oldest)
	}
	return writeJSONAtomic(s.RecentPath(), rec)
}

func (s *Store) PlaytimePath() string { return filepath.Join(s.Dir, "playtime.json") }

// LoadPlaytime returns how long each player has played each game through
// Lootnest, in seconds, by Library key and player id.
func (s *Store) LoadPlaytime() map[string]map[string]int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]map[string]int64{}
	if b, err := os.ReadFile(s.PlaytimePath()); err == nil {
		_ = json.Unmarshal(b, &out)
	}
	return out
}

// AddPlaytime adds a finished session's length to a player's time in a game.
// Sessions under a minute (a crash at start) and over a day (a clock jump)
// aren't counted.
func (s *Store) AddPlaytime(key, playerID string, d time.Duration) error {
	if playerID == "" || d < time.Minute || d > 24*time.Hour {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	pt := map[string]map[string]int64{}
	if b, err := os.ReadFile(s.PlaytimePath()); err == nil {
		_ = json.Unmarshal(b, &pt)
	}
	if pt[key] == nil {
		pt[key] = map[string]int64{}
	}
	pt[key][playerID] += int64(d / time.Second)
	return writeJSONAtomic(s.PlaytimePath(), pt)
}
