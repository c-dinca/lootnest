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

## 7. The rest, quickly

- The sidebar: filter the games, fold it (Ctrl+B), drag its edge.
- Library: the list view (the last button by the cover sizes).
- Cloud saves, if you use them: **Back up now** finishes.
- Quit from the tray icon, open Lootnest again: everything is as you left
  it.
