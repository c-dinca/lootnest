package core

import (
	"path/filepath"
	"strings"
)

// OneDriveFolders are those of folders (Documents, Saved Games) that
// OneDrive syncs: inside one of its roots (the OneDrive, OneDriveConsumer
// and OneDriveCommercial variables), or under a folder named "OneDrive" or
// "OneDrive - <organisation>" when those aren't set.
func OneDriveFolders(folders, roots []string) []string {
	var out []string
	for _, f := range folders {
		if f == "" {
			continue
		}
		synced := false
		for _, r := range roots {
			synced = synced || r != "" && isWithin(f, r)
		}
		for _, part := range strings.Split(filepath.ToSlash(filepath.Clean(f)), "/") {
			p := strings.ToLower(part)
			synced = synced || p == "onedrive" || strings.HasPrefix(p, "onedrive - ")
		}
		if synced {
			out = append(out, f)
		}
	}
	return out
}

// WithinAny says whether p is one of folders or inside one.
func WithinAny(p string, folders []string) bool {
	for _, f := range folders {
		if p != "" && isWithin(p, f) {
			return true
		}
	}
	return false
}
