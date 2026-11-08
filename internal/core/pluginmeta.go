package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// What a plugin knows about games: save locations in the Ludusavi manifest's
// format (saves/*.yaml), only ever suggested, and game details
// (metadata/*.json), under Steam's and the user's own.
// ---------------------------------------------------------------------------

// PluginSaves reads a plugin's save locations into one list.
func PluginSaves(dir string) (*SaveDB, error) {
	files := pluginFiles(dir, "saves", ".yaml", ".yml")
	if len(files) == 0 {
		return nil, errors.New("it provides saves but has no saves/*.yaml")
	}
	db := &SaveDB{}
	for _, f := range files {
		r, err := os.Open(f)
		if err != nil {
			return nil, err
		}
		one, err := ParseSaveDB(r)
		r.Close()
		if err != nil {
			return nil, fmt.Errorf("saves/%s isn't in the Ludusavi manifest's format: %s", filepath.Base(f), err)
		}
		db.Games = append(db.Games, one.Games...)
	}
	db.index()
	return db, nil
}

// PluginSaveDB is one plugin's save locations, for SuggestPaths.
type PluginSaveDB struct {
	Name string
	DB   *SaveDB
}

// PluginDetails is a game's details as a plugin gives them.
type PluginDetails struct {
	Summary   string   `json:"summary"`
	Genres    []string `json:"genres"`
	Developer string   `json:"developer"`
	Publisher string   `json:"publisher"`
	Released  string   `json:"released"`
}

// PluginMeta is a plugin's details, by Steam app id ("1145360") or by the
// game's name as norm makes it ("hades").
type PluginMeta map[string]PluginDetails

// PluginMetadata reads a plugin's metadata/*.json.
func PluginMetadata(dir string) (PluginMeta, error) {
	files := pluginFiles(dir, "metadata", ".json")
	if len(files) == 0 {
		return nil, errors.New("it provides metadata but has no metadata/*.json")
	}
	out := PluginMeta{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		var m map[string]PluginDetails
		if err := json.Unmarshal(b, &m); err != nil {
			return nil, fmt.Errorf("metadata/%s can't be read: %s", filepath.Base(f), shortErr(err))
		}
		for k, d := range m {
			if _, err := strconv.Atoi(k); err != nil {
				k = norm(k)
			}
			if k != "" {
				out[k] = d
			}
		}
	}
	return out, nil
}

// Find is a game's details by Steam app id, then by name, or nil.
func (m PluginMeta) Find(appID int, names ...string) *GameDetails {
	var d PluginDetails
	ok := false
	if appID > 0 {
		d, ok = m[strconv.Itoa(appID)]
	}
	for _, n := range names {
		if k := norm(n); !ok && k != "" {
			d, ok = m[k]
		}
	}
	if !ok {
		return nil
	}
	text := func(s string, n int) string {
		s = strings.Join(strings.Fields(s), " ")
		if r := []rune(s); len(r) > n {
			s = strings.TrimSpace(string(r[:n]))
		}
		return s
	}
	one := func(s string) []string {
		if s = text(s, 200); s == "" {
			return nil
		}
		return []string{s}
	}
	return &GameDetails{Summary: text(d.Summary, 2000), Genres: cleanNames(d.Genres), Developers: one(d.Developer),
		Publishers: one(d.Publisher), Released: text(d.Released, 60)}
}

// LayerDetails fills what Steam's details (over, may be nil) leave empty
// from a plugin's (under), naming the plugin in From.
func LayerDetails(under *GameDetails, from string, over *GameDetails) *GameDetails {
	if under == nil {
		return over
	}
	var d GameDetails
	if over != nil {
		d = *over
	}
	used := false
	fill := func(dst *string, v string) {
		if *dst == "" && v != "" {
			*dst, used = v, true
		}
	}
	list := func(dst *[]string, v []string) {
		if len(*dst) == 0 && len(v) > 0 {
			*dst, used = v, true
		}
	}
	fill(&d.Summary, under.Summary)
	fill(&d.Released, under.Released)
	list(&d.Genres, under.Genres)
	list(&d.Developers, under.Developers)
	list(&d.Publishers, under.Publishers)
	if used {
		d.From = from
	}
	return &d
}
