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
  never a catalogue;
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
  components), `app.css`, and the script in `js/`: one closure cut into
  ordered parts (`00-head.js` to `99-boot.js`, about 5,600 lines) that
  `app.go` (`uiJS`) and the harness join in name order. A part isn't valid
  on its own; put new code in the part it belongs to.
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
  - Query options: `?players=6`, `?games=400`, `?state=loading|offline|error`,
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
- The app icon is a red gem in an ice-grey nest on an obsidian tile.

## Where things stand (1.106.0, beta)

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
next): never from anywhere else, and never a catalogue.

## Ideas not started

- Disk space: what each game and profile takes, what can move to another
  drive (Game Mover exists for Steam), and what can go, with a backup first.
- Better "Will it run?" data: 1.36 uses Lootnest's own rough score tables
  (`core/perf.go`). They need updating as new parts ship.
- Signing in to Microsoft or Google directly for cloud saves (on hold;
  the folder approach works).
