# Roadmap

Plugins, mods, controllers, game details, timeline, UI, GOG and itch.io, updates you already downloaded, and Linux. Written November 2026 from research on each; every claim that depends on a source links it.

Bound by CLAUDE.md: copy first, nothing online unless switched on, no downloads, no DRM bypass, no injection. Sizes S to XL; each phase is one PR and one release.

## 1. Plugins

**What users get.** Plugins add library sources (RetroArch playlists, other launchers), save presets, game details and themes; scripts and panels later. Plugins never touch saves: they ask, and Lootnest copies first and asks the user.

**Approach.** Fixed extension points, not "any kind of plugin": a folder with `plugin.json`, packed as a zip with no native code. Data plugins first; scripts later in a pure-Go JS engine with memory and time limits ([modernc.org/quickjs](https://pkg.go.dev/modernc.org/quickjs)). Key constraint: Go's `plugin` package doesn't work on Windows, hosting .NET needs cgo, and full-trust extensions have real costs, as the GlassWorm worm showed in 2025 and 2026 ([hackmag](https://hackmag.com/news/glassworm)).

**Phases.**
- 0, S: harden the WebView2 bridge (top frame only, check message Source in Go, drop the upstream echo), as [Microsoft advises](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/security); safe mode.
- 1, L: install from file; library sources (JSON, INI, VDF, registry, folders); Ludusavi-format save presets as suggestions only; local metadata; token-only themes checked for AA contrast, per profile. Shipped in 1.89: Settings, Plugins installs a .zip into `plugins\<id>` after checking every file (`core/plugins.go`), keeps the replaced version for Roll back, and reads sources (`pluginsources.go`, registry through `win.ReadRegKey`), saves and metadata (`pluginmeta.go`) and themes (`plugintheme.go`, the contrast pairs in Go); the format is in [PLUGINS.md](PLUGINS.md).
- 2, M: online metadata, declared hosts only, a per-plugin switch in Settings, Online, keys in DPAPI, requests logged.
- 3, M: Ed25519 signatures with the key pinned per plugin; rollback. Shipped in 1.90: `signature.json` over a sorted list of every file's SHA-256 (`core/pluginsign.go`, `PluginDigest`, `VerifyPlugin`), the key pinned per id in plugins.json on first use, a key change only through a confirm for the key shown (`InstallPluginNewKey`), no unsigned version over a signed one, Roll back offered only for a kept version with the pinned key, and `cmd/plugsign` for authors; rollback shipped in 1.89.
- 4, XL: script plugins with a read-only API; save points only as requests the user confirms. The owner's change of 17 Nov 2026: a plugin may hold code. A script plugin (`core/pluginscript.go`) has one .js run by a JavaScript engine in Go ([goja](https://github.com/dop251/goja)), with permissions the user allows before it runs (net, library, updates, notify), keys kept by Lootnest, and Settings pages it describes as data; see [PLUGINS.md](PLUGINS.md). Lootnest ships none; a script can offer updates and hand their downloads to Lootnest, which replaces a game's folder with a copy first (`core/fullupdate.go`).
- 5, L: panels in sandboxed iframes under a strict CSP.

**Will not do.** DLL, .NET, PowerShell or Python plugins; Playnite extensions; any API that lets a plugin itself start programs or write game folders or saves (a script asks, Lootnest's own code does it with a copy first, after the user says yes); raw CSS themes, which could hide safety notices; plugin dependencies.

## 2. Mod manager

**What users get.** Bring a mod archive; Lootnest installs it for one profile or run. "Player 1 modded" has its own mods and saves, and when Player 2 plays, the game goes back to vanilla or to Player 2's set. "Put the game back as it was" is one click.

**Approach.** Real files, staged beside the game folder, deployed by copy and later by hardlink. Each vanilla file goes to `originals` before it is replaced, and a journal finishes a cut-short deploy. Key constraint: Mod Organizer 2's virtual folder hooks the game process ([STEP](https://stepmodifications.org/wiki/Guide:Mod_Organizer_Advanced)), and hardlinks need NTFS on one volume ([CreateHardLinkW](https://learn.microsoft.com/windows/win32/api/winbase/nf-winbase-createhardlinkw)), so copy is always the fallback.

**Phases.**
- 1, L: one set per game, copy deploy, originals, journal, Undo. Archives replacing store or DRM files (steam_api, EOSSDK, emulator configs) are refused. Mods locked for anti-cheat games, with Vanguard, FACEIT and EA AntiCheat added to detection. Program Files and WindowsApps refused. Shipped in 1.86: the Mods tab on a game's page; each mod is kept in Lootnest's data folder (`mods\<key hash>`), laid out as it goes into the game's folder, and `journal.json` and `deployed.json` there carry the run and what's in place (`core/mods.go`).
- 2, M: per-profile and per-run sets on Play, saves first and mods second; Start a modded run; updates undeploy first. Shipped in 1.87: `set.json` holds a set per owner (the game's, or a profile's own; a run without one uses its person's, a person without one the game's, `ModSetFor`), `deployed.json` says whose set is in, Play switches the set after the swap with the same journaled run as Apply, an update takes the mods out first and leaves them off, and New run has Start with mods.
- 3, M: hardlinks with copy fallback, config files always copied, hash checks rather than dates (Vortex's false [external-change alarms](https://forums.nexusmods.com/topic/13539247-serious-problem-with-incorrect-checking-for-external-changes-logic/)), conflicts, reorder. Shipped in 1.88: a file goes in as a hard link to Lootnest's read-only copy when the drive allows it, else by copy (`placeMod`); settings and save files are always copied (`modConfigFile`); `deployed.json` records how each went in; a linked file the game changed is kept in `changed\` and its mod marked to add again (`Mod.Changed`); `ModConflicts` on each row, drag to reorder, and the setting in the Mods tab's More menu (`Settings.ModsLinkOff`).
- 4, L: BepInEx, Thunderstore zips (dependencies named, never fetched), Unreal `~mods`, Bethesda `plugins.txt` per profile ([LOOT](https://loot.github.io/docs/help/Introduction-To-Load-Orders)), XML FOMOD.
- 5, M, owner's decision: an opt-in Nexus `nxm://` hand-off. It needs a registered app under the [API policy](https://help.nexusmods.com/article/114-api-acceptable-use-policy) and bends "no downloads". The owner chose Nexus (20 Nov 2026), in two steps:
  - 5a, shipped (Unreleased): browsing in the Mods tab through Nexus's public GraphQL API (v2, no key: search by words with `WILDCARD`, a mod, its files, `modRequirements`), adult mods left out (`core/nexus.go`). Download opens the file's page in Lootnest's own WebView2 window (`--nexus`, `nexuswin_windows.go`: Nexus, its CDN `*.nexus-cdn.com` and the sign-in providers stay in it, other sites go to the browser, `nxm://` is refused with a note); the user presses Manual download, WebView2 saves to Downloads, and `NexusArrivals` matches the file by "-<mod id>-" and the upload time at the end of its name, then the modplan drawer opens (`apiNexusAdd`, `core.NameMod` keeps `Mod.Nexus`). Lootnest itself downloads nothing.
  - 5a, second part (the owner, 20 Nov 2026: "they won't accept, so we need a workaround"): the window takes the download itself (WebView2's DownloadStarting, `edge/download_lootnest.go` on `ICoreWebView2_4`, layouts from the SDK's WebView2.h): it goes into `nexus-downloads` with a `.lootnest.json` note of the mod, file and size (`core.NexusIncomingPath`, `WriteNexusNote`), the browser's list stays hidden, and Lootnest shows its progress (`NexusIncoming`, event `nexusProgress`) and matches it by the note. A Mod Manager Download link (`nxm://`, `core.NexusNXM`) takes the window to that file's page for Manual download. Free accounts lose nothing to the lack of registration: Nexus asks them to click on its site in every manager. Not done, on purpose: pressing the site's buttons for the user (Nexus forbids automated downloads) and a personal API key pasted by each user (Nexus's policy is against personal keys in public apps).
  - 5b, after the owner registers Lootnest with Nexus (support, with a test build of 5a, its name, a description and a logo; a personal API key is only for testing): Sign in with Nexus, `nxm://` taken in the window (download_link.json with the link's key and expiry for free accounts, direct for Premium), and updates for mods kept with `Mod.Nexus`.

**Will not do.** A virtual file system or symlinks; mods for anti-cheat games, with no override ([BattlEye](https://battleye.com/support/faq) blocks proxy DLLs); shipping BepInEx, SKSE or ReShade; CurseForge, whose [terms](https://support.curseforge.com/support/solutions/articles/9000207405) forbid caching and sharing a key.

## 3. Controllers

**What users get.** DualSense, Xbox and Switch pads work properly in Lootnest and Big Picture, with the right glyphs. Settings lists controllers and translators, game pages say what each game expects, and folder games can use Steam Input through Lootnest's swap.

**Approach.** Lootnest did not translate controllers at first (phase 6 is the owner's change). A virtual pad games see needs a Microsoft-signed kernel driver ([signing rules](https://learn.microsoft.com/en-us/windows-hardware/drivers/dashboard/code-signing-reqs)), which one developer can't ship or maintain, and the usual route is ending: ViGEmBus [retired in 2023](https://docs.nefarius.at/projects/ViGEm/End-of-Life/), the main DS4Windows fork [archived in March 2026](https://github.com/schmaldeo/DS4Windows). Steam Input is free, signed and maintained, and covers any game Steam starts, shortcuts included. Detection is pure Go (SetupAPI, hid.dll, XInput).

**Phases.**
- 1, S: read every pad and ignore a translator's virtual twin; glyph families from the pad's id ([Chromium's mappings](https://chromium.googlesource.com/chromium/src/+/main/device/gamepad/gamepad_standard_mappings_win.cc)); PS and touchpad buttons; rumble only where offered.
- 2, M: controllers and translators on this PC, with no HID report reads or writes, which can switch a Bluetooth DualSense's mode. Shipped in 1.84: Settings, Controllers, from SetupAPI's device list, the device tree and XInput's state; no device is opened.
- 3, M: what each game needs, from Steam's controller categories, opt-in under Settings, Online. Shipped in 1.85: from the same appdetails request Game details makes (`controller_support` and the categories), on the game's page and in Big Picture, editable.
- 4, L: Play with Steam Input: append-only `shortcuts.vdf`, Steam closed with consent, a copy first, name clashes refused. Exe-launched games only; launcher hand-offs lose Steam Input.
- 5, M, optional: the seated profile's colour on the lightbar, USB only.

- 6, M, the owner's change of 16 Nov 2026: a PlayStation controller shown to games as an Xbox one, as DS4Windows does, inside Lootnest. Lootnest still ships no driver: it talks to ViGEmBus when that driver is already on the PC (DS4Windows and the TV streaming set-up install it), reads the pad's input reports without sending it anything, and feeds a virtual Xbox 360 pad only while a game Lootnest follows runs. Off unless turned on; a game's Launch options can leave the pad as it is. ViGEmBus is retired but still signed and working; if it stops loading, this stops and says so.

**Will not do.** Ship drivers, hide the real pad, remap buttons, inject into games.

## 4. Game details

**What users get.** Summary, genres, developer, publisher, release date, screenshots, Steam reviews, Metacritic and time to beat, cached on the PC and editable.

**Approach.** Steam appdetails first (already used, no key), GOG's product API, PCGamingWiki facts as fallback, IGDB only with the user's own keys. A "Game details" switch in Settings, Online, off by default; fetched when a page opens; 30-day cache; edits always win. Store HTML becomes plain text in Go. Key constraint: appdetails is undocumented and rate limited ([measurements](https://github.com/GoFurry/steam-go/wiki/Rate-Limiting-Strategy)), and [IGDB](https://api-docs.igdb.com/) needs a secret that would be public inside an exe.

**Phases.**
- 1, M: Steam details and [review summary](https://partner.steamgames.com/doc/store/getreviews), edits, a screenshot lightbox, trailer thumbnails opening the store page.
- 2, M: GOG, PCGamingWiki (CC BY-NC-SA, credited, never paid), Xbox manifests offline, Big Picture screenshots, a HowLongToBeat search link.
- 3, M: IGDB time to beat.
- 4, L: offline facts from `appinfo.vdf` ([format](https://github.com/SteamDatabase/SteamAppInfo)), a genre filter.

**Will not do.** Scrape HowLongToBeat or Metacritic; ship IGDB credentials; inline Steam trailers for now, since Steam serves only DASH and HLS ([LaunchBox](https://feedback.launchbox-app.com/p/steam-trailers-unavailable-due-to-new-steam-api-changes)); crawl the Library in the background.

## 5. Timeline, screenshots and video

**What users get.** A timeline per game and profile: sessions, save points, unlocks, updates, runs, and the captures Game Bar, Steam, NVIDIA, AMD and OBS already made. Plus a screenshot hotkey and a picture per save point.

**Approach.** Import, don't record. Sessions become records in an append-only log (today only daily totals exist). Captures are indexed, never moved, renamed or deleted, and matched by Steam's app id folder, name, then session time. Screenshots are a GDI copy from outside the game. Key constraint: a reliable recorder (Media Foundation H.264, audio, sync, replay buffer, HDR) is XL and testable only on the owner's PC, and Game Bar already does it everywhere.

**Phases.**
- 1, M: session log and a timeline without captures; no Windows-only code.
- 2, L: capture import through virtual hosts, shell thumbnails, a lightbox, an Unsorted list. File names are undocumented, so time matching carries it. Shipped in 1.73, with thumbnails as data: URLs instead of virtual hosts.
- 3, M: Ctrl+Shift+F12 to `Pictures\Lootnest\Screenshots\<Game>`, a borderless hint on black frames, a picture on each save point. Shipped in 1.70.
- 4, S: a tip naming Game Bar's clip keys, and a notice when capture is off. Shipped in 1.73.
- 5, L, optional: [Windows.Graphics.Capture](https://learn.microsoft.com/en-us/windows/uwp/audio-video-camera/screen-capture) by hand-written WinRT, only if GDI falls short; its yellow border stays.

**Will not do.** Record video or ship ffmpeg ([GPL builds](https://www.gyan.dev/ffmpeg/builds/), around 100 MB); capture exclusive fullscreen, which needs hooking; work around games that hide from capture.

## 6. A better UI

**What users get.** The game page becomes the centre, with tabs. Home's hero follows the seated profile, art-less games get type covers, Back works, Ctrl+K runs commands, and density, scale, contrast and motion are set per profile.

**Approach.** One hero and shelf language, tokens only, WCAG 2.2 AA and Steam's 2025 bar ([GamingOnLinux](https://gamingonlinux.com/2025/09/steam-update-gives-wider-store-pages-library-sorting-desktop-accessibility-controls-and-lots-more)). Key constraint: the CSP nonce and no build step rule out injected CSS, and the window's script is one closure in ordered parts in `ui/js`, joined in name order by `app.go` and the harness.

**Phases.**
- 0, M: split `app.js` into ordered files joined into one closure; role type tokens; routing with Back; an accessibility harness; the tab slot API. The split and Back shipped in 1.67.
- 1, M: art fallback chain, type covers, logo art, palette tint, custom art copied into `covers\custom`.
- 2, L: game page tabs, a facts column, a compact sticky head. The tabs and the head shipped in 1.72; the side column stays on Overview.
- 3, M: five nav items with a Saves hub; the command palette. Shipped in 1.91 with the top bar's six tabs kept: Saves takes Cloud saves' place, and Cloud saves is a section of it. Ctrl+K runs commands above its search.
- 4, M: density, WebView2 zoom, contrast, forced colours, motion, per profile.
- 5, M: post-session summary, cover-to-hero transition, spotlight shelves.
- 6, S, optional: OLED ground, [Mica](https://learn.microsoft.com/en-us/windows/win32/api/dwmapi/ne-dwmapi-dwm_systembackdrop_type) on Windows 11 22621 and later.

**Will not do.** Arbitrary CSS themes; autoplaying video or carousels (WCAG 2.2.2); a light theme or layout editor for now.

## 7. Your games from GOG and itch.io, downloaded and updated in Lootnest

Done: all three phases shipped by 1.83.

**What users get.** The games you own on GOG (DRM-free) and itch.io install, update and roll back inside Lootnest, with no other launcher. The update case the stores don't cover: a game in a folder that no launcher keeps current.

**Approach.** Only games the user owns, through each store's own API, signed in by the user: GOG's account and installer endpoints, and itch.io's API with the user's own key. Downloads go through the update center that already exists: saves copied first, the version checked, every replaced file kept, Undo. Off until the user signs in.

**Phases.**
- 1, M: itch.io: library, download and install a build, check for a newer upload. Done in 1.81.
- 2, L: GOG: sign in, library, offline installers for owned games, updates through the installer flow (1.40). Done in 1.82, with the full installer for updates rather than GOG's patches.
- 3, S: both show on the Updates page with the other folder games. Done in 1.83.

**Will not do.** Any source other than the store the user bought from; sharing or caching downloads between users.

## 8. Updates you already downloaded, found for you

**What users get.** When an update for a Library game lands in Downloads (a folder, a .zip, .7z or .rar, multi-part included, or an installer), Lootnest notices it, says which game and which version it moves you to, and applies it in one click with the saves copied first and Undo. It offers to delete the archive afterwards.

**Approach.** Watch the user's Downloads folder only, match by name and version (UpdateVersions, FolderVersion), and hand over to the existing PlanUpdate, ApplyUpdate and installer flow. It never looks anywhere else and never fetches anything.

**Phases.**
- 1, M: the Downloads watcher, the notice, one-click apply, delete the archive.

**Will not do.** Find, suggest or download updates from anywhere.

## 9. Lootnest on Linux

The owner's decision, 19 Nov 2026: the Linux desktop in general (not the Steam Deck first), with a WebKitGTK window through cgo.

**What users get.** The same Lootnest on a Linux PC: one Library of native games and Windows games run through Proton or Wine, profiles and runs with their own saves, settings and achievements, save points, the safety net and cloud saves in the user's own folder.

**Approach.** `internal/core` already builds for Linux; the work is the desktop layer and where games keep their files. The app talks to the system only through `internal/plat` (Windows forwards to `internal/win`; `tools/platgen` writes the forwards and the stubs other systems start with), so Linux replaces stubs one hand-written file at a time and Windows doesn't change. Key constraint: a Windows game on Linux keeps its saves inside a Wine prefix (Steam's `steamapps/compatdata/<appid>/pfx/drive_c/users/steamuser/...`; Heroic, Lutris and Bottles each keep their own), so every `%APPDATA%`-style path, the Ludusavi manifest's included, has to be read inside the right game's prefix, and names there are case-insensitive to the game but not to the file system. Wayland gives no window over another app's and no global keys without a portal ([GlobalShortcuts](https://flatpak.github.io/xdg-desktop-portal/docs/doc-org.freedesktop.portal.GlobalShortcuts.html)), so the overlay, console mode and hotkeys can't be the same as on Windows.

**Phases.**
- L0, M: `internal/plat` between the app and the system; `internal/app` compiles for Linux, its window code in `window_windows.go`, `webapp_windows.go` and `proc_windows.go`; CI builds it there and checks the generated files are current. Nothing changes on Windows.
- L1, L: the Linux path model in `core`: XDG folders for Lootnest's data, a prefix per game and tokens expanded inside it, scans of Steam (native and Flatpak, with `compatdata`), Heroic, Lutris, Bottles and the itch app, the Ludusavi manifest's Windows paths in prefixes, the tools' achievement files in prefixes. Tested with fixtures, no window needed. Done in two parts: L1a, the prefix model and Steam (`core/prefix.go`, `core/linux.go`); L1b, Heroic, Lutris, Bottles and the itch app (`core/linuxlaunchers.go`, with a read-only SQLite reader for Lutris's library).
- L2, L: the window: a small cgo binding over WebKitGTK 4.1 with the guarantees `go-webview2` has (only Lootnest's own page, navigation refused through [decide-policy](https://webkitgtk.org/reference/webkit2gtk/stable/signal.WebView.decide-policy.html), each message's frame checked, the CSP nonce), single instance over a socket in `$XDG_RUNTIME_DIR`, a tray ([StatusNotifierItem](https://www.freedesktop.org/wiki/Specifications/StatusNotifierItem/)), secrets in the [Secret Service](https://specifications.freedesktop.org/secret-service-spec/latest/) instead of DPAPI, the freedesktop Trash, `xdg-open`, file pickers through the portal, `cmd/lootnest` built for Linux on CI. In two parts: Done in two parts: L2a, the window, starting it, one Lootnest at a time, the user's folders, opening links and folders (`internal/webkit`); L2b, the tray, keys, the Trash, file pickers, dialogs and notifications (`internal/desk`).
- L3, L: games: running games from `/proc` (a Proton game by `SteamAppId` and its exe), starting native games and folder games through a runner the user picks (Wine, or Proton via [umu-launcher](https://github.com/Open-Wine-Components/umu-launcher)), the swap, save points, safety net and cloud on Linux, notifications over D-Bus, Linux tests in CI; the first Linux build to download (a .tar.gz, then an AppImage). In two parts: L3a, starting and following games, every place a game's files are read in its prefix, and the runner in Settings, done; L3b, the first Linux download.
- L4, M: Big Picture and controllers (the Gamepad API in WebKitGTK), full screen on X11 and Wayland, hotkeys and screenshots through the desktop portal where the desktop has it.
- L5, later and only if asked: TV streaming with Sunshine instead of Apollo, the pad bridge through uinput, Keep a game offline without its network.

**Will not do.** Lootnest's own Wine or Proton builds (the user's runner starts the game); a Flatpak at first, since its sandbox can't reach other launchers' prefixes without asking for the whole home folder; the native toasts over a game on Wayland (a notification instead); carrying DPAPI-encrypted keys between systems (they're entered again).

## Order of work

Groundwork and safety first, then offline work with no Windows-only code, then what only the owner's PC can confirm.

1. Updates found in your Downloads (M), then the top 8 from what players asked for on Reddit and forums (each S).
2. Plugins 0, bridge hardening (S): a security gain on its own.
3. Timeline 1, session log (M): history can't be rebuilt, so start recording.
4. UI 0, foundations and slot API (M), after the polish backlog merges.
5. Controllers 1 (S).
6. Game details 1 (M), then GOG and itch.io 1 to 3 (M, L, S).
7. Timeline 3, screenshots and save point pictures (M).
8. UI 1 and 2 (M, L).
9. Timeline 2 and 4 (L, S).
10. Controllers 2 and 3 (M, M).
11. Mods 1 to 3 (L, M, M).
12. Plugins 1 and 3 (L, M). Done in 1.89 and 1.90.
13. UI 3 to 5 (M each). UI 3 done in 1.91.
14. Game details 2 to 4; Plugins 2.
15. Controllers 4 and Mods 4 (L, L): undocumented Steam and game formats.
16. Later, if wanted: Plugins 4 and 5, Timeline 5, Controllers 5, UI 6. Mods 5 only on the owner's explicit decision.
17. Linux, L0 to L5 (M, L, L, L, M, later), one at a time, the owner choosing when each starts. L0, L1 and L2 done (unreleased).
