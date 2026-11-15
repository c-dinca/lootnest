package core

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Apollo (Lootnest Link M2, docs/lootnest-link-spec.md): the streaming
// engine Lootnest installs and drives so a TV can play the PC's games. It
// is a separate program (GPL v3, github.com/ClassicOldSong/Apollo), run as
// a Windows service, that Lootnest only talks to over its local web API on
// this PC: never its web pages, never from another PC.
//
// The API signs in with a user and a password (Lootnest makes both and
// keeps them with DPAPI) and answers over https with a certificate Apollo
// made for itself, so Lootnest pins that certificate the first time and
// refuses any other after.

// ApolloAPI is the engine's address: this PC only.
const ApolloAPI = "https://localhost:47990"

// ApolloTVPerm is what a TV Lootnest pairs may do: every kind of input,
// list the apps, watch and start them. Apollo gives a second TV only
// list and view, so it couldn't start Lootnest or use a controller.
// Clipboard, files and the host's commands stay off.
const ApolloTVPerm uint32 = 0x1F00 | 0x7000000

// ApolloAppName is the one app Lootnest adds to Apollo.
const ApolloAppName = "Lootnest"

// Apollo is a client of the engine's local API.
type Apollo struct {
	Base       string // ApolloAPI, or a test server's address
	User, Pass string

	mu     sync.Mutex
	cert   string // the pinned certificate's SHA-256, hex; "" until the first contact
	cookie string
	client *http.Client
}

// NewApollo makes a client that trusts only the certificate with this
// fingerprint, or, with "", the first one it sees (CertPin then says which).
func NewApollo(base, user, pass, pin string) (*Apollo, error) {
	u, err := url.Parse(base)
	if err != nil || u.Scheme != "https" || !loopbackHost(u.Hostname()) {
		return nil, errors.New("Lootnest only talks to the streaming engine on this PC")
	}
	a := &Apollo{Base: strings.TrimRight(base, "/"), User: user, Pass: pass, cert: strings.ToLower(pin)}
	a.client = &http.Client{
		Timeout: 20 * time.Second,
		Transport: &http.Transport{
			Proxy: nil, // never through a proxy: it's this PC
			TLSClientConfig: &tls.Config{
				// Apollo's certificate is its own, not from an authority:
				// it's checked by its fingerprint below instead.
				InsecureSkipVerify: true, //nolint:gosec
				VerifyConnection:   a.verify,
			},
		},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	return a, nil
}

func loopbackHost(h string) bool {
	return h == "localhost" || h == "127.0.0.1" || h == "::1"
}

func (a *Apollo) verify(cs tls.ConnectionState) error {
	if len(cs.PeerCertificates) == 0 {
		return errors.New("the streaming engine sent no certificate")
	}
	sum := sha256.Sum256(cs.PeerCertificates[0].Raw)
	got := hex.EncodeToString(sum[:])
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cert == "" {
		a.cert = got
		return nil
	}
	if got != a.cert {
		return ErrApolloCert
	}
	return nil
}

// ErrApolloCert is a certificate other than the one pinned: something else
// answers on the engine's port, or the engine was installed again.
var ErrApolloCert = errors.New("the streaming engine's certificate changed, so Lootnest stopped talking to it")

// CertPin is the pinned certificate's fingerprint, to keep with the
// credentials.
func (a *Apollo) CertPin() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cert
}

// ApolloError is the engine answering with an error.
type ApolloError struct {
	Code int
	Msg  string
}

func (e *ApolloError) Error() string {
	if e.Msg != "" {
		return "the streaming engine said: " + e.Msg
	}
	return fmt.Sprintf("the streaming engine answered %d", e.Code)
}

// ErrApolloSignIn is a refused sign-in: Apollo has a user Lootnest doesn't
// know (it was set up by hand before).
var ErrApolloSignIn = errors.New("the streaming engine didn't accept Lootnest's sign-in")

func (a *Apollo) send(ctx context.Context, method, path string, body any, withCookie bool) (*http.Response, []byte, error) {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, a.Base+path, rd)
	if err != nil {
		return nil, nil, err
	}
	// Apollo checks the type on every call, GETs included for some.
	req.Header.Set("Content-Type", "application/json")
	if withCookie {
		a.mu.Lock()
		if a.cookie != "" {
			req.AddCookie(&http.Cookie{Name: "auth", Value: a.cookie})
		}
		a.mu.Unlock()
	}
	resp, err := a.client.Do(req)
	if err != nil {
		if errors.Is(err, ErrApolloCert) || strings.Contains(err.Error(), ErrApolloCert.Error()) {
			return nil, nil, ErrApolloCert
		}
		return nil, nil, fmt.Errorf("couldn't reach the streaming engine (%w)", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	return resp, data, err
}

// login signs in and keeps the session cookie.
func (a *Apollo) login(ctx context.Context) error {
	resp, _, err := a.send(ctx, "POST", "/api/login", map[string]string{"username": a.User, "password": a.Pass}, false)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return ErrApolloSignIn
	}
	for _, c := range resp.Cookies() {
		if c.Name == "auth" && c.Value != "" {
			a.mu.Lock()
			a.cookie = c.Value
			a.mu.Unlock()
			return nil
		}
	}
	return ErrApolloSignIn
}

// call signs in when it must and decodes the answer into out.
func (a *Apollo) call(ctx context.Context, method, path string, body, out any) error {
	a.mu.Lock()
	signed := a.cookie != ""
	a.mu.Unlock()
	if !signed {
		if err := a.login(ctx); err != nil {
			return err
		}
	}
	resp, data, err := a.send(ctx, method, path, body, true)
	if err == nil && resp.StatusCode == http.StatusUnauthorized {
		// The session ended (Apollo keeps one, for a while): once more.
		if err := a.login(ctx); err != nil {
			return err
		}
		resp, data, err = a.send(ctx, method, path, body, true)
	}
	if err != nil {
		return err
	}
	return apolloAnswer(resp, data, out)
}

func apolloAnswer(resp *http.Response, data []byte, out any) error {
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(data, &e)
		return &ApolloError{Code: resp.StatusCode, Msg: e.Error}
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return errors.New("the streaming engine sent an answer Lootnest can't read")
	}
	return nil
}

// Reachable says whether the engine answers on this PC at all.
func (a *Apollo) Reachable(ctx context.Context) error {
	_, _, err := a.send(ctx, "GET", "/api/apps", nil, false)
	return err
}

// SetFirstUser sets the engine's user and password. Apollo allows it
// without signing in only while it has none.
func (a *Apollo) SetFirstUser(ctx context.Context) error {
	resp, data, err := a.send(ctx, "POST", "/api/password", map[string]string{
		"newUsername": a.User, "newPassword": a.Pass, "confirmNewPassword": a.Pass,
	}, false)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return ErrApolloSignIn
	}
	var st struct {
		Status bool `json:"status"`
	}
	if err := apolloAnswer(resp, data, &st); err != nil {
		return err
	}
	if !st.Status {
		return ErrApolloSignIn
	}
	return nil
}

// ApolloInfo is what Lootnest shows of the engine.
type ApolloInfo struct {
	Version     string `json:"version"`
	Name        string `json:"name"`
	VirtualOK   bool   `json:"virtualDisplay"` // the virtual display driver works
	Encoder     string `json:"encoder"`        // the encoder chosen, "" for automatic
	RunningApp  string `json:"runningApp"`     // the app streaming now, by its uuid
	LootnestApp string `json:"lootnestApp"`    // Lootnest's app's uuid, "" when it isn't there
}

// Config reads the engine's settings: its own keys, all as text.
func (a *Apollo) Config(ctx context.Context) (map[string]any, error) {
	var m map[string]any
	if err := a.call(ctx, "GET", "/api/config", nil, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// apolloConfigInfo are the keys GET /api/config adds that aren't settings.
var apolloConfigInfo = []string{"status", "platform", "version", "vdisplayStatus"}

// SetConfig changes these settings and keeps every other. Apollo writes
// the whole file from what it's sent, so the rest is read first. It takes
// effect once the engine restarts.
func (a *Apollo) SetConfig(ctx context.Context, set map[string]string) error {
	cur, err := a.Config(ctx)
	if err != nil {
		return err
	}
	out := MergeApolloConfig(cur, set)
	var st struct {
		Status bool `json:"status"`
	}
	return a.call(ctx, "POST", "/api/config", out, &st)
}

// MergeApolloConfig is the settings file Apollo is sent: what it has, its
// read-only keys left out, with set on top.
func MergeApolloConfig(cur map[string]any, set map[string]string) map[string]any {
	out := map[string]any{}
	for k, v := range cur {
		out[k] = v
	}
	for _, k := range apolloConfigInfo {
		delete(out, k)
	}
	for k, v := range set {
		out[k] = v
	}
	return out
}

// Info reads the engine's version, its virtual display and Lootnest's app.
func (a *Apollo) Info(ctx context.Context) (ApolloInfo, error) {
	cfg, err := a.Config(ctx)
	if err != nil {
		return ApolloInfo{}, err
	}
	var in ApolloInfo
	in.Version, _ = cfg["version"].(string)
	in.Name, _ = cfg["sunshine_name"].(string)
	in.Encoder, _ = cfg["encoder"].(string)
	if v, ok := cfg["vdisplayStatus"].(float64); ok {
		in.VirtualOK = v == 0 // VDISPLAY::DRIVER_STATUS::OK
	}
	apps, cur, err := a.Apps(ctx)
	if err != nil {
		return in, err
	}
	in.RunningApp = cur
	if app := LootnestAppIn(apps); app != nil {
		in.LootnestApp, _ = app["uuid"].(string)
	}
	return in, nil
}

// Apps lists the engine's apps and the uuid of the one running.
func (a *Apollo) Apps(ctx context.Context) ([]map[string]any, string, error) {
	var r struct {
		Apps    []map[string]any `json:"apps"`
		Current string           `json:"current_app"`
	}
	if err := a.call(ctx, "GET", "/api/apps", nil, &r); err != nil {
		return nil, "", err
	}
	return r.Apps, r.Current, nil
}

// LootnestAppIn finds Lootnest's app among the engine's.
func LootnestAppIn(apps []map[string]any) map[string]any {
	for _, app := range apps {
		if n, _ := app["name"].(string); n == ApolloAppName {
			return app
		}
	}
	return nil
}

// LootnestApp is Lootnest's app as Apollo keeps it: TV mode on a virtual
// display the size of the TV. The stream ends when the --tv holder does
// (wait-all off: Lootnest itself runs on, outside the stream), and it's
// never seen as detached. uuid keeps the one there, "" adds it.
func LootnestApp(exe, cover, uuid string) map[string]any {
	app := map[string]any{
		"name":                    ApolloAppName,
		"cmd":                     `"` + exe + `" --tv`,
		"output":                  "",
		"exclude-global-prep-cmd": false,
		"elevated":                false,
		"auto-detach":             false,
		"wait-all":                false,
		"exit-timeout":            5,
		"virtual-display":         true,
		"prep-cmd":                []any{},
		"detached":                []any{},
		"image-path":              cover,
		"uuid":                    uuid,
	}
	return app
}

// SaveApp adds or updates an app (by its uuid).
func (a *Apollo) SaveApp(ctx context.Context, app map[string]any) error {
	var st struct {
		Status bool `json:"status"`
	}
	return a.call(ctx, "POST", "/api/apps", app, &st)
}

// ApolloClient is a paired TV.
type ApolloClient struct {
	Name      string `json:"name"`
	UUID      string `json:"uuid"`
	Perm      uint32 `json:"perm"`
	Connected bool   `json:"connected"`
}

// Clients lists the paired TVs (and any other paired device).
func (a *Apollo) Clients(ctx context.Context) ([]ApolloClient, error) {
	var r struct {
		Certs []ApolloClient `json:"named_certs"`
	}
	if err := a.call(ctx, "GET", "/api/clients/list", nil, &r); err != nil {
		return nil, err
	}
	return r.Certs, nil
}

// Pair completes pairing with the PIN the TV shows. It answers false when
// no TV is waiting with that PIN.
func (a *Apollo) Pair(ctx context.Context, pin, name string) (bool, error) {
	if !ValidPairPIN(pin) {
		return false, errors.New("the PIN is the 4 digits your TV shows")
	}
	var st struct {
		Status bool `json:"status"`
	}
	if err := a.call(ctx, "POST", "/api/pin", map[string]string{"pin": pin, "name": name}, &st); err != nil {
		return false, err
	}
	return st.Status, nil
}

// ValidPairPIN is four digits.
func ValidPairPIN(pin string) bool {
	if len(pin) != 4 {
		return false
	}
	for _, r := range pin {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// AllowTV gives a paired TV what it needs to play (ApolloTVPerm), keeping
// its name.
func (a *Apollo) AllowTV(ctx context.Context, c ApolloClient) error {
	var st struct {
		Status bool `json:"status"`
	}
	if err := a.call(ctx, "POST", "/api/clients/update", map[string]any{
		"uuid": c.UUID, "name": c.Name, "perm": ApolloTVPerm,
		"enable_legacy_ordering": true, "allow_client_commands": false, "always_use_virtual_display": true,
	}, &st); err != nil {
		return err
	}
	if !st.Status {
		return errors.New("the streaming engine didn't let that TV play")
	}
	return nil
}

// Unpair forgets a TV: it has to pair again to stream.
func (a *Apollo) Unpair(ctx context.Context, uuid string) error {
	return a.call(ctx, "POST", "/api/clients/unpair", map[string]string{"uuid": uuid}, nil)
}

// CloseApp ends the stream's app, and with it the stream.
func (a *Apollo) CloseApp(ctx context.Context) error {
	return a.call(ctx, "POST", "/api/apps/close", map[string]any{}, nil)
}

// Restart restarts the engine, for settings to take effect. It answers
// before it goes, or not at all.
func (a *Apollo) Restart(ctx context.Context) error {
	err := a.call(ctx, "POST", "/api/restart", map[string]any{}, nil)
	a.mu.Lock()
	a.cookie = ""
	a.mu.Unlock()
	var ae *ApolloError
	if err != nil && !errors.As(err, &ae) {
		return nil // it went before it answered
	}
	return err
}

// Logs reads the engine's log, for Export diagnostics.
func (a *Apollo) Logs(ctx context.Context) (string, error) {
	a.mu.Lock()
	signed := a.cookie != ""
	a.mu.Unlock()
	if !signed {
		if err := a.login(ctx); err != nil {
			return "", err
		}
	}
	resp, data, err := a.send(ctx, "GET", "/api/logs", nil, true)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", &ApolloError{Code: resp.StatusCode}
	}
	return string(data), nil
}

// ApolloSettings are the engine settings Lootnest sets: its name on the
// TV, its web pages only from this PC.
func ApolloSettings(pcName string) map[string]string {
	name := "Lootnest"
	if pcName = strings.TrimSpace(pcName); pcName != "" {
		name += " on " + pcName
	}
	return map[string]string{
		"sunshine_name":         name,
		"origin_web_ui_allowed": "pc",
	}
}
