  // ------------------------------------------------------------------ mods --
  // A game's mods (core/mods.go), put into its folder as hard links to
  // Lootnest's copy or by copy, with every game file they replace kept and a
  // way back. The user brings each mod; Lootnest never downloads one. A game with profiles has a set for each
  // profile and run that wants one: the mods are shared, each set says which
  // are on and in what order, and Play switches to the set of whoever plays.
  // A run without one uses its person's, a person without one the game's.
  const MI = { up: gsvg('<path d="M5 12.5 10 7.5l5 5"/>'), down: gsvg('<path d="M5 7.5l5 5 5-5"/>'),
    grip: gsvg('<circle cx="7.5" cy="5.5" r="1"/><circle cx="12.5" cy="5.5" r="1"/><circle cx="7.5" cy="10" r="1"/><circle cx="12.5" cy="10" r="1"/><circle cx="7.5" cy="14.5" r="1"/><circle cx="12.5" cy="14.5" r="1"/>', true) };
  function modsOf(it) {
    const key = recentKey(it);
    if (!(key in S.mods)) { S.mods[key] = null; loadMods(key); }
    return S.mods[key];
  }
  function loadMods(key) {
    return api.mods(key).then((m) => { S.mods[key] = m; if (S.view === "game") render(); }).catch(() => { S.mods[key] = { none: true }; });
  }
  // The Mods tab shows for a game with a folder that can have mods, or that
  // still has some in it; an anti-cheat game says why in its note.
  const modsTabOn = (M) => !!(M && M.dir && !M.none && (!M.refused || M.deployed || M.pending));
  // With one profile the picker still shows once it has a set of its own,
  // as Play uses that one.
  const setsOn = (it, M) => it.kind === "game" && (cfg().players.length > 1 || Object.entries(M.uses || {}).some(([p, o]) => o === p));
  const modsOwner = (M, o) => (o ? (M.uses || {})[o] || "" : "");
  const setRows = (M, o) => (M.sets || {})[modsOwner(M, o)] || [];
  const onIds = (M, o) => setRows(M, o).filter((m) => m.on).map((m) => m.id);
  const modsDirty = (M, o) => onIds(M, o).join() !== ((M.deployed || {}).mods || []).join();
  const setName = (o) => (o && player(o) ? `${possessive(player(o).name)} mods` : "the game's mods");
  function modsState(M, o, sets) {
    if (M.pending) return "Not finished";
    const mine = !sets || (!!M.deployed && (M.deployed.owner || "") === modsOwner(M, o));
    if (mine && modsDirty(M, o)) return "Changes not applied yet";
    const n = M.deployed ? M.deployed.mods.length : 0;
    return n ? `Modded: ${n} mod${n === 1 ? "" : "s"}` : "Vanilla";
  }
  const modsRunText = (R) => `${R.what === "apply" ? "Putting the mods in" : "Putting the game back as it was"}${R.total ? `, ${R.done} of ${R.total} files` : ""}. Don't start the game until it's done.`;
  const nFiles = (n) => `${n} file${n === 1 ? "" : "s"}`;
  // What a mod that's on overrides and loses, from its set's conflicts (the
  // last mod with a file wins it): by the other mod, the files.
  function modVs(M, o, id) {
    const over = {}, lose = {};
    ((M.conflicts || {})[modsOwner(M, o)] || []).forEach((c) => {
      const w = c.mods[c.mods.length - 1];
      if (w === id) c.mods.slice(0, -1).forEach((x) => (over[x] = over[x] || []).push(c.rel));
      else if (c.mods.includes(id)) (lose[w] = lose[w] || []).push(c.rel);
    });
    return { over, lose };
  }
  const modName = (M, o, x) => (setRows(M, o).find((m) => m.id === x) || { name: "another mod" }).name;
  function modVsText(M, o, id) {
    const { over, lose } = modVs(M, o, id), name = (x) => modName(M, o, x);
    const line = (verb, by, to) => {
      const ks = Object.keys(by);
      if (!ks.length) return "";
      const n = new Set(ks.flatMap((k) => by[k])).size;
      return `${verb} ${nFiles(n)} ${to} ${ks.length === 1 ? name(ks[0]) : `${ks.length} mods`}.`;
    };
    return [line("Overrides", over, "from"), line("Loses", lose, "to")].filter(Boolean).join(" ");
  }
  const modWhere = (m) => (m.atRoot ? "At the game folder's top" : m.into ? `Into ${m.into}` : "Into the game's folder");
  function modsBlock(it) {
    const M = modsOf(it);
    if (!modsTabOn(M)) return "";
    const key = recentKey(it), R = S.modsRun[key], busy = !!R, sets = setsOn(it, M);
    const o = sets && player(S.modsFor[key]) ? S.modsFor[key] : "", uses = modsOwner(M, o), ro = o !== uses, list = setRows(M, o);
    const attrs = (m) => `data-mods="${esc(key)}" data-owner="${esc(o)}" data-id="${esc(m.id)}" data-name="${esc(m.name)}"`;
    const icon = (a, ic, label, off) => `<button class="ln-button ln-button--ghost ln-button--icon ln-button--sm" ${a} aria-label="${esc(label)}" ${busy || off ? "disabled" : ""}>${ic}</button>`;
    const drag = !busy && !ro && list.length > 1;
    const rows = list.map((m, i) => {
      const vs = m.on ? modVsText(M, o, m.id) : "";
      return `<li class="mod-row${m.on ? "" : " is-off"}" data-id="${esc(m.id)}">${drag ? `<span class="mod-grip" title="Drag to change the order">${MI.grip}</span>` : ""}<div class="pt-main"><b>${esc(m.name)}</b>
        <span class="pt-facts"><span>${nFiles(m.files)}</span><span>${esc(fmtBytes(m.bytes))}</span><span>${esc(modWhere(m))}</span></span>
        ${vs ? `<p class="mod-vs"><span>${esc(vs)}</span><button class="ln-button ln-button--ghost ln-button--sm" data-action="mod-conflicts" ${attrs(m)}>See the files</button></p>` : ""}
        ${m.changed ? `<p class="mod-vs">${GI.alert}<span>The game changed Lootnest's copy of this mod. Until you turn it off, or remove it and add it again, Play starts the game without mods.</span></p>` : ""}</div>
      <div class="mod-actions">${icon(`data-action="mod-move" data-by="-1" ${attrs(m)}`, MI.up, `Move ${m.name} up`, ro || i === 0)}${icon(`data-action="mod-move" data-by="1" ${attrs(m)}`, MI.down, `Move ${m.name} down`, ro || i === list.length - 1)}
        ${icon(`data-action="mod-remove" ${attrs(m)}`, GI.trash, `Remove ${m.name}`)}
        <span class="switch"><input type="checkbox" data-action="mod-toggle" ${attrs(m)} ${m.on ? "checked" : ""} ${busy || ro ? "disabled" : ""} aria-label="Use ${esc(m.name)}"><span></span></span></div></li>`;
    }).join("");
    const btn = (a, label, cls = "ln-button--secondary", off) => `<button class="ln-button ${cls} ln-button--sm" ${a} data-mods="${esc(key)}" data-owner="${esc(o)}" data-name="${esc(it.name)}" ${busy || off ? "disabled" : ""}>${label}</button>`;
    const D = M.deployed, pick = sets ? `<div class="mods-for"><label class="try-field"><span>Mods for</span>
        <select id="mods-for" class="ln-input" data-mods="${esc(key)}" ${busy ? "disabled" : ""}><option value="">The game, for everyone</option>
          ${grouped().map((p) => `<option value="${esc(p.id)}" ${p.id === o ? "selected" : ""}>${esc(p.name)}</option>`).join("")}</select></label>
        <p class="gpage-hint">${D ? `In the game's folder now: ${esc(setName(D.owner))}.` : "The game's folder has no mods in it now."} Play switches to the mods of whoever plays, after their saves.</p></div>` : "";
    const inherit = ro ? `<p class="gpage-note mods-uses">${GI.copy}<span>Uses ${esc(setName(uses))}. Changes to those show here too.</span>${btn('data-action="mods-own"', "Make its own")}</p>` : "";
    const run = R ? `<div class="mover"><div class="meter"><i style="width:${R.total ? Math.round((R.done / R.total) * 100) : 0}%"></i></div>
        <p class="gpage-hint" role="status">${esc(modsRunText(R))}</p></div>` : "";
    const why = M.refused || M.safe || "", back = M.deployed || M.pending;
    const actions = M.refused ? (back ? `<div class="gpage-actions">${btn('data-action="mods-restore"', "Put the game back as it was")}</div>` : "")
      : `<div class="gpage-actions">${btn('data-action="mod-add"', `${GI.plus}Add a mod`)}${icon('data-action="mods-more" aria-haspopup="menu" aria-expanded="false"', GI.more, "More mod options")}<span class="grow"></span>
        ${back ? btn('data-action="mods-restore"', "Put the game back as it was", "ln-button--ghost", M.safe) : ""}
        ${list.length || M.deployed ? btn('data-action="mods-apply"', "Apply", "ln-button--secondary", M.safe || !modsDirty(M, o) && !M.pending) : ""}</div>`;
    return `<section class="gpage-block gp-mods" data-key="${esc(key)}"><div class="ln-section-head"><div><h2>Mods</h2></div><span class="ln-tag${M.deployed && !modsDirty(M, o) ? " ln-tag--strong" : ""}">${esc(modsState(M, o, sets))}</span></div>
      <p class="gpage-hint">Lootnest keeps every game file the mods replace and copies the saves first. A later mod wins where two have the same file; drag a mod, or use its arrows, to change the order.</p>
      ${pick}${why ? `<p class="gpage-note">${GI.alert}<span>${esc(why)}</span></p>` : ""}
      ${M.pending ? `<p class="gpage-note">${GI.history}<span>The last change to the mods didn't finish. Lootnest finishes it when it starts, or when you apply.</span></p>` : ""}
      ${inherit}${rows ? `<ol class="mod-list${drag ? " can-drag" : ""}" data-mods="${esc(key)}" data-owner="${esc(o)}">${rows}</ol>` : M.refused ? "" : `<p class="gpage-hint">No mods yet. Add one you have, as a .zip, .7z or .rar, or a folder.</p>`}
      ${run}${actions}</section>`;
  }

  // Adding a mod: choose it, see what it would do, then add it to the list.
  function openAddMod(el) {
    S.dialog = { kind: "modplan", key: el.dataset.mods, owner: el.dataset.owner || "", name: el.dataset.name };
    renderOverlay(); focusOverlay();
  }
  function viewModPlan() {
    const D = S.dialog;
    const btn = (attrs, label, cls = "ln-button--secondary") => `<button class="ln-button ${cls} ln-button--sm" ${attrs} ${D.busy ? "disabled" : ""}>${label}</button>`;
    const choose = `<div class="gpage-actions">${btn('data-action="mod-pick" data-kind="mod"', "Choose a .zip, .7z or .rar")}${btn('data-action="mod-pick" data-kind="modDir"', "Choose a folder", "ln-button--ghost")}</div>`;
    const shell = (body, foot) => drawer({ label: `Add a mod to ${esc(D.name)}`, title: "Add a mod", sub: `${esc(D.name)}. Nothing in the game's folder changes until you apply.`, body, foot });
    const intro = `<p class="gpage-hint">Lootnest doesn't download mods. Choose one you have, and Lootnest shows what it would change first. It keeps its own copy, so the file you chose can go afterwards.</p>`;
    if (D.busy) return shell(lnInline("", "Opening the mod", "A big mod takes a minute."));
    if (D.error) return shell(`${lnInline("error", esc(D.error))}${choose}`);
    if (D.refused) return shell(`${lnInline("error", "Lootnest won't put this mod in", esc(D.refused))}${choose}`);
    const P = D.plan, m = D.mod;
    if (!P || !m) return shell(`${intro}${choose}`);
    const shown = P.replaces.slice(0, 8);
    const body = `<div class="gpage-sub">${gpHead(esc(m.name))}<dl class="ln-facts">
        <div><dt>Adds</dt><dd>${P.added} file${P.added === 1 ? "" : "s"}</dd></div>
        <div><dt>Replaces</dt><dd>${P.replaced} game file${P.replaced === 1 ? "" : "s"}</dd></div>
        <div><dt>Size</dt><dd>${esc(fmtBytes(P.bytes))}</dd></div>
        <div><dt>Where</dt><dd>${esc(modWhere(m))}</dd></div></dl>
        ${m.atRoot ? `<p class="find-low">None of its folders match the game's, so its files go at the top of the game's folder as they are. If the mod's notes say to put them somewhere else, extract it into that layout and add the folder instead.</p>` : ""}</div>
      ${shown.length ? `<div class="gpage-sub">${gpHead("Game files it replaces")}<ul class="mod-files">${shown.map((r) => `<li class="selectable">${esc(r)}</li>`).join("")}</ul>
        ${P.replaces.length > shown.length ? `<p class="gpage-hint">And ${P.replaces.length - shown.length} more.</p>` : ""}</div>` : ""}`;
    return shell(body, `${btn('data-action="mod-pick" data-kind="mod"', "Choose another", "ln-button--ghost")}<span class="grow"></span>
      ${btn('data-action="mod-cancel"', "Cancel", "ln-button--ghost")}${btn('data-action="mod-keep"', "Add to the list", "ln-button--primary")}`);
  }

  // The Mods tab's More menu: how mods go in. A link takes no space and is
  // made only on the drive Lootnest keeps the mod on; settings files are
  // always copied.
  function modsMoreHTML() {
    const on = !cfg().settings.modsLinkOff;
    return `<div class="ctx-title">For every game</div><button role="menuitemcheckbox" aria-checked="${on}" data-action="mods-link" title="On the drive Lootnest keeps your mods on, a mod's file goes in as a link to Lootnest's copy and takes no space. Settings files, and anything on another drive, are copied. This applies to every game."><span class="grow">Install mods by linking when possible</span>${on ? GI.tick : ""}</button>`;
  }

  // The files a mod overrides and loses, by the other mod.
  function viewModConflicts() {
    const D = S.dialog, M = S.mods[D.key] || {}, me = setRows(M, D.owner).find((m) => m.id === D.id);
    if (!me) return "";
    const { over, lose } = modVs(M, D.owner, D.id);
    const group = (by, head) => Object.keys(by).map((x) => `<div class="gpage-sub">${gpHead(esc(head(modName(M, D.owner, x))))}
        <ul class="mod-files">${by[x].map((r) => `<li class="selectable">${esc(r)}</li>`).join("")}</ul></div>`).join("");
    return drawer({ label: `${esc(me.name)} and other mods`, title: esc(me.name), sub: "When two mods that are on have the same file, the one lower in the list goes in.",
      body: `${group(over, (n) => `Overrides these from ${n}`)}${group(lose, (n) => `Loses these to ${n}`)}` });
  }

  // Reordering by dragging a row; the arrows do the same from the keyboard.
  // The new order is saved for the set, and Apply puts it in.
  function saveModOrder(key, owner, ids) {
    const M = S.mods[key], o = modsOwner(M, owner), rows = M.sets[o];
    M.sets[o] = ids.map((id) => rows.find((m) => m.id === id));
    api.modOrder(key, owner, ids).then(() => uiSound("move"), (e) => toast(errText(e), "error")).then(() => loadMods(key));
  }
  let modDrag = null;
  function modDragEnd(drop) {
    const d = modDrag;
    modDrag = null;
    if (!d || !d.on) return;
    d.ghost.remove();
    d.row.classList.remove("is-dragging");
    d.list.querySelectorAll(".ins-before, .ins-after").forEach((x) => x.classList.remove("ins-before", "ins-after"));
    if (!drop || !d.target) return;
    const was = [...d.list.children].map((r) => r.dataset.id), ids = was.filter((id) => id !== d.row.dataset.id);
    ids.splice(ids.indexOf(d.target.dataset.id) + (d.after ? 1 : 0), 0, d.row.dataset.id);
    if (ids.join() === was.join()) return;
    saveModOrder(d.list.dataset.mods, d.list.dataset.owner, ids);
  }
  document.addEventListener("pointerdown", (ev) => {
    const row = ev.button === 0 && ev.target.closest && ev.target.closest(".mod-list.can-drag > .mod-row");
    if (row && !ev.target.closest("button, input, label")) modDrag = { row, list: row.parentElement, x: ev.clientX, y: ev.clientY, id: ev.pointerId, on: false };
  });
  document.addEventListener("pointermove", (ev) => {
    const d = modDrag;
    if (!d || ev.pointerId !== d.id) return;
    if (!d.on) {
      if (Math.hypot(ev.clientX - d.x, ev.clientY - d.y) < 6) return;
      d.on = true;
      d.row.classList.add("is-dragging");
      d.ghost = document.createElement("div");
      d.ghost.className = "sg-ghost";
      d.ghost.textContent = d.row.querySelector("b").textContent;
      document.body.appendChild(d.ghost);
    }
    ev.preventDefault();
    d.ghost.style.transform = `translate(${ev.clientX + 14}px, ${ev.clientY + 10}px)`;
    d.list.querySelectorAll(".ins-before, .ins-after").forEach((x) => x.classList.remove("ins-before", "ins-after"));
    const el = document.elementFromPoint(ev.clientX, ev.clientY), t = el && el.closest(".mod-row");
    d.target = t && t !== d.row && t.parentElement === d.list ? t : null;
    if (!d.target) return;
    const r = t.getBoundingClientRect();
    d.after = ev.clientY > r.top + r.height / 2;
    t.classList.add(d.after ? "ins-after" : "ins-before");
  });
  document.addEventListener("pointerup", (ev) => { if (modDrag && ev.pointerId === modDrag.id) modDragEnd(true); });
  document.addEventListener("pointercancel", () => modDragEnd(false));
  document.addEventListener("keydown", (ev) => {
    if (ev.key !== "Escape" || !modDrag || !modDrag.on) return;
    ev.preventDefault(); ev.stopImmediatePropagation();
    modDragEnd(false);
  }, true);
