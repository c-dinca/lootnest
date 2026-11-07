package core

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// One player's profile as a file, to take to another PC or a friend's: their
// saves and settings in every game they have a profile for, their picture
// and banner, shelf, play time and activity, and those games' definitions.
// It's a backup file (archive.go) with Kind "player", optionally encrypted
// the same way; ImportPlayer puts it in place next to the players already
// there.
// ---------------------------------------------------------------------------

// ExportPlayer writes player pid's profile to w. live are the save files in
// place now; only the ones that are pid's go in.
func (s *Store) ExportPlayer(w io.Writer, cfg *Config, pid string, live []LiveSource, m BackupManifest) error {
	p := cfg.Player(pid)
	if p == nil {
		return errors.New("That profile doesn't exist.")
	}
	theirs := map[string]bool{}
	for _, g := range cfg.Games {
		if exists(filepath.Join(s.ProfileDir(pid, g.ID), initMarker)) {
			theirs[g.ID] = true
		}
	}
	var mine []LiveSource
	for _, l := range live {
		if l.Owner == pid && cfg.Game(l.GameID) != nil {
			theirs[l.GameID] = true
			mine = append(mine, l)
		}
	}
	out := &Config{Version: cfg.Version, Players: []Player{*p}}
	for _, g := range cfg.Games {
		if theirs[g.ID] {
			out.Games = append(out.Games, g)
		}
	}
	zw := zip.NewWriter(w)
	add := func(name string, v any) error {
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		f, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = f.Write(b)
		return err
	}
	if err := add("config.json", out); err != nil {
		return err
	}
	for _, g := range out.Games {
		if err := zipPath(zw, s.ProfileDir(pid, g.ID), "profiles/"+pid+"/"+g.ID); err != nil {
			return err
		}
	}
	m.Format, m.Kind, m.Live, m.Players, m.Games = backupFormat, "player", nil, []string{p.Name}, len(out.Games)
	for _, l := range mine {
		src := l.Path
		if real, err := filepath.EvalSymlinks(src); err == nil {
			src = real
		}
		if !exists(src) || !idRe.MatchString(l.GameID) || !idRe.MatchString(l.EntryID) {
			continue
		}
		if err := zipPath(zw, src, "live/"+l.GameID+"/"+l.EntryID); err != nil {
			return err
		}
		m.Live = append(m.Live, l.LiveEntry)
	}
	if err := zipPath(zw, filepath.Join(s.Dir, "avatars", pid+".png"), "avatars/"+pid+".png"); err != nil {
		return err
	}
	if err := zipPath(zw, filepath.Join(s.Dir, "banners", pid+".jpg"), "banners/"+pid+".jpg"); err != nil {
		return err
	}
	if sh := s.LoadShelf()[pid]; len(sh) > 0 {
		if err := add("shelf.json", map[string]map[string]ShelfEntry{pid: sh}); err != nil {
			return err
		}
	}
	act := s.LoadActivity()
	if err := add("activity.json", Activity{
		Days:     map[string]map[string]map[string]int64{pid: act.Days[pid]},
		Longest:  map[string]LongSession{pid: act.Longest[pid]},
		Sessions: map[string]int{pid: act.Sessions[pid]},
	}); err != nil {
		return err
	}
	pt := map[string]map[string]int64{}
	for k, by := range s.LoadPlaytime() {
		if t := by[pid]; t > 0 {
			pt[k] = map[string]int64{pid: t}
		}
	}
	if err := add("playtime.json", pt); err != nil {
		return err
	}
	if err := add(manifestName, m); err != nil {
		return err
	}
	return zw.Close()
}

// WritePlayerFile writes player pid's profile to path (see ExportPlayer).
func (s *Store) WritePlayerFile(path string, cfg *Config, pid string, live []LiveSource, m BackupManifest, opt BackupOptions) (int64, error) {
	return writeArchiveFile(path, func(w io.Writer) error { return s.ExportPlayer(w, cfg, pid, live, m) }, opt)
}

// PlayerFileName is the name of an exported profile.
func PlayerFileName(name string, t time.Time, encrypted bool) string {
	ext := ".zip"
	if encrypted {
		ext = ".lnbak"
	}
	return "Lootnest profile " + safeFileName(name) + " " + t.Format("2006-01-02") + ext
}

// ImportResult says what ImportPlayer did.
type ImportResult struct {
	PlayerID string `json:"playerId"`
	Name     string `json:"name"`
	Replaced bool   `json:"replaced"` // the player was already here
	Games    int    `json:"games"`    // games with their saves now
	Added    int    `json:"added"`    // of which new to this PC
	Skipped  int    `json:"skipped"`  // save locations that don't match this PC's game
}

// ImportPlayer puts an exported profile in place. The player keeps their id:
// one already here is updated (their saves replaced by the file's); one
// that isn't is added. Games are matched to the ones here (same id, Steam
// game, launch link, exe or name) or added. The caller has backed up
// everything first.
func (s *Store) ImportPlayer(ob *OpenedBackup, current *Config) (*Config, ImportResult, error) {
	var res ImportResult
	if ob.Manifest.Kind != "player" {
		return nil, res, errors.New("This is a backup of everything, not one profile; use Restore from a backup in Settings.")
	}
	in, err := ob.Config()
	if err != nil {
		return nil, res, err
	}
	if len(in.Players) != 1 {
		return nil, res, errors.New("This profile file is damaged.")
	}
	cfg := current.Clone()
	p := in.Players[0]
	old := p.ID
	if local := cfg.Player(p.ID); local != nil {
		res.Replaced = true
	} else if len(cfg.Players) >= 8 {
		return nil, res, errors.New("Lootnest supports up to 8 profiles; remove one first.")
	}
	// A different player here with the same name: tell them apart.
	for n := 2; nameTaken(cfg, p.Name, p.ID); n++ {
		p.Name = fmt.Sprintf("%s (%d)", in.Players[0].Name, n)
	}
	res.PlayerID, res.Name = p.ID, p.Name

	// Games: the file's game id -> the game here, and its save locations.
	gameMap := map[string]string{}
	pathMap := map[string]map[string]string{} // file's game id -> file's path id -> path id here
	for _, eg := range in.Games {
		g := matchGame(cfg, eg)
		if g == nil {
			ng := eg
			if cfg.Game(ng.ID) != nil {
				ng.ID = NewID("g")
			}
			cfg.Games = append(cfg.Games, ng)
			g = &cfg.Games[len(cfg.Games)-1]
			res.Added++
		}
		gameMap[eg.ID] = g.ID
		pm := map[string]string{}
		for _, ep := range eg.Paths {
			for _, lp := range g.Paths {
				if lp.ID == ep.ID || strings.EqualFold(lp.Path, ep.Path) {
					pm[ep.ID] = lp.ID
					break
				}
			}
			if pm[ep.ID] == "" {
				res.Skipped++
			}
		}
		pathMap[eg.ID] = pm
	}

	// Saves: their profile in each game, then the files that were in place.
	for eg, lg := range gameMap {
		dir := s.ProfileDir(p.ID, lg)
		src := filepath.Join(ob.Dir, "profiles", old, eg)
		copied := false
		for ep, lp := range pathMap[eg] {
			from := filepath.Join(src, ep)
			if !exists(from) {
				continue
			}
			if err := Mirror(from, filepath.Join(dir, lp)); err != nil {
				return nil, res, err
			}
			copied = true
		}
		for _, l := range ob.Manifest.Live {
			if l.Owner != old || l.GameID != eg || pathMap[eg][l.EntryID] == "" {
				continue
			}
			from := filepath.Join(ob.Dir, "live", eg, l.EntryID)
			if !exists(from) {
				continue
			}
			if err := Mirror(from, filepath.Join(dir, pathMap[eg][l.EntryID])); err != nil {
				return nil, res, err
			}
			copied = true
		}
		if copied || exists(filepath.Join(src, initMarker)) {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, res, err
			}
			_ = os.WriteFile(filepath.Join(dir, initMarker), []byte(ob.Manifest.Created.Format(time.RFC3339)), 0o644)
			res.Games++
		}
		// If their files are in place here, they're older than the file's:
		// the next swap asks whose they are instead of saving them over.
		if s.LoadState()[lg] == p.ID {
			if err := s.SetActive(lg, ""); err != nil {
				return nil, res, err
			}
		}
	}

	// Library keys in their profile point at the games here.
	remap := func(k string) string {
		if id, ok := strings.CutPrefix(k, "game:"); ok && gameMap[id] != "" {
			return "game:" + gameMap[id]
		}
		return k
	}
	p.Banner, p.AvatarGame, p.Now = remapKey(p.Banner, remap), remapKey(p.AvatarGame, remap), remapKey(p.Now, remap)
	for i := range p.Favorites {
		p.Favorites[i] = remap(p.Favorites[i])
	}
	// Pictures.
	p.Avatar = false
	if b, err := os.ReadFile(filepath.Join(ob.Dir, "avatars", old+".png")); err == nil {
		if err := writeFileAtomic(filepath.Join(s.Dir, "avatars", p.ID+".png"), b); err != nil {
			return nil, res, err
		}
		p.Avatar = true
	}
	if p.Banner == "image" {
		if b, err := os.ReadFile(filepath.Join(ob.Dir, "banners", old+".jpg")); err == nil {
			if err := writeFileAtomic(filepath.Join(s.Dir, "banners", p.ID+".jpg"), b); err != nil {
				return nil, res, err
			}
		} else {
			p.Banner = ""
		}
	}
	if res.Replaced {
		*cfg.Player(p.ID) = p
	} else {
		cfg.Players = append(cfg.Players, p)
	}
	if err := ValidatePlayers(cfg.Players); err != nil {
		return nil, res, err
	}
	if err := s.importPlayerData(ob, old, p.ID, remap); err != nil {
		return nil, res, err
	}
	if err := s.SaveConfig(cfg); err != nil {
		return nil, res, err
	}
	return cfg, res, nil
}

func remapKey(k string, remap func(string) string) string {
	if k == "" || k == "image" {
		return k
	}
	return remap(k)
}

func nameTaken(cfg *Config, name, id string) bool {
	for _, p := range cfg.Players {
		if p.ID != id && strings.EqualFold(p.Name, name) {
			return true
		}
	}
	return false
}

// matchGame finds the game here that is the file's game.
func matchGame(cfg *Config, eg Game) *Game {
	same := func(g *Game) bool {
		a, b := g.Launch, eg.Launch
		switch {
		case a.Type == "steam" && b.Type == "steam":
			return a.AppID == b.AppID
		case a.URI != "" && a.URI == b.URI:
			return true
		case a.Exe != "" && strings.EqualFold(a.Exe, b.Exe):
			return true
		}
		return false
	}
	for i := range cfg.Games {
		if g := &cfg.Games[i]; g.ID == eg.ID && (same(g) || strings.EqualFold(g.Name, eg.Name)) {
			return g
		}
	}
	for i := range cfg.Games {
		if same(&cfg.Games[i]) {
			return &cfg.Games[i]
		}
	}
	for i := range cfg.Games {
		if strings.EqualFold(cfg.Games[i].Name, eg.Name) {
			return &cfg.Games[i]
		}
	}
	return nil
}

// importPlayerData puts the file's shelf, play time and activity in place
// for the player, replacing theirs here.
func (s *Store) importPlayerData(ob *OpenedBackup, old, pid string, remap func(string) string) error {
	read := func(name string, v any) bool {
		b, err := os.ReadFile(filepath.Join(ob.Dir, name))
		return err == nil && json.Unmarshal(b, v) == nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var inShelf map[string]map[string]ShelfEntry
	if read("shelf.json", &inShelf) {
		sh := s.readShelf()
		mine := map[string]ShelfEntry{}
		for k, e := range inShelf[old] {
			if e.Validate() == nil {
				mine[remap(k)] = e
			}
		}
		sh[pid] = mine
		if err := writeJSONAtomic(s.ShelfPath(), sh); err != nil {
			return err
		}
	}
	var inAct Activity
	if read("activity.json", &inAct) {
		a := s.readActivity()
		days := map[string]map[string]int64{}
		for d, by := range inAct.Days[old] {
			days[d] = map[string]int64{}
			for k, v := range by {
				if v > 0 && v <= 86400 {
					days[d][remap(k)] += v
				}
			}
		}
		a.Days[pid], a.Sessions[pid] = days, inAct.Sessions[old]
		if l := inAct.Longest[old]; l.Seconds > 0 {
			l.Key = remap(l.Key)
			a.Longest[pid] = l
		} else {
			delete(a.Longest, pid)
		}
		if err := writeJSONAtomic(s.ActivityPath(), a); err != nil {
			return err
		}
	}
	var inPT map[string]map[string]int64
	if read("playtime.json", &inPT) {
		pt := map[string]map[string]int64{}
		if b, err := os.ReadFile(s.PlaytimePath()); err == nil {
			_ = json.Unmarshal(b, &pt)
		}
		for _, by := range pt {
			delete(by, pid)
		}
		for k, by := range inPT {
			if t := by[old]; t > 0 {
				k = remap(k)
				if pt[k] == nil {
					pt[k] = map[string]int64{}
				}
				pt[k][pid] = t
			}
		}
		if err := writeJSONAtomic(s.PlaytimePath(), pt); err != nil {
			return err
		}
	}
	return nil
}
