  // --------------------------------------------------------------- updates --
  // Games no store keeps up to date: whether a newer version is out, and an
  // update the user brings (a folder or a .zip), applied with a copy first.
  const UPDATABLE = ["folder", "added", "shortcut", "hydra", "other"];
  const updKey = (it) => (it.kind === "game" ? (it.g.launch.type === "exe" && UPDATABLE.includes(it.source) ? "game:" + it.id : "") : UPDATABLE.includes(it.source) && !it.appId ? "found:" + it.id : "");
  function updateNavCount() {
    const keys = new Set([...(S.updates ? S.updates.games : []), ...(S.dl || [])].filter((g) => g.newer).map((g) => g.key));
    const n = keys.size + (S.copies ? S.copies.length : 0);
    const el = $("#upd-count"); if (el) el.textContent = n ? String(n) : "";
  }
  // "Is this a new copy?": a game reinstalled as a whole new version.
  function copyCard(o, compact) {
    const C = S.carried[o.old.key] || {};
    const btn = (attrs, label, cls = "ln-button--secondary") => `<button class="ln-button ${cls} ln-button--sm" ${attrs} ${C.busy ? "disabled" : ""}>${label}</button>`;
    const ids = `data-old="${esc(o.old.key)}" data-new="${esc(o.new.key)}" data-name="${esc(o.old.name)}"`;
    const what = [o.profiles ? "its profiles and their saves" : o.saves.length ? `its saves (${o.saves.map(esc).join(", ")})` : "", "playtime, status, collections and settings"].filter(Boolean).join(", ");
    return `<div class="copy-card"><div><b>${esc(o.new.name)} looks like a new copy of ${esc(o.old.name)}</b>
      <p>${o.old.kept ? "Lootnest kept its saves when you removed it from this PC." : o.oldGone ? `The old folder, <span class="selectable">${esc(o.old.dir)}</span>, is gone.` : `The old copy is still in <span class="selectable">${esc(o.old.dir)}</span>.`} Lootnest can carry over ${what}.</p></div>
      <div class="gpage-actions">${btn(`data-action="copy-carry" ${ids}`, C.busy ? "Carrying over" : "Carry everything over")}${compact ? "" : btn(`data-action="copy-dismiss" ${ids}`, "Not the same game", "ln-button--ghost")}</div></div>`;
  }
  function carriedCard(oldKey) {
    const C = S.carried[oldKey];
    if (!C || !C.done) return "";
    const saves = C.profile ? `${esc(pname(C.profile))}'s files are in the new copy.` : C.saves.length ? `${C.saves.length} save location${C.saves.length === 1 ? "" : "s"} came along${C.saves.some((x) => x.replaced) ? ", and the new copy's own are in backups" : ""}.` : "It had no saves inside its folder.";
    return `<div class="copy-card copy-card--done">${GI.tick}<div><b>${C.name ? esc(C.name) + " is carried over to its new copy" : "Carried over"}</b><p>${saves} Playtime, status, collections and settings moved too.</p></div>
      ${C.oldThere && !C.deleted ? `<div class="gpage-actions"><button class="ln-button ln-button--ghost ln-button--sm" data-action="copy-delete-old" data-old="${esc(oldKey)}" ${C.busy ? "disabled" : ""}>Delete the old folder</button></div>` : ""}</div>`;
  }
  // The Updates page: a list of games you can filter by state, each with its
  // version against the latest, and beside it what needs a look.
  const vtxt = (v) => (v ? (v.startsWith("build ") ? "Build " + v.slice(6) : v) : "");
  const verIn = (t) => { const m = (t || "").match(/(?:^|[^a-z0-9])v?(\d+(?:\.\d+){1,3}[a-z]?)(?![\d.])/i) || (t || "").match(/\bbuild[\s._-]*(\d{4,})/i); return m ? (/build/i.test(m[0]) ? "build " + m[1] : m[1]) : ""; };
  const updItem = (key) => everyItem().find((x) => updKey(x) === key) || null;
  function updState(r, U) {
    const last = (r.applied || []).find((u) => u.state === "applied");
    if (last && last.broke) return { k: "low", text: "Closed right after starting" };
    if (r.newer) return { k: "new", text: "Newer version out" };
    if (r.noNews || !U.newsOn) return { k: "dim", text: "Can't tell" };
    return { k: "ok", text: "Up to date" };
  }
  // The game's icon, else its cover, else its initial; never its name again.
  function updArt(it, name) {
    if (it) ensureCover(it);
    const k = it && coverKey(it), ic = it && S.icons[k], c = !ic && it && artFor(it, "cover").url;
    return ic ? `<img src="${ic}" alt="">` : c ? `<img class="upd-cover" src="${c}" alt="">` : `<b>${esc(initial(name))}</b>`;
  }
  const updVer = (have, to) => `<div class="upd-ver">
      <div><span>Your copy</span><b class="ln-num">${have ? esc(have) : "Unknown"}</b></div>
      ${to ? `<svg class="upd-arrow" viewBox="0 0 20 20" aria-hidden="true"><path d="M4 10h11M11 6l4 4-4 4"/></svg><div><span>${to[0]}</span><b class="ln-num">${esc(to[1])}</b></div>` : ""}</div>`;
  const midTrunc = (s, n) => s.length <= n ? s : s.slice(0, Math.ceil(n / 2) - 1) + "…" + s.slice(-Math.floor(n / 2));
  // When the row is narrow, the start gives way and the last 16 characters stay.
  const updPath = (p) => { const t = midTrunc(p, 56), i = Math.max(t.length - 16, Math.ceil(t.length / 2)); return `<span class="upd-path selectable" title="${esc(p)}"><span>${esc(t.slice(0, i))}</span>${esc(t.slice(i))}</span>`; };
  function updCard(r, U, lead) {
    const st = updState(r, U);
    const last = (r.applied || []).find((u) => u.state === "applied");
    const latest = r.newer ? verIn(r.newer.title) : "";
    const have = r.version ? vtxt(r.version) : "";
    const ver = updVer(have, r.newer && ["Latest", latest ? vtxt(latest) : day(r.newer.date)]);
    const note = r.newer ? `<button class="upd-note" data-action="upd-notes" data-key="${esc(r.key)}" title="${esc(r.newer.title)}">Patch notes, ${esc(day(r.newer.date))}</button>`
      : last ? `<span title="${esc(last.source)}">Updated ${esc(ago(last.at))}${last.to ? " to " + esc(vtxt(last.to)) : ""}</span>`
      : r.noNews ? `<span>Not on the Steam store</span>`
      : U.newsOn ? `<span>Files from ${esc(day(r.changed))}</span>` : "";
    const primary = `<button class="ln-button ${lead ? "ln-button--primary" : "ln-button--secondary"} ln-button--sm" data-action="upd-start" data-key="${esc(r.key)}" data-name="${esc(r.name)}">${r.newer ? "Apply the update" : "Apply an update"}</button>`;
    return `<li class="upd-card upd-card--${st.k}">
      <button class="upd-art" data-action="upd-open-game" data-key="${esc(r.key)}" aria-label="Open ${esc(r.name)}'s page" tabindex="-1">${updArt(updItem(r.key), r.name)}</button>
      <div class="upd-main">
        <div class="upd-title-row"><button class="upd-name" data-action="upd-open-game" data-key="${esc(r.key)}">${esc(r.name)}</button>
          <span class="upd-pill upd-pill--${st.k}"><i aria-hidden="true"></i>${esc(st.text)}</span></div>
        <p class="upd-meta"><span>${esc(sourceLabel(r.source || "folder"))}</span>${updPath(r.dir)}${note}</p>
      </div>
      ${ver}
      <div class="upd-act">${primary}
        <button class="ln-button ln-button--ghost ln-button--sm ln-button--icon" data-action="upd-more" data-key="${esc(r.key)}" data-name="${esc(r.name)}" aria-label="More for ${esc(r.name)}" aria-haspopup="menu" title="More">${GI.more || "&#8943;"}</button></div>
    </li>`;
  }
  // Updates already in Downloads, matched by name: each goes through the
  // same drawer as one the user chooses.
  function dlVer(m) {
    const c = m.check || {}, mute = (t) => `<span class="dl-muted">${t}</span>`;
    if (m.newer) return `${esc(vtxt(c.have))} to ${esc(vtxt(c.to))}${c.state === "skips" ? " " + mute(`(made for ${esc(vtxt(c.from))})`) : ""}`;
    if (c.state === "same") return mute("Same version as your copy");
    if (c.state === "older") return mute(`${esc(vtxt(c.to))}, older than your copy`);
    return mute(c.to ? `Brings ${esc(vtxt(c.to))}. Your copy's version isn't known.` : "No version in its name");
  }
  const dlIds = (m) => `data-key="${esc(m.key)}" data-name="${esc(m.game)}" data-path="${esc(m.path)}"`;
  const dlRecycle = (m) => `<button class="ln-button ln-button--secondary ln-button--sm" data-action="dl-recycle" data-path="${esc(m.path)}">Delete the downloaded file</button>`;
  function dlCard(m) {
    const c = m.check || {}, same = c.state === "same";
    const note = c.state === "skips" ? `Made for ${esc(vtxt(c.from))}` : same ? "Same version as your copy" : c.state === "older" ? "Older than your copy" : c.to ? "" : "No version in its name";
    return `<li class="upd-card dl-card"><span class="upd-art" aria-hidden="true">${updArt(updItem(m.key), m.game)}</span>
      <div class="upd-main"><b class="upd-name">${esc(m.game)}</b><p class="upd-meta">${updPath(m.name)}${note ? `<span>${note}</span>` : ""}</p></div>
      ${updVer(vtxt(c.have), ["In Downloads", c.to ? vtxt(c.to) : "Unknown"])}
      <div class="upd-act">${same ? dlRecycle(m) : `<button class="ln-button ln-button--secondary ln-button--sm" data-action="dl-review" ${dlIds(m)} aria-haspopup="dialog">Review and apply</button>`}
        <button class="ln-button ln-button--ghost ln-button--sm" data-action="dl-ignore" ${dlIds(m)}>Ignore</button></div></li>`;
  }
  function dlNote(it) {
    const uk = updKey(it), m = uk && (S.dl || []).find((x) => x.key === uk && x.newer);
    return m ? `<p class="gpage-note">${GI.load}<span>An update is in your Downloads: <span class="selectable">${esc(m.name)}</span>, ${dlVer(m)}.</span>
      <button class="gpage-link" data-action="dl-review" ${dlIds(m)} aria-haspopup="dialog">Review and apply</button></p>` : "";
  }
  function viewUpdates() {
    const U = S.updates;
    const checking = S.updChecking;
    const refresh = `<button class="ln-button ln-button--secondary ln-button--sm" data-action="upd-refresh" ${checking ? 'aria-busy="true" disabled' : ""}>${checking ? '<span class="ln-spinner" aria-hidden="true"></span>Checking' : `${GI.refresh}Check again`}</button>`;
    const head = (act) => `<header class="upd-head"><div><h1>Updates</h1><p>Games no store keeps up to date. Lootnest compares your copy with Steam's patch notes, and applies an update you bring with a copy of everything first.</p></div>${act}</header>`;
    if (!U) { api.updates().catch(() => {}); if (itchOn()) api.itchCheck("", false).catch(() => {}); if (gogOn()) api.gogCheck("", false).catch(() => {}); return `<div class="upd-page ln-root">${head("")}<div class="upd-skel">${[0, 1, 2].map(() => '<div class="ln-skeleton"></div>').join("")}</div></div>`; }
    if (!S.copies) api.newCopies().catch(() => {});
    if (!U.games.length) {
      return `<div class="upd-page ln-root">${head(refresh)}${emptyState(GI.load, "Nothing to keep up to date here", "Steam, Epic, GOG and the other launchers keep their own games up to date. Games in a folder, from Hydra or added by hand show up here, with their version against the latest.", `<button class="ln-button ln-button--secondary" data-action="nav" data-view="library">Open the Library</button>`)}</div>`;
    }
    const offers = S.copies || [];
    const done = Object.keys(S.carried).filter((k) => S.carried[k].done && !S.carried[k].deleted);
    const broke = U.games.filter((g) => ((g.applied || []).find((u) => u.state === "applied") || {}).broke);
    const nNew = U.games.filter((g) => g.newer).length;
    const nLook = offers.length + broke.length;
    const F = S.updFilter || "all";
    const seg = (k, n, label) => `<button data-action="upd-filter" data-f="${k}" aria-pressed="${F === k}">${k === "new" && n ? '<i class="upd-dot" aria-hidden="true"></i>' : ""}${label}<span class="ln-count ln-num">${n}</span></button>`;
    const filter = `<div class="ln-segmented upd-filter" role="group" aria-label="Show">
      ${seg("all", U.games.length, "All")}${seg("new", nNew, "Newer version out")}
      ${seg("ok", U.games.filter((g) => updState(g, U).k === "ok").length, "Up to date")}${seg("dim", U.games.filter((g) => updState(g, U).k === "dim").length, "Can't tell")}</div>`;
    const news = U.newsOn ? "" : `<div class="upd-callout"><div><b>Find out when a newer version is out</b><p>Lootnest reads each game's patch notes from Steam's news feed and compares their date with your copy. Only the game's Steam id, or its name, is sent.</p></div><button class="ln-button ln-button--secondary ln-button--sm" data-action="news-on">Turn it on</button></div>`;
    const dl = S.dl || [];
    const found = dl.length ? `<section aria-label="Found in your Downloads"><div class="upd-sec-head"><h2>Found in your Downloads</h2><span class="upd-sec-n ln-num">${dl.length}</span></div><ul class="upd-cards">${dl.map(dlCard).join("")}</ul></section>` : "";
    const look = !(nLook || done.length) ? "" : `<section class="upd-look" aria-label="Needs a look"><div class="upd-sec-head"><h2>Needs a look</h2><span class="upd-sec-n ln-num">${nLook}</span></div>
      ${done.map(carriedCard).join("")}${offers.map((o) => copyCard(o, false)).join("")}
      ${broke.map((g) => { const u = (g.applied || []).find((x) => x.state === "applied"); return `<div class="copy-card copy-card--warn">${GI.alert}<div><b>${esc(g.name)} closed right after starting</b><p>It's the first start since the update from ${esc(u.source)}. If it doesn't work, Undo puts the game back as it was.</p></div><div class="gpage-actions">${u.noUndo ? "" : `<button class="ln-button ln-button--secondary ln-button--sm" data-action="upd-undo" data-key="${esc(g.key)}" data-id="${esc(u.id)}" data-name="${esc(g.name)}">Undo the update</button>`}</div></div>`; }).join("")}
      </section>`;
    const list = U.games.filter((g) => F === "all" || updState(g, U).k === F || (F === "new" && g.newer))
      .sort((a, b) => (b.newer ? 1 : 0) - (a.newer ? 1 : 0) || a.name.localeCompare(b.name));
    const cards = list.length ? `<ol class="upd-cards">${list.map((r, i) => updCard(r, U, i === 0 && !!r.newer)).join("")}</ol>` : `<p class="upd-empty">None of your games ${F === "new" ? "has a newer version out" : F === "ok" ? "is known to be up to date" : "is in this group"}.</p>`;
    return `<div class="upd-page ln-root">${head("")}<div class="upd-cols">
      <div class="upd-main-col">${found}${news}<section aria-label="Your games">${filter}${cards}</section></div>
      <aside class="upd-aside">${look}<div class="upd-check">${refresh}<span>Checked ${esc(ago(U.at))}</span></div></aside></div></div>`;
  }
  // An update's history: every one Lootnest applied, newest first.
  function viewUpdHistory() {
    const D = S.dialog, U = S.updates;
    const r = U && U.games.find((g) => g.key === D.key);
    if (!r) return "";
    const list = r.applied || [];
    const when = (t) => new Date(t).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
    const firstApplied = list.find((u) => u.state === "applied");
    const rows = list.map((u) => `<li class="pt-row${u.state !== "applied" ? " pt-row--auto" : ""}"><span class="pt-dot" aria-hidden="true"></span><div class="pt-main"><b>${u.to ? esc(vtxt(u.from ? u.from + " to " + u.to : u.to)) : esc(u.source)}</b>
      <span class="pt-facts"><span>${esc(when(u.at))}</span><span>${u.added.length} added, ${u.replaced.length} replaced</span><span class="selectable">${esc(u.source)}</span>${u.state !== "applied" ? "<span>Undone</span>" : ""}${u.broke ? "<span>Closed right after starting</span>" : ""}</span></div>
      <div class="pt-actions">${u === firstApplied && !u.noUndo ? `<button class="ln-button ln-button--secondary ln-button--sm" data-action="upd-undo" data-key="${esc(r.key)}" data-id="${esc(u.id)}" data-name="${esc(r.name)}">Undo</button>` : ""}</div></li>`).join("");
    return drawer({ label: `${esc(r.name)} update history`, title: "Update history", sub: `${esc(r.name)}${r.version ? ", version " + esc(vtxt(r.version)) : ""}`,
      body: list.length ? `<ol class="pt-list pt-list--all">${rows}</ol>` : `<p>Lootnest hasn't applied an update to ${esc(r.name)} yet.</p>`,
      foot: `<p class="gpage-hint ln-drawer__note">Updates come off newest first. Each one's replaced files are kept in Lootnest's backups.</p>` });
  }
  function viewUpdateDrawer() {
    const D = S.dialog;
    const btn = (attrs, label, cls = "ln-button--secondary") => `<button class="ln-button ${cls} ln-button--sm" ${attrs} ${D.busy ? "disabled" : ""}>${label}</button>`;
    const shell = (body, foot) => drawer({ label: `Update ${esc(D.name)}`, title: `Update ${esc(D.name)}`, sub: D.itch ? "Downloaded from itch.io. Nothing in the game's folder changes until you apply it." : D.gog ? "Downloaded from GOG. Nothing in the game's folder changes until you run it." : "With an update you have: a folder, a .zip, .7z or .rar, or an installer", body, foot });
    const choose = `<div class="gpage-actions">${btn('data-action="upd-pick" data-kind="update"', "Choose a .zip, .7z or .rar")}${btn('data-action="upd-pick" data-kind="updateDir"', "Choose a folder", "ln-button--ghost")}${btn('data-action="upd-pick" data-kind="installer"', "Run an installer", "ln-button--ghost")}</div>`;
    if (D.done) {
      const u = D.done, dlm = !D.undone && (S.dl || []).find((m) => m.path === D.path);
      return shell(D.undone ? `<div class="run-card run-card--ok"><div class="run-head"><span class="run-dot"></span><div><b>Undone</b><p>${esc(D.name)} is as it was before the update from ${esc(u.source)}.</p></div></div></div>`
        : `<div class="run-card run-card--ok"><div class="run-head"><span class="run-dot"></span><div><b>Updated</b><p>${u.added.length} files added and ${u.replaced.length} replaced or removed, from ${esc(u.source)}. ${u.noUndo ? "The saves are kept. No copy of the game was kept, so it can't be undone." : "The saves and the old files are kept, so you can undo it."} The first time you start it, Lootnest checks it runs.</p></div></div></div>`,
        D.undone ? `<span class="grow"></span>${btn('data-action="dialog-cancel"', "Done", "ln-button--primary")}`
          : `${u.noUndo ? "" : btn(`data-action="upd-undo" data-key="${esc(D.key)}" data-id="${esc(u.id)}" data-name="${esc(D.name)}"`, "Undo", "ln-button--ghost")}${btn('data-action="upd-another"', "Apply another update", "ln-button--ghost")}${dlm ? dlRecycle(dlm) : ""}<span class="grow"></span>${btn('data-action="dialog-cancel"', "Done", "ln-button--primary")}`);
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
      return shell(`<div class="gpage-sub">${gpHead("Run an installer")}<p class="gpage-hint">Lootnest copies the saves, runs <span class="selectable">${esc(P.name)}</span>, then records what it changed in <span class="selectable">${esc(D.dir)}</span>. ${D.gog ? "GOG's installer goes into that folder without questions and shows its own progress." : "Point the installer at that folder."}</p></div>
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

