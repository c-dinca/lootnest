# Using Lootnest

The README is the short version. This page has the details: how a swap works, what profiles can't separate, adding a game by hand, where Lootnest keeps its data, Playnite, and installing without a window.

## How profiles work

![The picker that opens before a game starts](screenshots/picker.jpg)

1. You start a game from the Library, from Steam, or from a Lootnest desktop shortcut.
2. Lootnest asks who's playing.
3. It saves the files currently on the PC back to whoever they belong to, backs them up, and copies in the chosen player's files.
4. The game starts. Lootnest watches its process.
5. When the game closes, Lootnest saves the files to that player's profile.

For Steam's Play button, Lootnest sets the game's launch options to `"…\Lootnest.exe" --steam <app id> -- %command%`. It edits Steam's `localconfig.vdf` only while Steam is closed, changes only `LaunchOptions`, and keeps a backup. Options you already had are kept, and they're put back when you turn profiles off or uninstall Lootnest.

### Anti-cheats (FACEIT AC)

Some anti-cheats only let Steam start the game, so with FACEIT AC running, CS2 can't start through Lootnest's launch option. When that happens, Lootnest offers **Fix it and start the game**. It switches the game to **Let Steam start this game** (also in **Edit game**), takes itself out of the launch options, and starts the game for the chosen player through Steam.

From then on, start the game from Lootnest or from the **<game> (Lootnest)** desktop shortcut, and both ask who's playing. Steam's own Play button can't ask for these games, so if you use it anyway, a notification says whose settings are loaded.

### What profiles can't separate

Lootnest swaps files on your PC. Anything that belongs to the Steam account stays shared: achievements, playtime, friends, inventory and online ranks. So do settings a game keeps in the Windows registry or on its own servers. If everyone can have their own Steam account, that separates everything; profiles are for when you share one.

## Adding a game by hand

If the scan misses a game, use **Add game** and choose the Steam game or the .exe. Lootnest fills in the save locations it finds: with **Find save locations online** on, first the ones PCGamingWiki lists for the game, then folders named like it. Add any others yourself. [PCGamingWiki](https://www.pcgamingwiki.com) lists them for most PC games, in the "Game data" section. Paths can use `%APPDATA%`, `%LOCALAPPDATA%`, `{LOCALLOW}`, `{DOCUMENTS}`, `{SAVEDGAMES}`, `{STEAM}` and `{STEAMUSER}`.

Data lives in `%LOCALAPPDATA%\Lootnest`:
- `profiles\<player>\<game>`
- `backups\<game>\<date-time>`
- `covers`
- `avatars` and `banners` (pictures for player profiles)
- `shelf.json` and `activity.json` (each player's shelf and time played per day)
- `playnite.json` (the time played each import from Playnite brought in, so it's added once)
- `backup.json` (when the last backup to your folder was made)
- `sync.json` (what each player's cloud saves were at the last sync)
- `gamesaves.json` and `savedb.json.gz` (cloud saves of games outside Steam, and the save locations list)
- `safety-net\` and `safetynet.json` (copies of saves games keep inside their own folder, the last 5 per game)
- `config.json`
- `lootnest.log`

## Lootnest and Playnite

Lootnest works alongside [Playnite](https://playnite.link); it doesn't replace it. **Import from Playnite**, in Settings, Data and activity, brings in what Playnite knows about the games on this PC: time played and when last played, statuses (Playing, Beaten, Completed, Abandoned, Plan to Play, On Hold), favourites, ratings (Playnite's score as 1 to 5 stars), notes, and, if you choose, categories and tags as collections. The import is one way: Lootnest reads Playnite's library (`%APPDATA%\Playnite\library`, or the `library` folder of a portable Playnite) and never writes to it. Close Playnite first, because it keeps its library to itself while it's open.

Games are matched by their store and id (Steam, Epic, GOG), then by install folder, then by name. Playnite's games that aren't on this PC aren't added. Statuses, ratings, notes and time played go to the profile you choose. A status, rating or note only comes in for a game that has none; time played is added to what Lootnest counts, and importing again adds only what's new. Lootnest copies its own records to `backups\import` first, and **Undo** puts them back.

## Playing on a TV

Lootnest has a TV mode for streaming the PC to a TV: Big Picture on the whole screen, made to read from the sofa and to work with a controller alone. Who's playing comes first when more than one person uses the PC, then your games; Play loads that profile's files and starts the game, and Lootnest comes back when the game closes. Start (Menu on an Xbox pad) opens a menu with Who's playing and **Exit TV mode**.

To set it up, open **Settings, TV** and choose **Set up**. Lootnest downloads Apollo, a free, open-source streaming engine, from its own releases on GitHub, checks it, and installs it; Windows asks once. Lootnest then signs it in with a password of its own, keeps its settings pages to this PC and its firewall rules to your home network, and adds TV mode as the one thing it streams. You never need to open Apollo.

For an LG TV, choose **Add an LG TV** in Settings, TV. The TV needs developer mode, once: in LG Content Store install Developer Mode, sign in with a free LG developer account, turn on Dev Mode Status (the TV restarts), then turn on Key Server. Lootnest finds the TV, takes the passphrase Key Server shows, installs Lootnest for TV on it and pairs it, with nothing to type on the TV. LG ends developer mode after about 50 hours unless it's extended, so Lootnest extends it every 12 hours while the PC and the TV are on; Settings, TV says if it couldn't. Lootnest for TV is open source too: it's a fork of Moonlight TV, at github.com/c-dinca/lootnest-tv-webos.

A controller can be plugged into the TV or paired with the PC. Over USB, older LG TVs only take the controllers LG lists (Xbox 360 wired, PlayStation 3, Logitech F310, F510 and F710); a DualSense or DualShock 4 needs a USB adapter such as the Mayflash Magic NS 2 or 8BitDo Wireless Adapter 1, or webOS 5 and later for Bluetooth. A controller paired with the PC itself always works, since the game reads it there.

On another TV, install Moonlight, open it and choose this PC (it shows as "Lootnest on" and the PC's name). The TV shows a PIN: type it in **Add a TV** in Settings, TV, with a name for the TV, and choose **Pair**. Then choose Lootnest on the TV: Lootnest opens there in TV mode, starting first if it wasn't running. Exit TV mode ends the stream; ending the stream from Moonlight leaves TV mode, and a game that's running keeps running, with its saves kept as always.

Settings, TV also forgets a TV, ends a stream that's stuck, turns the engine off and on again (Windows asks), and saves a report of the engine's log and Lootnest's to attach to a GitHub issue. To remove Apollo, use Windows Settings, Apps.

With a streaming host you set up yourself (Apollo or Sunshine), add an app whose command is Lootnest with `--tv`, for example `"%LOCALAPPDATA%\Programs\Lootnest\Lootnest.exe" --tv`, and turn off its "wait for all processes" option.

The app the host starts is a second, small Lootnest that only keeps the stream open. The one that loads profiles and starts games runs on its own, so the host ending the stream never stops it in the middle of loading a profile's files.

## Plugins

A plugin adds something Lootnest doesn't come with. Install one in Settings, Plugins, **Install from file**. Most are data only. One with a script is off until you've read what it asks to do and allowed it, and in safe mode (hold Shift while opening Lootnest) none runs. [PLUGINS.md](PLUGINS.md) has the details and how to write one.

### Updates from a plugin

A script plugin you install and allow can offer updates for games no store keeps up to date. Lootnest comes with no such plugin.

- **Updates**, under From plugins, and on the game's page: **Update** downloads beside the game, never over it.
- When the download is the whole game, Lootnest shows what it keeps, copies your saves, and replaces the folder. The old folder stays until you've played or for 14 days, and **Roll back** puts it back.

## Installing without a window

For scripts and package managers, run `start /wait Lootnest-Setup.exe --silent` in Command Prompt, or `Start-Process Lootnest-Setup.exe --silent -Wait` in PowerShell (`/S` and `/silent` work too). It installs for your Windows account with a Start menu shortcut; on an upgrade the desktop shortcut and Start with Windows stay as they are, and a first install adds neither. It doesn't open Lootnest afterwards. It exits with 0 when Lootnest is installed and 1 when it couldn't be, and then writes the reason to `Lootnest-Setup.log` in your `%TEMP%` folder.

## Updating and uninstalling

To update, use **Settings > Check for updates**. Your profiles, games and saves are kept. To uninstall, use Windows **Settings > Apps > Installed apps**.

## PlaySwap

Lootnest was called PlaySwap until version 1.3. Installing Lootnest over PlaySwap carries over players, games, profiles, backups and the Steam set-up.
