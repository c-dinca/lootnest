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
    const ck = coverKey(it), c = artFor(it, "cover").url, ic = !c && S.icons[ck];
    const s = it.g && session(it.g.id);
    const owner = it.g && S.d.active[it.g.id];
    const thumb = c ? `<img class="ready" src="${c}" alt="">` : ic ? `<img class="ready icon" src="${ic}" alt="">` : `<b>${esc(initial(it.name))}</b>`;
    const sub = s ? `<small>${esc(sessionText(s))}</small>` : "";
    const mark = s ? '<span class="sg-live" aria-hidden="true"></span>' : owner && player(owner) ? `<span class="sg-mark ln-player" style="--player:${esc(color(owner))}" title="${esc(possessive(pname(owner)))} files are loaded"><i></i></span>` : "";
    return `<button class="sg-row${s ? " playing" : ""}" data-action="open-sheet" ${itemAttr(it)} data-ck="${esc(ck)}"${cur ? ' aria-current="true"' : ""} title="${esc(it.name)}">
      <span class="sg-thumb">${thumb}</span><span class="sg-name">${esc(it.name)}${sub}</span>${mark}</button>`;
  }
  function renderSideGames() {
    const box = $("#side-games");
    if (!box) return;
    const all = libraryItems();
    const cur = S.view === "game" && S.page ? S.page : null;
    const isCur = (it) => cur && cur.kind === it.kind && copiesOf(it).some((c) => c.id === cur.id);
    const group = (id, title, list, opts = {}) => {
      if (!list.length) return "";
      const closed = !opts.open && S.sideClosed.has(id);
      return `<div class="sg-group${closed ? " closed" : ""}" data-g="${esc(id)}"><button class="sg-head" data-action="side-group" data-g="${esc(id)}" aria-expanded="${!closed}">
        <svg viewBox="0 0 12 12" aria-hidden="true"><path d="M3 4.5 6 7.5l3-3"/></svg><span>${esc(title)}</span><span class="sg-n">${list.length}</span></button>
        <div class="sg-rows">${list.map((it) => sideRow(it, isCur(it))).join("")}</div></div>`;
    };
    let html;
    const q = S.sideQ.trim();
    if (q) {
      const m = nameMatcher(q), hits = sideSorted(all.filter((it) => m(it.name)));
      html = hits.length ? group("matches", "Matches", hits, { open: true }) : `<p class="sg-empty">No game called that.</p>`;
    } else if (!all.length) {
      html = `<p class="sg-empty">${S.found.scanning ? "Looking for your games" : "No games yet."}</p>`;
    } else {
      const playing = all.filter((it) => it.g && session(it.g.id));
      const recent = recentItems(all, false).filter((it) => !playing.includes(it)).slice(0, 5);
      const tags = {};
      all.forEach((it) => tagsOf(it).forEach((t) => (tags[t] = tags[t] || []).push(it)));
      html = group("playing", "Playing now", playing, { open: true }) + group("recent", "Recent", recent)
        + Object.keys(tags).sort((a, b) => a.localeCompare(b)).map((t) => group("tag:" + t, t, sideSorted(tags[t]))).join("")
        + group("all", "All games", sideSorted(all));
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
  // Numbers count up and rings and bars fill the first time a page opens
  // in a session, for each seat.
  const arrived = new Set();
  function arriveOnce(root) {
    const key = `${S.view}|${S.view === "player" ? S.playerPage : ""}|${S.seat}`;
    if (reduced() || arrived.has(key) || !root.querySelector("[data-countup], .ach-ring, .ach-bar")) return;
    arrived.add(key);
    countUp(root);
    const page = root.firstElementChild;
    if (page) { page.classList.add("arrive"); setTimeout(() => page.classList.remove("arrive"), 1400); }
  }

