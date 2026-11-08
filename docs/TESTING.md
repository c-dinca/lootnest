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

1. Open Lootnest and visit every page in the sidebar, a game's page and
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
