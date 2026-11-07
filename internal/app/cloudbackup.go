//go:build windows

package app

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// ---------------------------------------------------------------------------
// Backups to a folder the user picks, usually one OneDrive, Google Drive or
// Dropbox syncs (core/archive.go has the file format). Lootnest only writes
// the file; uploading it is the sync app's job, so nothing here goes online.
// ---------------------------------------------------------------------------

// BackupStatus is what the last backup did, kept in backup.json.
type BackupStatus struct {
	Last    time.Time `json:"last,omitempty"`
	File    string    `json:"file,omitempty"`
	Size    int64     `json:"size,omitempty"`
	Error   string    `json:"error,omitempty"`
	ErrorAt time.Time `json:"errorAt,omitempty"`
	Running bool      `json:"running"`
}

type CloudFolder struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type BackupInfo struct {
	Sync      bool          `json:"sync"`   // cloud saves on
	Locked    bool          `json:"locked"` // the folder has a password this PC doesn't have
	Dir       string        `json:"dir"`
	Target    string        `json:"target"` // <dir>\Lootnest
	Auto      bool          `json:"auto"`
	Keep      int           `json:"keep"`
	Encrypted bool          `json:"encrypted"`
	Computer  string        `json:"computer"`
	Status    BackupStatus  `json:"status"`
	Folders   []CloudFolder `json:"folders"`
}

var (
	backupMu      sync.Mutex
	backupRunning bool
	backupAgain   bool
	restoreMu     sync.Mutex
	restoreOpen   *core.OpenedBackup
	restoreTmp    string
)

func backupStatusPath() string { return filepath.Join(DataDir(), "backup.json") }

func loadBackupStatus() BackupStatus {
	var st BackupStatus
	if b, err := os.ReadFile(backupStatusPath()); err == nil {
		_ = json.Unmarshal(b, &st)
	}
	st.Running = false
	return st
}

func saveBackupStatus(st BackupStatus) {
	b, _ := json.MarshalIndent(st, "", "  ")
	_ = os.WriteFile(backupStatusPath()+".tmp", b, 0o644)
	_ = os.Rename(backupStatusPath()+".tmp", backupStatusPath())
}

func computerName() string {
	if n := os.Getenv("COMPUTERNAME"); n != "" {
		return n
	}
	if n, err := os.Hostname(); err == nil && n != "" {
		return n
	}
	return "PC"
}

func backupTarget(dir string) string { return filepath.Join(dir, "Lootnest") }

// cloudFolders are the folders the cloud apps on this PC sync.
func cloudFolders() []CloudFolder {
	var out []CloudFolder
	seen := map[string]bool{}
	add := func(name, p string) {
		p = filepath.Clean(strings.TrimSpace(p))
		if p == "." || seen[strings.ToLower(p)] {
			return
		}
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			seen[strings.ToLower(p)] = true
			out = append(out, CloudFolder{name, p})
		}
	}
	add("OneDrive", os.Getenv("OneDriveConsumer"))
	add("OneDrive", os.Getenv("OneDrive"))
	add("OneDrive (work or school)", os.Getenv("OneDriveCommercial"))
	// Google Drive for desktop shows the drive as a letter with "My Drive".
	for c := 'D'; c <= 'Z'; c++ {
		add("Google Drive", string(c)+`:\My Drive`)
	}
	home := os.Getenv("USERPROFILE")
	add("Google Drive", filepath.Join(home, "My Drive"))
	add("Google Drive", filepath.Join(home, "Google Drive"))
	// Dropbox says where it syncs in info.json.
	for _, base := range []string{os.Getenv("APPDATA"), os.Getenv("LOCALAPPDATA")} {
		b, err := os.ReadFile(filepath.Join(base, "Dropbox", "info.json"))
		if err != nil {
			continue
		}
		var info map[string]struct {
			Path string `json:"path"`
		}
		if json.Unmarshal(b, &info) == nil {
			add("Dropbox", info["personal"].Path)
			add("Dropbox (business)", info["business"].Path)
		}
	}
	add("iCloud Drive", filepath.Join(home, "iCloudDrive"))
	return out
}

func (a *App) apiBackupInfo() BackupInfo {
	a.mu.Lock()
	s := a.cfg.Settings
	a.mu.Unlock()
	info := BackupInfo{Sync: s.CloudSync, Dir: s.BackupDir, Auto: s.BackupAuto, Keep: s.BackupKeep, Encrypted: s.BackupKey != "",
		Computer: computerName(), Status: loadBackupStatus(), Folders: cloudFolders()}
	if info.Keep == 0 {
		info.Keep = 5
	}
	if s.BackupDir != "" {
		info.Target = backupTarget(s.BackupDir)
		if core.CloudHasKey(s.BackupDir) {
			key, _, _ := backupKey(s)
			info.Locked = key == nil || !core.CloudKeyMatches(s.BackupDir, key)
			info.Encrypted = !info.Locked
		} else {
			info.Encrypted = false
		}
	}
	backupMu.Lock()
	info.Status.Running = backupRunning
	backupMu.Unlock()
	return info
}

type BackupSettings struct {
	Dir  string `json:"dir"`
	Auto bool   `json:"auto"`
	Keep int    `json:"keep"`
	Sync bool   `json:"sync"`
}

// apiSetBackup saves where backups go, whether they're made after each
// session and how many to keep.
func (a *App) apiSetBackup(b BackupSettings) error {
	b.Dir = strings.TrimSpace(b.Dir)
	if b.Keep < 1 || b.Keep > 50 {
		return errors.New("Keep between 1 and 50 backups.")
	}
	if b.Dir != "" {
		if !filepath.IsAbs(b.Dir) {
			return errors.New("Choose a folder for backups.")
		}
		if st, err := os.Stat(b.Dir); err != nil || !st.IsDir() {
			return errors.New("That folder doesn't exist.")
		}
		if isInside(b.Dir, DataDir()) || isInside(backupTarget(b.Dir), DataDir()) {
			return errors.New("Backups can't go inside Lootnest's own data folder; choose another folder.")
		}
	} else {
		b.Auto = false
	}
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	first := cfg.Settings.BackupDir == "" && b.Dir != ""
	if first {
		b.Auto, b.Sync = true, true // what a cloud folder is for
	}
	if b.Dir == "" {
		b.Sync = false
	}
	// A new folder and a password here: the folder's own password wins
	// (key.json); a folder without one gets this PC's.
	if s := &cfg.Settings; s.BackupKey != "" && b.Dir != "" && !strings.EqualFold(b.Dir, s.BackupDir) {
		key, salt, err := backupKey(*s)
		switch {
		case err != nil || core.CloudHasKey(b.Dir) && !core.CloudKeyMatches(b.Dir, key):
			s.BackupKey, s.BackupSalt = "", "" // needs that folder's password
		case !core.CloudHasKey(b.Dir):
			if err := core.EnsureCloudKey(b.Dir, key, salt, core.BackupKDFIters); err != nil {
				return fmt.Errorf("couldn't set the password up in that folder: %w", err)
			}
		}
	}
	cfg.Settings.BackupDir, cfg.Settings.BackupAuto, cfg.Settings.BackupKeep, cfg.Settings.CloudSync = b.Dir, b.Auto, b.Keep, b.Sync
	if err := a.saveConfig(cfg); err != nil {
		return err
	}
	if first {
		go a.runBackup()
	}
	if b.Sync {
		go func() { _ = a.apiSyncNow() }()
	}
	return nil
}

func isInside(p, dir string) bool {
	rel, err := filepath.Rel(strings.ToLower(filepath.Clean(dir)), strings.ToLower(filepath.Clean(p)))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, `..\`) && !filepath.IsAbs(rel)
}

// apiSetBackupPassword turns on encryption with a new password ("" turns it
// off). The key made from it is kept for this Windows user (DPAPI) so
// backups can be made by themselves; the password itself is never saved.
// It takes a moment, so it runs in the background and a "backup" event
// says when it's done.
func (a *App) apiSetBackupPassword(pw string) error {
	if pw != "" && utf8.RuneCountInString(pw) < 8 {
		return errors.New("Use at least 8 characters.")
	}
	if len(pw) > 1024 {
		return errors.New("That password is too long.")
	}
	go func() {
		out := map[string]any{"password": true}
		defer func() { a.emit("backup", out) }()
		a.mu.Lock()
		cfg := a.cfg.Clone()
		a.mu.Unlock()
		dir := cfg.Settings.BackupDir
		if dir == "" {
			out["error"] = "Choose a cloud folder first."
			return
		}
		cfg.Settings.BackupSalt, cfg.Settings.BackupKey = "", ""
		if pw == "" {
			// The folder isn't encrypted any more: new saves and backups go
			// up as they are.
			_ = os.Remove(filepath.Join(core.CloudDir(dir), "key.json"))
		} else {
			// The folder's password: the first PC sets it, the others must
			// use the same one (core.CloudKey checks it).
			key, err := core.CloudKey(dir, pw)
			var salt []byte
			if err == nil {
				salt, _, err = core.CloudKeyParams(dir)
			}
			if err == nil {
				cfg.Settings.BackupKey, err = protectSecret(base64.StdEncoding.EncodeToString(key))
			}
			if err != nil {
				out["error"] = err.Error()
				return
			}
			cfg.Settings.BackupSalt = base64.StdEncoding.EncodeToString(salt)
		}
		if err := a.saveConfig(cfg); err != nil {
			out["error"] = err.Error()
		}
	}()
	return nil
}

// backupKey is the key this PC encrypts backups with, or nil without a
// password.
func backupKey(s core.Settings) (key, salt []byte, err error) {
	if s.BackupKey == "" {
		return nil, nil, nil
	}
	key, err = base64.StdEncoding.DecodeString(revealSecret(s.BackupKey))
	if err == nil {
		salt, err = base64.StdEncoding.DecodeString(s.BackupSalt)
	}
	if err != nil || len(key) != 32 || len(salt) != 16 {
		return nil, nil, errors.New("The backup password can't be read on this PC or Windows account; set it again in Settings.")
	}
	return key, salt, nil
}

func (a *App) apiBackupNow() error {
	a.mu.Lock()
	dir := a.cfg.Settings.BackupDir
	a.mu.Unlock()
	if dir == "" {
		return errors.New("Choose a folder for backups first.")
	}
	go a.runBackup()
	return nil
}

// queueBackup makes a backup after a session, when they're turned on.
func (a *App) queueBackup() {
	a.mu.Lock()
	on := a.cfg.Settings.BackupAuto && a.cfg.Settings.BackupDir != ""
	a.mu.Unlock()
	if on {
		go a.runBackup()
	}
}

// backupIfDue makes a backup at start when the last one is over a day old.
func (a *App) backupIfDue() {
	if st := loadBackupStatus(); time.Since(st.Last) > 24*time.Hour {
		a.queueBackup()
	}
}

// runBackup makes one backup now; one asked for while another runs is made
// right after it.
func (a *App) runBackup() {
	backupMu.Lock()
	if backupRunning {
		backupAgain = true
		backupMu.Unlock()
		return
	}
	backupRunning = true
	backupMu.Unlock()
	a.emit("backup", map[string]any{"running": true})
	for {
		st := loadBackupStatus()
		file, size, err := a.writeBackup()
		if err != nil {
			st.Error, st.ErrorAt = err.Error(), time.Now().UTC()
			a.logf("ERROR backup: %v", err)
		} else {
			st = BackupStatus{Last: time.Now().UTC(), File: file, Size: size}
			a.logf("Backed up to %s (%d bytes)", file, size)
		}
		saveBackupStatus(st)
		backupMu.Lock()
		again := backupAgain
		backupAgain = false
		if !again {
			backupRunning = false
		}
		backupMu.Unlock()
		if !again {
			out := map[string]any{"done": true}
			if err != nil {
				out["error"] = err.Error()
			}
			a.emit("backup", out)
			return
		}
	}
}

// liveSources are the save files in place now, with their owner, for every
// game that isn't running (a running game's files may be half-written; its
// profile copy is from the end of its last session).
func (a *App) liveSources(cfg *core.Config) []core.LiveSource {
	active := a.store.LoadState()
	var out []core.LiveSource
	for _, g := range cfg.Games {
		owner := active[g.ID]
		if owner == "" || cfg.Player(owner) == nil || a.activeSessionFor(g.ID) != nil {
			continue
		}
		for _, p := range g.Paths {
			live, err := a.env.Resolve(p.Path)
			if err != nil {
				continue
			}
			out = append(out, core.LiveSource{LiveEntry: core.LiveEntry{GameID: g.ID, EntryID: p.ID, Owner: owner}, Path: live})
		}
	}
	return out
}

func (a *App) manifest(cfg *core.Config) core.BackupManifest {
	m := core.BackupManifest{App: Version, Computer: computerName(), Created: time.Now().UTC(), Games: len(cfg.Games)}
	for _, p := range cfg.Players {
		m.Players = append(m.Players, p.Name)
	}
	return m
}

func (a *App) writeBackup() (string, int64, error) {
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	s := cfg.Settings
	if s.BackupDir == "" {
		return "", 0, errors.New("Choose a folder for backups first.")
	}
	if st, err := os.Stat(s.BackupDir); err != nil || !st.IsDir() {
		return "", 0, fmt.Errorf("The backup folder (%s) isn't there. Is the drive connected, and the cloud app running?", s.BackupDir)
	}
	key, salt, err := backupKey(s)
	if err != nil {
		return "", 0, err
	}
	// Like cloud saves, the folder decides whether backups are encrypted.
	switch {
	case !core.CloudHasKey(s.BackupDir):
		key, salt = nil, nil
	case key == nil || !core.CloudKeyMatches(s.BackupDir, key):
		return "", 0, errCloudPassword
	}
	target := backupTarget(s.BackupDir)
	if err := os.MkdirAll(target, 0o755); err != nil {
		return "", 0, fmt.Errorf("couldn't make %s: %w", target, err)
	}
	name := core.BackupFileName(computerName(), time.Now(), key != nil)
	path := filepath.Join(target, name)
	size, err := core.WriteBackupFile(path, DataDir(), a.liveSources(cfg), a.manifest(cfg),
		core.BackupOptions{Key: key, Salt: salt, Iters: core.BackupKDFIters})
	if err != nil {
		return "", 0, fmt.Errorf("couldn't write the backup: %w", err)
	}
	keep := s.BackupKeep
	if keep == 0 {
		keep = 5
	}
	core.PruneBackups(target, computerName(), keep)
	return path, size, nil
}

// --------------------------------------------------------------- restore ---

type RestoreInfo struct {
	Kind         string    `json:"kind,omitempty"` // "" everything, "player" one profile
	NeedPassword bool      `json:"needPassword,omitempty"`
	Error        string    `json:"error,omitempty"`
	File         string    `json:"file"`
	Computer     string    `json:"computer,omitempty"`
	Created      time.Time `json:"created,omitempty"`
	App          string    `json:"app,omitempty"`
	Players      []string  `json:"players,omitempty"`
	Games        int       `json:"games,omitempty"`
}

// apiRestoreOpen opens a backup file in the background (it may need
// decrypting and is unpacked to check it) and sends a "restoreInfo" event
// with what's inside, or that it needs a password.
func (a *App) apiRestoreOpen(file, password string) error {
	if len(password) > 1024 {
		return errors.New("That password is too long.")
	}
	go func() {
		info := RestoreInfo{File: file}
		defer func() {
			a.emit("restoreInfo", map[string]any{"info": info})
		}()
		restoreMu.Lock()
		defer restoreMu.Unlock()
		dropRestore()
		a.mu.Lock()
		s := a.cfg.Settings
		a.mu.Unlock()
		key, _, _ := backupKey(s)
		tmp := filepath.Join(DataDir(), "restore-"+time.Now().Format("20060102-150405"))
		ob, err := core.OpenBackup(file, tmp, password, key)
		if errors.Is(err, core.ErrBadPassword) && password == "" && key != nil {
			err = core.ErrNeedPassword // made with another password
		}
		if errors.Is(err, core.ErrNeedPassword) {
			info.NeedPassword = true
			return
		}
		if err != nil {
			info.Error = err.Error()
			return
		}
		if _, err := ob.Config(); err != nil {
			_ = os.RemoveAll(tmp)
			info.Error = err.Error()
			return
		}
		restoreOpen, restoreTmp = ob, tmp
		m := ob.Manifest
		info.Kind, info.Computer, info.Created, info.App, info.Players, info.Games = m.Kind, m.Computer, m.Created, m.App, m.Players, m.Games
	}()
	return nil
}

func dropRestore() {
	if restoreTmp != "" {
		_ = os.RemoveAll(restoreTmp)
	}
	restoreOpen, restoreTmp = nil, ""
}

func (a *App) apiRestoreCancel() {
	restoreMu.Lock()
	dropRestore()
	restoreMu.Unlock()
}

// apiRestoreApply restores the opened backup. Everything as it is now is
// first saved to backups\before-restore, so a restore can be undone by
// restoring that file. A "restored" event says when it's done.
func (a *App) apiRestoreApply() error { return a.applyOpened(false) }

// apiImportApply adds the opened profile file's player (or updates them),
// after the same backup of everything; a "restored" event says when.
func (a *App) apiImportApply() error { return a.applyOpened(true) }

func (a *App) applyOpened(player bool) error {
	a.mu.Lock()
	busy := len(a.sessions) > 0
	a.mu.Unlock()
	if busy {
		return errors.New("Close the game that's running first.")
	}
	go func() {
		out := map[string]any{}
		defer func() { a.emit("restored", out) }()
		restoreMu.Lock()
		defer restoreMu.Unlock()
		if restoreOpen == nil {
			out["error"] = "Open the backup again."
			return
		}
		a.mu.Lock()
		cfg := a.cfg.Clone()
		a.mu.Unlock()
		safetyDir := filepath.Join(a.store.BackupsDir(), "before-restore")
		if err := os.MkdirAll(safetyDir, 0o755); err != nil {
			out["error"] = err.Error()
			return
		}
		safety := filepath.Join(safetyDir, core.BackupFileName(computerName(), time.Now(), false))
		if _, err := core.WriteBackupFile(safety, DataDir(), a.liveSources(cfg), a.manifest(cfg), core.BackupOptions{}); err != nil {
			out["error"] = "Nothing was restored: Lootnest couldn't save what's here now first (" + err.Error() + ")."
			return
		}
		var newCfg *core.Config
		var err error
		if player {
			var res core.ImportResult
			newCfg, res, err = a.store.ImportPlayer(restoreOpen, cfg)
			out["import"] = res
		} else {
			newCfg, err = a.store.Restore(restoreOpen, cfg)
		}
		if err != nil {
			a.logf("ERROR restore: %v (what was here before is in %s)", err, safety)
			out["error"] = fmt.Sprintf("The restore stopped: %v. What was here before is saved in %s.", err, safety)
			return
		}
		a.mu.Lock()
		a.cfg = newCfg
		a.firstRun = false
		a.mu.Unlock()
		a.logf("Put in place a backup (%s) from %s (%s); what was here before is in %s", map[bool]string{true: "one player", false: "everything"}[player], restoreOpen.Manifest.Computer, restoreOpen.Manifest.Created, safety)
		out["ok"], out["safety"] = true, safety
		dropRestore()
		a.emitSessions()
	}()
	return nil
}

// cleanRestoreLeftovers removes unpacked backups left by a crash.
func cleanRestoreLeftovers() {
	ents, _ := os.ReadDir(DataDir())
	for _, e := range ents {
		if e.IsDir() && strings.HasPrefix(e.Name(), "restore-") {
			_ = os.RemoveAll(filepath.Join(DataDir(), e.Name()))
		}
	}
}

// openBackupFolder shows the backups in Explorer.
func (a *App) openBackupFolder() error {
	a.mu.Lock()
	dir := a.cfg.Settings.BackupDir
	a.mu.Unlock()
	if dir == "" {
		return errors.New("Choose a folder for backups first.")
	}
	t := backupTarget(dir)
	if _, err := os.Stat(t); err != nil {
		t = dir
	}
	return win.ShellOpen(t)
}

// apiExportPlayer writes a player's profile to a file in dir (with a
// password, encrypted). An "exported" event says when it's done.
func (a *App) apiExportPlayer(playerID, dir, password string) error {
	if password != "" && utf8.RuneCountInString(password) < 8 {
		return errors.New("Use at least 8 characters, or no password.")
	}
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return errors.New("Choose a folder to save the profile in.")
	}
	a.mu.Lock()
	cfg := a.cfg.Clone()
	a.mu.Unlock()
	p := cfg.Player(playerID)
	if p == nil {
		return errors.New("That player doesn't exist.")
	}
	go func() {
		out := map[string]any{}
		defer func() { a.emit("exported", out) }()
		opt := core.BackupOptions{}
		if password != "" {
			salt := core.NewBackupSalt()
			key, err := core.BackupKey(password, salt, core.BackupKDFIters)
			if err != nil {
				out["error"] = err.Error()
				return
			}
			opt = core.BackupOptions{Key: key, Salt: salt, Iters: core.BackupKDFIters}
		}
		path := filepath.Join(dir, core.PlayerFileName(p.Name, time.Now(), password != ""))
		if _, err := a.store.WritePlayerFile(path, cfg, playerID, a.liveSources(cfg), a.manifest(cfg), opt); err != nil {
			out["error"] = "Couldn't save the profile: " + err.Error()
			return
		}
		a.logf("Exported %s's profile to %s", p.Name, path)
		out["file"] = path
	}()
	return nil
}
