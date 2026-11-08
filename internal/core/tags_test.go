package core

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
)

func tagConfig(t *testing.T) *Config {
	t.Helper()
	c := DefaultConfig()
	// As a config from before TagOrder: collections only on games.
	c.Tags = map[string][]string{"game:cs2": {"Shooters", "Co-op"}, "steam:1": {"Co-op"}, "steam:2": {"favourites"}}
	c.normalize()
	return c
}

func TestCollectionsOrder(t *testing.T) {
	c := tagConfig(t)
	if got := c.Collections(); !slices.Equal(got, []string{"Favourites", "Co-op", "Shooters"}) {
		t.Fatalf("old config: %q", got)
	}
	if c.Tags["steam:2"][0] != FavouritesTag {
		t.Fatalf("favourites in another case: %q", c.Tags["steam:2"])
	}
	if err := c.MoveTag("shooters", 0); err != nil {
		t.Fatal(err)
	}
	if got := c.Collections(); !slices.Equal(got, []string{"Favourites", "Shooters", "Co-op"}) {
		t.Fatalf("moved: %q", got)
	}
	if err := c.MoveTag("Shooters", 99); err != nil || !slices.Equal(c.TagOrder, []string{"Co-op", "Shooters"}) {
		t.Fatalf("clamped: %v %q", err, c.TagOrder)
	}
	if c.MoveTag(FavouritesTag, 1) == nil || c.MoveTag("Nope", 0) == nil {
		t.Fatal("Favourites stays first; unknown refused")
	}
	// The order survives a save and load.
	b, _ := json.Marshal(c)
	var back Config
	if err := json.Unmarshal(b, &back); err != nil || !slices.Equal(back.Collections(), c.Collections()) {
		t.Fatalf("%v %q", err, back.Collections())
	}
}

func TestAddRemoveTag(t *testing.T) {
	c := tagConfig(t)
	if err := c.AddToTag(nil, "  Weekend  "); err != nil || !slices.Contains(c.Collections(), "Weekend") {
		t.Fatalf("empty collection: %v %q", err, c.Collections())
	}
	if err := c.AddToTag([]string{"steam:1", "steam:2"}, "co-OP"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(c.Tags["steam:1"], []string{"Co-op"}) || !slices.Equal(c.Tags["steam:2"], []string{"Favourites", "Co-op"}) {
		t.Fatalf("joins the existing one once, others kept: %q", c.Tags)
	}
	if c.AddToTag([]string{"steam:1"}, "   ") == nil || c.AddToTag([]string{"steam:1"}, strings.Repeat("x", MaxTagName+1)) == nil {
		t.Fatal("empty and long names refused")
	}
	if c.AddToTag([]string{"steam:1"}, strings.Repeat("é", MaxTagName)) != nil {
		t.Fatal("40 characters fit")
	}
	if c.AddToTag([]string{""}, "Co-op") == nil {
		t.Fatal("empty key refused")
	}
	if err := c.RemoveFromTag([]string{"steam:1", "game:cs2"}, "Co-op"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(c.Tags["game:cs2"], []string{"Shooters"}) || slices.Contains(c.Tags["steam:1"], "Co-op") {
		t.Fatalf("removed: %q", c.Tags)
	}
	if err := c.RemoveFromTag([]string{"steam:2"}, "Co-op"); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(c.Collections(), "Co-op") {
		t.Fatal("a collection with no games left stays")
	}
	if c.RemoveFromTag([]string{"steam:1"}, "Nope") == nil {
		t.Fatal("unknown refused")
	}
	full := DefaultConfig()
	for i := 0; i < MaxTagsPerGame; i++ {
		if err := full.AddToTag([]string{"k"}, string(rune('a'+i))); err != nil {
			t.Fatal(err)
		}
	}
	if full.AddToTag([]string{"k"}, "one more") == nil {
		t.Fatal("per game limit")
	}
}

func TestRenameTag(t *testing.T) {
	c := tagConfig(t)
	if err := c.RenameTag("co-op", " Together "); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(c.Tags["game:cs2"], []string{"Shooters", "Together"}) || !slices.Equal(c.TagOrder, []string{"Together", "Shooters"}) {
		t.Fatalf("renamed in place: %q %q", c.Tags, c.TagOrder)
	}
	if err := c.RenameTag("Together", "together"); err != nil || c.TagOrder[0] != "together" {
		t.Fatalf("case only: %v %q", err, c.TagOrder)
	}
	// Onto another name: one collection, each game in it once.
	if err := c.RenameTag("Shooters", "TOGETHER"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(c.Tags["game:cs2"], []string{"together"}) || !slices.Equal(c.Tags["steam:1"], []string{"together"}) || !slices.Equal(c.TagOrder, []string{"together"}) {
		t.Fatalf("merged: %q %q", c.Tags, c.TagOrder)
	}
	if c.RenameTag(FavouritesTag, "Best") == nil || c.RenameTag("together", "") == nil || c.RenameTag("Nope", "x") == nil {
		t.Fatal("Favourites, empty and unknown refused")
	}
}

func TestDeleteTag(t *testing.T) {
	c := tagConfig(t)
	c.AddedGames = []FoundGame{{Key: "steam:1", Name: "One"}}
	if err := c.DeleteTag("CO-OP"); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(c.Collections(), "Co-op") || !slices.Equal(c.Tags["game:cs2"], []string{"Shooters"}) {
		t.Fatalf("label gone: %q %q", c.Collections(), c.Tags)
	}
	if _, ok := c.Tags["steam:1"]; ok {
		t.Fatal("a game in no collection has no entry")
	}
	if len(c.AddedGames) != 1 {
		t.Fatal("games stay in the Library")
	}
	if c.DeleteTag(FavouritesTag) == nil || c.DeleteTag("Nope") == nil {
		t.Fatal("Favourites and unknown refused")
	}
}

func TestSetTagsKnowsEmptyCollections(t *testing.T) {
	c := DefaultConfig()
	if err := c.AddToTag(nil, "Weekend"); err != nil {
		t.Fatal(err)
	}
	if err := c.SetTags("steam:1", []string{"weekend"}); err != nil || c.Tags["steam:1"][0] != "Weekend" {
		t.Fatalf("%v %q", err, c.Tags)
	}
}

func TestSetTagsFavouritesAndLimit(t *testing.T) {
	c := DefaultConfig()
	if err := c.SetTags("steam:1", []string{"favourites"}); err != nil || c.Tags["steam:1"][0] != FavouritesTag {
		t.Fatalf("%v %q", err, c.Tags)
	}
	for i := 0; len(c.Collections()) < MaxTags; i++ {
		if err := c.AddToTag(nil, fmt.Sprint("C", i)); err != nil {
			t.Fatal(err)
		}
	}
	if c.AddToTag(nil, "One more") == nil || c.SetTags("steam:2", []string{"One more"}) == nil {
		t.Fatal("a collection past the limit")
	}
	if err := c.SetTags("steam:1", nil); err != nil {
		t.Fatal("taking a game out refused:", err)
	}
}
