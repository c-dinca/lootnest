# Working on Lootnest

Notes for anyone (person or AI agent) picking this repo up in a new session.
Read this first; it holds the conventions and the state of the work, so a
fresh chat can carry on without the old one's context.

## The product

Lootnest is a Windows game manager. Every game on the PC, from Steam, Epic,
GOG, Xbox, EA, Ubisoft, Battle.net, Riot, Amazon, Rockstar, itch.io, game
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
- Ice Grey `#DFE5EB` is text and the primary button.
- Racing Red `#EF233C` is for touches only: the logo gem, the current
  nav icon, progress, switches that are on, checkboxes, reader marks and
  the text selection. It is never small text, because it is 4.4:1 on
  obsidian.
- The app icon is a red gem in an ice-grey nest on an obsidian tile.

## Where things stand (1.25.0)

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

Not checked on a real PC yet: the owner was on 1.19.x at the last log.
1.23.2 fixed achievements not showing for Unreal folder games such as
Phantom Blade Zero. Ask for the "Show recent activity" log after they
update. It names each game and says why none were found.

## Ideas not started

- Disk space: what each game and profile takes, what can move to another
  drive (Game Mover exists for Steam), and what can go, with a backup first.
- "Will it run?": a game's requirements (PCGamingWiki, Steam) against this
  PC's hardware. Opt-in.
- Racing Red on the large buttons, with bold dark text.
- Signing in to Microsoft or Google directly for cloud saves (on hold;
  the folder approach works).
- A market and community research pass, to decide what sets Lootnest
  apart. The owner is running it in another AI; its findings may come back
  as the next brief.
