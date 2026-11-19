package core

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Achievements marked by hand. A game that keeps no achievements on this PC
// (no Steam account, no Steam-compatible tool beside it) can still have
// them: the user turns on marking by hand for it, Lootnest asks Steam once
// for the game's list (names, descriptions, pictures, how rare), and each
// profile marks the ones it has done. The marks are Lootnest's own, in
// hand-achievements.json; nothing is written into the game or its saves.
// ---------------------------------------------------------------------------

// HandGame is one game's marking by hand.
type HandGame struct {
	AppID int `json:"appId"`
	// Off: the user stopped marking by hand. The marks are kept, so turning
	// it on again brings them back.
	Off bool `json:"off,omitempty"`
	// Marks: profile id -> achievement id -> when it was marked.
	Marks map[string]map[string]time.Time `json:"marks,omitempty"`
}

// HandAchievements is every game's, by Library key.
type HandAchievements struct {
	Games map[string]*HandGame `json:"games"`
}

func (s *Store) HandAchPath() string { return filepath.Join(s.Dir, "hand-achievements.json") }

// LoadHandAch reads hand-achievements.json; a file that isn't there reads
// as none.
func (s *Store) LoadHandAch() HandAchievements {
	s.mu.Lock()
	defer s.mu.Unlock()
	h, _ := s.readHandAch()
	return h
}

// readHandAch reads the file. One that doesn't parse is an error, so a
// change never replaces it.
func (s *Store) readHandAch() (HandAchievements, error) {
	h := HandAchievements{Games: map[string]*HandGame{}}
	b, err := os.ReadFile(s.HandAchPath())
	if errors.Is(err, os.ErrNotExist) {
		return h, nil
	}
	if err != nil {
		return h, err
	}
	if err := json.Unmarshal(b, &h); err != nil {
		return HandAchievements{Games: map[string]*HandGame{}}, err
	}
	if h.Games == nil {
		h.Games = map[string]*HandGame{}
	}
	return h, nil
}

// UpdateHandAch changes hand-achievements.json under the store's lock. A
// file that doesn't parse is kept as hand-achievements.json.broken first.
func (s *Store) UpdateHandAch(change func(*HandAchievements) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	h, err := s.readHandAch()
	if err != nil {
		var syn *json.SyntaxError
		var typ *json.UnmarshalTypeError
		if !errors.As(err, &syn) && !errors.As(err, &typ) {
			return err
		}
		if rerr := os.Rename(s.HandAchPath(), s.HandAchPath()+".broken"); rerr != nil {
			return rerr
		}
	}
	if err := change(&h); err != nil {
		return err
	}
	return writeJSONAtomic(s.HandAchPath(), h)
}

// SetHandOn turns marking by hand on or off for the game under key.
func (h *HandAchievements) SetHandOn(key string, appID int, on bool) {
	g := h.Games[key]
	if g == nil {
		if !on {
			return
		}
		g = &HandGame{}
		h.Games[key] = g
	}
	if appID > 0 {
		g.AppID = appID
	}
	g.Off = !on
}

// Mark marks (on) or unmarks an achievement for a profile.
func (h *HandAchievements) Mark(key, player, id string, on bool, at time.Time) error {
	g := h.Games[key]
	if g == nil || g.Off {
		return errors.New("This game's achievements aren't marked by hand.")
	}
	if player == "" || id == "" {
		return errors.New("Choose whose achievement this is.")
	}
	if on {
		if g.Marks == nil {
			g.Marks = map[string]map[string]time.Time{}
		}
		if g.Marks[player] == nil {
			g.Marks[player] = map[string]time.Time{}
		}
		if _, had := g.Marks[player][id]; !had {
			g.Marks[player][id] = at.UTC()
		}
		return nil
	}
	delete(g.Marks[player], id)
	if len(g.Marks[player]) == 0 {
		delete(g.Marks, player)
	}
	return nil
}

// On is the game's marking by hand when it's on, else nil.
func (h HandAchievements) On(key string) *HandGame {
	if g := h.Games[key]; g != nil && !g.Off {
		return g
	}
	return nil
}

// Rekey moves a game's marks to its new Library key (a new copy, a move).
// Marks already under the new key stay; the older mark of the same
// achievement wins.
func (h *HandAchievements) Rekey(from, to string) bool {
	o := h.Games[from]
	if o == nil || from == to {
		return false
	}
	n := h.Games[to]
	if n == nil {
		h.Games[to] = o
	} else {
		if n.AppID == 0 {
			n.AppID = o.AppID
		}
		n.Off = n.Off && o.Off
		for p, ms := range o.Marks {
			if n.Marks == nil {
				n.Marks = map[string]map[string]time.Time{}
			}
			if n.Marks[p] == nil {
				n.Marks[p] = map[string]time.Time{}
			}
			for id, at := range ms {
				if had, ok := n.Marks[p][id]; !ok || at.Before(had) {
					n.Marks[p][id] = at
				}
			}
		}
	}
	delete(h.Games, from)
	return true
}

// ParseGameAchievements reads Steam's public IPlayerService/
// GetGameAchievements answer (no key needed): every achievement's id,
// name, description, hidden flag, picture and the share of players who
// have it, in the game's own order.
func ParseGameAchievements(b []byte, appID int) []Achievement {
	var v struct {
		Response struct {
			Achievements []struct {
				ID      string `json:"internal_name"`
				Name    string `json:"localized_name"`
				Desc    string `json:"localized_desc"`
				Icon    string `json:"icon"`
				Hidden  any    `json:"hidden"`
				Percent any    `json:"player_percent_unlocked"`
			} `json:"achievements"`
		} `json:"response"`
	}
	if json.Unmarshal(b, &v) != nil {
		return nil
	}
	var out []Achievement
	seen := map[string]bool{}
	for _, a := range v.Response.Achievements {
		id := strings.TrimSpace(a.ID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		x := Achievement{ID: id, Name: strings.TrimSpace(a.Name), Description: strings.TrimSpace(a.Desc), Hidden: truthy(a.Hidden), Icon: steamAchIcon(a.Icon, appID)}
		switch p := a.Percent.(type) {
		case string:
			x.Percent, _ = strconv.ParseFloat(strings.TrimSpace(p), 64)
		case float64:
			x.Percent = p
		}
		if x.Percent < 0 || x.Percent > 100 {
			x.Percent = 0
		}
		out = append(out, x)
	}
	return out
}

// HandSet is a game's achievements marked by hand: list is the game's
// whole list (in its own order), players the profiles that can mark them,
// each with their own set. The top set is the first profile's, so a page
// that shows one set shows someone's.
func HandSet(list []Achievement, appID int, players []string, g *HandGame) *AchievementSet {
	if len(list) == 0 || len(players) == 0 {
		return nil
	}
	set := &AchievementSet{AppID: appID, Source: "hand", Players: map[string]*AchievementSet{}}
	for _, pid := range players {
		ms := map[string]time.Time{}
		if g != nil {
			for id, at := range g.Marks[pid] {
				ms[id] = at
			}
		}
		own := make([]Achievement, len(list))
		for i, a := range list {
			a.Seq = i + 1 // the page keeps the game's order while marking
			own[i] = a
		}
		ps := &AchievementSet{AppID: appID, Source: "hand"}
		ps.List, ps.Total, ps.Unlocked = MergeAchievements(own, len(list), ms)
		set.Players[pid] = ps
	}
	first := set.Players[players[0]]
	set.List, set.Total, set.Unlocked = first.List, first.Total, first.Unlocked
	return set
}
