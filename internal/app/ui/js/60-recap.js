  // ------------------------------------------------------------ recap --
  // Your year in games: a picture of a profile's year, drawn here from the
  // daily activity and saved in Pictures\Lootnest to share. Nothing is sent.
  const recapYears = (p) => [...new Set(Object.keys(daysOf(p.id)).filter((k) => daySum(daysOf(p.id)[k]) > 0).map((k) => +k.slice(0, 4)))].sort((a, b) => b - a);
  function recapData(p, year) {
    const days = daysOf(p.id), perGame = {}, perMonth = new Array(12).fill(0);
    let total = 0;
    const keys = Object.keys(days).filter((k) => k.startsWith(year + "-") && daySum(days[k]) > 0).sort();
    keys.forEach((k) => {
      const day = days[k];
      total += daySum(day);
      perMonth[+k.slice(5, 7) - 1] += daySum(day);
      Object.entries(day).forEach(([g, sec]) => { perGame[g] = (perGame[g] || 0) + sec; });
    });
    let best = 0, run = 0, prev = null;
    keys.forEach((k) => { const d = new Date(k + "T12:00:00"); run = prev && (d - prev) / 864e5 < 1.5 ? run + 1 : 1; best = Math.max(best, run); prev = d; });
    const top = Object.entries(perGame).sort((a, b) => b[1] - a[1]).slice(0, 5).map(([k, sec]) => { const it = itemByKey(k); return { it, name: it ? it.name : "A game no longer here", sec }; });
    return { total, days: keys.length, games: Object.keys(perGame).length, best, top, perMonth };
  }
  const loadImg = (src) => new Promise((ok) => { if (!src) { ok(null); return; } const i = new Image(); i.onload = () => ok(i); i.onerror = () => ok(null); i.src = src; });
  async function drawRecap(cv, p, year) {
    const R = recapData(p, year);
    const W = 1080, H = 1350, pad = 84;
    cv.width = W; cv.height = H;
    const x = cv.getContext("2d");
    try { await document.fonts.ready; } catch (e) { /* the fallbacks will do */ }
    const C = { bg: "#111217", fg: "#DFE5EB", fg2: "#A7B0BC", line: "rgba(223,229,235,0.14)", red: "#EF233C", me: p.color || "#6C9BD2" };
    const font = (w, px, fam, stretch) => { x.font = `${w} ${px}px ${fam === "t" ? "Bricolage, sans-serif" : "Figtree, sans-serif"}`; try { x.fontStretch = stretch || "normal"; } catch (e) { /* older engines */ } };
    const fit = (t, max) => { if (x.measureText(t).width <= max) return t; while (t.length > 1 && x.measureText(t + "…").width > max) t = t.slice(0, -1); return t + "…"; };
    x.fillStyle = C.bg; x.fillRect(0, 0, W, H);
    // The most played game's art, faded behind the year.
    const lead = R.top[0] && R.top[0].it;
    if (lead) { ensureCover(lead); loadArt(lead); }
    const art = lead ? await loadImg(artOf(lead) || S.covers[coverKey(lead)]) : null;
    if (art) {
      const h = 640, sc = Math.max(W / art.width, h / art.height);
      x.globalAlpha = 0.38; x.drawImage(art, (W - art.width * sc) / 2, (h - art.height * sc) / 3, art.width * sc, art.height * sc); x.globalAlpha = 1;
      const g = x.createLinearGradient(0, 0, 0, h); g.addColorStop(0, "rgba(17,18,23,0.35)"); g.addColorStop(1, C.bg);
      x.fillStyle = g; x.fillRect(0, 0, W, h);
    }
    // The mark and the name.
    x.strokeStyle = C.fg; x.lineWidth = 3.2; x.lineCap = "round";
    x.beginPath(); x.arc(pad + 18, 98, 15, 0.08 * Math.PI, 0.92 * Math.PI); x.stroke();
    x.beginPath(); x.arc(pad + 18, 98, 8.5, 0.08 * Math.PI, 0.92 * Math.PI); x.stroke();
    x.strokeStyle = C.red; x.beginPath(); x.arc(pad + 18, 86, 5, 0, 2 * Math.PI); x.stroke();
    font(700, 30, "u"); x.fillStyle = C.fg; x.textBaseline = "middle"; x.fillText("Lootnest", pad + 46, 92);
    // The year, big.
    x.textBaseline = "alphabetic";
    font(720, 260, "t", "condensed"); x.fillStyle = C.fg; x.fillText(String(year), pad - 8, 390);
    font(600, 40, "u"); x.fillStyle = C.fg2; x.fillText(fit(`${possessive(p.name)} year in games`, W - pad * 2), pad, 452);
    // Four numbers.
    const stats = [[Math.round(R.total / 3600), "hours"], [R.days, R.days === 1 ? "day played" : "days played"], [R.games, R.games === 1 ? "game" : "games"], [R.best, R.best === 1 ? "day in a row" : "days in a row"]];
    const sw = (W - pad * 2) / 4;
    stats.forEach(([v, l], i) => {
      const sx = pad + i * sw;
      x.fillStyle = C.line; x.fillRect(sx, 506, sw - 24, 2);
      font(720, 92, "t", "condensed"); x.fillStyle = i === 0 ? C.me : C.fg; x.fillText(String(v), sx, 604);
      font(500, 26, "u"); x.fillStyle = C.fg2; x.fillText(l, sx, 642);
    });
    // The most played, with its cover, then the next four as bars.
    let y = 706;
    if (R.top[0]) {
      const cov = await loadImg(lead && S.covers[coverKey(lead)]);
      if (cov) { const cw = 150, ch = 225; x.save(); x.beginPath(); x.roundRect ? x.roundRect(pad, y, cw, ch, 6) : x.rect(pad, y, cw, ch); x.clip(); x.drawImage(cov, pad, y, cw, ch); x.restore(); }
      const tx = cov ? pad + 186 : pad;
      font(600, 26, "u"); x.fillStyle = C.fg2; x.fillText("Most played", tx, y + 44);
      font(720, 64, "t", "condensed"); x.fillStyle = C.fg; x.fillText(fit(R.top[0].name, W - pad - tx), tx, y + 118);
      font(600, 30, "u"); x.fillStyle = C.me; x.fillText(hours(R.top[0].sec), tx, y + 166);
      y += 262;
    }
    const max = R.top[0] ? R.top[0].sec : 1;
    R.top.slice(1, 5).forEach((t, i) => {
      const ry = y + i * 52;
      font(600, 28, "u"); x.fillStyle = C.fg; x.fillText(fit(t.name, 430), pad, ry + 30);
      const bx = pad + 460, bw = W - pad - bx - 120;
      x.fillStyle = C.line; x.fillRect(bx, ry + 14, bw, 10);
      x.fillStyle = C.me; x.fillRect(bx, ry + 14, Math.max(6, bw * (t.sec / max)), 10);
      font(600, 26, "u"); x.fillStyle = C.fg2; x.textAlign = "right"; x.fillText(hours(t.sec), W - pad, ry + 30); x.textAlign = "left";
    });
    // The months, along the bottom.
    const my = H - 132, mh = 56, mw = (W - pad * 2) / 12, mmax = Math.max(...R.perMonth, 1);
    "JFMAMJJASOND".split("").forEach((m, i) => {
      const v = R.perMonth[i], bh = v ? Math.max(4, mh * (v / mmax)) : 2;
      x.fillStyle = v ? C.me : C.line; x.fillRect(pad + i * mw, my + mh - bh, mw - 14, bh);
      font(600, 22, "u"); x.fillStyle = C.fg2; x.fillText(m, pad + i * mw, my + mh + 32);
    });
    font(500, 22, "u"); x.fillStyle = C.fg2; x.textAlign = "right"; x.textBaseline = "middle"; x.fillText("Made on this PC", W - pad, 92); x.textAlign = "left"; x.textBaseline = "alphabetic";
    return R;
  }
  function viewRecap(D) {
    const p = player(D.player);
    if (!p) return "";
    const years = recapYears(p);
    const yearSel = years.length > 1 ? `<select class="ln-input recap-year" id="recap-year" aria-label="Year">${years.map((y) => `<option${y === D.year ? " selected" : ""}>${y}</option>`).join("")}</select>` : "";
    const body = years.length ? `<canvas class="recap-canvas" id="recap-canvas" width="1080" height="1350" role="img" aria-label="${esc(possessive(p.name))} ${D.year} in games"></canvas>`
      : `<p class="sub">Nothing played yet. Lootnest counts the time from the first game you start through it.</p>`;
    return `<div class="scrim" data-action="scrim"><div class="dialog recap-dialog" role="dialog" aria-modal="true" aria-label="Year in games">
      <div class="recap-head"><h2>${esc(p.name)} in ${D.year}</h2>${yearSel}</div>${body}
      <div class="dialog-foot"><span class="hint grow">Saved in Pictures\\Lootnest, to share as you like. Nothing is sent.</span><button class="btn quiet" data-action="dialog-cancel">Close</button>
      ${years.length ? `<button class="btn primary" data-action="recap-save" ${D.busy ? "disabled" : ""}>${D.busy ? "Saving the picture" : "Save the picture"}</button>` : ""}</div></div></div>`;
  }
  function ppSection(s, p, st) {
    const sec = (title, inner, extra) => `<section class="pp-section pp-${s}"><div class="ln-section-head"><div><h2>${title}</h2></div>${extra || ""}</div>${inner}</section>`;
    if (s === "now") {
      const it = p.now && itemByKey(p.now);
      if (!it) return "";
      loadArt(it); ensureCover(it);
      const t = (playtimeOf(it) || {})[p.id] || 0;
      return `<section class="pp-section pp-now"><button class="now-card" data-action="open-sheet" ${itemAttr(it)}>
        ${artHTML(it)}<span class="now-text"><span class="now-label">Currently playing</span><b>${esc(it.name)}</b>
        ${p.nowNote ? `<span class="now-note">${esc(p.nowNote)}</span>` : ""}
        <span class="now-meta">${t ? `<span>${esc(hours(t))} played</span>` : ""}</span></span></button></section>`;
    }
    if (s === "stats") {
      const stat = (l, v) => `<div><dt>${l}</dt><dd>${v}</dd></div>`, lg = st.longest && itemByKey(st.longest.key);
      return sec("Numbers", `<dl class="pp-facts">
        ${stat("Played with Lootnest", st.total ? esc(hours(st.total)) : "None yet")}
        ${stat("Games played", st.played)}
        ${stat("Games with their own saves", st.profiles)}
        ${stat("Days in a row", (st.streak || "None") + (st.best > st.streak ? `, best ${st.best}` : ""))}
        ${stat("Longest session", st.longest ? esc(hours(st.longest.seconds)) + (lg ? ", " + esc(lg.name) : "") : "None yet")}
        ${stat("Most played", st.top ? `${esc(st.top.name)}, ${esc(hours(st.topT))}` : "None yet")}</dl>`);
    }
    if (s === "favorites") {
      const favs = (p.favorites || []).map(itemByKey).filter(Boolean);
      favs.forEach(ensureCover);
      return favs.length ? sec("Favourite games", `<div class="lib-grid size-l">${favs.map((it) => cardHTML(it)).join("")}</div>`) : "";
    }
    if (s === "activity") return sec("Activity", activityHTML(p, st));
    if (s === "collection") {
      const items = libraryItems(), by = {};
      items.forEach((it) => {
        const t = (playtimeOf(it) || {})[p.id] || 0;
        if (!t && !st.recent.some((x) => x.it === it)) return;
        const l = sourceLabel(it.source);
        by[l] = by[l] || { n: 0, t: 0 }; by[l].n++; by[l].t += t;
      });
      const rows = Object.entries(by).sort((a, b) => b[1].n - a[1].n || b[1].t - a[1].t);
      const max = Math.max(1, ...rows.map((r) => r[1].n));
      return rows.length ? sec("Collection", `<ul class="bars">${rows.map(([l, v]) => `<li><span class="bl">${esc(l)}</span>
          <span class="bt"><span style="width:${Math.max(4, (v.n / max) * 100)}%"></span></span><span class="bv">${v.n} ${v.n === 1 ? "game" : "games"}${v.t ? ", " + esc(hours(v.t)) : ""}</span></li>`).join("")}</ul>
          <p class="hint">${items.length} games in the Library on this PC, from ${new Set(items.map((it) => sourceLabel(it.source))).size} places.</p>`) : "";
    }
    if (s === "recent") {
      if (!st.recent.length) return "";
      return sec("Played lately", `<ul class="pp-recent">${st.recent.slice(0, 8).map(({ it, r }) => `<li><button data-action="open-sheet" ${itemAttr(it)}><b>${esc(it.name)}${starsHTML(shelfOf(it, p.id).rating)}</b><span>${esc(ago(r.at))}</span></button></li>`).join("")}</ul>`);
    }
    return "";
  }

  // 26 weeks of play, a column a week, a square a day, darker for more.
  function activityHTML(p, st) {
    const days = daysOf(p.id), today = new Date();
    const start = new Date(today); start.setHours(12, 0, 0, 0);
    start.setDate(start.getDate() - ((start.getDay() + 6) % 7) - 25 * 7);
    const lvl = (s) => (s <= 0 ? 0 : s < 1800 ? 1 : s < 3600 ? 2 : s < 7200 ? 3 : 4);
    let cells = "", week = 0, month = 0, months = "";
    const tKey = dayKey(today), byGame = {};
    for (let i = 0, d = new Date(start); i < 26 * 7; i++, d.setDate(d.getDate() + 1)) {
      const k = dayKey(d), sum = daySum(days[k]), future = k > tKey;
      if (i % 7 === 0 && d.getDate() <= 7) months += `<span style="grid-column:${i / 7 + 1}">${d.toLocaleDateString(undefined, { month: "short" })}</span>`;
      cells += `<span class="hm l${future ? "x" : lvl(sum)}" style="grid-column:${Math.floor(i / 7) + 1};grid-row:${(i % 7) + 1}" title="${esc(d.toLocaleDateString(undefined, { weekday: "short", day: "numeric", month: "short" }))}${sum ? ": " + esc(hours(sum)) : ""}"></span>`;
    }
    for (let i = 0, d = new Date(today); i < 30; i++, d.setDate(d.getDate() - 1)) {
      const day = days[dayKey(d)] || {};
      if (i < 7) week += daySum(day);
      month += daySum(day);
      Object.entries(day).forEach(([k, v]) => { byGame[k] = (byGame[k] || 0) + v; });
    }
    const top = Object.entries(byGame).map(([k, v]) => ({ it: itemByKey(k), v })).filter((x) => x.it).sort((a, b) => b.v - a.v).slice(0, 4);
    const max = Math.max(1, ...top.map((x) => x.v));
    return `<div class="activity">
      <div class="heat-wrap"><div class="heat-months">${months}</div><div class="heat" aria-label="Time played per day">${cells}</div>
        <div class="heat-key"><span>Less</span><span class="hm l0"></span><span class="hm l1"></span><span class="hm l2"></span><span class="hm l3"></span><span class="hm l4"></span><span>More</span></div></div>
      <div class="act-side"><div class="act-n"><div><b>${week ? esc(hours(week)) : "0 min"}</b><span>the last 7 days</span></div><div><b>${month ? esc(hours(month)) : "0 min"}</b><span>the last 30 days</span></div><div><b>${st.sessions}</b><span>${st.sessions === 1 ? "session" : "sessions"} in all</span></div></div>
        ${top.length ? `<ul class="bars">${top.map(({ it, v }) => `<li><span class="bl">${esc(it.name)}</span><span class="bt"><span style="width:${Math.max(4, (v / max) * 100)}%"></span></span><span class="bv">${esc(hours(v))}</span></li>`).join("")}</ul>`
          : `<p class="hint">Games started from Lootnest are counted here, day by day.</p>`}</div></div>`;
  }

