# Lootnest design system

The direction is in [`design/DIRECTIONS.md`](design/DIRECTIONS.md), under *Chosen*. It combines:

- the seat and the single Play from **Couch**;
- the player labels from **Save Slot**, on the brand's Obsidian Black, Ice Grey and Racing Red (1.19.1);
- glass over the game's own art, with small radii and hairlines.

The tokens and components live in [`internal/app/ui/system.css`](internal/app/ui/system.css). Every component in every state is on the kitchen sink:

```bash
node design/harness/build.mjs        # writes design/harness/dist/kitchen-sink.html (open it in a browser)
node design/harness/shoot-ks.mjs     # screenshots into design/system/
```

The kitchen sink is dev only. Nothing in `design/` is built into `Lootnest.exe`.

## Principles

1. **Players are the colour.** The interface itself is neutral obsidian, with Racing Red only in small marks. The colour that fills things belongs to a player: their Play, their avatar, their bar on a cover. If you hid the logo, Lootnest should still be recognisable from how players appear.
2. **The game is the light.** On a game page, that game's art, blurred, lights the whole window from behind the glass. Each page takes on the colour of its game, and the chrome stays the same.
3. **One decision per screen.** Each page has one primary action. When the action is for a player, it takes that player's colour.
4. **Precise, not soft.** Radius goes by role, at 3, 4 and 6px, and only avatars are round. Edges are 1px hairlines. There are no drop shadows on things that sit on the page. These are what shipped launchers do, and it's the opposite of the soft, rounded, everything-in-a-card look of generated UI.
5. **Plain words.** The copy voice stays: "Load someone's files", "A copy is made before every swap". A toast uses the same words as the action that caused it.

## Tokens

The names below are CSS custom properties. Components use only these. Raw colours, sizes and durations appear only in the token block.

### Colour

The brand colours (1.19.1):
- **Obsidian Black** `#111217` is the window.
- **Ice Grey** `#DFE5EB` is the text.
- **Racing Red** `#EF233C` is the primary button (1.28) and a touch elsewhere, never a fill for content.

The steps around them are in OKLCH: hue 276 for the grounds, the obsidian's own, and hue 248 for the greys, Ice Grey's. Contrast is WCAG 2 against the ground and against the raised step.

| Token | Value | Role | On ground | On raised |
|---|---|---|---|---|
| `--bg-well` | `oklch(0.150 0.010 276)` | Sunk: inputs, wells | | |
| `--bg-ground` | `#111217` Obsidian Black | The window | | |
| `--bg-surface` | `oklch(0.220 0.011 276)` | Solid panels | | |
| `--bg-raised` | `oklch(0.260 0.011 276)` | Raised controls | | |
| `--bg-active` | `oklch(0.300 0.011 276)` | Selected | | |
| `--fg-1` | `#DFE5EB` Ice Grey | Text | 14.7:1 | 12.3:1 |
| `--fg-2` | `oklch(0.790 0.012 248)` | Secondary text | 9.7:1 | 8.1:1 |
| `--fg-3` | `oklch(0.700 0.012 248)` | Tertiary: dates, counts | 7.0:1 | 5.8:1 |
| `--fg-disabled` | `oklch(0.500 0.010 248)` | Disabled controls (exempt) | 3.1:1 | 2.6:1 |
| `--accent` | `#EF233C` Racing Red | The primary button, marks, bars, switches | 4.4:1 | 3.7:1 |
| `--accent-hover` | `oklch(0.660 0.230 22.6)` | The primary button on hover | | |
| `--fg-on-accent` | `oklch(0.140 0.020 20)` | Bold text on the primary button | 4.7:1 on red, 5.7:1 on hover | |
| `--danger` | `oklch(0.80 0.100 35)` | Destructive | 9.9:1 | 8.3:1 |
| `--success` | `oklch(0.82 0.110 155)` | Done | 11.4:1 | |
| `--warning` | `oklch(0.86 0.110 85)` | Needs attention | 12.4:1 | |
| `--edge-1/2/3` | white at 8 / 14 / 26% | Hairline, control edge, hover edge | | |

**Racing Red: the primary button, and touches.** It passes the 3:1 that non-text UI needs, but not AA's 4.5:1 for small text, so it is never text. The primary button (Save, Sync now, Play when profiles are off) is filled with it, with bold near-black text, `--fg-on-accent`, at 4.7:1 (5.7:1 on hover). White on it would be 4.2:1, so never white. Elsewhere it marks:
- the gem in the logo and the app icon;
- the icon of the page you're on;
- progress (achievement bars, loading bars);
- switches and checkboxes that are on;
- list markers, news dots and link underlines in the reader;
- the text selection, as a wash;
- the selected choice in a segmented control, as a 2px line under it;
- where you are, in the sidebar and in Settings' categories: a 3px bar at the left edge, the icon, and `--accent-wash` behind the label (1.30);
- a 40 by 3px stripe under every page title (`.lib-head h1`), like a livery;
- the edge of a cover you point at, 2px;
- what's new: `.ln-tag--accent`, filled red with bold `--fg-on-accent` text (4.7:1), as on a news item's **Update**;
- a game at 100% on the Achievements page, as its card's edge;
- the scrollbar you hold or point at.

Play as someone stays in that profile's colour: players are the colour. The focus ring is `--fg-1`, never red (red reads as an error) and never a player colour. Destructive stays `--danger`, a lighter coral that reads as a warning next to Racing Red.

### Players

Each player's own colour is set inline as `style="--player: #d9738e"` on an element with `.ln-player`. From it, `.ln-player` derives a set of tokens, using CSS relative colour:

| Token | Formula | Role |
|---|---|---|
| `--player-fill` | `oklch(from var(--player) 0.72 min(c, 0.12) h)` | Play, avatar, tile bar, tag |
| `--player-fill-hover` / `-active` | lightness +0.05 / −0.04 | States |
| `--player-on` | `oklch(from var(--player) 0.20 min(c, 0.03) h)` | Text on the fill |
| `--player-wash` | the fill at 16% | A selected chip |

The hue is kept and the lightness and chroma are fixed, so every colour a player picks works. Text on the fill measures 6.9–7.6:1 at rest across hues, and 6.0:1 or better pressed. A grey stays grey, because chroma is only capped, never raised.

A player's colour is never the only cue. Their name is always printed beside it.

### Glass

| Token | Value | Where |
|---|---|---|
| `--glass-chrome` | slate 0.15 at 62% | Sidebar, top bar |
| `--glass-bar` | slate 0.17 at 70% | The play bar on a game page |
| `--glass-panel` | slate 0.19 at 58% | Panels that sit on art |
| `--glass-overlay` | slate 0.21 at 86% | Toasts, menus, dialogs |
| `--glass-ctl` / `-hover` / `-active` / `-on` | white at 6 / 10 / 4 / 13% | Controls on glass |
| `--blur-glass` | `blur(32px) saturate(150%)` | Every glass surface |
| `--blur-backdrop`, `--backdrop-opacity` | `blur(100px) saturate(115%)`, 0.34 | The game's art behind the window |
| `--shadow-overlay` | one soft shadow | Floating layers only |

Measured over the brightest part of Counter-Strike 2's art, the glass fills keep secondary text at 8.6:1 or better and ink at 12:1 or better.

With `prefers-reduced-transparency: reduce`, every glass step becomes its solid twin and the backdrop goes away.

### Type

There are two families, both under the **SIL Open Font License 1.1**, which allows embedding them in the exe. They are bundled as woff2 in `internal/app/ui/fonts`.

- **Bricolage Grotesque** is used for titles, through `--font-title`. Its axes are optical size 12–96, weight 200–800 and width 75–100. Large titles use `opsz 96` and −0.02em tracking.
- **Figtree** is used for everything else, through `--font-ui`. Its weight axis runs 300–900.

| Token | Size | Use |
|---|---|---|
| `--fs-72` / `--fs-56` | 72 / 56 | Game title (72 at 1800px and wider) |
| `--fs-40` / `--fs-32` | 40 / 32 | Page titles |
| `--fs-24` | 24 | Section titles |
| `--fs-18` | 18 | Values: hours, counts |
| `--fs-15` | 15 | Strong lines: "Your save is in", large buttons |
| `--fs-14` | 14 | Body, buttons |
| `--fs-13` | 13 | Secondary lines |
| `--fs-12` | 12 | Labels under values, tags |

Numbers that sit in columns use tabular figures (`.ln-num`). Labels are sentence case: no all-caps and no tracked-out eyebrows.

### Space, size, radius, motion

- **Space:** 4px steps, from `--space-1` (4) up to `--space-16` (64).
- **Controls:**
  - `--ctl-sm` (32) for small buttons, `--ctl-md` (40) as the default, `--ctl-lg` (48) for Play.
  - Icons are 16 or 18px, drawn with 1.5px strokes.
- **Radius by role:**
  - `--radius-tag` 3px for tags and keys.
  - `--radius-ctl` 4px for buttons, rows, inputs and covers.
  - `--radius-panel` 6px for glass panels, toasts and dialogs.
  - `--radius-round` for avatars only.
- **Motion:**
  - `--ease-out` is `cubic-bezier(.2,.8,.2,1)`, and `--ease-in-out` is used for the skeleton pulse.
  - `--dur-1` (120ms) is for hover and press, and `--dur-2` (180ms) for state changes and toasts.
  - `--dur-3` (280ms) is for the seat changing and a save slot changing hands. Swap motion stays under 320ms in total.
  - There is no bounce or elastic easing.
  - **The swap moment** happens in the "Who's playing?" picker. Each player is a save slot with their label in their colour.
    - Choosing one swaps the saves: the avatar whose files were in slides out to the left and fades (`swap-out`, `--ease-in-out`), while the chosen one slides into the bay from the right (`swap-in`, `--ease-out`).
    - Both take `--dur-3` (280ms), and the whole swap stays under 320ms.
    - Under them: "Loading Mihai's files", then "A copy of Cristian's files was made first."
    - With reduced motion, the chosen avatar fades in and nothing slides.
  - **Elsewhere:**
    - slots lift 4px on hover and come in 30ms apart;
    - covers scale 3% on hover;
    - buttons move down 1px when pressed;
    - toasts rise 8px;
    - pages cross-fade in `--dur-2`.

## Components

All of them are in `system.css`, prefixed `ln-` so they can live beside the old `app.css` while pages move over in Steps 4 and 5. Every interactive one has its default, hover, active, focus-visible and disabled states, plus loading where it applies.

The kitchen sink forces states with `data-force="hover|active|focus"`. Nothing in the app sets that attribute.

| Component | Class | Variants and states | Rules |
|---|---|---|---|
| Button | `.ln-button` | `--primary`, `--secondary`, `--ghost`, `--danger`, `--play` (with `.ln-player`). Sizes `--sm`, `--lg`, `--icon`. Loading is `aria-busy="true"` plus `.ln-spinner`. | One primary or one Play per page. The label keeps its width while loading. Destructive actions are a text button, and the confirm step names the thing. |
| Avatar | `.ln-avatar` | 20 / 24 / 32 / 48 / 72; an initial, or an `<img>` | Always beside the name, or with `aria-label` on its control. |
| Chip | `.ln-chip` | Default, hover, active, focus, `aria-pressed="true"` (this player's files are loaded), disabled | Used to pick a player: "or play as… Cristian". |
| Tag | `.ln-tag`, `.ln-tag--player`, `.ln-mark` | Neutral, player-filled, an 8px player mark | Short states such as Seated, Playing, Steam. |
| Game tile | `.ln-tile` | Cover, title, one meta line. Owners bar (`.ln-tile__owners`, tall bar = loaded), badge, no-art cover (`.ln-tile__none`), skeleton | The cover is the content. Missing art is set in type on a quiet cover, never a broken image. Hover brightens the edge and scales the art 3%. |
| Metadata | `.ln-meta`, `.ln-facts` | A row of values (value first, then what it is), and side-column rows | It replaces the key/value admin table. Includes a loading skeleton and an empty value such as "Nobody yet". |
| Section header | `.ln-section-head`, `--minor` | 24px title with a count and an action, or a 13px label for groups | Every label names the content under it. A label never decorates. |
| Action list | `.ln-list`, `.ln-row`, `--danger` | Default, hover, focus, disabled | Hairline-ruled rows grouped by intent: Profiles, Game files, Shortcuts, Danger. |
| Segmented control | `.ln-segmented` | `role="radiogroup"`; selected, hover, focus, disabled, counts | Switches between views ("Everyone / Mine / With profiles"). The arrow keys move the choice. |
| Drawer | `.ln-drawer`, `--wide`, in `.scrim.ln-drawer-scrim` | Head (title, one line, Close), a body that scrolls, an optional foot. `data-menu`: each action in it closes it first | A panel from the right, over the page, so you keep your place: a game's **Manage** actions, its whole list of achievements, **Names and colours**. Escape and the scrim close it, and focus goes back to what opened it. It slides in 32px in `--dur-3`, and fades with reduced motion. |
| Toast | `.ln-toast`, `--success`, `--error` | Player toast with an avatar; one optional ghost action | Says what happened, with the same words as the action. An error says why, and what to do. |
| Empty / loading / error | `.ln-state`, `--error`, `--inline` | Block or inline. A spinner, an icon, a title, one line and actions | Each says what is going on and what to do next. It is never bare text. |
| Banner | `.ln-banner` | Title, one line and actions; a spinner while something runs | A notice that needs a decision. There's no coloured side edge. |
| Backdrop | `.ln-backdrop` | The game's art, blurred | Game pages, and Home behind the last-played game. |

## Accessibility

- **Contrast:** every text token passes AA on every solid step, and so does every glass step over the art (see the tables above). Text on any player colour passes too.
- **Focus:** a 2px `--fg-1` ring on everything interactive, offset 2px. Covers ring their cover image, and rows ring inside their own edge.
- **Keyboard:** everything is a real `<button>` or `<a>`. Segmented controls are radio groups that move with the arrow keys.
- **Reduced motion:** transitions become instant, Play and tiles stop moving on press and hover, toasts fade in, skeletons stop pulsing and spinners slow down.
- **Reduced transparency:** glass becomes solid, and the backdrop is removed.
- **Scaling:** the design is checked at 1280×720 at 150%, through to 2560×1440. Sizes are in CSS px, so Windows scaling multiplies them.

## Banned defaults used on purpose

- **Glass (backdrop blur).** The owner asked for it, and it does a job: each game page takes its game's light. It is used only for the chrome, the play bar, panels on art and overlays. It falls back to solid under reduced transparency.
- **An icon next to every action row.** The action lists are scanned more than they are read, and the icon is how you find "Open backups" among the folder actions. Navigation keeps its icons for the same reason, and labels elsewhere have none.
- **"or play as…" with an ellipsis.** It is the direction's own copy: the sentence the faces complete.

## Moving pages over

`system.css` loads first, then `app.css`. Pages move to `ln-` components one per PR, in this order:

- **Step 4:** the game page.
- **Step 5:** Home, Library, Players, Cloud saves, Settings, Add game, first run.

Each page's old rules are deleted from `app.css` in the same PR, until `app.css` is gone.

| Page | Status |
|---|---|
| Sidebar, backdrop, toasts | On the system (1.16.0) |
| Game page, profiles on and off, with Cloud save, Shelf and news | On the system (1.16.0) |
| Home, banners (`ln-banner`) | On the system (1.16.1) |
| Library, inputs (`ln-input`, `ln-search`) | On the system (1.16.2) |
| Players and a player's page | On the system (1.16.5) |
| Cloud saves, switches and setting rows | On the system (1.16.6) |
| The seat (who's at the PC) | On the system (1.18.0). It lives in the sidebar's **Who's here** rather than the mock's top-right control, so it never covers a page's own actions or the game art. |
| Settings | On the system (1.16.7). Since 1.30, one category at a time: a list on the left (`.set-nav`, `.set-tab`), as in Steam's settings, and the category's title, line and cards on the right. The open category is kept (`ln.setTab`). |
| Dialogs (Add game, first run, the profile editor, confirmations), the right-click menu, older buttons and form controls | On the system (1.16.8) |
| The "Who's playing?" picker and the swap moment | On the system (1.17.0) |
