# Roadmap

Plugins, mods, controllers, game details, timeline, UI, GOG and itch.io, and updates you already downloaded. Written November 2026 from research on each; every claim that depends on a source links it.

Bound by CLAUDE.md: copy first, nothing online unless switched on, no downloads, no DRM bypass, no injection. Sizes S to XL; each phase is one PR and one release.

## 1. Plugins

**What users get.** Plugins add library sources (RetroArch playlists, other launchers), save presets, game details and themes; scripts and panels later. Plugins never touch saves: they ask, and Lootnest copies first and asks the user.

**Approach.** Fixed extension points, not "any kind of plugin": a folder with `plugin.json`, packed as a zip with no native code. Data plugins first; scripts later in a pure-Go JS engine with memory and time limits ([modernc.org/quickjs](https://pkg.go.dev/modernc.org/quickjs)). Key constraint: Go's `plugin` package doesn't work on Windows, hosting .NET needs cgo, and full-trust extensions have real costs, as the GlassWorm worm showed in 2025 and 2026 ([hackmag](https://hackmag.com/news/glassworm)).

**Phases.**
- 0, S: harden the WebView2 bridge (top frame only, check message Source in Go, drop the upstream echo), as [Microsoft advises](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/security); safe mode.
- 1, L: install from file; library sources (JSON, INI, VDF, registry, folders); Ludusavi-format save presets as suggestions only; local metadata; token-only themes checked for AA contrast, per profile.
- 2, M: online metadata, declared hosts only, a per-plugin switch in Settings, Online, keys in DPAPI, requests logged.
- 3, M: Ed25519 signatures with the key pinned per plugin; rollback.
- 4, XL: script plugins with a read-only API; save points only as requests the user confirms.
- 5, L: panels in sandboxed iframes under a strict CSP.

**Will not do.** DLL, .NET, PowerShell or Python plugins; Playnite extensions; any API that downloads, starts arbitrary programs, writes game folders or writes saves; raw CSS themes, which could hide safety notices; plugin dependencies.

## 2. Mod manager

**What users get.** Bring a mod archive; Lootnest installs it for one profile or run. "Player 1 modded" has its own mods and saves, and when Player 2 plays, the game goes back to vanilla or to Player 2's set. "Put the game back as it was" is one click.

**Approach.** Real files, staged beside the game folder, deployed by copy and later by hardlink. Each vanilla file goes to `originals` before it is replaced, and a journal finishes a cut-short deploy. Key constraint: Mod Organizer 2's virtual folder hooks the game process ([STEP](https://stepmodifications.org/wiki/Guide:Mod_Organizer_Advanced)), and hardlinks need NTFS on one volume ([CreateHardLinkW](https://learn.microsoft.com/windows/win32/api/winbase/nf-winbase-createhardlinkw)), so copy is always the fallback.

**Phases.**
- 1, L: one set per game, copy deploy, originals, journal, Undo. Archives replacing store or DRM files (steam_api, EOSSDK, emulator configs) are refused. Mods locked for anti-cheat games, with Vanguard, FACEIT and EA AntiCheat added to detection. Program Files and WindowsApps refused.
- 2, M: per-profile and per-run sets on Play, saves first and mods second; Start a modded run; updates undeploy first.
- 3, M: hardlinks with copy fallback, config files always copied, hash checks rather than dates (Vortex's false [external-change alarms](https://forums.nexusmods.com/topic/13539247-serious-problem-with-incorrect-checking-for-external-changes-logic/)), conflicts, reorder.
- 4, L: BepInEx, Thunderstore zips (dependencies named, never fetched), Unreal `~mods`, Bethesda `plugins.txt` per profile ([LOOT](https://loot.github.io/docs/help/Introduction-To-Load-Orders)), XML FOMOD.
- 5, M, owner's decision: an opt-in Nexus `nxm://` hand-off. It needs a registered app under the [API policy](https://help.nexusmods.com/article/114-api-acceptable-use-policy) and bends "no downloads".

**Will not do.** A virtual file system or symlinks; mods for anti-cheat games, with no override ([BattlEye](https://battleye.com/support/faq) blocks proxy DLLs); shipping BepInEx, SKSE or ReShade; CurseForge, whose [terms](https://support.curseforge.com/support/solutions/articles/9000207405) forbid caching and sharing a key.

## 3. Controllers

**What users get.** DualSense, Xbox and Switch pads work properly in Lootnest and Big Picture, with the right glyphs. Settings lists controllers and translators, game pages say what each game expects, and folder games can use Steam Input through Lootnest's swap.

**Approach.** Lootnest does not translate controllers. A virtual pad games see needs a Microsoft-signed kernel driver ([signing rules](https://learn.microsoft.com/en-us/windows-hardware/drivers/dashboard/code-signing-reqs)), which one developer can't ship or maintain, and the usual route is ending: ViGEmBus [retired in 2023](https://docs.nefarius.at/projects/ViGEm/End-of-Life/), the main DS4Windows fork [archived in March 2026](https://github.com/schmaldeo/DS4Windows). Steam Input is free, signed and maintained, and covers any game Steam starts, shortcuts included. Detection is pure Go (SetupAPI, hid.dll, XInput).

**Phases.**
- 1, S: read every pad and ignore a translator's virtual twin; glyph families from the pad's id ([Chromium's mappings](https://chromium.googlesource.com/chromium/src/+/main/device/gamepad/gamepad_standard_mappings_win.cc)); PS and touchpad buttons; rumble only where offered.
- 2, M: controllers and translators on this PC, with no HID report reads or writes, which can switch a Bluetooth DualSense's mode.
- 3, M: what each game needs, from Steam's controller categories, opt-in under Settings, Online.
- 4, L: Play with Steam Input: append-only `shortcuts.vdf`, Steam closed with consent, a copy first, name clashes refused. Exe-launched games only; launcher hand-offs lose Steam Input.
- 5, M, optional: the seated profile's colour on the lightbar, USB only.

**Will not do.** Ship drivers, hide the real pad, remap input, inject into games.

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
- 3, M: five nav items with a Saves hub; the command palette.
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
12. Plugins 1 and 3 (L, M).
13. UI 3 to 5 (M each).
14. Game details 2 to 4; Plugins 2.
15. Controllers 4 and Mods 4 (L, L): undocumented Steam and game formats.
16. Later, if wanted: Plugins 4 and 5, Timeline 5, Controllers 5, UI 6. Mods 5 only on the owner's explicit decision.