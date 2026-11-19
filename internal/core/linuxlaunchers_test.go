//go:build !windows

package core

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mkPrefix makes a Wine prefix with one user and returns its AppData\Roaming.
func mkPrefix(t *testing.T, root, user string) string {
	t.Helper()
	write(t, filepath.Join(root, "system.reg"), "WINE REGISTRY Version 2")
	_ = os.MkdirAll(filepath.Join(root, "drive_c", "windows"), 0o755)
	roaming := filepath.Join(root, "drive_c", "users", user, "AppData", "Roaming")
	if err := os.MkdirAll(roaming, 0o755); err != nil {
		t.Fatal(err)
	}
	return roaming
}

// mkELF writes a small Linux program.
func mkELF(t *testing.T, p string, size int) {
	t.Helper()
	write(t, p, "\x7fELF"+strings.Repeat("x", size))
	if err := os.Chmod(p, 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestSQLiteReader(t *testing.T) {
	rows, err := sqliteTable(filepath.Join("testdata", "lutris", "pga.db"), "games")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 406 {
		t.Fatalf("every row, through interior pages: %d", len(rows))
	}
	byName := map[string]map[string]any{}
	for _, r := range rows {
		n, _ := r["name"].(string)
		byName[n] = r
	}
	ashen := byName["Ashen Ring"]
	if ashen["id"] != int64(401) || ashen["runner"] != "wine" || ashen["installed"] != int64(1) ||
		ashen["configpath"] != "ashen-ring-1700000000" || ashen["playtime"] != 12.5 || ashen["lastplayed"] != int64(1700000123456) {
		t.Fatalf("a row's values: %v", ashen)
	}
	if ashen["executable"] != nil || ashen["discord_id"] != nil {
		t.Fatal("NULL, and a column added after the row, read as nil")
	}
	if byName["Tux Racer"]["year"] != int64(-12) {
		t.Fatalf("negative numbers: %v", byName["Tux Racer"]["year"])
	}
	if byName["Newer Game"]["discord_id"] != "123" {
		t.Fatal("the column added later")
	}
	long := ""
	for n := range byName {
		if strings.HasPrefix(n, "Long ") {
			long = n
		}
	}
	if len(long) != 3005 {
		t.Fatalf("a row spilling into overflow pages: %d", len(long))
	}

	if _, err := sqliteTable(filepath.Join("testdata", "lutris", "pga.db"), "nothing"); err == nil {
		t.Fatal("a table that isn't there")
	}
	if _, err := parseSQLite([]byte(strings.Repeat("x", 200))); err == nil {
		t.Fatal("not a database")
	}
	// A damaged file never panics.
	b, _ := os.ReadFile(filepath.Join("testdata", "lutris", "pga.db"))
	for _, cut := range []int{100, 1024, 5000, len(b) / 2, len(b) - 1} {
		if db, err := parseSQLite(append([]byte(nil), b[:cut]...)); err == nil {
			_, _ = db.rows(1)
		}
	}
	bad := append([]byte(nil), b...)
	for i := 1024; i < len(bad); i += 97 {
		bad[i] ^= 0x5a
	}
	if db, err := parseSQLite(bad); err == nil {
		_, _ = db.rows(1)
	}
	cols, key := sqliteColumns(`CREATE TABLE "t" ("id" INTEGER PRIMARY KEY, name TEXT DEFAULT 'a,b', size NUMERIC(10, 2), UNIQUE (name))`)
	if strings.Join(cols, ",") != "id,name,size" || key != 0 {
		t.Fatalf("columns: %v %d", cols, key)
	}
}

func TestYAMLMini(t *testing.T) {
	m := readYAMLMap(`game:
  exe: /home/alex/Games/ashen-ring/drive_c/Program Files/Ashen Ring/Game/ashenring.exe
  prefix: '/home/alex/Games/ashen ring''s prefix'
  args: "-windowed \"x\""
system:
  disable_runtime: true
wine:
  version: lutris-GE-Proton8-26-x86_64
  overrides: {}
External_Programs:
  9f6c:
    executable: game.exe
    name: Long Game
    path: C:\Program Files\Some Studio\A Game With A Rather Long Folder Name\Binaries\Win64\the-game-binary-name.exe
      continued here
    removed: false
Installed_Dependencies:
- vcredist2019
- dotnet48
Indented:
  - one
  - two
`)
	if yamlStr(m, "game", "exe") != "/home/alex/Games/ashen-ring/drive_c/Program Files/Ashen Ring/Game/ashenring.exe" {
		t.Fatalf("plain: %q", yamlStr(m, "game", "exe"))
	}
	if yamlStr(m, "game", "prefix") != "/home/alex/Games/ashen ring's prefix" || yamlStr(m, "game", "args") != `-windowed "x"` {
		t.Fatalf("quoted: %q %q", yamlStr(m, "game", "prefix"), yamlStr(m, "game", "args"))
	}
	if yamlStr(m, "system", "disable_runtime") != "true" || yamlStr(m, "wine", "version") != "lutris-GE-Proton8-26-x86_64" {
		t.Fatal("nested maps")
	}
	if !strings.HasSuffix(yamlStr(m, "External_Programs", "9f6c", "path"), "the-game-binary-name.exe continued here") {
		t.Fatalf("a folded line: %q", yamlStr(m, "External_Programs", "9f6c", "path"))
	}
	if l, _ := m["Installed_Dependencies"].([]any); len(l) != 2 || l[1] != "dotnet48" {
		t.Fatalf("PyYAML's list: %v", m["Installed_Dependencies"])
	}
	if l, _ := m["Indented"].([]any); len(l) != 2 || l[0] != "one" {
		t.Fatalf("an indented list: %v", m["Indented"])
	}
}

func TestHeroicGames(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home", "alex")
	dir := filepath.Join(home, ".config", "heroic")
	games := filepath.Join(home, "Games", "Heroic")

	// Epic: a Windows game in its own Proton prefix (GamesConfig names the
	// folder; the prefix is its pfx).
	write(t, filepath.Join(games, "Ashen Ring", "AshenRing.exe"), "MZ")
	mkPrefix(t, filepath.Join(home, "Games", "Heroic", "Prefixes", "ashen", "pfx"), "steamuser")
	write(t, filepath.Join(dir, "legendaryConfig", "legendary", "installed.json"), `{
	  "Quail": {"app_name": "Quail", "title": "Ashen Ring", "install_path": "`+filepath.Join(games, "Ashen Ring")+`", "executable": "AshenRing.exe", "platform": "Windows", "is_dlc": false},
	  "QuailDLC": {"app_name": "QuailDLC", "title": "Ashen Ring DLC", "install_path": "`+filepath.Join(games, "Ashen Ring")+`", "executable": "AshenRing.exe", "platform": "Windows", "is_dlc": true}
	}`)
	write(t, filepath.Join(dir, "GamesConfig", "Quail.json"), `{"Quail": {"winePrefix": "~/Games/Heroic/Prefixes/ashen", "wineVersion": {"type": "proton"}}, "version": "v0"}`)

	// GOG: a native Linux build, named by its info file; and a Windows one
	// in Heroic's default prefix (config.json).
	mkELF(t, filepath.Join(games, "Tux Kart", "bin", "tuxkart"), 4000)
	mkELF(t, filepath.Join(games, "Tux Kart", "bin", "libsteam_api.so"), 9000)
	write(t, filepath.Join(games, "Tux Kart", "goggame-1207.info"), `{"name": "Tux Kart", "playTasks": [{"isPrimary": true, "path": "bin/tuxkart", "type": "FileTask"}]}`)
	write(t, filepath.Join(games, "Quiet Isle", "QuietIsle.exe"), "MZ")
	mkPrefix(t, filepath.Join(home, "Games", "Heroic", "Prefixes", "default", "Quiet Isle"), "alex")
	write(t, filepath.Join(dir, "gog_store", "installed.json"), `{"installed": [
	  {"appName": "1207", "install_path": "`+filepath.Join(games, "Tux Kart")+`", "executable": "", "platform": "linux", "is_dlc": false},
	  {"appName": "1300", "install_path": "`+filepath.Join(games, "Quiet Isle")+`", "executable": "", "platform": "windows", "is_dlc": false},
	  {"appName": "1400", "install_path": "`+filepath.Join(games, "Mac Only")+`", "executable": "", "platform": "osx", "is_dlc": false}
	]}`)
	write(t, filepath.Join(dir, "store_cache", "gog_library.json"), `{"games": [{"app_name": "1300", "title": "Quiet Isle"}]}`)

	// Amazon and a game added by hand.
	write(t, filepath.Join(games, "Fuel Run", "FuelRun.exe"), "MZ")
	write(t, filepath.Join(dir, "nile_config", "nile", "installed.json"), `[{"id": "amzn1.adg", "path": "`+filepath.Join(games, "Fuel Run")+`", "version": "1"}]`)
	write(t, filepath.Join(dir, "nile_config", "nile", "library.json"), `[{"id": "amzn1.adg", "product": {"title": "Fuel Run"}}]`)
	write(t, filepath.Join(home, "Apps", "Own", "own.exe"), "MZ")
	write(t, filepath.Join(dir, "sideload_apps", "library.json"), `{"games": [{"app_name": "own1", "title": "Own Game", "is_installed": true, "install": {"executable": "`+filepath.Join(home, "Apps", "Own", "own.exe")+`", "platform": "Windows"}}]}`)

	got := map[string]FoundGame{}
	for _, f := range HeroicGames(dir, home) {
		got[f.Key] = f
	}
	if len(got) != 5 {
		t.Fatalf("five games (no DLC, no Mac build): %v", got)
	}
	ashen := got["heroic:legendary:Quail"]
	if ashen.Name != "Ashen Ring" || ashen.Prefix != filepath.Join(home, "Games", "Heroic", "Prefixes", "ashen", "pfx") ||
		ashen.URI != "heroic://launch?appName=Quail&runner=legendary" || ashen.ProcessName != "AshenRing" || ashen.Source != "heroic" {
		t.Fatalf("Epic game: %+v", ashen)
	}
	tux := got["heroic:gog:1207"]
	if tux.Name != "Tux Kart" || tux.Prefix != "" || tux.Exe != filepath.Join(games, "Tux Kart", "bin", "tuxkart") {
		t.Fatalf("native GOG game: %+v", tux)
	}
	if q := got["heroic:gog:1300"]; q.Name != "Quiet Isle" || q.Prefix != filepath.Join(home, "Games", "Heroic", "Prefixes", "default", "Quiet Isle") {
		t.Fatalf("GOG game in the default prefix: %+v", q)
	}
	if a := got["heroic:nile:amzn1.adg"]; a.Name != "Fuel Run" || !SafeLaunchURI(a.URI) {
		t.Fatalf("Amazon game: %+v", a)
	}
	if o := got["heroic:sideload:own1"]; o.Name != "Own Game" || o.Dir != filepath.Join(home, "Apps", "Own") {
		t.Fatalf("added by hand: %+v", o)
	}
	if HeroicGames(filepath.Join(home, "nothing"), home) != nil {
		t.Fatal("no Heroic")
	}
}

func TestLutrisAndBottles(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home", "alex")
	data := filepath.Join(home, ".local", "share", "lutris")
	conf := filepath.Join(home, ".config", "lutris")
	db, err := os.ReadFile(filepath.Join("testdata", "lutris", "pga.db"))
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(data, "pga.db"), string(db))

	pfx := filepath.Join(home, "Games", "ashen-ring")
	mkPrefix(t, pfx, "alex")
	exe := filepath.Join(pfx, "drive_c", "Program Files", "Ashen Ring", "ashenring.exe")
	write(t, exe, "MZ")
	write(t, filepath.Join(conf, "games", "ashen-ring-1700000000.yml"), "game:\n  exe: "+exe+"\n  prefix: "+pfx+"\nwine:\n  version: lutris-GE\n")
	tux := filepath.Join(home, "Games", "tux", "tuxracer")
	mkELF(t, tux, 100)
	// Since Lutris 0.5.13 the game files can be in the data folder.
	write(t, filepath.Join(data, "games", "tux-racer-1700000001.yml"), "game:\n  exe: "+tux+"\n")
	write(t, filepath.Join(conf, "games", "steam-game-1.yml"), "game:\n  appid: '1'\n")

	games := LutrisGames(data, conf)
	if len(games) != 2 {
		t.Fatalf("the installed Wine and Linux games: %+v", games)
	}
	if g := games[0]; g.Key != "lutris:401" || g.Name != "Ashen Ring" || g.Prefix != pfx || g.Exe != exe ||
		g.URI != "lutris:rungameid/401" || g.StoreID != "ashen-ring" {
		t.Fatalf("Wine game: %+v", g)
	}
	if g := games[1]; g.Name != "Tux Racer" || g.Prefix != "" || g.ProcessName != "tuxracer" {
		t.Fatalf("Linux game: %+v", g)
	}

	// Bottles: a bottle with two programs, one taken out.
	bottles := filepath.Join(home, ".local", "share", "bottles", "bottles")
	bottle := filepath.Join(bottles, "Gaming")
	mkPrefix(t, bottle, "alex")
	bexe := filepath.Join(bottle, "drive_c", "Games", "Quiet Isle", "Binaries", "Win64", "QuietIsle.exe")
	write(t, bexe, "MZ")
	write(t, filepath.Join(bottle, "drive_c", "Games", "Quiet Isle", "Engine", "Binaries", "x"), "")
	write(t, filepath.Join(bottle, "bottle.yml"), `Name: Gaming
Path: Gaming
Custom_Path: false
Environment: Gaming
External_Programs:
  a1b2:
    executable: QuietIsle.exe
    name: Quiet Isle
    path: C:\Games\Quiet Isle\Binaries\Win64\QuietIsle.exe
    folder: C:\Games\Quiet Isle\Binaries\Win64
    id: a1b2
  c3d4:
    executable: old.exe
    name: Old
    path: C:\old.exe
    removed: true
`)
	write(t, filepath.Join(bottle, "drive_c", "old.exe"), "MZ")
	bs := BottlesGames(bottles)
	if len(bs) != 1 {
		t.Fatalf("one program: %+v", bs)
	}
	if b := bs[0]; b.Name != "Quiet Isle" || b.Exe != bexe || b.Prefix != bottle || b.Key != "bottles:Gaming:a1b2" ||
		b.Dir != filepath.Join(bottle, "drive_c", "Games", "Quiet Isle") {
		t.Fatalf("Bottles program: %+v", b)
	}
	if bottlesPath(bottle, `D:\x.exe`) != filepath.Join(bottle, "dosdevices", "d:", "x.exe") || bottlesPath(bottle, "relative") != "" {
		t.Fatal("other drives")
	}

	// Together, with saves found inside each prefix.
	write(t, filepath.Join(pfx, "drive_c", "users", "alex", "AppData", "Roaming", "Ashen Ring", "save.dat"), "s")
	env := NewLinuxEnv(home, func(k string) string { return map[string]string{"HOME": home, "USER": "alex"}[k] }, filepath.Join(home, ".local", "share", "lootnest"))
	found := env.FinishFound(env.ScanLinuxLaunchers())
	var ashen FoundGame
	for _, f := range found {
		if f.Key == "lutris:401" {
			ashen = f
		}
	}
	if ashen.Prefix != "%HOME%/Games/ashen-ring" || ashen.Exe != "%HOME%/Games/ashen-ring/drive_c/Program Files/Ashen Ring/ashenring.exe" {
		t.Fatalf("stored with tokens: %+v", ashen)
	}
	if len(ashen.Paths) == 0 || ashen.Paths[0] != "%APPDATA%/Ashen Ring" {
		t.Fatalf("its saves, inside its prefix: %+v", ashen.Suggestions)
	}
}

func TestItchLinux(t *testing.T) {
	apps := filepath.Join(t.TempDir(), "itch", "apps")
	dir := filepath.Join(apps, "tiny-game")
	mkELF(t, filepath.Join(dir, "TinyGame.x86_64"), 2000)
	write(t, filepath.Join(dir, "data", "level.pck"), "x")
	writeReceipt(t, filepath.Join(dir, ".itch", "receipt.json.gz"), "Tiny Game")
	if got := ItchGames(apps); len(got) != 0 {
		t.Fatalf("Windows reads only .exe: %+v", got)
	}
	got := itchGames(apps, mainAnyExe)
	if len(got) != 1 || got[0].Name != "Tiny Game" || got[0].ProcessName != "TinyGame" {
		t.Fatalf("a native itch game: %+v", got)
	}
}

func writeReceipt(t *testing.T, p, title string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	zw := gzip.NewWriter(f)
	_, _ = zw.Write([]byte(`{"game": {"title": "` + title + `"}}`))
	_ = zw.Close()
	_ = f.Close()
}

// FuzzSQLite: whatever the file holds, reading it never panics.
func FuzzSQLite(f *testing.F) {
	if b, err := os.ReadFile(filepath.Join("testdata", "lutris", "pga.db")); err == nil {
		f.Add(b)
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		db, err := parseSQLite(b)
		if err != nil {
			return
		}
		rows, err := db.rows(1)
		if err != nil {
			return
		}
		for _, r := range rows {
			if len(r.values) >= 4 {
				if root, ok := r.values[3].(int64); ok && root > 0 && root < 1<<31 {
					_, _ = db.rows(uint32(root))
				}
			}
		}
	})
}
