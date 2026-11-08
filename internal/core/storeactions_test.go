package core

import (
	"reflect"
	"testing"
)

func TestStoreAction(t *testing.T) {
	epic := "fn%3A4fe75bbc5a674f4f9b356b5c90567da5%3AFortnite"
	cases := []struct{ source, id, action, want string }{
		{"steam", "620", "install", "steam://install/620"},
		{"steam", "620", "uninstall", "steam://uninstall/620"},
		{"steam", "620", "verify", "steam://validate/620"},
		{"steam", "620", "open", ""},
		{"epic", epic, "install", "com.epicgames.launcher://apps/" + epic + "?action=install"},
		{"epic", epic, "uninstall", "com.epicgames.launcher://store/library"},
		{"epic", epic, "verify", ""},
		{"gog", "1207658924", "open", "goggalaxy://openGameView/1207658924"},
		{"gog", "1207658924", "install", ""},
		{"ubisoft", "4", "install", "uplay://install/4"},
		{"ubisoft", "4", "uninstall", "uplay://uninstall/4"},
		{"battlenet", "Pro", "open", "battlenet://Pro"},
		{"xbox", "", "uninstall", "ms-settings:appsfeatures"},
		{"ea", "", "uninstall", "ms-settings:appsfeatures"},
		{"riot", "", "uninstall", "ms-settings:appsfeatures"},
		{"rockstar", "", "uninstall", "ms-settings:appsfeatures"},
		{"amazon", "", "uninstall", "ms-settings:appsfeatures"},
		{"xbox", "", "install", ""},
		{"itch", "1", "install", ""},
		{"folder", "", "uninstall", ""},
		{"hydra", "", "install", ""},
		{"", "", "install", ""},
		// Ids that aren't the store's are refused, so nothing else rides along.
		{"steam", "0", "install", ""},
		{"steam", "620/../x", "install", ""},
		{"steam", "", "install", ""},
		{"epic", "Fortnite?action=launch", "install", ""},
		{"epic", "", "install", ""},
		{"gog", "12 34", "open", ""},
		{"battlenet", "x://y", "open", ""},
	}
	for _, c := range cases {
		got := StoreAction(c.source, c.id, c.action)
		if got != c.want {
			t.Errorf("StoreAction(%q, %q, %q) = %q, want %q", c.source, c.id, c.action, got, c.want)
		}
		if got != "" && !SafeHandoff(got) {
			t.Errorf("%q isn't a safe hand-off", got)
		}
	}
}

func TestStoreActions(t *testing.T) {
	if got := StoreActions("steam", "620"); !reflect.DeepEqual(got, []string{"install", "uninstall", "verify"}) {
		t.Errorf("steam: %v", got)
	}
	if got := StoreActions("gog", "1207658924"); !reflect.DeepEqual(got, []string{"open"}) {
		t.Errorf("gog: %v", got)
	}
	if got := StoreActions("epic", ""); !reflect.DeepEqual(got, []string{"uninstall"}) {
		t.Errorf("epic without an id: %v", got)
	}
	if got := StoreActions("folder", ""); got != nil {
		t.Errorf("folder: %v", got)
	}
}

func TestSafeHandoff(t *testing.T) {
	for _, u := range []string{"steam://install/620", "goggalaxy://openGameView/1", "ms-settings:appsfeatures", "uplay://install/4", "battlenet://Pro",
		"com.epicgames.launcher://store/library"} {
		if !SafeHandoff(u) {
			t.Errorf("%q refused", u)
		}
	}
	for _, u := range []string{"", "steam:", "https://example.com", "file:///C:/Windows/System32/calc.exe", `C:\Windows\notepad.exe`, "ms-windows-store://pdp/?productid=1",
		"steam://install/620 --x", `steam://"x"`, "itch://games/1", "amazon-games://play/x", "javascript:alert(1)", "STEAM://install/620"} {
		if SafeHandoff(u) {
			t.Errorf("%q allowed", u)
		}
	}
}
