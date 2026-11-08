package core

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// GOG: the games in the user's GOG library, through the sign-in and API GOG
// Galaxy uses (Heroic, minigalaxy and lgogdownloader use them too). Only
// their library, the installers of a game they own and the files of the
// one they choose. The refresh token never goes in an error or the log;
// the access token goes only to GOG's own hosts, in the Authorization
// header.
// ---------------------------------------------------------------------------

// GOG's hosts; tests point them at a local server.
var (
	GogAuth  = "https://auth.gog.com"
	GogEmbed = "https://embed.gog.com"
	GogAPI   = "https://api.gog.com"
)

// GogTrace, when set, is told the host and path of each request to GOG's
// API, never a query, and only the host of anything else: a download
// server's path carries a signed token.
var GogTrace func(host, path string)

var (
	gogTransport http.RoundTripper = http.DefaultTransport
	gogPause                       = 300 * time.Millisecond // between pages
	gogMu        sync.Mutex                                 // one API request at a time
)

// GOG Galaxy's client id and secret: public, shipped in Galaxy and in every
// client above.
const (
	gogClientID     = "46899977096215655"
	gogClientSecret = "9d85c43b1482497dbbce61f6e4aa173a433796eeae2ca8c5f6129f2dc4de46d9"
	gogRedirect     = "https://embed.gog.com/on_login_success?origin=client"
	gogPages        = 100
)

var (
	// ErrGogSignedOut: GOG refused the refresh token, so the user signs in again.
	ErrGogSignedOut = errors.New("GOG signed you out. Sign in again in Settings, Online.")
	// ErrGogAuth: GOG refused the access token for a request.
	ErrGogAuth   = errors.New("GOG didn't accept the sign-in. Sign in again in Settings, Online.")
	errGogCode   = errors.New("That isn't the address GOG showed after you signed in. Copy the whole address from the browser's address bar, or just the code after code=.")
	errGogBusy   = errors.New("GOG has had too many requests. Try again in a minute.")
	errGogReply  = errors.New("GOG sent a reply Lootnest can't read")
	errGogRefuse = errors.New("refused")
)

// GogLoginURL is GOG's sign-in page, the one Galaxy shows.
func GogLoginURL() string {
	q := url.Values{"client_id": {gogClientID}, "redirect_uri": {gogRedirect}, "response_type": {"code"}, "layout": {"client2"}}
	return "https://auth.gog.com/auth?" + q.Encode()
}

var gogCodeRe = regexp.MustCompile(`^[A-Za-z0-9._~-]{8,1000}$`)

// GogCode is the sign-in code in what the user pasted: the address GOG's
// page ends on (https://embed.gog.com/on_login_success?origin=client&code=...)
// or the code alone.
func GogCode(pasted string) (string, error) {
	s := strings.TrimSpace(pasted)
	if strings.HasPrefix(strings.ToLower(s), "embed.gog.com/") {
		s = "https://" + s
	}
	if strings.Contains(s, "://") {
		u, err := url.Parse(s)
		if err != nil || u.Scheme != "https" || !strings.EqualFold(u.Hostname(), "embed.gog.com") || u.Path != "/on_login_success" {
			return "", errGogCode
		}
		if u.Query().Get("error") != "" {
			return "", errors.New("GOG didn't sign you in. Choose Sign in to GOG and try again.")
		}
		s = u.Query().Get("code")
	} else {
		s = strings.TrimPrefix(s, "code=")
	}
	if !gogCodeRe.MatchString(s) {
		return "", errGogCode
	}
	return s, nil
}

// GogTokens is what GOG gives for a sign-in.
type GogTokens struct {
	Access  string
	Refresh string
	Expires time.Time
	UserID  string
}

func gogClient(timeout time.Duration) *http.Client {
	return &http.Client{Transport: gogTransport, Timeout: timeout, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if r.URL.Scheme != "https" {
			return errors.New("GOG sent the file to an address that isn't https")
		}
		if len(via) >= 10 {
			return errors.New("too many redirects")
		}
		gogTraceHost(r.URL)
		return nil
	}}
}

func hostOf(u string) string {
	p, _ := url.Parse(u)
	if p == nil {
		return ""
	}
	return strings.ToLower(p.Host)
}

// gogOwn: one of GOG's own hosts, the only ones the access token goes to.
func gogOwn(u *url.URL) bool {
	h := strings.ToLower(u.Host)
	return u.Scheme == "https" && (h == "gog.com" || strings.HasSuffix(h, ".gog.com") || h == hostOf(GogAPI) || h == hostOf(GogEmbed) || h == hostOf(GogAuth))
}

func gogTrace(u *url.URL) {
	if h := strings.ToLower(u.Host); GogTrace != nil && (h == hostOf(GogAPI) || h == hostOf(GogEmbed) || h == hostOf(GogAuth)) {
		GogTrace(u.Host, u.Path)
	} else {
		gogTraceHost(u)
	}
}

func gogTraceHost(u *url.URL) {
	if GogTrace != nil {
		GogTrace(u.Host, "")
	}
}

// gogFileStatus reads a download server's answer: a refused link is one
// that ran out, not a sign-in GOG refused.
func gogFileStatus(code int, status string) error {
	if code == http.StatusUnauthorized || code == http.StatusForbidden {
		return errors.New("GOG's download link ran out. Try again")
	}
	return gogStatus(code, status)
}

func gogStatus(code int, status string) error {
	switch {
	case code == http.StatusUnauthorized || code == http.StatusForbidden:
		return ErrGogAuth
	case code == http.StatusTooManyRequests:
		return errGogBusy
	case code < 200 || code > 299:
		return fmt.Errorf("GOG answered %s", status)
	}
	return nil
}

// gogRead sends req and reads the reply, up to 16 MB.
func gogRead(req *http.Request, status func(int, string) error) ([]byte, error) {
	gogMu.Lock()
	defer gogMu.Unlock()
	gogTrace(req.URL)
	resp, err := gogClient(30 * time.Second).Do(req)
	if err != nil {
		return nil, netErr("GOG", err)
	}
	defer resp.Body.Close()
	if err := status(resp.StatusCode, resp.Status); err != nil {
		return nil, err
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, netErr("GOG", err)
	}
	return b, nil
}

// gogGet asks GOG's API for u, as the signed-in user.
func gogGet(ctx context.Context, access, u string) (jsonObj, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil || !gogOwn(req.URL) {
		return nil, errGogReply
	}
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Accept", "application/json")
	b, err := gogRead(req, gogStatus)
	if err != nil {
		return nil, err
	}
	o, err := decodeObj(b)
	if err != nil {
		return nil, errGogReply
	}
	return o, nil
}

func gogTokenCall(ctx context.Context, q url.Values) (GogTokens, error) {
	q.Set("client_id", gogClientID)
	q.Set("client_secret", gogClientSecret)
	req, err := http.NewRequestWithContext(ctx, "GET", GogAuth+"/token?"+q.Encode(), nil)
	if err != nil {
		return GogTokens{}, errGogReply
	}
	b, err := gogRead(req, func(code int, status string) error {
		if code == http.StatusBadRequest || code == http.StatusUnauthorized || code == http.StatusForbidden {
			return errGogRefuse
		}
		return gogStatus(code, status)
	})
	if err != nil {
		return GogTokens{}, err
	}
	o, err := decodeObj(b)
	if err != nil {
		return GogTokens{}, errGogReply
	}
	t := GogTokens{Access: o.str("accesstoken"), Refresh: o.str("refreshtoken"), UserID: o.str("userid")}
	if t.Access == "" {
		if o.str("error") != "" {
			return GogTokens{}, errGogRefuse
		}
		return GogTokens{}, errGogReply
	}
	secs := o.num("expiresin")
	if secs <= 0 {
		secs = 3600
	}
	t.Expires = time.Now().Add(time.Duration(secs) * time.Second)
	return t, nil
}

// GogToken signs in with the code from GOG's sign-in page.
func GogToken(ctx context.Context, code string) (GogTokens, error) {
	t, err := gogTokenCall(ctx, url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {gogRedirect}})
	if errors.Is(err, errGogRefuse) {
		return GogTokens{}, errors.New("GOG didn't take that sign-in. A code works once and only for a few minutes: choose Sign in to GOG again and paste the new address.")
	}
	if err == nil && t.Refresh == "" {
		return GogTokens{}, errGogReply
	}
	return t, err
}

// GogRefresh gets a new access token with the refresh token. When GOG
// refuses it, the error is ErrGogSignedOut.
func GogRefresh(ctx context.Context, refresh string) (GogTokens, error) {
	t, err := gogTokenCall(ctx, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refresh}})
	if errors.Is(err, errGogRefuse) {
		return GogTokens{}, ErrGogSignedOut
	}
	if err == nil && t.Refresh == "" {
		t.Refresh = refresh
	}
	return t, err
}

// GogUser is the signed-in account's user name.
func GogUser(ctx context.Context, access string) (string, error) {
	o, err := gogGet(ctx, access, GogEmbed+"/userData.json")
	if err != nil {
		return "", err
	}
	if name := strings.TrimSpace(o.str("username")); name != "" {
		return name, nil
	}
	return "", errGogReply
}

// GogGame is a game the user owns on GOG.
type GogGame struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Image string `json:"image,omitempty"` // wide, not a box cover
}

// GogImage is a picture's address from GOG's library (//images-N.gog-statics.com/<hash>),
// as an https link with a size, or "" for anything else.
func GogImage(s string) string {
	if strings.HasPrefix(s, "//") {
		s = "https:" + s
	}
	u, err := url.Parse(s)
	if err != nil || u.Scheme != "https" || !strings.HasSuffix(strings.ToLower(u.Hostname()), ".gog-statics.com") || u.RawQuery != "" || u.Fragment != "" {
		return ""
	}
	if path.Ext(u.Path) == "" {
		s += "_392.jpg"
	}
	return s
}

func gogWait(ctx context.Context) error {
	t := time.NewTimer(gogPause)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// GogOwned lists the games the account owns that run on Windows.
func GogOwned(ctx context.Context, access string) ([]GogGame, error) {
	out := []GogGame{}
	seen := map[int64]bool{}
	for page := 1; page <= gogPages; page++ {
		if page > 1 {
			if err := gogWait(ctx); err != nil {
				return nil, err
			}
		}
		o, err := gogGet(ctx, access, fmt.Sprintf("%s/account/getFilteredProducts?mediaType=1&page=%d", GogEmbed, page))
		if err != nil {
			return nil, err
		}
		for _, p := range o.list("products") {
			id, title := p.num("id"), strings.TrimSpace(p.str("title"))
			if id <= 0 || title == "" || seen[id] {
				continue
			}
			if _, says := p["workson"]; says && !p.obj("workson").flag("windows") {
				continue
			}
			seen[id] = true
			out = append(out, GogGame{ID: id, Title: title, Image: GogImage(p.str("image"))})
		}
		if total := o.num("totalpages"); int64(page) >= total {
			break
		}
	}
	return out, nil
}

// GogFile is one part of an installer: the .exe, or a .bin beside it.
type GogFile struct {
	ID       string `json:"id"`
	Size     int64  `json:"size"`
	Downlink string `json:"-"` // GOG's API address that gives the file's link
}

// GogInstaller is one of a game's offline installers.
type GogInstaller struct {
	ID           string    `json:"id"`
	Name         string    `json:"name,omitempty"`
	Language     string    `json:"language"`
	LanguageName string    `json:"languageName,omitempty"`
	Version      string    `json:"version,omitempty"`
	Size         int64     `json:"size"`
	Files        []GogFile `json:"files"`
}

// GogInstallers lists a game's Windows installers. Only the full installers:
// GOG's patches (downloads.patches) aren't used, since a full installer
// always brings the game to its version, whatever version it's on.
func GogInstallers(ctx context.Context, access string, id int64) ([]GogInstaller, error) {
	o, err := gogGet(ctx, access, fmt.Sprintf("%s/products/%d?expand=downloads&locale=en-US", GogAPI, id))
	if err != nil {
		return nil, err
	}
	out := []GogInstaller{}
	for _, x := range o.obj("downloads").list("installers") {
		if !strings.EqualFold(x.str("os"), "windows") {
			continue
		}
		in := GogInstaller{ID: x.str("id"), Name: strings.TrimSpace(x.str("name")), Language: strings.ToLower(x.str("language")),
			LanguageName: x.str("languagefull"), Version: strings.TrimSpace(x.str("version")), Size: x.num("totalsize")}
		var sum int64
		for _, f := range x.list("files") {
			if dl := f.str("downlink"); dl != "" {
				in.Files = append(in.Files, GogFile{ID: f.str("id"), Size: f.num("size"), Downlink: dl})
				sum += f.num("size")
			}
		}
		if in.Size <= 0 {
			in.Size = sum
		}
		if in.ID != "" && len(in.Files) > 0 {
			out = append(out, in)
		}
	}
	return out, nil
}

// GogLang is GOG's language code for a Windows language tag ("ro-RO").
func GogLang(tag string) string {
	tag = strings.ToLower(strings.ReplaceAll(tag, "_", "-"))
	if tag == "pt-br" {
		return "br"
	}
	lang, _, _ := strings.Cut(tag, "-")
	switch lang {
	case "ja":
		return "jp"
	case "cs":
		return "cz"
	case "zh":
		return "cn"
	}
	return lang
}

// GogDefault is the installer to offer first: in lang, else English, else
// the first one.
func GogDefault(ins []GogInstaller, lang string) string {
	for _, want := range []string{lang, "en"} {
		for _, in := range ins {
			if want != "" && in.Language == want {
				return in.ID
			}
		}
	}
	if len(ins) > 0 {
		return ins[0].ID
	}
	return ""
}

// GogLink is where a file of an installer downloads from.
type GogLink struct {
	URL  string
	Name string // the file's own name: the installer finds its .bin parts by theirs
	MD5  string
	Size int64
}

var md5Re = regexp.MustCompile(`^[0-9a-fA-F]{32}$`)

// GogFileLink asks GOG for a file's link, and its name, md5 and size from
// the checksum GOG keeps beside it when there is one.
func GogFileLink(ctx context.Context, access string, f GogFile) (GogLink, error) {
	o, err := gogGet(ctx, access, f.Downlink)
	if err != nil {
		return GogLink{}, err
	}
	l := GogLink{URL: o.str("downlink"), Size: f.Size}
	u, err := url.Parse(l.URL)
	if err != nil || u.Scheme != "https" {
		return GogLink{}, errors.New("GOG sent a download address that isn't https")
	}
	if c := o.str("checksum"); c != "" {
		l.Name, l.MD5, l.Size = gogChecksum(ctx, c, l.Size)
	}
	if l.Name == "" {
		l.Name, _ = url.PathUnescape(path.Base(u.Path))
	}
	if l.Name = filepath.Base(strings.ReplaceAll(l.Name, `\`, "/")); l.Name == "" || l.Name == "." || l.Name == "/" {
		return GogLink{}, errGogReply
	}
	return l, nil
}

// gogChecksum reads the root of GOG's checksum XML: <file name="" md5=""
// total_size="">. Anything it can't read is left out.
func gogChecksum(ctx context.Context, u string, size int64) (name, sum string, total int64) {
	total = size
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil || req.URL.Scheme != "https" {
		return
	}
	b, err := gogRead(req, gogStatus)
	if err != nil {
		return
	}
	d := xml.NewDecoder(strings.NewReader(string(b)))
	for {
		tok, err := d.Token()
		if err != nil {
			return
		}
		if se, ok := tok.(xml.StartElement); ok {
			for _, a := range se.Attr {
				switch a.Name.Local {
				case "name":
					name = strings.TrimSpace(a.Value)
				case "md5":
					if md5Re.MatchString(a.Value) {
						sum = a.Value
					}
				case "total_size":
					var n int64
					if _, err := fmt.Sscan(a.Value, &n); err == nil && n > 0 {
						total = n
					}
				}
			}
			return
		}
	}
}

// GogDownload saves a file of an installer to dst (see downloader). One
// cut short or cancelled keeps its .part, so the next try goes on from there.
func GogDownload(ctx context.Context, l GogLink, dst string, progress func(done, total int64)) error {
	return downloader{client: gogClient(0), store: "GOG", status: gogFileStatus, trace: gogTraceHost, keep: true}.get(ctx, l.URL, dst, l.Size, l.MD5, progress)
}

// GogDownloadPath is where a file of a game's installer is downloaded to:
// Lootnest's own downloads\gog\<product id> folder.
func GogDownloadPath(dataDir string, id int64, name string) string {
	return downloadPath(dataDir, "gog", id, name)
}

// GogSetup is the installer's program among its files: the first .exe.
func GogSetup(paths []string) string {
	for _, p := range paths {
		if strings.EqualFold(filepath.Ext(p), ".exe") {
			return p
		}
	}
	return ""
}

// GogInnoArgs installs a GOG offline installer (Inno Setup) into dir
// without questions; it shows its own progress.
func GogInnoArgs(dir string) string {
	return `/DIR="` + strings.TrimRight(dir, `\`) + `" /SILENT /SUPPRESSMSGBOXES /NORESTART /SP-`
}

// --------------------------------------------------------- the record ---

// GogRecord is what Lootnest installed from GOG for a Library game.
type GogRecord struct {
	ProductID   int64      `json:"productId"`
	InstallerID string     `json:"installerId"`
	Language    string     `json:"language,omitempty"`
	Version     string     `json:"version,omitempty"`
	Size        int64      `json:"size,omitempty"`
	InstalledAt time.Time  `json:"installedAt"`
	CheckedAt   time.Time  `json:"checkedAt"`
	UpdateID    string     `json:"updateId,omitempty"` // the applied update that brought this installer in
	Prev        *GogRecord `json:"prev,omitempty"`     // the record before it, back on Undo
}

// GogRecordOf is the record for an installer of game id installed at now.
func GogRecordOf(id int64, in GogInstaller, now time.Time) GogRecord {
	return GogRecord{ProductID: id, InstallerID: in.ID, Language: in.Language, Version: in.Version, Size: in.Size, InstalledAt: now.UTC(), CheckedAt: now.UTC()}
}

// GogUpdate is the installer that updates a game Lootnest installed from
// GOG, and whether there is one: the Windows installer in the language
// installed (else the same installer) with another version.
func GogUpdate(rec GogRecord, ins []GogInstaller) (GogInstaller, bool) {
	for _, same := range []func(GogInstaller) bool{
		func(in GogInstaller) bool { return rec.Language != "" && in.Language == rec.Language },
		func(in GogInstaller) bool { return in.ID == rec.InstallerID },
	} {
		for _, in := range ins {
			if same(in) {
				return in, in.Version != "" && in.Version != strings.TrimSpace(rec.Version)
			}
		}
	}
	return GogInstaller{}, false
}

func (s *Store) GogPath() string { return filepath.Join(s.Dir, "gog.json") }

// GogRecords is every record, by Library key.
func (s *Store) GogRecords() map[string]GogRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	return readRecords[GogRecord](s.GogPath())
}

func (s *Store) SetGogRecord(key string, r GogRecord) error {
	if key == "" {
		return errors.New("invalid game")
	}
	return changeRecords(s, s.GogPath(), func(m map[string]GogRecord) { m[key] = r })
}
