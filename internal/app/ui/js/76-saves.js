  // ----------------------------------------------------------------- saves --
  // Saves: everything Lootnest keeps of every game's saves in one place.
  // What needs a look, the newest copies (safety net, save points, copies
  // before swaps) from ps_savesOverview, each game, and Cloud saves. Every
  // button here is one the game's page or Cloud saves already has.
  const SV_SHOWN = 8;
  const SV_KIND = { net: "Safety net copy", point: "Save point", swap: "Copy before a swap" };
  // What each kind of copy is, in one line, with its icon.
  const SV_KINDS = [
    ["point", "flag-pennant", "Save point", "One you made, by name or with Ctrl+Shift+F5 while playing."],
    ["swap", "arrows-left-right", "Copy before a swap", "Made by itself each time Lootnest swaps one profile's files for another's."],
    ["net", "vault", "Safety net copy", "Made by itself when the saves of a game from outside the stores change."],
  ];
  const svIcon = (kind) => ph((SV_KINDS.find((k) => k[0] === kind) || [])[1] || "file");
  const svKey = (c) => `${c.key}|${c.kind}|${c.id}`;
  const SV_COLS = [["name", "Game"], ["profiles", "Profiles"], ["last", "Newest copy"], ["points", "Save points"], ["size", "Size"]];
  // Arriving for Cloud saves: the section stays at the top while the rows
  // above it come in, until the page is scrolled by hand.
  function svToCloud() {
    const c = $("#sv-cloud");
    if (c) c.scrollIntoView({ block: "start" });
  }
  ["wheel", "pointerdown", "keydown"].forEach((t) => $("#view").addEventListener(t, () => { S.svCloud = false; }, { passive: true }));
  function askSaves() { api.savesOverview().catch(() => {}); api.saveCheck(false).catch(() => {}); }
  const svWhen = (t) => new Date(t).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
  const svOpen = (it, label, cls = "sv-game") => `<button class="${cls}" data-action="open-sheet" ${itemAttr(it)} data-tab="saves">${label}</button>`;

  function svLook() {
    const C = S.saveCheck || {}, B = S.backup, V = S.saves, rows = [];
    const row = (title, line, key) => rows.push(lookRow(title, line, key));
    const KEY = "ln-button ln-button--secondary ln-button--sm";
    const btn = (attrs, label) => `<button class="${KEY}" ${attrs}>${label}</button>`;
    (C.gone || []).forEach((g) => { const it = itemByKey(g.key); if (it) row(`${esc(possessive(it.name))} saves are gone from its folder`, "Lootnest kept copies of them.", svOpen(it, "Open its saves", KEY)); });
    Object.entries(S.d.conflicts || {}).forEach(([gid, c]) => {
      const g = game(gid), who = pname(c.player) || "Someone";
      if (g) row(`Steam Cloud replaced ${esc(possessive(who))} files in ${esc(g.name)}`, `${esc(svWhen(c.at))}. That session's files are kept as a copy.`,
        btn(`data-action="copies" data-game="${esc(gid)}" aria-haspopup="dialog"`, "See the copies")
        + `<button class="ln-button ln-button--ghost ln-button--sm ln-button--icon" data-action="conflict-ok" data-game="${esc(gid)}" aria-label="Got it" title="Got it">${GI.x}</button>`);
    });
    const two = V && !V.error ? [...new Set((V.games || []).filter((x) => x.state === "conflict" && game(x.gameId) && player(x.playerId)).map((x) => x.gameId))] : [];
    if (two.length) row(two.length === 1 ? `${esc(possessive(game(two[0]).name))} saves changed on two PCs` : `${two.length} games' saves changed on two PCs`, "", btn('data-action="nav" data-view="cloud"', "Choose"));
    const st = B && B.dir && B.status, failed = st && st.error && (!st.last || new Date(st.errorAt) > new Date(st.last));
    const err = (V && V.error) || (failed && st.error);
    if (err) row("Cloud saves didn't work", esc(err), btn('data-action="nav" data-view="cloud"', "See Cloud saves"));
    const unk = C.unknown || [];
    if (C.unknownN) row(C.unknownN === 1 ? `Lootnest doesn't know where ${esc(unk[0] ? unk[0].name : "a game")} keeps its saves` : `Lootnest doesn't know where ${C.unknownN} games keep their saves`, "",
      unk.length === 1 && itemByKey(unk[0].key) ? svOpen(itemByKey(unk[0].key), "Choose the folder", KEY) : btn('data-action="sc-games" data-kind="unknown" aria-haspopup="menu"', "Which games"));
    return rows;
  }

  // Search and filters over every copy: a game's or a copy's name, the
  // kind, whose it is, when, and the order. Typing redraws only the list,
  // so the search box keeps its place.
  const SV_WHEN = [["", "Any time"], ["day", "Today"], ["week", "This week"], ["month", "This month"]];
  const SV_ORDER = [["new", "Newest first"], ["old", "Oldest first"], ["size", "Largest first"]];
  const svF = () => S.svF || (S.svF = { q: "", kind: "", who: "", when: "", sort: "new" });
  const svFiltered = (list) => !svF().q.trim() && !svF().kind && !svF().who && !svF().when && svF().sort === "new" ? null : list;
  function svFilter(list) {
    const F = svF(), test = F.q.trim() ? nameMatcher(F.q) : null;
    const since = F.when ? Date.now() - { day: 864e5, week: 7 * 864e5, month: 30 * 864e5 }[F.when] : 0;
    const out = list.filter((c) => {
      const it = itemByKey(c.key);
      if (!it || (F.kind && c.kind !== F.kind) || (F.who && c.owner !== F.who) || (since && new Date(c.at).getTime() < since)) return false;
      return !test || test(it.name) || (c.name && test(c.name));
    });
    if (F.sort === "old") out.reverse();
    else if (F.sort === "size") out.sort((a, b) => (b.size || 0) - (a.size || 0));
    return out;
  }
  function svBar(list) {
    const F = svF(), n = (k) => list.filter((c) => c.kind === k).length;
    const owners = [...new Set(list.map((c) => c.owner).filter(Boolean))].map(player).filter(Boolean);
    const seg = (k, l, count) => `<button data-action="sv-f" data-f="kind" data-v="${k}" aria-pressed="${F.kind === k}">${count != null ? `<span class="ln-count ln-num">${count}</span>` : ""}${l}</button>`;
    const sel = (f, label, opts) => `<label class="sv-sel"><span class="ln-sr">${label}</span><select class="ln-input" data-svf="${f}" aria-label="${label}">${opts.map(([v, l]) => `<option value="${esc(v)}"${F[f] === v ? " selected" : ""}>${esc(l)}</option>`).join("")}</select></label>`;
    const active = F.q.trim() || F.kind || F.who || F.when;
    return `<div class="sv-bar">
      <label class="sv-search">${ph("magnifying-glass")}<input class="ln-input" id="sv-q" type="search" placeholder="Search a game or a copy's name" value="${esc(F.q)}" autocomplete="off" spellcheck="false" aria-label="Search the copies"></label>
      <div class="ln-segmented sv-kind-seg" role="group" aria-label="Kind">${seg("", "All", list.length)}${SV_KINDS.map(([k, , t]) => seg(k, t.replace("Copy before a swap", "Before a swap").replace("Safety net copy", "Safety net").replace("Save point", "Save points"), n(k))).join("")}</div>
      <div class="sv-sels">${owners.length ? sel("who", "Profile", [["", "Every profile"], ...owners.map((p) => [p.id, p.name])]) : ""}${sel("when", "When", SV_WHEN)}${sel("sort", "Order", SV_ORDER)}
        ${active ? `<button class="ln-button ln-button--ghost ln-button--sm" data-action="sv-clear">${ph("x-circle")}Clear</button>` : ""}</div></div>`;
  }
  function svResults(list) {
    const shown = svFilter(list), F = svF();
    if (!shown.length) return `<p class="upd-empty">${F.q.trim() ? `No copy of a game or with a name like "${esc(F.q.trim())}"` : "No copies like that"}.</p>`;
    const filtered = svFiltered(list) !== null;
    return `${filtered ? `<p class="sv-count">${shown.length === list.length ? `All ${list.length} copies` : `${shown.length} of ${list.length} copies`}</p>` : ""}${svRecent(shown)}`;
  }
  function svRecent(list) {
    const shown = S.svMore ? list : list.slice(0, SV_SHOWN);
    const rows = shown.map((c) => {
      const it = itemByKey(c.key);
      if (!it) return "";
      const p = player(c.owner), when = svWhen(c.at);
      const busy = c.kind === "point" ? (S.points[c.key] || {}).busy : S.svBusy === c.id;
      const [attrs, label] = c.kind === "net" ? [`data-action="gs-restore" data-local="1" data-key="${esc(c.key)}" data-id="${esc(c.id)}" data-when="${esc(c.at)}"`, "Restore"]
        : c.kind === "point" ? [`data-action="pt-restore" ${itemAttr(it)} data-id="${esc(c.id)}" data-name="${esc(c.name)}"`, "Go back to it"]
        : p ? [`data-action="copy-put-back" data-game="${esc(it.id)}" data-id="${esc(c.id)}" data-owner="${esc(c.owner)}" data-when="${esc(when)}"`, "Put back"] : [];
      const k = svKey(c), open = S.svOpenRow === k, F = S.svFiles[k];
      const naming = S.svNaming === k;
      const title = naming ? `<input class="ln-input sv-name-in" id="sv-name-in" type="text" maxlength="60" value="${esc(c.name || "")}" placeholder="Name this copy, such as Before the boss" data-k="${esc(k)}" aria-label="A name for this copy">`
        : `<b class="sv-name">${c.name ? esc(c.name) : SV_KIND[c.kind]}</b>`;
      const files = !open ? "" : !F || F.loading ? `<div class="sv-files">${lnInline("", "Reading the files")}</div>`
        : F.error ? `<div class="sv-files">${lnInline("error", esc(F.error))}</div>`
        : `<div class="sv-files"><p class="sv-files-head">${F.count === 1 ? "1 file" : `${F.count} files`}${F.count > F.files.length ? `, the ${F.files.length} newest shown` : ""}</p>
            <ul>${F.files.map((f) => `<li><span class="sv-file">${ph("file")}<span title="${esc(f.path)}">${esc(f.path)}</span></span><span class="ln-num">${esc(fmtBytes(f.size))}</span><span title="${esc(svWhen(f.mod))}">${esc(ago(f.mod))}</span></li>`).join("")}</ul></div>`;
      return `<li class="sv-row sv-row--${c.kind}${open ? " is-open" : ""}">
        <button class="sv-tog" data-action="sv-files" data-k="${esc(k)}" aria-expanded="${open}" aria-label="${open ? "Hide" : "Show"} the files in this copy" title="${open ? "Hide the files" : "Show the files"}">${svIcon(c.kind)}</button>
        <div class="sv-what">${title}<span>${svOpen(it, esc(it.name))}${c.name ? `<em>${SV_KIND[c.kind]}</em>` : ""}</span></div>
        <span class="sv-who">${p ? `<span class="ln-mark ln-player" style="--player:${esc(p.color)}"></span>${esc(p.name)}` : ""}</span>
        <span class="sv-when" title="${esc(when)}">${esc(ago(c.at))}</span><span class="sv-size ln-num">${esc(fmtBytes(c.size || 0))}</span>
        <span class="sv-act">${naming ? `<button class="ln-button ln-button--secondary ln-button--sm" data-action="sv-name-save" data-k="${esc(k)}">Save</button>` : `<button class="ln-button ln-button--ghost ln-button--sm ln-button--icon" data-action="sv-name" data-k="${esc(k)}" aria-label="Name this copy" title="Name this copy">${ph("pencil-simple")}</button>`}${attrs ? `<button class="ln-button ln-button--secondary ln-button--sm" ${attrs}${busy ? " disabled" : ""}>${label}</button>` : ""}</span>
        ${files}</li>`;
    }).join("");
    const more = list.length > SV_SHOWN ? `<button class="ln-button ln-button--ghost ln-button--sm" data-action="sv-more">${S.svMore ? "Show fewer" : `Show ${list.length - SV_SHOWN} more`}</button>` : "";
    return `<ol class="ln-rows sv-list">${rows}</ol>${more}`;
  }

  function svGames(games) {
    const by = new Map(games.map((g) => [g.key, g]));
    cfg().games.forEach((g) => { if (!by.has("game:" + g.id)) by.set("game:" + g.id, { key: "game:" + g.id, points: 0, size: 0 }); });
    const q = svF().q.trim(), test = q ? nameMatcher(q) : null;
    const rows = [...by.values()].map((r) => ({ r, it: itemByKey(r.key), n: r.key.startsWith("game:") ? ((S.d.profiles || {})[r.key.slice(5)] || []).length : 0 })).filter((x) => x.it && (!test || test(x.it.name)));
    const k = S.svSort || "last", val = (x) => (k === "profiles" ? x.n : k === "last" ? new Date(x.r.last || 0).getTime() : x.r[k] || 0);
    rows.sort((a, b) => (k === "name" ? 0 : val(b) - val(a)) || byName(a.it.name, b.it.name));
    const none = '<span class="sv-none">None</span>';
    const head = SV_COLS.map(([f, l]) => `<th scope="col"${f === k ? ` aria-sort="${f === "name" ? "ascending" : "descending"}"` : ""}><button class="st-sort" data-action="sv-sort" data-col="${f}">${l}</button></th>`).join("");
    return `<div class="st-wrap"><table class="st-table sv-table"><thead><tr>${head}</tr></thead><tbody>${rows.map(({ r, it, n }) => `
      <tr><th scope="row">${svOpen(it, `<b>${esc(it.name)}</b>`)}<span>${esc(sourceLabel(it.source, it.f))}</span></th><td class="ln-num">${n || none}</td>
        <td>${r.last ? `<span title="${esc(svWhen(r.last))}">${esc(ago(r.last))}</span>` : none}</td><td class="ln-num">${r.points || none}</td><td class="ln-num">${r.size ? esc(fmtBytes(r.size)) : none}</td></tr>`).join("")}</tbody></table></div>`;
  }

  function viewSaves() {
    const O = S.svo, look = svLook(), recent = (O && O.recent) || [], games = (O && O.games) || [];
    const sec = (title, n, body) => labelSec(title, n, body, "sv-sec");
    const points = games.reduce((n, g) => n + g.points, 0);
    const empty = O && !recent.length && !cfg().games.length;
    const stats = O && !empty ? sec("Your saves", 0, `<dl class="ln-meta ln-score">${gpStat(cfg().games.length, "Games with profiles")}${gpStat(games.length, "Games with copies")}${gpStat(points, "Save points")}
      <div class="ln-meta__item"><dd class="ln-meta__text">${recent.length ? esc(ago(recent[0].at)) : "None yet"}</dd><dt>Newest copy</dt></div></dl>`) : "";
    const body = !O ? `<div class="upd-skel">${[0, 1, 2].map(() => '<div class="ln-skeleton"></div>').join("")}</div>`
      : empty ? emptyState("No copies yet", "Copies of your saves show up here.", `<button class="ln-button ln-button--secondary" data-action="nav" data-view="library">Open the Library</button>`)
      : sec("Copies", recent.length, recent.length ? `${svBar(recent)}<div id="sv-results">${svResults(recent)}</div>` : `<p class="upd-empty">None yet</p>`);
    const perGame = O && !empty ? sec("Per game", 0, svGames(games)) : "";
    const kinds = O && !empty ? `<ul class="sv-kinds">${SV_KINDS.map(([k, ic, t, line]) => `<li class="sv-kind sv-kind--${k}">${ph(ic)}<div><b>${t}</b><span>${line}</span></div></li>`).join("")}</ul>
      <p class="sv-tip">Click a copy's icon to see its files, or the pencil to name it. Restore, Put back and Go back to it keep a copy of what's there first.</p>` : "";
    return `<div class="saves-page ln-root"><header class="upd-head"><h1>Saves</h1></header>${stats}${body}${look.length ? sec("Needs a look", look.length, `<ul class="ln-rows">${look.join("")}</ul>`) : ""}${kinds}${perGame}${cloudSection()}</div>`;
  }
