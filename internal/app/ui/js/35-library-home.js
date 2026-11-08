  // --------------------------------------------------------------- library --
  function sessionText(s) {
    const n = pname(s.playerId);
    return { loading: `Loading ${possessive(n)} files`, mods: s.mods && s.mods.off ? "Putting the game back as it was" : `Putting ${possessive(n)} mods in`, launching: "Starting", playing: `${n} is playing`, saving: `Saving ${possessive(n)} progress` }[s.phase] || "";
  }

  // A plugin's games carry their source's own name (f.sourceName).
  const sourceLabel = (src, f) => ({ steam: "Steam", epic: "Epic Games", gog: "GOG", xbox: "Xbox", ea: "EA app", ubisoft: "Ubisoft Connect", battlenet: "Battle.net",
    riot: "Riot Games", amazon: "Amazon Games", rockstar: "Rockstar Games", itch: "itch.io", hydra: "Hydra", shortcut: "Shortcut", folder: "Game folder", added: "Added by you" }[src]
    || ((src || "").startsWith("plugin:") ? (f && f.sourceName) || "Plugin" : "Other"));
  // Each store and launcher is its own filter; folders, shortcuts and games
  // added by hand are "Other".
  const LAUNCHERS = ["steam", "epic", "gog", "xbox", "ea", "ubisoft", "battlenet", "riot", "amazon", "rockstar", "itch", "hydra"];
  const filterOf = (src) => (LAUNCHERS.includes(src) ? src : "other");

  // Every game on the PC: the ones with profiles, then everything the scan
  // found, each copy of a game from more than one store on its own.
  function everyItem() {
    const items = cfg().games.map((g) => ({
      kind: "game", id: g.id, name: g.name, appId: g.launch.type === "steam" ? g.launch.appId : 0,
      source: g.launch.type === "steam" ? "steam" : (S.d.sources || {})[g.id] || "other", exe: g.launch.type !== "steam" ? g.launch.exe || "" : "", g,
    }));
    (S.found.games || []).forEach((f) => items.push({ kind: "found", id: f.key, name: f.name, appId: f.appId || 0, source: f.source, exe: f.appId ? "" : f.exe || "", f }));
    return items.sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: "base" }));
  }
  // The Library as shown: a game from more than one store once, as its
  // primary copy, with every copy in copies. Each copy keeps its own key.
  function libraryItems() {
    const all = everyItem(), drop = new Set();
    copyGroups(all).forEach((grp) => {
      grp.primary.copies = grp.copies; grp.primary.group = grp.key;
      grp.copies.forEach((c) => { if (c !== grp.primary) drop.add(c); });
    });
    return all.filter((it) => !drop.has(it));
  }
  // The scan's groups as items of all: { key, primary, copies }.
  function copyGroups(all) {
    const by = new Map(all.filter((it) => it.kind === "found").map((it) => [it.id, it]));
    return (S.found.groups || []).map((grp) => ({ key: grp.key, primary: by.get(grp.primary), copies: grp.keys.map((k) => by.get(k)).filter(Boolean) }))
      .filter((grp) => grp.primary && grp.copies.length > 1);
  }
  // The group a found game's copy is in, else null.
  const groupOf = (it) => (it.kind === "found" && copyGroups(everyItem()).find((grp) => grp.copies.some((c) => c.id === it.id))) || null;
  const copiesOf = (it) => it.copies || [it];
  const andList = (list) => new Intl.ListFormat("en", { type: "conjunction" }).format(list);
  const storeNames = (it) => andList(copiesOf(it).map((c) => sourceLabel(c.source, c.f)));
  // A small neutral mark per store a game is in.
  const storeMarks = (it) => it.copies ? `<span class="lib-stores">${it.copies.map((c) => `<span class="ln-tag">${esc(sourceLabel(c.source, c.f))}</span>`).join("")}</span>` : esc(sourceLabel(it.source, it.f));

  // Name search that forgives: accents, punctuation and spacing don't
  // matter, words can come in any order, and initials work ("bg3").
  const norm = (s) => String(s || "").normalize("NFD").replace(/\p{M}/gu, "").replace(/[™®©'’‘`]/g, "")
    .replace(/[^\p{L}\p{N}]+/gu, " ").trim().toLowerCase();
  const normNames = new Map();
  function nameMatcher(q) {
    const t = norm(q), words = t.split(" "), flat = t.replace(/ /g, "");
    return (name) => {
      if (!t) return true;
      let n = normNames.get(name);
      if (!n) {
        const ws = norm(name).split(" ");
        n = { text: ws.join(" "), flat: ws.join(""), initials: ws.map((w) => (/^\d+$/.test(w) ? w : w[0] || "")).join("") };
        normNames.set(name, n);
      }
      return words.every((w) => n.text.includes(w)) || n.flat.includes(flat) || n.initials === flat;
    };
  }
  function shownItems() {
    const m = nameMatcher(S.q);
    if (S.filter === "owned") return ownedItems().filter((it) => m(it.name) && (S.store === "all" || filterOf(it.source) === S.store));
    const list = libraryItems().filter((it) => {
      if (!m(it.name)) return false;
      if (S.filter === "profiles" && it.kind !== "game") return false;
      if (S.store !== "all" && !copiesOf(it).some((c) => filterOf(c.source) === S.store)) return false;
      if (S.tag && !tagsOf(it).includes(S.tag)) return false;
      if (S.status) {
        const pid = S.who || (seated() || {}).id;
        if (pid ? statusOf(it, pid) !== S.status : !cfg().players.some((p) => statusOf(it, p.id) === S.status)) return false;
      }
      if (S.who && !playedOrOwned(it, S.who)) return false;
      return true;
    });
    if (S.sort === "recent") list.sort((a, b) => playedAt(b) - playedAt(a));
    else if (S.sort === "played") list.sort((a, b) => timeOf(b) - timeOf(a) || playedAt(b) - playedAt(a));
    else if (S.sort === "rating") list.sort((a, b) => ratingOf(b) - ratingOf(a));
    else if (S.sort === "size") list.sort((a, b) => installOf(b) - installOf(a));
    return list;
  }
  // Collections (the user's own, for everyone) and each profile's status.
  const STATUSES = [["playing", "Playing"], ["next", "Up next"], ["finished", "Finished"], ["complete", "100%"], ["stopped", "Dropped"]];
  const statusName = (k) => (STATUSES.find((x) => x[0] === k) || [k, k])[1];
  // A game from more than one store: in every copy's collections, with the
  // primary copy's status first, then the others', and the best rating.
  const tagsOf = (it) => [...new Set(copiesOf(it).flatMap((c) => (cfg().tags || {})[recentKey(c)] || []))];
  // Every collection in the user's order, as core's Collections: Favourites,
  // the order kept, then any only on games, by name.
  const FAV = "Favourites";
  function allTags() {
    const C = cfg(), seen = new Set(), out = [];
    const add = (t) => { if (t && !seen.has(t.toLowerCase())) { seen.add(t.toLowerCase()); out.push(t); } };
    [FAV, ...(C.tagOrder || [])].forEach(add);
    Object.values(C.tags || {}).flat().map((t) => [t.toLowerCase(), t]).sort((a, b) => (a[0] < b[0] ? -1 : a[0] > b[0] ? 1 : 0)).forEach((x) => add(x[1]));
    return out;
  }
  const usedTags = () => { const u = new Set(Object.values(cfg().tags || {}).flat()); return allTags().filter((t) => u.has(t)); };
  const shelfAt = (pid, key) => ((S.d.shelf || {})[pid] || {})[key] || {};
  const shelfOf = (it, pid) => shelfAt(pid, recentKey(it));
  const statusOf = (it, pid) => shelfOf(it, pid).status || copiesOf(it).map((c) => shelfOf(c, pid).status).find(Boolean) || "";
  const rateOf = (it, pid) => Math.max(0, ...copiesOf(it).map((c) => shelfOf(c, pid).rating || 0));
  // A game's rating in the Library: from the profile it's filtered to or the
  // seated one, else the best any profile gave, as the Status filter does.
  const ratingOf = (it) => {
    const pid = S.who || (seated() || {}).id;
    return pid ? rateOf(it, pid) : Math.max(0, ...cfg().players.map((p) => rateOf(it, p.id)));
  };
  const starsHTML = (n) => n ? `<span class="stars" role="img" aria-label="${n} of 5 stars">${[1, 2, 3, 4, 5].map((i) => `<span${i <= n ? ' class="on"' : ""}>${GI.star}</span>`).join("")}</span>` : "";
  // Changes the fields in patch of a profile's take on a game; the others stay.
  async function saveShelf(pid, key, patch) {
    try { await api.setShelf(pid, key, patch); } catch (e) { toast(errText(e), "error"); return false; }
    await refresh();
    return true;
  }
  // The note is shown only in its field, so saving it leaves the page (and
  // focus) as it is.
  async function saveNote(el) {
    const { player: pid, key } = el.dataset, v = el.value;
    if (v === el.defaultValue) return;
    try { await api.setShelf(pid, key, { note: v }); } catch (e) { toast(errText(e), "error"); return; }
    el.defaultValue = v;
    const sh = S.d.shelf = S.d.shelf || {};
    sh[pid] = { ...sh[pid], [key]: { ...sh[pid] && sh[pid][key], note: v.trim() } };
  }
  const hoursBack = (pid) => { const b = $(`.gp-person-time[data-player="${pid}"]`); if (b) b.focus(); };
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
  // For a game from more than one store, across its copies (display only).
  const playedAt = (it) => Math.max(...copiesOf(it).map((c) => { const r = (S.d.recent || {})[recentKey(c)]; return r ? new Date(r.at).getTime() : 0; }));
  const playtimeOf = (it) => (S.d.playtime || {})[recentKey(it)] || {};
  const totalPlayed = (it) => copiesOf(it).reduce((n, c) => n + Object.values(playtimeOf(c)).reduce((a, b) => a + b, 0), 0);
  const steamTime = (it) => copiesOf(it).reduce((n, c) => n + ((S.d.steamTime || {})[recentKey(c)] || 0), 0);
  const timeOf = (it) => Math.max(totalPlayed(it), steamTime(it));
  // How much a game has been played: Lootnest's own count, or Steam's.
  const playedText = (it) => totalPlayed(it) ? hours(totalPlayed(it)) + " played in all" : steamTime(it) ? hours(steamTime(it)) + " on Steam" : "";
  function hours(sec) {
    if (sec < 3600) return Math.max(1, Math.round(sec / 60)) + " min";
    const h = sec / 3600;
    return (h < 10 ? Math.round(h * 10) / 10 : Math.round(h)) + " h";
  }

  function viewLibrary() {
    if (S.owned === null) { S.owned = []; api.ownedGames().catch(() => {}); }
    const all = libraryItems();
    const withProfiles = cfg().games.length;
    const scanning = S.found.scanning;
    const refreshing = S.found.refreshing;
    const have = new Set(all.flatMap((it) => copiesOf(it).map((c) => filterOf(c.source))));
    const stores = [["all", "All stores"], ...LAUNCHERS.filter((k) => have.has(k) || S.store === k).map((k) => [k, sourceLabel(k)]),
      ...(have.has("other") || S.store === "other" ? [["other", "Other"]] : [])];
    const head = `<header class="lib-head"><div class="lib-title"><h1>Library</h1>
        <span class="lib-meta">${all.length ? `${all.length === 1 ? "1 game" : all.length + " games"}, ${withProfiles === 1 ? "1 with profiles" : withProfiles ? withProfiles + " with profiles" : "none with profiles yet"}` : "Every game on this PC, in one place."}</span></div>
        <div class="lib-actions"><label class="ln-search">${GI.search}<input type="search" id="lib-q" class="ln-input" placeholder="Search your games" value="${esc(S.q)}" autocomplete="off" spellcheck="false" aria-label="Search your games (Ctrl+F)"></label>
        <button class="ln-button ln-button--ghost lib-views" id="lib-views" data-action="lib-views" aria-haspopup="menu" aria-expanded="false"><span>${esc((viewsSaved().find(viewIs) || {}).name || "Views")}</span>${gsvg('<path d="M6.5 8.25 10 11.75l3.5-3.5"/>')}</button>
        <button class="ln-button ln-button--secondary" data-action="add-game">${GI.plus}Add game</button>
        <button class="ln-button ln-button--ghost" data-action="rescan" ${refreshing ? 'aria-busy="true"' : ""}>${refreshing ? '<span class="ln-spinner" aria-hidden="true"></span>Scanning' : "Scan again"}</button>
        <button class="ln-button ln-button--ghost" data-action="sources">Sources</button></div></header>`;
    if (!all.length && !scanning) {
      return `<div class="lib ln-root"><header class="lib-head"><h1>Library</h1><p>Every game on this PC, in one place.</p></header>
        ${emptyState(homeIcon, "No games found yet", "Lootnest looks everywhere games are installed on this PC. If yours isn't found, add it by hand: point Lootnest at its folder or its .exe.",
          `<button class="ln-button ln-button--primary" data-action="add-game">${GI.plus}Add a game</button><button class="ln-button ln-button--secondary" data-action="rescan" ${refreshing ? 'aria-busy="true"' : ""}>${refreshing ? '<span class="ln-spinner" aria-hidden="true"></span>Scanning' : "Scan again"}</button><button class="ln-button ln-button--ghost" data-action="sources">Sources</button>`,
          `<p class="ln-empty__where">Looked in Steam, Epic, GOG, Xbox, EA and ${LAUNCHERS.length - 5 + 2} more places.</p>`, true)}</div>`;
    }
    const sorts = [["name", "Name"], ["recent", "Recently played"], ["played", "Most played"], ["rating", "Highest rated"], ["size", "Largest first"]];
    const sizes = [["s", "Small covers", 8], ["m", "Medium covers", 11], ["l", "Large covers", 14], ["list", "A list", 0]];
    const seg = (k, l, n) => `<button data-action="filter" data-filter="${k}" aria-pressed="${S.filter === k}">${l}<span class="ln-count ln-num">${n}</span></button>`;
    const ps = S.filter === "owned" ? [] : cfg().players; // profiles, collections, status and sort don't apply to games not installed
    const who = ps.length > 3 ? `<select id="lib-who" class="ln-input" aria-label="Whose games"><option value="">Anyone's games</option>${ps.map((p) => `<option value="${esc(p.id)}" ${S.who === p.id ? "selected" : ""}>${esc(possessive(p.name))} games</option>`).join("")}</select>`
      : ps.length > 1 ? `<div class="lib-who" role="group" aria-label="Whose games">${ps.map((p) => `<button class="ln-chip ln-player" style="--player:${esc(p.color)}" data-action="who" data-player="${esc(p.id)}" aria-pressed="${S.who === p.id}" title="${esc(possessive(p.name))} games">${lnAvatar(p, "20")}${esc(p.name)}</button>`).join("")}</div>` : "";
    return `<div class="lib ln-root">${head}${hookBanner() || coverHint(all) ? `<div class="lib-banners">${hookBanner()}${coverHint(all)}</div>` : ""}
      ${(() => { const top = recentShelf(all) + collectionShelves(all); return top ? top + `<div class="ln-section-head lib-all-head"${S.q ? " hidden" : ""}><div><h2>All games</h2></div></div>` : ""; })()}
      <div class="lib-tools">
        <div class="lib-tools-main"><div class="ln-segmented" role="group" aria-label="Show">${seg("all", "All", all.length)}${seg("profiles", "With profiles", withProfiles)}${S.owned.length || S.filter === "owned" ? seg("owned", "Not installed", S.owned.length) : ""}</div>${who}</div>
        <div class="lib-tools-side">
          <select id="lib-store" class="ln-input" aria-label="Store">${stores.map(([k, l]) => `<option value="${k}" ${S.store === k ? "selected" : ""}>${l}</option>`).join("")}</select>
          ${S.filter !== "owned" && (usedTags().length || S.tag) ? `<select id="lib-tag" class="ln-input" aria-label="Collection"><option value="">All collections</option>${usedTags().map((t) => `<option value="${esc(t)}" ${S.tag === t ? "selected" : ""}>${esc(t)}</option>`).join("")}</select>` : ""}
          ${S.filter === "owned" ? "" : `<select id="lib-status" class="ln-input" aria-label="Status"><option value="">Any status</option>${STATUSES.map(([k, l]) => `<option value="${k}" ${S.status === k ? "selected" : ""}>${l}</option>`).join("")}</select>
          <select id="lib-sort" class="ln-input" aria-label="Sort by">${sorts.map(([k, l]) => `<option value="${k}" ${S.sort === k ? "selected" : ""}>${l === "Name" ? "Sort by name" : l}</option>`).join("")}</select>`}
          <div class="ln-segmented lib-size" role="group" aria-label="Cover size">${sizes.map(([k, l, h]) => `<button data-action="density" data-size="${k}" aria-pressed="${S.density === k}" aria-label="${l}" title="${l}">${h ? `<span style="--h:${h}px"></span>` : '<i class="lib-size-list" aria-hidden="true"></i>'}</button>`).join("")}</div>
        </div></div>
      ${scanning ? lnInline("", "Looking for games on this PC", "Games appear as they're found.") : ""}
      <div class="lib-grid size-${esc(S.density)}" id="grid">${gridHTML()}</div>
      <div class="lib-sel" id="lib-sel" role="region" aria-label="Selected games"${S.libSel.size ? "" : " hidden"}>${libSelHTML()}</div></div>`;
  }

  // Saved views: the Library's filters under a name (core.LibraryView).
  // Applying one sets them all; it never changes a game.
  const viewsSaved = () => cfg().views || [];
  const VIEW_DEF = { show: "all", store: "all", sort: "name", density: "m" };
  const viewNow = () => ({ show: S.filter, store: S.store, collection: S.tag, status: S.status, profile: S.who, sort: S.sort, density: S.density });
  const viewIs = (v) => Object.entries(viewNow()).every(([k, x]) => (v[k] || VIEW_DEF[k] || "") === x);
  function applyView(name) {
    const v = viewsSaved().find((x) => x.name === name);
    if (!v) return;
    S.filter = v.show || "all"; S.store = v.store || "all"; S.tag = v.collection || ""; S.status = v.status || "";
    S.who = v.profile && player(v.profile) ? v.profile : "";
    S.sort = v.sort || "name"; setPref("sort", S.sort);
    S.density = v.density || "m"; setPref("density", S.density);
    uiSound("tab");
    render();
  }
  // naming: the view being renamed, or "" for a new one; left out, the list.
  function viewsMenuHTML(naming, value) {
    if (naming !== undefined) {
      return `<div class="ctx-title">${naming ? `Rename ${esc(naming)}` : "Save this view"}</div>
        <div class="ctx-name"><input id="view-name" class="ln-input" type="text" maxlength="40" autocomplete="off" spellcheck="false" value="${esc(value === undefined ? naming : value)}" data-old="${esc(naming)}" placeholder="Name this view" aria-label="${naming ? `A new name for ${esc(naming)}` : "The view's name"}">
        <button data-action="view-save">Save</button></div>`;
    }
    const vs = viewsSaved();
    const rows = vs.map((v) => {
      const on = viewIs(v), a = `data-name="${esc(v.name)}"`;
      return `<div class="ctx-view"><button role="menuitemradio" aria-checked="${on}" data-action="view-apply" ${a}><span class="grow">${esc(v.name)}</span>${on ? GI.tick : ""}</button>
        <button class="ctx-icon" data-action="view-rename" ${a} aria-label="Rename ${esc(v.name)}" title="Rename">${GI.edit}</button>
        <button class="ctx-icon" data-action="view-delete" ${a} aria-label="Delete ${esc(v.name)}" title="Delete">${GI.trash}</button></div>`;
    }).join("");
    return `<div class="ctx-title">Saved views</div>${rows || '<p class="ctx-note">Save the filters, sort and cover size you use often, and get them back in one click.</p>'}<hr>
      <button data-action="view-new"${vs.length >= 20 ? ' disabled title="Lootnest keeps up to 20 views."' : ""}>${GI.plus}Save this view</button>`;
  }
  function viewNaming(old, value) {
    const b = $("#lib-views");
    if (!b) return;
    const i = menuUnder(b, viewsMenuHTML(old, value)).querySelector("#view-name");
    i.focus(); i.select();
  }
  async function saveViewName(i) {
    const name = i.value.trim(), old = i.dataset.old;
    if (!name) { if (i.isConnected) i.focus(); else viewNaming(old, ""); return; }
    const v = old ? { ...viewsSaved().find((x) => x.name === old), name } : { name, ...viewNow() };
    try { await api.saveView(v, old); } catch (e) { toast(errText(e), "error"); viewNaming(old, name); return; }
    closeContext(true);
    await refresh();
    toast(old ? `Renamed ${old} to ${name}.` : `Saved ${name}. Find it under Views.`);
  }

  // Picking games in the Library: Ctrl+click one, Shift+click a run of
  // them, Ctrl+A all shown, Esc none. The bar at the bottom acts on them all,
  // and on every store's copy of each.
  const libSelItems = () => keyedItems([...S.libSel]);
  const libSelKeys = () => libSelItems().flatMap((it) => copiesOf(it).map(recentKey));
  const canHide = (it) => it.kind === "found" && !it.f.added;
  const gridKeys = () => [...document.querySelectorAll("#grid [data-k]")].map((c) => c.dataset.k);
  function libPick(keys, on) {
    const was = S.libSel.size;
    keys.forEach((k) => (on ? S.libSel.add(k) : S.libSel.delete(k)));
    if (!was && S.libSel.size) uiSound("tab");
    paintLibSel();
  }
  function libPickRange(a, b) {
    const ks = gridKeys(), i = ks.indexOf(a), j = ks.indexOf(b);
    if (i < 0 || j < 0) { libPick([b], true); S.libAnchor = b; return; }
    libPick(ks.slice(Math.min(i, j), Math.max(i, j) + 1), true);
  }
  function libSelClear() {
    if (!S.libSel.size) return;
    S.libSel.clear();
    paintLibSel();
  }
  function paintLibSel() {
    document.querySelectorAll("#grid [data-k]").forEach((c) => c.classList.toggle("is-sel", S.libSel.has(c.dataset.k)));
    const box = $("#lib-sel");
    if (!box) return;
    const was = !box.hidden;
    box.innerHTML = libSelHTML();
    box.hidden = !S.libSel.size;
    if (!was && !box.hidden) { box.classList.remove("in"); void box.offsetWidth; box.classList.add("in"); }
  }
  function libSelHTML() {
    const n = S.libSel.size;
    if (!n) return "";
    const items = libSelItems(), owned = S.filter === "owned", fav = items.length && items.every((it) => tagsOf(it).includes(FAV));
    const b = (act, label, extra = "") => `<button class="ln-button ln-button--ghost ln-button--sm" data-action="${act}"${extra}>${label}</button>`;
    const menu = ' aria-haspopup="menu" aria-expanded="false"';
    return `<span class="lib-sel-n" role="status"><b class="ln-num">${n}</b> selected</span>
      <div class="lib-sel-acts">${b("sel-tags", "Add to collection", menu)}
        ${S.tag && !owned ? b("sel-untag", `Remove from ${esc(S.tag)}`) : ""}
        ${owned ? "" : b("sel-status", "Set status", menu)}
        ${b("sel-fav", fav ? "Remove from Favourites" : "Favourite")}
        ${owned ? "" : b("sel-hide", "Hide from the Library", items.some(canHide) ? "" : ' disabled title="Only games Lootnest found can be hidden. Games with profiles and games you added are taken out from their own page."')}</div>
      ${b("sel-clear", `${GI.x}Clear selection`)}`;
  }
  function selStatusHTML() {
    const me = seated();
    if (!me) return `<div class="ctx-title">Nobody's at the PC</div><button data-action="sel-seat">Choose who's here</button>`;
    return `<div class="ctx-title">Status for ${esc(me.name)}</div>${STATUSES.map(([k, l]) => `<button data-action="sel-status-set" data-status="${k}">${l}</button>`).join("")}
      <hr><button data-action="sel-status-set" data-status="">Clear the status</button>`;
  }
  async function libSetStatus(status) {
    const me = seated(), n = S.libSel.size, text = gamesText([...S.libSel]);
    if (!me || !n) return;
    try { for (const k of libSelKeys()) await api.setShelf(me.id, k, { status }); } catch (e) { toast(errText(e), "error"); await refresh(); return; }
    await refresh();
    uiSound("move");
    toast(status ? `Set ${text} to ${statusName(status)} for ${me.name}.` : `Cleared the status of ${text} for ${me.name}.`);
  }
  async function libHide() {
    const items = libSelItems(), hide = items.filter(canHide);
    // A copy added by hand would leave the Library for good, so it stays.
    const keys = hide.flatMap((it) => copiesOf(it).filter((c) => !c.f.added).map((c) => c.id));
    if (!keys.length) return;
    const text = hide.length === 1 ? hide[0].name : hide.length + " games";
    const kept = keys.length < items.flatMap(copiesOf).length;
    if (await hideFound(keys, `Hid ${text} from the Library.${kept ? " Games with profiles and games you added stay." : ""}`)) uiSound("move");
  }
  // Takes found games out of the Library; with text, a toast offers Undo.
  async function hideFound(keys, text) {
    try { for (const k of keys) await api.ignoreFound(k); } catch (e) { toast(errText(e), "error"); await loadFound(); await refresh(); return false; }
    S.found.games = S.found.games.filter((x) => !keys.includes(x.key));
    await loadFound();
    await refresh();
    if (text) toast(text, "success", { label: "Undo", run: async () => { for (const k of keys) await actions["found-unhide"]({ dataset: { key: k } }); } });
    return true;
  }
  // What Steam Cloud replaced in a session, on the game's page until read.
  function conflictBanner(g) {
    const c = (S.d.conflicts || {})[g.id];
    if (!c) return "";
    const who = pname(c.player) || "Someone";
    const when = new Date(c.at).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
    return `<div class="gp-conflict">${banner(`Steam Cloud replaced ${esc(possessive(who))} files on ${esc(when)}`,
      `${esc(who)}'s profile stayed as it was. That session's files, from Steam Cloud, are kept as a copy, so nothing was lost. Turn Steam Cloud off for ${esc(g.name)} in its Steam properties if this keeps happening.`,
      bBtn("secondary", `data-action="copies" data-game="${esc(g.id)}"`, "See the copies") + bBtn("ghost", `data-action="conflict-ok" data-game="${esc(g.id)}"`, "Got it"), false, "warn")}</div>`;
  }
  // A game that ships with an anti-cheat: what Lootnest does and doesn't do.
  function antiCheatNote(it) {
    const key = recentKey(it), dir = it.f ? it.f.dir : it.exe;
    if (!dir) return "";
    if (!(key in S.facts)) { S.facts[key] = null; api.gameFacts(dir).then((f) => { S.facts[key] = f || {}; if (f && f.antiCheat && S.view === "game") render(); }).catch(() => { S.facts[key] = {}; }); }
    const f = S.facts[key];
    if (!f || !f.antiCheat) return "";
    return `<p class="gpage-note gp-ac">${GI.shield || GI.copy}Ships with ${esc(f.antiCheat)}. Lootnest never touches the game's own files: profiles swap only its settings and saves, no notes show over it, and mods are off for it. Progress kept on the game's servers stays with the account.</p>`;
  }
  // The copies made before each swap, with Put back on each.
  function viewCopies() {
    const D = S.dialog, g = game(D.gameId);
    if (!g) return "";
    const list = D.list;
    const when = (w) => (w ? new Date(w).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" }) : "");
    const body = !list ? lnInline("", "Reading the copies") : !list.length ? `<p>No copies yet. One is made before every swap.</p>`
      : `<ol class="pt-list pt-list--all">${list.map((b, i) => { const p = player(b.owner);
        return `<li class="pt-row"><span class="pt-dot" aria-hidden="true"></span><div class="pt-main"><b>${esc(when(b.when) || b.id)}${i === 0 ? " (newest)" : ""}</b>
          <span class="pt-facts">${p ? `<span><span class="ln-mark ln-player" style="--player:${esc(p.color)}"></span>${esc(possessive(p.name))} files</span>` : "<span>Whose isn't known</span>"}<span>${esc(fmtBytes(b.size || 0))}</span></span></div>
          <div class="pt-actions">${p ? `<button class="ln-button ln-button--secondary ln-button--sm" data-action="copy-put-back" data-game="${esc(g.id)}" data-id="${esc(b.id)}" data-owner="${esc(b.owner)}" data-when="${esc(when(b.when))}" ${D.busy ? "disabled" : ""}>Put back</button>` : ""}</div></li>`; }).join("")}</ol>`;
    return drawer({ label: `${esc(g.name)} copies`, title: "Copies before swaps", sub: `${esc(g.name)}, newest first`, body,
      foot: `<p class="gpage-hint ln-drawer__note">Put back gives a copy's files back to the profile they belong to. What's there now is copied first, so it can be undone too.</p><button class="ln-button ln-button--ghost ln-button--sm" data-action="open-backups" data-game="${esc(g.id)}">Open the folder</button>` });
  }
  // A page with nothing in it yet: what it's for, and the way to fill it.
  // ghosts draws dashed covers where a page's games will go.
  function emptyState(ic, title, text, actions, extra, ghosts) {
    return `<section class="ln-empty"><h2>${ic}${title}</h2><p>${text}</p>
      ${actions ? `<div class="ln-empty__act">${actions}</div>` : ""}${extra || ""}${ghosts ? `<div class="ln-ghosts" aria-hidden="true">${"<span></span>".repeat(16)}</div>` : ""}</section>`;
  }
  // Recent games, as at the top of Steam's library: the last few played, on
  // their wide art, one click from Play. Only on the Library as it opens,
  // not while it's filtered; a search hides it.
  function recentShelf(all) {
    if (S.filter !== "all" || S.who || S.store !== "all" || S.tag || S.status) return "";
    const rec = recentItems(all, false).slice(0, 5);
    if (rec.length < 2) return "";
    // The row leads with art: the second game moves up when only it has some.
    // Only the first two slots show at every width.
    const hasArt = (it) => artFor(it, "hero").url;
    if (!hasArt(rec[0]) && hasArt(rec[1])) rec.unshift(rec.splice(1, 1)[0]);
    return `<section class="lib-recent" aria-label="Recent games"${S.q ? " hidden" : ""}><div class="ln-section-head"><div><h2>Recent games</h2></div></div>
      <div class="lib-recent-row">${rec.map(capHTML).join("")}</div></section>`;
  }
  // A game on its wide art, with who played it last; Play on hover.
  function capHTML(it) {
    loadArt(it); ensureCover(it);
    const a = artFor(it, "hero"), art = a.url ? artImg(a, "hero") : "";
    const t = timeOf(it);
    const s = it.g && session(it.g.id);
    return `<div class="lib-cap${s ? " busy" : ""}${art ? "" : " no-art"}"><button class="lib-cap-open" data-action="open-sheet" ${itemAttr(it)} aria-label="${esc(it.name)}">
        <span class="lib-cap-art">${art || typeCover(it, "lib-cap-none")}</span>
        <span class="lib-cap-text"><b>${esc(it.name)}</b><span>${playedBy(it)}${t ? `<span class="lib-cap-time ln-num">${esc(hours(t))}</span>` : ""}</span></span></button>
      ${s ? `<span class="lib-cap-badge ln-tag ln-tag--player ln-player" style="--player:${esc(color(s.playerId))}">${esc(sessionText(s))}</span>`
        : `<button class="ln-button ln-button--primary ln-button--icon ln-button--sm lib-cap-play" data-action="${it.kind === "game" ? "play" : "play-found"}" ${itemAttr(it)} aria-label="Play ${esc(it.name)}" title="Play">${GI.play}</button>`}</div>`;
  }
  // A shelf for each collection, under Recent games, as in Steam: its
  // covers in one row that scrolls sideways, and See all to filter the
  // Library to it. Same rules as Recent games for when they show.
  function collectionShelves(all) {
    if (S.filter !== "all" || S.who || S.store !== "all" || S.tag || S.status) return "";
    const by = {};
    all.forEach((it) => tagsOf(it).forEach((t) => (by[t] = by[t] || []).push(it)));
    const names = allTags().filter((t) => (by[t] || []).length >= 4);
    if (!names.length) return "";
    // The first three of four games or more, in the sidebar's order; the rest are in the Library's collections menu.
    return `<div class="lib-shelves"${S.q ? " hidden" : ""}>${names.slice(0, 3).map((t) => {
      const list = by[t].slice().sort((a, b) => playedAt(b) - playedAt(a) || a.name.localeCompare(b.name));
      return `<section class="lib-shelf" aria-label="${esc(t)}"><div class="ln-section-head"><div><h2>${esc(t)}<span class="lib-shelf-n ln-num">${list.length}</span></h2></div>
          <button class="ln-button ln-button--ghost ln-button--sm" data-action="shelf-all" data-tag="${esc(t)}">See all</button></div>
        <div class="lib-shelf-row">${list.slice(0, 16).map((it) => cardHTML(it)).join("")}</div></section>`;
    }).join("")}</div>`;
  }

  // Home: the game played last with its play bar, the others played lately
  // and their news; beside them who's playing, tonight's pick and the saves.
  function viewHome() {
    const all = libraryItems();
    const recent = recentItems(all, false);
    const banners = safeBanner() + updateBanner() + hookBanner();
    const allBtn = all.length ? `<button class="ln-button ln-button--ghost ln-button--sm" data-action="nav" data-view="library">All ${all.length === 1 ? "1 game" : all.length + " games"}</button>` : "";
    if (!recent.length) {
      const step = (mark, title, sub, act) => `<li>${mark}<div><b>${title}</b><span>${sub}</span></div>${act}</li>`;
      const btn = (attrs, label) => `<button class="ln-button ln-button--secondary ln-button--sm" ${attrs}>${label}</button>`;
      const n = all.length, num = (i) => `<span class="ln-num">${i}</span>`;
      const steps = [
        S.found.scanning ? step('<span class="ln-spinner" aria-hidden="true"></span>', "Looking for games on this PC", "Games appear in the Library as they're found.", "")
          : n ? step(GI.tick, `Lootnest found ${n === 1 ? "1 game" : n + " games"} on this PC`, "From every launcher, game folder and shortcut it knows.", btn('data-action="nav" data-view="library"', "Open the Library"))
          : step(num(1), "Lootnest found no games on this PC yet", "It looks in every launcher, game folder and shortcut it knows.", btn('data-action="rescan"', "Scan again")),
        step(num(2), "Add a game from a folder", "A game that isn't found: point Lootnest at its folder or its .exe.", btn('data-action="add-game"', "Add a game")),
        solo() ? step(num(3), "Start a run of a game", "Open a game and choose New run: a fresh start with its own saves, and yours kept.", n ? btn('data-action="nav" data-view="library"', "Choose a game") : "")
          : step(GI.tick, "Make a profile for everyone who plays", `${people().length} people have one, each with their own saves, settings and achievements in every game.`, btn('data-action="nav" data-view="players"', "Open Profiles")),
      ];
      return `<div class="home-empty ln-root">${banners}
        <header class="lib-head"><div><h1>Home</h1></div>${allBtn}</header>
        <div class="home-cols">
          <div class="home-main">${emptyState(homeIcon, "Nothing played yet", n ? "Start a game from your Library and it shows up here, with who played it, the others you play and their news." : "Games you play show up here, with who played them and their news.", "", `<ol class="ln-steps">${steps.join("")}</ol>`)}</div>
          <aside class="gpage-aside">${playersBlock(all)}</aside>
        </div></div>`;
    }
    const first = recent[0];
    const head = gameView(first, false);
    return `<div class="gpage home ln-root${head.real ? "" : " no-art"}">${head.top}
      <div class="gpage-body">${head.bar}${head.note}${banners ? `<div class="home-banners">${banners}</div>` : ""}
        <div class="home-cols">
          <div class="home-main">${continueShelf(recent.slice(1), allBtn)}${homeNews(recent)}</div>
          <aside class="gpage-aside">${playersBlock(all)}${tonightBlock(all)}${saveCheckBlock()}</aside>
        </div>
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

  // "Mihai, yesterday": who played it last, in their colour.
  function playedBy(it) {
    const r = copiesOf(it).map((c) => (S.d.recent || {})[recentKey(c)]).filter(Boolean).sort((a, b) => new Date(b.at) - new Date(a.at))[0];
    if (!r) return `<span class="by">${esc(sourceLabel(it.source, it.f))}</span>`;
    const p = r.player && player(r.player);
    return p ? `<span class="ln-mark ln-player" style="--player:${esc(p.color)}"></span><span class="by">${esc(p.name)}, ${esc(ago(r.at))}</span>` : `<span class="by">${esc(cap(ago(r.at)))}</span>`;
  }
  const cap = (t) => t.charAt(0).toUpperCase() + t.slice(1);

  function continueShelf(list, allBtn) {
    const head = `<div class="ln-section-head"><div><h2>Continue playing</h2></div>${allBtn}</div>`;
    if (!list.length) return `<section>${head}<p class="home-hint">The other games you play show up here.</p></section>`;
    return `<section aria-label="Continue playing">${head}<div class="lib-recent-row">${list.slice(0, 5).map(capHTML).join("")}</div></section>`;
  }

  // What to play tonight: one game for whoever's here, from their Up next,
  // what they're playing and what they never started. The pick holds for
  // the day; Another spins again.
  function tonightPicks(all) {
    const me = seated() || cfg().players[0];
    const pid = me ? me.id : "", day = 864e5, now = Date.now();
    const out = [];
    all.forEach((it) => {
      const st = pid ? statusOf(it, pid) : "";
      if (st === "finished" || st === "complete" || st === "stopped") return;
      const at = playedAt(it);
      let w, why;
      if (st === "next") { w = 6; why = "Up next"; }
      else if (st === "playing") { w = 3; why = at ? `Playing, last played ${ago(at)}` : "Playing"; }
      else if (!at) { w = 1.5; why = "Never started"; }
      else if (now - at > 30 * day) { w = 1.2; why = "Not played in a while"; }
      else { w = 0.6; why = `Played ${ago(at)}`; }
      if (at && now - at < day) w *= 0.3; // already on Continue playing
      out.push({ it, w, why });
    });
    return out;
  }
  // A weighted pick; seed gives the same one all day.
  function tonightPick(picks, seed, avoid) {
    const list = picks.filter((p) => p.it.id !== avoid);
    if (!list.length) return picks[0] || null;
    let r = seed === undefined ? Math.random() : ((Math.sin(seed) + 1) / 2) % 1;
    const total = list.reduce((n, p) => n + p.w, 0);
    for (const p of list) { r -= p.w / total; if (r <= 0) return p; }
    return list[list.length - 1];
  }
  function tonightBlock(all) {
    const picks = tonightPicks(all);
    if (picks.length < 2) return "";
    const me = seated() || cfg().players[0];
    const today = Math.floor(Date.now() / 864e5);
    if (!S.tonight || S.tonight.day !== today || S.tonight.seat !== (me && me.id)) {
      const p = tonightPick(picks, today * 7.13 + ((me && me.id) || "").length);
      S.tonight = { day: today, seat: me && me.id, id: p && p.it.id };
    }
    const pick = picks.find((p) => p.it.id === S.tonight.id) || picks[0];
    const it = pick.it;
    ensureCover(it);
    const a = artFor(it, "cover");
    const play = it.kind === "game"
      ? `<button class="ln-button ln-button--secondary ln-button--sm" data-action="play" data-game="${esc(it.id)}">${GI.play}Play</button>`
      : `<button class="ln-button ln-button--secondary ln-button--sm" data-action="play-found" data-key="${esc(it.id)}">${GI.play}Play</button>`;
    return `<section class="tonight" aria-label="What to play tonight"><div class="ln-section-head"><div><h2>Tonight</h2></div></div>
      <div class="tn-card">
        <button class="tn-cover" data-action="open-sheet" ${itemAttr(it)} aria-label="${esc(it.name)}">${a.url ? artImg(a, "cover") : `<span class="tn-none">${esc(it.name)}</span>`}</button>
        <div class="tn-info"><span class="tn-why">${esc(pick.why)}</span><b class="tn-name">${esc(it.name)}</b>
          <div class="tn-actions">${play}<button class="ln-button ln-button--ghost ln-button--sm" data-action="tonight-spin">Another</button></div></div>
      </div></section>`;
  }
  // Your saves: how Lootnest looks after them, from the save check.
  let saveCheckAsked = false;
  function saveCheckBlock() {
    if (!saveCheckAsked) { saveCheckAsked = true; api.saveCheck(false).catch(() => {}); }
    const C = S.saveCheck;
    if (!C) return "";
    const n = (k, one, many) => (k === 1 ? one : many.replace("#", k));
    const stats = [C.profiles && gpStat(C.profiles, "With profiles"), C.net && gpStat(C.net, "With copies"),
      C.points && gpStat(C.points, "With save points")].filter(Boolean).join("");
    // A problem: its tone, and one link to see to it (a menu when it's more than one game).
    const fix = (kind, list, label) => list.length === 1 ? `<button class="gpage-link" data-action="open-sheet" ${list[0].key.startsWith("game:") ? `data-game="${esc(list[0].key.slice(5))}"` : `data-key="${esc(list[0].key)}"`} data-tab="saves">${label}</button>`
      : `<button class="gpage-link" data-action="sc-games" data-kind="${kind}" aria-haspopup="menu">${label}</button>`;
    const line = (tone, text) => `<li class="sc-line${tone ? " sc-line--" + tone : ""}">${tone ? GI.alert : "<span></span>"}<span>${text}</span></li>`;
    const gone = C.gone || [], lines = [];
    if (gone.length) lines.push(line("low", `${gone.length === 1 ? `${esc(gone[0].name)}'s saves are gone from its folder.` : `${gone.length} games' saves are gone from their folders.`} ${fix("gone", gone, gone.length === 1 ? "Put a copy back" : "Put copies back")}`));
    if (C.unknownN) lines.push(line("mid", fix("unknown", C.unknown || [], n(C.unknownN, "Choose where 1 game keeps saves", "Choose where # games keep saves"))));
    const od = C.oneDrive || [];
    if (od.length) lines.push(line("", `${n(od.length, "1 game keeps", "# games keep")} saves in ${esc(C.oneDriveIn)}, which OneDrive syncs. <button class="gpage-link" data-action="sc-games" data-kind="oneDrive" aria-haspopup="menu">Which games</button>`));
    lines.push(line("", C.cloud ? "Cloud saves are on." : `Cloud saves are off. <button class="gpage-link" data-action="nav" data-view="cloud">Set them up</button>`));
    return `<section class="home-block sc"><div class="ln-section-head"><div><h2>Your saves</h2>${stats ? "<p>Games Lootnest looks after</p>" : ""}</div><button class="ln-button ln-button--ghost ln-button--sm" data-action="nav" data-view="saves">See all</button></div>${stats ? `<dl class="ln-meta sc-stats">${stats}</dl>` : ""}<ul class="sc-lines">${lines.join("")}</ul></section>`;
  }
  // A game whose saves are in a folder OneDrive syncs (Documents moved
  // into OneDrive): said once, as information.
  function oneDriveNote(it, cls) {
    if (!saveCheckAsked) { saveCheckAsked = true; api.saveCheck(false).catch(() => {}); }
    const C = S.saveCheck;
    if (!C || !(C.oneDrive || []).some((g) => g.key === recentKey(it))) return "";
    return `<p class="${cls}">These saves are in ${esc(C.oneDriveIn)}, which OneDrive syncs.</p>`;
  }
  // Each person and the game they played last; a click sits them at the PC.
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
    const me = seated(), tag = ps.length > 1 && me ? `<span class="ln-tag ln-tag--player ln-player" style="--player:${esc(me.color)}">${solo() ? "Playing" : "Seated"}</span>` : "";
    return `<section class="gpage-block gp-players"><div class="ln-section-head"><div><h2>Who's playing</h2></div></div><div class="ln-list home-players">${ps.map((p) => {
      const l = lastOf[p.id], m = !l && most[p.id];
      const it = (l && itemByKey(l.k)) || (m && itemByKey(m.k));
      return `<button class="home-player" data-action="seat" data-player="${esc(p.id)}" aria-pressed="${!!(me && me.id === p.id)}">${lnAvatar(p, "32")}<span><b>${esc(p.name)}</b><span>${it ? (l ? `${esc(it.name)}, ${esc(ago(l.at))}` : `Most played: ${esc(it.name)}, ${esc(hours(m.sec))}`) : "Hasn't played yet"}</span></span>${me && me.id === p.id ? tag : ""}</button>`;
    }).join("")}</div></section>`;
  }

  function ensureNews(it) {
    const key = coverKey(it);
    if (S.news[key] || !cfg().settings.gameNews) return;
    S.news[key] = { loading: true };
    api.gameNews(key, it.appId, it.name, storeKept(it.source) ? "" : it.exe).catch(() => {});
  }
  // News comes after the games: the four newest posts, two to a row.
  function homeNews(recent) {
    if (!cfg().settings.gameNews) return "";
    const items = [];
    recent.slice(0, 8).forEach((it) => {
      ensureNews(it);
      const N = S.news[coverKey(it)];
      ((N && N.items) || []).slice(0, 3).forEach((n, i) => items.push({ it, n, i }));
    });
    items.sort((a, b) => new Date(b.n.date) - new Date(a.n.date));
    return newsSection("News from your games", items.slice(0, 4), true);
  }

  // Notices at the top of Home and the Library (system.css .ln-banner).
  const banner = (title, text, actions, busy, tone) => `<div class="ln-banner ln-root">${tone === "warn" ? GI.alert : busy ? '<span class="ln-spinner" aria-hidden="true"></span>' : ""}<div class="ln-banner__text"><b>${title}</b><p>${text}</p></div>${actions ? `<div class="ln-banner__actions">${actions}</div>` : ""}</div>`;
  const bBtn = (kind, attrs, label) => `<button class="ln-button ln-button--${kind}" ${attrs}>${label}</button>`;

  const safeBanner = () => S.d.safe ? banner("Safe mode", "The safety net, cloud uploads and other background work are off until Lootnest restarts normally.", bBtn("secondary", 'data-action="restart-normal"', "Restart normally")) : "";

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
    const items = shownItems(), shown = new Set(items.map(recentKey));
    S.libSel.forEach((k) => { if (!shown.has(k)) S.libSel.delete(k); }); // only games still shown stay picked
    if (!items.length) return `<p class="lib-none">${S.q ? `No game matches “${esc(S.q)}”.` : S.filter === "owned" ? `Every game ${["Steam", "Epic", gogOn() && "GOG", itchOn() && "itch.io"].filter(Boolean).join(", ").replace(/, ([^,]*)$/, " and $1")} know you have is installed.` : S.who ? `${esc(pname(S.who))} has no games here yet.` : "No games here."}</p>`;
    if (S.density === "list") {
      return `<div class="lib-list-head" aria-hidden="true"><span></span><span>Name</span><span>Last played</span><span>Time played</span><span>Profiles</span><span></span></div>${items.map(listRowHTML).join("")}`;
    }
    return items.map((it) => cardHTML(it, true)).join("");
  }
  // The Library as a list, as in Steam's: a row per game with when it was
  // last played, how long, and who has a profile.
  function listRowHTML(it) {
    const ck = coverKey(it), c = artFor(it, "cover").url, ic = !c && S.icons[ck];
    ensureCoverQuiet(it);
    const g = it.g, s = g && session(g.id), owner = g && S.d.active[g.id];
    const owners = g ? cfg().players.filter((p) => hasProfile(g.id, p.id)) : [];
    const t = timeOf(it);
    const thumb = c ? `<img src="${c}" alt="">` : ic ? `<img class="icon" src="${ic}" alt="">` : `<b>${esc(initial(it.name))}</b>`;
    if (it.kind === "owned") {
      return `<div class="lib-row is-owned${S.libSel.has(recentKey(it)) ? " is-sel" : ""}" data-k="${esc(recentKey(it))}"><button class="lib-row-open" data-action="open-sheet" ${itemAttr(it)}>
        <span class="lib-row-thumb">${thumb}</span>
        <span class="lib-row-name"><b>${esc(it.name)}</b><small>${esc(sourceLabel(it.source, it.f))}, not installed</small></span>
        <span class="lib-row-cell"></span><span class="lib-row-cell"></span><span class="lib-row-cell"></span></button>${installButton(it, "ln-button--secondary ln-button--sm lib-row-play", "Install")}</div>`;
    }
    const prof = owners.length ? owners.map((p) => `<span class="ln-mark ln-player${p.id === owner ? " is-loaded" : ""}" style="--player:${esc(p.color)}" title="${esc(p.name)}${p.id === owner ? ", files loaded" : ""}"></span>`).join("") : `<span class="lib-row-dim">Off</span>`;
    const play = s ? `<span class="ln-tag ln-tag--player ln-player" style="--player:${esc(color(s.playerId))}">${esc(sessionText(s))}</span>`
      : `<button class="ln-button ln-button--primary ln-button--sm lib-row-play" data-action="${g ? "play" : "play-found"}" ${itemAttr(it)} aria-label="Play ${esc(it.name)}">${GI.play}Play</button>`;
    return `<div class="lib-row${s ? " busy" : ""}${S.libSel.has(recentKey(it)) ? " is-sel" : ""}" data-k="${esc(recentKey(it))}"><button class="lib-row-open" data-action="open-sheet" ${itemAttr(it)}>
        <span class="lib-row-thumb">${thumb}</span>
        <span class="lib-row-name"><b>${esc(it.name)}</b><small>${storeMarks(it)}${starsHTML(ratingOf(it))}</small></span>
        <span class="lib-row-cell">${playedAt(it) ? playedBy(it) : '<span class="lib-row-dim">Never</span>'}</span>
        <span class="lib-row-cell ln-num">${t ? esc(hours(t)) : '<span class="lib-row-dim">None</span>'}</span>
        <span class="lib-row-cell lib-row-prof">${prof}</span></button>${play}</div>`;
  }
  // A cover for a row of the list, without drawing the whole page again.
  function ensureCoverQuiet(it) {
    const key = coverKey(it);
    if (S.covers[key] || S.icons[key] || coverAsked.has(key)) return;
    Promise.resolve().then(() => sideThumb(it)).then(() => {
      const c = S.covers[key] || S.icons[key];
      if (!c) return;
      document.querySelectorAll(".lib-row-open").forEach((b) => {
        const x = itemFor(b);
        if (x && coverKey(x) === key) b.querySelector(".lib-row-thumb").innerHTML = `<img${S.covers[key] ? "" : ' class="icon"'} src="${c}" alt="">`;
      });
    });
  }

  // Steam games have a cover in Steam's cache; other games only when the
  // user turned on "Download covers".
  // A cover with no art (system.css .ln-tile__none), washed in the game's icon when it has one.
  const plate = (ic, cls, inner, attr = "") => `<span class="ln-tile__none${cls ? " " + cls : ""}${ic ? " with-icon" : ""}"${ic ? ` style="--icon:url('${ic}')"` : ""}${attr}>${inner}</span>`;
  const initial = (name) => (name.match(/[A-Za-z0-9]/) || ["?"])[0].toUpperCase();
  const coverKey = (it) => (it.appId ? "a" + it.appId : "n:" + it.name.toLowerCase());
  // Art the user chose in Manage, read once per file and kept as a short
  // blob: URL, so a big picture isn't pasted into every render.
  const customAsked = new Set();
  function blobURL(d) {
    const i = d.indexOf(","), bin = atob(d.slice(i + 1)), b = new Uint8Array(bin.length);
    for (let j = 0; j < bin.length; j++) b[j] = bin.charCodeAt(j);
    return URL.createObjectURL(new Blob([b], { type: d.slice(5, d.indexOf(";")) }));
  }
  function customArt(it, kind) {
    const lk = recentKey(it), file = ((S.d.customArt || {})[lk] || {})[kind];
    if (!file) return "";
    const k = lk + ":" + file;
    if (!(k in S.custom) && !customAsked.has(k)) {
      customAsked.add(k);
      api.customArt(lk, kind).then((u) => { S.custom[k] = u ? blobURL(u) : ""; if (u) render(); }).catch(() => { S.custom[k] = ""; });
    }
    return S.custom[k] || "";
  }
  // After a change in Manage, the next read gets the new file.
  function forgetCustom(it) {
    const pre = recentKey(it) + ":";
    Object.keys(S.custom).forEach((k) => { if (k.startsWith(pre)) { if (S.custom[k]) URL.revokeObjectURL(S.custom[k]); delete S.custom[k]; } });
    customAsked.forEach((k) => { if (k.startsWith(pre)) customAsked.delete(k); });
  }
  // The best art of a kind (cover, hero, logo) as { url, from }: the user's,
  // then the store's, then the other kind to crop from (none for a logo).
  // No url: draw typeCover().
  function artFor(it, kind) {
    const key = coverKey(it);
    const own = (k) => customArt(it, k) || (k === "cover" ? S.covers[key] : S.art[k + ":" + key]) || "";
    const url = own(kind);
    if (url || kind === "logo") return { url, from: kind };
    const other = kind === "cover" ? "hero" : "cover", alt = own(other);
    return { url: alt, from: alt ? other : "" };
  }
  // artFor's art as an image: wide art made from a cover is the whole cover
  // over a blur of itself; a cover cut from wide art keeps its focal point.
  const artImg = (a, kind, cls = "") => (kind === "hero" && a.from === "cover"
    ? `<span class="ln-art-wide${cls ? " " + cls : ""}" style="--art:url('${a.url}')"><img class="ln-art-wide__img" src="${a.url}" alt=""></span>`
    : `<img class="${[cls, kind === "cover" && a.from === "hero" ? "ln-art-crop" : ""].filter(Boolean).join(" ")}" src="${a.url}" alt="">`);
  // A cover set in type: the game's logo, else its name, with the store
  // small at the bottom (o.store false leaves it to the caption below), on
  // a quiet plate washed in its icon.
  function typeCover(it, cls, o = {}) {
    const logo = artFor(it, "logo").url, ic = !logo && S.icons[coverKey(it)];
    const name = logo ? `<img class="ln-tile__logo" src="${logo}" alt="">` : `${o.icon && ic ? `<img src="${ic}" alt="">` : ""}<b>${esc(it.name)}</b>`;
    return plate(ic, "ln-tile__type" + (cls ? " " + cls : ""), `${name}${o.store === false ? "" : `<span>${esc(sourceLabel(it.source, it.f))}</span>`}`, o.attr);
  }
  function coverHTML(it) {
    const key = coverKey(it), a = artFor(it, "cover");
    if (a.url && a.url !== S.covers[key]) return artImg(a, "cover", "ready");
    const c = S.covers[key], logo = !c && artFor(it, "logo").url;
    const ic = !c && !logo && S.icons[key];
    const can = it.appId || cfg().settings.downloadCovers || it.exe;
    const show = can && (c !== "" || ic !== "" && it.exe);
    const src = c ? ` src="${c}" class="ready"` : ic ? ` src="${ic}" class="icon ready"` : "";
    const img = show ? `<img alt="" data-cover="${esc(key)}" data-app="${it.appId || 0}" data-name="${esc(it.name)}" data-exe="${esc(it.exe || "")}"${src}>` : "";
    return typeCover(it, "nocover", { attr: c ? " hidden" : "", store: false }) + img;
  }

  // A Library cover (system.css .ln-tile). Who has a profile is the colour
  // code along the bottom; the tall bar, and the line under the title, say
  // whose files are loaded; in the grid sorted by when or how much, it says that.
  function cardHTML(it, bySort) {
    const g = it.g;
    const owner = g && S.d.active[g.id];
    const s = g && session(g.id);
    const owners = g ? cfg().players.filter((p) => hasProfile(g.id, p.id)) : [];
    const bars = owners.length ? `<div class="ln-tile__owners" aria-hidden="true">${owners.map((p) => `<span class="ln-player" style="--player:${esc(p.color)}"${p.id === owner ? " data-loaded" : ""}></span>`).join("")}</div>` : "";
    const badge = s ? `<span class="ln-tile__badge ln-tag ln-tag--player ln-player" style="--player:${esc(color(s.playerId))}">${esc(sessionText(s))}</span>` : "";
    const sortCap = !bySort ? "" : S.sort === "recent" ? (playedAt(it) ? cap(ago(playedAt(it))) : "No date yet")
      : S.sort === "played" ? (timeOf(it) ? hours(timeOf(it)) : "No time yet")
      : S.sort === "size" ? (installOf(it) ? about(installOf(it)) : S.sizes ? "Size not counted yet" : "Counting") : "";
    const meta = sortCap ? `<span class="ln-num">${esc(sortCap)}</span>`
      : owner && player(owner) ? `<span class="ln-mark ln-player" style="--player:${esc(color(owner))}"></span>${esc(possessive(pname(owner)))} files`
      : g ? "Profiles on" : it.kind === "owned" ? `${esc(sourceLabel(it.source, it.f))}, not installed` : storeMarks(it);
    const attr = itemAttr(it), k = recentKey(it);
    return `<div class="card lib-card${s ? " busy" : ""}${it.kind === "owned" ? " is-owned" : ""}${bySort && S.libSel.has(k) ? " is-sel" : ""}"${bySort ? ` data-k="${esc(k)}"` : ""}>
      <button class="ln-tile" data-action="open-sheet" ${attr} aria-label="${esc(it.name)}${sortCap ? `, ${esc(sortCap)}` : owner && player(owner) ? `, ${esc(possessive(pname(owner)))} files loaded` : ""}"><div class="ln-tile__cover cover">${coverHTML(it)}${bars}${badge}${bySort ? `<span class="lib-check" aria-hidden="true">${GI.tick}</span>` : ""}</div>
        <span class="ln-tile__title">${esc(it.name)}</span><span class="ln-tile__meta">${meta}</span>${S.sort === "rating" && ratingOf(it) ? `<span class="ln-tile__meta">${starsHTML(ratingOf(it))}</span>` : ""}</button>
      ${it.kind === "owned" ? installButton(it, "ln-button--secondary ln-button--icon ln-button--sm lib-play", "") : s ? "" : `<button class="ln-button ln-button--secondary ln-button--icon ln-button--sm lib-play" data-action="${it.kind === "game" ? "play" : "play-found"}" ${attr} aria-label="Play ${esc(it.name)}" title="Play">${GI.play}</button>`}</div>`;
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
    // No cover (yet): its wide art or logo can stand in, and the game's own
    // icon, read from its exe on this PC.
    if (!S.covers[key]) loadArt({ appId: +img.dataset.app, name: img.dataset.name });
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
      } else if (ic && !(tile && tile.querySelector(".ln-tile__logo"))) {
        el.classList.add("icon");
        if (tile) { tile.classList.add("with-icon"); tile.style.setProperty("--icon", `url("${ic}")`); }
        el.onload = () => el.classList.add("ready");
        el.src = ic;
      } else el.remove();
    });
    if (!bdShown) backdropForView();
  }

