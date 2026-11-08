package core

import (
	"sort"
	"time"
)

// SaveCopy is one copy Lootnest keeps of a game's saves, for the Saves page.
type SaveCopy struct {
	Key   string    `json:"key"`  // the game's Library key
	Kind  string    `json:"kind"` // "net" (safety net), "point" (save point) or "swap" (before a swap)
	ID    string    `json:"id"`
	At    time.Time `json:"at"`
	Size  int64     `json:"size"`
	Name  string    `json:"name,omitempty"`  // a save point's name
	Owner string    `json:"owner,omitempty"` // the profile whose files they are
}

// SavesGame is what Lootnest keeps of one game's saves.
type SavesGame struct {
	Key    string    `json:"key"`
	Last   time.Time `json:"last"`
	Copies int       `json:"copies"` // every kind, points included
	Points int       `json:"points"`
	Size   int64     `json:"size"`
}

// SavesOverview sorts copies newest first and sums them per game, the game
// with the newest copy first. recent is how many copies it returns.
func SavesOverview(copies []SaveCopy, recent int) ([]SaveCopy, []SavesGame) {
	sorted := make([]SaveCopy, len(copies))
	copy(sorted, copies)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].At.After(sorted[j].At) })
	at := map[string]int{}
	games := []SavesGame{}
	for _, c := range sorted {
		i, ok := at[c.Key]
		if !ok {
			i = len(games)
			at[c.Key] = i
			games = append(games, SavesGame{Key: c.Key, Last: c.At})
		}
		g := &games[i]
		g.Copies++
		g.Size += c.Size
		if c.Kind == "point" {
			g.Points++
		}
	}
	return sorted[:min(recent, len(sorted))], games
}
