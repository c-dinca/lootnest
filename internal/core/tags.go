package core

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
)

// FavouritesTag is the collection every Library has: always first, never
// renamed or deleted.
const FavouritesTag = "Favourites"

// MaxTagName is the longest collection name, in characters.
const MaxTagName = 40

func cleanTagName(t string) (string, error) {
	t = strings.Join(strings.Fields(t), " ")
	if len([]rune(t)) > MaxTagName {
		return "", fmt.Errorf("Keep collection names to %d characters (%q is longer).", MaxTagName, t)
	}
	return t, nil
}

// Collections lists every collection in the user's order: Favourites, then
// TagOrder, then those only on games (from configs before TagOrder), by name.
func (c *Config) Collections() []string {
	out := []string{FavouritesTag}
	seen := map[string]bool{strings.ToLower(FavouritesTag): true}
	add := func(t string) {
		if t != "" && !seen[strings.ToLower(t)] {
			seen[strings.ToLower(t)] = true
			out = append(out, t)
		}
	}
	for _, t := range c.TagOrder {
		add(t)
	}
	var rest []string
	for _, ts := range c.Tags {
		rest = append(rest, ts...)
	}
	sort.Slice(rest, func(i, j int) bool { return strings.ToLower(rest[i]) < strings.ToLower(rest[j]) })
	for _, t := range rest {
		add(t)
	}
	return out
}

// tagNamed is the existing collection with this name in any case, or "".
func (c *Config) tagNamed(name string) string {
	for _, t := range c.Collections() {
		if strings.EqualFold(t, name) {
			return t
		}
	}
	return ""
}

// fixTagOrder writes the order down, so a collection whose last game
// leaves it stays until it's deleted.
func (c *Config) fixTagOrder() { c.TagOrder = c.Collections()[1:] }

func checkTagKeys(keys []string) error {
	for _, k := range keys {
		if k == "" || len(k) > 400 || strings.ContainsAny(k, "\r\n\x00") {
			return errors.New("invalid game")
		}
	}
	return nil
}

// AddToTag puts these games (by Library key) in a collection, making it if
// it's new; no keys only makes it. Their other collections stay.
func (c *Config) AddToTag(keys []string, tag string) error {
	if err := checkTagKeys(keys); err != nil {
		return err
	}
	name, err := cleanTagName(tag)
	if err != nil {
		return err
	}
	if name == "" {
		return errors.New("Give the collection a name.")
	}
	c.fixTagOrder()
	if have := c.tagNamed(name); have != "" {
		name = have
	} else {
		if len(c.Collections())+1 > MaxTags {
			return fmt.Errorf("Lootnest keeps up to %d collections.", MaxTags)
		}
		c.TagOrder = append(c.TagOrder, name)
	}
	for _, k := range keys {
		if slices.Contains(c.Tags[k], name) {
			continue
		}
		if len(c.Tags[k]) >= MaxTagsPerGame {
			return fmt.Errorf("A game can be in up to %d collections.", MaxTagsPerGame)
		}
		if c.Tags == nil {
			c.Tags = map[string][]string{}
		}
		c.Tags[k] = append(c.Tags[k], name)
	}
	return nil
}

// RemoveFromTag takes these games out of a collection. The games stay in
// the Library and the collection stays, empty or not.
func (c *Config) RemoveFromTag(keys []string, tag string) error {
	name := c.tagNamed(tag)
	if name == "" {
		return errors.New("That collection doesn't exist.")
	}
	c.fixTagOrder()
	for _, k := range keys {
		c.dropTag(k, name)
	}
	return nil
}

func (c *Config) dropTag(key, name string) {
	ts := []string{}
	for _, t := range c.Tags[key] {
		if t != name {
			ts = append(ts, t)
		}
	}
	if len(ts) == 0 {
		delete(c.Tags, key)
	} else {
		c.Tags[key] = ts
	}
}

// RenameTag renames a collection. Onto another collection's name, it
// merges the two: each game is in that one once.
func (c *Config) RenameTag(from, to string) error {
	old := c.tagNamed(from)
	if old == "" {
		return errors.New("That collection doesn't exist.")
	}
	if old == FavouritesTag {
		return errors.New("Favourites can't be renamed.")
	}
	name, err := cleanTagName(to)
	if err != nil {
		return err
	}
	if name == "" {
		return errors.New("Give the collection a name.")
	}
	c.fixTagOrder()
	merge := false
	if m := c.tagNamed(name); m != "" && m != old {
		name, merge = m, true
	}
	i := slices.Index(c.TagOrder, old)
	if merge {
		c.TagOrder = slices.Delete(c.TagOrder, i, i+1)
	} else {
		c.TagOrder[i] = name
	}
	for k, ts := range c.Tags {
		if j := slices.Index(ts, old); j >= 0 {
			if slices.Contains(ts, name) {
				c.dropTag(k, old)
			} else {
				ts[j] = name
			}
		}
	}
	c.retagViews(old, name)
	return nil
}

// DeleteTag removes a collection: the label goes from every game, and the
// games stay in the Library.
func (c *Config) DeleteTag(tag string) error {
	name := c.tagNamed(tag)
	if name == "" {
		return errors.New("That collection doesn't exist.")
	}
	if name == FavouritesTag {
		return errors.New("Favourites can't be deleted.")
	}
	c.fixTagOrder()
	for k := range c.Tags {
		c.dropTag(k, name)
	}
	c.TagOrder = slices.DeleteFunc(c.TagOrder, func(t string) bool { return t == name })
	c.retagViews(name, "")
	return nil
}

// MoveTag puts a collection at this place in the order after Favourites
// (0 is right after it), clamped to the ends.
func (c *Config) MoveTag(tag string, to int) error {
	name := c.tagNamed(tag)
	if name == "" {
		return errors.New("That collection doesn't exist.")
	}
	if name == FavouritesTag {
		return errors.New("Favourites stays first.")
	}
	c.fixTagOrder()
	rest := slices.DeleteFunc(c.TagOrder, func(t string) bool { return t == name })
	c.TagOrder = slices.Insert(rest, max(0, min(to, len(rest))), name)
	return nil
}
