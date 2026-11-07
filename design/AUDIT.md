# Lootnest UI audit (Step 1)

Method: dual-agent critique. A was a design review of the 67 screenshots and the source. B was the impeccable detector: a static CSS/JS scan plus in-page scans over CDP and an injected overlay, on 8 views at 1280×800. Neither saw the other's output; this file merges them.

All screenshots are in [`before/`](before/). They come from the [harness](harness/README.md), which runs the real UI with real Steam art and fixture data. On purpose, one game has no art, one has another game's art, and one has a long title. The fixtures also cover 0, 13 and 400 games, 1, 2 and 6 players, and loading, offline, error and playing states.

## Score

**24/40, "needs work".**

| # | Heuristic | Score | Key issue |
|---|---|---|---|
| 1 | Visibility of system status | 2 | Offline is silent: all art vanishes with no word. The swap is a sliding bar and one line of text. |
| 2 | Match with the real world | 3 | Plain, human copy. Exceptions: the raw `config.json … unexpected end of JSON input` toast, and two unexplained hour counts on one page. |
| 3 | User control and freedom | 3 | The picker has Esc and Cancel, and backups are made before every swap. But a cloud conflict resolves in one click, and the player remove button is a bare ×. |
| 4 | Consistency and standards | 2 | Gold marks the one thing to do, every toggle and the news dots. Ownership has two encodings. Cloud saves is centred; every other page is left-aligned. |
| 5 | Error prevention | 2 | Dan's ochre button reads as the gold primary. The conflict buttons give no evidence. "Save players" looks disabled. |
| 6 | Recognition rather than recall | 3 | Number keys in the picker help. The cover-size glyphs have no labels. |
| 7 | Flexibility and efficiency | 2 | No per-player filter, no A–Z jump, no way to move through 400 games. |
| 8 | Aesthetic and minimalist design | 2 | Ten equal action rows on the game page. Nine equal toggles in Settings. |
| 9 | Error recovery | 2 | The raw parser error covers content. "Sync now" stays primary while the folder is missing. |
| 10 | Help and documentation | 3 | Good inline explanations. First run never explains what a profile or a swap is. |

**Design specificity: mostly interchangeable.**
- The shell is the standard launcher template. Hide the player colours and the words "Play as", and Home, Library and the game page could ship in Playnite, Heroic or GOG Galaxy.
- Three things are clearly made for this product: the full-screen "Who's playing?" picker, the copy voice, and the player colour carried from the sidebar into the time bar and the profile heatmap.
- The idea that defines the product, whose files are in this game right now, shows up as data on the pages. Nothing on the screen is built around it.

## Top 10 problems, by impact

### 1. [P1] "Whose files are loaded" is a pill on a button, not the page's state
- The most important decision in the app is two equal buttons side by side. "loaded" is a 10px pill.
- **Loaded now: Mihai's files** is row 6 of a key/value table. At 1280×800 that is below the Play buttons, and with 6 players it is below the fold.
- Mihai can't tell at a glance whether Play gives him his own save.
- Screenshots: [`game-on-1280x800`](before/game-on-1280x800.jpg), [`game-on-6players-1280x800`](before/game-on-6players-1280x800.jpg), [`game-on-1280x720@1.5x`](before/game-on-1280x720@1.5x.jpg)
- **Owner's read:** confirmed. It is row 6, not row 5.

### 2. [P1] The picker breaks names mid-word and gives too little to choose by
- With 6 players, names wrap inside the word: "Crist/ian", "Mih/ai", "Ioan/a", "Rad/u". The cause is `overflow-wrap: anywhere` on a `7.5vw` name.
- The giant initials are cropped at the column edges.
- "Has a profile" doesn't say what is in each person's save (last played, hours, when it was saved). This is the product's best idea, and it breaks at the household size it is built for.
- Screenshots: [`picker-6players-1280x800`](before/picker-6players-1280x800.jpg), [`picker-1280x800`](before/picker-1280x800.jpg)
- **Owner's read:** new, not in your list.

### 3. [P1] Home is a streaming template, and its hero picks the wrong thing
- The hero is the most recent game, even when it has no art. Here that is Onimusha, so the biggest block on screen is an empty slab with a "Play" for nobody in particular. At 150% it fills the viewport.
- Nothing answers "what are we playing tonight, and who's playing?".
- Continue playing shows only who played last ("Cristian played 5 hours ago").
- The news grid competes with the play decision.
- The same hero with art and "Play as" buttons ([`home-loading`](before/home-loading-1280x800.jpg)) shows the template itself works. The choice of what goes in it is the failure.
- Screenshots: [`home-1280x800`](before/home-1280x800.jpg), [`home-1920x1080`](before/home-1920x1080.jpg), [`home-1280x720@1.5x`](before/home-1280x720@1.5x.jpg), [`home-6players-1280x800`](before/home-6players-1280x800.jpg)
- **Owner's read:** confirmed, with that refinement.

### 4. [P1] Secondary text fails WCAG AA contrast on every page
- The `--dim` token (`#6a7182`) measures 3.1 to 3.9:1 on the app's dark panels.
- It covers the sidebar's "Who's here", the Library count, every `dt` label on the game page, store labels under covers, every hint, the cloud-saves hints and the news dates.
- The "loaded" pill is 3.3:1 (white at 92% on white at 18% over the player colour).
- AA needs 4.5:1 for this size. The detector, its pixel reads and the overlay all agree.
- Screenshots: [`settings-1280x800`](before/settings-1280x800.jpg), [`library-1280x800`](before/library-1280x800.jpg), [`game-off-steam-1280x800`](before/game-off-steam-1280x800.jpg), [`cloud-1280x800`](before/cloud-1280x800.jpg)
- **Owner's read:** refines "one gray for all secondary text". There are two grays (`--muted`, `--dim`); the second fails AA.

### 5. [P1] The game page hero wastes its space and shows the logo twice
- 150 to 300px of empty hero sits above the poster at 1920 and 2560.
- The poster already carries the logo, and the logo then repeats as separate art.
- CS2's wordmark renders navy on a dark gradient and is barely readable.
- With no art, the title appears twice.
- With mismatched art (Sons Of The Forest showing The Forest), the hero is a crop of legs.
- Screenshots: [`game-on-1920x1080`](before/game-on-1920x1080.jpg), [`game-on-2560x1440`](before/game-on-2560x1440.jpg), [`game-off-noart-1280x800`](before/game-off-noart-1280x800.jpg), [`game-off-mismatched-1280x800`](before/game-off-mismatched-1280x800.jpg)
- **Owner's read:** confirmed. Adds the illegible logo and the duplicate title.

### 6. [P1] The game page actions read as a settings page
- Ten title-and-subtitle rows of equal weight, from Load someone's files and Edit game down to Turn off profiles.
- "Affects who plays" and "opens a folder" look the same. The destructive action is just the last row, in red.
- While someone plays, rows fade with no reason given.
- With profiles off, "Turn on profiles", the most important action, is one more card.
- Screenshots: [`game-on-2560x1440`](before/game-on-2560x1440.jpg), [`game-on-playing-1280x800`](before/game-on-playing-1280x800.jpg), [`game-off-steam-1280x800`](before/game-off-steam-1280x800.jpg)
- **Owner's read:** confirmed.

### 7. [P1] A cloud conflict is decided with no evidence and no reassurance
- "Changed on two PCs" gives **Keep this PC's** and **Use LAPTOP's** with no times, no hours, no "the other copy is kept" and no confirm.
- That reassurance exists in the picker's conflict question, but not here.
- Conflicts sit below four settings rows.
- Screenshots: [`cloud-1280x800`](before/cloud-1280x800.jpg), [`cloud-1920x1080`](before/cloud-1920x1080.jpg)
- **Owner's read:** new.

### 8. [P2] Offline and error states are missing or raw
- **Offline:** every cover vanishes with no banner, and the loaded badge prints over the placeholder title.
- **Config error:** it is a raw parser message in a toast over the content.
- **Cloud error:**
  - **Sync now** stays the gold primary while the folder is missing.
  - "Looking for its saves…" and "Scanning" are bare text.
- Screenshots: [`library-offline-1280x800`](before/library-offline-1280x800.jpg), [`game-off-offline-1280x800`](before/game-off-offline-1280x800.jpg), [`home-error-1280x800`](before/home-error-1280x800.jpg), [`cloud-error-1280x800`](before/cloud-error-1280x800.jpg), [`library-scanning-1280x800`](before/library-scanning-1280x800.jpg)
- **Owner's read:** refines "states are bare text". Some states don't exist at all.

### 9. [P2] The Library has no player filter and two unexplained ownership signals
- Ownership has two encodings, neither explained: a cover badge for who's loaded, and 7px dots for who has a profile.
- The filters are eight store chips of equal weight. "With profiles" is the only household one, and there is no "Mihai's games".
- The store label under every title is noise.
- With 400 games you see 10 covers per screen at 1280×800, and nothing helps you jump.
- Screenshots: [`library-1280x800`](before/library-1280x800.jpg), [`library-400-1280x800`](before/library-400-1280x800.jpg), [`library-2560x1440`](before/library-2560x1440.jpg)
- **Owner's read:** confirmed, except the dots are 7px, not 4px. Adds the second signal.

### 10. [P2] Player colours collide with the system gold, and colour is the only cue
- Dan's ochre "Play as Dan" reads as the primary action.
- Dan and Radu (ochre and terracotta) are hard to tell apart.
- Gold also marks every toggle and the news dots, so "the one thing to do" isn't one thing.
- Without a picture, a player is a colour and an initial.
- Screenshots: [`game-on-6players-1280x800`](before/game-on-6players-1280x800.jpg), [`picker-6players-1280x800`](before/picker-6players-1280x800.jpg), [`players-6-1280x800`](before/players-6-1280x800.jpg)
- **Owner's read:** new.

## Also found (below the top 10)

- **Players page:**
  - stats clip ("1 finish", "0 finis");
  - "No games played yet" appears next to "176 h played";
  - **Save players** looks disabled;
  - the × that removes a person has no label ([`players-6-1280x800`](before/players-6-1280x800.jpg)).
- **First run** asks for names and colours but never says what a profile or a swap is ([`first-run-1280x800`](before/first-run-1280x800.jpg)).
- **Cloud setup** has two equal gold primaries, Use OneDrive and Use Google Drive ([`cloud-setup-1280x800`](before/cloud-setup-1280x800.jpg)).
- **Settings:**
  - nine toggles of the same weight with no grouping;
  - one description runs about 141 characters a line ([`settings-1920x1080`](before/settings-1920x1080.jpg)).
- **"Achievements: soon"** is a dead nav item on every page.
- **Text stays 13px at 2560×1440** ([`library-2560x1440`](before/library-2560x1440.jpg)).
- **Continue playing** metadata truncates at 1280 and at 150%.

## Your read, checked

| Claim | Verdict |
|---|---|
| Game page: empty hero, poster + logo + Play floating | Confirmed (#5) |
| Logo twice | Confirmed. The CS2 logo is also near-illegible (#5) |
| Metadata is an admin key/value table | Confirmed (#1, #6) |
| Right column looks like a settings page, equal weight | Confirmed (#6) |
| "Play as A / Play as B (loaded)" as a button group | Confirmed (#1) |
| "Loaded now" in row 5 | Confirmed; it's row 6 |
| Library generic; ownership is 4px dots | Generic: confirmed. The dots are 7px, and there's a second, cover-badge encoding (#9) |
| Eight equal filter pills; store labels are noise | Confirmed (#9) |
| Home is the streaming template | Confirmed. The hero pick is the worst part (#3) |
| One gray for all secondary text | Two grays; the dim one fails AA (#4) |
| Same radius and bordered rows everywhere, flat tonal steps | Confirmed: panels are one small step above the background. There are three radius tokens plus a hardcoded 12px. |
| "Looking for its saves…" bare text | Confirmed. Some states (offline) don't exist at all (#8) |

## Detector findings (for the record)

- **Static scan.**
  - Four `side-tab` stripes in `app.css`: `.newer`, `.banner` and `.toast` at lines 388, 506 and 649, plus the blockquote border at 418.
  - One `marquee` on `.bar::after`. It is the loading bar, a false positive.
  - The scan sees little, because `app.js` renders everything at runtime.
- **In-page scans (CDP and overlay).**
  - **Contrast:** almost every finding is `--dim` (#4).
  - **`buried-raster`:** 382 on the 400-game Library and 2 on the 13-game one. These are false positives: lazy covers waiting at `opacity: 0`, and empty `<img>` elements on the no-art games.
  - **`edge-flush-cards`:** one, on the Continue playing scroller. Probably intentional.
  - **`line-length`:** one, in Settings.
  - **`straggler`:** two, minor.
- **Harness note.** The app's CSP has no `'wasm-unsafe-eval'`, so the overlay's WebAssembly core is blocked in the real window. The CDP scan works with the CSP in place.
