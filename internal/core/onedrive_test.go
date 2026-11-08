package core

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestOneDriveFolders(t *testing.T) {
	home := filepath.Join(string(filepath.Separator), "Users", "Ana")
	od := filepath.Join(home, "OneDrive")
	docs := filepath.Join(od, "Documents")
	saved := filepath.Join(home, "Saved Games")
	if got := OneDriveFolders([]string{docs, saved}, []string{od, ""}); !reflect.DeepEqual(got, []string{docs}) {
		t.Fatalf("synced %v", got)
	}
	// No variables set: the folder's own path says so.
	work := filepath.Join(home, "OneDrive - Contoso", "Documents")
	if got := OneDriveFolders([]string{work, saved}, nil); !reflect.DeepEqual(got, []string{work}) {
		t.Fatalf("by name %v", got)
	}
	plain := filepath.Join(home, "Documents")
	if got := OneDriveFolders([]string{plain, saved, ""}, []string{od}); got != nil {
		t.Fatalf("not synced %v", got)
	}
	// A folder only named like OneDrive isn't one.
	if got := OneDriveFolders([]string{filepath.Join(home, "OneDriveBackup", "Documents")}, nil); got != nil {
		t.Fatalf("lookalike %v", got)
	}

	if !WithinAny(filepath.Join(docs, "My Games", "Skyrim"), []string{docs}) || !WithinAny(docs, []string{docs}) {
		t.Fatal("saves in synced Documents")
	}
	if WithinAny(filepath.Join(home, "AppData", "Roaming", "Game"), []string{docs}) || WithinAny(filepath.Join(od, "Documents2"), []string{docs}) || WithinAny("", []string{docs}) {
		t.Fatal("saves elsewhere")
	}
}
