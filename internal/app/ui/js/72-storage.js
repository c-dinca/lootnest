  // --------------------------------------------------------------- storage --
  // Storage, a Settings category: the drives, then what each game takes
  // (ps_sizes, counted in the background), with a way to move a folder
  // game to another drive. Steam games move with Game Mover; other stores'
  // games with their own launchers.
  let sizesAsked = false;
  function askSizes() {
    if (sizesAsked) return;
    sizesAsked = true;
    api.sizes().then((r) => { S.sizes = r; if (S.view === "settings" || S.view === "library" || S.view === "game") render(); }).catch(() => { sizesAsked = false; });
  }
  // A game's sizes, or null while they aren't counted.
  let sizesBy = null, sizesFrom = null;
  function sizeOf(it) {
    askSizes();
    if (!S.sizes) return null;
    if (sizesFrom !== S.sizes) { sizesFrom = S.sizes; sizesBy = new Map((S.sizes.games || []).map((g) => [g.key, g])); }
    return sizesBy.get(recentKey(it)) || null;
  }
  // What a Library entry's install takes, every copy of it together.
  const installOf = (it) => copiesOf(it).reduce((n, c) => n + ((sizeOf(c) || {}).install || 0), 0);
  const about = (n) => "About " + fmtBytes(n);
  // Where a game is installed, and the drive it's on with that drive's free
  // space, or null while its sizes aren't counted.
  function installWhere(it) {
    const z = sizeOf(it), dir = (z && z.dir) || (it.f && it.f.dir) || "";
    if (!dir) return null;
    const up = dir.toUpperCase();
    const d = ((S.sizes || {}).drives || []).filter((x) => x.root && up.startsWith(x.root.toUpperCase())).sort((a, b) => b.root.length - a.root.length)[0];
    return { dir, drive: d ? d.root.replace(/[\\/]+$/, "") || d.root : "", free: d ? d.free : -1, size: d ? d.size : 0 };
  }
  const NEAR_FULL = 0.1; // a drive with less than this share free
  const MOVE_ROOM = 1.1; // a move needs the game's size and a tenth more
  // A game in a folder of its own that Lootnest moves itself. Hydra can't
  // move its games, so Lootnest does, and the user points Hydra at it.
  const movable = (it) => it.source !== "shortcut" && (it.kind === "found" ? removable(it) : it.g.launch.type === "exe" && UPDATABLE.includes(it.source));
  const moveItem = (it) => movable(it) ? `<button data-action="movefolder-open" ${itemAttr(it)}>Move to another drive</button>`
    : it.appId ? `<button data-action="move-open" data-app="${it.appId}" data-name="${esc(it.name)}">Move with Game Mover</button>`
    : LAUNCHERS.includes(it.source) ? `<button disabled>Installed by ${esc(sourceLabel(it.source, it.f))}, so move it there</button>` : "";

  // ------------------------------------------------- a move in progress --
  // Both kinds of move, a folder game's and a Steam game's through Game
  // Mover, draw the same way: how far, how fast, the time left, from where
  // to where. Minimise (or Escape) shrinks the dialog to a card in the
  // corner that keeps counting; a click on it brings the dialog back.
  const MV = { key: "", t0: 0, last: 0, at: 0, rate: 0 };
  // The speed, as an average that settles over about the last 20 updates.
  function moveTick(key, done) {
    const now = performance.now();
    if (MV.key !== key || done < MV.last) Object.assign(MV, { key, t0: now, last: done, at: now, rate: 0 });
    const dt = (now - MV.at) / 1000;
    if (dt < 0.2 || done <= MV.last) return;
    const r = (done - MV.last) / dt;
    MV.rate = MV.rate ? MV.rate * 0.9 + r * 0.1 : r;
    MV.last = done; MV.at = now;
  }
  const leftText = (s) => s < 60 ? "Less than a minute left" : s < 3600 ? `About ${Math.round(s / 60)} min left`
    : `About ${Math.floor(s / 3600)} h ${Math.round((s % 3600) / 60)} min left`;
  // What's moving now, whichever kind, or null.
  function moveNow() {
    const F = S.folderMove, G = S.moving;
    const m = F ? { kind: "folder", key: F.key, name: F.name || "The game", done: F.done || 0, total: F.total || 0, step: F.step || "", from: F.from || "", to: F.to || "" }
      : G ? { kind: "steam", key: "app:" + G.appId, appId: G.appId, name: G.name || "The game", done: G.done || 0, total: G.total || 0, step: G.step || "", from: G.from || "", to: G.lib || "" } : null;
    if (!m) return null;
    m.pct = m.total ? Math.min(100, Math.floor((m.done / m.total) * 100)) : 0;
    m.ending = m.step === "check" || m.step === "firewall";
    const steady = MV.key === m.key && MV.rate > 0 && performance.now() - MV.t0 > 3000;
    m.speed = steady && !m.ending ? fmtBytes(MV.rate) + "/s" : "";
    m.left = m.ending ? "Almost done" : m.step === "steam" ? "Closing Steam first" : !m.total ? "Counting its files" : steady ? leftText((m.total - m.done) / MV.rate) : "Working out the time left";
    m.say = m.step === "firewall" ? "Keeping it offline in its new folder. Allow it in Windows' prompt."
      : m.step === "check" ? "Checking the copy, then pointing the Library at it" : m.step === "steam" ? "Steam closes while the game moves" : "";
    return m;
  }
  // The dialog's view of a move. The numbers carry data-mv names so a
  // progress event changes only them (paintMove), and the bar glides.
  function moveBody(m) {
    const end = (k, p) => p ? `<div class="mv-end"><span class="mv-k">${k}</span><span class="mv-p selectable" title="${esc(p)}">${esc(p)}</span></div>` : "";
    return `<div class="mv${m.ending ? " is-ending" : ""}" data-mv="${esc(m.key)}">
      <div class="mv-head"><span class="mv-pct ln-num"><b data-mv="pct">${m.pct}</b><small>%</small></span>
        <span class="mv-size ln-num" data-mv="size">${m.total ? `${fmtBytes(m.done)} of ${fmtBytes(m.total)}` : ""}</span></div>
      <div class="mv-bar" role="progressbar" aria-label="Moved" aria-valuemin="0" aria-valuemax="100" aria-valuenow="${m.pct}"><i data-mv="fill" style="width:${m.pct}%"></i></div>
      <div class="mv-facts"><span data-mv="left">${esc(m.left)}</span><span class="ln-num" data-mv="speed">${esc(m.speed)}</span></div>
      ${m.say ? `<p class="mv-say" role="status">${esc(m.say)}</p>` : ""}
      ${m.from || m.to ? `<div class="mv-route">${end("From", m.from)}<span class="mv-flow" aria-hidden="true"><i></i><i></i><i></i>${ph("arrow-right")}</span>${end("To", m.to)}</div>` : ""}
      <p class="hint">${m.kind === "steam" ? "Keep Steam closed until it's done." : "Don't start the game until it's done."}</p></div>`;
  }
  // The move's dialog is open.
  const moveShown = (m) => !!(S.dialog && ((S.dialog.kind === "movefolder" && S.dialog.key === m.key) || (S.dialog.kind === "move" && S.dialog.appId === m.appId)));
  // A progress event: the numbers in place when the dialog shows the move
  // (a redraw only when its stage changes), and the corner card.
  function paintMove() {
    const m = moveNow(), box = m && $(`.mv[data-mv="${CSS.escape(m.key)}"]`);
    if (m && moveShown(m)) {
      if (!box || box.classList.contains("is-ending") !== m.ending) renderOverlay();
      else {
        const set = (n, v) => { const e = box.querySelector(`[data-mv="${n}"]`); if (e && e.textContent !== v) e.textContent = v; };
        set("pct", String(m.pct)); set("size", m.total ? `${fmtBytes(m.done)} of ${fmtBytes(m.total)}` : ""); set("left", m.left); set("speed", m.speed);
        const f = box.querySelector('[data-mv="fill"]'); if (f) f.style.width = m.pct + "%";
        const b = box.querySelector(".mv-bar"); if (b) b.setAttribute("aria-valuenow", m.pct);
      }
    }
    movePill();
  }
  // The corner card, while a move runs and its dialog isn't open.
  function movePill() {
    const m = moveNow(), show = !!m && !moveShown(m);
    let el = document.getElementById("mv-pill");
    if (!show) { if (el && !el.classList.contains("out")) { el.classList.add("out"); setTimeout(() => { if (el.classList.contains("out")) el.remove(); }, 260); } return; }
    if (!el || el.classList.contains("out")) {
      if (el) el.remove();
      el = document.createElement("button");
      el.id = "mv-pill"; el.className = "mv-pill ln-root"; el.type = "button"; el.dataset.action = "move-show";
      el.innerHTML = `<span class="mv-pill-ic">${GI.drive}</span><span class="mv-pill-txt"><b></b><span></span></span><span class="mv-pill-pct ln-num"></span><span class="mv-pill-bar"><i></i></span>`;
      document.body.appendChild(el);
    }
    el.querySelector("b").textContent = "Moving " + m.name;
    el.querySelector(".mv-pill-txt span").textContent = m.speed ? `${m.left}, ${m.speed}` : m.left;
    el.querySelector(".mv-pill-pct").textContent = m.pct + "%";
    el.querySelector(".mv-pill-bar i").style.width = m.pct + "%";
    el.setAttribute("aria-label", `Moving ${m.name}, ${m.pct}%. ${m.left}. Show it`);
  }

  const ST_COLS = [["name", "Game"], ["install", "Installed"], ["saves", "Saves"], ["profiles", "Profiles"], ["copies", "Lootnest's copies"]];
  function viewStorage() {
    askSizes();
    const Z = S.sizes;
    if (!Z) return lnInline("", "Counting what your games take");
    const drives = (Z.drives || []).map((d) => {
      const used = d.size > 0 ? Math.min(1, (d.size - d.free) / d.size) : 0, full = d.size > 0 && d.free / d.size < NEAR_FULL;
      return `<div class="bk-card"><div class="bk-row st-drive"><div><b>${esc(d.root.replace(/\\$/, ""))}</b><p>${d.free < 0 ? "Free space unknown" : `${fmtBytes(d.free)} free of ${fmtBytes(d.size)}`}</p></div>
        ${full ? `<span class="st-full">${GI.alert}Almost full</span>` : ""}
        <div class="meter st-meter" role="img" aria-label="${Math.round(used * 100)}% used"><i style="width:${(used * 100).toFixed(1)}%"></i></div></div></div>`;
    }).join("");
    const rows = everyItem().map((it) => ({ it, z: sizeOf(it) }));
    const k = S.stSort;
    rows.sort((a, b) => k === "name" ? byName(a.it.name, b.it.name) : ((b.z || {})[k] || 0) - ((a.z || {})[k] || 0) || byName(a.it.name, b.it.name));
    const cell = (z, f) => `<td class="ln-num">${z ? (z[f] ? fmtBytes(z[f]) : "None") : "Not counted yet"}</td>`;
    const head = ST_COLS.map(([f, l]) => `<th scope="col"${f === k ? ` aria-sort="${f === "name" ? "ascending" : "descending"}"` : ""}><button class="st-sort" data-action="st-sort" data-col="${f}">${l}</button></th>`).join("");
    const table = rows.length ? `<div class="st-wrap"><table class="st-table"><thead><tr>${head}<th scope="col" aria-label="Actions"></th></tr></thead><tbody>${rows.map(({ it, z }) => `
        <tr><th scope="row"><b>${esc(it.name)}</b><span>${esc(sourceLabel(it.source, it.f))}</span></th>${cell(z, "install")}${cell(z, "saves")}${cell(z, "profiles")}${cell(z, "copies")}
        <td><button class="ln-button ln-button--ghost ln-button--icon ln-button--sm" data-action="st-more" ${itemAttr(it)} aria-haspopup="menu" aria-expanded="false" aria-label="More for ${esc(it.name)}">${icon.dots}</button></td></tr>`).join("")}</tbody></table></div>`
      : `<p class="set-note">No games in the Library yet.</p>`;
    const note = Z.safe ? "Safe mode: sizes aren't counted." : Z.counting ? "Counting now" : "";
    return setSub("Drives", "", drives ? `<div class="st-drives">${drives}</div>` : `<p class="set-note">Windows didn't list any drives.</p>`)
      + setSub("Games", note, table);
  }
  function storageMenu(el) {
    const it = itemFor(el);
    if (!it) return;
    toggleMenu(el, [moveItem(it), `<button data-action="open-game-folder" ${itemAttr(it)}>Open the game's folder</button>`,
      removable(it) ? `<hr><button class="danger" data-action="remove-pc" ${itemAttr(it)}>Remove from this PC</button>` : ""].join(""));
  }

  // Move to another drive, for a folder game: each other drive's Games
  // folder, or one the user picks.
  function openMoveFolder(it) {
    S.dialog = { kind: "movefolder", ik: it.kind, iid: it.id, key: recentKey(it), name: it.name, pick: "", custom: "" };
    askSizes();
    renderOverlay(); focusOverlay();
  }
  function viewMoveFolder(D) {
    const it = dialogItem(D), M = S.folderMove;
    if (!it && !(M && M.key === D.key)) return "";
    const z = (it && sizeOf(it)) || {}, size = z.install || 0, dir = z.dir || (it && it.f && it.f.dir) || "";
    const wrap = (inner, sub) => moveWrap(D.name, inner, sub);
    if (M && M.key === D.key) {
      const m = moveNow();
      return moveWrap(D.name, `${moveBody(m)}
        <div class="dialog-foot"><button class="btn quiet" data-action="move-mini">Minimise</button>${M.step === "check" || M.step === "firewall" ? "" : `<button class="btn" data-action="movefolder-cancel">Stop the move</button>`}</div>`, "", true);
    }
    const only = (text) => wrap(`<p class="note" role="status">${text}</p><div class="dialog-foot"><button class="btn quiet" data-action="dialog-cancel">Close</button></div>`);
    if (M || S.moving) return only("Another game is moving. Wait for it to finish.");
    if (!dir) return only(S.sizes && S.sizes.safe ? "Safe mode: moving games is off." : "Still counting this game's files.");
    const here = dir.slice(0, 3).toUpperCase();
    const spots = ((S.sizes || {}).drives || []).filter((d) => d.root.toUpperCase() !== here).map((d) => ({ path: d.games, free: d.free }));
    if (D.custom) spots.push({ path: D.custom, free: -1 });
    const room = (s) => s.free < 0 || !size || s.free >= size * MOVE_ROOM;
    if (!D.pick) D.pick = (spots.find(room) || {}).path || "";
    const list = spots.length ? `<div class="choices">${spots.map((s) => `<button class="choice${D.pick === s.path ? " on" : ""}" data-action="movefolder-pick" data-path="${esc(s.path)}" ${room(s) ? "" : "disabled"} aria-pressed="${D.pick === s.path}">
        <span class="dot"></span><span><b>${esc(s.path)}</b><span>${s.free < 0 ? "A folder you chose" : fmtBytes(s.free) + " free"}${room(s) ? "" : ", not enough room"}</span></span></button>`).join("")}</div>`
      : `<p class="note">This PC has one drive. Choose a folder to move the game into.</p>`;
    const after = [it && it.source === "hydra" ? "Hydra keeps looking in the old folder, so choose the game's program again in Hydra afterwards." : "",
      it && isOffline(it) ? "It stays offline: Windows asks once to change the firewall rules." : ""].filter(Boolean).map((t) => `<p class="hint">${esc(t)}</p>`).join("");
    return wrap(`${list}
      <p class="hint">Copied and checked first, then the old folder goes to the Recycle Bin.</p>${after}
      <div class="dialog-foot"><button class="btn quiet" data-action="movefolder-browse">Choose a folder</button><span class="grow"></span><button class="btn quiet" data-action="dialog-cancel">Cancel</button>
        <button class="btn primary" data-action="movefolder-go" ${D.pick ? "" : "disabled"}>Move</button></div>`,
      `${size ? esc(about(size)) + ", now in " : "Now in "}<span class="selectable">${esc(dir)}</span>`);
  }
