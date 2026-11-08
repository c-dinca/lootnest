package core

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSaveViews(t *testing.T) {
	c := DefaultConfig()
	v := LibraryView{Name: "  Co-op   tonight ", Show: "all", Store: "steam", Collection: "Co-op", Status: "next", Profile: "p1", Sort: "recent", Density: "list"}
	if err := c.SaveView(v, ""); err != nil {
		t.Fatal(err)
	}
	if got := c.Views[0]; got.Name != "Co-op tonight" || got.Store != "steam" || got.Density != "list" {
		t.Fatalf("cleaned: %+v", got)
	}
	// A new view can't take a name already used, in any case.
	if err := c.SaveView(LibraryView{Name: "co-op TONIGHT", Sort: "name"}, ""); err == nil || len(c.Views) != 1 || c.Views[0].Sort != "recent" {
		t.Fatalf("same name: %v %+v", err, c.Views)
	}
	if err := c.SaveView(LibraryView{Name: "Short ones", Sort: "played"}, ""); err != nil {
		t.Fatal(err)
	}
	// A rename keeps its place; onto another view's name it's refused.
	if err := c.SaveView(LibraryView{Name: "Couch", Sort: "name"}, "Co-op tonight"); err != nil || c.Views[0].Name != "Couch" || len(c.Views) != 2 {
		t.Fatalf("rename: %v %+v", err, c.Views)
	}
	if err := c.SaveView(LibraryView{Name: "short ones"}, "Couch"); err == nil {
		t.Fatal("rename onto another view")
	}
	if err := c.SaveView(LibraryView{Name: "x"}, "Nope"); err == nil {
		t.Fatal("rename of a view that isn't there")
	}
	// It survives a save and load.
	b, _ := json.Marshal(c)
	var back Config
	if err := json.Unmarshal(b, &back); err != nil || len(back.Views) != 2 || back.Views[1].Name != "Short ones" {
		t.Fatalf("round trip: %v %+v", err, back.Views)
	}
	if err := c.DeleteView("COUCH"); err != nil || len(c.Views) != 1 || c.Views[0].Name != "Short ones" {
		t.Fatalf("delete: %v %+v", err, c.Views)
	}
	if c.DeleteView("Couch") == nil {
		t.Fatal("deleted twice")
	}
}

func TestViewsFollowCollections(t *testing.T) {
	c := DefaultConfig()
	c.Tags = map[string][]string{"steam:1": {"Co-op"}, "steam:2": {"Short"}}
	c.Views = []LibraryView{{Name: "Couch", Collection: "Co-op"}, {Name: "Quick", Collection: "Short"}, {Name: "All"}}
	if err := c.RenameTag("co-op", "Couch co-op"); err != nil || c.Views[0].Collection != "Couch co-op" {
		t.Fatalf("rename: %v %+v", err, c.Views)
	}
	if err := c.DeleteTag("Short"); err != nil || c.Views[1].Collection != "" || c.Views[0].Collection != "Couch co-op" {
		t.Fatalf("delete: %v %+v", err, c.Views)
	}
}

func TestViewChecks(t *testing.T) {
	bad := []LibraryView{
		{Name: ""},
		{Name: strings.Repeat("a", MaxTagName+1)},
		{Name: "a", Sort: "random"},
		{Name: "a", Density: "xl"},
		{Name: "a", Show: "hidden"},
		{Name: "a", Status: "won"},
		{Name: "a", Profile: "../p1"},
		{Name: "a", Store: "one\ntwo"},
		{Name: "a", Collection: strings.Repeat("c", MaxTagName+1)},
	}
	for _, v := range bad {
		if _, err := CleanLibraryView(v); err == nil {
			t.Errorf("accepted %+v", v)
		}
	}
	c := DefaultConfig()
	for i := 0; i < MaxViews; i++ {
		if err := c.SaveView(LibraryView{Name: "View " + string(rune('A'+i))}, ""); err != nil {
			t.Fatal(err)
		}
	}
	if c.SaveView(LibraryView{Name: "One more"}, "") == nil {
		t.Fatal("more than MaxViews")
	}
	if err := c.SaveView(LibraryView{Name: "View A", Sort: "size"}, "View A"); err != nil {
		t.Fatalf("updating one at the limit: %v", err)
	}
}
