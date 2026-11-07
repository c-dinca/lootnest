package core

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestPublisherID(t *testing.T) {
	// Every Microsoft package ends in _8wekyb3d8bbwe.
	if id := PublisherID("CN=Microsoft Corporation, O=Microsoft Corporation, L=Redmond, S=Washington, C=US"); id != "8wekyb3d8bbwe" {
		t.Fatalf("got %q", id)
	}
}

func fakeGame(t *testing.T, dir, exe string) {
	t.Helper()
	write(t, filepath.Join(dir, exe), "MZ"+string(make([]byte, 300<<10)))
}

func TestLauncherGames(t *testing.T) {
	root := t.TempDir()
	ac, valorant, ow, sims, gta, amazon, app := filepath.Join(root, "AC"), filepath.Join(root, "Riot Games", "VALORANT"),
		filepath.Join(root, "Overwatch"), filepath.Join(root, "The Sims 4"), filepath.Join(root, "GTAV"), filepath.Join(root, "NWorld"), filepath.Join(root, "EA app")
	for d, exe := range map[string]string{ac: "ACValhalla.exe", valorant: "VALORANT.exe", ow: "Overwatch.exe", sims: "TS4_x64.exe", gta: "GTA5.exe", amazon: "NewWorld.exe", app: "EADesktop.exe"} {
		fakeGame(t, d, exe)
	}
	client := filepath.Join(root, "Riot Games", "Riot Client", "RiotClientServices.exe")
	write(t, client, "MZ")
	got := LauncherGames([]UninstallEntry{
		{Key: "Uplay Install 13504", DisplayName: "Assassin's Creed Valhalla", InstallLocation: ac},
		{Key: "Riot Game valorant.live", DisplayName: "VALORANT", InstallLocation: valorant,
			UninstallString: `"` + client + `" --uninstall-product=valorant --uninstall-patchline=live`},
		{Key: "Overwatch", DisplayName: "Overwatch", InstallLocation: ow,
			UninstallString: `"C:\ProgramData\Battle.net\Agent\Blizzard Uninstaller.exe" --lang=enUS --uid=prometheus --displayname="Overwatch"`},
		{Key: "{abc}", DisplayName: "The Sims 4", Publisher: "Electronic Arts", InstallLocation: sims},
		{Key: "{eaapp}", DisplayName: "EA app", Publisher: "Electronic Arts", InstallLocation: app},
		{Key: "{gta}", DisplayName: "Grand Theft Auto V", Publisher: "Rockstar Games", InstallLocation: gta},
		{Key: "AmazonGames/New World", DisplayName: "New World", InstallLocation: amazon,
			UninstallString: `"C:\Users\x\AppData\Local\Amazon Games\App\Amazon Game Remover.exe" -m Game -p 9e5f0b4d-1d0a-4f8a-8a2b-3c8f1e0f9b11`},
		{Key: "{7zip}", DisplayName: "7-Zip", Publisher: "Igor Pavlov", InstallLocation: root},
		{Key: "Uplay Install 99", DisplayName: "Gone", InstallLocation: filepath.Join(root, "missing")},
	})
	by := map[string]FoundGame{}
	for _, f := range got {
		by[f.Source] = f
	}
	if len(got) != 6 {
		t.Fatalf("six games, no EA app, no 7-Zip, nothing missing: %+v", got)
	}
	if by["ubisoft"].URI != "uplay://launch/13504/0" || by["battlenet"].URI != "battlenet://Pro" || by["amazon"].URI != "amazon-games://play/9e5f0b4d-1d0a-4f8a-8a2b-3c8f1e0f9b11" {
		t.Fatalf("launch links: %+v", by)
	}
	if r := by["riot"]; r.Exe != client || r.Args != "--launch-product=valorant --launch-patchline=live" || r.ProcessName != "VALORANT" {
		t.Fatalf("riot: %+v", r)
	}
	if by["ea"].Name != "The Sims 4" || filepath.Base(by["ea"].Exe) != "TS4_x64.exe" || by["rockstar"].ProcessName != "GTA5" {
		t.Fatalf("ea and rockstar start from their exe: %+v %+v", by["ea"], by["rockstar"])
	}
	for _, f := range got {
		if f.URI != "" && !SafeLaunchURI(f.URI) {
			t.Fatalf("%s's link isn't accepted: %s", f.Name, f.URI)
		}
	}
}

func TestXboxGames(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "Forza Horizon 5", "Content")
	fakeGame(t, dir, "ForzaHorizon5.exe")
	write(t, filepath.Join(dir, "MicrosoftGame.config"), `<?xml version="1.0" encoding="utf-8"?>
<Game configVersion="1">
  <Identity Name="Microsoft.624F8B84B80" Publisher="CN=Microsoft Corporation, O=Microsoft Corporation, L=Redmond, S=Washington, C=US" Version="3.4.0.0"/>
  <ExecutableList><Executable Name="ForzaHorizon5.exe" Id="ForzaHorizon5" TargetDeviceFamily="PC"/></ExecutableList>
  <ShellVisuals DefaultDisplayName="Forza Horizon 5" PublisherDisplayName="Microsoft Studios"/>
</Game>`)
	write(t, filepath.Join(root, "Broken", "Content", "MicrosoftGame.config"), "<Game>")
	got := XboxGames([]string{root, filepath.Join(root, "nope")})
	if len(got) != 1 {
		t.Fatalf("%+v", got)
	}
	f := got[0]
	if f.Name != "Forza Horizon 5" || f.Source != "xbox" || f.ProcessName != "ForzaHorizon5" ||
		f.URI != `shell:AppsFolder\Microsoft.624F8B84B80_8wekyb3d8bbwe!ForzaHorizon5` || !SafeLaunchURI(f.URI) {
		t.Fatalf("%+v", f)
	}
}

func TestItchGames(t *testing.T) {
	apps := t.TempDir()
	dir := filepath.Join(apps, "celeste")
	fakeGame(t, dir, "Celeste.exe")
	_ = os.MkdirAll(filepath.Join(dir, ".itch"), 0o755)
	f, _ := os.Create(filepath.Join(dir, ".itch", "receipt.json.gz"))
	zw := gzip.NewWriter(f)
	_, _ = zw.Write([]byte(`{"game":{"title":"Celeste","id":1}}`))
	zw.Close()
	f.Close()
	fakeGame(t, filepath.Join(apps, "no-receipt"), "x.exe")
	got := ItchGames(apps)
	if len(got) != 1 || got[0].Name != "Celeste" || got[0].Source != "itch" || filepath.Base(got[0].Exe) != "Celeste.exe" {
		t.Fatalf("%+v", got)
	}
}

func TestSafeLaunchURI(t *testing.T) {
	for u, ok := range map[string]bool{
		"uplay://launch/13504/0":                                    true,
		"battlenet://Pro":                                           true,
		"amazon-games://play/amzn1.adg.product.x":                   false,
		"amazon-games://play/9e5f0b4d-1d0a":                         true,
		`shell:AppsFolder\Microsoft.624F8B84B80_8wekyb3d8bbwe!Game`: true,
		"com.epicgames.launcher://apps/fn%3A4fe75bbc5a674f4f9b356b5c90567da5%3AFortnite?action=launch&silent=true": true,
		"file:///C:/Windows/System32/cmd.exe":        false,
		"https://example.com":                        false,
		"uplay://launch/1/0 & calc":                  false,
		`shell:AppsFolder\..\..\cmd_8wekyb3d8bbwe!x`: false,
		"steam://run/730":                            false,
	} {
		if SafeLaunchURI(u) != ok {
			t.Errorf("SafeLaunchURI(%q) = %v", u, !ok)
		}
	}
}

func TestEpicLaunchURI(t *testing.T) {
	dir := t.TempDir()
	game := filepath.Join(dir, "Fortnite")
	fakeGame(t, filepath.Join(game, "FortniteGame", "Binaries", "Win64"), "FortniteLauncher.exe")
	write(t, filepath.Join(dir, "m", "a.item"), `{"DisplayName":"Fortnite","InstallLocation":"`+filepath.ToSlash(game)+`",
		"LaunchExecutable":"FortniteGame/Binaries/Win64/FortniteLauncher.exe","AppName":"Fortnite","CatalogNamespace":"fn",
		"CatalogItemId":"4fe75bbc5a674f4f9b356b5c90567da5","AppCategories":["public","games"]}`)
	got := EpicGames(filepath.Join(dir, "m"))
	if len(got) != 1 || got[0].URI != "com.epicgames.launcher://apps/fn%3A4fe75bbc5a674f4f9b356b5c90567da5%3AFortnite?action=launch&silent=true" {
		t.Fatalf("%+v", got)
	}
}
