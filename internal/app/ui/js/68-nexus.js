  // ----------------------------------------------------------------- nexus --
  // Mods from Nexus Mods (app/nexus.go), in a game's Mods tab: search, a
  // mod's page with what it needs and its files. Download opens the file's
  // page on nexusmods.com in a window of Lootnest's own, where the user
  // downloads it as on the site; the window puts it in Lootnest's folder
  // and the mod's row shows how far it has come (nxProgress). Once it's
  // there, Lootnest closes that window and shows what adding it would do
  // (the modplan drawer), as for any mod the user brings.
  const NX_SORTS = [["endorsements", "Most endorsed"], ["downloads", "Most downloaded"], ["updated", "Recently updated"], ["new", "Newest"], ["name", "Name"]];
  const NX_CATS = [["MAIN", "Main files"], ["UPDATE", "Updates"], ["OPTIONAL", "Optional files"], ["MISCELLANEOUS", "Other files"], ["OLD_VERSION", "Older versions"]];
  const NX_PAGE = 24;
  let nxSeq = 0, nxTimer = 0;
  const nxOf = (key) => S.nx[key] || (S.nx[key] = { q: "", sort: "endorsements", mods: null, total: 0, more: false, seq: 0, game: undefined });
  const nxOn = () => !!cfg().settings.nexusMods;
  const nxNum = (n) => (n >= 1e6 ? `${(n / 1e6).toFixed(n >= 1e7 ? 0 : 1)}M` : n >= 1e4 ? `${Math.round(n / 1e3)}K` : n >= 1e3 ? `${(n / 1e3).toFixed(1)}K` : String(n || 0));
  const nxWait = (key, id) => (S.nxWaits || []).find((w) => w.key === key && w.modId === id);
  // What an awaited file is doing: how far its download into Lootnest's
  // folder has come, else what to press. Its parts carry data-nxw, so
  // progress changes them in place (nxProgress).
  const nxProgOf = (w) => (S.nxProg || {})[w.key + "/" + w.modId];
  const nxPct = (P) => (P && P.total ? Math.min(100, Math.round((P.got / P.total) * 100)) : 0);
  const nxWaitSays = (P) => (P ? (P.total ? `Downloading, ${fmtBytes(P.got)} of ${fmtBytes(P.total)}` : `Downloading, ${fmtBytes(P.got)} so far`)
    : "Press Manual download on the Nexus Mods page. Lootnest takes the file from there.");
  const nxWaitLine = (w) => { const P = nxProgOf(w), k = esc(w.key + "/" + w.modId);
    return `<span data-nxw="${k}">${esc(nxWaitSays(P))}</span><span class="nx-meter${P ? "" : " is-idle"}" data-nxm="${k}" aria-hidden="true"><i style="width:${nxPct(P)}%"></i></span>`; };
  function nxProgress() {
    (S.nxWaits || []).forEach((w) => {
      const P = nxProgOf(w), k = w.key + "/" + w.modId;
      document.querySelectorAll("[data-nxw]").forEach((el) => { if (el.dataset.nxw === k) el.textContent = nxWaitSays(P); });
      document.querySelectorAll("[data-nxm]").forEach((el) => { if (el.dataset.nxm === k) { el.classList.toggle("is-idle", !P); el.firstElementChild.style.width = nxPct(P) + "%"; } });
    });
  }
  const nxAdded = (key, id) => Object.values((S.mods[key] || {}).sets || {}).some((rows) => rows.some((m) => m.nexusId === id));

  function nxGame(key) {
    const N = nxOf(key);
    if (N.game !== undefined) return;
    N.game = null;
    if (!S.nxAsked) { S.nxAsked = true; api.nexusWaits().catch(() => {}); } // asks still open from before
    api.nexusGame(key).catch((e) => { N.game = { error: errText(e) }; nxPaint(key); });
  }
  function nxSearch(key, more) {
    const N = nxOf(key);
    N.seq = ++nxSeq;
    N.loading = true; N.error = "";
    if (!more) { N.mods = null; N.more = false; }
    api.nexusSearch(key, N.q, N.sort, more ? (N.mods || []).length : 0, N.seq).catch((e) => { N.loading = false; N.error = errText(e); nxPaint(key); });
  }
  // The results alone are drawn again while typing, so the search box keeps the focus.
  function nxPaint(key) {
    const r = $("#nx-results");
    if (r && r.dataset.key === key) { r.innerHTML = nxResults(key); paintIcons(r); return; }
    if (S.view === "game") render();
  }

  // The Mods tab: your mods, or Nexus Mods.
  function modsPane(it, mods) {
    const key = recentKey(it), pane = S.modsPane[key] || "mine";
    const seg = `<div class="ln-segmented nx-switch" role="radiogroup" aria-label="Mods">${[["mine", "Your mods"], ["nexus", "Nexus Mods"]].map(([v, l]) =>
      `<button type="button" role="radio" aria-checked="${pane === v}" data-action="mods-pane" data-key="${esc(key)}" data-value="${v}">${l}</button>`).join("")}</div>`;
    return `<div class="nx-pane">${seg}${pane === "nexus" ? nexusBlock(it) : mods}</div>`;
  }

  function nexusBlock(it) {
    const key = recentKey(it), M = S.mods[key] || {};
    if (!nxOn()) return `<section class="gpage-block nx-block">${emptyState("Nexus Mods",
      "Search Nexus Mods and add mods to this game without leaving Lootnest. The game's name and what you search for go to Nexus Mods.",
      `<button class="ln-button ln-button--primary" data-action="nexus-on" data-key="${esc(key)}">Turn on Nexus Mods</button>`)}</section>`;
    if (M.refused) return `<section class="gpage-block nx-block"><p class="gpage-note">${GI.alert}<span>${esc(M.refused)}</span></p></section>`;
    const N = nxOf(key);
    nxGame(key);
    const G = N.game;
    if (!G) return `<section class="gpage-block nx-block">${lnInline("", "Looking for this game on Nexus Mods")}</section>`;
    if (G.error) return `<section class="gpage-block nx-block">${lnInline("error", esc(G.error))}<div class="gpage-actions"><button class="ln-button ln-button--secondary ln-button--sm" data-action="nexus-retry" data-key="${esc(key)}">Try again</button></div></section>`;
    const change = `<button class="ln-button ln-button--ghost ln-button--sm" data-action="nexus-pick" data-key="${esc(key)}" data-name="${esc(it.name)}">${G.none ? "Choose the game" : "Not this game?"}</button>`;
    if (G.none) return `<section class="gpage-block nx-block">${emptyState("Not on Nexus Mods", `Nexus Mods has no game named ${esc(it.name)}. If it's there under another name, choose it.`, change)}</section>`;
    if (N.mods === null && !N.loading && !N.error) nxSearch(key);
    const waits = (S.nxWaits || []).filter((w) => w.key === key);
    return `<section class="gpage-block nx-block" data-key="${esc(key)}">
      <div class="nx-tools">
        <label class="nx-search">${ph("magnifying-glass")}<input class="ln-input" id="nx-q" data-key="${esc(key)}" type="search" placeholder="Search ${esc(G.game.name)} mods" value="${esc(N.q)}" autocomplete="off" spellcheck="false" aria-label="Search mods on Nexus Mods"></label>
        <select class="ln-input nx-sort" id="nx-sort" data-key="${esc(key)}" aria-label="Order">${NX_SORTS.map(([v, l]) => `<option value="${v}" ${v === N.sort ? "selected" : ""}>${l}</option>`).join("")}</select>
        <span class="grow"></span><span class="nx-game">${esc(G.game.name)}, ${esc(nxNum(G.game.mods))} mods</span>${change}
      </div>
      ${waits.length ? `<ul class="ln-rows nx-waits">${waits.map((w) => lookRow(`Waiting for ${esc(w.mod)}`, nxWaitLine(w),
        `<button class="ln-button ln-button--secondary ln-button--sm" data-action="nexus-get" data-key="${esc(key)}" data-id="${w.modId}" data-file="${w.fileId}">Open the page again</button><button class="ln-button ln-button--ghost ln-button--sm" data-action="nexus-forget" data-key="${esc(key)}" data-id="${w.modId}">Stop waiting</button>`, "warn")).join("")}</ul>` : ""}
      <div id="nx-results" data-key="${esc(key)}">${nxResults(key)}</div></section>`;
  }

  function nxResults(key) {
    const N = nxOf(key);
    if (N.error && !(N.mods || []).length) return lnInline("error", esc(N.error));
    if (!N.mods) return `<div class="nx-grid" aria-busy="true">${'<div class="nx-card ln-skeleton"></div>'.repeat(8)}</div>`;
    if (!N.mods.length) return `<p class="gpage-hint">${N.q ? `No mods named like "${esc(N.q)}".` : "No mods yet."}</p>`;
    const more = N.mods.length < N.total;
    return `<p class="gpage-hint nx-count">${N.total === 1 ? "1 mod" : `${esc(N.total.toLocaleString())} mods`}</p>
      <div class="nx-grid">${N.mods.map((m) => nxCard(key, m)).join("")}</div>
      ${N.error ? `<p class="error-text">${esc(N.error)}</p>` : ""}
      ${more ? `<div class="gpage-actions nx-more"><button class="ln-button ln-button--secondary ln-button--sm" data-action="nexus-more" data-key="${esc(key)}" ${N.loading ? "disabled" : ""}>${N.loading ? "Loading" : "Show more"}</button></div>` : ""}`;
  }

  function nxCard(key, m) {
    const tag = nxAdded(key, m.id) ? `<span class="ln-tag ln-tag--strong">Added</span>` : nxWait(key, m.id) ? `<span class="ln-tag">Waiting</span>` : "";
    const pic = m.thumb ? `<img src="${esc(m.thumb)}" alt="" loading="lazy" decoding="async">` : `<span class="nx-plate" aria-hidden="true">${esc((m.name || "?").trim().charAt(0).toUpperCase())}</span>`;
    return `<button class="nx-card" data-action="nexus-open" data-key="${esc(key)}" data-id="${m.id}" aria-label="${esc(m.name)}">
      <span class="nx-pic">${pic}${tag}</span>
      <span class="nx-body"><b>${esc(m.name)}</b><span class="nx-by">${esc(m.author || "")}${m.category ? ` <span aria-hidden="true">/</span> ${esc(m.category)}` : ""}</span>
        <span class="nx-sum">${esc(m.summary || "")}</span>
        <span class="nx-stats"><span title="Endorsements">${ph("thumbs-up")}${esc(nxNum(m.endorsements))}</span><span title="Downloads">${ph("download-simple")}${esc(nxNum(m.downloads))}</span>${m.updated ? `<span>${esc(ago(m.updated))}</span>` : ""}</span></span></button>`;
  }

  // A mod's page, in a drawer.
  function openNexusMod(key, id) {
    S.dialog = { kind: "nexusmod", key, id, mod: null, error: "" };
    S.nxMods[key + "/" + id] = S.nxMods[key + "/" + id] || null;
    api.nexusMod(key, id).catch((e) => { if (S.dialog && S.dialog.kind === "nexusmod" && S.dialog.id === id) { S.dialog.error = errText(e); renderOverlay(); } });
    renderOverlay(); focusOverlay();
  }
  function viewNexusMod() {
    const D = S.dialog, P = S.nxMods[D.key + "/" + D.id], it = everyItem().find((x) => recentKey(x) === D.key);
    const btn = (attrs, label, cls = "ln-button--secondary") => `<button class="ln-button ${cls} ln-button--sm" ${attrs}>${label}</button>`;
    const shell = (title, body, foot) => drawer({ label: `${title} on Nexus Mods`, title, sub: it ? `For ${esc(it.name)}` : "", wide: true, body, foot });
    if (D.error || P && P.error) return shell("Nexus Mods", lnInline("error", esc(D.error || P.error)));
    if (!P) return shell("Nexus Mods", lnInline("", "Opening the mod"));
    if (P.none) return shell("Nexus Mods", lnInline("error", "Nexus Mods doesn't show this mod any more."));
    const m = P.mod, wait = nxWait(D.key, m.id), added = nxAdded(D.key, m.id);
    const open = btn(`data-action="nexus-get" data-key="${esc(D.key)}" data-id="${m.id}" data-file="0"`, "Open on Nexus Mods", "ln-button--ghost");
    const hero = `<div class="nx-hero">${m.picture ? `<img src="${esc(m.picture)}" alt="">` : ""}<div><p class="nx-by">${esc(m.author ? `By ${m.author}` : "")}</p>
        <p>${esc(m.summary || "")}</p>
        <dl class="ln-facts nx-facts">${m.version ? `<div><dt>Version</dt><dd>${esc(m.version)}</dd></div>` : ""}
          ${m.updated ? `<div><dt>Updated</dt><dd>${esc(ago(m.updated))}</dd></div>` : ""}
          ${m.category ? `<div><dt>Category</dt><dd>${esc(m.category)}</dd></div>` : ""}
          <div><dt>Endorsements</dt><dd class="ln-num">${esc((m.endorsements || 0).toLocaleString())}</dd></div>
          <div><dt>Downloads</dt><dd class="ln-num">${esc((m.downloads || 0).toLocaleString())}</dd></div></dl></div></div>`;
    const state = wait ? `<ul class="ln-rows">${lookRow(`Waiting for ${esc(wait.file)}`, nxWaitLine(wait),
        btn(`data-action="nexus-forget" data-key="${esc(D.key)}" data-id="${m.id}"`, "Stop waiting", "ln-button--ghost"), "warn")}</ul>`
      : added ? `<ul class="ln-rows">${lookRow("In your mods", "A file of this mod is on the game's list.", "", "on")}</ul>` : "";
    const needs = (m.needs || []).length ? `<div class="gpage-sub">${gpHead("It needs")}<ul class="ln-rows">${m.needs.map((n) => lookRow(esc(n.name), esc(n.notes || (n.external ? "From outside Nexus Mods" : "")),
        n.external ? (n.url ? btn(`data-action="nexus-link" data-url="${esc(n.url)}"`, "Open in the browser") : "")
          : btn(`data-action="nexus-open" data-key="${esc(D.key)}" data-id="${n.modId}"`, "Open"), n.external ? "warn" : "")).join("")}</ul></div>` : "";
    let first = true;
    const files = NX_CATS.map(([cat, label]) => {
      const fs = (m.files || []).filter((f) => f.category === cat);
      if (!fs.length) return "";
      const rows = fs.map((f) => {
        const main = first && cat === "MAIN";
        if (main) first = false;
        const facts = [f.version && `Version ${f.version}`, f.size ? fmtBytes(f.size) : "", f.date ? ago(f.date * 1000) : ""].filter(Boolean).map((x) => `<span>${esc(x)}</span>`).join("");
        return `<li class="nx-file"><div class="pt-main"><b>${esc(f.name)}</b><span class="pt-facts">${facts}</span>${f.description ? `<p class="nx-file-note">${esc(f.description)}</p>` : ""}</div>
          ${btn(`data-action="nexus-get" data-key="${esc(D.key)}" data-id="${m.id}" data-file="${f.id}"`, `${ph("download-simple")}Download`, main ? "ln-button--primary" : "ln-button--secondary")}</li>`;
      }).join("");
      return `<div class="gpage-sub">${gpHead(label)}<ol class="nx-files">${rows}</ol></div>`;
    }).join("");
    const desc = m.description ? `<details class="nx-desc"><summary>Description</summary><div class="selectable">${esc(m.description).replace(/\n/g, "<br>")}</div></details>` : "";
    return shell(esc(m.name), `${hero}${state}${needs}${files || `<p class="gpage-hint">This mod has no files to download.</p>`}${desc}`,
      `${open}<span class="grow"></span><p class="gpage-hint ln-drawer__note">Downloads happen on Nexus Mods' own page.</p>`);
  }

  // Choosing the game on Nexus by hand.
  function viewNexusPick() {
    const D = S.dialog;
    const list = D.error ? lnInline("error", esc(D.error)) : D.busy ? lnInline("", "Searching Nexus Mods") : D.games ? (D.games.length
      ? `<ul class="ln-rows">${D.games.map((g, i) => lookRow(esc(g.name), `${esc(nxNum(g.mods))} mods`, `<button class="ln-button ln-button--secondary ln-button--sm" data-action="nexus-pick-go" data-i="${i}">This one</button>`, "")).join("")}</ul>`
      : `<p class="gpage-hint">No game on Nexus Mods has those words.</p>`) : "";
    return drawer({ label: "Choose the game on Nexus Mods", title: "Choose the game", sub: `For ${esc(D.name)}`,
      body: `<div class="nx-pick"><label class="nx-search">${ph("magnifying-glass")}<input class="ln-input" id="nx-game-q" type="search" value="${esc(D.q)}" autocomplete="off" spellcheck="false" aria-label="The game's name on Nexus Mods"></label>
        <button class="ln-button ln-button--secondary ln-button--sm" data-action="nexus-pick-find">Search</button></div>${list}`,
      foot: D.picked ? `<button class="ln-button ln-button--ghost ln-button--sm" data-action="nexus-pick-clear">Find it by its name again</button>` : "" });
  }

  // Events (app/nexus.go).
  function nexusEvent(ev) {
    switch (ev.type) {
      case "nexusGame": {
        const N = nxOf(ev.key);
        N.game = ev.off ? { error: "Turn on Nexus Mods in Settings, Online." } : ev.error ? { error: ev.error } : ev.none ? { none: true, picked: ev.picked } : { game: ev.game, picked: ev.picked };
        N.mods = null; N.loading = false; N.error = "";
        if (S.dialog && S.dialog.kind === "nexuspick" && S.dialog.key === ev.key) S.dialog.picked = !!ev.picked;
        if (S.view === "game") render();
        break;
      }
      case "nexusGames": {
        const D = S.dialog;
        if (D && D.kind === "nexuspick" && D.q === ev.q) { D.busy = false; D.games = ev.games || []; D.error = ev.error || ""; renderOverlay(); }
        break;
      }
      case "nexusMods": {
        const N = nxOf(ev.key);
        if (ev.seq !== N.seq) break;
        N.loading = false;
        if (ev.none) { N.game = { none: true }; if (S.view === "game") render(); break; }
        if (ev.error) { N.error = ev.error; if (!N.mods) N.mods = []; nxPaint(ev.key); break; }
        N.error = "";
        N.mods = ev.offset ? [...(N.mods || []), ...(ev.mods || [])] : ev.mods || [];
        N.total = ev.total || 0;
        nxPaint(ev.key);
        break;
      }
      case "nexusMod":
        S.nxMods[ev.key + "/" + ev.modId] = ev.error ? { error: ev.error } : ev.none ? { none: true } : { mod: ev.mod, url: ev.url };
        if (S.dialog && S.dialog.kind === "nexusmod" && S.dialog.key === ev.key && S.dialog.id === ev.modId) renderOverlay();
        break;
      case "nexusProgress":
        S.nxProg = ev.progress || {};
        nxProgress();
        break;
      case "nexusWaits":
        S.nxWaits = ev.waits || [];
        if (S.dialog && S.dialog.kind === "nexusmod") renderOverlay();
        if (S.view === "game" && !(S.dialog && S.dialog.kind === "modplan")) render();
        break;
      case "nexusArrived": {
        const A = ev.arrival, it = everyItem().find((x) => recentKey(x) === A.key);
        uiSound("done");
        const M = S.mods[A.key] || {}, owner = player(S.modsFor[A.key]) && setsOn(it || { kind: "" }, M) ? S.modsFor[A.key] : "";
        S.dialog = { kind: "modplan", key: A.key, owner, name: it ? it.name : A.game, path: A.path, busy: true, nexus: true };
        renderOverlay(); focusOverlay();
        api.nexusAdd(A.key, A.path, A.modId, A.fileId).catch((e) => { const D = S.dialog; if (D && D.path === A.path) { D.busy = false; D.refused = errText(e); renderOverlay(); } });
        S.modsPane[A.key] = "mine";
        break;
      }
    }
  }

  // Typing searches after a short pause.
  document.addEventListener("input", (ev) => {
    const el = ev.target;
    if (el.id !== "nx-q") return;
    const key = el.dataset.key, N = nxOf(key);
    N.q = el.value;
    clearTimeout(nxTimer);
    nxTimer = setTimeout(() => nxSearch(key), 350);
  });
  document.addEventListener("change", (ev) => {
    const el = ev.target;
    if (el.id !== "nx-sort") return;
    nxOf(el.dataset.key).sort = el.value;
    nxSearch(el.dataset.key);
  });
  document.addEventListener("keydown", (ev) => {
    if (ev.key === "Enter" && ev.target.id === "nx-game-q") { ev.preventDefault(); nexusPickFind(); }
  });
  function nexusPickFind() {
    const D = S.dialog, q = $("#nx-game-q");
    if (!D || D.kind !== "nexuspick" || !q || !q.value.trim()) return;
    Object.assign(D, { q: q.value.trim(), busy: true, games: null, error: "" });
    renderOverlay();
    api.nexusGames(D.q).catch((e) => { D.busy = false; D.error = errText(e); renderOverlay(); });
  }
  const nexusActions = {
    "mods-pane"(el) { S.modsPane[el.dataset.key] = el.dataset.value; uiSound("tab"); render(); },
    async "nexus-on"() {
      const s = JSON.parse(JSON.stringify(cfg().settings));
      s.nexusMods = true;
      try { await api.saveSettings({ settings: s, autostart: S.d.autostart }); S.nx = {}; await refresh(); } catch (e) { toast(errText(e), "error"); }
    },
    "nexus-retry"(el) { delete S.nx[el.dataset.key]; render(); },
    "nexus-more"(el) { nxSearch(el.dataset.key, true); nxPaint(el.dataset.key); },
    "nexus-open"(el) { openNexusMod(el.dataset.key, +el.dataset.id); },
    async "nexus-get"(el) {
      try { await api.nexusGet(el.dataset.key, +el.dataset.id, +el.dataset.file || 0); } catch (e) { toast(errText(e), "error"); }
    },
    "nexus-forget"(el) { api.nexusForget(el.dataset.key, +el.dataset.id).catch((e) => toast(errText(e), "error")); },
    "nexus-link"(el) { api.open("link", el.dataset.url).catch((e) => toast(errText(e), "error")); },
    "nexus-pick"(el) {
      const G = nxOf(el.dataset.key).game || {};
      S.dialog = { kind: "nexuspick", key: el.dataset.key, name: el.dataset.name, q: el.dataset.name, games: null, picked: !!G.picked };
      renderOverlay(); focusOverlay();
      nexusPickFind();
    },
    "nexus-pick-find": nexusPickFind,
    async "nexus-pick-go"(el) {
      const D = S.dialog, g = D && D.games && D.games[+el.dataset.i];
      if (!g) return;
      delete S.nx[D.key];
      try { await api.nexusPick(D.key, g); closeDialog(); render(); } catch (e) { D.error = errText(e); renderOverlay(); }
    },
    async "nexus-pick-clear"() {
      const D = S.dialog;
      delete S.nx[D.key];
      try { await api.nexusPick(D.key, null); closeDialog(); render(); } catch (e) { D.error = errText(e); renderOverlay(); }
    },
  };
