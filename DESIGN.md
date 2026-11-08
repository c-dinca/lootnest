# Lootnest design system: Arcade

Lootnest looks like a piece of game hardware you like having on the desk. The owner chose this direction, **Arcade**, from seven mocked over the real UI (November 2026), and asked for it to be polished, with worthy animations and proper transitions.

- **A shell around a screen.** The top bar and the sidebar are one piece of dark indigo plastic (`--shell`). The page is a lighter screen set into it (`--bg-ground`), rounded where they meet.
- **A cartridge colour for every section.** Home yellow, Library teal, Store blue, Updates orange, Achievements lilac, Saves green, Profiles pink. It is the band of the current tab, and every section title is a label in that colour, dark ink on it, sitting on a 3px rule of it across the section. A game's page is Library teal. A Settings category takes a colour only when it is one of those sections' own (Games and swaps, Online, Updates, Cloud saves); the rest are neutral.
- **Keys.** Buttons sit on a lip and press down 2px. Switches that are on are lit green, like a power LED.
- **Racing Red is the gem and the Play key**, nothing else. Play as someone is that profile's colour.
- **Two faces.** Tilt Warp for page titles, a game's name, the Play key and big numbers; Atkinson Hyperlegible Next for everything else.

The tokens and components live in [`internal/app/ui/system.css`](internal/app/ui/system.css). Every component in every state is on the kitchen sink:

```bash
node design/harness/build.mjs        # writes design/harness/dist/kitchen-sink.html (open it in a browser)
node design/harness/shoot-ks.mjs     # screenshots into design/system/
```

The kitchen sink is dev only. Nothing in `design/` is built into `Lootnest.exe`.

## Principles

1. **Colour means a place or a person.** A section's colour marks only that section: its tab, its labels, its progress. A player's colour marks only that player: Play as them, their avatar, their bar on a cover, always with their name beside it. Green that is lit means on or safe. Racing Red is the gem and the Play key.
2. **Depth comes from the plastic, one step per layer** (shell, well, ground, surface, raised, active). Keys and panels sit on a darker lip (`--lip-key`, `--lip-panel`) with no outline; only floating layers (menus, dialogs, drawers, toasts) cast `--shadow-overlay`. Art shows in the hero and Big Picture only; the surfaces over it are nearly opaque.
3. **Panels are for objects:** a save point, a copy, an update, a conflict, the Tonight pick, a drawer. Facts, lists, stats and sections sit on the screen as type and rows. No card inside a card, no strip of equal stat cards.
4. **Every section starts with its label.** One treatment everywhere: the label in the section's colour on its rule. A note and the section's own button sit on the rule beside it. Groups inside a panel take a small title (`.ln-section-head--minor`), not a label.
5. **One page grid.** Every page starts at `--page-gutter` and grows with the window, splitting on unequal spans (8+4, 9+3). A wider window gets more content, not bigger boxes.
6. **Space shows grouping.** 4 to 12px inside a component, 12 to 16 between related items, 24 to 32 between groups, 48 to 64 between sections. Empty containers are hidden, not drawn.
7. **Few sizes, one display face used large.** Tilt Warp only for a page's title, a game's name, the Play key and scoreboard numbers. Everything else, section labels and dialog titles included, is Atkinson. Hierarchy comes from weight and `--fg-1/2/3` before size. Sentence case.
8. **Missing art is real content:** the game's icon, a blurred crop of its other art, or its name set in type. Never a grey box or a broken image.
9. **Motion is calm and exact.** Under 400ms all told, never bouncy, only transform and opacity (a label's wipe is a clip), once per arrival, and nothing on a redraw the keyboard caused. Content is all there without it, and under `prefers-reduced-motion` nothing moves.
10. **One dominant action per view.** Play leads Home and the game page. Everything else is a ghost or secondary key, or lives in Manage.
11. **The swap is the signature moment** and gets the craft: the profile's colour, the avatar, and an exact copy line. Routine screens stay quiet so it stands out.
12. **Big Picture is the same system at 10-foot scale:** type about 1.5x, a 5% safe area, a double focus ring, a fixed prompt bar, and every screen reachable with the D-pad.

Every UI change is checked in the harness at 1024, 1280, 1440, 1920 and 2560 wide, with `players=6`, `games=0` and `art=missing`, with no sideways scroll.

## Tokens

The names below are CSS custom properties. Components use only these. Raw colours, sizes and durations appear only in the token block.

### Colour

Indigo at hue 282 for the plastic, Ice Grey `#DFE5EB` for text. Contrast is WCAG 2.

| Token | Value | Role | On ground | On raised |
|---|---|---|---|---|
| `--shell` | `oklch(0.165 0.034 282)` | The top bar and the sidebar | | |
| `--shell-lip` | `oklch(0.120 0.030 282)` | The dark edge under a key or a panel | | |
| `--bg-well` | `oklch(0.175 0.036 282)` | Sunk: inputs, wells | | |
| `--bg-ground` | `oklch(0.205 0.040 282)` | The screen: every page | | |
| `--bg-surface` | `oklch(0.250 0.044 282)` | Panels | | |
| `--bg-raised` | `oklch(0.295 0.046 282)` | Keys | | |
| `--bg-active` | `oklch(0.345 0.048 282)` | A key pointed at, selected | | |
| `--fg-1` | `#DFE5EB` Ice Grey | Text | 14.2:1 | 11.0:1 |
| `--fg-2` | `oklch(0.830 0.024 280)` | Secondary text | 10.6:1 | 8.3:1 |
| `--fg-3` | `oklch(0.730 0.030 280)` | Dates, counts | 7.5:1 | 5.8:1 (4.9:1 on active) |
| `--fg-disabled` | `oklch(0.520 0.030 280)` | Disabled controls (exempt) | 3.3:1 | 2.5:1 |
| `--fg-on-light` | `oklch(0.180 0.030 282)` | Dark ink: on Ice Grey, a label, a lit lamp | | |
| `--accent` | `#EF233C` Racing Red | The gem and the Play key | 4.3:1 | |
| `--fg-on-accent` | `oklch(0.140 0.020 20)` | Bold text on the red key | 4.7:1 on red, 5.7:1 on hover | |
| `--lamp` | `oklch(0.82 0.170 148)` | A switch that is on, a game running, done | 11.0:1 | 8.5:1 |
| `--danger` | `oklch(0.80 0.110 30)` | Destructive | 9.3:1 | 7.2:1 |
| `--warning` | `oklch(0.85 0.120 78)` amber, apart from Home's yellow | Needs a look | 11.3:1 | 8.8:1 |
| `--edge-1/2/3` | indigo light at 12 / 20 / 38% | Hairline, control edge, hover edge | | |

**The sections.** Each is a fill with `--ink-on-sec` (the dark ink) on it, never text on the screen. `--sec` is the current one: `render()` sets `data-sec` on the window (and on the top bar, which keeps the tab a game's page came from), and `--sec` is registered with `@property`, so it eases from one section's colour to the next.

| Token | Colour | Section | Ink on it | Ink on a count patch |
|---|---|---|---|---|
| `--sec-home` | `oklch(0.86 0.160 92)` yellow | Home | 12.3:1 | 8.9:1 |
| `--sec-library` | `oklch(0.82 0.120 195)` teal | Library and every game's page | 11.3:1 | 8.2:1 |
| `--sec-store` | `oklch(0.77 0.130 255)` blue | Store | 9.0:1 | 6.6:1 |
| `--sec-updates` | `oklch(0.80 0.150 60)` orange | Updates | 9.6:1 | 7.0:1 |
| `--sec-ach` | `oklch(0.77 0.140 305)` lilac | Achievements | 8.7:1 | 6.4:1 |
| `--sec-saves` | `oklch(0.82 0.170 148)` green | Saves | 11.5:1 | 8.3:1 |
| `--sec-profiles` | `oklch(0.80 0.130 355)` pink | Profiles | 9.5:1 | 7.0:1 |
| `--sec-neutral` | `--fg-2` | Settings' own categories | 11.1:1 | 8.1:1 |

The section's colour marks progress on its pages (achievement bars and rings, loading bars), list markers and links in the reader, and the text selection as a wash (`--sec-wash`).

### Players

Each player's colour is set inline as `style="--player: #d9738e"` on an element with `.ln-player`, which derives:

| Token | Formula | Role |
|---|---|---|
| `--player-fill` | `oklch(from var(--player) 0.72 min(c, 0.12) h)` | Play, avatar, tile bar, tag |
| `--player-fill-hover` / `-active` | lightness +0.05 / −0.04 | States |
| `--player-on` | `oklch(from var(--player) 0.20 min(c, 0.03) h)` | Text on the fill: 6.9:1 or better at rest |
| `--player-lip` | lightness 0.50 | The lip under Play as them |
| `--player-wash` | the fill at 16% | A selected chip |

A player's colour is never the only cue: their name is always beside it.

### Surfaces

| Token | Where |
|---|---|
| `--glass-bar` (surface at 94%) | The play bar over the hero |
| `--glass-panel` (surface at 90%) | Panels on art |
| `--glass-overlay` (indigo 0.27 at 97%) | Menus, the search's list |
| `--shadow-overlay` | Floating layers only |
| `--shadow-lift` | A cover lifted under the pointer |
| `--lip`, `--press`, `--lip-key`, `--lip-down`, `--lip-panel` | A key's 3px lip, the 2px it goes down, and the lips as shadows |
| `--rule`, `--slot-band` | A label's 3px rule; the 4px band of the current tab |

With `prefers-reduced-transparency: reduce`, every see-through step becomes solid and the backdrop goes away.

### Type

Three families, all under the **SIL Open Font License 1.1**, embedded as woff2 in `internal/app/ui/fonts` (from @fontsource):

- **Tilt Warp** (`--font-title`), one weight, declared as a weight range so the browser never fakes bold. Page titles, a game's name, the Play key, scoreboard numbers (`.ln-meta dd`), the brand.
- **Atkinson Hyperlegible Next** (`--font-ui`), weight 200 to 800, latin and latin-ext. Everything else, at 450 for body, 700 to 800 for labels and titles under 24px.
- **Figtree's digits**: a face of `--font-ui` for 0 to 9 only, declared last, because Atkinson's zero is slashed.

| Role | Size | Use |
|---|---|---|
| `--text-hero` | 40 to 64 | A game's name on its page and on Home |
| `--text-page` | 34 | Page titles |
| `--text-score` | 26 | Numbers you read off a scoreboard |
| `--text-section` | 22 | Drawer, dialog and empty-page titles (Atkinson 800) |
| `--text-lead` | 18 | Values that stand out |
| `--text-label` | 16 | A section's label (Atkinson 800) |
| `--text-body` | 14 | Body, names, buttons |
| `--text-meta` | 13 | Secondary lines: dates, counts, notes |

Long prose is set to at most 65ch with `text-wrap: pretty`. Numbers in columns use tabular figures (`.ln-num`). No all-caps and no tracked-out eyebrows.

### Space, size, radius

- **Space:** 4px steps, `--space-1` (4) to `--space-16` (64); roles `--gap-in` (12), `--gap-group` (32), `--gap-section` (48).
- **Controls:** `--ctl-sm` 32, `--ctl-md` 40, `--ctl-lg` 48, `--ctl-play` 56 (Play in the play bar). Icons 16 or 18px with 1.5px strokes.
- **Radius by role:** `--radius-label` 2 (labels, like a sticker), `--radius-tag` 4 (tags, keyboard keys), `--radius-ctl` 6 (keys, rows, inputs, switches), `--radius-cover` 8 (covers), `--radius-panel` 10 (panels, toasts, menus), `--radius-dialog` 14 (dialogs, the play bar), `--radius-round` (avatars only). The screen's corner is `--screen-r` 14.

### Motion

| Token | Value | Use |
|---|---|---|
| `--ease-emph` | `cubic-bezier(.16, 1, .3, 1)` | Things arriving: fast out, long settle |
| `--ease-out` | `cubic-bezier(.2, .8, .2, 1)` | Hover, fades |
| `--dur-press` / `--dur-release` | 90 / 160ms | A key going down, coming back up |
| `--dur-1` | 120ms | Hover; the page you leave fading |
| `--dur-2` | 180ms | A switch, a dialog, a menu |
| `--dur-wipe` | 220ms | A label wiping in |
| `--dur-drawer` | 260ms | A drawer sliding in |
| `--dur-3` | 280ms | The cartridge sliding, a page's head settling, a toast rising |
| `--rise`, `--settle`, `--stagger`, `--lift` | 6px, 1.015, 40ms, 3px | How far a head comes up, the scale it settles from, the gap between labels, how far a cover lifts |

- **A page change** (`swapPage`, `enter` in `40-game-page.js`; the order in app.css, motion): the new page is drawn at once under a still copy of the one you leave (`.page-ghost`, which keeps its colour), and that copy fades out in `--dur-1`, so no frame is empty. Art on the PC is decoded first, for at most `--dur-1`. Then, once per arrival: the page's first part (its hero or head) settles from `--settle` and `--rise` in `--dur-3`; the labels in view wipe in left to right, `--stagger` apart, up to five (`.wipe`, `--i`); the rest of each section fades in after `--dur-1`. It all ends within 380ms. A redraw while it runs carries on from where it was. In the harness, a page change measures no frame below 0.5 opacity.
- **The top bar:** the current tab is a cartridge (`.bar-ind`) that slides to the new tab in `--dur-3` with `--ease-emph`, while `--sec` eases to the new section's colour, so the band and the labels change together.
- **Keys** go down `--press` with the lip folding under in `--dur-press`, and come back in `--dur-release`. Chips do the same.
- **Covers** lift `--lift` with `--shadow-lift` and a light edge (`--edge-hi`) under the pointer, and go back down when clicked.
- **Switches:** the knob slides and the lamp comes on in `--dur-2`.
- **Overlays:** drawers slide in from the right in `--dur-drawer` as the scrim fades, dialogs grow from 0.97, toasts rise 12px, menus grow from what opened them.
- **The seat:** a face pops in (0.9 to 1) when someone sits down.
- **The swap moment** in "Who's playing?": the avatar whose files were in slides out and fades, the chosen one slides in, both `--dur-3`.
- **Sounds** follow the same rule: one per thing you do, never on hover or a redraw, never while a game runs, none with Sounds off.

## Components

All in `system.css`, prefixed `ln-` (the switch is `.switch`). Every interactive one has default, hover, active, focus-visible and disabled, plus loading where it applies. The kitchen sink forces states with `data-force="hover|active|focus"`.

| Component | Class | Variants and states | Rules |
|---|---|---|---|
| Key (button) | `.ln-button` | `--primary` (the red key), `--secondary` (a raised key), `--ghost` (a key with no cap), `--danger`, `--play` (with `.ln-player`). Sizes `--sm`, `--lg`, `--icon`. Loading is `aria-busy="true"` plus `.ln-spinner` | One primary or one Play per page. Keys with a cap press down. |
| Section label | `.ln-section-head` (from its h2), `.ln-label` | A count as a darker patch (`.ln-count`), a note and an action on the rule; `--minor` for a small group title | One per section, in the section's colour, on its rule. |
| Switch | `.switch` | Off, on (lit), disabled | Off is a dark slot; on is lit green with a soft glow. |
| Avatar | `.ln-avatar` | 20 to 160, an initial or a picture, `.is-run` | Always beside the name. |
| Chip | `.ln-chip` | `aria-pressed="true"` (this player's files are loaded) | A small key: "or play as… Cristian". |
| Tag | `.ln-tag`, `--player`, `--strong`, `--sec`, `.ln-mark` | | Short states such as Playing, Steam, Update. |
| Game tile | `.ln-tile` | Owners bar (tall bar = loaded), badge, no-art cover, stand-in art, skeleton | The cover is the content, a cartridge that lifts. |
| Metadata | `.ln-meta`, `.ln-facts` | Values first (scoreboard numbers), then what they are; side-column rows | |
| Action list | `.ln-list`, `.ln-row`, `--danger` | | Hairline-ruled rows grouped by intent. |
| Tabs | `.ln-tabs` | The current tab is a cartridge: raised, the section's colour along its top | Arrows, Home and End move between tabs. |
| Segmented control | `.ln-segmented` | A well with keys; the chosen one raised with a band of the section's colour | Radio group; arrows move the choice. |
| Input | `.ln-input`, `.ln-search` | In a well | |
| Banner | `.ln-banner` | A panel on its lip; its icon a lit lamp | A notice that needs a decision. |
| Drawer | `.ln-drawer`, `--wide` | Head, a body that scrolls, an optional foot | Slides in from the right; Escape and the scrim close it. |
| Toast | `.ln-toast`, `--success`, `--error` | One optional ghost action | Same words as the action that caused it. |
| Empty / loading / error | `.ln-state`, `.ln-empty`, `.ln-ghosts`, `.ln-steps` | | Says what's going on and what to do. |
| Backdrop | `.ln-backdrop` | The game's art, blurred | Only behind the hero and Big Picture. |

## Accessibility

- **Contrast:** every text token passes AA on every solid step and the see-through ones over art; dark ink passes on every section colour and on a count patch (tables above); text on any player colour passes.
- **Focus:** a 2px `--fg-1` ring on everything interactive. Covers ring their cover; rows ring inside their edge.
- **Keyboard:** everything is a real `<button>` or `<a>`; segmented controls and tabs move with the arrows.
- **Reduced motion:** nothing moves: pages swap at once, keys don't go down, covers don't lift, toasts and drawers fade.
- **Reduced transparency:** see-through layers become solid and the backdrop goes away.

## Banned defaults used on purpose

- **A colour per section.** It would be decoration elsewhere; here it is navigation, the same colour in the bar, the labels and the progress of that section, and the owner chose it.
- **An icon next to every action row.** The action lists are scanned more than read.
- **"or play as…" with an ellipsis.** The direction's own copy: the sentence the faces complete.

## Moving pages over

The Arcade system, the shell (top bar and sidebar), the motion and Home shipped in 1.93. The rest follow, one batch per version:

1. Library, the game page and Big Picture.
2. Settings, Profiles, dialogs and drawers.
3. Saves, Updates, Achievements, the Store, the first run and empty states.

Until a page's batch, it already takes the new tokens, keys, labels and motion, and keeps its own layout.
