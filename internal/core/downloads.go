package core

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

// ---------------------------------------------------------------------------
// Updates the user already downloaded. Lootnest looks only in the user's
// own Downloads folder (its top level and the folders directly in it), at
// names only, and matches what's there to Library games no launcher
// updates. Applying one is the update centre's job, on the user's click.
// ---------------------------------------------------------------------------

// UpdateTarget is a Library game an update in Downloads may be for.
type UpdateTarget struct {
	Key     string
	Name    string
	Dir     string
	Version string // CopyVersion: the last applied update's, else the folder name's
}

// DownloadMatch is a file or folder in Downloads that looks like an update
// for a game.
type DownloadMatch struct {
	Key     string       `json:"key"`
	Game    string       `json:"game"`
	Path    string       `json:"path"`
	Name    string       `json:"name"`
	Kind    string       `json:"kind"` // "folder", "zip", "7z", "rar" (a set's first part) or "exe"
	Size    int64        `json:"size"`
	ModTime time.Time    `json:"modTime"`
	Check   VersionCheck `json:"check"`
	Newer   bool         `json:"newer"`
}

var rarPartRe = regexp.MustCompile(`(?i)\.part0*(\d+)$`)

// downloadKind is what an entry in Downloads would be as an update, or "".
func downloadKind(name string, dir bool) string {
	if dir {
		return "folder"
	}
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".zip", ".7z", ".exe":
		return ext[1:]
	case ".rar":
		if m := rarPartRe.FindStringSubmatch(strings.TrimSuffix(name, filepath.Ext(name))); m != nil && m[1] != "1" {
			return "" // a later part: the set is read from the first
		}
		return "rar"
	}
	return ""
}

// downloading: names browsers and download tools give a file still coming in.
func downloading(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".crdownload", ".part", ".partial", ".tmp", ".!qb", ".download", ".opdownload":
		return true
	}
	return false
}

func nameTokens(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
}

func allDigits(s string) bool {
	return s != "" && strings.Trim(s, "0123456789") == ""
}

var sequelWords = map[string]bool{"ii": true, "iii": true, "iv": true, "vi": true, "vii": true, "viii": true, "ix": true}

// nameFits: the entry's name starts with the game's name ("Hollow.Tale.v1.3"
// and "HollowTale-Update" for Hollow Tale), after an optional "setup" or
// "update", and doesn't go on to a sequel's number ("Hollow Tale 2").
func nameFits(tokens []string, key, version string) bool {
	if len(tokens) > 0 && (tokens[0] == "setup" || tokens[0] == "update" || tokens[0] == "patch") {
		tokens = tokens[1:]
	}
	joined := ""
	for i, t := range tokens {
		joined += t
		if len(joined) > len(key) || !strings.HasPrefix(key, joined) {
			return false
		}
		if joined != key {
			continue
		}
		if i+1 == len(tokens) {
			return true
		}
		next := tokens[i+1]
		if sequelWords[next] {
			return false
		}
		return !allDigits(next) || strings.HasPrefix(strings.TrimPrefix(version, "build "), next)
	}
	return false
}

// treeStat is a folder's size and the newest time of anything in it. A
// folder's own time counts: extracting keeps the archive's file times.
func treeStat(p string) (size int64, newest time.Time) {
	_ = filepath.WalkDir(p, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() && !d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			if !d.IsDir() {
				size += info.Size()
			}
			if info.ModTime().After(newest) {
				newest = info.ModTime()
			}
		}
		return nil
	})
	return size, newest
}

// rarParts is a .rar set's files, from its first part, and whether a part
// is still coming in.
func rarParts(first string) (files []string, busy bool) {
	files = []string{first}
	name := filepath.Base(first)
	stem := strings.TrimSuffix(name, filepath.Ext(name))
	at := rarPartRe.FindStringIndex(stem)
	if at == nil {
		return files, false
	}
	entries, _ := os.ReadDir(filepath.Dir(first))
	for _, e := range entries {
		n := e.Name()
		part := n
		if downloading(n) {
			part = strings.TrimSuffix(n, filepath.Ext(n))
		}
		ps := strings.TrimSuffix(part, filepath.Ext(part))
		i := rarPartRe.FindStringIndex(ps)
		if e.IsDir() || n == name || i == nil || !strings.EqualFold(filepath.Ext(part), ".rar") || !strings.EqualFold(ps[:i[0]], stem[:at[0]]) {
			continue
		}
		if part != n {
			busy = true
			continue
		}
		files = append(files, filepath.Join(filepath.Dir(first), n))
	}
	return files, busy
}

// MatchDownloads lists the updates in dir (the user's Downloads folder) for
// games: entries at its top level and in the folders directly in it, newer
// than since, at least 64 KB, not still being written, whose name starts
// with a game's name. A folder or an installer needs a version in its name.
// A folder that holds a match is not a match itself.
func MatchDownloads(dir string, games []UpdateTarget, since time.Time) []DownloadMatch {
	now := time.Now()
	var out []DownloadMatch
	ofGame := func(p string) bool {
		for _, g := range games {
			if g.Dir != "" && (isWithin(p, g.Dir) || isWithin(g.Dir, p)) {
				return true
			}
		}
		return false
	}
	var look func(d string, deeper bool)
	look = func(d string, deeper bool) {
		entries, err := os.ReadDir(d)
		if err != nil {
			return
		}
		names := map[string]bool{}
		for _, e := range entries {
			names[strings.ToLower(e.Name())] = true
		}
		for _, e := range entries {
			name, p := e.Name(), filepath.Join(d, e.Name())
			if e.Type()&fs.ModeSymlink != 0 || downloading(name) || names[strings.ToLower(name)+".part"] || names[strings.ToLower(name)+".crdownload"] {
				continue
			}
			if e.IsDir() && deeper {
				n := len(out)
				look(p, false)
				if len(out) > n {
					continue
				}
			}
			kind := downloadKind(name, e.IsDir())
			if kind == "" || ofGame(p) {
				continue
			}
			from, to := UpdateVersions(name)
			if (kind == "exe" || kind == "folder") && to == "" {
				continue // more likely a game's own program or folder than an update
			}
			tokens := nameTokens(name)
			best := 0
			var fits []UpdateTarget
			for _, g := range games {
				key := norm(g.Name)
				if len(key) < 3 || len(key) < best || !nameFits(tokens, key, to) {
					continue
				}
				if len(key) > best {
					best, fits = len(key), nil
				}
				fits = append(fits, g)
			}
			if len(fits) == 0 {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			size, newest := info.Size(), info.ModTime()
			arrived := newest
			switch kind {
			case "folder":
				size, newest = treeStat(p)
				arrived = laterOf(newest, created(info))
			case "rar":
				parts, busy := rarParts(p)
				if busy {
					continue
				}
				for _, f := range parts[1:] {
					if fi, err := os.Stat(f); err == nil {
						size += fi.Size()
						newest = laterOf(newest, fi.ModTime())
					}
				}
				arrived = newest
			}
			if size < 64<<10 || arrived.Before(since) || now.Sub(newest) < 10*time.Second {
				continue
			}
			for _, g := range fits {
				c := CheckVersions(g.Version, from, to)
				out = append(out, DownloadMatch{Key: g.Key, Game: g.Name, Path: p, Name: name, Kind: kind, Size: size, ModTime: newest, Check: c,
					Newer: c.State == "ok" || c.State == "skips"})
			}
		}
	}
	look(dir, true)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ModTime.After(out[j].ModTime) })
	return out
}

func laterOf(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

// Files is the match's file, with the rest of the set for a .rar in parts.
func (m DownloadMatch) Files() []string {
	if m.Kind != "rar" {
		return []string{m.Path}
	}
	files, _ := rarParts(m.Path)
	return files
}

// DownloadsSeen is downloads.json: matches already announced and matches
// the user ignored, by Stamp, so each file is announced once.
type DownloadsSeen struct {
	Announced map[string]bool `json:"announced"`
	Ignored   map[string]bool `json:"ignored"`
}

// Stamp names one version of a match's file: its path, size and time.
func (m DownloadMatch) Stamp() string {
	return fmt.Sprintf("%s|%d|%d", m.Path, m.Size, m.ModTime.Unix())
}

// UpdateDownloadsSeen changes downloads.json under the store's lock, and
// forgets files that aren't there any more.
func (s *Store) UpdateDownloadsSeen(change func(*DownloadsSeen)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := filepath.Join(s.Dir, "downloads.json")
	d := DownloadsSeen{}
	if b, err := os.ReadFile(p); err == nil {
		_ = json.Unmarshal(b, &d)
	}
	if d.Announced == nil {
		d.Announced = map[string]bool{}
	}
	if d.Ignored == nil {
		d.Ignored = map[string]bool{}
	}
	before, _ := json.Marshal(d)
	change(&d)
	for _, set := range []map[string]bool{d.Announced, d.Ignored} {
		for k := range set {
			if path, _, _ := strings.Cut(k, "|"); !exists(path) {
				delete(set, k)
			}
		}
	}
	if after, _ := json.Marshal(d); string(after) == string(before) {
		return nil
	}
	return writeJSONAtomic(p, d)
}
