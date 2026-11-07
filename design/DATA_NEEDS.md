# Data the directions need

The redesign changes the UI layer only. This page lists what a direction would show that Lootnest's data doesn't have yet, so it can be decided separately. Nothing here is built as part of the redesign.

What already exists, and what the mocks use as it is:

- **Time played per player per game.** `playtime.json`, sent to the page as `playtime` (Library key → player → seconds).
- **Whose files are loaded.** `active` (game → player).
- **Who played a game last, and when.** `recent.json` (`Played{At, Player}`). There is one entry per game, for the last person only.
- **A running session and its start time.** `session.since`.
- **Steam's total for the account.** This is `steamTime`.

## Needed

| What | Used by | Why | Where it could come from |
|---|---|---|---|
| **The last time each player played each game** (built in 1.17.3) | A (the *Last played* column for every row), B (each slot's "played 5 hours ago") | `recent.json` kept only the last person per game. | `RecordPlay` keeps `Players` (player → time) next to `Player`; entries written before it read as their one player. |
| **When each player's save was last written back** (built in 1.17.4, with the profile size) | B ("saved yesterday" in the bay) | The bay shows the slot as an object: what's in it, and how fresh it is. | This is the time of the save-back after a session, recorded where the profile is written. The newest file's modified time in the profile folder is a fallback. |
| **The seat** (who's at the PC, persisting until someone switches) | C | The whole Couch shell is tinted by the seated player, and every Play defaults to them. Today "who's playing" is asked at every launch. | A `Settings.Seat` player id, or none. Set from a Switch control, and cleared by "nobody" or at sign-out. This is an open decision in PRODUCT.md. |
| **A rank order that stays stable** | A | Rows are ranked by hours. Ties and new players with 0 h need a rule. | No new data: rank by seconds, then by name. This is noted so it's decided rather than left to whatever the sort happens to do. |
| **Profile size per player per game** (optional) | B (a slot's detail line) | It helps "keep everyone's saves safe" and shows the backup cost. | A folder size computed when the game page opens, not stored. It's optional: no mock depends on it. |

## Not needed

- No direction needs a new IPC call to start a game as someone. "Play as X" is the existing `play-as`.
- Sliding a slot into the bay (B) is "Load someone's files", which already exists (`load-only`).
