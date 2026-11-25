# Testing Lootnest on a real PC

About 20 minutes, most important first. CI runs the swap engine, the note
over a game, hot keys and the firewall rules on Windows already
(`.github/workflows/build.yml`, job `windows`). This list covers what only a
real PC with real games shows.

Before you start: Settings, Data and activity, **Show recent activity**.
After each step that doesn't do what it says, copy the log from there (or
use Settings, About, **Report a problem**) and say which step it was.

Use a game you don't mind, with saves you can lose, for steps 1 and 2. A
copy is made before every swap, but test as if it weren't.

## 1. Profiles: the swap (the heart of Lootnest)

1. Turn on profiles for a game (Library, the game, **Turn on profiles**).
   Choose where it keeps its saves if Lootnest asks.
2. Play as profile A. Change a setting you can see (sensitivity, a
   keybind), save, quit.
3. Play as profile B, starting fresh. The setting is back to the default.
   Change it to something else, save, quit.
4. Play as A again. A's setting is there.
5. Play as B again. B's setting is there.
6. Manage, **Open backups**: there's a copy from each swap.

If any step shows someone else's settings or saves, stop and send the log.

## 2. A game from outside a store

1. Add a game in a folder (Library, **Add game**), or use one Lootnest
   found (Hydra, a folder).
2. Its page, **Achievements**: they show, or **Check why** says which step
   is missing.
3. Make a save point (its page, **Save points**, **Make a save point**).
4. Play, make progress, save, quit.
5. **Go back to it** on the save point. The progress is gone and a point
   "Before going back to ..." is on the list.
6. Go back to that one. The progress is back.

## 3. While a game runs

1. Settings, Games and swaps, **Show one**. A note shows at the top right
   of Lootnest's window, and another at the bottom right of the screen.
   If the second doesn't, note it and send the log.
2. Start a game in borderless or windowed mode. Press **Ctrl+Shift+F5**. A
   note "Save point made" shows over the game.
3. If the game has achievements a tool keeps (step 2), unlock one. A note
   shows over the game.
4. While the game runs, the unlock (and a save point made with
   Ctrl+Shift+F5) also plays a short sound. If it stays silent, note it
   and send the log. Settings, Games and swaps turns these sounds off.
5. With Lootnest's window hidden behind the game, the unlock and the
   Ctrl+Shift+F5 save point are each heard once over the game, and not at
   all with those sounds off.

## 4. Updates (games in a folder only)

1. Updates: the game shows with its version, or "can't tell".
2. Its **More** menu, **Copy its saves first**. A message says the copy is
   in Lootnest's backups.
3. Only with an update you already have for it: **Apply an update**, look at
   what it says it changes, apply, start the game, then **Undo**.

## 5. Keep a game offline

1. A game in a folder, Manage, **Keep it offline**. Windows asks as
   administrator. Say yes.
2. Its page says **Kept offline**. Windows' own firewall settings
   (Windows Defender Firewall, Advanced settings, Outbound Rules) show rules
   named "Lootnest keeps ... offline".
3. Manage, **Let it go online again**. The rules are gone.

## 6. Big Picture

1. Press **F11**, or **Start** on a controller. Lootnest fills the screen.
2. Move with the stick and the D-pad, choose with A, go back with B, switch
   tabs with LB and RB.
3. Choose a game, **Play**. It starts as whoever's sitting down.
4. Press **Start** or F11. The window is back as it was.
5. With a DualSense and a Switch Pro controller (1.68): each shows its own
   buttons at the bottom (cross, circle, square, triangle; or A on the
   right, B at the bottom, + and minus), and the PS or Home button opens
   and leaves Big Picture. With DS4Windows or Steam Input running, one
   press moves or chooses once, not twice.

## 7. The rest, quickly

- The sidebar: filter the games, fold it (Ctrl+B), drag its edge.
- Library: the list view (the last button by the cover sizes).
- Cloud saves, if you use them: **Back up now** finishes.
- Quit from the tray icon, open Lootnest again: everything is as you left
  it.

## 8. Safe mode, and the page's calls (1.65)

Lootnest now only answers calls from its own page, so check nothing lost
its answers:

1. Open Lootnest and visit every page in the top bar, a game's page and
   Settings. Each one fills in as before. If one stays empty, the log says
   "ignored ... call(s) to Lootnest from a document that isn't its own
   page": copy it.
2. Right-click the tray icon, **Restart in safe mode**. Lootnest closes and
   opens again with a Safe mode notice at the top of Home, and the log says
   what's off. No unlock notes, no Ctrl+Shift+F5 while a game runs.
3. Play a game with profiles: the swap and the save back still happen.
4. **Restart normally** on the notice. Lootnest opens again without it.
5. Quit, then hold Shift while you open Lootnest: it starts in safe mode.

## 9. Back and Forward (1.67)

1. Scroll down the Library, open a game, then press the mouse's back button
   (the side button). The Library is back at the same spot. The forward
   button opens the game again.
2. Alt+Left and Alt+Right do the same; so does Backspace when you aren't
   typing in a field. The window never turns blank or reloads.

## 10. Game details (1.69)

1. Open a Steam game's page: with **Game details** off there's only a
   **Show details from Steam** link, and it opens Settings, Online.
2. Turn **Game details** on and go back to the page. The Details block
   fills in: a summary, Developer, Publisher, Released, Genres, Steam
   reviews, and screenshots. Their pictures show (none blocked).
3. Click a screenshot: it opens large. The arrow keys move between them,
   Tab stays inside, Esc closes it.
4. **Manage**, **Edit details**: change the developer, Save. The page shows
   yours. **Use Steam's again** puts Steam's back.
5. Open the page again later: the details show at once, without
   **Checking for newer details**.

## 11. Screenshots and save point pictures (1.70)

1. Start a game in borderless or windowed mode. Press **Ctrl+Shift+F12**.
   A note "Screenshot saved" with the file's name shows over the game.
   The picture is in `Pictures\Lootnest\Screenshots\<the game's name>`,
   named by the date and time, and shows the game, not a black frame or
   another window.
2. Press it twice within a second: the second file ends in " (2)".
3. Switch the game to exclusive fullscreen if it has the option, and press
   it again. Either the picture is right, or the note says "The picture
   came out black" and nothing is saved. Note which, with the game's name.
4. Press **Ctrl+Shift+F5** while the game runs. Close the game and open its
   page: the save point has a small picture of the game, and a click opens
   it large. The Timeline lists the screenshots with their pictures.
5. **Go back to it** on that save point: the game's save folder gets its
   saves back, and no `point.png` or `thumb.png` appears in it.
6. Settings, Games and swaps, turn **Ctrl+Shift+F12 saves a screenshot**
   off. While a game runs, Ctrl+Shift+F12 does nothing in Lootnest and
   reaches the game or other programs again. Settings, General no longer
   lists it.

## 12. Captures from other apps (1.73)

1. In a game, press **Win+Alt+PrtScn** (Game Bar's screenshot) and
   **Win+Alt+R** twice (a short clip). Close the game. Once it's closed
   (or within 10 minutes of the last scan, such as Lootnest starting),
   the Timeline tab of its page lists "Screenshot (Game Bar)" and
   "Clip (Game Bar)" with small pictures, and the Captures line counts
   them. If Game Bar's file names don't carry the game's name, the time of
   the session still matches them; note which.
2. Take a screenshot with Steam's key (F12 by default) in a Steam game: it
   shows as "Screenshot (Steam)". With the NVIDIA app, AMD Adrenalin or
   OBS, record a clip of a game too, if you have one of them.
3. Click a screenshot: it opens large, and the arrows go through the
   game's other pictures. Click a clip: it plays in the app Windows opens
   videos with. A clip shows the picture Explorer shows for it, or a plain
   plate if Windows has none.
4. In Explorer, check that every capture is still where it was, with its
   name and date. Lootnest never moves, renames or deletes one.
5. Delete a capture in Explorer, then start and close a game (or Scan
   again in the Library) after 10 minutes: it's gone from the Timeline.
6. Settings, Data and activity, Captures: the folders Lootnest reads are
   listed. **Choose their games** opens the ones matched to no game;
   choose a game for one, and it's on that game's Timeline, still in its
   own folder. Choose **Not a game** for a desktop capture: it leaves the
   list. Keyboard focus stays in the drawer after each choice.
7. A game with sessions and no captures says on its Timeline that
   Win+Alt+PrtScn and Win+Alt+R make some. Turn Game Bar's captures off in
   Windows (Settings, Gaming, Captures), open the page again: it says
   they're off and where to turn them on.
8. Turn **Captures from other apps** off: the Timeline shows only
   Lootnest's own entries. Start Lootnest in safe mode: no captures are
   read (the recent activity has no "Captures from other apps" line).

## 13. Install, uninstall and remove from Lootnest (1.74)

1. A Steam game in the Library: Manage, **Verify files with Steam**. Steam
   checks its files. Right-click it in the Library: the same entry is
   there. **Uninstall with Steam** opens Steam's own uninstall question;
   say no, and nothing changes.
2. A GOG Galaxy game: Manage, **Open in GOG Galaxy** opens Galaxy on that
   game. An Epic game: **Open your Epic library** opens it. An
   Xbox, EA, Riot, Rockstar or Amazon game: **Uninstall in Windows
   Settings** opens Installed apps.
3. Library, **Not installed**: the Steam games your account has played or
   viewed on this PC, and Epic games, if the Epic launcher is installed,
   that aren't installed. A game installed from another store, or as a
   folder game, isn't listed. Note any that are missing, or any that are
   installed or aren't yours. Open one: **Install with Steam** (or Epic)
   is the main button, and opens the store's install. Install a small
   one: about 90 seconds after the click, and again once the scan runs, it
   leaves Not installed and shows in All, ready to play.
4. A game in a folder of its own (copy a small DRM-free game to
   D:\Games\Test and Scan again), with a save in it. Start it and, while
   it runs, choose Manage, **Remove from this PC**, then the button: it
   says the game is running. Close it and try again. The dialog lists its
   save locations; after Remove, the folder is in the Recycle Bin and its
   saves are in Lootnest's data folder, backups\removed\<game>\<time>,
   with removed.json saying where each came from. A game added by hand
   leaves the Library too.
5. Restore the folder from the Recycle Bin (or copy the game back), Scan
   again: Updates shows it as a new copy, saying Lootnest kept its saves.
   Carry everything over: the kept saves are back in the folder, and the
   folder's own saves, if any, are in backups first.
6. Remove from this PC isn't offered for store games. For a folder that
   holds other games (add D:\Games itself as a game), the dialog says
   Lootnest won't remove it.

## 14. Launch options, and before and after play (1.77)

1. A folder game: Manage, **Launch options**. Type an option the game
   reads (such as `-windowed`), choose another folder in **Start in**,
   Save, then Play: the game gets the option and starts in that folder
   (Process Explorer shows both). **Use the game's folder** puts it back.
2. Turn on **Run as administrator** and Play: Windows asks first, and
   saying no says so in a toast. Say yes: the game runs, its session shows
   on the Timeline when it closes, and its saves are still kept.
3. A Steam game: the drawer has Options, a note that the rest is set in
   Steam, and
   **Open its properties in Steam**, which opens the game's Properties in
   Steam. If it opens something else, or nothing, note your Steam
   version.
4. Before you play: open Discord and Notepad (with unsaved text). Add
   Discord from **Add an open app** (apps in the Windows folder, such as
   Windows 10's Notepad, aren't listed there) and type notepad.exe; turn
   on Force for Discord only. Add a
   program to start before (such as OBS) and one after. Play: a toast says
   Lootnest is closing Discord and Notepad, Notepad
   asks to save and the game starts once you answer, or after 10 seconds
   if you don't; Discord, which goes to the tray, is ended after 10
   seconds. Notepad is never ended. Close the game: the program for after
   starts, and Discord (and Notepad, if it closed) open again. The log
   (Settings, Show recent activity) says each step.
5. The same with a profile game, from Play and from Steam's own Play
   button: the steps run around the swap. Pressing Play twice doesn't
   run them twice.
6. Restart in safe mode and Play: nothing is closed or started, and the
   drawer says so.

## 15. Storage and moving a folder game (1.78)

1. Settings, **Storage**: every fixed drive with its free space and a bar;
   a drive with less than a tenth free says Almost full. The games list
   fills in a little after the scan (it says It's counting now). Click a
   column to sort; the sizes look about right next to Explorer's
   Properties for two games, one from Steam and one in a folder. Games
   with profiles show their profiles' size, and Lootnest's copies count
   the copies before swaps, the safety net and save points.
2. Library, sort by **Largest first**: each cover says about how much it
   takes. A game's page says it under its name.
3. A folder game with a save in it (D:\Games\Test): its row's More
   button, **Move to another drive**. The dialog lists every other drive's
   Games folder with its free space, greys out one without room, and
   **Choose a folder** adds your own. Move: a bar shows the copy. When
   it's done, the toast says where it is, the game plays from the new
   folder, its save is there, its playtime and collections are kept, and
   the old folder is in the Recycle Bin. Updates doesn't offer it as a
   new copy.
4. Start a move and press **Stop the move**: the toast says it stays where
   it was, and the new drive has no folder of it (no .lootnest-part).
5. Start the game, then try to move it: it says to close it first. Keep it
   offline (Manage) and try: it says to let it go online first.
6. The same with a profile game started by its .exe: after the move, Play
   loads the profile into the new folder, and Edit game shows the new
   paths.
7. A Steam game's More has **Move with Game Mover**; an Epic or GOG game
   says its launcher moves it. Restart in safe mode: Storage says sizes
   aren't counted, and Move says to restart normally.

## 16. Import from Playnite (1.80)

1. With Playnite 10 installed and a few games in it (one Steam, one Epic
   or GOG, one in a folder, one not installed), give some a status, a
   score, a note, a category and Favorite, and close Playnite.
2. Settings, **Data and activity**, **Import from Playnite**: the panel
   shows `%APPDATA%\Playnite\library` and a list of the games found on
   this PC, each with what it gets (+12 h, Finished, 4 stars). The game
   that isn't installed is under **Show**, not in the list.
3. Turn switches off and on: the list follows. Choose another profile: the
   list is read again for it.
4. Import: a toast says how many games and hours. The games' pages show
   the time, status, rating and note for that profile, Favourites and
   collections have them. Playnite's `library` folder has the same files,
   dates and sizes as before.
5. Import again: nothing new is listed. Play a game for an hour in
   Playnite, import again: only that hour is added.
6. Press **Undo** on the toast: time, statuses, collections and last
   played are as before the import. `backups\import\<time>` has the
   copies, and `undone` in it what Undo replaced.
7. With Playnite open, import: it says to close Playnite. A portable
   Playnite: **Choose another**, pick its folder, and its games show.
8. First start with Playnite installed: the welcome screen has a quiet line
   about it.

## 17. Your itch.io games (1.81)

Nothing here could be tried against the real API before release: the
parsers were written from itch.io's own Go client and tested against
local servers. Check each step, and send the log if one goes wrong (it
names each request's host and path, never the key).

1. Make a key at https://itch.io/user/settings/api-keys (Settings, Online,
   **Make a key on itch.io** opens that page). Paste it and **Connect**: it
   says Connected as your user name. A wrong key says itch.io didn't
   accept it. `config.json` has `itchKey` starting with `dpapi:`, never
   the key itself.
2. Library, **Not installed**: the games in your itch.io library show with
   their covers, and not the tools, asset packs or books. A game you
   already have installed isn't listed. **Refresh library** in Settings
   reads it again.
3. Install a game that comes as a .zip: the panel shows its download (a
   list when it has more than one, 64-bit first, demos last), its size,
   the folder (Games in your user folder, or the one you chose) and the
   free space there. Install: the bar fills, then Unpacking, then a toast
   with **Open game**. The game has a folder of its own, without the
   archive's wrapping folder, starts with Play, and
   `downloads\itch` in Lootnest's data folder is empty again.
4. Make a folder named like another of your itch.io games in that folder,
   with a file in it, then install that game: it refuses, and nothing in
   the folder changes.
5. Install a game that comes as an installer (an .msi, or an .exe named
   setup or install): the installer runs; finish it. Lootnest scans and adds the game. If the
   installer names the game differently, it says it didn't find it, and
   nothing is recorded. A game that comes as one .exe, with no setup in
   its name, isn't run: it gets a folder of its own and joins the Library.
6. Start a big download and **Cancel the download**: it stops, and nothing
   is left in `downloads\itch`.
7. On a game installed from itch.io, its page says `Version <version>,
   installed from itch.io. Checked ...` (or only Installed from itch.io
   when the developer gave no version), with profiles on or off.
   **Check for an update**. When the developer has uploaded a newer
   build, **Update** downloads it and opens the update panel, which lists
   what it replaces; apply it, then **Undo** puts the old files back and
   the page shows the old version again, with the update on offer. Its
   saves are in `backups\saves-...-before-update`.
   The game is also on the Updates page (1.83), once, saying From itch.io
   with the same version and update; its **Update** does the same, with
   the bar and Cancel in its row. The page's **Check again** asks itch.io
   again at once. Disconnected, the row stays, saying Connect itch.io to
   check, which opens Settings, Online.
8. **Disconnect**: the itch.io games leave Not installed, the installed
   ones stay and start, and `itchKey` is gone from `config.json`.

## 18. Your GOG games (1.82)

Nothing here could be tried against GOG's servers before release: the
parsers follow minigalaxy, which uses the same sign-in and API as GOG
Galaxy, and were tested against local servers. Check each step, and send
the log if one goes wrong (it names each request's host and path, only
the host of a download, and never a token).

1. Settings, Online, GOG: **Sign in to GOG** opens GOG's sign-in page in
   your browser. Sign in; GOG ends on a nearly empty page whose address
   starts with `https://embed.gog.com/on_login_success`. Copy the whole
   address, paste it and choose **Finish signing in**: it says Signed in
   as your user name. Pasting only the code after `code=` works too;
   pasting anything else says it isn't the address GOG showed. A code
   used twice, or after a few minutes, is refused with a plain message.
   `config.json` has `gogToken` starting with `dpapi:`.
2. Library, **Not installed**: your GOG games that run on Windows show,
   with their pictures. A game GOG Galaxy already installed isn't listed.
   **Refresh library** reads it again.
3. Install a game that comes as one file: the panel lists its languages
   (yours first if GOG has it, else English) with the version, the size,
   the folder and the free space. Install: the bar fills, then GOG's own
   installer window shows its progress (Windows may ask first), then a
   toast with **Open game**. The game is in the folder you chose, starts
   with Play, shows as a GOG game, and `downloads\gog` in Lootnest's data
   folder is empty again.
4. Install a big game that comes in several files (an .exe and .bin
   parts): the bar says which file it's on and fills once across all of
   them.
5. Start a big download and **Cancel the download**, then Install again:
   the panel says how much is already downloaded, its button says
   **Continue installing**, and the bar starts from there rather than from
   the start. If GOG's servers don't take a Range request, it starts over,
   which is fine.
6. Make a folder named like another GOG game in the games folder, with a
   file in it, then install that game: it refuses, and nothing changes.
   Close the installer's window part way, or say no at Windows' prompt:
   it says so, and Install again doesn't download the files again.
7. On a game installed from GOG, its page says `Version <version>,
   installed from GOG. Checked ...`. When GOG has a newer installer,
   **Update** downloads it and opens the update panel: **Run the
   installer** (with the copy of the game on) copies the saves, runs it
   into the game's folder without questions, and lists what it changed.
   **Undo** puts the old files back and the page shows the old version
   again, with the update on offer. A game GOG Galaxy installed shows no
   such line: Galaxy updates it.
   The game is also on the Updates page (1.83), saying From GOG with the
   same version and update; its **Update** does the same, with the bar
   and Cancel in its row. Signed out, the row stays, saying Sign in to
   GOG to check.
8. **Sign out**: the GOG games leave Not installed, the installed ones
   stay and start, and `gogToken` is gone from `config.json`. Lootnest
   also signs out with a plain message if GOG stops taking the sign-in.

## 19. Controllers on this PC (1.84)

Lootnest only reads here. The thing to watch is that nothing changes a
pad: a Bluetooth DualSense must keep working in games afterwards.

1. With no pad connected, Settings, Controllers says **No controller
   connected**.
2. Plug in a pad by USB (an Xbox pad and a DualSense if you have them):
   each shows under **On this PC** with its name, USB, and for the Xbox
   pad Xbox-style (XInput) with its slot. Press a button: it shows under
   **What Lootnest sees** too, with its buttons' family.
3. Pair a DualSense by Bluetooth: it shows as DualSense, Bluetooth. Open
   the page a few times, then play a game that uses the DualSense over
   Bluetooth (one with its own DualSense support, and one through Steam
   Input): the pad, its lightbar, rumble and triggers work as before.
   Nothing Lootnest did should have switched its mode.
4. Start DS4Windows with the DualSense connected: the page now also lists
   DS4Windows (Running now), ViGEmBus, HidHide if installed, and a
   virtual Xbox 360 pad made by DS4Windows. Without HidHide, a line says
   games may see two pads. Under What Lootnest sees, the Xbox twin says
   Lootnest ignores it while the other pad is in use.
5. With Steam running as well, a line says Steam Input and DS4Windows
   both translating can make games see double input.
6. Press a button on a pad, then unplug it while the page is open: it
   goes from both lists. A pad you never pressed goes when you open the
   page again.
7. With **Game details** on (1.85), open a Steam game Steam lists with
   full controller support (details kept from before 1.85 show it once
   they're read again, at 30 days): the Details block says **Full controller
   support**, with the buttons of the pad you're using, and for a game with
   DualShock or DualSense support, that PlayStation controllers work too.
   With DS4Windows running, no HidHide, and this page opened first, a line
   says the game may see your pad twice. **Edit details**, Controller,
   **Keyboard and mouse only**: the page says **Keyboard and mouse only**.
   With Game details off, this page links to it.

## 20. Mods (1.86)

Use a folder game you can reinstall, outside Program Files (D:\Games\Test),
and make a copy of its folder before you start (D:\Games\Test-copy).

1. Its page, **Mods**: Vanilla, and a line that every replaced file is kept.
   **Add a mod**, **Choose a .zip, .7z or .rar**: a mod archive whose
   files sit in a folder of its own (a texture pack wrapped in
   "ModName\Data\..."). The panel shows how many files it adds and
   replaces, the replaced files, and Into Data (or the folder it matched).
   Nothing in the game's folder changed (compare with the copy).
   **Add to the list**.
2. Add a second mod that replaces one of the same files, and put it
   below the first. **Apply**: the bar shows the files, the toast says
   the mods are in, the state says Modded: 2 mods, and the shared file is
   the second mod's. %LOCALAPPDATA%\Lootnest\backups has a
   saves-...-before-mods copy. Play: the game runs with both mods.
3. Turn the second off: Changes not applied yet. **Apply**: the shared
   file is the first mod's again.
4. **Put the game back as it was**: Vanilla. Compare the game's folder
   with the copy (WinMerge, or `fc /b` on the replaced files): the same
   files, byte for byte, and no folders the mods made.
5. Apply again, then change one replaced file by hand (or update the game
   in between) and edit a file a mod added. Put it back: the toast says 2
   files changed, and its Show opens where they are kept
   (%LOCALAPPDATA%\Lootnest\mods\<hash>\changed\<time>); both are there, and
   the game's file is the original again.
6. Apply a big mod and end Lootnest in Task Manager part way. Start it
   again: the log says it finished the last mods run, and the tab says
   Modded. Start the game, then Apply or Put back: it says to close it
   first.
7. Refusals: a mod with steam_api64.dll, steam_settings\ or
   ColdClientLoader.ini, and one with the game's own .exe, each say why
   in the panel and add nothing to the list. A game with EasyAntiCheat or
   BattlEye has no Mods tab and its note says mods are off; a Steam game
   under Program Files has no Mods tab either. Restart in safe mode:
   Apply and Put back are off and the tab says why.
8. With two mods applied, remove one: it asks, takes only that one out
   (the other stays in the game), then it's gone from the list; the
   archive you added it from is still in Downloads.

## 21. Mods for each profile (1.87)

Use the folder game from section 20 with profiles on, two people (Player 1,
Player 2), and two mods kept for it, A and B, that replace the same file.

1. **Mods**, **Mods for**: Player 1, **Make its own**, A on and B off.
   Player 2, **Make its own**, B on and A off. The line under the select
   says which set is in the game's folder now.
2. Play as Player 1: the swap moment says it's putting Player 1's mods
   in, then the game starts with A. Close it. Play as Player 2: Player 1's
   saves go back, Player 2's come in, then A comes out and B goes in. The
   shared file is B's. Both games keep their own saves.
3. **New run** for Player 1 with **Start with mods** on: the run's set has
   A on. Another run with it off: its set has every mod off, and playing
   it puts the game back as it was first.
4. Restart in safe mode and Play as Player 2 while A is in: the game
   starts as it is, and the log says nothing about mods.
5. With B in, apply an update from the update panel (and once with an
   installer): the mods come out first, the update goes in, and a toast
   says mods are off after the update. The Mods tab says Vanilla; nothing
   went back in on its own. Play as Player 1, then Player 2: the game stays
   vanilla. Apply from the Mods tab, then Play as the other person: the
   sets switch again.
6. Start Play while Lootnest is still applying mods from the Mods tab (a
   big mod): the game doesn't start, and the message says Lootnest is
   changing its mods. Rename a mod's folder under
   %LOCALAPPDATA%\Lootnest\mods\<hash> and Play as the other person: the
   game is put back as it was and starts without mods, and the log says
   why.

## 22. Mods by link, conflicts and order (1.88)

Use the folder game from section 20 on the same drive as
%LOCALAPPDATA%\Lootnest (C:), with mods A and B that replace the same
file, and a copy of the same game on another drive (D:).

1. **Mods**, More (the dots): **Install mods by linking when possible** is
   ticked. Apply A. In a command prompt, `fsutil hardlink list` on one of
   A's files in the game's folder: two names, the game's and the one under
   %LOCALAPPDATA%\Lootnest\mods\<hash>\<mod id>\files. Its .ini, .cfg,
   .json or .txt files, and anything under a Config or Saves folder, list
   one name only: they were copied. The file is read-only (Properties).
   deployed.json says "how": "link" or "copy" for each file.
2. On the copy on D:, Add a mod (the same archive as A) and Apply. Every
   file lists one name: they were copied, and the game runs the same.
3. Untick linking and Apply again on C:: every file lists one name.
   Tick it again.
4. A game that rewrites a config file: with a mod that brings one (an
   .ini), play, change a setting in the game, close it. Put the game back:
   the toast says a file changed, it's kept under changed\<time>, and the
   mod's own copy of the .ini is as it was added.
5. Clear the read-only box on a linked file in the game's folder and edit
   it. Put the game back: the change is kept under changed\<time>, the
   game's own file is back, and the mod's row says the game changed
   Lootnest's copy and Play starts the game without mods until it's off
   or added again. Apply with it on: it refuses, and nothing in the
   game's folder changes. Play: the game starts without mods.
6. Conflicts: with A then B on, A's row says it loses files to B and B's
   that it overrides files from A. **See the files** lists them. Drag B
   above A (and once with the arrows): the lines swap, the state says
   Changes not applied yet, and after **Apply** the shared file is A's.
   Press Esc during a drag: nothing moves.

## 23. Plugins (1.89)

Make four small .zip files from the examples in docs/PLUGINS.md: a
sources plugin (a JSON list in %APPDATA% naming a game folder on the
PC), a saves plugin (a game whose save folder exists), a metadata plugin
(a game in the Library) and a theme plugin.

1. Settings, Plugins, **Install from file**, each in turn: each shows in
   the list, on, with what it provides. The sources plugin's game shows
   in the Library after the scan, its store named as the source's
   `name`.
2. The saves plugin's game: Turn on profiles for it. The setup lists its
   location among the suggestions, marked From and the plugin's name, and
   it isn't one of the locations picked for you until you add it.
3. The metadata plugin's game page, with Game details off: Details shows
   the plugin's summary and says From and the plugin's name. With Game
   details on, Steam's win where it has them. Edit details: your edit wins.
4. Edit profile, Look, Theme: choose the theme and Save. Its colours show
   while that profile is at the PC; seat another profile (or Nobody) and
   Lootnest's own come back. Check text on the sidebar, menus and the red
   (now the theme's) button by eye.
5. Refusals, each with its reason and nothing left in
   %LOCALAPPDATA%\Lootnest\plugins: a .zip with an .exe or a .js in it, a
   file named ..\x.json (made with a zip tool that allows it), a .txt over
   8 MB, an id with capitals, a theme setting --fs-14, a theme with dark
   text (#333333 for --fg-1).
6. Install the theme again with version 1.1.0: **Roll back to 1.0.0**
   shows and works; plugins\.old\<id>\1.0.0 held the old one.
7. Turn the sources plugin off: its game leaves the Library at once (a
   game with profiles from it stays). On again: the scan brings it back.
   **Remove**: plugins\<id> goes, nothing else in the data folder changes.
8. Safe mode (Shift on open): Plugins says every plugin is off, the
   switches and Install from file are disabled, the theme isn't shown and
   the sources plugin's game isn't in the Library.

## 24. Signed plugins (1.90)

With Go on any PC: `go run ./cmd/plugsign keygen` in an empty folder, then
make a theme plugin folder from docs/PLUGINS.md and
`go run ./cmd/plugsign sign <folder>`. Zip the folder's contents as
1.0.0. Make 1.1.0 the same way, and 1.2.0 signed with a second key
(keygen in another folder).

1. Install 1.0.0: its row says Signed, key and the fingerprint keygen
   printed. An unsigned plugin's row says Not signed, in the same grey.
2. Change one colour in theme.json inside a copy of the 1.0.0 .zip, without
   signing again: it's refused, the signature doesn't match its files.
3. Install 1.2.0 (the second key): a dialog names both keys, with Cancel
   focused. Cancel: 1.0.0 stays. Install 1.1.0: it installs, no dialog.
4. Install 1.2.0 again and choose **Install with the new key**: the row
   shows the new fingerprint and no **Roll back**, as 1.1.0 has the old
   key. With the dialog open, replace the .zip with one signed with a third
   key and accept: the dialog asks again, naming the third key.
5. Install an unsigned 1.3.0 of the same id: refused, it isn't signed and
   the version you have is.
6. Edit theme.json in %LOCALAPPDATA%\Lootnest\plugins\<id>, then open
   Settings, Plugins without restarting: the plugin is off with the reason.
   Delete its signature.json: the reason says the signature is gone, with
   the key.
7. **Remove** it: the dialog says Lootnest forgets its key; installing the
   1.0.0 .zip again works with no dialog.

## 25. Saves and Ctrl+K (1.91)

1. Open **Saves** in the top bar (where Cloud saves was). The numbers under
   the title match the Library: games with profiles, games with copies,
   save points and the newest copy's time.
2. Delete a folder game's in-folder saves and let the safety net notice:
   **Needs a look** names the game; **Open its saves** opens its Saves tab.
3. Under **Recent copies**, **Restore** on a safety net copy, **Go back to
   it** on a save point and **Put back** on a copy before a swap each ask
   first, then do what the same button on the game's page does, and the
   list is read again.
4. Sort **Per game** by each column; a game's name opens its Saves tab.
5. Home's **Set them up** and Settings, Cloud saves' button open Saves at
   its Cloud saves section, with everything that page had.
6. Ctrl+K with the box empty lists a few commands. Type `set`: Open
   Settings and the categories; arrow down to one and Enter opens it.
   Type the start of a game's name and press Enter without choosing: the
   game opens (or the Store search, when it's on), never a command. With a game
   running, **Make a save point** shows Ctrl+Shift+F5 and makes one.
   Esc closes the list.

## 26. TV mode (Lootnest Link, M1)

Needs Apollo (or Sunshine) on the PC and Moonlight TV on the LG TV, paired.

1. In Apollo's web settings, Applications, add an app named Lootnest with
   the command `"<Lootnest's folder>\Lootnest.exe" --tv`. Keep its own
   options as they are.
2. With Lootnest closed, start Lootnest from Moonlight TV. Lootnest opens
   on the TV, not on the PC's own screen, filling it, on Who's playing (or
   Home with one person). The log says "TV mode on".
3. Only with the controller: pick a profile, move to a game with profiles,
   A, A on Play. The swap screen shows, then the game. Lootnest is gone
   from the TV while the game runs.
4. Quit the game from its own menu. Lootnest is back on the TV within a few
   seconds and the controller moves it at once (no click needed). Say if
   it needed a key or a click.
5. Do 3 and 4 again with a Library game without profiles (a Steam game).
6. Start: the menu. Exit TV mode: the stream ends on the TV, and on the PC
   Lootnest is back as it was (in the tray if it was there). The log says
   "left from its menu".
7. Start the app again, then end the stream from Moonlight (hold Start +
   Select, or quit from Moonlight's menu). Lootnest leaves TV mode on the
   PC and keeps running; the log says "the stream ended".
8. Repeat 7 while a game runs: the game keeps running and its saves are
   kept when it closes, as without TV mode.
9. If the log says "Lootnest runs inside another program's job", Apollo
   doesn't let Lootnest start on its own: say so, with the log.

## 27. TV streaming set-up (Lootnest Link, M2)

On a PC without Apollo, with the LG TV and Moonlight on it.

1. Settings, TV, **Set up**. The download shows its progress, then
   Windows asks once. Allow it. The page ends on Streaming engine with
   Apollo 0.4.6 and On. Apollo's own pages never open.
2. `https://localhost:47990` from another PC on the network doesn't
   answer (pages kept to this PC). Windows Defender Firewall, the Apollo
   rules are for Private only.
3. On the TV, Moonlight finds "Lootnest on <PC>". Choose it; type its PIN
   in **Add a TV** with a name, **Pair**. The TV is listed as Paired.
4. Moonlight lists Lootnest. Do section 26 from its step 2.
5. Pair a second device (a phone with Moonlight). It can start Lootnest
   and use a controller too.
6. **Turn off**: Windows asks; the TV can't find the PC. **Turn on**: it can.
7. **Forget** the TV: Moonlight on it has to pair again.
8. Update Lootnest, or move its folder, then start it: within two minutes
   the log says nothing about the engine, or "TV mode's app now starts".
   Section 26 still works.
9. **Save a report**: Explorer shows `tv-diagnostics.txt`; search it for
   the word "password". It only appears as "[password]".
10. With Apollo installed by hand before (its own user): **Use it** asks
    Windows once and takes over its sign-in; its apps and paired devices
    stay.

## 28. Add an LG TV (Lootnest Link, M3)

On the LG TV, with section 27 done on the PC.

1. On the TV: LG Content Store, Developer Mode; sign in with an LG
   developer account; Dev Mode Status on (the TV restarts); Key Server
   on, left open.
2. Settings, TV, **Add an LG TV**. The TV is listed (or type its
   address). Type the passphrase, **Install**. The steps go by: key,
   download, connecting, installing, pairing.
3. (2.2.9) While it streams, the TV's display is Windows' main one:
   Lootnest's TV mode and any game it starts open on the TV, and the PC's
   monitors stay on as extra displays.
3a. On the TV, Lootnest opens by itself on its own screen (the gem,
   "Lootnest", "Pairing with your PC", then "Starting Lootnest"; never
   Moonlight's list of computers), pairs without a PIN and goes straight
   into Lootnest's TV mode. Settings, TV lists the TV as Paired.
4. Quit it and open Lootnest from the TV's home screen: "Connecting to
   <PC>", then straight into TV mode again.
5. Put the PC to sleep (with Wake-on-LAN on in its network card), open
   Lootnest on the TV: "Waking <PC>", then TV mode. With the PC off for
   good: after 90 s, "<PC> isn't answering" with Try again, Wake the PC,
   Settings and Quit.
6. In a game, hold Start and Select for two seconds: the stream ends, the
   PC leaves TV mode, and the TV shows Lootnest's screen with Play,
   Settings and Quit. Play goes back in.
7. With the TV app open, **Install again** from the PC: the app closes,
   opens again and pairs.
8. Leave the PC and TV on for a day: the TV's row says developer mode was
   extended in the last 12 hours; the log says "developer mode ... extended".
9. **Install again** updates the app; **Forget** removes the pairing.
10. In Windows Settings, set this network to Public, then **Install**: it
    stops before the key with "Windows treats this network as public".
    **Make this network private**, accept Windows' prompt: "This network
    is private now". **Install** works again.

## 29. TV mode as a console (2.3.0)

1. In TV mode, start a game: its art fills the TV with "Starting", then
   the game shows. The desktop and taskbar never appear on the TV.
2. Quit the game: Lootnest is back at once, where you started it.
3. Start another game: the same, with no desktop in between.
4. While a game starts, B comes back to Big Picture.
5. Move the PC's mouse to every edge of your monitors: it doesn't go onto
   the TV's display.
6. Exit TV mode: the taskbar is back on your monitor.

## 30. The Home button in TV mode (2.4.0)

1. In TV mode, start a game, then press Home on the controller (or PS on
   a DualSense connected to the PC): the menu opens over the game.
2. Resume game, or Home again, or B: back in the game, controls working.
3. Lootnest: the menu closes and Lootnest stays; Play on the same game
   goes back to it, still running.
4. Close the game: the label asks to press again; then the game quits
   and Lootnest is back.
5. With no game running, Home opens TV mode's menu.

## 31. Apps in TV mode (2.5.0)

1. With Spotify and Stremio on the PC, TV mode's Home has Apps
   with all three.
2. Spotify: opens over Lootnest; Home opens the menu; Close the game
   closes its window and Lootnest is back (Spotify may stay in the tray).
3. YouTube: its TV layout in a window of its own, driven by the
   controller; your own Edge windows are untouched when it closes.
4. Stremio: as Spotify.

## 32. Leaving TV mode from the PC (2.5.2)

1. With TV mode on and the PC's monitor on, the mouse stays on the PC's
   monitor, and a bar at its bottom right says Lootnest (or the game) is
   on the TV, with Leave TV mode.
2. Typing in another program on the PC keeps its focus: the bar never
   takes it.
3. Leave TV mode on the bar: the TV's stream ends, the taskbar comes back
   and the mouse is free; Lootnest is back where it was, or in the tray.
4. Again with Ctrl+Shift+F11, and again from the tray's Leave TV mode.
5. YouTube: full screen in its TV layout, no browser window or extension
   page; Sign in with the code at youtube.com/activate, still signed in
   the next time.

## 33. Lootnest stays behind the game (Big Picture and TV mode)

1. In Big Picture (F11, not TV mode), start a game Lootnest starts by its
   exe: its art fills the screen with "Starting", then the game is in
   front. The desktop, the taskbar's programs and other windows don't
   show in between.
2. The same for a Steam game (Steam's own "launching" note may show over
   Lootnest for a moment) and for a game from a folder with no known
   process.
3. A windowed game: what's around its window is Lootnest, not the desktop.
4. Press the controller's Home button (Xbox's Guide, a PlayStation pad's
   PS) over the game: the menu comes over a still picture of the game.
   Resume goes back to the game.
5. Home again, Lootnest: Big Picture is in front, the game shows as
   Playing, and its page has Go back to the game and Close the game.
   Go back to the game returns to it.
6. Home, Close the game (twice): the game closes and Lootnest is what's
   there, with the keyboard and the controller.
7. Quit a game from its own menu: Lootnest is in front again within a few
   seconds, with no desktop in between.
8. All of it again in TV mode on the TV.
9. Show recent activity names each start: "its window is up after ...",
   or "no window found in 25 s" for a game Lootnest couldn't see.

## 34. A PlayStation controller as an Xbox one (Settings, Controllers)

1. With ViGEmBus on the PC (DS4Windows or the TV streaming set-up
   installed it), Settings, Controllers says so under In games; turn
   **Show a PlayStation controller to games as an Xbox one** on.
2. With a DualSense on USB, start a game that only knows Xbox
   controllers: it answers to the DualSense, with Xbox signs. Sticks,
   both triggers, the D-pad, every face button, L1 and R1, L3 and R3,
   Create as View and Options as Menu.
3. The same over Bluetooth, and with a DualShock 4.
4. Unplug the pad while the game runs and plug it back: it works again
   within a few seconds.
5. Close the game: Settings, Controllers no longer lists Lootnest's Xbox
   360 controller under On this PC.
6. A game that knows the DualSense itself, with Launch options,
   PlayStation controllers, **Leave them as they are**: only its own pad.
7. Without ViGEmBus: the switch says the driver isn't on this PC, and
   nothing else changes.
8. The PS button in Big Picture still opens Lootnest's menu over the
   game, once.
9. Show recent activity has "Controller as Xbox: ..." lines for each
   change.

## 35. YouTube as Lootnest's own window

1. Big Picture or TV mode, Home, Apps, YouTube: the game-start screen,
   then YouTube full screen in its TV layout (Get started, a QR code),
   never the desktop site or a cookie question over it.
2. The controller: D-pad and left stick move, a held direction repeats,
   A chooses, B goes back. With a DualSense and A PlayStation controller
   as an Xbox one on, one press moves one step, not two.
3. Sign in with the code on a phone; play a video with sound. Close
   YouTube from the Home menu and open it again: still signed in.
4. The Home button opens Lootnest's menu over it; while the menu is
   open the controller doesn't move YouTube; Resume goes back to it;
   Close the game closes it and Lootnest is back.
5. With two monitors and Lootnest in Big Picture on the second: YouTube
   opens on that monitor.
6. Task Manager: a second Lootnest.exe while YouTube is open, gone after
   it closes. No Edge window is opened.

## 36. The Home screen over a game, and Achievements in Big Picture

1. A game running from Big Picture, the Home (Guide or PS) button: the
   game's picture dimmed, the choices on the left, its achievements on
   the right. One press of the D-pad moves one choice; holding it moves
   on after a moment and then steadily; the stick does the same and never
   skips. The light slides, nothing blinks.
2. Make a save point: the choice says so, then Save point made, and the
   point is in the game's Save points afterwards.
3. Achievements: Lootnest comes up on the game's page with the focus on
   its achievements; the game still runs (Go back to the game).
4. Close the game asks for a second press. For Spotify or YouTube the
   choices name the app and there are no achievements.
5. Big Picture, Achievements: Up and Down change the game and the right
   side follows; OK goes into the list, which slides by itself; B comes
   back to the games; Up from the first game reaches the bar.
6. Who's playing: the chosen profile grows, its ring spreads, the others
   fade, then Home. With Motion at Less or Off it happens at once.

## 37. Spotify's web player as Lootnest's own window

1. Big Picture, Apps, Spotify: Spotify's web player full screen, larger
   than in a browser (about 1280 of its pixels across the screen).
2. Sign in with the PC's keyboard and mouse. Close it from the Home menu
   and open it again: still signed in.
3. The controller: a white frame moves card to card with the D-pad or
   stick; Down past the last row on screen scrolls the list and moves on;
   A opens a playlist; B goes back; B with a menu open closes the menu.
4. A on a Play button starts the music, with sound (this is protected
   playback: if it says it can't play here, note what it says). X pauses
   and plays; LB and RB change track.
5. Apps has one Spotify, this one, also when the Spotify program is
   installed.

## 38. The controller as a mouse

1. Big Picture, a game or a store launcher that wants a mouse, the Home
   button, Controller as a mouse: a note says the
   controller is a mouse now. The left stick moves the pointer, slowly
   for a small push; A clicks and, held, drags; X opens the right-click
   menu; the right stick scrolls; the D-pad moves as the arrow keys; B
   is Escape.
2. Y opens Windows' on-screen keyboard and the pointer can type on it;
   Y again closes it. (If clicks don't reach the keyboard, note it.)
3. The Home button: the menu comes up and the stick moves the menu, not
   the pointer. Controller as a mouse says On; turn it off, go back: the
   stick no longer moves the pointer.
4. Over YouTube or Spotify's web player it's off; turned on from the
   Home menu, the pointer moves and the D-pad no longer moves the site's
   own focus twice; off again, the site's own navigation is back.
5. Over a game it's off unless turned on from the Home menu (for a
   launcher's window); it's off again for the next game.
6. Close the program: the pointer stops following the stick.

## 39. Discord as Lootnest's own window

1. Big Picture, Apps, Discord: its sign-in page, full screen and large.
   Scan the QR code with Discord on a phone: signed in, and still signed
   in the next time.
2. The controller moves a white frame through servers, channels and
   messages; A opens a server and a channel; B goes back or closes what's
   on top.
3. Join a voice channel: no question about the microphone, the others
   are heard (on the TV while it streams), and with a microphone on the
   PC they hear you.
4. Share your screen from the voice channel: note whether a window to
   choose the screen comes up and whether the others see it.
5. Typing a message needs a keyboard: with Controller as a mouse on from
   the Home menu, Y opens Windows' on-screen keyboard.

## 40. The TV app kept up to date

1. With an older Lootnest for TV on a saved TV, Settings, TV: the TV's
   row says its version and that the new one is ready; the key is
   Update to that version.
2. Update: no passphrase is asked for; the drawer counts through
   connecting and installing, the TV opens Lootnest, and the row then
   shows the new version. On the TV, the corner of Lootnest's screen
   says the same version.
3. End the stream (hold Start and Select two seconds): Size and Quality
   are beside Play.
4. With the TV on another app and an older Lootnest for TV on it, start
   Lootnest on the PC and wait a minute: a note says the TV app is now
   the new version, and the log has "went from ... to ...".
5. With Lootnest for TV open on the TV, the log says it's left for
   later, and nothing closes on the TV.
6. After developer mode was turned on again on the TV (a new key):
   Update says the kept key no longer opens the TV and to use the
   passphrase; Add an LG TV with the passphrase works as before.

## 41. A stream's numbers, and drawing at twice the rate

1. Stream a game from the TV for two minutes or more, then leave TV
   mode. Within half a minute Settings, TV has Last stream for that TV:
   one line that says what held it back, and its numbers. Lootnest's
   log has the TV's "Stream: ended after ..." line.
2. Save a report with the TV on: its section has "Wi-Fi link" when the
   TV is on Wi-Fi.
3. Turn on Draw at twice the stream's rate with no stream running: the
   log says the engine's settings are up to date, and Settings, TV shows
   the engine On again within a minute.
4. Start a stream: Windows' display settings show the TV's display at
   120 Hz while the stream is 60. A game with V-Sync on runs at 120.
5. Turn the switch off while a stream runs: the log says the engine
   takes its settings at the next start, and the stream isn't cut.

## 42. Stremio with a TV interface

1. In TV mode or Big Picture, Apps has one Stremio, whether or not the
   Stremio program is installed.
2. Open it: Stremio's web version fills the screen. The stick and the
   D-pad move a white frame over posters, the side bar and the search
   box; A opens, B goes back a page.
3. With the Stremio program installed and not running: no yellow
   "Streaming server is not available" bar. Task Manager has
   stremio-runtime.exe while the window is open, and not after it closes.
4. Sign in (Stremio's own page), open a title and a stream: it plays
   with sound on the TV. X pauses.
5. With Stremio already open on the PC, Lootnest starts no second
   server, and closing Lootnest's window leaves Stremio's running.
6. With no Stremio installed, the yellow bar shows and Lootnest
   installs nothing.
7. The Home button opens Lootnest's menu over it; Close ends it and
   Lootnest is back.

## 43. Script plugins, and updates from a plugin

None of this has run on a PC. Use a test folder game with saves you don't mind losing first, and a script plugin of your own that offers an update for it (docs/PLUGINS.md; `updateScript` in `core/fullupdate_integration_test.go` is a small one).

1. Settings, Plugins, Install from file: the plugin's .zip. It's listed as off, with what it asks. Turn it on: the dialog lists the same asks; Cancel leaves it off; Allow turns it on and its pages appear in Settings. Restart with Shift (safe mode): the pages are gone and nothing of the plugin runs.
2. Updates: the offered update shows under From plugins, and on the game's page.
3. Update with a direct https archive of a whole build: the row downloads (Pause, Resume, close Lootnest in the middle and open it again), opens, and says Ready. The game's folder hasn't changed. Review and update lists the saves and what only your folder has. Replace: the steps move, the game starts, saves are there, `<folder>.lootnest-old` exists, and the Saves tab has "Before the update". Play over a minute and close: the old folder goes to the Recycle Bin.
4. Again, then Roll back before playing: the old build is back, saves as they were, the new build in the Recycle Bin.
5. Pull the plug (or end Lootnest in Task Manager) right after Replace on a large game on another drive than Lootnest's data: at the next start the game is as it was and the Updates page says the update was cut short.
6. Not enough room: Update on a drive with less free space than the download needs is refused with both numbers.
7. A patch and an installer: each ends as Ready with Open the update, which opens the update panel; the installer runs only on Run the installer.
8. Only what changed: an update that is one .zip at an https address, for a large game where only a few files differ. The row says Compare, then Get what changed with files done of all; Downloaded stays far under the archive's size; `<folder>.lootnest-new` holds only the changed files. Pause, close Lootnest, open it, Resume: it goes on. Review and update says replaced, new and already the same; Update the game: the game starts, saves are there, `<folder>.lootnest-old` holds only the old versions of what changed. Roll back: every file is back (compare a few by size and date). Do the same with a .rar in parts and a .7z.
9. Only what changed, cut short: end Lootnest in Task Manager during Replacing the changed files. At the next start the game is as before and the row says so.

## 44. Add an Android TV (Lootnest Link, M4a)

None of this has run on a TV. Use an Android TV, Google TV or Fire TV on
the same network as the PC, with TV streaming set up.

1. On the TV, turn on Developer options and Network debugging (Fire TV:
   ADB debugging). Settings, TV, **Add an Android TV**: the TV's address
   is listed, or type it.
2. Install: the TV asks "Allow USB debugging?" with Lootnest@<PC>. Choose
   Always allow, then OK. Lootnest downloads, installs and opens
   Lootnest for TV, which pairs and opens Lootnest's TV mode on the PC.
3. Choose Cancel on the TV's question instead: the drawer says the TV
   didn't allow this computer, and nothing is installed.
4. The row shows the TV's name, model and Android version, and Paired.
   Play a game, then Start, Select, LB and RB together: Lootnest for TV's
   screen with Play, Settings, PCs and Quit.
5. Quit, then open Lootnest from the TV's home screen: it goes straight to
   the PC. Put the PC to sleep and open it again: "Waking" (when the PC
   wakes on LAN), then TV mode.
6. Install again on a paired TV: no question on the TV, no pairing PIN
   asked, and it ends paired.
7. Settings, TV, Save a report: the report has the TV app's log (tag
   LootnestTV) for each Android TV.

## 45. Achievements that come by themselves

1. A folder game whose tool writes achievements at the first unlock:
   open its page (None found), play until something unlocks, close the
   game. Within about 10 seconds its Achievements show, with no button.
2. Settings, Online: turn on **Achievements from my Steam account** with
   a Steam Web API key. A game owned on Steam whose copy on the PC has no
   achievements (Red Dead Redemption 2 from Hydra) shows the Steam
   account's, with pictures, under "From Steam, for this PC's account".
3. With the profile's Game details private on Steam: nothing shows, and
   the log ("Show recent activity") says to make them public.
4. Turn it off: the game shows None found again.

## 46. TV fixes of 19 Nov: the emergency key, a keyboard, Stremio, the TV apps

1. In TV mode, press **Ctrl+Shift+Q** on the PC's keyboard: the mouse
   moves freely over every display, the taskbar is back, the TV's stream
   ends, the PC's displays are as before, and Lootnest is gone from the
   tray. The log says "Ctrl+Shift+Q pressed". The bar on the PC's screen
   names the key while TV mode is on.
2. The same in Big Picture on the PC (no TV): Lootnest quits.
3. Stremio from Big Picture's Apps: put the focus on the search box and
   press A. A keyboard comes up at the bottom; type a title with the
   D-pad and A, X deletes, Y adds a space, Start searches, B closes it.
   The same in Spotify's search and in a Discord message box.
4. Stremio, with the Stremio program installed: open a film and pick a
   stream. It plays, with no "streaming server" warning. The log says
   whether the server was already running or was started.
5. LG TV app 1.6.2 (Install again or let Lootnest update it): on its
   home screen, press Right many times and then Left: the focus walks
   the keys and stops at the ends, never leaving them. Nothing is
   drawn over the keys. After a stream, its log has a "Sound: ... ms
   left out" line when it had to trim the sound.
6. Android TV app 1.0.1: the same on its home screen; the TV app's log
   in Save a report has a "Sound: ..." line saying whether the TV gave
   low latency.

## 47. The FPS counter

1. Settings, Games and swaps: **Let Lootnest read frame timings** shows
   with **Allow**. Press it: Windows asks for administrator once; the row
   becomes "Lootnest's frame reader" with **Remove**. Task Scheduler has
   `\Lootnest\Frame counter`, and `C:\Program Files\Lootnest\Frame
   counter\lootnest-frames.exe` is there.
2. Without signing out, turn on **FPS counter**. The log says "listening
   to frames through the frame reader".
3. Start a game (windowed or borderless): a small box with a number and
   "FPS" sits in the top-left corner of its screen within about 6 seconds,
   and the number follows the game (cap the game's frame rate at 60 and
   it reads 59 to 61; compare with the game's own counter, Steam's or
   Game Bar's). Clicks go through the box to the game.
4. **Ctrl+Shift+F10** hides it; again shows "--" for a moment, then the
   number. The log says "FPS counter on/off: Ctrl+Shift+F10".
5. Change the corner in Settings: the box moves within a second.
6. Alt+Tab to the desktop or a browser: the box goes within 3 seconds;
   back to the game, it comes back.
7. A Vulkan game (DOOM Eternal), an OpenGL one and a Direct3D 9 one: a
   number shows for each.
8. Big Picture or TV mode: the Home button over a game has **FPS
   counter** On/Off; Resume, and the box shows or not as chosen. Big
   Picture's Settings, While you play, has the same row.
9. A game in exclusive fullscreen: no box; after it closes, Settings
   says the game ran in exclusive fullscreen, and the log names it.
10. Turn it on before step 1 is done: a toast says it needs permission
    first, and the switch goes back off.
11. With the counter on, look at Lootnest's CPU in Task Manager while a
    game runs at a high frame rate: it stays near 0%.
12. While a game runs with the counter on, Task Manager shows
    lootnest-frames.exe; it's gone a few seconds after the game closes or
    the counter is turned off.
13. **Remove**: Windows asks, the task and the Program Files folder are
    gone, and the row is **Allow** again.

## 48. Mods from Nexus Mods

1. A folder game with mods allowed (Red Dead Redemption 2 kept offline
   is the owner's case): its page, Mods tab, **Nexus Mods**. Off, it
   offers **Turn on Nexus Mods**; on, it finds the game on Nexus and
   shows its mods. Search "trainer", change the order: the grid follows.
   **Not this game?** lists Nexus's games for its words; choose one.
2. Open a mod: its picture, summary, facts, **It needs** (a Nexus mod
   opens in the drawer; one from elsewhere opens in the browser) and
   its files by kind, the first main file's Download lit.
3. **Download**: a window titled Nexus Mods opens on the file, not the
   browser. Sign in to Nexus there (by e-mail, and once with Google or
   Steam: those pages must open in the same window). Press Manual
   download, then Slow download: no download list appears, a note at
   the bottom of the page says Lootnest is taking it, and the file goes
   to `nexus-downloads` in Lootnest's data folder (with a
   `.lootnest.json` beside it), not to Downloads.
4. While it downloads, the mod's row in Lootnest says "Downloading, x
   of y" with a bar that fills. When it's done the Nexus window closes,
   Lootnest comes to the front and shows "Add a mod" with the plan. Add
   it: it's on the list with the mod's name from Nexus, not the file's.
5. In the Nexus window, **Mod Manager Download** (and its Slow download)
   opens nothing outside (no Vortex prompt): the window goes to the same
   file's page with a note to press Manual download, and that works as
   in 3. A link to another site (a mod's own page elsewhere) opens in
   the browser.
6. Close Lootnest while a file is awaited, download it, start Lootnest:
   the file is still found (waits last two hours).
7. Settings, Updates: a Nexus file in Downloads isn't offered as a
   game's update.
8. Close the Nexus window and open it again: still signed in.
9. Download a second mod's file in the same window while the first is
   still awaited: each arrives for its own mod.
10. A month later (or set a file's date back), Lootnest's start removes
    downloads in `nexus-downloads` that were never added.

## 49. Go back to a save point, all the way

1. A game outside Steam, not running: make a save point, play on and save
   in the game, close it. On its page, Saves, **Go back to it**: the
   dialog says the game starts on that save. Answer **Go back and play**:
   the game starts, and its Continue (or newest save) is the point's.
2. The same with the game running: the dialog says **Close and go back**.
   The game closes as it would from its window (no force), the button
   reads Closing the game, then Going back, and the game starts again on
   the point.
3. A game that asks "Quit?" when its window closes: leave the question
   unanswered. After about 30 s Lootnest says the game didn't close and
   nothing changed; the saves are as they were.
4. A Steam game with Steam Cloud: go back, let it start. In Steam, the
   game's page says Cloud is up to date (no conflict), and the point's
   save is the one loaded. Look at the save files' times: the newest is
   the moment of going back.
5. A game with profiles, running as one profile: go back to a point of
   that profile. The session ends as usual (Lootnest's log: the profile
   saved), then the point goes in and Play starts as the same profile.
6. Big Picture: a game's Save points, choose one, the question says the
   same, and the game starts.

## 50. Save now (the game's quick save key)

1. A game with a quick save key, such as Skyrim Special Edition: its page,
   Saves. Quick save key says None with **Use F5**; press it. The line
   says a save point presses F5 first.
2. Play, then in the game press **Ctrl+Shift+F5**. The game shows its own
   "Quick saving" (no Ctrl or Shift shortcut happens), the toast says
   "Saved and kept", and the new point says "Saved by the game first".
   Go back to it later: the game loads that exact moment.
3. With the game running, Lootnest's window in front: **Save now** on the
   game's page brings the game to the front, it saves, and the point is
   made. The button reads Saving in the game, then Copying.
4. During a fight or a scene where the game refuses to save: after about
   12 s Lootnest says the game didn't save, and no point is made.
5. Big Picture or TV mode, the Home menu over the game, Make a save point:
   the menu closes, the game saves, the point is made.
6. A game with Easy Anti-Cheat or BattlEye in its folder: the Saves tab
   says Lootnest never presses keys in it, and a save point is a copy as
   before (the log says why no key was pressed).
7. A game that reads keys by raw input or DirectInput (most newer ones):
   F5 from Lootnest works there too. If one ignores it, note which game.

## 51. Every save the game makes

1. Start a game whose saves Lootnest knows (its Saves tab has save
   points). Save in the game's menu, or reach an autosave. Within about
   15 s the Saves tab has a point "Saved by the game, <time>", with a
   picture, and no toast or sound in Lootnest's window.
   The game shouldn't hitch when it saves (the picture is taken then);
   note any game that does.
2. Save again without moving: no second point (the files didn't change).
3. Press Ctrl+Shift+F5 (or Save now): one point, not two.
4. Play long enough for over 30 game saves (or set the limit lower in a
   debug build): the oldest "Saved by the game" points go, the others
   stay. Rename one: it stays for good.
5. Settings, Games and swaps, **Keep each save a game makes** off: no new
   points while playing.
6. A game that saves every minute (Red Dead Redemption 2): after 15
   minutes the Saves tab has a game save about every 5 minutes plus the
   latest one, not 15, and each point's time played grows with the
   sitting.
6. A game whose saves are over 512 MB: the log says Lootnest doesn't keep
   each save of it, and play is unaffected.
7. A Steam game without profiles started from Steam: its saves are kept
   too, when Lootnest knows the game's process.

## 52. The controller in TV mode while the PC is in use

The TV app connected, a DualSense or DualShock 4 on the TV (Apollo makes
a virtual DualShock 4 on the PC for it) or an Xbox pad.

1. TV mode on, then click on a window on one of the PC's own monitors, so
   Lootnest no longer has the keyboard. On the TV, the D-pad and the stick
   move in Big Picture, the one chosen is lit, A opens, B goes back, LB
   and RB change section. Before this, only the PS button did anything.
2. Type in the window on the PC at the same time: nothing typed goes to
   Lootnest, and the TV keeps answering the controller.
3. Start a game from the TV: while the game is in front on the TV, the
   controller is the game's and Big Picture doesn't move behind it. The
   PS button still opens the Home menu over it, and its choices move.
4. Click Lootnest's window again (if the mouse can reach it): the
   controller works as before, with no step taken twice.
5. Lootnest's log says "Controller for Lootnest's page from Go: true" when
   the PC's window was clicked, and "false" when a game came in front.

## 53. The PC's monitors in TV mode

Two monitors on the PC, the TV app connected.

1. Before TV mode, note Windows' Settings, Personalisation, Taskbar,
   Taskbar behaviours: Show my taskbar on all displays, and Show my
   taskbar apps on.
2. TV mode on: both PC monitors have a taskbar, with Lootnest's button on
   each, and each has Lootnest's bar (Leave TV mode) at its bottom right.
   The TV shows no taskbar. Both bars' keys leave TV mode.
3. Open a browser, a folder (Win+E from the PC's taskbar or Start) and
   Settings on the PC: each opens on the monitor the mouse is on, never on
   the TV. Lootnest's log says "moved to the PC's display" for each that
   Windows had put on the TV.
4. From the TV, start a game through a launcher (an EA or Ubisoft game):
   the launcher and the game stay on the TV. YouTube, Spotify and Discord
   from Big Picture stay on the TV.
5. A game's own dialog while it plays on the TV stays on the TV.
6. Leave TV mode: the taskbar settings are back as noted in step 1, and
   the main taskbar is back on the PC's main monitor.
7. With TV mode on, press Ctrl+Shift+Q: the same settings come back. And
   if Lootnest is ended from Task Manager in TV mode, its next start puts
   them back.
## 54. Big Picture: music, Triangle for Settings, Mods

1. Open Big Picture (or TV mode): after a moment a calm piece plays.
   Start a game: it fades out within a second or two and stays silent
   while the game runs; back in Big Picture it fades in again. The same
   with a trailer playing behind a game, and over the Home menu.
2. Press Triangle (Y) on a game's cover: the Settings key in the bar (the
   gear) is chosen; Triangle again goes back to the cover. Triangle, then
   the cross (A): Settings opens; B from the parts on the left goes back to
   the gear.
3. Settings, Sound and music: Music off stops it at once; Music volume Low,
   Medium and High change it while it plays. Settings, Sounds for unlocks
   off silences the music too.
4. In TV mode, Settings has a TV part with Draw at twice the stream's rate
   and Leave TV mode.
5. A game with mods (added on the PC): its page in Big Picture has Mods.
   Turn one off, press Play: the game starts without it, and the PC's Mods
   tab shows it off for the same profile.

## 55. Big Picture's Home and its glass

1. Big Picture, Home: move along Continue playing. The cover in focus
   rises with a glow in the seated profile's colour and the others dim.
   Collections come right under the row.
2. The bar's sections sit in one glass pill; keys, rows, the achievements
   and the menu are see-through glass over the game's art, as Lootnest's
   window on the PC. With Settings, Appearance, Glass off on the PC, they
   are solid.
3. On the TV, the glass looks smooth (no flicker or bands from the
   stream) and moving between games stays as quick as before.
4. With music on, move to another game and wait a few seconds: the piece
   moves to another key at its next chord.

## 56. The FPS counter inside a game

1. A Direct3D 11 story game with no anti-cheat (a folder game, or Steam
   without Valve Anti-Cheat), counter on: within a few seconds the number
   and a frame's time are drawn in the game's own picture, in the chosen
   corner, and Lootnest's own box doesn't show. The log says "inside
   <game> (no anti-cheat), drawn in the game" and "draws with Direct3D 11".
2. The same game in exclusive fullscreen: the number still shows.
3. Ctrl+Shift+F10 hides and shows it in the game; changing the corner in
   Settings moves it within a second.
4. A Direct3D 9 game and an OpenGL one: drawn inside the same way.
5. A Direct3D 12 game: the log says it draws with Direct3D 12, and the
   number shows in Lootnest's box (counted inside the game).
6. A game with Easy Anti-Cheat or BattlEye (Elden Ring, a BattlEye game),
   GTA V and CS2: the log says "not inside <game>, since ...", and the
   counter shows outside as before (with the frame reader allowed).
7. Turn off **Draw it inside games without an anti-cheat**, start a story
   game: the log has no "inside", and the counter is outside.
8. With Steam's overlay and Discord's on, a story game: both overlays and
   the number work, and the game runs as before.
9. A 32-bit game: the log says it's a 32-bit program; the counter is
   outside.

## 57. Start with a launcher

1. A folder game whose own launcher has to start it: Manage, Launch
   options, **Start with**, choose the launcher's .exe, Save. Manage says
   "Starts with <launcher>.exe"; the game's name, cover and icon are as
   before.
2. Play: the launcher opens (the log says "Starting <launcher> instead
   of <game>.exe"), press its own Play: Lootnest sees the game run (Playing
   in the sidebar, the session on its Timeline when it closes), even when
   the game starts minutes later.
3. The same for a game with profiles: the swap happens, the launcher
   opens, and the profile's saves are kept when the game closes.
4. **Use <game>.exe** goes back to starting the game's own exe.
5. Create desktop shortcut: it starts the launcher, with the game's icon.
6. Move the game to another drive: Start with follows to the launcher in
   the new folder.
7. Start with set to the game itself under another name (a game with
   profiles, Red Dead Redemption 2 with its own exe): Play, close any
   other launcher that opens. "Loading <profile>'s files and starting"
   goes away once the game runs, Close the game works, and the profile's
   saves are kept when the game closes.

## 58. Mods added before profiles were turned on

1. A folder game without profiles: add a mod on its Mods tab, but don't
   apply. Turn its profiles on. The Mods tab shows **Added before profiles
   were on** with the mod. **Bring here**: it moves to the list above, on,
   and the section goes. Lootnest's data folder, `mods`, still has the old
   copy (More, **Open Lootnest's copies of the mods** opens the new one).
2. The same with the mod applied before turning profiles on: the section
   says they're in the game's folder and offers only **Bring all here**.
   After it, the game's files are the mod's, the Mods tab says Modded, and
   **Put the game back as it was** brings back the game's own files.
3. Before bringing them (case 2), Apply on the new list refuses and says
   to bring them first; Play starts the game with the old mods in, and the
   log says the set waits.
4. Turn profiles off again: the Library game's Mods tab shows **Added
   while profiles were on** for mods added with profiles.

## 59. A save brought as a mod

1. Red Dead Redemption 2 with profiles: on the Mods tab, Add a mod and
   choose a save download (a "Prologue Skip" or 100% save: one SRDR3 file,
   or a .zip with one). The drawer says **This is a saved game, not a
   mod** and nothing is added to the list.
2. **Try it as a save**: Try a save opens with that file; Start the run.
   Play the new run: the save is in Story, Load Game, and it sits in the
   run's `Profiles\<id>` folder beside the other SRDR3 files. The other
   profiles' saves are as they were.
3. The same file downloaded from Nexus Mods inside Lootnest: the same
   drawer.
4. A save added before this version: its row on the Mods tab says it's a
   saved game, with **Try it as a save**; remove the row afterwards.
5. A real mod at the game folder's top (a .dll, ScriptHook): added as
   before.

## 60. The window's own bar

1. Lootnest opens with no Windows title bar: the game's art runs to the
   top edge, and the top right has Minimize, Maximize and Close drawn by
   Lootnest. No logo or "Lootnest" caption anywhere at the top.
2. Drag the window by its top strip (over the art, or the sidebar's
   "Lootnest" line): it moves; dragged to a screen's edge it snaps, to the
   top it maximizes. Double-click the strip: maximizes, again restores.
3. The left, right and bottom edges and the corners resize as before; the
   top edge (the cursor turns into arrows there) resizes too.
4. Maximized: nothing is cut off at the top or the sides, the Maximize key
   shows the restore glyph, and a maximized window dragged down by its
   strip comes back to its size.
5. Close follows Close to tray (Settings) as the old X did. Minimize goes
   to the taskbar; the taskbar's click brings it back.
6. Windows 11: the window keeps its rounded corners, border and shadow.
   Windows 10: no white line at the top.
7. Big Picture (F11) and TV mode: no bar and no gap at the top; leaving
   Big Picture brings the bar back. Win+Up, Win+Down and Win+Left work.
8. A second monitor with another scale (150%): drag the window across; the
   bar and keys keep their size and nothing is cut off at the top.

## 61. A game followed under another program's name

1. Red Dead Redemption 2 with profiles, with Launcher.exe under Start
   before (the owner's set-up): Play as a profile. Within a few seconds
   of the game's window, the log says "Red Dead Redemption 2 runs as
   rdr2.exe, not under the names Lootnest knew (...); Lootnest follows
   it", the page shows the game playing, and the FPS counter (if on)
   listens.
2. Save in the game: a "Saved by the game" point comes (spaced 5 min).
3. Quit the game, leave its launcher open: within about 20 s the session
   ends, the log says the profile's files were saved, and the time played
   and the Timeline grow.
4. A game that really doesn't start (cancel in its launcher): after the
   wait the log says "... didn't start for <profile>: Lootnest looked for
   ..., and for a program with a window in <folder>".

## 62. Game page retouches

1. A game in a folder (profiles off) that was played: its page shows no
   Profiles list, and the card on the art says the hours "Played on this
   PC", the same as the sessions on its Timeline added up. Play it for a
   few minutes and close it: the hours grow without reopening Lootnest.
2. A game Steam counts (a Steam game, or one with Steam hours): the card
   and Home's card say "On Steam" with Steam's hours.
3. A game in a folder whose Steam id isn't in the scan (Hydra or a plain
   folder), with Game details on: Will it run shows.
4. A game with profiles: the profile beside Play opens the list; pick
   another profile. The swap screen says "Loading <name>'s files", the
   game doesn't start, then Play, the card's hours, the achievements and
   the Saves tab are that profile's. Everyone's beside the count shows
   all the points.
5. A Goldberg game with local_save.txt beside steam_api64.dll (or
   local_save_path in steam_settings\configs.user.ini): its achievements
   show, and Check why lists the folder.
6. Profiles, Side by side: with two or more profiles, the board and the
   games list match what each profile's page says.
7. Overview, Community (Game details on): reviews as rows, Read all of it
   opens one, More reviews shows the rest.

## 63. A run reads as its person's

1. A game with a run of Cristian (Skip Prologue) loaded: Play says Play
   as Cristian, the profile beside it shows Cristian and a Skip Prologue
   run label, the card on the art says In the Skip Prologue run.
2. The Profiles list on the page shows Skip Prologue run under Cristian.
3. Play: the swap screen shows Cristian with the run's label, and
   "Loading Cristian's Skip Prologue files".

## 64. Runs per game, and where a game is installed

1. With an older run (Skip Prologue, made from a save brought as a mod in
   Red Dead Redemption 2): after the update, the log says "Runs are per
   game now", and the run shows only in RDR2's Who's playing, under
   Cristian, never in Counter-Strike 2's or another game's.
2. Counter-Strike 2 (or Valorant, League of Legends): Who's playing shows
   only people, with no Runs list or New run; the game page has no New
   run, and Manage has no Try a save.
3. RDR2's Who's playing: each person is a column, their runs as rows
   under them; 1 to 9 pick the people first, then the runs. With more
   than five runs the list fades at its foot and scrolls.
4. New run in a person's column: a name, Make it and play; it asks to
   start fresh or from a copy, and the run shows only in that game.
5. The seat menu lists people only; a run picked on a game's page doesn't
   become who's at the PC.
6. A game's page: beside the size, the folder it's installed in (a click
   opens it), and how much is free on that drive; a nearly full drive
   says so in amber.

## 65. Several PlayStation controllers, and games that know them

With **Show a PlayStation controller to games as an Xbox one** on and
ViGEmBus on the PC (section 34).

1. Two DualSenses (USB or Bluetooth) and a game outside Steam that only
   knows Xbox controllers, started from Lootnest: Settings, Controllers,
   Now says "games see your 2 DualSenses as Xbox controllers, one each";
   in the game each pad moves its own player. Windows' Game Controllers
   (joy.cpl) lists two Xbox 360 controllers.
2. TEKKEN 8 (Steam lists DualSense support) started from Lootnest with
   two DualSenses: no Xbox 360 controller appears in joy.cpl, Show recent
   activity says "TEKKEN 8 knows PlayStation controllers itself (Steam
   says so)", and each pad is its own player with PlayStation signs.
3. The same game with Launch options, PlayStation controllers, **Show
   them as Xbox controllers**: two Xbox 360 controllers again.
4. A Bluetooth pad turned off and on while a game runs: it gets the same
   player back (its Xbox controller stays 30 seconds for it).
5. A game closed and another started within 20 seconds: the Xbox
   controllers aren't unplugged and plugged in between.
6. A DualSense on Bluetooth with Lootnest's window open: "Controller
   connected" shows once. Turning the pad off and on, hiding and showing
   the window, or starting and closing a game doesn't show it again. A
   second pad shows it once more.
7. The log has no "The requested resource is in use" after a pad is
   plugged in (Lootnest waits for ViGEmBus to have the pad ready).

## 66. Naming controllers, and Vibrate

1. Settings, Controllers with a DualSense on Bluetooth: **Vibrate**
   makes it buzz for about half a second, and the key says Vibrating
   meanwhile. The same on USB, and with a DualShock 4 on both.
2. One Xbox controller: Vibrate buzzes it. With two, neither row has the
   key (Windows doesn't say whose slot is whose).
3. The pencil: type "Mihai's", Enter. The row shows Mihai's, with
   DualSense, Bluetooth under it. Escape leaves without saving; an empty
   name takes it away.
4. Turn the Bluetooth pad off and on: still Mihai's. Plug a USB pad into
   another port: its name doesn't follow (shown as a new pad).
5. After Vibrate over Bluetooth: Big Picture, the Xbox bridge
   (section 34) and a game started from Lootnest still read the pad.
6. With two named pads and a game shown them as Xbox controllers,
   Settings, Controllers, Now says "games see Ana's and Mihai's as Xbox
   controllers, one each".

## 67. Controller batteries and the battery saver

1. A DualSense on USB: Settings, Controllers shows its charge and
   "charging" in green within a few seconds; unplugged and on Bluetooth,
   the charge alone. The same with a DualShock 4.
2. On Bluetooth, the first look asks the pad for its full reports
   (the log says so if it didn't answer): the charge appears within 30
   seconds. A game, Steam and Big Picture still read the pad afterwards.
3. An Xbox controller on batteries: Low, Half or Full; on its cable,
   nothing.
4. Let a pad run down: at 20% a note says it's running low, at 10% that
   it's almost empty; over a game, as a note over it. Plug it in: no
   more notes; when full, "is charged".
5. Big Picture's bar shows an icon per controller, with the charge when
   it's low or charging, in red when low.
6. Battery saver, Always: the light bar and player lights go out, and a
   DualSense doesn't vibrate in a game (try one with rumble) or from
   Vibrate (the key is gone from its row). Off: the light bar is blue
   again and the game's vibration works.
7. Battery saver, When a controller runs low: nothing changes at 25%;
   at 20% it goes on (Saver beside the charge); plugged in, it goes off.
8. Show how charged controllers are, off: no charge, no notes, and a
   Bluetooth pad turned off and on stays on its short reports.

## 68. Lootnest on Linux

On a Linux PC with WebKitGTK 4.1 (Ubuntu 24.04, Fedora 40 or newer):

1. Unpack `lootnest-linux-x64.tar.gz` and run `./install.sh`: Lootnest
   shows in the applications menu with its icon and opens its window.
   A second start brings the first one to the front.
2. The Library has the games of Steam (a native one and a Proton one),
   Heroic, Lutris and Bottles, each with its source; Library, Sources
   lists those launchers.
3. A Proton game's page suggests saves inside
   `steamapps/compatdata/<id>/pfx`; with profiles on, a swap moves them
   and Play starts it through Steam.
4. A Windows game in a folder starts with umu-launcher, then (Settings,
   Start them with Wine) with Wine, in `~/.wine`; the log says which and
   where. A native game in a folder starts and is seen running and
   closing (Timeline).
5. Heroic and Lutris games start through their launcher and are seen
   running.
6. Close to tray: with AppIndicator, closing leaves the icon and its
   menu works (Open, Play, Quit). Without it, closing quits.
7. A Steam Web API key saved in Settings opens again after a restart
   (the keyring has "Lootnest").
8. `./install.sh --remove` takes the program and menu entry away and
   leaves `~/.local/share/lootnest`.
9. Big Picture (F11) fills the screen, on X11 and on Wayland; F11 again
   puts the window back as it was. Leave it from its menu too.
10. A controller in Big Picture: the stick, the D-pad, A and B move and
   choose. Click another window so Lootnest loses the keyboard: the
   controller still moves Big Picture while its window is the one in
   front (the log says "Controller for Lootnest's page from Go: true"
   only while it has the keyboard).
11. Start a game from Big Picture, press the Xbox or PS button: the Home
   menu opens over the game; Resume brings the game back, and closing the
   game brings Big Picture back.
12. Settings, Controllers lists a DualSense (USB and Bluetooth) and an
   Xbox controller with the right buttons and how each is connected, and
   no Vibrate key.
13. On X11 (an Xorg session), while a game runs: Ctrl+Shift+F5 makes a
   save point with the game's picture, Ctrl+Shift+F12 saves a screenshot
   of the game's window in Pictures/Lootnest/Screenshots/<game>. Once the
   game closes, the keys do nothing (other programs get them again).
14. On GNOME 48 or newer (Wayland), the first game run asks once to allow
   Lootnest's keys; the log has "Keys while you play, from the desktop:"
   with the keys given. Ctrl+Shift+F12 saves a picture of the whole
   screen, and no screenshot file is left in Pictures/Screenshots.
15. On KDE Plasma (Wayland), the same; the keys show in System Settings,
   Shortcuts, where they can be changed.

## 69. Moving a game: small files and the time left

1. Move a large Unreal game (It Takes Two, 45 GB, from one drive to
   another; a hard drive as the target shows it best). Past 90%, the size
   line also says how many files are left; while small files copy, the
   speed is "N files/s", not a few KB/s, and the time left stays a few
   minutes at most, never "About 25 min" at 99%.
2. Time the whole move against 2.23.1 on the same two drives: it should be
   as fast or faster. The log has "Moved ..." and the game plays from the
   new folder with its saves.
3. Stop the move while it copies small files: it stops within a second or
   two, and the new drive has no folder of it (no .lootnest-part).
4. Game Mover (a Steam game): the same speed line and time left.

## 70. Trying a save, and a window that stops answering

1. Red Dead Redemption 2 with profiles: add a Skip Prologue save as a mod
   (or Manage, Try a save) and choose Try it as a save. Play the new run:
   Story Mode continues from the brought save, not from your latest one.
   The run's Profiles\<id> holds the brought SRDR3 file(s) and no other
   SRDR3 slots; your own profile still has all of yours.
2. An archive holding two or more save files (maybe in a folder): they
   land in Profiles\<id> beside the game's saves, not at the top.
3. If Lootnest's window shows "Not Responding" for more than about 8 s,
   the log has "The window stopped answering" with the window thread's
   stack, and `%LOCALAPPDATA%\Lootnest\hangs` a file with every thread;
   Report a problem copies the newest. Dragging the window or holding a
   menu open writes nothing.

## 71. Starting a game with profiles

1. Any game with profiles (Red Dead Redemption 2 with Start with
   Launcher.exe, and one without launch options): press Play. Within a
   few seconds of the game's window the picker says it's playing, the
   window stays responsive, and the log has no "The window stopped
   answering".
2. Close the game: the log says its files were saved back to the
   profile, the time played goes up, and the Timeline has the session.
