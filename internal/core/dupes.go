package core

import (
	"strconv"
	"strings"
	"unicode"
)

// ---------------------------------------------------------------------------
// One game from more than one store: the same game installed from Steam and
// Epic, or GOG and a folder, shows once in the Library. Grouping is only how
// the Library shows them; every copy keeps its own Library key, profiles,
// saves and playtime, and nothing on disk changes.
// ---------------------------------------------------------------------------

// CopyGroup is the copies of one game across stores.
type CopyGroup struct {
	Key     string   `json:"key"`     // "steam:<id>" or "name:<normalised name>"
	Keys    []string `json:"keys"`    // the copies' Library keys, in the order found
	Primary string   `json:"primary"` // the copy Start with uses
}

// sourceFamily: folders, shortcuts and games added by hand are one source,
// so two copies in folders stay apart (that's a new copy, core/newcopy.go).
func sourceFamily(src string) string {
	switch src {
	case "folder", "shortcut", "added", "":
		return "folder"
	}
	return src
}

func steamIDOf(f FoundGame) int {
	if f.Source == "steam" && f.AppID > 0 {
		return f.AppID
	}
	return f.SteamID
}

// A trailing "Game of the Year Edition" (or GOTY) is the same game; other
// editions, remasters and sequels are not.
var gotySuffixes = [][]string{{"game", "of", "the", "year", "edition"}, {"game", "of", "the", "year"}, {"goty", "edition"}, {"goty"}}

// CopyName is a game's name as copies are matched: lowercase letters and
// digits only (no ™ ® © or punctuation), without a trailing Game of the Year.
func CopyName(name string) string {
	words := strings.FieldsFunc(strings.ToLower(name), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	for _, s := range gotySuffixes {
		n := len(words) - len(s)
		if n > 0 && strings.Join(words[n:], " ") == strings.Join(s, " ") {
			words = words[:n]
			break
		}
	}
	return strings.Join(words, "")
}

// GroupCopies groups the found games that are the same game from different
// sources: by Steam app id when both copies know one, else by name. A group
// with two copies from one source, or two Steam ids, is left apart, as are
// the groups the user split (separate). Primary is the copy the user chose,
// else the one played most recently, else the Steam copy, else the first.
func GroupCopies(found []FoundGame, separate []string, primary map[string]string, recent map[string]Played) []CopyGroup {
	parent := make([]int, len(found))
	for i := range parent {
		parent[i] = i
	}
	var root func(int) int
	root = func(i int) int {
		if parent[i] != i {
			parent[i] = root(parent[i])
		}
		return parent[i]
	}
	names := make([]string, len(found))
	byName := map[string][]int{}
	byID := map[int][]int{}
	for i, f := range found {
		if id := steamIDOf(f); id > 0 {
			byID[id] = append(byID[id], i)
		}
		if names[i] = CopyName(f.Name); names[i] != "" {
			byName[names[i]] = append(byName[names[i]], i)
		}
	}
	join := func(idx []int, ok func(a, b FoundGame) bool) {
		for _, a := range idx {
			for _, b := range idx {
				if a < b && ok(found[a], found[b]) {
					parent[root(b)] = root(a)
				}
			}
		}
	}
	for _, idx := range byID {
		join(idx, func(a, b FoundGame) bool { return true })
	}
	for _, idx := range byName {
		join(idx, func(a, b FoundGame) bool { return steamIDOf(a) == 0 || steamIDOf(b) == 0 })
	}

	members := map[int][]int{}
	var order []int
	for i := range found {
		r := root(i)
		if members[r] == nil {
			order = append(order, r)
		}
		members[r] = append(members[r], i)
	}
	split := map[string]bool{}
	for _, k := range separate {
		split[k] = true
	}
	var out []CopyGroup
	for _, r := range order {
		idx := members[r]
		if len(idx) < 2 {
			continue
		}
		g := CopyGroup{Key: "name:" + names[idx[0]]}
		sources, id, clash := map[string]bool{}, 0, false
		for _, i := range idx {
			f := found[i]
			fam := sourceFamily(f.Source)
			if sid := steamIDOf(f); sid > 0 {
				clash = clash || (id > 0 && sid != id)
				id = sid
			}
			clash = clash || sources[fam]
			sources[fam] = true
			g.Keys = append(g.Keys, f.Key)
		}
		if id > 0 {
			g.Key = "steam:" + strconv.Itoa(id)
		}
		if clash || split[g.Key] {
			continue
		}
		g.Primary = primaryCopy(g, found, idx, primary[g.Key], recent)
		out = append(out, g)
	}
	return out
}

func primaryCopy(g CopyGroup, found []FoundGame, idx []int, chosen string, recent map[string]Played) string {
	for _, k := range g.Keys {
		if k == chosen {
			return k
		}
	}
	best := ""
	for _, k := range g.Keys {
		if at := recent[k].At; !at.IsZero() && (best == "" || at.After(recent[best].At)) {
			best = k
		}
	}
	if best != "" {
		return best
	}
	for _, i := range idx {
		if found[i].Source == "steam" {
			return found[i].Key
		}
	}
	return g.Keys[0]
}
