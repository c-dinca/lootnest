  // -------------------------------------------------------------- settings --
  // invert: the setting turns the thing off (on by default), so the switch
  // shows the opposite.
  function sw(field, title, desc, checked, invert) {
    return `<label class="toggle"><span><b style="font-weight:500">${title}</b>${desc.startsWith("<") ? desc : `<p>${desc}</p>`}</span>
      <span class="switch"><input type="checkbox" data-setting="${field}"${invert ? ' data-invert="1"' : ""} ${checked ? "checked" : ""}><span></span></span></label>`;
  }

  const SHORTCUTS = (s) => [
    [["Ctrl+K"], "Run a command, or search your games, and the store if it's on"],
    [["Ctrl+F", "/"], "Find a game in the Library"],
    [["Arrow keys"], "Move between covers in the Library"],
    [["Home", "End"], "First or last cover"],
    [["Esc"], "Close a dialog, or go back from a game's page"],
    [["Ctrl+Tab", "Ctrl+Shift+Tab"], "Next or previous tab on a game's page"],
    [["Alt+Left", "Backspace", "Mouse back button"], "Go back to the page you were on"],
    [["Alt+Right", "Mouse forward button"], "Go forward again"],
    [["Ctrl+B"], "Fold the sidebar, or open it again"],
    [["F5"], "Look for games again"],
    [["1 to 9"], "Choose a profile in Who's playing, or in the menu under your picture in the top bar"],
    [["F11"], "Open or leave Big Picture"],
    [["Q", "E"], "Switch tabs in Big Picture"],
    ...(s.pointHotkeyOff ? [] : [[["Ctrl+Shift+F5"], "Make a save point while a game runs"]]),
    ...(s.screenshotsOff ? [] : [[["Ctrl+Shift+F12"], "Save a picture of the game that's running"]]),
  ];
  // Captures from other apps: the switch, the folders read, and the way to
  // the ones matched to no game.
  let unsortedAsked = false;
  function capturesRows(s) {
    const U = S.unsorted;
    if (!s.capturesOff && !unsortedAsked) { unsortedAsked = true; api.unsortedCaptures().catch(() => {}); }
    const rows = !s.capturesOff && U ? `
      <div class="bk-row"><div><b>Folders Lootnest reads</b>${U.folders.length ? `<ul class="set-folders selectable">${U.folders.map((f) => `<li><b>${esc(f.source)}</b> ${esc(f.dir)}</li>`).join("")}</ul>` : "<p>None of these apps has a captures folder on this PC yet.</p>"}</div></div>
      <div class="bk-row"><div><b>Unsorted captures</b><p>${U.total ? `${U.total} ${U.total === 1 ? "capture" : "captures"} Lootnest couldn't match to a game. Choose the game of each, and it shows on that game's Timeline.` : "Every capture is matched to a game."}</p></div>${U.total ? `<button class="ln-button ln-button--secondary ln-button--sm" data-action="captures-unsorted" aria-haspopup="dialog">Choose their games</button>` : ""}</div>` : "";
    return sw("capturesOff", "Captures from other apps", "Screenshots and clips you made with Game Bar, Steam, NVIDIA, AMD or OBS show on each game's Timeline, matched by the game's Steam folder, its name, or when you played it. Lootnest only reads these folders on this PC: it never moves, renames or deletes a capture, and nothing goes online.", !s.capturesOff, true) + rows;
  }
  const setSub = (title, note, body) => `<section class="set-sub"><div class="ln-section-head ln-section-head--minor"><h2>${title}</h2></div>${note ? `<p class="set-note">${note}</p>` : ""}${body}</section>`;
  // Settings: plain on purpose. Groups of rows, each saying what it does,
  // with its control on the right. Everything online is off until turned on.
  // The categories, by key, as the list on the left names them.
  const SET_CATS = { general: "General", games: "Games and swaps", controllers: "Controllers", storage: "Storage", plugins: "Plugins", online: "Online", updates: "Updates", cloud: "Cloud saves", data: "Data and activity", about: "About" };
  function viewSettings() {
    const s = cfg().settings, st = S.d.steam, hidden = Object.entries((S.found && S.found.hidden) || {});
    const row = (forId, title, desc, control) => `<div class="bk-row"><div><label for="${forId}"><b>${title}</b></label>${desc ? `<p>${desc}</p>` : ""}</div>${control}</div>`;
    const b = (kind, attrs, label) => `<button class="ln-button ln-button--${kind} ln-button--sm" ${attrs}>${label}</button>`;
    const update = S.update.available
      ? (S.update.step ? `<button class="ln-button ln-button--primary ln-button--sm" aria-busy="true"><span class="ln-spinner" aria-hidden="true"></span>Updating</button>` : b("primary", 'data-action="update-now"', `Update to ${esc(S.update.version)}`))
      : (S.update.checking ? `<button class="ln-button ln-button--secondary ln-button--sm" aria-busy="true"><span class="ln-spinner" aria-hidden="true"></span>Checking</button>` : b("secondary", 'data-action="update-check"', "Check for updates"));
    const card = (rows) => `<div class="bk-card">${rows}</div>`;
    const sub = (title, note, rows) => setSub(title, note, card(rows));
    const versionRow = `<div class="bk-row set-version"><div><b>Lootnest ${esc(S.d.version)}${S.d.stage ? " " + esc(S.d.stage) : ""}</b><p>${S.update.available ? `Version ${esc(S.update.version)} is ready to install.` : "Updates come from Lootnest's releases on GitHub."}</p></div>${update}</div>`;
    // One category at a time, picked on the left, as in Steam's settings.
    const tabs = [
      ["general", SET_CATS.general, GI.sliders, "How Lootnest starts, closes and tells you things.", () => card(`
          ${sw("autostart", "Start with Windows", "Lootnest opens quietly in the tray when you sign in, so it's always watching your games.", S.d.autostart)}
          ${sw("closeToTray", "Keep running when the window is closed", "Closing the window leaves Lootnest in the tray. Quit from the tray icon.", s.closeToTray)}
          ${sw("hideWhilePlaying", "Hide while a game runs", "The window gets out of the way once the game starts. In Big Picture, it comes back when the game closes.", s.hideWhilePlaying)}
          ${sw("openBigPicture", "Open in Big Picture when Lootnest starts", "For a PC by the TV: Lootnest opens on the whole screen, ready for a controller. Not when it starts in the tray with Windows.", s.openBigPicture)}
          ${sw("notifications", "Notifications", "A short Windows notification when progress is saved.", s.notifications)}`)
        + sub("Keyboard shortcuts", "", `<dl class="set-keys">${SHORTCUTS(s).map(([keys, what]) => `<dt>${keys.map((k) => `<kbd>${k}</kbd>`).join(", ")}</dt><dd>${what}</dd>`).join("")}</dl>`)],
      ["games", SET_CATS.games, GI.play, "How profiles' files are swapped, and how Lootnest works with Steam.", () => sub("Swaps", "", `
          ${row("s-backups", "Copies kept per game before each swap", "A copy is made before every swap. The oldest go once there are more than this.", `<input id="s-backups" class="ln-input set-num" type="number" min="1" max="100" data-setting="backupsToKeep" value="${s.backupsToKeep}">`)}
          ${row("s-timeout", "Seconds to wait for a game to start", "If it hasn't started by then, the profile's files stay loaded, and Lootnest still saves them if the game starts later.", `<input id="s-timeout" class="ln-input set-num" type="number" min="30" max="1800" data-setting="startTimeoutSeconds" value="${s.startTimeoutSeconds}">`)}
          ${sw("watchOutsideLaunches", "Follow games started from Steam or elsewhere", "If a game starts without Lootnest, its progress is saved to whoever's files are loaded.", s.watchOutsideLaunches)}`)
        + sub("While you play", "", `
          ${sw("achToastsOff", "Show achievements as they unlock", "For games whose achievements a Steam-compatible tool keeps on this PC, a small note shows over the game when one unlocks, in the colour of the profile playing. It's a window of Lootnest's own: nothing goes into the game. It shows over windowed and borderless games, and over most fullscreen ones; switch a game to borderless if it doesn't. Never for games with an anti-cheat.", !s.achToastsOff, true)}
          ${sw("soundsOff", "Sounds", "<p>A short sound when an achievement unlocks, a save point is made or a profile's files are loaded, also over the game while Lootnest is hidden. Never for games with an anti-cheat.</p><p>Interface sounds: clicks when you change pages, open menus and press Play.</p>", !s.soundsOff, true)}
          ${sw("pointHotkeyOff", "Ctrl+Shift+F5 makes a save point", "While a game runs, the keys copy its saves as a save point, to go back to from its page. Lootnest holds the keys only while a game runs.", !s.pointHotkeyOff, true)}
          ${sw("screenshotsOff", "Ctrl+Shift+F12 saves a screenshot", "While a game runs, the keys save a picture of it to Pictures\\Lootnest\\Screenshots, taken from outside the game. If it comes out black, switch the game to borderless windowed. Lootnest holds the keys only while a game runs.", !s.screenshotsOff, true)}
          ${s.achToastsOff ? "" : row("s-toast", "See how it looks", "Shows one now, here and over the screen at the bottom right, as it does over a game.", `<button id="s-toast" class="ln-button ln-button--secondary ln-button--sm" data-action="test-toast">Show one</button>`)}`)
        + sub("Updates for your games", "", `
          ${sw("updatePointOff", "A save point before a game updates", "The first time a game starts after an update, Lootnest copies its saves as a save point first, so you can go back if the new version breaks them. It notices an update by Steam's build number, or by the game's .exe changing.", !s.updatePointOff, true)}
          ${sw("downloadsOff", "Look in Downloads for updates to your games", "For games no launcher keeps up to date, Lootnest reads the names of the files and folders in your Downloads folder and tells you when one is a newer version. It looks nowhere else, nothing goes online, and nothing is applied until you review it.", !s.downloadsOff, true)}`)
        + sub("Steam", "", `
          ${sw("autoSteamHooks", "Set up Steam automatically", "Steam games you add get a launch option so Steam's Play button asks who's playing. Removed games get their own options back.", s.autoSteamHooks)}
          ${st.found ? row("s-steam", "Steam account", st.accountError ? `<span class="error-text">${esc(st.accountError)}</span>` : "The account on this PC whose Steam launch options Lootnest sets.", `<select id="s-steam" class="ln-input" data-setting="steamAccountId">
            <option value="">Automatic${st.accountInUse && !s.steamAccountId ? " (" + esc(st.accountInUse) + ")" : ""}</option>
            ${(st.accounts || []).map((a) => `<option value="${esc(a)}" ${a === s.steamAccountId ? "selected" : ""}>${esc(a)}</option>`).join("")}</select>`) : `<div class="bk-row"><div><b>Steam isn't on this PC</b><p>Steam games show up here once it's installed.</p></div></div>`}`)
        + (hidden.length ? sub("Hidden games", "Games you hid from the Library. Scans leave them out until you show them again.", hidden.map(([k, name]) =>
          `<div class="bk-row"><div><b>${esc(name)}</b></div>${b("secondary", `data-action="found-unhide" data-key="${esc(k)}"`, "Show again")}</div>`).join("")) : "")],
      ["controllers", SET_CATS.controllers, GI.pad, "The controllers on this PC, and the apps that change how games see them.", viewControllers],
      ["storage", SET_CATS.storage, GI.drive, "What your games take on each drive, and moving a game to another one.", viewStorage],
      ["plugins", SET_CATS.plugins, GI.puzzle, "Plugins are data only: they can add games to the Library, suggest save locations, fill in game details and change colours. They can't run anything, go online, or touch your saves.", viewPlugins],
      ["online", SET_CATS.online, GI.globe, "Each of these sends as little as it can, and only once you turn it on.", () => sub("What Lootnest may look up", "", `
          ${sw("findGames", "Store", "The Store page, and the search in the top bar, look any game up on the Steam store, with its deals and requirements, and on CheapShark for its price in other official stores. Only what you search for, or a game's Steam id, is sent.", s.findGames)}
          ${sw("saveDb", "Find save locations online", "Downloads PCGamingWiki's list of where over 20,000 games keep their saves (from the Ludusavi project, on GitHub) about once a week, so new games come with the right locations for profiles and cloud saves. Nothing about you or your games is sent.", s.saveDb)}
          ${sw("gameNews", "Show game updates and news", "Each game's page shows its latest updates and patch notes from Steam's news feed, and tells you when a game no store keeps up to date has a newer version than yours. Only the game's Steam id, or its name, is sent.", s.gameNews)}
          ${sw("gameDetails", "Game details", "Reads a game's summary, genres, release date, screenshots and review score from Steam when you open its page. Kept on this PC for 30 days.", s.gameDetails)}
          ${sw("achRarity", "How rare achievements are", "Each achievement shows the share of Steam's players who have it, from Steam's public numbers, on the game's page and as it unlocks. Only the game's Steam id is sent; no key is needed.", s.achRarity)}
          ${sw("downloadCovers", "Download covers and artwork", "Covers, wide artwork and logos for games Steam hasn't saved on this PC, looked up on the Steam store by name (and on SteamGridDB with a key below), saved on this PC. Only the game's name or Steam id is sent. Off: those games show their own icon.", s.downloadCovers)}`)
        + sub("itch.io", "Install and update the games in your itch.io library. Lootnest uses your own API key and only reads your library.", itchRows())
        + sub("GOG", "Install and update the games in your GOG library. You sign in the same way GOG Galaxy does, and Lootnest only reads your library and downloads your games' installers.", gogRows())
        + sub("Your own keys", "Optional. Each is encrypted on this PC and sent only to the site it's for.", `
          <div class="bk-row set-wide"><div><label for="s-grid"><b>SteamGridDB API key</b></label>
            <p>Covers and artwork for games the Steam store doesn't know. Get a free key on steamgriddb.com, under Preferences, API.</p></div>
            <input id="s-grid" class="ln-input" type="password" autocomplete="off" spellcheck="false" data-setting="gridDbKey" value="${esc(s.gridDbKey || "")}" placeholder="Paste your key"></div>
          <div class="bk-row set-wide"><div><label for="s-steamkey"><b>Steam Web API key</b></label>
            <p>Names, descriptions and pictures for achievements Lootnest finds on this PC without them, such as a game in a folder. Get a free key on steamcommunity.com/dev/apikey. Only the game's Steam id and your key are sent, to Steam; what comes back is kept on this PC.</p></div>
            <input id="s-steamkey" class="ln-input" type="password" autocomplete="off" spellcheck="false" data-setting="steamApiKey" value="${esc(s.steamApiKey || "")}" placeholder="Paste your key"></div>`)],
      ["updates", SET_CATS.updates, GI.load, "New versions of Lootnest, from its releases on GitHub.", () => card(versionRow + `
          ${sw("checkUpdates", "Check for updates automatically", "Every few hours, Lootnest asks GitHub for its latest version number and tells you with a notification when a new one is out. Nothing about you or your games is sent.", s.checkUpdates)}
          ${s.checkUpdates ? sw("autoInstallUpdates", "Install updates by themselves", "A new version installs while Lootnest is in the tray and no game is running, then Lootnest opens again in the tray. Your profiles, games and saves stay as they are.", s.autoInstallUpdates) : ""}`)],
      ["cloud", SET_CATS.cloud, GI.cloud, "Your saves in your own cloud folder, on every PC, with backups of everything.", () => card(`
          <div class="bk-row"><div><b>Cloud saves and backups</b><p>Where the saves go, sync between PCs, the password and backups of everything are on the Saves page.</p></div>${b("primary", 'data-action="nav" data-view="cloud"', "Open Cloud saves")}</div>`)],
      ["data", SET_CATS.data, GI.folder, "Where Lootnest keeps profiles and backups, and what it did lately.", () => card(`
          <div class="bk-row"><div><b>Data folder</b><p class="selectable">Profiles and backups are stored in ${esc(S.d.dataDir)}</p></div>${b("secondary", 'data-action="open-data"', "Open data folder")}</div>
          <div class="bk-row"><div><b>Import from Playnite</b><p>Bring in time played, statuses, favourites, ratings and notes from Playnite on this PC. Playnite's own files are only read.</p></div><button class="ln-button ln-button--secondary ln-button--sm" data-action="playnite-open" aria-haspopup="dialog">Import from Playnite</button></div>
          <div class="bk-row"><div><b>Recent activity</b><p>Every swap, save and scan, with the reason when something didn't work. Useful when you ask for help.</p></div>${b("secondary", 'data-action="show-log"', "Show recent activity")}</div>
          ${S.logText ? `<div class="log">${esc(S.logText)}</div>` : ""}`) + sub("Captures", "", capturesRows(s))],
      ["about", SET_CATS.about, GI.info, "The version, how to tell us about a problem, and how to leave.", () => card(`
          ${versionRow}
          <div class="bk-row"><div><b>Lootnest is in beta</b><p>Everything that moves your files makes a copy first. If something doesn't work as it should, Report a problem copies the version and the recent activity for you to paste, and opens a new issue on GitHub in your browser. Lootnest sends nothing itself. Read it before you post: it names your games and folders.</p></div>${b("secondary", 'data-action="report-problem"', "Report a problem")}</div>
          <div class="bk-row"><div><b>Uninstalling</b><p>Lootnest uses the Microsoft Edge WebView2 runtime. To uninstall, use Settings, Apps in Windows. Your profiles and backups stay in the data folder.</p></div></div>
          <div class="bk-row"><div><b>Quit Lootnest</b><p>It stops watching your games until you open it again.</p></div>${b("danger", 'data-action="quit"', "Quit Lootnest")}</div>`)],
    ];
    const cur = tabs.find((t) => t[0] === S.setTab) || tabs[0];
    return `<section class="settings-page ln-root" aria-label="Settings">
      <div class="set-layout">
        <nav class="set-nav" aria-label="Settings categories">${tabs.map(([k, label, ic]) => `<button class="set-tab" data-action="set-tab" data-tab="${k}"${k === cur[0] ? ' aria-current="true"' : ""}>${ic}<span>${label}</span>${k === "updates" && S.update.available ? '<span class="set-dot" aria-label="An update is ready"></span>' : ""}</button>`).join("")}</nav>
        <section class="set-panel" aria-labelledby="set-title"><div class="set-panel-head"><h1 id="set-title">${cur[1]}</h1><p>${cur[3]}</p></div>${cur[4]()}</section>
      </div></section>`;
  }

  // ---------------------------------------------------------------- backup --
  // Backups go to a folder a cloud app syncs; Lootnest writes the file and
  // the app uploads it.
  async function loadBackupInfo() {
    try { S.backup = await api.backupInfo(); } catch (e) { return; }
    if ((S.view === "settings" || S.view === "saves") && !S.dialog) render();
  }
  function fmtBytes(n) {
    if (n < 1024 * 1024) return Math.max(1, Math.round(n / 1024)) + " KB";
    if (n < 1024 * 1024 * 1024) return (n / 1048576).toFixed(n < 10 * 1048576 ? 1 : 0) + " MB";
    return (n / 1073741824).toFixed(1) + " GB";
  }
  // Achievements: every game's on this PC, for one profile at a time. Games
  // whose achievements go with the profile show that profile's; the others
  // (Steam's account, or one set kept on the PC) are shared.
  // Lootnest's own milestones, for any game, from what Lootnest already
  // counts for each profile: time, days in a row, games, sessions and the
  // achievements read on this PC.
  function milestones(p, rows) {
    const st = playerStats(p), h = st.total / 3600, unlocked = rows.reduce((n, r) => n + r.unlocked, 0);
    const perfect = rows.filter((r) => r.unlocked && r.unlocked === r.g.total).length;
    const top = st.top && st.topT ? st.topT / 3600 : 0;
    const longest = st.longest ? st.longest.seconds / 3600 : 0;
    const m = (name, desc, cur, goal, unit) => ({ name, desc, cur: Math.min(cur, goal), goal, unit, done: cur >= goal });
    return [
      m("First session", "Play a game through Lootnest", st.sessions, 1, ""),
      m("Ten hours in", "Play for 10 hours in all", h, 10, "h"),
      m("A hundred hours", "Play for 100 hours in all", h, 100, "h"),
      m("Regular", "Play 7 days in a row", st.best, 7, "days"),
      m("Every day for a month", "Play 30 days in a row", st.best, 30, "days"),
      m("Explorer", "Play 5 different games", st.played, 5, "games"),
      m("Wide library", "Play 25 different games", st.played, 25, "games"),
      m("Marathon", "Play for 4 hours in one go", longest, 4, "h"),
      m(st.top ? "Devoted to " + st.top.name : "Devoted", "Play one game for 50 hours", top, 50, "h"),
      m("Collector", "Unlock 100 achievements", unlocked, 100, ""),
      m("Completionist", "Get every achievement in a game", perfect, 1, ""),
    ];
  }
  const fmtGoal = (x) => (x.unit === "h" ? Math.floor(x.cur) + " of " + x.goal + " h" : Math.floor(x.cur) + " of " + x.goal + (x.unit ? " " + x.unit : ""));

