# The rework: every screen, one look

The owner, 13 Nov 2026: rework the whole launcher, every screen, nothing
left in the old look. Direction A of three mockups, with a sidebar: the
website's colours, see-through glass, premium, simple, spectacular, and
not "AI slop". The reference is `design/rework/reference.html` (open it in
a browser; it's interactive). Work through the phases below in order, one
PR and one release each, and tick each surface in the inventory when its
PR merges. A surface isn't done until it matches the reference's rules,
including the small ones (empty states, errors, menus, toasts).

## What it looks like

- **The art is the room.** The game in focus (Home's selection, a game's
  page, the picker) fills the window behind everything: its wide art,
  crossfading in 700 ms when the focus changes, fading to obsidian at the
  bottom and on the left. Pages without a game of their own (Saves,
  Updates, Store, Achievements, Profiles, Settings) sit on flat obsidian;
  the Library keeps the art behind, blurred and dark.
- **Glass only over art.** The sidebar, the play dock, the sheet on a
  game's page, toasts and the picker's keys are glass (`--glass`,
  `--glass-blur`, a 1px light edge, a top highlight). Over flat obsidian,
  surfaces are solid `--raised`. Text on glass is never below AA: the
  glass carries at least 42% obsidian, more under body text. Settings,
  Appearance has "See-through panels" (off: every glass is solid) and
  "Art behind pages" (off: obsidian everywhere), and Windows' "Show
  transparency effects" off means solid by default.
- **The website's colours.** Obsidian `#111217` ground, raised `#181a21`
  and `#1f222b`, lines `#2b2e38` / `#3a3e4a`, Ice Grey `#dfe5eb` text with
  `#a6aeb9` and `#7d8591`. Racing Red `#ef233c` for the gem, the Updates
  badge and one primary key where there's no Play key; never text. The
  Play key is the seated profile's colour. Other colours: profile colours,
  achievement rarity, lamp green for on, warning and danger. No section
  colours (`--sec-*` goes), no indigo shell.
- **The mesh.** A soft radial wash of the seated profile's colour low on
  the left, and a faint red one top left, as on the website.
- **One sidebar.** A floating glass panel: the gem, search (Ctrl+K), the
  sections (Home, Library, Store, Updates, Achievements, Saves) with a
  light that slides to the current one, the games (Recent, collections,
  all; drag and drop as today), and at the foot the seated profile (click
  for Who's here) and Settings. No top bar. It still folds to a rail
  (Ctrl+B) and resizes.
- **Type.** Tilt Warp for page titles, game names, the Play key and big
  numbers; game logos where Lootnest has them; Atkinson Hyperlegible
  Next for everything else. Fewer words: names, numbers, states and keys;
  one short line only before something changes files.
- **Keys.** The Play key (profile colour, Tilt Warp, a soft glow), ghost
  keys (white 8% on glass, raised on obsidian), the red primary key,
  icon keys. Pills for tabs and filters. All press (2px, 0.98).
- **Motion.** Transform and opacity only. Art crossfades 700 ms; pages
  rise 18px over 600 ms (`--out`); the sidebar light and tab lights slide
  on a spring; covers lift on hover and grow when selected; the picker's
  keys arrive staggered; the swap runs its three steps (back to its slot,
  a copy read back, in). Settings, Appearance, Motion: Full, Less, Off;
  reduced motion means Off.

## Rules for every phase

- Keep every feature and every `data-action`; this is a new look, not a
  new app. If a screen loses a control, it moves into a "More" menu.
- Tokens only (system.css), WCAG AA, the copy voice in CLAUDE.md.
- Screenshot every surface the PR touches in the harness, before and
  after, at 1440x900 and 1280x720, and look at them; check keyboard focus.
- Update DESIGN.md, CLAUDE.md (Brand, Where things stand) and the
  website's window (`site/site.js`, `site/style.css`) when the look they
  describe changes.
- Every phase is a release: VERSION and CHANGELOG, in plain words.

## Phases

1. **R0, the system and the shell (1.97.0).** New tokens in system.css
   (palette, glass, mesh, motion, radius), the room (`#room`: art layers
   with crossfade, the dim and flat states), the mesh, the sidebar
   replacing the top bar (sections, search, games, seat, Settings, fold,
   resize), keys and pills restyled everywhere, Appearance in Settings
   (the two glass switches, Motion). After R0 every page already wears
   the new colours; later phases redo their layouts.
2. **R1, Home.** The feature (logo, meta, status, the play dock with
   who's in it), the side card (achievements, rarest, save points),
   the Continue shelf that moves the room; Tonight, the save check, news
   and Up next as quiet rows under it; the first run and empty Home.
3. **R2, the Library.** Covers on the blurred room, filters as pills,
   Recent and collection shelves, list view, Not installed, selection
   bar, Views, Sources, scan progress, the sidebar's game menus.
4. **R3, a game's page.** Logo over the art, the glass sheet with the
   play row and tabs (Overview, Achievements, Saves, Mods, Timeline,
   News), every block inside them, details, screenshots, the compact head
   on scroll, conflict and anti-cheat notes.
5. **R4, overlays.** Who's playing and the swap over the game's blurred
   art, every dialog and drawer (Manage, Launch options, Cover and art,
   Copies, Restore, Move, Remove, Try a save, New run, Edit details,
   Mods plan and conflicts, update drawers, Achievement check, Keep
   offline...), the context menu, Ctrl+K, toasts and unlock pops, Back
   and Forward.
6. **R5, the other pages.** Saves, Updates, Achievements, the Store and
   its offer drawer, Profiles and a profile's page, the profile editor,
   Year in games (the recap image's colours too).
7. **R6, Settings.** Every category, Controllers, Plugins, Storage,
   Playnite import, Steam setup, hidden games, shortcuts.
8. **R7, Big Picture, start and outside the window.** Big Picture in the
   same room-and-glass language at 10-foot scale, the boot splash, the
   native unlock toast and tray (`win/toast.go` colours), the installer
   page (`cmd/setup/setup.html`).
9. **R8, motion and speed.** Measure in the harness first (traces, long
   tasks, dropped frames) for page changes, the game page's first 600 ms,
   ?games=400, drawers, the picker, Big Picture; fix redraws; a cover
   flies into its game page (FLIP); numbers before and after in the PR.
10. **R9, themes and the sweep.** Built-in themes on the plugin theme
    machinery (Lootnest default, Arcade kept for those who liked it, High
    contrast), per profile; a final pass over every surface against the
    inventory; the website's window in step.

## Inventory

Tick when merged. The names are the view functions in `internal/app/ui/js`.

- Shell: [ ] sidebar (`30-side-games.js`, sections, seat, Settings) [ ] room and mesh [ ] `renderBar` removed [ ] boot splash (`index.html` #boot) [ ] Back/Forward [ ] Ctrl+K (`renderGq`) [ ] context menu (#ctx) [ ] toasts (#toasts) [ ] safe-mode banner
- Home: [ ] `viewHome` [ ] Tonight [ ] save check [ ] news [ ] `viewWelcome` (first run)
- Library: [ ] `viewLibrary` [ ] shelves [ ] list view [ ] Not installed (`42-owned.js`) [ ] selection bar [ ] Views [ ] `viewSources` [ ] `viewAddDialog`
- Game page: [ ] `viewGame` [ ] tabs and blocks (`62-game-blocks.js`) [ ] `viewShots` [ ] `viewDetailsEdit` [ ] `viewEdit` [ ] `viewGameSettings` [ ] `viewManage` [ ] `viewLaunch` [ ] Mods (`67-mods.js`, `viewModPlan`, `viewModConflicts`) [ ] `viewCopies` [ ] `viewRestore` [ ] `viewMove` [ ] `viewMoveFolder` [ ] `viewRemovePC` [ ] `viewTrySave` [ ] `viewAchDoctor` [ ] `viewBlocked` [ ] `viewUnsorted` [ ] `viewItchInstall` [ ] `viewGogInstall`
- Overlays: [ ] `viewPicker` and the swap [ ] `viewNaming` [ ] confirm dialogs [ ] `viewBackupPassword` [ ] `viewExport` [ ] `viewReader`
- Pages: [ ] `viewSaves` [ ] `viewUpdates` [ ] `viewUpdateDrawer` [ ] `viewUpdHistory` [ ] `viewAchievements` [ ] `viewFind` [ ] `viewOffer` [ ] `viewPlayers` [ ] `viewPlayer` [ ] `viewProfilesDrawer` [ ] `viewProfileEditor` [ ] `viewRecap` (and its canvas)
- Settings: [ ] `viewSettings` (every category) [ ] `viewControllers` [ ] `viewPlugins` [ ] `viewStorage` [ ] `viewPlaynite` [ ] `viewSteamSetup`
- Big Picture and outside: [ ] `25-bigpicture.js` [ ] `win/toast.go` [ ] tray [ ] `cmd/setup/setup.html`
- Also: [ ] DESIGN.md [ ] CLAUDE.md Brand [ ] the website's window [ ] harness screenshots for the README

## To continue in another chat

Paste this, with Opus and the highest effort:

> Read CLAUDE.md, then docs/REWORK.md, then open design/rework/reference.html. Continue the rework from the first phase whose inventory items aren't all ticked: one PR and one release per phase, as CLAUDE.md says. Keep every feature; tick the inventory in the same PR. Answer me in Romanian.

Only one chat at a time works on the rework; the phases touch the same
files.
