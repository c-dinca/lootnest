# Sources and updates: findings and plan

Status: built, one phase per commit on this branch; not seen on a PC yet (docs/TESTING.md, section 43). Left open are listed at the end.

The request: the user adds their own feeds (JSON files listing downloads), Lootnest matches them to games already in the Library (games Hydra installed above all), says when a feed lists a newer version, downloads it and replaces the game's folder with the saves kept. Lootnest ships no feed, suggests none, and does nothing about DRM.

## The owner's decisions (17 Nov 2026)

1. **It is a plugin, and a plugin may now hold code.** Plugins were data only. A script plugin has one .js that Lootnest runs inside itself, with permissions the user allows first ([PLUGINS.md](../PLUGINS.md), `core/pluginscript.go`). Sources and updates is the first one, in `plugins/sources-updates`, installed by the user like any plugin.
2. **Lootnest provides no sources; it is a place where the user's own can be used.** The plugin, the sources added to it and what is downloaded from them are the user's doing. Lootnest's part is general: plugins with permissions, a download queue, and an update that keeps the saves. The README's line was reworded to say that.
3. **Downloads from sources, and Real-Debrid as Hydra has it.** https links download directly. A magnet link, or a link to a file host, goes through the user's own Real-Debrid account, which turns it into https links (so nothing is uploaded from this PC). There is no BitTorrent client inside Lootnest; without Real-Debrid a magnet link opens in the user's torrent app and the Downloads watcher (1.58) picks the result up. A built-in client stays the owner's to ask for.

What stays true: no feed in the app, the installer, the site, the docs or the tests (fixtures use made-up names and a local server); no search across feeds for games the user doesn't have, so no catalogue; nothing about DRM, emulators' files or making a game start; saves and game folders are changed only by Lootnest's own code, with a copy first, after the user says yes.

## Who does what

| The plugin's script (`plugins/sources-updates/main.js`) | Lootnest (`internal/core`, `internal/app`) |
|---|---|
| Sources: add, edit, remove, on and off; the notice | Runs the script with a time limit; draws its page from data |
| Reading feeds: formats behind adapters, every field checked, the last good copy kept, ETag and Last-Modified | `lootnest.net.fetch`: size and time limits, never this PC or its network, each request logged |
| Matching entries to games, versions from titles, what counts as newer | The Library list a script may read, with Hydra's download record per game |
| Real-Debrid: magnet or host link to https links, with the user's token | Keeps the token (DPAPI), sends it only to api.real-debrid.com |
| Offers on the Updates page; ignore lists | The download queue, free space, staging |
| | Saves copied, the folder swapped, checked, rolled back; the log |

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

Each phase is one commit with `## Unreleased` lines and no VERSION bump.

### Data

The plugin's own, in `plugin-data\sources-updates` (JSON, through `lootnest.storage`):

- `sources`: `{notice, hours, list: [{id, name, url, on, format, added, checked, changed, count, skipped, http, etag, modified, error}]}`.
- `feed-<id>`: the last good copy of each feed, already checked and cut down to the fields used.
- `links`, by Library key: the installed baseline, confirmed and rejected matches, ignored versions and sources.

Lootnest's, in its data folder:

- `plugin-secrets.json`: each plugin's keys, encrypted for the Windows user.
- `fullupdates\<sha1 of key>\`: `state.json` (the state machine), `log.txt`.
- Save copies: the safety net's format, in `backups\saves-before-update\<game>` through a local `core.Cloud`, keep 5 by default.

### Phase 1: script plugins, sources and feeds. Done.

- `core/pluginscript.go`: the manifest's `script`, `permissions`, `secrets`, `pages`; a script is off until what it asks is allowed (`GrantPlugin`, `PluginAsks`), and asks again when a version asks for something else; the host (`StartScript`, `Call`, a time limit per call, `schedule`, `storage`, `net.fetch` with `guardedDial`, `library.games`, `updates.offer`, `notify`). `app/pluginscript.go`: one host per plugin that's on and allowed, none in safe mode; keys in `plugin-secrets.json`; `ps_pluginGrant`, `ps_pluginView`, `ps_pluginAction`. `79-pluginscript.js` draws a plugin's page and forms.
- The plugin: `ADAPTERS` (`lootnest` and `hydra`; each has `sniff(doc)` and `parse(doc, now)`), `parseFeed`, `readURI`. Feeds are untrusted: at most 64 MB and 120 s, at most 200,000 entries, a title of 1 to 300 characters with control and direction characters dropped, 1 to 20 addresses per entry of at most 4 KB each, only `magnet:` (with an info hash), `https:` and `http:` (kept, counted and shown as a warning), no user or password part, a date that parses or is dropped, a size read as bytes or dropped. An entry that fails is skipped and counted; a feed with none left is an error and the last good copy stays. Unknown fields are ignored. Titles are only ever drawn as text. Nothing from a feed is run, opened or used as a path.
- Refresh 5 s after start and every 6 h (1, 3, 6, 12, 24 h, or only by hand), one source per call, `If-None-Match` / `If-Modified-Since`.
- The Sources page in Settings: rows with the name, address, a lamp, entries, format, last refresh, what was left out, the error; Add, Edit, Remove, Refresh, a switch. The first Add shows the notice once.
- Tests: `core/pluginscript_test.go` (install rules, permissions, limits, keys, this PC out of reach), `core/sourcesplugin_test.go` (the real plugin against `httptest`: every bad shape, conditional requests, a feed that breaks).

### Phase 2: linking, Hydra, detection. Done.

Built as planned below, with these names: `core.HydraOrigins`, `core.LocalVersion` (the folder's name, else a version file at its top), `win.FileVersion`, `ScriptGame.Origin`, `.Version`, `.ExeVersion`; in the script `normTitle`, `titleScore` (0.9 links, 0.6 asks), `titleVersion`, `compareVer`, `installed`, `isNewer`, `detect`, `publish`, `offerAction`. A game with no version anywhere counts entries uploaded after the plugin first saw it. The exe's version is used only when it continues the baseline's numbering, since many exes carry their engine's version. Offers show on the Updates page (`offersSection` in `79-pluginscript.js`) and on the game's page (`offerBanner`).

- `HydraOrigins(dbDir, tmp, containers)` beside `HydraGames`, read-only; `ScriptGame.Origin` carries a game's record to the script. No import step is needed for the games themselves: they are in the Library already.
- In the script. Matching an entry to a game: Steam id when the feed has one; else `normTitle` (lowercase, versions, build numbers, `+ DLC`, edition words, bracketed tags and punctuation dropped, roman numerals to digits) scored by token overlap with a penalty for leftover number tokens (so "2" never matches "3"); 0.9 and up links by itself, 0.6 to 0.9 asks, under that nothing. Answers are kept. A game can be linked to an entry by hand from the plugin's page.
- Versions: `titleVersion(title)` gives semver, build and date by a chain: `vA.B.C` and `A.B.C.D`, `Build 12345` / `b12345`, a date in the title, and "Update vX to vY" (then the entry is a patch). Same kind compares; otherwise `uploadDate` against the baseline's date; otherwise "can't tell", shown as a possible update.
- The installed version: the folder's name (`FolderVersion`) and the exe's file version, preferred over the stored baseline when they parse.
- Offers: rows on the Updates page (`lootnest.updates.offer`) with the source, title, version, size, date, and **Update**, **Ignore this version**, **Ignore this source for this game**. A notice when new ones are found, once per version.

### Phases 3 and 4: done

As planned below, with these names and differences.

- `core/dlqueue.go` `DownloadQueue` (one at a time, `Pause`, `Resume`, `Remove`, `Job` with speed and ETA, kept in `downloads.json`, `GuardedClient`); `downloader.plain` lets http through only with a SHA-256.
- `core/fullupdate.go`: `FindBuild`, `PlanFullUpdate` (saves inside the folder, the user's ticks, and what only the old folder has at its top; never a store's files or a stand-in tool's), `RunFullUpdate` (backup, swap, restore, verify, kept), `RecoverFullUpdate`, `RollbackFullUpdate`, `FinishFullUpdate`, `state.json` written before each step.
- `app/fullupdate.go`: jobs in `fullupdates\<hash>\job.json` with `log.txt`; staging is `<game folder>.lootnest-new` (`dl` and `build`); `pluginUpdate` (the script's `updates.run`), `jobUnpack` (whole build, else patch or installer handed to the update panel), `jobApply`, `jobRollback`, `jobFinish`, `watchKept` (the old folder goes once the game has run a minute and closed, or after 14 days; a first start that ends within 20 seconds is told), `recoverJobs` at start, `errUpdating` refuses a launch during the swap.
- The saves' copy is a version in the safety net's store, named "Before the update to x", so it shows on the game's Saves tab under Copies on this PC with Restore; five are kept. That tab's save points are "Back up saves now" and its copies "Restore saves from a copy", for any game, with or without an update.
- In the script: `startUpdate`, `begin`, `updateState`, Real-Debrid (`rdCall`, `rdTorrent`, `rdFiles`, `rdStart`, `rdTick`).
- Different from the plan: the save copy isn't in a store of its own; registry saves aren't exported (the manifest's registry entries are still unread, as before); the free-space check is the listed size times 2.2 beside the game only; the days an old folder is kept (14) and the save copies kept (5) are constants, not settings yet.

### Only what changed (after 2.11.2)

The owner (17 Nov 2026): a 100 GB game shouldn't need 100 GB free, or a whole download, for an update. So a job whose files are one archive at https addresses is read in place (`core/delta.go`, `core/remotefile.go`, `app/deltaupdate.go`, `Mode` "delta"):

- `RemoteFile` reads a file over https by Range requests in 4 MB blocks (a few kept), as an `io.ReaderAt`; `RemoteVolumes` is an `fs.FS` of a .rar's parts, each one stream that picks up where a dropped connection stopped. A site that won't send parts (`ErrNoRanges`) gets the whole download first (`Mode` "delta-local"), and the archive is deleted once read.
- `StageDelta` lists the archive, finds its root (`rootFromNames`: the game's exe, else one wrapping folder), compares each file's CRC-32 and size with the game's file, and stages only the ones that differ into `<folder>.lootnest-new/files`, with `plan.json`. A .zip or non-solid .7z fetches only those files' bytes; a .rar or solid .7z is streamed in order, the same files read past, never saved. Room is checked as the changed size is known. Every staged file's CRC is checked; a feed's SHA-256 of the whole archive can't be, so it isn't over https (plain http still downloads whole, to check it).
- `Full`: the archive has the game's exe, so it's the whole game, and files it doesn't have (`DeltaRemovals`, minus saves inside the folder and what the user keeps) move aside. Otherwise it's a patch and nothing else is touched.
- `ApplyDelta` moves files on the game's drive: each old file to `<folder>.lootnest-old/<rel>`, the new one in; a record (`delta.json`) is saved before every move, so `RevertDelta` (Roll back, a failure, or a start after a crash) puts each back, and can run twice. Saves are copied first, as for a full build.
- The page: a "fetch" phase (Compare, Get what changed, Copy saves, Replace) with files done of all, bytes downloaded and speed, Pause, Resume and Cancel; the review dialog says replaced, new, already the same and moved aside. Harness `?sources=1&srcupd=compare|fetch|fetchpaused|deltaready|deltakept`.
- Limits: an installer needs its whole download; a .rar in parts resumes from its first part after a pause or a restart (what was staged is kept and not fetched again); a solid .7z is read whole, though not saved.

### Phase 3: downloads (as planned)

- Lootnest: a queue over the existing `downloader` (one active, the rest waiting; pause and resume on the `.part` and Range; cancel; speed, ETA; kept in `downloads.json`). Free space checked first (the listed size times 2.2 on the staging drive) and refused with the numbers when short. Always into a staging folder: `<game folder>.lootnest-new` beside the game when that drive has room, else under Lootnest's data folder. `http:` only with a `sha256`.
- The script: Real-Debrid (`api.real-debrid.com/rest/1.0`: `torrents/addMagnet`, `torrents/selectFiles`, `torrents/info`, `unrestrict/link`), with the user's API token as the plugin's one key. A magnet without Real-Debrid: **Open in your torrent app**.

### Phase 4: the save copy and the full update (`core/fullupdate.go`)

Kinds of package, decided after staging: a **full build** (has the game's exe, or one top folder that does), a **patch** (versions "from to", or far smaller than the game and without its exe), an **installer** (a lone setup exe, or one with `.bin` parts).

- A patch goes through the existing `PlanUpdate` / `ApplyUpdate` with `CheckVersions`, as an update the user brought does.
- An installer goes through the 1.40 flow: Lootnest shows the file, its size and the exact command line, and runs it only on the user's click, never silently, with the snapshot diff after.
- A full build goes through the state machine below.

States, each written to `state.json` (temp file, fsync, rename) before its work starts, each step safe to run twice:

1. `prepare`: the download's `sha256` when the feed gave one; unpack with `StageUpdate` (cap from free space); find the build's root and its exe. Refused here: the game is running, has a pending mod or swap journal, is in Program Files or WindowsApps, or a profile's files are loaded and the game has in-folder saves the swap engine owns.
2. `backup`: mods out (`modsOffBefore`); every save location (`savesOf` plus the Ludusavi manifest's, plus the game's extra paths) into a new version in `backups\saves-before-update`, with its list of what was copied, read back before going on. Registry saves: exported with `reg export` to a file in the same version.
3. `preserve`: the list of what stays from the old folder: in-folder save locations, known config files (`modConfigFile`), paths the user chose, and, shown and ticked by default, files the old folder has that the new build doesn't know. The user sees the list before the swap.
4. `swap`: rename `<folder>` to `<folder>.lootnest-old`; move the staged build to `<folder>` (a rename on one volume; otherwise copy with read-back, as `MoveGame`, then rename a temp name into place); copy the preserved files in from `.lootnest-old`.
5. `verify`: the exe exists (same relative path, else the main exe found again, and the stored path updated); the baseline becomes the new entry.
6. `finish`: `.lootnest-old` stays until the game has run once without closing in its first 20 s (the 1.40 `watchFirstStart` rule) or 14 days pass (a setting), then goes to the Recycle Bin. **Roll back** until then.

On a failure at any state, and at start for a `state.json` left behind (not in safe mode, not for a running game): if `<folder>.lootnest-old` exists, whatever is at `<folder>` is renamed aside (`.lootnest-failed`, kept), the old folder renamed back, and the saves restored from the version made in `backup` only where a location changed since. The result and the log are shown. Roll back is the same path, by hand.

**Back up saves now** and **Restore saves from a copy** on every game's Saves tab use state 2's code alone.

- Tests: traversal in each archive kind, every transition, a stop between each pair of states followed by recovery, preserve and restore, rollback, retention. One integration test: `httptest` serving a feed and a zip, a made-up game folder with saves in the folder and outside, a full update, the saves compared byte for byte, then rollback and the same compare.

### Phase 5: the rest

The progress drawer (downloading, unpacking, copying saves, replacing, checking) with each step's log lines; notices for found, finished and failed; `docs/GUIDE.md`, `docs/TESTING.md` (a new section: all of this is unseen on a PC).

## Left open

- Nothing here has run on a PC: the swap on NTFS with files in use, the Recycle Bin, the process watch and DPAPI above all.
- Registry saves; settings for the kept days and copies; a queue page of its own (downloads show as rows on the Updates page).
- Games with profiles: an update is offered only for the Library's copy of a game; one whose saves a profile's swap owns isn't handled specially yet.
- A feed is read whole into the script's memory, and the engine has no memory limit.
- Real-Debrid was only run against a stand-in server in tests, never the real one.

## Differences from the request

- Go and a script engine, so no zod, WebTorrent or libtorrent; the feed checks are hand-written in the plugin.
- No separate Updates view and no import step for Hydra games: both exist.
- "Download title as the baseline" can't come from Hydra's records; see above.
- Linux: Lootnest is Windows only; `core` and its tests run on both.
- Not built: copying the user's sources out of Hydra; any search of feeds beyond matching the Library; running an installer without a click; a BitTorrent client.
