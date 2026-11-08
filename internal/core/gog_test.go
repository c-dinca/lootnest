package core

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const gogRefreshTok = "REFRESHsecret0123456789"

// gogServer stands in for auth.gog.com, embed.gog.com and api.gog.com over
// https, with Lootnest's client pointed at it.
func gogServer(t *testing.T, h http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewTLSServer(h)
	auth, embed, api, tr, pause := GogAuth, GogEmbed, GogAPI, gogTransport, gogPause
	GogAuth, GogEmbed, GogAPI, gogTransport, gogPause = srv.URL, srv.URL, srv.URL, srv.Client().Transport, 0
	t.Cleanup(func() { GogAuth, GogEmbed, GogAPI, gogTransport, gogPause = auth, embed, api, tr, pause; srv.Close() })
	return srv
}

func noRefresh(t *testing.T, err error) {
	t.Helper()
	if err != nil && strings.Contains(err.Error(), gogRefreshTok) {
		t.Errorf("the refresh token is in an error: %v", err)
	}
}

func TestGogCode(t *testing.T) {
	code := "Abc-123_xyz.Q9~LongEnough"
	for in, want := range map[string]string{
		"https://embed.gog.com/on_login_success?origin=client&code=" + code:             code,
		"  https://embed.gog.com/on_login_success?code=" + code + "&origin=client&x=1 ": code,
		"embed.gog.com/on_login_success?origin=client&code=" + code:                     code,
		code:           code,
		"code=" + code: code,
	} {
		if got, err := GogCode(in); err != nil || got != want {
			t.Errorf("GogCode(%q) = %q, %v", in, got, err)
		}
	}
	for _, in := range []string{"", "hello there", "short",
		"https://evil.example/on_login_success?code=" + code,
		"http://embed.gog.com/on_login_success?code=" + code,
		"https://embed.gog.com/other?code=" + code,
		"https://embed.gog.com/on_login_success?origin=client",
		"https://embed.gog.com/on_login_success?code=" + code + "%22%3E",
		"javascript:alert(1)"} {
		if got, err := GogCode(in); err == nil {
			t.Errorf("GogCode(%q) = %q, want it refused", in, got)
		}
	}
	if _, err := GogCode("https://embed.gog.com/on_login_success?origin=client&error=access_denied"); err == nil || !strings.Contains(err.Error(), "didn't sign you in") {
		t.Errorf("error param: %v", err)
	}
}

func TestGogLoginLink(t *testing.T) {
	u := GogLoginURL()
	p, err := url.Parse(u)
	if err != nil || p.Host != "auth.gog.com" || p.Path != "/auth" {
		t.Fatalf("login link %s", u)
	}
	q := p.Query()
	if q.Get("client_id") != "46899977096215655" || q.Get("redirect_uri") != gogRedirect || q.Get("response_type") != "code" || q.Get("layout") != "client2" {
		t.Errorf("login link %s", u)
	}
	if !SafeStoreLink(u) {
		t.Error("the login link isn't allowed")
	}
	for _, bad := range []string{u + "&x=1", strings.Replace(u, "client2", "x", 1), "https://auth.gog.com/auth", "https://auth.gog.com/token?x=1"} {
		if SafeStoreLink(bad) {
			t.Errorf("%s is allowed", bad)
		}
	}
}

func TestGogTokens(t *testing.T) {
	var asked []string
	refuse := false
	gogServer(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Path != "/token" || q.Get("client_id") != gogClientID || q.Get("client_secret") != gogClientSecret {
			http.NotFound(w, r)
			return
		}
		asked = append(asked, q.Get("grant_type"))
		switch {
		case refuse:
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"error":"invalid_grant","error_description":"bad token %s"}`, q.Get("refresh_token"))
		case q.Get("grant_type") == "authorization_code" && q.Get("code") == "thecode123" && q.Get("redirect_uri") == gogRedirect:
			fmt.Fprintf(w, `{"access_token":"acc1","expires_in":3600,"token_type":"bearer","refresh_token":"%s","user_id":"4801"}`, gogRefreshTok)
		case q.Get("grant_type") == "refresh_token" && q.Get("refresh_token") == gogRefreshTok:
			fmt.Fprint(w, `{"accessToken":"acc2","expiresIn":"60","refreshToken":"new-refresh"}`)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	})
	ctx := context.Background()
	tk, err := GogToken(ctx, "thecode123")
	if err != nil || tk.Access != "acc1" || tk.Refresh != gogRefreshTok || tk.UserID != "4801" || time.Until(tk.Expires) < 50*time.Minute {
		t.Fatalf("token = %+v, %v", tk, err)
	}
	tk, err = GogRefresh(ctx, gogRefreshTok)
	if err != nil || tk.Access != "acc2" || tk.Refresh != "new-refresh" || time.Until(tk.Expires) > 2*time.Minute {
		t.Fatalf("refresh = %+v, %v", tk, err)
	}
	if _, err := GogToken(ctx, "wrongcode1"); err == nil || !strings.Contains(err.Error(), "Sign in to GOG again") {
		t.Errorf("a refused code: %v", err)
	}
	refuse = true
	_, err = GogRefresh(ctx, gogRefreshTok)
	if err != ErrGogSignedOut {
		t.Errorf("a refused refresh: %v", err)
	}
	noRefresh(t, err)
	if strings.Join(asked, ",") != "authorization_code,refresh_token,authorization_code,refresh_token" {
		t.Errorf("asked %v", asked)
	}
	// A server error, and nothing there at all: the address, with the
	// refresh token in it, must not come back.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down "+r.URL.RawQuery, http.StatusBadGateway)
	}))
	defer srv.Close()
	GogAuth = srv.URL
	_, err = GogRefresh(ctx, gogRefreshTok)
	if err == nil || err == ErrGogSignedOut || !strings.Contains(err.Error(), "502") {
		t.Errorf("502: %v", err)
	}
	noRefresh(t, err)
	GogAuth = "https://127.0.0.1:1"
	_, err = GogRefresh(ctx, gogRefreshTok)
	if err == nil || err == ErrGogSignedOut {
		t.Errorf("unreachable: %v", err)
	}
	noRefresh(t, err)
}

func TestGogLibrary(t *testing.T) {
	var traced []string
	GogTrace = func(host, path string) { traced = append(traced, path) }
	defer func() { GogTrace = nil }()
	gogServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer T0KEN" {
			http.Error(w, "no", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/userData.json":
			fmt.Fprint(w, `{"username":"player1","galaxyUserId":"1"}`)
		case "/account/getFilteredProducts":
			if r.URL.Query().Get("mediaType") != "1" {
				http.NotFound(w, r)
				return
			}
			if r.URL.Query().Get("page") == "1" {
				fmt.Fprint(w, `{"page":1,"totalPages":2,"products":[
					{"id":1207658924,"title":"The Witcher","image":"//images-1.gog-statics.com/abc","worksOn":{"Windows":true,"Mac":false}},
					{"id":5,"title":"Mac only","worksOn":{"Windows":false,"Mac":true}},
					{"id":6,"title":"  Says nothing  "}]}`)
				return
			}
			fmt.Fprint(w, `{"page":2,"totalPages":2,"products":[{"id":1207658924,"title":"The Witcher again","worksOn":{"Windows":true}},
				{"id":"7","title":"Id as text","image":"https://evil.example/x.jpg","works_on":{"windows":true}}]}`)
		}
	})
	ctx := context.Background()
	if name, err := GogUser(ctx, "T0KEN"); err != nil || name != "player1" {
		t.Fatalf("user = %q, %v", name, err)
	}
	got, err := GogOwned(ctx, "T0KEN")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].ID != 1207658924 || got[0].Image != "https://images-1.gog-statics.com/abc_392.jpg" || got[1].Title != "Says nothing" || got[2].ID != 7 || got[2].Image != "" {
		t.Errorf("owned = %+v", got)
	}
	if _, err := GogOwned(ctx, "other"); err != ErrGogAuth {
		t.Errorf("401: %v", err)
	}
	for _, p := range traced {
		if strings.Contains(p, "?") || strings.Contains(p, "T0KEN") {
			t.Errorf("traced %q", p)
		}
	}
}

func TestGogInstallersAndDownload(t *testing.T) {
	exe, bin := strings.Repeat("setup bytes ", 300), strings.Repeat("bin part ", 700)
	sum := func(s string) string { h := md5.Sum([]byte(s)); return hex.EncodeToString(h[:]) }
	var srv *httptest.Server
	var cdnAuth []string
	srv = gogServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/products/42":
			if r.URL.Query().Get("expand") != "downloads" {
				http.NotFound(w, r)
				return
			}
			fmt.Fprintf(w, `{"id":42,"downloads":{"installers":[
				{"id":"en1installer0","name":"Game","os":"windows","language":"en","language_full":"English","version":"1.2 (gog-3)","total_size":%d,
				 "files":[{"id":"en1installer0","size":%d,"downlink":"%s/products/42/downlink/installer/en1installer0"},{"id":"en1installer1","size":%d,"downlink":"%s/products/42/downlink/installer/en1installer1"}]},
				{"id":"de1installer0","os":"windows","language":"de","version":"1.2 (gog-3)","files":[{"id":"x","size":5,"downlink":"%s/d"}]},
				{"id":"en2installer0","os":"mac","language":"en","files":[{"id":"y","size":5,"downlink":"%s/d"}]},
				{"id":"empty","os":"windows","language":"fr","files":[]}],
				"patches":[{"id":"p","os":"windows"}]}}`, len(exe)+len(bin), len(exe), srv.URL, len(bin), srv.URL, srv.URL, srv.URL)
		case "/products/42/downlink/installer/en1installer0":
			fmt.Fprintf(w, `{"downlink":"%s/cdn/secure/setup_game_1.2_%%2812345%%29.exe?t=1","checksum":"%s/cdn/setup.xml"}`, srv.URL, srv.URL)
		case "/products/42/downlink/installer/en1installer1":
			fmt.Fprintf(w, `{"downlink":"%s/cdn/secure/setup_game_1.2_(12345)-1.bin","checksum":""}`, srv.URL)
		case "/cdn/setup.xml":
			fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?><file name="setup_game_1.2_(12345).exe" available="1" md5="%s" chunks="1" timestamp="x" total_size="%d"><chunk id="0"/></file>`, sum(exe), len(exe))
		case "/cdn/secure/setup_game_1.2_(12345).exe":
			cdnAuth = append(cdnAuth, r.Header.Get("Authorization"))
			fmt.Fprint(w, exe)
		case "/cdn/secure/setup_game_1.2_(12345)-1.bin":
			cdnAuth = append(cdnAuth, r.Header.Get("Authorization"))
			http.ServeContent(w, r, "x.bin", time.Time{}, strings.NewReader(bin)) // answers Range with 206
		}
	})
	ctx := context.Background()
	ins, err := GogInstallers(ctx, "acc", 42)
	if err != nil {
		t.Fatal(err)
	}
	if len(ins) != 2 || ins[0].ID != "en1installer0" || ins[0].LanguageName != "English" || ins[0].Version != "1.2 (gog-3)" || len(ins[0].Files) != 2 || ins[1].Size != 5 {
		t.Fatalf("installers = %+v", ins)
	}
	l, err := GogFileLink(ctx, "acc", ins[0].Files[0])
	if err != nil || l.Name != "setup_game_1.2_(12345).exe" || l.MD5 != sum(exe) || l.Size != int64(len(exe)) {
		t.Fatalf("exe link = %+v, %v", l, err)
	}
	dir := t.TempDir()
	dst := GogDownloadPath(dir, 42, l.Name)
	if filepath.Base(dst) != l.Name || filepath.Base(filepath.Dir(dst)) != "42" {
		t.Errorf("path %s", dst)
	}
	if err := GogDownload(ctx, l, dst, nil); err != nil {
		t.Fatal(err)
	}
	lb, err := GogFileLink(ctx, "acc", ins[0].Files[1])
	if err != nil || lb.Name != "setup_game_1.2_(12345)-1.bin" || lb.MD5 != "" || lb.Size != int64(len(bin)) {
		t.Fatalf("bin link = %+v, %v", lb, err)
	}
	// A .part from a try cut short goes on from where it stopped.
	bdst := GogDownloadPath(dir, 42, lb.Name)
	write(t, bdst+".part", bin[:1000])
	var first int64 = -1
	if err := GogDownload(ctx, lb, bdst, func(done, total int64) {
		if first < 0 {
			first = done
		}
	}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(bdst); string(b) != bin || first <= 1000 {
		t.Errorf("resumed: %d bytes, first progress %d", len(b), first)
	}
	for _, a := range cdnAuth {
		if a != "" {
			t.Error("the access token went to the download server")
		}
	}
	if _, err := GogFileLink(ctx, "acc", GogFile{Downlink: "https://evil.example/x"}); err == nil {
		t.Error("a downlink off GOG's hosts was asked with the token")
	}
	if !strings.HasSuffix(GogSetup([]string{lb.Name, l.Name}), ".exe") || GogSetup([]string{"a.bin"}) != "" {
		t.Error("GogSetup")
	}
	if a := GogInnoArgs(`D:\Games\The Witcher\`); a != `/DIR="D:\Games\The Witcher" /SILENT /SUPPRESSMSGBOXES /NORESTART /SP-` {
		t.Errorf("args %s", a)
	}
}

func TestDownloaderResume(t *testing.T) {
	body := strings.Repeat("0123456789", 400)
	h := md5.Sum([]byte(body))
	sum := hex.EncodeToString(h[:])
	ranged := true
	hold := make(chan struct{})
	defer close(hold)
	srv := gogServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/f":
			if ranged {
				http.ServeContent(w, r, "f", time.Time{}, strings.NewReader(body))
				return
			}
			fmt.Fprint(w, body) // ignores Range
		case "/slow":
			w.Header().Set("Content-Length", strconv.Itoa(len(body)))
			fmt.Fprint(w, body[:500])
			w.(http.Flusher).Flush()
			<-hold
		}
	})
	ctx := context.Background()
	dst := filepath.Join(t.TempDir(), "f.bin")
	l := GogLink{URL: srv.URL + "/f", Size: int64(len(body)), MD5: sum}

	ranged = false
	write(t, dst+".part", "garbage that isn't the start")
	if err := GogDownload(ctx, l, dst, nil); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(dst); string(b) != body {
		t.Error("a server that ignores Range: the download didn't start over")
	}
	ranged = true
	write(t, dst+".part", "garbage of the right length..")
	if err := GogDownload(ctx, l, dst, nil); err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Errorf("a bad .part resumed: %v", err)
	}
	if _, err := os.Stat(dst + ".part"); !os.IsNotExist(err) {
		t.Error("the .part stayed after a bad checksum")
	}
	// Cancelled part way: GOG's .part stays, itch.io's doesn't.
	for _, keep := range []bool{true, false} {
		c, cancel := context.WithCancel(ctx)
		d := downloader{client: gogClient(0), store: "GOG", status: gogStatus, keep: keep}
		err := d.get(c, srv.URL+"/slow", dst, int64(len(body)), "", func(done, total int64) {
			if done >= 500 {
				cancel()
			}
		})
		if err == nil {
			t.Fatal("a cancelled download went on")
		}
		if _, serr := os.Stat(dst + ".part"); keep == os.IsNotExist(serr) {
			t.Errorf("keep %v: .part there = %v", keep, serr == nil)
		}
	}
}

func TestGogDownloadTraceAndRefusal(t *testing.T) {
	var traced []string
	GogTrace = func(host, path string) { traced = append(traced, host+path) }
	defer func() { GogTrace = nil }()
	refuse := false
	srv := gogServer(t, func(w http.ResponseWriter, r *http.Request) {
		if refuse {
			http.Error(w, "expired", http.StatusForbidden)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/r/") {
			http.Redirect(w, r, "/token=nva=1~token=SIGNED/secure/f.bin", http.StatusFound)
			return
		}
		fmt.Fprint(w, "file bytes")
	})
	ctx := context.Background()
	dst := filepath.Join(t.TempDir(), "f.bin")
	if err := GogDownload(ctx, GogLink{URL: srv.URL + "/r/token=nva=1~token=SIGNED/x"}, dst, nil); err != nil {
		t.Fatal(err)
	}
	for _, p := range traced {
		if strings.Contains(p, "token=") {
			t.Errorf("traced %q", p)
		}
	}
	// A refused link keeps what's downloaded and doesn't ask to sign in again.
	refuse = true
	write(t, dst+".part", "half")
	err := GogDownload(ctx, GogLink{URL: srv.URL + "/f", Size: 10}, dst, nil)
	if err == nil || err == ErrGogAuth || !strings.Contains(err.Error(), "ran out") {
		t.Errorf("refused link: %v", err)
	}
	if _, err := os.Stat(dst + ".part"); err != nil {
		t.Error("the .part went after a refused link")
	}
}

func TestGogUpdate(t *testing.T) {
	ins := []GogInstaller{{ID: "en1installer0", Language: "en", Version: "1.3", Files: []GogFile{{ID: "a"}}}, {ID: "de1installer0", Language: "de", Version: "1.2", Files: []GogFile{{ID: "b"}}}}
	for name, c := range map[string]struct {
		rec  GogRecord
		want string
	}{
		"newer in its language":  {GogRecord{InstallerID: "en1installer0", Language: "en", Version: "1.2"}, "en1installer0"},
		"same version":           {GogRecord{InstallerID: "de1installer0", Language: "de", Version: "1.2"}, ""},
		"language gone, same id": {GogRecord{InstallerID: "de1installer0", Language: "pl", Version: "1.1"}, "de1installer0"},
		"nothing like it":        {GogRecord{InstallerID: "pl1installer0", Language: "pl", Version: "1.1"}, ""},
		"no version recorded":    {GogRecord{InstallerID: "en1installer0", Language: "en"}, "en1installer0"},
	} {
		in, ok := GogUpdate(c.rec, ins)
		if got := map[bool]string{true: in.ID, false: ""}[ok]; got != c.want {
			t.Errorf("%s: update = %q", name, got)
		}
	}
	if _, ok := GogUpdate(GogRecord{Language: "en", Version: "1"}, []GogInstaller{{ID: "x", Language: "en"}}); ok {
		t.Error("an installer with no version is an update")
	}
}

func TestGogLangAndImage(t *testing.T) {
	for in, want := range map[string]string{"ro-RO": "ro", "en-US": "en", "pt-BR": "br", "pt-PT": "pt", "ja-JP": "jp", "cs-CZ": "cz", "zh-Hans-CN": "cn", "": ""} {
		if got := GogLang(in); got != want {
			t.Errorf("GogLang(%q) = %q", in, got)
		}
	}
	ins := []GogInstaller{{ID: "de", Language: "de"}, {ID: "en", Language: "en"}}
	if GogDefault(ins, "de") != "de" || GogDefault(ins, "ro") != "en" || GogDefault(ins[:1], "ro") != "de" || GogDefault(nil, "en") != "" {
		t.Error("GogDefault")
	}
	for in, want := range map[string]string{
		"//images-4.gog-statics.com/abc":             "https://images-4.gog-statics.com/abc_392.jpg",
		"https://images.gog-statics.com/abc_196.jpg": "https://images.gog-statics.com/abc_196.jpg",
		"http://images.gog-statics.com/abc":          "",
		"//evil.example/abc":                         "",
		"//images.gog-statics.com.evil.example/a":    "",
		"": "",
	} {
		if got := GogImage(in); got != want {
			t.Errorf("GogImage(%q) = %q", in, got)
		}
	}
}

func TestGogRecords(t *testing.T) {
	s := &Store{Dir: t.TempDir()}
	rec := GogRecordOf(42, GogInstaller{ID: "en1installer0", Language: "en", Version: "1.2", Size: 9}, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err := s.SetGogRecord("dir:c:\\games\\g", rec); err != nil {
		t.Fatal(err)
	}
	if err := s.RekeyGame("dir:c:\\games\\g", "dir:d:\\games\\g"); err != nil {
		t.Fatal(err)
	}
	all := s.GogRecords()
	if _, old := all["dir:c:\\games\\g"]; old || all["dir:d:\\games\\g"].ProductID != 42 || all["dir:d:\\games\\g"].Version != "1.2" {
		t.Errorf("after the rekey: %+v", all)
	}
	write(t, s.GogPath(), "{broken")
	if err := s.SetGogRecord("k", rec); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(s.GogPath() + ".broken"); string(b) != "{broken" {
		t.Error("the unreadable gog.json wasn't kept")
	}
}
