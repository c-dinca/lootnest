  // ---------------------------------------------------------- side by side --
  // Profiles side by side, the Profiles page's second tab: who's chosen
  // (S.cmpWho, up to CMP_MAX), a board of each one's numbers with the
  // leader of each marked, then every game any of them played: each one's
  // time (a bar in their colour), status and achievements, and who leads.
  // Only what Lootnest already knows; nothing is read for it but the
  // achievements the Achievements page reads.
  const CMP_MAX = 6;
  function cmpWho() {
    const ps = cfg().players, ids = (S.cmpWho || []).filter((id) => player(id));
    if (ids.length >= 2) return ids.map(player);
    const people = ps.filter((p) => !parentOf(p));
    return (people.length >= 2 ? people : ps).slice(0, CMP_MAX);
  }
  // Each one's achievements in a game, when they go with the profile.
  function cmpAch(key) {
    const g = S.allAch && S.allAch.games.find((x) => x.key === key);
    return g && g.players ? g : null;
  }
  // Every game at least one of them played or has saves in, with each
  // one's time, status, last play and achievements.
  function cmpGames(who) {
    return everyItem().map((it) => {
      const key = recentKey(it), pt = playtimeOf(it), A = cmpAch(key);
      const per = who.map((p) => ({ p, t: pt[p.id] || 0, last: lastBy(it, p.id), status: shelfAt(p.id, key).status || "",
        saves: it.kind === "game" && hasProfile(it.id, p.id), ach: A && A.players[p.id] ? A.players[p.id].unlocked : null }));
      return { it, key, per, total: A ? A.total : 0, sum: per.reduce((n, x) => n + x.t, 0), by: per.filter((x) => x.t || x.last || x.saves || x.status).length };
    }).filter((r) => r.by > 0);
  }
  // Who's ahead in a game: the most achievements, else the most time.
  function cmpLead(r) {
    const by = (f) => { const top = Math.max(...r.per.map(f)); return top > 0 && r.per.filter((x) => f(x) === top).length === 1 ? r.per.find((x) => f(x) === top).p : null; };
    return r.total ? by((x) => x.ach || 0) || by((x) => x.t) : by((x) => x.t);
  }
  const CMP_SHOW = [["all", "Every game"], ["shared", "In common"], ["alone", "Only one"]];
  function viewCompare() {
    if (!S.allAch && !cmpAsked) { cmpAsked = true; api.allAchievements().catch(() => {}).finally(() => { cmpAsked = false; }); }
    const all = cfg().players, who = cmpWho(), ids = who.map((p) => p.id);
    const chips = `<div class="cmp-who" role="group" aria-label="Who's compared">${grouped(all).map((p) => {
      const on = ids.includes(p.id), full = !on && ids.length >= CMP_MAX;
      return `<button class="ln-chip ln-player${parentOf(p) ? " is-run" : ""}" style="--player:${esc(p.color)}" data-action="cmp-who" data-player="${esc(p.id)}" aria-pressed="${on}"${full ? " disabled" : ""}>${lnAvatar(p, "20")}${esc(parentOf(p) ? `${parentOf(p).name}, ${runLabel(p)}` : p.name)}</button>`;
    }).join("")}</div>`;
    const rows = cmpGames(who);
    // The board: one column each, a line per number, its leader lit.
    const stats = who.map((p) => {
      const st = playerStats(p), mine = rows.filter((r) => r.per.find((x) => x.p === p).t || r.per.find((x) => x.p === p).last);
      const done = rows.filter((r) => ["finished", "complete"].includes(r.per.find((x) => x.p === p).status)).length;
      const ach = rows.reduce((n, r) => n + (r.per.find((x) => x.p === p).ach || 0), 0);
      const leads = rows.filter((r) => r.by > 1 && cmpLead(r) === p).length;
      return { p, total: st.total, games: mine.length, done, ach, leads, longest: st.longest ? st.longest.seconds : 0, streak: st.best };
    });
    const LINES = [["total", "Time played", (v) => (v ? hours(v) : "0 h")], ["games", "Games played", String], ["done", "Finished or 100%", String],
      ["ach", "Achievements unlocked", (v) => v.toLocaleString("en-US")], ["leads", "Games where they lead", String],
      ["longest", "Longest sitting", (v) => (v ? hours(v) : "None")], ["streak", "Most days in a row", (v) => (v === 1 ? "1 day" : v + " days")]];
    const board = `<div class="cmp-board" style="--cols:${who.length}" role="table" aria-label="Side by side">
      <div class="cmp-line cmp-line--head" role="row"><span role="columnheader"></span>${stats.map((s) => `<span class="cmp-head ln-player" style="--player:${esc(s.p.color)}" role="columnheader">${lnAvatar(s.p, "40")}<b>${esc(s.p.name)}</b>${parentOf(s.p) ? `<span>${esc(possessive(parentOf(s.p).name))} run</span>` : ""}</span>`).join("")}</div>
      ${LINES.map(([k, label, fmt]) => {
        const top = Math.max(...stats.map((s) => s[k])), one = top > 0 && stats.filter((s) => s[k] === top).length === 1;
        return `<div class="cmp-line" role="row"><span class="cmp-what" role="rowheader">${label}</span>${stats.map((s) => {
          const lead = one && s[k] === top;
          return `<span class="cmp-val ln-player${lead ? " is-lead" : ""}" style="--player:${esc(s.p.color)}" role="cell">${lead ? `<span class="cmp-crown" title="Leads">${ph("crown-simple", "fill")}</span>` : ""}<b class="ln-num">${esc(fmt(s[k]))}</b></span>`;
        }).join("")}</div>`;
      }).join("")}</div>`;
    // Every game: the filter, then a row each.
    const show = S.cmpShow || "all";
    const list = rows.filter((r) => show === "all" || (show === "shared" ? r.by > 1 : r.by === 1))
      .sort((a, b) => b.sum - a.sum || byName(a.it.name, b.it.name));
    const games = list.map((r) => {
      ensureCover(r.it);
      const c = artFor(r.it, "cover").url, top = Math.max(...r.per.map((x) => x.t)) || 1, lead = r.by > 1 ? cmpLead(r) : null;
      return `<div class="cmp-game" style="--cols:${who.length}" role="row">
        <button class="cmp-title" data-action="open-sheet" ${itemAttr(r.it)} role="rowheader">${c ? `<img src="${c}" alt="" loading="lazy">` : `<span class="cmp-plate" aria-hidden="true">${esc(initial(r.it.name))}</span>`}<span><b>${esc(r.it.name)}</b>${lead ? `<span class="cmp-leader ln-player" style="--player:${esc(lead.color)}">${ph("crown-simple", "fill")}${esc(lead.name)} leads</span>` : ""}</span></button>
        ${r.per.map((x) => {
          const said = [x.status ? statusName(x.status) : "", x.ach != null && r.total ? `${x.ach} of ${r.total}` : ""].filter(Boolean);
          if (!x.t && !x.last && !x.saves && !x.status) return `<span class="cmp-cell is-none" role="cell"><span class="gpage-hint">Not played</span></span>`;
          return `<span class="cmp-cell ln-player${lead === x.p ? " is-lead" : ""}" style="--player:${esc(x.p.color)}" role="cell">
            <span class="cmp-time"><b class="ln-num">${x.t ? esc(hours(x.t)) : "0 h"}</b>${x.last ? `<span>${esc(ago(x.last))}</span>` : ""}</span>
            <span class="cmp-bar" aria-hidden="true"><i style="width:${((x.t / top) * 100).toFixed(1)}%"></i></span>
            ${said.length ? `<span class="cmp-said">${said.map((t, i) => `<span${i === 0 && x.status ? ` class="cmp-st cmp-st--${esc(x.status)}"` : ""}>${esc(t)}</span>`).join("")}</span>` : ""}</span>`;
        }).join("")}</div>`;
    }).join("");
    const seg = `<div class="ln-segmented" role="group" aria-label="Show">${CMP_SHOW.map(([k, l]) => `<button data-action="cmp-show" data-k="${k}" aria-pressed="${k === show}">${l}</button>`).join("")}</div>`;
    return `<div class="cmp">${chips}${board}
      <section class="cmp-games"><div class="ln-section-head"><div><h2>Game by game</h2></div>${seg}</div>
        ${list.length ? `<div class="cmp-list" role="table" aria-label="Game by game">${games}</div>` : `<p class="gpage-hint">${show === "shared" ? "No game played by more than one of them yet." : "Nothing played yet."}</p>`}
        ${S.allAch ? "" : '<p class="gpage-hint">Reading achievements.</p>'}</section></div>`;
  }
  let cmpAsked = false;
