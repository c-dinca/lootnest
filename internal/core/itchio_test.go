package core

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const itchKey = "SECRETkey0123456789"

// itchServer stands in for api.itch.io over https, with Lootnest's client
// pointed at it.
func itchServer(t *testing.T, h http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewTLSServer(h)
	api, tr, pause := ItchAPI, itchTransport, itchPause
	ItchAPI, itchTransport, itchPause = srv.URL, srv.Client().Transport, 0
	t.Cleanup(func() { ItchAPI, itchTransport, itchPause = api, tr, pause; srv.Close() })
	return srv
}

func noKey(t *testing.T, err error) {
	t.Helper()
	if err != nil && strings.Contains(err.Error(), itchKey) {
		t.Errorf("the key is in an error: %v", err)
	}
}

func TestItchOwnedPages(t *testing.T) {
	var pages []string
	itchServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+itchKey || r.URL.Path != "/profile/owned-keys" {
			http.Error(w, "no", http.StatusUnauthorized)
			return
		}
		pages = append(pages, r.URL.Query().Get("page"))
		if r.URL.Query().Get("page") == "1" {
			var keys []string
			for i := 1; i <= 48; i++ {
				keys = append(keys, fmt.Sprintf(`{"id":%d,"game_id":%d,"game":{"id":%d,"title":"Game %d","classification":"game"}}`, 1000+i, i, i, i))
			}
			keys = append(keys, `{"id":2000,"game_id":1,"game":{"id":1,"title":"Game 1 again"}}`,
				`{"id":2001,"game_id":900,"game":{"id":900,"title":"Some asset pack","classification":"assets"}}`)
			fmt.Fprintf(w, `{"page":1,"per_page":50,"owned_keys":[%s]}`, strings.Join(keys, ","))
			return
		}
		fmt.Fprint(w, `{"page":2,"perPage":50,"ownedKeys":[{"id":3001,"gameId":77,"createdAt":"2024-01-02T03:04:05Z","game":{"id":77,"title":" Camel Quest ","coverUrl":"https://img.itch.zone/c.png","shortText":"A game","url":"https://dev.itch.io/camel"}},
			{"id":3002,"game":{"id":78,"title":"No classification"}}]}`)
	})
	got, err := ItchOwned(context.Background(), itchKey)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(pages, ",") != "1,2" {
		t.Fatalf("pages asked: %v", pages)
	}
	if len(got) != 50 {
		t.Fatalf("%d games, want 50 (48, then two camelCase; no duplicate, no assets)", len(got))
	}
	c := got[48]
	if c.KeyID != 3001 || c.GameID != 77 || c.Title != "Camel Quest" || c.CoverURL != "https://img.itch.zone/c.png" || c.ShortText != "A game" || c.URL == "" {
		t.Errorf("camelCase game = %+v", c)
	}
	if got[0].KeyID != 1001 || got[49].GameID != 78 {
		t.Errorf("first %+v, last %+v", got[0], got[49])
	}
}

func TestItchUploadShapes(t *testing.T) {
	itchServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/games/5/uploads" || r.URL.Query().Get("download_key_id") != "9" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `{"uploads":[
			{"id":1,"filename":"game-win.zip","size":100,"type":"default","p_windows":true,"md5_hash":"abc","build_id":12,"build":{"id":12,"user_version":"1.2.0"},"updated_at":"2024-05-01T10:00:00.000000000Z"},
			{"id":2,"filename":"demo.zip","type":"default","traits":["p_windows","demo"],"updated_at":"2024-05-01 10:00:00"},
			{"id":3,"filename":"game.7z","platforms":{"windows":"all"}},
			{"id":4,"displayName":"Linux build","filename":"game.tar.gz","p_linux":true,"md5Hash":"x","buildId":"44","updatedAt":"2024-06-01T00:00:00Z"},
			{"id":5,"filename":"ost.zip","type":"soundtrack","pWindows":true,"preorder":true}
		]}`)
	})
	ups, err := ItchUploads(context.Background(), itchKey, 5, 9)
	if err != nil {
		t.Fatal(err)
	}
	if len(ups) != 5 {
		t.Fatalf("%d uploads", len(ups))
	}
	a, b, c, d, e := ups[0], ups[1], ups[2], ups[3], ups[4]
	if !a.Windows || a.BuildID != 12 || a.UserVersion != "1.2.0" || a.MD5 != "abc" || a.Size != 100 || a.UpdatedAt.IsZero() {
		t.Errorf("snake_case upload = %+v", a)
	}
	if !b.Windows || !b.Demo || b.UpdatedAt.IsZero() || b.Type != "default" {
		t.Errorf("traits upload = %+v", b)
	}
	if !c.Windows || c.Type != "default" {
		t.Errorf("platforms upload = %+v", c)
	}
	if d.Windows || d.DisplayName != "Linux build" || d.BuildID != 44 || d.MD5 != "x" || d.UpdatedAt.IsZero() {
		t.Errorf("camelCase upload = %+v", d)
	}
	if !e.Windows || !e.Preorder || e.Type != "soundtrack" {
		t.Errorf("soundtrack = %+v", e)
	}
}

func TestItchPickUpload(t *testing.T) {
	at := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ups := []ItchUpload{
		{ID: 1, Filename: "game-win32.zip", Type: "default", Windows: true, UpdatedAt: at.Add(3 * time.Hour)},
		{ID: 2, Filename: "game-demo-x64.zip", Type: "default", Windows: true, Demo: true},
		{ID: 3, Filename: "game.zip", Type: "default", Windows: true, UpdatedAt: at},
		{ID: 4, Filename: "game-x64.zip", Type: "default", Windows: true},
		{ID: 5, Filename: "game-linux.zip", Type: "default"},
		{ID: 6, Filename: "ost.zip", Type: "soundtrack", Windows: true},
		{ID: 7, Filename: "game-preorder.zip", Type: "default", Windows: true, Preorder: true},
		{ID: 8, Filename: "game-setup.exe", Type: "default", Windows: true, UpdatedAt: at.Add(time.Hour)},
		{ID: 9, Filename: "game.apk", Type: "default", Windows: true},
	}
	var ids []string
	for _, u := range ItchPickUpload(ups) {
		ids = append(ids, fmt.Sprint(u.ID, u.Kind[:1]))
	}
	if got := strings.Join(ids, ","); got != "4a,8i,3a,1a,2a" {
		t.Errorf("order = %s, want 4a,8i,3a,1a,2a", got)
	}
	for in, want := range map[ItchUpload]string{
		{Filename: "Game.exe"}: "exe", {Filename: "game-win64.exe", DisplayName: "Installer"}: "installer",
		{Filename: "game.msi"}: "installer", {Filename: "game.7z"}: "archive", {Filename: "game.pdf"}: "",
	} {
		if got := ItchKind(in); got != want {
			t.Errorf("ItchKind(%s, %s) = %q, want %q", in.Filename, in.DisplayName, got, want)
		}
	}
}

func TestItchNewer(t *testing.T) {
	at := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	rec := ItchRecord{UploadID: 1, BuildID: 10, Size: 100, Filename: "g.zip", MD5: "aa", UpdatedAt: at}
	same := ItchUpload{ID: 1, BuildID: 10, Size: 100, Filename: "g.zip", MD5: "AA", UpdatedAt: at}
	for name, c := range map[string]struct {
		up   func(ItchUpload) ItchUpload
		want bool
	}{
		"same":                  {func(u ItchUpload) ItchUpload { return u }, false},
		"another upload":        {func(u ItchUpload) ItchUpload { u.ID = 2; return u }, true},
		"higher build":          {func(u ItchUpload) ItchUpload { u.BuildID = 11; return u }, true},
		"later, same file":      {func(u ItchUpload) ItchUpload { u.UpdatedAt = at.Add(time.Hour); return u }, false},
		"later, other size":     {func(u ItchUpload) ItchUpload { u.UpdatedAt, u.Size = at.Add(time.Hour), 101; return u }, true},
		"later, other md5":      {func(u ItchUpload) ItchUpload { u.UpdatedAt, u.MD5 = at.Add(time.Hour), "bb"; return u }, true},
		"later, other filename": {func(u ItchUpload) ItchUpload { u.UpdatedAt, u.Filename = at.Add(time.Hour), "g2.zip"; return u }, true},
		"earlier, other size":   {func(u ItchUpload) ItchUpload { u.UpdatedAt, u.Size = at.Add(-time.Hour), 1; return u }, false},
	} {
		if got := ItchNewer(rec, c.up(same)); got != c.want {
			t.Errorf("%s: newer = %v", name, got)
		}
	}
}

func TestItchErrors(t *testing.T) {
	code := http.StatusUnauthorized
	itchServer(t, func(w http.ResponseWriter, r *http.Request) {
		if code == 200 {
			fmt.Fprint(w, `{"errors":["invalid key"]}`)
			return
		}
		http.Error(w, "no", code)
	})
	ctx := context.Background()
	_, err := ItchProfile(ctx, itchKey)
	if err == nil || !strings.Contains(err.Error(), "didn't accept the API key") || !strings.Contains(err.Error(), "API keys") {
		t.Errorf("401: %v", err)
	}
	noKey(t, err)
	code = http.StatusForbidden
	_, err = ItchOwned(ctx, itchKey)
	if err != errItchKey {
		t.Errorf("403: %v", err)
	}
	code = http.StatusTooManyRequests
	_, err = ItchUploads(ctx, itchKey, 1, 2)
	if err == nil || !strings.Contains(err.Error(), "in a minute") {
		t.Errorf("429: %v", err)
	}
	noKey(t, err)
	code = http.StatusBadGateway
	_, err = ItchProfile(ctx, itchKey)
	if err == nil || !strings.Contains(err.Error(), "502") {
		t.Errorf("502: %v", err)
	}
	noKey(t, err)
	code = 200
	_, err = ItchProfile(ctx, itchKey)
	if err != errItchKey {
		t.Errorf("errors list: %v", err)
	}
	code = http.StatusUnauthorized
	err = ItchDownload(ctx, ItchDownloadURL(itchKey, 3, 4), filepath.Join(t.TempDir(), "g.zip"), 0, "", nil)
	if err != errItchKey {
		t.Errorf("download 401: %v", err)
	}
	// Nothing listens there: the address with the key must not come back.
	ItchAPI = "https://127.0.0.1:1"
	err = ItchDownload(ctx, ItchDownloadURL(itchKey, 3, 4), filepath.Join(t.TempDir(), "g.zip"), 0, "", nil)
	if err == nil {
		t.Fatal("no error from a server that isn't there")
	}
	noKey(t, err)
	_, err = ItchProfile(ctx, itchKey)
	noKey(t, err)
	if u := ItchDownloadURL(itchKey, 3, 4); !strings.Contains(u, "/uploads/3/download?") || !strings.Contains(u, "download_key_id=4") {
		t.Errorf("download link = %s", u)
	}
}

func TestItchProfileTraced(t *testing.T) {
	itchServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"user":{"id":1,"username":"maker","display_name":"The Maker"}}`)
	})
	var traced []string
	ItchTrace = func(host, path string) { traced = append(traced, host+path) }
	defer func() { ItchTrace = nil }()
	name, err := ItchProfile(context.Background(), itchKey)
	if err != nil || name != "maker" {
		t.Fatalf("profile = %q, %v", name, err)
	}
	if len(traced) != 1 || !strings.HasSuffix(traced[0], "/profile") || strings.Contains(traced[0], itchKey) {
		t.Errorf("traced %v", traced)
	}
}

func TestItchDownload(t *testing.T) {
	body := strings.Repeat("game bytes ", 500)
	sum := md5.Sum([]byte(body))
	good := hex.EncodeToString(sum[:])
	srv := itchServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/uploads/1/download":
			http.Redirect(w, r, "/files/g.zip", http.StatusFound)
		case "/uploads/2/download":
			http.Redirect(w, r, "http://example.com/g.zip", http.StatusFound)
		case "/files/g.zip":
			fmt.Fprint(w, body)
		}
	})
	dir := t.TempDir()
	dst := filepath.Join(dir, "downloads", "g.zip")
	ctx := context.Background()

	err := ItchDownload(ctx, ItchDownloadURL(itchKey, 1, 0), dst, int64(len(body)), "0123456789abcdef0123456789abcdef", nil)
	if err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Errorf("md5 mismatch: %v", err)
	}
	if _, serr := os.Stat(dst + ".part"); !os.IsNotExist(serr) {
		t.Error("the .part stayed after a bad checksum")
	}
	if _, serr := os.Stat(dst); !os.IsNotExist(serr) {
		t.Error("a bad download was kept")
	}
	if err := ItchDownload(ctx, ItchDownloadURL(itchKey, 1, 0), dst, int64(len(body))+5, "", nil); err == nil || !strings.Contains(err.Error(), "size") {
		t.Errorf("size mismatch: %v", err)
	}
	if err := ItchDownload(ctx, ItchDownloadURL(itchKey, 2, 0), dst, 0, "", nil); err == nil || !strings.Contains(err.Error(), "https") {
		t.Errorf("http redirect: %v", err)
	}
	if err := ItchDownload(ctx, "http://example.com/g.zip", dst, 0, "", nil); err == nil {
		t.Error("an http link was downloaded")
	}
	ctxC, cancel := context.WithCancel(ctx)
	cancel()
	if err := ItchDownload(ctxC, srv.URL+"/files/g.zip", dst, 0, "", nil); err == nil {
		t.Error("a cancelled download went on")
	}
	if _, serr := os.Stat(dst + ".part"); !os.IsNotExist(serr) {
		t.Error("a cancelled download left its .part")
	}
	var last int64
	if err := ItchDownload(ctx, ItchDownloadURL(itchKey, 1, 0), dst, int64(len(body)), strings.ToUpper(good), func(done, total int64) { last = done }); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(dst); string(b) != body || last != int64(len(body)) {
		t.Errorf("downloaded %d bytes, progress %d", len(b), last)
	}
}

func TestItchInstallArchive(t *testing.T) {
	dir := t.TempDir()
	zp := filepath.Join(dir, "game.zip")
	zipOf(t, zp, map[string]string{"Wrapped Game v1.0/game.exe": "exe", "Wrapped Game v1.0/data/level.pak": "pak"})
	parent := filepath.Join(dir, "Games")

	got, err := ItchInstallArchive(zp, parent, "Wrapped: Game?")
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(parent, "Wrapped- Game-") {
		t.Errorf("installed into %s", got)
	}
	if b, _ := os.ReadFile(filepath.Join(got, "game.exe")); string(b) != "exe" {
		t.Error("the wrapping folder wasn't taken off")
	}
	if b, _ := os.ReadFile(filepath.Join(got, "data", "level.pak")); string(b) != "pak" {
		t.Error("data/level.pak is missing")
	}
	if es, _ := os.ReadDir(parent); len(es) != 1 {
		t.Errorf("parent holds %d entries, want only the game", len(es))
	}

	// Again into the same, now full, folder: refused, nothing written.
	write(t, filepath.Join(got, "game.exe"), "mine")
	if _, err := ItchInstallArchive(zp, parent, "Wrapped: Game?"); err == nil || !strings.Contains(err.Error(), "already has files") {
		t.Errorf("into a full folder: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(got, "game.exe")); string(b) != "mine" {
		t.Error("a file in the folder was overwritten")
	}
	if es, _ := os.ReadDir(parent); len(es) != 1 {
		t.Errorf("a refused install left %d entries", len(es))
	}

	// An empty folder of that name is fine; a broken archive leaves nothing.
	if err := os.MkdirAll(filepath.Join(parent, "Other"), 0o755); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(dir, "bad.zip")
	write(t, bad, "not a zip")
	if _, err := ItchInstallArchive(bad, parent, "Other"); err == nil {
		t.Error("a broken archive installed")
	}
	if es, _ := os.ReadDir(parent); len(es) != 2 {
		t.Errorf("a failed install left %d entries", len(es))
	}
	if _, err := ItchInstallArchive(zp, parent, "Other"); err != nil {
		t.Errorf("into an empty folder: %v", err)
	}

	// A game that is one .exe gets a folder of its own; the download stays.
	exe := filepath.Join(dir, "Tiny.exe")
	write(t, exe, "tiny")
	got, err = ItchInstallArchive(exe, parent, "Tiny")
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(got, "Tiny.exe")); string(b) != "tiny" {
		t.Error("the .exe isn't in its folder")
	}
	if _, err := os.Stat(exe); err != nil {
		t.Error("the downloaded .exe went")
	}
}

func TestItchNames(t *testing.T) {
	for in, want := range map[string]string{`..\..\evil.exe`: "evil.exe", "../x/setup.msi": "setup.msi", "": "download", "con.zip": "download.zip", "...": "download"} {
		if got := filepath.Base(ItchDownloadPath("D", 5, in)); got != want {
			t.Errorf("ItchDownloadPath(%q) = %s", in, got)
		}
	}
	if p := ItchDownloadPath("D", 5, "g.zip"); p != filepath.Join("D", "downloads", "itch", "5", "g.zip") {
		t.Errorf("path = %s", p)
	}
	for in, want := range map[string]string{"": "Game", "  ": "Game", "CON": "CON game", "A. B. ": "A. B", "Doom: Eternal": "Doom- Eternal"} {
		if got := ItchFolderName(in); got != want {
			t.Errorf("ItchFolderName(%q) = %q", in, got)
		}
	}
}

func TestItchRecords(t *testing.T) {
	s := &Store{Dir: t.TempDir()}
	at := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	rec := ItchRecordOf(ItchGame{GameID: 5, KeyID: 9}, ItchUpload{ID: 3, Filename: "g.zip", Size: 10, UserVersion: "1.0"}, at)
	if err := s.SetItchRecord("dir:d:\\games\\g", rec); err != nil {
		t.Fatal(err)
	}
	if err := s.RekeyGame("dir:d:\\games\\g", "dir:e:\\games\\g"); err != nil {
		t.Fatal(err)
	}
	all := s.ItchRecords()
	if _, old := all["dir:d:\\games\\g"]; old || all["dir:e:\\games\\g"].UploadID != 3 || all["dir:e:\\games\\g"].KeyID != 9 {
		t.Errorf("after the rekey: %+v", all)
	}
	if err := s.DeleteItchRecord("dir:e:\\games\\g"); err != nil || len(s.ItchRecords()) != 0 {
		t.Errorf("delete: %v, %v", err, s.ItchRecords())
	}
	write(t, s.ItchPath(), "{broken")
	if err := s.SetItchRecord("k", rec); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(s.ItchPath() + ".broken"); string(b) != "{broken" {
		t.Error("the unreadable itch.json wasn't kept")
	}
}
