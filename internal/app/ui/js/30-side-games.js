  // ---------------------------------------------------------- side games --
  // Every game in the sidebar, as in Steam's library: what's being played,
  // the last few played, each collection, then all of them. The filter
  // narrows it to the names that match.
  const SIDE_SORTS = { name: "Sorted by name", recent: "Sorted by last played", played: "Sorted by time played" };
  function sideSorted(list) {
    const out = list.slice();
    if (S.sideSort === "recent") out.sort((a, b) => playedAt(b) - playedAt(a) || a.name.localeCompare(b.name));
    else if (S.sideSort === "played") out.sort((a, b) => timeOf(b) - timeOf(a) || a.name.localeCompare(b.name));
    return out;
  }
  function sideRow(it, cur) {
    const ck = coverKey(it), c = artFor(it, "cover").url, ic = !c && S.icons[ck], k = recentKey(it);
    const s = it.g && session(it.g.id);
    const owner = it.g && S.d.active[it.g.id];
    const thumb = c ? `<img class="ready" src="${c}" alt="">` : ic ? `<img class="ready icon" src="${ic}" alt="">` : `<b>${esc(initial(it.name))}</b>`;
    const sub = s ? `<small>${esc(sessionText(s))}</small>` : "";
    const mark = s ? '<span class="sg-live" aria-hidden="true"></span>' : owner && player(owner) ? `<span class="sg-mark ln-player" style="--player:${esc(color(owner))}" title="${esc(possessive(pname(owner)))} files are loaded"><i></i></span>` : "";
    const cls = (s ? " playing" : "") + (it.kind === "owned" ? " is-owned" : "") + (S.sideSel.has(k) ? " sel" : "");
    return `<button class="sg-row${cls}" data-action="open-sheet" ${itemAttr(it)} data-ck="${esc(ck)}" data-k="${esc(k)}"${cur ? ' aria-current="true"' : ""} title="${esc(it.name)}">
      <span class="sg-thumb">${thumb}</span><span class="sg-name">${esc(it.name)}${sub}</span>${mark}</button>`;
  }
  // The games the sidebar's menu shows: All (installed or not), Installed,
  // Favourites, Not installed, or those a profile has played or has files for.
  function sideBase() {
    const v = S.sideShow, lib = libraryItems();
    if ((v === "all" || v === "owned" || v === "fav") && S.owned === null && !S.found.scanning) { S.owned = []; api.ownedGames().catch(() => {}); }
    const owned = S.owned ? ownedItems() : [];
    const list = v === "installed" ? lib.slice() : v === "owned" ? owned
      : v === "fav" ? lib.concat(owned).filter((it) => tagsOf(it).includes(FAV))
      : v.startsWith("p:") ? lib.filter((it) => playedOrOwned(it, v.slice(2))) : lib.concat(owned);
    return list.sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: "base" }));
  }
  function sideShowMenu() {
    const sel = $("#side-show");
    if (!sel) return;
    const ps = cfg().players;
    if (S.sideShow.startsWith("p:") && !player(S.sideShow.slice(2))) S.sideShow = "all";
    const opts = `<option value="all">All games</option><option value="installed">Installed</option><option value="fav">Favourites</option><option value="owned">Not installed</option>`
      + (ps.length ? `<optgroup label="Played by">${ps.map((p) => `<option value="p:${esc(p.id)}">${esc(p.name)}</option>`).join("")}</optgroup>` : "");
    if (sel.dataset.opts !== opts) { sel.innerHTML = opts; sel.dataset.opts = opts; }
    sel.value = S.sideShow;
  }
  function renderSideGames() {
    const box = $("#side-games");
    if (!box) return;
    sideShowMenu();
    const a = document.activeElement;
    if (a && a.matches("#side-games .sg-input") && (S.sideNew || S.sideRename)) return; // a name being typed stays
    const all = sideBase();
    const cur = S.view === "game" && S.page ? S.page : null;
    const isCur = (it) => cur && cur.kind === it.kind && copiesOf(it).some((c) => c.id === cur.id);
    const rows = (list) => list.map((it) => sideRow(it, isCur(it))).join("");
    const chev = '<svg viewBox="0 0 12 12" aria-hidden="true"><path d="M3 4.5 6 7.5l3-3"/></svg>';
    const group = (id, title, list, opts = {}) => {
      if (!list.length) return "";
      const closed = !opts.open && S.sideClosed.has(id);
      return `<div class="sg-group${closed ? " closed" : ""}" data-g="${esc(id)}"><button class="sg-head" data-action="side-group" data-g="${esc(id)}" aria-expanded="${!closed}">
        ${chev}<span>${esc(title)}</span><span class="sg-n">${list.length}</span></button>
        <div class="sg-rows">${rows(list)}</div></div>`;
    };
    // A collection: a folder to open, drop games on and drag into order.
    const folder = (t, list) => {
      const id = "tag:" + t, closed = S.sideClosed.has(id), fav = t === FAV;
      const head = S.sideRename === t
        ? `<input id="sg-rename" class="sg-input" type="text" maxlength="40" autocomplete="off" spellcheck="false" value="${esc(t)}" data-tag="${esc(t)}" aria-label="A new name for ${esc(t)}">`
        : `<button class="sg-head" data-action="side-group" data-g="${esc(id)}" aria-expanded="${!closed}">${chev}<span>${esc(t)}</span><span class="sg-n">${list.length}</span></button>`
          + (fav ? "" : `<button class="sg-more" data-action="tag-menu" data-tag="${esc(t)}" aria-haspopup="menu" aria-expanded="false" aria-label="${esc(t)}: rename, move or delete" title="More">${GI.more}</button>`);
      return `<div class="sg-group sg-folder${fav ? " is-fav" : ""}${list.length ? "" : " is-empty"}${closed ? " closed" : ""}" data-g="${esc(id)}" data-tag="${esc(t)}">
        <div class="sg-headrow">${head}</div><div class="sg-rows">${list.length ? rows(list) : '<p class="sg-hint">Drag games here.</p>'}</div></div>`;
    };
    let html;
    const q = S.sideQ.trim();
    if (q) {
      const m = nameMatcher(q), hits = sideSorted(all.filter((it) => m(it.name)));
      html = hits.length ? group("matches", "Matches", hits, { open: true }) : `<p class="sg-empty">No game called that.</p>`;
    } else if (!all.length && S.sideShow !== "all" && S.sideShow !== "installed") {
      html = `<p class="sg-empty">${{ fav: "No favourites yet. Right-click a game to add it.", owned: "No games that aren't installed." }[S.sideShow] || `${esc(pname(S.sideShow.slice(2)))} hasn't played a game here yet.`}</p>`;
    } else if (!all.length) {
      html = `<p class="sg-empty">${S.found.scanning ? "Looking for your games" : "No games yet."}</p>`;
    } else {
      const playing = all.filter((it) => it.g && session(it.g.id));
      const recent = recentItems(all, false).filter((it) => !playing.includes(it)).slice(0, 5);
      const tags = {}, used = new Set(Object.values(cfg().tags || {}).flat());
      all.forEach((it) => tagsOf(it).forEach((t) => (tags[t] = tags[t] || []).push(it)));
      const add = S.sideNew
        ? `<div class="sg-newrow is-open"><input id="sg-new" class="sg-input" type="text" maxlength="40" autocomplete="off" spellcheck="false" placeholder="Name the collection" aria-label="The new collection's name"></div>`
        : `<div class="sg-newrow"><button class="sg-new" data-action="tag-new">${GI.plus}<span>New collection</span></button></div>`;
      html = group("playing", "Playing now", playing, { open: true }) + group("recent", "Recent", recent)
        + `<div class="sg-folders">${allTags().filter((t) => tags[t] || S.sideShow === "all" || !used.has(t)).map((t) => folder(t, sideSorted(tags[t] || []))).join("")}${add}</div>`
        + group("loose", "Not in a collection", sideSorted(all.filter((it) => !tagsOf(it).length)))
        + group("all", "All", sideSorted(all));
    }
    if (box.dataset.html === html) return; // nothing changed: keep it as it is
    const top = box.scrollTop;
    box.innerHTML = html;
    box.dataset.html = html;
    box.scrollTop = top;
    const sb = $("#side-sort");
    if (sb) { sb.title = SIDE_SORTS[S.sideSort]; sb.setAttribute("aria-label", SIDE_SORTS[S.sideSort] + ". Change it"); }
    watchSideThumbs();
  }

  // Collections from the sidebar: one change, saved once, then drawn again.
  async function tagOp(o) {
    try { await api.tagOp(o); } catch (e) { toast(errText(e), "error"); return false; }
    await refresh();
    return true;
  }
  // A game shown once is in a collection with every copy of it.
  const sideItem = (k) => libraryItems().find((it) => recentKey(it) === k) || ownedItem(k);
  const tagKeys = (keys) => keys.map(sideItem).filter(Boolean).flatMap((it) => copiesOf(it).map(recentKey));
  const gamesText = (keys) => (keys.length === 1 ? (sideItem(keys[0]) || {}).name || "1 game" : `${keys.length} games`);
  async function addToTag(keys, tag, from) {
    if (!(await tagOp({ op: "add", tag, keys: tagKeys(keys), from: from || "" }))) return;
    uiSound("move");
    toast(`${from ? "Moved" : "Added"} ${gamesText(keys)} to ${tag}.`);
  }
  async function removeFromTag(keys, tag) {
    const all = tagKeys(keys), back = all.filter((k) => ((cfg().tags || {})[k] || []).includes(tag));
    if (!(await tagOp({ op: "remove", tag, keys: all }))) return;
    S.sideSel.clear();
    renderSideGames();
    toast(`Took ${gamesText(keys)} out of ${tag}. ${keys.length === 1 ? "It stays" : "They stay"} in your Library.`, "", { label: "Undo", run: () => tagOp({ op: "add", tag, keys: back }) });
  }
  function startNewTag(keys) {
    if (S.folded) setFold(false);
    S.sideRename = "";
    S.sideNew = { keys };
    renderSideGames();
    const i = $("#sg-new");
    if (i) { i.focus(); i.scrollIntoView({ block: "nearest" }); }
  }
  async function saveNewTag(i) {
    const name = i.value.trim(), keys = (S.sideNew || {}).keys || [];
    if (!name) { i.focus(); return; }
    try { await api.tagOp({ op: "add", tag: name, keys: tagKeys(keys) }); } catch (e) { toast(errText(e), "error"); i.focus(); return; }
    S.sideNew = null;
    await refresh();
    if (keys.length) uiSound("move");
  }
  async function saveRename(i) {
    const from = i.dataset.tag, to = i.value.trim();
    S.sideRename = "";
    if (!to || to === from) { renderSideGames(); return; }
    const onto = allTags().find((t) => t !== from && t.toLowerCase() === to.toLowerCase());
    if (onto && !(await confirmDialog({ title: `Put ${from} into ${onto}?`, body: `${onto} already exists. ${possessive(from)} games join it, and ${from} goes away. No game leaves your Library.`, ok: "Put them together" }))) { renderSideGames(); return; }
    if (!(await tagOp({ op: "rename", tag: from, to }))) return;
    const now = onto || to;
    if (S.tag === from) S.tag = now;
    if (BP.tag === from) BP.tag = now;
    if (S.sideClosed.delete("tag:" + from)) { S.sideClosed.add("tag:" + now); setPref("sideClosed", JSON.stringify([...S.sideClosed])); }
  }
  function tagMenuHTML(t) {
    const order = allTags().slice(1), i = order.indexOf(t), a = `data-tag="${esc(t)}"`;
    return `<div class="ctx-title">${esc(t)}</div><button data-action="tag-rename" ${a}>Rename</button>
      <button data-action="tag-move" ${a} data-index="${i - 1}"${i <= 0 ? " disabled" : ""}>Move up</button>
      <button data-action="tag-move" ${a} data-index="${i + 1}"${i >= order.length - 1 ? " disabled" : ""}>Move down</button><hr>
      <button class="danger" data-action="tag-delete" ${a}>Delete collection</button>`;
  }
  // Add to collection, from a game's right-click: every collection, ticked
  // when the games are in it, and a new one.
  function tagPickHTML() {
    const items = S.tagPick.map(sideItem).filter(Boolean);
    const rows = allTags().map((t) => {
      const on = items.length > 0 && items.every((it) => tagsOf(it).includes(t));
      return `<button role="menuitemcheckbox" aria-checked="${on}" data-action="tag-toggle" data-tag="${esc(t)}" data-on="${on ? "1" : ""}"><span class="grow">${esc(t)}</span>${on ? GI.tick : ""}</button>`;
    }).join("");
    return `<div class="ctx-title">Collections for ${esc(gamesText(S.tagPick))}</div>${rows}<hr><button data-action="tag-new-for">${GI.plus}New collection</button>`;
  }
  // Right-click on a game: for the selection when the game is in it.
  function collectionItems(it, host) {
    const k = recentKey(it), row = host && host.closest("#side-games .sg-row");
    S.tagPick = row && S.sideSel.has(k) ? [...S.sideSel] : [k];
    const f = row && row.closest(".sg-folder");
    return [`<button data-action="tag-pick" aria-haspopup="menu"><span class="grow">Add to collection</span>${GI.next}</button>`,
      f ? `<button data-action="tag-out" data-tag="${esc(f.dataset.tag)}">Remove from ${esc(f.dataset.tag)}</button>` : ""];
  }
  // Delete on rows inside a collection takes them out of it.
  function sideRemoveKey(ev) {
    const row = ev.target.closest && ev.target.closest("#side-games .sg-folder .sg-row");
    if (!row || ev.ctrlKey || ev.altKey) return false;
    ev.preventDefault();
    removeFromTag(S.sideSel.has(row.dataset.k) ? [...S.sideSel] : [row.dataset.k], row.closest(".sg-folder").dataset.tag);
    return true;
  }
  function paintSel() { $("#side-games").querySelectorAll(".sg-row").forEach((r) => r.classList.toggle("sel", S.sideSel.has(r.dataset.k))); }

  // Selecting with Ctrl and Shift, and dragging with the pointer: games onto
  // a collection (Shift moves them out of the one they came from), onto New
  // collection, and collections into an order. Esc cancels.
  let drag = null, swallow = false;
  const sideBox = () => $("#side-games");
  function dragStart() {
    const d = drag;
    d.on = true;
    if (d.row) {
      const k = d.row.dataset.k;
      d.keys = S.sideSel.has(k) ? [...S.sideSel] : [k];
      d.from = (d.row.closest(".sg-folder") || { dataset: {} }).dataset.tag || "";
    } else d.tag = d.head.closest(".sg-folder").dataset.tag;
    d.ghost = document.createElement("div");
    d.ghost.className = "sg-ghost";
    d.ghost.textContent = d.keys ? gamesText(d.keys) : d.tag;
    document.body.appendChild(d.ghost);
    sideBox().classList.add(d.keys ? "drag-games" : "drag-tags");
  }
  function dragTarget(ev) {
    const d = drag, el = document.elementFromPoint(ev.clientX, ev.clientY);
    sideBox().querySelectorAll(".drop, .ins-before, .ins-after").forEach((x) => x.classList.remove("drop", "ins-before", "ins-after"));
    d.target = null;
    if (d.keys) {
      const t = el && el.closest("#side-games .sg-folder, #side-games .sg-newrow");
      if (t && !(t.dataset.tag && t.dataset.tag === d.from)) { t.classList.add("drop"); d.target = t; }
      return;
    }
    const t = el && el.closest("#side-games .sg-folder");
    if (!t || t.dataset.tag === d.tag) return;
    const h = t.querySelector(".sg-headrow").getBoundingClientRect();
    d.after = t.classList.contains("is-fav") || ev.clientY > h.top + h.height / 2;
    t.classList.add(d.after ? "ins-after" : "ins-before");
    d.target = t;
  }
  function dragStop() {
    if (!drag) return;
    if (drag.ghost) drag.ghost.remove();
    sideBox().classList.remove("drag-games", "drag-tags");
    sideBox().querySelectorAll(".drop, .ins-before, .ins-after").forEach((x) => x.classList.remove("drop", "ins-before", "ins-after"));
    swallow = drag.on || !!drag.cancel;
    drag = null;
  }
  function dragDrop(ev) {
    const d = drag, t = d.on && !d.cancel && d.target;
    dragStop();
    if (!t) return;
    if (d.keys && t.classList.contains("sg-newrow")) { startNewTag(d.keys); return; }
    if (d.keys) { addToTag(d.keys, t.dataset.tag, ev.shiftKey && d.from && d.from !== t.dataset.tag ? d.from : ""); return; }
    const rest = allTags().slice(1).filter((x) => x !== d.tag);
    const i = t.classList.contains("is-fav") ? 0 : rest.indexOf(t.dataset.tag) + (d.after ? 1 : 0);
    tagOp({ op: "move", tag: d.tag, index: i }).then((done) => { if (done) uiSound("move"); });
  }
  (function sideDrag() {
    const b = sideBox();
    if (!b) return;
    b.addEventListener("dragstart", (ev) => ev.preventDefault()); // covers aren't dragged out of the window
    b.addEventListener("pointerdown", (ev) => {
      swallow = false;
      if (ev.button !== 0 || S.folded || ev.target.closest("input")) return;
      const row = ev.target.closest(".sg-row"), head = !row && ev.target.closest(".sg-folder:not(.is-fav) .sg-head");
      if (row || head) drag = { x: ev.clientX, y: ev.clientY, row, head, id: ev.pointerId, on: false };
    });
    document.addEventListener("pointermove", (ev) => {
      if (!drag || ev.pointerId !== drag.id || drag.cancel) return;
      if (!drag.on) { if (Math.hypot(ev.clientX - drag.x, ev.clientY - drag.y) < 6) return; dragStart(); }
      drag.ghost.style.transform = `translate(${ev.clientX + 14}px, ${ev.clientY + 10}px)`;
      const r = b.getBoundingClientRect();
      if (ev.clientY < r.top + 32) b.scrollTop -= 12; else if (ev.clientY > r.bottom - 32) b.scrollTop += 12;
      dragTarget(ev);
    });
    document.addEventListener("pointerup", (ev) => { if (drag && ev.pointerId === drag.id) dragDrop(ev); });
    document.addEventListener("pointercancel", () => dragStop());
    document.addEventListener("keydown", (ev) => {
      if (ev.key !== "Escape" || !drag || !drag.on) return;
      ev.preventDefault(); ev.stopImmediatePropagation();
      const id = drag.id; dragStop(); drag = { id, on: false, cancel: true };
    }, true);
    // After a drag, the click it ends with opens nothing; Ctrl and Shift
    // select instead of opening.
    b.addEventListener("click", (ev) => {
      if (swallow) { swallow = false; ev.preventDefault(); ev.stopPropagation(); return; }
      const row = ev.target.closest(".sg-row");
      if (!row) return;
      const k = row.dataset.k;
      if (!ev.ctrlKey && !ev.metaKey && !ev.shiftKey) { if (S.sideSel.size) { S.sideSel.clear(); paintSel(); } S.sideAnchor = k; return; }
      ev.preventDefault(); ev.stopPropagation();
      const rs = [...b.querySelectorAll(".sg-row")].filter((r) => r.offsetParent), j = rs.indexOf(row);
      const anchors = rs.map((r, i) => (r.dataset.k === S.sideAnchor ? i : -1)).filter((i) => i >= 0);
      if (ev.shiftKey && anchors.length) {
        const a = anchors.reduce((x, y) => (Math.abs(y - j) < Math.abs(x - j) ? y : x));
        rs.slice(Math.min(a, j), Math.max(a, j) + 1).forEach((r) => S.sideSel.add(r.dataset.k));
      } else {
        if (S.sideSel.has(k)) S.sideSel.delete(k); else S.sideSel.add(k);
        S.sideAnchor = k;
      }
      paintSel();
    }, true);
    b.addEventListener("keydown", (ev) => {
      const i = ev.target;
      if (i.id === "sg-new") {
        if (ev.key === "Enter") { ev.preventDefault(); saveNewTag(i); }
        else if (ev.key === "Escape") { ev.preventDefault(); ev.stopPropagation(); S.sideNew = null; renderSideGames(); const n = $(".sg-new"); if (n) n.focus(); }
      } else if (i.id === "sg-rename") {
        if (ev.key === "Enter") { ev.preventDefault(); saveRename(i); }
        else if (ev.key === "Escape") { ev.preventDefault(); ev.stopPropagation(); S.sideRename = ""; renderSideGames(); }
      }
    });
    b.addEventListener("focusout", (ev) => {
      const i = ev.target;
      if (i.id === "sg-new" && !i.value.trim()) { S.sideNew = null; setTimeout(renderSideGames); }
      else if (i.id === "sg-rename" && S.sideRename) { S.sideRename = ""; setTimeout(renderSideGames); }
    });
    $("#side-show").addEventListener("change", (ev) => { S.sideShow = ev.target.value; setPref("sideShow", S.sideShow); renderSideGames(); });
  })();
  // Covers for the sidebar's list, as rows come into sight; only the
  // picture changes when one arrives.
  let sideObserver = null;
  function watchSideThumbs() {
    const box = $("#side-games");
    if (!sideObserver) {
      sideObserver = new IntersectionObserver((entries) => entries.forEach((e) => {
        if (!e.isIntersecting) return;
        sideObserver.unobserve(e.target);
        const it = itemFor(e.target);
        if (it) sideThumb(it);
      }), { root: box, rootMargin: "200px" });
    }
    box.querySelectorAll(".sg-row").forEach((r) => { if (!r.querySelector("img")) sideObserver.observe(r); });
  }
  async function sideThumb(it) {
    const key = coverKey(it);
    if (S.covers[key] || S.icons[key]) return fillThumbs(key);
    if (coverAsked.has(key) || key in S.covers) return;
    coverAsked.add(key);
    const can = it.appId || cfg().settings.downloadCovers;
    let url = "";
    if (can) { try { url = (await api.cover(it.appId || 0, it.name, key)) || ""; } catch (e) { /* none */ } }
    if (url) S.covers[key] = url;
    else if (it.exe && !(key in S.icons)) { try { S.icons[key] = (await api.gameIcon(it.exe)) || ""; } catch (e) { S.icons[key] = ""; } }
    fillThumbs(key);
  }
  function fillThumbs(key) {
    const c = S.covers[key], ic = !c && S.icons[key];
    if (!c && !ic) return;
    document.querySelectorAll(`#side-games .sg-row[data-ck="${CSS.escape(key)}"] .sg-thumb`).forEach((t) => {
      t.innerHTML = `<img class="${ic ? "icon" : ""}" src="${c || ic}" alt="">`;
      const img = t.firstChild;
      if (img.complete) img.classList.add("ready"); else img.onload = () => img.classList.add("ready");
    });
    const box = $("#side-games");
    if (box) box.dataset.html = ""; // the next render redraws with the picture
  }
  // Folding the sidebar to a rail, and dragging its edge.
  function applyFold(animate) {
    const app = $("#app");
    if (!animate) app.classList.add("no-anim");
    app.classList.toggle("side-folded", S.folded);
    const b = $(".side-collapse");
    if (b) { const t = S.folded ? "Open the sidebar (Ctrl+B)" : "Fold the sidebar (Ctrl+B)"; b.title = t; b.setAttribute("aria-label", t); }
    if (!animate) requestAnimationFrame(() => app.classList.remove("no-anim"));
  }
  function setFold(on) { S.folded = on; setPref("folded", on ? "1" : ""); applyFold(true); }
  function applySideWidth() {
    const w = parseInt(pref("sideW", ""), 10);
    $("#app").style.setProperty("--side-w", w >= 200 && w <= 440 ? w + "px" : "");
  }
  (function sideResize() {
    const h = $("#side-resize");
    if (!h) return;
    h.addEventListener("pointerdown", (ev) => {
      if (S.folded) return;
      ev.preventDefault();
      h.setPointerCapture(ev.pointerId);
      const app = $("#app");
      app.classList.add("resizing");
      const move = (e) => { const w = Math.max(200, Math.min(440, Math.round(e.clientX))); app.style.setProperty("--side-w", w + "px"); };
      const up = (e) => {
        h.removeEventListener("pointermove", move); h.removeEventListener("pointerup", up);
        app.classList.remove("resizing");
        setPref("sideW", String(Math.max(200, Math.min(440, Math.round(e.clientX)))));
      };
      h.addEventListener("pointermove", move); h.addEventListener("pointerup", up);
    });
    h.addEventListener("dblclick", () => { setPref("sideW", ""); applySideWidth(); });
  })();
  // Numbers count up and rings and bars fill each time a page arrives,
  // once: or as soon as they're there, when they come in after the page.
  let arrivedAt = -1;
  function arriveOnce(root) {
    if (reduced() || arrivedAt === arrival || !root.querySelector("[data-countup], .ach-ring, .ach-bar")) return;
    arrivedAt = arrival;
    countUp(root);
    const page = root.firstElementChild;
    if (page) { page.classList.add("arrive"); setTimeout(() => page.classList.remove("arrive"), 1400); }
  }
