  // ----------------------------------------------------------------- saves --
  // Saves: everything Lootnest keeps of every game's saves in one place.
  // What needs a look, the newest copies (safety net, save points, copies
  // before swaps) from ps_savesOverview, each game, and Cloud saves. Every
  // button here is one the game's page or Cloud saves already has.
  const SV_SHOWN = 8;
  const SV_KIND = { net: "Copy of its saves", point: "Save point", swap: "Copy before a swap" };
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
    const card = (title, text, act) => rows.push(`<div class="copy-card copy-card--warn">${GI.alert}<div><b>${title}</b><p>${text}</p></div><div class="gpage-actions">${act}</div></div>`);
    const btn = (attrs, label, kind = "secondary") => `<button class="ln-button ln-button--${kind} ln-button--sm" ${attrs}>${label}</button>`;
    (C.gone || []).forEach((g) => { const it = itemByKey(g.key); if (it) card(`${esc(possessive(it.name))} saves are gone from its folder`, "Lootnest kept copies of them. Put one back from the game's Saves tab.", svOpen(it, "Open its saves", "ln-button ln-button--secondary ln-button--sm")); });
    Object.entries(S.d.conflicts || {}).forEach(([gid, c]) => {
      const g = game(gid), who = pname(c.player) || "Someone";
      if (g) card(`Steam Cloud replaced ${esc(possessive(who))} files in ${esc(g.name)}`, `On ${esc(svWhen(c.at))}. ${esc(who)}'s profile stayed as it was, and that session's files are kept as a copy.`,
        btn(`data-action="copies" data-game="${esc(gid)}" aria-haspopup="dialog"`, "See the copies") + btn(`data-action="conflict-ok" data-game="${esc(gid)}"`, "Got it", "ghost"));
    });
    const two = V && !V.error ? [...new Set((V.games || []).filter((x) => x.state === "conflict" && game(x.gameId) && player(x.playerId)).map((x) => x.gameId))] : [];
    if (two.length) card(two.length === 1 ? `${esc(possessive(game(two[0]).name))} saves changed on two PCs` : `${two.length} games' saves changed on two PCs`, "Choose which to keep under Cloud saves.", btn('data-action="nav" data-view="cloud"', "Choose"));
    const st = B && B.dir && B.status, failed = st && st.error && (!st.last || new Date(st.errorAt) > new Date(st.last));
    const err = (V && V.error) || (failed && st.error);
    if (err) card("Cloud saves didn't work", esc(err), btn('data-action="nav" data-view="cloud"', "See Cloud saves"));
    const unk = C.unknown || [];
    if (C.unknownN) card(C.unknownN === 1 ? `Lootnest doesn't know where ${esc(unk[0] ? unk[0].name : "a game")} keeps its saves` : `Lootnest doesn't know where ${C.unknownN} games keep their saves`,
      "Without that, it can't keep copies of them. Choose the folder on the game's Saves tab.",
      unk.length === 1 && itemByKey(unk[0].key) ? svOpen(itemByKey(unk[0].key), "Choose the folder", "ln-button ln-button--secondary ln-button--sm") : btn('data-action="sc-games" data-kind="unknown" aria-haspopup="menu"', "Which games"));
    return rows;
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
      return `<li class="sv-row"><div class="sv-what">${svOpen(it, esc(it.name))}<span>${c.kind === "point" ? `${SV_KIND.point}: ${esc(c.name)}` : SV_KIND[c.kind]}</span></div>
        <span class="sv-who">${p ? `<span class="ln-mark ln-player" style="--player:${esc(p.color)}"></span>${esc(p.name)}` : ""}</span>
        <span class="sv-when" title="${esc(when)}">${esc(ago(c.at))}</span><span class="sv-size ln-num">${esc(fmtBytes(c.size || 0))}</span>
        <span class="sv-act">${attrs ? `<button class="ln-button ln-button--ghost ln-button--sm" ${attrs}${busy ? " disabled" : ""}>${label}</button>` : ""}</span></li>`;
    }).join("");
    const more = list.length > SV_SHOWN ? `<button class="ln-button ln-button--ghost ln-button--sm" data-action="sv-more">${S.svMore ? "Show fewer" : `Show ${list.length - SV_SHOWN} more`}</button>` : "";
    return `<ol class="sv-list">${rows}</ol>${more}`;
  }

  function svGames(games) {
    const by = new Map(games.map((g) => [g.key, g]));
    cfg().games.forEach((g) => { if (!by.has("game:" + g.id)) by.set("game:" + g.id, { key: "game:" + g.id, points: 0, size: 0 }); });
    const rows = [...by.values()].map((r) => ({ r, it: itemByKey(r.key), n: r.key.startsWith("game:") ? ((S.d.profiles || {})[r.key.slice(5)] || []).length : 0 })).filter((x) => x.it);
    const k = S.svSort || "last", val = (x) => (k === "profiles" ? x.n : k === "last" ? new Date(x.r.last || 0).getTime() : x.r[k] || 0);
    rows.sort((a, b) => (k === "name" ? 0 : val(b) - val(a)) || a.it.name.localeCompare(b.it.name));
    const none = '<span class="sv-none">None</span>';
    const head = SV_COLS.map(([f, l]) => `<th scope="col"${f === k ? ` aria-sort="${f === "name" ? "ascending" : "descending"}"` : ""}><button class="st-sort" data-action="sv-sort" data-col="${f}">${l}</button></th>`).join("");
    return `<div class="st-wrap"><table class="st-table sv-table"><thead><tr>${head}</tr></thead><tbody>${rows.map(({ r, it, n }) => `
      <tr><th scope="row">${svOpen(it, `<b>${esc(it.name)}</b>`)}<span>${esc(sourceLabel(it.source, it.f))}</span></th><td class="ln-num">${n || none}</td>
        <td>${r.last ? `<span title="${esc(svWhen(r.last))}">${esc(ago(r.last))}</span>` : none}</td><td class="ln-num">${r.points || none}</td><td class="ln-num">${r.size ? esc(fmtBytes(r.size)) : none}</td></tr>`).join("")}</tbody></table></div>`;
  }

  function viewSaves() {
    const O = S.svo, look = svLook(), recent = (O && O.recent) || [], games = (O && O.games) || [];
    const sec = (id, title, n, body) => `<section class="sv-sec" aria-labelledby="${id}"><div class="ln-section-head"><div><h2 id="${id}">${title}${n ? `<span class="ln-count ln-num">${n}</span>` : ""}</h2></div></div>${body}</section>`;
    const points = games.reduce((n, g) => n + g.points, 0);
    const empty = O && !recent.length && !cfg().games.length;
    const stats = O && !empty ? `<dl class="ln-meta sv-stats">${gpStat(cfg().games.length, "Games with profiles")}${gpStat(games.length, "Games with copies")}${gpStat(points, "Save points")}
      <div class="ln-meta__item"><dd class="ln-meta__text">${recent.length ? esc(ago(recent[0].at)) : "None yet"}</dd><dt>Newest copy</dt></div></dl>` : "";
    const head = `<header class="upd-head"><div><h1>Saves</h1><p>Lootnest keeps a copy before every swap, copies the saves of games outside the stores when they change, and keeps the save points you make. All of it stays on this PC unless Cloud saves is on.</p>${stats}</div></header>`;
    const body = !O ? `<div class="upd-skel">${[0, 1, 2].map(() => '<div class="ln-skeleton"></div>').join("")}</div>`
      : empty ? emptyState(GI.save, "No copies yet", "Turn on profiles for a game, play a game from outside the stores, or make a save point, and its copies show up here.", `<button class="ln-button ln-button--secondary" data-action="nav" data-view="library">Open the Library</button>`)
      : sec("sv-recent-h", "Recent copies", 0, recent.length ? svRecent(recent) : `<p class="upd-empty">No copies yet. Lootnest makes one before the next swap.</p>`);
    const perGame = O && !empty ? sec("sv-games-h", "Per game", 0, svGames(games)) : "";
    return `<div class="saves-page ln-root">${head}${look.length ? sec("sv-look-h", "Needs a look", look.length, look.join("")) : ""}${body}${perGame}${cloudSection()}</div>`;
  }
