package core

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// fakeApollo answers the calls Lootnest makes as Apollo 0.4.8 does: a
// session cookie from /api/login, JSON everywhere, 401 without the cookie.
type fakeApollo struct {
	mu      sync.Mutex
	user    string
	pass    string
	cookie  string
	config  map[string]any
	saved   map[string]any
	apps    []map[string]any
	pins    map[string]bool // PINs a TV is waiting with
	clients []ApolloClient
	updates []map[string]any
}

func (f *fakeApollo) handler(t *testing.T) http.Handler {
	mux := http.NewServeMux()
	authed := func(r *http.Request) bool {
		c, err := r.Cookie("auth")
		return err == nil && f.cookie != "" && c.Value == f.cookie
	}
	read := func(r *http.Request) map[string]any {
		var m map[string]any
		_ = json.NewDecoder(r.Body).Decode(&m)
		return m
	}
	write := func(w http.ResponseWriter, v any) { _ = json.NewEncoder(w).Encode(v) }
	mux.HandleFunc("/api/password", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.user != "" && !authed(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		m := read(r)
		if m["newPassword"] != m["confirmNewPassword"] {
			w.WriteHeader(http.StatusBadRequest)
			write(w, map[string]any{"status": false, "error": "Password Mismatch"})
			return
		}
		f.user, f.pass = m["newUsername"].(string), m["newPassword"].(string)
		write(w, map[string]any{"status": true})
	})
	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		m := read(r)
		if f.user == "" || !strings.EqualFold(m["username"].(string), f.user) || m["password"] != f.pass {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		f.cookie = "c00kie"
		http.SetCookie(w, &http.Cookie{Name: "auth", Value: f.cookie, Path: "/", Secure: true})
	})
	guarded := func(h func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			f.mu.Lock()
			defer f.mu.Unlock()
			if r.Header.Get("Content-Type") != "application/json" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if !authed(r) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			h(w, r)
		}
	}
	mux.HandleFunc("/api/config", guarded(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			f.saved = read(r)
			write(w, map[string]any{"status": true})
			return
		}
		out := map[string]any{"status": true, "platform": "windows", "version": "0.4.8", "vdisplayStatus": 0}
		for k, v := range f.config {
			out[k] = v
		}
		write(w, out)
	}))
	mux.HandleFunc("/api/apps", guarded(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			app := read(r)
			if app["uuid"] == "" {
				app["uuid"] = "new-uuid"
			}
			f.apps = append(f.apps, app)
			write(w, map[string]any{"status": true})
			return
		}
		write(w, map[string]any{"apps": f.apps, "current_app": ""})
	}))
	mux.HandleFunc("/api/pin", guarded(func(w http.ResponseWriter, r *http.Request) {
		m := read(r)
		ok := f.pins[m["pin"].(string)]
		if ok {
			f.clients = append(f.clients, ApolloClient{Name: m["name"].(string), UUID: "tv-1", Perm: 0x3000000})
		}
		write(w, map[string]any{"status": ok})
	}))
	mux.HandleFunc("/api/clients/list", guarded(func(w http.ResponseWriter, r *http.Request) {
		write(w, map[string]any{"named_certs": f.clients, "status": true})
	}))
	mux.HandleFunc("/api/clients/update", guarded(func(w http.ResponseWriter, r *http.Request) {
		f.updates = append(f.updates, read(r))
		write(w, map[string]any{"status": true})
	}))
	_ = t
	return mux
}

func TestApolloFirstRunPairAndPin(t *testing.T) {
	f := &fakeApollo{config: map[string]any{"encoder": "nvenc", "sunshine_name": "DESKTOP"}, pins: map[string]bool{"4821": true}}
	srv := httptest.NewTLSServer(f.handler(t))
	defer srv.Close()
	ctx := context.Background()

	a, err := NewApollo(srv.URL, "lootnest-x", "s3cret-pass", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Reachable(ctx); err != nil {
		t.Fatalf("reachable: %v", err)
	}
	pin := a.CertPin()
	if len(pin) != 64 {
		t.Fatalf("no certificate pinned on first contact: %q", pin)
	}
	if err := a.SetFirstUser(ctx); err != nil {
		t.Fatalf("first user: %v", err)
	}
	// A second time it's refused: Apollo has a user now and this call
	// carries no session.
	if err := a.SetFirstUser(ctx); !errors.Is(err, ErrApolloSignIn) {
		t.Fatalf("second first user: %v", err)
	}

	if err := a.SetConfig(ctx, ApolloSettings("DESKTOP-7")); err != nil {
		t.Fatalf("set config: %v", err)
	}
	if f.saved["encoder"] != "nvenc" || f.saved["sunshine_name"] != "Lootnest on DESKTOP-7" || f.saved["origin_web_ui_allowed"] != "pc" {
		t.Fatalf("config lost or not set: %v", f.saved)
	}
	for _, k := range apolloConfigInfo {
		if _, ok := f.saved[k]; ok {
			t.Fatalf("read-only key %q written back", k)
		}
	}

	if err := a.SaveApp(ctx, LootnestApp(`C:\Users\a\AppData\Local\Programs\Lootnest\Lootnest.exe`, "", "")); err != nil {
		t.Fatal(err)
	}
	info, err := a.Info(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if info.LootnestApp != "new-uuid" || info.Version != "0.4.8" || !info.VirtualOK {
		t.Fatalf("info: %+v", info)
	}
	app := LootnestAppIn(f.apps)
	if app["cmd"] != `"C:\Users\a\AppData\Local\Programs\Lootnest\Lootnest.exe" --tv` || app["wait-all"] != false || app["virtual-display"] != true {
		t.Fatalf("app: %v", app)
	}

	if _, err := a.Pair(ctx, "48a1", "TV"); err == nil {
		t.Fatal("a PIN with a letter was sent")
	}
	if ok, err := a.Pair(ctx, "1111", "TV"); err != nil || ok {
		t.Fatalf("a PIN no TV waits with: %v %v", ok, err)
	}
	if ok, err := a.Pair(ctx, "4821", "Living room"); err != nil || !ok {
		t.Fatalf("pair: %v %v", ok, err)
	}
	cs, err := a.Clients(ctx)
	if err != nil || len(cs) != 1 || cs[0].Name != "Living room" {
		t.Fatalf("clients: %v %v", cs, err)
	}
	if err := a.AllowTV(ctx, cs[0]); err != nil {
		t.Fatal(err)
	}
	if got := f.updates[0]["perm"].(float64); uint32(got) != ApolloTVPerm || f.updates[0]["allow_client_commands"] != false {
		t.Fatalf("permissions sent: %v", f.updates[0])
	}

	// A new client with the pin talks to it; one pinned to another
	// certificate refuses.
	b, _ := NewApollo(srv.URL, "lootnest-x", "s3cret-pass", pin)
	if _, err := b.Clients(ctx); err != nil {
		t.Fatalf("pinned client: %v", err)
	}
	c, _ := NewApollo(srv.URL, "lootnest-x", "s3cret-pass", strings.Repeat("0", 64))
	if _, err := c.Clients(ctx); !errors.Is(err, ErrApolloCert) {
		t.Fatalf("wrong certificate accepted: %v", err)
	}
	// A wrong password says so.
	d, _ := NewApollo(srv.URL, "lootnest-x", "wrong", pin)
	if _, err := d.Clients(ctx); !errors.Is(err, ErrApolloSignIn) {
		t.Fatalf("wrong password: %v", err)
	}
}

func TestApolloOnlyThisPC(t *testing.T) {
	for _, base := range []string{"https://192.168.1.20:47990", "http://localhost:47990", "https://example.com"} {
		if _, err := NewApollo(base, "u", "p", ""); err == nil {
			t.Errorf("%s accepted", base)
		}
	}
	if _, err := NewApollo(ApolloAPI, "u", "p", ""); err != nil {
		t.Error(err)
	}
}

func TestApolloTVPerm(t *testing.T) {
	// Apollo's crypto::PERM: inputs 0x100..0x1000, list 0x1000000, view
	// 0x2000000, launch 0x4000000; clipboard, files and commands
	// (0x10000..0x100000) stay off.
	if ApolloTVPerm != 0x7001F00 {
		t.Fatalf("%#x", ApolloTVPerm)
	}
	if ApolloTVPerm&0x1F0000 != 0 {
		t.Fatal("a TV may use the clipboard, files or commands")
	}
}

func TestApolloScripts(t *testing.T) {
	s := ApolloInstallScript(`C:\Users\O'Neil\AppData\Local\Lootnest\downloads\apollo\Apollo.exe`, `C:\Program Files\Apollo`, `C:\Users\O'Neil\AppData\Local\Temp\ln-apollo.json`)
	for _, want := range []string{
		`'C:\Users\O''Neil\AppData\Local\Lootnest\downloads\apollo\Apollo.exe'`,
		`-ArgumentList '/S', '/D=C:\Program Files\Apollo'`, // NSIS: /D last and unquoted
		`'--creds', $c.user, $c.pass`,
		`Remove-Item -LiteralPath 'C:\Users\O''Neil\AppData\Local\Temp\ln-apollo.json'`,
		`Set-NetFirewallRule -Profile Private`,
		`Start-Service -Name 'ApolloService'`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("install script lacks %s", want)
		}
	}
	if strings.Contains(ApolloServiceScript(false), "Start-Service") || !strings.Contains(ApolloServiceScript(true), "Automatic") {
		t.Error("service scripts")
	}
}

func TestDownloadApolloNeedsPin(t *testing.T) {
	err := DownloadApollo(context.Background(), ApolloRelease{URL: "https://example.com/a.exe"}, t.TempDir()+"/a.exe", nil)
	if !errors.Is(err, ErrApolloNotPinned) {
		t.Fatal(err)
	}
}
