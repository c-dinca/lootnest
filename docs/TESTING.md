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
