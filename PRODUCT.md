# Lootnest

## What it is
A Windows game manager for a household that shares one PC (and often one Steam account). It gathers every game on the PC, from Steam, Epic, GOG, Xbox, EA, Ubisoft, Battle.net, Riot, Amazon, Rockstar and itch.io, plus game folders and shortcuts, into one Library.

## The mechanism nobody else has
Per-player profiles. Each person in the household gets their own settings and saves for the same game. Before a game starts, Lootnest swaps the right person's files into place, and when the game closes it saves their progress back to their profile. A backup is made before every swap.

"Who's playing" and "whose files are loaded right now" are the product's identity. Players show up everywhere in their own colour (sidebar, Play, loaded state, shelf, cloud saves). Someone should recognise Lootnest by how players appear, not by its logo.

## Who uses it, and when
- **The household.** Siblings, partners, parents and kids sharing one gaming PC, at home, in the evening, often from a couch or desk with the window maximised.
  - One person usually sets it up: adds players, turns on profiles per game, chooses a cloud folder.
  - Everyone else mostly picks a game and says who's playing.
- **Library size.** From a handful of games to hundreds, across several launchers.

## Jobs
1. Play this game, as me, with my own save. This is the most frequent job, and the one the product exists for.
2. See whose files are in a game right now, and switch them without losing anyone's progress.
3. Find a game among everything installed, from any launcher.
4. Keep everyone's saves safe: backups, cloud saves in the user's own OneDrive or Google Drive folder, and restore on another PC.
5. Manage the household: players with a picture, colour, shelf and stats.

## Platform
- Native Windows desktop app: Go, with a WebView2 window rendering a local page (`internal/app/ui`).
- Window sizes from 1280×720 to 2560×1440, at 100% to 150% Windows scaling.
- The window is dark today.

## Constraints that stay
- **Never lose user data.** Every swap, restore and import keeps a copy first. The UI must make that reassurance visible at high-stakes moments.
- **No network without opt-in.** Every online feature is off until turned on, and listed in the README's Privacy table.
- **Nothing piracy-specific.**
- **Accessibility.** WCAG AA contrast, visible focus, full keyboard use, and respect for reduced motion.
- **Fonts.** Must be licensed for bundling in a desktop app (e.g. SIL OFL). They're embedded in the exe; nothing loads from the internet.
- **Copy voice.** Plain, specific and calm: "Load someone's files", "A copy is made before every swap". No hype, no filler, no emoji.

## Out of scope for the redesign
Swap logic, file handling, the Go/IPC API and data models. The redesign is the UI layer only; data a design needs but doesn't exist yet goes in `design/DATA_NEEDS.md`.

## Open decisions
- **The seat.** Decided in 1.18.0: a seat that stays until someone switches, or until Lootnest quits. Every Play is the seated player's, and the picker is the exception. With nobody seated, Play asks every time, as before.
- **Achievements.** In the nav as "soon": show it with intent, or hide it until it exists?
