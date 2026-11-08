package core

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// itch.io: the games in the user's library there, through itch.io's own
// API with the API key they made. Only their library, its uploads and the file of
// an upload they choose; nothing else is asked for. The key goes in the
// Authorization header (and, as itch.io requires, in the download link),
// never in an error or the log.
// ---------------------------------------------------------------------------

// ItchAPI is itch.io's API; tests point it at a local server.
var ItchAPI = "https://api.itch.io"

// ItchTrace, when set, is told the host and path of each request: never
// its query, which can hold the key.
var ItchTrace func(host, path string)

var (
	itchTransport http.RoundTripper = http.DefaultTransport
	itchPause                       = 400 * time.Millisecond // between pages
	itchMu        sync.Mutex                                 // one API request at a time
)

const itchPages = 40

var (
	errItchKey   = errors.New("itch.io didn't accept the API key. Make a new one in itch.io's settings, under API keys, and connect again.")
	errItchBusy  = errors.New("itch.io has had too many requests. Try again in a minute.")
	errItchReply = errors.New("itch.io sent a reply Lootnest can't read")
)

// ItchGame is a game the user owns on itch.io, by the download key that
// says so.
type ItchGame struct {
	KeyID          int64  `json:"keyId"`
	GameID         int64  `json:"gameId"`
	Title          string `json:"title"`
	URL            string `json:"url,omitempty"`
	CoverURL       string `json:"coverUrl,omitempty"`
	ShortText      string `json:"shortText,omitempty"`
	Classification string `json:"classification,omitempty"`
}

// ItchUpload is one file a game has on itch.io.
type ItchUpload struct {
	ID          int64     `json:"id"`
	Filename    string    `json:"filename"`
	DisplayName string    `json:"displayName,omitempty"`
	Size        int64     `json:"size"`
	Type        string    `json:"type"`
	Windows     bool      `json:"windows"`
	Demo        bool      `json:"demo,omitempty"`
	Preorder    bool      `json:"preorder,omitempty"`
	BuildID     int64     `json:"buildId,omitempty"`
	UserVersion string    `json:"userVersion,omitempty"`
	MD5         string    `json:"md5,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Kind        string    `json:"kind,omitempty"` // ItchKind, set by ItchPickUpload
}

func itchClient(timeout time.Duration) *http.Client {
	return &http.Client{Transport: itchTransport, Timeout: timeout, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if r.URL.Scheme != "https" {
			return errors.New("itch.io sent the file to an address that isn't https")
		}
		if len(via) >= 10 {
			return errors.New("too many redirects")
		}
		itchTrace(r.URL)
		return nil
	}}
}

func itchTrace(u *url.URL) {
	if ItchTrace != nil {
		ItchTrace(u.Host, u.Path)
	}
}

// itchNetErr drops the address from a request's error: it can hold the key.
func itchNetErr(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		err = ue.Err
	}
	if errors.Is(err, context.Canceled) {
		return err
	}
	return fmt.Errorf("couldn't reach itch.io (%w)", err)
}

func itchStatus(code int, status string) error {
	switch {
	case code == http.StatusUnauthorized || code == http.StatusForbidden:
		return errItchKey
	case code == http.StatusTooManyRequests:
		return errItchBusy
	case code < 200 || code > 299:
		return fmt.Errorf("itch.io answered %s", status)
	}
	return nil
}

func itchGet(ctx context.Context, key, path string) (itchObj, error) {
	itchMu.Lock()
	defer itchMu.Unlock()
	req, err := http.NewRequestWithContext(ctx, "GET", ItchAPI+path, nil)
	if err != nil {
		return nil, errItchReply
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Accept", "application/json")
	itchTrace(req.URL)
	resp, err := itchClient(30 * time.Second).Do(req)
	if err != nil {
		return nil, itchNetErr(err)
	}
	defer resp.Body.Close()
	if err := itchStatus(resp.StatusCode, resp.Status); err != nil {
		return nil, err
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, itchNetErr(err)
	}
	o, err := itchDecode(b)
	if err != nil {
		return nil, err
	}
	if errs := o.strs("errors"); len(errs) > 0 {
		e := strings.ToLower(errs[0])
		if strings.Contains(e, "key") || strings.Contains(e, "auth") || strings.Contains(e, "login") {
			return nil, errItchKey
		}
		return nil, fmt.Errorf("itch.io said: %s", strings.ReplaceAll(errs[0], key, ""))
	}
	return o, nil
}

func itchWait(ctx context.Context) error {
	t := time.NewTimer(itchPause)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// ItchProfile checks the key and returns the account's user name.
func ItchProfile(ctx context.Context, key string) (string, error) {
	o, err := itchGet(ctx, key, "/profile")
	if err != nil {
		return "", err
	}
	u := o.obj("user")
	name := u.str("username")
	if name == "" {
		name = u.str("displayname")
	}
	if name == "" {
		return "", errItchReply
	}
	return name, nil
}

// ItchOwned lists the games the account owns, one per game, leaving out
// tools, assets, books and the like.
func ItchOwned(ctx context.Context, key string) ([]ItchGame, error) {
	out := []ItchGame{}
	seen := map[int64]bool{}
	for page := 1; page <= itchPages; page++ {
		if page > 1 {
			if err := itchWait(ctx); err != nil {
				return nil, err
			}
		}
		o, err := itchGet(ctx, key, fmt.Sprintf("/profile/owned-keys?page=%d&per_page=50", page))
		if err != nil {
			return nil, err
		}
		keys := o.list("ownedkeys")
		for _, k := range keys {
			g := k.obj("game")
			id := k.num("gameid")
			if id == 0 {
				id = g.num("id")
			}
			class := strings.ToLower(g.str("classification"))
			if id == 0 || seen[id] || (class != "" && class != "game") {
				continue
			}
			seen[id] = true
			out = append(out, ItchGame{KeyID: k.num("id"), GameID: id, Title: strings.TrimSpace(g.str("title")), URL: g.str("url"),
				CoverURL: g.str("coverurl"), ShortText: g.str("shorttext"), Classification: class})
		}
		per := o.num("perpage")
		if per <= 0 {
			per = 50
		}
		if int64(len(keys)) < per {
			break
		}
	}
	return out, nil
}

// ItchUploads lists a game's files, as the owner of keyID sees them.
func ItchUploads(ctx context.Context, key string, gameID, keyID int64) ([]ItchUpload, error) {
	path := fmt.Sprintf("/games/%d/uploads", gameID)
	if keyID > 0 {
		path += fmt.Sprintf("?download_key_id=%d", keyID)
	}
	o, err := itchGet(ctx, key, path)
	if err != nil {
		return nil, err
	}
	out := []ItchUpload{}
	for _, u := range o.list("uploads") {
		b := u.obj("build")
		up := ItchUpload{ID: u.num("id"), Filename: u.str("filename"), DisplayName: u.str("displayname"), Size: u.num("size"),
			Type: strings.ToLower(u.str("type")), Windows: u.flag("pwindows"), Demo: u.flag("demo"), Preorder: u.flag("preorder"),
			BuildID: u.num("buildid"), UserVersion: b.str("userversion"), MD5: u.str("md5hash"), UpdatedAt: u.when("updatedat")}
		if up.BuildID == 0 {
			up.BuildID = b.num("id")
		}
		if up.UserVersion == "" {
			up.UserVersion = b.str("version")
		}
		if up.Type == "" {
			up.Type = "default"
		}
		for _, t := range u.strs("traits") {
			switch itchNorm(t) {
			case "pwindows":
				up.Windows = true
			case "demo":
				up.Demo = true
			case "preorder":
				up.Preorder = true
			}
		}
		if p := u.obj("platforms"); p["windows"] != nil && (p.flag("windows") || (p.str("windows") != "" && p.str("windows") != "false")) {
			up.Windows = true
		}
		if up.ID > 0 {
			out = append(out, up)
		}
	}
	return out, nil
}

var (
	bits64 = regexp.MustCompile(`(?i)(x64|x86[-_]?64|win64|amd64|64[-_ ]?bit)`)
	bits32 = regexp.MustCompile(`(?i)(x86|win32|i[36]86|32[-_ ]?bit)`)
)

func itchBits(u ItchUpload) int {
	n := u.Filename + " " + u.DisplayName
	switch {
	case bits64.MatchString(n):
		return 0
	case bits32.MatchString(n):
		return 2
	}
	return 1
}

// ItchPickUpload is the uploads Lootnest can install, best first: for
// Windows, the game itself (no soundtracks or pre-orders), the full game
// before a demo, 64-bit before 32-bit, then the newest.
func ItchPickUpload(ups []ItchUpload) []ItchUpload {
	out := []ItchUpload{}
	for _, u := range ups {
		if u.Kind = ItchKind(u); u.Windows && u.Type == "default" && !u.Preorder && u.Kind != "" {
			out = append(out, u)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Demo != b.Demo {
			return !a.Demo
		}
		if x, y := itchBits(a), itchBits(b); x != y {
			return x < y
		}
		return a.UpdatedAt.After(b.UpdatedAt)
	})
	return out
}

var itchSetup = regexp.MustCompile(`(?i)setup|install`)

// ItchKind says how an upload is installed: "archive" (.zip, .7z, .rar),
// "installer" (an .msi, or an .exe named as a setup), "exe" (an .exe that
// is the game itself, as many itch.io games are) or "" when Lootnest can't.
func ItchKind(u ItchUpload) string {
	switch strings.ToLower(filepath.Ext(u.Filename)) {
	case ".zip", ".7z", ".rar":
		return "archive"
	case ".msi":
		return "installer"
	case ".exe":
		if itchSetup.MatchString(u.Filename + " " + u.DisplayName) {
			return "installer"
		}
		return "exe"
	}
	return ""
}

// ItchDownloadURL is the link to an upload's file. itch.io answers it with
// a redirect to its download servers, where only the key in the link counts.
func ItchDownloadURL(key string, uploadID, keyID int64) string {
	q := url.Values{"api_key": {key}}
	if keyID > 0 {
		q.Set("download_key_id", strconv.FormatInt(keyID, 10))
	}
	return fmt.Sprintf("%s/uploads/%d/download?%s", ItchAPI, uploadID, q.Encode())
}

// ItchNewer says whether up is newer than what was installed: another
// upload, a higher build, or a later one whose file is different.
func ItchNewer(rec ItchRecord, up ItchUpload) bool {
	if up.ID != rec.UploadID || up.BuildID > rec.BuildID {
		return true
	}
	return up.UpdatedAt.After(rec.UpdatedAt) && (up.Size != rec.Size || up.Filename != rec.Filename || !strings.EqualFold(up.MD5, rec.MD5))
}

// ItchDownload saves the file at u to dst, through dst+".part": only over
// https, checked against size and md5 when they're known. A download that
// fails or is cancelled leaves nothing behind.
func ItchDownload(ctx context.Context, u, dst string, size int64, sum string, progress func(done, total int64)) (err error) {
	pu, err := url.Parse(u)
	if err != nil || pu.Scheme != "https" {
		return errors.New("Lootnest only downloads over https")
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	part := dst + ".part"
	defer func() {
		if err != nil {
			_ = os.Remove(part)
		}
	}()
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return errItchReply
	}
	itchTrace(pu)
	resp, err := itchClient(0).Do(req)
	if err != nil {
		return itchNetErr(err)
	}
	defer resp.Body.Close()
	if err := itchStatus(resp.StatusCode, resp.Status); err != nil {
		return err
	}
	total := resp.ContentLength
	var body io.Reader = resp.Body
	if size > 0 {
		total = size
		body = io.LimitReader(body, size+1)
	}
	f, err := os.Create(part)
	if err != nil {
		return err
	}
	h := md5.New()
	w := io.MultiWriter(f, h)
	var done int64
	buf := make([]byte, 1<<20)
	for {
		n, rerr := body.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				rerr = werr
			}
			done += int64(n)
			if progress != nil {
				progress(done, total)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			f.Close()
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return itchNetErr(rerr)
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	if size > 0 && done != size {
		return errors.New("the download from itch.io isn't the size itch.io lists. Try again")
	}
	if sum != "" && !strings.EqualFold(hex.EncodeToString(h.Sum(nil)), sum) {
		return errors.New("the download from itch.io doesn't match its checksum. Try again")
	}
	return os.Rename(part, dst)
}

// ItchDownloadPath is where an upload is downloaded to: Lootnest's own
// downloads\itch\<game id> folder, under a name safe on Windows.
func ItchDownloadPath(dataDir string, gameID int64, filename string) string {
	name := filepath.Base(strings.ReplaceAll(filename, `\`, "/"))
	name = strings.TrimLeft(safeFileName(name), ". ")
	if name == "" || windowsDevice(name) {
		name = "download" + filepath.Ext(name)
	}
	return filepath.Join(dataDir, "downloads", "itch", strconv.FormatInt(gameID, 10), name)
}

// ItchFolderName is a game's title as a folder name Windows takes.
func ItchFolderName(title string) string {
	r := []rune(safeFileName(title))
	if len(r) > 80 {
		r = r[:80]
	}
	name := strings.Trim(string(r), ". ")
	if name == "" || strings.TrimSpace(title) == "" {
		return "Game"
	}
	if windowsDevice(name) {
		name += " game"
	}
	return name
}

// ItchInstallArchive extracts a downloaded .zip, .7z or .rar (or copies a
// game's own .exe) into parent\<the game's name>, without the one folder an
// archive often wraps everything in. It never writes into a folder that has anything in it:
// the files go to a folder of Lootnest's own beside it first, then move
// in at the end.
func ItchInstallArchive(archive, parent, name string) (string, error) {
	dir := filepath.Join(parent, ItchFolderName(name))
	if st, err := os.Lstat(dir); err == nil {
		if es, _ := os.ReadDir(dir); !st.IsDir() || len(es) > 0 {
			return "", fmt.Errorf("%s already has files in it. Choose another folder for the game, or move them first", dir)
		}
	}
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", err
	}
	h := sha1.Sum([]byte(strings.ToLower(dir)))
	tmp := filepath.Join(parent, ".lootnest-install-"+hex.EncodeToString(h[:4]))
	defer os.RemoveAll(tmp)
	if strings.EqualFold(filepath.Ext(archive), ".exe") {
		info, err := os.Stat(archive)
		if err != nil {
			return "", err
		}
		_ = os.RemoveAll(tmp) // only ever Lootnest's own, from an install cut short
		if err := copyFile(archive, filepath.Join(tmp, filepath.Base(archive)), info); err != nil {
			return "", err
		}
	} else if _, err := StageUpdate(archive, tmp); err != nil {
		return "", err
	}
	root := importRoot(tmp, "")
	if es, _ := os.ReadDir(root); len(es) == 0 {
		return "", errors.New("the download has no files in it")
	}
	_ = os.Remove(dir) // only ever an empty folder
	if err := os.Rename(root, dir); err != nil {
		return "", err
	}
	return dir, nil
}

// --------------------------------------------------------- the record ---

// ItchRecord is what Lootnest installed from itch.io for a Library game.
type ItchRecord struct {
	GameID      int64       `json:"gameId"`
	KeyID       int64       `json:"keyId"`
	UploadID    int64       `json:"uploadId"`
	BuildID     int64       `json:"buildId,omitempty"`
	UserVersion string      `json:"userVersion,omitempty"`
	Filename    string      `json:"filename,omitempty"`
	Size        int64       `json:"size,omitempty"`
	MD5         string      `json:"md5,omitempty"`
	UpdatedAt   time.Time   `json:"updatedAt"`
	InstalledAt time.Time   `json:"installedAt"`
	CheckedAt   time.Time   `json:"checkedAt"`
	UpdateID    string      `json:"updateId,omitempty"` // the applied update that brought this upload in
	Prev        *ItchRecord `json:"prev,omitempty"`     // the record before it, back on Undo
}

// ItchRecordOf is the record for an upload installed at now.
func ItchRecordOf(g ItchGame, up ItchUpload, now time.Time) ItchRecord {
	return ItchRecord{GameID: g.GameID, KeyID: g.KeyID, UploadID: up.ID, BuildID: up.BuildID, UserVersion: up.UserVersion, Filename: up.Filename,
		Size: up.Size, MD5: up.MD5, UpdatedAt: up.UpdatedAt, InstalledAt: now.UTC(), CheckedAt: now.UTC()}
}

func (s *Store) ItchPath() string { return filepath.Join(s.Dir, "itch.json") }

// ItchRecords is every record, by Library key.
func (s *Store) ItchRecords() map[string]ItchRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]ItchRecord{}
	if b, err := os.ReadFile(s.ItchPath()); err == nil {
		_ = json.Unmarshal(b, &out)
	}
	if out == nil {
		out = map[string]ItchRecord{}
	}
	return out
}

// readItchToChange reads itch.json before a write. A file that doesn't
// parse is kept as itch.json.broken, so the write never replaces it.
func (s *Store) readItchToChange() (map[string]ItchRecord, error) {
	out := map[string]ItchRecord{}
	b, err := os.ReadFile(s.ItchPath())
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	if json.Unmarshal(b, &out) != nil {
		if err := os.Rename(s.ItchPath(), s.ItchPath()+".broken"); err != nil {
			return nil, err
		}
		out = nil
	}
	if out == nil {
		out = map[string]ItchRecord{}
	}
	return out, nil
}

func (s *Store) changeItch(change func(map[string]ItchRecord)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	recs, err := s.readItchToChange()
	if err != nil {
		return err
	}
	change(recs)
	return writeJSONAtomic(s.ItchPath(), recs)
}

func (s *Store) SetItchRecord(key string, r ItchRecord) error {
	if key == "" {
		return errors.New("invalid game")
	}
	return s.changeItch(func(m map[string]ItchRecord) { m[key] = r })
}

func (s *Store) DeleteItchRecord(key string) error {
	return s.changeItch(func(m map[string]ItchRecord) { delete(m, key) })
}

// ------------------------------------------------------------- parsing ---

// itchObj is a JSON object with its keys lowercased and without
// underscores, so snake_case and camelCase read the same.
type itchObj map[string]json.RawMessage

func itchNorm(k string) string { return strings.ToLower(strings.ReplaceAll(k, "_", "")) }

func itchDecode(b []byte) (itchObj, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, errItchReply
	}
	o := itchObj{}
	for k, v := range raw {
		o[itchNorm(k)] = v
	}
	return o, nil
}

func (o itchObj) obj(k string) itchObj {
	x, err := itchDecode(o[k])
	if err != nil {
		return itchObj{}
	}
	return x
}

func (o itchObj) list(k string) []itchObj {
	var raw []json.RawMessage
	_ = json.Unmarshal(o[k], &raw)
	var out []itchObj
	for _, r := range raw {
		if x, err := itchDecode(r); err == nil {
			out = append(out, x)
		}
	}
	return out
}

func (o itchObj) strs(k string) []string {
	var out []string
	if json.Unmarshal(o[k], &out) != nil {
		return nil
	}
	return out
}

func (o itchObj) str(k string) string {
	var s string
	if json.Unmarshal(o[k], &s) == nil {
		return s
	}
	var n json.Number
	if json.Unmarshal(o[k], &n) == nil {
		return n.String()
	}
	return ""
}

func (o itchObj) num(k string) int64 {
	var n json.Number
	if json.Unmarshal(o[k], &n) != nil {
		var s string
		if json.Unmarshal(o[k], &s) != nil {
			return 0
		}
		n = json.Number(strings.TrimSpace(s))
	}
	if i, err := n.Int64(); err == nil {
		return i
	}
	f, _ := n.Float64()
	return int64(f)
}

func (o itchObj) flag(k string) bool {
	var b bool
	if json.Unmarshal(o[k], &b) == nil {
		return b
	}
	switch strings.ToLower(o.str(k)) {
	case "true", "1", "yes":
		return true
	}
	return false
}

func (o itchObj) when(k string) time.Time {
	s := o.str(k)
	for _, f := range []string{time.RFC3339Nano, "2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02 15:04:05.999999"} {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}
