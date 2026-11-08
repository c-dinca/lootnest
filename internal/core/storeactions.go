package core

import (
	"regexp"
	"strings"
)

// ---------------------------------------------------------------------------
// Store hand-offs: Lootnest asks a game's own store to install, uninstall,
// verify or show it, through the links each store's launcher handles. The
// store does the work; Lootnest downloads nothing.
// ---------------------------------------------------------------------------

// StoreActs are the actions, in the order the UI lists them.
var StoreActs = []string{"install", "uninstall", "verify", "open"}

// storeLinks: "{id}" is the game's id in its store.
var storeLinks = map[string]string{
	"steam install":     "steam://install/{id}",
	"steam uninstall":   "steam://uninstall/{id}",
	"steam verify":      "steam://validate/{id}",
	"epic install":      "com.epicgames.launcher://apps/{id}?action=install",
	"epic uninstall":    "com.epicgames.launcher://store/library", // uninstalling is done from the library
	"gog open":          "goggalaxy://openGameView/{id}",          // install, uninstall and repair are done there
	"ubisoft install":   "uplay://install/{id}",
	"ubisoft uninstall": "uplay://uninstall/{id}",
	"battlenet open":    "battlenet://{id}",
	// Games these launchers install are in Windows' Installed apps.
	"xbox uninstall":     "ms-settings:appsfeatures",
	"ea uninstall":       "ms-settings:appsfeatures",
	"riot uninstall":     "ms-settings:appsfeatures",
	"rockstar uninstall": "ms-settings:appsfeatures",
	"amazon uninstall":   "ms-settings:appsfeatures",
}

var storeIDs = map[string]*regexp.Regexp{
	"steam":     regexp.MustCompile(`^[1-9]\d{0,9}$`),
	"epic":      regexp.MustCompile(`^[A-Za-z0-9._-]{1,100}%3A[A-Za-z0-9._-]{1,100}%3A[A-Za-z0-9._-]{1,100}$`),
	"gog":       regexp.MustCompile(`^[1-9]\d{0,11}$`),
	"ubisoft":   regexp.MustCompile(`^\d{1,10}$`),
	"battlenet": regexp.MustCompile(`^[A-Za-z0-9]{1,12}$`),
}

// StoreAction is the link that asks a game's store to do action, or ""
// when the store has no official way to (or id isn't one of its ids).
func StoreAction(source, id, action string) string {
	l := storeLinks[source+" "+action]
	if !strings.Contains(l, "{id}") {
		return l
	}
	if re := storeIDs[source]; re == nil || !re.MatchString(id) {
		return ""
	}
	return strings.Replace(l, "{id}", id, 1)
}

// StoreActions lists what a game's store can do for it.
func StoreActions(source, id string) []string {
	var out []string
	for _, a := range StoreActs {
		if StoreAction(source, id, a) != "" {
			out = append(out, a)
		}
	}
	return out
}

var handoffSchemes = []string{"steam", "com.epicgames.launcher", "goggalaxy", "uplay", "battlenet", "ms-settings"}

// SafeHandoff accepts only links to the stores' launchers above: a web
// page, a file or another program's link is never opened from here.
func SafeHandoff(u string) bool {
	scheme, rest, ok := strings.Cut(u, ":")
	if !ok || rest == "" || strings.ContainsAny(u, " \t\r\n\"'<>\\") {
		return false
	}
	for _, s := range handoffSchemes {
		if s == scheme {
			return true
		}
	}
	return false
}
