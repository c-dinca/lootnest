package core

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Cloud saves: each player's saves for each game, kept in a folder a cloud
// app syncs (or on a USB drive), so every PC with Lootnest has the newest.
//
//	<cloud>/Lootnest/key.json                  the password's salt, when set
//	<cloud>/Lootnest/sync/<player>/player.json who it is (name, colour)
//	<cloud>/Lootnest/sync/<player>/<game>.zip  their saves (.lnbak encrypted)
//	<cloud>/Lootnest/sync/<player>/<game>.json what's in it (SyncMeta)
//
// <game> is the same on every PC (GameSyncKey), and the save locations
// inside are named by their path (%APPDATA%\...), not by this PC's ids. A
// hash of the saves (SyncHash) is the same on every PC for the same files,
// so Lootnest can tell which side changed since the last sync:
//
//   - only this PC: the saves go up (after each session);
//   - only the cloud: they come down (before the game starts);
//   - both: the player chooses, and the other copy is kept as a backup.
// ---------------------------------------------------------------------------

// SyncMeta describes a player's saves for a game in the cloud.
type SyncMeta struct {
	Hash      string    `json:"hash"`
	Version   int       `json:"version"`
	Computer  string    `json:"computer"`
	Changed   time.Time `json:"changed"`
	Game      string    `json:"game"`
	Player    string    `json:"player"`
	Encrypted bool      `json:"encrypted"`
	Size      int64     `json:"size"`
}

// SyncRecord is what this PC knew at the last sync of a player's game.
type SyncRecord struct {
	Local string    `json:"local"` // SyncHash of the saves here then
	Cloud string    `json:"cloud"` // the cloud's Hash then
	At    time.Time `json:"at"`
}

// SyncState is this PC's sync.json.
type SyncState struct {
	// Records: local player id -> GameSyncKey -> record.
	Records map[string]map[string]SyncRecord `json:"records"`
	// Links: the cloud folder of a player here, when it isn't their id
	// (the same person set up on another PC first).
	Links map[string]string `json:"links,omitempty"`
}

// CloudDir is where Lootnest keeps its files in a cloud folder.
func CloudDir(root string) string { return filepath.Join(root, "Lootnest") }

var syncKeyBad = regexp.MustCompile(`[^a-z0-9]+`)

// GameSyncKey names a game the same way on every PC: by its Steam app id,
// its launcher link, or its name.
func GameSyncKey(g *Game) string {
	switch {
	case g.Launch.Type == "steam" && g.Launch.AppID > 0:
		return "steam-" + strconv.Itoa(g.Launch.AppID)
	case g.Launch.URI != "":
		sum := sha256.Sum256([]byte(strings.ToLower(g.Launch.URI)))
		return "link-" + hex.EncodeToString(sum[:6])
	}
	n := strings.Trim(syncKeyBad.ReplaceAllString(strings.ToLower(g.Name), "-"), "-")
	if len(n) > 60 {
		n = n[:60]
	}
	if n == "" {
		n = g.ID
	}
	return "name-" + n
}

// SavesChanged is the newest time a file in a game's save locations
// changed (dir as for SyncHash), zero when none is there. Folder times are
// left out: copies and swaps set them without changing the saves.
func SavesChanged(g *Game, dir func(PathEntry) string) time.Time {
	var t time.Time
	for _, p := range g.Paths {
		_ = filepath.WalkDir(dir(p), func(_ string, d fs.DirEntry, err error) error {
			if err != nil || !d.Type().IsRegular() {
				return nil
			}
			if info, err := d.Info(); err == nil && info.ModTime().After(t) {
				t = info.ModTime()
			}
			return nil
		})
	}
	return t
}

// SyncHash is a hash of a game's saves that is the same on every PC for the
// same files: dir(entry) is where each save location's files are (the
// player's profile, or the files in place).
func SyncHash(g *Game, dir func(PathEntry) string) (string, error) {
	h := sha256.New()
	paths := append([]PathEntry{}, g.Paths...)
	sort.Slice(paths, func(i, j int) bool { return strings.ToLower(paths[i].Path) < strings.ToLower(paths[j].Path) })
	any := false
	for _, p := range paths {
		root := dir(p)
		st, err := os.Stat(root)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		any = true
		fmt.Fprintf(h, "path\x00%s\x00", strings.ToLower(p.Path))
		if !st.IsDir() {
			if err := hashFile(h, root, "."); err != nil {
				return "", err
			}
			continue
		}
		var files []string
		err = filepath.WalkDir(root, func(fp string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.Type().IsRegular() {
				rel, _ := filepath.Rel(root, fp)
				files = append(files, filepath.ToSlash(rel))
			}
			return nil
		})
		if err != nil {
			return "", err
		}
		sort.Strings(files)
		for _, f := range files {
			if err := hashFile(h, filepath.Join(root, filepath.FromSlash(f)), f); err != nil {
				return "", err
			}
		}
	}
	if !any {
		return "", nil // no saves at all
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func hashFile(h io.Writer, path, rel string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	st, _ := f.Stat()
	fmt.Fprintf(h, "file\x00%s\x00%d\x00", rel, st.Size())
	_, err = io.Copy(h, f)
	return err
}

// ------------------------------------------------------------- the cloud ---

// Cloud is one cloud folder with the key its files are encrypted with.
type Cloud struct {
	Root     string // the folder the user chose
	Key      []byte // nil: not encrypted
	Computer string
}

func (c *Cloud) dir() string { return CloudDir(c.Root) }

func (c *Cloud) playerDir(cloudPlayer string) string {
	return filepath.Join(c.dir(), "sync", filepath.Base(cloudPlayer))
}

func (c *Cloud) files(cloudPlayer, game string) (data, meta string) {
	base := filepath.Join(c.playerDir(cloudPlayer), game)
	ext := ".zip"
	if c.Key != nil {
		ext = ".lnbak"
	}
	return base + ext, base + ".json"
}

// Meta reads a player's saves' description for a game, or nil.
func (c *Cloud) Meta(cloudPlayer, game string) *SyncMeta {
	_, mp := c.files(cloudPlayer, game)
	b, err := os.ReadFile(mp)
	if err != nil {
		return nil
	}
	var m SyncMeta
	if json.Unmarshal(b, &m) != nil || m.Hash == "" {
		return nil
	}
	return &m
}

// CloudPlayer is a player with saves in the cloud.
type CloudPlayer struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// Players lists the players with saves in the cloud folder.
func (c *Cloud) Players() []CloudPlayer {
	ents, _ := os.ReadDir(filepath.Join(c.dir(), "sync"))
	var out []CloudPlayer
	for _, e := range ents {
		if !e.IsDir() || !idRe.MatchString(e.Name()) {
			continue
		}
		p := CloudPlayer{ID: e.Name()}
		if b, err := os.ReadFile(filepath.Join(c.playerDir(e.Name()), "player.json")); err == nil {
			_ = json.Unmarshal(b, &p)
			p.ID = e.Name()
		}
		out = append(out, p)
	}
	return out
}

// Upload puts a player's saves for a game in the cloud: dir(entry) is
// where each save location's files are.
func (c *Cloud) Upload(cloudPlayer string, p *Player, g *Game, dir func(PathEntry) string, hash string, version int) (*SyncMeta, error) {
	pd := c.playerDir(cloudPlayer)
	if err := os.MkdirAll(pd, 0o755); err != nil {
		return nil, err
	}
	pj, _ := json.Marshal(CloudPlayer{ID: cloudPlayer, Name: p.Name, Color: p.Color})
	_ = writeFileAtomic(filepath.Join(pd, "player.json"), pj)
	key := GameSyncKey(g)
	data, mp := c.files(cloudPlayer, key)
	opt, err := c.archiveOptions()
	if err != nil {
		return nil, err
	}
	size, err := writeArchiveFile(data, func(w io.Writer) error { return writeSaves(w, g.Paths, dir) }, opt)
	if err != nil {
		return nil, err
	}
	// The other kind of file, from before a password was set or removed.
	other := strings.TrimSuffix(data, filepath.Ext(data))
	if c.Key != nil {
		_ = os.Remove(other + ".zip")
	} else {
		_ = os.Remove(other + ".lnbak")
	}
	meta := &SyncMeta{Hash: hash, Version: version, Computer: c.Computer, Changed: time.Now().UTC(), Game: g.Name, Player: p.Name,
		Encrypted: c.Key != nil, Size: size}
	b, _ := json.MarshalIndent(meta, "", "  ")
	if err := writeFileAtomic(mp, b); err != nil {
		return nil, err
	}
	return meta, nil
}

func (c *Cloud) archiveOptions() (BackupOptions, error) {
	if c.Key == nil {
		return BackupOptions{}, nil
	}
	salt, iters, err := c.keyParams()
	if err != nil {
		return BackupOptions{}, err
	}
	return BackupOptions{Key: c.Key, Salt: salt, Iters: iters}, nil
}

// writeSaves zips save locations: each as p/<n>, and paths.json naming the
// location (its tokenized path) of each n.
func writeSaves(w io.Writer, paths []PathEntry, dir func(PathEntry) string) error {
	zw := zip.NewWriter(w)
	m := map[string]string{}
	for i, pe := range paths {
		if src := dir(pe); exists(src) {
			if err := zipPath(zw, src, "p/"+strconv.Itoa(i)); err != nil {
				return err
			}
			m[strconv.Itoa(i)] = pe.Path
		}
	}
	b, _ := json.Marshal(m)
	f, err := zw.Create("paths.json")
	if err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		return err
	}
	return zw.Close()
}

// readSaves unpacks a saves file into place: each location to dir of the
// entry with the same path. Locations paths doesn't have are counted.
func readSaves(file string, key []byte, paths []PathEntry, dir func(PathEntry) string, tmp string) (skipped int, err error) {
	defer os.RemoveAll(tmp)
	ob, err := openArchive(file, tmp, key)
	if err != nil {
		return 0, err
	}
	b, err := os.ReadFile(filepath.Join(ob, "paths.json"))
	if err != nil {
		return 0, errors.New("The cloud copy is damaged.")
	}
	var m map[string]string
	if json.Unmarshal(b, &m) != nil {
		return 0, errors.New("The cloud copy is damaged.")
	}
	for idx, path := range m {
		var to *PathEntry
		for i := range paths {
			if strings.EqualFold(paths[i].Path, path) {
				to = &paths[i]
			}
		}
		if to == nil {
			skipped++
			continue
		}
		if err := Mirror(filepath.Join(ob, "p", idx), dir(*to)); err != nil {
			return skipped, err
		}
	}
	// Locations here that the copy doesn't have are left as they are:
	// nothing is deleted because another PC lacks it.
	return skipped, nil
}

// Download puts a player's saves for a game from the cloud into place:
// dir(entry) is where each save location's files go. Save locations this
// PC's game doesn't have are left out (their count is returned).
func (c *Cloud) Download(cloudPlayer string, g *Game, dir func(PathEntry) string, tmp string) (skipped int, err error) {
	data, _ := c.files(cloudPlayer, GameSyncKey(g))
	if !exists(data) {
		// Made with the other setting (a password set or removed since).
		alt := strings.TrimSuffix(data, filepath.Ext(data))
		if c.Key != nil {
			alt += ".zip"
		} else {
			alt += ".lnbak"
		}
		data = alt
	}
	return readSaves(data, c.Key, g.Paths, dir, tmp)
}

// openArchive unpacks a sync file (zip, or encrypted with key) into tmp and
// returns the folder with its files. Names are checked like a backup's.
func openArchive(file, tmp string, key []byte) (string, error) {
	enc, err := IsEncryptedBackup(file)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(tmp, 0o700); err != nil {
		return "", err
	}
	zipFile := file
	if enc {
		if key == nil {
			return "", ErrNeedPassword
		}
		zipFile = filepath.Join(tmp, "sync.zip")
		out, err := os.Create(zipFile)
		if err != nil {
			return "", err
		}
		in, err := os.Open(file)
		if err != nil {
			out.Close()
			return "", err
		}
		err = decryptBackup(out, in, func([]byte, int) ([]byte, error) { return key, nil })
		in.Close()
		if cerr := out.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return "", err
		}
	}
	zr, err := zip.OpenReader(zipFile)
	if err != nil {
		return "", errors.New("The cloud copy is damaged.")
	}
	defer zr.Close()
	dst := filepath.Join(tmp, "files")
	var total uint64
	if len(zr.File) > maxBackupEntries {
		return "", errors.New("The cloud copy has too many files.")
	}
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, "/") {
			continue
		}
		if !safeSyncName(f.Name) {
			return "", fmt.Errorf("The cloud copy has a file Lootnest won't use (%q).", f.Name)
		}
		if total += f.UncompressedSize64; total > maxBackupBytes {
			return "", errors.New("The cloud copy is too big.")
		}
		if err := unzipFile(f, filepath.Join(dst, filepath.FromSlash(f.Name))); err != nil {
			return "", err
		}
	}
	return dst, nil
}

// safeSyncName: "paths.json", and "p/<number>" or files under it, with the
// same rules for each part of the name as a backup's.
func safeSyncName(name string) bool {
	if name == "paths.json" {
		return true
	}
	if strings.ContainsAny(name, "\\:\x00") || path.Clean(name) != name {
		return false
	}
	parts := strings.Split(name, "/")
	if len(parts) < 2 || parts[0] != "p" {
		return false
	}
	if n, err := strconv.Atoi(parts[1]); err != nil || n < 0 || n > 999 || strconv.Itoa(n) != parts[1] {
		return false
	}
	for _, p := range parts[2:] {
		if !safeComponent(p) {
			return false
		}
	}
	return true
}

// ------------------------------------------------------------- the key ---

type cloudKeyFile struct {
	Salt  string `json:"salt"`
	Iters int    `json:"iters"`
	Check string `json:"check"` // a known text encrypted with the key
}

const keyCheckText = "lootnest cloud key"

// CloudKey makes the key for a cloud folder from the password. The first
// PC to set one writes the folder's key.json (its salt and a check), and
// every other PC makes the same key from the same password with it; a
// different password is refused.
func CloudKey(root, password string) ([]byte, error) {
	path := filepath.Join(CloudDir(root), "key.json")
	if b, err := os.ReadFile(path); err == nil {
		var kf cloudKeyFile
		if json.Unmarshal(b, &kf) != nil {
			return nil, errors.New("The cloud folder's key.json is damaged.")
		}
		salt, err := base64.StdEncoding.DecodeString(kf.Salt)
		if err != nil || len(salt) != backupSaltLen || kf.Iters < 10_000 {
			return nil, errors.New("The cloud folder's key.json is damaged.")
		}
		key, err := BackupKey(password, salt, kf.Iters)
		if err != nil {
			return nil, err
		}
		check, err := base64.StdEncoding.DecodeString(kf.Check)
		if err != nil {
			return nil, errors.New("The cloud folder's key.json is damaged.")
		}
		var out bytes.Buffer
		if decryptBackup(&out, bytes.NewReader(check), func([]byte, int) ([]byte, error) { return key, nil }) != nil || out.String() != keyCheckText {
			return nil, errors.New("That isn't the password this cloud folder uses. Use the one you set on your other PC.")
		}
		return key, nil
	}
	salt := NewBackupSalt()
	key, err := BackupKey(password, salt, BackupKDFIters)
	if err != nil {
		return nil, err
	}
	var check bytes.Buffer
	if err := encryptBackup(&check, strings.NewReader(keyCheckText), key, salt, BackupKDFIters); err != nil {
		return nil, err
	}
	b, _ := json.MarshalIndent(cloudKeyFile{Salt: base64.StdEncoding.EncodeToString(salt), Iters: BackupKDFIters,
		Check: base64.StdEncoding.EncodeToString(check.Bytes())}, "", "  ")
	if err := os.MkdirAll(CloudDir(root), 0o755); err != nil {
		return nil, err
	}
	if err := writeFileAtomic(path, b); err != nil {
		return nil, err
	}
	return key, nil
}

// CloudHasKey says whether a cloud folder's files need a password.
func CloudHasKey(root string) bool { return exists(filepath.Join(CloudDir(root), "key.json")) }

// CloudKeyMatches says whether key is the one the folder's key.json checks.
func CloudKeyMatches(root string, key []byte) bool {
	b, err := os.ReadFile(filepath.Join(CloudDir(root), "key.json"))
	if err != nil || key == nil {
		return false
	}
	var kf cloudKeyFile
	if json.Unmarshal(b, &kf) != nil {
		return false
	}
	check, err := base64.StdEncoding.DecodeString(kf.Check)
	if err != nil {
		return false
	}
	var out bytes.Buffer
	return decryptBackup(&out, bytes.NewReader(check), func([]byte, int) ([]byte, error) { return key, nil }) == nil && out.String() == keyCheckText
}

func (c *Cloud) keyParams() ([]byte, int, error) {
	b, err := os.ReadFile(filepath.Join(c.dir(), "key.json"))
	if err != nil {
		return nil, 0, errors.New("The cloud folder has no key.json; set the password again.")
	}
	var kf cloudKeyFile
	if json.Unmarshal(b, &kf) != nil {
		return nil, 0, errors.New("The cloud folder's key.json is damaged.")
	}
	salt, err := base64.StdEncoding.DecodeString(kf.Salt)
	if err != nil || len(salt) != backupSaltLen {
		return nil, 0, errors.New("The cloud folder's key.json is damaged.")
	}
	return salt, kf.Iters, nil
}

// --------------------------------------------------------------- deciding ---

// SyncAction is what a sync of one player's game should do.
type SyncAction string

const (
	SyncNothing  SyncAction = ""         // the same on both sides
	SyncUp       SyncAction = "up"       // only this PC changed
	SyncDown     SyncAction = "down"     // only the cloud changed
	SyncConflict SyncAction = "conflict" // both changed since the last sync
)

// DecideSync compares the saves here (local hash), the cloud's and what
// both were at the last sync.
func DecideSync(local string, cloud *SyncMeta, rec SyncRecord, seen bool) SyncAction {
	cloudHash := ""
	if cloud != nil {
		cloudHash = cloud.Hash
	}
	switch {
	case local == cloudHash:
		return SyncNothing
	case cloud == nil:
		if local == "" {
			return SyncNothing
		}
		return SyncUp
	case local == "":
		return SyncDown
	case !seen:
		return SyncConflict // never synced here, and both have saves
	}
	localChanged, cloudChanged := local != rec.Local, cloudHash != rec.Cloud
	switch {
	case localChanged && cloudChanged:
		return SyncConflict
	case cloudChanged:
		return SyncDown
	case localChanged:
		return SyncUp
	}
	return SyncNothing
}

// ------------------------------------------------------------ sync.json ---

func (s *Store) SyncPath() string { return filepath.Join(s.Dir, "sync.json") }

func (s *Store) LoadSync() SyncState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readSync()
}

func (s *Store) readSync() SyncState {
	var st SyncState
	if b, err := os.ReadFile(s.SyncPath()); err == nil {
		_ = json.Unmarshal(b, &st)
	}
	if st.Records == nil {
		st.Records = map[string]map[string]SyncRecord{}
	}
	if st.Links == nil {
		st.Links = map[string]string{}
	}
	return st
}

// UpdateSync changes sync.json under the store's lock.
func (s *Store) UpdateSync(change func(*SyncState)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.readSync()
	change(&st)
	return writeJSONAtomic(s.SyncPath(), st)
}

// CloudPlayerID is the cloud folder of a player here: their link, else
// their id.
func (st SyncState) CloudPlayerID(pid string) string {
	if l := st.Links[pid]; l != "" {
		return l
	}
	return pid
}

// LinkPlayers links players here to the cloud's players of the same name
// that aren't linked yet, so the same person set up on two PCs shares one
// set of saves. It returns whether anything changed.
func LinkPlayers(st *SyncState, players []Player, cloud []CloudPlayer) bool {
	changed := false
	taken := map[string]bool{}
	for _, p := range players {
		taken[st.CloudPlayerID(p.ID)] = true
	}
	for _, p := range players {
		cid := st.CloudPlayerID(p.ID)
		has := false
		for _, c := range cloud {
			has = has || c.ID == cid
		}
		if has {
			continue
		}
		for _, c := range cloud {
			if !taken[c.ID] && strings.EqualFold(strings.TrimSpace(c.Name), strings.TrimSpace(p.Name)) {
				st.Links[p.ID] = c.ID
				taken[c.ID] = true
				changed = true
				break
			}
		}
	}
	return changed
}

// CloudKeyParams are the salt and rounds of a cloud folder's key.json.
func CloudKeyParams(root string) ([]byte, int, error) { return (&Cloud{Root: root}).keyParams() }

// EnsureCloudKey writes key.json for a folder that has none, from a key this
// PC already has (the folder changed); a folder with one is left alone.
func EnsureCloudKey(root string, key, salt []byte, iters int) error {
	path := filepath.Join(CloudDir(root), "key.json")
	if exists(path) {
		return nil
	}
	var check bytes.Buffer
	if err := encryptBackup(&check, strings.NewReader(keyCheckText), key, salt, iters); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(cloudKeyFile{Salt: base64.StdEncoding.EncodeToString(salt), Iters: iters,
		Check: base64.StdEncoding.EncodeToString(check.Bytes())}, "", "  ")
	if err := os.MkdirAll(CloudDir(root), 0o755); err != nil {
		return err
	}
	return writeFileAtomic(path, b)
}
