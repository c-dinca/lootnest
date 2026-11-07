# Working on Lootnest

Notes for anyone (person or AI agent) picking this repo up in a new session.
Read this first; it holds the conventions and the state of the work, so a
fresh chat can carry on without the old one's context.

## The product

Lootnest is a Windows game manager. Every game on the PC, from Steam, Epic,
GOG, Xbox, EA, Ubisoft, Battle.net, Riot, Amazon, Rockstar, itch.io, Hydra, game
folders and shortcuts, sits in one Library. Its core is **profiles**: each
profile has its own settings, saves and achievements in every game, and
Lootnest swaps the right files into place before a game starts, then saves
them back when it closes. A copy is made before every swap.

Profiles work like a console's. A profile can be a person, or a **run** of a
person, such as "Cristian hardcore" of Cristian (`Player.Of`). That serves a
household sharing a PC and one person keeping several runs of a game
(completionists, replays) with the same mechanism. PRODUCT.md has the full
brief; DESIGN.md has the design system.

What makes it different, and the direction to keep pushing:

- per-profile saves, settings and achievements, including for games that
  aren't from a store;
- it works with games however they were installed, games in a folder
  included. It reads the files Steam-compatible tools leave on the PC
  (Goldberg/GSE, CODEX, RUNE, OnlineFix, RLD, SKIDROW, CreamAPI, EMPRESS
  achievements), and swaps and syncs them per profile.
  Lootnest itself never downloads, cracks or runs anything for the user,
  and never bypasses DRM. Reading what's already on the PC is fine;
- cloud saves through the user's own OneDrive or Google Drive folder, with
  versions, encryption and per-profile sync;
- nothing goes online unless the user turns it on (README, Privacy table).

## Stack and layout

- Go (no cgo) plus a WebView2 window (`third_party/go-webview2`, modified).
  The UI is plain HTML, CSS and JS with no build step, embedded with
  `go:embed` and inlined at start with a CSP nonce.
- `internal/core`: platform-neutral logic, with tests. It covers config,
  the swap engine, scanning, achievements, cloud sync, archives and covers.
- `internal/app`: the Windows app. `api.go` binds `ps_*` functions on
  `window`; Go sends events through `window.__ps({type, ...})`. Slow work
  runs in a goroutine and comes back as an event.
- `internal/win`: Win32 helpers (DPAPI, window closing, tray and so on).
- `internal/app/ui`: `index.html`, `system.css` (tokens and `ln-`
  components), `app.css`, and `app.js` (one module, about 3,000 lines).
- `cmd/lootnest`, `cmd/setup` (the installer and its `setup.html`),
  `tools/mkicon.py` (draws the icon), `build.sh`.
- `design/harness`: a dev-only fixture harness that renders the real UI in
  Chromium with stubbed `ps_*` calls.
  - `node design/harness/build.mjs` writes `dist/page.html`.
  - `node design/harness/shoot.mjs <outdir> <filter>` screenshots into
    `design/<outdir>` (the filter is a substring). Delete the output
    afterwards.
  - Query options: `?players=6`, `?games=400`, `?state=loading|offline|error`,
    `?runs=1`, `?first=1`.
  - Stubs live in `stub.js`; add one for each new `ps_*` call.

## How changes are made

- **One feature per branch and PR.** Branch from fresh `origin/main`.
  Squash merges conflict with branches stacked on unmerged ones, so wait for
  the previous PR to merge, then branch again.
- **Every change is a release.** Bump `VERSION` and add a `## x.y.z`
  section at the top of `CHANGELOG.md`, in plain user language. A push to
  main whose VERSION has no tag yet publishes the release, using that
  changelog section as its notes. Don't push tags by hand.
- Before pushing, run:
  - `gofmt -l internal`
  - `go test ./internal/core/`
  - `GOOS=windows go vet ./internal/app`
  - `GOOS=windows go build ./cmd/lootnest`
  - `node --check internal/app/ui/app.js`

  For UI changes, also screenshot the change in the harness and look at it.
- CI is one `build` job (`.github/workflows/build.yml`, including
  govulncheck). The owner wants PRs merged as soon as CI is green. Open them
  as drafts, mark them ready, then squash-merge.
- Commits are authored by **Cristian Dinca
  <93765979+c-dinca@users.noreply.github.com>** and end with
  `Co-Authored-By: Claude <noreply@anthropic.com>`. Claude is never the
  author, and there's no session link in commits.
- PR and issue bodies carry **no** "Generated with Claude Code" footer or
  session link, because the repo is public. If a tool adds one, update the
  body again without it.
- The owner writes in Romanian. Answer them in Romanian. Code, UI text,
  commits, PRs and docs are in English.
- The design rules in DESIGN.md:
  - tokens only, with no raw colours or sizes outside the token block;
  - WCAG AA contrast;
  - small radii and hairlines, never "rounded AI slop";
  - full-width, responsive pages;
  - fonts licensed under the OFL.
- Copy voice: plain, specific and calm. No hype, no emoji.
  - Avoid em dashes and middle dots as separators, all-caps labels, and
    "→" in buttons.
- **Never lose user data.** Anything that moves user files makes a copy
  first. Prefer adding over replacing.
- Scripts that edit files: never `open(p, 'w')` and read the same file in
  one expression, because that empties it. If it happens,
  `git checkout <file>`.

## Brand

- Obsidian Black `#111217` is the ground.
- Ice Grey `#DFE5EB` is text.
- Racing Red `#EF233C` fills the primary button, with bold near-black
  text (`--fg-on-accent`, 4.7:1), and is a touch elsewhere: the logo gem,
  the current nav icon, progress, switches that are on, checkboxes, the
  selected segment, reader marks and the text selection. It is never small
  text, because it is 4.4:1 on obsidian. Play as someone keeps that
  profile's colour.
- The app icon is a red gem in an ice-grey nest on an obsidian tile.

## Where things stand (1.49.0, beta)

These shipped recently, newest last:

- **1.17** The swap moment; Close the game; hide the window only once the
  game runs.
- **1.18** The seat ("Who's here": every Play is the seated profile's).
- **1.19** Achievements on the game page, read from the PC: Steam's
  `librarycache` and the tool files listed above.
- **1.19.1** The brand colours.
- **1.20** Per-profile achievements: the tool's folder or file is swapped
  with the profile; `Game.SharedAchievements` opts out. Players became
  Profiles.
- **1.21** New profile from "Who's playing?"; start from a copy of any
  profile (`ActivateOptions.CopyFrom`).
- **1.22** The Achievements page in the sidebar.
- **1.23** Lootnest milestones; an optional Steam Web API key for
  achievement names; more ways to find a folder game's Steam app id;
  achievements in cloud sync (`{ACHIEVEMENTS}/<id>` paths).
- **1.24** Profiles grouped by person: runs (`Player.Of`), up to 16
  profiles.
- **1.25** Per-profile status (playing, up next, finished, 100%, dropped,
  in `shelf.json`) and collections (`Config.Tags`), with Library filters.
- **1.26** Swap safety: a journal (`swaps.json`, `swapjournal.go`) so a
  swap cut short is finished on the next start, and `Mirror` reads every
  copy back before it replaces anything.
- **1.27** Profile path suggestions use the Ludusavi manifest
  (`SuggestPaths` via `Env.SaveDB`, `listedSaves` in `suggest.go`) when
  Settings.SaveDB is on; the switch is also in Settings, Online.
- **1.28** Design pass: the red primary button; drawers (`.ln-drawer`,
  `drawer()` in app.js) for a game's Manage actions, its full achievement
  list and Names and colours; achievements at the top of a game's page;
  the Achievements page with game cards; Cloud saves and Profiles laid out
  to fill the window.
- **1.29** Save Safety Net (`app/safetynet.go`): in-folder saves of
  Library games outside the stores are copied to
  `safety-net\Lootnest\saves\<game>` (the cloud versions format, via a
  local `core.Cloud`) when they change; restore from the game's page.
  `Engine.keepVanished` keeps a profile's copy in backups before a save
  would empty it.
- **1.30** Settings in categories (a list on the left, as in Steam's);
  drawers scroll (they were as tall as their content); more Racing Red:
  the current page's bar, page-title stripes, cover hover, `ln-tag--accent`.
- **1.31** Library: a Recent games shelf on wide art (`recentShelf()`),
  as in Steam's library, above the filters and the grid.
- **1.32** Library: a shelf for each collection (`collectionShelves()`,
  up to six), then an All games heading over the filters.
- **1.33** Hydra Launcher as one more launcher (`core/hydra.go`): its
  installed games are read from a copy of its LevelDB library
  (`%APPDATA%\hydralauncher\hydra-db`, `!games!<shop>:<id>`, via
  `syndtr/goleveldb`). `FoundGame.SteamID` carries the Steam id Hydra
  knows, for save locations and achievements. Only games already on the
  PC; Lootnest never offers Hydra's downloads or sources.
- **1.34** Find games (`core/findgames.go`, `app/findgames.go`, view
  `find`, drawer `offer`): Steam storesearch and appdetails (prices in the
  Windows country's currency, requirements), CheapShark for other official
  stores and the lowest price ever, this PC's CPU, GPU, RAM and free space
  (`win/hw.go`). Opt-in (`Settings.FindGames`). Links open only the Steam
  store and CheapShark's deal redirect (`SafeStoreLink`).
- **1.35** The Store (view `find`, renamed in the UI): rows before a search
  from `FetchStoreFront` (Steam `featuredcategories`: daily deal, specials,
  top sellers; CheapShark best deals with Metacritic 75 or more). The
  sidebar search (`#gq`, Ctrl+K, `renderGq()`) shows Library matches, then
  store hits.
- **1.36** Will it run here? (`core/perf.go`): rough GPU and CPU scores
  (`gpuScores`, Intel and Ryzen tables by generation, laptop factors),
  `WillItRun` gives high, recommended, minimum, struggles or below from
  the weakest of graphics and processor (memory and space only lower it),
  shown as `runCard()` with bars in the offer drawer. Add new cards and
  chips to the tables as they come out.
- **1.37** Updates (`core/gameupdate.go`, `app/gameupdates.go`, view
  `updates`, drawer `update`): for games no launcher updates, the newer
  version from Steam's news (`NewerThanCopy`), and `PlanUpdate` /
  `ApplyUpdate` / `UndoUpdate` for an update the user brings (folder or
  .zip; the update's own top folder is skipped by `bestPrefix`). Replaced
  files are kept in `backups\updates\<game>\<time>\files`. Lootnest never
  finds or downloads an update. Installers came in 1.40; not yet: rolling a whole
  folder back.
- **1.38** .7z and .rar updates (`core/updatearchive.go`: `StageUpdate`
  into `.lootnest-update-<hash>` beside the game's folder, via
  `bodgit/sevenzip` and `nwaples/rardecode/v2`), versions from names
  (`UpdateVersions`, `FolderVersion`, `CheckVersions`: ok, skips, older,
  same), a version history, and Undo newest first.
- **1.39** A new copy of the same game (`core/newcopy.go`,
  `app/newcopy.go`): `copies.json` is a ledger of folder games seen
  (`First` orders copies), `Matches` by Steam id or name, `CarrySaves`
  moves in-folder saves, `RekeyGame`/`RekeyConfig` move what's keyed by
  the Library key, `Engine.Reseat` puts the owner's profile into a game
  with profiles after its paths are retargeted.
- **1.40** Installer updates (`apiUpdateInstaller`, `win.RunAndWait`):
  saves first, an optional full copy of the game, a snapshot diff
  (`SnapshotDir`, `DiffSnapshots`) and `RecordInstallerUpdate`, which keeps
  only the changed files' old versions (`NoUndo` without the full copy).
  Apply another update in a row. `watchFirstStart` (from `apiLaunchFound`)
  marks an update `Broke` when the game closes within 20 s of its first
  start.
- **1.41** Unlock toasts (`core/achlive.go`, `app/achlive.go`,
  `win/toast.go`): `achTick` every ~6 s from `watch()` reads `PCUnlocks`
  for running games (sessions, and Library games outside Steam by
  process) and shows `NewUnlocks` in a native layered, top-most,
  click-through window on its own thread (GDI, no WebView). Off for games
  with `AntiCheat(dir)`; `Settings.AchToastsOff` turns it off.
  `Settings.AchRarity` (opt-in) reads Steam's public global percentages
  (`globalPercent`, kept 3 days) into `Achievement.Percent`. The seat is
  sent to Go (`ps_setSeat`) for the colour.
- **1.42** Achievement Doctor (`core/achdoctor.go`, `app/achdoctor.go`,
  drawer `achdoc`): `DetectTool`, `Env.AchievementFiles`, steps as
  `DoctorStep` (ok, warn, fail, info). `SetToolAccountName` writes
  `account_name` in `steam_settings\configs.user.ini` (and the legacy
  `force_account_name.txt` if present), with a copy kept first; only from
  the button.
- **1.43** Keep a game offline (`core/offline.go`, `app/offline.go`):
  `GameExes` (main first, no redists, up to 30), `FirewallScript`
  (PowerShell `New-NetFirewallRule`, group "Lootnest", description
  `OfflineTag(key)`, always removing the game's old rules first) run via
  `-EncodedCommand` with `win.RunElevatedAndWait` (runas).
  `Config.Offline` records what's blocked. Only for games Lootnest
  starts itself (exe launch).
- **1.44** Save points (`core/savepoints.go`, `app/savepoints.go`, block
  `savePointsBlock`, drawer `points`): `savepoints\<game>\<id>` with
  `point.json` and `files\<loc id>`; `MakeSavePoint` compares stamps
  before and after each copy and tries again while the game writes;
  `RestoreSavePoint` after an Auto point; `ProfileFromSavePoint` starts a
  new run (`Player.Of`). Locations: `Engine.SaveLocs` (profile paths
  without achievements) or `savesOf(f)` for Library games. Ctrl+Shift+F5
  (`win/hotkey.go`) is registered only while a game runs (`hotkeyTick`).
- **1.45** Beta (`app.Stage`, shown by the name and in Settings). Keep
  VERSION numeric (x.y.z): `core.NewerVersion` only parses that, so a
  "-beta" version would stop updates. The save check on Home
  (`app/savecheck.go`, `saveCheckBlock()`), and Report a problem (copies
  the version and log, opens a new GitHub issue; nothing is sent).
- **1.46** The shell: games in the sidebar (`renderSideGames()`, groups
  playing, recent, collections, all; `#side-q` filter; `S.sideSort`;
  folded groups in `S.sideClosed`), the sidebar folds to a rail
  (`.app.side-folded`, Ctrl+B, `S.folded`) and resizes (`#side-resize`,
  pref `sideW`). Motion tokens (`--dur-4`, `--ease-emph`, `--stagger`,
  `--rise`), view transitions on nav, `staggerIn()`. The Library's list
  view (`S.density === "list"`, `listRowHTML`). `win.ToastFailed` logs why
  a toast couldn't be shown and falls back to the tray.
- **1.47** The Updates page (`viewUpdates`, `updCard`, `updState`: new,
  ok, dim, low): stat filters (`S.updFilter`), a Needs a look band, cards
  with art and version against the latest (`verIn` reads it from the
  patch notes' title), a More menu in `#ctx`, a history drawer
  (`updhist`). `emptyState()` and `.ln-empty` for pages with nothing yet.
- **1.48** Achievement tiers (`--tier-ultra/rare/uncommon/common`,
  `tierOf`, `achRow`, `core.RarityColor`), rings (`ring()`), count-up
  (`countUp`, `data-countup`), `.arrive` animates bars and rings once per
  page entry, Rarest unlocked (`core.RarestUnlocked`, `rarest` in
  allAchievements), in-window unlock pops (`achPop`, from `achUnlocked`'s
  `list`), and the native toast slides in with a trophy badge
  (`Toast.Badge`, `drawTrophy`).
- **1.49** Big Picture (`BP`, `bpOpen`, `renderBP`, `bpScreen`, `#bp`
  above the app at z-index 45, under the picker): tabs, rows and grid,
  game detail, Who's playing; `bpMove` is spatial navigation over
  `[data-bp]`; the Gamepad API loop (`padLoop`: Start toggles, A B Y,
  LB RB, stick and D-pad with repeat); WebAudio `sfx`; F11.
  `win.SetFullscreen` (borderless on the window's monitor), `ps_bigPicture`,
  tray item "Big Picture".

Not checked on a real PC yet: the owner was on 1.19.x at the last log.
Everything from 1.40 on that touches Windows is unchecked there too:
installers run and watched, the toast window over games, the firewall
rules (PowerShell as administrator), Ctrl+Shift+F5, and save points made
while a game runs.
1.23.2 fixed achievements not showing for Unreal folder games such as
Phantom Blade Zero. Ask for the "Show recent activity" log after they
update. It names each game and says why none were found.

## Next, from the market research (November 2026)

The research brief's conclusion: Lootnest's position is **the PC's vault of
profiles and saves**. Unified libraries exist (Playnite, GOG Galaxy,
LaunchBox); per-person and per-run saves with file swaps don't, and
Playnite's author declined multi-user profiles (Playnite #3364). Steam
Families separates saves only for games on the Steam Cloud API. The main
risk is the swap's reliability, Steam Cloud above all, not competitors.
Position: profiles like a console for the shared PC, with "a safety net for
games from outside stores" as the growth engine, achievements and runs per
profile as signature features, and local-first (no account, your own
cloud) as an attribute. Aim for "Playnite + Lootnest", not a replacement.

Build in this order, safety before new features:

1. **A Steam Cloud-safe swap engine.** Done so far: the Steam Cloud guard
   (`steamcloud.go`, `app/cloud.go`), the swap journal and verified copies
   (1.26). Still open: put games with an anti-cheat (EAC, BattlEye,
   Vanguard) or online-only saves out of swaps by default, with a note on
   their page; a conflict banner on the game page, not only a toast; a
   one-click restore of the copy made before each swap.
2. **Save paths from the Ludusavi manifest.** Done in 1.27. The manifest
   is downloaded on the user's PC, never shipped with Lootnest: its data
   comes from PCGamingWiki (CC BY-NC-SA), so bundling it would bind a
   paid tier. Still open: the manifest's registry entries (saves in the
   Windows registry) aren't used.
3. **Save Safety Net** for folder games. Done in 1.29: a copy whenever
   in-folder saves change (Lootnest doesn't uninstall games, and Game
   Mover moves the whole folder), and a notice when they're gone. Still
   open: restore a profile's "kept" backup in one click (now it's Open
   backups).
4. **Achievement Doctor**: a read-only diagnosis of why a game shows no
   achievements (tool found, path, app id, empty file). Any edit, such as
   `account_name` per profile, only when asked. Never generate, ship or
   "repair" anything that affects whether a game starts.
5. **Run branching**: start a run from a snapshot (NG+, hardcore), with a
   timeline per run.

Then: import from Playnite (library, playtime, statuses); a page "How
Lootnest handles your saves" before any public post; a controller seat
screen; kid profiles (local limits).

Rules from the brief: never say "cracked" in public material (say games
from outside stores, DRM-free, portable, emulated); no downloads, no
account, no subscription. The owner chose to add Find games (1.34)
against the brief's "no catalogue": it stays official stores only, opt-in,
and never links to unofficial sources, hidden or not. Money, if any: free core forever,
donations, and an optional one-time Supporter for comfort features, never
for safety.

## Agreed with the owner, next up (November 2026)

These came after the research brief, in conversation with the owner. Build
them one per PR, in this order unless the owner says otherwise.

**The achievement overlay.** No injection, no reading game memory: a
separate always-on-top window, as Xbox Game Bar or a desktop companion
app does. It shows over windowed and borderless games, and over most
"fullscreen" ones on Windows 10 and 11 (fullscreen optimizations); not
over true exclusive fullscreen, where it says to switch to borderless.
Off by default for games with an anti-cheat (EAC, BattlEye, Vanguard).
WebView2 can do a transparent background (`PutDefaultBackgroundColor`
with alpha 0 in `third_party/go-webview2`). Window behaviour over a real
game can only be checked on the owner's PC.
1. **Unlock toasts** (done in 1.41): while a game runs, watch its achievement files
   (Steam's cache and the tools' files) and show "Achievement unlocked"
   with the name, its rarity (Steam's public global percentages, no key
   needed) and the profile's colour.
2. **The panel** on a hotkey (Ctrl+Shift+A, not Shift+Tab, Steam's):
   the seated profile's achievements for this game, progress, rarity,
   hidden ones covered until clicked, the profile's own notes per
   achievement, "Find a guide" in the browser.
3. **Tracking**: pin up to 3 achievements to a small corner HUD, with the
   session's time.
4. Later, live match data where the game offers it officially: CS2 and
   Dota 2 Game State Integration, League's local Live Client Data API.

**Things only Lootnest can do** (they need the swap engine and profiles):
1. **Save points and a time machine** (done in 1.44, without the screenshot): a hotkey during play copies the
   save files as a named point ("before the boss"), with a screenshot.
   Each game gets a timeline (time, playtime, achievements then): rewind
   to any point (the current state is kept), or branch a new run from it.
   This is the research brief's run branching. Copies made while the game
   runs are read back and retried if the game was mid-write.
2. **Try a save safely**: import a save from a friend or a mod site into a
   temporary run; the profile's own saves stay untouched; one click back,
   or keep the run.
3. **Lootnest on a USB stick**: a portable Lootnest carries its profiles;
   on a friend's PC it swaps your saves in and theirs back afterwards,
   with a copy first, installing nothing.
4. **What do we play tonight?**: from who's seated (co-op games none of
   them finished when two sit), the time they have, Up next, and a
   backlog roulette.
5. **Your year in games**: a yearly recap per profile and per household,
   across every store and folder game, as an image made on the PC to
   share.

**For games from outside stores** (repacks, folders, emulated). Only what's
already on the PC, as always:
1. The overlay above: most Steam emulators have no overlay or toasts.
2. **The profile's name in the game** (by hand from the check in 1.42; per swap still open): when the owner asks, write the
   profile's name and id into the emulator's own settings file (Goldberg
   `configs.user.ini` / `account_name.txt` and the like), so LAN friends
   see the right name and each run has its own identity.
3. **Achievement Doctor** (research step 4). Done in 1.42.
4. **Your version against the latest**: read the version from the folder
   name ("v1.0.3") and compare with the latest update in Steam's news;
   on an update the safety net keeps the saves, and optionally the old
   game folder, to go back.
5. **Remove a folder game cleanly**: copy its saves first, then delete the
   folder; reinstalled later, its saves come back.
6. **Block a game from going online**: a Windows Firewall rule for its exe. Done in 1.43.
7. **Import achievement history** from Hydra and Achievement Watcher.

The line, which holds for all of these: Lootnest never shows where to get
games (no sources, official-looking or hidden), never bypasses DRM, and
never generates, ships or "repairs" anything that makes a game start.

## Ideas not started

- Disk space: what each game and profile takes, what can move to another
  drive (Game Mover exists for Steam), and what can go, with a backup first.
- Better "Will it run?" data: 1.36 uses Lootnest's own rough score tables
  (`core/perf.go`). They need updating as new parts ship.
- Signing in to Microsoft or Google directly for cloud saves (on hold;
  the folder approach works).
