  // -------------------------------------------------------------- settings --
  // A Settings row: its label on the left, at most one short line under it
  // (only where it changes files or sends something online), its control on
  // the right. invert: the setting turns the thing off (on by default), so
  // the switch shows the opposite.
  const setLine = (desc) => (!desc ? "" : desc.startsWith("<") ? desc : `<p>${desc}</p>`);
  function sw(field, title, desc, checked, invert) {
    return `<label class="toggle"><span><b>${title}</b>${setLine(desc)}</span>
      <span class="switch"><input type="checkbox" data-setting="${field}"${invert ? ' data-invert="1"' : ""} ${checked ? "checked" : ""}><span></span></span></label>`;
  }
  const setRow = (title, desc, control, forId) => `<div class="bk-row"><div>${forId ? `<label for="${forId}"><b>${title}</b></label>` : `<b>${title}</b>`}${setLine(desc)}</div>${control || ""}</div>`;

  const SHORTCUTS = (s) => [
    [["Ctrl+K"], "Run a command or search"],
    [["Ctrl+F", "/"], "Find a game in the Library"],
    [["Arrow keys"], "Move between covers"],
    [["Home", "End"], "First or last cover"],
    [["Esc"], "Close, or go back from a game's page"],
    [["Ctrl+Tab", "Ctrl+Shift+Tab"], "Next or previous tab on a game's page"],
    [["Alt+Left", "Backspace", "Mouse back button"], "Back"],
    [["Alt+Right", "Mouse forward button"], "Forward"],
    [["Ctrl+B"], "Fold or open the sidebar"],
    [["F5"], "Look for games again"],
    [["1 to 9"], "Choose a profile in Who's playing"],
    [["F11"], "Open or leave Big Picture"],
    [["Q", "E"], "Switch tabs in Big Picture"],
    ...(s.pointHotkeyOff ? [] : [[["Ctrl+Shift+F5"], "Make a save point while a game runs"]]),
    ...(s.screenshotsOff ? [] : [[["Ctrl+Shift+F12"], "Save a screenshot while a game runs"]]),
  ];
  // Captures from other apps: the switch, the folders read, and the way to
  // the ones matched to no game.
  let unsortedAsked = false;
  function capturesRows(s) {
    const U = S.unsorted;
    if (!s.capturesOff && !unsortedAsked) { unsortedAsked = true; api.unsortedCaptures().catch(() => {}); }
    const rows = !s.capturesOff && U ? setRow("Folders Lootnest reads", U.folders.length ? `<ul class="set-folders selectable">${U.folders.map((f) => `<li><b>${esc(f.source)}</b> ${esc(f.dir)}</li>`).join("")}</ul>` : "None on this PC yet")
      + setRow("Unsorted captures", U.total ? `${U.total} not matched to a game` : "None", U.total ? `<button class="ln-button ln-button--secondary ln-button--sm" data-action="captures-unsorted" aria-haspopup="dialog">Choose their games</button>` : "") : "";
    return sw("capturesOff", "Show captures from other apps", "Game Bar, Steam, NVIDIA, AMD and OBS. Only read, never moved.", !s.capturesOff, true) + rows;
  }
  // A group of rows under its label, with an optional short line.
  const setSub = (title, note, body) => `<section class="set-sub">${title ? `<div class="ln-section-head"><div><h2>${title}</h2></div></div>` : ""}${note ? `<p class="set-note">${note}</p>` : ""}${body}</section>`;
  // The categories, by key, as the list on the left names them.
  const SET_CATS = { general: "General", appearance: "Appearance", games: "Games and swaps", controllers: "Controllers", storage: "Storage", plugins: "Plugins", online: "Online", updates: "Updates", cloud: "Cloud saves", data: "Data and activity", about: "About" };
  function viewSettings() {
    const s = cfg().settings, st = S.d.steam, hidden = Object.entries((S.found && S.found.hidden) || {});
    const b = (kind, attrs, label) => `<button class="ln-button ln-button--${kind} ln-button--sm" ${attrs}>${label}</button>`;
    const num = (id, field, min, max) => `<input id="${id}" class="ln-input set-num ln-num" type="number" min="${min}" max="${max}" data-setting="${field}" value="${s[field]}">`;
    const key = (id, field) => `<input id="${id}" class="ln-input set-input" type="password" autocomplete="off" spellcheck="false" data-setting="${field}" value="${esc(s[field] || "")}" placeholder="Paste your key">`;
    const update = S.update.available
      ? (S.update.step ? `<button class="ln-button ln-button--primary ln-button--sm" aria-busy="true"><span class="ln-spinner" aria-hidden="true"></span>Updating</button>` : b("primary", 'data-action="update-now"', `Update to ${esc(S.update.version)}`))
      : (S.update.checking ? `<button class="ln-button ln-button--secondary ln-button--sm" aria-busy="true"><span class="ln-spinner" aria-hidden="true"></span>Checking</button>` : b("secondary", 'data-action="update-check"', "Check for updates"));
    const sub = (title, note, rows) => setSub(title, note, `<div class="bk-card">${rows}</div>`);
    const versionRow = setRow(`Lootnest <span class="ln-num">${esc(S.d.version)}</span>${S.d.stage ? " " + esc(S.d.stage) : ""}`, S.update.available ? `Version ${esc(S.update.version)} is ready` : "", update);
    // One category at a time, picked on the left, as in Steam's settings.
    const tabs = [
      ["general", SET_CATS.general, GI.sliders, () => sub("Start and close", "", `
          ${sw("autostart", "Start with Windows", "", S.d.autostart)}
          ${sw("closeToTray", "Keep running in the tray when the window is closed", "", s.closeToTray)}
          ${sw("hideWhilePlaying", "Hide while a game runs", "", s.hideWhilePlaying)}
          ${sw("openBigPicture", "Open in Big Picture", "", s.openBigPicture)}
          ${sw("notifications", "Notify me when progress is saved", "", s.notifications)}`)
        + sub("Keyboard shortcuts", "", SHORTCUTS(s).map(([keys, what]) => `<div class="bk-row set-key"><div>${what}</div><span class="set-kbds">${keys.map((k) => `<kbd>${k}</kbd>`).join("")}</span></div>`).join(""))],
      ["appearance", SET_CATS.appearance, ph("palette"), () => sub("", "", `
          ${sw("glassOff", "See-through panels", "Off makes every panel solid", !s.glassOff, true)}
          ${sw("roomOff", "Art behind pages", "The game you're looking at fills the window", !s.roomOff, true)}
          ${setRow("Motion", "", `<div class="ln-segmented" role="radiogroup" aria-label="Motion">${[["", "Full"], ["less", "Less"], ["off", "Off"]].map(([v, l]) => `<button type="button" role="radio" aria-checked="${(s.motion || "") === v}" data-action="set-motion" data-value="${v}">${l}</button>`).join("")}</div>`)}`)],
      ["games", SET_CATS.games, ph("arrows-left-right"), () => sub("Swaps", "", `
          ${setRow("Copies kept per game before each swap", "The oldest go once there are more.", num("s-backups", "backupsToKeep", 1, 100), "s-backups")}
          ${setRow("Seconds to wait for a game to start", "", num("s-timeout", "startTimeoutSeconds", 30, 1800), "s-timeout")}
          ${sw("watchOutsideLaunches", "Follow games started from Steam or elsewhere", "Their progress is saved to whoever's files are loaded.", s.watchOutsideLaunches)}`)
        + sub("While you play", "", `
          ${sw("achToastsOff", "Show achievements as they unlock", "", !s.achToastsOff, true)}
          ${s.achToastsOff ? "" : setRow("Preview an unlock", "", b("secondary", 'id="s-toast" data-action="test-toast"', "Show one"))}
          ${sw("soundsOff", "Sounds", "", !s.soundsOff, true)}
          ${sw("pointHotkeyOff", "Ctrl+Shift+F5 makes a save point", "", !s.pointHotkeyOff, true)}
          ${sw("screenshotsOff", "Ctrl+Shift+F12 saves a screenshot", "Into Pictures\\Lootnest\\Screenshots.", !s.screenshotsOff, true)}`)
        + sub("Updates for your games", "", `
          ${sw("updatePointOff", "Make a save point before a game updates", "", !s.updatePointOff, true)}
          ${sw("downloadsOff", "Look in Downloads for updates", "", !s.downloadsOff, true)}`)
        + sub("Steam", "", `
          ${sw("autoSteamHooks", "Set up Steam automatically", "Changes each game's launch options in Steam.", s.autoSteamHooks)}
          ${st.found ? setRow("Steam account", st.accountError ? `<p class="error-text">${esc(st.accountError)}</p>` : "", `<select id="s-steam" class="ln-input set-input" data-setting="steamAccountId">
            <option value="">Automatic${st.accountInUse && !s.steamAccountId ? " (" + esc(st.accountInUse) + ")" : ""}</option>
            ${(st.accounts || []).map((a) => `<option value="${esc(a)}" ${a === s.steamAccountId ? "selected" : ""}>${esc(a)}</option>`).join("")}</select>`, "s-steam") : setRow("Steam isn't on this PC", "")}`)
        + (hidden.length ? sub("Hidden games", "", hidden.map(([k, name]) => setRow(esc(name), "", b("secondary", `data-action="found-unhide" data-key="${esc(k)}"`, "Show again"))).join("")) : "")],
      ["controllers", SET_CATS.controllers, GI.pad, viewControllers],
      ["storage", SET_CATS.storage, GI.drive, viewStorage],
      ["plugins", SET_CATS.plugins, GI.puzzle, viewPlugins],
      ["online", SET_CATS.online, GI.globe, () => sub("What Lootnest looks up", "", `
          ${sw("findGames", "Store", "Sends what you search for to Steam and CheapShark.", s.findGames)}
          ${sw("saveDb", "Find save locations online", "Downloads a list of save locations from GitHub weekly.", s.saveDb)}
          ${sw("gameNews", "Game updates and news", "Sends a game's Steam id or name to Steam.", s.gameNews)}
          ${sw("gameDetails", "Game details", "Sends a game's Steam id to Steam when you open its page.", s.gameDetails)}
          ${sw("achRarity", "How rare achievements are", "Sends a game's Steam id to Steam.", s.achRarity)}
          ${sw("downloadCovers", "Download covers and artwork", "Sends a game's name or Steam id to Steam and SteamGridDB.", s.downloadCovers)}`)
        + sub("itch.io", "", itchRows())
        + sub("GOG", "", gogRows())
        + sub("Your own keys", "Encrypted on this PC and sent only to the site each is for.", `
          ${setRow("SteamGridDB API key", "Free on steamgriddb.com, under Preferences, API.", key("s-grid", "gridDbKey"), "s-grid")}
          ${setRow("Steam Web API key", "Free on steamcommunity.com/dev/apikey.", key("s-steamkey", "steamApiKey"), "s-steamkey")}`)],
      ["updates", SET_CATS.updates, GI.load, () => sub("", "", versionRow + `
          ${sw("checkUpdates", "Check for updates automatically", "Asks GitHub for the latest version every few hours.", s.checkUpdates)}
          ${s.checkUpdates ? sw("autoInstallUpdates", "Install updates by themselves", "", s.autoInstallUpdates) : ""}`)],
      ["cloud", SET_CATS.cloud, GI.cloud, () => sub("", "", setRow("Sync folder, versions and encryption", "", b("secondary", 'data-action="nav" data-view="cloud"', "Open")))],
      ["data", SET_CATS.data, GI.folder, () => sub("Your data", "", `
          ${setRow("Data folder", `<p class="selectable">${esc(S.d.dataDir)}</p>`, b("secondary", 'data-action="open-data"', "Open"))}
          ${setRow("Import from Playnite", "", `<button class="ln-button ln-button--secondary ln-button--sm" data-action="playnite-open" aria-haspopup="dialog">Import</button>`)}
          ${setRow("Recent activity", "", b("secondary", 'data-action="show-log"', "Show"))}
          ${S.logText ? `<div class="log">${esc(S.logText)}</div>` : ""}`) + sub("Captures", "", capturesRows(s))],
      ["about", SET_CATS.about, GI.info, () => sub("", "", `
          ${versionRow}
          ${setRow("Report a problem", "Copies the recent activity to paste on GitHub. It names your games and folders.", b("secondary", 'data-action="report-problem"', "Copy and open GitHub"))}
          ${setRow("Uninstall", "In Windows Settings, Apps. Profiles and backups stay in the data folder.")}
          ${setRow("Quit Lootnest", "", b("danger", 'data-action="quit"', "Quit"))}`)],
    ];
    const cur = tabs.find((t) => t[0] === S.setTab) || tabs[0];
    return `<section class="settings-page ln-root" aria-label="Settings">
      <div class="set-layout">
        <nav class="set-nav" aria-label="Settings categories">${tabs.map(([k, label, ic]) => `<button class="set-tab" data-action="set-tab" data-tab="${k}" data-sec="${SET_SEC[k] || "neutral"}"${k === cur[0] ? ' aria-current="true"' : ""}><span class="set-ic">${ic}</span><span>${label}</span>${k === "updates" && S.update.available ? '<span class="set-dot" aria-label="An update is ready"></span>' : ""}</button>`).join("")}</nav>
        <section class="set-panel" aria-labelledby="set-title"><h1 id="set-title">${cur[1]}</h1>${cur[3]()}</section>
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

