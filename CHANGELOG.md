# Changelog

## 1.7.0

- New setting, off by default: **Show game updates and news**. A game's page then lists its latest updates and announcements from Steam's news feed, with patch notes marked; click one to read it on Steam. For a game no store keeps up to date (added by hand, or found in a game folder), Lootnest also says when an update came out after your copy's files. Only the game's Steam id, or its name, is sent; each feed is read at most every 6 hours.

## 1.6.0

- A new look. Deep blue-graphite instead of brown, game titles set in Bricolage Grotesque (built in, nothing is downloaded), and each player's colour used for their own Play button, the same as on the "Who's playing?" screen.
- The Library opens on the game played last, in Steam's wide artwork for it, with **Play as** buttons for each player, or **Play** for games without profiles.
- Every game has its own page instead of a side panel: its artwork and cover, a Play button per player, what Lootnest knows about it (store, who has a profile, whose files are loaded, Steam Cloud, when it was last played) and everything you can do with it. The cover moves from the Library into the page when you open it, and back when you return (turned off with Windows' animation effects). Esc goes back.
- Cards in the Library show a dot for each player with a profile, and a round Play button on the cover.

## 1.5.3

- Every game has a picture now. A game without a cover shows its own icon, taken from its .exe on your PC (nothing goes online), over a soft wash of the icon's colours.
- Covers from the Steam store (with **Download covers for games outside Steam** on) are found for more games: when the full name isn't on the store, Lootnest tries it without its last word or two, such as "Deluxe Edition".

## 1.5.2

- **Add game** now puts the game in your Library, ready to play, instead of setting it up with profiles. Turn on profiles from its panel when more than one person plays it. A game you add stays in the Library even if a scan doesn't find it; **Remove from the Library** takes it out.
- Fixed: a game added by its .exe was named after the folder the exe is in, such as "steam", "bin" or "Win64". It's now named after the game's own folder, and folder names like `Game.Name-Tag` become "Game Name".

## 1.5.1

- Fixed: a game outside Steam showed as "Other" once its profiles were on (GTA V from Epic, for example). It keeps the store it was found in, in its panel and in the Library's filters.
- Every game in the Library has settings now, profiles or not: **Game settings** (the name it's shown with, and launch options Lootnest adds when it starts the game), **Create desktop shortcut** (starts the game directly), **Open the game's folder**, and for Steam games **Check the game's files in Steam**. Names and launch options carry over when profiles are turned on.
- Fixed: dialogs and the game panel flickered (played their opening animation again) each time something in them changed, such as picking a drive in Game Mover or the progress bar moving.

## 1.5.0

- New **Game Mover**: a Steam game's panel has **Move to another drive**. Pick one of the PC's Steam libraries; Lootnest shows the game's size and each drive's free space, closes Steam, copies the game (with its workshop items) and checks every file, and only then moves Steam's records and removes the old copy. If anything goes wrong first, the game stays where it was. Steam reopens when it's done.
- The Library is saved: when Lootnest opens, your games are there at once, and the scan updates them quietly in the background. A game the scan misses stays while it's still installed; uninstalled games drop out.
- Fixed: for Steam games without a preset, Lootnest proposed swapping Steam's own folder for the game (`Steam\userdata\<account>\<appid>`), which also swapped Steam Cloud's records. It now proposes the folders inside it (such as `local` and `remote`) and refuses the folder itself. Games already set up that way are switched over automatically at start; each player's files are moved to the new entries, nothing is deleted.

## 1.4.1

- Steam Cloud can stay on. Each game's panel says whether Steam Cloud syncs the files its profiles swap. When it does, Lootnest checks them every time the game starts; if Steam replaced a player's files with the cloud's copy, Lootnest says so, keeps that player's profile as it was, and saves the session as a backup instead of over the profile.

## 1.4.0

- PlaySwap is now **Lootnest**, with a new name and icon. Installing Lootnest over PlaySwap moves everything over: the data folder (players, games, profiles, backups, log) moves to `%LOCALAPPDATA%\Lootnest`, PlaySwap's Start menu, desktop, "Start with Windows" and Apps & features entries are replaced, and Steam launch options and "(PlaySwap)" game shortcuts keep working until Lootnest switches them to its own exe.
- New **Library**: every game on the PC in one place, as a grid of covers with search and filters. Covers come from Steam's own cache on your PC, so nothing is downloaded. Any game can be played from the library; profiles are now something you turn on per game, not something every game needs.
- A calmer look: flat warm colours with one gold accent, no gradients, a sidebar, a side panel for each game, and short animations that turn off when Windows' animation effects are off. Player colours are softer; existing players move to the matching new colour. New flat gold icon.
- The title bar and window border use Lootnest's own colours, scrollbars are thin and rounded, and the window no longer acts like a web page (no browser right-click menu, no Ctrl+wheel zoom).
- Update from inside Lootnest: **Settings > Check for updates** (or the opt-in daily check) finds a newer release, and **Update now** downloads, verifies and installs it, then reopens Lootnest with everything kept.
- New setting, off by default: **Download covers for games outside Steam**. Epic, GOG and other games get their cover from the Steam store, looked up by name; only the name is sent. The Library offers it when games are missing covers.
- Fixed: the Library showed covers for only a few Steam games. Lootnest now finds the cover in Steam's cache whatever Steam named the file.

## 1.3.0

- New per-game setting for Steam games: "Let Steam start this game". Use it when an anti-cheat such as FACEIT AC blocks the game when PlaySwap starts it. PlaySwap takes itself out of the game's launch options, and its own Play button loads the player's files before asking Steam to start the game.
- When an anti-cheat such as FACEIT AC stops PlaySwap from starting a game, PlaySwap opens with a "Fix it and start the game" button instead of showing an error. It switches the game to "Let Steam start this game", restarts Steam, and starts the game for the chosen player.
- Turning on "Let Steam start this game" puts a "<game> (PlaySwap)" shortcut on the desktop that asks who's playing every time. If the game is started from Steam's Play button instead, a notification says whose settings are loaded.
- Add game fills in the save locations PlaySwap finds, as the "Found on this PC" list already did. When it finds the game's process and at least one location, the game is added in one click.
- "Found on this PC" no longer lists apps and launchers as games: browsers, Discord, Battle.net, Ubisoft Connect, CurseForge, FACEIT, Logitech G HUB and other apps built on Chromium, Electron or CEF, plus launcher, installer and version-numbered folders.
- Fixed: on a PC with more than one Steam account, PlaySwap could check the wrong account's launch options right after Windows started (before Steam signed in), and asked to restart Steam on every boot. It now uses the account Steam signs in to (from `loginusers.vdf`).
- Fixed: after a game failed to start from Steam, its row stayed on "Starting" for up to 4 minutes and Edit game couldn't be opened.
- Fixed: a game's ⋯ menu was cut off at the edge of its row, so only "Edit game" showed and it couldn't be clicked when there was one game.

## 1.2.0

- Scans the PC for games: Steam, Epic, GOG, game folders on every drive, and desktop and Start menu shortcuts. Results appear under "Found on this PC" with Add, Add all and Hide.
- Sets up Steam automatically: writes PlaySwap's launch option for managed Steam games (closing and reopening Steam once if it's running), backs up `localconfig.vdf`, and restores each game's original options when it's removed or PlaySwap is uninstalled.
- New setting: "Set up Steam automatically".

## 1.1.0

- Steam's Play button can open the "Who's playing?" picker, through a per-game launch option.
- "Don't start" cancels a launch from Steam.

## 1.0.1

- Fixed a blank white window when the app started.

## 1.0.0

- First release: players, per-player profiles, swap on Play, backups, tray icon, desktop shortcuts, installer.
