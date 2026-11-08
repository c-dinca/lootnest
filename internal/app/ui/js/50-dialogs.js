  // ---------------------------------------------------------------- dialogs --
  let confirmResolve = null;
  function confirmDialog(o) {
    return new Promise((resolve) => { confirmResolve = resolve; S.dialog = Object.assign({ kind: "confirm" }, o); renderOverlay(); setTimeout(() => { const b = $(o.safe ? '.dialog [data-action="dialog-cancel"]' : ".dialog .btn.primary, .dialog .btn.danger-primary"); if (b) b.focus(); }, 20); });
  }
  function closeDialog(v) {
    const wasSteam = S.dialog && S.dialog.kind === "steam"; S.dialog = null; renderOverlay(); if (wasSteam) render();
    if (drawerFrom) { focusBack(drawerFrom); drawerFrom = null; } if (confirmResolve) { const r = confirmResolve; confirmResolve = null; r(v); } }

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
    if (!S.dialog) { o.innerHTML = ""; if (BP.on) bpSoon(); return; }
    const D = S.dialog;
    if (D.kind === "confirm") {
      o.innerHTML = `<div class="scrim" data-action="scrim"><div class="dialog narrow" role="alertdialog" aria-modal="true">
        <div><h2>${esc(D.title)}</h2><p class="sub">${esc(D.body)}</p></div>
        <div class="dialog-foot"><button class="btn quiet" data-action="dialog-cancel">${esc(D.cancel || "Cancel")}</button>
        <button class="btn primary" data-action="dialog-ok">${esc(D.ok || "OK")}</button></div></div></div>`;
    } else if (D.kind === "newrun") {
      const g = game(D.gameId), per = player(D.person);
      o.innerHTML = g && per ? `<div class="scrim" data-action="scrim"><div class="dialog narrow" role="dialog" aria-modal="true" aria-label="A new run">
        <div><h2>A new run of ${esc(g.name)}</h2><p class="sub">A playthrough of its own, with its own saves and settings, such as a hardcore try or new game+. Next, choose to start it from nothing or from a copy of ${esc(possessive(per.name))} saves. Those stay as they are.</p></div>
        <input type="text" class="ln-input" id="newrun-name" maxlength="24" placeholder="${esc(per.name)} hardcore" value="${esc(D.name || "")}" aria-label="The run's name">
        <div class="dialog-foot"><span class="error-text grow" id="newrun-error">${esc(D.error || "")}</span><button class="btn quiet" data-action="dialog-cancel" ${D.busy ? "disabled" : ""}>Cancel</button>
        <button class="btn primary" data-action="new-run-go" ${D.busy ? "disabled" : ""}>Start the run</button></div></div></div>` : "";
      setTimeout(() => { const i = $("#newrun-name"); if (i && document.activeElement !== i) i.focus(); }, 20);
    } else if (D.kind === "recap") {
      // The picture is drawn once for each year, not again on every redraw.
      const old = $("#recap-canvas"), key = D.player + ":" + D.year;
      o.innerHTML = viewRecap(D);
      const cv = $("#recap-canvas"), p = player(D.player);
      if (cv && old && D.drawnFor === key) cv.replaceWith(old);
      else if (cv && p) { D.drawnFor = key; D.drawn = drawRecap(cv, p, D.year).catch(() => {}); }
    } else if (D.kind === "movefolder") {
      o.innerHTML = viewMoveFolder(D);
    } else if (D.kind === "removepc") {
      o.innerHTML = viewRemovePC(D);
    } else if (D.kind === "trysave") {
      o.innerHTML = viewTrySave(D);
      setTimeout(() => { const i = $("#try-name"); if (i && !D.src && document.activeElement === document.body) i.focus(); }, 20);
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
    } else if (D.kind === "details-edit") {
      const a = document.activeElement, at = a && a.id.startsWith("de-") ? { id: a.id, pos: a.selectionStart } : null;
      o.innerHTML = viewDetailsEdit();
      if (at) { const q = $("#" + at.id); q.focus(); q.setSelectionRange(at.pos, at.pos); }
    } else if (D.kind === "shots") {
      o.innerHTML = viewShots();
    } else if (D.kind === "manage") {
      o.innerHTML = viewManage();
    } else if (D.kind === "launch") {
      o.innerHTML = viewLaunch();
    } else if (D.kind === "copies") {
      o.innerHTML = viewCopies();
    } else if (D.kind === "unsorted") {
      o.innerHTML = viewUnsorted();
    } else if (D.kind === "playnite") {
      o.innerHTML = viewPlaynite();
    } else if (D.kind === "updhist") {
      o.innerHTML = viewUpdHistory();
    } else if (D.kind === "achdoc") {
      o.innerHTML = viewAchDoctor();
    } else if (D.kind === "sources") {
      o.innerHTML = viewSources();
    } else if (D.kind === "offer") {
      o.innerHTML = viewOffer();
    } else if (D.kind === "update") {
      o.innerHTML = viewUpdateDrawer();
    } else if (D.kind === "profiles") {
      o.innerHTML = viewProfilesDrawer();
    }
    if (!o.innerHTML) S.dialog = null; // what it was for is gone
  }
  // Focus moves into a panel or dialog when it opens, onto its first action.
  // Closing it gives focus back to what opened it.
  let drawerFrom = null;
  function focusOverlay() {
    drawerFrom = document.activeElement;
    setTimeout(() => {
      const d = $(".ln-drawer"), g = $("#overlay .dialog, #overlay .welcome"), f = g && g.querySelector("input, select, button:not(:disabled)");
      if (d) d.focus(); else if (f && !g.contains(document.activeElement)) f.focus();
    }, 30);
  }
  // What opened it, or the same button drawn again since.
  function focusBack(el) {
    const same = (e) => e.dataset.action && JSON.stringify({ ...e.dataset }) === JSON.stringify({ ...el.dataset });
    const b = document.contains(el) ? el : [...document.querySelectorAll("[data-action]")].find(same);
    if (b) b.focus();
  }

  // ------------------------------------------------------------ game mover --
  const MOVE_MARGIN = 512 * 1024 * 1024; // room kept free beyond the game's size

  function openMove(appId, name) {
    S.dialog = { kind: "move", appId, name, info: null, pick: "" };
    renderOverlay();
    if (!S.moving) api.moveInfo(appId).catch((e) => { if (S.dialog && S.dialog.kind === "move") { S.dialog.info = { error: errText(e), libraries: [] }; renderOverlay(); } });
  }

  // The frame and the progress of both Move dialogs: Game Mover's and a
  // folder game's (viewMoveFolder).
  const moveWrap = (name, inner, sub) => `<div class="scrim" data-action="scrim"><div class="dialog narrow" role="dialog" aria-modal="true" aria-label="Move ${esc(name)}">
    <div><h2>Move ${esc(name)} to another drive</h2>${sub ? `<p class="sub">${sub}</p>` : ""}</div>${inner}</div></div>`;
  const moveProgress = (done, total, text, until) => `<div class="mover"><div class="meter"><i style="width:${total ? Math.min(100, Math.round((done / total) * 100)) : 0}%"></i></div><p class="hint" role="status">${esc(text)}</p></div>
        <p class="hint">You can close this window; Lootnest keeps going and tells you when it's done. ${until}</p>`;
  function viewMove() {
    const D = S.dialog, M = S.moving, I = D.info;
    const wrap = (inner, sub) => moveWrap(D.name, inner, sub);
    if (M && M.appId === D.appId) {
      const text = M.step === "steam" ? "Closing Steam" : M.total ? `Copying ${fmtSize(M.done)} of ${fmtSize(M.total)}` : "Starting";
      return wrap(`${moveProgress(M.done, M.total, text, "Keep Steam closed until then.")}
        <div class="dialog-foot"><button class="btn quiet" data-action="dialog-cancel">Close</button></div>`);
    }
    if (M || S.folderMove) {
      return wrap(`<p class="note">Lootnest is moving ${esc((M && M.name) || "another game")}. Wait for it to finish, then move this one.</p>
        <div class="dialog-foot"><button class="btn quiet" data-action="dialog-cancel">Close</button></div>`);
    }
    if (!I) return wrap(`<div class="dialog-foot"><button class="btn quiet" data-action="dialog-cancel">Cancel</button></div>`, "Working out the game's size");
    if (I.error) return wrap(`<p class="note">${esc(I.error)}</p><div class="dialog-foot"><button class="btn quiet" data-action="dialog-cancel">Close</button></div>`);
    const cur = I.libraries.find((l) => l.current);
    const others = I.libraries.filter((l) => !l.current);
    const room = (l) => l.free < 0 || l.free >= I.size + MOVE_MARGIN;
    const list = others.length
      ? `<div class="choices">${others.map((l) => `<button class="choice${D.pick === l.path ? " on" : ""}" data-action="move-pick" data-lib="${esc(l.path)}" ${room(l) ? "" : "disabled"} aria-pressed="${D.pick === l.path}">
          <span class="dot"></span><span><b>${esc(l.path)}</b><span>${l.free < 0 ? "Free space unknown" : fmtSize(l.free) + " free"}${room(l) ? "" : " · not enough room"}</span></span></button>`).join("")}</div>`
      : `<p class="note">Steam has one library on this PC. To move games to another drive, add a library there in Steam: <b>Settings > Storage</b>, then <b>Add drive</b>. Then come back here.</p>`;
    return wrap(`${list}
      ${others.length ? `<p class="hint">The game is copied and checked before anything is removed from where it is now.${I.steamRunning ? " Steam closes while the game moves and opens again when it's done." : ""}</p>` : ""}
      <div class="dialog-foot"><button class="btn quiet" data-action="dialog-cancel">Cancel</button>
        ${others.length ? `<button class="btn primary" data-action="move-go" ${D.pick ? "" : "disabled"}>Move</button>` : ""}</div>`, `${fmtSize(I.size)} · now in ${esc(cur ? cur.path : "")}`);
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
    return `<div class="scrim" data-action="scrim"><div class="dialog narrow" role="dialog" aria-modal="true" aria-label="${esc(f.name)} settings">
      <div><h2>${esc(f.name)} settings</h2><p class="sub">These settings don't depend on profiles and stay if you turn them on.</p></div>
      <div class="field"><label for="gs-name">Name in the Library</label>
        <input id="gs-name" type="text" value="${esc(f.name)}" maxlength="60" placeholder="${esc(f.scanName || f.name)}" spellcheck="false">
        ${f.scanName ? `<p class="hint">Found as ${esc(f.scanName)}. Leave it empty to use that name again.</p>` : ""}</div>
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
    const inLibrary = new Set((S.found.games || []).filter((f) => f.appId).map((f) => f.appId)
      .concat(cfg().games.filter((g) => g.launch.type === "steam").map((g) => g.launch.appId)));
    let list;
    if (!S.d.steam.found) list = `<div class="note">Steam wasn't found on this PC. You can still add any game by choosing its .exe.</div>`;
    else if (!D.steam) list = `<p class="hint">Looking for installed Steam games</p>`;
    else {
      const m = nameMatcher(D.q), items = D.steam.filter((g) => m(g.name));
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

