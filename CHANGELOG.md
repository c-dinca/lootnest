# Changelog

## 1.20.0

- **Each profile has its own achievements**, in games whose achievements a Steam-compatible tool keeps on the PC:
  - the achievements folder goes with the profile when you swap, like the saves, so Mihai's unlocks stay Mihai's;
  - the game page shows whose achievements they are, with a button per profile and their count, for example "Cristian 7, Mihai 2".
  - It's on by default. **Share one set** on the game page keeps one set for everyone, as before. Each profile's own stay in their profile, so you can switch back.
- From the next swap, whoever's files are loaded keeps the achievements on the PC now. A copy is made before the swap, as always.
- Steam's own achievements belong to the Steam account, so they stay the same for every profile.

## 1.19.1

- **New brand colours.** Obsidian Black (`#111217`) for the window, Ice Grey (`#DFE5EB`) for text and the main buttons, and touches of Racing Red (`#EF233C`):
  - the gem in the logo and in the app icon;
  - the icon of the page you're on;
  - progress bars, and switches and checkboxes that are on;
  - small marks in the news reader.
- The installer has the same colours.
- Text stays Ice Grey everywhere, so it reads as clearly as before (WCAG AA).

## 1.19.0

- **Achievements on every game's page**, for games from Steam and for games in a folder:
  - how many are unlocked, the latest five with when you got them, and **Show all** for the rest;
  - Steam games read Steam's own library cache on this PC, with names, pictures and how rare each one is;
  - other games read the achievements file a Steam-compatible tool keeps next to the game's saves (Goldberg and GSE, CODEX, RUNE, OnlineFix, RLD, SKIDROW, CreamAPI, EMPRESS), with names from the game's `steam_settings` when it has them.
- Nothing goes online for this: everything is read from this PC. **Show recent activity** in Settings lists where each game's achievements were found.

## 1.18.0

- **Who's at the PC.** Click your name under **Who's here** in the sidebar, or use **Switch**, and Lootnest keeps you in the seat:
  - every **Play** is yours, straight away, without asking "Who's playing?";
  - the game page's big button is **Play as** you, with "or play as…" for anyone else;
  - your name shows **Seated**.
- Choose **Nobody** in **Switch**, or click your name again, to go back to being asked each time. The seat is also cleared when Lootnest quits, so the next person isn't playing as you by mistake.
- Starting a game from Steam's Play button still asks who's playing.

## 1.17.4

- **When each player's profile was last saved, and how big it is.** The game page's **Players** panel shows it under each player, for example "Saved yesterday, 24 MB". It's the time the game last wrote that player's files.
- In **Who's playing?**, a save slot with no play recorded shows when it was last saved instead.

## 1.17.3

- **Lootnest remembers when each player last played each game**, not only who played it last. So Mihai's "yesterday" still shows after Cristian has played since:
  - each save slot in **Who's playing?** says when that player last played, for example "Played 2 days ago";
  - **Who's playing** on Home shows each player's own last game;
  - the game page's **Players** panel shows when each of them last played;
  - player cards and **Played lately** on a player's page list all the games they played, not only the ones they played last.
- Games you played before this version show only their last player until the others play them again.

## 1.17.2

- **Starting a game no longer flashes the desktop.** Lootnest used to hide its window the moment it started a game, so while the game was still loading you saw the desktop, then the game. Now the picker shows **Starting Counter-Strike 2** (or your game) until the game is actually running, and only then does Lootnest step aside (with **Hide while a game runs** on).

## 1.17.1

- **Close the game from Lootnest.** While a game runs, its page shows **Close the game**. Lootnest asks the game to quit, as if you closed its window. Once it closes, the player's progress goes back to their profile as usual. If the game has no window to close, Lootnest tells you to quit it from its own menu.
- While a game runs, its play bar shows the player's colour instead of a spinner.

## 1.17.0

The new look is complete, ending with the moment Lootnest is about: **who's playing**.
- **The "Who's playing?" picker** sits over the game's colours. Each player is a save slot:
  - their picture or initial, their name, and whether they have a profile;
  - a label in their colour, with the key to press: 1, 2, 3…
  - The slot of whoever's files are in the game says so.
- **When you choose someone**, the save that was in slides out and theirs slides in. Under it: "Loading Mihai's files", and "A copy of Cristian's files was made first."
- **Opening a game** from the Library or Home cross-fades quickly instead of sliding a cover across.
- With Windows' animation effects off, nothing slides. Things only fade.

## 1.16.8

The rest in the new look: **Add game**, the first-run **Who shares this PC?**, the profile editor, confirmations and the right-click menu.
- Dialogs are frosted glass over a dimmed window.
- Buttons, text fields, menus and tabs match the rest of Lootnest.
- Steam game choices and options use small corners and hairline edges.
- In the first-run dialog, each player's colours stay on one line.

## 1.16.7

Settings in the new look:
- **Groups:** General, Games, Online, Updates, Cloud saves and backup, Data, About. They sit side by side on wide windows.
- **Rows:** each says what it does, with its switch, number or menu on the right.
- **Online** groups everything that goes online. Each item there stays off until you turn it on.
- **Updates** shows your version, with Check for updates or Update next to it.

## 1.16.6

**Cloud saves in the new look:**
- It uses the whole window, in two columns:
  - **Games** on the left: each player's state, with a coloured dot (up to date, newer elsewhere, changed on two PCs) and which PC the saves came from.
  - **Where** and **Backups of everything** on the right.
- Choosing a folder is now a list: OneDrive, Google Drive, Dropbox, iCloud Drive, or a USB drive or another folder.
- Switches and setting rows are redone, on Settings too.
- The pages that aren't redone yet (Settings, dialogs, the profile editor) now share the new colours and type. The gold is gone.

## 1.16.5

**Players in the new look:**
- **A card for each person:**
  - their banner;
  - what they played last, or the game they've played most;
  - the covers of the games they played lately;
  - their hours, games, and games with a profile.
- **Names and colours** is below the cards. Each row has the player's avatar, their name and their colours.
- **A player's own page** takes their colour:
  - Play is in their colour;
  - the activity map and the bars use it too;
  - the numbers are larger and easier to scan.

## 1.16.4

- **Shelf is gone.** Playing / Up next / Finished / Stopped, the ratings and the notes are removed from game pages and player pages. What you had saved stays in Lootnest's data folder, untouched.
- On a game page, **Players** shows each player's time, when they last played, and whose files are loaded.
- On a player's page, **Numbers** shows the games with their profile instead of "finished". The players list shows games played and games with a profile.
- **Fixes:**
  - Favourite games on a player's page show as a cover grid again.
  - **Played lately** lines up with the rest of the page.

## 1.16.3

Fuller, wider pages:
- **Full width.** Home, the Library and game pages now use the whole window. There's no empty band on the right on wide screens.
- **Game page**
  - **Updates and news** is a section of its own under the play bar. The newest post is large, with its picture and summary, and the others are listed with pictures underneath.
  - **Players** sits beside the news. For each player it shows:
    - how long they've played and when they last played;
    - whose files are loaded;
    - where the game stands for them, their rating and their note.
  - **Manage** moves below, across the page: Profiles, Game files, Shortcuts, Danger.
- **Home:** News from your games is now a full section under Continue playing, in the same style.
- **Who's playing** shows the game a player has played most when Lootnest doesn't know which one they played last.
- The "Update" tag lines up with the date, and Steam's summaries no longer show stray backslashes ("\[ GAMEPLAY ]").
- On smaller windows, the columns stack instead of squeezing.

## 1.16.2

The Library in the new look:
- Covers fill the whole window, and the controls take less room.
- Under each title you can see whose files are loaded, in their colour. Who has a profile shows as the colour bars along the bottom of the cover.
- Show **All** or **With profiles**, and pick a player to see only their games: the ones they have a profile in, or played last.
- The store (Steam, Epic, GOG…) is now a menu instead of a row of buttons.
- A game without a cover shows its title on a quiet cover.
- Play appears on a cover when you point at it.

## 1.16.1

Home in the new look. It answers "what are we playing tonight, and who's playing?":
- The game played last fills the top, with the same play bar as its game page: **Play as** whoever's files are loaded, and "or play as…" for everyone else.
- **Continue playing** shows the other games played lately as covers. Each cover shows who played it last, and the colour bars along the bottom show who has a profile.
- **Who's playing** lists everyone with the game they played last, and a Play button that carries on as them.
- News comes last, as a short list.
- Banners (updates, Steam's launch options, covers) are calmer and no longer have a coloured edge.

## 1.16.0

A new look, starting with the game page:
- **The game page** is built around who's playing. The game's art fills the top of the page, and under it a single bar holds everything about playing:
  - **Play as** the person whose files are loaded, in their colour;
  - whose files are in the game right now;
  - everyone else, under "or play as…";
  - how long each player has played.
- **"A copy is made before every swap"** sits right under the bar.
- **Actions are grouped** as Profiles, Game files, Shortcuts and Danger. Steam Cloud, Cloud save, Shelf and news are in a side column.
- **The sidebar** is frosted glass over the game's colours. Players show with their own colour.
- **Toasts** have an icon for done or for a problem, and no coloured edge.
- **New type:** Figtree for the interface, next to Bricolage Grotesque for titles. Both are embedded.
- **Accessibility:**
  - every text passes WCAG AA, including text over the game's art and on any player colour;
  - every control has a visible focus ring;
  - glass turns solid when Windows' transparency effects are off.
- Achievements is no longer in the sidebar until it exists.

The other pages move to the new look in the next updates.

## 1.15.1

- **Choose the save folder** (or a save file) in a game's **Cloud save** section, for games Lootnest can't find the saves of by itself, such as a game you added by hand or one too new for PCGamingWiki's list. Its saves go to the cloud from then on, no profiles needed. **Look it up on PCGamingWiki** opens the game's page there, which says where it keeps them.
- For games whose saves Lootnest found, **Choose them yourself** replaces them with the folders you pick, and you can add or remove folders and files.
- A game's **Cloud save** section now says where everything is: **On this PC** (each save folder, with **Open**) and **In OneDrive / Google Drive / …** (the folder in your cloud with its versions, with **Open folder**), so you can find them yourself, in Explorer or on the cloud's website.
- **Back up now** no longer makes a second, identical version when the saves haven't changed; it says so instead.

## 1.15.0

- **Cloud saves for games outside Steam**, like a launcher's cloud saves:
  - every Library game that isn't on Steam and has no profiles gets a **Cloud save** section on its page;
  - a new version of its saves goes to your cloud folder whenever they changed and the game is closed;
  - the last 10 versions are kept, with when and on which PC each was made;
  - **Restore** any of them, on this PC or another; the saves here are kept in Lootnest's backups first;
  - **Back up now** makes one at any time.
- Steam games keep using Steam Cloud. Games with profiles keep syncing each player's own saves.
- **Find save locations online** (Cloud saves page, off until you turn it on): Lootnest downloads PCGamingWiki's list of where over 20,000 games keep their saves, about once a week, and uses it to find them without asking. Without it, folders named like the game are used.

## 1.14.1

- Fixed: a new version could go unnoticed for hours. With **Check for updates automatically** on, Lootnest now checks every hour instead of every 6, and again whenever you open its window, so a new version shows within minutes.
- A new version now shows as **Update to …** in the sidebar on every page, not only on Home, and a short message says when one is found.

## 1.14.0

- **Cloud saves**, a new page under **Library**: every player's saves for every game, in your own cloud.
  - Choose OneDrive, Google Drive, Dropbox or iCloud Drive (Lootnest finds the ones installed on your PC, or tells you where to get them), or a USB drive or any folder.
  - Saves go up after each session. On another PC with the same cloud folder, the newest come down before the game starts, like Steam Cloud, for every game and every launcher.
  - If a player's saves changed on two PCs since they last synced, Lootnest asks which to keep; the other copy is kept in that game's backups.
  - Players are matched across PCs by name, and games by their Steam game, launcher link or name, so a game added on each PC on its own still syncs.
  - The page shows every game and player: up to date, changed here, newer on another PC, or changed on both, with **Sync now**.
- The password now belongs to the cloud folder: set it once, and enter the same one on your other PCs. Saves and backups in the folder are encrypted (AES-256); a PC without the password can't read them.
- Backups of everything (and **Restore**) moved to the Cloud saves page; Settings links to it.
- Signing in to Microsoft or Google right in Lootnest, without their apps, comes in a later version.

## 1.13.0

- **Export profile** on a player's page: one file with their saves and settings in every game they have a profile for, their picture and banner, shelf and play time. Take it to another PC or a friend's, optionally with a password (AES-256).
- **Import a profile** on **Players** adds that player, or updates them if they're already here. Their games are matched to the ones on this PC (same Steam game, launcher link, exe or name), and games this PC doesn't have yet are added. Everything here is saved first, so an import can be undone like a restore.
- A profile file and a full backup can't be mixed up: **Restore from a backup** and **Import a profile** each say what a file is and do the right thing with it.

## 1.12.1

- Fixed: the Battle.net app itself showed in the Library as a game, with another game's cover (Mega Man Battle Network, because "Battle.net" is inside that name). Launchers and store apps (Battle.net, Ubisoft Connect, EA app, Riot Client, Rockstar Games Launcher and others) are never taken for games, and one an earlier version added leaves the Library at the next scan.
- Covers looked up by name now need the names to match from the start ("Game" and "Game Deluxe Edition"), not just one name appearing inside the other.
- A program whose install folder is a whole drive or Program Files is never taken for one game.

## 1.12.0

- The Library now finds games from every launcher on your PC:
  - Xbox and PC Game Pass;
  - EA app;
  - Ubisoft Connect;
  - Battle.net;
  - Riot Games;
  - Amazon Games;
  - Rockstar Games;
  - itch.io.

  Each store with games gets its own filter.
- Games a launcher must start (Ubisoft Connect, Battle.net, Xbox, Amazon Games, and now Epic too) start through that launcher, the same as pressing Play there, so sign-in, the overlay and cloud saves keep working. In **Edit game** you can still choose to start a game from its .exe.
- Lootnest only opens the launch links of those launchers, nothing else, and only reads what each launcher already keeps on the PC; nothing goes online.

## 1.11.0

- **Backup** in Settings: a copy of every player's saves and profiles, your games, profile pictures, shelf and play time, in a folder you choose. Lootnest finds OneDrive, Google Drive, Dropbox and iCloud Drive folders on your PC, so it's one click; any other folder (a USB drive, a NAS) works too. Lootnest only writes the file there; your cloud app uploads it.
- Backups are made after every game session (and when Lootnest starts, if the last one is over a day old), or with **Back up now**. Each PC keeps its own last 5 (you choose how many), so two PCs can share one folder.
- **Set a password** to encrypt backups (AES-256). Lootnest remembers it for your Windows account so backups keep being made by themselves; you need it to restore on another PC. If you forget it, nobody can open those backups.
- **Restore from a backup**, on a new PC or to go back: Lootnest shows where and when it was made and who's in it, saves everything here first (in `backups\before-restore`, so a restore can be undone), then puts the backup's players, games and saves in place. This PC's settings stay. Before each game's next swap, Lootnest asks whose files are on this PC, so nothing here is saved over someone's profile by mistake.

## 1.10.0

- **Home** is its own page now, and where Lootnest opens: the game you played last, **Continue playing**, **News from your games**. The **Library** is just your games: search, filters, sort and cover size.
- The window's colours change when you choose a game, not as the pointer passes over covers: Home takes the game played last, a game's page that game, and the Library the game you opened last.
- A game's page goes back to where you opened it from (Home or Library), scrolled where you were.
- **Player profiles.** Click a player (in the sidebar or on **Players**) for their page:
  - a banner (their colour, their own picture or a game's art), their picture (or a game's cover), name, a line and a few words about them;
  - **Currently playing**: a game they pick, with a note;
  - their numbers: time played, games played, finished, days in a row, longest session, most played;
  - **Favourite games** (up to six, in their order), **Activity** (a square a day for the last six months, the last 7 and 30 days, top games), **Shelf**, **Collection** (where their games come from) and **Played lately**;
  - a **Play** button for the last game they played.
- **Edit profile** has four parts: **Look** (picture, colour, any colour, banner and which part of it shows), **About**, **Showcase** (favourites, in order) and **Page** (which parts show, and in what order).
- **Shelf** on every game's page: for each player, where the game stands (**Playing**, **Up next**, **Finished**, **Stopped**), a rating out of five and a note.
- Everything stays on this PC: pictures in `avatars\` and `banners\`, the shelf in `shelf.json`, time per day in `activity.json` (counted from now on, for games started from Lootnest).

## 1.9.2

Security hardening, with nothing to change on your side:
- Built with Go 1.27.3. The previous Go (1.24.7) had known vulnerabilities in its URL parsing and HTTPS certificate checks; `govulncheck` now reports none, and every build runs it.
- Lootnest's window only ever shows its own page: every request it makes goes through an allowlist (its page, and pictures on Steam's image servers), so nothing can load from, send to or navigate to anywhere else.
- A Content-Security-Policy with a per-start nonce: only Lootnest's own script runs in the window.
- Updates install only from this repository's releases, and only with a SHA-256 to check the installer against.
- The SteamGridDB key is encrypted for your Windows account (DPAPI) instead of kept in plain text; a key saved by 1.8 or 1.9 is encrypted at the next start.
- New SECURITY.md describes all of this and how to report a problem privately.

## 1.9.1

- Lootnest can keep itself up to date. It asks once, **Keep Lootnest up to date?**; with yes, it checks every 6 hours, shows a Windows notification when a new version is out, and installs it by itself while it's in the tray and no game is running. Both are in Settings (**Check for updates automatically**, **Install updates by themselves**).
- Fixed: the Library's hero and shelves stayed empty until a game had been started from Lootnest. They now also use what Steam remembers on this PC: when each Steam game was last played and how many hours it has on the account ("412 h on Steam"), shown on the hero, the shelves and the game's page, and used by **Most played**.
- Fixed: on a big or maximised window, a game's page and Settings stayed in the left half. Pages now grow with the window: a game's page uses the whole width (centred on very wide screens), Settings and Players are a centred column, and covers, shelves and titles get bigger on large screens.

## 1.9.0

- The Library has shelves now, like the best launchers: under the game you played last, **Continue playing** shows the other recent games in their wide artwork, and **News from your games** the latest posts from them (with **Show game updates and news** on).
- The window takes on the colours of the game in focus: the art of the game you point at, blurred, behind everything; on a game's page, that game's.
- Time played, per player: Lootnest counts how long each person plays each game with profiles, and the game's page shows it as a bar in the players' colours. The Library's hero shows the total.
- Sort the Library by name, recently played or most played, and choose small, medium or large covers. Both are remembered.
- Right-click any game for a menu: play as each player, open its page, settings, its folder, Steam's file check, hide or turn off profiles. Arrow keys and Esc work in it.
- Ctrl+F (or /) jumps to the search box from anywhere.
- Players in the sidebar show as round initials in their colour; whoever is playing gets a ring and the game's name.

## 1.8.0

- **Download covers and artwork** (was "Download covers for games outside Steam") now also gets each game's wide artwork and logo, so games outside Steam get the same page and Library hero as Steam games. Steam games whose art Steam hasn't kept on this PC get it too.
- Optional **SteamGridDB API key** in Settings: games the Steam store doesn't know get their cover, artwork and logo from SteamGridDB. The key is free and is sent only to SteamGridDB.
- The Library asks once, clearly, when games have no cover: **Find covers**.

## 1.7.1

- Updates and news open inside Lootnest: click one on a game's page to read the whole post, with its pictures, headings and lists. Links in it open in your browser; **Open on Steam** shows the original.
- Fixed: news summaries showed `{STEAM_CLAN_LOC_IMAGE}/…` instead of text when a post started with a picture. Posts with a picture now show it as a small thumbnail in the list.

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
