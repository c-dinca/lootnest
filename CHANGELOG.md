# Changelog

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
