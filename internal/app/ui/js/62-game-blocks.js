  // A game outside Steam without profiles: its saves, kept as versions in
  // the cloud folder (Steam games have Steam Cloud; games with profiles sync
  // each player's saves).
  const gsAsked = new Set();
  // Collections the game is in, with a way to add it to one.
  function collectionsBlock(it) {
    const key = recentKey(it), tags = tagsOf(it), others = allTags().filter((t) => !tags.includes(t));
    return `<section class="gpage-block gp-coll"><div class="ln-section-head"><div><h2>Collections</h2></div></div>
      <div class="gp-tags">${tags.map((t) => `<span class="ln-tag gp-tag">${esc(t)}<button data-action="tag-remove" data-key="${esc(key)}" data-tag="${esc(t)}" aria-label="Take it out of ${esc(t)}">${GI.x}</button></span>`).join("") || `<span class="gpage-hint">None</span>`}</div>
      ${tags.length < 8 ? `<div class="gp-tag-add"><input id="tag-new" class="ln-input" type="text" maxlength="40" list="tag-list" autocomplete="off" placeholder="Add to a collection, such as Co-op" data-key="${esc(key)}">
        <datalist id="tag-list">${others.map((t) => `<option value="${esc(t)}"></option>`).join("")}</datalist>
        <button class="ln-button ln-button--secondary ln-button--sm" data-action="tag-add" data-key="${esc(key)}">Add</button></div>` : ""}</section>`;
  }

  // Will it run here: this PC against the game's requirements on Steam (read
  // with Game details), as a score on a dial, the settings to start with,
  // and each part against the minimum and the recommended.
  const RUN_TONE = (n) => (n >= 70 ? "ok" : n >= 35 ? "mid" : "low");
  function willRunBlock(it) {
    // The Steam id may be known only to Lootnest (a folder game's, by its
    // name or a file beside it): the details say which they were read by.
    const key = recentKey(it), outside = it.kind === "found" ? !storeKept(it.source) : it.g && it.g.launch.type !== "steam";
    const known = it.appId || (it.f && it.f.steamId) || (S.details[key] && S.details[key].appId);
    const head = `<div class="ln-section-head"><div><h2>Will it run</h2></div></div>`;
    if (!cfg().settings.gameDetails) {
      if (!known && !outside) return "";
      return `<section class="gpage-block wr wr--off">${head}<p class="wr-off">${ph("gauge", "duotone")}<span>Lootnest can compare this PC with the game's requirements on Steam.</span></p>
        <button class="ln-button ln-button--secondary ln-button--sm" data-action="details-on">Turn on Game details</button></section>`;
    }
    if (!S.details[key]) { S.details[key] = { loading: true }; api.gameDetails(key).catch(() => {}); }
    const R = S.details[key];
    if (R.loading) return known || outside ? `<section class="gpage-block wr">${head}<div class="ln-skeleton wr-skel"></div></section>` : "";
    if (!R.appId && !it.appId) return ""; // no Steam page found for it
    const run = R.run;
    if (!run || !run.plan || !run.plan.score) {
      const why = !run ? (R.error ? R.error : "Steam lists no requirements for this game.") : "Lootnest doesn't know this PC's parts or the game's well enough.";
      return `<section class="gpage-block wr">${head}<p class="gpage-hint">${why}</p></section>`;
    }
    const P = run.plan, V = run.verdict, pc = run.pc || {}, tone = RUN_TONE(P.score);
    // The dial: 240 degrees of arc, filled to the score.
    const r = 52, len = 2 * Math.PI * r * (240 / 360), fill = (len * P.score) / 100;
    const dial = `<svg class="wr-dial" viewBox="0 0 132 116" role="img" aria-label="Score ${P.score} out of 100">
      <path class="wr-track" d="${arc(66, 66, r, -120, 120)}" pathLength="${len.toFixed(1)}" stroke-dasharray="${len.toFixed(1)}"/>
      <path class="wr-fill" d="${arc(66, 66, r, -120, 120)}" style="--len:${len.toFixed(1)};--fill:${fill.toFixed(1)}"/>
      ${[0, 35, 70, 90].map((t) => { const a = ((-120 + (240 * t) / 100) * Math.PI) / 180; return `<line class="wr-tick" x1="${(66 + 60 * Math.sin(a)).toFixed(1)}" y1="${(66 - 60 * Math.cos(a)).toFixed(1)}" x2="${(66 + 64 * Math.sin(a)).toFixed(1)}" y2="${(66 - 64 * Math.cos(a)).toFixed(1)}"/>`; }).join("")}
    </svg>`;
    const say = P.preset ? `<b class="wr-preset">${esc(P.preset)}</b><span class="wr-sub">settings to start with</span>` : `<b class="wr-preset">Under the minimum</b><span class="wr-sub">it may not start or keep up</span>`;
    const chips = P.preset ? `<div class="wr-chips">
        <span class="wr-chip">${ph("monitor")}${esc(P.resolution)}</span>
        <span class="wr-chip">${ph("lightning")}${esc(P.fps)} fps</span>
        ${P.upscaling ? `<span class="wr-chip wr-chip--tip">${ph("sparkle")}Turn on DLSS, FSR or XeSS</span>` : ""}</div>` : "";
    const val = (list, label) => ((list || []).find((l) => l.label === label) || {}).value || "";
    const icon = { Graphics: "graphics-card", Processor: "cpu", Memory: "memory" };
    const mine = { Graphics: pc.gpu, Processor: pc.cpu, Memory: pc.ram ? gb(pc.ram) : "" };
    const lanes = (V.parts || []).filter((p) => p.known && icon[p.part]).map((p) => {
      const top = Math.max(p.have, (p.rec || p.min) * 1.5, p.min) || 1, pct = (x) => Math.max(0, Math.min(100, (x / top) * 100)).toFixed(1);
      // The first card or chip a requirement names ("GTX 1060 or RX 580"),
      // in full in its tooltip.
      const needAll = val(run.recommended, p.part) || val(run.minimum, p.part);
      const need = needAll.split(/\s+or\s+|\s*,\s*|\s+and higher|\s*\/\s*|\s+-\s+/i)[0].replace(/\s*\(.*$/, "");
      return `<li class="wr-lane wr-lane--${runsTone(p.runs)}" style="--have:${Math.min(97, pct(p.have))}%;--min:${pct(p.min)}%;--rec:${pct(p.rec)}%">
        <span class="wr-part">${ph(icon[p.part])}<b>${esc(p.part)}</b><em>${esc(RUNS[p.runs] ? RUNS[p.runs][2] : "")}</em></span>
        <span class="wr-bar" aria-hidden="true">${p.min ? '<i class="wr-mark wr-mark--min"></i>' : ""}${p.rec ? '<i class="wr-mark wr-mark--rec"></i>' : ""}<i class="wr-you"></i></span>
        <span class="wr-names"><span title="This PC">${esc(mine[p.part] || "This PC")}</span>${need ? `<span title="${val(run.recommended, p.part) ? "Recommended" : "Minimum"}: ${esc(needAll)}">${esc(need)}</span>` : ""}</span></li>`;
    }).join("");
    const limit = V.limit && P.score < 70 ? `<p class="wr-limit">${ph("warning")}The ${esc(PART[V.limit] || V.limit.toLowerCase())} holds it back</p>` : "";
    return `<section class="gpage-block wr wr--${tone}" style="--score:${P.score}">${head}
      <div class="wr-top">
        <div class="wr-gauge">${dial}<span class="wr-score ln-num" data-countup="${P.score}">${P.score}</span><span class="wr-of">of 100</span></div>
        <div class="wr-say">${say}${chips}</div>
      </div>
      ${limit}
      ${lanes ? `<ul class="wr-lanes">${lanes}</ul><p class="wr-legend"><i class="wr-mark wr-mark--min"></i>Minimum<i class="wr-mark wr-mark--rec"></i>Recommended<i class="wr-you"></i>This PC</p>` : ""}
      <p class="gpage-hint">An estimate from Steam's requirements, not a benchmark.</p>${peersBlock(it, pc)}</section>`;
  }
  // Players with a PC like this one: Steam's reviews by the writer's
  // graphics card (this PC's and the two closest), only once the user has
  // asked for it, since the cards' names go to Steam.
  function peersBlock(it, pc) {
    if (!pc.gpu) return "";
    const key = recentKey(it), head = `<h3 class="wr-peers-head">${ph("users-three")}Players with a PC like yours</h3>`;
    if (!cfg().settings.hardwareReviews) {
      return `<div class="wr-peers">${head}<p class="gpage-hint">See what Steam reviews say from players with your ${esc(shortGPU(pc.gpu))} or a close card. Its name is sent to Steam.</p>
        <button class="ln-button ln-button--secondary ln-button--sm" data-action="peers-on" data-k="${esc(key)}">${ph("magnifying-glass")}Look at their reviews</button></div>`;
    }
    if (!(key in S.peers)) { S.peers[key] = { loading: true }; api.gamePeers(key).catch(() => {}); }
    const P = S.peers[key];
    if (P.loading) return `<div class="wr-peers">${head}<div class="ln-skeleton wr-peers-skel"></div></div>`;
    if (P.none) return "";
    if (P.error) return `<div class="wr-peers">${head}${lnInline("error", esc(P.error))}</div>`;
    const R = P.report;
    if (!R || !R.count) return `<div class="wr-peers">${head}<p class="gpage-hint">No reviews yet from players with your card or a close one.</p></div>`;
    const pct = Math.round((R.positive * 100) / R.count), cards = R.gpus.map(shortGPU);
    const open = S.peersOpen.has(key), quotes = R.quotes.slice(0, open ? 6 : 2);
    const say = [
      R.fps ? `<li>${ph("lightning")}<span><b class="ln-num">${R.fps} fps</b> is what they name most (${R.fpsCount} said a number)</span></li>` : "",
      R.perf ? `<li>${ph("chat-circle")}<span><b class="ln-num">${R.perf}</b> of ${R.read} talk about how it runs${R.perfBad ? `, <b class="ln-num">${R.perfBad}</b> of them unhappy` : ""}</span></li>` : "",
      R.hours ? `<li>${ph("clock-counter-clockwise")}<span>Half had played <b class="ln-num">${R.hours} h</b> or more when they wrote</span></li>` : "",
    ].join("");
    return `<div class="wr-peers">${head}
      <div class="wr-peers-top"><span class="wr-peers-pct ln-num">${pct}<small>%</small></span><span><b>recommend it</b><span class="gpage-hint">${R.count.toLocaleString("en-US")} reviews from players with ${esc(cards.slice(0, -1).join(", "))}${cards.length > 1 ? " or " : ""}${esc(cards[cards.length - 1])}</span></span></div>
      <span class="wr-peers-bar" aria-hidden="true"><i style="width:${pct}%"></i></span>
      ${say ? `<ul class="wr-peers-say">${say}</ul>` : ""}
      ${quotes.length ? `<ul class="wr-quotes">${quotes.map((q) => `<li class="wr-quote${q.up ? "" : " is-down"}"><span class="wr-quote-ic" aria-hidden="true">${ph(q.up ? "thumbs-up" : "thumbs-down")}</span>
        <div><p>“${esc(q.text)}”</p><span>${esc([shortGPU(q.gpu), shortCPU(q.cpu), q.ram ? q.ram + " GB" : "", q.hours ? q.hours + " h played" : ""].filter(Boolean).join(", "))}</span></div></li>`).join("")}</ul>` : ""}
      ${R.quotes.length > 2 ? `<button class="gpage-link" data-action="peers-more" data-k="${esc(key)}">${open ? "Fewer" : `${R.quotes.length - 2} more`}</button>` : ""}
      <p class="gpage-hint">From Steam reviews whose writers shared their hardware. What people wrote, not a measurement.</p></div>`;
  }
  const shortGPU = (g) => (g || "").replace(/\b(NVIDIA|AMD|Intel\(R\)|GeForce|Radeon)\s*/gi, "").trim() || g;
  const shortCPU = (c) => (c || "").replace(/\(R\)|\(TM\)|CPU|Processor|\d+-Core|@.*$/gi, "").replace(/\s+/g, " ").trim();
  // An SVG arc around (cx, cy), from angle a to b in degrees (0 is up).
  function arc(cx, cy, r, a, b) {
    const pt = (d) => { const t = (d * Math.PI) / 180; return `${(cx + r * Math.sin(t)).toFixed(2)} ${(cy - r * Math.cos(t)).toFixed(2)}`; };
    return `M ${pt(a)} A ${r} ${r} 0 ${b - a > 180 ? 1 : 0} 1 ${pt(b)}`;
  }

  // Achievements: read from this PC (Steam's library cache, or the file a
  // Steam-compatible tool keeps next to the game's saves). The latest
  // unlocked first; the whole list on request.
  // Library key -> when its achievements were last asked for. A page open
  // again after a minute asks again, so a new file or unlock shows by itself.
  const achAsked = new Map();
  const ACH_AGAIN = 60000;
  // Whose achievements a game shows, and the pieces both the page and the
  // panel with the whole list are made of.
  function achFor(it) {
    const key = recentKey(it);
    if (!achAsked.has(key) || Date.now() - achAsked.get(key) > ACH_AGAIN) {
      achAsked.set(key, Date.now());
      const dir = it.f ? it.f.dir : it.exe;
      if (it.appId || dir) api.achievements(key, it.appId, dir, it.name).catch(() => {});
    }
    const top = S.ach[key];
    if (!top || !top.total) return null;
    // Each player's own (they go with their profile): whose to show is the
    // one picked here, else whoever's at the PC, else whose files are in.
    const ids = top.players ? cfg().players.map((p) => p.id).filter((id) => top.players[id]) : [];
    const seat = it.g ? pageWho(it) : seated(), owner = it.g && S.d.active[it.g.id];
    const who = ids.includes(S.achWho[key]) ? S.achWho[key] : seat && ids.includes(seat.id) ? seat.id : ids.includes(owner) ? owner : ids[0];
    const A = who ? top.players[who] : top;
    const pct = A.total ? Math.round((A.unlocked / A.total) * 100) : 0;
    const whoRow = ids.length ? `<div class="ach-who" role="group" aria-label="Whose achievements">${ids.map((id) => { const p = player(id); return `<button class="ln-chip ln-player" style="--player:${esc(p.color)}" data-action="ach-who" data-key="${esc(key)}" data-player="${esc(id)}" aria-pressed="${id === who}">${lnAvatar(p, "20")}${esc(p.name)}<span class="ln-num">${top.players[id].unlocked}</span></button>`; }).join("")}</div>` : "";
    const g = it.g;
    const note = A.source !== "pc" ? "From Steam, for this PC's account" : ids.length ? "One set per profile" : g && g.sharedAchievements ? "Shared by everyone" : "";
    const toggle = g && A.source === "pc" ? `<button class="ln-button ln-button--ghost ln-button--sm" data-action="ach-shared" data-game="${esc(g.id)}" data-on="${g.sharedAchievements ? "0" : "1"}">${g.sharedAchievements ? "Give each profile its own" : "Share one set"}</button>` : "";
    const bar = `<div class="ach-bar" role="progressbar" aria-valuemin="0" aria-valuemax="${A.total}" aria-valuenow="${A.unlocked}" aria-label="${A.unlocked} of ${A.total} achievements"><span style="width:${pct}%"></span></div>`;
    const row = (a) => achRow(a, key);
    return { key, who, A, pct, whoRow, note, toggle, bar, row };
  }
  // How far the story is, beside Play: from the achievements that mark its
  // chapters or its end (core.StoryOf), for whoever's achievements the page
  // shows. On a game's page it opens the Achievements tab.
  function storyChip(it, page) {
    const X = achFor(it), st = X && X.A.story;
    if (!st || !st.steps || !st.steps.length) return "";
    const done = st.steps.filter((s) => s.done), last = done[done.length - 1];
    const line = st.finished ? "Finished" : last ? `${last.label} done` : "Not started";
    const said = st.steps.map((s) => `${s.label}: ${s.done ? "done" : "not yet"}`).join("\n");
    const inner = `<span class="ach-gring">${ring(st.percent, 44)}</span><span class="gpage-state"><b>Story</b><span>${esc(line)}</span></span>`;
    const label = `Story ${st.percent}%, ${line}`;
    return page
      ? `<button class="gpage-story" data-action="game-tab" data-tab="achievements" title="${esc(said)}" aria-label="${esc(label)}">${inner}</button>`
      : `<div class="gpage-story" title="${esc(said)}" role="img" aria-label="${esc(label)}">${inner}</div>`;
  }
  // Achievements by how rare they are, as gamers know them: ultra rare in
  // gold, rare in blue, uncommon in green, common in silver.
  const TIERS = { ultra: "Ultra rare", rare: "Rare", uncommon: "Uncommon", common: "Common" };
  const tierOf = (p) => (!p ? "" : p < 2 ? "ultra" : p < 10 ? "rare" : p < 30 ? "uncommon" : "common");
  const pctText = (p) => (p < 10 ? p.toFixed(1) : Math.round(p)) + "%";
  const achIcon = (a) => `<span class="ach-ico">${a.icon ? `<img src="${esc(a.icon)}" alt="" loading="lazy">` : GI.trophy}${a.unlocked ? "" : `<span class="ach-lock">${GI.lock || ""}</span>`}</span>`;
  // Every achievement drawn, by its row key, for the card that shows one.
  const ACH_SEEN = new Map();
  // go: the row is on the Achievements page, so a click goes to the game's
  // Achievements tab first.
  function achRow(a, key, sub, go) {
    const t = a.unlocked || !a.hidden ? tierOf(a.percent) : "";
    const rk = key + "|" + a.id;
    const hid = a.hidden && !a.unlocked && !S.revealed.has(rk);
    ACH_SEEN.set(rk, { a, key });
    const line = sub || (a.unlocked ? (a.at && new Date(a.at).getFullYear() > 2000 ? "Unlocked " + esc(ago(a.at)) : "Unlocked") : hid ? "Click to show" : esc(a.description || "Not unlocked yet"));
    const open = hid ? ` data-action="ach-reveal" data-k="${esc(rk)}" role="button" tabindex="0" aria-label="Show this hidden achievement"`
      : ` data-action="ach-card" data-k="${esc(rk)}"${go ? ' data-go="1"' : ""} role="button" tabindex="0" aria-haspopup="dialog" aria-label="${esc(a.name)}: look at it"`;
    return `<li class="ach${a.unlocked ? " is-on" : " locked"}${t ? " tier-" + t : ""}${hid ? " is-hidden" : ""}"${open}>${achIcon(a)}
      <div><b>${hid ? "Hidden achievement" : esc(a.name)}</b><span>${line}</span>${a.percent && !hid ? `<span class="ach-rar" aria-hidden="true"><i style="width:${Math.max(3, Math.min(100, a.percent))}%"></i></span>` : ""}</div>
      ${a.percent && !hid ? `<span class="ach-tier" title="${esc(pctText(a.percent))} of players have it"><em>${TIERS[tierOf(a.percent)]}</em><b class="ln-num">${pctText(a.percent)}</b></span>` : ""}</li>`;
  }
  // One achievement as a card in the middle of the screen: it tilts toward
  // the pointer with a light that follows it, turns when dragged, and flips
  // to its back (the game, when, and how rare) on a click, Enter or Space.
  // Arrow keys turn it too. Nothing moves with reduced motion.
  function viewAchCard() {
    const x = ACH_SEEN.get(S.dialog.rk);
    if (!x) return "";
    const a = x.a, t = tierOf(a.percent) || "none", it = itemByKey(x.key);
    const dated = a.unlocked && a.at && new Date(a.at).getFullYear() > 2000;
    const when = dated ? new Date(a.at).toLocaleDateString(undefined, { day: "numeric", month: "long", year: "numeric" }) : a.unlocked ? "Unlocked" : "Not unlocked yet";
    const icon = a.icon ? `<img src="${esc(a.icon)}" alt="">` : ph("trophy", "fill");
    const pct = a.percent ? pctText(a.percent) : "";
    return `<div class="scrim achc-scrim" data-action="scrim"><div class="achc" role="dialog" aria-modal="true" aria-label="${esc(a.name)}">
      <div class="achc-stage"><div class="achc-card tier-${t}${a.unlocked ? " is-on" : " is-locked"}" tabindex="0" role="button" aria-label="Flip the card" aria-pressed="false">
        <div class="achc-face achc-front">
          <span class="achc-glow" aria-hidden="true"></span>
          <span class="achc-top"><span class="achc-tier">${t !== "none" ? TIERS[t] : "Achievement"}</span>${pct ? `<span class="achc-pct ln-num">${pct}</span>` : ""}</span>
          <span class="achc-ico">${icon}${a.unlocked ? "" : `<span class="achc-lock">${ph("lock")}</span>`}</span>
          <b class="achc-name">${esc(a.name)}</b>
          <p class="achc-desc">${esc(a.description || (a.hidden ? "A hidden achievement" : ""))}</p>
          <span class="achc-foot">${esc(it ? it.name : "")}</span>
          <span class="achc-shine" aria-hidden="true"></span>
        </div>
        <div class="achc-face achc-back">
          <span class="achc-brand">${ph("trophy", "fill")}<span>Lootnest</span></span>
          <dl class="achc-facts">
            <div><dt>Game</dt><dd>${esc(it ? it.name : "")}</dd></div>
            <div><dt>${dated ? "Unlocked" : "Status"}</dt><dd>${esc(when)}</dd></div>
            ${pct ? `<div><dt>Players who have it</dt><dd class="ln-num">${pct}</dd></div>` : ""}
          </dl>
          ${a.percent ? `<span class="achc-rar" aria-hidden="true"><i style="width:${Math.max(3, Math.min(100, a.percent))}%"></i></span><span class="achc-rar-note">${t !== "none" ? TIERS[t] : ""}: ${pct} of Steam players</span>` : ""}
          <span class="achc-shine" aria-hidden="true"></span>
        </div>
      </div></div>
      <p class="achc-hint">Move the pointer to tilt it, drag to turn it, click to flip it</p>
      <button class="ln-button ln-button--ghost ln-button--icon achc-close" data-action="dialog-cancel" aria-label="Close">${GI.x}</button>
    </div></div>`;
  }
  // The card's tilt, turn and flip, set as CSS variables on it.
  (function achCardMotion() {
    let drag = null, spin = 0, flip = false;
    const card = () => $(".achc-card");
    const put = (c, rx, ry) => {
      c.style.setProperty("--rx", rx.toFixed(2) + "deg");
      c.style.setProperty("--ry", (ry + spin + (flip ? 180 : 0)).toFixed(2) + "deg");
    };
    // The card follows the pointer, so it moves unless Motion says not to
    // (Off, or Like Windows while Windows has animation effects off).
    const still = () => reduced() || document.documentElement.classList.contains("motion-off");
    document.addEventListener("pointermove", (ev) => {
      const c = card();
      if (!c || still()) return;
      const r = c.getBoundingClientRect(), x = (ev.clientX - r.left) / r.width, y = (ev.clientY - r.top) / r.height;
      c.style.setProperty("--mx", (x * 100).toFixed(1) + "%");
      c.style.setProperty("--my", (y * 100).toFixed(1) + "%");
      if (drag) { spin = drag.spin + (ev.clientX - drag.x) * 0.6; drag.moved = drag.moved || Math.abs(ev.clientX - drag.x) > 4; c.classList.add("is-dragging"); }
      const near = x > -0.4 && x < 1.4 && y > -0.4 && y < 1.4;
      put(c, near ? (0.5 - y) * 22 : 0, near && !drag ? (x - 0.5) * 26 : 0);
    });
    document.addEventListener("pointerdown", (ev) => {
      const c = ev.target.closest && ev.target.closest(".achc-card");
      if (!c) { if (!card()) { spin = 0; flip = false; } return; }
      drag = { x: ev.clientX, spin, moved: false };
      c.setPointerCapture && c.setPointerCapture(ev.pointerId);
    });
    document.addEventListener("pointerup", () => {
      const c = card();
      if (!drag || !c) { drag = null; return; }
      c.classList.remove("is-dragging");
      if (!drag.moved) { flip = !flip; c.setAttribute("aria-pressed", String(flip)); uiSound("tab"); }
      // It settles on the nearest face.
      spin = Math.round(spin / 180) * 180;
      if (Math.round(spin / 180) % 2) { flip = !flip; spin = 0; }
      drag = null;
      put(c, 0, 0);
    });
    document.addEventListener("keydown", (ev) => {
      const c = card();
      if (!c || !S.dialog || S.dialog.kind !== "achcard") return;
      if ((ev.key === "Enter" || ev.key === " ") && ev.target === c) { ev.preventDefault(); flip = !flip; c.setAttribute("aria-pressed", String(flip)); put(c, 0, 0); }
      else if (ev.key === "ArrowLeft" || ev.key === "ArrowRight") { ev.preventDefault(); spin += ev.key === "ArrowLeft" ? -20 : 20; put(c, 0, 0); }
      else if (ev.key === "ArrowUp" || ev.key === "ArrowDown") { ev.preventDefault(); put(c, ev.key === "ArrowUp" ? 14 : -14, 0); }
    });
    // A new card starts face up.
    new MutationObserver(() => { if (!card()) { spin = 0; flip = false; drag = null; } }).observe(document.getElementById("overlay"), { childList: true });
  })();
  // A ring of how much of a game is done.
  const ring = (pct, size = 44) => `<svg class="ach-ring${pct === 100 ? " is-full" : ""}" viewBox="0 0 36 36" width="${size}" height="${size}" aria-hidden="true"><circle class="bg" cx="18" cy="18" r="15.5" pathLength="100"/><circle class="fg" cx="18" cy="18" r="15.5" pathLength="100" style="--p:${pct}"/></svg><span class="ach-ring-n ln-num">${pct}<small>%</small></span>`;
  // Numbers that count up as a page comes in.
  function countUp(root) {
    if (reduced()) return;
    root.querySelectorAll("[data-countup]").forEach((el) => {
      const to = +el.dataset.countup, suffix = el.dataset.suffix || "";
      if (!(to > 0)) return;
      const t0 = performance.now(), d = 900;
      const step = (t) => {
        const k = Math.min(1, (t - t0) / d), e = 1 - Math.pow(1 - k, 4);
        el.innerHTML = Math.round(to * e) + suffix;
        if (k < 1 && el.isConnected) requestAnimationFrame(step);
      };
      requestAnimationFrame(step);
    });
  }
  // An unlock, in the window: it comes in at the top right, shines once for
  // the rare ones, and goes after a few seconds.
  function achPop(list, game, pid) {
    let box = $("#ach-pops");
    if (!box) { box = document.createElement("div"); box.id = "ach-pops"; box.className = "ach-pops ln-root"; box.setAttribute("aria-live", "polite"); document.body.appendChild(box); }
    const p = pid && player(pid);
    (list || []).slice(0, 4).forEach((a, i) => {
      const t = tierOf(a.percent);
      const el = document.createElement("div");
      el.className = `ach-pop${t ? " tier-" + t : ""}`;
      if (p) { el.classList.add("ln-player"); el.style.setProperty("--player", p.color); }
      el.style.animationDelay = i * 160 + "ms";
      el.innerHTML = `${achIcon(Object.assign({}, a, { unlocked: true }))}<div><span class="ach-pop-k">${p ? '<span class="dot" aria-hidden="true"></span>' : ""}Achievement unlocked${p && cfg().players.length > 1 ? " for " + esc(p.name) : ""}</span><b>${esc(a.name || "Achievement")}</b>
        <span class="ach-pop-l">${a.percent ? `<em>${TIERS[t]}</em>, ${esc(pctText(a.percent))} of players` : esc(a.description || game || "")}</span></div>`;
      box.appendChild(el);
      setTimeout(() => { el.classList.add("out"); el.addEventListener("animationend", () => el.remove(), { once: true }); setTimeout(() => el.remove(), 800); }, 5600 + i * 160);
    });
  }
  // The Achievements tab: the whole list. few: Overview's latest three.
  function achievementsBlock(it, few) {
    const X = achFor(it);
    if (!X) {
      // Read and none found: say why, and what Steam's way still needs.
      const W = S.achWhy[recentKey(it)];
      if (few || S.ach[recentKey(it)] !== null || (!docWorth(it) && !(W && W.appId))) return "";
      return `<section class="gpage-block gp-ach">${gpHead("Achievements")}<p>None found on this PC</p>${achSteamSteps(W)}
        <div class="gpage-actions">${W && W.appId && (!W.key || !W.on) ? `<button class="ln-button ln-button--secondary ln-button--sm" data-action="set-open" data-tab="online">Open Settings, Online</button>` : ""}
          ${docWorth(it) ? `<button class="ln-button ln-button--ghost ln-button--sm" data-action="ach-doctor" ${itemAttr(it)} aria-haspopup="dialog">Check why</button>` : ""}</div></section>`;
    }
    const { key, who, A, pct } = X;
    const list = few ? A.list.filter((a) => a.unlocked).slice(0, 3) : A.list;
    const acts = few ? `<button class="ln-button ln-button--secondary ln-button--sm" data-action="game-tab" data-tab="achievements">Show all ${A.total}</button>`
      : `<button class="ln-button ln-button--secondary ln-button--sm" data-action="ach-share" data-who="${esc(who || "")}" data-scope="${esc(key)}" aria-haspopup="dialog">${ph("share-network")}Share</button>`
        + (docWorth(it) ? `<button class="ln-button ln-button--ghost ln-button--sm" data-action="ach-doctor" ${itemAttr(it)} aria-haspopup="dialog">Check</button>` : "") + X.toggle;
    return `<section class="gpage-block gp-ach"><div class="ln-section-head"><div><h2>${who ? esc(whoseOf(who)) + " achievements" : "Achievements"}</h2></div>
        <span class="ach-count"><b class="ln-num">${A.unlocked}</b> of <span class="ln-num">${A.total}</span>, ${pct}%</span></div>
      ${X.whoRow}${X.bar}
      ${list.length ? `<ul class="ach-list${few ? "" : " ach-grid"}">${list.map(X.row).join("")}</ul>` : `<p>None unlocked yet.</p>`}
      <div class="gp-ach-foot"><p class="gpage-hint">${X.note}</p>${acts ? `<div class="gpage-actions">${acts}</div>` : ""}</div></section>`;
  }
  // A game whose copy records no achievements but whose Steam id is known
  // can show the user's Steam account's: what that needs, each step lit
  // when it's done, and what Steam said last.
  function achSteamSteps(W) {
    if (!W || !W.appId) return "";
    const said = W.said || "";
    const steps = [
      [W.steam, "Steam signed in on this PC", W.steam ? "" : "Lootnest reads which account is yours from Steam on this PC."],
      [W.key, "Your Steam Web API key", W.key ? "" : "Free on steamcommunity.com/dev/apikey. Paste it in Settings, Online."],
      [W.on, "Achievements from my Steam account turned on", W.on ? "" : "In Settings, Online."],
      [said === "ok" || said === "none", "Your Steam profile's Game details are public",
        said === "private" ? "Steam says they're private. In Steam: your profile, Edit Profile, Privacy Settings, Game details: Public." : said ? "" : "In Steam: your profile, Edit Profile, Privacy Settings, Game details: Public."],
    ];
    const end = said === "none" ? "Steam has no achievements for this game on your account. They show only if you own it there."
      : said === "error" ? "Lootnest couldn't reach Steam. It asks again when you come back to this page."
      : W.steam && W.key && W.on && !said ? "All set: Lootnest asks Steam when you open this page." : "";
    return `<p class="gpage-hint">This copy doesn't keep its achievements on the PC. If you own the game on Steam, Lootnest can show your Steam account's.</p>
      <ol class="ach-steps">${steps.map(([ok, t, line]) => `<li class="${ok ? "is-done" : ""}" aria-label="${esc(t)}: ${ok ? "done" : "not yet"}"><span class="ln-lamp${ok ? " ln-lamp--on" : ""}" aria-hidden="true"></span><div><b>${esc(t)}</b>${line ? `<span>${esc(line)}</span>` : ""}</div></li>`).join("")}</ol>
      ${end ? `<p class="gpage-hint">${esc(end)}</p>` : ""}`;
  }
  // Save points: named copies of a game's saves, on a timeline.
  const pointsAsked = new Set();
  let ptEdit = null; // the save point being renamed
  function pointsFor(it) {
    const key = recentKey(it);
    if (!pointsAsked.has(key)) { pointsAsked.add(key); api.savePoints(key).catch(() => {}); }
    return S.points[key];
  }
  // Going back to a point takes the game there: closed if it runs, the
  // point's saves in as the newest, then started on them.
  const pointGameRuns = (it) => !!((S.points[recentKey(it)] || {}).running || (it.kind === "game" && session(it.id)));
  function goBackAsk(it, name, runs) {
    return runs
      ? { title: `Go back to ${name}?`, body: `Lootnest asks ${it.name} to close, as closing its window would, puts that save in place and starts the game on it. Anything the game hasn't saved yet is lost. The saves as they are now are kept as a save point first.`, ok: "Close and go back" }
      : { title: `Go back to ${name}?`, body: `${it.name} starts on that save: Lootnest puts it in place as the newest, then starts the game. The saves as they are now are kept as a save point first, so you can come back to them too.`, ok: "Go back and play" };
  }
  // The game a point went back to starts, the way its Play key starts it.
  function playAfterPoint(ev) {
    const it = everyItem().find((x) => recentKey(x) === ev.key);
    if (!it) return;
    if (BP.on) { bpPlay(it, ev.player || undefined); return; }
    if (it.kind === "found") { actions["play-found"]({ dataset: { key: it.id } }); return; }
    const pid = ev.player || (seated() || {}).id;
    if (pid) actions["play-as"]({ dataset: { game: it.id, player: pid } });
    else actions["play-pick"]({ dataset: { game: it.id } });
  }
  const POINT_STEPS = { closing: "Closing the game", copying: "Copying", saving: "Saving in the game" };
  // Save now: the game saves itself first when it runs and has a quick
  // save key chosen (core/quicksave.go).
  const savesFirst = (P) => !!(P.running && P.quickSave && !P.antiCheat);
  function quickSaveRow(it, P) {
    if (P.antiCheat) return `<p class="gpage-hint pt-qs">${esc(P.antiCheat)} is in this game, so Lootnest never presses keys in it. A save point has its last save: save in the game first.</p>`;
    const opts = ["", ...QS_KEYS].map((k) => `<option value="${k}"${(P.quickSave || "") === k ? " selected" : ""}>${k || "None"}</option>`).join("");
    const use = !P.quickSave && P.suggest ? `<button class="ln-button ln-button--ghost ln-button--sm" data-action="qs-use" ${itemAttr(it)} data-k="${esc(P.suggest)}">Use ${esc(P.suggest)}</button>` : "";
    const say = P.quickSave ? `While the game runs, a save point presses ${esc(P.quickSave)} first and waits for the game to save.`
      : P.suggest ? `This game's quick save is usually ${esc(P.suggest)}. With it chosen, a save point saves the game first.`
      : "With the game's quick save key chosen, a save point saves the game first. Without one, it keeps the game's last save.";
    return `<div class="pt-qs"><label for="pt-qs">Quick save key</label><select class="ln-input" id="pt-qs" data-qs="${esc(recentKey(it))}">${opts}</select>${use}<span class="gpage-hint">${say}</span></div>`;
  }
  const QS_KEYS = ["F1", "F2", "F3", "F4", "F5", "F6", "F7", "F8", "F9", "F10", "F11", "F12"];
  function pointRow(it, p, full, P) {
    const when = new Date(p.created).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
    const facts = [esc(when)];
    if (p.playtime) facts.push(`${esc(hours(p.playtime))} played`);
    if (p.unlocked) facts.push(`${p.unlocked} achievement${p.unlocked === 1 ? "" : "s"}`);
    const who = p.player && player(p.player) ? `<span class="ln-mark ln-player" style="--player:${esc(color(p.player))}"></span>${esc(pname(p.player))}` : "";
    const attrs = `${itemAttr(it)} data-id="${esc(p.id)}" data-name="${esc(p.name)}"`;
    const b = (cls, a, label) => `<button class="ln-button ${cls} ln-button--sm" ${a} ${P.busy ? "disabled" : ""}>${label}</button>`;
    const editing = full && ptEdit === p.id;
    const name = editing ? `<input class="ln-input pt-name-in" id="pt-rename" maxlength="60" value="${esc(p.name)}" aria-label="Name">` : `<b>${esc(p.name)}</b>`;
    const actions = editing ? b("ln-button--secondary", `data-action="pt-rename-save" ${attrs}`, "Save") + b("ln-button--ghost", 'data-action="pt-rename-cancel"', "Cancel")
      : b("ln-button--secondary", `data-action="pt-restore" ${attrs}`, "Go back to it")
        + (full && P.profiles && (!it.g || runsOk(it.g.id)) ? b("ln-button--ghost", `data-action="pt-branch" ${attrs} data-player="${esc(p.player || "")}"`, "New run from it") : "")
        + (full ? b("ln-button--ghost", `data-action="pt-rename" ${attrs}`, "Rename") + b("ln-button--ghost", `data-action="pt-delete" ${attrs}`, "Delete") : "");
    const pic = p.thumb ? picButton(it, "point:" + p.id, p.thumb, `Picture of ${p.name}`) : "";
    return `<li class="pt-row${p.auto ? " pt-row--auto" : ""}${pic ? " pt-row--pic" : ""}"><span class="pt-dot" aria-hidden="true"></span><div class="pt-main">${name}
      <span class="pt-facts">${who ? `<span>${who}</span>` : ""}${facts.map((x) => `<span>${x}</span>`).join("")}${p.update ? "<span>Made before the game updated</span>" : p.auto ? "<span>Kept before going back</span>" : p.inGame ? "<span>Saved by the game first</span>" : p.game ? "<span>Kept when the game saved</span>" : ""}</span></div>${pic}<div class="pt-actions">${actions}</div></li>`;
  }
  const shotLabel = (e) => "Screenshot, " + new Date(e.at).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "medium" });
  // A picture's thumbnail that opens it large (viewShots).
  const picButton = (it, ref, thumb, label) => `<button class="gp-shot pic-thumb" data-action="pic-open" ${itemAttr(it)} data-ref="${esc(ref)}" aria-label="${esc(label)}" aria-haspopup="dialog"><img src="${esc(thumb)}" alt=""></button>`;
  // Every picture the game page shows, in order: save points', screenshots',
  // then other apps' captures.
  function pagePictures(it) {
    const key = recentKey(it), P = S.points[key] || {}, C = S.captures[key];
    return [...(P.points || []).filter((p) => p.thumb).map((p) => ({ ref: "point:" + p.id, thumb: p.thumb, label: p.name })),
      ...(S.timeline[key] || []).filter((e) => e.kind === "shot" && e.thumb).map((e) => ({ ref: "shot:" + e.name, thumb: e.thumb, label: shotLabel(e) })),
      ...((C && C.items) || []).filter(capInBox).map((c) => ({ ref: "capture:" + c.path, thumb: c.thumb || "", label: capLabel(c) }))];
  }
  // Captures from other apps (ps_gameCaptures): Game Bar, Steam, NVIDIA,
  // AMD and OBS, on the game's Timeline. A picture opens in the lightbox; a
  // clip, or a JPEG XR picture the window can't show, opens in the app
  // Windows opens it with.
  const capAsked = new Set();
  const capWhat = (c) => `${(c.ckind || c.kind) === "clip" ? "Clip" : "Screenshot"} (${c.source})`;
  const capLabel = (c) => capWhat(c) + ", " + new Date(c.at).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
  const capInBox = (c) => (c.ckind || c.kind) === "picture" && !/\.jxr$/i.test(c.path);
  function capturesFor(key) {
    if (cfg().settings.capturesOff) return null;
    if (!capAsked.has(key)) { capAsked.add(key); api.gameCaptures(key).catch(() => {}); }
    return S.captures[key] || null;
  }
  function capThumb(it, c) {
    const clip = (c.ckind || c.kind) === "clip", inner = c.thumb ? `<img src="${esc(c.thumb)}" alt="">${clip ? `<span class="cap-mark" aria-hidden="true">${GI.play}</span>` : ""}`
      : `<span class="cap-plate" aria-hidden="true">${clip ? GI.clip : GI.image}</span>`;
    if (it && capInBox(c)) return `<button class="gp-shot pic-thumb" data-action="pic-open" ${itemAttr(it)} data-ref="${esc("capture:" + c.path)}" aria-label="${esc(capLabel(c))}" aria-haspopup="dialog">${inner}</button>`;
    return `<button class="gp-shot pic-thumb cap-open" data-action="capture-open" data-path="${esc(c.path)}" aria-label="Open ${esc(capLabel(c))}">${inner}</button>`;
  }
  // A game with profiles shows the page's profile's save points (and
  // those kept for no one), until everyone's are asked for.
  const ptEveryone = new Set();
  function pointsOf(it, list) {
    if (it.kind !== "game" || ptEveryone.has(recentKey(it)) || cfg().players.length < 2) return list;
    const who = pageWho(it).id;
    return list.filter((p) => !p.player || p.player === who);
  }
  function savePointsBlock(it) {
    const P = pointsFor(it);
    if (!P || !P.loaded || P.error) return "";
    const all = P.points || [], list = pointsOf(it, all), hidden = all.length - list.length;
    const whose = it.kind === "game" && cfg().players.length > 1 ? (ptEveryone.has(recentKey(it))
      ? `<button class="gpage-link" data-action="pt-everyone" ${itemAttr(it)}>Only ${esc(whoseOf(pageWho(it).id))}</button>`
      : hidden ? `<button class="gpage-link" data-action="pt-everyone" ${itemAttr(it)}>Everyone's, ${all.length}</button>` : "") : "";
    const hint = cfg().settings.pointHotkeyOff ? "" : '<span class="gpage-hint"><kbd>Ctrl+Shift+F5</kbd> in game</span>';
    const make = `<div class="pt-make"><input class="ln-input" id="pt-new" maxlength="60" placeholder="Name, such as Before the boss" aria-label="Save point name" ${P.busy ? "disabled" : ""}>
      <button class="ln-button ln-button--secondary ln-button--sm" data-action="pt-make" ${itemAttr(it)} ${P.busy ? "disabled" : ""}>${P.busy ? `<span class="ln-spinner" aria-hidden="true"></span>${POINT_STEPS[P.step] || "Copying"}` : savesFirst(P) ? "Save now" : "Make a save point"}</button>${hint}</div>${quickSaveRow(it, P)}`;
    const title = it.kind === "game" && cfg().players.length > 1 && !ptEveryone.has(recentKey(it)) ? `${esc(whoseOf(pageWho(it).id))} save points` : "Save points";
    return `<section class="gpage-block gp-points"><div class="ln-section-head"><div><h2>${title}</h2></div>${list.length || whose ? `<span class="ach-count">${list.length ? `<b class="ln-num">${list.length}</b> kept` : ""}${whose}</span>` : ""}</div>
      ${make}
      ${cfg().settings.gamePointsOff ? "" : '<p class="gpage-hint pt-auto">Each save the game makes while it runs is kept here by itself, the newest 30.</p>'}
      ${list.length ? `<ol class="pt-list pt-list--all">${shownPoints(it, list).map((p) => pointRow(it, p, true, P)).join("")}</ol>${moreGameSaves(it, list)}` : ""}</section>`;
  }
  // The game's own saves (kept by themselves) show the newest few until
  // asked for all; points the user made always show.
  const GAME_SHOWN = 5, ptAllGame = new Set();
  function shownPoints(it, list) {
    if (ptAllGame.has(recentKey(it))) return list;
    let n = 0;
    return list.filter((p) => !p.game || ++n <= GAME_SHOWN);
  }
  function moreGameSaves(it, list) {
    const n = list.filter((p) => p.game).length;
    if (n <= GAME_SHOWN || ptAllGame.has(recentKey(it))) return "";
    return `<button class="ln-button ln-button--ghost ln-button--sm pt-more" data-action="pt-all-game" ${itemAttr(it)}>Show all ${n} saves the game made</button>`;
  }
  // Overview: the newest save point, and the way to the rest.
  function latestPoint(it) {
    const P = S.points[recentKey(it)] || {}, list = pointsOf(it, P.points || []);
    if (!list.length) return "";
    return `<section class="gpage-block gp-points"><div class="ln-section-head"><div><h2>Latest save point</h2></div>
        <button class="ln-button ln-button--ghost ln-button--sm" data-action="game-tab" data-tab="saves">${list.length > 1 ? `All ${list.length} save points` : "Go to Saves"}</button></div>
      <ol class="pt-list">${pointRow(it, list[0], false, P)}</ol></section>`;
  }
  // Overview: the safety net's copies, for a game outside the stores.
  function latestCopy(it) {
    const G = it.kind === "found" && S.gsaves[it.id];
    if (!G || G.error || G.off === "steam") return "";
    const lv = G.local || [];
    const text = lv.length ? `${lv.length} ${lv.length === 1 ? "copy" : "copies"} on this PC, the newest from ${ago(lv[0].created)}.`
      : (G.locations || []).length ? "No copies yet" : "Save folder not set";
    return `<section class="gpage-block"><div class="ln-section-head"><div><h2>Saves</h2></div>
        <button class="ln-button ln-button--ghost ln-button--sm" data-action="game-tab" data-tab="saves">Go to Saves</button></div><p>${esc(text)}</p></section>`;
  }
  // Overview: the last session, by whom and for how long.
  function activityLine(it, timeline) {
    const e = (S.timeline[recentKey(it)] || []).find((x) => x.kind === "session");
    if (!e) return "";
    const who = e.profile && player(e.profile) && cfg().players.length > 1 ? pname(e.profile) : "";
    const text = `${who ? who + " played" : "Played"} ${playSpan(e.seconds || 0)}, ${ago(e.at)}`;
    return `<p class="gpage-note gp-activity">${GI.history}<span>${esc(text)}.</span>${timeline ? `<button class="gpage-link" data-action="game-tab" data-tab="timeline">See the timeline</button>` : ""}</p>`;
  }
  // The Saves tab, beside the save points: the copies made before swaps,
  // and trying someone else's save in a run of its own (as in Manage).
  function otherSavesBlock(it) {
    const g = it.g;
    if (!g) return "";
    const s = session(g.id);
    return `<section class="gpage-block"><div class="ln-section-head"><div><h2>${runsOk(g.id) ? "Copies and runs" : "Copies"}</h2></div></div><div class="ln-list">
      ${gpRow(`data-action="copies" data-game="${esc(g.id)}" aria-haspopup="dialog"`, GI.history, "Copies before swaps", "")}
      ${runsOk(g.id) ? gpRow(`data-action="try-save" data-game="${esc(g.id)}" aria-haspopup="dialog"`, GI.flask, "Try a save", "", { disabled: s || !canNewRun(g.id) }) : ""}</div></section>`;
  }
  // The game's timeline (ps_gameTimeline): sessions, save points, unlocks
  // and updates, by day, the latest first.
  const tlAsked = new Set(), TL_SHOWN = 40;
  function playSpan(sec) {
    const m = Math.max(1, Math.round(sec / 60));
    return m < 60 ? m + " min" : Math.floor(m / 60) + " h" + (m % 60 ? " " + (m % 60) + " min" : "");
  }
  function tlDay(at) {
    const d = new Date(at), now = new Date();
    const day = (x) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime();
    const diff = Math.round((day(now) - day(d)) / 86400000);
    if (diff === 0) return "Today";
    if (diff === 1) return "Yesterday";
    return d.toLocaleDateString(undefined, { weekday: "long", day: "numeric", month: "long", year: d.getFullYear() === now.getFullYear() ? undefined : "numeric" });
  }
  function tlRow(it, e) {
    const who = e.profile && player(e.profile) && cfg().players.length > 1 ? esc(pname(e.profile)) : "";
    const [ic, text] = {
      session: [GI.history, who ? `${who} played ${esc(playSpan(e.seconds || 0))}` : `Played ${esc(playSpan(e.seconds || 0))}`],
      point: [GI.save, who ? `${who} made a save point: ${esc(e.name)}` : `Save point: ${esc(e.name)}`],
      unlock: [GI.trophy, who ? `${who} unlocked ${esc(e.name)}` : `Unlocked: ${esc(e.name)}`],
      update: [GI.refresh, e.version ? `Updated to ${esc(e.version)}` : `Updated with ${esc(e.name)}`],
      shot: [GI.camera, "Screenshot"],
      capture: [e.ckind === "clip" ? GI.clip : GI.camera, esc(capWhat(e))],
    }[e.kind] || [];
    if (!text) return "";
    const hm = (t) => new Date(t).toLocaleTimeString(undefined, { hour: "numeric", minute: "2-digit" });
    const time = e.kind === "session" && e.start ? `${hm(e.start)} to ${hm(e.at)}` : hm(e.at);
    const pic = e.kind === "capture" ? capThumb(it, e) : e.kind === "shot" && e.thumb ? picButton(it, "shot:" + e.name, e.thumb, shotLabel(e)) : "";
    return `<li class="tl-row${pic ? " tl-row--shot" : ""}"><span class="tl-ic" aria-hidden="true">${ic}</span><span class="tl-what">${text}</span>${pic}<time class="tl-time ln-num" datetime="${esc(e.at)}">${esc(time)}</time></li>`;
  }
  function timelineBlock(it) {
    const key = recentKey(it);
    if (!tlAsked.has(key)) { tlAsked.add(key); api.gameTimeline(key).catch(() => {}); }
    const C = capturesFor(key), caps = ((C && C.items) || []).map((c) => ({ ...c, ckind: c.kind, kind: "capture" }));
    const list = [...(S.timeline[key] || []), ...caps].sort((a, b) => new Date(b.at) - new Date(a.at));
    if (!list.length) return "";
    // How many captures each app made, Lootnest's own screenshots included;
    // with none, how to make them.
    const counts = Object.entries({ ...((C && C.counts) || {}), Lootnest: list.filter((e) => e.kind === "shot").length }).filter(([, n]) => n);
    const tip = C && C.gameBarOff ? "Turn on Game Bar captures in Windows Settings, Gaming, Captures"
      : "Game Bar: Win+Alt+PrtScn for a screenshot, Win+Alt+R for a clip";
    const capsLine = counts.length ? `<p class="tl-caps"><span>Captures</span>${counts.map(([s, n]) => `<span class="ln-tag">${esc(s)} <b class="ln-num">${n}</b></span>`).join("")}</p>`
      : C ? `<p class="gpage-hint tl-tip">${tip}</p>` : "";
    const head = `<div class="ln-section-head"><div><h2>Timeline</h2>${capsLine}</div></div>`;
    const n = S.tlMore[key] || TL_SHOWN, days = [];
    for (const e of list.slice(0, n)) {
      const d = tlDay(e.at);
      if (!days.length || days[days.length - 1].d !== d) days.push({ d, rows: [] });
      days[days.length - 1].rows.push(tlRow(it, e));
    }
    return `<section class="gpage-block gp-timeline">${head}${days.map((g) => `<div class="tl-day"><h3 class="tl-date">${esc(g.d)}</h3><ol class="tl-list">${g.rows.join("")}</ol></div>`).join("")}
      ${list.length > n ? `<div class="gpage-actions"><button class="ln-button ln-button--ghost ln-button--sm" data-action="tl-more" data-key="${esc(key)}">Show more</button></div>` : ""}</section>`;
  }
  // Unsorted captures (Settings, Data and activity): the newest 50 Lootnest
  // couldn't match to a game, each given one here, or marked as of no game.
  // Only the index changes.
  const UNSORTED_SHOWN = 50;
  function viewUnsorted() {
    const U = S.unsorted, when = (c) => new Date(c.at).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
    const opts = `<option value="">Choose a game</option><option value="none">Not a game</option>${everyItem().sort((a, b) => byName(a.name, b.name)).map((it) => `<option value="${esc(recentKey(it))}">${esc(it.name)}</option>`).join("")}`;
    const body = !U ? lnInline("", "Reading the captures") : !U.items.length ? "<p>Every capture is matched to a game.</p>"
      : `<ul class="cap-list">${U.items.slice(0, UNSORTED_SHOWN).map((c) => `<li class="cap-row">${capThumb(null, c)}<div class="cap-main"><b>${esc(capWhat(c))}</b><span>${esc(when(c))}</span><span class="cap-file" title="${esc(c.path)}">${esc(c.path.split(/[\\/]/).pop())}</span></div>
          <select class="ln-input" data-assign="${esc(c.path)}" aria-label="${esc("The game of " + capLabel(c))}">${opts}</select></li>`).join("")}</ul>`;
    return drawer({ label: "Unsorted captures", title: "Unsorted captures", sub: U && U.total ? `${U.total} not matched to a game${U.total > UNSORTED_SHOWN ? `, the newest ${UNSORTED_SHOWN} here` : ""}` : "", body,
      foot: `<p class="gpage-hint ln-drawer__note">The files stay where they are.</p>` });
  }
  function gameSavesBlock(it) {
    if (it.kind !== "found" || it.source === "steam" || it.appId) return "";
    const head = `<div class="ln-section-head"><div><h2>Saves</h2></div></div>` + oneDriveNote(it, "gpage-hint");
    const G = S.gsaves[it.id];
    if (!G) { if (!gsAsked.has(it.id)) { gsAsked.add(it.id); api.gameSaves(it.id).catch(() => {}); } return `<section class="gpage-block">${head}${lnInline("", "Looking for its saves")}</section>`; }
    if (G.off === "steam") return "";
    if (G.error) return `<section class="gpage-block">${head}${lnInline("error", esc(G.error))}</section>`;
    const btn = (cls, attrs, label) => `<button class="ln-button ${cls} ln-button--sm" ${attrs}>${label}</button>`;
    const when = (v) => esc(new Date(v.created).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" }));
    // The Save Safety Net's copies of the game's saves, on this PC.
    const lv = G.local || [];
    const local = lv.length ? `<div class="gpage-sub">${gpHead("Copies on this PC")}<dl class="ln-facts">${lv.map((v, i) => `<div><dt><span><b>${when(v)}</b>
        <span class="gpage-hint">${esc(fmtBytes(v.size))}${i === 0 ? ", newest" : ""}</span></span></dt>
        <dd>${btn("ln-button--ghost", `data-action="gs-restore" data-local="1" data-key="${esc(it.id)}" data-id="${esc(v.id)}" data-when="${esc(v.created)}" ${G.busy ? "disabled" : ""}`, "Restore")}</dd></div>`).join("")}</dl></div>` : "";
    if (G.off === "nocloud") return `<section class="gpage-block">${head}${local}<p>Not in a cloud folder</p>
      <div class="gpage-actions"><button class="ln-button ln-button--secondary ln-button--sm" data-action="nav" data-view="cloud">Set up Cloud saves</button></div></section>`;
    if (G.off === "locked") return `<section class="gpage-block">${head}<p>The cloud folder has a password. Enter it on <b>Cloud saves</b>.</p>${local}</section>`;
    const s = cfg().settings;
    const choose = `<div class="gpage-actions">${btn("ln-button--secondary", `data-action="gs-add" data-key="${esc(it.id)}" data-kind="folder"`, G.locations.length ? "Add a folder" : "Choose the save folder")}
      ${btn("ln-button--ghost", `data-action="gs-add" data-key="${esc(it.id)}" data-kind="file"`, G.locations.length ? "Add a file" : "Choose a save file")}
      ${btn("ln-button--ghost", `data-action="gs-pcgw" data-name="${esc(it.name)}"`, "Look it up on PCGamingWiki")}</div>`;
    let where;
    if (!G.locations.length) {
      where = `<p>Save folder not set</p>
        ${choose}
        ${s.saveDb ? "" : `<p class="gpage-hint"><button class="gpage-link" data-action="savedb-on">Find save locations online</button></p>`}`;
    } else {
      const yours = G.source === "you";
      where = `${gpHead("On this PC")}<dl class="ln-facts gpage-paths">${G.locations.map((l) => `<div><dt class="selectable" title="${esc(l.resolved)}">${esc(l.path)}</dt><dd><span class="ln-num">${esc(fmtBytes(l.size || 0))}</span>
          ${btn("ln-button--ghost", `data-action="open-path" data-path="${esc(l.path)}"`, "Open")}${yours ? btn("ln-button--ghost", `data-action="gs-remove" data-key="${esc(it.id)}" data-path="${esc(l.path)}"`, "Remove") : ""}</dd></div>`).join("")}</dl>
        <p class="gpage-hint">${yours ? "Chosen by you" : G.source === "list" ? "From PCGamingWiki" : "Named like the game"}
          ${yours ? "" : `<button class="gpage-link" data-action="gs-own" data-key="${esc(it.id)}">Choose them yourself</button>`}</p>
        ${yours ? choose : ""}`;
    }
    const vs = G.versions || [];
    const inCloud = G.cloudDir ? `<div class="gpage-sub">${gpHead("In " + esc(G.provider || "your cloud folder"))}
      <dl class="ln-facts gpage-paths"><div><dt class="selectable">${esc(G.cloudDir)}</dt><dd>${btn("ln-button--ghost", `data-action="gs-open" data-key="${esc(it.id)}"`, "Open folder")}</dd></div></dl></div>` : "";
    const list = G.locations.length || vs.length ? `<div class="gpage-sub">${gpHead("Versions")}${vs.length ? `<dl class="ln-facts">${vs.map((v, i) => `<div><dt><span><b>${esc(new Date(v.created).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" }))}</b>
        <span class="gpage-hint">${esc(v.computer)}, ${esc(fmtBytes(v.size))}${i === 0 ? ", newest" : ""}</span></span></dt>
        <dd>${btn("ln-button--ghost", `data-action="gs-restore" data-key="${esc(it.id)}" data-id="${esc(v.id)}" data-when="${esc(v.created)}" data-pc="${esc(v.computer)}" ${G.busy ? "disabled" : ""}`, "Restore")}</dd></div>`).join("")}</dl>`
        : `<p class="gpage-hint">No versions yet</p>`}
      ${G.off === "disabled" ? `<p class="gpage-hint">Automatic versions are off</p>` : ""}
      ${G.locations.length ? `<div class="gpage-actions">${G.busy ? `<button class="ln-button ln-button--secondary ln-button--sm" aria-busy="true"><span class="ln-spinner" aria-hidden="true"></span>Saving</button>` : btn("ln-button--secondary", `data-action="gs-now" data-key="${esc(it.id)}"`, "Back up now")}</div>` : ""}</div>` : "";
    return `<section class="gpage-block">${head}${where}${inCloud}${list}${local}</section>`;
  }


  // Game details from the Steam store (ps_gameDetails, opt-in): the copy
  // kept on this PC, with the user's own edits over it.
  function detailsBlock(it) {
    const key = recentKey(it);
    const steam = cfg().settings.gameDetails;
    const offer = !steam && it.appId ? `<p class="gp-det-off"><button class="gpage-link" data-action="details-on">Show details from Steam</button></p>` : "";
    if (!steam && !S.d.pluginMeta) return offer;
    if (!S.details[key]) { S.details[key] = { loading: true }; api.gameDetails(key).catch(() => {}); }
    const R = S.details[key], head = `<div class="ln-section-head"><div><h2>Details</h2></div></div>`;
    if (R.loading) return it.appId && steam ? `<section class="gpage-block gp-det">${head}${lnInline("", "Reading the details from Steam")}</section>` : "";
    if (R.error && !R.details) return `<section class="gpage-block gp-det">${head}${lnInline("error", esc(R.error))}</section>`;
    const d = R.details;
    if (!d) return offer;
    const open = S.detOpen.has(key), sum = d.summary || "";
    const facts = [
      ["Developer", esc((d.developers || []).join(", "))],
      ["Publisher", esc((d.publishers || []).join(", "))],
      ["Released", esc(d.released || "")],
      ["Genres", esc((d.genres || []).join(", "))],
      ["Steam reviews", d.reviewScore ? `${esc(d.reviewScore)}<span class="gp-det-n">${d.reviewPercent}% of ${d.reviewCount.toLocaleString("en-US")} reviews</span>` : ""],
      ["Metacritic", d.metacritic ? String(d.metacritic) : ""],
    ].filter((f) => f[1]);
    const also = d.from ? `, and ${esc(d.from)}` : "", edits = R.edited ? ", with your edits" : "";
    const from = d.storeUrl ? `From the Steam store${also}${edits}. <button class="gpage-link" data-action="open-link" data-url="${esc(d.storeUrl)}">Open on Steam</button>` : d.from ? `From ${esc(d.from)}${edits}.` : "Your own details.";
    return `<section class="gpage-block gp-det">${mediaBlock(it, d)}${head}<div class="gp-det-cols"><div class="gp-det-main">
        ${sum ? `<p class="gp-det-sum${open ? " open" : ""}" id="det-sum">${esc(sum)}</p><button class="gpage-link" data-action="details-more" data-k="${esc(key)}" aria-expanded="${open}" aria-controls="det-sum"${open ? "" : " hidden"}>${open ? "Show less" : "Show more"}</button>` : ""}
        ${gameCtlLine(d)}<p class="gpage-hint">${from}${R.fetching ? " Checking for newer details." : ""}</p></div>
        ${facts.length ? `<dl class="ln-facts gp-det-facts">${facts.map(([k, v]) => `<div><dt>${k}</dt><dd>${v}</dd></div>`).join("")}</dl>` : ""}</div></section>`;
  }
  // Community, at the end of Overview: what players say on Steam, the most
  // helpful reviews of the past year. Read only once the block scrolls into
  // view (watchCommunity), and only with Game details on. Laid out as the
  // page's other lists: a head with the share that recommends it and a
  // bar, then reviews as rows, each led by how long its writer had played.
  const REV_KINDS = [["all", "Most helpful"], ["positive", "Recommended"], ["negative", "Not recommended"]];
  const REV_SHOWN = 4;
  function communityBlock(it) {
    const key = recentKey(it), steam = cfg().settings.gameDetails;
    const D = S.details[key] || {}, known = it.appId || D.appId || (D.details && D.details.storeUrl);
    if (!known) return "";
    const head = (right) => `<div class="ln-section-head"><div><h2>Community</h2></div>${right || ""}</div>`;
    if (!steam) return `<section class="gpage-block gp-com">${head()}<p class="gp-det-off"><button class="gpage-link" data-action="details-on">Show what players say on Steam</button></p></section>`;
    const kind = S.revKind[key] || "all", R = S.reviews[key + "|" + kind];
    const d = D.details || {};
    // Steam's words and share: from the details, else from the reviews themselves.
    const sum = d.reviewCount ? { score: d.reviewScore, percent: d.reviewPercent, count: d.reviewCount } : (S.reviews[key + "|all"] || {}).summary;
    const pct = sum ? sum.percent : null;
    const count = pct == null ? "" : `<span class="ach-count"><b class="ln-num">${pct}%</b> ${esc(sum.score || "recommend it")}</span>`;
    const bar = pct == null ? "" : `<div class="gp-com-bar" role="img" aria-label="${pct}% of ${sum.count.toLocaleString("en-US")} reviews recommend it"><span style="width:${pct}%"></span></div>`;
    const seg = `<div class="ln-segmented gp-com-seg" role="group" aria-label="Show">${REV_KINDS.map(([k, l]) => `<button data-action="rev-kind" data-k="${esc(key)}" data-kind="${k}" aria-pressed="${k === kind}">${l}</button>`).join("")}</div>`;
    const tools = `<div class="gp-com-tools">${seg}${sum ? `<span class="gpage-hint">${sum.count.toLocaleString("en-US")} reviews on Steam</span>` : ""}</div>`;
    const store = (R && R.appId) || it.appId || D.appId;
    const steamLink = store ? `<button class="ln-button ln-button--ghost ln-button--sm" data-action="open-link" data-url="https://store.steampowered.com/app/${store}/#app_reviews_hash">${ph("arrow-square-out")}Read more on Steam</button>` : "";
    let body, more = "";
    if (!R || R.loading) body = `<ol class="gp-com-list" data-want="${esc(key)}" data-kind="${kind}">${[0, 1, 2].map(() => '<li class="gp-rev gp-rev--skel"><span class="ln-skeleton gp-rev-skel-h"></span><span class="ln-skeleton gp-rev-skel-t"></span></li>').join("")}</ol>`;
    else if (R.error) body = `${lnInline("error", esc(R.error))}<button class="gpage-link" data-action="rev-again" data-k="${esc(key)}" data-kind="${kind}">Try again</button>`;
    else if (R.none) return "";
    else if (!(R.items || []).length) body = `<p class="gpage-hint gp-com-none">No ${kind === "positive" ? "recommending " : kind === "negative" ? "negative " : ""}reviews with text from the past year.</p>`;
    else {
      const all = S.revAll.has(key), list = all ? R.items : R.items.slice(0, REV_SHOWN);
      body = `<ol class="gp-com-list">${list.map(reviewRow).join("")}</ol>`;
      if (R.items.length > REV_SHOWN) more = `<button class="ln-button ln-button--secondary ln-button--sm" data-action="rev-all" data-k="${esc(key)}">${all ? "Fewer reviews" : `${R.items.length - REV_SHOWN} more reviews`}</button>`;
    }
    return `<section class="gpage-block gp-com" id="gp-community">${head(count)}${bar}${tools}${body}${more || steamLink ? `<div class="gp-com-foot">${more}<span class="grow"></span>${steamLink}</div>` : ""}</section>`;
  }
  // One review: its writer's hours when they wrote it, large, then the
  // words (cut at a few lines until opened) and what Steam says about it.
  function reviewRow(r) {
    const open = S.revOpen.has(r.id), long = r.cut || r.text.length > 360 || (r.text.match(/\n/g) || []).length > 4;
    const h = (n) => (n >= 10 ? Math.round(n).toLocaleString("en-US") : String(n));
    const facts = [r.total > r.hours ? `${h(r.total)} h now` : "", r.early ? "Early Access" : "", r.free ? "Got it free" : ""].filter(Boolean);
    return `<li class="gp-rev gp-rev--${r.up ? "up" : "down"}${open ? " open" : ""}">
      <span class="gp-rev-hours"><b class="ln-num">${esc(h(r.hours))}<small>h</small></b><span>when written</span></span>
      <div class="gp-rev-main">
        <span class="gp-rev-say">${ph(r.up ? "thumbs-up" : "thumbs-down")}<b>${r.up ? "Recommended" : "Not recommended"}</b><time datetime="${esc(r.at)}">${esc(day(r.at))}</time></span>
        <p class="gp-rev-text" id="rev-${esc(r.id)}">${esc(r.text)}</p>
        <span class="gp-rev-facts">${facts.map((f) => `<span>${esc(f)}</span>`).join("")}${r.helpful ? `<span><b class="ln-num">${r.helpful.toLocaleString("en-US")}</b> found it helpful</span>` : ""}${long ? `<button class="gpage-link" data-action="rev-more" data-id="${esc(r.id)}" aria-expanded="${open}" aria-controls="rev-${esc(r.id)}">${open ? "Show less" : "Read all of it"}</button>` : ""}</span>
      </div></li>`;
  }
  // Asks for the reviews once the Community block comes into view.
  let comSeen = null;
  function watchCommunity() {
    if (comSeen) { comSeen.disconnect(); comSeen = null; }
    const el = $("[data-want]");
    if (!el) return;
    const ask = () => {
      const key = el.dataset.want, kind = el.dataset.kind, k = key + "|" + kind;
      if (S.reviews[k]) return;
      S.reviews[k] = { loading: true };
      api.gameReviews(key, kind).catch(() => { S.reviews[k] = { key, kind, error: "Couldn't read the reviews." }; render(); });
    };
    if (!("IntersectionObserver" in window)) { ask(); return; }
    comSeen = new IntersectionObserver((es) => { if (es.some((e) => e.isIntersecting)) { comSeen.disconnect(); comSeen = null; ask(); } }, { rootMargin: "0px 0px 200px 0px" });
    comSeen.observe(el);
  }
  // Trailers and screenshots: the one in focus large on a stage, the rest in
  // a strip beside it. Trailers play muted (Steam serves them only as HLS,
  // so hls.js plays them; it's loaded the first time it's needed);
  // screenshots move on by themselves every few seconds unless the pointer
  // rests on them or one was just picked; a screenshot on the stage opens
  // large. The video element is kept between redraws, so a redraw never
  // restarts a trailer.
  const NONCE = (document.currentScript && document.currentScript.nonce) || "";
  const MEDIA_EVERY = 6000, MEDIA_HOLD = 15000;
  let hlsReady = null, mediaTimer = 0;
  const vids = new Map(); // "key|n" -> { el, hls }
  function loadHls() {
    if (window.Hls) return Promise.resolve(window.Hls);
    if (!hlsReady) hlsReady = api.vendor("hls").then((src) => {
      const sc = document.createElement("script");
      sc.nonce = NONCE; sc.textContent = src; document.head.append(sc);
      return window.Hls;
    });
    return hlsReady;
  }
  function mediaList(d) {
    return [...(d.movies || []).map((m) => ({ kind: "movie", thumb: m.thumb, src: m.hls, name: m.name })),
      ...(d.screenshots || []).map((x, i) => ({ kind: "shot", thumb: x.thumb, full: x.full, i }))];
  }
  function mediaBlock(it, d) {
    const list = mediaList(d);
    if (!list.length) return "";
    const key = recentKey(it), M = S.media[key] || (S.media[key] = { i: 0, hold: 0 });
    Object.assign(M, { it, d, list });
    M.i = Math.min(M.i, list.length - 1);
    const cur = list[M.i];
    const stage = cur.kind === "movie"
      ? `<div class="gp-stage is-movie" id="gp-stage" data-k="${esc(key)}"><img src="${esc(cur.thumb)}" alt=""></div>`
      : `<button class="gp-stage" id="gp-stage" data-k="${esc(key)}" data-action="shot-open" ${itemAttr(it)} data-i="${cur.i}" aria-label="Screenshot ${cur.i + 1}, open it large" aria-haspopup="dialog"><img src="${esc(cur.full)}" alt=""><span class="gp-stage-zoom" aria-hidden="true">${ph("arrows-out")}</span></button>`;
    const strip = list.map((m, n) => `<button class="gp-thumb${n === M.i ? " on" : ""}" data-action="media-pick" data-k="${esc(key)}" data-n="${n}" aria-pressed="${n === M.i}" aria-label="${m.kind === "movie" ? esc(m.name || "Trailer") : `Screenshot ${m.i + 1}`}">
        <img src="${esc(m.thumb)}" alt="" loading="lazy">${m.kind === "movie" ? `<span class="gp-thumb-play" aria-hidden="true">${ph("play", "fill")}</span>` : ""}</button>`).join("");
    return `<div class="gp-media">${stage}<div class="gp-strip" role="group" aria-label="Trailers and screenshots">${strip}</div></div>`;
  }
  function dropVideos(keep) {
    vids.forEach((v, k) => { if (k === keep) return; try { if (v.hls) v.hls.destroy(); v.el.pause(); v.el.removeAttribute("src"); v.el.load(); } catch (e) { /* gone */ } vids.delete(k); });
  }
  // After every redraw of a game's page: the trailer back on its stage, or
  // the next screenshot on its timer.
  function mountMedia() {
    clearTimeout(mediaTimer);
    const st = $("#gp-stage");
    if (!st) { dropVideos(""); return; }
    const key = st.dataset.k, M = S.media[key];
    if (!M) return;
    const cur = M.list[M.i];
    if (cur.kind !== "movie") {
      dropVideos("");
      if (!reduced() && M.list.length > 1) mediaTimer = setTimeout(() => mediaAuto(key), MEDIA_EVERY);
      return;
    }
    const id = key + "|" + M.i;
    dropVideos(id);
    let v = vids.get(id);
    if (!v) {
      const el = document.createElement("video");
      el.className = "gp-video"; el.muted = true; el.playsInline = true; el.controls = true; el.poster = cur.thumb; el.preload = "auto";
      el.addEventListener("ended", () => { if (M.list.length > 1) mediaGo(key, (M.i + 1) % M.list.length); });
      v = { el };
      vids.set(id, v);
      loadHls().then((Hls) => {
        if (vids.get(id) !== v) return;
        if (Hls && Hls.isSupported()) {
          const h = new Hls({ enableWorker: false, capLevelToPlayerSize: true });
          h.on(Hls.Events.ERROR, (_, e) => { if (e.fatal) { h.destroy(); el.closest(".gp-stage")?.classList.add("is-broken"); } });
          h.loadSource(cur.src); h.attachMedia(el); v.hls = h;
        } else if (el.canPlayType("application/vnd.apple.mpegurl")) el.src = cur.src;
        if (!reduced()) el.play().catch(() => {});
      }).catch(() => st.classList.add("is-broken"));
    }
    st.replaceChildren(v.el);
    if (v.hls && !reduced() && !v.el.ended) v.el.play().catch(() => {});
  }
  // Screenshots move on by themselves, but not under the pointer, not right
  // after one was picked, and not off screen.
  function mediaAuto(key) {
    const M = S.media[key], box = $(".gp-media");
    if (!M || !box) return;
    const r = box.getBoundingClientRect();
    if (box.matches(":hover") || Date.now() < M.hold || document.hidden || r.bottom < 0 || r.top > innerHeight) { mediaTimer = setTimeout(() => mediaAuto(key), MEDIA_EVERY); return; }
    // Trailers are only played when picked or first; the timer goes round the screenshots.
    let n = (M.i + 1) % M.list.length;
    while (M.list[n].kind === "movie" && n !== M.i) n = (n + 1) % M.list.length;
    mediaGo(key, n);
  }
  // Puts item n on the stage without drawing the whole page again.
  function mediaGo(key, n) {
    const M = S.media[key], box = $(".gp-media");
    if (!M || !box || n === M.i) return;
    const back = n < M.i;
    M.i = n;
    box.outerHTML = mediaBlock(M.it, M.d);
    mountMedia();
    const st = $("#gp-stage"), on = $(".gp-thumb.on");
    if (st && moving()) {
      st.animate([{ opacity: 0, transform: `translateX(${back ? -24 : 24}px) scale(0.985)`, filter: "blur(6px)" }, { opacity: 1, transform: "none", filter: "none" }], { duration: SPRINGS.soft.ms * 0.8, easing: SPRINGS.soft.ease });
    }
    if (on) on.scrollIntoView({ block: "nearest", inline: "nearest", behavior: reduced() ? "auto" : "smooth" });
  }
  // Show more only when the three lines don't hold the whole summary.
  function fitDetails() {
    const p = $("#det-sum"), b = $('[data-action="details-more"]');
    if (p && b) b.hidden = !p.classList.contains("open") && p.scrollHeight <= p.clientHeight;
  }
  // Edit details: the fields the user changes win over Steam's.
  function viewDetailsEdit() {
    const D = S.dialog, it = dialogItem(D);
    const R = it && S.details[recentKey(it)];
    if (!it) return "";
    const d = (R && R.details) || {}, typed = D.vals || {};
    const field = (id, label, value, hint) => `<div class="field"><label for="${id}">${label}</label><input id="${id}" type="text" value="${esc(typed[id] ?? value ?? "")}" maxlength="200" spellcheck="false">${hint ? `<p class="hint">${hint}</p>` : ""}</div>`;
    return `<div class="scrim" data-action="scrim"><div class="dialog narrow" role="dialog" aria-modal="true" aria-label="Edit details of ${esc(it.name)}">
      <div><h2>Edit details</h2><p class="sub">Your changes win over Steam's.</p></div>
      <div class="field"><label for="de-summary">Summary</label><textarea id="de-summary" rows="4" maxlength="2000">${esc(typed["de-summary"] ?? d.summary ?? "")}</textarea></div>
      ${field("de-developer", "Developer", (d.developers || []).join(", "))}
      ${field("de-publisher", "Publisher", (d.publishers || []).join(", "))}
      ${field("de-released", "Released", d.released)}
      ${field("de-genres", "Genres", (d.genres || []).join(", "), "Separated by commas.")}
      <div class="field"><label for="de-controller">Controller</label><select id="de-controller" class="ln-input">${[["", "Steam's"], ["full", "Full"], ["partial", "Partial"], ["none", "Keyboard and mouse only"]].map(([v, l]) => `<option value="${v}"${(typed["de-controller"] ?? d.controller ?? "") === v ? " selected" : ""}>${l}</option>`).join("")}</select></div>
      <div class="dialog-foot">${R && R.edited ? `<button class="btn quiet" data-action="details-reset" ${itemAttr(it)} ${D.busy ? "disabled" : ""}>Use Steam's again</button><span class="grow"></span>` : ""}
        <button class="btn quiet" data-action="dialog-cancel" ${D.busy ? "disabled" : ""}>Cancel</button>
        <button class="btn primary" data-action="details-save" ${itemAttr(it)} ${D.busy ? "disabled" : ""}>Save</button></div></div></div>`;
  }
  // The screenshots, one at a time: the full picture loads only here. With
  // D.pics, the game's own pictures (save points', screenshots'), read from
  // this PC by ps_picture; the thumbnail shows until it arrives.
  function viewShots() {
    const D = S.dialog, it = dialogItem(D);
    const R = it && S.details[recentKey(it)];
    const shots = D.pics ? D.pics.map((p) => ({ full: D.full[p.ref] || p.thumb, alt: p.label })) : (R && R.details && R.details.screenshots) || [];
    if (!shots.length) return "";
    const i = ((D.i % shots.length) + shots.length) % shots.length, n = shots.length;
    const want = D.pics && D.pics[i].ref;
    if (want && !(want in D.full)) {
      D.full[want] = "";
      const done = (u) => { D.full[want] = u; if (S.dialog === D) renderOverlay(); };
      api.picture(recentKey(it), want).then(done, () => done(null));
    }
    const what = D.pics ? "picture" : "screenshot", gone = want && D.full[want] === null;
    const nav = (dir, label, ic) => `<button class="ln-button ln-button--secondary ln-button--icon shots-nav" data-action="shot-step" data-d="${dir}" aria-label="${label}"${n < 2 ? " disabled" : ""}>${ic}</button>`;
    return `<div class="scrim" data-action="scrim"><div class="dialog shots" role="dialog" aria-modal="true" aria-label="${D.pics ? "Pictures" : "Screenshots"} of ${esc(it.name)}">
      <header class="shots-head"><h2>${esc(it.name)}</h2><span class="ln-num" aria-live="polite">${D.pics ? esc(shots[i].alt) + ", " : ""}${i + 1} of ${n}</span><span class="grow"></span>
        <button class="ln-button ln-button--ghost ln-button--icon ln-button--sm" data-action="dialog-cancel" aria-label="Close">${GI.x}</button></header>
      <div class="shots-stage">${nav(-1, `Previous ${what}`, GI.back)}${gone ? `<p class="shots-gone">That picture is gone, or too big to show.</p>` : !shots[i].full ? `<p class="shots-gone">Reading the picture</p>` : `<img src="${esc(shots[i].full)}" alt="${esc(shots[i].alt || `Screenshot ${i + 1} of ${n}`)}">`}${nav(1, `Next ${what}`, GI.next)}</div></div></div>`;
  }
