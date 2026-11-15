  // ------------------------------------------------------- share achievements --
  // A picture of a profile's achievements, all of them or one game's, drawn
  // here to paste into Discord (Copy) or keep (Save, in Pictures\Lootnest\
  // Achievements). Square for chats, tall for a phone's story. Nothing is
  // sent anywhere.
  const SHARE_FMT = { square: [1080, 1080], phone: [1080, 1920] };
  // A profile's games with achievements, as the Achievements page counts them.
  function achRowsFor(who) {
    if (!S.allAch) return [];
    return S.allAch.games.map((g) => {
      const own = g.players ? g.players[who] : null;
      if (g.players && !own) return null;
      const it = itemByKey(g.key);
      return it ? { g, it, unlocked: own ? own.unlocked : g.unlocked, recent: (own ? own.recent : g.recent) || [], rarest: (own ? own.rarest : g.rarest) || [] } : null;
    }).filter(Boolean);
  }
  function openAchShare(who, scope) {
    who = who || (seated() || cfg().players[0] || {}).id;
    // From a game's page the whole list may not be read yet; it's drawn when it comes.
    if (!S.allAch) api.allAchievements().catch(() => {});
    S.dialog = { kind: "achshare", who, scope, fmt: "square" };
    renderOverlay(); focusOverlay();
  }
  function viewAchShare(D) {
    const p = player(D.who), rows = achRowsFor(D.who), r = D.scope !== "all" && rows.find((x) => x.g.key === D.scope);
    const title = r ? r.g.name : `${p ? possessive(p.name) : "Your"} achievements`;
    const [w, h] = SHARE_FMT[D.fmt];
    const fmt = (k, l, ic) => `<button data-action="share-fmt" data-v="${k}" aria-pressed="${D.fmt === k}">${ph(ic)}${l}</button>`;
    return `<div class="scrim" data-action="scrim"><div class="dialog share-dialog" role="dialog" aria-modal="true" aria-label="Share ${esc(title)}">
      <div class="share-head"><div><h2>Share</h2><p class="sub">${esc(title)}</p></div>
        <div class="ln-segmented" role="group" aria-label="Size">${fmt("square", "Square", "image-square")}${fmt("phone", "Phone", "monitor")}</div></div>
      <div class="share-stage share-stage--${D.fmt}"><canvas id="share-canvas" width="${w}" height="${h}" role="img" aria-label="${esc(title)}"></canvas></div>
      <div class="dialog-foot"><span class="hint grow">Copy, then paste it in Discord or a chat. Saved ones go to Pictures\\Lootnest\\Achievements.</span>
        <button class="btn quiet" data-action="dialog-cancel">Close</button>
        <button class="btn" data-action="share-go" data-how="save" ${D.busy ? "disabled" : ""}>${ph("download-simple")}Save</button>
        <button class="btn primary" data-action="share-go" data-how="copy" ${D.busy ? "disabled" : ""}>${ph("copy")}${D.copied ? "Copied" : "Copy image"}</button></div></div></div>`;
  }
  const loadShareImg = (src) => new Promise((ok) => {
    if (!src) { ok(null); return; }
    const i = new Image();
    // Steam's servers send CORS headers; without them the picture is left out
    // rather than let it lock the canvas.
    if (/^https:/.test(src)) i.crossOrigin = "anonymous";
    i.onload = () => ok(i); i.onerror = () => ok(null); i.src = src;
  });
  async function drawAchShare(cv, D) {
    const [W, H] = SHARE_FMT[D.fmt], tall = H > W, pad = 80;
    cv.width = W; cv.height = H;
    const x = cv.getContext("2d");
    try { await document.fonts.ready; } catch (e) { /* the fallbacks will do */ }
    const css = getComputedStyle(document.documentElement), tok = (n, f) => css.getPropertyValue(n).trim() || f;
    const p = player(D.who), me = (p && p.color) || "#6C9BD2";
    const C = { bg: "#111217", panel: "rgba(255,255,255,0.05)", fg: "#DFE5EB", fg2: "#A7B0BC", line: "rgba(223,229,235,0.14)", red: "#EF233C", me,
      tiers: { ultra: tok("--tier-ultra", "#f5c451"), rare: tok("--tier-rare", "#7cb6ff"), uncommon: tok("--tier-uncommon", "#7be08f"), common: tok("--tier-common", "#c9d1db") } };
    const font = (w, px, fam) => { x.font = `${w} ${px}px ${fam === "t" ? '"Tilt Warp", sans-serif' : '"Atkinson Next", sans-serif'}`; };
    const fit = (t, max) => { if (x.measureText(t).width <= max) return t; while (t.length > 1 && x.measureText(t + "…").width > max) t = t.slice(0, -1); return t + "…"; };
    const rows = achRowsFor(D.who), one = D.scope !== "all" ? rows.find((r) => r.g.key === D.scope) : null;
    const list = one ? [one] : rows;
    const got = list.reduce((n, r) => n + r.unlocked, 0), all = list.reduce((n, r) => n + r.g.total, 0), pct = all ? Math.round((got / all) * 100) : 0;
    const rarest = list.flatMap((r) => r.rarest.map((a) => ({ a, r }))).sort((m, n) => m.a.percent - n.a.percent);
    x.fillStyle = C.bg; x.fillRect(0, 0, W, H);
    // The game's art, or the most complete game's, faded behind the top.
    const lead = one ? one.it : list.slice().sort((m, n) => n.unlocked / n.g.total - m.unlocked / m.g.total)[0]?.it;
    if (lead) { ensureCover(lead); loadArt(lead); }
    const art = lead ? await loadShareImg(artOf(lead)) : null;
    const top = tall ? 760 : 560;
    if (art) {
      const sc = Math.max(W / art.width, top / art.height);
      x.save(); x.beginPath(); x.rect(0, 0, W, top); x.clip();
      x.globalAlpha = 0.42; x.drawImage(art, (W - art.width * sc) / 2, (top - art.height * sc) / 3, art.width * sc, art.height * sc); x.globalAlpha = 1;
      x.restore();
      const g = x.createLinearGradient(0, 0, 0, top); g.addColorStop(0, "rgba(17,18,23,0.25)"); g.addColorStop(1, C.bg);
      x.fillStyle = g; x.fillRect(0, 0, W, top);
    }
    // The mark.
    x.strokeStyle = C.fg; x.lineWidth = 3.2; x.lineCap = "round";
    x.beginPath(); x.arc(pad + 18, 92, 15, 0.08 * Math.PI, 0.92 * Math.PI); x.stroke();
    x.beginPath(); x.arc(pad + 18, 92, 8.5, 0.08 * Math.PI, 0.92 * Math.PI); x.stroke();
    x.strokeStyle = C.red; x.beginPath(); x.arc(pad + 18, 80, 5, 0, 2 * Math.PI); x.stroke();
    x.textBaseline = "middle"; font(700, 28, "u"); x.fillStyle = C.fg; x.fillText("Lootnest", pad + 46, 86);
    font(600, 26, "u"); x.fillStyle = C.fg2; x.textAlign = "right"; x.fillText(new Date().toLocaleDateString(undefined, { day: "numeric", month: "long", year: "numeric" }), W - pad, 86); x.textAlign = "left";
    // Who, and what.
    x.textBaseline = "alphabetic";
    let y = tall ? 300 : 210;
    font(600, 30, "u"); x.fillStyle = C.me; x.fillText(fit(p ? p.name : "", W - pad * 2), pad, y);
    font(400, one ? 72 : 84, "t"); x.fillStyle = C.fg; x.fillText(fit(one ? one.g.name : "Achievements", W - pad * 2), pad, y + 84);
    // The ring and the numbers.
    y += tall ? 170 : 146;
    const R = tall ? 150 : 104, cx = pad + R, cy = y + R;
    x.lineWidth = tall ? 26 : 22; x.lineCap = "round";
    x.strokeStyle = C.line; x.beginPath(); x.arc(cx, cy, R, 0, Math.PI * 2); x.stroke();
    x.strokeStyle = pct >= 100 ? C.tiers.uncommon : C.me; x.beginPath(); x.arc(cx, cy, R, -Math.PI / 2, -Math.PI / 2 + (Math.PI * 2 * pct) / 100); x.stroke();
    x.textAlign = "center"; x.textBaseline = "middle"; font(400, tall ? 92 : 74, "t"); x.fillStyle = C.fg; x.fillText(pct + "%", cx, cy - 6);
    font(600, 24, "u"); x.fillStyle = C.fg2; x.fillText("complete", cx, cy + (tall ? 58 : 46)); x.textAlign = "left"; x.textBaseline = "alphabetic";
    const sx = cx + R + (tall ? 64 : 88), stats = one
      ? [[got, `of ${all} unlocked`], [rarest[0] ? pctText(rarest[0].a.percent) : "None", "rarest unlocked"]]
      : [[got, `of ${all} unlocked`], [list.length, list.length === 1 ? "game" : "games"], [list.filter((r) => r.unlocked >= r.g.total).length, "at 100%"]];
    const step = tall ? 116 : 84;
    stats.forEach(([v, l], i) => {
      const ty = cy - ((stats.length - 1) * step) / 2 + i * step + 8;
      font(400, tall ? 64 : 50, "t"); x.fillStyle = i === 0 ? C.me : C.fg; x.fillText(String(v), sx, ty);
      font(600, tall ? 24 : 22, "u"); x.fillStyle = C.fg2; x.fillText(l, sx, ty + (tall ? 36 : 30));
    });
    y = cy + R + (tall ? 110 : 70);
    // The rarest, by name, in their tier's colour.
    const tierCol = (a) => C.tiers[tierOf(a.percent)] || C.tiers.common;
    const icon = async (a, ix, iy, s) => {
      x.save(); x.beginPath(); x.roundRect(ix, iy, s, s, 14); x.clip();
      const im = await loadShareImg(a.icon);
      if (im) x.drawImage(im, ix, iy, s, s); else { x.fillStyle = C.panel; x.fillRect(ix, iy, s, s); }
      x.restore();
      x.lineWidth = 4; x.strokeStyle = tierCol(a); x.beginPath(); x.roundRect(ix, iy, s, s, 14); x.stroke();
    };
    const head = (t) => { font(700, 26, "u"); x.fillStyle = C.fg2; x.fillText(t, pad, y); y += 26; };
    if (rarest.length) {
      head("Rarest unlocked");
      const n = Math.min(3, rarest.length), rowH = tall ? 118 : 96, s = rowH - 22;
      for (let i = 0; i < n; i++) {
        const { a, r } = rarest[i], ry = y + i * rowH;
        await icon(a, pad, ry, s);
        font(700, 32, "u"); x.fillStyle = C.fg; x.fillText(fit(a.name, W - pad * 2 - s - 220), pad + s + 24, ry + s / 2 - 4);
        font(500, 24, "u"); x.fillStyle = C.fg2; x.fillText(fit(one ? (a.description || "") : r.g.name, W - pad * 2 - s - 220), pad + s + 24, ry + s / 2 + 30);
        x.textAlign = "right"; font(400, 40, "t"); x.fillStyle = tierCol(a); x.fillText(pctText(a.percent), W - pad, ry + s / 2 + 12); x.textAlign = "left";
      }
      y += n * rowH + (tall ? 50 : 30);
    }
    // The rest: games with bars (all), or more unlocked icons (one game).
    if (y < H - 200) {
      if (!one) {
        head("Most complete");
        const games = list.slice().sort((m, n) => n.unlocked / n.g.total - m.unlocked / m.g.total || n.unlocked - m.unlocked);
        const rowH = 66, n = Math.max(0, Math.min(games.length, Math.floor((H - 130 - y) / rowH)));
        for (let i = 0; i < n; i++) {
          const r = games[i], ry = y + i * rowH + 10, gp = Math.round((r.unlocked / r.g.total) * 100);
          font(700, 28, "u"); x.fillStyle = C.fg; x.fillText(fit(r.g.name, W - pad * 2 - 260), pad, ry + 18);
          x.textAlign = "right"; font(600, 24, "u"); x.fillStyle = C.fg2; x.fillText(`${r.unlocked} of ${r.g.total}`, W - pad, ry + 18); x.textAlign = "left";
          x.fillStyle = C.line; x.beginPath(); x.roundRect(pad, ry + 32, W - pad * 2, 8, 4); x.fill();
          x.fillStyle = gp >= 100 ? C.tiers.uncommon : C.me; x.beginPath(); x.roundRect(pad, ry + 32, Math.max(8, ((W - pad * 2) * gp) / 100), 8, 4); x.fill();
        }
      } else {
        const latest = one.recent.filter((a) => !rarest.slice(0, 3).some((q) => q.a.id === a.id));
        if (latest.length) {
          head("Latest unlocked");
          const s = 104, gap = 20, per = Math.floor((W - pad * 2 + gap) / (s + gap)), rowsN = Math.max(0, Math.floor((H - 130 - y) / (s + gap)));
          const shown = latest.slice(0, per * rowsN);
          for (let i = 0; i < shown.length; i++) await icon(shown[i], pad + (i % per) * (s + gap), y + 12 + Math.floor(i / per) * (s + gap), s);
        }
      }
    }
    font(600, 22, "u"); x.fillStyle = C.fg2; x.fillText("Made with Lootnest", pad, H - 56);
  }
  function drawShareSoon() {
    const D = S.dialog, cv = $("#share-canvas");
    if (!D || D.kind !== "achshare" || !cv) return;
    const key = D.fmt + "|" + D.scope + "|" + D.who;
    if (D.canvas && D.drawnFor === key) { cv.replaceWith(D.canvas); return; }
    D.drawnFor = key; D.canvas = cv;
    D.drawn = drawAchShare(cv, D).catch(() => {});
  }
