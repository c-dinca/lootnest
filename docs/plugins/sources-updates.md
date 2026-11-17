# Sources and updates: findings and plan

Status: a proposal, waiting for the owner's answer on the three decisions below. Nothing here is built.

The request: the user adds their own feeds (JSON files listing downloads), Lootnest matches them to games already in the Library (games Hydra installed above all), says when a feed lists a newer version, downloads it and replaces the game's folder with the saves kept. Lootnest ships no feed, suggests none, and does nothing about DRM.

## Three decisions before any code

### 1. It can't be a plugin

A Lootnest plugin is data, never code (`docs/PLUGINS.md`, `core/plugins.go`): a .zip of `.json`, `.ini`, `.vdf`, `.yaml`, `.txt` and images, checked file by file. It has no lifecycle, no API, no settings and no place in the UI beyond what its four kinds (`sources`, `saves`, `metadata`, `theme`) draw. A plugin can't go online, download, write a game's folder or write saves, and the roadmap lists exactly those under "Will not do" for every later phase, script plugins included.

This feature needs all four. So it has to be one of:

- **A. Part of Lootnest itself** (recommended): Go in `internal/core` and `internal/app`, a part in `ui/js`, off until the user adds a first source. It reuses the update centre, the save copies and the journals that exist, and it is the only route where the safety rules ("a copy before anything moves") are enforced by code that has tests.
- **B. Script plugins first** (roadmap Plugins 4, XL) with a new API that may download and replace folders. That undoes the plugin system's security model for every plugin, to host one feature. Not recommended.

The name "plugin" can stay in the UI sense (a feature that isn't there until turned on), but the code would be A.

### 2. It reverses a line Lootnest states in public

These say the opposite today, and each would need the owner's new wording:

| Where | What it says |
|---|---|
| `README.md`, Never | "It never downloads games, except ones you own from itch.io or GOG, through their own API" |
| `CLAUDE.md`, The product and the line | never downloads, "never other sources, never a catalogue", "never shows where to get games" |
| `docs/ROADMAP.md`, 7 and 8 | "Any source other than the store the user bought from"; "Find, suggest or download updates from anywhere" under Will not do |
| `docs/PRIVACY.md`, the website's Never section | the same promise |
| 1.33's note on Hydra | "Lootnest never offers Hydra's downloads or sources" |

The owner has changed lines before (controllers, 16 Nov), so this is theirs to change. What changes with it, plainly: the Hydra feed format carries magnet links with no owner, licence or checksum, so Lootnest can't tell a publisher's own build from a copy nobody has the right to share. "Source-agnostic" is true of the code and is also how the feature will be read from outside: the market research's position (the vault of profiles and saves, "Playnite + Lootnest") and the rule about public wording were written to keep Lootnest out of that reading. Two practical checks before a release with this in it: SignPath's terms for the signing set up in 1.59, and winget's.

What stays true whatever the answer: no feed in the app, the installer, the site, the docs or the tests' fixtures (fixtures use made-up names and a local server); no search across feeds for games the user doesn't have (so no catalogue: feeds are only matched against the Library); nothing about DRM, emulators' files or making a game start.

### 3. BitTorrent means uploading

Lootnest is Go without cgo, so libtorrent is out. The maintained pure-Go library is `github.com/anacrolix/torrent` (MPL 2.0). Costs: a large dependency tree for govulncheck and the exe's size, a listening port and a Windows Firewall prompt, DHT and tracker traffic, and the fact that a BitTorrent client uploads the pieces it has while it downloads. "Seeding off after completion" doesn't change that; uploading can be refused altogether (`NoUpload`), at the price of slow or refused downloads on many swarms.

Options: **(a)** https only in the first version, magnet entries shown with **Open in your torrent app** (`magnet:` handed to Windows) and the finished download picked up by the existing Downloads watcher (1.58); **(b)** the built-in client as its own phase after the rest works. Recommended: (a) first, (b) as phase 4b once the owner has seen the rest on a PC. The plan below is written so (b) slots in behind one interface.

## What's in the repo

**Stack.** Go, no cgo; a WebView2 window; the page is plain HTML, CSS and JS with no build step (`internal/app/ui`, one closure in ordered parts in `ui/js`). `internal/core` is platform-neutral with tests, `internal/app` binds `ps_*` calls and sends events with `window.__ps`, `internal/win` holds Win32. Windows only; `core` tests run on Linux and on NTFS in CI. `design/harness` renders the real UI in Chromium with stubbed calls.

**The Library.** Nothing is stored as "the library": each scan builds `[]core.FoundGame` (`Key`, `Name`, `Source`, `Dir`, `Exe`, `ProcessName`, `SteamID`, `StoreID`) from every launcher's own files; folder games have the key `dir:<lowercase folder>`. What Lootnest knows about a game is kept beside that, keyed by the Library key, in JSON files in its data folder (`config.json` for profiles, tags, launch options; `shelf.json`, `copies.json`, `sizes.json`, `itch.json`, `gog.json` and so on). `RekeyGame` moves all of them when a folder game's key changes.

**Hydra is already a source** (1.33, `core/hydra.go`): games with `Source: "hydra"` and the Steam id Hydra knows. They already count as games no launcher updates (`updatableSource`).

**Downloads.** There is a downloader, not a download manager: `core/download.go` `downloader.get` (https only, `.part`, Range resume, size and md5 or SHA-256 checks), used by itch.io and GOG, one download at a time with progress and Cancel through events. No queue, pause, speed or ETA, and no BitTorrent.

**Pieces this would reuse.**

| Need | Exists |
|---|---|
| Archives into a staging folder, traversal refused, a size cap | `StageUpdate`, `StageUpdateMax` (.zip, .7z, .rar; `updateRel`) |
| Versions from names, comparing them | `UpdateVersions`, `FolderVersion`, `CompareVersions`, `CheckVersions`, `CopyVersion` |
| Names matched to games | `nameTokens`, `nameFits` (`core/downloads.go`), `norm` |
| Save locations from the Ludusavi manifest | `core/savedb.go` (`SaveDB.Find`, `Env.FindSaves`), `savesOf`, `vaultLocs`; the manifest is downloaded on the PC when the user turns it on, never shipped |
| Timestamped copies of a game's saves, with retention | `core.Cloud` on a local folder (`SaveGameVersion`, `GameVersions`, `RestoreGameVersion`), as the safety net uses; `keepSavesBefore` |
| A journal that finishes or undoes a run cut short | `swapjournal.go`, the mods' `journal.json`, `finishMods` at start |
| A folder copied with read-back and compare; same volume or not | `MoveGame`, `volumeOf`, `moveRoom` |
| Free space | `win.DiskSpace` |
| An installer shown, confirmed, run and diffed | the 1.40 flow (`apiUpdateInstaller`, `win.RunAndWait`, `SnapshotDir`, `DiffSnapshots`) |
| Updates page rows, the update drawer, toasts with actions | `storeUpdRows`, `updRow`, `75-updates.js` |
| Mods out before an update, launches refused during a run | `modsOffBefore`, `errModding` |

Two gaps in those: the manifest's registry entries aren't read today (so registry saves wouldn't be copied without new work), and `StageUpdateMax` callers cap at 2 GB (a game needs a cap from the free space instead).

## How Hydra keeps its library

Not seen on a PC: this session has no Windows machine and the container has no Hydra. What follows is read from Hydra's source (`hydralauncher/hydra`, main, 17 Nov 2026: `src/main/level`, `src/types/level.types.ts`, `src/shared/constants.ts`) and from Lootnest's reader, which the owner's Hydra games already go through. Phase 2 starts with a dump of the keys from the owner's `hydra-db` to confirm it.

- **Where.** Electron's userData: `%APPDATA%\hydralauncher\hydra-db` on Windows, `~/.config/hydralauncher/hydra-db` on Linux. A LevelDB database (`classic-level`), values as JSON. Older Hydra versions used SQLite (a `hydra.db` file; the move to LevelDB was in early 2025 as far as I know, and today's source has no SQLite left, so this part isn't read from code). A PC with only the SQLite file has an old Hydra, and the plan is to say "open Hydra once to update its library" rather than add a SQLite reader.
- **Reading it.** As today: a copy of the folder without `LOCK`, opened with `goleveldb`, so Hydra can be running and nothing of Hydra's is written.
- **Games**: key `!games!<shop>:<objectId>`.

  | Field | Meaning |
  |---|---|
  | `title` | the game's name |
  | `objectId`, `shop` | the id in `shop`: `steam` (then `objectId` is the Steam app id), `epic`, `custom`, `launchbox` |
  | `executablePath` | the exe, set by the user in Hydra; may be null |
  | `isDeleted` | removed from Hydra's library |
  | `installedSizeInBytes`, `installerSizeInBytes` | sizes, when Hydra measured them |
  | `playTimeInMilliseconds`, `lastTimePlayed` | not needed here |

  There is no install folder field: Lootnest derives it from the exe (`exeRoot`), as now.
- **Downloads**: key `!downloads!<shop>:<objectId>`, the last download per game.

  | Field | Meaning |
  |---|---|
  | `uri`, `parts` | the address used: a magnet link or a hoster URL (multi-part ones in `parts`) |
  | `downloader` | a number: 1 torrent, 0 Real-Debrid, 2 Gofile, 3 PixelDrain, 6 TorBox and others |
  | `downloadPath`, `folderName` | where it was saved and under which name |
  | `fileSize`, `bytesDownloaded`, `status`, `timestamp` | size, progress, `complete` or not, and when |
  | `extracting`, `installerPath`, `hasInstaller` | what Hydra did after the download |

  **The download's title isn't stored.** The record has the address, not the feed entry. So the "installed version" baseline is found in this order: the feed entry among the user's sources whose address is the same (for magnets, the same info hash), which gives its title and `uploadDate`; else `folderName`; else the download's `timestamp` as a date only.
- **Not read**: `auth`, `user`, `userPreferences` beyond the download folder (1.58 reads that), and `downloadSources`. Lootnest won't copy the sources the user added in Hydra: the user adds feeds in Lootnest by hand.

## Plan

Decision 1 as A, decision 3 as (a) then (b). Each phase is one PR with `## Unreleased` lines and no VERSION bump.

### Data, in Lootnest's data folder

- `sources.json`: `{id, name, url, on, format, added, etag, lastModified, checked, count, error}` per source, and `noticeSeen`.
- `feeds\<id>.json.gz`: the last good copy of each feed, already validated and cut down to the fields used.
- `sourcelinks.json`, by Library key: `{baseline: {title, version, date, from}, confirmed: [...], rejected: [...], ignoredVersions: [...], ignoredSources: [...], extraPaths: [...], keepPaths: [...]}`. Moved by `RekeyGame`.
- `fullupdates\<sha1 of key>\`: `state.json` (the state machine), `log.txt`, and the staging folder's place when it isn't beside the game.
- Save copies: the safety net's format, in `backups\saves-before-update\<game>` through a local `core.Cloud`, keep 5 by default.

### Phase 1: sources and feeds (`core/feeds.go`, `app/feeds.go`, `83-sources.js`)

- `FeedAdapter` interface: `Name() string`, `Sniff(json.RawMessage) bool`, `Parse(io.Reader, limits) (Feed, []Warning, error)`. Two adapters: `hydra` (the format in the request) and `lootnest` (the same plus optional `steamAppId`, `version`, `changelog`, `sha256`, `kind`).
- Feeds are untrusted. A hand-written strict decoder (Go has no zod; `encoding/json` with `DisallowUnknownFields` off and every field checked): body at most 64 MB, 30 s to connect and 2 min in all, at most 5 redirects and only to http or https, at most 200,000 entries, a title of 1 to 300 runes with control characters dropped, 1 to 20 addresses per entry of at most 4 KB each, schemes `magnet:` and `https:` (and `http:` kept with a warning shown on the source), a date that parses or is dropped, a size read as a number of bytes or dropped. An entry that fails is skipped and counted; a feed with none left is an error. Titles are only ever drawn as text. Nothing from a feed is run, opened or used as a path.
- The feed's own address: https, or http with the same warning. Requests carry no cookies and Lootnest's plain user agent, and are logged.
- Refresh at start and every 6 h (setting: 1 h to 7 days, or only by hand), `If-None-Match` / `If-Modified-Since`, never in safe mode. A failed refresh keeps the last good copy and shows the error.
- UI: Settings, a **Sources** category in `SET_CATS`: rows with the name, address, a lamp, last refresh, entries, the error; Add, Edit, Remove, a switch. The first Add shows the notice once ("You add sources yourself. What they list, and whether you may download it, is between you, the law and the game's publisher."), kept as `noticeSeen`. Settings, Online lists the feature, as it does everything that goes online.
- Tests: the decoder's limits, each bad shape, schemes, the adapters, conditional requests against `httptest`.

### Phase 2: linking, Hydra, detection (`core/feedmatch.go`, `core/titleversion.go`)

- Hydra: `HydraDownloads(dbDir, tmp)` beside `HydraGames`, read-only, giving the fields above; baselines filled as described. No import step is needed for the games themselves: they are in the Library already.
- Matching an entry to a game: Steam id when the feed has one; else `NormTitle` (lowercase, versions, build numbers, `+ DLC`, edition words, bracketed tags and punctuation dropped, roman numerals to digits) scored by token overlap with a penalty for leftover number tokens (so "2" never matches "3"); 0.9 and up links by itself, 0.6 to 0.9 asks ("Is this Game X?"), under that nothing. Answers are kept in `sourcelinks.json`. **Link to an entry** on any folder game's Manage menu searches that game's name in the user's feeds only.
- Versions: `TitleVersion(title)` gives `{semver, build, date, from}` by a chain: `vA.B.C` and `A.B.C.D`, `Build 12345` / `b12345`, a date in the title, and "Update vX to vY" (then the entry is a patch, see phase 5). Comparison: same kind compares; otherwise `uploadDate` against the baseline's date; otherwise "can't tell", which is shown as a possible update and never applied by itself.
- The installed version: the exe's file version (`win.FileVersion`, new, read-only) and `FolderVersion`, preferred over the stored baseline when they parse.
- The badge and the Updates page: these are rows in the existing Updates page (`storeUpdRows`' pattern, kind "From a source"), with the source, title, version, size, date, and **Update**, **Ignore this version**, **Ignore this source for this game**. A toast when new ones are found, once per version.
- Tests: tables of real-shaped titles for normalising, scoring and versions.

### Phase 3: downloads (`core/dlqueue.go`, `app/dlqueue.go`)

- A queue over the existing `downloader`: one active download, the rest waiting; pause and resume (the `.part` and Range are there), cancel, speed over a 5 s window, ETA; state in `downloads.json` so a queue survives a restart. `Fetcher` interface (`Start`, `Pause`, `Resume`, `Cancel`, `Progress`) with `httpsFetcher` now.
- Before starting: free space on the staging drive against the listed size times 2.2 (the download, what it unpacks to, a margin) plus, when the game's drive is the same, nothing more (the old folder is renamed, not copied); refused with the numbers when short.
- Always into a staging folder: `<game folder>.lootnest-new` beside the game when that drive has the room (so the swap is a rename), else under Lootnest's data folder.
- `http:` downloads only with a `sha256` in the feed. Magnet entries, until 4b: **Open in your torrent app**, and the Downloads watcher offers the result.
- 4b, if the owner says yes to decision 3 (b): `torrentFetcher` on `anacrolix/torrent`, no seeding after completion by default, upload limit and the choice in Settings, the port closed when the queue is empty.

### Phase 4: the save copy and the full update (`core/fullupdate.go`)

Kinds of package, decided after staging: a **full build** (has the game's exe, or one top folder that does), a **patch** (versions "from to", or far smaller than the game and without its exe), an **installer** (a lone setup exe, or one with `.bin` parts).

- A patch goes through the existing `PlanUpdate` / `ApplyUpdate` with `CheckVersions`, as an update the user brought does.
- An installer goes through the 1.40 flow: Lootnest shows the file, its size, publisher from its signature if it has one and the exact command line, and runs it only on the user's click, never silently, into the staging folder when the installer takes a target (Inno `/DIR=`), with the snapshot diff after.
- A full build goes through the state machine below.

States, each written to `state.json` (temp file, fsync, rename) before its work starts, each step safe to run twice:

1. `prepare`: the download's `sha256` when the feed gave one; unpack with `StageUpdate` (cap from free space); find the build's root and its exe. Refused here: the game is running, has a pending mod or swap journal, is in Program Files or WindowsApps, or a profile's files are loaded and the game has in-folder saves the swap engine owns (then: finish the swap first).
2. `backup`: mods out (`modsOffBefore`); every save location (`savesOf` plus the Ludusavi manifest's, plus the game's `extraPaths`) into a new version in `backups\saves-before-update`, with its list of what was copied, read back before going on. Registry saves: exported with `reg export` to a file in the same version (new; restore is `reg import` only from the Restore action).
3. `preserve`: the list of what stays from the old folder: in-folder save locations, known config files (`modConfigFile`), `keepPaths` the user chose, and, shown and ticked by default, files the old folder has that neither the new build nor the previous version's file list knows. The user sees the list before the swap.
4. `swap`: rename `<folder>` to `<folder>.lootnest-old`; move the staged build to `<folder>` (a rename on one volume; otherwise copy with read-back, as `MoveGame`, then rename a temp name into place); copy the preserved files in from `.lootnest-old`, never over a newer file of the same name without keeping it.
5. `verify`: the exe exists (same relative path, else the main exe found again, and the stored path updated in the one place it lives for that game); the baseline becomes the new entry; sizes recounted.
6. `finish`: `.lootnest-old` stays until the game has run once for 60 s without closing in the first 20 (the 1.40 `watchFirstStart` rule) or 14 days pass (setting), then goes to the Recycle Bin. **Roll back** until then.

On a failure at any state, and at start for a `state.json` left behind (not in safe mode, not for a running game): if `<folder>.lootnest-old` exists, whatever is at `<folder>` is renamed aside (`.lootnest-failed`, kept), the old folder renamed back, and the saves restored from the version made in `backup` only where a location changed since. The result and the log are shown. Roll back is the same path, by hand.

**Back up saves now** and **Restore saves from a copy** on every game's Saves tab use state 2's code alone; for games outside the stores this is what the safety net and Copies on this PC already do, so the work is the manifest locations, extra paths and the registry export.

- Tests: traversal in each archive kind, every transition, a kill between each pair of states followed by recovery, preserve and restore, rollback, retention. One integration test: `httptest` serving a feed and a zip, a made-up game folder with saves in the folder and outside, a full update, the saves compared byte for byte, then rollback and the same compare.

### Phase 5: the rest

The progress drawer (downloading, unpacking, copying saves, replacing, checking) with each step's log lines, read from `log.txt`; toasts for found, finished and failed; harness stubs and `?sources=1|empty|error&srcupd=hold`; `docs/GUIDE.md`, `docs/PRIVACY.md`, `docs/TESTING.md` (a new section: all of this is unseen on a PC), the README's wording as the owner decides it.

## Differences from the request

- Go, so no zod and no WebTorrent or libtorrent; see decision 3.
- No separate Updates view and no import step for Hydra games: both exist.
- "Download title as the baseline" can't come from Hydra's records; see above.
- Linux: Lootnest is Windows only, so the Windows parts aren't built for Linux; `core` and its tests run on both.
- Not built: copying the user's sources out of Hydra; any search of feeds beyond matching the Library; running an installer without a click.
