package core

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Backups of everything Lootnest keeps, as one file in a folder the user
// picks (usually one OneDrive, Google Drive or Dropbox syncs). Lootnest
// itself never uploads anything: the file is written to that folder and the
// user's own sync app does the rest.
//
// A backup is a zip:
//
//	lootnest-backup.json            what's inside (BackupManifest)
//	config.json shelf.json ...      Lootnest's own files
//	profiles/<player>/<game>/...    every player's saves and settings
//	avatars/ banners/               profile pictures
//	live/<game>/<path id>/...       the files in place right now, whose
//	                                owner the manifest names
//
// With a password, the zip is encrypted (see encryptBackup): a file that
// starts with "LNBK". Without, it's a plain .zip anyone can open.
// ---------------------------------------------------------------------------

const (
	backupFormat   = 1
	manifestName   = "lootnest-backup.json"
	backupMagic    = "LNBK"
	backupChunk    = 1 << 20
	backupSaltLen  = 16
	backupPrefix   = 7
	BackupKDFIters = 600_000
	// Limits for reading a backup, so a damaged or hostile file can't fill
	// the disk or run for ever.
	maxBackupEntries = 500_000
	maxBackupBytes   = 200 << 30
)

// Files and folders of Lootnest's data that a backup holds.
var (
	backupFiles = []string{"config.json", "shelf.json", "activity.json", "playtime.json", "recent.json"}
	backupDirs  = []string{"profiles", "avatars", "banners"}
)

// ErrBadPassword: the password doesn't open the backup (or it's damaged).
var ErrBadPassword = errors.New("That password doesn't open this backup, or the file is damaged.")

// ErrNeedPassword: the backup is encrypted and no password was given.
var ErrNeedPassword = errors.New("need-password")

type BackupManifest struct {
	Format int `json:"format"`
	// Kind is "" for a backup of everything, "player" for one player's
	// profile (ExportPlayer).
	Kind     string      `json:"kind,omitempty"`
	App      string      `json:"app"` // Lootnest version that made it
	Computer string      `json:"computer"`
	Created  time.Time   `json:"created"`
	Players  []string    `json:"players"`
	Games    int         `json:"games"`
	Live     []LiveEntry `json:"live"`
}

// LiveEntry is one save location copied as it was in place: whose files
// they were, for which game and path.
type LiveEntry struct {
	GameID  string `json:"gameId"`
	EntryID string `json:"entryId"`
	Owner   string `json:"owner"`
}

// LiveSource is a save location to copy into a backup.
type LiveSource struct {
	LiveEntry
	Path string
}

// WriteBackup writes a zip of dataDir's Lootnest files and the live sources
// to w.
func WriteBackup(w io.Writer, dataDir string, live []LiveSource, m BackupManifest) error {
	zw := zip.NewWriter(w)
	m.Format = backupFormat
	m.Live = nil
	for _, f := range backupFiles {
		if err := zipPath(zw, filepath.Join(dataDir, f), f); err != nil {
			return err
		}
	}
	for _, d := range backupDirs {
		if err := zipPath(zw, filepath.Join(dataDir, d), d); err != nil {
			return err
		}
	}
	for _, l := range live {
		if !idRe.MatchString(l.GameID) || !idRe.MatchString(l.EntryID) || !idRe.MatchString(l.Owner) || !exists(l.Path) {
			continue
		}
		src := l.Path
		if real, err := filepath.EvalSymlinks(src); err == nil {
			src = real // a save folder may be a link to another drive
		}
		if err := zipPath(zw, src, "live/"+l.GameID+"/"+l.EntryID); err != nil {
			return err
		}
		m.Live = append(m.Live, l.LiveEntry)
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	f, err := zw.Create(manifestName)
	if err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		return err
	}
	return zw.Close()
}

// zipPath adds a file, or a folder's regular files, under name. Links are
// left out; a missing path is no error.
func zipPath(zw *zip.Writer, src, name string) error {
	st, err := os.Lstat(src)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if st.Mode().IsRegular() {
		return zipFile(zw, src, name, st)
	}
	if !st.IsDir() {
		return nil
	}
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return zipFile(zw, p, name+"/"+filepath.ToSlash(rel), info)
	})
}

func zipFile(zw *zip.Writer, src, name string, info fs.FileInfo) error {
	h, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	h.Name, h.Method = name, zip.Deflate
	w, err := zw.CreateHeader(h)
	if err != nil {
		return err
	}
	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("couldn't read %s: %w", src, err)
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}

// ------------------------------------------------------------ encryption ---

// BackupKey turns a password into the key for encrypted backups.
func BackupKey(password string, salt []byte, iters int) ([]byte, error) {
	if password == "" {
		return nil, errors.New("empty password")
	}
	return pbkdf2.Key(sha256.New, password, salt, iters, 32)
}

// NewBackupSalt makes the random salt a password key is made with.
func NewBackupSalt() []byte {
	b := make([]byte, backupSaltLen)
	_, _ = rand.Read(b)
	return b
}

// encryptBackup writes src to dst with AES-256-GCM in 1 MiB chunks:
//
//	"LNBK" | 1 | PBKDF2 iterations (uint32) | salt (16) | nonce prefix (7)
//	then each chunk sealed with nonce = prefix | chunk number (uint32) |
//	1 for the last chunk, else 0, and the header as additional data.
//
// The last-chunk flag means a cut-off file never decrypts as complete, and
// the chunk number that chunks can't be reordered.
func encryptBackup(dst io.Writer, src io.Reader, key, salt []byte, iters int) error {
	if len(salt) != backupSaltLen {
		return errors.New("bad salt")
	}
	aead, err := backupAEAD(key)
	if err != nil {
		return err
	}
	header := make([]byte, 0, 32)
	header = append(header, backupMagic...)
	header = append(header, 1)
	header = binary.BigEndian.AppendUint32(header, uint32(iters))
	header = append(header, salt...)
	prefix := make([]byte, backupPrefix)
	if _, err := rand.Read(prefix); err != nil {
		return err
	}
	header = append(header, prefix...)
	if _, err := dst.Write(header); err != nil {
		return err
	}
	br := bufio.NewReaderSize(src, backupChunk)
	buf := make([]byte, backupChunk)
	out := make([]byte, 0, backupChunk+aead.Overhead())
	for n := uint32(0); ; n++ {
		k, err := io.ReadFull(br, buf)
		if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
			return err
		}
		_, perr := br.Peek(1)
		last := perr != nil
		if n == ^uint32(0) && !last {
			return errors.New("backup too big")
		}
		out = aead.Seal(out[:0], chunkNonce(prefix, n, last), buf[:k], header)
		if _, err := dst.Write(out); err != nil {
			return err
		}
		if last {
			return nil
		}
	}
}

// decryptBackup reads what encryptBackup wrote. key gets the salt and the
// iterations from the header and returns the key (from a password, or the
// one saved for this PC).
func decryptBackup(dst io.Writer, src io.Reader, key func(salt []byte, iters int) ([]byte, error)) error {
	header := make([]byte, len(backupMagic)+1+4+backupSaltLen+backupPrefix)
	if _, err := io.ReadFull(src, header); err != nil || string(header[:4]) != backupMagic {
		return errors.New("This isn't a Lootnest backup.")
	}
	if header[4] != 1 {
		return errors.New("This backup was made by a newer Lootnest; update Lootnest to open it.")
	}
	iters := int(binary.BigEndian.Uint32(header[5:9]))
	if iters < 10_000 || iters > 50_000_000 {
		return errors.New("This backup is damaged.")
	}
	salt, prefix := header[9:9+backupSaltLen], header[9+backupSaltLen:]
	k, err := key(salt, iters)
	if err != nil {
		return err
	}
	aead, err := backupAEAD(k)
	if err != nil {
		return err
	}
	br := bufio.NewReaderSize(src, backupChunk+64)
	buf := make([]byte, backupChunk+aead.Overhead())
	var plain []byte
	for n := uint32(0); ; n++ {
		m, err := io.ReadFull(br, buf)
		if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
			if errors.Is(err, io.EOF) {
				return ErrBadPassword // ended before a last chunk
			}
			return err
		}
		_, perr := br.Peek(1)
		last := perr != nil
		plain, err = aead.Open(plain[:0], chunkNonce(prefix, n, last), buf[:m], header)
		if err != nil {
			return ErrBadPassword
		}
		if _, err := dst.Write(plain); err != nil {
			return err
		}
		if last {
			return nil
		}
		if n == ^uint32(0) {
			return ErrBadPassword
		}
	}
}

func backupAEAD(key []byte) (cipher.AEAD, error) {
	if len(key) != 32 {
		return nil, errors.New("bad key")
	}
	b, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(b)
}

func chunkNonce(prefix []byte, n uint32, last bool) []byte {
	nonce := make([]byte, 0, 12)
	nonce = append(nonce, prefix...)
	nonce = binary.BigEndian.AppendUint32(nonce, n)
	if last {
		return append(nonce, 1)
	}
	return append(nonce, 0)
}

// ------------------------------------------------------- writing a file ---

// BackupOptions say how to write a backup file.
type BackupOptions struct {
	// Key and Salt encrypt the backup (see BackupKey); without, it's a zip.
	Key, Salt []byte
	Iters     int
}

// WriteBackupFile writes a backup to path: first to path+".partial", then
// moved into place, so a sync app never uploads half a backup.
func WriteBackupFile(path, dataDir string, live []LiveSource, m BackupManifest, opt BackupOptions) (int64, error) {
	return writeArchiveFile(path, func(w io.Writer) error { return WriteBackup(w, dataDir, live, m) }, opt)
}

// writeArchiveFile writes what write makes to path, encrypted with opt's
// key if it has one, through path+".partial".
func writeArchiveFile(path string, write func(io.Writer) error, opt BackupOptions) (int64, error) {
	tmp := path + ".partial"
	f, err := os.Create(tmp)
	if err != nil {
		return 0, err
	}
	ok := false
	defer func() {
		if !ok {
			f.Close()
			os.Remove(tmp)
		}
	}()
	bw := bufio.NewWriterSize(f, 1<<20)
	if opt.Key == nil {
		if err := write(bw); err != nil {
			return 0, err
		}
	} else {
		pr, pw := io.Pipe()
		go func() { pw.CloseWithError(write(pw)) }()
		err := encryptBackup(bw, pr, opt.Key, opt.Salt, opt.Iters)
		pr.CloseWithError(err)
		if err != nil {
			return 0, err
		}
	}
	if err := bw.Flush(); err != nil {
		return 0, err
	}
	if err := f.Sync(); err != nil {
		return 0, err
	}
	st, _ := f.Stat()
	if err := f.Close(); err != nil {
		return 0, err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return 0, err
	}
	ok = true
	return st.Size(), nil
}

// IsEncryptedBackup says whether a backup file needs a password.
func IsEncryptedBackup(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	b := make([]byte, 4)
	if _, err := io.ReadFull(f, b); err != nil {
		return false, errors.New("This isn't a Lootnest backup.")
	}
	return string(b) == backupMagic, nil
}

// ------------------------------------------------------ reading a backup ---

// OpenedBackup is a backup unpacked into a folder, ready to restore.
type OpenedBackup struct {
	Dir      string
	Manifest BackupManifest
}

// OpenBackup checks a backup file and unpacks it into tmpDir (which must
// not exist yet). password opens an encrypted one; key (optional) is the
// key saved on this PC, tried first.
func OpenBackup(file, tmpDir, password string, key []byte) (*OpenedBackup, error) {
	enc, err := IsEncryptedBackup(file)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(tmpDir, 0o700); err != nil {
		return nil, err
	}
	fail := func(err error) (*OpenedBackup, error) {
		_ = os.RemoveAll(tmpDir)
		return nil, err
	}
	zipPath := file
	if enc {
		zipPath = filepath.Join(tmpDir, "backup.zip")
		out, err := os.Create(zipPath)
		if err != nil {
			return fail(err)
		}
		in, err := os.Open(file)
		if err != nil {
			out.Close()
			return fail(err)
		}
		bw := bufio.NewWriterSize(out, 1<<20)
		err = decryptBackup(bw, bufio.NewReader(in), func(salt []byte, iters int) ([]byte, error) {
			if key != nil && password == "" {
				return key, nil
			}
			if password == "" {
				return nil, ErrNeedPassword
			}
			return BackupKey(password, salt, iters)
		})
		in.Close()
		if err == nil {
			err = bw.Flush()
		}
		if cerr := out.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return fail(err)
		}
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return fail(errors.New("This isn't a Lootnest backup, or it's damaged."))
	}
	defer zr.Close()
	ob := &OpenedBackup{Dir: filepath.Join(tmpDir, "data")}
	if len(zr.File) > maxBackupEntries {
		return fail(errors.New("This backup has too many files."))
	}
	var total uint64
	found := false
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, "/") {
			continue
		}
		if !safeBackupName(f.Name) {
			return fail(fmt.Errorf("This backup has a file Lootnest won't restore (%q).", f.Name))
		}
		total += f.UncompressedSize64
		if total > maxBackupBytes {
			return fail(errors.New("This backup is too big to restore."))
		}
		if f.Name == manifestName {
			found = true
			rc, err := f.Open()
			if err != nil {
				return fail(err)
			}
			b, err := io.ReadAll(io.LimitReader(rc, 1<<20))
			rc.Close()
			if err != nil || json.Unmarshal(b, &ob.Manifest) != nil {
				return fail(errors.New("This backup is damaged."))
			}
			continue
		}
		if err := unzipFile(f, filepath.Join(ob.Dir, filepath.FromSlash(f.Name))); err != nil {
			return fail(err)
		}
	}
	if !found || ob.Manifest.Format < 1 {
		return fail(errors.New("This isn't a Lootnest backup."))
	}
	if ob.Manifest.Format > backupFormat {
		return fail(errors.New("This backup was made by a newer Lootnest; update Lootnest to open it."))
	}
	if enc {
		_ = os.Remove(zipPath)
	}
	return ob, nil
}

// safeBackupName accepts only the names a backup can hold: clean relative
// paths under the known files and folders, with valid ids.
func safeBackupName(name string) bool {
	if name == "" || strings.ContainsAny(name, "\\:\x00") || path.IsAbs(name) || path.Clean(name) != name || strings.HasPrefix(name, "../") || name == ".." {
		return false
	}
	parts := strings.Split(name, "/")
	for _, p := range parts {
		if !safeComponent(p) {
			return false
		}
	}
	if len(parts) == 1 {
		if name == manifestName {
			return true
		}
		for _, f := range backupFiles {
			if name == f {
				return true
			}
		}
		return false
	}
	switch parts[0] {
	case "profiles":
		return len(parts) >= 4 && idRe.MatchString(parts[1]) && idRe.MatchString(parts[2])
	case "live":
		return len(parts) >= 3 && idRe.MatchString(parts[1]) && idRe.MatchString(parts[2])
	case "avatars", "banners":
		return len(parts) == 2
	}
	return false
}

// safeComponent is one part of a file name inside a backup.
func safeComponent(p string) bool {
	return p != "" && p != "." && p != ".." && !strings.HasSuffix(p, ".") && !strings.HasSuffix(p, " ") && !windowsDevice(p)
}

// windowsDevice: names Windows reserves for devices (CON, NUL, COM1...),
// with or without an extension.
func windowsDevice(name string) bool {
	base := strings.ToUpper(strings.SplitN(name, ".", 2)[0])
	switch base {
	case "CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$":
		return true
	}
	if len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '0' && base[3] <= '9' {
		return true
	}
	return false
}

func unzipFile(f *zip.File, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return errors.New("This backup is damaged.")
	}
	defer rc.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	// Never more than the size the zip says (zip.Reader also checks it).
	_, err = io.Copy(out, io.LimitReader(rc, int64(f.UncompressedSize64)+1))
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return errors.New("This backup is damaged.")
	}
	_ = os.Chtimes(dst, f.Modified, f.Modified)
	return nil
}

// Config returns the backup's config.
func (ob *OpenedBackup) Config() (*Config, error) {
	b, err := os.ReadFile(filepath.Join(ob.Dir, "config.json"))
	if err != nil {
		return nil, errors.New("This backup has no Lootnest settings in it.")
	}
	cfg := DefaultConfig()
	cfg.Games, cfg.Players = nil, nil
	if err := json.Unmarshal(b, cfg); err != nil {
		return nil, errors.New("This backup's settings are damaged.")
	}
	cfg.normalize()
	if err := ValidatePlayers(cfg.Players); err != nil {
		return nil, fmt.Errorf("This backup's players can't be used: %w", err)
	}
	// Games are checked like ones added here; their folders are checked
	// again (Env.CheckSafe) whenever a swap resolves them.
	for i := range cfg.Games {
		if err := ValidateGame(&cfg.Games[i]); err != nil {
			return nil, fmt.Errorf("This backup's game %q can't be used: %w", cfg.Games[i].Name, err)
		}
	}
	return cfg, nil
}

// Restore puts an opened backup in place of the Store's data. The caller
// has made a backup of the current data first (it is replaced). This PC's
// own settings stay; players, games, saves, pictures, shelf and play time
// come from the backup. Every restored game then asks whose files are in
// place before its next swap, so the files on this PC are never saved over
// anyone's profile by mistake.
func (s *Store) Restore(ob *OpenedBackup, current *Config) (*Config, error) {
	if ob.Manifest.Kind == "player" {
		return nil, errors.New("This is one player's profile, not a backup of everything; use Import a profile on Players.")
	}
	cfg, err := ob.Config()
	if err != nil {
		return nil, err
	}
	cfg.Settings = current.Settings
	cfg.Shortcuts = current.Shortcuts
	cfg.SteamRestores = current.SteamRestores
	// Saves: each player's profiles, then the files that were in place,
	// which go to their owner's profile (they're newer than it).
	if ents, err := os.ReadDir(filepath.Join(ob.Dir, "profiles")); err == nil {
		for _, p := range ents {
			games, _ := os.ReadDir(filepath.Join(ob.Dir, "profiles", p.Name()))
			for _, g := range games {
				if err := Mirror(filepath.Join(ob.Dir, "profiles", p.Name(), g.Name()), s.ProfileDir(p.Name(), g.Name())); err != nil {
					return nil, err
				}
			}
		}
	}
	for _, l := range ob.Manifest.Live {
		if !idRe.MatchString(l.GameID) || !idRe.MatchString(l.EntryID) || cfg.Player(l.Owner) == nil {
			continue
		}
		src := filepath.Join(ob.Dir, "live", l.GameID, l.EntryID)
		if !exists(src) {
			continue
		}
		dir := s.ProfileDir(l.Owner, l.GameID)
		if err := Mirror(src, filepath.Join(dir, l.EntryID)); err != nil {
			return nil, err
		}
		_ = os.WriteFile(filepath.Join(dir, initMarker), []byte(ob.Manifest.Created.Format(time.RFC3339)), 0o644)
	}
	for _, d := range []string{"avatars", "banners"} {
		if src := filepath.Join(ob.Dir, d); exists(src) {
			if err := Mirror(src, filepath.Join(s.Dir, d)); err != nil {
				return nil, err
			}
		}
	}
	if err := s.restoreFiles(ob); err != nil {
		return nil, err
	}
	// Nobody's files are known to be in place now.
	for _, g := range cfg.Games {
		if err := s.SetActive(g.ID, ""); err != nil {
			return nil, err
		}
	}
	if err := s.SaveConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// restoreFiles puts the backup's shelf, activity, play time and recent
// games in place.
func (s *Store) restoreFiles(ob *OpenedBackup) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, f := range backupFiles[1:] {
		b, err := os.ReadFile(filepath.Join(ob.Dir, f))
		if err != nil || !json.Valid(b) {
			continue
		}
		if err := writeFileAtomic(filepath.Join(s.Dir, f), b); err != nil {
			return err
		}
	}
	return nil
}

// BackupFileName is the name of a backup made now on this computer.
func BackupFileName(computer string, t time.Time, encrypted bool) string {
	ext := ".zip"
	if encrypted {
		ext = ".lnbak"
	}
	return "Lootnest " + safeFileName(computer) + " " + t.Format("2006-01-02 150405") + ext
}

// PruneBackups keeps the newest keep backups this computer made in dir;
// other computers' backups are left alone.
func PruneBackups(dir, computer string, keep int) {
	if keep < 1 {
		keep = 1
	}
	prefix := "Lootnest " + safeFileName(computer) + " "
	ents, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var mine []string
	for _, e := range ents {
		n := e.Name()
		if e.Type().IsRegular() && strings.HasPrefix(n, prefix) && (strings.HasSuffix(n, ".zip") || strings.HasSuffix(n, ".lnbak")) {
			mine = append(mine, n)
		}
	}
	// Newest first: the date and time are at the end of the name.
	sort.Slice(mine, func(i, j int) bool { return stamp(mine[i]) > stamp(mine[j]) })
	for _, n := range mine[min(keep, len(mine)):] {
		_ = os.Remove(filepath.Join(dir, n))
	}
}

func stamp(name string) string {
	name = strings.TrimSuffix(strings.TrimSuffix(name, ".zip"), ".lnbak")
	if len(name) < 17 {
		return name
	}
	return name[len(name)-17:]
}

func safeFileName(s string) string {
	var b bytes.Buffer
	for _, r := range strings.TrimSpace(s) {
		if r < 32 || strings.ContainsRune(`<>:"/\|?*`, r) {
			r = '-'
		}
		b.WriteRune(r)
	}
	if b.Len() == 0 {
		return "PC"
	}
	return b.String()
}
