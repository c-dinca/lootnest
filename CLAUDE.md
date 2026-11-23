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
  and never bypasses DRM. Reading what's already on the PC is fine.
  The owner's one exception (8 Nov 2026): games the user owns, from the
  store they bought them from (itch.io now, GOG next), through that
  store's official API after the user connects it; never other sources,
  never a catalogue. Lootnest ships no sources, no plugin that reads
  them, and looks for none. A script plugin the user installs may offer
  updates; the plugin and what it downloads are the user's doing. The
  owner's decision of 17 Nov 2026, after a talk about legal risk: the
  Sources and updates plugin left the repo and the releases, nothing in
  Lootnest names Hydra's source format, and nothing is specific to
  piracy (no repack tags, no getting past a site's blocks, no real
  source addresses in tests, docs or screenshots);
- cloud saves through the user's own OneDrive or Google Drive folder, with
  versions, encryption and per-profile sync;
- nothing goes online unless the user turns it on (README, Privacy table).

## Story games and online games (the owner, 20 Nov 2026)

Lootnest treats two kinds of game differently. The owner: single-player
story games have no anti-cheat, so there Lootnest can do what it needs
without anyone being banned; multiplayer games have one, and Lootnest
stays out of them.

- **Online games** have an anti-cheat (`core.AntiCheat(dir)`: Easy
  Anti-Cheat, BattlEye, Vanguard, FACEIT, EA AntiCheat), or are played
  mainly online. Lootnest works around them only from the outside, as it
  always has: its own windows, the game's files while it's closed, ETW.
  It never goes into their process. A ban costs the user their account.
- **Story games** are single-player with no anti-cheat. In these,
  Lootnest may go into the game's process: load a DLL of its own, hook
  its Present (the overlay and the FPS counter over exclusive
  fullscreen), read and write its memory, and call the game's own save
  and load code.

Telling them apart (`core.InGameAllowed`, since the in-game FPS counter):
- An anti-cheat anywhere in the game's folder makes it online. So does a
  story mode that shares its exe with an online mode: GTA V (BattlEye),
  Red Dead Redemption 2 (Rockstar's own), Elden Ring (EAC).
- Steam's categories (Single-player, Multi-player, Online PvP, MMO)
  help; a game with only online categories is online.
- When unsure, the game is online. The user may mark a game online.

**Story games with an online mode** (the owner, same day: many people
play only the story of such a game). On such a game's page, a key lets
the user allow Lootnest inside it, after a warning: an anti-cheat that's
running sees anything that goes into the game, in the story mode too,
and the ban is on the user's account. Then:
- Never for competitive games: games with no story (CS2, Deadlock,
  Valorant, League of Legends, Dota 2...), and any game with a kernel
  anti-cheat (Vanguard, FACEIT, EA AntiCheat).
- Lootnest goes in only while none of the game's anti-cheat runs (its
  processes and services checked right before, every time), and
  otherwise refuses and says why.
- Lootnest never turns an anti-cheat off itself, and never starts a game
  in a way that skips it. If the game has its own way to play the story
  without it (a setting in its launcher), that's the user's to use.
- The choice is per game, and can be taken back.
The owner agreed to these conditions (20 Nov 2026). Not built yet.

What still holds for story games:
- No DRM bypass. Denuvo and other anti-tamper count as DRM, and going
  into such a process can crash it or look like tampering, so games with
  anti-tamper stay out until there's a safe way to tell.
- Nothing that makes a game start without its owner's copy (no cracks,
  no emulators shipped).
- A copy of the user's files before anything is written.

What injection can't do: freeze a running PC game and resume it later,
as an emulator's save state does. A game's state lives in GPU memory,
driver and Windows handles, threads and open files, and a copy of its
memory can't bring those back. Injection can make the game save itself
(per engine: Bethesda's `save` console command, Unreal's
`SaveGameToSlot`, Unity games' own save managers, or the quicksave key),
load a chosen save, and read facts for a save point (chapter, place,
level). That's the plan for the save engine's rework (below).

## Stack and layout

- Go (no cgo) plus a WebView2 window (`third_party/go-webview2`, modified).
  On Linux, cgo with a WebKitGTK 4.1 window (`internal/webkit`) and the
  desktop's dialogs, keyring and tray (`internal/desk`); build it with
  `./build.sh linux` (needs libwebkit2gtk-4.1-dev and libsecret-1-dev).
  The UI is plain HTML, CSS and JS with no build step, embedded with
  `go:embed` and inlined at start with a CSP nonce.
- `internal/core`: platform-neutral logic, with tests. It covers config,
  the swap engine, scanning, achievements, cloud sync, archives and covers.
- `internal/app`: the Windows app. `api.go` binds `ps_*` functions on
  `window`; Go sends events through `window.__ps({type, ...})`. Slow work
  runs in a goroutine and comes back as an event.
- `internal/win`: Win32 helpers (DPAPI, window closing, tray and so on).
- `internal/plat`: the only way `internal/app` reaches the system. On
  Windows each name forwards to `internal/win` (`win_windows.go`); other
  systems get do-nothing stubs answering `plat.ErrUnsupported`
  (`stub_other.go`). Both are written by `go run ./tools/platgen`: run it
  after adding to `internal/win`, and call `plat.X`, never `win.X`, from
  app. A hand-written file in plat (say `trash_linux.go`) takes its names
  out of the stubs. `internal/app` builds for Linux; what can only exist
  on Windows (the WebView2 window, `RunWebApp`, process flags) is in its
  `*_windows.go` files with a `*_other.go` twin. Linux is docs/ROADMAP.md,
  section 9.
- `internal/app/ui`: `index.html`, `system.css` (tokens and `ln-`
  components), `app.css`, and the script in `js/`: one closure cut into
  ordered parts (`00-head.js` to `99-boot.js`, about 5,600 lines) that
  `app.go` (`uiJS`) and the harness join in name order. A part isn't valid
  on its own; put new code in the part it belongs to.
- `design/trailer`: the trailer, a dev-only film made from this repo (its
  README has the steps). `record.mjs` records the harness's page frame by
  frame (Playwright's clock plus every animation stepped by hand, a drawn
  pointer over the real mouse; takes and their cues are in `takes`);
  `trailer.html`/`trailer.js` is one page where `seek(t)` places every
  frame (titles, the swap diagram, the keys, the TV, the end; each take's
  offset in the film); `music.mjs` makes the sound: "Voltaic" by Kevin MacLeod
  (the owner's pick by ear, 16 Nov 2026; CC BY 4.0, so the film needs the
  credit in the README wherever it is shown), cut on its bars at 120 BPM,
  under hits and the app's own interface sounds at the picture's cues
  (`TRACK=` tries another piece, `TRACK=score` a synthesized one;
  `render.mjs mux <name>` lays a sound under the rendered picture);
  `render.mjs` screenshots each frame into ffmpeg
  (`out/lootnest-trailer.mp4`, 1080p60, 70 s; `sheet` and `still` to look
  at parts). `out/` isn't committed. Names are "Player 1" and "Player 2"
  (`anon=1`). When the launcher's look changes, record and render again.
- `cmd/lootnest`, `cmd/setup` (the installer and its `setup.html`),
  `tools/mkicon.py` (draws the icon), `build.sh`.
- `site/`: the website (https://c-dinca.github.io/lootnest/), plain
  HTML, CSS and JS published to GitHub Pages by `.github/workflows/site.yml`
  when `site/**` changes on main. Its hero is a live swap demo
  (`site.js`), a game's "Load game" screen with slots. Screenshots in
  `site/img` come from the harness at 2x with `?anon=1&runs=1`: the site
  shows no real names, only "Player 1" and "Player 2" (the owner asked).
  The site is short on purpose (the owner: less text, more alive, no
  pile of screenshots): two scroll stories played in a Lootnest window
  drawn in HTML and CSS (the owner: "make the animations look like the
  actual launcher"): profiles and runs, then achievements, save points,
  mods, updates and Big Picture; then Also in it, Never, Download. The
  owner chose this over two other directions (13 Nov 2026). The window
  is drawn in the site's palette, which the launcher is to follow; when
  the launcher's look changes, keep the window in step with it.
  The README is short too; details live in `docs/GUIDE.md` and
  `docs/PRIVACY.md`, its screenshots in `docs/screenshots` come from the
  harness without `anon`. The harness's profiles are made-up names
  (Alex, Sam...): never the owner's or anyone real.
  A change only to `site/` or docs ships no app, so it needs no VERSION
  bump or release.
- `design/harness`: a dev-only fixture harness that renders the real UI in
  Chromium with stubbed `ps_*` calls.
  - `node design/harness/build.mjs` writes `dist/page.html`.
  - `node design/harness/shoot.mjs <outdir> <filter>` screenshots into
    `design/<outdir>` (the filter is a substring). Delete the output
    afterwards.
  - Query options: `?players=6`, `?games=400`, `?state=loading|offline|error`, `?tv=1`,
    `?runs=1`, `?first=1`.
  - Stubs live in `stub.js`; add one for each new `ps_*` call.

## How changes are made

- **One feature per branch and PR.** Branch from fresh `origin/main`.
  Squash merges conflict with branches stacked on unmerged ones, so wait for
  the previous PR to merge, then branch again.
- **A release only when the owner asks** (the owner, 13 Nov 2026: too
  many releases; 1.97 to 1.106 came out in one afternoon). A change that
  ships in the app adds its lines under `## Unreleased` at the top of
  `CHANGELOG.md`, in plain user language, and leaves `VERSION` alone.
  When the owner asks for a release: bump `VERSION`, rename
  `## Unreleased` to `## x.y.z`, merge. A push to main whose VERSION has
  no tag yet publishes the release, using that section as its notes.
  Don't push tags by hand. Old releases were pruned to milestones with
  `.github/workflows/prune-releases.yml` (run by hand; tags are kept).
- Before pushing, run:
  - `gofmt -l internal`
  - `go test ./internal/core/`
  - `GOOS=windows go vet ./internal/app`
  - `GOOS=linux go vet ./internal/app` (and `go run ./tools/platgen`
    after changing `internal/win`; CI fails when plat is out of date).
    On Linux with WebKitGTK installed, also `go vet ./internal/webkit
    ./internal/desk ./cmd/lootnest`, `xvfb-run -a go test ./internal/webkit/`
    and `xvfb-run -a dbus-run-session -- go test ./internal/desk/`.
  - `GOOS=windows go build ./cmd/lootnest`
  - `node tools/checkjs.mjs` (joins `ui/js` and checks the syntax)

  For UI changes, also screenshot the change in the harness and look at it.
- CI has a `windows` job (windows-latest: `go test ./internal/core/
  ./internal/win/` on NTFS, with the Windows-only tests in
  `core/swap_windows_test.go` and `win/win_windows_test.go`) and the
  `build` job (govulncheck, the exe, releases), which waits for it.
  `docs/TESTING.md` is the checklist for a real PC. The owner wants PRs merged as soon as CI is green. Open them
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
  - radius by role (labels square-ish, keys and panels a little rounder),
    keys on a lip, section titles as labels in the section's colour,
    never "rounded AI slop";
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

Since the rework (1.97, `docs/REWORK.md`, DESIGN.md) the look is the
website's: obsidian, glass over the game's art, premium and quiet.

- Obsidian `#111217` ground, raised `#181a21` / `#1f222b`; Ice Grey
  `#DFE5EB` text with `#a6aeb9` and `#8c94a0`.
- Racing Red `#EF233C` is the logo gem, the Updates count and one primary
  key where there's no Play, with bold near-black text on it. Never text.
  The Play key is the seated profile's colour.
- No section colours (`--sec-*` are all one grey now). Colour only for
  profiles, achievement rarity, lamp green (on), amber and danger.
- The room: the game in focus fills the window behind everything (`#room`,
  `renderRoom()` in 20-render.js); glass (`--glass*`) only over art.
- One floating glass sidebar, no top bar; the sections' light slides.
- Themes per profile: Lootnest (default), Arcade, High contrast, and any
  a plugin brings (only colour tokens).
- Icons: Phosphor Icons (MIT) via `tools/icons.py` into `js/01-icons.js`,
  `ph(name, weight)`, 14 to 18px; never Lucide.
- Keys: a fine top light and rim (`--key-light`), 1px press; Play in the
  profile's colour with a sheen and a soft glow.
- Type: Tilt Warp for titles, a game's name, the Play key and big
  numbers; Atkinson Hyperlegible Next for everything else, with Figtree's
  digits (plain zero). All OFL, embedded.
- The app icon is the brand mark (`BRAND_MARK`: a Racing Red ring over
  two Ice Grey arcs) on an obsidian tile, the same sign as the sidebar's
  (the owner, 22 Nov 2026; the red gem in a nest is gone).

## Where things stand (2.23.1, beta)

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
  PC; Lootnest never offers Hydra's downloads or sources. The owner, 14
  Nov 2026: an update for a game from Hydra may open Hydra itself (the
  app, never a source or a page of it) for the user to get it there.
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
- **1.49.1** The Windows CI job. It found that Keep it offline failed
  (firewall rules need full, not 8.3, paths: `win.LongPath`; the script
  ends in `exit 0` so "no rules left" isn't a failure). Tokenized paths
  keep the PC's separator after the token (`%APPDATA%\Game` on Windows);
  compare them with `filepath.ToSlash` in tests.
- **1.50** Safety you can see (research step 1's open items):
  `Engine.RestoreBackup` puts a pre-swap copy back for its owner (live if
  loaded, else the profile; what's there is copied first, no pruning),
  drawer `copies` (`ps_gameBackups`, `ps_restoreBackup`). Steam Cloud
  replacements are kept in `conflicts.json` (`noteConflict`,
  `AppState.Conflicts`, `conflictBanner()`, `ps_dismissConflict`). The
  anti-cheat note (`antiCheatNote()`, `ps_gameFacts`). Still open: taking
  anti-cheat or online-only games out of swaps by default.
- **1.51** Solo first (the owner: "one by default, with profiles and
  runs"): a first start keeps one profile, named from `USERNAME`
  (`windowsUserName`), with no "who uses this PC" question. In app.js
  `people()`, `solo()`, `personOf()`; `#app.is-solo` and `.is-single`
  hide Who's here and auto-seat the only profile. A game's page has a
  **New run** chip (dialog `newrun`, action `new-run-go`: saves a
  `Player` with `Of`, then `play-as`, whose picker asks fresh or copy).
  Lootnest is positioned as the saves vault for anyone; the shared PC
  is the second use.
- **1.52** The safety net covers every save location a game outside
  Steam has (`vaultLocs`: `netLocs` plus `savesOf`, locations over
  `vaultLocMax` 256 MB outside the folder left out), not only in-folder
  ones; `netEntry.Folder` and `Vault` keep the "gone from its folder"
  notice about in-folder saves. Home counts them; the game page's block
  is **Saves**, with **Copies on this PC**. The website (`site/`, Pages)
  shipped with it.
- **1.53** Try a save safely (`core/trysave.go`, `app/trysave.go`,
  dialog `trysave` from Manage): `Engine.ProfileFromImport` starts a new
  run (`Player.Of`) as a copy of a profile's files (live if loaded), with
  one save location holding the brought save: a folder (`importRoot`
  goes down through an archive's single wrapping folder), one file (joins
  the folder), or a .zip/.7z/.rar (`StageUpdate` now reads .zip too).
  Nothing of the source profile changes. On anti-cheat or online-only
  games "out of swaps by default": profiles are only ever turned on by
  the user, so there's nothing to take out; the page's anti-cheat note
  (1.50) says what's swapped.
- **1.54** Tonight on Home (`tonightBlock`, `tonightPicks` weights: Up
  next 6, Playing 3, never started 1.5, not in 30 days 1.2, else 0.6,
  played today x0.3; finished, 100% and dropped out), a pick seeded by
  the day and the seat (`S.tonight`), action `tonight-spin` with a short
  cover roulette (none with reduced motion). This is "What do we play
  tonight?" from the agreed list, without the time-you-have part.
- **1.55** Your year in games (dialog `recap` from a profile's page):
  `recapData` sums `activity.days` for the year, `drawRecap` paints a
  1080x1350 canvas (art of the most played behind the year, four
  numbers, top five, months), `ps_saveRecap` (`app/recap.go`,
  `core.DecodePNGDataURL`, `core.RecapFileName`) writes it to
  Pictures\Lootnest and shows it in Explorer. Per profile; a household
  recap is still open.

- **1.55.1 to 1.55.4** Fixes from an audit run by agents (scratchpad
  backlog): restores copy every location a version writes
  (`Cloud.VersionTargets`, `keepBeforeRestore`); run names fit 24 runes
  and don't repeat (`app/names.go`: `trimRunes`, `trimWords`,
  `freeName`); one-file archives in Try a save (`onlyFile`), archives
  over 2 GB refused (`StageUpdateMax`); Big Picture toggle, Back, dialogs
  above it, no background blink; solo seat with runs, focus kept inside
  dialogs (`focusOverlay`, `focusBack`).
- **1.56** Library: scan results render at once, toasts can carry an
  action (Undo for Hide, `ps_unhideFound`), hidden games in Settings,
  right-click on every game row.
- **1.57** Real UI sounds: Kenney's Interface Sounds (CC0) embedded in
  `ui/sounds` and inlined like the fonts; `sfx()` over decoded buffers;
  unlock, save point and swap sounds behind `Settings.SoundsOff`.
- **1.58** Updates found in Downloads (`core/downloads.go`
  `MatchDownloads`, `app/downloads.go`): only the user's Downloads
  folder, matched by name and version to Library games outside the
  stores, announced once, applied through the existing update drawer;
  the downloaded file goes to the Recycle Bin on request
  (`win.Recycle`). `Settings.DownloadsOff` turns it off. Lootnest still
  never finds or fetches anything.

- **1.58.1 to 1.58.19** The design pass from research (19 batches;
  DESIGN.md's twelve principles): one page grid (`.view` owns
  `--page-gutter`), opaque surfaces (glass only over art or scrolling
  content), at most three Racing Red marks per screen, one entrance per
  page (`arriveOnce`, no loops), missing art as real content (`plate()`,
  icon washes, letter plates), Home 8+4 (`capHTML`, seat rows, save
  counts), game page with a one-row play bar (`play-pick`) and an
  unboxed 8+4 body, Library head and shelves (`--lib-cover`), Big Picture
  at 10-foot scale (`--bp-safe-*`, `--focus-ring-bp`, Collections row),
  empty pages (`emptyState`, `.ln-ghosts`, `.ln-steps`), Updates as rows
  beside a side column (`midTrunc`), Achievements headline and side
  column, Store feature plus Top sellers list, Settings and Profiles
  boards, Cloud conflicts first (`core.SavesChanged`), the profile page
  on most-played art, a full-window first run (`viewWelcome`), role type
  tokens (`--text-*`, Bricolage only at 24px and up), spacing roles
  (`--gap-in/group/section`), and short windows (Who's here as one row
  of faces under 820px tall).
- **1.59** Launch splash (`#boot`, `bootDone`, the window class brush is
  Obsidian so there's no white flash); the installer page uses the app's
  system (`app.SystemCSS()`), with steps that tick off; `--silent`
  (`/S`) for package managers, logging failures to
  `%TEMP%\Lootnest-Setup.log`; `build.sh app|setup` stages; SignPath
  signing and a winget job in build.yml, off until the owner sets
  `SIGNPATH_ENABLED` / `WINGET_ENABLED` (docs/RELEASING.md,
  packaging/winget/). The README has the code signing policy.

- **1.60** Ratings, notes and corrected hours per profile
  (`core.PatchShelf`, `ShelfPatch`, `ShelfEntry.Offset`, `WithOffsets`;
  an unreadable `shelf.json` is kept as `shelf.json.broken`).
- **1.61** A save point "Before the update" when a game changed since its
  last play (`core.GameBuild`: Steam build id or the exe's size and time,
  `Store.NoteBuild` in `builds.json`; `Settings.UpdatePointOff`), and the
  OneDrive note for saves under a synced folder (`core.OneDriveFolders`).
- **1.62** Big Picture comes back after a game it hid for (`a.hidFor`,
  cleared in `a.show()`; `win.ForegroundIs`), Close the game on X, and
  `Settings.OpenBigPicture`; the Library's Sources drawer
  (`core.SourceChecks`, `ps_sources`).
- **1.63** One Library entry for a game found in several stores
  (`core.GroupCopies`, `CopyName`; `Config.Primary`, `Config.Separate`;
  `copyGroups`, `libraryItems` against `everyItem` in app.js): Start with,
  Also in, Show copies separately. Display only; games with profiles are
  never grouped, and tags, status and rating read across copies.
- **1.64** Name search that forgives (`norm`, `nameMatcher(q)` returns a
  test per name; words in any order, no spaces, initials such as "bg3"):
  the Library, the sidebar, Ctrl+K, Add a game and favourites. Library
  covers in `#grid` sorted by Recently or Most played show that under the
  name (`cardHTML(it, true)`; `timeOf` is the one time figure for sorts,
  cards and rows); arrows and Home/End move between covers; F5 scans
  again; Settings, General lists `SHORTCUTS` (`.set-keys`); the first run
  shows the scan (`#wl-scan`, `welcomeScan`).

- **1.64.1** Unlock and save point sounds over a running game
  (`win.PlaySound`, embedded WAVs in `app/sounds`, `App.nativeSound`;
  events carry `native` so the window doesn't play them twice).
- **1.65** Roadmap Plugins 0: the bridge only answers the app's own top
  document (`MessageReceived` passes the source, `WebViewOptions.AllowMessage`,
  `core.PageSource.Trusted` pins the first caller's address; the upstream
  echo is gone). Safe mode (`--safe-mode`, Shift on a plain open, tray
  item, `a.safe`, `ps_restart`, `safeBanner()`). If a real PC ever shows
  an empty window from 1.65 on, check the log for "ignored N call(s)".

- **1.66** Roadmap Timeline 1: `sessions.jsonl` (`core.AppendSession`,
  `ReadSessions`; append-only, fsynced, bad lines skipped, under 60 s or
  over 24 h dropped), written where playtime is counted and for Library
  games Lootnest starts (`followFound`, `foundTick`); `ps_gameTimeline`
  merges sessions, save points, timed unlocks and applied updates;
  `timelineBlock` on the game page.
- **1.67** Roadmap UI 0, first part: the script split into `ui/js` (a pure
  move; the join was byte-identical to the old `app.js`), and Back and
  Forward (`hist`, `noteVisit` in `render()`, `goHist`, `pageBack`; Alt+Left
  and Alt+Right, Backspace, the mouse's side buttons), in memory, up to 50
  pages with their scroll. The editor isn't a stop.

- **1.68** Roadmap Controllers 1: `padFamily(id)` (vendor ids first,
  then words) gives xbox, playstation, nintendo or generic; `PADS`,
  `PAD_SHAPES` and `glyph()` draw each family; `padRole` maps Nintendo's
  confirm to its right-hand A, PS/Home to Start, the touchpad to View;
  an Xbox twin's presses are dropped while a real pad holds a button
  (`realPad`). Harness `?pad=ds5|ds4|xbox|switch|generic`.

- **1.69** Roadmap Game details 1 (`Settings.GameDetails`, off by
  default): `core.FetchGameDetails` (appdetails and appreviews, plain text
  via `HTMLText`), `details/<appid>.json` kept 30 days, the user's edits
  in `details-edits/` always win (`MergeDetails`); fetched only when a
  game's page opens (`ps_gameDetails`), never for the whole Library.
  `detailsBlock`, a screenshot lightbox (`viewShots`), Edit details.

- **1.70** Roadmap Timeline 3: Ctrl+Shift+F12 while a game runs
  (`quickShot`, `Settings.ScreenshotsOff`) saves a GDI copy from outside
  the game (`win.GameWindow`, `CaptureWindow` with PrintWindow, then
  `CaptureScreen` only when it's in front) to Pictures\Lootnest\
  Screenshots\<Game> (`core.SaveNewScreenshot` claims names with
  O_EXCL); a black frame saves nothing and says to go borderless. Save
  points made while the game runs keep `point.png` and `thumb.png`
  beside `point.json` (never restored into save folders); `ps_picture`
  serves them as data URLs; the 1.69 lightbox shows them.

- **1.71** Roadmap UI 1: custom cover, wide art and logo per game
  (`Store.SetCustomArt` copies a checked PNG, JPEG or WebP into
  `covers\custom` with `art.json`; `ClearCustomArt` deletes only that
  copy; `RekeyGame` moves it last), Manage, Cover and art. In app.js
  `artFor(it, kind)` picks custom, store, the other kind (crop or a
  cover over its own blur), then `typeCover` (name or logo); a logo
  replaces the title on the game page. Harness `?art=partial|missing`.

- **1.72** Roadmap UI 2: the game page in tabs (`gameTabs`, `tabRow`,
  `.ln-tabs`, `S.gameTab` per game, kept in history entries but not in
  their key), Overview, Achievements (inline, the drawer is gone), Saves
  (every point inline, the points drawer is gone), Timeline, News; a
  compact sticky head (`.gp-dock`, one IntersectionObserver in
  `stickHead`, `--gp-head-h`, `scroll-margin-top` on the page's parts).

- **1.73** Roadmap Timeline 2 and 4: captures other apps made (Game
  Bar, Steam's 760 folders, NVIDIA and AMD game folders, OBS names,
  Lootnest's own) are indexed in `captures.json`, never moved or
  changed (`core.CaptureSources`, `ScanCaptures`, `MatchCapture` by app
  id, name, then session time; `UpdateCaptures`, `AssignCapture`,
  `CaptureNoGame`); `Settings.CapturesOff`; the Timeline tab shows them,
  an Unsorted drawer assigns them, and a Game Bar tip shows when none.

- **1.73.1** Updates fall back to GitHub's API asset url
  (`Release.AssetURL`, `downloadRelease`) when the release link answers
  504, as it did for this repository on 8 Nov 2026.
- **1.74** Complete game manager 1: store hand-offs (`core.StoreAction`,
  `SafeHandoff` scheme allowlist, `FoundGame.StoreID`; Install, Uninstall,
  Verify files with Steam, Open in GOG Galaxy...), owned but not installed
  read from files on the PC (`SteamOwned` from localconfig and appinfo.vdf,
  `EpicOwned` from catcache.bin; Library segment Not installed, `ownedView`),
  and Remove from this PC for folder games (`PlanRemove`, `CopySaves` into
  backups\removed, then the Recycle Bin; the copies ledger offers the
  saves back on reinstall).
- **Owner, 8 Nov 2026**: yes to GOG and itch.io downloads of owned games
  through their official APIs after sign-in (roadmap section 7); next is
  a Steam-like shell (a top bar with the sections, profile and Settings;
  the sidebar only games, with collections you create and drag games
  into), real page transitions, animations and UI sounds on navigation.
  Plan in the session's manager-plan.

- **1.75** The owner's Steam-like shell, part 1: a top bar (`.topbar`,
  `--bar-h`) with Back/Forward, the section tabs and a sliding underline
  (`renderBar`, `#bar-ind`, a More menu under 1000px), search (`#gq`),
  Big Picture, Profiles, the seat as an avatar menu (`renderSeat`,
  `meMenu`: who's at the PC, Nobody, Add a run, Profiles) and Settings;
  the sidebar holds only games. Page changes: no View Transitions; the
  old page fades (`swapPage`, `.leaving`), the new one rises in parts
  staggered (`enter`, `--stagger`); `arriveOnce` per arrival. Interface
  sounds (`uiSound`, half Big Picture's gain, one per action, never while
  playing or with Sounds off). Next: 1.76 collections in the sidebar
  with drag and drop.

- **1.76** The owner's Steam-like shell, part 2: the sidebar's menu
  (`#side-show`: All games, Installed, Favourites, Not installed, Played
  by a profile), collections as folders in the user's order
  (`Config.TagOrder`, built-in `FavouritesTag`; `core/tags.go`
  `Collections`, `AddToTag`, `RemoveFromTag`, `RenameTag` merging,
  `DeleteTag` keeping games, `MoveTag`; `ps_tagOp`), New collection
  inline, pointer drag and drop (`sideDrag`; Shift moves; folder
  reorder), Ctrl/Shift selection, Delete takes out of a folder with Undo,
  and Add to collection in every game's right-click menu.

- **1.77** Launch options per game (`core.LaunchOptions` in
  `Config.LaunchOpts`: WorkDir, RunAsAdmin, CloseApps with Force,
  StartBefore, StartAfter, ReopenClosed; `CleanLaunchOptions`, `Moved`
  for new copies); arguments stay in `Game.Launch.Args`/`LibraryPref.Args`.
  `stepsBefore`/`stepsAfter(played)` around every start (WM_CLOSE, 10 s,
  Force only if ticked; never in safe mode), `startExe` (start folder,
  runas), `ps_runningApps`, Steam properties link. Drawer in `66-launch.js`.

- **1.78** Storage (a Settings category, `72-storage.js`): sizes counted
  in the background after scans (`core.DirSize` without following links,
  `SizeCache` in sizes.json, `sizesSoon`/`sizePass`, never in safe mode),
  drives (`win.DiskSpace`), a sortable table; Library sort Largest first.
  Move a folder game to another drive (`MoveGamePlan` refusals,
  `MoveGame` copy with read-back and compare, `retargetMoved` in one
  config write, ledger `NoteMoved`, then the Recycle Bin; launches into
  the moving folder are refused). Store games move with their launcher.

- **1.79** Library power: pick games with Ctrl/Shift click, Ctrl+A,
  Space and Shift+arrows (`S.libSel`, `libPick`, `paintLibSel`), a
  selection bar (`libSelHTML`: Add to collection, Remove from a
  collection, Set status for the seated profile, Favourite, Hide with
  Undo for scanned games only). Saved views (`Config.Views`,
  `core/views.go` `SaveView`/`DeleteView`, max 20; `ps_saveView`), a
  Views menu in the Library head and in the sidebar's menu.
- **1.80** Import from Playnite (`core/playnite.go`): a read-only
  LiteDB 4 reader (`readLiteDB`, pages and BSON, files over 512 MB
  refused; a locked file says to close Playnite), `ReadPlaynite`,
  `PlayniteImport` matches by store id then name and plans per game;
  time played goes to `playnite.json` per profile (added once, only
  what's new), statuses, ratings (`PlayniteStars`) and notes only where
  the profile has none, favourites and optionally categories or tags as
  collections. `BackupBeforeImport` copies Lootnest's records into
  backups\import\<id> first; `UndoImport`. Drawer in `71-playnite.js`,
  from Settings, Data and activity and the first run. Test libraries in
  `testdata/playnite` were written by LiteDB 4.1.4 itself.
- **1.81** itch.io, roadmap 7 phase 1 (the owner's exception: owned
  games, official API, after the user connects): `core/itchio.go`
  (`ItchAPI`, `ItchOwned`, `ItchUploads` reading snake or camel case and
  all three platform shapes, `ItchPickUpload`, `ItchKind` installer,
  exe or archive, `ItchNewer`, `ItchDownload` to .part with size and
  md5, https only, `ItchInstallArchive` into a new folder via a temp
  folder, never into one with files; `itch.json` records with `Prev`
  for Undo, moved by `RekeyGame`). `Settings.ItchKey` is DPAPI and never
  sent to the page. `app/itchio.go`: the owned list in the background
  (`itchLibrary`), `ps_itchInstall` with progress and Cancel,
  `ps_itchCheck` at most every 6 h, `ps_itchUpdate` downloads then opens
  the existing update drawer (`itchApplied`, `itchUndone`). UI in
  `73-itch.js`; harness `?itch=1&itchrun=hold`. Checked only with
  httptest: docs/TESTING.md section 17 on a real PC.
- **1.82** GOG, roadmap 7 phase 2 (`core/gog.go`, `app/gog.go`,
  `74-gog.js`): sign-in with the public client id Galaxy, Heroic and
  minigalaxy use, by pasting the on_login_success address (`GogCode`;
  no embedded browser yet); refresh token in DPAPI, access token only in
  memory and sent only to GOG's hosts. `GogOwned` (Windows games, wide
  art as `OwnedGame.Hero`), `GogInstallers` (patches not used),
  `GogFileLink` (downlink, checksum XML md5), the shared
  `core/download.go` (.part, https only, md5, Range resume), Inno
  `/DIR /SILENT` via `win.RunAndWait(path, args)`, then a scan by GOG id;
  `gog.json` via the shared `core/records.go`. Updates through the 1.40
  installer flow (`gogArgs`, `gogApplied`, `gogUndone`); a failed
  installer keeps the old record. UI notes shared in 73-itch.js
  (`storeNote`, `storeRunEvent`). Harness `?gog=1&gogrun=hold`.
- **1.83** Roadmap 7 phase 3, so section 7 is done: itch.io and GOG
  installs are rows on the Updates page (`storeUpdRows`, `storeUpdCard`,
  sharing `updRow`/`updMore`; the store's row wins over a folder row with
  the same key), counted under New and Needs a look only while the store
  is connected; Check again asks both stores too. `itchBad`/`gogBad`
  carry why a check failed. The 6 h throttle (`storeDue`) asks each
  record once per run, since found updates live only in memory.
  `S.updAsking` keeps the page from asking again while it loads.
  Harness `?itch=1|out&gog=1|out`.
- **1.84** Roadmap Controllers 2: Settings, Controllers (`77-controllers.js`).
  `win.PadDevices` lists HID and XUSB interfaces with SetupAPI and
  CM_* properties only: no device is ever opened (no CreateFile, HidD_
  or HidP_), so a Bluetooth DualSense's mode is never switched; usage
  from compatible ids (`gameUsage`). `win.XInputPads`, `win.ServiceState`
  (read-only SCM). `core/controllers.go`: `modelOf` (vid/pid to model and
  1.68 family), `parseVIDPID` (USB, BT Classic and BLE ids), `Pads`,
  virtual pads (vJoy, Steam, ViGEm, ROOT/SWD), the `controllerTools`
  table and `ControllerNotes` (double input, missing HidHide).
  `ps_controllers` on page open and gamepad connect, never polled.
  Shared `setSub` in 70-settings.js. Harness `?translators=1`.
- **1.85** Roadmap Controllers 3: `GameDetails.Controller` (full or
  partial, from appdetails' controller_support, else category words) and
  `Pads` (only "playstation": Steam has DualShock/DualSense categories,
  no Xbox one, since full or partial already means an Xbox pad), in the
  same 1.69 request (`categories,controller_support`), only with Game
  details on. `DetailsEdit.Controller` ('none' clears Pads). UI
  `gameCtlLine`/`CTL_SAYS` in 77-controllers.js (Overview and Big
  Picture); the double-input note from `core.PadCopiers` (1.84's rule,
  sent as `copiers`). Harness `?hidhide=0`.
- **1.86** Roadmap Mods 1 (`core/mods.go`, `app/mods.go`, `67-mods.js`,
  a Mods tab): one set per game under `mods\<sha1 of key>` (each mod's
  staged `files` and `mod.json`, `set.json`, `deployed.json`,
  `originals\`, `changed\<time>\`, `journal.json`). `AddMod` stages
  (StageUpdateMax) and places by `modPlace`; `PlanMods` refuses Program
  Files/WindowsApps, anti-cheat (Vanguard, FACEIT, EA AntiCheat added
  to `AntiCheat`), the `modBlocked` table (steam_api, EOSSDK, emulator
  configs...), the main exe and traversal. `ApplyMods` = journaled
  undeploy then deploy; every replaced file kept and read back first;
  a changed file goes to `changed\` before anything replaces or removes
  it; added files deleted only when their hash matches. Put back works
  even when a deploy can't finish (undeploy from the journal's plan).
  `finishMods` at start (not in safe mode, skips running games);
  launches refused during a run or with a pending journal
  (`errModding`); moves refused with a pending journal. Saves copied
  first (`keepSavesBefore`). Harness `?mods=1`.
- **1.87** Roadmap Mods 2: `set.json` is a map by owner ('' the game's,
  else a profile id; the 1.86 list reads as the game's); `ModSetFor` /
  `ModSetIn` resolve profile, then its person (runs), then the game's;
  `ModsDeployed.Owner`. `modsForPlay` runs in `runSwap` right after
  `eng.Activate` (saves first, then mods; journaled `ApplyMods`); a set
  that can't go in puts the game back and starts it vanilla. Updates
  take mods out first (`modsOffBefore`) and leave the
  `off-after-update` marker until the next Apply. `ps_modSetOwn`; the
  Mods for select; New run's Start with mods.
- **1.88** Roadmap Mods 3: `placeMod` hardlinks (`modLink` = os.Link to a
  temp name, hash-checked, renamed in) when linking is on
  (`Settings.ModsLinkOff`, More menu), the file isn't a config
  (`modConfigFile`) and the staged copy can be made read-only; else the
  1.86 copy. `deployed.json` records How (from `os.SameFile`). Undeploy
  unlinks unchanged links; a staged copy the game changed marks the mod
  (`Mod.Changed`, `ErrModChanged`; Play then starts vanilla), the
  changed content kept in `changed\`. `ModConflicts`, conflict lines
  and the `modconf` drawer; drag to reorder (`ps_modOrder` takes the
  full order).
- **1.89** Roadmap Plugins 1, data only (docs/PLUGINS.md): a .zip with
  `plugin.json` and only data files (`checkPluginZip`,
  `CheckPluginDir`; 8 MB per file, 32 MB, 500 files), installed into
  `plugins\<id>` with the old version kept in `plugins\.old` for
  `RollbackPlugin`; on/off in plugins.json; all off in safe mode.
  Sources (`pluginsources.go`: folder, json, ini, vdf, registry read via
  `win.ReadRegKey` KEY_READ; Source `plugin:<id>`, ranked after Lootnest's
  own), save presets as suggestions only (`Env.PluginSaves`, never
  proposed), local metadata layered under Steam's and the user's
  (`LayerDetails`), themes (`plugintheme.go`: an allowlist of 14 colour
  tokens, contrast pairs checked in Go incl. glass over ground and the
  status colours; `Player.Theme`, `applyTheme` on the seat). UI
  `79-plugins.js`, Settings, Plugins. Harness `?plugins=1&theme=1`.
- **1.90** Roadmap Plugins 3 (`core/pluginsign.go`, `cmd/plugsign`):
  signature.json {key, sig} over `PluginDigest` (sorted
  'relpath\tsha256' lines); `VerifyPlugin`, `KeyFingerprint`. Trust on
  first use: the key is pinned per id in plugins.json; a different key
  is refused with both fingerprints unless the user accepts that exact
  fingerprint (`InstallPluginNewKey(..., accepted)`); unsigned can't
  replace signed; `olderPlugin` only offers rollbacks matching the pin;
  a plugin whose files changed on disk is turned off with a reason.
  Remove forgets the pin. `plugsign keygen|sign` for authors
  (plugsign.key 0600, gitignored).
- **1.91** Roadmap UI 3: the Saves hub (view `saves`, `76-saves.js`) in
  Cloud saves' tab (six tabs kept): stats, Needs a look, Recent copies
  (`core.SavesOverview` over `eng.Backups`, `SavePoints` and the safety
  net; `ps_savesOverview`), Per game (sortable), then the old Cloud view
  as `cloudSection()` (`nav` maps view `cloud` to it). Ctrl+K runs
  commands (`gqCommands` in 80-store.js; existing actions only; Enter
  runs a command only when one is picked). `SET_CATS` is the one list of
  Settings categories; `open-sheet` takes `data-tab`.
- **1.92** Smooth page changes (the owner: "not smooth"): `swapPage`
  draws the new page at once while a still, inert copy of the old one
  (`.page-ghost`) fades out over it; on-screen images decode first (at
  most 120 ms); `enter()` animates the page as one piece; the backdrop
  changes with the page (`#backdrop.with-page`, 280 ms). Still open:
  the game page is redrawn several times by events in its first 600 ms.
  Design directions being tried live in `design/directions/*.css`
  (harness only, `dist/page-<name>.html`).
- **1.93** Arcade, part 1 (the owner chose direction 7 of seven mocked;
  they want it premium, with real motion, and later all explanatory
  text cut, "like other launchers"): fonts Tilt Warp (titles, the Play
  key, scores) and Atkinson Hyperlegible Next with Figtree's digits
  (Bricolage gone); system.css tokens for the indigo shell and screen,
  section colours (`--sec-*`, `--sec` via `[data-sec]` set by `secOf()`
  in render), lamp-green switches, keys on a lip that press 2px,
  `.ln-label` section labels; motion: the 1.92 ghost crossfade plus an
  arrival choreography in `enter()` (head settles, labels wipe, content
  fades, under 400 ms), the top bar cartridge sliding in the section
  colour, key press, cover lift, drawers, dialogs, toasts, the seat pop.
  The window backdrop is gone. Next: 1.94 Library/game page/Big
  Picture, 1.95 Settings/Profiles/dialogs, 1.96 the other pages, 1.97 a
  copy sweep.
- **1.94** Arcade, part 2: the Library (teal labels, the selection bar
  as a front panel, install keys on Not installed), the game page (keys
  beside Play, tabs as cartridges in a well, the compact head in the
  shell colour), Big Picture (shell, `data-sec` per tab, a sliding
  cartridge via the shared `slideInd` in 20-render.js, keys, Play red or
  the profile's colour, `bp-screen-in`), and the explanatory text cut
  from those pages (one short line kept only before file changes).
- **1.95** Arcade, part 3: Settings on one row system (`setRow`,
  `setLine`, `setSub` drawing the section label; label left, control
  right, `--set-row-h` 56px, content capped at `--set-nav-w + --set-w`),
  Controllers as two aligned columns and app rows with a status lamp
  (`.ctl-lamp`), Profiles as person cards with runs inside, the picker
  as big keys that light and press (`.picked`), the profile editor in a
  wide drawer, every older `.btn` restyled as a key, and dialogs with
  one foot (Cancel and the main key). Explanatory text cut across all
  of them.
- **1.96** Arcade, part 4: Saves, Updates, Achievements, the Store, the
  first run and empty pages. Shared pieces in system.css: `.ln-score`
  (scoreboard, `--text-board`), `.ln-rows` (rows on a panel, `--row-h`),
  `.ln-lamp` (`--on`, `--warn`); `lookRow` for Needs a look rows;
  `emptyState(title, line, key, ghosts)` is one pattern (label, line,
  key). The first run is a boot screen (the gem draws in, a lamp-green
  scan bar, name and colour, Continue; under 600 ms). Achievement rings
  are the section colour, green when finished; tier colours only for
  rarity. site/img regenerated in the new look.
- **1.97** The rework, R0 (`docs/REWORK.md`): the website's palette in
  system.css (obsidian, glass tokens, no section colours), the room
  (`#room`, `renderRoom()`, `applyAppearance()`), one floating glass
  sidebar replacing the top bar (index.html; ids kept), Phosphor icons
  (`js/01-icons.js`, `ph()`, `paintIcons()`, `tools/icons.py`; `GI` maps
  roles to them), keys with a top light, pill tabs and segmented
  controls, plain section titles, glass drawers and toasts, Settings,
  Appearance (`Settings.GlassOff`, `RoomOff`, `Motion`; action
  `set-motion`). Plugin themes accept `rgb()`.
- **1.98** The rework, R1, Home: the game in focus (`S.homePick`, else
  the last played) is the room (`S.roomPick`); `.home-stage` holds its
  title and a glass play dock (`gameView`'s bar) beside `homeCard()`
  (achievements, rarest, time, save points); `homeShelf()` covers, action
  `home-pick` (first click picks, second opens), arrows move; the room
  veils as the page scrolls (`veil()`, `.room-veil`).
- **1.99** The rework, R2, the Library: the room shows the game the
  pointer rests on (`S.libFocus`, a 220 ms pointerover in 20-render.js),
  else the latest with art, dimmed (`--room-dim`; the veil stops at 35%
  there); covers at `--radius-cover` 10 lift 5px on `--ease-spring`; the
  tools bar is glass, its menus pills; the selection bar is glass.
- **1.100** The rework, R3, a game's page: the room is the game (the
  page's own art, plate and hero scrim are hidden), the logo or name and
  facts over it (`--scrim-title` under the facts), Back and Manage glass;
  `.gpage-body` is one glass sheet (`--glass-strong`, `--radius-glass`)
  holding the play row, the tabs in a well and every tab; the docked
  head (`.gp-head`) and the screenshot lightbox are glass. Home keeps its
  own stage (`.gpage.home`). Drawers from Manage are R4.
- **1.101** The rework, R4, overlays: `.dialog` glass over a blurred
  scrim (`--blur-scrim`), Tilt Warp titles, no rule above the foot;
  `.ctx` and Ctrl+K glass (`gqCover` puts a game's cover in its row);
  the picker's `.slot`s are glass keys that lift and glow in the
  profile's colour; the editor (`viewEdit`) has its game as a dim room
  (`roomItem`) and a floating glass `.form-foot`. Drawers were glass
  since R0.
- **1.102** The rework, R5, the other pages: Saves, Updates,
  Achievements and Profiles (`PAGE_ROOM` in 20-render.js) have the last
  game played with wide art as the room (`lastWithArt`), dimmer than the
  Library's (`.room.is-page`, `--room-dim-page`); `html.art-behind`
  turns `--bg-surface` inside `.view` into `--glass-panel-page`, so
  their panels are glass. A profile's page keeps its own banner; the
  Store keeps its feature art; dialogs and drawers were R4.
- **1.103** The rework, R6, Settings: `settings` joins `PAGE_ROOM`, so
  every category's boards are glass over the dim room; `.set-nav` is a
  glass column, `.set-ic` bare icons (no tiles, no section colour), the
  current category lit with `--key-light`; Games and swaps uses
  `arrows-left-right`.
- **1.104** The rework, R7, Big Picture and outside the window: `.bp` on
  the ground with its art at 0.8; `slideInd` now moves and stretches
  (translateX plus width, no scaleX) so `.bp-ind` is a glass pill; row
  labels plain Tilt Warp without the rule; `.bp-btn`, `.bp-exit`,
  `.bp-sound`, `.bp-achcard`, `.bp-person` glass with `--key-light`. Its
  block sits at the end of app.css so it wins over the 1.49 rules.
  `win/toast.go` colours follow the tokens (`toastBadge`). The boot
  splash and setup.html already used the system.
- **1.105** The rework, R8, speed (measured in the harness with
  `?games=400`): covers, wide art and icons are kept as blob: URLs
  (`artStore()` in 10-state.js wraps `S.covers`, `S.icons`, `S.art`;
  a data: URL over 4 KB goes through `blobURL`), so page HTML no longer
  carries megabytes of base64; one `Intl.Collator` (`byName`) for every
  sort by name; the sidebar keeps what it drew in `sideDrawn`, not a DOM
  attribute; `render()` under Big Picture draws only the overlay and Big
  Picture (`behindBP`), and the window catches up when it closes. Big
  Picture opened in 10.2 s, now 0.1 s; the Library 815 ms, now 345 ms.
  The cover flying into its game page (FLIP) is still open.
- **1.106** The rework, R9, so the rework is done: built-in themes on the
  plugin theme machinery (`BUILTIN_THEMES` in 79-plugins.js, `allThemes()`
  for the profile editor and `applyTheme`): `ln-arcade` (the 1.93 to
  1.96 indigo tokens, glass tinted to match) and `ln-contrast` (black,
  white text, solid glass, no blur). Plugin ids starting "ln-" are
  refused (`core/plugins.go`). Harness `?theme=ln-arcade|ln-contrast`
  sets the first profile's theme. README screenshots regenerated; the
  website's window has the sidebar with the sections (no top bar).
- **2.0.0** The owner's release of the whole rework (1.97 to 1.106) as
  one version, with a summary in CHANGELOG.md; no code change. From here
  on, releases only when the owner asks (see How changes are made).
- **2.1.0** The owner's list of 14 Nov: the profiles pill
  (`profPill`), Patch notes to News (`S.newsOpen`), Updates' All without
  up-to-date games, the sidebar's speaker (`sound-toggle`,
  `renderSound`); Will it run on a game's page (`willRunBlock` in
  62-game-blocks.js) from Game details' `pc_requirements`
  (`GameDetails.Minimum/Recommended/ReqsRead`; old cached details are
  read again once) and `core.PlanRun` (score, preset, resolution, fps,
  upscaling); the page's Status, rating and note were removed (Library
  filters and the selection bar still set status). Achievement cards:
  `achRow` registers each in `ACH_SEEN` and opens dialog `achcard`
  (`viewAchCard`, pointer tilt, drag to turn, click/Enter to flip, arrows;
  `data-go` from the Achievements page opens the game's tab first).
  Saves: `SV_KINDS` explains each kind; a row's icon opens its files
  (`ps_copyFiles`, `core.CopyFiles` on a folder or .zip, at most 200);
  the pencil names any copy (`ps_nameCopy`: save points through
  `RenameSavePoint`, others in `copynames.json`, `core.SetCopyName`).
  Updates: the drawer's first screen is a guide (`updGuide` in
  75-updates.js: get it, choose it here, check and apply); for a Hydra
  game `ps_openHydra` copies the name and opens `hydralauncher://` (Hydra
  has no link to a game's page, only `run`), and `downloadsLook` also
  reads Hydra's own download folder (`core.HydraDownloadsPath`, the root
  key `userPreferences`, `downloadsPath`).
  Community at the end of Overview (`communityBlock`, `reviewCard` in
  62-game-blocks.js): `core.FetchReviews` (appreviews, filter=all, the
  past year, BBCode to text by `ReviewText`), `ps_gameReviews` with a
  six-hour memory cache, asked only when the block scrolls into view
  (`watchCommunity`), only with Game details on. Giscus comments wait on
  the owner: Discussions on, the giscus app installed, then its repo and
  category ids; the page's CSP (`frame-src 'none'`) and
  `core.AllowedPageRequest` would need giscus.app.
  Motion: `moving()` (40-game-page.js) is false with reduced motion or
  Motion Less/Off; `coverFrom`/`openCover` grow a copy of the clicked
  cover (`.cover-fly`) into the page; `noteGlide`/`glide` (20-render.js)
  move a stand-in `.glide` pill from the old lit choice of any
  `.ln-segmented` or `.ln-tabs` to the new one after the redraw; the game
  page's panel slides from the tab's side (`setGameTab`); `bump(el, n)`
  pops a count that rises.
- **2.2.0, the owner's fixes of 15 Nov**: Steam's `appreviews`
  redirects to the store front since 22 Oct 2026, so reviews and the
  review score use `IUserReviewsService/GetAppReviews/v1` on
  api.steampowered.com (`reviewsQuery`, `input_json`, no key, the answer
  under `response`). Profiles off stays on the page (`remove-game` finds
  the scanned game by app id, exe or name before drawing).
  `apiGameFacts` stats the path instead of trusting `filepath.Ext`
  (dotted folder names read the parent). Will it run: `gpuFromWords`
  (VRAM, DirectX) and `cpuFromWords` (cores, threads, GHz) when no model
  is known (`orWords`), older cards in `gpuScores`, i3/i5 first
  generation, and `rate` gives high at 3x and recommended at 1.8x a lone
  minimum.
  Motion on springs (`js/03-motion.js`): `spring(bounce, visual)` samples
  a damped spring (Motion's mapping: stiffness (2π/visual)², damping
  2(1-bounce)√stiffness) into `linear()`; `SPRINGS` press, snappy,
  bouncy, soft set `--spring-*` and `--spring-*-dur` (fallbacks in
  system.css). The "springs" block at the end of app.css moves keys,
  covers (`coverTilt` sets `--tx/--ty/--lx/--ly`), dialogs, toasts,
  menus, drawers, switches; `enter()` marks up to `RISE_MAX` blocks in
  view (`RISERS`) as `.rise` with `--i`; `revealIn` does the same for
  content replacing a skeleton (`storeSkeleton`, `.ln-skeleton::after`
  shimmer). `arriveOnce` keys on arrival, view, page and tab, so rings
  and bars fill when achievements arrive after their page.
  Trailers: `GameDetails.Movies` (appdetails `movies`, only `hls_h264`:
  Steam serves no mp4 any more; `SafeVideo` pins video.akamai.steamstatic.com
  and .m3u8; `MediaRead` makes old copies stale once). hls.js 1.7.4
  (Apache 2.0, `ui/vendor/hls.min.js`, embedded as `uiHLS`) is fetched by
  `ps_vendor("hls")` and run as a script with the page's nonce the first
  time a trailer shows (`loadHls`). CSP: `media-src blob:` and the video
  host, `connect-src` the video host; `AllowedPageRequest` allows `blob:`.
  `mediaBlock`/`mountMedia`/`mediaGo` in 62-game-blocks.js keep the
  video element in `vids` across redraws (moved, never recreated).
  Will it run from players (`core/peers.go`, `Settings.HardwareReviews`,
  off by default): `SimilarGPUs` is this PC's card plus the two closest in
  `gpuScores` within 15%, in Steam's spelling (`gpuSteamName`);
  `FetchPeers` asks GetAppReviews with `hardware_gpu` (exact names) per
  card and sums recommend share, performance mentions (`perfMention`),
  named frame rates (median at 3 or more) and up to six quotes with the
  writer's hardware (`perfSnippet`). `ps_gamePeers`, 12 h memory cache,
  `peersBlock` under the lanes. `win.TotalRAM` reads
  GetPhysicallyInstalledSystemMemory first (16 GB, not 15.x).
  Saves: `apiSavesOverview` sends every copy (`hubRecent` 2000); `svBar`
  (search `#sv-q`, kind, profile, when, order in `S.svF`) and
  `svResults` filter them; typing redraws only `#sv-results` and the Per
  game table so the box keeps focus.
  Achievements: `.ach-hero` (ring in the profile's colour, the count,
  three rarest as medals, Share), `S.achSort`. Sharing (`81-achshare.js`):
  dialog `achshare` (`achRowsFor`, `drawAchShare` on a 1080x1080 or
  1080x1920 canvas; remote icons load with crossOrigin and are left out
  without CORS so the canvas stays exportable), `ps_shareImage` (`app/
  share.go`): "copy" via `win.SetClipboardImage` (CF_DIB plus the
  registered "PNG" format), "save" into Pictures\Lootnest\Achievements
  (`core.ShareFileName`).
- **2.2.0, Lootnest Link M1, TV mode** (`docs/lootnest-link-spec.md`,
  the owner's plan for playing on a TV: M1 TV mode, M2 a bundled Apollo,
  M3 to M4 TV apps; one milestone at a time, each checked on the owner's
  LG TV). `lootnest.exe --tv` is only a holder (`app/tv.go` `TVHolder`):
  a streaming host ends its app's whole job when the stream stops, so the
  Lootnest that swaps files is never the host's app. The holder starts
  Lootnest with `--tray` outside the job when it can
  (`win.StartOutsideJob`, CREATE_BREAKAWAY_FROM_JOB; `win.InJob` logs a
  warning otherwise), sends `tv:<pid>`, and asks `tv?:<pid>` every second
  (`win.AskCommand` returns the WM_COPYDATA answer, `tvAnswerOn`) until
  TV mode ends. `enterTV` (Big Picture on the primary display,
  `win.FullscreenPrimary`, again on WM_DISPLAYCHANGE; `win.ForceForeground`
  with AttachThreadInput, then a lone Alt), `leaveTV` when the holder
  dies (`win.WaitProcess`) or from the menu (`ps_tvExit`), back to the
  tray if it was there. In TV mode the window hides for every game,
  Library games too (`foundTick` returns up/down), and comes back with
  focus (`tvBack`). UI: `BP.tv`, `tvOpen`/`tvClose`/`bpShut`, `BP.who`
  (Who's playing first unless solo), `BP.menu` on Start (`bpMenu`,
  `startWord`), `html.is-tv` raises small sizes to `--tv-text-min` (24px
  at 1080p); the pad drives the picker and dialogs over Big Picture
  (`overlayMove`, shared `nearest`). Harness `?tv=1`. Next in M1: the
  controller as a mouse over store launchers (the owner, 15 Nov: make it
  work with the controller, as in games, rather than a "may need mouse"
  badge).
- **2.2.0, Lootnest Link M2, TV streaming set-up** (Settings, TV,
  `82-link.js`; `app/link.go`; `core/apollo.go`). Apollo 0.4.6's API (the
  newest release; 0.4.8 is only a tag, same API), read in its source (`src/confighttp.cpp`): a session cookie from
  `/api/login` (not Basic auth, no CSRF), `/api/password` without
  sign-in only while no user exists, `/api/config` GET gives every key as
  text plus status/platform/version/vdisplayStatus and POST rewrites the
  whole file (`MergeApolloConfig`), `/api/apps` saves by uuid,
  `/api/pin`, `/api/clients/list|update|unpair` (perm bits in
  `crypto.h`; a new client gets only list and view, so `AllowTV` sets
  `ApolloTVPerm`), `/api/otp` (host-made PIN for Artemis, for M3).
  `core.Apollo` pins the certificate's SHA-256 on first contact
  (`VerifyConnection`), talks only to localhost, never through a proxy.
  `ApolloPinned` (version, URL, SHA-256; `DownloadApollo` refuses without
  the hash; `downloader.get` takes a 64-digit sum as SHA-256). One
  elevated PowerShell (`ApolloInstallScript`): NSIS `/S /D=`, then
  `sunshine.exe --creds` from a temp file the script deletes, the
  "Apollo" firewall rules to Private, the ApolloService started.
  `link.json` (user, DPAPI password, cert pin, Off) is never sent to the
  page. `configureEngine`: `ApolloSettings` (name "Lootnest on <PC>",
  `origin_web_ui_allowed=pc`), `LootnestApp` (`--tv`, virtual display,
  wait-all off, auto-detach off), restart. `linkAtStart` points the app
  at the current exe. Bound calls run on the window's thread, so every
  engine call is a goroutine with `link*` events. Harness
  `?link=none|busy|set|off|unpinned`. Not done: uninstalling Apollo with
  Lootnest (needs elevation).
- **2.2.0, Lootnest Link M3, Add an LG TV** (`core/webos.go`,
  `app/lgtv.go`, `viewLGTV` in 82-link.js). The TV app is
  c-dinca/lootnest-tv-webos (a fork of Moonlight TV, GPL v3; its own
  changes in `src/app/lootnest.c`: launch params `host` + `pairPin`,
  auto-launch of the app named Lootnest, first paired PC, one WoL,
  Start+Select 2 s quits). `FindLGTVs` (SSDP: LG second-screen and DIAL
  targets, `LooksLikeLG`, friendlyName from the description),
  `FetchTVKey` (Key Server `http://<ip>:9991/webos_rsa`, `UnlockTVKey`
  with the passphrase), `ConnectTV` (SSH prisoner@9922, host key pinned),
  `InstallTVApp` (cat into /media/developer/temp, `luna-send-pub -i`
  appInstallService/dev/install, `InstallOutcome`), `LaunchTVApp`
  (applicationManager/launch with params), `DevModeToken`
  (/var/luna/preferences/devmode_enabled) and `ExtendDevMode`
  (developer.lge.com ResetDevModeSession). `installTV`: key, download
  the pinned IPK (`TVAppPinned`: Lootnest for TV 1.0.0, its
  release's asset digest; a TV app release is made by bumping VERSION on
  its main), install, save the TV (key DPAPI in link.json `tvs`), launch
  with a random PIN and this PC's address on the TV's network
  (`TV.LocalIP`), then `/api/pin` every 2 s for 75 s, `AllowTV`.
  `keepDevModes` hourly, extends each TV every 12 h. SSH via
  golang.org/x/crypto/ssh (the only new dependency).
- **2.2.1** Add an LG TV on the owner's TV: the install service ended its
  answers without an outcome and the deferred `rm -f` removed the package
  under the TV's background install. `QuietInstallError` now waits up to
  2 min for `HasTVApp` (the app's folder, or getAppInfo) before the
  package goes; a failure carries what the TV printed (`2>&1`). SSDP found
  nothing (Windows Firewall drops replies from another address than the
  one asked), so `FindLGTVs` probes the private /24s for port 3000
  (`probeLGPort`).
- **2.2.2** Add an LG TV, second try on the owner's TV: the install
  printed nothing and nothing was installed. `luna-send-pub -i` reads
  commands from stdin and, with none (the SSH session's stdin is closed),
  ends before it calls the service. `InstallTVApp` now checks the copy's
  size (`wc -c`), sends one `luna-send-pub -n 1` request with
  `subscribe: false`, and polls `HasTVApp` for 3 min; a refusal
  (`returnValue: false`) says the TV's `errorText`. The subscription
  reader (`InstallOutcome`) is gone.
- **2.2.3** Third try: it still printed nothing. As ares-install and
  ares-cli-rs do, `InstallTVApp` runs `luna-send-pub -i` subscribed with
  its stdin held open (`StdinPipe`, closed after), reads
  `InstallOutcome` again, then polls `HasTVApp` 90 s; the error names
  `command -v luna-send-pub`. Every SSH command has a limit
  (`runFor`, a minute by default). The fake TV quits silently when its
  input closes first (`TestTVInstallKeepsInputOpen`). App: a busy engine
  answers `linkTVDone` with the step instead of ignoring Install; each
  step is logged ("Add an LG TV on <ip>: ..."); TV steps have their own
  names (`tvdownload`, `tvinstall`, `tvpair`) so they don't borrow the
  engine's "Allow it in Windows' prompt"; the drawer counts seconds
  (`D.started`, `.lg-secs`).

- **2.2.4** The TV app installed on the owner's TV but showed Moonlight
  TV's launcher. Lootnest for TV 1.1.0 (c-dinca/lootnest-tv-webos#3)
  has its own home screen (`src/app/ui/lootnest/home.controller.c`, the
  root fragment instead of the launcher; state kept module-level since
  some TVs close and reopen the UI around a stream): pairs from the
  launch parameters, connects to a saved PC (saved PCs read as unpaired
  until queried, so `known` ones count), wakes it for 90 s, opens the
  app named Lootnest; Play, Settings, Quit after a stream; the gem, Tilt
  Warp and Atkinson Hyperlegible Next embedded. `LaunchTVApp` closes the
  app first (`closeByAppId`, `tvCloseWait`), since an open app never
  reads new launch parameters. The fork's desktop CI fails only on
  `test-gamestream-crypt-sign`, as upstream does.

- **2.2.5** On the owner's TV, 1.1.0 installed but nothing appeared and
  the engine's log showed no contact from the TV; the network was also
  Public. `LaunchTVApp` waits until the app's process is gone
  (`AppRunning`, `ps` on the TV), launches, checks it runs, launches once
  more if not, and returns a trace that's logged. A failed pairing logs
  and names why (`AppRunning`, `ReachesPC`: wget or curl to
  `:47989/serverinfo` from the TV, the network category). Public
  networks: `core/netprofile.go` (`NetworkCategoryScript`,
  `MakePrivateScript`), `win.RunHiddenOutput`, `PublicNetworkError`
  before the key step, **Make this network private** (`ps_linkMakePrivate`).

- **2.2.6** The 2.2.5 log: close and launch answered nothing, the app
  never ran, the TV reached the engine. Every `applicationManager` call
  goes through `TV.luna` (`luna-send-pub -i` with its input held open
  until the first `returnValue` line, `tvLunaLimit`), as the install has
  since 2.2.3: an SSH command given no input has it closed at once and
  luna-send-pub then calls nothing. The fake TV ignores such calls.

- **2.2.7** With 2.2.6 the TV app opened but showed "This TV isn't
  paired with a PC yet": it was already open, the owner's TV has no
  `closeByAppId` ("Unknown method"), and an open app gets no new launch
  parameters. `TV.LeavePairing` writes
  `/media/developer/temp/lootnest-pair.json` (`id`, `host`, `pairPin`)
  before the launch, `ClearPairing` after; Lootnest for TV 1.2.0
  (c-dinca/lootnest-tv-webos#4, `lootnest_check_pair_file`) reads it at
  start and every 3 s while its screen waits, once per id, and notes its
  steps in `lootnest-tv.log` beside it (`TV.TVAppLog`, logged when
  pairing doesn't finish). `AppRunning` reads `getForegroundAppInfo`: a
  developer session's `ps` doesn't list developer apps. closeByAppId is
  tried on `com.webos.service.applicationManager` first.

- **2.2.8** The TV app's own log (2.2.7 reads it) showed the launch
  parameters arrive and the PC answer, then "no PC": in the fork,
  `pcmanager_update_by_host` never set the worker context's uuid for a
  host added by address, so the callback got an empty id. Lootnest for
  TV 1.2.1 (c-dinca/lootnest-tv-webos#5) sets it from the host's answer.
  The owner (15 Nov): fix and publish without asking for these TV fixes.

- **2.2.9** The owner's TV paired and streamed with 2.2.8, but showed
  only the virtual display's desktop: TV mode went full screen on the
  PC's primary monitor. `ApolloSettings` adds `dd_configuration_option =
  ensure_primary` (Apollo 0.4.6 `config.cpp`; `process.cpp` sets
  `output_name` to the new virtual display before `configure_display`),
  so the virtual display is primary while a stream runs and
  `FullscreenPrimary` and games land there. `linkAtStart` brings an older
  engine's settings up to date (`ApolloSettingsDiffer`, SetConfig,
  Restart) when no app is running (`current_app` empty).

- **2.2.10** A controller in the owner's TV's USB port showed "HID
  device connected" but did nothing: the TV's SDL had no mapping, so the
  fork ignored it. Lootnest for TV 1.3.0 (c-dinca/lootnest-tv-webos#6,
  `lootnest_guess_mapping`): Sony HID order for vendor 054c, else the
  Linux xpad order; ids, GUID and counts in `lootnest-tv.log`; the home
  screen's controller line (`lootnest_controller_line`). Save a report
  (`apiLinkLogs`) adds each saved TV's `lootnest-tv.log` (`tvAppLog`).
  Open with the owner: using the PC while the TV streams (one Windows
  session; the TV's display is primary during a stream since 2.2.9).

- **2.2.11** The owner's DualSense on the TV's USB still did nothing:
  the TV app's log had no controller at all, and the engine's log no
  gamepad. Older LG TVs pass only LG's listed USB controllers to apps
  (moonlight-tv wiki, Gamepad Setup); DualSense needs an adapter, webOS 5+
  Bluetooth, or pairing with the PC (docs/GUIDE.md). Install again on a
  paired TV no longer fails: the pairing loop reads the TV app's log
  (`core.TVAppStarted`: "Screen: starting" or "ready" after the last
  launch). Save a report adds `TV.WebOSVersion` (/etc/starfish-release).

- **2.3.0** The owner's TV list of 16 Nov, part 1 (console mode). In TV
  mode the window no longer hides for a game: `tvGameStarting` (after
  every launch: `startFound`, `runSwap`) emits `tvGame` "starting" and
  sends the window to the bottom of the z-order (`win.SendToBack`) after
  1.5 s; `tick`'s hide calls `tvGameUp` (`procOf`, `win.GameWindow`,
  `ForceForeground`, "playing"); back always runs `tvBack` in TV mode.
  `win.AllowAnyForeground` before launches. The taskbar (`Shell_TrayWnd`,
  the TV's while it streams) is hidden in TV mode (`tvTaskbar`, a
  `taskbar-hidden` mark shown again at the next start). Apollo gets
  `isolated_virtual_display_option`. The holder ends only on an explicit
  "off" or 20 s without answers (`tvPatience`). UI: `tvLaunch`,
  `drawLaunch`, `tvReturn`, `#bp-launch` (`--launch-*` tokens); B while
  starting comes back; a game Lootnest can't follow returns on focus.
  Still to do from that list: the Home button overlay, Spotify, YouTube
  and Stremio as apps, and the TV's sound (the engine sends it; the TV
  app's audio backend is the suspect).

- **2.4.0** The owner's TV list, part 2: the Home button. `win/guide.go`:
  `XInputGuideDown` (xinput1_4 ordinal 100, `XInputGetStateEx`, Guide
  0x0400; the TV's controller arrives as a ViGEm Xbox pad) polled every
  60 ms by `watchGuide` in TV mode; PlayStation pads by raw input
  (`WatchPadRawInput`, RIDEV_INPUTSINK on gamepad and joystick usages;
  `WM_INPUT` in `hook`, `RawPSButton`, `PSButtonIn` by report: DualSense
  USB byte 10, Bluetooth 0x31 byte 11, DS4 USB and simple reports byte
  7, DS4 Bluetooth 0x11 byte 10; nothing is opened). `tvHome`: over a
  game (`hidFor`) toggles the menu (`tvMenuOn`, `tvBack`, event `tvMenu`)
  or resumes (`tvResume` -> `tvGameUp`); with none, event `tvHome` opens
  `bpMenu`. `ps_tvResume`, `ps_tvStay`, `ps_tvCloseGame`
  (`CloseProcessWindows(procOf)`). UI `tvMenu`/`drawTVMenu`/`tvMenuGo`,
  `#tv-menu`; the Gamepad API's button 16 is left to Go in TV mode;
  `BP.playing` makes Play resume the running game.

- **2.5.0** The owner's TV list, part 3: apps. `core/mediaapps.go`
  `FindMediaApps` (Spotify in %APPDATA% or the Store's WindowsApps alias,
  Stremio 4/5 paths, YouTube at youtube.com/tv in Edge or Chrome
  `--app` with a TV user agent and its own `--user-data-dir` under
  DataDir); `apiMediaApps`, `apiLaunchApp` register a `foundRun` keyed
  `app:<id>` (by process name, or by the started pid for the browser:
  `win.StartTracked`, `ProcessAlive`, `WindowOfPID`, `CloseWindowsOfPID`)
  and call `tvGameStarting`, so the 2.3.0/2.4.0 machinery (behind, Home
  menu, back on close) covers them; an app is done when it has no
  visible window (Spotify hides to the tray); apps log no sessions. UI
  `bpAppsRow`, `bpApp`, `.bp-app` (`--app-*` colours), Phosphor
  `spotify-logo`, `youtube-logo`, `popcorn`. Lootnest for TV 1.3.1
  (c-dinca/lootnest-tv-webos#7) notes its picture and sound backends and
  whether a stream's sound opened, for the owner's silent TV.

- **2.5.1** YouTube in 2.5.0 opened youtube.com/tv in the desktop layout,
  and the fresh Edge profile signed itself in and synced the user's
  extensions, whose pages opened in browser windows on the TV.
  `youTubeArgs`: `--kiosk` (Edge adds `--edge-kiosk-type=fullscreen`),
  extensions, sync and implicit sign-in off, a PS4 Leanback user agent
  and `UserAgentClientHint` off (client hints still said Windows).
  Signing in is YouTube TV's own code at youtube.com/activate.
  The mouse was trapped on the TV: Apollo's isolated display is primary
  at (0,0) with the PC's displays moved so they touch it only at its
  top-left corner (`rearrangeVirtualDisplayForLowerRight` in vdw.cpp), and
  Windows puts the cursor on the primary. `core.CursorKeep` (only for
  that isolated layout: no shared edge, other displays' bounds outside
  the TV's) and `keepCursor` in app/tv.go hold it on the PC's displays
  (`win.ClipCursor`, set again whenever it's let go; a cursor on the TV
  goes to the largest PC display) and let it go when TV mode ends.
- **2.5.2** Leaving TV mode from the PC (the owner: "easy, not a
  struggle"; the main taskbar and tray are on the TV and hidden there).
  `win/tvbar.go`: a GDI bar on its own thread (`ShowTVBar`, `HideTVBar`;
  top-most, tool window, never activates, `MA_NOACTIVATE`) at the bottom
  right of `core.PCDisplay` (the largest display not overlapping the
  TV's), with a Racing Red Leave TV mode key (`tvLeaveFromPC`).
  `keepCursor` shows and places it every second. Ctrl+Shift+F11
  (`hotkeyTV`, held from `enterTV` to `leaveTV`, `tvKeyHeld`) and the
  tray's Leave TV mode do the same. `tvPlaying` names the game in the bar.

- **2.6.0, Big Picture rebuilt, part 1** (the owner, 16 Nov: a
  console-quality TV interface; they approved the direction in
  `design/mocks/bp-tv.html`: bar on top, OK opens a game's page, the
  achievements row, Save points key, "files are loaded" line and trailers
  behind, off by default; and the webOS connection screen last. They
  asked for one sign for the menu, not two, and not to forget
  animations). `25-bigpicture.js` keeps state, TV mode, the pad and
  sounds; `26-bp-screens.js` has the screens and the focus: zones
  (`data-zone`; a row, `data-grid`, `data-col`) of `[data-bp]` things,
  `bpMove` (left/right in a zone, up/down between zones via `bpInto`,
  which goes back to what the zone had in focus), `BP.mem` per screen
  (`bpScreenKey`), `bpPlace` (rows and the screen slide with transforms:
  `.bp-track`, `.bp-scroll`, `BP.sx`; rows above fade), `bpAmbient` (the
  art behind and Home's hero, 180 ms after the focus rests), `bpHints`,
  `bpMorph` (a redraw changes only what differs; `data-live` keeps a
  slide's style, `data-own` its children), `bpEnter` (how a new screen
  arrives), `bpBack` (top layer, page, up to the bar, Home, then
  `bpMenu(true, "leave")`). The menu is an overlay (`.bp-over`,
  `bpMenuHTML`). Sizes are `--u` tokens (`--bp-*` in system.css); no
  blur in Big Picture. Harness shots `tv-*`.
- **2.6.0, Big Picture rebuilt, part 2**: a game's page has its
  achievements as a row (`bpAchCards`: unlocked first, rarest leading;
  cards take the focus but do nothing) and Save points (`BP.points`,
  screen key `points:<key>`, `bpPointsScreen`, `bpPoint` over
  `api.makeSavePoint`/`restoreSavePoint`). `bpAsk({title, body, ok,
  run})` asks before anything that changes files, on the menu's sheet
  with Cancel in focus (`bpAnswer`, `bpOver()` is menu or question).
  Column zones (`data-col`) slide like grids. Toasts over Big Picture
  sit under the bar at its scale. `bpOpen` starts the pad loop when a
  pad is already there. The launch screen and `#tv-menu` use `--u`.
- **2.6.0, Big Picture rebuilt, part 3**: Settings for a remote
  (`BP.settings`, screen `settings`, from the menu's sheet:
  `bpSettingsParts` is the one list of parts and rows, each row with
  `on` or `value`, `say` and `run`; `bpSet` saves a setting; column
  zones name the column beside them with `data-left`/`data-right`; the
  part in focus shows its rows; `.bp-say` says the row in focus). Only
  settings Lootnest has. Button signs (`BP.signs`, pref `bpSigns`,
  read first by `glyph`), since a pad on the TV reaches the PC as an
  Xbox one. Achievements: a total, cards with cover and `.bp-meter`,
  OK opens the game at `ach:0` (`BP.want` takes the focus when what it
  names is drawn). `bpMorph` takes both child lists before it walks
  them (a replaced child leaves the new tree).
- **2.6.0, Big Picture rebuilt, part 4**: trailers behind the game
  in focus (`BP.trailers`, pref `bpTrailers`, off by default, a row in
  Big Picture's Settings; `bpTrailer` from `bpAmbient`: only with Game
  details on, 2 s after the focus rests, asking for that game's details
  and waiting for them, a muted looping `video.bp-bd-v` under the
  scrims through the 2.2.0 `loadHls`; `bpTrailerStop` when the focus
  moves, a game starts, the screen is deep or dim, or Big Picture
  shuts). The scrim behind the picker and dialogs over Big Picture is
  plain, not blurred.
- **2.6.0, Lootnest for TV 1.3.2** (c-dinca/lootnest-tv-webos#8)
  is pinned: its screen was already in Lootnest's look at TV sizes
  (`ui->dpi` is the height over 3, so a unit is 2.25 px at 1080p), so
  only the keys changed: the one in focus is Ice Grey with dark words
  and grows a little (`key_tr`, 160 ms), the main key keeps red and the
  ring. Not seen on a TV. The Big Picture rebuild is done; next is what
  the owner finds on the real TV. The owner asked for 2.6.0 as its
  release (16 Nov), and for the older releases to be pruned: kept are
  the 1.x milestones, 2.0.0, 2.2.11, 2.5.2 and the latest.
- **2.7.0, Lootnest as a console's screen** (the owner, 16 Nov: "I
  want Lootnest there all the time, as on a PlayStation", in TV mode and
  Big Picture; the Home button as an overlay; go back to the game or
  close it without leaving Lootnest). `app/console.go`: `console()` is
  TV mode or Big Picture; `tvGameStarting` snapshots the windows on show
  and starts `curtain`, which finds the game's window (its process when
  known, else `core.NewGameWindow`: the front-most new window on
  Lootnest's display taking a quarter of it), puts it in front and
  Lootnest right behind it (`front`, `win.PlaceBehind`), keeps looking
  30 s for a launcher followed by the game, falls back to `SendToBack`
  after 25 s with nothing, and for a game with no known process brings
  Lootnest back when the window is gone (`watchGone`). `conGame` and
  `conWin` are the game behind Lootnest; `show()` clears `hidFor` but
  not them (the 2.4.0 Home menu forgot the game once it opened, so
  Resume and Close did nothing). The tick stands behind games in Big
  Picture as in TV mode and always comes back. `tvHome` works in both:
  `gameShot` (a JPEG of the game, taken before `raise`) goes with
  `tvMenu`. `startGuide`/`watchGuide` and raw input run in Big Picture
  too. Page: the launch screen and `#tv-menu` in Big Picture
  (`.tv-menu-shot`), `bpBehind`, Playing on the tile, Go back to the
  game. `win/console.go`: `TopWindows`, `ScreenOf`, `PlaceBehind`,
  `IsWindow`, `Foreground`. Not seen on a PC: docs/TESTING.md 33.
- **2.7.0, a PlayStation controller as an Xbox one** (the owner,
  16 Nov 2026: "do the DualSense to Xbox translation with ViGEmBus",
  reversing the roadmap's "Lootnest does not translate controllers";
  ROADMAP Controllers phase 6). `Settings.PadAsXbox`, off by default.
  `core/padmap.go`: `PSPadPath` (a HID path's Sony vendor and product,
  USB or Bluetooth), `PSToXbox(dualsense, bluetooth, report)` (which
  report layout follows from how the pad is connected, never from the
  length: Windows pads every report to the pad's longest), `PadState`,
  `XUSBReport`. `win/vigem.go`: ViGEmBus's interface by its GUID
  (`CM_Get_Device_Interface_List`), `NewViGEmPad` (version handshake,
  `VIGEM_PLUGIN_TARGET` with the first free serial), `Submit`, `Close`;
  a pad goes when the handle that plugged it closes. `win/pshid.go`:
  `HIDPaths`, `OpenPad` (read only, shared), `Read`, `Close` with
  `CancelIoEx`; nothing is ever sent to the pad. `app/padbridge.go`:
  `padTick` from `achTick` (on while a followed game runs, or Lootnest
  stands behind one; never in safe mode), `runPad`, `padSay` (logged
  once per change, shown in Settings). The PS button isn't passed on:
  it is Lootnest's Home, read by raw input. `LaunchOptions.KeepPad`
  leaves the pad alone for one game. Games started by Steam itself
  aren't followed, so Steam Input stays in charge of those. Not seen on
  a PC: docs/TESTING.md 34. Not done: rumble, the lightbar, hiding the
  real pad.

- **2.8.0, YouTube as Lootnest's own window** (the owner, 16 Nov:
  "pagina de youtube inca arata asa", the desktop site in Edge's kiosk).
  Edge didn't take `--user-agent` on the owner's PC, and its kiosk window
  is InPrivate, so a sign-in wouldn't have lasted. `lootnest.exe --app
  youtube --on <hwnd>` (`app/webapp.go` `RunWebApp`, before the single
  instance check in main.go) is a program of its own: a WebView2 window
  full screen on Lootnest's monitor (`win.FullscreenLike`), data in
  `apps\youtube`, the PS4 Leanback name set through
  `WebViewOptions.UserAgent` (a patch to third_party/go-webview2:
  `PutUserAgent`). `core.WebApp` (`Host`), `WebAppByID`; `MediaApp.Own`,
  `MediaPaths.Self`. Leanback reads keys, never the Gamepad API: `padKeys`
  (added to every page) reads every pad as one and asks `ln_key`, bound
  only for youtube.com pages, which presses a real key (`win.TapKey`,
  SendInput) only while that window is in front (`WebApp.Owns`: https and the host exactly; a Copilot Autofix merged with #254 had made it refuse every page). Checked here against
  the real youtube.com/tv in Chromium with a stubbed pad; not seen on a
  PC: docs/TESTING.md 35. The old Edge profile `youtube-tv` is left.

- **2.8.0, Big Picture design details** (the owner, 16 Nov: the
  Home menu jumped between its buttons; a nicer overlay with
  achievements; achievements "as on PS" with the game's on the right;
  animations when choosing). `padLoop` has one pace for every screen
  (`step`: first press, 400 ms, then every 120 ms, never twice in
  100 ms; `pad.held` used to be kept only outside the Home menu, so a
  held direction moved every frame) and `padDir(g, was)` has hysteresis
  (on at 0.6, off at 0.35). The Home screen over a game (`#tv-menu`,
  `tvMenuItems`, `tvMenuInfo`, `tvMenuLight`, `tvMenuMove`, `tvMenuGo`,
  `tvMenuPoint`): choices by key (`M.at`), drawn once per change of
  content (`put`), the light (`.tv-menu-ind`) slides; Make a save point
  (`api.makeSavePoint`, answered by `savePointMade`), Achievements
  (`BP.detail` with `BP.want = "ach:0"`); `BP.since` is when and what
  started. Its keyboard is handled first in 26's keydown; `renderBP`
  doesn't take the focus while it's up. Achievements in Big Picture is
  two lists (`.bp-achs`): games (`data-agame`, `BP.achKey`, set on
  focus) and `bpAchPanel` (`bpAchRow`, `bpAchRing`, `bpTierLine`, shared
  with the Home screen); a zone marked `data-vtrack` slides inside its
  `.bp-clip` frame (`bpPlace`), and Up from a column's top goes to the
  bar. Choosing: `bpPick` (`.is-picked`, `.is-picking`, `BP.picking`
  holds input for 460 ms), `bpPush` (the art comes closer),
  `data-leave` on the screen that goes, a sheen on the cover in focus
  (`--bp-sheen`), the seat popping in the bar. Harness shots
  `tv-achievements-in`, `tv-achievements-loading`. Not seen on a TV:
  docs/TESTING.md 36.

- **2.8.0, Lootnest for TV 1.4.0** (c-dinca/lootnest-tv-webos#9; the
  owner, 16 Nov: less latency and stutter, a perfect picture). Moonlight's
  default stream was 720p at 10 Mbps; `lootnest_tune_picture` sets
  1920x1080 at 60 and 25 Mbps with HEVC once (`lootnest_tuned` in the
  TV app's settings file), so Settings on the TV still wins afterwards.
  `lootnest_stream_begin/second/end` sum the stats overlay's counts into
  one "Stream: ended after ..." line in `lootnest-tv.log` (frames
  arrived and shown, lost on the network and in how many seconds, a
  frame's time on the PC and the TV, round trip), which Save a report
  carries. Apollo's settings were read (`src/config.cpp` at v0.4.6:
  nvenc preset 1, `fec_percentage` 20, `limit_framerate` on) and left
  alone: they're already latency first, so the next change waits for a
  report's numbers. A TV that has the app needs Install again. Spotify
  has no TV web interface (open.spotify.com serves the same page to TV
  and console user agents; no /tv), so a console-like Spotify would be a
  Big Picture screen over the Web API with the owner's client id: asked,
  not decided.

- **2.9.0, Spotify's web player as an app** (the owner, 16 Nov:
  Spotify "as on consoles"; a client id of their own was a no, since
  Spotify's development mode allows 5 listed users and extended access
  is for registered businesses). `core.WebApp` has `Nav` ("keys":
  YouTube's TV interface reads arrows itself; "spatial": `padSpatial`
  in app/webapp.go) and `Width` (the site's pixels across the screen;
  `WebAppZoom`, applied with `WebViewOptions.Zoom` through
  `ICoreWebView2Controller.PutZoomFactor`, a patch that passes the
  double as Go's amd64 calls do). `padSpatial` keeps its own focus
  (`data-ln-focus`, a white frame): the nearest thing that way among
  what's visible and not covered, inside the open dialog or menu when
  there is one, staying in the list being scrolled while it has more;
  A is a real Enter for links and buttons (so sound may start) and a
  click for the rest; B is Escape for what's on top, else
  `history.back()`; X, LB, RB are the media keys. `ln_key` takes at
  most 30 presses a second. Apps: `youtube`, `spotify` (web),
  `spotify-app` (the program, when installed), `stremio`. Tried against
  the real open.spotify.com signed out in Chromium with a stubbed pad;
  protected playback (Widevine) in WebView2 and the zoom call are not
  seen on a PC: docs/TESTING.md 37. Next: the controller as a mouse
  over any app, from the Home menu (the owner: "when it's needed and
  only then").

- **2.9.0, the controller as a mouse** (the owner, 16 Nov: "over
  everything, when it's needed and only then"; it was M1's open item
  for store launchers). `core/padmouse.go`: `Pointer.Step` (dead zone
  0.24, a curve that's fine for small pushes, 1.15 screen heights a
  second, leftovers kept; the right stick is the wheel). `win/
  padmouse.go`: `XInputRead` (every XInput pad as one), `MouseMove`,
  `MouseButton`, `MouseWheel` (SendInput). `app/pointer.go`:
  `pointerFor` is the key of the one program it's on for; `pointerLoop`
  (8 ms) acts only while that program is `conGame`, the Home menu is
  shut and Lootnest's window isn't in front, lets the button go
  otherwise, and ends 3 s after the program is gone. A left button,
  X right, B Escape, Y `osk.exe`, D-pad arrows with repeat. On by
  itself from `apiLaunchApp` for apps that aren't Lootnest's own
  windows; the Home menu's Controller as a mouse (`ps_tvPointer`,
  `tvMenu`'s `pointer`). A site window is told with `wmPointer`
  (WM_APP+7) and stops passing `ln_key` on. XInput only: a PlayStation
  pad counts when it reaches the PC as an Xbox one. Not seen on a PC,
  the on-screen keyboard least of all (it may refuse made-up clicks):
  docs/TESTING.md 38.

- **2.10.0, Discord as an app** (the owner, 16 Nov: Discord on the
  TV, joining a server's voice, hearing the others, sharing the
  screen). `core.WebApp` `discord` (discord.com/app, `Nav` spatial,
  `Mic`). `WebViewOptions.Microphone` (go-webview2 patch) allows the
  microphone for the pages it says yes to (`WebApp.Owns`), through
  `Chromium.PermissionFilter`; the library's `PermissionRequested` read
  the permission's kind by value, so no request had ever been answered:
  fixed, which also makes the main window's clipboard permission real.
  Icon `discord-logo` (tools/icons.py), `--app-discord`. Sound comes
  out where the PC's does (Apollo's sink, so the TV). A stream carries
  no microphone back (Apollo 0.4.6 and moonlight-common-c have none),
  so talking needs a microphone on the PC; a headset in a controller
  works only when that controller is on the PC by USB. Tried only on
  discord.com's sign-in page (the QR code, the frame moving); voice and
  screen sharing in WebView2 are not seen: docs/TESTING.md 39.

- **2.10.0, Lootnest for TV 1.6.0** (c-dinca/lootnest-tv-webos#10 and
  #11; the owner, 16 Nov: the TV is on Wi-Fi, the PC wired, the delay
  makes games hard to play; then: "720p is no way to play", and the
  settings should be theirs to change). 1.4.0's 25 Mbps was too much
  for Wi-Fi; 1.5.0's Automatic could drop to 720p and was replaced
  before it was pinned. The TV app's home screen has two keys. Size
  (`lootnest_size_next`: 1080p, 1440p, 4K, 720p, up to
  `video_cap.maxWidth/maxHeight`) is only the person's. Quality
  (`lootnest_picture_t`: Automatic, Light, Balanced, Sharp, As in
  Settings; `picture_kbps` by size and frame rate, 1080p at 60 is 8,
  14, 25 Mbps). Automatic (`lootnest_picture_review`, once per stream
  of 45 s or more, from the 1.4.0 counts) has five steps of bits at the
  chosen size: down at 1% of frames lost, loss in 20% of the seconds,
  or the TV showing a tenth fewer than arrived (two steps at 5%); up
  only after a clean 5 minutes and not for three clean streams after a
  step down (`lootnest_hold`); it never changes the size.
  `lootnest_picture_sync`: a size or frame rate changed in the TV
  app's Settings keeps the quality, a bitrate moved there makes it As
  in Settings. `lootnest_tuned` 3 moves everyone to Automatic once.
  Under Ready: `lootnest_picture_line` and `lootnest_last_stream_line`.
  The protocol can't change bitrate in a stream, so a step is for the
  next one. Not seen on a TV. Still to come from the owner's numbers:
  whether the host's `fec_percentage` or the TV's decoder is the limit;
  setting a TV's picture from Lootnest on the PC. The TV's picture
  mode (Game) and the 5 GHz band are the owner's to set.

- **2.10.1, the TV app kept up to date** (the owner, 16 Nov, after
  2.10.0: "I don't see the Size key"). Install again opened the whole
  Add an LG TV drawer and asked for Developer Mode's passphrase, so
  their TV had most likely stayed on an older app; and Size and Quality
  only show on the TV app's own screen, which a stream covers at once.
  `installTV` with an empty passphrase uses the key kept in link.json
  (`againLGTV` in 82-link.js; a key that no longer opens the TV says
  to use the passphrase). `TV.TVAppVersion` reads the app's
  appinfo.json. `updateTVApps` (from `keepDevModes` at start and every
  hour, and 2 minutes after `leaveTV`): for each saved TV with an older
  app than `TVAppPinned`, installs it, unless this PC is streaming or
  the app is what's on the TV's screen (`AppRunning`); it never opens
  the app; `linkTVUpdated` toasts. `LGStatus.Latest`; the row's key is
  Update to x when they differ. Lootnest for TV 1.6.1
  (c-dinca/lootnest-tv-webos#12) has its version in a corner of its
  home screen. Published without asking, as a TV fix (the owner, 15
  Nov). Not seen on a TV: docs/TESTING.md 40. Still open: setting a
  TV's Size and Quality from Lootnest (Big Picture's Settings), since
  the TV app's screen is hidden while a stream runs.

- **2.10.2, a slow stream** (the owner, 16 Nov: Size and Quality show
  now, "but I still have latency"; their report had no stream numbers,
  the TV being off when it was saved). Read first: Apollo 0.4.6 on
  their PC encodes with AMD's AMF (RX 6700 XT; `amd_usage`
  ultralowlatency, `amd_rc` vbr_latency are its defaults, left alone),
  and the TV app's decoders already run without buffering (ss4s: NDL
  before SMP by weight, SMP with `lowDelayMode`); Moonlight TV's FAQ
  puts the TV's own share at about 30 ms at 1080p. So what's left is
  the network, the TV's picture processing and how the PC draws.
  `core/streamcheck.go`: `LastStream` reads the TV app's "Stream: ended
  after ..." line, `StreamVerdict` says which part held it back (short,
  network, tv, pc, clean). `noteStreams` (app/lgtv.go, 8 s after
  `leaveTV`) logs it with the TV's Wi-Fi link (`TV.WiFi`,
  /proc/net/wireless) and keeps it in link.json (`savedTV.Stream`);
  `LGStatus.Stream*`; `lastStreams` in 82-link.js. `Settings.TVQuick`
  (off by default) is Apollo's `double_refreshrate`:
  `ApolloSettings(pc, quick)`, `engineSettings`, `bringEngine`,
  `engineSettingsSoon` from `apiSaveSettings` (applied when no stream
  runs, else at the next start); `ApolloSettingsDiffer` reads a missing
  key as "disabled". Not seen on a PC or TV: docs/TESTING.md 41. The
  TV's picture mode and Wi-Fi band are the owner's to set.

- **2.10.3, the owner's first real stream numbers** (16 Nov, Onimusha,
  411 s of H264 1080p60): 5 ms there and back, 4.4 ms a frame on the
  PC, 51.7 frames a second sent, 1.24% lost but all of it in 4 seconds.
  So the network and the PC aren't the delay. Their TV decodes through
  "NetCast Legacy" (ss4s `lgnc`: an LG before webOS 5, H.264 only, up
  to 40 Mbps, no low-delay control; the app can't measure or shorten
  what happens after it hands a frame over), so what's left is inside
  the TV and in the game's frame rate. `StreamVerdict` no longer calls
  a few bad seconds a slow network (`spread`: lost in over 5 seconds
  and 2% of them; else a hiccup said beside the verdict) and has
  `StreamGame` (under 92% of the stream's rate sent). `WebOSVersion`
  falls back to `systemproperty/getSystemInfo` (`tvSystemInfoIn`:
  model, sdkVersion, firmware), since /etc/starfish-release wasn't
  readable on their TV; the model decides whether NDL could be used.
  The controller's disconnects in that log were the owner's own.
  The TV app's Automatic has the same hiccup flaw
  (`lootnest_picture_review` steps down at 1% lost however few the
  seconds): fix it with the next TV app version. Also seen in that
  log, not looked into: the pad bridge's first `Submit` after plugging
  in says "The requested resource is in use", and the engine refused
  connections on localhost:47990 when the report was saved.

- **2.10.4, one Spotify and Stremio as its program** (the owner,
  16 Nov). `spotify-app` (Spotify on this PC) is no longer listed: one
  Spotify in Apps, the web one. Stremio's web version (web.stremio.com
  with `padSpatial`, which drives it well) was built and taken out the
  same day: the owner wants only the Stremio program, which has the
  streaming server the web version needs. So Stremio shows only when
  the program is installed (Stremio 5's folder added to the paths
  looked in), with the controller as a mouse over it. They also said
  no, for now, to dropping Developer Mode through store apps: an
  unmodified LG takes apps from outside its store only in Developer
  Mode.

- **2.11.0, Stremio with a TV interface** (the owner, 16 Nov,
  after 2.10.4: "can't we make Stremio with the TV interface, and the
  app?"). One Stremio tile: `core.WebApp` `stremio` (web.stremio.com,
  `padSpatial`) in Lootnest's own window, with the installed program's
  streaming server behind it. `core.StremioServer` finds
  `stremio-runtime.exe` and `server.js` in `stremioDirs` (Stremio 5's
  installer puts it in `%LOCALAPPDATA%\Programs\Stremio`, exe
  `stremio-shell-ng.exe`; read in stremio-shell-ng's `setup/Stremio.iss`
  and `server.rs`, which runs exactly that pair with no window);
  `startStremioServer` (app/stremio.go, in the `--app stremio` process)
  starts it when nothing answers on 127.0.0.1:11470 and kills it when
  the window closes; a Stremio that's already running is left alone.
  `FindMediaApps` lists only site windows now, so no app turns the
  controller into a mouse by itself; the Home menu's switch remains.
  Tried against the real site in Chromium with a stubbed pad; the
  server start and playback in WebView2 aren't seen: docs/TESTING.md 42.

- **2.11.0, script plugins** (the owner, 17 Nov 2026: "we change what
  a plugin is"; and: the owner makes the decisions, don't make them for
  them). A plugin that provides `script` has one .js run inside Lootnest
  by goja (pure Go; a time limit per call, no memory limit) and reaches
  only the `lootnest` object (`core/pluginscript.go`: `StartScript`,
  `Call`, `on`, `schedule`, `storage`, `net.fetch` behind `guardedDial`,
  which refuses this PC and its network, `library.games`,
  `updates.offer`, `notify`; only JSON crosses). `permissions` (net,
  library, updates, notify) and `secrets` (a key and the one host it goes
  to) are shown in words (`PluginAsks`) and allowed before it runs
  (`GrantPlugin`, `plugins.json` `granted`); a version asking for
  something else is off again. `app/pluginscript.go`: `syncScripts` (from
  `pluginsChanged` and at start, never in safe mode), keys in
  `plugin-secrets.json` (DPAPI), `ps_pluginGrant`, `ps_pluginView`,
  `ps_pluginAction`, events `pluginPages`, `pluginView`,
  `pluginChanged`, `pluginActed`. A plugin's pages are Settings
  categories drawn from data by `79-pluginscript.js` (`viewPluginPage`,
  `ppRow`, forms as dialog `ppform`; every word escaped, no HTML from a
  plugin). goja has no locales (`toLocaleString` fails) and no `URL`,
  `fetch` or timers. Write a plugin's .js in ASCII with `\uXXXX`
  escapes: a raw U+2028 in a regular expression parses in goja and not
  in Node.
  Offers: `core.LocalVersion`, `win.FileVersion`; `app.scriptGames`
  gives scripts the version of games no store updates;
  `scriptsLibraryChanged` after a scan; `ps_pluginOffers`,
  `ps_pluginOfferAction` (the script's `offer` handler), event
  `pluginOffers`; `S.plugOffers`, `offersSection` (From plugins) on the
  Updates page, `offerBanner` on a game's page.
  Downloads and the update: `core/dlqueue.go` (`DownloadQueue`: one at a
  time, Pause, Resume, Remove, `Job`, downloads.json, `GuardedClient`;
  http only with a SHA-256), `core/fullupdate.go` (`FindBuild`,
  `PlanFullUpdate`, `RunFullUpdate`: backup, swap, restore, verify, kept,
  with `state.json` written before each step; `RecoverFullUpdate`,
  `RollbackFullUpdate`, `FinishFullUpdate`; the old folder is renamed to
  `<folder>.lootnest-old`, never changed; a build that failed goes
  aside, never deleted; `keepBlocked` never carries a store's files or a
  stand-in tool's). `app/fullupdate.go`: a script's
  `lootnest.updates.run("start", ...)` makes a job
  (`fullupdates\<hash>\job.json`, `log.txt`), staging in
  `<folder>.lootnest-new`; `jobUnpack` finds a whole build, else hands a
  patch or an installer to the 1.37/1.40 update drawer; `jobApply`
  (after the user's yes in dialog `ppjob`), `copySaves` (a version in the
  safety net's store, so it shows under Copies on this PC), `watchKept`,
  `recoverJobs` at start, `errUpdating`. `ps_pluginJobs`,
  `ps_pluginJobPlan`, `ps_pluginJobOp`, `ps_pluginJobLog`, event
  `pluginJobs`; `jobCard`, `jobSteps`, `ppJobHTML` in
  79-pluginscript.js. (`updates.run("magnet")` was removed after 2.11.2.) `core/fullupdate_integration_test.go`
  runs the whole way with a small inline script (`updateScript`).
  Harness `?plugins=1` (installed, not allowed), `?sources=1|empty|error`
  (a made-up plugin, Update notices, with a Settings page and offers)
  and `&srcupd=download|paused|ready|hand|apply|kept|failed`.
  docs/TESTING.md 43. Released as 2.11.0, unseen on a PC. It shipped with
  a first plugin, Sources and updates, in `plugins/`; 2.11.1 attached it
  to releases. Both were taken out after 2.11.2 (see Unreleased below).
- **2.11.2, switches that take a click** (the owner, 17 Nov, on
  2.11.1: "I see nothing asking me anything"). A `.switch` outside a
  `<label>` (plugin rows, a plugin page's rows, mods) never got a mouse
  click: its drawing (`span`) lay over the hidden input. `.switch input`
  has `z-index: 1` now. Harness checks that call `el.click()` on the
  input miss this; click with the mouse at the switch's box. A script
  plugin's row has **Allow and turn on** (`plugin-allow`) until granted,
  and `pluginInstalled` asks (`allowPlugin`) right after install.
- **2.12.0, updates that take only what changed** (the owner, 17
  Nov: a 100 GB game shouldn't need the whole download or 100 GB free).
  A job whose files are one .zip, .7z or .rar (parts too) at https
  addresses gets `Mode` "delta" (`deltaOK`, `runDelta` in
  `app/deltaupdate.go`, phase "fetch"). `core/remotefile.go`:
  `RemoteFile` (Range reads in 4 MB blocks as an `io.ReaderAt`,
  `ErrNoRanges`), `RemoteVolumes` (an `fs.FS` of a .rar's parts, each
  one stream that reconnects where it dropped). `core/delta.go`:
  `StageDelta` (`rootFromNames`, CRC-32 and size against the game's
  files, only differing files into `<folder>.lootnest-new/files`,
  `plan.json`, room checked as it's known, staged files kept across
  tries), `DeltaExtras`, `DeltaRemovals` (only when `Full`: the archive
  has the game's exe), `ApplyDelta` (moves on the game's drive, old
  files to `<folder>.lootnest-old/<rel>`, `delta.json` saved before each
  move), `RevertDelta` (Roll back, failures and `deltaRecover` at
  start; safe twice). A site without Range gets the whole download
  first (`delta-local`, `deltaFromDisk`, the archive deleted after).
  UI: `DELTA_STEPS`, `isDelta`, the "fetch" case in `jobParts`, the
  delta review in `ppJobHTML` (`JobPlan.Delta/Full/Changed/Added/Same/
  Removed/Bytes`). Harness `?sources=1&srcupd=compare|fetch|
  fetchpaused|deltaready|deltakept`. Tests in `core/delta_test.go`
  (zip, solid and non-solid 7z fixtures, .rar written by a small RAR5
  writer, over httptest with a dropped connection, a crash at every
  move). Not seen on a PC: docs/TESTING.md 43, items 8 and 9.
- **2.12.0, no plugins in Lootnest** (the owner, 17 Nov 2026, after
  asking whether the sources plugin was legal: "scoate-l"). The Sources
  and updates plugin, its plan, its tests, `./build.sh plugins` and the
  release zips are gone; `core.HydraOrigins` and `ScriptGame.Origin`
  (Hydra's download record of a game, given to scripts) too. Hydra as a
  launcher (1.33) stays. The plugin's code is in git history (tag
  v2.11.2, `plugins/sources-updates`). Don't add it back, or anything
  that names a source format or helps find pirated copies, without the
  owner asking.
- **2.13.0, Lootnest Link M4a, Add an Android TV** (the owner, 18
  Nov 2026: Moonlight on Android TV, Google TV and Fire TV, installed by
  Lootnest over ADB rather than through Google Play, in a fork they asked
  for, c-dinca/lootnest-tv-android). `core/adb.go` speaks the host side
  of ADB over TCP 5555 itself (no adb.exe): CNXN, AUTH with Lootnest's
  own RSA key (`NewADBKey`, PKCS#1 v1.5 over the 20-byte token as SHA-1;
  `ADBPublicKey` is adbd's RSAPublicKey struct, checked byte for byte
  against Google's `adb keygen` in `testdata/adb`, public halves only), one stream at a time
  (`stream`, WRTE waits for OKAY), `Shell`, `Push` (sync SEND/DATA/DONE),
  `Device`. The TV asks once to allow the computer (`ErrADBNotAllowed`,
  `adbTimeToAllow` 90 s). `core/androidtv.go`: `AndroidAppPinned`
  (Lootnest for TV 1.0.0, c-dinca/lootnest-tv-android's first release,
  signed in CI with the owner's key from two secrets;
  `AndroidAppReady` gates the button), `InstallAndroidApp` (push to /data/local/tmp, `pm install -r`,
  `InstallAnswer`), `LaunchAndroidApp` (`am start -S -n
  com.lootnest.tv/com.limelight.LootnestActivity --es host --es pairPin`,
  both checked before the shell), `AndroidAppLog` (logcat tag
  `LootnestTV`), `AndroidAppStarted`, `FindAndroidTVs` (port 5555 on the
  /24s). `app/androidtv.go`: `installAndroidTV` mirrors `installTV`
  (public network check, connect, install when the version differs, PIN
  to the TV and `ap.Pair` every 2 s for 75 s, `AllowTV`), `link.json`
  `androids` and `adbKey` (DPAPI); Save a report adds each one's app
  log. UI in 82-link.js (`openAndroidTV`, `viewAndroidTV`, drawer `atv`,
  rows, step `adbconnect`); harness `?link=set&atv=1` and
  `&atvrun=allow|public|done`. The app: Moonlight Android with
  applicationId `com.lootnest.tv`, `LootnestActivity` as the launcher
  (adds the PC by `host`, pairs with `pairPin`, opens the host app
  named Lootnest, wakes the PC for 90 s, then Play, Settings, PCs, Quit),
  Lootnest's icons and banner, fonts in assets; `VERSION` drives
  versionName/Code; release.yml signs with four secrets
  (`LOOTNEST_KEYSTORE_B64` and the passwords) and attaches the APK and
  SHA256SUMS. Not done: Android 11+ Wireless debugging (TLS pairing
  with a code), Google Play. Not seen on a TV: docs/TESTING.md 44.
- **2.12.0, the clean-up after the plugin** (the owner, 17 Nov
  2026: "fa tot"): `updates.run("magnet")` is gone (a plugin's only
  update call is "start", over http(s) with a SHA-256), the
  `Steam\dodi` achievements path is gone (named after a repack group),
  PRIVACY.md and PLUGINS.md no longer describe the plugin or a launcher's
  download record, and the 2.11.0 and 2.11.1 release notes on GitHub are
  set from CHANGELOG.md by `.github/workflows/release-notes.yml` (run
  by hand, `tags`, `dry_run`), so they describe only script plugins. Git history still holds the
  plugin; rewriting it is the owner's call (GitHub keeps pull request
  refs, so only GitHub Support can purge them).

- **Unreleased, Linux L0** (the owner, 19 Nov 2026: Lootnest for the
  Linux desktop in general, a WebKitGTK window through cgo; the plan is
  docs/ROADMAP.md, section 9, L0 to L5). `internal/plat` sits between
  the app and the system (`tools/platgen` writes `win_windows.go`, the
  forwards, and `stub_other.go`; a `Set<Name>` for a win variable app
  assigns, `SetToastFailed`); `internal/app` lost its build tags and
  compiles for Linux, with `view` (window.go) for the window,
  `openWindow` in window_windows.go/window_other.go, `RunWebApp` in
  webapp_windows.go/webapp_other.go, `noConsole` in proc_*.go. CI's
  build job regenerates plat, vets the app for Linux and runs its tests.
  Nothing changes on Windows. Next: L1, the Linux path model in core
  (prefixes), when the owner says.
- **Unreleased, Linux L1a, Wine prefixes and Steam on Linux** (the
  owner, 19 Nov 2026: "go ahead with L1"). `core/prefix.go`: `Prefix`
  (Root holds drive_c, User is `steamuser` for Proton), `IsPrefix`,
  `OpenPrefix` (the one user folder besides Public), `ProtonPrefix`
  (`<lib>/steamapps/compatdata/<appid>/pfx`), `Env.InPrefix` (%APPDATA%
  and the other Windows folders inside the prefix, the Linux home and the
  prefix's own folders protected), `Env.For(prefix)` (a stored prefix
  read with this Env's tokens). `Game.Prefix` and `FoundGame.Prefix`; the
  engine reads a game's paths and the tools' achievement folders in its
  prefix (`Engine.envOf`), and the app does where it reads a game's
  saves (`a.envFor`: gameFromFound, Set saves, cloud sources and
  liveDirs, `savesOf`); the other `a.env` calls wait for L3. `expand`
  reads `\` as `/` on Linux and finds a prefix path's spelling on disk
  (`matchCase`). `core/linux.go`: `XDGDataHome`, `XDGConfigHome`,
  `XDGUserDir` (user-dirs.dirs), `FindSteamLinux` (native, ~/.steam
  links, Flatpak, Snap), `NewLinuxEnv` (`OS` "linux", %HOME% as a token,
  `NeverInside` for /usr, /etc...). `ScanSteam` gives a Proton game its
  prefix and finds its saves there; `SuggestPaths` looks in the XDG
  folders for native games; the manifest keeps Linux locations apart
  (`DBGame.Linux`, `<xdgData>`, `<xdgConfig>`), used when `OS` is
  "linux". Tests in `core/prefix_test.go` (not run on Windows).
- **Unreleased, Linux L1b, the other Linux launchers**
  (`core/linuxlaunchers.go`, `Env.ScanLinuxLaunchers`, native and
  Flatpak folders, prefixes stored with tokens). Formats read in each
  launcher's source. Heroic (`HeroicGames`, ~/.config/heroic):
  legendary's installed.json, `gog_store/installed.json` with titles from
  `store_cache/gog_library.json` or the game's `goggame-<id>.info`,
  nile's installed.json and library.json, `sideload_apps/library.json`;
  a Windows game's prefix from `GamesConfig/<app>.json` (winePrefix),
  config.json's defaults, then `~/Games/Heroic/Prefixes/default/<title>`,
  with Proton's `pfx` inside (`prefixAt`); `heroic://launch?appName=&runner=`.
  Lutris (`LutrisGames`): `pga.db` through `core/sqlite.go`, a read-only
  SQLite 3 reader (`sqliteTable`: sqlite_master, table b-trees, overflow
  pages, columns from the CREATE TABLE, the INTEGER PRIMARY KEY as rowid;
  no -wal), installed `wine` and `linux` games only, `games/<configpath>.yml`
  (config folder, else data) via `core/yamlmini.go` (`readYAMLMap`, the
  block YAML PyYAML writes); `lutris:rungameid/<id>`. Bottles
  (`BottlesGames`): each bottle is the prefix, `bottle.yml`
  External_Programs (not `removed`), Windows paths through drive_c or
  dosdevices (`bottlesPath`). itch on Linux finds native programs
  (`MainLinuxExe`: the largest ELF, else .x86_64 or .sh). Sources
  `heroic`, `lutris`, `bottles` (labels and filters in 35-library-home.js).
  Fixture `testdata/lutris/pga.db` comes from Python's SQLite
  (`make.py` there); `FuzzSQLite`.
- **Unreleased, Linux L2a, the window** (the owner, 19 Nov 2026: "go
  ahead with L2"). `internal/webkit` (cgo, GTK 3 and WebKitGTK 4.1): the
  page is loaded from a string under `lootnest://app/` (`BaseURI`, a
  scheme registered to answer nothing and marked secure); `decide-policy`
  refuses every navigation but that address and every new window;
  permissions are denied (clipboard allowed); the page's own context menus
  only (the inspector with `LOOTNEST_DEBUG`); calls reach Go only through
  the `lootnest` message handler, with the address of the last committed
  load (a refused navigation leaves `webkit_web_view_get_uri` on the
  refused one) checked against `BaseURI` and `AllowMessage`; `Bind` works
  as go-webview2's (JSON arguments, a promise). GTK must stay on the main
  thread: `main_linux.go` locks it, and the test does its window work in
  `TestMain` (tests run on other goroutines; GTK aborted now and then
  before). App: `view` is met by both windows; `platWindow`, `present`,
  `closeWindow` per system; `onClose` quits (the tray is L2b);
  `systemEnv` (env_windows.go, env_linux.go: `core.NewLinuxEnv` with the
  Settings Steam account), `systemDataDir` (~/.local/share/lootnest);
  one Lootnest per user over the abstract socket `@lootnest-<uid>`
  (`AcquireInstance`, `SendToRunning`, `serveInstance`: show, safe,
  launch only); `cmd/lootnest/main_linux.go`; `AppState.Platform`, and on
  Linux no updates of Lootnest (releases are Windows installers) and no
  banner asking. plat on Linux: the XDG folders, `ShellOpen` (xdg-open),
  `RevealInExplorer` (FileManager1, else the folder), `StartDetached`,
  `DiskSpace`, `CPUName`, `TotalRAM`, `ProcessNames` (/proc, Wine's
  `C:\...\x.exe` command lines too). CI builds `lootnest-linux`, runs
  the webkit test under xvfb-run and starts the app once, failing when
  its log lacks "Interface loaded".
- **Unreleased, Linux L2b, the desktop** (`internal/desk`, cgo over GTK
  and libsecret, called from plat's Linux files). `OnMain` runs GTK work
  on the main thread from any goroutine (the main goroutine's thread id
  is noted at package init; others go through `g_idle_add` and wait).
  Dialogs (`MessageBox` maps MB_ flags and answers with Windows' IDs,
  `desk.Message`), the desktop's own picker (`Browse`,
  GtkFileChooserNative, so the portal where there is one), notifications
  over D-Bus (`org.freedesktop.Notifications`; a note over a game,
  `ShowToast`, is a notification on Linux), the clipboard (text and PNG),
  the window's icon (`SetParent`, `desk/icon.png` at 128 px, written to
  `~/.local/share/lootnest/icons`). Keys (`Protect`/`Unprotect`,
  secrets_linux.go): AES-256-GCM, "LNX1" + nonce + sealed, with a key kept
  in the Secret Service (`lootnest-secrets`), else in `secret.key` (0600)
  when there's no keyring; a key in that file keeps opening what it made.
  The tray: Ayatana AppIndicator loaded with dlopen (no build or run
  dependency; no tray without it), menu-only, made again when its lines
  change (`trayReady`, every 2 s); `trayMenu` is now `trayItems` and
  `trayChoose`, shared with Windows. Closing goes to the tray when Close
  to tray is on and the tray shows (`onClose`). The Trash
  (trash_linux.go): the freedesktop spec, the home Trash or the drive's
  own (`.Trash/$uid` when sticky, else `.Trash-$uid`), the .trashinfo
  written first with O_EXCL and taken back when the move fails.
  Autostart: `~/.config/autostart/lootnest.desktop`. CI installs
  libsecret-1-dev and dbus and runs the desk test in `dbus-run-session`.
  Next: L3, games and the swap on Linux, when the owner says.
- **2.14.0, moves for Hydra and offline games** (the owner, 19 Nov
  2026: no way to move Red Dead Redemption 2, from Hydra and kept
  offline). `MoveGamePlan` takes source `hydra` (Hydra can't move a game;
  the user points Hydra at the new program, and the entry stays added with
  its Steam id until then); a game kept offline moves, then one elevated
  `FirewallMoveScript` takes away the old tag's rules and blocks the new
  folder's programs under the new key (`offlineMoved`, `runFirewall`).
  Harness `?kept=1`. The owner asked for 2.14.0 as their release
  (19 Nov); the Linux work above ships in no release yet (releases
  carry only the Windows installer and exe).
- **2.14.0, achievements marked by hand, then taken out** (the owner,
  19 Nov 2026, asked for variant A for RDR2 outside Steam, then right
  after 2.14.0 was published: "that thing feels like cheating, don't
  give the option to mark them as done"). 2.14.0 shipped it; the next
  version removes it (the code is in git history, #286, reverted).
  `hand-achievements.json` files left on a PC are no longer read and
  never deleted. Don't offer marking achievements by hand again; a game
  shows achievements only from files on the PC.
- **2.15.0, Motion Full** (the owner, 19 Nov 2026: the achievement
  card's tilt worked on a friend's PC, not theirs). Windows' Animation
  effects off sets `prefers-reduced-motion`, which cut nearly all motion.
  `Settings.Motion` "" is Like Windows (default), "full" keeps every
  animation: each rule in a `prefers-reduced-motion` block is scoped to
  `:root:not(.motion-full)`, `reduced()` ignores Windows with Full, and
  Motion's row says when Windows has them off. New reduce-only rules need
  the same `:root:not(.motion-full)` scope.
- **2.15.0, moving a game, its look** (the owner, 19 Nov 2026: time
  left, nicer and animated, minimise it and keep using Lootnest).
  `moveBody`, `moveNow`, `moveTick` (an averaged speed), `paintMove`
  (numbers changed in place so the bar glides) and the corner card
  `movePill` (#mv-pill, actions `move-mini`, `move-show`) in
  72-storage.js, shared by a folder game's move and Game Mover's;
  `moveSays` gives progress events the name and both folders. Harness
  `?move=hold` keeps a move going at about 160 MB/s.
- **2.15.0, achievements that come by themselves** (the owner, 19
  Nov 2026, after marking by hand was taken out: "sync automatically, not
  a button"; both of the two ways offered). `achAfterPlay`
  (app/achauto.go) reads a game's achievements again 8 s after it closes
  (`tick`'s `downFound` for Library games, `finishSession` for games with
  profiles) and sends them with `after`; the page asks again when it's
  opened after a minute (`achAsked` is a Map of times, `ACH_AGAIN`).
  `Settings.SteamAccountAch` (off by default, needs the Steam Web API
  key): when nothing on the PC has a game's achievements,
  `accountAchievements` asks `ISteamUserStats/GetPlayerAchievements`
  (`core.SteamID64` from the userdata folder, `ParsePlayerAchievements`,
  `ErrSteamPrivate` when Game details aren't public), kept an hour in
  `achievements/account-<sid>-<appid>.json`, made stale after a play
  (`accountStale`), pictures from the key's schema (`CompleteList`).
  The owner asked for 2.15.0 (19 Nov) once these were all in.
- **2.15.1, why no achievements** (the owner, 19 Nov 2026, on
  2.15.0: "I turned it on and it still doesn't work" for RDR2, which they
  had moved). `MergeAdded` lost a moved Hydra game's Steam id: the scan
  found the new folder under the same key and its details won; now an
  added entry's `SteamID` and launcher `Source` fill a plain folder
  scan's. `achWhy` (app/achauto.go) goes with an `achievements` event
  whose set is null: the app id, Steam signed in, key, setting, and what
  Steam last said (`accountSaid`: ok, private from a 403, none, error).
  `achSteamSteps` in 62-game-blocks.js lists the steps with lamps;
  Settings, Online's row says the same. Harness `?why=start|private|none|set`.
  Released as 2.15.1 at the owner's ask (19 Nov).
- **2.16.0, story progress** (the owner, 19 Nov 2026: "how much of
  the story I've done, beside Play"; of three ways offered they chose the
  one from achievements, not an estimate by time or reading saves).
  `core/story.go`: `StoryOf` reads achievements whose words mark a
  numbered chapter, act, part, episode, mission or stage ("Complete
  Chapter 3.", "Leave Act II - ...", "Clear Mission 03.", roman, words,
  "the first chapter"; a rest only "of/in ..." or flavour after a dash,
  never a challenge: `storyNotStep`), a prologue, an epilogue, and the end
  ("Finish the game on ...", "Achieved the "X" ending": any one unlocked
  finishes it). The kind with the most numbers wins; under two numbers
  only a reached end shows (Finished). Percent is by chapter number to the
  last step (the end, else the epilogue); a later step done marks those
  before. Tuned on 36 games' real lists (RDR2, BG3, Celeste, DMC5,
  Spider-Man 2, Tekken 8...). `AchievementSet.Story`, per player too.
  `Settings.StoryAch` (off by default): `storyWords` reads
  `IPlayerService/GetGameAchievements` (no key; hidden ones' words, which
  GetSchemaForGame leaves out), kept 30 days as `<appid>.public.json`;
  `StoryWords` fills only a copy. UI `storyChip` in 62-game-blocks.js
  (ring and line in the play bar; on a page it opens the Achievements
  tab), `.gpage-story`. English words only. Not in Big Picture yet.
- **2.16.0, games in folders of the user's own** (the owner, 19
  Nov 2026: Tekken 8 didn't show on a friend's PC, in the Library or Big
  Picture). `ScanFolders` took a folder of games with its own name
  (`D:\Jocuri PC`) for one game, named after the folder (`LooksLikeGame`
  walks two levels). `gamesFolder` tells them apart (nothing of a game at
  its top, and two game-like folders, or one whose name isn't in
  `gameInsides`); such a folder is looked into once more. `Settings.GameDirs`
  (Library, Sources, Folders with games; `cleanGameDirs`, at most 20,
  browse kind `gameDir`, `saveGameDirs` then a scan) are roots, set for
  `gameFolders` at each scan's start (`setUserGameDirs`), not containers:
  a chosen folder that is a game itself is added as one.
- **2.16.0, Installed in the sidebar** (the owner, 19 Nov 2026:
  every installed game on the left beside the collections).
  `renderSideGames`: after the folders, groups `installed` (every item
  but `owned`), `owned` (Not installed), then Not in a collection; the old
  `all` group is gone. Harness shot `side-installed`.
- **2.16.0, Big Picture by controller** (the owner, 19 Nov 2026: no
  CS2, Valorant or League of Legends on the TV, "with all games").
  `core/padplay.go`: `padOfCommon` reads appinfo.vdf's `controller_support`
  or store categories (28 full, 18 partial; categories but neither is
  none; the binary reader now keeps int32s as digits), `SteamControllers`,
  `PadGuess` (Riot's games but 2XKO, a few by name: `noPadGames`),
  `DetailsPad` (kept store details; "" Controller is none only with
  `ReqsRead`). `app/padplay.go` `padSupport` per Library key: the user's
  Edit details Controller, appinfo, kept details, the guess; cached 30 min,
  `padStale` after a scan or a change; `ps_padSupport` (event
  `padSupport`, `S.pads`), `ps_setPadPlay` (writes the Controller edit).
  UI: `padOf`, `bpGames`, `bpLeftOut` and chip `BP_KBM` (Keyboard and
  mouse) in 26-bp-screens.js; `BP.padOnly` (pref `bpPadOnly`, on), a row in
  Big Picture's Settings, Controller; right-click `pad-play` (Leave out of
  / Show in Big Picture). Nothing goes online for it. Achievements in Big
  Picture still lists every game with some. Harness `ctx-pad`,
  `tv-library-kbm`.
  The owner asked for 2.16.0 once these four were in (19 Nov).
- **2.16.1, every game on Big Picture's Home** (the owner, 19 Nov
  2026: on a friend's PC, TEKKEN 8 was in the Library on the PC but not on
  the TV). Home's first row with nothing played was `all.slice(0, 12)`,
  and Your games `slice(0, 40)`, so most of a Library was only in the
  Library tab. Now Your games holds every game up to `BP_HOME_MAX` (120),
  then `bpAllTile` (opens the Library tab). Harness `?fresh=1` (no recent
  plays), shots `tv-fresh`, `tv-fresh-400`.
  Released as 2.16.1 at the owner's ask (19 Nov).
- **2.16.2, TV fixes of 19 Nov** (the owner: the TV app's sound
  late, its home screen stuck after too many presses left or right,
  words over its keys, no way out of TV mode on the PC when it stuck,
  no keyboard for Stremio's search, Stremio's "streaming not working").
  Emergency key Ctrl+Shift+Q (`app/emergency.go`,
  `win.WatchEmergencyKey`: a thread hotkey on its own thread while
  `consoleArmed`; cursor, taskbar, TV bar, `win.KillPID` on the holder,
  then leaveTV and quit, `os.Exit(3)` if the window doesn't close).
  `padSpatial` has a keyboard (`textField`, `kb` in a shadow root;
  A types, X deletes, Y space, B closes, Start presses Enter).
  `WebApp.Loopback` (Stremio): `WebAppBrowserArgs` turn off Chromium's
  Local Network Access in that window's process (seen in Chromium 157:
  a public page's fetch to 127.0.0.1 fails without them);
  `startStremioServer` logs through `appLog`. Lootnest for TV 1.6.2
  (c-dinca/lootnest-tv-webos#13): `key_nav` stops at the ends and the
  screen leaves the focus group while it has keys (it was a trap), the
  home screen in three bands, `in_time()` in `session_audio.c` trims
  sound more than 60 ms ahead of its low (direct submit never dropped
  any). Android 1.0.1 (c-dinca/lootnest-tv-android#2): the same layout
  and focus (`row()`), the granted AudioTrack logged. docs/TESTING.md 46.
  Both pinned; released as 2.16.2 without asking, as a TV fix (15 Nov).
- **2.17.0, the FPS counter** (the owner, 20 Nov 2026: on and off
  from the overlay or the keyboard; variant 1 for the permission,
  Ctrl+Shift+F10; and asked why not over exclusive fullscreen: only by
  hooking the game's Present, which is injection, so no; since the
  owner's decision of the same day, story games may get that, see Story
  games and online games). `win/etw.go`: a
  real-time ETW session of Lootnest's own ("Lootnest frame counter",
  stopped and started again if a crashed Lootnest left it) with
  EVENT_FILTER_TYPE_EVENT_ID filters: DXGI Present_Start 42 and
  PresentMultiplaneOverlay_Start 55, D3D9 Present_Start 1, DxgKrnl
  Present_Info 184; times are QPC ticks (`ClientContext` 1), FlushTimer
  1 s. `core.FrameCounter` keeps two seconds per process and source and
  takes the largest source, never the sum (a DX11 frame is heard by DXGI
  and the kernel); the rate is the second before the newest frame heard,
  since ETW delivers about a second late. `win/fpsbox.go`: a GDI,
  top-most, click-through window in `Settings.FPSCorner` of the front
  window's monitor. `app/fps.go`: `fpsTick` (from `hotkeyTick`) holds
  Ctrl+Shift+F10 while a game runs (or `conGame`) and runs `fpsLoop`
  only while on and playing; the number is the front process's, hidden
  when that's Lootnest or draws nothing for 3 s ("--" for 3 s after
  turning on). `Settings.FPSCounter` and `FPSGroupAdded` are the app's
  (apiSaveSettings keeps them; the page calls `ps_fpsSet`). ETW needs an
  administrator or Performance Log Users (S-1-5-32-559):
  `core.PerfLogUsersScript` adds the account's SID once through one
  elevated PowerShell (`ps_fpsAllow`), effective from the next sign-in;
  `win.FramesAllowed` checks the token. `win.ExclusiveFullscreen`
  (SHQueryUserNotificationState) is logged and shown in Settings. UI:
  `fpsRows` in 70-settings.js, the Home menu's `fps` item, Big Picture's
  Settings row, event `fps`; harness `?fps=ask|signout|exclusive|on`.
  CI's windows job checks the structure layouts and starts and stops the
  session; frames over a real game are docs/TESTING.md 47.
- **2.17.0, mods from Nexus Mods, step 1** (the owner, 20 Nov
  2026: "a mod manager in the Mods tab where you search and install
  mods without leaving Lootnest"; Nexus chosen; then: "a small page
  embedded in Lootnest for the download, not the whole browser").
  `core/nexus.go`: Nexus's public GraphQL API (v2), no key or account
  (`NexusFindGame` by domain, name, then words; `NexusSearch`,
  `NexusModInfo`, `NexusText` for BBCode; adult, hidden and removed mods
  left out; `SafeNexusImage`; CSP and `AllowedPageRequest` allow
  staticdelivery and images.nexusmods.com). `Settings.NexusMods`, off by
  default. `68-nexus.js`: the Mods tab's Your mods / Nexus Mods switch
  (`modsPane`, `S.modsPane`), the grid (`nxCard`), drawer `nexusmod`,
  Not this game? (drawer `nexuspick`, `nexus.json` `picked`). Download
  (`ps_nexusGet`) records a `NexusWait` in `nexus.json` and starts
  `lootnest.exe --nexus <domain>/<mod>/<file>` (`RunNexusWindow`): a
  WebView2 window with its own sign-in in `apps\nexus`; a go-webview2
  patch adds NavigationStarting and NewWindowRequested
  (`edge/navigation_lootnest.go`, `WebViewOptions.Navigate`/`NewWindow`;
  `core.NexusWindowGo` says show, browser, nxm or refuse). The user
  presses Manual download there; `nexusWatch` looks in Downloads every
  3 s (`core.NexusArrivals`: "-<mod id>-" and the upload time in Nexus's
  file name, else the one archive since when one ask is open), closes
  the window, brings Lootnest up and opens the modplan drawer
  (`ps_nexusAdd`, `core.NameMod` names it and keeps `Mod.Nexus`). Nexus
  files aren't offered as game updates (`downloadsLook`). Lootnest still
  downloads nothing itself. Step 2 (Sign in with Nexus, `nxm://`, mod
  updates) waits on the owner registering Lootnest with Nexus
  (docs/ROADMAP.md, Mods 5b). Harness `?mods=1&nexus=1|none|error`,
  `&nxwait=hold`; shots `nexus*`. Not seen on a PC: docs/TESTING.md 48.
- **2.17.0, mods from Nexus Mods, the download in Lootnest** (the
  owner, 20 Nov 2026: Nexus won't register Lootnest, so a workaround;
  they chose these two). The window takes the download: a go-webview2
  patch adds DownloadStarting (`edge/download_lootnest.go`,
  `ICoreWebView2_4`, `WebViewOptions.Download`; the vtables were checked
  against WebView2.h of SDK 1.0.2903.40, NavigationStarting's and
  NewWindowRequested's too). A file from Nexus's servers goes into
  `nexus-downloads` (`core.NexusIncomingDir`, `NexusIncomingPath`) with
  a note beside it (`<file>.lootnest.json`: mod, file, size;
  `WriteNexusNote`, `NexusNoteOf`), the download list stays hidden, and
  a note in the page says Lootnest has it (`nexusNotes`, `__lnNote`).
  `nexusLook` (every 1.5 s) looks there first; `NexusArrivals` trusts a
  note over the name; `NexusIncoming` gives the progress (event
  `nexusProgress`, `S.nxProg`, `nxWaitLine`/`nxProgress` change the row
  in place). A runtime without `ICoreWebView2_4` leaves files in
  Downloads as before. `nxm://` (`core.NexusNXM`) sends the window to
  the file's page with `ln=manual`, which says to press Manual download.
  `PruneNexusIncoming` at start lets go of files over 30 days (added
  ones are copied into the game's mods). Won't do: pressing the site's
  buttons for the user, or users' personal API keys. Harness
  `&nxprog=1`, shots `nexus-progress*`. docs/TESTING.md 48, items 3 to 5
  and 9 to 10. The owner asked for 2.17.0 once this was in (20 Nov).

- **2.18.0, Go back to a save point, all the way** (the owner, 20
  Nov 2026: Go back to it "doesn't really do anything"; the first part of
  the save engine's rework). `apiRestoreSavePoint(key, id, play)`: a
  running game is asked to close (`closeForPoint`:
  `CloseProcessWindows`, never forced; `pointCloseWait` 30 s, then for a
  game with profiles its session's end, `pointSessionWait`, and 3 s for
  Steam Cloud's upload), events `pointStep` (closing, copying), the Auto
  point, then `core.RestoreSavePoint`, which now ends in `MakeNewest`:
  every restored file's time moves forward by one amount so the newest is
  now and their order stays (a game's Continue and Steam Cloud take
  them). With play, `savePointMade` carries `play` and `player`, and the
  page starts the game as its Play key does (`playAfterPoint`; Big
  Picture's `bpPlay`). `goBackAsk` words the question by whether the game
  runs (`pointGameRuns`). docs/TESTING.md 49.
- **2.18.0, Save now** (the owner, 20 Nov 2026: "mergem pe save
  now"; the second part of the save engine's rework, without injection).
  `LaunchOptions.QuickSave` is the game's own quick save key ("F1" to
  "F12", the user's choice; `apiSaveLaunchOptions` keeps it, since the
  launch drawer doesn't send it); `core/quicksave.go`: `QuickSaveVK`,
  `CleanQuickSave`, `QuickSaveSuggestion` (a table by Steam app id and by
  name: Bethesda, CD Projekt Red, Larian, Arkane, Owlcat, Obsidian F5;
  Valve's Source games and Black Mesa F6; only offered, never pressed
  until chosen), `StampSaves`, `WaitForGameSave` (a change, then still for
  `settle`). `app/quicksave.go` `saveInGame`: only while the game runs,
  with a key, no `AntiCheat(gameDir)`; the game's window to the front
  (`tvResume` when it's behind Lootnest's Home menu, else
  `ForceForeground`), waits for Ctrl/Shift/Alt to be let go
  (`win.ModifiersDown`), presses only into the game in front
  (`ForegroundIs`), `win.TapGameKey` (scan code, held 80 ms), then waits
  12 s for the save; no save, no point. Used by `apiMakeSavePoint` (Save
  now, the Home menu's point) and `quickPoint` (Ctrl+Shift+F5);
  `SavePoint.InGame`; step `saving`. UI: `quickSaveRow`, `savesFirst`,
  `qs-use`, `#pt-qs`, `ps_setQuickSave`; `savePoints` carries
  `quickSave`, `suggest`, `antiCheat`. Harness `?pointrun=1&qs=F5`.
  docs/TESTING.md 50.
- **2.18.0, every save the game makes** (the owner, 20 Nov 2026:
  "hai sa incepem", the third part of the save engine's rework).
  `app/gamepoints.go`: `gamePointsTick` from `achTick` (every ~6 s) over
  `runningForSaves` (achTick's games plus Steam Library games without
  profiles, by process); per running game `gameSaves` (the target, `seen`
  stamps from the start or the last point, a candidate change); a change
  still for `gamePointsStill` 5 s becomes a point (`keepGameSave`,
  `SavePoint.Game`, "Saved by the game, hh:mm", event `savePointMade`
  with `game`, no toast or sound). `makePoint` calls `savesSeen`, so a
  point made by any path is never kept twice. Saves over 512 MB aren't
  followed (logged). `core.PruneGamePoints` keeps the newest
  `gamePointsKeep` 30 game points; `RenameSavePoint` clears `Game`.
  `Settings.GamePointsOff`, never in safe mode. UI: `shownPoints`,
  `moreGameSaves` (`GAME_SHOWN` 5, `pt-all-game`), `.pt-auto` line.
  Harness `?gamesaves=0`. docs/TESTING.md 51. Next in the rework, when
  the owner says: the injected save for story games (a plan first).

- **2.18.0, the controller in TV mode while the PC is in use**
  (the owner, 20 Nov 2026: on the TV only the PS button worked). The
  page skipped every pad while `document.hasFocus()` was false, WebView2
  gives an unfocused page no pad input, and Chromium's `:focus` stops
  matching then too (checked in Chromium). `app/padfeed.go`: `padFeed`
  (started by `startGuide`) every 16 ms while `padOwned` (Lootnest's
  window is the front one on its screen, `TopWindows`) sends `padFeed`
  events on change: XInput (`XInputRead`) and PlayStation pads from raw
  input (`win.RawPSReport`, replacing RawPSButton: product, Bluetooth from
  the device path, the newest report; `notePSPad`, `core.PSToXbox`),
  merged by `core.MergePads`, sticks by `core.PadAxis`. The page
  (`padFeed`, `padAway`, `padAwayCSS` in 25-bigpicture.js) uses the feed
  while it lacks focus or sees no pad, runs `padLoop` at once per event,
  marks the focused element `.pf` under `html.pad-away`, and copies every
  `:focus` rule to `.pf` (`:focus-within` to `:has(.pf)`) once. Tried in
  a headful Chromium with another window in front; not on a PC:
  docs/TESTING.md 52.

- **2.18.0, Big Picture towards a console's feel, part 1** (the
  owner, 20 Nov 2026: "like PS5", music, animations, better settings, mods,
  Triangle for Settings, not too different from the PC). `27-bp-music.js`:
  a piece made with WebAudio as it plays (`MUSIC_CHORDS`, `musicChord`,
  `musicBell`, a convolver hall and an echo; nothing recorded or
  downloaded), `musicWanted` (Big Picture on, `BP.music`, visible, Sounds
  on, no `BP.launch`/`BP.playing`/`BP.tvMenu`, no session playing, no
  trailer), `musicTick` every 500 ms fades in or out and schedules a bar
  ahead; prefs `bpMusic`, `bpMusicVol`. `audioCtx()` in 25-bigpicture.js
  is the one context. The bar has a Settings key (`tab:settings`, go
  `settings`, a gear beside the menu); Y (`bpY`) only takes the focus to
  it, and back to `BP.yFrom` when pressed again (the owner: "not open
  Settings, select the Settings key"); "Play as someone else" is only
  the key on a game's page now. `bpSettingsParts` have `icon`s; rows are `.bp-opt--set` with
  the say text inside (`.bp-opt-say`, opens on focus; the old `.bp-say`
  line is gone); new rows Music, Music volume, Put mods in as links, and
  a TV part (tvQuick, Leave TV mode). `bpModsScreen` (`BP.mods`, screen
  key `mods:<key>`, `bpMod` through `api.modToggle` for the seat) from a
  game page's Mods key (`modsTabOn`). Harness shots `tv-settings-y`,
  `tv-settings-tv`, `tv-mods`. Next, when the owner says: Home's layout
  and motion. docs/TESTING.md 54.
- **2.18.0, the PC's monitors in TV mode** (the owner, 20 Nov 2026,
  with two monitors: windows opened on the PC landed on the TV, the
  taskbar and Lootnest's bar were on one monitor only; they asked for
  both bars on both). `app/tvwindows.go` `tvWindows` (from `enterTV`,
  every 300 ms, only when `core.CursorKeep` says the TV's display is the
  isolated primary): a window judged once as it comes onto the TV
  (`core.OnTV`) moves to the PC display under the cursor
  (`core.PCDisplays`, `PlaceIn`, `win.WorkAreaOf`, `MoveWindowTo`, which
  maximises again) unless it's Lootnest's, `conWin`, within
  `tvLaunchGrace` of `conStart` (set by `tvGameStarting`), fills the TV,
  is a shell class, its process is the TV's (`tvSide`: own exe,
  `foundRuns`, the `conGame` process; `win.ProcessName`), or the TV's side
  had the keyboard at the last look. `tvTaskbar` is a method now and keeps
  `MMTaskbarEnabled`/`MMTaskbarMode` (`win.TaskbarSettings`,
  `SetTaskbarSettings`, a WM_SETTINGCHANGE "TraySettings" broadcast) in
  the `taskbar-hidden` mark as JSON, sets 1 and 0 (every app on every
  taskbar) and puts them back at leave, the emergency key and the next
  start; `keepCursor` hides the TV's taskbar again each second.
  `win.ShowTVBar` takes every PC display: one bar each, one thread, a
  thread timer (`barSync`, `barWin` per window). Not seen on a PC:
  docs/TESTING.md 53.

- **2.18.0, Big Picture's Home, part 2** (the owner, 20 Nov 2026:
  what's best on a PS5's home, "but not a copy"; researched, they agreed
  to the list in this order). Home's first block holds the hero, the
  first row and `bpCards(it)` for `BP.homeKey` (set in `bpAmbient` when
  the focus rests on a first-row cover; the cards rise in one by one):
  `hc:play` (Play for the seat, `bpPlay`), `hc:points`, `hc:ach` (the
  most common locked one), `hc:story` (`X.A.story`), `hc:mods`; go
  `hc-points`/`hc-ach`/`hc-mods` open the game one step in, and Back
  comes straight home (`BP.from` starts "hc:"). `bpPlace` moves the
  screen only as far as the cards need to clear `.bp-fade`, so the name
  stays. Collections follow, then Apps, the big collections, Your games.
  The first row's cover in focus lifts (`--bp-lift`) with a glow in
  `--seat` (set on `#bp` in `renderBP`) and the rest dims. Music:
  `musicShift` moves the piece -2 to +2 semitones by `BP.artKey`, at the
  next chord. Harness shots `tv-home-cards`, `tv-home-focus`.
  docs/TESTING.md 55.

- **2.18.0, Lootnest for TV 1.6.3** (c-dinca/lootnest-tv-webos#14;
  the owner, 20 Nov 2026: the TV app's Settings before connecting "looks
  broken", words over the keys). Moonlight TV's Settings used the TV's
  font (`FONT_FAMILY` Museo Sans on webOS, by fontconfig); `app_font_init`
  now loads the embedded Atkinson Hyperlegible Next first with the TV's
  font as its fallback, and Settings' parts are Ice Grey in focus. Seen
  in the desktop build under xvfb, not on a TV. The owner asked for
  2.18.0 with everything of 20 Nov in it.
- **2.19.0, the FPS counter without signing out** (the owner, 20
  Nov 2026: it didn't work, and "asking to sign in and out is awful";
  they chose, for games with an anti-cheat, the outside counter with no
  sign-out, and the in-game one, as Steam's, for the rest: the next PR).
  Performance Log Users is gone (`PerfLogUsersScript`,
  `Settings.FPSGroupAdded`). `core.FrameTaskScript` (one elevated
  PowerShell from **Allow**, `apiFPSAllow`) copies lootnest.exe to
  `%ProgramFiles%\Lootnest\Frame counter\lootnest-frames.exe` (only
  administrators can change it, so the task can't run anything else with
  those rights) and registers `\Lootnest\Frame counter` (the user's SID
  turned into its account name, Interactive, RunLevel Highest,
  `--frames`); `FrameUntaskScript` (**Remove**, `ps_fpsRemove`) takes
  both away. `app/framereader.go`: `startFrames` listens itself when
  `FramesAllowed`, else `frameTaskThere` (the program's stat, then
  `schtasks /query`, kept 30 s) and `startReader` (`schtasks /run`, then
  the pipe `core.FramePipeName(SID)` opened for 8 s);
  `RunFrameReader` (`--frames`, before anything in main.go) is the pipe's
  server (`win.ServeOutboundPipe`: outbound, first instance, no remote
  clients; `WaitPipeReader`), exits after 20 s with no reader and when a
  write fails, and writes `core.EncodeFrameReport` lines (protocol
  `FrameProtocol` 1; another is `ErrFrameReaderOld`, access "update")
  twice a second. DxgKrnl PresentHistory_Start (171) is a fourth source
  (`FrameHistory`), the event PresentMon always reads. CI's windows job
  registers and removes the task (`TestFrameTask`) and checks the pipe;
  the reader started by the task is docs/TESTING.md 47.
- **2.19.0, the FPS counter inside story games** (the owner, 20
  Nov 2026: "as Steam does, in the games we can, not those with an
  anti-cheat"; Denuvo allowed; "FPS and a frame's time"). The first thing
  Lootnest puts into a game's process (Story games and online games).
  `native/overlay/overlay.c` (C, MinGW-w64; MinHook 8af6b4a vendored in
  `third_party/minhook`, BSD 2-clause, in THIRD_PARTY_NOTICES): its
  thread opens `Local\LootnestOverlay-<pid>` (`ln_share`, 64 bytes;
  `core.Over*` offsets), finds Present through a throwaway swap chain or
  device of its own (DXGI Present 8 and Present1 22, D3D9 Present 17 and
  PresentEx 121, opengl32's wglSwapBuffers) for a minute as the game loads
  them, counts frames (a ring of QPC times; shown values change 4 times
  a second) and, while `on`, draws "N FPS N.N ms" from a 5x7 font as
  rectangles on an Obsidian panel: D3D11 `ClearView` on the back buffer
  (typeless formats made typed), D3D9 `Clear` with the back buffer as
  target (viewport, scissor and target restored), OpenGL scissored
  `glClear` on framebuffer 0 (state restored; GL functions found at run
  time). D3D12 frames are only counted (`api` 2). A per-thread depth
  counts nested Presents once. `tools/overlay/build.sh` builds it the
  same bytes every time (fixed object names, `-frandom-seed`,
  `--no-insert-timestamp`) into `internal/app/overlay/
  lootnest-overlay64.dll`, committed and embedded (`overlaydll_windows.go`);
  CI's build job runs `--check` in an `ubuntu:24.04` container (GCC 13.2,
  binutils 2.41.90; the runner's newer binutils makes other bytes). `core/overlay.go`: `InGameAllowed`
  (`AntiCheatDeep`, four levels, more names; `SteamOnline` from
  appinfo categories: 8 Valve Anti-Cheat, online categories without 2
  Single-player; `OnlineGame` by app id and name words; `AntiCheatRunning`
  by process). `win/inject.go`: `InjectDLL` (64-bit only,
  CreateRemoteThread on LoadLibraryW), `OverlayShare`. `app/overlay.go`:
  `overlayConsider` once per pid for the front window's game
  (`runningOfPID`), `overlayIn` (log says why not), the DLL written once
  per hash to `DataDir\overlay` (`OverlayDLLName`, older pruned),
  `overlayTick` (corner, `on`, drawn within 1.5 s means inside, else the
  DLL's count in Lootnest's box), `overlaysOff`. `fpsLoop` starts the
  outside frames only when a game needs them; a refusal no longer turns
  the counter off. `Settings.FPSInGameOff`. CI's windows job loads the
  DLL into its own process and draws with D3D11 (WARP), D3D9 and OpenGL
  (`TestOverlayDLLDraws`). Not on a real game: docs/TESTING.md 56.
- **2.19.0, Start with** (the owner, 21 Nov 2026: folder games
  whose launcher must start them, "the game stays the same, only what
  starts changes"). `LaunchOptions.StartWith` (an absolute .exe, checked
  in `CleanLaunchOptions`, moved by `Moved` with the folder, cleared for
  games not started by their exe or when it's the game's own exe);
  `startExe` starts it instead, in its folder unless Start in says
  otherwise, so both Library games and games with profiles get it.
  `Exe`, `ProcessName`, the icon, the key and the art are untouched, so
  the game is still followed by its own process; `launcherWait` gives
  `followFound` and the session's `deadline` 20 minutes. Desktop
  shortcuts target it, with the game's icon. UI: Start with in
  66-launch.js (`lo-exe`, `lo-exe-clear`, browse kind `startWith`), the
  Manage row says "Starts with X". docs/TESTING.md 57.
- **2.19.1, Start with, followed as either program** (the owner,
  21 Nov 2026, on 2.19.0: RDR2 with profiles stayed on "Loading ... and
  starting" while they played; their Start with program is the game
  itself under another name). `runNames(o, own)` (launchopts.go) is the
  game's own process name plus `exeProcName(StartWith)`; `liveName`
  picks the one running. Used by `tick` (sessions; `s.front`), the
  method `a.isRunning`/`a.gameNames`, `apiCloseGame` (closes each),
  `achRunning`, `procOf`, `pointTarget`, and found runs
  (`foundRun.names`, `proc` set to the live name each `foundTick`).
  `startedGrace`: 8 looks instead of 2 before a session with two names
  ends, for a launcher handing over. docs/TESTING.md 57, item 7.
  Released as 2.19.1 at the owner's ask (21 Nov).
- **2.20.0, a save brought as a mod** (the owner, 21 Nov 2026: a
  "Prologue Skip with Gold Medals" for RDR2 added as a mod went into the
  game's folder and did nothing; "make it detect a save by itself").
  `core/modsave.go`: `ModLooksLikeSave(m, category, saves)` (only
  `AtRoot` mods; Nexus's "Saved Games" category, else every file but
  pictures a save: `saveExts`, `saveNameRe` (SRDR3, SGTA5, savegame,
  slot...), or `saveShape` (digits as #) of a numbered file in the
  game's save folders, `SaveNames`). `addMod` takes Nexus's category
  (`apiNexusAdd`), removes such a mod and answers `modPlan` with `save`;
  `apiMods` marks old rows (`ModRow.Save`, `SaveFrom` = the kept
  `files` folder); `gameSaveNames` reads `pointTarget`'s locations. UI:
  the modplan drawer's save screen and the row's note, `saveKey`,
  `trySaveFrom`, action `mod-try-save` (games with profiles only).
  Try a save: `saveHome` puts one file beside the saves like it (same
  name, shape, then extension, three folders down). Harness
  `?mods=1&modsave=1`, shots `mods-save-*`. docs/TESTING.md 59.
  Released as 2.20.0 at the owner's ask (21 Nov).
- **2.20.1, game saves spaced out** (the owner, 21 Nov 2026: RDR2
  left a "Saved by the game" point every minute; "does it cost
  performance?"). `core.SpaceGamePoints(root, player, gap)`: after each
  game point, the newest but one goes when it's within `gamePointsGap`
  (5 min) of the one before it, so kept ones are a gap apart plus the
  latest; per player, only `Game` points. `makePoint` adds
  `playingFor(key, player)` (the session's `Since`, or the found run's
  `since`) to the playtime, which is only added when a game closes.
  docs/TESTING.md 51, item 6. Released as 2.20.1 at the owner's ask
  (21 Nov).

- **2.21.0, a game followed under another program's name** (the
  owner, 22 Nov 2026: Mihai's RDR2 sessions on the nights of 21 and 22
  left no time, points or Timeline). The log showed every RDR2 launch
  from 21 Nov 10:55 (2.19.0) on as "Launched ..." (most after "Started
  Launcher.exe before Red Dead Redemption 2"), and nothing after: no "FPS counter: listening", no
  save back, the session waiting until its 20 minutes ran out. The game
  ran under a name not in `runNames` (Launcher.exe is a Start before
  program, not Start with). `tick`: while a session launches or plays
  and its known names never ran (`Session.seenOwn`), a program with a
  window inside the game's folder is the game (`sessionFolders`,
  `inGameFolder`, `underDir`; `Session.alt`, logged once); a Start
  before/after program (`helperNames`) only when nothing else is there,
  and never again once the real game was followed (`altGame`), so a
  launcher left open doesn't hold the session. Only for games whose
  launch options have Start with or Start before (the owner, same day:
  "only if the launch options say so; other games, no checks"), and only
  until a known name runs (`sessionFolders`). The owner's RDR2: RDR2.exe
  opens only Rockstar's launcher and Launcher.exe is the game, so the set-up
  meant is Start with Launcher.exe (followed as the game since 2.19.1,
  RDR2.exe never started), not Launcher.exe under Before you play.
  `gameNames` includes
  `alt` (Close the game, save points). A launch that runs out logs what
  was looked for. Lootnest also had no "quit" line on 21 Nov 22:47 to
  01:30 and 22 Nov 01:30 to 12:04 (closed with the PC); sessions live
  only in memory, so one cut that way isn't counted (open item: keep
  them on disk). docs/TESTING.md 61. Released as 2.21.0 at the owner's
  ask (22 Nov).

- **2.18.1, Big Picture in the PC's glass** (the owner, 20 Nov 2026,
  on 2.18.0: Home's cards made no sense to them, and Big Picture should
  have the PC's "liquid glass", same theme). `bpCards` and the `hc-*`
  handlers are gone (Home is the hero, the first row, Collections...).
  `--bp-fill` is `--glass-strong` and every key, row, card and panel gets
  `--blur-glass` with the glass rim (the block at the end of app.css);
  `.bp-tabs` is one glass pill like the PC's sidebar; `.bp-over` and the
  overlay's scrim blur as the PC's dialogs (`--blur-scrim`), `.bp-dlg` is
  `--glass-overlay`; `--bp-scrim-deep` 0.7. This reverses 2.6.0's "no blur
  in Big Picture"; Glass off still makes it all solid. docs/TESTING.md 55.
  Released as 2.18.1 at the owner's ask (20 Nov).

- **2.19.0, mods from before profiles** (the owner, 21 Nov 2026:
  a mod added with profiles off was gone once they were on; "a button to
  port it"). Mods are kept by Library key (`core.ModsDir`), and turning
  profiles on makes the key `game:<id>`. `modsElsewhere` (app/mods.go)
  finds the game's other key with mods kept (`isFoundGame` both ways);
  `apiMods` sends `elsewhere` (`ModsElsewhere`: mods not kept here yet,
  `InFolder`); `apiModsBring` (`ps_modsBring`, run "bring") takes the old
  key's mods out of the folder by its record first when they're in (all
  of them come together), `core.CopyMod` (same id, hash-checked, temp
  folder then rename; the old copy stays), turns them on in the owner's
  set, then applies it when it took some out. While the other key's mods
  are in the folder, `applyModSet` refuses (`errModsElsewhere`),
  `modsForPlay` starts as is, and `modsOffFor` takes those out before an
  update. UI `modsAway` in 67-mods.js; More has **Open Lootnest's copies
  of the mods** (`ps_open("mods", key)`). Harness `&away=1|in`, shots
  `mods-away*`. docs/TESTING.md 58.
  The owner asked for 2.19.0 once this was in (21 Nov), with the FPS
  counter's two entries and Start with a launcher.

- **2.21.0, the window's bar inside the page** (the owner, 21 Nov
  2026: the Windows title bar "not like this, inside Lootnest, as Steam
  and Claude do", and the logo gone). The window keeps
  WS_OVERLAPPEDWINDOW (snap, shadow, Windows 11 corners and border) but no
  caption: `win.FrameSize` answers WM_NCCALCSIZE (the system's frame at
  the sides and bottom, none at the top; maximized, the top kept in by as
  much as the left), `RefreshFrame` in `OnCreate`; full screen (no
  WS_CAPTION) is left alone. `app/winframe.go`: `ps_win` ("drag", "top",
  "topleft", "topright" through `win.DragWindow`: ReleaseCapture, then
  WM_NCLBUTTONDOWN with the hit code; "min", "max", "close" through
  `WindowKey`, close is WM_CLOSE so Close to tray holds), `noteWinSize`
  sends `winState` {frame, max} on WM_SIZE. Page: `#winbar` (keys, the top
  edge `.wb-edge`), `16-winbar.js` (`paintWinbar`, `winbarStart`: a press
  on the top `--winbar-h` 32px or `.side-top` that isn't on `WB_OWN`
  drags; a second within 500 ms maximizes); `html.win-chrome` moves
  `.view`, drawers, Ctrl+K and unlock pops below it; hidden under Big
  Picture and on Linux. The sidebar's brand mark went with it and came
  back the next day, when the owner asked for that mark as the app icon
  too (`tools/mkicon.py` draws it; `assets/lootnest.ico`, `desk/icon.png`). Win11's snap
  layouts flyout on Maximize needs HTMAXBUTTON, so it isn't there
  (Win+Z is). docs/TESTING.md 60. Released as 2.21.0 at the owner's ask
  (22 Nov).

- **2.21.0, game page retouches** (the owner, 22 Nov 2026).
  Games without profiles get their time from the session log
  (`Store.LibraryTime`, sessions with no profile summed by key, cached by
  the log's size and time; `AppState.LibraryTime`, event `libraryTime`
  after `foundTick` logs one; `libTime` in `totalPlayed`); their page has
  no Profiles panel. `heroCard` (40-game-page.js) is Home's glass card as
  one row at the art's lower right (`.gp-hero-card`): time (`shownTime`:
  Steam's when there is some, else Lootnest's; `timeRow`), achievements,
  save points. Games with profiles: `pageWho(it)` is whose the page is
  (`S.gameWho[gid]`, else seat, files in, last player); `whoSwitch` beside
  Play opens `whoMenu`, and `who-pick` loads the files through the picker
  in "load" mode without starting the game; Play, `achFor`, `heroCard`
  and `pointsOf` (save points of that profile; `pt-everyone`) follow it.
  "Someone else" is "Play as someone else". Will it run no longer needs
  `it.appId`: the details event carries `appId` (`detailsAppID`).
  `Env.ToolAchFiles(dir, appID)` (local_save.txt beside steam_api(64).dll,
  configs.user.ini `local_save_path` and `saves_folder_name`), read by
  `PCUnlocksIn` and `AchievementFiles(appID, dir)`; not swapped per
  profile. Profiles, **Side by side** (`56-compare.js`, `S.playersTab`,
  `S.cmpWho`, `S.cmpShow`): a board with each line's leader crowned, then
  every game with each one's time, status and achievements (`S.allAch`).
  Community is rows (`reviewRow`, `.gp-rev`, `--rev-hours-w`, `rev-all`).
  docs/TESTING.md 62. Released as 2.21.0 at the owner's ask (22 Nov).

- **2.22.0, a run reads as its person's** (the owner, 22 Nov 2026:
  "Play as Skip Prologue" read like another profile; a run is a sub-save,
  the profile is still Cristian). 10-state.js: `pperson(id)` (the person's
  name), `whoseOf(id)` ("Cristian’s Skip Prologue"), `runName(p)`
  ("Cristian, Skip Prologue run"), `runTag(p)` (`.gp-run-tag`). Play keys
  say the person (game page, Big Picture); `whoSwitch` shows the person
  with the run's tag; `sessionText`, the docked head, the achievements and
  save points headings, the picker's swap screen use `whoseOf`/`pperson`;
  `heroCard` says "In the X run"; `whoMenu` and `playersPanel` list runs
  under their person (`grouped`, `.gp-person--run`). docs/TESTING.md 63.
- **2.22.0, the brand mark as the app icon** (the owner, 22 Nov 2026:
  the mark back beside the name in the sidebar, and the same ring over
  two lines in the taskbar and the tray instead of the gem in a nest).
  `tools/mkicon.py` draws `BRAND_MARK`'s shapes on the obsidian tile,
  heavier strokes at 16 to 48 px; `assets/lootnest.ico`, `lootnest.png`,
  `internal/desk/icon.png`. `mkicon.mark(size, fill)` is the mark on a
  clear ground; with it the site's favicon (`site/img/icon.png`) and the
  TV apps' icons, logo, splash and banner moved to the mark the same day
  (lootnest-tv-webos 1.6.4, lootnest-tv-android 1.0.2, pinned in
  `TVAppPinned`/`AndroidAppPinned` after 2.22.0). Released as 2.22.0 with the run line above, at the
  owner's ask (22 Nov).
- **2.23.0, the icon at small sizes** (the owner, 23 Nov 2026: the
  ring sat in the lines and leaned left in some icons). `mkicon.geometry(up)`
  keeps strokes 1.5 px, 1.1 px between the arcs and 1.8 px between the
  ring and the inner arc's ends (the inner arc narrows, the ring rises);
  `fit` sizes the mark to the tile; `render` mirrors the left half so
  both sides are the same pixels; the .ico has 16 to 256 with 30, 36, 60,
  72, 80 and 96. The TV apps keep their 1.6.4 and 1.0.2 icons.

- **2.23.0, runs per game** (the owner, 23 Nov 2026: a run made in
  RDR2 from a save brought as a mod showed in Counter-Strike 2's Who's
  playing). `Player.Game` is the game a run is for; `ValidatePlayers`
  clears it for people, keeps names apart per person and game
  (`runNameKey`), and limits `MaxPeople` 16, `MaxRunsPerGame` 16.
  `core.AssignRunGames` (at start, app.go) ties an older run to the one
  game it has files in; `core.RunFor` says whether a profile shows in a
  game. `core.CompetitiveGame` (core/runs.go: app ids, name words, a
  kernel anti-cheat by `AntiCheat`) gives no runs; `app/runs.go`
  `runsBlocked` (cached 10 min), `AppState.NoRuns`, `errRunHere` in
  `startSwap`, `errNewRun` in branch, Try a save and `apiSavePlayers`.
  Page: `runFor`, `runsOk`, `runsIn`, `runsOf`, `canNewRun`,
  `playersFor`, `pickOrder` (10-state.js); the seat is a person
  (`seated()` returns `personOf`); Who's playing is a column per person
  (`pickCol`, `.pk-col`, `.pk-runs-list` scrolls past `--pk-runs-shown`
  5 and fades), New run there (`np-run`, `newRunRow`). A game's page
  shows its folder and the drive's free space (`installWhere`,
  `shortDir` in 72-storage.js, `.gpage-where`); the hero card is now a
  flex item beside the title, not absolute. Harness `?runs=1|many`
  (runs in bg3; cs2 has `noRuns`). docs/TESTING.md 64.

- **2.23.0, PlayStation pads per player and per game** (the owner,
  23 Nov 2026: "Controller connected" over and over on Bluetooth; two
  DualSenses in TEKKEN 8 played as one; a game that knows the DualSense
  needs no Xbox translation). `app/padbridge.go`: `runPad` scans every
  2 s (`psPadsPresent`, sorted, deduped paths) and gives each PlayStation
  pad its own virtual pad (`feedPad`, `padFeeder`, back over a channel);
  a pad that goes leaves its virtual pad `padResting` 30 s for the same
  path; the bridge stays `padHold` 20 s after the last game. Which pads
  games want: `padTick` per running game, `LaunchOptions.KeepPad`, new
  `XboxPad` (one or the other, `CleanLaunchOptions`), else `psKnows`
  (`readPSKnows` once per play: `core.SteamPSPads` from appinfo.vdf
  categories 55/56 DualShock and 57/58 DualSense, `core.DetailsPS` from
  kept details' `Pads`, `core.FolderPS` for libScePad*.dll four levels
  down; core/padknows.go), per family. `win.NewViGEmPad` calls
  VIGEM_WAIT_DEVICE_READY and `Submit` retries ERROR_BUSY (the 2.10.3
  "resource is in use"). Page: Launch options' PlayStation controllers
  select (`LO_PAD`, `padMode`), the "Controller connected" toast once per
  `id#n` within 30 min (`pad.met`, `gamepaddisconnected`), never for an
  Xbox pad while a PlayStation one is there. Harness `?padx=1|2`, shots
  `game-launch-pad`, `settings-controllers-padx`. docs/TESTING.md 65.

- **2.23.0, naming controllers** (the owner, 23 Nov 2026: names for
  pads, and a buzz to tell which is which; of light, rumble and the
  DualSense's speaker they chose rumble, and names only, not tied to
  profiles). `core.Pad.ID` is `PadID(chain)` (padKey lower case: the
  BTHENUM or USB device), `Given`, `Buzz` (a real Sony pad, or the one
  XInput pad whose slot is known). `Config.PadNames` (`SetPadName`,
  `CleanPadName`, 32 runes, 64 pads); `ps_padName`, `ps_padBuzz` (event
  `padBuzz`). `core.PSRumble` builds the motor report (DualSense USB
  0x02 / BT 0x31 with seq and CRC, DualShock 4 USB 0x05 / BT 0x11; only
  the motor flags; buffer as long as `HidP_GetCaps` says, CRC at 74).
  `win/padbuzz.go`: `HIDInstance` (interface path to instance id),
  `OpenPadWrite`/`PadWriter`, `XInputRumble`. Settings, Controllers rows
  (`ctlKeys`, `padTitle`, `S.padNaming`, `S.padBuzz`, actions `pad-buzz`,
  `pad-name`, `pad-name-save`); the bridge's note names fed pads
  (`fedNames`, `padNameOfPath`, `andList`). Icon `vibrate`. Harness
  `?pad=ds5,ds5b,xbox&padnames=1&buzz=hold|fail`, shots
  `settings-controllers-names`, `-naming`. docs/TESTING.md 66.

- **2.23.0, controller batteries** (the owner, 23 Nov 2026: battery
  level, a low warning, a battery saver with no vibration or other
  drains, charging and charged). `core/padbattery.go`: `PSBattery`
  (status byte at 53 DualSense USB 0x01, 54 BT 0x31, 30 DualShock 4 USB
  0x01, 32 BT 0x11; a Bluetooth short 0x01 carries none), `XInputBattery`
  (type and level), `BatteryWarn` (low 20, empty 10, again over 30 or
  charging; "charged" once from charging to full), `SaverWanted` ("",
  "low", "on"), `PSSaver` (light bar and player lights out, DualSense
  power save 0x04 = haptics asleep; off sets haptics back and the light
  bar blue). `app/padbattery.go`: `batteryLoop` every 30 s and on
  `batteryAsk` (apiControllers, apiSaveSettings); `readPSBattery` reads
  reports for 1.5 s; a Bluetooth pad sending short reports is asked for
  feature 0x05 once (`PadWriter.Feature`); `batterySay` is the event
  `padBatterySay` when the window shows, else `plat.ShowToast`; event
  `padBattery`; `saverHolds` stops Vibrate. `Settings.PadBatteryOff`,
  `PadSaver`. `win.XInputBattery`. UI: `battOf`, `battChip`, `battLine`
  in 77-controllers.js, the Battery section, `bpBattery` in Big
  Picture's bar (`.bp-batt`), Big Picture's Settings row `saver`. Harness
  `?batt=1|saver`, shots `settings-controllers-battery`, `-saver`,
  `tv-battery`. docs/TESTING.md 67.
  The owner asked for 2.23.0 with these five (23 Nov).
- **2.23.1, an empty window on 2.23.0** (the owner, 23 Nov 2026:
  the window showed nothing, the log "the interface hasn't loaded after
  20s"). WebView2's NavigateToString takes 2 MB once it has made the page a
  base64 data: URL, about 1.5 MB of HTML; 2.22.0's page was 1.55 MB,
  2.23.0's 1.58 MB, and a page over the limit shows nothing at all, not
  even the splash. `pageHTML` now holds a loader (`pageLoader`) in place of
  the script: it asks `ps_vendor("app")` (`appScript`, the script with the
  sounds) and runs it with the page's nonce, as `loadHls` does. The page is
  0.5 MB; `page_test.go` fails above `maxPageHTML` (1.2 MB). The harness
  still inlines the script; Linux's window loads it the same way.
  Released as 2.23.1 without asking, since 2.23.0 showed nothing.
- **Unreleased, the play bar in one height** (the owner, 23 Nov 2026:
  the full path and keys of different sizes looked bad; of two mocks
  they chose A). `playSplit(it, p)` (40-game-page.js) is Play and the
  profile as one split key in `--player` (`.gp-play`, `.gp-play-go`
  plays as, `.gp-play-who` opens `whoMenu` when the game has more than
  one profile; `whoSwitch` is gone); `whoState` is the line beside it
  with a lamp (`.gp-ready`, `.gp-lamp.is-on`). Every key in the bar is
  `--ctl-bar` 52px (`.gp-key`, `.prof-pill`); "Play as someone else" is
  **Someone else** (Another run when solo). Under the title, chips
  (`.gp-where`, `.gp-chip`): the folder's name (`folderName`, the path
  as its title, opens it) and the size with a drive meter (`.gp-meter`,
  `--used`, `--meter-w`/`--meter-h`) and the space free, amber when
  low (`.is-low`). `shortDir` is gone. Tokens `--on-player-*`,
  `--bar-avatar`.

**The rework (owner, 13 Nov 2026):** every screen of the launcher in one
new look, direction A of three mockups with a sidebar: the game in focus
fills the window, glass only over art, the website's colours, no section
colours, no top bar. The plan, the phases (R0 to R9, one release each)
and the inventory of every surface are in `docs/REWORK.md`; the look is
`design/rework/reference.html`. Read both before touching the UI, and
tick the inventory as phases merge. Only one chat works on it at a time.

The website is `site/`; `docs/ROADMAP.md` holds the plan for bigger
features; docs/RELEASING.md the owner's signing and winget steps.

Not checked on a real PC yet: the owner was on 1.19.x at the last log.
Everything from 1.40 on that touches Windows is unchecked there too:
installers run and watched, the toast window over games, the firewall
rules (PowerShell as administrator), Ctrl+Shift+F5, and save points made
while a game runs.
1.23.2 fixed achievements not showing for Unreal folder games such as
Phantom Blade Zero. Ask for the "Show recent activity" log after they
update. It names each game and says why none were found.

## Roadmap

`docs/ROADMAP.md` is the plan for the bigger features the owner asked for
(plugins, mods, controllers, game details, timeline and captures, UI, GOG
and itch.io, updates found in Downloads), each phased with sizes, its
key constraint and what Lootnest won't do, plus the order of work. Read
it before starting any of them.

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
   (1.26), a conflict banner, one-click restore of the pre-swap copies,
   and an anti-cheat note on the page (1.50). Still open: online-only
   saves out of swaps by default.
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

**The achievement overlay.** For online games, no injection and no
reading game memory: a separate always-on-top window, as Xbox Game Bar or a desktop companion
app does. It shows over windowed and borderless games, and over most
"fullscreen" ones on Windows 10 and 11 (fullscreen optimizations); not
over true exclusive fullscreen, where it says to switch to borderless.
Off by default for games with an anti-cheat (EAC, BattlEye, Vanguard).
Story games may get it inside the game instead, over exclusive
fullscreen too (Story games and online games, above).
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
2. **Try a save safely** (done in 1.53): import a save from a friend or a mod site into a
   temporary run; the profile's own saves stay untouched; one click back,
   or keep the run.
3. **Lootnest on a USB stick**: a portable Lootnest carries its profiles;
   on a friend's PC it swaps your saves in and theirs back afterwards,
   with a copy first, installing nothing.
4. **What do we play tonight?** (Tonight on Home, 1.54): from who's seated (co-op games none of
   them finished when two sit), the time they have, Up next, and a
   backlog roulette.
5. **Your year in games** (per profile, 1.55): a yearly recap per profile and per household,
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
never generates, ships or "repairs" anything that makes a game start. It
downloads only games the user owns, from the store they bought them from,
through its official API once the user connects it (itch.io now, GOG
next): never from anywhere else, and never a catalogue. A script plugin
the user installs and allows may offer updates; Lootnest itself brings
no such plugin and no sources, suggests none, and never searches for
games the user doesn't have. Opening Hydra for an
update of a game that came from it is the owner's one other exception (14
Nov 2026): Lootnest opens the app and copies the game's name, nothing more.
Mods from Nexus Mods are the owner's choice of 20 Nov 2026: browsing
through Nexus's public API, and the download on nexusmods.com itself, in
Lootnest's window for that site, by the user; never mods from sites that
copy other people's, and never CurseForge (its terms, docs/ROADMAP.md).

**The save engine's rework** (the owner, 20 Nov 2026: save points
"don't work", and Go back to it "doesn't really do anything"; rework it
all and make it unique). What's wrong now: a point copies only what the
game last wrote to disk, so one made mid-play holds an older save; Go
back to it refuses while the game runs and never starts it again, so
nothing seems to change; a game outside the stores may have its save
folder guessed by name, and the point says nothing. Ideas put to the
owner, not chosen yet: a save that makes the game save first (its own
save code in story games, else its quicksave key), every save the game
makes kept on the timeline by itself, Go back that closes the game, puts
the save in place as the newest and starts it again on that save, and
points that show their files and say when nothing changed. Go back
that closes the game, puts the save in as the newest and starts it again
is done, Save now with the game's quick save key, and every save the
game makes kept by itself (all Unreleased, above); ask before building
the rest.

## Ideas not started

- Disk space: what each game and profile takes, what can move to another
  drive (Game Mover exists for Steam), and what can go, with a backup first.
- Better "Will it run?" data: 1.36 uses Lootnest's own rough score tables
  (`core/perf.go`). They need updating as new parts ship.
- Signing in to Microsoft or Google directly for cloud saves (on hold;
  the folder approach works).
