package core

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// MaxViews is how many Library views can be saved.
const MaxViews = 20

var (
	viewShows     = []string{"", "all", "profiles", "owned"}
	viewSorts     = []string{"", "name", "recent", "played", "rating", "size"}
	viewDensities = []string{"", "s", "m", "l", "list"}
)

// CleanLibraryView trims a view and checks every field is one the Library
// knows.
func CleanLibraryView(v LibraryView) (LibraryView, error) {
	v.Name = strings.Join(strings.Fields(v.Name), " ")
	v.Collection = strings.Join(strings.Fields(v.Collection), " ")
	if v.Name == "" {
		return v, errors.New("Give the view a name.")
	}
	if len([]rune(v.Name)) > MaxTagName {
		return v, fmt.Errorf("Keep view names to %d characters.", MaxTagName)
	}
	if len([]rune(v.Collection)) > MaxTagName || len(v.Store) > 40 {
		return v, errors.New("invalid view")
	}
	if strings.ContainsAny(v.Store, "\r\n\t\x00") || (v.Profile != "" && !idRe.MatchString(v.Profile)) ||
		!slices.Contains(viewShows, v.Show) || (v.Status != "" && !slices.Contains(ShelfStatuses, v.Status)) ||
		!slices.Contains(viewSorts, v.Sort) || !slices.Contains(viewDensities, v.Density) {
		return v, errors.New("invalid view")
	}
	return v, nil
}

func (c *Config) viewIndex(name string) int {
	return slices.IndexFunc(c.Views, func(v LibraryView) bool { return strings.EqualFold(v.Name, name) })
}

// SaveView keeps a view. With old, it replaces that view (a rename keeps
// its place); otherwise it's added, and a name already used is refused.
func (c *Config) SaveView(v LibraryView, old string) error {
	v, err := CleanLibraryView(v)
	if err != nil {
		return err
	}
	at, i := c.viewIndex(v.Name), -1
	if old != "" {
		if i = c.viewIndex(old); i < 0 {
			return errors.New("That view doesn't exist.")
		}
	}
	if at >= 0 && at != i {
		return fmt.Errorf("A view called %s already exists.", c.Views[at].Name)
	}
	if i >= 0 {
		c.Views[i] = v
		return nil
	}
	if len(c.Views) >= MaxViews {
		return fmt.Errorf("Lootnest keeps up to %d views.", MaxViews)
	}
	c.Views = append(c.Views, v)
	return nil
}

// DeleteView forgets a view. The games it showed stay as they are.
func (c *Config) DeleteView(name string) error {
	i := c.viewIndex(name)
	if i < 0 {
		return errors.New("That view doesn't exist.")
	}
	c.Views = slices.Delete(c.Views, i, i+1)
	return nil
}

// retagViews follows a collection's rename (to "" when it's deleted) in
// the views that show it.
func (c *Config) retagViews(from, to string) {
	for i := range c.Views {
		if strings.EqualFold(c.Views[i].Collection, from) {
			c.Views[i].Collection = to
		}
	}
}
