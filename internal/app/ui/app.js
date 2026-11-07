"use strict";
(function () {
  // Lootnest's window never opens other windows (links open in the browser).
  try { window.open = () => null; } catch (e) { /* read-only */ }
  // ------------------------------------------------------------------ api --
  const call = (name, ...args) => {
    const f = window[name];
    if (typeof f !== "function") return Promise.reject("Lootnest isn't ready yet.");
    return f(...args);
  };
  const api = {
    ready: () => call("ps_ready"),
    state: () => call("ps_state"),
    savePlayers: (p) => call("ps_savePlayers", p),
    saveGame: (g) => call("ps_saveGame", g),
    deleteGame: (id) => call("ps_deleteGame", id),
    saveSettings: (s) => call("ps_saveSettings", s),
    steamGames: () => call("ps_steamGames"),
    inspectSteam: (id) => call("ps_inspectSteam", id),
    inspectExe: (p) => call("ps_inspectExe", p),
    checkPath: (p) => call("ps_checkPath", p),
    browse: (id, kind, start) => call("ps_browse", id, kind, start || ""),
    play: (r) => call("ps_play", r),
    load: (r) => call("ps_load", r),
    closeGame: (id) => call("ps_closeGame", id),
    profileInfo: (id) => call("ps_profileInfo", id),
    achievements: (key, appId, dir, name) => call("ps_achievements", key, appId || 0, dir || "", name || ""),
    setSharedAchievements: (id, shared) => call("ps_setSharedAchievements", id, !!shared),
    allAchievements: () => call("ps_allAchievements"),
    setTags: (key, tags) => call("ps_setTags", key, tags),
    setShelf: (pid, key, e) => call("ps_setShelf", pid, key, e),
    shortcut: (id) => call("ps_shortcut", id),
    open: (kind, arg) => call("ps_open", kind, arg || ""),
    log: () => call("ps_log"),
    quit: () => call("ps_quit"),
    steamSetup: () => call("ps_steamSetup"),
    copy: (t) => call("ps_copy", t),
    steamPick: (r) => call("ps_steamPick", r),
    cancelRequest: (id) => call("ps_cancelRequest", id),
    found: () => call("ps_found"),
    scan: () => call("ps_scan"),
    ignoreFound: (k) => call("ps_ignoreFound", k),
    addFound: (keys) => call("ps_addFound", keys),
    foundGame: (k) => call("ps_foundGame", k),
    applyHooks: (restart) => call("ps_applyHooks", restart),
    cover: (appId, name, key) => call("ps_cover", appId, name, key),
    launchFound: (key) => call("ps_launchFound", key),
    checkUpdate: () => call("ps_checkUpdate"),
    installUpdate: () => call("ps_installUpdate"),
    cloudStatus: (id) => call("ps_cloudStatus", id),
    fixBlocked: (gameId, playerId) => call("ps_fixBlocked", gameId, playerId || ""),
    moveInfo: (appId) => call("ps_moveInfo", appId),
    moveGame: (appId, lib) => call("ps_moveGame", appId, lib),
    saveLibraryPref: (key, name, args) => call("ps_saveLibraryPref", key, name, args),
    addLibraryGame: (kind, appId, path) => call("ps_addLibraryGame", kind, appId || 0, path || ""),
    gameIcon: (exe) => call("ps_gameIcon", exe),
    art: (appId, name, key, kind) => call("ps_art", appId || 0, name, key, kind),
    setAvatar: (id, path) => call("ps_setAvatar", id, path),
    removeAvatar: (id) => call("ps_removeAvatar", id),
    setBanner: (id, path) => call("ps_setBanner", id, path),
    removeBanner: (id) => call("ps_removeBanner", id),
    backupInfo: () => call("ps_backupInfo"),
    setBackup: (b) => call("ps_setBackup", b),
    setBackupPassword: (pw) => call("ps_setBackupPassword", pw),
    backupNow: () => call("ps_backupNow"),
    restoreOpen: (file, pw) => call("ps_restoreOpen", file, pw || ""),
    restoreApply: () => call("ps_restoreApply"),
    restoreCancel: () => call("ps_restoreCancel"),
    importApply: () => call("ps_importApply"),
    exportPlayer: (id, dir, pw) => call("ps_exportPlayer", id, dir, pw || ""),
    savesStatus: () => call("ps_savesStatus"),
    syncNow: () => call("ps_syncNow"),
    syncResolve: (gameId, playerId, answer) => call("ps_syncResolve", gameId, playerId, answer),
    gameSaves: (key) => call("ps_gameSaves", key),
    gameSaveNow: (key) => call("ps_gameSaveNow", key),
    gameSaveRestore: (key, id) => call("ps_gameSaveRestore", key, id),
    gameSaveRestoreLocal: (key, id) => call("ps_gameSaveRestoreLocal", key, id),
    findGames: (q) => call("ps_findGames", q),
    gameOffer: (appId, name) => call("ps_gameOffer", appId, name),
    storeFront: () => call("ps_storeFront"),
    updates: () => call("ps_updates"),
    updatePlan: (key, path) => call("ps_updatePlan", key, path),
    updateApply: (key, path) => call("ps_updateApply", key, path),
    updateUndo: (key, id) => call("ps_updateUndo", key, id),
    updateInstaller: (key, path, full) => call("ps_updateInstaller", key, path, !!full),
    setSeat: (id) => call("ps_setSeat", id || ""),
    testToast: () => call("ps_testToast"),
    achDoctor: (key, appId, dir, name) => call("ps_achDoctor", key, appId || 0, dir || "", name),
    setNameInGame: (key, dir, pid) => call("ps_setNameInGame", key, dir || "", pid),
    setOffline: (key, on) => call("ps_setOffline", key, !!on),
    savePoints: (key) => call("ps_savePoints", key),
    makeSavePoint: (key, name) => call("ps_makeSavePoint", key, name || ""),
    restoreSavePoint: (key, id) => call("ps_restoreSavePoint", key, id),
    renameSavePoint: (key, id, name) => call("ps_renameSavePoint", key, id, name),
    deleteSavePoint: (key, id) => call("ps_deleteSavePoint", key, id),
    branchSavePoint: (key, id, name) => call("ps_branchSavePoint", key, id, name || ""),
    saveCheck: (fresh) => call("ps_saveCheck", !!fresh),
    reportProblem: () => call("ps_reportProblem"),
    newCopies: () => call("ps_newCopies"),
    carryOver: (oldKey, newKey) => call("ps_carryOver", oldKey, newKey),
    dismissCopy: (oldKey, newKey) => call("ps_dismissCopy", oldKey, newKey),
    deleteOldCopy: (oldKey) => call("ps_deleteOldCopy", oldKey),
    keepSavesNow: (key) => call("ps_keepSavesNow", key),
    setGameSaves: (key, paths) => call("ps_setGameSaves", key, paths),
    gameNews: (key, appId, name, exe) => call("ps_gameNews", key, appId || 0, name, exe || ""),
    openGameFolder: (kind, id) => call("ps_openGameFolder", kind, id),
    foundShortcut: (key) => call("ps_foundShortcut", key),
    steamVerify: (appId) => call("ps_steamVerify", appId),
  };

  // Small per-PC preferences (sort, cover size), kept in the WebView's own
  // storage; the app works the same without it.
  function pref(k, d) { try { return localStorage.getItem("ln." + k) || d; } catch (e) { return d; } }
  function setPref(k, v) { try { localStorage.setItem("ln." + k, v); } catch (e) { /* not kept */ } }

  // ---------------------------------------------------------------- state --
  const S = {
    pinfo: {},        // game id -> player id -> {saved, size} of their profile
    ach: {},          // Library key -> achievements read from this PC (null: none)
    doctor: {},       // Library key -> Achievement Doctor's check (achDoctor event)
    points: {},       // Library key -> its save points (savePoints event)
    achWho: {},       // Library key -> whose achievements are shown
    setTab: pref("setTab", "general"), // the Settings category open
    allAch: null,     // the Achievements page: every game's, null until read
    achPage: "",      // whose the Achievements page shows
    statusWho: {},    // Library key -> whose status the game page shows
    tag: "",          // the Library's collection filter ("" for all)
    status: "",       // the Library's status filter ("" for any)
    gsaves: {},       // Library key -> cloud saves of a game outside Steam
    find: { q: "", hits: null, error: "" }, // Store: the search and what came back
    front: null,      // Store rows before a search: {daily, specials, top, deals} or {error}
    gq: { q: "", hits: null, sel: -1 }, // the search in the sidebar
    updates: null,    // the Updates page: {games, newsOn} from ps_updates
    copies: null,     // "Is this a new copy?" offers from ps_newCopies
    carried: {},      // old key -> the result of carrying it over (for the page)
    offers: {},       // Steam app id -> {offer, pc, owned} or {error}, for Find games
    d: null,          // AppState from Go
    view: "home",
    from: "home",      // the view a game's page goes back to
    selected: null,    // {kind, id} of the game last opened: the Library's colours follow it
    q: "",             // library search
    filter: "all",     // library: "all" or "profiles"
    store: "all",      // library: a store or launcher, or "all"
    who: "",           // library: only the games this player has a profile in or played last
    covers: {},        // cover key -> data: URL, or "" when there is none (yet)
    icons: {},         // cover key -> the game exe's icon as a data: URL, or "" (shown when there's no cover)
    art: {},           // "hero:<appid>" / "logo:<appid>" -> Steam's wide art and logo, or ""
    news: {},          // cover key -> {items, newer, none, error} from Steam's news feed, or {loading: true}
    page: null,        // the game page: {kind: "game"|"found", id, cloud}
    libScroll: 0,      // where the Library was scrolled to, for coming back
    sort: pref("sort", "name"),       // Library order: name | recent | played
    density: pref("density", "m"),    // cover size: s | m | l
    ctx: null,         // the right-click menu: {x, y, kind, id}
    update: {},        // {available, version, page, upToDate, checking, error, step}
    animateView: true,  // fade the page in on the next render
    menuFor: null,
    picker: null,     // {gameId, mode, ask, playerId, owner, newMode, working}
    dialog: null,     // {kind, ...}
    edit: null,       // editor state
    players: null,    // players being edited
    logText: "",
    hooks: null,        // Steam launch option status per game
    hideHookBanner: false,
    found: { scanning: true, games: [] },
    adding: false,
    hooksBusy: false,
    moving: null,     // {appId, name, step, done, total} while Game Mover copies a game
  };
  const browseWaiters = {};
  let seq = 0;

  const $ = (sel, root = document) => root.querySelector(sel);
  const esc = (s) => String(s == null ? "" : s).replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
  const errText = (e) => (typeof e === "string" ? e : (e && e.message) || "Something went wrong.");
  const cfg = () => S.d.config;
  const player = (id) => cfg().players.find((p) => p.id === id);
  const game = (id) => cfg().games.find((g) => g.id === id);
  const color = (id) => (player(id) || {}).color || "var(--dim)";
  const pname = (id) => (player(id) || {}).name || "Someone";
  // Runs: a profile can be a run of a person ("Cristian hardcore" of
  // Cristian). It shows under them, with their colour and picture.
  const MAX_PROFILES = 16;
  const parentOf = (p) => (p && p.of && p.of !== p.id && player(p.of)) || null;
  // "hardcore" for "Cristian hardcore" of Cristian.
  const runLabel = (p) => {
    const par = parentOf(p);
    if (!par) return "";
    const n = p.name.trim();
    return n.toLowerCase().startsWith(par.name.toLowerCase() + " ") ? n.slice(par.name.length).trim() : n;
  };
  // People in order, each followed by their runs.
  function grouped(list) {
    list = list || cfg().players;
    const ids = new Set(list.map((p) => p.id));
    const isRun = (p) => p.of && p.of !== p.id && ids.has(p.of);
    const out = [];
    list.filter((p) => !isRun(p)).forEach((p) => { out.push(p); list.filter((r) => isRun(r) && r.of === p.id).forEach((r) => out.push(r)); });
    list.forEach((p) => { if (!out.includes(p)) out.push(p); });
    return out;
  }
  const session = (gid) => (S.d.sessions || []).find((s) => s.gameId === gid);
  const hasProfile = (gid, pid) => ((S.d.profiles || {})[gid] || []).includes(pid);
  const possessive = (n) => (/s$/i.test(n) ? n + "’" : n + "’s");
  const fmtSize = (b) => (b < 1024 ? b + " B" : b < 1048576 ? (b / 1024).toFixed(0) + " KB" : b < 1073741824 ? (b / 1048576).toFixed(1) + " MB" : (b / 1073741824).toFixed(2) + " GB");
  const short = (name) => (name.length > 22 ? name.slice(0, 20).trim() + "…" : name);

  const icon = {
    dots: '<svg viewBox="0 0 16 16"><circle cx="3" cy="8" r="1.4" fill="currentColor"/><circle cx="8" cy="8" r="1.4" fill="currentColor"/><circle cx="13" cy="8" r="1.4" fill="currentColor"/></svg>',
    folder: '<svg viewBox="0 0 16 16"><path d="M1.75 3.5h4l1.5 1.5h7v7.25a.75.75 0 0 1-.75.75H2.5a.75.75 0 0 1-.75-.75z" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linejoin="round"/></svg>',
    x: '<svg viewBox="0 0 16 16"><path d="M4 4l8 8M12 4l-8 8" stroke="currentColor" stroke-width="1.4" stroke-linecap="round"/></svg>',
    back: '<svg viewBox="0 0 16 16"><path d="M10 3 5 8l5 5" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>',
    plus: '<svg viewBox="0 0 16 16"><path d="M8 3v10M3 8h10" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>',
    play: '<svg viewBox="0 0 16 16" aria-hidden="true"><path d="M5 3.2v9.6a.6.6 0 0 0 .92.5l7.3-4.8a.6.6 0 0 0 0-1L5.92 2.7A.6.6 0 0 0 5 3.2z" fill="currentColor"/></svg>',
  };

  // --------------------------------------------------------------- toasts --
  function toast(text, kind) {
    const el = document.createElement("div");
    el.className = "ln-toast ln-root" + (kind === "error" ? " ln-toast--error" : " ln-toast--success");
    el.setAttribute("role", kind === "error" ? "alert" : "status");
    el.innerHTML = kind === "error" ? GI.alert : '<svg class="ln-icon" viewBox="0 0 20 20" aria-hidden="true"><circle cx="10" cy="10" r="6.75"/><path d="M7.25 10.25 9.1 12.1l3.65-3.85"/></svg>';
    const p = document.createElement("p");
    p.textContent = text;
    el.appendChild(p);
    $("#toasts").appendChild(el);
    setTimeout(() => el.remove(), kind === "error" ? 7000 : 3800);
  }

  // ---------------------------------------------------------------- events --
  window.__ps = function (ev) {
    switch (ev.type) {
      case "sessions":
        if (!S.d) return;
        S.d.sessions = ev.sessions || [];
        S.d.active = ev.active || {};
        S.pinfo = {}; // a swap or a session's end writes profiles: read again
        achAsked.clear();
        S.allAch = null;
        onSessions();
        refresh();
        break;
      case "launch":
        openPicker(ev.gameId, ev.reqId ? "steam" : "play", ev.reqId);
        break;
      case "found":
        loadFound();
        break;
      case "hooks":
        loadHooks().then(() => { render(); if (S.dialog && S.dialog.kind === "steam") renderOverlay(); });
        break;
      case "blocked":
        if (S.picker) { S.picker = null; render(); }
        S.menuFor = null;
        S.dialog = { kind: "blocked", gameId: ev.gameId, playerId: ev.playerId || "" };
        renderOverlay();
        setTimeout(() => { const b = $('[data-action="fix-blocked"]'); if (b) b.focus(); }, 20);
        break;
      case "blockedDone": {
        const D = S.dialog && S.dialog.kind === "blocked" ? S.dialog : null;
        if (ev.error) {
          if (D) { D.busy = false; renderOverlay(); }
          toast(ev.error, "error");
          break;
        }
        if (D) { S.dialog = null; renderOverlay(); }
        refresh();
        loadHooks().then(render);
        const g = game(ev.gameId);
        if (ev.need) openPicker(ev.gameId, "play");
        else toast(`Starting ${g ? g.name : "the game"} for ${pname(ev.playerId)}. Next time, start it from the ${g ? g.name : "game"} (Lootnest) shortcut on your desktop to choose who's playing.`);
        break;
      }
      case "update":
        if (ev.available && !ev.manual && S.update.version !== ev.version) toast(`Lootnest ${ev.version} is out. Update from the sidebar.`);
        S.update = Object.assign({}, ev, { checking: false });
        if (ev.manual && ev.upToDate) toast(`Lootnest ${S.d.version} is the latest version.`);
        if (ev.error) toast(ev.error, "error");
        render();
        break;
      case "updateStep":
        if (ev.error) { S.update.step = ""; toast(ev.error, "error"); } else S.update.step = ev.step;
        render();
        break;
      case "cover":
        pendingCovers.delete(ev.key);
        S.covers[ev.key] = ev.url;
        showCover(ev.key);
        break;
      case "art":
        S.art[ev.key] = ev.url;
        if (S.view === "game" || S.view === "library" || S.view === "home") render();
        break;
      case "achievements":
        S.ach[ev.key] = ev.set || null;
        if (S.view === "game") render();
        break;
      case "saveCheck":
        S.saveCheck = ev.check || null;
        if (S.view === "home") render();
        break;
      case "savePoints":
        S.points[ev.key] = Object.assign({}, S.points[ev.key] || {}, { points: ev.points || [], error: ev.error || "", player: ev.player || "", running: !!ev.running, profiles: !!ev.profiles, loaded: true });
        if (S.view === "game") render();
        if (S.dialog && S.dialog.kind === "points") renderOverlay();
        break;
      case "savePointMade": {
        const P = S.points[ev.key] || (S.points[ev.key] = {});
        P.busy = false;
        if (ev.error) toast(ev.error, "error");
        else if (ev.restored) toast(`Back to ${ev.point ? ev.point.name : "the save point"}. The saves from before are kept as a point.`);
        else if (ev.point) toast(`Save point made: ${ev.point.name}`);
        if (S.view === "game") render();
        if (S.dialog && S.dialog.kind === "points") renderOverlay();
        break;
      }
      case "offline":
        if (ev.error) toast(ev.error, "error");
        else toast(ev.on ? "Kept offline. Windows Firewall blocks it from connecting." : "It can go online again.");
        refresh();
        break;
      case "achDoctor":
        S.doctor[ev.key] = ev;
        if (S.dialog && S.dialog.kind === "achdoc") renderOverlay();
        break;
      case "achUnlocked":
        // A game running now unlocked something: its page and the
        // Achievements page read it again.
        achAsked.delete(ev.key); S.allAch = null;
        if (S.view === "game" || S.view === "achievements") render();
        break;
      case "allAchievements":
        S.allAch = { games: ev.games || [] };
        if (S.view === "achievements") render();
        break;
      case "profileInfo":
        S.pinfo[ev.gameId] = ev.info || {};
        if (S.picker && S.picker.gameId === ev.gameId) renderOverlay();
        if (S.view === "game" || S.view === "home") render();
        break;
      case "moveInfo": {
        const D = S.dialog && S.dialog.kind === "move" && S.dialog.appId === ev.info.appId ? S.dialog : null;
        if (!D) break;
        D.info = ev.info;
        const room = (l) => l.free < 0 || l.free >= ev.info.size + MOVE_MARGIN;
        const first = (ev.info.libraries || []).find((l) => !l.current && room(l));
        D.pick = first ? first.path : "";
        renderOverlay();
        break;
      }
      case "moveProgress":
        S.moving = Object.assign(S.moving || {}, ev);
        if (S.dialog && S.dialog.kind === "move") renderOverlay();
        break;
      case "moveDone": {
        S.moving = null;
        if (S.dialog && S.dialog.kind === "move" && S.dialog.appId === ev.appId) { S.dialog = null; renderOverlay(); }
        if (ev.error) toast(`Couldn't move ${ev.name}: ${ev.error}${/[.!?]$/.test(ev.error) ? "" : "."}`, "error");
        else if (ev.leftover) toast(`${ev.name} moved. Some old files were in use and are still in ${ev.leftover}; you can delete that folder.`);
        else toast(`${ev.name} is now in ${ev.lib}.`);
        refresh();
        break;
      }
      case "news":
        S.news[ev.key] = ev;
        if (S.view === "game" || S.view === "library" || S.view === "home") render();
        break;
      case "avatar":
      case "banner": {
        const D = S.dialog && S.dialog.kind === "profile" ? S.dialog : null;
        if (D) { D.busy = false; if (!ev.error && ev.type === "banner") D.p.banner = "image"; if (!ev.error && ev.type === "avatar") D.p.avatar = true; }
        if (ev.error) toast(ev.error, "error");
        refresh();
        break;
      }
      case "backup": {
        const D = S.dialog;
        if (ev.password && D && D.kind === "backup-pw") {
          if (ev.error) { D.busy = false; D.error = ev.error; renderOverlay(); }
          else { const on = D.on; S.dialog = null; renderOverlay(); toast(on ? "Saves and backups are encrypted" : "Saves and backups aren't encrypted any more"); S.saves = null; loadBackupInfo(); loadSaves(); api.syncNow().catch(() => {}); }
        }
        if (ev.done && S.backupAsked) { S.backupAsked = false; if (ev.error) toast(ev.error, "error"); else toast("Backed up"); }
        loadBackupInfo();
        break;
      }
      case "findGames": {
        if (ev.q === S.gq.q) { S.gq.hits = ev.hits || []; renderGq(); }
        if (ev.q !== S.find.q) break; // an older search
        S.find.hits = ev.hits || null; S.find.error = ev.error || "";
        const r = $("#find-results"); if (r) r.innerHTML = findResults();
        break;
      }
      case "updates": {
        S.updates = { games: ev.games || [], newsOn: !!ev.newsOn };
        updateNavCount();
        if (S.view === "updates") render();
        break;
      }
      case "newCopies": {
        if (ev.offers) { S.copies = ev.offers; updateNavCount(); if (S.view === "updates" || S.view === "game") render(); }
        else api.newCopies().catch(() => {});
        break;
      }
      case "carried": {
        const C = S.carried[ev.old] || {};
        C.busy = false;
        if (ev.error) { C.error = ev.error; toast(ev.error, "error"); }
        else if (ev.deleted) { C.deleted = true; toast("The old folder was deleted. Its saves are in Lootnest's backups."); }
        else Object.assign(C, { done: true, saves: ev.saves || [], profile: ev.profile || "", oldThere: !!ev.oldThere, error: "" });
        S.carried[ev.old] = C;
        S.updates = null; api.newCopies().catch(() => {}); api.updates().catch(() => {});
        if (S.view === "updates" || S.view === "game") render();
        break;
      }
      case "updatePlan": {
        const D = S.dialog;
        if (D && D.kind === "update" && D.key === ev.key && D.path === ev.path) {
          if (ev.opening) { D.opening = true; renderOverlay(); break; }
          D.busy = false; D.opening = false; D.plan = ev.plan || null; D.check = ev.check || null; D.error = ev.error || ""; D.dir = ev.dir || ""; D.free = ev.free; renderOverlay();
        }
        break;
      }
      case "updateProgress": {
        const D = S.dialog;
        if (D && D.kind === "update" && D.key === ev.key) { D.step = ev.step; renderOverlay(); }
        break;
      }
      case "updateBroke": {
        toast("The game closed right after starting. If the update broke it, Undo it on the Updates page.", "error");
        S.updates = null; api.updates().catch(() => {});
        break;
      }
      case "updateDone": {
        const D = S.dialog;
        if (ev.kept) { toast(ev.error || "Its saves are copied to Lootnest's backups. You can reinstall it now.", ev.error ? "error" : ""); break; }
        if (D && D.kind === "update" && D.key === ev.key) { D.busy = false; D.error = ev.error || ""; if (!ev.error) { D.done = ev.update; D.undone = !!ev.undone; } renderOverlay(); }
        else if (ev.error) toast(ev.error, "error");
        else toast(ev.undone ? "The update was undone" : "Updated");
        S.updates = null; S.news = {}; api.updates().catch(() => {});
        break;
      }
      case "storeFront": {
        S.front = ev.front || { error: ev.error || "Couldn't reach the store." };
        if (S.view === "find" && !S.find.q.trim()) { const r = $("#find-results"); if (r) r.innerHTML = findResults(); }
        break;
      }
      case "gameOffer": {
        S.offers[ev.appId] = ev;
        if (S.dialog && S.dialog.kind === "offer" && S.dialog.appId === ev.appId) renderOverlay();
        break;
      }
      case "gameSaves": {
        if (ev.info) { S.gsaves[ev.info.key] = ev.info; if (S.view === "game") render(); break; }
        if (ev.error) toast(ev.error, "error");
        else if (ev.unchanged) toast("Nothing new to back up: the saves are the same as the newest version.");
        else if (ev.saved) toast("Saved to the cloud");
        else if (ev.restored) toast(ev.skipped ? `Restored; ${ev.skipped} save location${ev.skipped === 1 ? "" : "s"} this PC doesn't have ${ev.skipped === 1 ? "was" : "were"} left out` : "Restored");
        if (ev.changed) { S.gsaves = {}; if (S.view === "game") render(); }
        if (ev.key) { const G = S.gsaves[ev.key]; if (G) G.busy = false; }
        break;
      }
      case "savesStatus":
        S.saves = ev.status;
        if (S.view === "cloud") render();
        break;
      case "savesChanged":
        if (ev.done && S.syncAsked) {
          S.syncAsked = false;
          if (ev.error) toast(ev.error, "error");
          else toast(ev.up || ev.down ? [ev.up ? `${ev.up} uploaded` : "", ev.down ? `${ev.down} brought down` : ""].filter(Boolean).join(", ") : "Everything is up to date");
        }
        if (S.view === "cloud") loadSaves();
        break;
      case "restoreInfo": {
        const D = S.dialog;
        if (D && D.kind === "restore") { D.busy = false; D.info = ev.info; D.error = ev.info.error || (ev.info.needPassword && D.triedPassword ? "That password doesn't open this backup." : ""); renderOverlay(); setTimeout(() => { const f = $("#rs-pw"); if (f) f.focus(); }, 30); }
        break;
      }
      case "restored": {
        const D = S.dialog;
        if (D && D.kind === "restore") { D.busy = false; D.applying = false; if (ev.ok) { D.done = ev.safety; D.result = ev.import; } else D.error = ev.error; renderOverlay(); }
        refresh();
        break;
      }
      case "exported": {
        const D = S.dialog;
        if (D && D.kind === "export") { D.busy = false; if (ev.error) D.error = ev.error; else D.done = ev.file; renderOverlay(); }
        else if (ev.error) toast(ev.error, "error");
        break;
      }
      case "pickerClosed":
        if (S.picker && S.picker.reqId) { S.picker = null; render(); }
        break;
      case "toast":
        toast(ev.text, ev.kind);
        break;
      case "browse": {
        const w = browseWaiters[ev.reqId];
        if (w) { delete browseWaiters[ev.reqId]; w(ev); }
        break;
      }
    }
  };

  function browse(kind, start) {
    const id = "b" + ++seq;
    return new Promise((resolve) => {
      browseWaiters[id] = resolve;
      api.browse(id, kind, start).catch((e) => { delete browseWaiters[id]; resolve({ error: errText(e) }); });
    });
  }

  async function refresh() {
    try { S.d = await api.state(); } catch (e) { toast(errText(e), "error"); return; }
    render();
  }

  // ---------------------------------------------------------------- render --
  function render() {
    if (!S.d) return;
    document.documentElement.style.setProperty("--p1", (cfg().players[0] || {}).color || "#6C9BD2");
    document.documentElement.style.setProperty("--p2", (cfg().players[1] || {}).color || "#D9738E");
    const top = S.view === "game" ? S.from : S.view === "edit" ? "library" : S.view === "player" ? "players" : S.view;
    document.querySelectorAll(".nav[data-view]").forEach((t) => (t.dataset.view === top ? t.setAttribute("aria-current", "page") : t.removeAttribute("aria-current")));
    const n = libraryItems().length;
    $("#lib-count").textContent = n ? n : "";
    renderWho();
    renderSideUpdate();
    const v = $("#view");
    const keepScroll = v.scrollTop;
    const typing = document.activeElement && document.activeElement.id === "lib-q" ? document.activeElement.selectionStart : -1;
    const findTyping = document.activeElement && document.activeElement.id === "find-q" ? document.activeElement.selectionStart : -1;
    if (S.view === "home") v.innerHTML = viewHome();
    else if (S.view === "library") v.innerHTML = viewLibrary();
    else if (S.view === "game") v.innerHTML = viewGame();
    else if (S.view === "edit") v.innerHTML = viewEdit();
    else if (S.view === "players") v.innerHTML = viewPlayers();
    else if (S.view === "player") v.innerHTML = viewPlayer();
    else if (S.view === "settings") v.innerHTML = viewSettings();
    else if (S.view === "cloud") v.innerHTML = viewCloud();
    else if (S.view === "achievements") v.innerHTML = viewAchievements();
    else if (S.view === "find") v.innerHTML = viewFind();
    else if (S.view === "updates") v.innerHTML = viewUpdates();
    if (S.animateView && v.firstElementChild) v.firstElementChild.classList.add("enter");
    S.animateView = false;
    v.scrollTop = keepScroll;
    if (typing >= 0) { const q = $("#lib-q"); if (q) { q.focus(); q.setSelectionRange(typing, typing); } }
    if (findTyping >= 0) { const q = $("#find-q"); if (q) { q.focus(); q.setSelectionRange(findTyping, findTyping); } }
    renderOverlay();
    loadCovers();
    backdropForView();
  }

  // ------------------------------------------------------------ backdrop --
  // The window takes on the colours of the game in focus: its art, blurred
  // far out of shape, behind everything. It follows the game the pointer
  // rests on in the Library, and the game whose page is open.
  let bdShown = "", bdFront = 0, bdTimer = 0;
  function setBackdrop(url) {
    if (url === bdShown) return;
    bdShown = url;
    const layers = document.querySelectorAll("#backdrop .bd");
    if (layers.length < 2) return;
    const next = layers[bdFront ^ 1], cur = layers[bdFront];
    if (url) next.style.backgroundImage = `url("${url}")`;
    next.classList.toggle("on", !!url);
    cur.classList.remove("on");
    bdFront ^= 1;
  }
  const artOf = (it) => (it && (S.art["hero:" + coverKey(it)] || S.covers[coverKey(it)] || S.icons[coverKey(it)])) || "";
  function itemFor(el) {
    const host = el && el.closest("[data-game], [data-key]");
    if (!host) return null;
    const id = host.dataset.game || host.dataset.key, kind = host.dataset.game ? "game" : "found";
    return libraryItems().find((x) => x.kind === kind && x.id === id) || null;
  }
  // The colours change when a game is chosen, not as the pointer passes:
  // Home takes the game played last, a game's page that game, and the
  // Library the game opened last.
  function defaultBackdrop() {
    const find = (p) => p && libraryItems().find((x) => x.kind === p.kind && x.id === p.id);
    if (S.view === "game" && S.page) return artOf(find(S.page));
    if (S.view === "home") return artOf(recentItems(libraryItems(), false)[0]);
    if (S.view === "library") return artOf(find(S.selected));
    return "";
  }
  function backdropForView() { clearTimeout(bdTimer); setBackdrop(defaultBackdrop()); }

  // ---------------------------------------------------------- right-click --
  function openContext(ev, it) {
    const g = it.g, f = it.f;
    const attr = g ? `data-game="${esc(g.id)}"` : `data-key="${esc(f.key)}"`;
    const s = g && session(g.id);
    const items = [];
    if (!s) {
      if (g) cfg().players.forEach((p) => items.push(`<button data-action="play-as" data-game="${esc(g.id)}" data-player="${esc(p.id)}"><span class="dot" style="background:${esc(p.color)}"></span>Play as ${esc(p.name)}</button>`));
      else items.push(`<button data-action="play-found" ${attr}>${icon.play}Play</button>`);
    }
    items.push(`<button data-action="open-sheet" ${attr}>Open the game's page</button>`, "<hr>");
    if (g) items.push(`<button data-action="edit-game" ${attr} ${s ? "disabled" : ""}>Edit game</button>`);
    else items.push(`<button data-action="game-settings" ${attr}>Game settings</button>`);
    items.push(`<button data-action="open-game-folder" ${attr}>Open the game's folder</button>`);
    if (it.appId) items.push(`<button data-action="steam-verify" data-app="${it.appId}" ${s ? "disabled" : ""}>Check the game's files</button>`);
    items.push("<hr>", g ? `<button class="danger" data-action="remove-game" ${attr}>Turn off profiles</button>`
      : `<button class="danger" data-action="found-hide" ${attr}>${f.added ? "Remove from the Library" : "Hide from the Library"}</button>`);
    const m = $("#ctx");
    m.innerHTML = `<div class="ctx-title">${esc(it.name)}</div>${items.join("")}`;
    m.hidden = false;
    const r = m.getBoundingClientRect();
    m.style.left = Math.min(ev.clientX, innerWidth - r.width - 8) + "px";
    m.style.top = Math.min(ev.clientY, innerHeight - r.height - 8) + "px";
    const first = m.querySelector("button:not(:disabled)");
    if (first) first.focus({ preventScroll: true });
  }
  function closeContext() { const m = $("#ctx"); if (m && !m.hidden) { m.hidden = true; m.innerHTML = ""; } }
  document.addEventListener("contextmenu", (ev) => {
    const host = ev.target.closest(".card, .capsule-wrap");
    if (!host || S.picker) return;
    const it = itemFor(host.querySelector("[data-game], [data-key]"));
    if (!it) return;
    ev.preventDefault();
    openContext(ev, it);
  });
  document.addEventListener("pointerdown", (ev) => { if (!ev.target.closest("#ctx")) closeContext(); }, true);
  window.addEventListener("blur", closeContext);
  document.addEventListener("scroll", closeContext, true);

  // The seat: who's at the PC, until someone switches or Lootnest quits.
  // Every Play is theirs; "or play as…" and the picker are for anyone else.
  const seated = () => (S.seat && player(S.seat)) || null;
  function renderWho() {
    const playing = {}, seat = seated(), ps = cfg().players;
    (S.d.sessions || []).forEach((s) => { if (s.phase === "playing") playing[s.playerId] = s; });
    $("#who-head").innerHTML = `<span>Who's here</span>${ps.length ? `<button class="who-switch" data-action="seat-menu" aria-haspopup="menu">${GI.swap}${seat ? "Switch" : "Sit down"}</button>` : ""}`;
    $("#who").innerHTML = grouped(ps).map((p) => {
      const s = playing[p.id];
      const g = s && game(s.gameId);
      const on = seat && seat.id === p.id;
      return `<button class="who-item ln-player${s ? " playing" : ""}${on ? " seated" : ""}${parentOf(p) ? " who-run" : ""}" style="--player:${esc(p.color)}" data-action="seat" data-player="${esc(p.id)}" aria-pressed="${!!on}"
        title="${on ? `${esc(p.name)} is at the PC. Click to leave the seat empty.` : `Sit ${esc(p.name)} at the PC: Play starts as ${esc(p.name)}.`}">${lnAvatar(p)}<span class="who-name">${esc(p.name)}${on ? ' <span class="ln-tag ln-tag--player">Seated</span>' : ""}</span>
        <span class="what">${g ? `Playing ${esc(g.name)}` : ""}</span></button>`;
    }).join("");
  }
  function seatMenu(el) {
    const seat = seated(), m = $("#ctx");
    m.innerHTML = `<div class="ctx-title">Who's at the PC?</div>${cfg().players.map((p) => `<button data-action="seat-set" data-player="${esc(p.id)}" role="menuitemradio" aria-checked="${!!(seat && seat.id === p.id)}"><span class="dot" style="background:${esc(p.color)}"></span><span class="grow">${esc(p.name)}</span>${seat && seat.id === p.id ? GI.tick : ""}</button>`).join("")}
      <hr><button data-action="seat-set" data-player="" role="menuitemradio" aria-checked="${!seat}">Nobody: ask who's playing each time</button>`;
    m.hidden = false;
    const b = el.getBoundingClientRect(), r = m.getBoundingClientRect();
    m.style.left = Math.min(b.left, innerWidth - r.width - 8) + "px";
    m.style.top = Math.min(b.bottom + 4, innerHeight - r.height - 8) + "px";
    const first = m.querySelector('[aria-checked="true"]') || m.querySelector("button");
    if (first) first.focus({ preventScroll: true });
  }
  function setSeat(pid) {
    S.seat = pid && player(pid) ? pid : "";
    api.setSeat(S.seat).catch(() => {});
    render();
    toast(S.seat ? `${pname(S.seat)} is at the PC. Play starts as ${pname(S.seat)}.` : "Nobody's at the PC. Play asks who's playing.");
  }

  // --------------------------------------------------------------- library --
  function sessionText(s) {
    const n = pname(s.playerId);
    return { loading: `Loading ${possessive(n)} files`, launching: "Starting", playing: `${n} is playing`, saving: `Saving ${possessive(n)} progress` }[s.phase] || "";
  }

  const sourceLabel = (src) => ({ steam: "Steam", epic: "Epic Games", gog: "GOG", xbox: "Xbox", ea: "EA app", ubisoft: "Ubisoft Connect", battlenet: "Battle.net",
    riot: "Riot Games", amazon: "Amazon Games", rockstar: "Rockstar Games", itch: "itch.io", hydra: "Hydra", shortcut: "Shortcut", folder: "Game folder", added: "Added by you" }[src] || "Other");
  // Each store and launcher is its own filter; folders, shortcuts and games
  // added by hand are "Other".
  const LAUNCHERS = ["steam", "epic", "gog", "xbox", "ea", "ubisoft", "battlenet", "riot", "amazon", "rockstar", "itch", "hydra"];
  const filterOf = (src) => (LAUNCHERS.includes(src) ? src : "other");

  // Every game on the PC: the ones with profiles, then everything the scan found.
  function libraryItems() {
    const items = cfg().games.map((g) => ({
      kind: "game", id: g.id, name: g.name, appId: g.launch.type === "steam" ? g.launch.appId : 0,
      source: g.launch.type === "steam" ? "steam" : (S.d.sources || {})[g.id] || "other", exe: g.launch.type !== "steam" ? g.launch.exe || "" : "", g,
    }));
    (S.found.games || []).forEach((f) => items.push({ kind: "found", id: f.key, name: f.name, appId: f.appId || 0, source: f.source, exe: f.appId ? "" : f.exe || "", f }));
    return items.sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: "base" }));
  }

  function shownItems() {
    const q = S.q.trim().toLowerCase();
    const list = libraryItems().filter((it) => {
      if (q && !it.name.toLowerCase().includes(q)) return false;
      if (S.filter === "profiles" && it.kind !== "game") return false;
      if (S.store !== "all" && filterOf(it.source) !== S.store) return false;
      if (S.tag && !tagsOf(it).includes(S.tag)) return false;
      if (S.status) {
        const pid = S.who || (seated() || {}).id;
        if (pid ? statusOf(it, pid) !== S.status : !cfg().players.some((p) => statusOf(it, p.id) === S.status)) return false;
      }
      if (S.who && !playedOrOwned(it, S.who)) return false;
      return true;
    });
    if (S.sort === "recent") list.sort((a, b) => playedAt(b) - playedAt(a));
    else if (S.sort === "played") list.sort((a, b) => Math.max(totalPlayed(b), steamTime(b)) - Math.max(totalPlayed(a), steamTime(a)) || playedAt(b) - playedAt(a));
    return list;
  }
  // Collections (the user's own, for everyone) and each profile's status.
  const STATUSES = [["playing", "Playing"], ["next", "Up next"], ["finished", "Finished"], ["complete", "100%"], ["stopped", "Dropped"]];
  const statusName = (k) => (STATUSES.find((x) => x[0] === k) || [k, k])[1];
  const tagsOf = (it) => (cfg().tags || {})[recentKey(it)] || [];
  const allTags = () => [...new Set(Object.values(cfg().tags || {}).flat())].sort((a, b) => a.localeCompare(b, undefined, { sensitivity: "base" }));
  const shelfOf = (it, pid) => ((S.d.shelf || {})[pid] || {})[recentKey(it)] || {};
  const statusOf = (it, pid) => shelfOf(it, pid).status || "";
  // A player's games: the ones they have a profile in, or played last.
  const playedOrOwned = (it, pid) => (it.kind === "game" && hasProfile(it.id, pid)) || !!lastBy(it, pid);
  // When a player's profile for a game was last saved, and its size, read
  // once per game until the next swap ({} until it arrives).
  function profileInfo(gid, pid) {
    if (!gid) return {};
    if (!(gid in S.pinfo)) { S.pinfo[gid] = null; api.profileInfo(gid).catch(() => {}); }
    const i = (S.pinfo[gid] || {})[pid];
    return i && new Date(i.saved).getFullYear() > 2000 ? i : {};
  }
  // When this player last started the game ("" if they never have).
  function lastBy(it, pid) {
    const r = (S.d.recent || {})[recentKey(it)];
    if (!r || !pid) return "";
    return (r.players || {})[pid] || (r.player === pid ? r.at : "");
  }
  const playedAt = (it) => { const r = (S.d.recent || {})[recentKey(it)]; return r ? new Date(r.at).getTime() : 0; };
  const playtimeOf = (it) => (S.d.playtime || {})[recentKey(it)] || {};
  const totalPlayed = (it) => Object.values(playtimeOf(it)).reduce((a, b) => a + b, 0);
  const steamTime = (it) => (S.d.steamTime || {})[recentKey(it)] || 0;
  // How much a game has been played: Lootnest's own count, or Steam's.
  const playedText = (it) => totalPlayed(it) ? hours(totalPlayed(it)) + " played in all" : steamTime(it) ? hours(steamTime(it)) + " on Steam" : "";
  function hours(sec) {
    if (sec < 3600) return Math.max(1, Math.round(sec / 60)) + " min";
    const h = sec / 3600;
    return (h < 10 ? Math.round(h * 10) / 10 : Math.round(h)) + " h";
  }

  function viewLibrary() {
    const all = libraryItems();
    const withProfiles = cfg().games.length;
    const scanning = S.found.scanning;
    const refreshing = S.found.refreshing;
    const have = new Set(all.map((it) => filterOf(it.source)));
    const stores = [["all", "All stores"], ...LAUNCHERS.filter((k) => have.has(k) || S.store === k).map((k) => [k, sourceLabel(k)]),
      ...(have.has("other") || S.store === "other" ? [["other", "Other"]] : [])];
    const head = `<header class="lib-head"><div><h1>Library</h1>
        <p>${all.length ? `${all.length === 1 ? "1 game" : all.length + " games"} on this PC, ${withProfiles === 1 ? "1 with profiles" : withProfiles ? withProfiles + " with profiles" : "none with profiles yet"}` : "Every game on this PC, in one place."}</p></div>
        <div class="lib-actions"><label class="ln-search">${GI.search}<input type="search" id="lib-q" class="ln-input" placeholder="Search your games" value="${esc(S.q)}" autocomplete="off" spellcheck="false" aria-label="Search your games (Ctrl+F)"></label>
        <button class="ln-button ln-button--secondary" data-action="add-game">${GI.plus}Add game</button>
        <button class="ln-button ln-button--ghost" data-action="rescan" ${refreshing ? 'aria-busy="true"' : ""}>${refreshing ? '<span class="ln-spinner" aria-hidden="true"></span>Scanning' : "Scan again"}</button></div></header>`;
    if (!all.length && !scanning) {
      return `<div class="lib ln-root">${head}<div class="ln-glass ln-state">${homeIcon}<div><h3>No games found yet</h3>
        <p>Lootnest looks in Steam, Epic, GOG, Xbox, EA, Ubisoft, Battle.net, Riot, Amazon, Rockstar, itch.io, game folders and your shortcuts. Add a game by hand if yours isn't found.</p>
        <div class="ln-actions"><button class="ln-button ln-button--primary" data-action="add-game">${GI.plus}Add a game</button><button class="ln-button ln-button--secondary" data-action="rescan">Scan again</button></div></div></div></div>`;
    }
    const sorts = [["name", "Name"], ["recent", "Recently played"], ["played", "Most played"]];
    const sizes = [["s", "Small covers", 10], ["m", "Medium covers", 13], ["l", "Large covers", 16]];
    const seg = (k, l, n) => `<button data-action="filter" data-filter="${k}" aria-pressed="${S.filter === k}">${l}<span class="ln-count ln-num">${n}</span></button>`;
    const ps = cfg().players;
    const who = ps.length > 1 ? `<div class="lib-who" role="group" aria-label="Whose games">${ps.map((p) => `<button class="ln-chip ln-player" style="--player:${esc(p.color)}" data-action="who" data-player="${esc(p.id)}" aria-pressed="${S.who === p.id}" title="${esc(possessive(p.name))} games">${lnAvatar(p, "20")}${esc(p.name)}</button>`).join("")}</div>` : "";
    return `<div class="lib ln-root">${head}${hookBanner() || coverHint(all) ? `<div class="lib-banners">${hookBanner()}${coverHint(all)}</div>` : ""}
      ${(() => { const top = recentShelf(all) + collectionShelves(all); return top ? top + `<div class="ln-section-head lib-all-head"${S.q ? " hidden" : ""}><div><h2>All games</h2></div></div>` : ""; })()}
      <div class="lib-tools">
        <div class="lib-tools-main"><div class="ln-segmented" role="group" aria-label="Show">${seg("all", "All", all.length)}${seg("profiles", "With profiles", withProfiles)}</div>${who}</div>
        <div class="lib-tools-side">
          <select id="lib-store" class="ln-input" aria-label="Store">${stores.map(([k, l]) => `<option value="${k}" ${S.store === k ? "selected" : ""}>${l}</option>`).join("")}</select>
          ${allTags().length || S.tag ? `<select id="lib-tag" class="ln-input" aria-label="Collection"><option value="">All collections</option>${allTags().map((t) => `<option value="${esc(t)}" ${S.tag === t ? "selected" : ""}>${esc(t)}</option>`).join("")}</select>` : ""}
          <select id="lib-status" class="ln-input" aria-label="Status"><option value="">Any status</option>${STATUSES.map(([k, l]) => `<option value="${k}" ${S.status === k ? "selected" : ""}>${l}</option>`).join("")}</select>
          <select id="lib-sort" class="ln-input" aria-label="Sort by">${sorts.map(([k, l]) => `<option value="${k}" ${S.sort === k ? "selected" : ""}>${l === "Name" ? "Sort by name" : l}</option>`).join("")}</select>
          <div class="ln-segmented lib-size" role="group" aria-label="Cover size">${sizes.map(([k, l, h]) => `<button data-action="density" data-size="${k}" aria-pressed="${S.density === k}" aria-label="${l}" title="${l}"><span style="--h:${h}px"></span></button>`).join("")}</div>
        </div></div>
      ${scanning ? lnInline("", "Looking for games on this PC", "Games appear as they're found.") : ""}
      <div class="lib-grid size-${esc(S.density)}" id="grid">${gridHTML()}</div></div>`;
  }
  // Recent games, as at the top of Steam's library: the last few played, on
  // their wide art, one click from Play. Only on the Library as it opens,
  // not while it's filtered; a search hides it.
  function recentShelf(all) {
    if (S.filter !== "all" || S.who || S.store !== "all" || S.tag || S.status) return "";
    const rec = recentItems(all, false).slice(0, 5);
    if (rec.length < 2) return "";
    const cap = (it) => {
      loadArt(it); ensureCover(it);
      const key = coverKey(it), hero = S.art["hero:" + key], cover = S.covers[key];
      const art = hero ? `<img src="${hero}" alt="">` : cover ? `<img class="soft" src="${cover}" alt="">` : "";
      const t = totalPlayed(it) || steamTime(it);
      const s = it.g && session(it.g.id);
      return `<div class="lib-cap${s ? " busy" : ""}${art ? "" : " no-art"}"><button class="lib-cap-open" data-action="open-sheet" ${itemAttr(it)} aria-label="${esc(it.name)}">
          <span class="lib-cap-art">${art || `<span class="lib-cap-none">${esc(it.name)}</span>`}</span>
          <span class="lib-cap-text"><b>${esc(it.name)}</b><span>${playedBy(it)}${t ? `<span class="lib-cap-time ln-num">${esc(hours(t))}</span>` : ""}</span></span></button>
        ${s ? `<span class="lib-cap-badge ln-tag ln-tag--player ln-player" style="--player:${esc(color(s.playerId))}">${esc(sessionText(s))}</span>`
          : `<button class="ln-button ln-button--primary ln-button--icon ln-button--sm lib-cap-play" data-action="${it.kind === "game" ? "play" : "play-found"}" ${itemAttr(it)} aria-label="Play ${esc(it.name)}" title="Play">${GI.play}</button>`}</div>`;
    };
    return `<section class="lib-recent" aria-label="Recent games"${S.q ? " hidden" : ""}><div class="ln-section-head"><div><h2>Recent games</h2></div></div>
      <div class="lib-recent-row">${rec.map(cap).join("")}</div></section>`;
  }
  // A shelf for each collection, under Recent games, as in Steam: its
  // covers in one row that scrolls sideways, and See all to filter the
  // Library to it. Same rules as Recent games for when they show.
  function collectionShelves(all) {
    if (S.filter !== "all" || S.who || S.store !== "all" || S.tag || S.status) return "";
    const tags = cfg().tags || {}, by = {};
    all.forEach((it) => (tags[recentKey(it)] || []).forEach((t) => (by[t] = by[t] || []).push(it)));
    const names = Object.keys(by).sort((a, b) => by[b].length - by[a].length || a.localeCompare(b, undefined, { sensitivity: "base" }));
    if (!names.length) return "";
    // The six largest; the rest are in the Library's collections menu.
    return `<div class="lib-shelves"${S.q ? " hidden" : ""}>${names.slice(0, 6).map((t) => {
      const list = by[t].slice().sort((a, b) => playedAt(b) - playedAt(a) || a.name.localeCompare(b.name));
      return `<section class="lib-shelf" aria-label="${esc(t)}"><div class="ln-section-head"><div><h2>${esc(t)}<span class="lib-shelf-n ln-num">${list.length}</span></h2></div>
          <button class="ln-button ln-button--ghost ln-button--sm" data-action="shelf-all" data-tag="${esc(t)}">See all</button></div>
        <div class="lib-shelf-row">${list.slice(0, 16).map(cardHTML).join("")}</div></section>`;
    }).join("")}</div>`;
  }

  // Home: what are we playing tonight, and who's playing? The game played
  // last with its play bar, the others played lately with who played them,
  // each player's last game, and the news last.
  function viewHome() {
    const all = libraryItems();
    const recent = recentItems(all, false);
    const banners = updateBanner() + hookBanner();
    const allBtn = all.length ? `<button class="ln-button ln-button--ghost ln-button--sm" data-action="nav" data-view="library">All ${all.length === 1 ? "1 game" : all.length + " games"}</button>` : "";
    if (!recent.length) {
      return `<div class="home-empty ln-root">${banners}
        <div class="ln-section-head"><div><h2>Home</h2></div>${allBtn}</div>
        <div class="ln-glass ln-state">${homeIcon}<div><h3>Nothing played yet</h3><p>Start a game from your Library and it shows up here, with who played it, the others you play and their news.</p>
          <div class="ln-actions"><button class="ln-button ln-button--primary" data-action="nav" data-view="library">Open the Library</button></div></div></div></div>`;
    }
    const first = recent[0];
    const head = gameView(first, false);
    return `<div class="gpage home ln-root${head.art ? "" : " no-art"}">${head.top}
      <div class="gpage-body">${head.bar}${head.note}${banners ? `<div class="home-banners">${banners}</div>` : ""}
        <div class="home-cols">
          <div>${continueShelf(recent.slice(1), allBtn)}</div>
          <aside class="gpage-aside">${playersBlock(all)}${saveCheckBlock()}</aside>
        </div>
        ${homeNews(recent)}
      </div></div>`;
  }
  const homeIcon = '<svg class="ln-icon" viewBox="0 0 20 20" aria-hidden="true"><rect x="3.25" y="5.5" width="13.5" height="9" rx="2.5"/><path d="M6.75 8.5v3M5.25 10h3M12.5 9.25v.01M14 10.75v.01"/></svg>';

  // Shelves, as in the best launchers: the games played lately, then what's
  // new in them. Both only show what Lootnest actually knows.
  function recentItems(all, skipFirst) {
    const list = all.filter((it) => playedAt(it) > 0).sort((a, b) => playedAt(b) - playedAt(a));
    return skipFirst ? list.slice(1) : list;
  }
  // Covers for games shown outside the grid (Home), loaded once each.
  const coverAsked = new Set();
  async function ensureCover(it) {
    const key = coverKey(it);
    if (coverAsked.has(key) || key in S.covers) return;
    coverAsked.add(key);
    const can = it.appId || cfg().settings.downloadCovers;
    let url = "";
    if (can) { try { url = (await api.cover(it.appId || 0, it.name, key)) || ""; } catch (e) { /* none */ } }
    if (url) S.covers[key] = url;
    else if (it.exe && !(key in S.icons)) { try { S.icons[key] = (await api.gameIcon(it.exe)) || ""; } catch (e) { S.icons[key] = ""; } }
    if (url || S.icons[key]) render();
  }

  // A game as a cover tile (system.css .ln-tile): who has a profile is the
  // colour code along the bottom, whose files are loaded the tall bar.
  function tileHTML(it, meta) {
    ensureCover(it);
    const k = coverKey(it), c = S.covers[k];
    const g = it.g, owner = g && S.d.active[g.id], s = g && session(g.id);
    const owners = g ? cfg().players.filter((p) => hasProfile(g.id, p.id)) : [];
    const bars = owners.length ? `<div class="ln-tile__owners" aria-hidden="true">${owners.map((p) => `<span class="ln-player" style="--player:${esc(p.color)}"${p.id === owner ? " data-loaded" : ""}></span>`).join("")}</div>` : "";
    const badge = s ? `<span class="ln-tile__badge ln-tag ln-tag--player ln-player" style="--player:${esc(color(s.playerId))}">${esc(sessionText(s))}</span>` : "";
    const cover = c ? `<img src="${c}" alt="">` : `<div class="ln-tile__none"><b>${esc(it.name)}</b><span>${esc(sourceLabel(it.source))}</span></div>`;
    return `<button class="ln-tile" data-action="open-sheet" ${itemAttr(it)}>
      <div class="ln-tile__cover">${cover}${bars}${badge}</div>
      <span class="ln-tile__title">${esc(it.name)}</span><span class="ln-tile__meta">${meta}</span></button>`;
  }
  // "Mihai, yesterday": who played it last, in their colour.
  function playedBy(it) {
    const r = (S.d.recent || {})[recentKey(it)];
    if (!r) return esc(sourceLabel(it.source));
    const p = r.player && player(r.player);
    return p ? `<span class="ln-mark ln-player" style="--player:${esc(p.color)}"></span>${esc(p.name)}, ${esc(ago(r.at))}` : esc(cap(ago(r.at)));
  }
  const cap = (t) => t.charAt(0).toUpperCase() + t.slice(1);

  function continueShelf(list, allBtn) {
    const head = `<div class="ln-section-head"><div><h2>Continue playing</h2></div>${allBtn}</div>`;
    if (!list.length) return `<section>${head}<p class="home-hint">The other games you play show up here.</p></section>`;
    return `<section aria-label="Continue playing">${head}<div class="home-tiles">${list.slice(0, 10).map((it) => tileHTML(it, playedBy(it))).join("")}</div></section>`;
  }

  // Each player and the game they played last: one click to carry on.
  // Your saves: how Lootnest looks after them, from the save check.
  let saveCheckAsked = false;
  function saveCheckBlock() {
    if (!saveCheckAsked) { saveCheckAsked = true; api.saveCheck(false).catch(() => {}); }
    const C = S.saveCheck;
    if (!C) return "";
    const line = (tone, text) => `<li class="sc-line sc-line--${tone}"><span class="run-dot" aria-hidden="true"></span><span>${text}</span></li>`;
    const n = (k, one, many) => (k === 1 ? one : many.replace("#", k));
    const lines = [];
    if (C.profiles) lines.push(line("ok", n(C.profiles, "1 game with profiles: a copy before every swap.", "# games with profiles: a copy before every swap.")));
    if (C.net) lines.push(line("ok", n(C.net, "1 game keeps its saves in its folder, and Lootnest keeps a copy.", "# games keep their saves in their folder, and Lootnest keeps a copy of each.")));
    if (C.points) lines.push(line("ok", n(C.points, "1 game has save points.", "# games have save points.")));
    (C.gone || []).forEach((g) => lines.push(line("low", `${esc(g.name)}'s saves are gone from its folder. <button class="gpage-link" data-action="open-sheet" data-key="${esc(g.key)}">Put a copy back</button>`)));
    if (C.unknownN) lines.push(line("mid", `${n(C.unknownN, "1 game", "# games")} from outside the stores: Lootnest doesn't know where ${C.unknownN === 1 ? "it keeps its" : "they keep their"} saves. ${(C.unknown || []).map((g) => `<button class="gpage-link" data-action="open-sheet" data-key="${esc(g.key)}">${esc(g.name)}</button>`).join(", ")}${C.unknownN > (C.unknown || []).length ? ", and more" : ""}`));
    lines.push(line(C.cloud ? "ok" : "", C.cloud ? "Cloud saves are on." : `Cloud saves are off. <button class="gpage-link" data-action="nav" data-view="cloud">Set them up</button> to have them on every PC.`));
    return `<section class="home-block sc"><div class="ln-section-head"><div><h2>Your saves</h2></div></div><ul class="sc-lines">${lines.join("")}</ul></section>`;
  }
  function playersBlock(all) {
    const ps = cfg().players;
    if (!ps.length) return "";
    const rec = S.d.recent || {};
    const lastOf = {};
    Object.entries(rec).forEach(([k, r]) => {
      const by = Object.assign({}, r.player ? { [r.player]: r.at } : {}, r.players || {});
      Object.entries(by).forEach(([pid, at]) => {
        const t = new Date(at).getTime();
        if (!lastOf[pid] || t > lastOf[pid].t) lastOf[pid] = { t, k, at };
      });
    });
    // Games without profiles don't record who played; then the game they
    // have the most time in stands in.
    const most = {};
    Object.entries(S.d.playtime || {}).forEach(([k, byP]) => Object.entries(byP || {}).forEach(([pid, sec]) => {
      if (!most[pid] || sec > most[pid].sec) most[pid] = { k, sec };
    }));
    return `<section class="gpage-block gp-players"><div class="ln-section-head"><div><h2>Who's playing</h2></div></div><div class="ln-list home-players">${ps.map((p) => {
      const l = lastOf[p.id], m = !l && most[p.id];
      const it = (l && itemByKey(l.k)) || (m && itemByKey(m.k));
      const busy = it && it.g && session(it.g.id);
      const go = !it || busy ? "" : it.g
        ? `<button class="ln-button ln-button--secondary ln-button--sm" data-action="play-as" data-game="${esc(it.g.id)}" data-player="${esc(p.id)}" aria-label="Play ${esc(it.name)} as ${esc(p.name)}">${GI.play}Play</button>`
        : `<button class="ln-button ln-button--secondary ln-button--sm" data-action="play-found" data-key="${esc(it.id)}" aria-label="Play ${esc(it.name)}">${GI.play}Play</button>`;
      return `<div class="home-player">${lnAvatar(p, "32")}<div><b>${esc(p.name)}</b><span>${it ? (l ? `${esc(it.name)}, ${esc(ago(l.at))}` : `Most played: ${esc(it.name)}, ${esc(hours(m.sec))}`) : "Hasn't played yet"}</span></div>${go}</div>`;
    }).join("")}</div></section>`;
  }

  function ensureNews(it) {
    const key = coverKey(it);
    if (S.news[key] || !cfg().settings.gameNews) return;
    S.news[key] = { loading: true };
    api.gameNews(key, it.appId, it.name, storeKept(it.source) ? "" : it.exe).catch(() => {});
  }
  // News comes after the games: the newest post large, the others beside it.
  function homeNews(recent) {
    if (!cfg().settings.gameNews) return "";
    const items = [];
    recent.slice(0, 8).forEach((it) => {
      ensureNews(it);
      const N = S.news[coverKey(it)];
      ((N && N.items) || []).slice(0, 3).forEach((n, i) => items.push({ it, n, i }));
    });
    items.sort((a, b) => new Date(b.n.date) - new Date(a.n.date));
    return newsSection("News from your games", items.slice(0, 5), true);
  }

  // Notices at the top of Home and the Library (system.css .ln-banner).
  const banner = (title, text, actions, busy) => `<div class="ln-banner ln-root">${busy ? '<span class="ln-spinner" aria-hidden="true"></span>' : ""}<div class="ln-banner__text"><b>${title}</b><p>${text}</p></div>${actions ? `<div class="ln-banner__actions">${actions}</div>` : ""}</div>`;
  const bBtn = (kind, attrs, label) => `<button class="ln-button ln-button--${kind}" ${attrs}>${label}</button>`;

  // Asked once: keep Lootnest up to date by itself, or not.
  function updatesPrompt() {
    const s = cfg().settings;
    if (s.updatesAsked || s.checkUpdates || S.update.available) return "";
    return banner("Keep Lootnest up to date?", "Lootnest checks GitHub every few hours, tells you with a notification when a new version is out, and installs it while it's in the tray and no game is running. Nothing about you or your games is sent.",
      bBtn("ghost", 'data-action="updates-no"', "Not now") + bBtn("primary", 'data-action="updates-yes"', "Keep it up to date"));
  }

  // A newer Lootnest shows in the sidebar on every page, not only on Home.
  function renderSideUpdate() {
    const el = $("#side-update"), U = S.update;
    if (!el) return;
    el.innerHTML = U.available
      ? `<button class="side-update-btn" data-action="update-now" ${U.step ? "disabled" : ""}><b>${U.step ? (U.step === "installing" ? "Installing" : "Downloading") : "Update to " + esc(U.version)}</b><span>${U.step ? "Lootnest opens again by itself" : "Your games and profiles stay"}</span></button>`
      : "";
  }

  function updateBanner() {
    const U = S.update;
    if (!U.available) return updatesPrompt();
    if (U.step) {
      return banner(U.step === "installing" ? `Installing Lootnest ${esc(U.version)}` : `Downloading Lootnest ${esc(U.version)}`,
        U.step === "installing" ? "Lootnest closes and opens again in a few seconds." : "Your profiles, games and saves stay as they are.", "", true);
    }
    return banner(`Lootnest ${esc(U.version)} is ready to install`, `You have ${esc(S.d.version)}. Updating keeps your profiles, games and saves.`,
      (U.page ? bBtn("ghost", 'data-action="update-notes"', "What's new") : "") + bBtn("primary", 'data-action="update-now"', "Update now"));
  }

  function coverHint(all) {
    if (cfg().settings.downloadCovers || S.hideCoverHint) return "";
    const n = all.filter((it) => !it.appId).length;
    if (!n) return "";
    return banner(`Find covers and artwork for ${n === 1 ? "1 game" : n + " games"}`,
      `Lootnest looks ${n === 1 ? "it" : "them"} up on the Steam store by name, and on SteamGridDB if you add a key in Settings. Only the games' names are sent. Everything found is saved on this PC.`,
      bBtn("ghost", 'data-action="covers-later"', "Not now") + bBtn("primary", 'data-action="covers-on"', "Find covers"));
  }

  function gridHTML() {
    const items = shownItems();
    if (!items.length) return `<p class="lib-none">${S.q ? `No game matches “${esc(S.q)}”.` : S.who ? `${esc(pname(S.who))} has no games here yet.` : "No games here."}</p>`;
    return items.map(cardHTML).join("");
  }

  // Steam games have a cover in Steam's cache; other games only when the
  // user turned on "Download covers".
  const coverKey = (it) => (it.appId ? "a" + it.appId : "n:" + it.name.toLowerCase());
  function coverHTML(it) {
    const key = coverKey(it);
    const c = S.covers[key];
    const ic = !c && S.icons[key];
    const can = it.appId || cfg().settings.downloadCovers || it.exe;
    const show = can && (c !== "" || ic !== "" && it.exe);
    const src = c ? ` src="${c}" class="ready"` : ic ? ` src="${ic}" class="icon ready"` : "";
    const img = show ? `<img alt="" data-cover="${esc(key)}" data-app="${it.appId || 0}" data-name="${esc(it.name)}" data-exe="${esc(it.exe || "")}"${src}>` : "";
    return `<div class="nocover${ic ? " with-icon" : ""}"${c ? " hidden" : ""}${ic ? ` style="--icon:url('${ic}')"` : ""}><span>${esc(it.name)}</span></div>${img}`;
  }

  // A Library cover (system.css .ln-tile). Who has a profile is the colour
  // code along the bottom; the tall bar, and the line under the title, say
  // whose files are loaded.
  function cardHTML(it) {
    const g = it.g;
    const owner = g && S.d.active[g.id];
    const s = g && session(g.id);
    const owners = g ? cfg().players.filter((p) => hasProfile(g.id, p.id)) : [];
    const bars = owners.length ? `<div class="ln-tile__owners" aria-hidden="true">${owners.map((p) => `<span class="ln-player" style="--player:${esc(p.color)}"${p.id === owner ? " data-loaded" : ""}></span>`).join("")}</div>` : "";
    const badge = s ? `<span class="ln-tile__badge ln-tag ln-tag--player ln-player" style="--player:${esc(color(s.playerId))}">${esc(sessionText(s))}</span>` : "";
    const meta = owner && player(owner) ? `<span class="ln-mark ln-player" style="--player:${esc(color(owner))}"></span>${esc(possessive(pname(owner)))} files`
      : g ? "Profiles on" : esc(sourceLabel(it.source));
    const attr = itemAttr(it);
    return `<div class="card lib-card${s ? " busy" : ""}">
      <button class="ln-tile" data-action="open-sheet" ${attr} aria-label="${esc(it.name)}${owner && player(owner) ? `, ${esc(possessive(pname(owner)))} files loaded` : ""}"><div class="ln-tile__cover cover">${coverHTML(it)}${bars}${badge}</div>
        <span class="ln-tile__title">${esc(it.name)}</span><span class="ln-tile__meta">${meta}</span></button>
      ${s ? "" : `<button class="ln-button ln-button--secondary ln-button--icon ln-button--sm lib-play" data-action="${it.kind === "game" ? "play" : "play-found"}" ${attr} aria-label="Play ${esc(it.name)}" title="Play">${GI.play}</button>`}</div>`;
  }

  // Covers load lazily as cards scroll into view; Steam keeps them on this PC.
  let coverObserver = null;
  function loadCovers() {
    if (!coverObserver) {
      coverObserver = new IntersectionObserver((entries) => {
        entries.forEach((e) => { if (e.isIntersecting) { coverObserver.unobserve(e.target); fetchCover(e.target); } });
      }, { root: $("#view"), rootMargin: "300px" });
    }
    document.querySelectorAll("img[data-cover]:not(.ready)").forEach((img) => coverObserver.observe(img));
  }

  const pendingCovers = new Set(); // downloads Go is doing; a "cover" event ends them
  async function fetchCover(img) {
    const key = img.dataset.cover;
    const can = +img.dataset.app || cfg().settings.downloadCovers;
    if (!(key in S.covers) && !pendingCovers.has(key)) {
      let url = "";
      if (can) { try { url = (await api.cover(+img.dataset.app, img.dataset.name, key)) || ""; } catch (e) { /* no cover */ } }
      if (url) S.covers[key] = url;
      else if (can && cfg().settings.downloadCovers) pendingCovers.add(key); // may still arrive
      else S.covers[key] = "";
    }
    // No cover (yet): the game's own icon, read from its exe on this PC.
    if (!S.covers[key] && img.dataset.exe && !(key in S.icons)) {
      try { S.icons[key] = (await api.gameIcon(img.dataset.exe)) || ""; } catch (e) { S.icons[key] = ""; }
    }
    showCover(key);
  }

  function showCover(key) {
    const url = S.covers[key], ic = S.icons[key];
    if (!url && !ic && !(key in S.covers)) return; // still waiting for a download
    document.querySelectorAll(`img[data-cover="${CSS.escape(key)}"]`).forEach((el) => {
      const tile = el.previousElementSibling;
      if (url) {
        el.onload = () => { el.classList.remove("icon"); el.classList.add("ready"); if (tile) { tile.hidden = true; tile.classList.remove("with-icon"); } };
        el.src = url;
      } else if (ic) {
        el.classList.add("icon");
        if (tile) { tile.classList.add("with-icon"); tile.style.setProperty("--icon", `url("${ic}")`); }
        el.onload = () => el.classList.add("ready");
        el.src = ic;
      } else el.remove();
    });
    if (!bdShown) backdropForView();
  }

  // ------------------------------------------------------------ game page --
  // A game opens as its own page, with a short cross-fade (--dur-2).
  const reduced = () => window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  function transition(update, before, after) {
    if (!document.startViewTransition || reduced()) { update(); if (after) after(); return; }
    if (before) before();
    const t = document.startViewTransition(update);
    t.finished.finally(() => { if (after) after(); });
  }

  function openSheet(kind, id, fromEl) {
    if (S.view === "edit" && !confirmLeaveEdit()) return;
    if (S.dialog && S.dialog.kind !== "welcome") S.dialog = null;
    const P = { kind, id, cloud: null };
    const go = () => {
      if (S.view === "library") S.libScroll = $("#view").scrollTop;
      if (S.view === "library" || S.view === "home") { S.from = S.view; S.fromScroll = $("#view").scrollTop; }
      S.page = P; S.selected = { kind, id };
      S.view = "game"; S.menuFor = null;
      render(); $("#view").scrollTop = 0;
      $("#view").focus({ preventScroll: true });
    };
    transition(go);
    const g = kind === "game" && game(id);
    if (g && g.launch.type === "steam") {
      api.cloudStatus(id).then((st) => { P.cloud = st; if (S.page === P && S.view === "game") render(); }).catch(() => {});
    }
  }

  function backToLibrary() {
    const P = S.page;
    transition(() => {
      S.view = S.from === "home" ? "home" : "library"; S.page = null;
      render(); $("#view").scrollTop = S.fromScroll || 0;
    });
  }

  // When something was last played, in words.
  function ago(t) {
    const m = Math.round((Date.now() - new Date(t).getTime()) / 60000);
    if (!(m >= 0)) return "";
    if (m < 2) return "just now";
    if (m < 60) return m + " minutes ago";
    const h = Math.round(m / 60);
    if (h < 24) return h === 1 ? "an hour ago" : h + " hours ago";
    const d = Math.round(h / 24);
    if (d < 2) return "yesterday";
    if (d < 30) return d + " days ago";
    return new Date(t).toLocaleDateString(undefined, { day: "numeric", month: "long" });
  }
  const recentKey = (it) => (it.kind === "game" ? "game:" + it.id : it.id);
  function lastPlayed(it) {
    const r = (S.d.recent || {})[recentKey(it)];
    if (!r) return "";
    const when = ago(r.at);
    return r.player && player(r.player) ? `${pname(r.player)} played ${when}` : `Played ${when}`;
  }

  // Wide art and logos: from Steam's cache on this PC, or downloaded when
  // "Download covers and artwork" is on.
  function artHTML(it) {
    const hero = S.art["hero:" + coverKey(it)];
    const c = S.covers[coverKey(it)] || S.icons[coverKey(it)] || "";
    if (hero) return `<div class="art" style="background-image:url('${hero}')"></div>`;
    if (c) return `<div class="art soft" style="background-image:url('${c}')"></div>`;
    return `<div class="art none"></div>`;
  }

  const artWanted = new Set();
  async function loadArt(it) {
    const key = coverKey(it);
    if (!it || artWanted.has(key)) return;
    artWanted.add(key);
    try {
      const [h, l] = await Promise.all([api.art(it.appId, it.name, key, "hero"), api.art(it.appId, it.name, key, "logo")]);
      if (h) S.art["hero:" + key] = h;
      if (l) S.art["logo:" + key] = l;
      if (h || l) render();
    } catch (e) { /* no art */ }
  }



  const day = (d) => new Date(d).toLocaleDateString(undefined, { day: "numeric", month: "short", year: new Date(d).getFullYear() === new Date().getFullYear() ? undefined : "numeric" });

  // An update or announcement, read inside Lootnest. Its body is HTML that
  // Lootnest made safe in Go (core.NewsHTML): only known elements, images
  // from Steam's servers, links that open in the browser.
  function viewReader() {
    const D = S.dialog;
    const n = ((S.news[D.key] || {}).items || [])[D.i];
    if (!n) return "";
    const it = libraryItems().find((x) => coverKey(x) === D.key);
    return `<div class="scrim" data-action="scrim"><article class="dialog reader" role="dialog" aria-modal="true" aria-label="${esc(n.title)}">
      <header class="reader-head"><div>
        <p class="reader-meta">${n.update ? '<span class="dot"></span>Update' : "News"}<span>${esc(new Date(n.date).toLocaleDateString(undefined, { day: "numeric", month: "long", year: "numeric" }))}</span>${it ? `<span>${esc(it.name)}</span>` : ""}</p>
        <h2>${esc(n.title)}</h2></div>
        <button class="btn quiet icon" data-action="dialog-cancel" aria-label="Close">${icon.x}</button></header>
      <div class="reader-body selectable">${n.body || `<p>${esc(n.summary)}</p>`}</div>
      <footer class="dialog-foot"><button class="btn quiet" data-action="open-news" data-url="${esc(n.url)}">Open on Steam</button>
        <button class="btn" data-action="dialog-cancel">Close</button></footer>
    </article></div>`;
  }

  // Updates and news from Steam's feed for the game (opt-in).
  const storeKept = (src) => src === "steam" || src === "epic" || src === "gog";
  // Steam's summaries keep BBCode escapes ("\[ GAMEPLAY ]"); show them plain.
  const plainSummary = (t) => String(t || "").replace(/\\([[\]])/g, "$1");
  // The news as a section: the newest post large, with its picture, and the
  // ones before it as a list beside it. items: [{ it, n, i }].
  function newsSection(title, items, showGame) {
    if (!items.length) return "";
    const pic = (it, n) => n.image || S.art["hero:" + coverKey(it)] || S.covers[coverKey(it)] || "";
    const meta = (it, n) => `<span class="news-meta"><time class="ln-num">${esc(day(n.date))}</time>${n.update ? '<span class="ln-tag ln-tag--accent">Update</span>' : ""}${showGame ? `<span>${esc(it.name)}</span>` : ""}</span>`;
    const [f, ...rest] = items;
    const fp = pic(f.it, f.n);
    const featured = `<button class="news-feature" data-action="read-news" data-key="${esc(coverKey(f.it))}" data-i="${f.i}">
      <span class="news-pic"${fp ? ` style="background-image:url('${esc(fp)}')"` : ""}></span>
      <span class="news-body">${meta(f.it, f.n)}<b>${esc(f.n.title)}</b>${f.n.summary ? `<span class="news-sum">${esc(plainSummary(f.n.summary))}</span>` : ""}
        <span class="news-read">Read</span></span></button>`;
    const list = rest.length ? `<ul class="news-list">${rest.map(({ it, n, i }) => { const p = pic(it, n); return `<li><button data-action="read-news" data-key="${esc(coverKey(it))}" data-i="${i}">
      <span class="news-thumb"${p ? ` style="background-image:url('${esc(p)}')"` : ""}></span>
      <span class="news-body">${meta(it, n)}<b>${esc(n.title)}</b>${n.summary ? `<span class="news-sum">${esc(plainSummary(n.summary))}</span>` : ""}</span></button></li>`; }).join("")}</ul>` : "";
    return `<section class="news" aria-label="${esc(title)}"><div class="ln-section-head"><div><h2>${esc(title)}</h2></div></div><div class="news-grid${rest.length ? "" : " single"}">${featured}${list}</div></section>`;
  }
  function newsHTML(it) {
    const title = "Updates and news";
    const head = `<div class="ln-section-head"><div><h2>${title}</h2></div></div>`;
    if (!cfg().settings.gameNews) {
      return `<section class="news">${head}${banner("See this game's updates and patch notes", `From Steam's news feed${storeKept(it.source) ? "" : ", and find out when a newer version than yours is out"}. Only the game's Steam id, or its name, is sent.`, bBtn("secondary", 'data-action="news-on"', "Show updates and news"))}</section>`;
    }
    const key = coverKey(it);
    const N = S.news[key];
    if (!N) {
      S.news[key] = { loading: true };
      api.gameNews(key, it.appId, it.name, storeKept(it.source) ? "" : it.exe).catch(() => {});
    }
    if (!N || N.loading) return `<section class="news">${head}${lnInline("", `Reading Steam's news for ${esc(it.name)}`)}</section>`;
    if (N.none) return `<section class="news">${head}<p class="news-none">${esc(it.name)} isn't on the Steam store, so there's no news feed for it.</p></section>`;
    if (N.error) return `<section class="news">${head}${lnInline("error", esc(N.error))}</section>`;
    const items = (N.items || []).slice(0, 5).map((n, i) => ({ it, n, i }));
    const uk = updKey(it);
    const offer = it.kind === "found" && (S.copies || []).find((o) => o.new.key === it.id);
    const newer = N.newer ? banner("A newer version is out", `${esc(N.newer.title)}, released ${esc(day(N.newer.date))}. The files of your copy are older than that.`,
      uk ? `<button class="ln-button ln-button--secondary ln-button--sm" data-action="upd-start" data-key="${esc(uk)}" data-name="${esc(it.name)}">Apply an update</button>` : "") : "";
    const copyBanner = offer ? `<div class="news-newer">${copyCard(offer, true)}</div>` : "";
    if (!items.length) return `${copyBanner}<section class="news">${head}${newer}<p class="news-none">Steam has no news for this game yet.</p></section>`;
    return copyBanner + (newer ? `<div class="news-newer">${newer}</div>` : "") + newsSection(title, items, false);
  }

  // ------------------------------------------------------------ game page --
  // Built on system.css (DESIGN.md): the game's art under a glass play bar.
  // With profiles on, the player whose files are loaded is the one Play and
  // everyone else is a chip; the actions below are grouped by intent.
  const gsvg = (d, fill) => `<svg class="ln-icon" viewBox="0 0 20 20" aria-hidden="true"${fill ? ' style="fill:currentColor;stroke:none"' : ""}>${d}</svg>`;
  const GI = {
    back: gsvg('<path d="M12 4.75 6.75 10 12 15.25"/>'),
    swap: gsvg('<path d="M4 7h11.5M12.75 4.25 15.5 7l-2.75 2.75M16 13H4.5M7.25 10.25 4.5 13l2.75 2.75"/>'),
    tick: gsvg('<path d="M5 10.5l3.25 3.25L15 7"/>'),
    trophy: gsvg('<path d="M6.5 3.75h7v4.5a3.5 3.5 0 0 1-7 0z"/><path d="M6.5 5.25H4.25v1a2.5 2.5 0 0 0 2.4 2.5M13.5 5.25h2.25v1a2.5 2.5 0 0 1-2.4 2.5M10 11.75v2.5M7 16.25h6"/>'),
    play: gsvg('<path d="M6.5 4.4v11.2a.9.9 0 0 0 1.37.77l9.05-5.6a.9.9 0 0 0 0-1.54L7.87 3.63A.9.9 0 0 0 6.5 4.4z"/>', true),
    load: gsvg('<path d="M10 3.5v8.25M6.75 8.5 10 11.75l3.25-3.25"/><path d="M3.75 13v2.25c0 .7.55 1.25 1.25 1.25h10c.7 0 1.25-.55 1.25-1.25V13"/>'),
    ask: gsvg('<path d="M5.25 3.75h9.5c.83 0 1.5.67 1.5 1.5v6.5c0 .83-.67 1.5-1.5 1.5H9.5l-3.75 3v-3h-.5c-.83 0-1.5-.67-1.5-1.5v-6.5c0-.83.67-1.5 1.5-1.5z"/><path d="M8.5 7.1a1.6 1.6 0 1 1 2.25 1.46c-.45.2-.75.6-.75 1.1v.1M10 11.6v.01"/>'),
    edit: gsvg('<path d="M12.75 4.25a1.77 1.77 0 0 1 2.5 2.5L7.5 14.5l-3.25.75.75-3.25z"/><path d="M11.5 5.5l3 3"/>'),
    folder: gsvg('<path d="M3.5 6c0-.83.67-1.5 1.5-1.5h2.9l1.6 1.75H15c.83 0 1.5.67 1.5 1.5v6.75c0 .83-.67 1.5-1.5 1.5H5c-.83 0-1.5-.67-1.5-1.5z"/>'),
    save: gsvg('<path d="M4.75 3.75h8.4l2.6 2.6v8.9c0 .55-.45 1-1 1H4.75c-.55 0-1-.45-1-1V4.75c0-.55.45-1 1-1z"/><path d="M7 3.75v3.5h5.25v-3.5M6.75 16.25v-4.5h6.5v4.5"/>'),
    history: gsvg('<path d="M4 10a6 6 0 1 0 1.76-4.24"/><path d="M4 3.75V6.5h2.75"/><path d="M10 7v3.25l2.25 1.5"/>'),
    check: gsvg('<path d="M10 3.25 15.75 5.4v4.35c0 3.35-2.4 5.85-5.75 7-3.35-1.15-5.75-3.65-5.75-7V5.4z"/><path d="M7.4 10.1l1.85 1.85 3.4-3.6"/>'),
    drive: gsvg('<rect x="3.25" y="10.5" width="13.5" height="5.75" rx="1.5"/><path d="M4.75 10.5 6.2 5.3a1.5 1.5 0 0 1 1.45-1.05h4.7A1.5 1.5 0 0 1 13.8 5.3l1.45 5.2M13.25 13.4v.01M10.75 13.4v.01"/>'),
    desktop: gsvg('<rect x="3.25" y="3.75" width="13.5" height="9.5" rx="1.5"/><path d="M7.25 16.5h5.5M10 13.25v3.25"/>'),
    power: gsvg('<path d="M10 3.25v6.25"/><path d="M6.4 5.75a5.75 5.75 0 1 0 7.2 0"/>'),
    hide: gsvg('<path d="M3.5 10s2.4-4.75 6.5-4.75S16.5 10 16.5 10s-2.4 4.75-6.5 4.75S3.5 10 3.5 10z"/><path d="M4 4l12 12"/>'),
    copy: gsvg('<rect x="7.25" y="7.25" width="9" height="9" rx="1.75"/><path d="M12.75 7.25V5.5c0-.97-.78-1.75-1.75-1.75H5.5c-.97 0-1.75.78-1.75 1.75V11c0 .97.78 1.75 1.75 1.75h1.75"/>'),
    sliders: gsvg('<path d="M3.5 6.25h7.25M15.25 6.25h1.25M3.5 13.75h1.25M9.25 13.75h7.25"/><circle cx="13" cy="6.25" r="2.25"/><circle cx="7" cy="13.75" r="2.25"/>'),
    profiles: gsvg('<circle cx="7.75" cy="7" r="2.75"/><path d="M2.75 16.25c.45-2.65 2.45-4.25 5-4.25s4.55 1.6 5 4.25M13 4.6a2.5 2.5 0 0 1 0 4.8M14.5 12.25c1.5.4 2.5 1.65 2.75 3.5"/>'),
    search: gsvg('<circle cx="8.75" cy="8.75" r="4.75"/><path d="M12.25 12.25 16 16"/>'),
    x: gsvg('<path d="M5.5 5.5l9 9M14.5 5.5l-9 9"/>'),
    plus: gsvg('<path d="M10 4.5v11M4.5 10h11"/>'),
    globe: gsvg('<circle cx="10" cy="10" r="6.75"/><path d="M3.25 10h13.5M10 3.25c1.9 1.85 2.85 4.1 2.85 6.75S11.9 14.9 10 16.75C8.1 14.9 7.15 12.65 7.15 10S8.1 5.1 10 3.25z"/>'),
    cloud: gsvg('<path d="M6.25 15.25h7.5a3.25 3.25 0 0 0 .5-6.46 4.75 4.75 0 0 0-9.1 1.3A2.6 2.6 0 0 0 6.25 15.25z"/>'),
    info: gsvg('<circle cx="10" cy="10" r="6.75"/><path d="M10 9v4.5M10 6.6v.01"/>'),
    alert: gsvg('<path d="M10 3.5 17 16H3z"/><path d="M10 8.25v3.5M10 13.9v.01"/>'),
  };
  // A player's avatar in the system's style: their picture, a game's cover,
  // or their initial on their colour.
  const lnAvatar = (p, size) => {
    // A run wears its person's picture and colour, with a ring.
    const par = parentOf(p), who = par || p;
    const u = (S.d.avatars || {})[who.id];
    const g = !u && who.avatarGame && itemByKey(who.avatarGame);
    if (g) ensureCover(g);
    const gu = g && (S.covers[coverKey(g)] || S.icons[coverKey(g)]);
    const inner = u ? `<img src="${u}" alt="">` : gu ? `<img src="${gu}" alt="">` : esc((who.name || "?").trim().charAt(0).toUpperCase());
    return `<span class="ln-avatar ln-player${size ? " ln-avatar--" + size : ""}${par ? " is-run" : ""}" style="--player:${esc(who.color || p.color)}" aria-hidden="true">${inner}</span>`;
  };
  const gpHead = (t) => `<div class="ln-section-head ln-section-head--minor"><h2>${t}</h2></div>`;
  const gpRow = (attrs, ic, title, sub, opt = {}) => `<button class="ln-row${opt.danger ? " ln-row--danger" : ""}" ${attrs}${opt.disabled ? " disabled" : ""}>${ic}<span><b>${title}</b>${sub ? `<span>${sub}</span>` : ""}</span></button>`;
  const gpGroup = (title, rows) => (rows.filter(Boolean).length ? `<section class="gpage-group" data-g="${title}">${gpHead(title)}<div class="ln-list">${rows.filter(Boolean).join("")}</div></section>` : "");
  const lnInline = (kind, title, text) => `<div class="ln-state ln-state--inline${kind === "error" ? " ln-state--error" : ""}">${kind === "error" ? GI.alert : '<span class="ln-spinner" aria-hidden="true"></span>'}<div><h3>${title}</h3>${text ? `<p>${text}</p>` : ""}</div></div>`;
  const gpStat = (value, label, p) => `<div class="ln-meta__item"><dd class="ln-num">${value}</dd><dt>${p ? `<span class="ln-mark ln-player" style="--player:${esc(p.color)}"></span>` : ""}${label}</dt></div>`;
  const hoursValue = (sec) => { const t = hours(sec), m = t.match(/^([\d.]+) (h|min)$/); return m ? `${m[1]}<small>${m[2]}</small>` : esc(t); };

  function viewGame() {
    const P = S.page;
    const it = P && libraryItems().find((x) => x.kind === P.kind && x.id === P.id);
    if (!it) { S.view = "library"; return viewLibrary(); }
    return gameView(it, true);
  }
  // The game page (page = true), or just its art and play bar, as on Home.
  function gameView(it, page) {
    const P = page ? S.page : {};
    loadArt(it);
    const key = coverKey(it);
    const hero = S.art["hero:" + key], cover = S.covers[key] || "";
    const art = hero ? `<img class="gpage-art" src="${hero}" alt="">` : cover ? `<img class="gpage-art soft" src="${cover}" alt="">` : "";
    const rec = (S.d.recent || {})[recentKey(it)];
    const meta = [esc(sourceLabel(it.source))];
    if (isOffline(it)) meta.push(`${GI.globe}Kept offline`);
    const st = steamTime(it);
    let bar, note = "", side, news;
    const details = page ? "" : `<button class="ln-button ln-button--ghost" data-action="open-sheet" ${itemAttr(it)}>Details</button>`;
    // Everything else you can do with the game opens in a panel at the side.
    const manage = page ? `<button class="ln-button ln-button--secondary ln-button--sm gpage-manage-btn" data-action="manage" ${itemAttr(it)} aria-haspopup="dialog">${GI.sliders}Manage</button>` : "";

    if (it.kind === "found") {
      const f = it.f;
      meta.push("Profiles off");
      if (rec) meta.push(esc(lastPlayed(it)));
      const stats = [st ? gpStat(hoursValue(st), "Steam account") : ""].join("");
      bar = `<section class="gpage-bar" aria-label="Play">
        <button class="ln-button ln-button--primary ln-button--lg" data-action="play-found" data-key="${esc(f.key)}">${GI.play}Play</button>
        <div class="gpage-state"><b>Profiles off</b><span>Everyone shares the same settings and saves</span></div>
        <div class="gpage-others"><button class="ln-button ln-button--secondary" data-action="found-add" data-key="${esc(f.key)}" ${S.adding ? "disabled" : ""}>${GI.profiles}Turn on profiles</button>${details}</div>
        ${stats ? `<dl class="ln-meta gpage-stats">${stats}</dl>` : ""}</section>`;
      note = `<p class="gpage-note">Turn on profiles so everyone who plays gets their own settings and saves. ${f.ready ? "Lootnest already found where this game keeps them." : "You'll choose where this game keeps them."}</p>`;
      side = page ? playersPanel(it) + collectionsBlock(it) + gameSavesBlock(it) : "";
      news = page ? newsHTML(it) : "";
    } else {
      const g = it.g, s = session(g.id), owner = S.d.active[g.id];
      const ps = cfg().players;
      const withProfile = ps.filter((p) => hasProfile(g.id, p.id));
      meta.push(withProfile.length ? `Profiles on for ${esc(withProfile.map((p) => p.name).join(", ").replace(/, ([^,]*)$/, " and $1"))}` : "Profiles on");
      if (rec) meta.push(esc(lastPlayed(it)));
      // The one Play: whoever's files are loaded, else whoever played last.
      const lead = seated() || player(owner) || (rec && player(rec.player)) || withProfile[0] || ps[0];
      const others = ps.filter((p) => p !== lead);
      const pt = playtimeOf(it);
      let play, state;
      if (s) {
        play = `<span class="gpage-session ln-player" style="--player:${esc(color(s.playerId))}" role="status">${s.phase === "playing" ? '<span class="ln-mark ln-player" style="--player:' + esc(color(s.playerId)) + '"></span>' : '<span class="ln-spinner" aria-hidden="true"></span>'}${esc(sessionText(s))}</span>`;
        state = `<div class="gpage-state"><b>${esc(possessive(pname(s.playerId)))} files are loaded</b><span>Their progress goes back to their profile when the game closes</span></div>`
          + (s.phase === "playing" ? `<div class="gpage-others"><button class="ln-button ln-button--secondary" data-action="close-game" data-game="${esc(g.id)}">${GI.x}Close the game</button></div>` : "");
      } else {
        play = lead ? `<button class="ln-button ln-button--play ln-button--lg ln-player" style="--player:${esc(lead.color)}" data-action="play-as" data-game="${esc(g.id)}" data-player="${esc(lead.id)}">${GI.play}Play as ${esc(lead.name)}</button>` : "";
        const sub = owner && lastBy(it, owner) ? `Played ${esc(ago(lastBy(it, owner)))}` : owner && pt[owner] ? `${esc(hours(pt[owner]))} played` : owner ? "Loaded without playing" : "Playing loads the profile's files first";
        state = `<div class="gpage-state"><b>${owner && player(owner) ? `${esc(possessive(pname(owner)))} files are loaded` : "Nobody's files yet"}</b><span>${sub}</span></div>`;
      }
      const chips = !s && others.length ? `<div class="gpage-others"><span class="gpage-others-label">or play as…</span>${others.map((p) => `<button class="ln-chip ln-player" style="--player:${esc(p.color)}" data-action="play-as" data-game="${esc(g.id)}" data-player="${esc(p.id)}" aria-label="Play as ${esc(p.name)}">${lnAvatar(p)}${esc(p.name)}</button>`).join("")}</div>` : "";
      const top = ps.filter((p) => pt[p.id]).sort((a, b) => pt[b.id] - pt[a.id]).slice(0, 3);
      const stats = top.map((p) => gpStat(hoursValue(pt[p.id]), esc(p.name), p)).join("") + (st ? gpStat(hoursValue(st), "Steam account") : "");
      bar = `<section class="gpage-bar" aria-label="Play">${play}${state}${chips}${details}${stats ? `<dl class="ln-meta gpage-stats">${stats}</dl>` : ""}</section>`;
      note = `<p class="gpage-note">${GI.copy}A copy is made before every swap.</p>`;
      const cloud = P.cloud && {
        none: "Doesn't sync the files profiles swap.",
        elsewhere: "Syncs other files of this game, not the ones profiles swap.",
        overlap: `Syncs ${P.cloud.count === 1 ? "1 of the files" : P.cloud.count + " of the files"} profiles swap. Lootnest checks them each time the game starts, so a cloud copy never ends up in someone's profile.`,
      }[P.cloud.state];
      side = page ? playersPanel(it) + collectionsBlock(it) + (cloud ? `<section class="gpage-block">${gpHead("Steam Cloud")}<p>${esc(cloud)}</p></section>` : "") : "";
      news = page ? newsHTML(it) : "";
    }
    const back = page ? `<button class="ln-button ln-button--ghost ln-button--sm gpage-back" data-action="back-library">${GI.back}${S.from === "home" ? "Home" : "Library"}</button>` : "";
    const top = `<section class="gpage-hero">${art}${back}${manage}
        <div class="gpage-title"><h1 class="${it.name.length > 22 ? "long" : ""}">${esc(it.name)}</h1>
          <p class="gpage-meta">${meta.map((m) => `<span>${m}</span>`).join("")}</p></div>
      </section>`;
    if (!page) return { top, bar, note, art };
    return `<div class="gpage ln-root${art ? "" : " no-art"}">${top}
      <div class="gpage-body">${bar}${note}
        <div class="gpage-main"><div class="gpage-col">${achievementsBlock(it)}${page ? savePointsBlock(it) : ""}<div class="gpage-news">${news}</div></div><aside class="gpage-side">${side}</aside></div>
      </div></div>`;
  }
  // The Manage panel: what you can do with a game besides playing it,
  // grouped by intent. Each row closes the panel and does its one thing.
  // Keep it offline: Windows Firewall rules for games Lootnest starts itself.
  const isOffline = (it) => !!(cfg().offline || {})[recentKey(it)];
  const offlineWorth = (it) => (it.kind === "found" ? !it.appId && !(it.f && it.f.uri) && !storeKept(it.source) : it.g.launch.type === "exe");
  function offlineRow(it) {
    if (!offlineWorth(it)) return "";
    const n = ((cfg().offline || {})[recentKey(it)] || []).length;
    return isOffline(it) ? gpRow(`data-action="offline-toggle" ${itemAttr(it)} data-on="0"`, GI.globe, "Let it go online again", `Windows Firewall blocks ${n === 1 ? "its program" : n + " of its programs"} now`)
      : gpRow(`data-action="offline-toggle" ${itemAttr(it)} data-on="1"`, GI.globe, "Keep it offline", "Windows Firewall stops it connecting to the internet");
  }
  function manageGroups(it) {
    if (it.kind === "found") {
      const f = it.f;
      return gpGroup("Game", [gpRow(`data-action="game-settings" data-key="${esc(f.key)}"`, GI.sliders, "Game settings", f.args ? `Name and launch options: <code>${esc(f.args)}</code>` : "Name and launch options")])
        + gpGroup("Game files", [
          gpRow(`data-action="open-game-folder" data-key="${esc(f.key)}"`, GI.folder, "Open the game's folder", "Where it's installed"),
          f.source === "steam" && f.appId ? gpRow(`data-action="steam-verify" data-app="${f.appId}"`, GI.check, "Check the game's files", "Steam downloads any that are missing or damaged") : "",
          f.source === "steam" && f.appId ? gpRow(`data-action="move-open" data-app="${f.appId}" data-name="${esc(f.name)}"`, GI.drive, "Move to another drive", "To another of your Steam libraries") : "",
        ])
        + gpGroup("Online", [offlineRow(it)])
        + gpGroup("Shortcuts", [gpRow(`data-action="found-shortcut" data-key="${esc(f.key)}"`, GI.desktop, "Create desktop shortcut", "Starts the game directly")])
        + gpGroup("Danger", [gpRow(`data-action="found-hide" data-key="${esc(f.key)}"`, GI.hide, f.added ? "Remove from the Library" : "Hide from the Library", "", { danger: true })]);
    }
    const g = it.g, s = session(g.id);
    return gpGroup("Profiles", [
      gpRow(`data-action="load-only" data-game="${esc(g.id)}"`, GI.load, "Load someone's files", "Without starting the game", { disabled: s }),
      g.launch.type === "steam" && !g.steamDirect ? gpRow(`data-action="steam-setup" data-game="${esc(g.id)}"`, GI.ask, "Ask who's playing in Steam", "When the game starts from Steam's Play button") : "",
      gpRow(`data-action="edit-game" data-game="${esc(g.id)}"`, GI.edit, "Edit game", "How it starts and what profiles swap", { disabled: s }),
    ]) + gpGroup("Game files", [
      gpRow(`data-action="open-game-folder" data-game="${esc(g.id)}"`, GI.folder, "Open the game's folder", "Where it's installed"),
      gpRow(`data-action="open-path" data-path="${esc((g.paths[0] || {}).path || "")}"`, GI.save, "Open the save folder", "The files profiles swap"),
      gpRow(`data-action="open-backups" data-game="${esc(g.id)}"`, GI.history, "Open backups", "A copy is made before every swap"),
      g.launch.type === "steam" ? gpRow(`data-action="steam-verify" data-app="${g.launch.appId}"`, GI.check, "Check the game's files", "Steam downloads any that are missing or damaged", { disabled: s }) : "",
      g.launch.type === "steam" ? gpRow(`data-action="move-open" data-app="${g.launch.appId}" data-name="${esc(g.name)}"`, GI.drive, "Move to another drive", "To another of your Steam libraries", { disabled: s }) : "",
    ]) + gpGroup("Online", [offlineRow(it)])
      + gpGroup("Shortcuts", [gpRow(`data-action="shortcut" data-game="${esc(g.id)}"`, GI.desktop, "Create desktop shortcut", "Asks who's playing every time")])
      + gpGroup("Danger", [gpRow(`data-action="remove-game" data-game="${esc(g.id)}"`, GI.power, "Turn off profiles", "", { danger: true })]);
  }
  // A panel from the right. menu: each action in it closes it first.
  const drawer = (o) => `<div class="scrim ln-drawer-scrim" data-action="scrim"><aside class="ln-drawer ln-root${o.wide ? " ln-drawer--wide" : ""}" tabindex="-1" role="dialog" aria-modal="true" aria-label="${o.label}"${o.menu ? " data-menu" : ""}>
      <header class="ln-drawer__head"><div><h2>${o.title}</h2>${o.sub ? `<p>${o.sub}</p>` : ""}</div><button class="ln-button ln-button--ghost ln-button--icon ln-button--sm" data-action="dialog-cancel" aria-label="Close">${GI.x}</button></header>
      <div class="ln-drawer__body">${o.body}</div>${o.foot ? `<footer class="ln-drawer__foot">${o.foot}</footer>` : ""}</aside></div>`;
  const dialogItem = (D) => libraryItems().find((x) => x.kind === D.ik && x.id === D.iid) || null;
  function viewManage() {
    const it = dialogItem(S.dialog);
    if (!it) return "";
    return drawer({ label: `Manage ${esc(it.name)}`, title: esc(it.name), sub: it.kind === "found" ? "Profiles are off for this game" : "Its files, shortcuts and how it starts", body: `<div class="gpage-manage-list">${manageGroups(it)}</div>`, menu: true });
  }
  // Achievement Doctor: games whose achievements a tool keeps on this PC.
  const docWorth = (it) => !(it.kind === "found" && (storeKept(it.source) || it.appId > 0));
  function docDir(it) { return it.f ? it.f.dir : it.exe; }
  function viewAchDoctor() {
    const it = dialogItem(S.dialog);
    if (!it) return "";
    const key = recentKey(it), R = S.doctor[key];
    const shell = (body, foot) => drawer({ label: `Check ${esc(possessive(it.name))} achievements`, title: "Achievement check", sub: `${esc(it.name)}, step by step. Lootnest only looks; nothing changes unless you press a button below.`, body, foot });
    if (!R) return shell(lnInline("", "Checking", "Looking next to the game and where tools keep their files"));
    if (R.error) return shell(lnInline("error", esc(R.error)));
    const tone = { ok: "ok", warn: "mid", fail: "low", info: "" };
    const steps = (R.steps || []).map((x) => `<li class="doc-step run-card${tone[x.state] ? " run-card--" + tone[x.state] : ""}"><div class="run-head"><span class="run-dot"></span><div><b>${esc(x.title)}</b><p>${esc(x.detail)}</p>
        ${x.path ? `<p class="doc-path selectable">${esc(x.path)}</p>` : ""}</div></div></li>`).join("");
    const bad = (R.steps || []).find((x) => x.state === "fail") || (R.steps || []).find((x) => x.state === "warn");
    const N = R.nameInGame;
    const seat = (S.seat && player(S.seat)) || null;
    const nameBlock = N ? `<div class="gpage-sub">${gpHead("The name in the game")}<p class="gpage-hint">The tool shows ${N.name ? `<b class="selectable">${esc(N.name)}</b>` : "its own default name"} in the game, and to friends on LAN. ${seat ? `Set it to ${esc(possessive(seat.name))} name; its settings are kept in backups first.` : "Sit down in Who's here to set it to your profile's name."}</p>
        ${seat && seat.name !== N.name ? `<div class="gpage-actions"><button class="ln-button ln-button--secondary ln-button--sm" data-action="doc-name" ${itemAttr(it)} data-player="${esc(seat.id)}">Use ${esc(possessive(seat.name))} name</button></div>` : ""}</div>` : "";
    return shell(`<p class="gpage-hint">${bad ? `Most likely: ${esc(bad.title.charAt(0).toLowerCase() + bad.title.slice(1))}.` : "Everything Lootnest checks is in place."}</p><ol class="doc-steps">${steps}</ol>${nameBlock}`,
      `<button class="ln-button ln-button--ghost ln-button--sm" data-action="show-log-nav">Show recent activity</button><span class="grow"></span><button class="ln-button ln-button--secondary ln-button--sm" data-action="ach-doctor" ${itemAttr(it)}>Check again</button>`);
  }
  function viewAchDrawer() {
    const it = dialogItem(S.dialog);
    if (!it) return "";
    const A = achFor(it);
    if (!A) return "";
    return drawer({ label: `${esc(it.name)} achievements`, title: A.who ? esc(possessive(pname(A.who))) + " achievements" : "Achievements",
      sub: `${esc(it.name)}, ${A.A.unlocked} of ${A.A.total}`, body: `<div class="gp-ach gp-ach--all">${A.whoRow}${A.bar}<ul class="ach-list">${A.A.list.map(A.row).join("")}</ul></div>`, foot: `<p class="gpage-hint ln-drawer__note">${A.note}</p>${A.toggle}` });
  }
  function hookBanner() {
    if (!S.hooks || S.hideHookBanner) return "";
    const missing = S.hooks.games.filter((h) => !h.detected && !h.direct && game(h.gameId));
    const leaving = S.hooks.games.filter((h) => h.detected && h.direct && game(h.gameId));
    if (!missing.length && leaving.length) {
      const what = leaving.length === 1 ? esc(leaving[0].name) : leaving.length + " games";
      const text = S.hooks.steamRunning
        ? `So that Steam starts ${what} by itself, Steam has to restart once (about 10 seconds) while Lootnest takes itself out of the launch options.`
        : `Lootnest takes itself out of the launch options for ${what}. Steam is closed, so this takes a second.`;
      return banner(`Let Steam start ${what}`, text, bBtn("ghost", 'data-action="hide-banner"', "Later")
        + bBtn("secondary", `data-action="hooks-apply" ${S.hooksBusy ? 'aria-busy="true"' : ""}`, (S.hooksBusy ? '<span class="ln-spinner" aria-hidden="true"></span>' : "") + (S.hooksBusy ? "Updating" : S.hooks.steamRunning ? "Restart Steam and update" : "Update now")));
    }
    if (!missing.length) return "";
    const what = missing.length === 1 ? esc(missing[0].name) : missing.length + " games";
    const text = S.hooks.steamRunning
      ? `So that Steam's Play button asks who's playing ${what}, Steam has to restart once (about 10 seconds). Lootnest does it for you.`
      : `Lootnest can set Steam up so its Play button asks who's playing ${what}.`;
    return banner("Make Steam ask who's playing", text, bBtn("ghost", 'data-action="hide-banner"', "Later")
      + bBtn("secondary", `data-action="hooks-apply" ${S.hooksBusy ? 'aria-busy="true"' : ""}`, (S.hooksBusy ? '<span class="ln-spinner" aria-hidden="true"></span>' : "") + (S.hooksBusy ? "Setting up" : S.hooks.steamRunning ? "Restart Steam and set up" : "Set up")));
  }

  async function loadFound() {
    try { S.found = await api.found(); } catch (e) { /* not ready */ }
    if (S.view === "library" && !S.picker) render();
  }

  // ---------------------------------------------------------------- picker --
  function openPicker(gameId, mode, reqId) {
    const g = game(gameId);
    if (!g) { toast("That game isn't in Lootnest anymore.", "error"); if (reqId) api.cancelRequest(reqId); return; }
    if (S.picker && S.picker.reqId && S.picker.reqId !== reqId) api.cancelRequest(S.picker.reqId);
    S.menuFor = null;
    if (S.dialog && S.dialog.kind !== "welcome") S.dialog = null;
    S.picker = { gameId, mode: mode || "play", reqId: reqId || "" };
    render();
    setTimeout(() => { const pk = $(".picker"); if (pk) pk.focus(); }, 30); // keyboard 1-9 works without favouring anyone
  }

  function closePicker() {
    if (S.picker && S.picker.reqId && !S.picker.working) api.cancelRequest(S.picker.reqId);
    S.picker = null;
    render();
  }

  // A slot's line: when they last played it (whose files are in shows on
  // the slot's label), else whether they have a profile yet.
  function slotState(gid, pid) {
    const at = lastBy({ kind: "game", id: gid }, pid), i = profileInfo(gid, pid);
    if (at) return "Played " + ago(at);
    if (i.saved) return "Saved " + ago(i.saved);
    return S.d.active[gid] === pid ? "Files loaded now" : hasProfile(gid, pid) ? "Has saves here" : "First time";
  }

  // A new profile from the picker: a new person, or another run of this
  // game ("Cristian, hardcore"). It starts fresh or from a copy of whose
  // files are in, as any first-timer does.
  function newProfileSlot(P) {
    if (!P.creating) return `<button class="slot slot-new" data-action="np-open">
        <span class="slot-new-plus" aria-hidden="true">${GI.plus}</span>
        <span class="name">New profile</span><span class="state">Someone new, or another run</span>
        <span class="slot-label"><span>Add a profile</span></span></button>`;
    return `<div class="slot slot-new editing" role="group" aria-label="New profile">
        <span class="slot-new-plus" aria-hidden="true">${GI.plus}</span>
        <label class="name" for="np-name">New profile</label>
        <input id="np-name" class="ln-input" type="text" maxlength="24" autocomplete="off" spellcheck="false" placeholder="Name, such as Cristian hardcore" value="${esc(P.newName || "")}">
        <select id="np-of" class="ln-input" aria-label="Whose profile">
          <option value="">Someone new</option>
          ${cfg().players.filter((p) => !parentOf(p)).map((p) => `<option value="${esc(p.id)}" ${P.newOf === p.id ? "selected" : ""}>A run for ${esc(p.name)}</option>`).join("")}</select>
        <span class="error-text" id="np-error" role="alert">${esc(P.newError || "")}</span>
        <div class="slot-new-actions"><button class="ln-button ln-button--ghost ln-button--sm" data-action="np-cancel">Cancel</button><button class="ln-button ln-button--primary ln-button--sm" data-action="np-create">${P.mode === "load" ? "Make it" : "Make it and play"}</button></div></div>`;
  }

  function viewPicker() {
    const P = S.picker, g = game(P.gameId);
    if (!g) return "";
    if (P.working) {
      const to = player(P.playerId), from = P.from && P.from !== P.playerId && player(P.from);
      const s = session(g.id);
      const line = s && s.phase === "launching" ? `Starting ${esc(g.name)}` : P.mode !== "load" ? `Loading ${esc(possessive(pname(P.playerId)))} files and starting ${esc(g.name)}` : `Loading ${esc(possessive(pname(P.playerId)))} files`;
      return `<div class="picker ln-root" role="dialog" aria-label="${esc(line)}"><div class="picker-head"><div></div></div>
        <div class="swap" aria-live="polite">
          <div class="swap-bay">${from ? `<div class="swap-out" aria-hidden="true">${lnAvatar(from, "120")}</div>` : ""}<div class="swap-in">${to ? lnAvatar(to, "120") : ""}</div></div>
          <div class="swap-text"><b>${esc(pname(P.playerId))}</b><p><span class="ln-spinner" aria-hidden="true"></span>${line}</p>
            ${from ? `<p class="swap-safe">${GI.copy}A copy of ${esc(possessive(from.name))} files was made first.</p>` : `<p class="swap-safe">${GI.copy}A copy is made before every swap.</p>`}</div>
        </div><div></div></div>`;
    }
    if (P.ask === "owner") {
      return `<div class="picker"><div class="picker-head"><div></div><button class="btn quiet" data-action="picker-back">${icon.back}Back</button></div>
        <div class="question"><div><h2>Whose files are in ${esc(g.name)} right now?</h2>
        <p>This is the first time Lootnest handles this game. The settings and saves on this PC become that profile's.</p></div>
        <div class="choices">
          ${cfg().players.map((p) => `<button class="choice" data-action="answer-owner" data-player="${esc(p.id)}"><span class="dot" style="background:${esc(p.color)}"></span><span><b>${esc(possessive(p.name))}</b></span></button>`).join("")}
          <button class="choice" data-action="answer-owner" data-player="none"><span class="dot"></span><span><b>Nobody's, or not sure</b><span>They're kept as a backup in case you need them.</span></span></button>
        </div></div><div></div></div>`;
    }
    if (P.ask === "cloud") {
      const C = P.conflict || {}, n = pname(P.playerId), other = C.computer || "another PC";
      return `<div class="picker"><div class="picker-head"><div></div><button class="btn quiet" data-action="picker-back">${icon.back}Back</button></div>
        <div class="question"><div><h2>${esc(possessive(n))} ${esc(g.name)} saves changed on two PCs</h2>
        <p>They changed on this PC and on ${esc(other)}${C.changed ? " (" + esc(ago(C.changed)) + ")" : ""} since they last synced. Choose which to play with; the other is kept in this game's backups.</p></div>
        <div class="choices">
          <button class="choice" data-action="answer-cloud" data-answer="here"><span class="dot" style="background:${esc(color(P.playerId))}"></span><span><b>Keep this PC's saves</b><span>They go to the cloud after you play.</span></span></button>
          <button class="choice" data-action="answer-cloud" data-answer="cloud"><span class="dot"></span><span><b>Use the saves from ${esc(other)}</b><span>They come down now, before the game starts.</span></span></button>
        </div></div><div></div></div>`;
    }
    if (P.ask === "newMode") {
      const owner = P.owner && P.owner !== "none" ? P.owner : S.d.active[g.id];
      const ownerName = owner && player(owner) ? possessive(pname(owner)) + " files" : "the current files";
      const n = pname(P.playerId);
      return `<div class="picker"><div class="picker-head"><div></div><button class="btn quiet" data-action="picker-back">${icon.back}Back</button></div>
        <div class="question"><div><h2>${esc(n)} hasn't played ${esc(g.name)} yet</h2>
        <p>Choose what ${esc(n)} starts with. Whatever happens, the files on the PC now are kept.</p></div>
        <div class="choices">
          <button class="choice" data-action="answer-new" data-mode="fresh"><span class="dot" style="background:${esc(color(P.playerId))}"></span><span><b>Start fresh</b><span>Game defaults and no saves, as if ${esc(n)} just installed it.</span></span></button>
          <button class="choice" data-action="answer-new" data-mode="copy"><span class="dot"${owner && player(owner) ? ` style="background:${esc(color(owner))}"` : ""}></span><span><b>Start from a copy of ${esc(ownerName)}</b><span>Same settings and saves to begin with; changes stay separate from then on.</span></span></button>
          ${cfg().players.filter((p) => p.id !== owner && p.id !== P.playerId && hasProfile(g.id, p.id)).map((p) => `<button class="choice" data-action="answer-new" data-mode="copy" data-from="${esc(p.id)}"><span class="dot" style="background:${esc(p.color)}"></span><span><b>Start from a copy of ${esc(possessive(p.name))} files</b><span>Their saves as they were when they last played; theirs stay as they are.</span></span></button>`).join("")}
        </div></div><div></div></div>`;
    }
    const ps = cfg().players;
    return `<div class="picker ln-root" role="dialog" tabindex="-1" aria-label="Who's playing ${esc(g.name)}?">
      <div class="picker-head"><div><h1>Who's playing ${esc(g.name)}?</h1>
        <p class="hint">${P.mode === "steam" ? "Steam is starting the game. Their settings and saves load first." : P.mode === "play" ? "Their settings and saves load before the game starts." : "Their settings and saves are loaded; the game isn't started."}</p></div>
        <button class="ln-button ln-button--ghost" data-action="picker-close">${P.mode === "steam" ? "Don't start" : "Cancel"}</button></div>
      <div class="slots">
        ${grouped(ps).map((p, i) => `<button class="slot ln-player${S.d.active[g.id] === p.id ? " in" : ""}${parentOf(p) ? " slot-run" : ""}" style="--player:${esc(p.color)}" data-action="pick" data-player="${esc(p.id)}">
          ${lnAvatar(p, parentOf(p) ? "72" : "120")}
          ${parentOf(p) ? `<span class="slot-of">${esc(possessive(parentOf(p).name))} run</span>` : ""}<span class="name">${esc(parentOf(p) ? runLabel(p) : p.name)}</span><span class="state">${esc(slotState(g.id, p.id))}</span>
          <span class="slot-label"><span>${S.d.active[g.id] === p.id ? "In the game now" : parentOf(p) ? "Play this run" : "Play as " + esc(p.name)}</span>${i < 9 ? `<kbd aria-hidden="true">${i + 1}</kbd>` : ""}</span></button>`).join("")}
        ${ps.length < MAX_PROFILES ? newProfileSlot(P) : ""}
      </div>
      <div class="picker-foot"><span>Press ${ps.length > 1 ? "1–" + Math.min(ps.length, 9) : "1"} to choose, Esc ${P.mode === "steam" ? "to not start the game" : "to cancel"}</span><span>${esc(g.name)}</span></div>
    </div>`;
  }

  async function runPick() {
    const P = S.picker;
    const req = { gameId: P.gameId, playerId: P.playerId, firstOwner: P.owner || "", newProfile: P.newMode || "", copyFrom: P.copyFrom || "", reqId: P.reqId || "", cloud: P.cloud || "" };
    try {
      const res = await (P.mode === "steam" ? api.steamPick(req) : P.mode === "play" ? api.play(req) : api.load(req));
      if (!S.picker) return;
      if (res && res.need) { S.picker.ask = res.need; S.picker.conflict = res.cloud || null; render(); return; }
      S.picker.working = true;
      render();
    } catch (e) {
      toast(errText(e), "error");
      if (S.picker) { S.picker.ask = null; S.picker.working = false; render(); }
    }
  }

  function onSessions() {
    const P = S.picker;
    if (!P || !P.working) return;
    const s = session(P.gameId);
    // The picker stays up ("Starting …") until the game is actually running.
    if (!s || s.phase === "playing") { S.picker = null; return; }
    render();
  }

  // ---------------------------------------------------------------- dialogs --
  let confirmResolve = null;
  function confirmDialog(o) {
    return new Promise((resolve) => { confirmResolve = resolve; S.dialog = Object.assign({ kind: "confirm" }, o); renderOverlay(); setTimeout(() => { const b = $(".dialog .btn.primary, .dialog .btn.danger-primary"); if (b) b.focus(); }, 20); });
  }
  function closeDialog(v) { const wasSteam = S.dialog && S.dialog.kind === "steam"; S.dialog = null; renderOverlay(); if (wasSteam) render();
    if (drawerFrom) { if (document.contains(drawerFrom)) drawerFrom.focus(); drawerFrom = null; } if (confirmResolve) { const r = confirmResolve; confirmResolve = null; r(v); } }

  // Redrawing the dialog that's already open (a choice clicked, progress, an
  // answer from Go) must not play its opening animation again.
  let shownOverlay = null, shownStep = "";
  function renderOverlay() {
    const o = $("#overlay");
    const P = S.picker;
    const now = P || S.dialog;
    const step = P ? (P.ask || "") + (P.working ? ":working" : "") : "";
    o.classList.toggle("still", now !== null && now === shownOverlay && step === shownStep);
    shownOverlay = now; shownStep = step;
    if (S.picker) { o.innerHTML = viewPicker(); return; }
    if (!S.dialog) { o.innerHTML = ""; return; }
    const D = S.dialog;
    if (D.kind === "confirm") {
      o.innerHTML = `<div class="scrim" data-action="scrim"><div class="dialog narrow" role="alertdialog" aria-modal="true">
        <div><h2>${esc(D.title)}</h2><p class="sub">${esc(D.body)}</p></div>
        <div class="dialog-foot"><button class="btn quiet" data-action="dialog-cancel">${esc(D.cancel || "Cancel")}</button>
        <button class="btn primary" data-action="dialog-ok">${esc(D.ok || "OK")}</button></div></div></div>`;
    } else if (D.kind === "add") {
      o.innerHTML = viewAddDialog();
    } else if (D.kind === "welcome") {
      o.innerHTML = viewWelcome();
    } else if (D.kind === "steam") {
      o.innerHTML = viewSteamSetup();
    } else if (D.kind === "blocked") {
      o.innerHTML = viewBlocked();
    } else if (D.kind === "move") {
      o.innerHTML = viewMove();
    } else if (D.kind === "backup-pw") {
      o.innerHTML = viewBackupPassword();
    } else if (D.kind === "restore") {
      o.innerHTML = viewRestore();
    } else if (D.kind === "export") {
      o.innerHTML = viewExport();
    } else if (D.kind === "profile") {
      const q = $("#pe-q"), pos = q && document.activeElement === q ? q.selectionStart : -1;
      o.innerHTML = viewProfileEditor();
      if (pos >= 0) { const q2 = $("#pe-q"); q2.focus(); q2.setSelectionRange(pos, pos); }
    } else if (D.kind === "news") {
      o.innerHTML = viewReader();
      if (!$("#overlay").classList.contains("still")) { const b = $(".reader-body"); if (b) b.scrollTop = 0; }
    } else if (D.kind === "game-settings") {
      o.innerHTML = viewGameSettings();
    } else if (D.kind === "manage") {
      o.innerHTML = viewManage();
    } else if (D.kind === "achievements") {
      o.innerHTML = viewAchDrawer();
    } else if (D.kind === "points") {
      o.innerHTML = viewPointsDrawer();
      const r = $("#pt-rename"); if (r && document.activeElement !== r) { r.focus(); r.select(); }
    } else if (D.kind === "achdoc") {
      o.innerHTML = viewAchDoctor();
    } else if (D.kind === "offer") {
      o.innerHTML = viewOffer();
    } else if (D.kind === "update") {
      o.innerHTML = viewUpdateDrawer();
    } else if (D.kind === "profiles") {
      o.innerHTML = viewProfilesDrawer();
    }
    if (!o.innerHTML) S.dialog = null; // what it was for is gone
  }
  // Focus moves into a panel when it opens, onto its first action.
  // Closing it gives focus back to what opened it.
  let drawerFrom = null;
  function focusDrawer() {
    drawerFrom = document.activeElement;
    setTimeout(() => { const d = $(".ln-drawer"); if (d) d.focus(); }, 30);
  }

  // ------------------------------------------------------------ game mover --
  const MOVE_MARGIN = 512 * 1024 * 1024; // room kept free beyond the game's size

  function openMove(appId, name) {
    S.dialog = { kind: "move", appId, name, info: null, pick: "" };
    renderOverlay();
    if (!S.moving) api.moveInfo(appId).catch((e) => { if (S.dialog && S.dialog.kind === "move") { S.dialog.info = { error: errText(e), libraries: [] }; renderOverlay(); } });
  }

  function viewMove() {
    const D = S.dialog, M = S.moving, I = D.info;
    const head = (sub) => `<div><h2>Move ${esc(D.name)} to another drive</h2>${sub ? `<p class="sub">${sub}</p>` : ""}</div>`;
    const wrap = (inner) => `<div class="scrim" data-action="scrim"><div class="dialog narrow" role="dialog" aria-modal="true" aria-label="Move ${esc(D.name)}">${inner}</div></div>`;
    if (M && M.appId === D.appId) {
      const pct = M.total ? Math.min(100, Math.round((M.done / M.total) * 100)) : 0;
      const text = M.step === "steam" ? "Closing Steam" : M.total ? `Copying ${fmtSize(M.done)} of ${fmtSize(M.total)}` : "Starting";
      return wrap(`${head("")}<div class="mover"><div class="meter"><i style="width:${pct}%"></i></div><p class="hint">${esc(text)}</p></div>
        <p class="hint">You can close this window; Lootnest keeps going and tells you when it's done. Keep Steam closed until then.</p>
        <div class="dialog-foot"><button class="btn quiet" data-action="dialog-cancel">Close</button></div>`);
    }
    if (M) {
      return wrap(`${head("")}<p class="note">Lootnest is moving ${esc(M.name || "another game")}. Wait for it to finish, then move this one.</p>
        <div class="dialog-foot"><button class="btn quiet" data-action="dialog-cancel">Close</button></div>`);
    }
    if (!I) return wrap(`${head("Working out the game's size")}<div class="dialog-foot"><button class="btn quiet" data-action="dialog-cancel">Cancel</button></div>`);
    if (I.error) return wrap(`${head("")}<p class="note">${esc(I.error)}</p><div class="dialog-foot"><button class="btn quiet" data-action="dialog-cancel">Close</button></div>`);
    const cur = I.libraries.find((l) => l.current);
    const others = I.libraries.filter((l) => !l.current);
    const room = (l) => l.free < 0 || l.free >= I.size + MOVE_MARGIN;
    const list = others.length
      ? `<div class="choices">${others.map((l) => `<button class="choice${D.pick === l.path ? " on" : ""}" data-action="move-pick" data-lib="${esc(l.path)}" ${room(l) ? "" : "disabled"} aria-pressed="${D.pick === l.path}">
          <span class="dot"></span><span><b>${esc(l.path)}</b><span>${l.free < 0 ? "Free space unknown" : fmtSize(l.free) + " free"}${room(l) ? "" : " · not enough room"}</span></span></button>`).join("")}</div>`
      : `<p class="note">Steam has one library on this PC. To move games to another drive, add a library there in Steam: <b>Settings > Storage</b>, then <b>Add drive</b>. Then come back here.</p>`;
    return wrap(`${head(`${fmtSize(I.size)} · now in ${esc(cur ? cur.path : "")}`)}${list}
      ${others.length ? `<p class="hint">The game is copied and checked before anything is removed from where it is now.${I.steamRunning ? " Steam closes while the game moves and opens again when it's done." : ""}</p>` : ""}
      <div class="dialog-foot"><button class="btn quiet" data-action="dialog-cancel">Cancel</button>
        ${others.length ? `<button class="btn primary" data-action="move-go" ${D.pick ? "" : "disabled"}>Move</button>` : ""}</div>`);
  }

  // --------------------------------------------- game settings (no profiles) --
  function openGameSettings(key) {
    const f = (S.found.games || []).find((x) => x.key === key);
    if (!f) return;
    S.dialog = { kind: "game-settings", key, busy: false };
    renderOverlay();
    setTimeout(() => { const i = $("#gs-name"); if (i) i.focus(); }, 30);
  }

  function viewGameSettings() {
    const D = S.dialog;
    const f = (S.found.games || []).find((x) => x.key === D.key);
    if (!f) return "";
    const steam = f.source === "steam";
    return `<div class="scrim" data-action="scrim"><div class="dialog narrow" role="dialog" aria-modal="true" aria-label="${esc(f.name)} settings">
      <div><h2>${esc(f.name)} settings</h2><p class="sub">These settings don't depend on profiles and stay if you turn them on.</p></div>
      <div class="field"><label for="gs-name">Name in the Library</label>
        <input id="gs-name" type="text" value="${esc(f.name)}" maxlength="60" placeholder="${esc(f.scanName || f.name)}" spellcheck="false">
        ${f.scanName ? `<p class="hint">Found as ${esc(f.scanName)}. Leave it empty to use that name again.</p>` : ""}</div>
      <div class="field"><label for="gs-args">Launch options</label>
        <input id="gs-args" type="text" value="${esc(f.args || "")}" placeholder="${steam ? "-novid -high" : "-windowed"}" spellcheck="false">
        <p class="hint">${steam
          ? "Added when Lootnest or its desktop shortcut starts the game. Launch options set in Steam still apply too."
          : "Added when Lootnest or its desktop shortcut starts the game."}</p></div>
      <div class="dialog-foot"><button class="btn quiet" data-action="dialog-cancel" ${D.busy ? "disabled" : ""}>Cancel</button>
        <button class="btn primary" data-action="game-settings-save" ${D.busy ? "disabled" : ""}>Save</button></div>
    </div></div>`;
  }

  // ---------------------------------------------------- anti-cheat block --
  function viewBlocked() {
    const D = S.dialog, g = game(D.gameId);
    if (!g) return "";
    const who = D.playerId && player(D.playerId) ? pname(D.playerId) : "";
    return `<div class="scrim"><div class="dialog narrow" role="alertdialog" aria-modal="true" aria-label="An anti-cheat blocked ${esc(g.name)}">
      <div><h2>An anti-cheat blocked ${esc(g.name)}</h2>
        <p class="sub">Anti-cheats such as FACEIT AC only let Steam start the game, so Lootnest couldn't start it.</p></div>
      <div class="note">Lootnest can fix this now: Steam will start ${esc(g.name)} by itself (Steam restarts once, about 10 seconds), and ${who ? `the game starts for <b>${esc(who)}</b>` : "you choose who's playing"}.
        <br>Lootnest also puts a <b>${esc(g.name)} (Lootnest)</b> shortcut on your desktop. Start the game from it, or from Play here, and it asks who's playing every time.</div>
      ${who ? `<p class="hint">${esc(possessive(who))} files are loaded.</p>` : ""}
      <div class="dialog-foot"><button class="btn quiet" data-action="dialog-cancel" ${D.busy ? "disabled" : ""}>Not now</button>
        <button class="btn primary" data-action="fix-blocked" ${D.busy ? "disabled" : ""}>${D.busy ? "Fixing and starting" : "Fix it and start the game"}</button></div>
    </div></div>`;
  }

  // ------------------------------------------------------- steam set-up --
  async function loadHooks() {
    try { S.hooks = await api.steamSetup(); } catch (e) { S.hooks = null; }
  }

  async function openSteamSetup(gameId) {
    S.menuFor = null;
    S.dialog = { kind: "steam", focus: gameId || "", copied: "" };
    renderOverlay();
    await loadHooks();
    if (S.dialog && S.dialog.kind === "steam") renderOverlay();
  }

  function viewSteamSetup() {
    const D = S.dialog;
    const H = S.hooks;
    let body;
    if (!H) body = `<p class="hint">Reading Steam's settings</p>`;
    else if (!H.games.length) body = `<div class="note">No Steam games in Lootnest yet. Add one first.</div>`;
    else {
      const list = (D.focus ? H.games.filter((h) => h.gameId === D.focus) : H.games).filter((h) => !h.direct);
      body = list.map((h) => `<div class="hook${h.detected ? " done" : ""}">
          <div class="row" style="justify-content:space-between"><b>${esc(h.name)}</b>
            <span class="${h.detected ? "ok-text" : "hint"}">${h.detected ? "Set up: Steam asks who's playing" : "Not set up yet"}</span></div>
          <div class="hook-line"><code class="selectable">${esc(h.option)}</code>
            <button class="btn small" data-action="copy-hook" data-game="${esc(h.gameId)}">${D.copied === h.gameId ? "Copied" : "Copy"}</button></div>
          ${h.current && !h.detected ? `<span class="hint">Includes the launch options this game already had.</span>` : ""}
        </div>`).join("");
    }
    return `<div class="scrim" data-action="scrim"><div class="dialog" role="dialog" aria-modal="true" aria-label="Ask who's playing when started from Steam">
      <div class="page-head" style="margin:0"><div><h2>Ask who's playing when started from Steam</h2>
        <p class="sub">Do this once per game. Afterwards, pressing Play in Steam opens the picker, loads that person's files, then starts the game.</p></div>
        <button class="btn quiet icon" data-action="dialog-cancel" aria-label="Close">${icon.x}</button></div>
      ${H && H.games.some((h) => !h.detected && !h.direct) ? `<div class="note"><b>Easiest:</b> let Lootnest do it. ${H.steamRunning ? "Steam closes and reopens by itself (about 10 seconds)." : "Steam is closed, so this takes a second."}
        <div class="row" style="margin-top:10px"><button class="btn primary" data-action="hooks-apply" ${S.hooksBusy ? "disabled" : ""}>${S.hooksBusy ? "Setting up" : "Set up automatically"}</button></div></div>
      <p class="hint">Or by hand:</p>` : ""}
      <ol class="steps">
        <li>Copy the line for the game.</li>
        <li>In Steam, right-click the game in your library and choose <b>Properties</b>.</li>
        <li>On the <b>General</b> page, paste it into <b>Launch options</b>, replacing what's there.</li>
      </ol>
      ${H && H.error ? `<div class="note">${esc(H.error)}</div>` : ""}
      <div class="hooks">${body}</div>
      <div class="dialog-foot" style="justify-content:space-between"><span class="hint">Steam can take a minute to save the change. Games outside Steam: use the desktop shortcut from the game's menu.</span>
        <button class="btn" data-action="steam-recheck">Check again</button></div>
    </div></div>`;
  }

  // -------------------------------------------------------------- add game --
  function viewAddDialog() {
    const D = S.dialog;
    const q = (D.q || "").toLowerCase();
    const inLibrary = new Set((S.found.games || []).filter((f) => f.appId).map((f) => f.appId)
      .concat(cfg().games.filter((g) => g.launch.type === "steam").map((g) => g.launch.appId)));
    let list;
    if (!S.d.steam.found) list = `<div class="note">Steam wasn't found on this PC. You can still add any game by choosing its .exe.</div>`;
    else if (!D.steam) list = `<p class="hint">Looking for installed Steam games</p>`;
    else {
      const items = D.steam.filter((g) => !q || g.name.toLowerCase().includes(q));
      list = items.length
        ? `<div class="steam-list">${items.map((g) => `<button class="steam-item${inLibrary.has(g.appId) ? " added" : ""}" data-action="add-steam" data-app="${g.appId}"><b>${esc(g.name)}</b><span>${inLibrary.has(g.appId) ? "In your Library" : "Steam"}</span></button>`).join("")}</div>`
        : `<p class="hint">No installed Steam game matches “${esc(D.q)}”.</p>`;
    }
    return `<div class="scrim" data-action="scrim"><div class="dialog" role="dialog" aria-modal="true" aria-label="Add a game">
      <div class="page-head" style="margin:0"><div><h2>Add a game</h2><p class="sub">It goes into your Library, ready to play. Turn on profiles from its panel if more than one person plays it.</p></div>
      <button class="btn quiet icon" data-action="dialog-cancel" aria-label="Close">${icon.x}</button></div>
      ${S.d.steam.found ? `<input type="text" id="steam-q" placeholder="Search your Steam games" value="${esc(D.q || "")}" autocomplete="off">` : ""}
      ${list}
      <div class="dialog-foot" style="justify-content:space-between"><span class="hint">Epic, GOG, Battle.net, Xbox and standalone games: choose the .exe.</span>
      <button class="btn" data-action="add-exe">Choose an .exe</button></div>
    </div></div>`;
  }

  async function startAddDialog() {
    S.menuFor = null;
    S.dialog = { kind: "add", q: "" };
    renderOverlay();
    setTimeout(() => { const i = $("#steam-q"); if (i) i.focus(); }, 30);
    if (S.d.steam.found) {
      try { S.dialog.steam = await api.steamGames(); } catch (e) { S.dialog.steam = []; toast(errText(e), "error"); }
      if (S.dialog && S.dialog.kind === "add") { const q = $("#steam-q"); const pos = q ? q.selectionStart : 0; renderOverlay(); const q2 = $("#steam-q"); if (q2) { q2.focus(); q2.setSelectionRange(pos, pos); } }
    }
  }

  // Adding a game puts it in the Library, without profiles; its panel then
  // opens, where profiles can be turned on.
  async function addToLibrary(kind, appId, path) {
    S.dialog = null; renderOverlay();
    try {
      const r = await api.addLibraryGame(kind, appId, path);
      await loadFound();
      if (S.view !== "library") { S.view = "library"; S.animateView = true; render(); }
      if (r.gameId) { openSheet("game", r.gameId); return; }
      toast(`${r.name} is in your Library`);
      openSheet("found", r.key);
    } catch (e) { toast(errText(e), "error"); }
  }

  const addSteam = (appId) => addToLibrary("steam", Number(appId), "");

  async function addExe() {
    const r = await browse("exe");
    if (!r.path) { if (r.error) toast(r.error, "error"); return; }
    await addToLibrary("exe", 0, r.path);
  }

  // ---------------------------------------------------------------- editor --
  function openEditor(e) {
    e.game = JSON.parse(JSON.stringify(e.game));
    e.game.paths.forEach((p) => { if (!p._k) p._k = "k" + ++seq; });
    e.checks = {};
    e.error = "";
    S.edit = e;
    S.view = "edit";
    render();
    $("#view").scrollTop = 0;
    e.game.paths.forEach(checkPath);
  }

  function editGame(id) {
    const g = game(id);
    if (!g) return;
    S.menuFor = null;
    const e = { isNew: false, game: g, inspect: null };
    openEditor(e);
    if (g.launch.type === "steam") api.inspectSteam(g.launch.appId).then((ins) => { if (S.edit === e) { e.inspect = ins; render(); } }).catch(() => {});
  }

  async function checkPath(p) {
    const e = S.edit;
    if (!e) return;
    const raw = p.path;
    if (!raw.trim()) { e.checks[p._k] = null; updatePathInfo(p); return; }
    try {
      const c = await api.checkPath(raw);
      if (S.edit !== e || p.path !== raw) return;
      e.checks[p._k] = c;
    } catch (err) { e.checks[p._k] = { error: errText(err) }; }
    updatePathInfo(p);
  }

  function pathInfo(p) {
    const c = S.edit.checks[p._k];
    if (!c) return `<div class="path-info">Checking</div>`;
    if (c.error) return `<div class="path-info bad">${esc(c.error)}</div>`;
    if (!c.exists) return `<div class="path-info">${esc(c.resolved)}. Doesn't exist yet; it's created the first time the game saves.</div>`;
    return `<div class="path-info">${esc(c.resolved)} (${c.isDir ? "folder" : "file"}, ${fmtSize(c.size)})</div>`;
  }

  function updatePathInfo(p) {
    const row = document.querySelector(`.path[data-k="${p._k}"] .path-info`);
    if (row) row.outerHTML = pathInfo(p);
  }

  function viewEdit() {
    const E = S.edit, g = E.game, ins = E.inspect || {};
    const taken = new Set(g.paths.map((p) => p.path.toLowerCase()));
    const sugg = (ins.suggestions || []).filter((s) => !taken.has(s.path.toLowerCase()));
    const cands = (ins.candidates || []).filter((c) => c.name.toLowerCase() !== (g.processName || "").toLowerCase()).slice(0, 6);
    const launch = g.launch.type === "steam"
      ? `<div class="note">Launched through Steam (app ${esc(g.launch.appId)}).</div>
        <label class="toggle"><span><b style="font-weight:500">Let Steam start this game</b><p>Turn this on if an anti-cheat, such as FACEIT AC, blocks the game when Lootnest starts it. Steam's Play button then starts the game without asking who's playing. Lootnest adds a desktop shortcut that asks every time.</p></span>
          <span class="switch"><input type="checkbox" data-field="steamDirect" ${g.steamDirect ? "checked" : ""}><span></span></span></label>`
      : g.launch.type === "uri"
      ? `<div class="note">Started through ${esc(sourceLabel((S.d.sources || {})[g.id] || "") === "Other" ? "its launcher" : sourceLabel((S.d.sources || {})[g.id]))}, like from the launcher itself.</div>
        ${g.launch.exe ? `<div class="row"><button class="btn quiet" data-action="launch-from-exe">Start it from its .exe instead</button></div>` : ""}`
      : `<div class="row"><input type="text" class="grow" data-field="exe" value="${esc(g.launch.exe)}" spellcheck="false"><button class="btn" data-action="browse-exe">Browse</button></div>`;
    return `<div class="page">
      <div class="page-head"><div><button class="btn quiet small" data-action="edit-cancel" style="margin:0 0 10px -12px">${icon.back}Library</button>
      <h1>${E.isNew ? "Add " + esc(g.name || "a game") : "Edit " + esc(g.name)}</h1></div></div>
      <div class="form">
        <section class="section">
          <div class="field"><label for="f-name">Name</label><input id="f-name" type="text" data-field="name" value="${esc(g.name)}" maxlength="60"></div>
          <div class="field"><span class="label">How it starts</span>${launch}</div>
        </section>

        <section class="section">
          <div class="section-head"><h2>Game process</h2><p>The program that runs while someone plays. Lootnest watches it to know when to save that person's progress.</p></div>
          <div class="row"><input type="text" class="grow" data-field="processName" value="${esc(g.processName)}" placeholder="for example cs2" spellcheck="false" style="max-width:320px"><span class="hint">.exe</span></div>
          ${cands.length ? `<div class="row"><span class="hint">Other programs in the game folder:</span>${cands.map((c) => `<button class="btn small" data-action="set-proc" data-name="${esc(c.name)}">${esc(c.name)}</button>`).join("")}</div>` : ""}
        </section>

        <section class="section">
          <div class="section-head"><h2>What gets swapped</h2><p>Every folder or file where the game keeps settings or saves. Each person gets their own copy of everything listed here.</p></div>
          ${ins.preset && ins.preset.note && g.paths.length ? `<div class="note">${esc(ins.preset.note)}</div>` : ""}
          <div class="paths">
            ${g.paths.length ? g.paths.map((p) => `<div class="path" data-k="${p._k}">
                <input type="text" data-field="path" data-k="${p._k}" value="${esc(p.path)}" spellcheck="false" aria-label="Save location">
                <div class="path-tools"><button class="btn icon quiet" data-action="open-path" data-path="${esc(p.path)}" aria-label="Open in Explorer">${icon.folder}</button>
                <button class="btn icon quiet" data-action="remove-path" data-k="${p._k}" aria-label="Remove">${icon.x}</button></div>
                ${pathInfo(p)}</div>`).join("") : `<p class="hint">Nothing yet. Add the game's settings folder and its saves folder.</p>`}
          </div>
          <div class="row"><button class="btn" data-action="add-path" data-kind="folder">${icon.plus}Add folder</button><button class="btn" data-action="add-path" data-kind="file">${icon.plus}Add file</button>
            <button class="btn quiet" data-action="pcgw">Look up where ${esc(g.name || "this game")} saves</button></div>
          ${sugg.length ? `<div class="suggest"><span class="label">Found on this PC</span>${sugg.map((s) => `<div class="suggest-item"><div class="grow"><code title="${esc(s.resolved)}">${esc(s.path)}</code><span class="hint">${esc(s.reason)}, ${fmtSize(s.size)}</span></div><button class="btn small" data-action="add-sugg" data-path="${esc(s.path)}">Add</button></div>`).join("")}</div>` : ""}
        </section>

        <section class="section">
          <div class="section-head"><h2>Launch options</h2><p>Optional. Added when Lootnest starts the game, for example <span class="selectable">-novid</span> or <span class="selectable">-w 1920 -h 1080</span>.</p></div>
          <div class="args">
            <span class="label">Everyone</span><input type="text" data-field="args" value="${esc(g.launch.args || "")}" spellcheck="false">
            ${cfg().players.map((p) => `<span class="who"><span class="dot" style="background:${esc(p.color)}"></span>${esc(p.name)}</span><input type="text" data-field="parg" data-player="${esc(p.id)}" value="${esc((g.playerArgs || {})[p.id] || "")}" spellcheck="false">`).join("")}
          </div>
        </section>
      </div>
      <div class="form-foot"><button class="btn primary" data-action="save-game" ${E.saving ? "disabled" : ""}>${E.isNew ? "Add game" : "Save changes"}</button>
        <button class="btn quiet" data-action="edit-cancel">Cancel</button><span class="error-text" id="edit-error">${esc(E.error)}</span></div>
    </div>`;
  }

  async function saveGame() {
    const E = S.edit;
    if (!E) return;
    E.saving = true; E.error = "";
    const g = JSON.parse(JSON.stringify(E.game));
    g.paths = g.paths.filter((p) => p.path.trim()).map((p) => ({ id: p.id || "", path: p.path.trim() }));
    try {
      const saved = await api.saveGame(g);
      toast(E.isNew ? `Added ${saved.name}` : `Saved ${saved.name}`);
      S.edit = null; S.view = "library"; S.animateView = true;
      await refresh();
      loadFound();
      loadHooks().then(render);
    } catch (e) {
      E.saving = false; E.error = errText(e);
      const b = $('[data-action="save-game"]'); if (b) b.disabled = false;
      const t = $("#edit-error"); if (t) t.textContent = E.error;
    }
  }

  // --------------------------------------------------------------- players --
  function viewWelcome() {
    const ps = S.players || (S.players = JSON.parse(JSON.stringify(cfg().players)));
    return `<div class="scrim"><div class="dialog narrow" role="dialog" aria-modal="true" aria-label="Welcome">
      <div><h2 style="font-size:22px">Who plays on this PC?</h2><p class="sub">Each profile gets a colour and its own settings, saves and achievements in every game you add. Make one per person, or a few for yourself, one per run. You can add more later.</p></div>
      <div class="players">${ps.map((p, i) => playerRow(p, i, ps.length, true)).join("")}</div>
      <div class="dialog-foot"><span class="error-text grow" id="players-error"></span><button class="btn primary" data-action="welcome-done">Continue</button></div>
    </div></div>`;
  }

  function playerRow(p, i, total, welcome) {
    // While editing, a row can be a run of a person in the list (not of
    // itself, not of a run, and not while it has runs of its own).
    const list = S.players || [];
    const par = !welcome && p.of && list.find((x) => x.id === p.of && x.id !== p.id && !x.of);
    const hasRuns = p.id && list.some((x) => x.of === p.id);
    const people = list.filter((x) => x !== p && !x.of && x.id && x.name.trim());
    const of = welcome ? "" : `<select class="ln-input player-of" data-field="pof" data-i="${i}" aria-label="Whose profile" ${hasRuns ? "disabled title=\"It has runs of its own\"" : ""}>
        <option value="">A person</option>${people.map((x) => `<option value="${esc(x.id)}" ${par && par.id === x.id ? "selected" : ""}>A run for ${esc(x.name)}</option>`).join("")}</select>`;
    return `<div class="player ln-player${par ? " player-run" : ""}" style="--player:${esc(p.color || "#6c9bd2")}">
      ${lnAvatar({ ...p, name: p.name || "?" }, "32")}
      <input type="text" class="ln-input" data-field="pname" data-i="${i}" value="${welcome && /^Player \d$/.test(p.name) ? "" : esc(p.name)}" placeholder="${welcome ? (i === 0 ? "Your name" : "Their name") : par ? par.name + " hardcore" : "Name"}" maxlength="24" aria-label="Name">
      ${of}
      ${par ? `<span class="player-same">${esc(possessive(par.name))} colour</span>` : `<div class="swatches" role="radiogroup" aria-label="Colour for ${esc(p.name || "player " + (i + 1))}">${S.d.colors.map((c) => `<button class="swatch" style="background:${c}" data-action="swatch" data-i="${i}" data-color="${c}" aria-pressed="${c.toLowerCase() === (p.color || "").toLowerCase()}" aria-label="${c}"></button>`).join("")}</div>`}
      ${welcome ? "" : `<button class="ln-button ln-button--ghost ln-button--icon ln-button--sm" data-action="remove-player" data-i="${i}" aria-label="Remove ${esc(p.name)}" title="Remove" ${total <= 1 ? "disabled" : ""}>${GI.x}</button>`}
    </div>`;
  }

  // ------------------------------------------------------------ profiles --
  // Each player's own page, which they arrange themselves: a banner, their
  // picture, a few words, the game they're into, their numbers, activity
  // and collection.
  const SECTIONS = [
    ["now", "Currently playing", "A game you're playing, with a note"],
    ["stats", "Numbers", "Time played, games, streak, longest session"],
    ["favorites", "Favourite games", "Up to six games, in your order"],
    ["activity", "Activity", "When you played, the last six months"],
    ["collection", "Collection", "Where your games come from"],
    ["recent", "Played lately", "The last games you played"],
  ];
  const sectionsOf = (p) => { const known = SECTIONS.map((s) => s[0]), on = (p.sections || []).filter((s) => known.includes(s)); return on.length ? on : known; };
  const daysOf = (pid) => ((S.d.activity || {}).days || {})[pid] || {};
  const dayKey = (d) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
  const daySum = (day) => Object.values(day || {}).reduce((a, b) => a + b, 0);

  const avatarHTML = (p, cls) => {
    const u = (S.d.avatars || {})[p.id];
    const g = !u && p.avatarGame && itemByKey(p.avatarGame);
    if (g) ensureCover(g);
    const gu = g && (S.covers[coverKey(g)] || S.icons[coverKey(g)]);
    const inner = u ? `<img src="${u}" alt="">` : gu ? `<img src="${gu}" alt="">` : esc(p.name.trim().charAt(0).toUpperCase());
    return `<span class="avatar ${cls || ""}" style="--c:${esc(p.color)}" aria-hidden="true">${inner}</span>`;
  };
  const itemByKey = (k) => libraryItems().find((it) => recentKey(it) === k || it.id === k);
  const itemAttr = (it) => (it.kind === "game" ? `data-game="${esc(it.id)}"` : `data-key="${esc(it.id)}"`);

  function playerStats(p) {
    const items = libraryItems();
    let total = 0, top = null, topT = 0, played = 0;
    items.forEach((it) => { const t = (playtimeOf(it) || {})[p.id] || 0; total += t; if (t) played++; if (t > topT) { top = it; topT = t; } });
    // Their own last time in each game, not whoever played it last.
    const recent = items.map((it) => ({ it, r: { at: lastBy(it, p.id), player: p.id } })).filter((x) => x.r.at)
      .sort((a, b) => new Date(b.r.at) - new Date(a.r.at));
    played = Math.max(played, recent.length);
    const profiles = cfg().games.filter((g) => hasProfile(g.id, p.id)).length;
    // Streaks: days in a row with some play, counted to today (or yesterday).
    const days = daysOf(p.id), keys = Object.keys(days).filter((k) => daySum(days[k]) > 0).sort();
    let best = 0, run = 0, prev = null;
    keys.forEach((k) => { const d = new Date(k + "T12:00:00"); run = prev && (d - prev) / 864e5 < 1.5 ? run + 1 : 1; best = Math.max(best, run); prev = d; });
    let streak = 0;
    for (let d = new Date(), i = 0; i < 400; i++, d.setDate(d.getDate() - 1)) {
      if (daySum(days[dayKey(d)]) > 0) streak++;
      else if (i > 0 || streak > 0) break;
    }
    const act = S.d.activity || {};
    const longest = (act.longest || {})[p.id], sessions = (act.sessions || {})[p.id] || 0;
    const sources = new Set(items.filter((it) => ((playtimeOf(it) || {})[p.id] || 0) > 0 || recent.some((x) => x.it === it)).map((it) => sourceLabel(it.source)));
    return { total, top, topT, recent, profiles, played, streak, best, longest, sessions, sources };
  }

  function bannerHTML(p) {
    const y = `background-position:center ${p.bannerY || 0}%`;
    if (p.banner === "image" && (S.d.banners || {})[p.id]) return `<div class="art" style="background-image:url('${S.d.banners[p.id]}');${y}"></div>`;
    const it = p.banner && p.banner !== "image" && itemByKey(p.banner);
    if (it) { loadArt(it); ensureCover(it); }
    const u = it ? artOf(it) : "";
    return u ? `<div class="art" style="background-image:url('${u}');${y}"></div>` : `<div class="art plain"></div>`;
  }

  function viewPlayer() {
    const p = player(S.playerPage);
    if (!p) { S.view = "players"; return viewPlayers(); }
    const st = playerStats(p);
    const last = st.recent.find((x) => x.it.kind === "game" && !session(x.it.id));
    const body = sectionsOf(p).map((s) => ppSection(s, p, st)).filter(Boolean).join("");
    return `<div class="profile-page ln-root ln-player" style="--player:${esc(p.color)}">
      <div class="pp-banner">${bannerHTML(p)}<button class="ln-button ln-button--ghost ln-button--sm gpage-back" data-action="nav" data-view="players">${GI.back}Profiles</button></div>
      <div class="pp-head">${lnAvatar(p, "120")}
        <div class="pp-name"><h1>${esc(p.name)}</h1>${p.tagline ? `<p class="pp-tag">${esc(p.tagline)}</p>` : ""}${p.bio ? `<p class="pp-bio">${esc(p.bio).replace(/\n/g, "<br>")}</p>` : ""}</div>
        <div class="pp-actions">${last ? `<button class="ln-button ln-button--play ln-button--lg" data-action="play-as" data-game="${esc(last.it.id)}" data-player="${esc(p.id)}">${GI.play}Play ${esc(last.it.name)}</button>` : ""}
          <button class="ln-button ln-button--secondary" data-action="edit-profile" data-player="${esc(p.id)}">Edit profile</button>
          <button class="ln-button ln-button--ghost" data-action="export-player" data-player="${esc(p.id)}">Export profile</button></div></div>
      <div class="pp-body">${body || `<p class="hint">Nothing to show yet: <b>Edit profile</b> chooses what this page shows.</p>`}</div>
    </div>`;
  }

  function ppSection(s, p, st) {
    const sec = (title, inner, extra) => `<section class="pp-section pp-${s}"><div class="ln-section-head"><div><h2>${title}</h2></div>${extra || ""}</div>${inner}</section>`;
    if (s === "now") {
      const it = p.now && itemByKey(p.now);
      if (!it) return "";
      loadArt(it); ensureCover(it);
      const t = (playtimeOf(it) || {})[p.id] || 0;
      return `<section class="pp-section pp-now"><button class="now-card" data-action="open-sheet" ${itemAttr(it)}>
        ${artHTML(it)}<span class="now-text"><span class="now-label">Currently playing</span><b>${esc(it.name)}</b>
        ${p.nowNote ? `<span class="now-note">${esc(p.nowNote)}</span>` : ""}
        <span class="now-meta">${t ? `<span>${esc(hours(t))} played</span>` : ""}</span></span></button></section>`;
    }
    if (s === "stats") {
      const stat = (v, l) => `<div class="pstat"><b>${v}</b><span>${l}</span></div>`;
      return `<section class="pp-section pp-stats">
        ${stat(st.total ? esc(hours(st.total)) : "None yet", "played with Lootnest")}
        ${stat(String(st.played), st.played === 1 ? "game played" : "games played")}
        ${stat(String(st.profiles), st.profiles === 1 ? "game with its saves" : "games with its saves")}
        ${stat(st.streak ? st.streak + (st.streak === 1 ? " day" : " days") : "None", st.best > st.streak ? `in a row now, best ${st.best}` : "in a row")}
        ${stat(st.longest ? esc(hours(st.longest.seconds)) : "None yet", st.longest && itemByKey(st.longest.key) ? "longest session, " + esc(itemByKey(st.longest.key).name) : "longest session")}
        ${stat(st.top ? esc(st.top.name) : "None yet", st.top ? esc(hours(st.topT)) + ", most played" : "most played")}
      </section>`;
    }
    if (s === "favorites") {
      const favs = (p.favorites || []).map(itemByKey).filter(Boolean);
      favs.forEach(ensureCover);
      return sec("Favourite games", favs.length ? `<div class="lib-grid size-l">${favs.map((it) => cardHTML(it)).join("")}</div>`
        : `<p class="hint">Pick up to six games to show here: <b>Edit profile</b>, then <b>Showcase</b>.</p>`);
    }
    if (s === "activity") return sec("Activity", activityHTML(p, st));
    if (s === "collection") {
      const items = libraryItems(), by = {};
      items.forEach((it) => {
        const t = (playtimeOf(it) || {})[p.id] || 0;
        if (!t && !st.recent.some((x) => x.it === it)) return;
        const l = sourceLabel(it.source);
        by[l] = by[l] || { n: 0, t: 0 }; by[l].n++; by[l].t += t;
      });
      const rows = Object.entries(by).sort((a, b) => b[1].n - a[1].n || b[1].t - a[1].t);
      const max = Math.max(1, ...rows.map((r) => r[1].n));
      return sec("Collection", rows.length ? `<ul class="bars">${rows.map(([l, v]) => `<li><span class="bl">${esc(l)}</span>
          <span class="bt"><span style="width:${Math.max(4, (v.n / max) * 100)}%"></span></span><span class="bv">${v.n} ${v.n === 1 ? "game" : "games"}${v.t ? ", " + esc(hours(v.t)) : ""}</span></li>`).join("")}</ul>
          <p class="hint">${items.length} games in the Library on this PC, from ${new Set(items.map((it) => sourceLabel(it.source))).size} places.</p>`
        : `<p class="hint">Games you play show up here, by where they come from.</p>`);
    }
    if (s === "recent") {
      if (!st.recent.length) return "";
      return sec("Played lately", `<ul class="pp-recent">${st.recent.slice(0, 8).map(({ it, r }) => `<li><button data-action="open-sheet" ${itemAttr(it)}><b>${esc(it.name)}</b><span>${esc(ago(r.at))}</span></button></li>`).join("")}</ul>`);
    }
    return "";
  }

  // 26 weeks of play, a column a week, a square a day, darker for more.
  function activityHTML(p, st) {
    const days = daysOf(p.id), today = new Date();
    const start = new Date(today); start.setHours(12, 0, 0, 0);
    start.setDate(start.getDate() - ((start.getDay() + 6) % 7) - 25 * 7);
    const lvl = (s) => (s <= 0 ? 0 : s < 1800 ? 1 : s < 3600 ? 2 : s < 7200 ? 3 : 4);
    let cells = "", week = 0, month = 0, months = "";
    const tKey = dayKey(today), byGame = {};
    for (let i = 0, d = new Date(start); i < 26 * 7; i++, d.setDate(d.getDate() + 1)) {
      const k = dayKey(d), sum = daySum(days[k]), future = k > tKey;
      if (i % 7 === 0 && d.getDate() <= 7) months += `<span style="grid-column:${i / 7 + 1}">${d.toLocaleDateString(undefined, { month: "short" })}</span>`;
      cells += `<span class="hm l${future ? "x" : lvl(sum)}" style="grid-column:${Math.floor(i / 7) + 1};grid-row:${(i % 7) + 1}" title="${esc(d.toLocaleDateString(undefined, { weekday: "short", day: "numeric", month: "short" }))}${sum ? ": " + esc(hours(sum)) : ""}"></span>`;
    }
    for (let i = 0, d = new Date(today); i < 30; i++, d.setDate(d.getDate() - 1)) {
      const day = days[dayKey(d)] || {};
      if (i < 7) week += daySum(day);
      month += daySum(day);
      Object.entries(day).forEach(([k, v]) => { byGame[k] = (byGame[k] || 0) + v; });
    }
    const top = Object.entries(byGame).map(([k, v]) => ({ it: itemByKey(k), v })).filter((x) => x.it).sort((a, b) => b.v - a.v).slice(0, 4);
    const max = Math.max(1, ...top.map((x) => x.v));
    return `<div class="activity">
      <div class="heat-wrap"><div class="heat-months">${months}</div><div class="heat" aria-label="Time played per day">${cells}</div>
        <div class="heat-key"><span>Less</span><span class="hm l0"></span><span class="hm l1"></span><span class="hm l2"></span><span class="hm l3"></span><span class="hm l4"></span><span>More</span></div></div>
      <div class="act-side"><div class="act-n"><div><b>${week ? esc(hours(week)) : "0 min"}</b><span>the last 7 days</span></div><div><b>${month ? esc(hours(month)) : "0 min"}</b><span>the last 30 days</span></div><div><b>${st.sessions}</b><span>${st.sessions === 1 ? "session" : "sessions"} in all</span></div></div>
        ${top.length ? `<ul class="bars">${top.map(({ it, v }) => `<li><span class="bl">${esc(it.name)}</span><span class="bt"><span style="width:${Math.max(4, (v / max) * 100)}%"></span></span><span class="bv">${esc(hours(v))}</span></li>`).join("")}</ul>`
          : `<p class="hint">Games started from Lootnest are counted here, day by day.</p>`}</div></div>`;
  }

  // A game outside Steam without profiles: its saves, kept as versions in
  // the cloud folder (Steam games have Steam Cloud; games with profiles sync
  // each player's saves).
  const gsAsked = new Set();
  // Status and collections: where the game is for one profile (playing,
  // finished, 100%), and which of the user's collections it's in.
  function collectionsBlock(it) {
    const key = recentKey(it), ps = grouped();
    if (!ps.length) return "";
    const seat = seated(), owner = it.g && S.d.active[it.g.id];
    const who = (ps.find((p) => p.id === S.statusWho[key]) || (seat && ps.find((p) => p.id === seat.id)) || ps.find((p) => p.id === owner) || ps[0]).id;
    const st = statusOf(it, who), tags = tagsOf(it), others = allTags().filter((t) => !tags.includes(t));
    const chips = ps.length > 1 ? `<div class="ach-who" role="group" aria-label="Whose status">${ps.map((p) => `<button class="ln-chip ln-player" style="--player:${esc(p.color)}" data-action="status-who" data-key="${esc(key)}" data-player="${esc(p.id)}" aria-pressed="${p.id === who}">${lnAvatar(p, "20")}${esc(parentOf(p) ? runLabel(p) : p.name)}${statusOf(it, p.id) ? `<span class="ln-num">${esc(statusName(statusOf(it, p.id)))}</span>` : ""}</button>`).join("")}</div>` : "";
    return `<section class="gpage-block gp-coll"><div class="ln-section-head"><div><h2>Status</h2></div></div>${chips}
      <div class="ln-segmented gp-status" role="group" aria-label="${esc(possessive(pname(who)))} status">${STATUSES.map(([k, l]) => `<button data-action="set-status" data-key="${esc(key)}" data-player="${esc(who)}" data-status="${k}" aria-pressed="${st === k}">${l}</button>`).join("")}</div>
      <div class="ln-section-head ln-section-head--minor"><h2>Collections</h2></div>
      <div class="gp-tags">${tags.map((t) => `<span class="ln-tag gp-tag">${esc(t)}<button data-action="tag-remove" data-key="${esc(key)}" data-tag="${esc(t)}" aria-label="Take it out of ${esc(t)}">${GI.x}</button></span>`).join("") || `<span class="gpage-hint">In no collection yet.</span>`}</div>
      ${tags.length < 8 ? `<div class="gp-tag-add"><input id="tag-new" class="ln-input" type="text" maxlength="24" list="tag-list" autocomplete="off" placeholder="Add to a collection, such as Co-op" data-key="${esc(key)}">
        <datalist id="tag-list">${others.map((t) => `<option value="${esc(t)}"></option>`).join("")}</datalist>
        <button class="ln-button ln-button--secondary ln-button--sm" data-action="tag-add" data-key="${esc(key)}">Add</button></div>` : ""}</section>`;
  }

  // Achievements: read from this PC (Steam's library cache, or the file a
  // Steam-compatible tool keeps next to the game's saves). The latest
  // unlocked first; the whole list on request.
  const achAsked = new Set();
  // Whose achievements a game shows, and the pieces both the page and the
  // panel with the whole list are made of.
  function achFor(it) {
    const key = recentKey(it);
    if (!achAsked.has(key)) {
      achAsked.add(key);
      const dir = it.f ? it.f.dir : it.exe;
      if (it.appId || dir) api.achievements(key, it.appId, dir, it.name).catch(() => {});
    }
    const top = S.ach[key];
    if (!top || !top.total) return null;
    // Each player's own (they go with their profile): whose to show is the
    // one picked here, else whoever's at the PC, else whose files are in.
    const ids = top.players ? cfg().players.map((p) => p.id).filter((id) => top.players[id]) : [];
    const seat = seated(), owner = it.g && S.d.active[it.g.id];
    const who = ids.includes(S.achWho[key]) ? S.achWho[key] : seat && ids.includes(seat.id) ? seat.id : ids.includes(owner) ? owner : ids[0];
    const A = who ? top.players[who] : top;
    const pct = A.total ? Math.round((A.unlocked / A.total) * 100) : 0;
    const whoRow = ids.length ? `<div class="ach-who" role="group" aria-label="Whose achievements">${ids.map((id) => { const p = player(id); return `<button class="ln-chip ln-player" style="--player:${esc(p.color)}" data-action="ach-who" data-key="${esc(key)}" data-player="${esc(id)}" aria-pressed="${id === who}">${lnAvatar(p, "20")}${esc(p.name)}<span class="ln-num">${top.players[id].unlocked}</span></button>`; }).join("")}</div>` : "";
    const g = it.g;
    const note = A.source !== "pc" ? "From Steam, for the account signed in on this PC."
      : ids.length ? "Each profile has its own. They go with it when you swap, like its saves."
      : g && g.sharedAchievements ? "Shared by everyone who plays on this PC." : "Saved on this PC next to the game's saves.";
    const toggle = g && A.source === "pc" ? `<button class="ln-button ln-button--ghost ln-button--sm" data-action="ach-shared" data-game="${esc(g.id)}" data-on="${g.sharedAchievements ? "0" : "1"}">${g.sharedAchievements ? "Give each profile its own" : "Share one set"}</button>` : "";
    const bar = `<div class="ach-bar" role="progressbar" aria-valuemin="0" aria-valuemax="${A.total}" aria-valuenow="${A.unlocked}" aria-label="${A.unlocked} of ${A.total} achievements"><span style="width:${pct}%"></span></div>`;
    const row = (a) => `<li class="ach${a.unlocked ? "" : " locked"}">${a.icon ? `<img src="${esc(a.icon)}" alt="" loading="lazy">` : `<span class="ach-none">${GI.trophy}</span>`}
      <div><b>${a.hidden && !a.unlocked ? "Hidden achievement" : esc(a.name)}</b><span>${a.unlocked ? (a.at && new Date(a.at).getFullYear() > 2000 ? "Unlocked " + esc(ago(a.at)) : "Unlocked") : a.hidden ? "Keep playing to find out" : esc(a.description || "Not unlocked yet")}</span></div>
      ${a.percent ? `<span class="ach-pct ln-num" title="Players who have it">${a.percent < 10 ? a.percent.toFixed(1) : Math.round(a.percent)}%</span>` : ""}</li>`;
    return { key, who, A, pct, whoRow, note, toggle, bar, row };
  }
  // On the game's page: the latest unlocked, and the whole list in a panel.
  function achievementsBlock(it) {
    const X = achFor(it);
    if (!X) {
      // Read and none found: for a game from outside the stores, say why.
      if (S.ach[recentKey(it)] !== null || !docWorth(it)) return "";
      return `<section class="gpage-block gp-ach">${gpHead("Achievements")}<p>None found on this PC for ${esc(it.name)}.</p>
        <div class="gpage-actions"><button class="ln-button ln-button--secondary ln-button--sm" data-action="ach-doctor" ${itemAttr(it)} aria-haspopup="dialog">Check why</button></div></section>`;
    }
    const { key, who, A, pct } = X;
    const list = A.list.filter((a) => a.unlocked).slice(0, 6);
    return `<section class="gpage-block gp-ach"><div class="ln-section-head"><div><h2>${who ? esc(possessive(pname(who))) + " achievements" : "Achievements"}</h2></div>
        <span class="ach-count"><b class="ln-num">${A.unlocked}</b> of <span class="ln-num">${A.total}</span>, ${pct}%</span></div>
      ${X.whoRow}${X.bar}
      ${list.length ? `<ul class="ach-list ach-grid">${list.map(X.row).join("")}</ul>` : `<p>None unlocked yet.</p>`}
      <div class="gp-ach-foot"><p class="gpage-hint">${X.note}</p><div class="gpage-actions"><button class="ln-button ln-button--secondary ln-button--sm" data-action="ach-all" ${itemAttr(it)} aria-haspopup="dialog">Show all ${A.total}</button>${docWorth(it) ? `<button class="ln-button ln-button--ghost ln-button--sm" data-action="ach-doctor" ${itemAttr(it)} aria-haspopup="dialog">Check</button>` : ""}${X.toggle}</div></div></section>`;
  }
  // Save points: named copies of a game's saves, on a timeline.
  const pointsAsked = new Set();
  function pointsFor(it) {
    const key = recentKey(it);
    if (!pointsAsked.has(key)) { pointsAsked.add(key); api.savePoints(key).catch(() => {}); }
    return S.points[key];
  }
  function pointRow(it, p, full, P) {
    const when = new Date(p.created).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
    const facts = [esc(when)];
    if (p.playtime) facts.push(`${esc(hours(p.playtime))} played`);
    if (p.unlocked) facts.push(`${p.unlocked} achievement${p.unlocked === 1 ? "" : "s"}`);
    const who = p.player && player(p.player) ? `<span class="ln-mark ln-player" style="--player:${esc(color(p.player))}"></span>${esc(pname(p.player))}` : "";
    const attrs = `${itemAttr(it)} data-id="${esc(p.id)}" data-name="${esc(p.name)}"`;
    const b = (cls, a, label) => `<button class="ln-button ${cls} ln-button--sm" ${a} ${P.busy ? "disabled" : ""}>${label}</button>`;
    const editing = full && S.dialog && S.dialog.editing === p.id;
    const name = editing ? `<input class="ln-input pt-name-in" id="pt-rename" maxlength="60" value="${esc(p.name)}" aria-label="Name">` : `<b>${esc(p.name)}</b>`;
    const actions = editing ? b("ln-button--secondary", `data-action="pt-rename-save" ${attrs}`, "Save") + b("ln-button--ghost", 'data-action="pt-rename-cancel"', "Cancel")
      : b("ln-button--secondary", `data-action="pt-restore" ${attrs}`, "Go back to it")
        + (full && P.profiles ? b("ln-button--ghost", `data-action="pt-branch" ${attrs} data-player="${esc(p.player || "")}"`, "New run from it") : "")
        + (full ? b("ln-button--ghost", `data-action="pt-rename" ${attrs}`, "Rename") + b("ln-button--ghost", `data-action="pt-delete" ${attrs}`, "Delete") : "");
    return `<li class="pt-row${p.auto ? " pt-row--auto" : ""}"><span class="pt-dot" aria-hidden="true"></span><div class="pt-main">${name}
      <span class="pt-facts">${who ? `<span>${who}</span>` : ""}${facts.map((x) => `<span>${x}</span>`).join("")}${p.auto ? '<span>Kept before going back</span>' : ""}</span></div><div class="pt-actions">${actions}</div></li>`;
  }
  function savePointsBlock(it) {
    const P = pointsFor(it);
    if (!P || !P.loaded || P.error) return "";
    const list = P.points || [];
    const make = `<div class="pt-make"><input class="ln-input" id="pt-new" maxlength="60" placeholder="Name it, such as Before the boss" aria-label="Save point name" ${P.busy ? "disabled" : ""}>
      <button class="ln-button ln-button--secondary ln-button--sm" data-action="pt-make" ${itemAttr(it)} ${P.busy ? "disabled" : ""}>${P.busy ? '<span class="ln-spinner" aria-hidden="true"></span>Copying' : "Make a save point"}</button></div>`;
    const hint = cfg().settings.pointHotkeyOff ? "" : " Ctrl+Shift+F5 makes one while you play.";
    return `<section class="gpage-block gp-points"><div class="ln-section-head"><div><h2>Save points</h2></div>${list.length ? `<span class="ach-count"><b class="ln-num">${list.length}</b> kept</span>` : ""}</div>
      <p class="gpage-hint">A copy of the saves as they are now, to come back to.${hint}</p>${make}
      ${list.length ? `<ol class="pt-list">${list.slice(0, 3).map((p) => pointRow(it, p, false, P)).join("")}</ol>` : ""}
      ${list.length ? `<div class="gpage-actions"><button class="ln-button ln-button--ghost ln-button--sm" data-action="pt-all" ${itemAttr(it)} aria-haspopup="dialog">${list.length > 3 ? `Show all ${list.length}` : "Rename, delete or start a run"}</button></div>` : ""}</section>`;
  }
  function viewPointsDrawer() {
    const it = dialogItem(S.dialog);
    if (!it) return "";
    const P = S.points[recentKey(it)] || {};
    const list = P.points || [];
    return drawer({ label: `${esc(it.name)} save points`, title: "Save points", sub: `${esc(it.name)}, newest first`,
      body: list.length ? `<ol class="pt-list pt-list--all">${list.map((p) => pointRow(it, p, true, P)).join("")}</ol>` : `<p>No save points yet.</p>`,
      foot: `<p class="gpage-hint ln-drawer__note">Going back keeps the saves as they are now as a point first.${P.profiles ? " A new run is a profile of its own, starting from that moment." : ""}</p>` });
  }
  function gameSavesBlock(it) {
    if (it.kind !== "found" || it.source === "steam" || it.appId) return "";
    const head = gpHead("Cloud save");
    const G = S.gsaves[it.id];
    if (!G) { if (!gsAsked.has(it.id)) { gsAsked.add(it.id); api.gameSaves(it.id).catch(() => {}); } return `<section class="gpage-block">${head}${lnInline("", "Looking for its saves")}</section>`; }
    if (G.off === "steam") return "";
    if (G.error) return `<section class="gpage-block">${head}${lnInline("error", esc(G.error))}</section>`;
    const btn = (cls, attrs, label) => `<button class="ln-button ${cls} ln-button--sm" ${attrs}>${label}</button>`;
    const when = (v) => esc(new Date(v.created).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" }));
    // The Save Safety Net's copies of saves kept inside the game's folder.
    const lv = G.local || [];
    const local = lv.length ? `<div class="gpage-sub">${gpHead("Copies kept by Lootnest")}<dl class="ln-facts">${lv.map((v, i) => `<div><dt><span><b>${when(v)}</b>
        <span class="gpage-hint">${esc(fmtBytes(v.size))}${i === 0 ? ", newest" : ""}</span></span></dt>
        <dd>${btn("ln-button--ghost", `data-action="gs-restore" data-local="1" data-key="${esc(it.id)}" data-id="${esc(v.id)}" data-when="${esc(v.created)}" ${G.busy ? "disabled" : ""}`, "Restore")}</dd></div>`).join("")}</dl>
      <p class="gpage-hint">${esc(it.name)} keeps saves inside its own folder, so reinstalling or updating it could take them. Lootnest copies them here when they change and the game is closed.</p></div>` : "";
    if (G.off === "nocloud") return `<section class="gpage-block">${head}<p>Keep ${esc(possessive(it.name))} saves in your own cloud, with every version to go back to.</p>
      <div class="gpage-actions"><button class="ln-button ln-button--secondary ln-button--sm" data-action="nav" data-view="cloud">Set up Cloud saves</button></div>${local}</section>`;
    if (G.off === "locked") return `<section class="gpage-block">${head}<p>The cloud folder has a password. Enter it on <b>Cloud saves</b>.</p>${local}</section>`;
    const s = cfg().settings;
    const choose = `<div class="gpage-actions">${btn("ln-button--secondary", `data-action="gs-add" data-key="${esc(it.id)}" data-kind="folder"`, G.locations.length ? "Add a folder" : "Choose the save folder")}
      ${btn("ln-button--ghost", `data-action="gs-add" data-key="${esc(it.id)}" data-kind="file"`, G.locations.length ? "Add a file" : "Choose a save file")}
      ${btn("ln-button--ghost", `data-action="gs-pcgw" data-name="${esc(it.name)}"`, "Look it up on PCGamingWiki")}</div>`;
    let where;
    if (!G.locations.length) {
      where = `<p>${G.source === "you" ? `Choose where ${esc(it.name)} keeps its saves.` : `Lootnest doesn't know where ${esc(it.name)} keeps its saves yet. Choose the folder, and its saves go to the cloud from then on.`}</p>
        <p class="gpage-hint">Usually in Documents, Saved Games, or AppData (Roaming, Local or LocalLow), in a folder named after the game or its studio. PCGamingWiki's page for the game says where.</p>
        ${choose}
        ${s.saveDb ? "" : `<p class="gpage-hint">Or turn on <b>Find save locations online</b>: Lootnest then uses PCGamingWiki's list of where over 20,000 games keep their saves. <button class="gpage-link" data-action="savedb-on">Turn it on</button></p>`}`;
    } else {
      const yours = G.source === "you";
      where = `${gpHead("On this PC")}<dl class="ln-facts gpage-paths">${G.locations.map((l) => `<div><dt class="selectable" title="${esc(l.resolved)}">${esc(l.path)}</dt><dd><span class="ln-num">${esc(fmtBytes(l.size || 0))}</span>
          ${btn("ln-button--ghost", `data-action="open-path" data-path="${esc(l.path)}"`, "Open")}${yours ? btn("ln-button--ghost", `data-action="gs-remove" data-key="${esc(it.id)}" data-path="${esc(l.path)}"`, "Remove") : ""}</dd></div>`).join("")}</dl>
        <p class="gpage-hint">${yours ? "Chosen by you." : G.source === "list" ? "Found with PCGamingWiki's list of save locations." : "Folders named like the game."}
          ${yours ? "" : `<button class="gpage-link" data-action="gs-own" data-key="${esc(it.id)}">Choose them yourself</button>`}</p>
        ${yours ? choose : ""}`;
    }
    const vs = G.versions || [];
    const inCloud = G.cloudDir ? `<div class="gpage-sub">${gpHead("In " + esc(G.provider || "your cloud folder"))}
      <dl class="ln-facts gpage-paths"><div><dt class="selectable">${esc(G.cloudDir)}</dt><dd>${btn("ln-button--ghost", `data-action="gs-open" data-key="${esc(it.id)}"`, "Open folder")}</dd></div></dl></div>` : "";
    const list = G.locations.length || vs.length ? `<div class="gpage-sub">${gpHead("Versions")}${vs.length ? `<dl class="ln-facts">${vs.map((v, i) => `<div><dt><span><b>${esc(new Date(v.created).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" }))}</b>
        <span class="gpage-hint">${esc(v.computer)}, ${esc(fmtBytes(v.size))}${i === 0 ? ", newest" : ""}</span></span></dt>
        <dd>${btn("ln-button--ghost", `data-action="gs-restore" data-key="${esc(it.id)}" data-id="${esc(v.id)}" data-when="${esc(v.created)}" data-pc="${esc(v.computer)}" ${G.busy ? "disabled" : ""}`, "Restore")}</dd></div>`).join("")}</dl>`
        : `<p class="gpage-hint">No versions yet.${G.off === "disabled" ? "" : " One is made when the saves change and the game is closed."}</p>`}
      ${G.off === "disabled" ? `<p class="gpage-hint">Automatic versions are off on <b>Cloud saves</b>.</p>` : ""}
      ${G.locations.length ? `<div class="gpage-actions">${G.busy ? `<button class="ln-button ln-button--secondary ln-button--sm" aria-busy="true"><span class="ln-spinner" aria-hidden="true"></span>Saving</button>` : btn("ln-button--secondary", `data-action="gs-now" data-key="${esc(it.id)}"`, "Back up now")}</div>` : ""}</div>` : "";
    return `<section class="gpage-block">${head}${where}${inCloud}${list}${local}</section>`;
  }

  // "Saved yesterday, 24 MB": when the game last wrote this player's
  // profile, and how much it holds.
  function saved(g, pid) {
    const i = g ? profileInfo(g.id, pid) : {};
    return i.saved ? `<span class="gp-person-saved">Saved ${esc(ago(i.saved))}, ${esc(fmtBytes(i.size))}</span>` : "";
  }
  // Everyone who plays this game: their time, when they played, and whose
  // files are loaded.
  function playersPanel(it) {
    const pt = playtimeOf(it);
    const g = it.g, owner = g && S.d.active[g.id];
    const ps = cfg().players;
    if (!ps.length) return "";
    const total = ps.reduce((a, p) => a + (pt[p.id] || 0), 0);
    const split = total ? `<div class="gp-split" aria-hidden="true">${ps.filter((p) => pt[p.id]).map((p) => `<span class="ln-player" style="--player:${esc(p.color)};flex:${pt[p.id]}"></span>`).join("")}</div>` : "";
    return `<section class="gpage-block gp-players"><div class="ln-section-head"><div><h2>Profiles</h2></div></div>${split}<div class="gp-people">${ps.map((p) => {
      const sub = [pt[p.id] ? `${hours(pt[p.id])} played` : g && hasProfile(g.id, p.id) ? "Has saves here" : g ? "Not played yet" : "", lastBy(it, p.id) ? `last ${ago(lastBy(it, p.id))}` : ""].filter(Boolean).join(", ");
      return `<div class="gp-person ln-player" style="--player:${esc(p.color)}">
        <div class="gp-person-top">${lnAvatar(p, "32")}<div class="gp-person-name"><b>${esc(p.name)}${owner === p.id ? ' <span class="ln-tag">Files loaded</span>' : ""}</b><span>${esc(sub || "Hasn't played yet")}</span>${saved(g, p.id)}</div>
</div></div>`;
    }).join("")}</div></section>`;
  }
  function viewProfileEditor() {
    const D = S.dialog, p = D.p, tab = D.tab || "look";
    const tabs = [["look", "Look"], ["about", "About"], ["showcase", "Showcase"], ["page", "Page"]];
    const gameOptions = (sel, none) => `<option value="">${none}</option>${libraryItems().map((it) => `<option value="${esc(recentKey(it))}" ${sel === recentKey(it) ? "selected" : ""}>${esc(it.name)}</option>`).join("")}`;
    let body = "";
    if (tab === "look") {
      const hasPic = !!(S.d.avatars || {})[p.id], hasBanner = !!(S.d.banners || {})[p.id];
      const kind = p.banner === "image" ? "image" : p.banner && !p.banner.startsWith("style:") ? "game" : "plain";
      body = `
      <div class="field"><span class="label">Picture</span><div class="pe-row">
        <button class="btn" data-action="avatar-pick" ${D.busy ? "disabled" : ""}>${D.busy === "avatar" ? "Reading the picture" : hasPic ? "Change picture" : "Choose a picture"}</button>
        ${hasPic ? `<button class="btn quiet" data-action="avatar-remove">Remove picture</button>` : `<select data-pe="avatarGame" aria-label="Or a game's cover">${gameOptions(p.avatarGame || "", "Or use a game's cover")}</select>`}</div></div>
      <div class="field"><span class="label">Colour</span><div class="pe-row"><div class="swatches">${S.d.colors.map((c) => `<button class="swatch" style="background:${c}" data-action="pe-color" data-color="${c}" aria-pressed="${c.toLowerCase() === (p.color || "").toLowerCase()}" aria-label="${c}"></button>`).join("")}</div>
        <label class="custom-color"><input type="color" value="${esc((p.color || "#6c9bd2").toLowerCase())}" data-pe="color" aria-label="Any colour"><span>Any colour</span></label></div></div>
      <div class="field"><span class="label">Banner</span>
        <div class="segt">${[["plain", "Colour"], ["image", "Your picture"], ["game", "Game art"]].map(([v, l]) => `<button data-action="pe-banner" data-v="${v}" aria-pressed="${kind === v}">${l}</button>`).join("")}</div>
        ${kind === "game" ? `<select data-pe="banner" aria-label="Game">${gameOptions(p.banner, "Choose a game")}</select>` : ""}
        ${kind === "image" ? `<div class="pe-row"><button class="btn" data-action="banner-pick" ${D.busy ? "disabled" : ""}>${D.busy === "banner" ? "Reading the picture" : "Choose another picture"}</button><button class="btn quiet" data-action="banner-remove">Remove picture</button></div>` : ""}
        ${kind === "image" || kind === "game" ? `<label class="range"><span>Position</span><input type="range" min="0" max="100" step="1" value="${p.bannerY || 0}" data-pe="bannerY"></label>` : ""}
        <span class="hint">A picture is cropped to a wide strip (3:1) and kept on this PC.</span></div>`;
    } else if (tab === "about") {
      body = `
      <div class="field"><label for="pe-name">Name</label><input id="pe-name" type="text" maxlength="24" value="${esc(p.name)}" data-pe="name"></div>
      <div class="field"><label for="pe-tag">Tagline</label><input id="pe-tag" type="text" maxlength="80" value="${esc(p.tagline || "")}" placeholder="Optional" data-pe="tagline"></div>
      <div class="field"><label for="pe-bio">About <span class="hint">up to 300 characters</span></label><textarea id="pe-bio" rows="4" maxlength="300" placeholder="Optional" data-pe="bio">${esc(p.bio || "")}</textarea></div>
      <div class="field"><label for="pe-now">Currently playing</label><select id="pe-now" data-pe="now">${gameOptions(p.now || "", "None")}</select>
        <input type="text" maxlength="80" value="${esc(p.nowNote || "")}" placeholder="A note (optional)" data-pe="nowNote" aria-label="A word on it" style="margin-top:8px"></div>`;
    } else if (tab === "showcase") {
      const q = (D.q || "").toLowerCase(), fav = p.favorites || [];
      const games = libraryItems().filter((it) => !q || it.name.toLowerCase().includes(q));
      body = `
      <div class="field"><span class="label">Favourite games <span class="hint">${fav.length} of 6, in this order</span></span>
        ${fav.length ? `<ol class="fav-order">${fav.map((k, i) => { const it = itemByKey(k); return `<li><span>${esc(it ? it.name : "A game no longer here")}</span>
          <button class="btn quiet icon" data-action="pe-fav-move" data-i="${i}" data-d="-1" aria-label="Earlier" ${i === 0 ? "disabled" : ""}>↑</button>
          <button class="btn quiet icon" data-action="pe-fav-move" data-i="${i}" data-d="1" aria-label="Later" ${i === fav.length - 1 ? "disabled" : ""}>↓</button>
          <button class="btn quiet icon" data-action="pe-fav" data-k="${esc(k)}" aria-label="Remove">${icon.x}</button></li>`; }).join("")}</ol>` : ""}
        <input type="search" id="pe-q" placeholder="Find a game" value="${esc(D.q || "")}" autocomplete="off">
        <div class="pe-games">${games.map((it) => { const k = recentKey(it), on = fav.includes(k); return `<button class="pe-game${on ? " on" : ""}" data-action="pe-fav" data-k="${esc(k)}" aria-pressed="${on}" ${!on && fav.length >= 6 ? "disabled" : ""}><span class="check"></span>${esc(it.name)}</button>`; }).join("")}</div></div>`;
    } else {
      const on = sectionsOf(p);
      const order = [...on, ...SECTIONS.map((s) => s[0]).filter((s) => !on.includes(s))];
      body = `<p class="sub" style="margin:0">What your page shows, top to bottom.</p><ul class="sec-list">${order.map((k) => {
        const s = SECTIONS.find((x) => x[0] === k), i = on.indexOf(k), shown = i >= 0;
        return `<li class="${shown ? "" : "off"}"><button class="sec-toggle" data-action="pe-sec" data-k="${k}" aria-pressed="${shown}" ${shown && on.length === 1 ? "disabled" : ""}><span class="check"></span><span><b>${s[1]}</b><span>${s[2]}</span></span></button>
          ${shown ? `<button class="btn quiet icon" data-action="pe-sec-move" data-k="${k}" data-d="-1" aria-label="Higher" ${i === 0 ? "disabled" : ""}>↑</button><button class="btn quiet icon" data-action="pe-sec-move" data-k="${k}" data-d="1" aria-label="Lower" ${i === on.length - 1 ? "disabled" : ""}>↓</button>` : ""}</li>`;
      }).join("")}</ul>`;
    }
    return `<div class="scrim" data-action="scrim"><div class="dialog pe" role="dialog" aria-modal="true" aria-label="Edit ${esc(p.name)}'s profile" style="--c:${esc(p.color)}">
      <div class="pe-preview"><div class="pe-pbanner">${bannerHTML(p)}</div>${avatarHTML(p, "lg")}
        <div class="pe-pname"><b id="pe-prev-name">${esc(p.name)}</b><span id="pe-prev-tag">${esc(p.tagline || "")}</span></div>
        <button class="btn quiet icon" data-action="dialog-cancel" aria-label="Close">${icon.x}</button></div>
      <div class="segt pe-tabs" role="tablist">${tabs.map(([k, l]) => `<button role="tab" aria-selected="${k === tab}" data-action="pe-tab" data-tab="${k}">${l}</button>`).join("")}</div>
      <div class="pe-body">${body}</div>
      <div class="dialog-foot"><span class="error-text grow" id="pe-error"></span><button class="btn quiet" data-action="dialog-cancel">Cancel</button><button class="btn primary" data-action="pe-save">Save profile</button></div>
    </div></div>`;
  }

  // Players: the household. A card per person (their banner, what they
  // play, their numbers), then their names and colours.
  function viewPlayers() {
    const ps = S.players || (S.players = JSON.parse(JSON.stringify(cfg().players)));
    const dirty = JSON.stringify(ps) !== JSON.stringify(cfg().players);
    const card = (p) => {
      const st = playerStats(p);
      const last = st.recent[0];
      // Their games: the latest first, then the ones they've played or have
      // saves in, so a card is never empty while they have games.
      const seen = new Set(), mine = [...st.recent.map((x) => x.it),
        ...libraryItems().filter((it) => playtimeOf(it)[p.id]).sort((a, b) => playtimeOf(b)[p.id] - playtimeOf(a)[p.id]),
        ...libraryItems().filter((it) => it.kind === "game" && hasProfile(it.id, p.id))]
        .filter((it) => !seen.has(recentKey(it)) && seen.add(recentKey(it))).slice(0, 8);
      const covers = mine.map((it) => { ensureCover(it); const c = S.covers[coverKey(it)]; return `<span class="pcard-cover" title="${esc(it.name)}"${c ? ` style="background-image:url('${c}')"` : ""}></span>`; }).join("");
      return `<button class="pcard ln-player" data-action="open-player" data-player="${esc(p.id)}" style="--player:${esc(p.color)}">
        <span class="pcard-banner">${bannerHTML(p)}</span>
        <span class="pcard-who">${lnAvatar(p, "72")}<span class="pcard-name"><b>${esc(p.name)}</b><span>${p.tagline ? esc(p.tagline) : last ? `${esc(last.it.name)}, ${esc(ago(last.r.at))}` : st.top ? `Most played: ${esc(st.top.name)}` : "No games played yet"}</span></span></span>
        ${covers ? `<span class="pcard-covers" aria-hidden="true">${covers}</span>` : ""}
        <span class="pcard-nums"><span><b class="ln-num">${st.total ? esc(hours(st.total)) : "0 h"}</b>played</span><span><b class="ln-num">${st.played}</b>${st.played === 1 ? "game" : "games"}</span><span><b class="ln-num">${st.profiles}</b>with saves</span></span></button>`;
    };
    // Each person, with their runs listed under their card.
    const people = cfg().players.filter((p) => !parentOf(p));
    const cards = people.map((p) => {
      const runs = cfg().players.filter((r) => parentOf(r) === p);
      return `<div class="pgroup">${card(p)}${runs.length ? `<ul class="pruns" aria-label="${esc(possessive(p.name))} runs">${runs.map((r) => { const st = playerStats(r); return `<li><button class="prun ln-player" data-action="open-player" data-player="${esc(r.id)}" style="--player:${esc(r.color)}">${lnAvatar(r, "32")}<span><b>${esc(runLabel(r))}</b><span>Run, ${st.total ? esc(hours(st.total)) : "0 h"} played</span></span></button></li>`; }).join("")}</ul>` : ""}</div>`;
    }).join("");
    const n = cfg().players.length;
    return `<div class="players-page ln-root">
      <header class="lib-head"><div><h1>Profiles</h1><p>Each profile has its own settings, saves and achievements in every game. Make one for everyone who plays, or a few for yourself, one per run.</p></div>
        <div class="lib-actions"><button class="ln-button ln-button--ghost" data-action="restore-pick" data-mode="import" title="A profile someone exported from Lootnest on another PC: their saves, picture and play time">Import a profile</button>
          <button class="ln-button ln-button--secondary" data-action="edit-profiles" aria-haspopup="dialog">${GI.edit}Names and colours${dirty ? '<span class="ln-tag">Not saved</span>' : ""}</button>
          ${ps.length < MAX_PROFILES ? `<button class="ln-button ln-button--primary" data-action="new-profile" aria-haspopup="dialog">${GI.plus}New profile</button>` : ""}</div></header>
      <p class="players-count">${n} of ${MAX_PROFILES} profiles. Open one to change its picture, banner and page.</p>
      <div class="pcards${people.length === 1 ? " one" : ""}">${cards}</div></div>`;
  }
  // Names and colours, in a panel: every profile at once, saved together.
  function viewProfilesDrawer() {
    const ps = S.players || (S.players = JSON.parse(JSON.stringify(cfg().players)));
    const dirty = JSON.stringify(ps) !== JSON.stringify(cfg().players);
    return drawer({ label: "Names and colours", title: "Names and colours", wide: true,
      sub: "A profile's colour marks its files, its Play button and its games everywhere in Lootnest. A run takes its person's colour.",
      body: `<div class="players">${grouped(ps).map((p) => playerRow(p, ps.indexOf(p), ps.length, false)).join("")}</div>
        ${ps.length < MAX_PROFILES ? `<button class="ln-button ln-button--secondary" data-action="add-player">${GI.plus}Add profile</button>` : `<p class="gpage-hint">Lootnest keeps up to ${MAX_PROFILES} profiles.</p>`}`,
      foot: `<span class="error-text players-grow" id="players-error"></span>
        ${dirty ? `<button class="ln-button ln-button--ghost" data-action="reset-players">Discard changes</button>` : ""}
        <button class="ln-button ln-button--primary" data-action="save-players" ${dirty ? "" : "disabled"}>Save profiles</button>` });
  }

  async function savePlayers(welcome) {
    const ps = S.players.map((p, i) => ({ ...p, name: (p.name || "").trim() || (welcome ? "Player " + (i + 1) : "") }));
    const removed = cfg().players.filter((o) => !ps.some((p) => p.id === o.id));
    const inDrawer = S.dialog && S.dialog.kind === "profiles";
    if (removed.length && !welcome) {
      const ok = await confirmDialog({ title: `Remove ${removed.map((p) => p.name).join(" and ")}?`, body: "Their profiles are moved to the “removed” folder inside Lootnest's data folder, so nothing is deleted for good.", ok: "Remove" });
      // Back to the panel either way, with the changes still in it.
      if (inDrawer) { S.dialog = { kind: "profiles" }; renderOverlay(); }
      if (!ok) return;
    }
    try {
      await api.savePlayers(ps);
      S.players = null;
      if (welcome || inDrawer) S.dialog = null;
      await refresh();
      if (!welcome) toast("Profiles saved");
    } catch (e) { const t = $("#players-error"); if (t) t.textContent = errText(e); }
  }

  // -------------------------------------------------------------- settings --
  // invert: the setting turns the thing off (on by default), so the switch
  // shows the opposite.
  function sw(field, title, desc, checked, invert) {
    return `<label class="toggle"><span><b style="font-weight:500">${title}</b><p>${desc}</p></span>
      <span class="switch"><input type="checkbox" data-setting="${field}"${invert ? ' data-invert="1"' : ""} ${checked ? "checked" : ""}><span></span></span></label>`;
  }

  // Settings: plain on purpose. Groups of rows, each saying what it does,
  // with its control on the right. Everything online is off until turned on.
  function viewSettings() {
    const s = cfg().settings, st = S.d.steam;
    const row = (forId, title, desc, control) => `<div class="bk-row"><div><label for="${forId}"><b>${title}</b></label>${desc ? `<p>${desc}</p>` : ""}</div>${control}</div>`;
    const b = (kind, attrs, label) => `<button class="ln-button ln-button--${kind} ln-button--sm" ${attrs}>${label}</button>`;
    const update = S.update.available
      ? (S.update.step ? `<button class="ln-button ln-button--primary ln-button--sm" aria-busy="true"><span class="ln-spinner" aria-hidden="true"></span>Updating</button>` : b("primary", 'data-action="update-now"', `Update to ${esc(S.update.version)}`))
      : (S.update.checking ? `<button class="ln-button ln-button--secondary ln-button--sm" aria-busy="true"><span class="ln-spinner" aria-hidden="true"></span>Checking</button>` : b("secondary", 'data-action="update-check"', "Check for updates"));
    const card = (rows) => `<div class="bk-card">${rows}</div>`;
    const sub = (title, note, rows) => `<section class="set-sub"><div class="ln-section-head ln-section-head--minor"><h2>${title}</h2></div>${note ? `<p class="set-note">${note}</p>` : ""}${card(rows)}</section>`;
    const versionRow = `<div class="bk-row set-version"><div><b>Lootnest ${esc(S.d.version)}${S.d.stage ? " " + esc(S.d.stage) : ""}</b><p>${S.update.available ? `Version ${esc(S.update.version)} is ready to install.` : "Updates come from Lootnest's releases on GitHub."}</p></div>${update}</div>`;
    // One category at a time, picked on the left, as in Steam's settings.
    const tabs = [
      ["general", "General", GI.sliders, "How Lootnest starts, closes and tells you things.", () => card(`
          ${sw("autostart", "Start with Windows", "Lootnest opens quietly in the tray when you sign in, so it's always watching your games.", S.d.autostart)}
          ${sw("closeToTray", "Keep running when the window is closed", "Closing the window leaves Lootnest in the tray. Quit from the tray icon.", s.closeToTray)}
          ${sw("hideWhilePlaying", "Hide while a game runs", "The window gets out of the way once the game starts.", s.hideWhilePlaying)}
          ${sw("notifications", "Notifications", "A short Windows notification when progress is saved.", s.notifications)}`)],
      ["games", "Games and swaps", GI.play, "How profiles' files are swapped, and how Lootnest works with Steam.", () => sub("Swaps", "", `
          ${row("s-backups", "Copies kept per game before each swap", "A copy is made before every swap. The oldest go once there are more than this.", `<input id="s-backups" class="ln-input set-num" type="number" min="1" max="100" data-setting="backupsToKeep" value="${s.backupsToKeep}">`)}
          ${row("s-timeout", "Seconds to wait for a game to start", "If it hasn't started by then, the profile's files stay loaded, and Lootnest still saves them if the game starts later.", `<input id="s-timeout" class="ln-input set-num" type="number" min="30" max="1800" data-setting="startTimeoutSeconds" value="${s.startTimeoutSeconds}">`)}
          ${sw("watchOutsideLaunches", "Follow games started from Steam or elsewhere", "If a game starts without Lootnest, its progress is saved to whoever's files are loaded.", s.watchOutsideLaunches)}`)
        + sub("While you play", "", `
          ${sw("achToastsOff", "Show achievements as they unlock", "For games whose achievements a Steam-compatible tool keeps on this PC, a small note shows over the game when one unlocks, in the colour of the profile playing. It's a window of Lootnest's own: nothing goes into the game. It shows over windowed and borderless games, and over most fullscreen ones; switch a game to borderless if it doesn't. Never for games with an anti-cheat.", !s.achToastsOff, true)}
          ${sw("pointHotkeyOff", "Ctrl+Shift+F5 makes a save point", "While a game runs, the keys copy its saves as a save point, to go back to from its page. Lootnest holds the keys only while a game runs.", !s.pointHotkeyOff, true)}
          ${s.achToastsOff ? "" : row("s-toast", "See how it looks", "Shows one now, at the bottom right of the screen.", `<button id="s-toast" class="ln-button ln-button--secondary ln-button--sm" data-action="test-toast">Show one</button>`)}`)
        + sub("Steam", "", `
          ${sw("autoSteamHooks", "Set up Steam automatically", "Steam games you add get a launch option so Steam's Play button asks who's playing. Removed games get their own options back.", s.autoSteamHooks)}
          ${st.found ? row("s-steam", "Steam account", st.accountError ? `<span class="error-text">${esc(st.accountError)}</span>` : "The account on this PC whose Steam launch options Lootnest sets.", `<select id="s-steam" class="ln-input" data-setting="steamAccountId">
            <option value="">Automatic${st.accountInUse && !s.steamAccountId ? " (" + esc(st.accountInUse) + ")" : ""}</option>
            ${(st.accounts || []).map((a) => `<option value="${esc(a)}" ${a === s.steamAccountId ? "selected" : ""}>${esc(a)}</option>`).join("")}</select>`) : `<div class="bk-row"><div><b>Steam isn't on this PC</b><p>Steam games show up here once it's installed.</p></div></div>`}`)],
      ["online", "Online", GI.globe, "Each of these sends as little as it can, and only once you turn it on.", () => sub("What Lootnest may look up", "", `
          ${sw("findGames", "Store", "The Store page, and the search in the sidebar, look any game up on the Steam store, with its deals and requirements, and on CheapShark for its price in other official stores. Only what you search for, or a game's Steam id, is sent.", s.findGames)}
          ${sw("saveDb", "Find save locations online", "Downloads PCGamingWiki's list of where over 20,000 games keep their saves (from the Ludusavi project, on GitHub) about once a week, so new games come with the right locations for profiles and cloud saves. Nothing about you or your games is sent.", s.saveDb)}
          ${sw("gameNews", "Show game updates and news", "Each game's page shows its latest updates and patch notes from Steam's news feed, and tells you when a game no store keeps up to date has a newer version than yours. Only the game's Steam id, or its name, is sent.", s.gameNews)}
          ${sw("achRarity", "How rare achievements are", "Each achievement shows the share of Steam's players who have it, from Steam's public numbers, on the game's page and as it unlocks. Only the game's Steam id is sent; no key is needed.", s.achRarity)}
          ${sw("downloadCovers", "Download covers and artwork", "Covers, wide artwork and logos for games Steam hasn't saved on this PC, looked up on the Steam store by name (and on SteamGridDB with a key below), saved on this PC. Only the game's name or Steam id is sent. Off: those games show their own icon.", s.downloadCovers)}`)
        + sub("Your own keys", "Optional. Each is encrypted on this PC and sent only to the site it's for.", `
          <div class="bk-row set-wide"><div><label for="s-grid"><b>SteamGridDB API key</b></label>
            <p>Covers and artwork for games the Steam store doesn't know. Get a free key on steamgriddb.com, under Preferences, API.</p></div>
            <input id="s-grid" class="ln-input" type="password" autocomplete="off" spellcheck="false" data-setting="gridDbKey" value="${esc(s.gridDbKey || "")}" placeholder="Paste your key"></div>
          <div class="bk-row set-wide"><div><label for="s-steamkey"><b>Steam Web API key</b></label>
            <p>Names, descriptions and pictures for achievements Lootnest finds on this PC without them, such as a game in a folder. Get a free key on steamcommunity.com/dev/apikey. Only the game's Steam id and your key are sent, to Steam; what comes back is kept on this PC.</p></div>
            <input id="s-steamkey" class="ln-input" type="password" autocomplete="off" spellcheck="false" data-setting="steamApiKey" value="${esc(s.steamApiKey || "")}" placeholder="Paste your key"></div>`)],
      ["updates", "Updates", GI.load, "New versions of Lootnest, from its releases on GitHub.", () => card(versionRow + `
          ${sw("checkUpdates", "Check for updates automatically", "Every few hours, Lootnest asks GitHub for its latest version number and tells you with a notification when a new one is out. Nothing about you or your games is sent.", s.checkUpdates)}
          ${s.checkUpdates ? sw("autoInstallUpdates", "Install updates by themselves", "A new version installs while Lootnest is in the tray and no game is running, then Lootnest opens again in the tray. Your profiles, games and saves stay as they are.", s.autoInstallUpdates) : ""}`)],
      ["cloud", "Cloud saves", GI.cloud, "Your saves in your own cloud folder, on every PC, with backups of everything.", () => card(`
          <div class="bk-row"><div><b>Cloud saves and backups</b><p>Where the saves go, sync between PCs, the password and backups of everything are on their own page.</p></div>${b("primary", 'data-action="nav" data-view="cloud"', "Open Cloud saves")}</div>`)],
      ["data", "Data and activity", GI.folder, "Where Lootnest keeps profiles and backups, and what it did lately.", () => card(`
          <div class="bk-row"><div><b>Data folder</b><p class="selectable">Profiles and backups are stored in ${esc(S.d.dataDir)}</p></div>${b("secondary", 'data-action="open-data"', "Open data folder")}</div>
          <div class="bk-row"><div><b>Recent activity</b><p>Every swap, save and scan, with the reason when something didn't work. Useful when you ask for help.</p></div>${b("secondary", 'data-action="show-log"', "Show recent activity")}</div>
          ${S.logText ? `<div class="log">${esc(S.logText)}</div>` : ""}`)],
      ["about", "About", GI.info, "The version, how to tell us about a problem, and how to leave.", () => card(`
          ${versionRow}
          <div class="bk-row"><div><b>Lootnest is in beta</b><p>Everything that moves your files makes a copy first. If something doesn't work as it should, Report a problem copies the version and the recent activity for you to paste, and opens a new issue on GitHub in your browser. Lootnest sends nothing itself. Read it before you post: it names your games and folders.</p></div>${b("secondary", 'data-action="report-problem"', "Report a problem")}</div>
          <div class="bk-row"><div><b>Uninstalling</b><p>Lootnest uses the Microsoft Edge WebView2 runtime. To uninstall, use Settings, Apps in Windows. Your profiles and backups stay in the data folder.</p></div></div>
          <div class="bk-row"><div><b>Quit Lootnest</b><p>It stops watching your games until you open it again.</p></div>${b("danger", 'data-action="quit"', "Quit Lootnest")}</div>`)],
    ];
    const cur = tabs.find((t) => t[0] === S.setTab) || tabs[0];
    return `<div class="settings-page ln-root">
      <header class="lib-head"><div><h1>Settings</h1><p>Lootnest ${esc(S.d.version)}${S.d.stage ? " " + esc(S.d.stage) : ""}. Anything that goes online is off until you turn it on.</p></div></header>
      <div class="set-layout">
        <nav class="set-nav" aria-label="Settings">${tabs.map(([k, label, ic]) => `<button class="set-tab" data-action="set-tab" data-tab="${k}"${k === cur[0] ? ' aria-current="true"' : ""}>${ic}<span>${label}</span>${k === "updates" && S.update.available ? '<span class="set-dot" aria-label="An update is ready"></span>' : ""}</button>`).join("")}</nav>
        <section class="set-panel" aria-labelledby="set-title"><div class="set-panel-head"><h2 id="set-title">${cur[1]}</h2><p>${cur[3]}</p></div>${cur[4]()}</section>
      </div></div>`;
  }

  // ---------------------------------------------------------------- backup --
  // Backups go to a folder a cloud app syncs; Lootnest writes the file and
  // the app uploads it.
  async function loadBackupInfo() {
    try { S.backup = await api.backupInfo(); } catch (e) { return; }
    if ((S.view === "settings" || S.view === "cloud") && !S.dialog) render();
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

  // --------------------------------------------------------------- updates --
  // Games no store keeps up to date: whether a newer version is out, and an
  // update the user brings (a folder or a .zip), applied with a copy first.
  const UPDATABLE = ["folder", "added", "shortcut", "hydra", "other"];
  const updKey = (it) => (it.kind === "game" ? (it.g.launch.type === "exe" && UPDATABLE.includes(it.source) ? "game:" + it.id : "") : UPDATABLE.includes(it.source) && !it.appId ? "found:" + it.id : "");
  function updateNavCount() {
    const n = (S.updates ? S.updates.games.filter((g) => g.newer).length : 0) + (S.copies ? S.copies.length : 0);
    const el = $("#upd-count"); if (el) el.textContent = n ? String(n) : "";
  }
  // "Is this a new copy?": a game reinstalled as a whole new version.
  function copyCard(o, compact) {
    const C = S.carried[o.old.key] || {};
    const btn = (attrs, label, cls = "ln-button--secondary") => `<button class="ln-button ${cls} ln-button--sm" ${attrs} ${C.busy ? "disabled" : ""}>${label}</button>`;
    const ids = `data-old="${esc(o.old.key)}" data-new="${esc(o.new.key)}" data-name="${esc(o.old.name)}"`;
    const what = [o.profiles ? "its profiles and their saves" : o.saves.length ? `its saves (${o.saves.map(esc).join(", ")})` : "", "playtime, status, collections and settings"].filter(Boolean).join(", ");
    return `<div class="copy-card"><div><b>${esc(o.new.name)} looks like a new copy of ${esc(o.old.name)}</b>
      <p>${o.oldGone ? `The old folder, <span class="selectable">${esc(o.old.dir)}</span>, is gone.` : `The old copy is still in <span class="selectable">${esc(o.old.dir)}</span>.`} Lootnest can carry over ${what}.</p></div>
      <div class="gpage-actions">${btn(`data-action="copy-carry" ${ids}`, C.busy ? "Carrying over" : "Carry everything over", "ln-button--primary")}${compact ? "" : btn(`data-action="copy-dismiss" ${ids}`, "Not the same game", "ln-button--ghost")}</div></div>`;
  }
  function carriedCard(oldKey) {
    const C = S.carried[oldKey];
    if (!C || !C.done) return "";
    const saves = C.profile ? `${esc(pname(C.profile))}'s files are in the new copy.` : C.saves.length ? `${C.saves.length} save location${C.saves.length === 1 ? "" : "s"} came along${C.saves.some((x) => x.replaced) ? ", and the new copy's own are in backups" : ""}.` : "It had no saves inside its folder.";
    return `<div class="copy-card copy-card--done"><div><b>${C.name ? esc(C.name) + " is carried over to its new copy" : "Carried over"}</b><p>${saves} Playtime, status, collections and settings moved too.</p></div>
      ${C.oldThere && !C.deleted ? `<div class="gpage-actions"><button class="ln-button ln-button--ghost ln-button--sm" data-action="copy-delete-old" data-old="${esc(oldKey)}" ${C.busy ? "disabled" : ""}>Delete the old folder</button></div>` : ""}</div>`;
  }
  function viewUpdates() {
    const head = `<header class="lib-head"><div><h1>Updates</h1><p>For games no store keeps up to date. Lootnest tells you when a newer version is out, and applies an update you have, with a copy of everything it replaces first.</p></div></header>`;
    const U = S.updates;
    if (!U) { api.updates().catch(() => {}); return `<div class="upd-page ln-root">${head}${lnInline("", "Checking your games")}</div>`; }
    if (!U.games.length) return `<div class="upd-page ln-root">${head}<p class="find-hint">No games in folders yet. Steam, Epic, GOG and the other launchers keep their own games up to date.</p></div>`;
    const news = U.newsOn ? "" : banner("Find out when a newer version is out", "Lootnest reads each game's patch notes from Steam's news feed and compares their date with your copy's files. Only the game's Steam id, or its name, is sent.", `<button class="ln-button ln-button--secondary ln-button--sm" data-action="news-on">Turn it on</button>`);
    const row = (r) => {
      const applied = (r.applied || []).filter((u) => u.state === "applied");
      const last = applied[0];
      const status = r.newer ? `<b class="upd-new">${esc(r.newer.title)}</b><span>Released ${esc(day(r.newer.date))}. Your copy's files are older.</span>`
        : r.noNews ? `<span>Not on the Steam store, so Lootnest can't tell when it has a new version.</span>`
        : U.newsOn ? `<span>Up to date, as far as Steam's news says.</span>` : `<span>Files from ${esc(day(r.changed))}.</span>`;
      const vtxt = (v) => (v ? (v.startsWith("build ") ? "Build " + v.slice(6) : v) : "");
      const chain = applied.length > 1 ? applied.slice().reverse().map((u) => esc(vtxt(u.to) || u.source)).join(" → ") : "";
      const broke = last && last.broke ? `<span class="find-low">It closed right after starting, the first time since this update. If it doesn't work, Undo puts it back.</span>` : "";
      const hist = last ? `${broke}<span class="upd-last">${r.version ? `Version ${esc(vtxt(r.version))}. ` : ""}Updated ${esc(ago(last.at))} from ${esc(last.source)}${chain ? ` (${chain})` : ""}.${last.noUndo ? "" : ` <button class="gpage-link" data-action="upd-undo" data-key="${esc(r.key)}" data-id="${esc(last.id)}" data-name="${esc(r.name)}">Undo</button>`}</span>`
        : r.version ? `<span class="upd-last">Version ${esc(vtxt(r.version))}, from its folder's name.</span>` : "";
      const keepLink = `<button class="gpage-link upd-keep" data-action="upd-keep" data-key="${esc(r.key)}">Reinstalling it? Copy its saves first</button>`;
      return `<li class="upd-row${r.newer ? " upd-row--new" : ""}"><div class="upd-name"><button class="gpage-link upd-title" data-action="upd-open-game" data-key="${esc(r.key)}">${esc(r.name)}</button><span class="upd-src">${esc(sourceLabel(r.source || "folder"))}</span></div>
        <div class="upd-status">${status}${hist}${keepLink}</div>
        <div class="gpage-actions"><button class="ln-button ${r.newer ? "ln-button--primary" : "ln-button--secondary"} ln-button--sm" data-action="upd-start" data-key="${esc(r.key)}" data-name="${esc(r.name)}">Apply an update</button></div></li>`;
    };
    const newer = U.games.filter((g) => g.newer), rest = U.games.filter((g) => !g.newer);
    if (!S.copies) api.newCopies().catch(() => {});
    const offers = S.copies || [];
    const done = Object.keys(S.carried).filter((k) => S.carried[k].done && !S.carried[k].deleted).map(carriedCard).join("");
    const copies = offers.length || done ? `<section class="find-row"><div class="ln-section-head"><div><h2>New copies</h2><p>Games reinstalled as a whole new version. Carry everything over so nothing stays behind with the old folder.</p></div></div>${done}${offers.map((o) => copyCard(o, false)).join("")}</section>` : "";
    return `<div class="upd-page ln-root">${head}${news}${copies}
      ${newer.length ? `<section class="find-row"><div class="ln-section-head"><div><h2>A newer version is out</h2></div></div><ul class="upd-list">${newer.map(row).join("")}</ul></section>` : ""}
      <section class="find-row"><div class="ln-section-head"><div><h2>${newer.length ? "Your other games in folders" : "Your games in folders"}</h2></div></div><ul class="upd-list">${rest.map(row).join("")}</ul></section></div>`;
  }
  function viewUpdateDrawer() {
    const D = S.dialog;
    const btn = (attrs, label, cls = "ln-button--secondary") => `<button class="ln-button ${cls} ln-button--sm" ${attrs} ${D.busy ? "disabled" : ""}>${label}</button>`;
    const shell = (body, foot) => drawer({ label: `Update ${esc(D.name)}`, title: `Update ${esc(D.name)}`, sub: "With an update you have: a folder, a .zip, .7z or .rar, or an installer", body, foot });
    const choose = `<div class="gpage-actions">${btn('data-action="upd-pick" data-kind="update"', "Choose a .zip, .7z or .rar")}${btn('data-action="upd-pick" data-kind="updateDir"', "Choose a folder", "ln-button--ghost")}${btn('data-action="upd-pick" data-kind="installer"', "Run an installer", "ln-button--ghost")}</div>`;
    if (D.done) {
      const u = D.done;
      return shell(D.undone ? `<div class="run-card run-card--ok"><div class="run-head"><span class="run-dot"></span><div><b>Undone</b><p>${esc(D.name)} is as it was before the update from ${esc(u.source)}.</p></div></div></div>`
        : `<div class="run-card run-card--ok"><div class="run-head"><span class="run-dot"></span><div><b>Updated</b><p>${u.added.length} files added and ${u.replaced.length} replaced or removed, from ${esc(u.source)}. ${u.noUndo ? "The saves are kept. No copy of the game was kept, so it can't be undone." : "The saves and the old files are kept, so you can undo it."} The first time you start it, Lootnest checks it runs.</p></div></div></div>`,
        D.undone ? `<span class="grow"></span>${btn('data-action="dialog-cancel"', "Done", "ln-button--primary")}`
          : `${u.noUndo ? "" : btn(`data-action="upd-undo" data-key="${esc(D.key)}" data-id="${esc(u.id)}" data-name="${esc(D.name)}"`, "Undo", "ln-button--ghost")}${btn('data-action="upd-another"', "Apply another update", "ln-button--ghost")}<span class="grow"></span>${btn('data-action="dialog-cancel"', "Done", "ln-button--primary")}`);
    }
    const intro = `<p class="gpage-hint">Lootnest doesn't download updates. Choose one you have, and Lootnest shows what it changes before anything happens. If an update comes in parts (.part1.rar, .part2.rar), choose the first. For an installer (.exe), run it yourself.</p>`;
    if (!D.path) return shell(`${intro}${choose}`);
    if (D.busy && !D.plan) return shell(lnInline("", D.opening ? "Opening the archive" : "Reading the update", D.opening ? "A big update takes a minute or two. Nothing in the game's folder changes yet." : ""));
    if (D.error && !D.plan) return shell(`${lnInline("error", esc(D.error))}${choose}`);
    const P = D.plan;
    if (!P) return shell(`${intro}${choose}`);
    if (P.kind === "installer") {
      const free = D.free ?? P.free ?? -1, room = free < 0 || free >= P.gameSize * 1.05;
      if (D.full === undefined) D.full = room;
      const steps = { copying: "Copying the game first. A big game takes a few minutes.", installer: "The installer is running. Finish it, then come back here.", checking: "Checking what the installer changed" };
      if (D.busy) return shell(lnInline("", steps[D.step] || "Getting ready", D.step === "installer" ? "Lootnest records what changes in the game's folder when it closes." : ""));
      return shell(`<div class="gpage-sub">${gpHead("Run an installer")}<p class="gpage-hint">Lootnest copies the saves, runs <span class="selectable">${esc(P.name)}</span>, then records what it changed in <span class="selectable">${esc(D.dir)}</span>. Point the installer at that folder.</p></div>
        <label class="toggle"><span><b>Keep a copy of the game first, so it can be undone</b><p>${esc(fmtBytes(P.gameSize))}${room ? "" : `, more than the room free (${esc(fmtBytes(free))})`}. Once the installer is done, only the files it changed are kept.</p></span>
          <span class="switch"><input type="checkbox" data-action="upd-full" ${D.full ? "checked" : ""} ${room ? "" : "disabled"}><span></span></span></label>
        ${D.error ? lnInline("error", esc(D.error)) : ""}`,
        `${btn('data-action="upd-pick" data-kind="installer"', "Choose another", "ln-button--ghost")}<span class="grow"></span>${btn('data-action="upd-run"', "Run the installer", "ln-button--primary")}`);
    }
    const where = P.where.slice(0, 6).map((w) => `<div><dt class="selectable">${w.path ? esc(w.path) : "The game's own folder"}</dt><dd>${w.files} file${w.files === 1 ? "" : "s"}</dd></div>`).join("");
    const room = D.free >= 0 && D.free < P.bytes + P.kept ? `<p class="find-low">There isn't enough room on this drive: it needs ${esc(fmtBytes(P.bytes + P.kept))}.</p>` : "";
    const C = D.check || {};
    const ver = (v) => (v ? esc(v.startsWith("build ") ? "Build " + v.slice(6) : v) : "");
    const versions = C.to ? `<div class="gpage-sub">${gpHead("Versions")}<dl class="ln-facts">
        <div><dt>Your copy</dt><dd>${C.have ? ver(C.have) : "Not known"}</dd></div>
        ${C.from ? `<div><dt>This update is for</dt><dd>${ver(C.from)}</dd></div>` : ""}
        <div><dt>It brings</dt><dd>${ver(C.to)}</dd></div></dl>
        ${C.state === "skips" ? `<p class="find-low">This update is for ${ver(C.from)}, and your copy is ${ver(C.have)}. Updates usually need to go in order: apply the one${C.from ? ` that brings ${ver(C.from)}` : "s in between"} first, or the game may not start.</p>`
          : C.state === "older" ? `<p class="find-low">This update brings ${ver(C.to)}, older than your copy's ${ver(C.have)}.</p>`
          : C.state === "same" ? `<p class="find-low">Your copy already has ${ver(C.to)}.</p>`
          : C.state === "ok" ? `<p class="find-ok">The next update for your copy.</p>`
          : `<p class="gpage-hint">Lootnest reads versions from names, such as "v1.0.2 to v1.0.3". ${C.have ? "" : "Your copy's folder name has none, so it can't check the order."}</p>`}</div>` : "";
    const body = `${versions}<div class="gpage-sub">${gpHead("What it does")}<dl class="ln-facts">
        <div><dt>Replaces</dt><dd>${P.replace} file${P.replace === 1 ? "" : "s"}</dd></div>
        <div><dt>Adds</dt><dd>${P.added} file${P.added === 1 ? "" : "s"}</dd></div>
        <div><dt>Size</dt><dd>${esc(fmtBytes(P.bytes))}</dd></div></dl>
        <p class="gpage-hint">From <span class="selectable">${esc(P.source)}</span>${P.skipped ? `. The files inside its folder "${esc(P.skipped)}" go into the game's folder` : ""}.</p></div>
      <div class="gpage-sub">${gpHead("Where they go")}<dl class="ln-facts gpage-paths">${where}</dl><p class="gpage-hint selectable">${esc(D.dir)}</p></div>
      ${P.readme && P.readme.length ? `<div class="gpage-sub">${gpHead("Notes that came with it")}<p class="gpage-hint">${P.readme.map(esc).join(", ")}: shown, not copied. Read them in case the update asks for anything else.</p></div>` : ""}
      <div class="gpage-sub"><p class="gpage-hint">First, Lootnest copies the saves and the ${P.replace} files it replaces (${esc(fmtBytes(P.kept))}), so <b>Undo</b> puts everything back. ${room}</p></div>
      ${D.error ? lnInline("error", esc(D.error)) : ""}`;
    return shell(body, `${btn('data-action="upd-pick" data-kind="' + (P.kind === "zip" ? "update" : "updateDir") + '"', "Choose another", "ln-button--ghost")}<span class="grow"></span>${D.busy ? `<button class="ln-button ln-button--primary ln-button--sm" aria-busy="true" disabled><span class="ln-spinner" aria-hidden="true"></span>Applying</button>` : btn('data-action="upd-apply"', "Apply the update", "ln-button--primary")}`);
  }

  // --------------------------------------------------------- sidebar search --
  // One search box on every page: your games first, then the store's.
  let gqTimer = 0;
  function gqItems() {
    const q = S.gq.q.trim().toLowerCase();
    if (!q) return [];
    const mine = libraryItems().filter((it) => it.name.toLowerCase().includes(q)).slice(0, 5).map((it) => ({ mine: it }));
    const ownedIds = new Set(mine.map((m) => m.mine.appId).filter(Boolean));
    const store = (S.gq.hits || []).filter((h) => !ownedIds.has(h.appId) && !h.owned).slice(0, 6).map((h) => ({ hit: h }));
    return [...mine, ...store];
  }
  function renderGq() {
    const pop = $("#gq-pop"), input = $("#gq");
    if (!pop || !input) return;
    const q = S.gq.q.trim();
    if (!q || document.activeElement !== input && !pop.contains(document.activeElement)) { pop.hidden = true; input.setAttribute("aria-expanded", "false"); return; }
    const items = gqItems(), on = cfg().settings.findGames;
    let i = -1;
    const opt = (inner, extra = "") => { i++; return `<button class="gq-item${i === S.gq.sel ? " is-sel" : ""}" role="option" aria-selected="${i === S.gq.sel}" data-action="gq-open" data-i="${i}" ${extra}>${inner}</button>`; };
    const mine = items.filter((x) => x.mine), store = items.filter((x) => x.hit);
    const mineHTML = mine.length ? `<div class="gq-group"><span class="gq-label">Your games</span>${mine.map((m) => opt(`<span class="gq-name">${esc(m.mine.name)}</span><span class="gq-meta">${esc(m.mine.kind === "game" ? "Profiles on" : sourceLabel(m.mine.source))}</span>`)).join("")}</div>` : "";
    const storeHTML = !on ? `<div class="gq-group"><span class="gq-label">Store</span><p class="gq-note">Turn on the Store to search every game, with prices and whether it runs here.</p></div>`
      : S.gq.hits === null ? `<div class="gq-group"><span class="gq-label">Store</span><p class="gq-note">Looking it up</p></div>`
      : store.length ? `<div class="gq-group"><span class="gq-label">Store</span>${store.map((x) => opt(`${x.hit.image ? `<img src="${esc(x.hit.image)}" alt="">` : ""}<span class="gq-name">${esc(x.hit.name)}</span><span class="gq-meta">${x.hit.price ? esc(cents(x.hit.price)) : ""}</span>`)).join("")}</div>` : "";
    const none = !mine.length && on && S.gq.hits && !store.length ? `<p class="gq-note">Nothing called "${esc(q)}".</p>` : "";
    pop.innerHTML = `${mineHTML}${storeHTML}${none}${on ? `<button class="gq-all" data-action="gq-all">See everything for "${esc(q)}" in the Store</button>` : ""}`;
    pop.hidden = false; input.setAttribute("aria-expanded", "true");
  }
  function gqClose() { S.gq.sel = -1; const pop = $("#gq-pop"); if (pop) pop.hidden = true; const i = $("#gq"); if (i) i.setAttribute("aria-expanded", "false"); }
  function gqOpen(i) {
    const x = gqItems()[i];
    if (!x) return;
    gqClose(); $("#gq").blur();
    if (x.mine) openSheet(x.mine.kind, x.mine.id, null);
    else actions["find-open"]({ dataset: { app: String(x.hit.appId), name: x.hit.name } });
  }
  function gqAll() {
    const q = S.gq.q.trim();
    if (!q) return;
    gqClose();
    S.find = { q, hits: null, error: "" };
    S.view = "find"; S.page = null; S.dialog = null; S.animateView = true; render();
    api.findGames(q).catch((e) => { S.find.error = errText(e); const r = $("#find-results"); if (r) r.innerHTML = findResults(); });
  }

  // ------------------------------------------------------------ find games --
  // Any game, installed or not: its price in the official stores and its
  // requirements beside this PC. Opt-in, in Settings, Online.
  const money = (n, cur) => { try { return new Intl.NumberFormat(undefined, { style: "currency", currency: cur }).format(n); } catch (e) { return n.toFixed(2) + " " + cur; } };
  const cents = (p) => (p ? money(p.final / 100, p.currency) : "");
  const offTag = (pct) => (pct > 0 ? `<span class="ln-tag ln-tag--accent">${pct}% off</span>` : "");
  function viewFind() {
    const on = cfg().settings.findGames;
    const search = on ? `<div class="lib-actions"><label class="ln-search">${GI.search}<input type="search" id="find-q" class="ln-input" placeholder="Search the store" value="${esc(S.find.q)}" autocomplete="off" spellcheck="false" aria-label="Search the store"></label></div>` : "";
    const head = `<header class="lib-head"><div><h1>Store</h1><p>Any game, with its price in every official store and whether it runs on this PC.</p></div>${search}</header>`;
    if (!on) return `<div class="find-page ln-root">${head}<section class="find-off"><h2>Look up any game</h2>
      <p>The Store asks the Steam store for games, deals and requirements, and CheapShark for prices in other official stores, such as GOG, Humble and Fanatical. Only what you type, or a game's Steam id, is sent. Lootnest opens the store you choose in your browser.</p>
      <button class="ln-button ln-button--primary" data-action="find-on">Turn on the Store</button></section></div>`;
    return `<div class="find-page ln-root">${head}<div id="find-results">${findResults()}</div></div>`;
  }
  // A game in a Store row, on its wide art.
  function storeCard(it) {
    const price = it.store ? `<span class="ln-num">${esc(money(it.usd, "USD"))}</span>${offTag(it.price && it.price.discount)}<span class="find-store-name">${esc(it.store)}</span>`
      : it.price ? `<span class="ln-num">${esc(cents(it.price))}</span>${offTag(it.price.discount)}` : `<span>Free</span>`;
    return `<li><button class="find-card find-card--wide" data-action="find-open" data-app="${it.appId}" data-name="${esc(it.name)}">
      ${it.image ? `<img src="${esc(it.image)}" alt="" loading="lazy">` : `<span class="find-noimg"></span>`}
      <span class="find-card-body"><b>${esc(it.name)}</b><span class="find-card-meta">${price}${it.score ? `<span class="find-score" title="Metacritic">${it.score}</span>` : ""}${it.owned ? `<span class="ln-tag">On this PC</span>` : ""}</span></span></button></li>`;
  }
  function storeFront() {
    const F = S.front;
    if (!F) { api.storeFront().catch((e) => { S.front = { error: errText(e) }; const r = $("#find-results"); if (r) r.innerHTML = findResults(); }); S.front = { loading: true }; return lnInline("", "Opening the store"); }
    if (F.loading) return lnInline("", "Opening the store");
    if (F.error) return `${lnInline("error", esc(F.error))}<div class="gpage-actions"><button class="ln-button ln-button--secondary ln-button--sm" data-action="store-retry">Try again</button></div>`;
    const row = (title, sub, list) => (list && list.length ? `<section class="find-row"><div class="ln-section-head"><div><h2>${title}</h2>${sub ? `<p>${sub}</p>` : ""}</div></div><ul class="find-grid find-grid--wide">${list.map(storeCard).join("")}</ul></section>` : "");
    const d = F.daily;
    const daily = d ? `<button class="find-daily" data-action="find-open" data-app="${d.appId}" data-name="${esc(d.name)}">${d.image ? `<img src="${esc(d.image)}" alt="">` : ""}
      <span class="find-daily-body"><span class="find-daily-label">Today's deal on Steam</span><b>${esc(d.name)}</b><span class="find-card-meta">${d.price ? `<span class="ln-num">${esc(cents(d.price))}</span>${offTag(d.price.discount)}` : ""}${d.owned ? `<span class="ln-tag">On this PC</span>` : ""}</span></span></button>` : "";
    return `${daily}${row("Deals on Steam", "", F.specials)}${row("Best deals in other stores", "Well-reviewed games on sale in official stores, from CheapShark, in US dollars.", F.deals)}${row("Top sellers on Steam", "", F.top)}`;
  }
  function findResults() {
    const F = S.find;
    if (!F.q.trim()) return storeFront();
    if (F.error) return lnInline("error", esc(F.error));
    if (!F.hits) return lnInline("", "Looking it up");
    if (!F.hits.length) return `<p class="find-hint">The Steam store has nothing called "${esc(F.q)}".</p>`;
    return `<ul class="find-grid">${F.hits.map((h) => `<li><button class="find-card" data-action="find-open" data-app="${h.appId}" data-name="${esc(h.name)}">
      ${h.image ? `<img src="${esc(h.image)}" alt="" loading="lazy">` : `<span class="find-noimg"></span>`}
      <span class="find-card-body"><b>${esc(h.name)}</b><span class="find-card-meta">${h.price ? `<span class="ln-num">${esc(cents(h.price))}</span>${offTag(h.price.discount)}` : ""}${h.owned ? `<span class="ln-tag">On this PC</span>` : ""}</span></span></button></li>`).join("")}</ul>`;
  }
  // Sizes in requirements, as in core.ReqBytes.
  const reqBytes = (v) => { const m = /(\d+(?:[.,]\d+)?)\s*(GB|MB|TB)/i.exec(v || ""); if (!m) return 0; const n = parseFloat(m[1].replace(",", ".")); return n * { tb: 2 ** 40, gb: 2 ** 30, mb: 2 ** 20 }[m[2].toLowerCase()]; };
  const gb = (b) => (b >= 2 ** 30 ? `${Math.round(b / 2 ** 30)} GB` : b > 0 ? `${Math.round(b / 2 ** 20)} MB` : "");
  // "Will it run here?": core.WillItRun's verdict, in words and bars.
  const RUNS = {
    high: ["Runs great here", "Above the recommended requirements, so high settings should be fine.", "Above recommended"],
    recommended: ["Runs well here", "At the recommended requirements: medium to high settings.", "Recommended"],
    minimum: ["Runs, on low settings", "At the minimum requirements.", "Minimum"],
    struggles: ["May struggle", "Just under the minimum: expect low settings and slowdowns.", "Under the minimum"],
    below: ["Probably won't run well", "Well under the minimum requirements.", "Well under"],
  };
  const PART = { Graphics: "graphics card", Processor: "processor", Memory: "memory", Storage: "free space" };
  const runsTone = (r) => (r === "high" || r === "recommended" ? "ok" : r === "minimum" ? "mid" : "low");
  function runGauge(p) {
    const top = Math.max(p.have, (p.rec || p.min) * 1.6, p.min) || 1;
    const pct = (x) => `${Math.max(0, Math.min(100, (x / top) * 100)).toFixed(1)}%`;
    return `<div class="run-gauge run-gauge--${runsTone(p.runs)}" style="--have:${pct(p.have)};--min:${pct(p.min)};--rec:${pct(p.rec)}">
      <span class="run-gauge-label">${esc(p.part)}</span>
      <span class="run-bar" role="img" aria-label="${esc(p.part)}: ${esc(RUNS[p.runs][2])}">${p.min ? `<i class="run-mark run-mark--min"></i>` : ""}${p.rec ? `<i class="run-mark run-mark--rec"></i>` : ""}</span>
      <span class="run-gauge-val">${esc(RUNS[p.runs][2])}</span></div>`;
  }
  function runCard(V) {
    if (!V || !V.parts) return "";
    const r = RUNS[V.runs];
    if (!r) {
      const g = V.parts[0], c = V.parts[1];
      const what = !g.have && !c.have ? "this PC's graphics card and processor" : !g.min && !g.rec && !c.min && !c.rec ? "the parts this game asks for" : "these parts";
      return `<div class="run-card run-card--unknown"><div class="run-head"><span class="run-dot"></span><div><b>Lootnest can't tell</b><p>It doesn't know ${what} well enough to compare. Compare the names below.</p></div></div></div>`;
    }
    const limit = V.limit && V.runs !== "high" && V.runs !== "recommended" ? ` The ${PART[V.limit]} holds it back.` : "";
    const gauges = V.parts.filter((p) => p.known && (p.part === "Graphics" || p.part === "Processor")).map(runGauge).join("");
    return `<div class="run-card run-card--${runsTone(V.runs)}"><div class="run-head"><span class="run-dot"></span><div><b>${r[0]}</b><p>${r[1]}${limit}</p></div></div>
      ${gauges ? `<div class="run-gauges">${gauges}</div><p class="run-legend"><i class="run-mark run-mark--min"></i>Minimum<i class="run-mark run-mark--rec"></i>Recommended</p>` : ""}
      <p class="gpage-hint">An estimate from typical benchmark results; Steam's requirements usually mean 1080p.</p></div>`;
  }
  function viewOffer() {
    const D = S.dialog, R = S.offers[D.appId];
    const shell = (body, foot) => drawer({ label: `${esc(D.name)}, prices and requirements`, title: esc(D.name), sub: "Prices and requirements", wide: true, body, foot });
    if (!R) return shell(lnInline("", "Getting prices and requirements"));
    if (R.error || !R.offer) return shell(lnInline("error", esc(R.error || "Couldn't get this game.")));
    const O = R.offer, pc = R.pc || {};
    const btn = (attrs, label, cls = "ln-button--ghost") => `<button class="ln-button ${cls} ln-button--sm" ${attrs}>${label}</button>`;
    const owned = R.owned ? `<div class="find-owned"><p><b>On this PC.</b> It's in your Library.</p>${btn(`data-action="find-library" data-owned="${esc(R.owned)}"`, "Open its page", "ln-button--secondary")}</div>` : "";
    const steam = O.free ? "Free to play" : O.steam ? `<span class="ln-num">${esc(cents(O.steam))}</span>${offTag(O.steam.discount)}` : "Not for sale";
    const deals = (O.deals || []).filter((d) => d.store !== "Steam");
    const prices = `<div class="gpage-sub">${gpHead("Prices")}<dl class="ln-facts">
      <div><dt><b>Steam</b></dt><dd>${steam}${btn(`data-action="find-store" data-url="${esc(O.storeUrl)}"`, "Open")}</dd></div>
      ${deals.map((d) => `<div><dt>${esc(d.store)}</dt><dd><span class="ln-num">${esc(money(d.price, "USD"))}</span>${offTag(d.savings)}${btn(`data-action="find-store" data-url="${esc(d.url)}"`, "Open")}</dd></div>`).join("")}</dl>
      ${O.free ? "" : `<p class="gpage-hint">${deals.length ? "Other stores' prices are in US dollars, from CheapShark, which lists authorized stores only." : "No other store sells it, according to CheapShark."}${O.lowest ? ` Lowest price ever: ${esc(money(O.lowest, "USD"))}${O.lowestAt ? ", " + esc(new Date(O.lowestAt).toLocaleDateString(undefined, { month: "short", year: "numeric" })) : ""}.` : ""}</p>`}</div>`;
    const val = (list, label) => ((list || []).find((l) => l.label === label) || {}).value || "";
    const known = ["OS", "Processor", "Memory", "Graphics", "Storage"];
    const labels = [...known, ...new Set([...(O.minimum || []), ...(O.recommended || [])].map((l) => l.label).filter((l) => l && !known.includes(l)))].filter((l) => val(O.minimum, l) || val(O.recommended, l));
    const mark = (have, label) => {
      const need = (w) => reqBytes(val(w, label));
      const min = need(O.minimum), rec = need(O.recommended);
      if (!have || !min) return "";
      return have >= (rec || min) ? `<span class="find-ok">Enough</span>` : have >= min ? `<span class="find-ok">Meets the minimum</span>` : `<span class="find-low">Below the minimum</span>`;
    };
    const mine = {
      OS: esc(pc.windows || ""),
      Processor: esc(pc.cpu || ""),
      Graphics: esc(pc.gpu || "") + (pc.vram ? `, ${gb(pc.vram)}` : ""),
      Memory: pc.ram ? `${gb(pc.ram)} ${mark(pc.ram, "Memory")}` : "",
      Storage: pc.free ? `${gb(pc.free)} free on ${esc(pc.freeOn)} ${mark(pc.free, "Storage")}` : "",
    };
    const notes = [...(O.minimum || [])].filter((l) => !l.label).map((l) => esc(l.value));
    const V = R.verdict, vpart = (name) => (V && V.parts ? V.parts.find((p) => p.part === name) : null);
    const partMark = (name) => { const p = vpart(name); return p && p.known ? `<span class="find-${runsTone(p.runs) === "low" ? "low" : "ok"}">${esc(RUNS[p.runs][2])}</span>` : ""; };
    mine.Graphics += partMark("Graphics");
    mine.Processor += partMark("Processor");
    const reqs = labels.length ? `<div class="gpage-sub">${gpHead("Requirements and this PC")}<div class="find-req" role="table" aria-label="Requirements and this PC">
      <div class="find-req-row find-req-head" role="row"><span role="columnheader"></span><span role="columnheader">Minimum</span><span role="columnheader">Recommended</span><span role="columnheader">This PC</span></div>
      ${labels.map((l) => `<div class="find-req-row" role="row"><b role="rowheader">${esc(l)}</b><span role="cell">${esc(val(O.minimum, l)) || "–"}</span><span role="cell">${esc(val(O.recommended, l)) || "–"}</span><span role="cell">${mine[l] || ""}</span></div>`).join("")}</div>
      ${notes.length ? `<p class="gpage-hint">${notes.join(". ")}.</p>` : ""}</div>`
      : `<div class="gpage-sub">${gpHead("Will it run here?")}<p class="gpage-hint">The Steam store lists no requirements for this game.</p></div>`;
    const wiki = `<div class="gpage-sub">${gpHead("Known issues")}<p class="gpage-hint">PCGamingWiki says what's known to go wrong on PC, and how to fix it.</p><div class="gpage-actions">${btn(`data-action="gs-pcgw" data-name="${esc(O.name || D.name)}"`, "Open PCGamingWiki", "ln-button--secondary")}</div></div>`;
    const hero = O.header ? `<img class="find-hero" src="${esc(O.header)}" alt="">` : "";
    const run = labels.length ? `<div class="gpage-sub">${runCard(V)}</div>` : "";
    return shell(`${hero}${owned}${run}${prices}${reqs}${wiki}`, `<span class="grow"></span>${btn(`data-action="find-store" data-url="${esc(O.storeUrl)}"`, "Open on Steam", "ln-button--primary")}`);
  }

  let allAchAsked = false;
  function viewAchievements() {
    const head = `<header class="lib-head"><div><h1>Achievements</h1><p>Every game's achievements on this PC. Each profile has its own where they go with the profile; Steam's belong to the Steam account.</p></div></header>`;
    if (!S.allAch) {
      if (!allAchAsked) { allAchAsked = true; api.allAchievements().catch(() => {}).finally(() => { allAchAsked = false; }); }
      return `<div class="ach-page ln-root">${head}${lnInline("", "Reading achievements on this PC")}</div>`;
    }
    const ps = cfg().players, seat = seated();
    const who = (ps.find((p) => p.id === S.achPage) || seat || ps[0] || {}).id;
    const rows = S.allAch.games.map((g) => {
      const own = g.players ? g.players[who] : null;
      if (g.players && !own) return null; // this profile hasn't played it
      const it = itemByKey(g.key);
      return it ? { g, it, unlocked: own ? own.unlocked : g.unlocked, recent: own ? own.recent : g.recent, shared: !g.players } : null;
    }).filter(Boolean);
    const chips = ps.length > 1 ? `<div class="ach-who" role="group" aria-label="Whose achievements">${ps.map((p) => `<button class="ln-chip ln-player" style="--player:${esc(p.color)}" data-action="ach-page" data-player="${esc(p.id)}" aria-pressed="${p.id === who}">${lnAvatar(p, "20")}${esc(p.name)}</button>`).join("")}</div>` : "";
    if (!rows.length) {
      return `<div class="ach-page ln-root">${head}${chips}<div class="ln-glass ln-state">${GI.trophy}<div><h3>No achievements found yet</h3>
        <p>Lootnest reads them from Steam's library on this PC, and from the file a Steam-compatible tool keeps next to a game's saves. Play a game that has achievements and they show up here.</p></div></div></div>`;
    }
    const total = rows.reduce((n, r) => n + r.unlocked, 0), perfect = rows.filter((r) => r.unlocked && r.unlocked === r.g.total).length;
    const ms = who && player(who) ? milestones(player(who), rows) : [];
    const msDone = ms.filter((x) => x.done).length;
    const msHTML = ms.length ? `<section class="ach-ms"><div class="ln-section-head"><div><h2>Lootnest milestones</h2><p>${msDone} of ${ms.length}, for any game, from the time ${esc(pname(who))} plays through Lootnest.</p></div></div>
      <ul class="ach-ms-grid">${ms.map((x) => `<li class="${x.done ? "done" : ""}"><span class="ach-ms-icon">${GI.trophy}</span><div><b>${esc(x.name)}</b><span>${x.done ? esc(x.desc) : esc(x.desc) + ". " + esc(fmtGoal(x))}</span>${x.done ? "" : `<span class="ach-bar" aria-hidden="true"><span style="width:${Math.round((x.cur / x.goal) * 100)}%"></span></span>`}</div></li>`).join("")}</ul></section>` : "";
    const latest = rows.flatMap((r) => (r.recent || []).map((a) => ({ a, r }))).sort((x, y) => new Date(y.a.at || 0) - new Date(x.a.at || 0)).slice(0, 10);
    const games = rows.slice().sort((x, y) => y.unlocked / y.g.total - x.unlocked / x.g.total || y.unlocked - x.unlocked || x.g.name.localeCompare(y.g.name));
    const icon = (a) => a.icon ? `<img src="${esc(a.icon)}" alt="" loading="lazy">` : `<span class="ach-none">${GI.trophy}</span>`;
    const p = player(who);
    const all = rows.reduce((n, r) => n + r.g.total, 0), allPct = all ? Math.round((total / all) * 100) : 0;
    return `<div class="ach-page ln-root">${head}
      <div class="ach-summary ln-glass">${chips ? `<div class="ach-summary-who">${chips}</div>` : ""}
        <dl class="ln-meta ach-stats">${gpStat(String(total), "unlocked", p)}${gpStat(allPct + "<small>%</small>", "of every achievement in these games")}${gpStat(String(perfect), perfect === 1 ? "game at 100%" : "games at 100%")}${gpStat(String(rows.length), rows.length === 1 ? "game with achievements" : "games with achievements")}</dl>
        <div class="ach-bar ach-summary-bar" role="progressbar" aria-valuemin="0" aria-valuemax="${all}" aria-valuenow="${total}" aria-label="${total} of ${all} achievements"><span style="width:${allPct}%"></span></div></div>
      <div class="ach-cols">
        <section><div class="ln-section-head"><div><h2>Games</h2></div><span class="ach-count ln-num">${rows.length}</span></div>
          <ul class="ach-games">${games.map((r) => { ensureCover(r.it); const pct = Math.round((r.unlocked / r.g.total) * 100); return `<li><button class="ach-gcard${pct === 100 ? " done" : ""}" data-action="open-sheet" ${itemAttr(r.it)}>
            <span class="ach-gcover cover">${coverHTML(r.it)}</span>
            <span class="ach-gbody"><span class="ach-game-name"><b>${esc(r.g.name)}</b></span>
              <span class="ach-game-num"><b class="ln-num">${r.unlocked}</b> of <span class="ln-num">${r.g.total}</span>${pct === 100 ? ", all of them" : `, ${pct}%`}</span>
              <span class="ach-bar" aria-hidden="true"><span style="width:${pct}%"></span></span>
              ${r.shared ? `<span class="ln-tag">${r.g.source === "steam" ? "Steam account" : "Shared"}</span>` : `<span class="ach-gnote">${esc(possessive(pname(who)))} own</span>`}</span></button></li>`; }).join("")}</ul></section>
        <section class="ach-latest"><div class="ln-section-head"><div><h2>Latest unlocked</h2></div></div>
          ${latest.length ? `<ul class="ach-list">${latest.map(({ a, r }) => `<li class="ach">${icon(a)}<div><b>${esc(a.name)}</b><span>${esc(r.g.name)}${a.at && new Date(a.at).getFullYear() > 2000 ? ", " + esc(ago(a.at)) : ""}</span></div><span></span></li>`).join("")}</ul>` : `<p class="home-hint">Nothing unlocked yet.</p>`}</section>
      </div>${msHTML}</div>`;
  }

  // Cloud saves: each player's saves in a folder a cloud app syncs (or a
  // USB drive), up after every session and down before a game starts.
  const PROVIDERS = [
    { name: "OneDrive", match: /onedrive/i, account: "your Microsoft account", url: "https://www.microsoft.com/microsoft-365/onedrive/download" },
    { name: "Google Drive", match: /google drive/i, account: "your Google account", url: "https://www.google.com/drive/download/" },
    { name: "Dropbox", match: /dropbox/i, account: "your Dropbox account", url: "https://www.dropbox.com/install" },
    { name: "iCloud Drive", match: /icloud/i, account: "your Apple ID", url: "https://apps.microsoft.com/detail/9pktq5699m62" },
  ];
  async function loadSaves() { try { await api.savesStatus(); } catch (e) { /* not ready */ } }
  function providerOf(B) {
    const f = (B.folders || []).find((x) => x.path.toLowerCase() === (B.dir || "").toLowerCase());
    return f ? f.name : "Your folder";
  }
  // Cloud saves: where the saves go, every game's state for each player,
  // and the backups of everything.
  function viewCloud() {
    const B = S.backup;
    const head = (actions) => `<header class="lib-head"><div><h1>Cloud saves</h1>
      <p>Every profile's saves go up after each session. On another PC, the newest come down before the game starts.</p></div>${actions ? `<div class="lib-actions">${actions}</div>` : ""}</header>`;
    if (!B) { loadBackupInfo(); return `<div class="cloud-page ln-root">${head("")}${lnInline("", "Reading your cloud folder")}</div>`; }
    const b = (kind, attrs, label) => `<button class="ln-button ln-button--${kind}" ${attrs}>${label}</button>`;
    if (!B.dir) {
      const rows = PROVIDERS.map((pv) => {
        const found = (B.folders || []).filter((f) => pv.match.test(f.name));
        return found.length
          ? found.map((f) => `<div class="cs-option"><div><b>${esc(pv.name)}</b><span class="selectable">${esc(f.path)}</span></div>${b("primary", `data-action="backup-use" data-dir="${esc(f.path)}"`, "Use " + esc(f.name))}</div>`).join("")
          : `<div class="cs-option"><div><b>${esc(pv.name)}</b><span>Install ${esc(pv.name)} and sign in with ${esc(pv.account)}, then come back here.</span></div>${b("secondary", `data-action="open-download" data-url="${esc(pv.url)}"`, "Get " + esc(pv.name))}</div>`;
      }).join("");
      return `<div class="cloud-page ln-root">${head("")}
        <section class="cloud-setup"><div class="ln-section-head"><div><h2>Where should your saves go?</h2><p>Lootnest writes the files to that folder, and the cloud app uploads them with your account.</p></div></div>
          <div class="cs-options">${rows}
            <div class="cs-option"><div><b>USB drive or another folder</b><span>A drive you carry between PCs, a NAS, or any folder.</span></div>${b("secondary", 'data-action="backup-pick"', "Choose a folder")}</div></div>
          <p class="cs-note">Signing in to Microsoft or Google right in Lootnest, without their apps, comes in a later version.</p>
          <div class="bk-restore">${b("ghost", 'data-action="restore-pick"', "Restore from a backup")}<span>On a new PC, or to go back.</span></div></section></div>`;
    }
    const st = B.status, V = S.saves;
    const last = st.running ? "Backing up now" : st.last ? `Last backup ${esc(ago(st.last))}, ${esc(fmtBytes(st.size))}` : "No backup yet";
    const failed = st.error && (!st.last || new Date(st.errorAt) > new Date(st.last));
    const label = { same: "Up to date", up: "Changed here, not uploaded yet", none: "Not in the cloud yet", down: "Newer on another PC", conflict: "Changed on two PCs" };
    let games = "";
    if (B.locked || (V && V.locked)) {
      games = banner("This cloud folder has a password", "Saves from your other PC are encrypted. Enter the same password to sync them here.", b("primary", 'data-action="backup-pw" data-on="1"', "Enter the password"));
    } else if (!B.sync) {
      games = `<p class="hint">Sync is off: only backups of everything are made.</p>`;
    } else if (!V) {
      games = lnInline("", "Checking every game's saves");
    } else if (V.error) {
      games = lnInline("error", esc(V.error));
    } else if (!V.games.length) {
      games = `<div class="ln-glass ln-state">${GI.save}<div><h3>No saves yet</h3><p>Turn on profiles for a game, play it, and its saves go up here.</p></div></div>`;
    } else {
      const byGame = {};
      V.games.forEach((x) => { (byGame[x.gameId] = byGame[x.gameId] || []).push(x); });
      games = `<div class="cs-games">${Object.entries(byGame).map(([gid, rows]) => {
        const g = game(gid);
        if (!g) return "";
        const it = libraryItems().find((x) => x.kind === "game" && x.id === gid);
        if (it) ensureCover(it);
        return `<div class="cs-game"><div class="cs-cover">${it ? `<div class="cover">${coverHTML(it)}</div>` : ""}</div><div class="cs-rows"><b class="cs-name">${esc(g.name)}</b>
          ${rows.map((x) => {
            const pl = player(x.playerId);
            if (!pl) return "";
            const where = x.computer ? `${x.state === "same" ? "Last from" : "From"} ${esc(x.computer)}, ${esc(ago(x.changed))}` : "";
            return `<div class="cs-row state-${esc(x.state)}">${lnAvatar(pl, "20")}<span class="cs-who">${esc(pl.name)}</span>
              <span class="cs-state">${esc(label[x.state] || "")}</span><span class="cs-where">${where}</span>
              ${x.state === "conflict" ? `<span class="cs-fix">${b("secondary ln-button--sm", `data-action="sync-resolve" data-game="${esc(gid)}" data-player="${esc(pl.id)}" data-answer="here"`, "Keep this PC's")}
                ${b("ghost ln-button--sm", `data-action="sync-resolve" data-game="${esc(gid)}" data-player="${esc(pl.id)}" data-answer="cloud"`, `Use ${x.computer ? esc(possessive(x.computer)) : "the cloud's"}`)}</span>` : "<span></span>"}</div>`;
          }).join("")}</div></div>`;
      }).join("")}</div>
      ${V.players && V.players.length ? `<p class="cs-note">The cloud also has saves for ${esc(V.players.join(", "))}. Add a profile with the same name on <b>Profiles</b> (or import it) to use them here.</p>` : ""}`;
    }
    return `<div class="cloud-page ln-root">${head(B.sync && !B.locked ? b("primary", 'data-action="sync-now"', "Sync now") : "")}
      <div class="cloud-cols">
        <div class="cloud-main"><section><div class="ln-section-head"><div><h2>Games</h2></div></div>${games}</section>
          <section class="cloud-backups"><div class="ln-section-head"><div><h2>Backups of everything</h2></div></div>
            <div class="bk-card">
              <div class="bk-row"><div><b class="${failed ? "error-text" : ""}">${failed ? "The last backup didn't work" : last}</b>${failed ? `<p class="error-text">${esc(st.error)}</p>` : `<p>Profiles, games and saves in one file, to restore on a new PC.</p>`}</div>
                ${st.running ? `<button class="ln-button ln-button--secondary ln-button--sm" aria-busy="true"><span class="ln-spinner" aria-hidden="true"></span>Backing up</button>` : b("secondary ln-button--sm", 'data-action="backup-now"', "Back up now")}</div>
              <label class="toggle"><span><b>Back up after every game session</b><p>And when Lootnest starts, if the last backup is over a day old.</p></span>
                <span class="switch"><input type="checkbox" data-backup="auto" ${B.auto ? "checked" : ""}><span></span></span></label>
              <div class="bk-row"><label for="bk-keep"><b>Backups this PC keeps</b></label><input id="bk-keep" class="ln-input" type="number" min="1" max="50" data-backup="keep" value="${B.keep}" style="width:88px"></div>
            </div>
            <div class="bk-restore">${b("ghost ln-button--sm", 'data-action="restore-pick"', "Restore from a backup")}<span>On a new PC, or to go back. What's here now is saved first.</span></div></section>
        </div>
        <aside class="cloud-side">
          <section><div class="ln-section-head"><div><h2>Your cloud folder</h2></div></div>
            <div class="bk-card">
              <div class="bk-row"><div class="bk-where"><span class="label">${esc(providerOf(B))}</span><b class="selectable">${esc(B.target)}</b></div>
                <div class="gpage-actions">${b("ghost ln-button--sm", 'data-action="backup-pick"', "Change")}${b("ghost ln-button--sm", 'data-action="backup-open"', "Open folder")}</div></div>
              <label class="toggle"><span><b>Sync saves between PCs</b><p>Each profile's saves go up after a session and the newest come down before a game starts. If they changed on two PCs, Lootnest asks which to keep.</p></span>
                <span class="switch"><input type="checkbox" data-backup="sync" ${B.sync ? "checked" : ""}><span></span></span></label>
              ${sw("gameSaves", "Keep versions of games outside Steam", "For games without profiles that aren't on Steam (Steam has its own cloud): a new version of their saves when they change and the game is closed. Restore any version from the game's page.", cfg().settings.gameSaves)}
              ${sw("saveDb", "Find save locations online", "Downloads PCGamingWiki's list of where over 20,000 games keep their saves (from the Ludusavi project, on GitHub) about once a week, so new games come with the right locations for profiles and cloud saves. Nothing about you or your games is sent.", cfg().settings.saveDb)}
              <div class="bk-row"><div><b>${B.locked ? "Has a password" : B.encrypted ? "Encrypted with your password" : "Not encrypted"}</b>
                <p>${B.encrypted ? "Only someone with the password can open these files. Use the same password on every PC." : B.locked ? "Enter it to sync on this PC." : "Anyone who can open the folder can read the saves. A password encrypts them (AES-256), on every PC that uses this folder."}</p></div>
                <div class="gpage-actions">${B.encrypted ? b("ghost ln-button--sm", 'data-action="backup-pw" data-on="1"', "Change password") + b("ghost ln-button--sm", 'data-action="backup-pw" data-on="0"', "Remove password") : b("secondary ln-button--sm", 'data-action="backup-pw" data-on="1"', B.locked ? "Enter the password" : "Set a password")}</div></div>
            </div></section>
        </aside>
      </div></div>`;
  }
  async function saveBackup(change) {
    const B = S.backup;
    try { await api.setBackup({ dir: B.dir, auto: B.auto, keep: B.keep, sync: B.sync, ...change }); } catch (e) { toast(errText(e), "error"); }
    loadBackupInfo();
  }
  function viewBackupPassword() {
    const D = S.dialog;
    if (!D.on) {
      return `<div class="scrim" data-action="scrim"><div class="dialog narrow" role="dialog" aria-modal="true" aria-label="Remove the password">
        <div><h2>Remove the password?</h2><p class="sub">New saves and backups in the cloud folder won't be encrypted, on any PC. Backups already made keep their password.</p></div>
        <div class="dialog-foot"><span class="error-text grow">${esc(D.error || "")}</span><button class="btn quiet" data-action="dialog-cancel">Cancel</button><button class="btn primary" data-action="backup-pw-save" ${D.busy ? "disabled" : ""}>Remove password</button></div></div></div>`;
    }
    if (S.backup && S.backup.locked) {
      return `<div class="scrim" data-action="scrim"><div class="dialog narrow" role="dialog" aria-modal="true" aria-label="The cloud folder's password">
        <div><h2>The cloud folder's password</h2><p class="sub">The one you set on your other PC. Lootnest remembers it for this Windows account.</p></div>
        <div class="field"><label for="bk-pw">Password</label><input id="bk-pw" type="password" autocomplete="current-password" maxlength="200"></div>
        <input id="bk-pw2" type="hidden">
        <div class="dialog-foot"><span class="error-text grow" id="bk-error">${esc(D.error || "")}</span><button class="btn quiet" data-action="dialog-cancel">Cancel</button><button class="btn primary" data-action="backup-pw-save" ${D.busy ? "disabled" : ""}>${D.busy ? "Checking" : "Unlock"}</button></div></div></div>`;
    }
    return `<div class="scrim" data-action="scrim"><div class="dialog narrow" role="dialog" aria-modal="true" aria-label="Password">
      <div><h2>Password for your saves</h2><p class="sub">Saves and backups in the cloud folder are encrypted with it. Use the same password on every PC with this folder. Lootnest remembers it for this Windows account, so syncing keeps working by itself.</p></div>
      <div class="field"><label for="bk-pw">Password <span class="hint">at least 8 characters</span></label><input id="bk-pw" type="password" autocomplete="new-password" maxlength="200"></div>
      <div class="field"><label for="bk-pw2">The same password again</label><input id="bk-pw2" type="password" autocomplete="new-password" maxlength="200"></div>
      <p class="warn">If you forget it, nobody can open these files, Lootnest included. Keep it in a password manager.</p>
      <div class="dialog-foot"><span class="error-text grow" id="bk-error">${esc(D.error || "")}</span><button class="btn quiet" data-action="dialog-cancel">Cancel</button><button class="btn primary" data-action="backup-pw-save" ${D.busy ? "disabled" : ""}>${D.busy ? "Setting" : "Set password"}</button></div></div></div>`;
  }
  function viewRestore() {
    const D = S.dialog, I = D.info;
    const foot = (btn) => `<div class="dialog-foot"><span class="error-text grow">${esc(D.error || "")}</span><button class="btn quiet" data-action="restore-close">${D.done ? "Close" : "Cancel"}</button>${btn || ""}</div>`;
    let body;
    if (D.done && D.result) {
      const r = D.result;
      body = `<div><h2>${r.replaced ? "Updated" : "Added"} ${esc(r.name)}</h2><p class="sub">${r.games === 1 ? "Their saves in 1 game are" : `Their saves in ${r.games} games are`} in place${r.added ? `, ${r.added === 1 ? "1 game was" : r.added + " games were"} added` : ""}.${r.skipped ? ` ${r.skipped === 1 ? "1 save location doesn't" : r.skipped + " save locations don't"} match this PC's game and ${r.skipped === 1 ? "was" : "were"} left out.` : ""}</p></div>
        <p class="hint selectable">What was here before is saved in ${esc(D.done)}. To go back, restore that file.</p>${foot()}`;
    } else if (D.done) {
      body = `<div><h2>Restored</h2><p class="sub">Profiles, games and saves from the backup are in place. Before each game's next swap, Lootnest asks whose files are on this PC.</p></div>
        <p class="hint selectable">What was here before is saved in ${esc(D.done)}. To go back, restore that file.</p>${foot()}`;
    } else if (!I || D.busy) {
      body = `<div><h2>${D.applying ? "Restoring" : "Opening the backup"}</h2><p class="sub">${D.applying ? "Saving what's here now, then putting the backup in place." : "Checking every file in it."}</p></div>${foot()}`;
    } else if (I.needPassword) {
      body = `<div><h2>This backup has a password</h2><p class="sub selectable">${esc(D.file)}</p></div>
        <div class="field"><label for="rs-pw">Password</label><input id="rs-pw" type="password" autocomplete="current-password" maxlength="200"></div>${foot(`<button class="btn primary" data-action="restore-pw">Open</button>`)}`;
    } else if (I.error) {
      body = `<div><h2>Can't restore this file</h2><p class="sub">${esc(I.error)}</p></div>${foot()}`;
    } else {
      const when = new Date(I.created).toLocaleString(undefined, { dateStyle: "long", timeStyle: "short" });
      if (I.kind === "player") {
        const name = (I.players || [])[0] || "this profile";
        const here = cfg().players.some((p) => p.name.toLowerCase() === name.toLowerCase());
        body = `<div><h2>Import ${esc(possessive(name))} profile?</h2><p class="sub">Exported on ${esc(I.computer || "a PC")}, ${esc(when)}. Their saves in ${I.games === 1 ? "1 game" : I.games + " games"}, picture and play time.</p></div>
          <p class="sub">${here ? `If this is the same ${esc(name)} as here, their profile is updated with this one; otherwise it's added as another profile.` : "It's added as a new profile."} Games this PC doesn't have yet are added. Everything here now is saved first, so you can go back.</p>
          ${foot(`<button class="btn primary" data-action="restore-apply">Import</button>`)}`;
        return `<div class="scrim"><div class="dialog narrow" role="dialog" aria-modal="true" aria-label="Import a profile">${body}</div></div>`;
      }
      body = `<div><h2>Restore this backup?</h2><p class="sub">Made on ${esc(I.computer || "a PC")}, ${esc(when)}${I.app ? ", Lootnest " + esc(I.app) : ""}.</p></div>
        <dl class="facts"><div><dt>Profiles</dt><dd>${esc((I.players || []).join(", ") || "None")}</dd></div><div><dt>Games with profiles</dt><dd>${I.games}</dd></div></dl>
        <p class="sub">Its profiles, games, saves, pictures and play time replace the ones here. This PC's settings stay. Everything here now is saved first, so you can go back.</p>
        ${foot(`<button class="btn primary" data-action="restore-apply">Restore</button>`)}`;
    }
    return `<div class="scrim"><div class="dialog narrow" role="dialog" aria-modal="true" aria-label="Restore a backup">${body}</div></div>`;
  }

  function viewExport() {
    const D = S.dialog, p = player(D.pid);
    const name = p ? p.name : "";
    if (D.done) {
      return `<div class="scrim" data-action="scrim"><div class="dialog narrow" role="dialog" aria-modal="true" aria-label="Profile exported">
        <div><h2>${esc(possessive(name))} profile is saved</h2><p class="sub selectable">${esc(D.done)}</p></div>
        <p class="sub">On the other PC, open Lootnest, then <b>Profiles</b>, <b>Import a profile</b>, and choose this file.</p>
        <div class="dialog-foot"><button class="btn primary" data-action="dialog-cancel">Done</button></div></div></div>`;
    }
    return `<div class="scrim" data-action="scrim"><div class="dialog narrow" role="dialog" aria-modal="true" aria-label="Export a profile">
      <div><h2>Export ${esc(possessive(name))} profile</h2><p class="sub">One file with their saves in every game they have a profile for, their picture, banner and play time, to take to another PC.</p></div>
      <div class="field"><label for="ex-pw">Password <span class="hint">optional, at least 8 characters</span></label><input id="ex-pw" type="password" autocomplete="new-password" maxlength="200"></div>
      <div class="field"><label for="ex-pw2">The same password again</label><input id="ex-pw2" type="password" autocomplete="new-password" maxlength="200"></div>
      <p class="hint">With a password, the file is encrypted (AES-256) and needs it to be imported.</p>
      <div class="dialog-foot"><span class="error-text grow">${esc(D.error || "")}</span><button class="btn quiet" data-action="dialog-cancel">Cancel</button><button class="btn primary" data-action="export-save" ${D.busy ? "disabled" : ""}>${D.busy ? "Saving" : "Choose a folder and save"}</button></div></div></div>`;
  }

  async function saveSetting(el) {
    const s = JSON.parse(JSON.stringify(cfg().settings));
    let autostart = S.d.autostart;
    const f = el.dataset.setting;
    if (f === "autostart") autostart = el.checked;
    else if (el.type === "checkbox") s[f] = el.dataset.invert ? !el.checked : el.checked;
    else if (el.type === "number") s[f] = parseInt(el.value, 10) || 0;
    else s[f] = el.value;
    try {
      await api.saveSettings({ settings: s, autostart });
      if (f === "downloadCovers" || f === "gridDbKey") { S.covers = {}; S.icons = {}; S.art = {}; artWanted.clear(); pendingCovers.clear(); }
      if (f === "steamApiKey" || f === "achRarity") { S.ach = {}; achAsked.clear(); S.allAch = null; }
      await refresh();
    } catch (e) { toast(errText(e), "error"); await refresh(); }
  }

  // --------------------------------------------------------------- actions --
  const actions = {
    nav(el) {
      if (S.view === "edit" && !confirmLeaveEdit()) return;
      S.view = el.dataset.view; S.menuFor = null; S.edit = null; S.animateView = true;
      if (S.view === "cloud") { loadBackupInfo(); loadSaves(); }
      if (S.view === "library") S.animateCards = true;
      if (S.view !== "players") S.players = null;
      render(); $("#view").scrollTop = 0;
    },
    "add-game": startAddDialog,
    "add-steam": (el) => addSteam(el.dataset.app),
    "add-exe": addExe,
    play: (el) => { const p = seated(); if (p) actions["play-as"]({ dataset: { game: el.dataset.game, player: p.id } }); else openPicker(el.dataset.game, "play"); },
    seat(el) { const p = el.dataset.player; setSeat(S.seat === p ? "" : p); },
    "seat-set"(el) { setSeat(el.dataset.player); },
    async "pt-make"(el) {
      const it = itemFor(el);
      if (!it) return;
      const key = recentKey(it), inp = $("#pt-new"), P = S.points[key] || (S.points[key] = {});
      P.busy = true; render();
      try { await api.makeSavePoint(key, inp ? inp.value : ""); } catch (e) { P.busy = false; toast(errText(e), "error"); render(); }
    },
    "pt-all"(el) { const it = itemFor(el); if (it) { S.dialog = { kind: "points", ik: it.kind, iid: it.id }; renderOverlay(); focusDrawer(); } },
    async "pt-restore"(el) {
      const it = itemFor(el);
      if (!it) return;
      const key = recentKey(it), back = S.dialog && S.dialog.kind === "points" ? { ...S.dialog } : null;
      const ok = await confirmDialog({ title: `Go back to ${el.dataset.name}?`, body: `The saves of ${it.name} become what they were then. The saves as they are now are kept as a save point first, so you can come back to them too.`, ok: "Go back to it" });
      if (back) { S.dialog = back; renderOverlay(); }
      if (!ok) return;
      const P = S.points[key] || (S.points[key] = {});
      P.busy = true; render(); if (back) renderOverlay();
      try { await api.restoreSavePoint(key, el.dataset.id); } catch (e) { P.busy = false; toast(errText(e), "error"); render(); if (back) renderOverlay(); }
    },
    async "pt-branch"(el) {
      const it = itemFor(el);
      if (!it) return;
      const key = recentKey(it), back = S.dialog && S.dialog.kind === "points" ? { ...S.dialog } : null;
      const who = el.dataset.player && player(el.dataset.player) ? pname(el.dataset.player) : "this profile";
      const ok = await confirmDialog({ title: `Start a new run from ${el.dataset.name}?`, body: `A new run of ${who} starts with the saves of that moment: a profile of its own, with its own saves from then on. Everyone else's stay as they are.`, ok: "Start the run" });
      if (back) { S.dialog = back; renderOverlay(); }
      if (!ok) return;
      try { const id = await api.branchSavePoint(key, el.dataset.id, ""); await refresh(); toast(`New run: ${pname(id)}. Choose it when you play.`); } catch (e) { toast(errText(e), "error"); }
    },
    "pt-rename"(el) { if (S.dialog && S.dialog.kind === "points") { S.dialog.editing = el.dataset.id; renderOverlay(); } },
    "pt-rename-cancel"() { if (S.dialog) { S.dialog.editing = null; renderOverlay(); } },
    async "pt-rename-save"(el) {
      const it = itemFor(el), inp = $("#pt-rename");
      if (!it || !inp) return;
      const key = recentKey(it);
      try { await api.renameSavePoint(key, el.dataset.id, inp.value); S.dialog.editing = null; api.savePoints(key).catch(() => {}); } catch (e) { toast(errText(e), "error"); }
    },
    async "pt-delete"(el) {
      const it = itemFor(el);
      if (!it) return;
      const key = recentKey(it), back = S.dialog && S.dialog.kind === "points" ? { ...S.dialog } : null;
      const ok = await confirmDialog({ title: `Delete ${el.dataset.name}?`, body: "This save point's copy of the saves goes. The game's saves now aren't touched.", ok: "Delete it", danger: true });
      if (back) { S.dialog = back; renderOverlay(); }
      if (!ok) return;
      try { await api.deleteSavePoint(key, el.dataset.id); } catch (e) { toast(errText(e), "error"); }
    },
    async "offline-toggle"(el) {
      const it = itemFor(el);
      if (!it) return;
      const on = el.dataset.on === "1";
      const ok = await confirmDialog(on
        ? { title: `Keep ${it.name} offline?`, body: "Windows Firewall will stop its programs from connecting to the internet, and to other PCs on your network. Online play, updates from inside the game and anything else online stop working until you let it go online again. Windows asks first, as administrator.", ok: "Keep it offline" }
        : { title: `Let ${it.name} go online again?`, body: "Lootnest takes away the Windows Firewall rules it made for this game. Windows asks first, as administrator.", ok: "Let it go online" });
      if (!ok) return;
      try { await api.setOffline(recentKey(it), on); toast("Waiting for Windows' prompt"); } catch (e) { toast(errText(e), "error"); }
    },
    "ach-doctor"(el) {
      const it = itemFor(el);
      if (!it) return;
      const key = recentKey(it);
      delete S.doctor[key];
      const open = S.dialog && S.dialog.kind === "achdoc";
      S.dialog = { kind: "achdoc", ik: it.kind, iid: it.id };
      renderOverlay(); if (!open) focusDrawer();
      api.achDoctor(key, it.appId, docDir(it), it.name).catch((e) => { S.doctor[key] = { error: errText(e) }; renderOverlay(); });
    },
    async "doc-name"(el) {
      const it = itemFor(el);
      if (!it) return;
      try { await api.setNameInGame(recentKey(it), docDir(it), el.dataset.player); toast(`The name in ${it.name} is now ${pname(el.dataset.player)}`); actions["ach-doctor"](el); }
      catch (e) { toast(errText(e), "error"); }
    },
    async "show-log-nav"() {
      S.dialog = null; renderOverlay(); S.setTab = "data"; S.view = "settings"; S.menuFor = null; S.animateView = true;
      try { S.logText = (await api.log()) || "Nothing yet."; } catch (e) { S.logText = errText(e); }
      render(); $("#view").scrollTop = 0;
    },
    "ach-all"(el) { const it = itemFor(el); if (it) { S.dialog = { kind: "achievements", ik: it.kind, iid: it.id }; renderOverlay(); focusDrawer(); } },
    manage(el) { const it = itemFor(el); if (it) { S.dialog = { kind: "manage", ik: it.kind, iid: it.id }; renderOverlay(); focusDrawer(); } },
    "ach-who"(el) { S.achWho[el.dataset.key] = el.dataset.player; render(); },
    "ach-page"(el) { S.achPage = el.dataset.player; render(); },
    "status-who"(el) { S.statusWho[el.dataset.key] = el.dataset.player; render(); },
    async "set-status"(el) {
      const { key, player: pid, status } = el.dataset, cur = ((S.d.shelf || {})[pid] || {})[key] || {};
      const next = cur.status === status ? "" : status; // pressing it again clears it
      try { await api.setShelf(pid, key, { status: next, rating: cur.rating || 0, note: cur.note || "" }); } catch (e) { toast(errText(e), "error"); return; }
      await refresh();
    },
    async "tag-add"(el) {
      const key = el.dataset.key, i = $("#tag-new"), t = ((i && i.value) || "").trim();
      if (!t) { if (i) i.focus(); return; }
      const it = itemByKey(key), cur = it ? tagsOf(it) : [];
      try { await api.setTags(key, cur.concat([t])); } catch (e) { toast(errText(e), "error"); return; }
      await refresh();
      const n = $("#tag-new"); if (n) n.focus();
    },
    async "tag-remove"(el) {
      const key = el.dataset.key, it = itemByKey(key);
      try { await api.setTags(key, (it ? tagsOf(it) : []).filter((t) => t !== el.dataset.tag)); } catch (e) { toast(errText(e), "error"); return; }
      if (S.tag === el.dataset.tag && !Object.entries(cfg().tags || {}).some(([k, ts]) => k !== key && ts.includes(el.dataset.tag))) S.tag = "";
      await refresh();
    },
    async "ach-shared"(el) {
      const id = el.dataset.game, shared = el.dataset.on === "1", g = game(id);
      if (shared && !(await confirmDialog({ title: "Share one set of achievements?", body: `Everyone who plays ${g ? g.name : "this game"} will see the same achievements, the ones on the PC now. Each profile's own stay with it, and come back if you change your mind.`, ok: "Share one set" }))) return;
      try { await api.setSharedAchievements(id, shared); } catch (e) { toast(errText(e), "error"); return; }
      await refresh();
      achAsked.delete("game:" + id);
      toast(shared ? "Achievements are shared now." : "Each profile gets its own achievements from the next swap.");
    },
    "seat-menu"(el) { seatMenu(el); },
    "open-sheet": (el) => openSheet(el.dataset.game ? "game" : "found", el.dataset.game || el.dataset.key, el.closest(".card")),
    "back-library": () => backToLibrary(),
    "open-player"(el) { S.playerPage = el.dataset.player; S.view = "player"; S.animateView = true; render(); $("#view").scrollTop = 0; },
    "edit-profile"(el) { const p = player(el.dataset.player); if (!p) return; S.dialog = { kind: "profile", p: JSON.parse(JSON.stringify(p)), q: "", tab: "look" }; renderOverlay(); },
    "pe-tab"(el) { S.dialog.tab = el.dataset.tab; renderOverlay(); },
    "pe-set"(el) { S.dialog.p[el.dataset.f] = el.dataset.v; renderOverlay(); },
    "pe-color"(el) { S.dialog.p.color = el.dataset.color; renderOverlay(); },
    "pe-banner"(el) {
      const D = S.dialog, v = el.dataset.v;
      if (v === "image") { if (!(S.d.banners || {})[D.p.id]) return actions["banner-pick"](); D.p.banner = "image"; }
      else if (v === "game") { const f = (D.p.favorites || [])[0] || (libraryItems()[0] && recentKey(libraryItems()[0])) || ""; D.p.banner = D.p.banner && D.p.banner !== "image" && !D.p.banner.startsWith("style:") ? D.p.banner : f; }
      else D.p.banner = "";
      renderOverlay();
    },
    async "banner-pick"() {
      const D = S.dialog;
      const r = await browse("image");
      if (!r.path) { if (r.error) toast(r.error, "error"); return; }
      D.busy = "banner"; renderOverlay();
      try { await api.setBanner(D.p.id, r.path); } catch (e) { D.busy = false; renderOverlay(); toast(errText(e), "error"); }
    },
    async "banner-remove"() {
      try { await api.removeBanner(S.dialog.p.id); if (S.dialog.p.banner === "image") S.dialog.p.banner = ""; await refresh(); } catch (e) { toast(errText(e), "error"); }
    },
    "pe-fav"(el) {
      const f = S.dialog.p.favorites = S.dialog.p.favorites || [];
      const i = f.indexOf(el.dataset.k);
      if (i >= 0) f.splice(i, 1); else if (f.length < 6) f.push(el.dataset.k);
      renderOverlay();
    },
    "pe-fav-move"(el) {
      const f = S.dialog.p.favorites, i = +el.dataset.i, j = i + +el.dataset.d;
      if (j < 0 || j >= f.length) return;
      [f[i], f[j]] = [f[j], f[i]]; renderOverlay();
    },
    "pe-sec"(el) {
      const p = S.dialog.p, on = sectionsOf(p).slice(), i = on.indexOf(el.dataset.k);
      if (i >= 0) { if (on.length > 1) on.splice(i, 1); } else on.push(el.dataset.k);
      p.sections = on; renderOverlay();
    },
    "pe-sec-move"(el) {
      const p = S.dialog.p, on = sectionsOf(p).slice(), i = on.indexOf(el.dataset.k), j = i + +el.dataset.d;
      if (i < 0 || j < 0 || j >= on.length) return;
      [on[i], on[j]] = [on[j], on[i]]; p.sections = on; renderOverlay();
    },
    async "avatar-pick"() {
      const D = S.dialog;
      const r = await browse("image");
      if (!r.path) { if (r.error) toast(r.error, "error"); return; }
      D.busy = "avatar"; renderOverlay();
      try { await api.setAvatar(D.p.id, r.path); } catch (e) { D.busy = false; renderOverlay(); toast(errText(e), "error"); }
    },
    async "avatar-remove"() { try { await api.removeAvatar(S.dialog.p.id); S.dialog.p.avatar = false; await refresh(); } catch (e) { toast(errText(e), "error"); } },
    async "pe-save"() {
      const D = S.dialog, p = { ...D.p };
      p.avatar = !!(S.d.avatars || {})[p.id];
      if (p.banner === "image" && !(S.d.banners || {})[p.id]) p.banner = "";
      p.bannerY = Math.round(+p.bannerY || 0);
      const ps = cfg().players.map((x) => (x.id === p.id ? p : x));
      try { await api.savePlayers(ps); S.dialog = null; S.players = null; await refresh(); toast("Profile saved"); }
      catch (e) { const t = $("#pe-error"); if (t) t.textContent = errText(e); }
    },
    async "gs-add"(el) {
      const key = el.dataset.key, G = S.gsaves[key] || {};
      const r = await browse(el.dataset.kind === "file" ? "file" : "folder");
      if (!r.path) { if (r.error) toast(r.error, "error"); return; }
      const now = G.source === "you" ? (G.locations || []).map((l) => l.path) : [];
      try { await api.setGameSaves(key, [...now, r.tokenized || r.path]); await refresh(); delete S.gsaves[key]; gsAsked.delete(key); render(); }
      catch (e) { toast(errText(e), "error"); }
    },
    async "gs-remove"(el) {
      const key = el.dataset.key, G = S.gsaves[key] || {};
      const rest = (G.locations || []).map((l) => l.path).filter((p) => p !== el.dataset.path);
      try { await api.setGameSaves(key, rest); await refresh(); delete S.gsaves[key]; gsAsked.delete(key); render(); }
      catch (e) { toast(errText(e), "error"); }
    },
    "gs-own"(el) { const G = S.gsaves[el.dataset.key]; if (G) { G.source = "you"; G.locations = []; G.versions = G.versions || []; render(); } },
    "gs-open"(el) { api.open("gameSaves", el.dataset.key).catch((e) => toast(errText(e), "error")); },
    "gs-pcgw"(el) { api.open("pcgw", el.dataset.name).catch((e) => toast(errText(e), "error")); },
    async "gs-now"(el) {
      const G = S.gsaves[el.dataset.key];
      if (G) G.busy = true; render();
      try { await api.gameSaveNow(el.dataset.key); } catch (e) { if (G) G.busy = false; toast(errText(e), "error"); render(); }
    },
    async "gs-restore"(el) {
      const when = new Date(el.dataset.when).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
      const local = !!el.dataset.local;
      const body = local ? `The copy from ${when} goes back into the game's folder, over the saves there now. Those are kept in Lootnest's backups first.`
        : `The saves from ${when} on ${el.dataset.pc} replace the ones on this PC. The saves here now are kept in Lootnest's backups first.`;
      const ok = await confirmDialog({ title: "Restore these saves?", body, ok: "Restore" });
      if (!ok) return;
      const G = S.gsaves[el.dataset.key];
      if (G) G.busy = true; render();
      try { await (local ? api.gameSaveRestoreLocal : api.gameSaveRestore)(el.dataset.key, el.dataset.id); } catch (e) { if (G) G.busy = false; toast(errText(e), "error"); render(); }
    },
    async "find-on"() {
      const s = JSON.parse(JSON.stringify(cfg().settings));
      s.findGames = true;
      try { await api.saveSettings({ settings: s, autostart: S.d.autostart }); await refresh(); setTimeout(() => { const q = $("#find-q"); if (q) q.focus(); }, 30); } catch (e) { toast(errText(e), "error"); }
    },
    "find-open"(el) {
      const appId = +el.dataset.app;
      S.dialog = { kind: "offer", appId, name: el.dataset.name };
      const R = S.offers[appId];
      if (!R || R.error) { delete S.offers[appId]; api.gameOffer(appId, el.dataset.name).catch((e) => { S.offers[appId] = { error: errText(e) }; renderOverlay(); }); }
      renderOverlay(); focusDrawer();
    },
    async "copy-carry"(el) {
      const old = el.dataset.old, nw = el.dataset.new;
      const ok = await confirmDialog({ title: `Carry ${el.dataset.name} over to its new copy?`, body: "Its saves go into the new copy's folder, at the same places (what the new copy has there is kept in backups first). Its playtime, status, collections and settings move to the new copy. The old folder isn't touched.", ok: "Carry over" });
      if (!ok) return;
      S.carried[old] = { busy: true, name: el.dataset.name }; render();
      try { await api.carryOver(old, nw); } catch (e) { S.carried[old] = { error: errText(e) }; toast(errText(e), "error"); render(); }
    },
    async "copy-dismiss"(el) {
      try { await api.dismissCopy(el.dataset.old, el.dataset.new); S.copies = (S.copies || []).filter((o) => !(o.old.key === el.dataset.old && o.new.key === el.dataset.new)); updateNavCount(); render(); } catch (e) { toast(errText(e), "error"); }
    },
    async "copy-delete-old"(el) {
      const ok = await confirmDialog({ title: "Delete the old folder?", body: "Its saves are copied to Lootnest's backups first. Everything else in the old folder is deleted.", ok: "Delete it", danger: true });
      if (!ok) return;
      const C = S.carried[el.dataset.old] || {}; C.busy = true; S.carried[el.dataset.old] = C; render();
      try { await api.deleteOldCopy(el.dataset.old); } catch (e) { C.busy = false; toast(errText(e), "error"); render(); }
    },
    async "upd-keep"(el) { try { await api.keepSavesNow(el.dataset.key); toast("Copying its saves"); } catch (e) { toast(errText(e), "error"); } },
    async "report-problem"() { try { await api.reportProblem(); toast("Copied. Paste it into the new issue."); } catch (e) { toast(errText(e), "error"); } },
    "test-toast"() { api.testToast().catch((e) => toast(errText(e), "error")); },
    "upd-another"() { const D = S.dialog; if (D) { Object.assign(D, { path: null, plan: null, check: null, done: null, undone: false, error: "", step: "", full: undefined }); renderOverlay(); } },
    "upd-full"(el) { const D = S.dialog; if (D) D.full = el.checked; },
    async "upd-run"() {
      const D = S.dialog;
      if (!D || !D.plan) return;
      D.busy = true; D.step = ""; D.error = ""; renderOverlay();
      try { await api.updateInstaller(D.key, D.path, D.full); } catch (e) { D.busy = false; D.error = errText(e); renderOverlay(); }
    },
    "upd-start"(el) { S.dialog = { kind: "update", key: el.dataset.key, name: el.dataset.name }; renderOverlay(); focusDrawer(); },
    async "upd-pick"(el) {
      const D = S.dialog;
      if (!D || D.kind !== "update") return;
      const r = await browse(el.dataset.kind, "");
      if (r.cancelled || !r.path) return;
      if (r.error) { toast(r.error, "error"); return; }
      Object.assign(D, { path: r.path, plan: null, error: "", busy: true, done: null });
      renderOverlay();
      api.updatePlan(D.key, r.path).catch((e) => { D.busy = false; D.error = errText(e); renderOverlay(); });
    },
    async "upd-apply"() {
      const D = S.dialog;
      if (!D || !D.plan) return;
      D.busy = true; D.error = ""; renderOverlay();
      try { await api.updateApply(D.key, D.path); } catch (e) { D.busy = false; D.error = errText(e); renderOverlay(); }
    },
    async "upd-undo"(el) {
      const ok = await confirmDialog({ title: `Undo the update of ${el.dataset.name}?`, body: "The files it added go, and the ones it replaced come back. The saves stay as they are now.", ok: "Undo" });
      if (!ok) return;
      if (!S.dialog || S.dialog.kind !== "update") S.dialog = { kind: "update", key: el.dataset.key, name: el.dataset.name, busy: true };
      else S.dialog.busy = true;
      renderOverlay();
      try { await api.updateUndo(el.dataset.key, el.dataset.id); } catch (e) { S.dialog.busy = false; S.dialog.error = errText(e); renderOverlay(); }
    },
    "upd-open-game"(el) {
      const [kind, ...rest] = el.dataset.key.split(":"), id = rest.join(":");
      openSheet(kind === "game" ? "game" : "found", id, null);
    },
    "store-retry"() { S.front = null; const r = $("#find-results"); if (r) r.innerHTML = findResults(); },
    "gq-open"(el) { gqOpen(+el.dataset.i); },
    "gq-all"() { gqAll(); },
    "find-store"(el) { api.open("store", el.dataset.url).catch((e) => toast(errText(e), "error")); },
    "find-library"(el) {
      const o = el.dataset.owned || "", [kind, ...rest] = o.split(":"), id = rest.join(":");
      const it = libraryItems().find((x) => (kind === "game" ? x.kind === "game" : x.kind === "found") && x.id === id);
      if (!it) { toast("That game isn't in the Library any more.", "error"); return; }
      S.dialog = null; renderOverlay();
      openSheet(it.kind, it.id, null);
    },
    async "savedb-on"() {
      const s = JSON.parse(JSON.stringify(cfg().settings));
      s.saveDb = true;
      try { await api.saveSettings({ settings: s, autostart: S.d.autostart }); toast("Downloading the save locations list"); await refresh(); } catch (e) { toast(errText(e), "error"); }
    },
    async "sync-now"() { S.syncAsked = true; try { await api.syncNow(); } catch (e) { S.syncAsked = false; toast(errText(e), "error"); } },
    async "sync-resolve"(el) {
      try { S.syncAsked = true; await api.syncResolve(el.dataset.game, el.dataset.player, el.dataset.answer); } catch (e) { S.syncAsked = false; toast(errText(e), "error"); }
    },
    "open-download"(el) { api.open("link", el.dataset.url).catch((e) => toast(errText(e), "error")); },
    "backup-use"(el) { saveBackup({ dir: el.dataset.dir, auto: true }).then(() => toast("Saves go to " + el.dataset.dir)); },
    async "backup-pick"() {
      const r = await browse("backupDir");
      if (!r.path) { if (r.error) toast(r.error, "error"); return; }
      const first = !S.backup.dir;
      await saveBackup({ dir: r.path, ...(first ? { auto: true } : {}) });
    },
    "backup-open": () => api.open("backupFolder").catch((e) => toast(errText(e), "error")),
    async "backup-now"() { S.backupAsked = true; try { await api.backupNow(); } catch (e) { S.backupAsked = false; toast(errText(e), "error"); } loadBackupInfo(); },
    "backup-pw"(el) { S.dialog = { kind: "backup-pw", on: el.dataset.on === "1" }; renderOverlay(); setTimeout(() => { const f = $("#bk-pw"); if (f) f.focus(); }, 30); },
    async "backup-pw-save"() {
      const D = S.dialog;
      let pw = "";
      if (D.on) {
        pw = $("#bk-pw").value;
        const unlock = S.backup && S.backup.locked;
        if (!unlock && pw !== $("#bk-pw2").value) { $("#bk-error").textContent = "The two passwords aren't the same."; return; }
        if ([...pw].length < 8) { $("#bk-error").textContent = "Use at least 8 characters."; return; }
      }
      D.busy = true; D.error = ""; renderOverlay();
      try { await api.setBackupPassword(pw); } catch (e) { D.busy = false; D.error = errText(e); renderOverlay(); }
    },
    async "restore-pick"(el) {
      const r = await browse("backup");
      if (!r.path) { if (r.error) toast(r.error, "error"); return; }
      S.dialog = { kind: "restore", file: r.path, busy: true, mode: (el && el.dataset.mode) || "restore" }; renderOverlay();
      try { await api.restoreOpen(r.path, ""); } catch (e) { S.dialog.busy = false; S.dialog.info = { error: errText(e) }; renderOverlay(); }
    },
    async "restore-pw"() {
      const D = S.dialog, pw = $("#rs-pw").value;
      if (!pw) return;
      D.busy = true; D.triedPassword = true; D.error = ""; renderOverlay();
      try { await api.restoreOpen(D.file, pw); } catch (e) { D.busy = false; D.error = errText(e); renderOverlay(); }
    },
    async "restore-apply"() {
      const D = S.dialog;
      D.busy = true; D.applying = true; D.error = ""; renderOverlay();
      try { await (D.info && D.info.kind === "player" ? api.importApply() : api.restoreApply()); } catch (e) { D.busy = false; D.applying = false; D.error = errText(e); renderOverlay(); }
    },
    "export-player"(el) { S.dialog = { kind: "export", pid: el.dataset.player }; renderOverlay(); },
    async "export-save"() {
      const D = S.dialog, pw = $("#ex-pw").value;
      if (pw && pw !== $("#ex-pw2").value) { D.error = "The two passwords aren't the same."; renderOverlay(); return; }
      if (pw && [...pw].length < 8) { D.error = "Use at least 8 characters, or no password."; renderOverlay(); return; }
      const r = await browse("exportDir", (S.backup && S.backup.target) || "");
      if (!r.path) { if (r.error) toast(r.error, "error"); return; }
      D.busy = true; D.error = ""; renderOverlay();
      try { await api.exportPlayer(D.pid, r.path, pw); } catch (e) { D.busy = false; D.error = errText(e); renderOverlay(); }
    },
    "restore-close"() { const D = S.dialog; S.dialog = null; renderOverlay(); if (D && !D.done) api.restoreCancel().catch(() => {}); },
    "launch-from-exe"() { const g = S.edit.game; g.launch = { ...g.launch, type: "exe", uri: "" }; render(); },
    density(el) { S.density = el.dataset.size; setPref("density", S.density); render(); },
    "open-news": (el) => api.open("news", el.dataset.url).catch((e) => toast(errText(e), "error")),
    "open-link": (el) => api.open("link", el.dataset.url).catch((e) => toast(errText(e), "error")),
    "read-news"(el) { S.dialog = { kind: "news", key: el.dataset.key, i: +el.dataset.i }; renderOverlay(); setTimeout(() => { const b = $(".reader .btn.quiet.icon"); if (b) b.focus({ preventScroll: true }); }, 30); },
    async "news-on"() {
      const s = JSON.parse(JSON.stringify(cfg().settings));
      s.gameNews = true;
      try { await api.saveSettings({ settings: s, autostart: S.d.autostart }); await refresh(); } catch (e) { toast(errText(e), "error"); }
    },
    "play-as"(el) { openPicker(el.dataset.game, "play"); S.picker.playerId = el.dataset.player; S.picker.from = S.d.active[el.dataset.game] || ""; runPick(); },
    async "play-found"(el) {
      const f = (S.found.games || []).find((x) => x.key === el.dataset.key);
      if (S.dialog && S.dialog.kind === "sheet") closeDialog(false);
      try { await api.launchFound(el.dataset.key); toast(`Starting ${f ? f.name : "the game"}`); } catch (e) { toast(errText(e), "error"); }
    },
    filter(el) { S.filter = el.dataset.filter; render(); },
    async "close-game"(el) {
      const g = game(el.dataset.game), s = g && session(g.id);
      if (!g) return;
      const n = s ? pname(s.playerId) : "";
      const ok = await confirmDialog({ title: `Close ${g.name}?`, body: `The game is asked to quit, as if you closed its window. Anything it hasn't saved yet may be lost.${n ? ` Once it closes, ${n}'s progress goes back to their profile.` : ""}`, ok: "Close the game" });
      if (!ok) return;
      try { await api.closeGame(g.id); toast(`Asked ${g.name} to close`); } catch (e) { toast(errText(e), "error"); }
    },
    who(el) { S.who = S.who === el.dataset.player ? "" : el.dataset.player; render(); },
    async "covers-on"() {
      const s = JSON.parse(JSON.stringify(cfg().settings));
      s.downloadCovers = true;
      try { await api.saveSettings({ settings: s, autostart: S.d.autostart }); S.covers = {}; S.icons = {}; S.art = {}; artWanted.clear(); pendingCovers.clear(); await refresh(); } catch (e) { toast(errText(e), "error"); }
    },
    "covers-later"() { S.hideCoverHint = true; render(); },
    async "updates-yes"() {
      const s = JSON.parse(JSON.stringify(cfg().settings));
      s.checkUpdates = true; s.autoInstallUpdates = true; s.updatesAsked = true;
      try { await api.saveSettings({ settings: s, autostart: S.d.autostart }); await refresh(); toast("Lootnest keeps itself up to date. You can change this in Settings."); api.checkUpdate().catch(() => {}); } catch (e) { toast(errText(e), "error"); }
    },
    async "updates-no"() {
      const s = JSON.parse(JSON.stringify(cfg().settings));
      s.updatesAsked = true;
      try { await api.saveSettings({ settings: s, autostart: S.d.autostart }); await refresh(); } catch (e) { toast(errText(e), "error"); }
    },
    "update-check"() { S.update = { checking: true }; render(); api.checkUpdate().catch((e) => { S.update = {}; toast(errText(e), "error"); render(); }); },
    async "update-now"() {
      try { await api.installUpdate(); } catch (e) { toast(errText(e), "error"); }
    },
    "update-notes"() { api.open("release").catch((e) => toast(errText(e), "error")); },
    "load-only": (el) => openPicker(el.dataset.game, "load"),
    menu(el) { S.menuFor = S.menuFor === el.dataset.game ? null : el.dataset.game; render(); const m = $(".menu button"); if (m) m.focus(); },
    "edit-game": (el) => { if (S.dialog && S.dialog.kind === "sheet") S.dialog = null; editGame(el.dataset.game); },
    async shortcut(el) {
      S.menuFor = null; render();
      try { await api.shortcut(el.dataset.game); toast("Shortcut added to your desktop"); } catch (e) { toast(errText(e), "error"); }
    },
    "open-path": (el) => { S.menuFor = null; render(); api.open("path", el.dataset.path).catch((e) => toast(errText(e), "error")); },
    "open-backups": (el) => { S.menuFor = null; render(); api.open("backups", el.dataset.game).catch((e) => toast(errText(e), "error")); },
    async "remove-game"(el) {
      S.menuFor = null; render();
      const g = game(el.dataset.game);
      const ok = await confirmDialog({ title: `Turn off profiles for ${g.name}?`, body: "The game stays in your library. Everyone's profiles are moved to the “removed” folder in Lootnest's data, and the game keeps the files of whoever played last.", ok: "Turn off profiles" });
      if (!ok) return;
      try { await api.deleteGame(g.id); toast(`Profiles are off for ${g.name}`); await refresh(); loadFound(); } catch (e) { toast(errText(e), "error"); }
    },
    "np-open"() { S.picker.creating = true; S.picker.newError = ""; renderOverlay(); setTimeout(() => { const i = $("#np-name"); if (i) i.focus(); }, 20); },
    "np-cancel"() { S.picker.creating = false; S.picker.newName = ""; S.picker.newError = ""; renderOverlay(); },
    async "np-create"() {
      const P = S.picker, i = $("#np-name"), name = ((i && i.value) || "").trim(), ofSel = $("#np-of");
      if (!P) return;
      P.newName = name;
      P.newOf = (ofSel && ofSel.value) || "";
      const of = player(P.newOf);
      const fail = (t) => { P.newError = t; renderOverlay(); setTimeout(() => { const x = $("#np-name"); if (x) x.focus(); }, 20); };
      if (!name) return fail("Give it a name.");
      if (cfg().players.some((p) => p.name.trim().toLowerCase() === name.toLowerCase())) return fail(`There's already a profile called ${name}.`);
      const used = new Set(cfg().players.map((p) => p.color.toLowerCase()));
      const list = cfg().players.map((p) => ({ ...p })).concat([{ id: "", name, of: of ? of.id : "", color: of ? of.color : S.d.colors.find((c) => !used.has(c.toLowerCase())) || S.d.colors[cfg().players.length % S.d.colors.length] }]);
      try { await api.savePlayers(list); } catch (e) { return fail(errText(e)); }
      await refresh();
      const np = cfg().players.find((p) => p.name === name);
      if (!S.picker || !np) return;
      S.picker.creating = false; S.picker.newName = ""; S.picker.newError = ""; S.picker.newOf = "";
      S.picker.playerId = np.id; S.picker.owner = ""; S.picker.newMode = ""; S.picker.copyFrom = ""; S.picker.from = S.d.active[S.picker.gameId] || "";
      runPick();
    },
    pick(el) { S.picker.playerId = el.dataset.player; S.picker.owner = ""; S.picker.newMode = ""; S.picker.copyFrom = ""; S.picker.from = S.d.active[S.picker.gameId] || ""; runPick(); },
    "answer-owner"(el) { S.picker.owner = el.dataset.player; S.picker.ask = null; runPick(); },
    "answer-new"(el) { S.picker.newMode = el.dataset.mode; S.picker.copyFrom = el.dataset.from || ""; S.picker.ask = null; runPick(); },
    "answer-cloud"(el) { S.picker.cloud = el.dataset.answer; S.picker.ask = null; runPick(); },
    "picker-back"() { S.picker.ask = null; S.picker.owner = ""; S.picker.newMode = ""; render(); },
    "picker-close": closePicker,
    "dialog-ok": () => closeDialog(true),
    async "fix-blocked"() {
      const D = S.dialog;
      if (!D || D.kind !== "blocked" || D.busy) return;
      D.busy = true; renderOverlay();
      try { await api.fixBlocked(D.gameId, D.playerId); } catch (e) { D.busy = false; renderOverlay(); toast(errText(e), "error"); }
    },
    "dialog-cancel": () => closeDialog(false),
    "game-settings"(el) { if (S.dialog && S.dialog.kind === "sheet") S.dialog = null; openGameSettings(el.dataset.key); },
    async "game-settings-save"() {
      const D = S.dialog;
      if (!D || D.kind !== "game-settings") return;
      const name = $("#gs-name").value, args = $("#gs-args").value;
      D.busy = true; renderOverlay();
      try {
        await api.saveLibraryPref(D.key, name, args);
        await loadFound();
        if (S.dialog === D) { S.dialog = null; renderOverlay(); }
        toast("Saved");
      } catch (e) { D.busy = false; renderOverlay(); toast(errText(e), "error"); }
    },
    async "open-game-folder"(el) {
      try { await (el.dataset.game ? api.openGameFolder("game", el.dataset.game) : api.openGameFolder("found", el.dataset.key)); } catch (e) { toast(errText(e), "error"); }
    },
    async "found-shortcut"(el) {
      try { await api.foundShortcut(el.dataset.key); toast("Shortcut added to your desktop"); } catch (e) { toast(errText(e), "error"); }
    },
    async "steam-verify"(el) {
      try { await api.steamVerify(+el.dataset.app); toast("Steam is checking the game's files. It downloads any that are missing or damaged."); } catch (e) { toast(errText(e), "error"); }
    },
    "move-open"(el) { if (S.dialog && S.dialog.kind === "sheet") S.dialog = null; openMove(+el.dataset.app, el.dataset.name); },
    "move-pick"(el) { if (S.dialog && S.dialog.kind === "move") { S.dialog.pick = el.dataset.lib; renderOverlay(); } },
    async "move-go"() {
      const D = S.dialog;
      if (!D || D.kind !== "move" || !D.pick) return;
      try {
        await api.moveGame(D.appId, D.pick);
        if (S.dialog !== D) return; // already done
        S.moving = Object.assign({ appId: D.appId, name: D.name, step: "", done: 0, total: 0 }, S.moving && S.moving.appId === D.appId ? S.moving : {});
        renderOverlay();
      } catch (e) { toast(errText(e), "error"); }
    },
    scrim(el, ev) { if (ev.target === el && S.dialog && S.dialog.kind !== "welcome") closeDialog(false); },
    "edit-cancel"() { if (!confirmLeaveEdit()) return; S.edit = null; S.view = "library"; S.animateView = true; render(); },
    async "browse-exe"() {
      const r = await browse("exe", S.edit.game.launch.exe);
      if (r.path && S.edit) { S.edit.game.launch.exe = r.tokenized || r.path; render(); }
    },
    "set-proc"(el) { S.edit.game.processName = el.dataset.name; render(); },
    async "add-path"(el) {
      const start = (S.edit.game.paths[0] || {}).path || "";
      const r = await browse(el.dataset.kind, start);
      if (r.error) toast(r.error, "error");
      if (!r.path || !S.edit) return;
      addPath(r.tokenized || r.path);
    },
    "add-sugg"(el) { addPath(el.dataset.path); },
    "remove-path"(el) { S.edit.game.paths = S.edit.game.paths.filter((p) => p._k !== el.dataset.k); render(); },
    pcgw() { api.open("pcgw", S.edit.game.name).catch((e) => toast(errText(e), "error")); },
    "save-game": saveGame,
    swatch(el) { S.players[+el.dataset.i].color = el.dataset.color; S.dialog && S.dialog.kind === "welcome" ? renderOverlay() : render(); },
    "add-player"() { const used = new Set(S.players.map((p) => p.color)); S.players.push({ id: "", name: "", color: S.d.colors.find((c) => !used.has(c)) || S.d.colors[0] }); render(); const ins = document.querySelectorAll('[data-field="pname"]'); ins[ins.length - 1].focus(); },
    "remove-player"(el) { S.players.splice(+el.dataset.i, 1); render(); },
    "shelf-all"(el) { S.tag = el.dataset.tag; render(); const t = $(".lib-tools"); if (t) t.scrollIntoView({ block: "start" }); },
    "set-tab"(el) { S.setTab = el.dataset.tab; setPref("setTab", S.setTab); render(); const v = $("#view"); if (v) v.scrollTop = 0; },
    "reset-players"() { S.players = null; render(); },
    "edit-profiles"() { S.dialog = { kind: "profiles" }; renderOverlay(); focusDrawer(); },
    "new-profile"() { S.dialog = { kind: "profiles" }; if (!S.players) S.players = JSON.parse(JSON.stringify(cfg().players)); actions["add-player"](); },
    "save-players": () => savePlayers(false),
    "welcome-done": () => savePlayers(true),
    "open-data": () => api.open("data").catch((e) => toast(errText(e), "error")),
    "steam-setup": (el) => openSteamSetup(el.dataset.game),
    async "steam-recheck"() { S.hooks = null; renderOverlay(); await loadHooks(); if (S.dialog && S.dialog.kind === "steam") renderOverlay(); render(); },
    async "copy-hook"(el) {
      const h = S.hooks && S.hooks.games.find((x) => x.gameId === el.dataset.game);
      if (!h) return;
      try { await api.copy(h.option); S.dialog.copied = h.gameId; renderOverlay(); } catch (e) { toast(errText(e), "error"); }
    },
    "hide-banner"() { S.hideHookBanner = true; render(); toast("Lootnest will finish this the next time Steam is closed."); },
    async "hooks-apply"() {
      S.hooksBusy = true; render(); if (S.dialog) renderOverlay();
      try {
        const onlyLeaving = S.hooks && !S.hooks.games.some((h) => !h.detected && !h.direct);
        const r = await api.applyHooks(true);
        await loadHooks();
        if (r && r.changed && onlyLeaving) toast(r.restarted ? "Steam restarted. It now starts those games by itself." : "Done. Steam now starts those games by itself.");
        else if (r && r.changed) toast(r.restarted ? "Steam restarted. Its Play button now asks who's playing." : "Done. Steam's Play button now asks who's playing.");
      } catch (e) { toast(errText(e), "error"); }
      S.hooksBusy = false; render(); if (S.dialog) renderOverlay();
    },
    rescan() { S.found = { scanning: !(S.found.games || []).length, refreshing: true, games: S.found.games || [] }; render(); api.scan(); },
    async "add-all"() {
      const keys = (S.found.games || []).filter((f) => f.ready).map((f) => f.key);
      await addFound(keys);
    },
    async "found-add"(el) {
      const f = (S.found.games || []).find((x) => x.key === el.dataset.key);
      if (!f) return;
      if (S.dialog && S.dialog.kind === "sheet") closeDialog(false);
      if (f.ready) { await addFound([f.key]); return; }
      openEditor({
        isNew: true,
        game: { id: "", name: f.name, launch: f.source === "steam" ? { type: "steam", appId: f.appId, args: f.args || "" } : { type: "exe", exe: f.exe, args: f.args || "" },
          processName: f.processName || "", playerArgs: {}, paths: (f.paths || []).map((p) => ({ id: "", path: p })) },
        inspect: { suggestions: f.suggestions || [], candidates: [] },
      });
    },
    async "found-hide"(el) {
      if (S.dialog && S.dialog.kind === "sheet") closeDialog(false);
      try { await api.ignoreFound(el.dataset.key); S.found.games = S.found.games.filter((f) => f.key !== el.dataset.key); render(); } catch (e) { toast(errText(e), "error"); }
    },
    async "show-log"() { try { S.logText = (await api.log()) || "Nothing yet."; } catch (e) { S.logText = errText(e); } render(); },
    quit: () => api.quit(),
  };

  async function addFound(keys) {
    S.adding = true; render();
    try {
      const r = await api.addFound(keys);
      if (r.added.length) toast(r.added.length === 1 ? `Profiles are on for ${r.added[0]}` : `Profiles are on for ${r.added.length} games`);
      if (r.failed.length) toast(r.failed.join("\n"), "error");
      await refresh();
      if (S.view === "game" && S.page && S.page.kind === "found" && r.added.length === 1) {
        const g = cfg().games.find((x) => x.name === r.added[0]);
        if (g) S.page = { kind: "game", id: g.id, cloud: null };
      }
      await loadFound();
      await loadHooks();
    } catch (e) { toast(errText(e), "error"); }
    S.adding = false; render();
  }

  function addPath(path) {
    const E = S.edit;
    if (E.game.paths.some((p) => p.path.toLowerCase() === path.toLowerCase())) return;
    const p = { id: "", path, _k: "k" + ++seq };
    E.game.paths.push(p);
    render();
    checkPath(p);
  }

  function confirmLeaveEdit() {
    return true; // edits are explicit; leaving discards them
  }

  document.addEventListener("click", (ev) => {
    const el = ev.target.closest("[data-action]");
    if (!el) {
      if (S.menuFor && !ev.target.closest(".menu")) { S.menuFor = null; render(); }
      return;
    }
    if (el.dataset.action !== "menu" && S.menuFor && !el.closest(".menu")) S.menuFor = null;
    if (el.closest("#ctx")) closeContext();
    if (el.closest(".ln-drawer[data-menu]") && el.dataset.action !== "dialog-cancel" && el.dataset.action !== "scrim") { S.dialog = null; renderOverlay(); }
    const f = actions[el.dataset.action];
    if (f) f(el, ev);
  });

  let findTimer = 0;
  document.addEventListener("input", (ev) => {
    const el = ev.target;
    if (S.dialog && S.dialog.kind === "profile") {
      const f = el.dataset.pe, D = S.dialog;
      if (f) D.p[f] = f === "bannerY" ? +el.value : el.value;
      if (f === "name") { const n = $("#pe-prev-name"); if (n) n.textContent = el.value; }
      else if (f === "tagline") { const n = $("#pe-prev-tag"); if (n) n.textContent = el.value; }
      else if (f === "color") { const d = $(".dialog.pe"); if (d) d.style.setProperty("--c", el.value); document.querySelectorAll('.dialog.pe [style*="--c"]').forEach((a) => a.style.setProperty("--c", el.value)); }
      else if (f === "bannerY") { const a = $(".pe-pbanner .art"); if (a) a.style.backgroundPosition = `center ${el.value}%`; }
      else if (f === "banner" || f === "avatarGame") renderOverlay();
      if (el.id === "pe-q") { D.q = el.value; renderOverlay(); }
      return;
    }
    if (el.id === "gq") {
      S.gq.q = el.value; S.gq.hits = null; S.gq.sel = -1;
      clearTimeout(gqTimer);
      if (el.value.trim() && cfg().settings.findGames) gqTimer = setTimeout(() => { api.findGames(S.gq.q).catch(() => { S.gq.hits = []; renderGq(); }); }, 300);
      renderGq();
      return;
    }
    if (el.id === "find-q") {
      S.find.q = el.value; S.find.hits = null; S.find.error = "";
      clearTimeout(findTimer);
      const r = $("#find-results");
      if (!el.value.trim()) { if (r) r.innerHTML = findResults(); return; }
      if (r) r.innerHTML = findResults();
      findTimer = setTimeout(() => { api.findGames(S.find.q).catch((e) => { S.find.error = errText(e); const r2 = $("#find-results"); if (r2) r2.innerHTML = findResults(); }); }, 350);
      return;
    }
    if (el.id === "lib-q") {
      S.q = el.value;
      const grid = $("#grid");
      if (grid) { grid.innerHTML = gridHTML(); loadCovers(); }
      document.querySelectorAll(".lib-recent, .lib-shelves, .lib-all-head").forEach((x) => { x.hidden = !!S.q; }); // a search shows only what it finds
      return;
    }
    if (el.id === "steam-q" && S.dialog) {
      S.dialog.q = el.value;
      const pos = el.selectionStart;
      renderOverlay();
      const q = $("#steam-q"); q.focus(); q.setSelectionRange(pos, pos);
      return;
    }
    const f = el.dataset.field;
    if (!f) return;
    if (f === "pof") {
      // A run of a person takes their colour; back to "a person" keeps it.
      const p = S.players[+el.dataset.i], par = S.players.find((x) => x.id === el.value);
      p.of = par ? par.id : "";
      if (par) p.color = par.color;
      render();
      return;
    }
    if (f === "pname") { S.players[+el.dataset.i].name = el.value; if (S.dialog && S.dialog.kind === "profiles") { const b = $('[data-action="save-players"]'); if (b) b.disabled = false; } return; }
    const E = S.edit;
    if (!E) return;
    const g = E.game;
    if (f === "name") g.name = el.value;
    else if (f === "exe") g.launch.exe = el.value;
    else if (f === "processName") g.processName = el.value;
    else if (f === "steamDirect") g.steamDirect = el.checked;
    else if (f === "args") g.launch.args = el.value;
    else if (f === "parg") { g.playerArgs = g.playerArgs || {}; g.playerArgs[el.dataset.player] = el.value; }
    else if (f === "path") { const p = g.paths.find((x) => x._k === el.dataset.k); if (p) p.path = el.value; }
  });

  document.addEventListener("change", (ev) => {
    const el = ev.target;
    if (el.dataset.setting) { saveSetting(el); return; }
    if (el.dataset.backup === "auto") { saveBackup({ auto: el.checked }); return; }
    if (el.dataset.backup === "sync") { S.saves = null; saveBackup({ sync: el.checked }).then(loadSaves); return; }
    if (el.dataset.backup === "keep") { saveBackup({ keep: parseInt(el.value, 10) || 0 }); return; }
    if (el.id === "lib-sort") { S.sort = el.value; setPref("sort", S.sort); render(); return; }
    if (el.id === "lib-store") { S.store = el.value; render(); return; }
    if (el.id === "lib-tag") { S.tag = el.value; render(); return; }
    if (el.id === "lib-status") { S.status = el.value; render(); return; }
    if (el.dataset.field === "path" && S.edit) { const p = S.edit.game.paths.find((x) => x._k === el.dataset.k); if (p) checkPath(p); }
  });

  // The sidebar search: Ctrl+K anywhere, arrows and Enter in its list.
  document.addEventListener("keydown", (ev) => {
    // Enter in a save point's name: make it, or save the new name.
    if (ev.key === "Enter" && (ev.target.id === "pt-new" || ev.target.id === "pt-rename")) {
      ev.preventDefault();
      const b = ev.target.id === "pt-new" ? ev.target.closest(".pt-make").querySelector('[data-action="pt-make"]') : ev.target.closest(".pt-row").querySelector('[data-action="pt-rename-save"]');
      if (b) b.click();
      return;
    }
    if (ev.ctrlKey && !ev.altKey && ev.key.toLowerCase() === "k") {
      ev.preventDefault();
      const q = $("#gq"); if (q) { q.focus(); q.select(); renderGq(); }
      return;
    }
    if (ev.target.id !== "gq") return;
    const n = gqItems().length;
    if (ev.key === "ArrowDown" || ev.key === "ArrowUp") {
      ev.preventDefault();
      if (n) { S.gq.sel = ev.key === "ArrowDown" ? (S.gq.sel + 1) % n : (S.gq.sel <= 0 ? n - 1 : S.gq.sel - 1); renderGq(); }
    } else if (ev.key === "Enter") {
      ev.preventDefault();
      if (S.gq.sel >= 0 && S.gq.sel < n) gqOpen(S.gq.sel); else if (cfg().settings.findGames) gqAll(); else if (n) gqOpen(0);
    } else if (ev.key === "Escape") {
      ev.preventDefault(); ev.stopPropagation();
      gqClose(); ev.target.blur();
    }
  }, true);
  document.addEventListener("focusin", (ev) => { if (ev.target.id === "gq") renderGq(); });
  document.addEventListener("pointerdown", (ev) => { if (!ev.target.closest(".side-search")) gqClose(); });
  document.addEventListener("keydown", (ev) => {
    if (ev.target.id === "tag-new" && ev.key === "Enter") { ev.preventDefault(); actions["tag-add"]({ dataset: { key: ev.target.dataset.key } }); return; }
    if (!$("#ctx").hidden) {
      if (ev.key === "Escape") { ev.preventDefault(); closeContext(); return; }
      if (ev.key === "ArrowDown" || ev.key === "ArrowUp") {
        ev.preventDefault();
        const b = [...document.querySelectorAll("#ctx button:not(:disabled)")];
        const i = b.indexOf(document.activeElement);
        b[(i + (ev.key === "ArrowDown" ? 1 : b.length - 1)) % b.length].focus();
        return;
      }
    }
    // Ctrl+F or / finds a game.
    if (!S.picker && !S.dialog && ((ev.ctrlKey && ev.key.toLowerCase() === "f") || (ev.key === "/" && !ev.target.closest("input, textarea, select")))) {
      ev.preventDefault();
      if (S.view !== "library") { S.view = "library"; S.page = null; render(); }
      const q = $("#lib-q"); if (q) { q.focus(); q.select(); }
      return;
    }
    if (S.picker && !S.picker.working) {
      // Naming a new profile: Enter makes it, Esc goes back to the slots.
      if (S.picker.creating && ev.target.id === "np-name") {
        if (ev.key === "Enter") { ev.preventDefault(); actions["np-create"](); }
        else if (ev.key === "Escape") { ev.preventDefault(); actions["np-cancel"](); }
        return;
      }
      if (ev.key === "Escape") { ev.preventDefault(); if (S.picker.ask) actions["picker-back"](); else closePicker(); return; }
      if (!S.picker.ask && /^[1-9]$/.test(ev.key)) {
        const p = grouped()[+ev.key - 1];
        if (p) { ev.preventDefault(); S.picker.playerId = p.id; S.picker.from = S.d.active[S.picker.gameId] || ""; runPick(); }
      }
      return;
    }
    if (ev.key === "Escape") {
      if (S.dialog && S.dialog.kind !== "welcome") { closeDialog(false); return; }
      if (S.menuFor) { S.menuFor = null; render(); return; }
      if (S.view === "game" && !ev.target.closest("input, textarea, select")) backToLibrary();
    }
    if (ev.key === "Enter" && S.dialog && S.dialog.kind === "welcome" && ev.target.dataset.field === "pname") savePlayers(true);
    if (ev.key === "Enter" && ev.target.id === "rs-pw") actions["restore-pw"]();
    if (ev.key === "Enter" && (ev.target.id === "bk-pw2" || ev.target.id === "bk-pw" && S.backup && S.backup.locked)) actions["backup-pw-save"]();
  });

  // block browser shortcuts that make no sense in an app window
  document.addEventListener("keydown", (ev) => {
    if ((ev.ctrlKey && ["p", "f", "g", "r", "u", "s", "+", "-", "=", "0"].includes(ev.key.toLowerCase())) || ev.key === "F5" || ev.key === "F3") ev.preventDefault();
  }, true);
  // No page zoom (Ctrl+wheel, pinch) and no browser menu outside text fields.
  document.addEventListener("wheel", (ev) => { if (ev.ctrlKey) ev.preventDefault(); }, { passive: false });
  document.addEventListener("contextmenu", (ev) => {
    if (!ev.target.closest("input, textarea, .selectable")) ev.preventDefault();
  });
  document.addEventListener("dragover", (ev) => ev.preventDefault());
  document.addEventListener("drop", (ev) => ev.preventDefault());

  // ------------------------------------------------------------------ boot --
  async function boot() {
    try {
      S.d = await api.state();
    } catch (e) {
      setTimeout(boot, 150); // bindings not injected yet
      return;
    }
    if (S.d.loadError) toast(S.d.loadError, "error");
    if (S.d.firstRun) { S.players = JSON.parse(JSON.stringify(cfg().players)); S.dialog = { kind: "welcome" }; }
    render();
    loadHooks().then(() => { if (!S.picker) render(); });
    loadFound();
    setTimeout(() => { if (!S.updates) api.updates().catch(() => {}); }, 8000); // the Updates count in the sidebar
    if (S.d.firstRun) setTimeout(() => { const i = $('[data-field="pname"]'); if (i) i.focus(); }, 50);
    try {
      const r = await api.ready();
      if (r && r.launch) openPicker(r.launch, r.reqId ? "steam" : "play", r.reqId);
    } catch (e) { /* ignore */ }
  }
  boot();
})();
