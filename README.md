<p align="center"><img src="assets/lootnest.png" width="96" alt=""></p>

<h1 align="center">Lootnest</h1>

<p align="center">A home for every game on your PC.<br>See them all in one place, play anything, and give everyone who shares the PC their own settings and saves.</p>

<p align="center"><a href="../../releases/latest"><b>Download for Windows</b></a> · <a href="CHANGELOG.md">What's new</a></p>

---

![The Library: every game on the PC with its cover, and whose files are loaded in each](docs/screenshots/library.png)

Lootnest finds the games on your PC (Steam, Epic, GOG and the rest) and puts them in one library. For the games you share, turn on **profiles**: press **Play**, pick who's playing, and Lootnest puts that person's resolution, sensitivity, keybinds and saves in place before the game starts. When the game closes, their progress goes back to their own profile.

It's made for couples, siblings and housemates who share a gaming PC and one Steam account, and keep overwriting each other's settings.

> **Lootnest was called PlaySwap** until version 1.3. Installing Lootnest over PlaySwap carries over players, games, profiles, backups and the Steam set-up.

## Features

**Library**
- Every game on the PC in one grid, with search and a filter for each store: Steam, Epic, GOG, Xbox and PC Game Pass, EA app, Ubisoft Connect, Battle.net, Riot, Amazon Games, Rockstar, itch.io, game folders on every drive, and your shortcuts. Games a launcher must start (Ubisoft, Battle.net, Xbox, Amazon, Epic) start through it, like from the launcher.
- Covers come from Steam's own cache on your PC. For the rest, turn on **Download covers and artwork**: covers, wide art and logos from the Steam store, and from SteamGridDB for any other game if you add your free API key.
- **Home** opens on the game you played last, in its own artwork, with a Play button for each player, then **Continue playing** with the other recent games and **News from your games**. The **Library** has all your games.
- The window takes on the colours of the game you choose. Sort by name, recently played or most played, pick the cover size, and right-click a game for everything you can do with it. Ctrl+F finds a game.
- Lootnest counts how long each player has played each game, shown on the game's page in the players' colours.
- Every game has its own page: its art, who has a profile, whose files are loaded, and everything else about it. Rename it, add launch options, open its folder, put a shortcut on the desktop, or have Steam check its files. None of this needs profiles.
- Games without a cover show their own icon, read from the game on your PC.
- **Updates and news**, if you turn it on: each game's page lists its latest updates and patch notes from Steam's news feed. For a game no store keeps up to date, Lootnest says when a newer version than yours is out.

**Profiles**, per game, when you want them
- Everyone gets their own settings and saves, on one Windows account and one Steam account.
- Works with any game that keeps its settings and saves in files. Lootnest finds where most games keep them.
- Steam's own Play button can ask "Who's playing?" too, after a one-time automatic setup.
- Desktop shortcuts that ask every time.

**Safe by design**
- A backup before every swap, no half-written files, and folders too broad to swap are refused.
- **Steam Cloud can stay on.** Each game says whether Steam Cloud syncs its swapped files. If Steam ever replaces a player's files with the cloud's copy, Lootnest says so and keeps that player's profile as it was.
- **Anti-cheat friendly.** It never injects into, hooks or reads the memory of a game. With FACEIT AC, one click switches a game to being started by Steam (see [below](#anti-cheats-faceit-ac)).

**Game Mover**
- Move a Steam game to another of your Steam libraries, such as a bigger or faster drive, from its panel. The game is copied and checked before anything is removed, and Steam is closed and reopened for you.

**Stays out of the way**
- Lives in the tray and notices games started outside Lootnest.
- Keeps itself up to date if you let it (it asks once): a notification when a new version is out, installed while Lootnest is in the tray and no game is running. Or **Settings > Check for updates**, then **Update now**.

**Coming next:** Game Mover for games outside Steam, cloud backup of profiles, and achievements.

![A game's panel: play, load someone's files, and the game's options](docs/screenshots/game-panel.png)

## Install

1. Download `Lootnest-Setup.exe` from the [latest release](../../releases/latest) and run it.
2. If SmartScreen says it protected your PC, choose **More info**, then **Run anyway**. Lootnest isn't code-signed yet.
3. Name the players. The Library fills itself; turn on profiles for the games you share.

You need Windows 10 or 11 (64-bit) with the Microsoft Edge WebView2 Runtime, which comes with Windows 11 and current Windows 10. No administrator rights are needed.

To update, use **Settings > Check for updates**. Your players, games and profiles are kept. To uninstall, use Windows **Settings > Apps > Installed apps**.

## Security

How Lootnest protects your files, its window and its updates is in [SECURITY.md](SECURITY.md), along with how to report a problem privately.

## Privacy

No account and no telemetry; Lootnest works offline. It only goes online when you ask it to:

| What | When | What is sent |
| --- | --- | --- |
| **Check for updates** | When you press it, or every 6 hours if **Check for updates automatically** is on (off until you answer "Keep Lootnest up to date?") | A request for the latest Lootnest version, to `api.github.com` |
| **Update now** | When you press it | Downloads the installer from the GitHub release, and checks its SHA-256 before running it |
| **Download covers and artwork** | Only if you turn it on (off by default) | The game's name, to `store.steampowered.com`; covers, wide art and logos come from Steam's image server and are saved on your PC |
| **SteamGridDB API key** (optional, in Settings) | With covers on, for games the Steam store doesn't know | The game's name or Steam id and your key, to `www.steamgriddb.com`; images come from its CDN and are saved on your PC |
| **Steam Web API key** (optional, in Settings) | When you open a game's page and its achievements on this PC have no names (a game in a folder, or Steam's partial list); each game's list is kept for a week | The game's Steam app id and your key, to `api.steampowered.com`; the names, descriptions and picture links are saved on your PC |
| **Show game updates and news** | Only if you turn it on (off by default), when you open a game's page; each feed is read at most every 6 hours | The game's Steam app id to `api.steampowered.com` (for a game outside Steam, first its name to `store.steampowered.com`); an update's pictures load from Steam's image server when you open it |
| **Find save locations online** (Settings, Online, or Cloud saves) | Only if you turn it on (off by default), then about once a week | Nothing about you or your games: Lootnest downloads the Ludusavi project's list of where games keep their saves (from PCGamingWiki) from `raw.githubusercontent.com` and keeps it on your PC |

**Cloud saves and backup** never go online themselves: Lootnest writes the backup file to the folder you choose. If that folder is one OneDrive, Google Drive or Dropbox syncs, that app uploads it, under that app's own terms. With a password, the file is encrypted on your PC first (AES-256-GCM), so the cloud only ever has the encrypted file.

## How profiles work

![The picker that opens before a game starts](docs/screenshots/picker.png)

1. You start a game from the Library, from Steam, or from a Lootnest desktop shortcut.
2. Lootnest asks who's playing.
3. It saves the files currently on the PC back to whoever they belong to, backs them up, and copies in the chosen player's files.
4. The game starts. Lootnest watches its process.
5. When the game closes, Lootnest saves the files to that player's profile.

For Steam's Play button, Lootnest sets the game's launch options to `"…\Lootnest.exe" --steam <app id> -- %command%`. It edits Steam's `localconfig.vdf` only while Steam is closed, changes only `LaunchOptions`, and keeps a backup. Options you already had are kept, and they're put back when you turn profiles off or uninstall Lootnest.

### Anti-cheats (FACEIT AC)

Some anti-cheats only let Steam start the game, so with FACEIT AC running, CS2 can't start through Lootnest's launch option. When that happens, Lootnest offers **Fix it and start the game**. It switches the game to **Let Steam start this game** (also in **Edit game**), takes itself out of the launch options, and starts the game for the chosen player through Steam.

From then on, start the game from Lootnest or from the **<game> (Lootnest)** desktop shortcut, and both ask who's playing. Steam's own Play button can't ask for these games, so if you use it anyway, a notification says whose settings are loaded.

### What profiles can't separate

Lootnest swaps files on your PC. Anything that belongs to the Steam account stays shared: achievements, playtime, friends, inventory and online ranks. So do settings a game keeps in the Windows registry or on its own servers. If everyone can have their own Steam account, that separates everything; profiles are for when you share one.

## Adding a game by hand

If the scan misses a game, use **Add game** and choose the Steam game or the .exe. Lootnest fills in the save locations it finds: with **Find save locations online** on, first the ones PCGamingWiki lists for the game, then folders named like it. Add any others yourself. [PCGamingWiki](https://www.pcgamingwiki.com) lists them for most PC games, in the "Game data" section. Paths can use `%APPDATA%`, `%LOCALAPPDATA%`, `{LOCALLOW}`, `{DOCUMENTS}`, `{SAVEDGAMES}`, `{STEAM}` and `{STEAMUSER}`.

Data lives in `%LOCALAPPDATA%\Lootnest`:
- `profiles\<player>\<game>`
- `backups\<game>\<date-time>`
- `covers`
- `avatars` and `banners` (pictures for player profiles)
- `shelf.json` and `activity.json` (each player's shelf and time played per day)
- `backup.json` (when the last backup to your folder was made)
- `sync.json` (what each player's cloud saves were at the last sync)
- `gamesaves.json` and `savedb.json.gz` (cloud saves of games outside Steam, and the save locations list)
- `config.json`
- `lootnest.log`

## Building from source

You need Go (go.mod's `toolchain` line names the release; recent Go downloads it by itself), Python 3 and bash (Git Bash on Windows). Pillow is optional; it only redraws the icon.

```bash
./build.sh                 # dist/Lootnest-Setup.exe and dist/Lootnest.exe
VERSION=1.4.0 ./build.sh   # stamps a version (default: the VERSION file)
go test ./internal/core/   # tests
```

It cross-compiles from Windows, macOS or Linux with no C compiler. To release, bump `VERSION`, rename `## Unreleased` in CHANGELOG.md to the new version, and merge to `main`: GitHub Actions builds both .exe files and publishes the release with that section as its notes. Pushing a `v*` tag publishes one too.

| Folder | Contents |
| --- | --- |
| `internal/core` | Swap engine, path safety, Steam, Steam Cloud, Game Mover, covers, updates, game scanning (platform-neutral, tested) |
| `internal/win` | Windows calls: tray, file dialogs, shortcuts, registry |
| `internal/app` | The app; its interface is plain HTML, CSS and JS in `internal/app/ui` |
| `cmd/lootnest`, `cmd/setup` | The app and installer entry points |
| `tools` | Icon, Windows resources (icon, manifest, version), exe checks |
| `third_party` | Vendored, lightly patched go-webview2, go-winloader and golang.org/x/sys |

## Contributing

Bug reports and pull requests are welcome; see [CONTRIBUTING.md](CONTRIBUTING.md). The most useful small contribution is a **game preset**: a game's process name and settings and save locations, added to `KnownSteamGames` in `internal/core/steam.go`.

## License

[MIT](LICENSE). Third-party licenses are in [assets/THIRD_PARTY_NOTICES.txt](assets/THIRD_PARTY_NOTICES.txt).
