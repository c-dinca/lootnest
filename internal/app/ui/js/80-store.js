  // --------------------------------------------------------- sidebar search --
  // One search box on every page: commands, your games, then the store's.
  let gqTimer = 0;
  // The commands: each runs an action the window already has, with the
  // data its button would carry. key is its shortcut from SHORTCUTS; to is
  // the page it goes to first, where its result shows.
  const GQ_TOP = 6; // shown while the box is empty
  function gqCommands() {
    const keys = SHORTCUTS(cfg().settings), key = (what) => ((keys.find(([, w]) => w === what) || [[""]])[0][0]);
    const cmd = (name, action, data = {}, k = "", to = "") => ({ name, action, data, key: k, to });
    const out = [];
    const open = S.view === "game" && S.page && everyItem().find((x) => x.kind === S.page.kind && x.id === S.page.id);
    const run = (S.d.sessions || []).find((x) => x.phase === "playing" && game(x.gameId)), it = run ? itemByKey("game:" + run.gameId) : open;
    const P = it && it.kind !== "owned" && pointsFor(it); // asked once, the box redrawn when it comes
    if (P && P.loaded && !P.error) out.push(cmd(`Make a save point in ${it.name}`, "pt-make", it.kind === "game" ? { game: it.id } : { key: it.id }, run ? key("Make a save point while a game runs") : ""));
    const M = open && S.mods[recentKey(open)];
    if (M && modsTabOn(M) && !M.refused) {
      const k = recentKey(open), o = setsOn(open, M) && player(S.modsFor[k]) ? S.modsFor[k] : "";
      out.push(cmd(`Add a mod to ${open.name}`, "mod-add", { mods: k, owner: o, name: open.name }));
    }
    out.push(cmd("Scan again", "rescan", {}, key("Look for games again")), cmd("Add a game", "add-game"));
    if (S.view !== "settings") out.push(cmd("Open Settings", "nav", { view: "settings" }));
    out.push(cmd("Big Picture", "bp-open", {}, key("Open or leave Big Picture")));
    ["home", "library", "find", "updates", "achievements", "saves", "players"].filter((v) => v !== S.view).forEach((v) => out.push(cmd(`Go to ${PAGE_NAMES[v]}`, "nav", { view: v })));
    out.push(cmd("Go to Cloud saves", "nav", { view: "cloud" }), cmd("Check for game updates", "upd-refresh", {}, "", "updates"), cmd("Check for Lootnest updates", "update-check"));
    Object.entries(SET_CATS).filter(([tab]) => tab !== "cloud").forEach(([tab, l]) => out.push(cmd(`Settings: ${l}`, "set-open", { tab })));
    grouped().filter((p) => p.id !== S.seat).forEach((p) => out.push(cmd(`${solo() ? "Play as" : "Sit down as"} ${p.name}`, "seat-set", { player: p.id })));
    out.push(S.d.safe ? cmd("Restart normally", "restart-normal") : cmd("Restart in safe mode", "restart-safe"), cmd("Report a problem", "report-problem"));
    return out;
  }
  function gqItems() {
    const q = S.gq.q.trim();
    if (!q) return gqCommands().slice(0, GQ_TOP).map((c) => ({ cmd: c }));
    const m = nameMatcher(q), cmds = gqCommands().filter((c) => m(c.name)).slice(0, 5).map((c) => ({ cmd: c }));
    const mine = libraryItems().filter((it) => m(it.name)).slice(0, 5).map((it) => ({ mine: it }));
    const ownedIds = new Set(mine.map((m) => m.mine.appId).filter(Boolean));
    const store = (S.gq.hits || []).filter((h) => !ownedIds.has(h.appId) && !h.owned).slice(0, 6).map((h) => ({ hit: h }));
    return [...cmds, ...mine, ...store];
  }
  function renderGq() {
    const pop = $("#gq-pop"), input = $("#gq");
    if (!pop || !input) return;
    const q = S.gq.q.trim();
    if (document.activeElement !== input && !pop.contains(document.activeElement)) { pop.hidden = true; input.setAttribute("aria-expanded", "false"); return; }
    const items = gqItems(), on = cfg().settings.findGames;
    let i = -1;
    const opt = (inner, extra = "") => { i++; return `<button class="gq-item${i === S.gq.sel ? " is-sel" : ""}" role="option" aria-selected="${i === S.gq.sel}" data-action="gq-open" data-i="${i}" ${extra}>${inner}</button>`; };
    const cmds = items.filter((x) => x.cmd), mine = items.filter((x) => x.mine), store = items.filter((x) => x.hit);
    const cmdHTML = cmds.length ? `<div class="gq-group"><span class="gq-label">Commands</span>${cmds.map((x) => opt(`<span class="gq-name">${esc(x.cmd.name)}</span>${x.cmd.key ? `<kbd class="gq-key">${esc(x.cmd.key)}</kbd>` : ""}`)).join("")}</div>` : "";
    if (!q) { pop.innerHTML = cmdHTML; pop.hidden = false; input.setAttribute("aria-expanded", "true"); return; }
    const mineHTML = mine.length ? `<div class="gq-group"><span class="gq-label">Your games</span>${mine.map((m) => opt(`<span class="gq-name">${esc(m.mine.name)}</span><span class="gq-meta">${esc(m.mine.kind === "game" ? "Profiles on" : storeNames(m.mine))}</span>`)).join("")}</div>` : "";
    const storeHTML = !on ? `<div class="gq-group"><span class="gq-label">Store</span><p class="gq-note">Turn on the Store to search every game, with prices and whether it runs here.</p></div>`
      : S.gq.hits === null ? `<div class="gq-group"><span class="gq-label">Store</span><p class="gq-note">Looking it up</p></div>`
      : store.length ? `<div class="gq-group"><span class="gq-label">Store</span>${store.map((x) => opt(`${x.hit.image ? `<img src="${esc(x.hit.image)}" alt="">` : ""}<span class="gq-name">${esc(x.hit.name)}</span><span class="gq-meta">${x.hit.price ? esc(cents(x.hit.price)) : ""}</span>`)).join("")}</div>` : "";
    const none = !cmds.length && !mine.length && on && S.gq.hits && !store.length ? `<p class="gq-note">Nothing called "${esc(q)}".</p>` : "";
    pop.innerHTML = `${cmdHTML}${mineHTML}${storeHTML}${none}${on ? `<button class="gq-all" data-action="gq-all">See everything for "${esc(q)}" in the Store</button>` : ""}`;
    pop.hidden = false; input.setAttribute("aria-expanded", "true");
  }
  function gqClose() { S.gq.sel = -1; const pop = $("#gq-pop"); if (pop) pop.hidden = true; const i = $("#gq"); if (i) i.setAttribute("aria-expanded", "false"); }
  function gqOpen(i) {
    const x = gqItems()[i];
    if (!x) return;
    gqClose(); $("#gq").blur();
    if (x.cmd) {
      $("#gq").value = S.gq.q = "";
      const el = document.createElement("button"); // stands in for the command's button
      Object.assign(el.dataset, x.cmd.data);
      if (x.cmd.to && S.view !== x.cmd.to) actions.nav({ dataset: { view: x.cmd.to } });
      actions[x.cmd.action](el);
    } else if (x.mine) openSheet(x.mine.kind, x.mine.id, null);
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
  const offTag = (pct) => (pct > 0 ? `<span class="ln-tag ln-tag--strong ln-num">-${pct}%</span>` : "");
  function viewFind() {
    const on = cfg().settings.findGames;
    const search = on ? `<div class="lib-actions"><label class="ln-search">${GI.search}<input type="search" id="find-q" class="ln-input" placeholder="Search the store" value="${esc(S.find.q)}" autocomplete="off" spellcheck="false" aria-label="Search the store"></label></div>` : "";
    const head = `<header class="lib-head"><h1>Store</h1>${search}<p>Any game, with its price in every official store and whether it runs on this PC.</p></header>`;
    if (!on) return `<div class="find-page ln-root">${head}<section class="find-off"><h2>Look up any game</h2>
      <p>The Store asks the Steam store for games, deals and requirements, and CheapShark for prices in other official stores, such as GOG, Humble and Fanatical. Only what you type, or a game's Steam id, is sent. Lootnest opens the store you choose in your browser.</p>
      <button class="ln-button ln-button--primary" data-action="find-on">Turn on the Store</button></section></div>`;
    return `<div class="find-page ln-root">${head}<div id="find-results">${findResults()}</div></div>`;
  }
  // A Store picture that's missing, or didn't load: the name set in type.
  const noImg = (name) => plate("", "find-noimg", `<b>${esc(name)}</b>`);
  document.addEventListener("error", (ev) => {
    const img = ev.target;
    if (img.tagName !== "IMG" || !img.closest(".find-card, .find-daily, .find-tops")) return;
    if (img.closest(".find-card")) img.outerHTML = noImg(img.alt);
    else img.remove();
  }, true);
  const storePrice = (it) => (it.store ? `<span class="ln-num">${esc(money(it.usd, "USD"))}</span>${offTag(it.price && it.price.discount)}<span class="find-store-name">${esc(it.store)}</span>`
    : it.price ? `<span class="ln-num">${esc(cents(it.price))}</span>${offTag(it.price.discount)}` : `<span>Free</span>`);
  // A game in a Store row, on its wide art.
  function storeCard(it) {
    return `<li><button class="find-card" data-action="find-open" data-app="${it.appId}" data-name="${esc(it.name)}">
      ${it.image ? `<img src="${esc(it.image)}" alt="${esc(it.name)}" loading="lazy">` : noImg(it.name)}
      <span class="find-card-body"><b>${esc(it.name)}</b><span class="find-card-meta">${storePrice(it)}${it.score ? `<span class="find-score" title="Metacritic">${it.score}</span>` : ""}${it.owned ? `<span class="ln-tag">On this PC</span>` : ""}</span></span></button></li>`;
  }
  function storeFront() {
    const F = S.front;
    if (!F) { api.storeFront().catch((e) => { S.front = { error: errText(e) }; const r = $("#find-results"); if (r) r.innerHTML = findResults(); }); S.front = { loading: true }; return lnInline("", "Opening the store"); }
    if (F.loading) return lnInline("", "Opening the store");
    if (F.error) return `${lnInline("error", esc(F.error))}<div class="gpage-actions"><button class="ln-button ln-button--secondary ln-button--sm" data-action="store-retry">Try again</button></div>`;
    const row = (title, sub, list) => (list && list.length ? `<section class="find-row"><div class="ln-section-head"><div><h2>${title}</h2>${sub ? `<p>${sub}</p>` : ""}</div></div><ul class="find-grid find-grid--wide">${list.map(storeCard).join("")}</ul></section>` : "");
    const d = F.daily;
    const daily = d ? `<button class="find-daily" data-action="find-open" data-app="${d.appId}" data-name="${esc(d.name)}">${d.image ? `<img src="${esc(d.image)}" alt="">` : ""}
      <span class="find-daily-body"><span class="find-daily-label">Today's deal on Steam</span><b>${esc(d.name)}</b><span class="find-card-meta">${d.price ? storePrice(d) : ""}${d.owned ? `<span class="ln-tag">On this PC</span>` : ""}</span></span></button>` : "";
    const top = (F.top || []).slice(0, 10);
    const tops = top.length ? `<section class="find-tops"><div class="ln-section-head"><h2>Top sellers on Steam</h2></div><ol class="find-top-list">${top.map((it, i) => `<li><button data-action="find-open" data-app="${it.appId}" data-name="${esc(it.name)}">
      <span class="ln-num find-rank">${i + 1}</span><span class="find-cap">${it.image ? `<img src="${esc(it.image)}" alt="" loading="lazy">` : ""}</span><b>${esc(it.name)}</b><span class="find-card-meta">${storePrice(it)}</span></button></li>`).join("")}</ol></section>` : "";
    const head = daily && tops ? `<div class="find-top">${daily}${tops}</div>` : daily + tops;
    return `${head}${row("Deals on Steam", "", F.specials)}${row("Best deals in other stores", "Well-reviewed games on sale in official stores, from CheapShark, in US dollars.", F.deals)}`;
  }
  function findResults() {
    const F = S.find;
    if (!F.q.trim()) return storeFront();
    if (F.error) return lnInline("error", esc(F.error));
    if (!F.hits) return lnInline("", "Looking it up");
    if (!F.hits.length) return `<p class="find-hint">The Steam store has nothing called "${esc(F.q)}".</p>`;
    return `<ul class="find-grid">${F.hits.map((h) => `<li><button class="find-card" data-action="find-open" data-app="${h.appId}" data-name="${esc(h.name)}">
      ${h.image ? `<img src="${esc(h.image)}" alt="${esc(h.name)}" loading="lazy">` : noImg(h.name)}
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
    const head = `<header class="lib-head"><h1>Achievements</h1><p>Every game's achievements on this PC. Each profile has its own where they go with the profile; Steam's belong to the Steam account.</p></header>`;
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
      return it ? { g, it, unlocked: own ? own.unlocked : g.unlocked, recent: own ? own.recent : g.recent, rarest: (own ? own.rarest : g.rarest) || [], shared: !g.players } : null;
    }).filter(Boolean);
    const chips = ps.length > 4 ? `<select id="ach-pick" class="ln-input ach-pick" aria-label="Whose achievements">${ps.map((p) => `<option value="${esc(p.id)}" ${p.id === who ? "selected" : ""}>${esc(possessive(p.name))} achievements</option>`).join("")}</select>`
      : ps.length > 1 ? `<div class="ln-segmented ach-pick" role="group" aria-label="Whose achievements">${ps.map((p) => `<button data-action="ach-page" data-player="${esc(p.id)}" aria-pressed="${p.id === who}">${lnAvatar(p, "20")}${esc(p.name)}</button>`).join("")}</div>` : "";
    if (!rows.length) {
      return `<div class="ach-page ln-root">${head}${chips}${emptyState(GI.trophy, "No achievements found yet", "Lootnest reads them from Steam's library on this PC, and from the file a Steam-compatible tool keeps next to a game's saves. Play a game that has achievements and they show up here.")}</div>`;
    }
    const total = rows.reduce((n, r) => n + r.unlocked, 0);
    const ms = who && player(who) ? milestones(player(who), rows) : [];
    const msDone = ms.filter((x) => x.done).length;
    const msHTML = ms.length ? `<details class="ach-ms"${S.achMs ? " open" : ""}><summary><svg viewBox="0 0 12 12" aria-hidden="true"><path d="M3 4.5 6 7.5l3-3"/></svg>Lootnest milestones <span class="ln-count ln-num">${msDone} of ${ms.length}</span></summary>
      <p>For any game, from the time ${esc(pname(who))} plays through Lootnest.</p>
      <ul class="ach-ms-grid">${ms.map((x) => `<li class="${x.done ? "done" : ""}"><span class="ach-ms-icon">${GI.trophy}</span><div><b>${esc(x.name)}</b><span>${x.done ? esc(x.desc) : esc(x.desc) + ". " + esc(fmtGoal(x))}</span>${x.done ? "" : `<span class="ach-bar" aria-hidden="true"><span style="width:${Math.round((x.cur / x.goal) * 100)}%"></span></span>`}</div></li>`).join("")}</ul></details>` : "";
    const latest = rows.flatMap((r) => (r.recent || []).map((a) => ({ a, r }))).sort((x, y) => new Date(y.a.at || 0) - new Date(x.a.at || 0)).slice(0, 10);
    const games = rows.slice().sort((x, y) => y.unlocked / y.g.total - x.unlocked / x.g.total || y.unlocked - x.unlocked || x.g.name.localeCompare(y.g.name));
    const p = player(who);
    const all = rows.reduce((n, r) => n + r.g.total, 0), allPct = all ? Math.round((total / all) * 100) : 0;
    // The rarest they've unlocked anywhere, on show.
    const rarest = rows.flatMap((r) => r.rarest.map((a) => ({ a, r }))).sort((x, y) => x.a.percent - y.a.percent).slice(0, 4);
    const show = rarest.length ? `<section aria-label="Rarest unlocked"><div class="ln-section-head"><div><h2>Rarest unlocked</h2></div></div>
      <ul class="ach-list">${rarest.map(({ a, r }) => achRow(a, r.g.key, esc(r.g.name))).join("")}</ul></section>` : "";
    return `<div class="ach-page ln-root">${head}
      <div class="ach-cols">
        <div class="ach-main">${chips}
          <div class="ach-headline">${p ? lnAvatar(p, "48") : ""}<p><span class="ach-hl-num"><b class="ln-num" data-countup="${total}">${total}</b> of <span class="ln-num">${all}</span></span><span class="ach-hl-pct ln-num">${allPct}%</span></p>
            <span class="ach-bar" aria-hidden="true"><span style="width:${allPct}%"></span></span></div>
          <section><div class="ln-section-head"><div><h2>Games</h2></div><span class="ach-count ln-num">${rows.length}</span></div>
          <ul class="ach-games">${games.map((r) => { ensureCover(r.it); const pct = Math.round((r.unlocked / r.g.total) * 100); return `<li><button class="ach-gcard" data-action="open-sheet" ${itemAttr(r.it)}>
            <span class="ach-gcover cover">${coverHTML(r.it)}</span>
            <span class="ach-gbody"><span class="ach-game-name"><b>${esc(r.g.name)}</b></span>
              ${r.shared ? `<span class="ln-tag">${r.g.source === "steam" ? "Steam account" : "Shared"}</span>` : `<span class="ach-gnote">${esc(possessive(pname(who)))} own</span>`}</span>
            <span class="ach-bar" aria-hidden="true"><span style="width:${pct}%"></span></span>
            <span class="ach-game-num"><b class="ln-num">${r.unlocked}</b> of <span class="ln-num">${r.g.total}</span></span></button></li>`; }).join("")}</ul></section>
          ${msHTML}
        </div>
        <div class="ach-side">${show}
          <section class="ach-latest"><div class="ln-section-head"><div><h2>Latest unlocked</h2></div></div>
          ${latest.length ? `<ul class="ach-list">${latest.map(({ a, r }) => achRow(a, r.g.key, `${esc(r.g.name)}${a.at && new Date(a.at).getFullYear() > 2000 ? ", " + esc(ago(a.at)) : ""}`)).join("")}</ul>` : `<p class="home-hint">Nothing unlocked yet.</p>`}</section>
        </div>
      </div></div>`;
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
  // Cloud saves, a section of Saves: where the saves go, every game's state
  // for each player, and the backups of everything.
  function cloudSection() {
    const B = S.backup;
    const wrap = (body, sum) => `<section class="cloud-page sv-sec" id="sv-cloud" aria-labelledby="sv-cloud-h"><div class="ln-section-head"><div><h2 id="sv-cloud-h">Cloud saves</h2>
      <p>Every profile's saves go up after each session. On another PC, the newest come down before the game starts.</p></div></div>${sum || ""}${body}</section>`;
    if (!B) { loadBackupInfo(); return wrap(lnInline("", "Reading your cloud folder")); }
    const b = (kind, attrs, label) => `<button class="ln-button ln-button--${kind}" ${attrs}>${label}</button>`;
    if (!B.dir) {
      const rows = PROVIDERS.map((pv) => {
        const found = (B.folders || []).filter((f) => pv.match.test(f.name));
        return found.length
          ? found.map((f) => `<div class="cs-option"><div><b>${esc(pv.name)}</b><span class="selectable">${esc(f.path)}</span></div>${b("primary", `data-action="backup-use" data-dir="${esc(f.path)}"`, "Use " + esc(f.name))}</div>`).join("")
          : `<div class="cs-option"><div><b>${esc(pv.name)}</b><span>Install ${esc(pv.name)} and sign in with ${esc(pv.account)}, then come back here.</span></div>${b("secondary", `data-action="open-download" data-url="${esc(pv.url)}"`, "Get " + esc(pv.name))}</div>`;
      }).join("");
      return wrap(`
        <div class="cloud-setup"><div class="ln-section-head"><div><h3>Where should your saves go?</h3><p>Lootnest writes the files to that folder, and the cloud app uploads them with your account.</p></div></div>
          <div class="cs-options">${rows}
            <div class="cs-option"><div><b>USB drive or another folder</b><span>A drive you carry between PCs, a NAS, or any folder.</span></div>${b("secondary", 'data-action="backup-pick"', "Choose a folder")}</div></div>
          <p class="cs-note">Signing in to Microsoft or Google right in Lootnest, without their apps, comes in a later version.</p>
          <div class="bk-restore">${b("ghost", 'data-action="restore-pick"', "Restore from a backup")}<span>On a new PC, or to go back.</span></div></div>`);
    }
    const st = B.status, V = S.saves;
    const last = st.running ? "Backing up now" : st.last ? `Last backup ${esc(ago(st.last))}, ${esc(fmtBytes(st.size))}` : "No backup yet";
    const failed = st.error && (!st.last || new Date(st.errorAt) > new Date(st.last));
    const label = { same: "Up to date", up: "Changed here, not uploaded yet", none: "Not in the cloud yet", down: "Newer on another PC", conflict: "Changed on two PCs" };
    let games = "", conflicts = "", counts = "";
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
      const byGame = {}, n = {};
      V.games.forEach((x) => { (byGame[x.gameId] = byGame[x.gameId] || []).push(x); n[x.state] = (n[x.state] || 0) + 1; });
      const short = { same: "up to date", up: "to upload", none: "not in the cloud yet", down: "newer on another PC", conflict: "changed on two PCs" };
      counts = Object.keys(short).filter((k) => n[k]).map((k) => `${n[k]} ${short[k]}`).join(", ");
      conflicts = V.games.filter((x) => x.state === "conflict" && game(x.gameId) && player(x.playerId)).map((x) => {
        const pl = player(x.playerId), at = `data-game="${esc(x.gameId)}" data-player="${esc(pl.id)}"`;
        const when = [x.here && `this PC ${ago(x.here)}`, x.computer && `${x.computer} ${ago(x.changed)}`].filter(Boolean).join(", ");
        return banner(`${esc(game(x.gameId).name)} <span class="ln-tag ln-tag--player ln-player" style="--player:${esc(pl.color)}">${esc(pl.name)}</span>`,
          `Changed on two PCs${when ? ": " + esc(when) : ""}.`,
          b("secondary ln-button--sm", `data-action="sync-resolve" ${at} data-answer="here"`, "Keep this PC's") + b("secondary ln-button--sm", `data-action="sync-resolve" ${at} data-answer="cloud"`, `Use ${x.computer ? esc(possessive(x.computer)) : "the cloud's"}`), false, "warn");
      }).join("");
      games = `<div class="cs-games">${Object.entries(byGame).map(([gid, rows]) => {
        const g = game(gid);
        if (!g) return "";
        const it = everyItem().find((x) => x.kind === "game" && x.id === gid);
        if (it) ensureCover(it);
        return `<div class="cs-game"><div class="cs-cover">${it ? `<div class="cover">${coverHTML(it)}</div>` : ""}</div><div class="cs-rows"><b class="cs-name">${esc(g.name)}</b>
          ${rows.map((x) => {
            const pl = player(x.playerId);
            if (!pl) return "";
            const where = x.computer ? `${x.state === "same" ? "Last from" : "From"} ${esc(x.computer)}, ${esc(ago(x.changed))}` : "";
            return `<div class="cs-row state-${esc(x.state)}">${lnAvatar(pl, "20")}<span class="cs-who">${esc(pl.name)}</span>
              <span class="cs-state">${esc(label[x.state] || "")}</span><span class="cs-where">${where}</span></div>`;
          }).join("")}</div></div>`;
      }).join("")}</div>
      ${V.players && V.players.length ? `<p class="cs-note">The cloud also has saves for ${esc(V.players.join(", "))}. Add a profile with the same name on <b>Profiles</b> (or import it) to use them here.</p>` : ""}`;
    }
    const sum = B.sync && !B.locked ? `<div class="cloud-sum">${counts ? `<span>${esc(counts[0].toUpperCase() + counts.slice(1))}</span>` : ""}${b("primary ln-button--sm", 'data-action="sync-now"', "Sync now")}</div>` : "";
    // An explanation shows its first sentence; the rest folds under More.
    const brief = (t) => { const i = t.indexOf(". "); return i < 0 || t.length <= 72 ? `<p class="tg-line">${t}</p>` : `<div class="tg-brief"><p class="tg-line">${t.slice(0, i + 1)}</p><details class="tg-more"><summary>More</summary><p>${t.slice(i + 2)}</p></details></div>`; };
    const path = B.target || "";
    return wrap(`
      <div class="cloud-cols">
        <div class="cloud-main">${conflicts ? `<section class="cs-conflicts">${conflicts}</section>` : ""}<section><div class="ln-section-head"><div><h3>Games</h3></div></div>${games}</section>
          <section class="cloud-backups"><div class="ln-section-head"><div><h3>Backups of everything</h3></div></div>
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
          <section><div class="ln-section-head"><div><h3>Your cloud folder</h3></div></div>
            <div class="bk-card">
              <div class="bk-row"><div class="bk-where"><span class="label">${esc(providerOf(B))}</span><span class="bk-path"><b class="selectable" title="${esc(path)}">${esc(midTrunc(path, 44))}</b><button class="ln-button ln-button--ghost ln-button--icon ln-button--sm" data-action="copy-path" data-path="${esc(path)}" aria-label="Copy the path" title="Copy the path">${GI.copy}</button></span></div>
                <div class="gpage-actions">${b("ghost ln-button--sm", 'data-action="backup-pick"', "Change")}${b("ghost ln-button--sm", 'data-action="backup-open"', "Open folder")}</div></div>
              <label class="toggle"><span><b>Sync saves between PCs</b>${brief("Each profile's saves go up after a session. The newest come down before a game starts. If they changed on two PCs, Lootnest asks which to keep.")}</span>
                <span class="switch"><input type="checkbox" data-backup="sync" ${B.sync ? "checked" : ""}><span></span></span></label>
              ${sw("gameSaves", "Keep versions of games outside Steam", brief("A new copy of their saves after they change and the game closes. For games without profiles and not on Steam, which has its own cloud. Restore any version from the game's page."), cfg().settings.gameSaves)}
              ${sw("saveDb", "Find save locations online", brief("A weekly list of where over 20,000 games keep their saves. It comes from PCGamingWiki, through the Ludusavi project on GitHub, so new games get the right locations for profiles and cloud saves. Nothing about you or your games is sent."), cfg().settings.saveDb)}
              <div class="bk-row"><div><b>${B.locked ? "Has a password" : B.encrypted ? "Encrypted with your password" : "Not encrypted"}</b>
                <p>${B.encrypted ? "Only someone with the password can open these files. Use the same password on every PC." : B.locked ? "Enter it to sync on this PC." : "Anyone with the folder can read the saves. A password encrypts them (AES-256), on every PC that uses this folder."}</p></div>
                <div class="gpage-actions">${B.encrypted ? b("ghost ln-button--sm", 'data-action="backup-pw" data-on="1"', "Change password") + b("ghost ln-button--sm", 'data-action="backup-pw" data-on="0"', "Remove password") : b("secondary ln-button--sm", 'data-action="backup-pw" data-on="1"', B.locked ? "Enter the password" : "Set a password")}</div></div>
            </div></section>
        </aside>
      </div>`, sum);
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
      if (f === "downloadsOff") api.downloads().catch(() => {});
      await refresh();
    } catch (e) { toast(errText(e), "error"); await refresh(); }
  }

