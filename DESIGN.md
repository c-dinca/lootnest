# Lootnest design system

Lootnest looks like the website: obsidian, Ice Grey, a Racing Red gem, and the game you're looking at filling the window behind glass. The owner chose this in the rework of November 2026 (direction A of three mockups, with a sidebar), asking for it to be premium, see-through, simple and spectacular, with nothing left in the old look. `docs/REWORK.md` has the plan and the phases; `design/rework/reference.html` is the interactive reference.

- **The art is the room.** The game in focus (Home's game, a game's page) fills the window behind everything (`#room`), fading to obsidian low and on the left. Pages without a game of their own sit on flat obsidian; the Library keeps the last art, blurred and dark.
- **Glass only over art.** The sidebar, the play dock, the sheet of a game's page, menus, drawers and toasts are glass: obsidian at 46% or more, blurred, a 1px light edge and a light along the top. Over flat obsidian, surfaces are solid.
- **The website's colours.** Obsidian ground, Ice Grey text. Racing Red is the gem, the Updates badge and one primary key where there's no Play; never text. Play is the seated profile's colour. No section colours.
- **One sidebar**, no top bar: the brand, Back and Forward, search, the sections with a light that slides to the current one, the games, and at the foot who's at the PC, Profiles, Big Picture and Settings.
- **Small icons, premium keys.** Phosphor Icons, 14 to 18px; every key has a fine light along its top and a faint rim, and presses 1px.
- **Two faces.** Tilt Warp for titles, a game's name, the Play key and big numbers; Atkinson Hyperlegible Next for everything else.

The tokens and components live in [`internal/app/ui/system.css`](internal/app/ui/system.css), the shell and pages in `app.css`. Every component in every state is on the kitchen sink:

```bash
node design/harness/build.mjs        # writes design/harness/dist/kitchen-sink.html (open it in a browser)
node design/harness/shoot-ks.mjs     # screenshots into design/system/
```

Nothing in `design/` is built into `Lootnest.exe`.

## Principles

1. **Content first.** The game's art and the player's things lead; the interface steps back into glass and quiet type. Nothing decorates.
2. **Colour means a person or a state.** A profile's colour marks only that profile: Play as them, their avatar, their bar on a cover, the room's wash while they're seated, always with their name beside it. Lamp green means on or done; amber needs a look; the danger colour is destructive. Racing Red is the gem, the Updates badge and the one primary key. Achievement rarity has its tiers.
3. **Glass only over art.** A see-through surface needs something to see through. On flat obsidian, panels are solid `--bg-surface`. Text on glass stays AA: the room under it is scrimmed (`--scrim-room`) and the glass carries enough obsidian.
4. **Depth by light, not lines.** Layers step up in lightness (well, ground, surface, raised, active); edges are 1px light at 7 to 12%; keys carry a 1px top light (`--key-light`). Only floating layers cast `--shadow-overlay`. No heavy borders.
5. **Panels are for objects:** a save point, a copy, an update, a conflict, a drawer. Facts, lists and stats sit as type and rows. No card inside a card, no strip of equal stat cards.
6. **Plain titles.** A section's title is Tilt Warp in Ice Grey (`--text-section`), with a note and its own key beside it; a count is a small pill. No labels, patches or rules.
7. **One page grid.** Every page starts at `--page-gutter` and grows with the window, splitting on unequal spans (8+4, 9+3).
8. **Space shows grouping.** 4 to 12px inside a component, 12 to 16 between related items, 24 to 32 between groups, 48 to 64 between sections.
9. **Few words.** Names, numbers, states and keys; one short line only before something changes files.
10. **Missing art is real content:** the game's icon, a blurred crop of its other art, or its name set in type. Never a grey box or a broken image.
11. **Motion answers what you did.** Transform and opacity only. Art crossfades; pages rise and settle; lights slide on a spring; covers lift; nothing loops. Settings, Appearance, Motion: Full, Less (fades only) or Off; reduced motion means Off.
12. **One dominant action per view.** Play leads Home and a game's page. Everything else is a secondary or ghost key, or lives in Manage.
13. **The swap is the signature moment:** the profile's colour, the avatar, three steps (back to its slot, a copy read back, in).
14. **Big Picture is the same system at 10-foot scale:** the room and glass, type about 1.5x, a 5% safe area, a double focus ring, a fixed prompt bar, every screen reachable with the D-pad.

Every UI change is checked in the harness at 1024, 1280, 1440, 1920 and 2560 wide, with `players=6`, `games=0` and `art=missing`, with no sideways scroll.

## Tokens

The names below are CSS custom properties. Components use only these; raw colours, sizes and durations appear only in the token block.

### Colour

Contrast is WCAG 2.

| Token | Value | Role | On ground | On active |
|---|---|---|---|---|
| `--bg-well` | `#0f1015` | Sunk: inputs, wells | | |
| `--bg-ground` (`--shell`) | `#111217` obsidian | Every page; the ground the sidebar floats over | | |
| `--bg-surface` | `#181a21` | Solid panels | | |
| `--bg-raised` | `#1f222b` | Raised panels | | |
| `--bg-active` | `#282b35` | Pointed at, selected | | |
| `--fg-1` | `#DFE5EB` Ice Grey | Text | 14.7:1 | 11.1:1 |
| `--fg-2` | `#a6aeb9` | Secondary text | 8.4:1 | 6.3:1 |
| `--fg-3` | `#8c94a0` | Dates, counts | 6.1:1 | 4.6:1 |
| `--fg-disabled` | `#5d636e` | Disabled controls (exempt) | 3.1:1 | |
| `--fg-on-light` | `#111217` | Dark ink: on Ice Grey, a lit lamp, a profile's colour | | |
| `--accent` | `#EF233C` Racing Red | The gem, the Updates badge, the primary key | 4.3:1 | |
| `--fg-on-accent` | `#1a0a0d` | Bold text on the red key | 4.6:1 on red | |
| `--lamp` | `#7be08f` | On, running, done | | |
| `--danger` | `#ff8a7a` | Destructive | 8.4:1 | |
| `--warning` | `#f2c14e` | Needs a look | 10.4:1 | |
| `--edge-1/2/3` | white at 7 / 12 / 22% | Hairline, control edge, hover edge | | |

The old section tokens (`--sec-*`, `--sec`) still exist so older rules read something, but they're all the same quiet grey.

### Glass and the room

| Token | Value | Where |
|---|---|---|
| `--glass` | obsidian at 46% | The sidebar, the play dock, the sheet of a game's page |
| `--glass-strong` | obsidian at 62% | Glass that holds body text; panels on art |
| `--glass-overlay` | `#181a21` at 94% | Menus, drawers, dialogs, toasts, the search's list |
| `--glass-edge`, `--glass-light` | white 9%; a 1px top light at 7% | The rim and the light of every glass surface |
| `--blur-glass` | `blur(22px) saturate(150%)` | Behind every glass surface |
| `--glass-ctl`, `-hover`, `-active`, `-on` | white 7.5%, 13%; black 25%; white 16% | Secondary keys, wells for search and tabs, the lit pill |
| `--room-h`, `--room-fade` | 82%, 700ms | How far the art reaches; one art giving way to the next |
| `--scrim-room` | obsidian gradients | The art fading to obsidian low and on the left |
| `--room-dim` | `blur(30px) brightness(0.42)` | The room behind the Library |
| `--mesh-seat`, `--mesh-red` | 22%; red at 6% | The seated profile's wash low on the left; the faint red one top left |

Solid instead: Windows' transparency effects off (`prefers-reduced-transparency`), or Settings, Appearance, See-through panels off (`.glass-off` on the window). Art behind pages off is `.room-off`.

### Players

Each player's colour is set inline as `style="--player: #d9738e"` on an element with `.ln-player`, which derives:

| Token | Formula | Role |
|---|---|---|
| `--player-fill` | `oklch(from var(--player) 0.72 min(c, 0.12) h)` | Play, avatar, tile bar, tag |
| `--player-fill-hover` / `-active` | lightness +0.05 / −0.04 | The sheen's top, pressed |
| `--player-on` | `oklch(from var(--player) 0.20 min(c, 0.03) h)` | Text on the fill: 6.9:1 or better |
| `--player-wash` | the fill at 16% | A selected chip |

A player's colour is never the only cue: their name is always beside it.

### Icons

Phosphor Icons (MIT), copied by `tools/icons.py` into `js/01-icons.js` and drawn with `ph(name, weight)`; static markup asks with `<i data-ph="name">` (`data-two` for both weights). Regular in keys, rows and menus; fill for the current item; duotone only for an empty page's big icon. Sizes: `--icon-xs` 14 (small keys, chips), `--icon-sm` 16 (keys, rows), `--icon-md` 18 (the sidebar), `--icon-lg` 24 (Big Picture), `--icon-empty` 40 (an empty page). An icon goes with a word; icon-only keys carry an aria-label and a tooltip.

### Type

Three families, all under the **SIL Open Font License 1.1**, embedded as woff2 in `internal/app/ui/fonts`:

- **Tilt Warp** (`--font-title`): page and section titles, a game's name, the Play key, scoreboard numbers, drawer and dialog titles, the brand.
- **Atkinson Hyperlegible Next** (`--font-ui`), 200 to 800: everything else, 450 for body, 700 for keys.
- **Figtree's digits** for 0 to 9, because Atkinson's zero is slashed.

| Role | Size | Use |
|---|---|---|
| `--text-hero` | 40 to 64 | A game's name on its page and on Home |
| `--text-page` | 34 | Page titles |
| `--text-board` | 40 | A page's own numbers |
| `--text-score` | 26 | Numbers you read off a scoreboard |
| `--text-section` | 22 | Section, drawer, dialog and empty-page titles (Tilt Warp) |
| `--text-lead` | 18 | Values that stand out |
| `--text-play` | 17 | The Play key |
| `--text-body` | 14 | Body, names |
| `--text-meta` | 13 | Secondary lines: dates, counts, notes |

Long prose is set to at most 65ch. Numbers in columns use tabular figures (`.ln-num`). Sentence case; no all-caps, no tracked-out eyebrows.

### Space, size, radius

- **Space:** 4px steps, `--space-1` (4) to `--space-16` (64); roles `--gap-in` (12), `--gap-group` (32), `--gap-section` (48).
- **Keys:** `--ctl-sm` 32, `--ctl-key` 38, `--ctl-lg` 48 (Play).
- **Radius by role:** `--radius-tag` 4 (tags, keyboard keys), `--radius-ctl` 8 (rows, inputs), `--radius-key-sm` 8, `--radius-key` 10, `--radius-play` 12, `--radius-cover` 8, `--radius-panel` 14 (panels, toasts, menus), `--radius-dialog` 18 (dialogs, drawers), `--radius-glass` 20 (the sidebar, a game's sheet), `--radius-round` (avatars, pills, switches).

### Motion

| Token | Value | Use |
|---|---|---|
| `--ease-emph` | `cubic-bezier(.16, 1, .3, 1)` | Things arriving: fast out, long settle |
| `--ease-out` | `cubic-bezier(.2, .8, .2, 1)` | Hover, fades |
| `--ease-spring` | `cubic-bezier(.34, 1.4, .5, 1)` | Lights that slide, a switch's knob, a cover landing |
| `--dur-press` / `--dur-release` | 90 / 160ms | A key going down, coming back |
| `--dur-1`, `--dur-2`, `--dur-3` | 120, 180, 280ms | Hover; a switch, a dialog, a menu; a head settling, a toast |
| `--dur-drawer` | 260ms | A drawer sliding in |
| `--dur-page` | 600ms | The sidebar's light, a page rising |
| `--room-fade`, `--dur-backdrop` | 700, 800ms | The room's art giving way; its dimming |

- **A page change** (`swapPage`, `enter` in `40-game-page.js`): the new page is drawn at once under a still copy of the one you leave (`.page-ghost`), which fades out, so no frame is empty; then the page settles in. The room crossfades to the new game's art, or to flat obsidian.
- **The sidebar:** the light (`.bar-ind`) slides to the current section on a spring; that section's icon fills.
- **Keys** go down 1px and to 98% in `--dur-press`, their top light becoming a shade, and come back in `--dur-release`.
- **Covers** lift with `--shadow-lift` and a light edge under the pointer.
- **Switches:** the knob springs across and lights green.
- **Overlays:** drawers slide in as floating glass, dialogs grow from 0.97, toasts slide in from the right, menus grow from what opened them (upward from the sidebar's foot).
- **Sounds** follow the same rule: one per thing you do, never on hover or a redraw, never while a game runs, none with Sounds off.

## Components

All in `system.css`, prefixed `ln-` (the switch is `.switch`). Every interactive one has default, hover, active, focus-visible and disabled, plus loading where it applies. The kitchen sink forces states with `data-force="hover|active|focus"`.

| Component | Class | Variants and states | Rules |
|---|---|---|---|
| Key (button) | `.ln-button` | `--play` (with `.ln-player`: the profile's colour, a sheen, a soft glow, Tilt Warp), `--primary` (Racing Red), `--secondary` (white 7.5%, on glass or obsidian), `--ghost` (no cap until pointed at), `--danger`. Sizes `--sm`, `--lg`, `--icon`. Loading is `aria-busy="true"` plus `.ln-spinner` | One Play or one primary per view. Icon then word. |
| Section title | `.ln-section-head` (from its h2), `.ln-label` | A count as a pill (`.ln-count`), a note and a key beside it; `--minor` for a small group title | Tilt Warp, Ice Grey; no patch, no rule. |
| Tabs | `.ln-tabs` | Pills in a well, the current one lit | Arrows, Home and End move between tabs. |
| Segmented control | `.ln-segmented` | Pills in a well, the chosen one lit | Radio group; arrows move the choice. |
| Switch | `.switch` | Off, on (lit green), disabled | A pill; the knob springs across. |
| Avatar | `.ln-avatar` | 20 to 160, an initial or a picture, `.is-run` | Always beside the name. |
| Chip | `.ln-chip` | `aria-pressed="true"` | A small key with a face. |
| Tag | `.ln-tag`, `--player`, `--strong`, `.ln-mark` | | Short states such as Playing, Steam, Update. |
| Game tile | `.ln-tile` | Owners bar, badge, no-art cover, stand-in art, skeleton | The cover is the content; it lifts. |
| Metadata | `.ln-meta`, `.ln-facts` | Values first, then what they are | |
| Rows on a panel | `.ln-rows` | Each row `--row-h` high | Copies, updates, what needs a look. |
| Lamp | `.ln-lamp`, `--on`, `--warn` | Grey off, lit green on, amber needs a look | Always beside its words. |
| Action list | `.ln-list`, `.ln-row`, `--danger` | | Hairline-ruled rows grouped by intent. |
| Input | `.ln-input`, `.ln-search` | In a well | |
| Banner | `.ln-banner` | Its icon a lit lamp | A notice that needs a decision. |
| Drawer | `.ln-drawer`, `--wide` | Floating glass from the right, inset 12px, rounded | Escape and the scrim close it. |
| Toast | `.ln-toast`, `--success`, `--error` | Glass; one optional ghost action | Same words as the action that caused it. |
| Empty / loading / error | `.ln-state`, `.ln-empty`, `.ln-ghosts` | An empty page: its title, one line, one key | Says what's going on and what to do. |

## Accessibility

- **Contrast:** every text token passes AA on every solid step and on the glass over the scrimmed room; text on any profile colour passes.
- **Focus:** a 2px `--fg-1` ring on everything interactive.
- **Keyboard:** everything is a real `<button>` or `<a>`; segmented controls and tabs move with the arrows; the sidebar's sections are buttons in order.
- **Reduced motion:** nothing moves; pages swap at once; toasts and drawers fade.
- **Reduced transparency:** every glass surface becomes solid; Settings has the same switch.

## Themes

A profile can choose another look, set only through colour tokens, never
raw CSS: Lootnest (this document), Arcade (the indigo grounds of 1.93 to
1.96, its glass tinted to match), High contrast (black grounds, white
text, solid panels and no blur), or a plugin's theme, whose pairs are
checked for AA before it installs (`core/plugintheme.go`).

## Banned defaults used on purpose

- **Glass.** It's often decoration; here it's only ever over the game's art, where seeing through it is the point.
- **Icons beside action rows.** The action lists are scanned more than read.
