# Changelog

## 1.76.0

- The sidebar works like Steam's library. The menu at its top picks which games it lists: All games, Installed, Favourites, Not installed, or the games one profile has played or has files for.
- Your collections are folders in the sidebar, each with its count, under Recent. Click one to fold or open it; Lootnest remembers which are folded.
- Drag a game onto a collection to add it. It stays in its other collections; hold Shift as you drop to move it out of the one you dragged it from. Drag a collection's name to change the order. Esc cancels a drag.
- Ctrl+click picks several games and Shift+click a run of them; drag any of them to move them all. Delete takes the picked games out of the collection they're in. They stay in your Library, and Undo puts them back.
- New collection, under your collections, asks for a name right there. Drop games on it to make a collection with them.
- Right-click a collection, or use the More button beside its name, to rename it, move it up or down, or delete it. Deleting a collection only takes its name off the games: every game stays in your Library. Renaming one to the name of another puts the two together, after asking.
- Every game's right-click menu has Add to collection, with a tick by the ones it's in. Favourites is always there and always first.
- The Library's collection shelves and Big Picture's Collections follow the order you choose. Collection names can be up to 40 characters.

## 1.75.0

- A bar across the top of the window, as in Steam: Home, Library, Store, Updates, Achievements and Cloud saves as tabs, with a line under the one you're on that moves when you change. Back and Forward sit beside them, and the search (Ctrl+K) is in the bar on every page.
- Who's at the PC is now your picture at the top right. Click it to choose who's here (or press 1 to 9 while it's open), add a run, or go to Profiles. With one profile, it shows you, your runs and Add a run. Settings is the gear beside it.
- The sidebar is for your games only, from the top of the window to the bottom. Ctrl+B still folds it to a row of covers, and you can still drag its edge.
- In a narrower window, Profiles and your name show as icons, and below 1000 pixels wide the last three tabs go under More.
- Changing pages no longer stretches the page. The page you leave fades out quickly and the new one comes up into place, a part at a time. The bar and the sidebar stay still.
- Menus open from the button you clicked, covers lift a little under the pointer, and buttons press in.
- Lootnest makes a quiet click when you change pages or a game page's tab, open or close a menu, dialog or panel, go Back, press Play, and move between covers with the arrow keys. Never while a game runs. To turn them off, switch off Sounds in Settings, under Games and swaps. That turns off every sound, Big Picture's included.
- With "Animation effects" off in Windows, pages change at once and nothing moves.

## 1.74.0

- Install, uninstall and repair games from Lootnest, through each game's own store. A Steam game's Manage panel and right-click menu have **Verify files with Steam**, and **Uninstall with Steam** while its profiles are off; a GOG game has **Open in GOG Galaxy**, where you install, uninstall and repair it; Epic, Ubisoft Connect and Battle.net games have what their launchers offer; Xbox, EA, Riot, Rockstar and Amazon games open Installed apps in Windows Settings. Each says which happens. Lootnest downloads nothing itself.
- The Library has **Not installed**: the Steam and Epic games you have that aren't on this PC from any store, greyed, each with **Install with Steam** or **Install with Epic**. They come only from the files Steam and Epic keep on this PC: no sign-in, nothing online. A game you install shows in the Library once the scan finds it.
- **Remove from this PC** for a game in a folder of its own, outside any store, in Manage and the right-click menu. Lootnest copies its saves to its backups first, checks each copy, then moves the folder to the Recycle Bin. It won't while the game runs, or for a folder that holds more than the game, such as a drive, Program Files, your user folder or the Desktop.
- Add a removed game again later, in the same folder or another, and Updates offers its saves back.
## 1.73.1

- Updating Lootnest from inside the app works again when GitHub's download link for a release fails, as it did on 8 November with "504 Gateway Timeout". Lootnest now tries the link twice, then gets the same file through GitHub's API, and still checks it against the release's SHA-256 before it runs.

## 1.73.0

- The screenshots and clips you already make with Game Bar, Steam, NVIDIA, AMD and OBS show on each game's Timeline, as "Screenshot (Steam)" or "Clip (Game Bar)" with a small picture, beside your sessions, save points and unlocks. A line at the top counts them by app.
- Click a screenshot to see it large, with the game's other pictures. Click a clip and it plays in the app Windows opens videos with.
- Lootnest matches each capture to its game by Steam's folder for it, by the game's name in the folder or file name, or by when you were playing. It only reads these folders: it never moves, renames or deletes a capture, and nothing goes online.
- Captures it can't match wait in Settings, Data and activity, Unsorted captures. Choose the game for each, and it shows on that game's Timeline, or mark one as not a game. The same place lists the folders Lootnest reads, and a switch turns it all off.
- A game with no captures says on its Timeline that Win+Alt+PrtScn takes a screenshot with Game Bar and Win+Alt+R records a clip. If Game Bar's captures are off in Windows, it says so, and where to turn them on.
- Captures are looked for when the Library is scanned and when a game closes, at most every 10 minutes, and not in safe mode.

## 1.72.0

- A game's page is now in tabs: Overview, Achievements, Saves, Timeline and News. A tab shows only when the game has something for it, so a game with no news has no News tab.
- Overview has the details, when it was last played and by whom, the three latest achievements, the latest save point, how the saves of a game from outside the stores are kept, and the profiles, status, rating, note and collections beside them.
- Achievements lists them all on the page, no longer in a panel at the side. Saves has every save point, to go back to, rename, delete or start a run from, and the copies on this PC, the copies made before swaps, and Try a save. Timeline shows more of it at once.
- Scroll down a game's page and a small bar stays at the top with the game, whose files are loaded, the tabs and Play, so Play is always one click away.
- The arrow keys, Home and End move between the tabs, and Ctrl+Tab and Ctrl+Shift+Tab do too from anywhere on the page. Each game remembers its tab while Lootnest is open, and Back returns to a game's page on the tab you left it on.

## 1.71.0

- Choose your own cover, wide art or logo for any game: open the game, then Manage, Cover and art. Pick a PNG, JPEG or WebP picture of up to 20 MB. Lootnest keeps its own copy, and your picture stays where it was, as it was.
- Remove your cover (or wide art, or logo), under each one you changed, deletes Lootnest's copy, and the game's own art shows again.
- Your art follows the game when Lootnest carries it over to a new copy in another folder.
- Games with only some art use what they have: a game with only wide art gets a cover cut from it, and one with only a cover gets wide art made from the whole cover over a blur of itself, on its page, on Home and in Big Picture.
- A game with no art at all gets a cover set in type: its name, or its logo when there is one, with its store at the bottom, so no game shows an empty box.
- A game's logo, when it has one, stands in for its name at the top of its page.

## 1.70.0

- Ctrl+Shift+F12 while a game runs saves a picture of it to Pictures\Lootnest\Screenshots, in a folder named after the game. A note over the game says it's saved, with the file's name. The picture is taken from outside the game, as any screenshot tool does; nothing goes into the game.
- If the picture comes out black, as it can with a game in exclusive fullscreen, nothing is saved and the note says to switch the game to borderless windowed.
- A save point made while its game runs keeps a picture of the game too. Save points show it, and a click opens it large. Going back to a save point never puts the picture in the game's save folder.
- The Timeline on a game's page lists your screenshots with a small picture of each; click one to see it large.
- Turn the hotkey off in Settings, Games and swaps, Ctrl+Shift+F12 saves a screenshot. Like Ctrl+Shift+F5, Lootnest holds the keys only while a game runs. Pictures stay on this PC.

## 1.69.0

- A game's page can show its details from the Steam store: a summary, the developer and publisher, the release date, its genres, the Steam review score and Metacritic's, and screenshots. Turn on Game details in Settings, Online; it's off until you do.
- The details are read only when you open that game's page, and kept on this PC for 30 days, so the next visit shows them at once. Only the game's Steam id is sent. Games in a folder or from Hydra get them too when Lootnest knows their Steam id.
- Click a screenshot to see it large. The arrow keys go to the next or previous one, and Esc closes it.
- Edit details in Manage changes the summary, developer, publisher, release date or genres. Your changes stay on this PC and always win over Steam's; Use Steam's again puts Steam's back.

## 1.68.0

- Big Picture shows your controller's own buttons: cross, circle, square and triangle with L1, R1, Options and Create on a PlayStation controller; A, B, X, Y with L, R, + and minus on a Switch Pro controller or Joy-Cons, where A on the right chooses and B goes back, as on the Switch; and Menu and View on an Xbox controller. Other controllers show which side of the four buttons to press.
- The buttons follow the controller you last pressed, so two different controllers each get theirs.
- The PS or Home button opens and leaves Big Picture like Options or Start, and a PlayStation controller's touchpad click turns the sounds on or off like Create.
- With DS4Windows, Steam Input or x360ce running, the copy of your controller they add no longer makes one press count twice.

## 1.67.0

- Back and Forward work as in a browser: Alt+Left or Backspace goes back to the page you were on, Alt+Right goes forward again, and so do the back and forward buttons on a mouse. Back puts the page where you had scrolled it, so leaving a game's page returns you to the same spot in the Library.
- The back link at the top of a game's page goes to the page you came from and names it, such as Library or the game you were looking at before. Esc does the same.
- Settings, General lists the new shortcuts.

## 1.66.0

- A game's page has a Timeline under Save points: your sessions (when they started and ended), save points, achievement unlocks and applied updates, grouped by day. It starts with the latest few, and Show more goes further back.
- Each entry says which profile it belongs to when the PC has more than one, for example "Player 2 unlocked Hag Hunter".
- Lootnest now keeps a log of every play session of 60 seconds or more, for games played as a profile and for Library games it starts. The log stays on this PC and only grows; a damaged line is skipped, never deleted.

## 1.65.0

- Safe mode: start Lootnest with --safe-mode, or hold Shift while it opens, or choose Restart in safe mode from the tray icon. It starts with its background work off: no safety net copies, no unlock notes over games, no checks of Downloads or for Lootnest updates, no Steam set-up, no save point hotkey, and no cloud uploads or backups after a game. Profiles still swap when you play. Home says it's in safe mode, with Restart normally to go back. Starting it with --safe-mode while Lootnest is already open restarts that one in safe mode. Use it if something stops Lootnest from starting.
- Lootnest's window only answers its own page. Anything else that ends up in the window can't ask Lootnest to do anything, and nothing the page sends is played back to it.

## 1.64.1

- While a game runs and Lootnest's window is hidden, an achievement unlock and a save point made with Ctrl+Shift+F5 now play their short sound over the game, as they do in the window. Turning off the sounds (Settings, Games and swaps) silences these too, and games with an anti-cheat never get them.

## 1.64.0

- Searching for a game forgives more: accents, apostrophes and punctuation don't matter, words can come in any order, and initials work, so "baldurs" and "bg3" both find Baldur's Gate 3. This holds in the Library, the sidebar's filter, Ctrl+K and wherever else you look a game up by name.
- Sorted by Recently played or Most played, the Library's covers say when each game was last played or how long, in place of the line under the name.
- In the Library, the arrow keys move from cover to cover, and Home and End go to the first and the last.
- F5 looks for games again, as Scan again does.
- Settings, General lists the keyboard shortcuts.
- The first screen says while Lootnest looks for games on this PC, then how many it found.

## 1.63.0

- A game you have from more than one store, such as Steam and Epic Games, or GOG and a copy in a folder, shows once in the Library, the sidebar, search and Big Picture. Its cover has a small mark for each store, and its time played adds up every copy.
- The game's page has Start with, to choose the copy the Library opens and starts. Lootnest remembers your choice; until you choose, it uses the copy you played last, else the Steam one.
- The page says where else you have the game, with a link to that copy's page. Each copy keeps its own saves, play time, status and profiles, exactly as before. In the Library the game shows every copy's collections and status.
- Hide from the Library on such a game hides every copy, with Undo.
- Show copies separately, in Manage, gives each copy its own entry again, with Undo. Nothing on your PC moves or changes either way.

## 1.62.0

- Big Picture keeps you on the couch: when a game you started from Big Picture closes, Lootnest comes back on the whole screen, on the game you played. If you switched to another app or opened Lootnest yourself while playing, it stays out of the way.
- A game's screen in Big Picture has Close the game while it runs, on the X button, and the hints at the bottom say so. Lootnest asks first, with Keep playing chosen, because anything the game hasn't saved may be lost. The desktop's Close the game asks the same way.
- Open in Big Picture when Lootnest starts, in Settings, General. It's off unless you turn it on, and it never applies when Lootnest starts in the tray with Windows.
- The Library has Sources, next to Scan again (also when no games were found). For each store, launcher and folder Lootnest reads, it shows how many games it found, when it last looked, and why it found none: installed with no games yet, or its library couldn't be read. Stores that aren't on this PC are listed on one line. Lootnest only reads what's on this PC; it never installs or fetches anything for a store.

## 1.61.0

- When a game has updated since you last played it, Lootnest makes a save point before it starts, named Before the update, so you can go back if the new version breaks your saves. It notices an update by Steam's build number, or by the game's .exe changing. If the point can't be made, the game starts anyway. Turn it off in Settings, Games and swaps.
- If Windows keeps your Documents folder in OneDrive, Your saves on Home says how many games keep their saves there, with Which games to see them, and those games' pages say so too. Nothing is moved or changed.

## 1.60.0

- Rate your games: a game's page has five stars and a short note under Status, for the profile shown there. Each profile keeps its own rating and note. Click the current star again to clear the rating, or use the arrow keys.
- The Library can sort by Highest rated, with unrated games last. Sorted that way, covers show their stars; in the list view, rated games always show them under the name, and so does Played lately on a profile's page.
- Correct play time a launcher got wrong: on a game's page, click a profile's hours under Profiles and type the right number. Lootnest keeps your correction apart from the time it recorded, so Reset brings the recorded time back. The corrected hours show on the game's page, in the Library, on Home and on the profile's page. Your year in games still counts only the days Lootnest saw you play.

## 1.59.0

- Lootnest opens on its own mark: the nest draws in, the red gem drops into it, and the app fades in as soon as it's ready. The window is Obsidian from the first moment, with no white flash. With reduced motion on, the mark simply shows.
- The installer has the same look as the app: the Lootnest mark, switches for the desktop shortcut and Start with Windows, and a red Install button. While it works, it ticks off each step (Copy Lootnest, Start menu and shortcuts, Add to Installed apps), and it ends with Open Lootnest.
- If installing fails, the installer shows which step stopped and why, with Try again.
- `Lootnest-Setup.exe --silent` (or `/S`) installs without a window, for scripts and package managers. It keeps your current choices on an upgrade and doesn't open Lootnest afterwards.
- Releases are ready to be code signed through SignPath Foundation and published to winget, which over time will stop Windows SmartScreen from asking about Lootnest. Both stay off until they're set up.

## 1.58.19

- On a short window, Who's here in the sidebar is one row of pictures, with the seated profile's name, so the games list keeps its room. Click a picture to sit that profile down, as before.
- A game's page and Big Picture fit windows of 1024 by 700: the art is shorter on short windows, the other ways to play take their own row in the play bar, and side panels use the whole width on narrow windows.
- The first start sits in the middle of the window, with your profile and the welcome side by side and the Lootnest mark at the top. On a narrow window they stack.

## 1.58.18

- Home, a game's page and the Library leave more room between sections than inside them, so each section reads as its own block.
- A game's Manage panel fits a 900 px tall window without scrolling. Its groups stand apart, and Danger sits a little further down.
- The page behind side panels, such as Manage, stays readable.
- Try a save has its own icon, and the lines under each action in Manage are shorter.

## 1.58.17

- Text comes in fewer sizes with clearer steps between them: page titles, section titles, then body text and secondary lines.
- Small titles, such as a game's name on a missing cover or the headings on Updates, use the same face as the rest of the text, so they read more clearly at that size.
- Update notes are set in the same light grey as the rest of the text.
- Longer notes and profile bios wrap at a comfortable line length.
- Tiny text in the sidebar and in Big Picture's button hints is a little larger.

## 1.58.16

- Cloud saves opens on any save that changed on two PCs, with when it changed on each and the two choices, Keep this PC's or Use the other PC's. The buttons wrap instead of being cut off.
- Sync now sits beside a one-line summary of where your saves stand.
- The cloud folder's path keeps its start and its end when it's long, and a button copies the whole path.
- Each switch on Cloud saves has a one-line explanation. More shows the rest.
- A profile's page shows the art of its most played game, unless you chose another banner (Edit profile, Banner, where Colour still gives your colour). With no art, the page starts with the name.
- A profile's numbers are a list beside Activity and Played lately, so nothing is cut short. Favourite games shows only once some are picked, from Pick favourites.
- The first start is a full window: your profile's picture, name and colour on the left, with a preview of your Play button and your place in the sidebar, and what Lootnest does on the right. More profiles and runs are added later in Profiles.

## 1.58.15

- Settings has one title, the name of the category you're in. The version is under About.
- The categories in Settings are a plain list, and each switch sits right after what it says, not at the far edge of the window.
- Profiles is a list of people, a row each: their picture and name, their hours and games, then as many of their recent games as the window fits.
- A profile's banner shows faintly behind its row, only when it has one.
- Runs sit under their person as short rows.
- How many profiles you have, out of 16, sits beside the Profiles title.

## 1.58.14

- The Store opens on today's deal across its full picture, with the name and price over it, and Top sellers on Steam beside it as a numbered list of ten.
- Deals on Steam and Best deals in other stores are each one row of pictures that scrolls sideways, without boxes around them.
- Search results in the Store use the same plain pictures, without boxes.

## 1.58.13

- The Achievements page opens on one line: whose achievements they are, how many are unlocked out of how many, and one bar. The ring and the four numbers beside it are gone.
- With up to four profiles, you pick whose achievements to see from a row of names. With more, it's a menu.
- Games with achievements are a list, a row each: the cover, the name, a bar and the count.
- Rarest unlocked and Latest unlocked sit in a column beside the games, as rows. Long achievement names show in full.
- Lootnest milestones are folded away under the games, with how many are done. Open them to see each one as a plain row.

## 1.58.12

- The Updates page is one list of your games, a row each: the game's icon, its name, where it is, your version against the latest and one button. Long folder paths keep their start and their end.
- The counts at the top of Updates are one row of filters: All, Newer version out, Up to date and Can't tell.
- Only the first game with a newer version out has the red button. The rest are plain.
- What needs a look, Check again and when Lootnest last checked sit in a narrow column beside the list. On a smaller window they come first, above it.
- Updates found in your Downloads are rows like the rest.

## 1.58.11

- Pages with nothing in them yet start on the same left edge as every other page, without a box in the middle. The empty Library shows dashed covers where your games will go, and one line on where Lootnest looked.
- Home before anything is played lists three first steps: the games Lootnest found (or Scan again), adding a game from a folder, and a profile for everyone who plays, or a new run when you play alone. Who's playing sits beside them.
- The Achievements page with nothing found yet looks like the other empty pages.

## 1.58.10

- Big Picture is easier to read from the sofa: larger tabs that start beside the name, and room at every edge of the screen.
- The game in focus in Big Picture has a clear light outline that shows on dark and bright pictures alike, without the red glow.
- The prompts at the bottom of Big Picture no longer cut into the last row. Rows fade out above them, and "Play as…" now reads "Play as someone else".
- Collections with five games or more keep their own row in Big Picture, up to three. The smaller ones share one Collections row. Choose one to see its games in the Library, and press B to go back to it on Home.

## 1.58.9

- The Library's title, its game count and the search sit on one line.
- A collection gets its own shelf in the Library only when it has four games or more, up to three shelves. Their covers are the same size as the rest of the Library, and the last one runs off the edge when there are more to scroll to. Smaller collections are still in the collections menu.
- The Library's filters fit on one row on a wide window. With more than three profiles, whose games you see is a menu instead of a row of names.
- The cover size buttons show three filled covers, small to large, and a list.
- The Library fits one more column of covers on a 1440 pixel wide window.

## 1.58.8

- A game's achievements sit on the page like the rest, without a box around them. The profiles beside them stay in view as you scroll, when they fit.
- A game's picture stays at the top of its page, and the rest of the page is the same dark colour whatever the art, also as you scroll. On Home, the blurred art behind the page fades to dark sooner.
- A game's updates and news are a short list of up to six posts, two to a row, as on Home. Each one still opens in full with a click.

## 1.58.7

- A game opened from the sidebar, search or Achievements marks Library in the sidebar, and its Back button says Library.
- The line under a game's name says how many profiles it has instead of listing every name.
- A game's play bar is one row: Play, whose files are loaded, Someone else to pick another profile, and New run.
- Each profile's hours in a game sit beside its name under Profiles, with the Steam account's hours and a link to the copies made before swaps.

## 1.58.6

- Home's play bar is simpler for a game without profiles: Play, when you last played it and for how long, and Turn on profiles. "Profiles off" shows once, under the game's name, and the name opens the game's page.
- Continue playing is one row of wide pictures that fits the page, with the game's news under it. Who's playing, Tonight and Your saves sit in the column beside them.
- Click a person under Who's playing to sit them at the PC. The one sitting there is marked Seated, as in the sidebar.
- Your saves shows how many games Lootnest looks after in three figures, and a line only for what needs a look, with one link each. "Choose where games keep saves" now lists every such game, not only the first eight.
- News on Home is four short posts, two to a row, without the large one at the top.

## 1.58.5

- A game with no picture no longer opens on a dark empty band. Its page and Home show a soft wash of the game's own icon, or its first letter set large.
- Covers without a picture show the game's name, and its icon when Lootnest can read one, in the Library, on Home and in Big Picture. Big Picture no longer shows an empty grey box.
- When your last game has no picture and the one before it has one, Recent games in the Library shows that one first.
- A Store picture that doesn't load shows the game's name instead of a broken image.

## 1.58.4

- Pages come in with one short movement instead of every card and row arriving one after another.
- Achievement rings and bars fill, and numbers count up, only the first time you open a page, not on every visit.
- The rarest achievements shine once when you unlock them, instead of shining over and over on every page.
- Rows and cards stay still under the pointer: a game's picture in the Library list and on Updates no longer grows, a game's card on the Achievements page and an unlocked achievement's icon no longer lift, and the Updates figures no longer shrink when clicked. Covers in the Library still lift.

## 1.58.3

- Racing Red is kept for a few things on each page: the main button, the game's progress and the page you're on in the sidebar. It no longer marks a cover or the scrollbar you point at, and page titles have no red line under them.
- The Update tag on patch notes and the discount on store prices are plain grey tags now, and a discount reads as "-50%".
- In Settings, the category you're in is a plain highlight. The sidebar's current page keeps its red bar and icon on a grey fill instead of a pink one.
- On the Updates page, Apply the update is the only red button; a newer version keeps a small red dot without the pulse, and the card has no red edge. Review and apply, Carry everything over and Undo the update are plain buttons.
- On the Achievements page, the summary keeps its red ring and drops the long bar under it, which showed the same figure. Each game's ring, here and in Big Picture, and Lootnest's milestones are grey. A game at 100% keeps its gold ring.

## 1.58.2

- Panels, cards and pop-up dialogs are solid now instead of see-through, so text on them is easier to read. The Library's filters bar stays put at the top of the page without the games showing through it, and lines up with the grid.
- Fewer glows and stripes. Achievement cards, the rarest unlocked and the Updates summary are plain panels; a game at 100% keeps its full gold ring.
- An achievement unlock in the window shows the profile's colour as a small dot instead of a coloured edge.
- Warnings, such as Steam Cloud replacing a profile's files or a game closing right after an update, show a warning sign instead of a coloured stripe down the side.

## 1.58.1

- Every page now starts at the same distance from the sidebar. The Store, Updates and Achievements pages, and Home before you've played anything, sat a little further to the right than the Library, Home and Cloud saves.

## 1.58.0

- **Updates you already downloaded, found for you.** For games no launcher keeps up to date, Lootnest looks at the names in your Downloads folder (and the folders directly in it) and spots an update for one of your games: a folder, a .zip, .7z or .rar (the first part of a set), or an installer. It tells you once with a notification, after that game closes if it's running.
- The Updates page has a Found in your Downloads list, with the version each one moves you to. Review and apply opens the usual update panel with the file already chosen, so the saves are copied first and Undo works as before. Ignore hides one until the file changes.
- A game's page says when an update for it is in your Downloads, with the same Review and apply.
- Once an update from Downloads is applied, or for a file with the version you already have, Delete the downloaded file moves it (every part of a .rar set) to the Recycle Bin.
- Lootnest reads only names, on this PC, and looks nowhere else. Turn it off in Settings, Games and swaps.
- Versions in a folder's name, such as "Hollow Tale v1.3", are read correctly when you choose a folder as an update.

## 1.57.0

- **Real sounds in Big Picture.** Moving, choosing, switching tabs and going back play short recorded sounds instead of the old beeps. They also play when your PC is set to reduce motion; the sound button at the bottom of Big Picture still turns them off.
- **A sound for unlocks, save points and swaps.** Lootnest's window plays a short sound when an achievement unlocks (with an extra chime for ultra rare ones), when a save point is made and when a profile's files are loaded. Never for games with an anti-cheat. Turn it off in Settings, Games and swaps.
- The sounds are Kenney's Interface Sounds, free to use (CC0), and come inside Lootnest: nothing is downloaded.

## 1.56.0

- **Games a scan finds show up at once.** Home, the sidebar and the Library's count update as soon as a scan ends, on any page. Scan again says how many new games it found, or that there were none.
- **Hide can be undone.** Hiding a game from the Library shows a note with Undo. The note stays while the pointer or the keyboard is on it.
- Settings, Games and swaps lists the games you hid, each with Show again.
- Right-click works on every game in the sidebar, in the Library's list, in Recent games and on Home, not only on covers. The menu also has Create desktop shortcut.

## 1.55.4

- **On your own, you're always the one playing.** With runs of your own and nobody else, Lootnest seats you from the start. Clicking yourself or a run in the sidebar makes Play start as that one, and the seat is never left empty. Big Picture's Profiles tab uses the same words.
- In Try a save, choosing someone else in Start from now updates the sentence above it and the suggested name of the run.
- If starting a new run fails, the name you typed stays in the box next to the error. Cancel can't be clicked while the run is being made. If the dialog is closed then anyway, the run stays in Profiles and the game doesn't start.
- The roulette in What do we play tonight shows a different cover on every step. Clicking Another again while it spins doesn't start it over.
- Year in games says "1 day in a row" instead of "1 days", and Save the picture waits until the picture is finished. Its button can't be clicked again while it saves.
- The Tab key stays inside an open dialog, panel or Big Picture, and closing a dialog puts the keyboard back on the button that opened it.

## 1.55.3

- **Big Picture opens reliably.** Pressing F11 or Start twice in quick succession no longer leaves Big Picture on with an empty screen that takes every key.
- Back in Big Picture returns to the game you opened, instead of the first one in the list.
- Dialogs that open while Big Picture is on show above it, and the keyboard and a controller reach their buttons.
- With no games yet, Big Picture's Home and Library say so and offer a button to leave.
- The background in Big Picture no longer blinks to black when Lootnest updates something in the background.
- The View button on a controller turns Big Picture's sounds on and off.

## 1.55.2

- **Starting a run from a save point works again.** The name Lootnest gives the new run, such as "Ana from Before the boss", now fits the 24 characters a profile's name can have, and gets a number at the end when another profile has it. Before, a long or repeated name made the run fail to start.
- **Try a save works for people with longer names.** The run's name is now, for example, "Alexandra trying", shortened to fit when needed and numbered when it's taken, and the name box stops at 24 characters.
- A folder, .zip, .7z or .rar with one save file in it now counts as that file. It can go where the game keeps a single save file, and in a save folder it joins your files instead of taking the place of the whole folder.
- When a save brought to Try a save holds one folder, and the game's save folder holds only folders (save slots), it's kept as a slot of its own.
- If starting a run from a save point fails, its half-made files are removed.
- Try a save doesn't open an archive that holds more than 2 GB: that's too big to be a save.

## 1.55.1

- **Putting saves back is safer.** When you put back a copy from **Copies on this PC** or from the cloud folder, Lootnest now first copies every save location that copy will write to, including ones the game doesn't use on this PC yet. Before, only the locations the game knew of here were copied first.
- After putting back a copy from this PC, Lootnest keeps all the extra copies it held on to when a game's saves went missing from its folder. Before, the next check could clear out older copies you might still need.
- When a game's saves go missing from its folder, the notice now gives the date of the copy Lootnest has, or no date at all, instead of 1 January when it didn't know one.
- If putting back an encrypted copy fails, for example with the wrong password, the unpacked files are removed right away. Ones left by a crash are removed when Lootnest next starts.

## 1.55.0

- **Your year in games.** A profile's page has **Year in games**: a picture of that profile's year, ready to share.
  - It shows the hours played, the days you played, how many games, and your longest run of days in a row.
  - Then your most played game, with its cover and art, the four after it, and each month.
  - **Save the picture** puts it in Pictures\Lootnest. It's made on your PC from what Lootnest counted, and nothing is sent anywhere.
  - Choose another year when there's more than one.

## 1.54.0

- **Tonight, on Home**: one game to play, picked for whoever's here from their Up next, what they're playing and what they never started. Finished, 100% and dropped games are left out. **Another** spins for a different one; the pick holds for the day.

## 1.53.0

- **Try a save safely.** Got a save from a friend, or one from a mod site? A game's **Manage**, **Try a save** puts it in a new run of yours: your settings, with the save you brought. Your own saves stay exactly as they were, and playing as you again is one click. Keep the run, or delete it in Profiles.
  - Bring a folder, a single save file (it joins your other saves in that folder), or a .zip, .7z or .rar.
  - Choose whose files the run starts from, and, for a game with more than one, where the save goes.

## 1.52.0

- **Copies of every game's saves, on this PC.** For games outside Steam (in a folder, from Epic, GOG, Hydra or added by hand), Lootnest now copies all the saves it knows of whenever they change and the game is closed: the ones in Documents or AppData too, not only those inside the game's folder. The last five are kept, and nothing is sent anywhere.
  - Put one back from the game's page, under **Saves**, **Copies on this PC**. What's there now is copied first.
  - Home's **Your saves** says how many games Lootnest keeps copies for.
  - Locations bigger than 256 MB outside the game's folder are left out, since a folder named like the game can hold caches or mods.

## 1.51.0

- **Lootnest starts with one profile: yours.** It's named after your Windows account, and there's no question about who uses the PC. Rename it, or add someone, in Profiles. Sharing the PC? **Add someone else** is on the first screen too.
- **Runs, for everyone.** A game's page has **New run**: another playthrough of the game with its own saves and settings, such as "Cristian hardcore" or a new game+. Start it from nothing or from a copy of your saves; yours stay as they are. Switch between your runs from the game's page.
- **When it's just you**, Who's here is gone from the sidebar and Play needs no choice. It comes back when you add someone.
- A game without profiles says **Keep runs** instead of asking who plays it.

## 1.50.0

- **Put back the files from before any swap.** A copy is made before every swap; now you can use them. **Manage, Copies before swaps** (or **See the copies** under the Play bar) lists them, newest first, with whose files each holds.
  - **Put back** gives a copy's files back to that profile: into the game's folders if their files are loaded, else into their profile for the next time they play.
  - What's there now is copied first, so putting a copy back can be undone too.
- **When Steam Cloud replaces a profile's files**, the game's page now says so until you've read it: when it happened, that the profile stayed as it was, and where that session's files are kept. Before, it was only a passing message.
- **Games with an anti-cheat** (Easy Anti-Cheat, BattlEye) say so on their page, with what Lootnest does there: it never touches the game's own files, swaps only its settings and saves, and shows no notes over it.

## 1.49.1

- **Keep it offline works.** Windows Firewall refused the rules for a game in some folders, and **Let it go online again** said the firewall didn't take the change when the rules were already gone. Both are fixed.
- **Lootnest is now tested on Windows itself** before every release:
  - the swap while a game still holds its save open, saves marked read-only, and save folders with very long paths;
  - the note over a game, the Ctrl+Shift+F5 key, and running as administrator;
  - Windows Firewall rules.
- A checklist for trying Lootnest on your own PC is in `docs/TESTING.md`.

## 1.49.0

- **Big Picture**: Lootnest as a console's screen, on the whole screen, for playing from the sofa with a controller.
  - Open it with the controller button at the top of the sidebar, with F11, from the tray icon, or by pressing **Start** on a controller.
  - **Home** has rows to move through: Continue playing on wide art, your collections, and your games. The background takes on the art of the game in focus.
  - Choose a game for its own screen, with **Play** as whoever's sitting down and **Play as someone else**.
  - **Library** has every game, **Achievements** a ring for each game, and **Profiles** lets whoever's at the PC sit down.
  - Controls: the stick or the D-pad to move (hold to keep moving), **A** to choose, **B** to go back, **Y** to play as someone else, **LB** and **RB** to switch tabs, **Start** to leave. On a keyboard: the arrows, Enter, Escape, Q and E.
  - Soft sounds as you move and choose, made by Lootnest itself; turn them off at the bottom of the screen.
- Connecting a controller says that Start opens Big Picture.

## 1.48.0

- **Achievements, the way you know them from consoles.**
  - Each one wears its tier, from how many players have it: **Ultra rare** in gold (under 2%), **Rare** in blue, **Uncommon** in green, **Common** in silver. You see the tier and a bar of how rare it is, and ultra rare ones catch the light.
  - Locked achievements are grey, with a lock. Hidden ones stay covered until you click them.
  - The Achievements page opens on a ring of how much you've unlocked, with the numbers counting up. A new **Rarest unlocked** row shows the four your profile should be proudest of.
  - Each game has a ring of how much of it is done; gold at 100%.
- **Unlocks in the window.** When a game unlocks an achievement while Lootnest is open, a note slides in at the top right of Lootnest's window too, with its tier and how rare it is.
- **The note over the game, redone.** It now slides in from the right, with a trophy in the tier's colour, the achievement's name, and how rare it is in that colour.
- **Show one** shows both: the note in the window and the one over the screen.

## 1.47.0

- **The Updates page, redone.**
  - At the top, how many games Lootnest keeps track of, how many have a newer version out, how many are up to date and how many it can't tell. Click any of them to show only those games.
  - **Needs a look** gathers what's waiting for you: a new copy of a game to carry over, or an update after which the game closed right away, with Undo.
  - Each game is a card with its art, where it's installed, and its state: newer version out, up to date, or can't tell. Your copy's version sits beside the latest, the latest patch notes are one click away, and there's one clear button.
  - **More** on each card has the game's update history, Undo the last update, and Copy its saves first.
  - **Update history** lists every update Lootnest applied to a game, newest first, with what each one changed.
  - **Check again** looks for newer versions right away.
- **Empty pages look the part.** An empty Library now shows a proper starting point, with what Lootnest looks for and the two ways to fill it. The same goes for Updates when there's nothing to keep up to date.

## 1.46.0

- **Your games in the sidebar**, as in Steam's library.
  - Every game is in a list under the sections: what's being played now, the last five played, each of your collections, then all of them. Each has its cover, and a dot in the colour of whoever's files are loaded.
  - **Filter games** narrows the list as you type; the arrow keys move from game to game.
  - Sort by name, last played or time played. Fold any group.
- **Fold the sidebar** to a rail of icons with the button at the top, or Ctrl+B. Drag its edge to make it wider or narrower; double-click the edge to reset it. Lootnest remembers both.
- **The Library as a list**: next to the cover sizes, a list with a row per game, showing when it was last played, how long, and who has a profile, with Play on each row. The Library's filters stay at the top while you scroll.
- **Motion**: pages slide in when you move between them, covers and blocks come in one after another, and covers lift under the pointer. With Windows set to show fewer animations, they don't move.
- **Show one**, the button that shows what an unlock looks like over a game, now says what it did. If the note can't be shown over the game, Lootnest writes why in its recent activity and shows it as a Windows notification instead.

## 1.45.0

Lootnest is now in **beta**: everything it was built to do is in, and the work from here is making it right on as many PCs as possible.

- **Your saves, on Home.** A short list of how Lootnest looks after the saves of everything in your Library:
  - games with profiles, with a copy before every swap;
  - games that keep their saves in their own folder, which the safety net copies;
  - games with save points;
  - a game whose saves are gone from its folder, with **Put a copy back**;
  - games from outside the stores whose saves Lootnest can't find yet, each a link to its page;
  - whether cloud saves are on.
- **Report a problem**, in **Settings, About**, copies Lootnest's version and its recent activity for you to paste, and opens a new issue on GitHub in your browser. Lootnest sends nothing itself.
- **Beta** shows next to Lootnest's name, and in Settings with the version.
- Save points of games without profiles now also take the saves a game keeps inside its own folder, as the safety net does.

## 1.44.0

- **Save points.** Each game's page has **Save points**: a copy of its saves as they are now, with a name such as "Before the boss", to come back to.
  - **Ctrl+Shift+F5** makes one while you play, with a note over the game when it's done. Lootnest holds those keys only while a game runs, and **Settings, Games and swaps** turns them off.
  - Each point says when it was made, whose saves they are, how long had been played and how many achievements were unlocked by then.
  - **Go back to it** puts those saves back. The saves as they are now are kept as a point first, so going back never loses anything. Lootnest waits for the game to be closed.
  - **New run from it**, for a game with profiles, starts a new run of that person from that moment: a profile of its own, such as a hardcore try from the last boss. Everyone else's saves stay as they are.
  - Rename or delete points from **Show all**.
  - A copy made while the game runs is checked against the saves afterwards, and made again if the game was writing to them.
- For games without profiles, save points use the save locations from the game's **Cloud save** section.

## 1.43.0

- **Keep a game offline.** **Manage, Keep it offline** has Windows Firewall stop a game from connecting to the internet and to other PCs on your network. It's for games Lootnest starts itself: games in a folder, from Hydra, or added by hand.
  - Lootnest finds the game's programs in its folder: the game itself, its launcher and its helpers, without installers and uninstallers. It makes one rule for each, up to 30.
  - Windows asks first, as administrator. The rules are in the **Lootnest** group, so you can see them in Windows' own firewall settings too.
  - The game's page says **Kept offline**. **Let it go online again** takes the rules away.
- In the Manage panel, the actions to do with going online are in a group of their own.

## 1.42.0

- **Why no achievements? An achievement check.** A game from outside the stores with no achievements on its page now has **Check why**. Games that have achievements get **Check**. It goes through each step and says which one is missing:
  - which Steam game it is, and where Lootnest found its id;
  - the Steam-compatible tool next to the game, and whether Lootnest reads its files;
  - the achievement file that tool keeps: where it is, when it was written, and how many it has unlocked, or that it isn't there yet;
  - where the names and pictures come from, or that a Steam Web API key would add them;
  - whether each profile has its own;
  - whether toasts show over it.
  It only looks. Nothing changes unless you press a button.
- **Your profile's name in the game.** For a game with a Goldberg-style tool (a steam_settings folder), the check shows the name the game uses, which is also what friends see on LAN. **Use Cristian's name** sets it to the seated profile's name. Its settings files are copied to backups first.

## 1.41.0

- **Achievements as they unlock.** While you play a game whose achievements a Steam-compatible tool keeps on this PC (a game in a folder, from Hydra, or added by hand), a small note shows over the game the moment one unlocks: its name, how rare it is, and the colour of the profile playing.
  - It's a window of Lootnest's own, at the bottom right of the screen the game is on. It never takes the focus or a click, and nothing goes into the game.
  - It shows over windowed and borderless games, and over most fullscreen ones on Windows 10 and 11. A game in exclusive fullscreen draws over it: switch that game to borderless.
  - Never for a game that ships with an anti-cheat (Easy Anti-Cheat, BattlEye).
  - **Settings, Games and swaps, While you play** turns it off, and **Show one** shows how it looks.
- **How rare achievements are**, if you turn it on in **Settings, Online**: each achievement shows the share of Steam's players who have it, on the game's page and as it unlocks: ultra rare, rare, uncommon or common. Only the game's Steam id is sent, and no key is needed.
- Who's at the PC (**Who's here**) is now whose colour an unlock shows in, for games without profiles.

## 1.40.0

- **Updates that come as an installer.** **Apply an update** now also runs an update's installer (an .exe you have). Lootnest:
  - copies the game's saves first and, if there's room on the drive, the whole game, so the update can be undone;
  - runs the installer and waits for it to finish. Point it at the game's folder;
  - records which files it added, changed or removed, and keeps only the old versions of those once it's done.
  - Without the copy of the game, the saves are still kept, but that update can't be undone.
- **Apply another update** right after one, for games that need several in a row.
- **A check after the first start.** The first time a game starts after an update, Lootnest watches it. If it closes within a few seconds, Lootnest says so and points to **Undo**, since that usually means the update doesn't fit the copy.

## 1.39.0

- **A new copy of the same game.** When a game in a folder is reinstalled as a whole new version, in a new folder or beside the old one, Lootnest notices it's the same game, by its name or Steam id, and asks: **Is this a new copy of Phantom Blade Zero?** It asks on the Updates page and on the new copy's page.
  - **Carry everything over** moves:
    - the saves the old copy kept inside its folder, into the same places in the new one. Whatever the new copy had there is kept in backups first. If the old folder is already gone, the newest copy the safety net kept goes in instead;
    - for a game with profiles, the game now starts from the new copy, and the files of whoever owns it go in;
    - its playtime, when it was played, its status, collections and settings.
  - Then **Delete the old folder**, if it's still there, copies its saves to backups first.
  - **Not the same game** stops asking.
- **Reinstalling it? Copy its saves first**, on each game on the Updates page, copies all of a game's saves to backups in one click before you reinstall it.
- Many games keep their saves inside their own folder, Phantom Blade Zero among them, so deleting the folder deletes the saves. Lootnest now covers that from start to end.

## 1.38.0

- **Updates in .7z and .rar.** **Apply an update** now takes a .7z or .rar as well as a folder or a .zip, so there's nothing to extract first. For an update in parts (.part1.rar, .part2.rar, ...), choose the first part. Lootnest opens it next to the game's folder, shows what it changes, and removes what it opened once the update is in.
- **Updates in the right order.** Lootnest reads the versions in an update's name, such as "v1.0.2 to v1.0.3" or "Build 16251443", and your copy's version from the last update it applied or from the game folder's name. Before anything changes, it says:
  - **The next update for your copy**, when it fits;
  - that the update is for another version than yours, when one would be skipped, since that usually stops a game from starting;
  - that your copy already has it, or has something newer.
- **Each game's update history** on the Updates page, such as "1.1 → 1.2", with its version. **Undo** takes updates off one at a time, newest first.

## 1.37.0

- **Updates**, a new page in the sidebar, for games no store keeps up to date: games in a folder, added by hand, or from Hydra.
  - It lists the games with a newer version out, from their patch notes on Steam (with **Show game updates and news** on), and the sidebar shows how many.
  - **Apply an update** takes an update you have, as a folder or a .zip. Lootnest doesn't download updates.
  - Before anything changes, it shows what the update does: how many files it replaces and adds, and where they go. An update often comes inside a folder of its own, and Lootnest puts what's in it into the game's folder. Notes such as README.txt are shown, not copied.
  - It copies the game's saves and every file it replaces first, checks each new file as it goes in, and puts everything back if something fails.
  - **Undo** takes the update back out: the files it added go, and the ones it replaced come back.
  - A game's page offers **Apply an update** too, under "A newer version is out".
  - .rar and .7z archives need to be extracted first. For an installer (.exe), run it yourself.

## 1.36.1

- A game's news no longer shows bits of code such as `[p align="justify"]` in the short text under its title.

## 1.36.0

- **Will it run here? Now with an answer.** A game's page in the Store says how it should run on this PC:
  - **Runs great here**, **Runs well here**, **Runs, on low settings**, **May struggle** or **Probably won't run well**;
  - which part holds it back, such as "The graphics card holds it back";
  - a bar for the graphics card and the processor, with lines for the minimum and recommended requirements, so you see how far above or below them this PC is.
- Lootnest knows most graphics cards and processors of the last fifteen years, desktop and laptop, NVIDIA, AMD and Intel, including graphics built into the processor. It compares them using typical benchmark results, so the answer is an estimate. When it doesn't know a part, it says so and shows the names side by side.
- Memory and free space still count: too little of either lowers the answer.

## 1.35.0

- **Search from every page.** The **Search** box at the top of the sidebar (Ctrl+K) finds your games as you type and, with the Store on, any game in the store too. Pick one with the arrows and Enter: your games open their page, the others open their prices and requirements. Press Enter to see everything in the Store.
- **Find games is now the Store**, with something to look at before you search:
  - today's deal on Steam;
  - the deals and top sellers on Steam, in your currency;
  - the best deals in other official stores, on games with good reviews (from CheapShark, in US dollars), with each game's Metacritic score.
- Games already on this PC are marked everywhere in the Store.

## 1.34.0

- **Find games**, a new page in the sidebar. Look up any game, installed or not, and open it to see:
  - **its prices**: on Steam in your currency, and in other official stores such as GOG, Humble and Fanatical (from CheapShark, which lists authorized stores only, in US dollars), with the discount and the lowest price it has had;
  - **will it run here?**: its minimum and recommended requirements beside this PC's processor, graphics card, memory and free space. Lootnest says whether the memory and the space are enough; for the processor and graphics card, you see the names side by side;
  - a link to its **PCGamingWiki** page, for known problems and fixes.
- **Open** takes you to that store's page in your browser. Games already on this PC are marked, and **Open its page** takes you to them in the Library.
- It's off until you turn it on, on the page itself or in **Settings, Online**. Only what you type, or the game's Steam id, is sent. Your hardware is read on this PC and never sent.

## 1.33.0

- **Games installed with Hydra are in the Library.** Lootnest reads Hydra's own list of the games it installed on this PC, like it does for itch.io or GOG, and shows them with Hydra as their store. Pick **Hydra** in the store filter to see only those.
  - Hydra can stay open: Lootnest reads a copy of its list and never changes it.
  - They get everything a game in a folder gets: profiles, cloud saves, the safety net for saves kept in the game's folder, and achievements.
  - Hydra knows each game's Steam id, so Lootnest finds its save locations and achievements more often.
  - Only games already installed are listed. Lootnest doesn't download anything.

## 1.32.0

- **A shelf for each collection in the Library**, as in Steam. Under **Recent games**, each of your collections, such as "Co-op" or "Weekend", has a row of its games, the ones played last first. Scroll a row sideways to see more, or press **See all** to show only that collection.
  - The six largest collections get a shelf. The others are in the **All collections** menu, as before.
  - Then **All games** shows the whole Library with its filters.
  - Searching or filtering the Library shows only the games you asked for.

## 1.31.0

- **Recent games, at the top of the Library**, as in Steam. The last five games you played are on their wide artwork, each with who played it last, when, and how long it's been played in all. Point at one to get a red **Play** button, or click it to open its page.
  - It shows when the Library opens. Filter the Library, or search it, and only the games you asked for are left.
  - The filters now sit under it, right above the games they filter.
- The **Play** button on a cover in the Library turns red when you point at it.

## 1.30.0

- **Settings, one part at a time.** Settings now has a list on the left, as in Steam: **General**, **Games and swaps**, **Online**, **Updates**, **Cloud saves**, **Data and activity** and **About**. The one you pick fills the right side, and Lootnest opens it again next time. A red dot on **Updates** says a new version is ready.
- **Panels at the side scroll again.** A long list, such as a game's achievements after **Show all**, could run off the bottom of the window. It now scrolls inside the panel, and the bottom row of buttons stays in view.
- **More Racing Red.**
  - The page you're on, in the sidebar and in Settings, has a red bar beside it.
  - Every page title has a short red stripe under it.
  - A cover you point at gets a red edge.
  - News marked **Update** has a red tag.
  - A game at 100% on the Achievements page has a red edge.
  - The scrollbar turns red when you point at it.

## 1.29.0

- **A safety net for games that keep their saves in their own folder.** Many games outside the stores do, such as Ren'Py and RPG Maker games, so reinstalling one, or updating it by replacing its folder, used to take the saves with it. Lootnest now keeps a copy of those saves on this PC whenever they change and the game is closed, the last 5 of them.
  - They're on the game's page, under **Copies kept by Lootnest**. **Restore** puts one back; the saves there before are kept in Lootnest's backups first.
  - If a game's saves are gone from its folder, Lootnest tells you once and keeps more copies from then on, so the ones from before stay.
  - Lootnest finds them in the save locations you chose, the ones PCGamingWiki lists, and folders such as `game\saves`, `www\save` and `Saved\SaveGames` inside the game's folder.
  - It's always on. Nothing is sent anywhere, and saves over 1 GB aren't copied.
- **A profile is never emptied by a reinstall.** If a game's files are gone from the PC when Lootnest saves a session (the game was reinstalled, or its saves deleted), the profile's last copy is kept in the game's backups first, as that profile's, and you get a notification.

## 1.28.0

- **Racing Red buttons.** The main button on each page, such as **Save**, **Sync now** or **Play** for a game without profiles, is now Racing Red with bold dark text. **Play as** someone keeps that profile's colour.
- **A game's page puts what matters first.**
  - **Manage**, at the top right of the game's art, opens a panel at the side with everything else: load someone's files, edit the game, its folders and backups, check its files, move it, the desktop shortcut and turning profiles off. These used to sit at the bottom of the page.
  - The game's achievements come right under Play, with the latest you unlocked side by side. **Show all** opens the whole list in a panel, where you can switch between profiles.
  - The art at the top takes less room, so more of the page is in view.
- **Achievements page.** A summary with how much of everything you've unlocked, then each game as a card with its cover and progress, beside the latest you unlocked.
- **Cloud saves.** Backups of everything sit under your games, and the cloud folder's settings beside them, so the page no longer leaves half of it empty.
- **Profiles.** The cards use the whole width and show more of each profile's games. **New profile** and **Names and colours** open a panel at the side; changes there are saved together, and anything you haven't saved stays until you save or discard it.
- The selected choice in a row of buttons, such as a game's status, is marked with a red line.

## 1.27.0

- **The right save locations for new games, from PCGamingWiki.** With **Find save locations online** on, Lootnest now uses PCGamingWiki's list (from the Ludusavi project) to fill in where a game keeps its saves and settings when you add it or the Library finds it, for profiles as well as cloud saves. Before, only cloud saves used it, and profiles looked for folders named like the game.
  - The list's locations come first, marked **Listed on PCGamingWiki**, then Steam's folders for the game, then folders named like it.
  - Saves kept inside the game's own folder are found too, such as Ren'Py's `game\saves`. The game's folder itself is never swapped.
  - Several save files in one folder become that folder, so a profile swaps one location instead of each file.
- **Find save locations online is now in Settings, Online**, too, so you can turn it on without setting up cloud saves. When it's turned on, the Library looks for games again with the new list.
- Games you already set up keep the locations they have.

## 1.26.0

- **Swaps that can't be cut in half.** Loading a profile's files takes a few steps. If Lootnest closed or the PC turned off in the middle, the files on the PC could be a mix of two profiles, and the next swap would save that mix over someone's profile. Lootnest now writes down whose files are going in before it changes anything. When it starts again, it finishes that swap first, and says so. Nothing saves the files in place until it has.
- **Every copy is checked.** When Lootnest copies a game's files, into a profile, a backup or back onto the PC, it reads the copy back and compares it with the original. A copy that doesn't match replaces nothing, and the swap stops with an error instead.

## 1.25.0

- **Status, for each profile.** On a game's page, set where it is for you: **Playing**, **Up next**, **Finished**, **100%** or **Dropped**. Each profile and each run has its own, and the buttons show everyone's at a glance. Press it again to clear it.
- **Collections.** Put a game in your own collections, such as "Co-op", "For the kids" or "Weekend": type a name on its page, or pick one you already have. They're shared by everyone on the PC.
- **The Library filters by both:**
  - **All collections** shows the games in one collection;
  - **Any status** shows the games at that status, for the profile picked above the grid, or for the one at the PC.
- Statuses set on the old Shelf (before 1.16.4) come back, since they were never deleted.

## 1.24.0

- **Profiles grouped by person, as on a console.** A profile can be **a run for** someone: "Cristian hardcore" and "Cristian 100%" are Cristian's runs. Each has its own saves, settings and achievements, as any profile does.
  - A run has its person's colour and picture, with a thin ring around it so you can tell them apart.
  - Runs are listed under their person in the sidebar, in **Who's playing?** (as "Cristian's run, hardcore") and on the **Profiles** page, where a person's card has their runs under it.
  - Make one from **New profile** in **Who's playing?** by choosing **A run for Cristian**, or on **Profiles** with the new **Whose profile** menu on each row.
- Lootnest now keeps up to 16 profiles, so there's room for everyone's runs.

## 1.23.3

- **Each profile's achievements go to the cloud with its saves.** With **Sync saves between PCs** on, a profile's own achievements (1.20) now go up after a session with its saves, and come down on your other PC before the game starts. That works even on a PC that doesn't have them yet.
- The first sync after this update uploads each profile's saves again, this time with their achievements.

## 1.23.2

- **Achievements show for more games in a folder.** Lootnest finds a game's Steam id in more places:
  - deeper in its folder, where Unreal games keep it (`<Game>\\Binaries\\Win64\\steam_appid.txt`);
  - from the id Lootnest already looked up by name for the game's cover or news. Nothing extra goes online for this.
- **Show recent activity** now names the game and says why when a game shows no achievements, such as "no Steam app id next to the game".

## 1.23.1

- **Names and pictures for every achievement, with your own Steam Web API key** (optional, in Settings, Online). Without it, Lootnest can show only an id for a game in a folder without a `steam_settings` file, such as "Kill ten". With it, Steam gives the real names, descriptions and pictures.
  - Steam games show their whole list too, not only the part Steam's library keeps on the PC.
  - Each game's list is asked for once when you open its page, then kept on this PC for a week.
  - Only the game's Steam id and your key are sent, to Steam. The key is encrypted on this PC like the SteamGridDB one, and kept out of the log.

## 1.23.0

- **Lootnest milestones**, for any game, even one with no achievements of its own. They're on the Achievements page, for each profile:
  - first session, 10 and 100 hours in all;
  - 7 and 30 days in a row;
  - 5 and 25 different games;
  - a 4-hour session;
  - 50 hours in your most played game;
  - 100 achievements unlocked, and a game at 100%.
- Each one not reached yet shows how far along it is, such as "44 of 50 h".
- They're worked out from what Lootnest already counts for each profile, so they include play before this version.

## 1.22.0

- **Achievements in the sidebar**: one page with every game's achievements on this PC, for one profile at a time.
  - **Latest unlocked**, across every game, with the game and when.
  - **Games**, ordered by how close each is to 100%, with a bar and "21 of 54". Click one to open its page.
  - The totals: achievements unlocked, games at 100% and games with achievements.
  - Choose whose with the profile buttons. Games whose achievements go with the profile show that profile's own. Steam's belong to the Steam account, and one set kept on the PC is marked **Shared**.
- Everything is read from this PC, as before. Nothing is sent anywhere.

## 1.21.0

- **New profile, right from "Who's playing?"** The last slot makes a profile on the spot. It can be someone new, or another run for yourself, such as "Cristian hardcore". Name it, press Enter, and it's chosen.
- **Start a profile from anyone's saves.** A profile that hasn't played a game yet can start:
  - fresh;
  - from a copy of the files loaded now;
  - from a copy of any other profile's files for that game, such as Cristian's saves while Mihai's are loaded.
  The profile it was copied from is never changed.
- Pressing 1 to 9 in "Who's playing?" now plays the swap animation too.

## 1.20.1

- **Players are now Profiles**, as on a console. Each profile has its own settings, saves and achievements in every game:
  - make one for everyone who plays;
  - or make a few for yourself, one per run, such as "Cristian 100%" and "Cristian hardcore".
- The sidebar, the **Profiles** page, the game page's **Profiles** panel, first run, the backup and import screens, and Lootnest's messages all say "profile".
- "Has a profile" on a game is now **Has saves here**. Nothing else changes: your profiles, games and saves are the same.

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
