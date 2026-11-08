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
  const NEAR_FULL = 0.1; // a drive with less than this share free
  const MOVE_ROOM = 1.1; // a move needs the game's size and a tenth more
  // A game in a folder of its own that Lootnest moves itself.
  const movable = (it) => !["shortcut", "hydra"].includes(it.source) && (it.kind === "found" ? removable(it) : it.g.launch.type === "exe" && UPDATABLE.includes(it.source));
  const moveItem = (it) => movable(it) ? `<button data-action="movefolder-open" ${itemAttr(it)}>Move to another drive</button>`
    : it.appId ? `<button data-action="move-open" data-app="${it.appId}" data-name="${esc(it.name)}">Move with Game Mover</button>`
    : LAUNCHERS.includes(it.source) ? `<button disabled>Installed by ${esc(sourceLabel(it.source, it.f))}, so move it there</button>` : "";

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
    rows.sort((a, b) => k === "name" ? a.it.name.localeCompare(b.it.name) : ((b.z || {})[k] || 0) - ((a.z || {})[k] || 0) || a.it.name.localeCompare(b.it.name));
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
      const text = M.step === "check" ? "Checking the copy, then pointing the Library at it" : M.total ? `Copying ${fmtBytes(M.done)} of ${fmtBytes(M.total)}` : "Counting its files";
      return wrap(`${moveProgress(M.done, M.total, text, "Don't start the game until it's done.")}
        <div class="dialog-foot"><button class="btn quiet" data-action="dialog-cancel">Close</button>${M.step === "check" ? "" : `<button class="btn" data-action="movefolder-cancel">Stop the move</button>`}</div>`);
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
    return wrap(`${list}
      <p class="hint">Copied and checked first, then the old folder goes to the Recycle Bin.</p>
      <div class="dialog-foot"><button class="btn quiet" data-action="movefolder-browse">Choose a folder</button><span class="grow"></span><button class="btn quiet" data-action="dialog-cancel">Cancel</button>
        <button class="btn primary" data-action="movefolder-go" ${D.pick ? "" : "disabled"}>Move</button></div>`,
      `${size ? esc(about(size)) + ", now in " : "Now in "}<span class="selectable">${esc(dir)}</span>`);
  }
