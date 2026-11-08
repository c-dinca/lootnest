  // --------------------------------------------------------------- players --
  // First run: the one profile Lootnest starts with, named from Windows.
  // More people and runs come later, from Profiles.
  const SWATCHES = (p, i) => `<div class="swatches" role="radiogroup" aria-label="Colour for ${esc(p.name || "player " + (i + 1))}">${S.d.colors.map((c) => `<button class="swatch" style="background:${c}" data-action="swatch" data-i="${i}" data-color="${c}" aria-pressed="${c.toLowerCase() === (p.color || "").toLowerCase()}" aria-label="${c}"></button>`).join("")}</div>`;
  const welcomeName = (p) => (p.name || "").trim() || "Player 1";
  function welcomeLive() {
    const p = S.players[0], big = $("#wl-av");
    if (big) big.innerHTML = lnAvatar({ ...p, name: welcomeName(p) }, "72");
  }
  // The first scan: a bar that runs while it looks, full once it's done.
  // Drawn once; later scan events only set its state and text.
  function welcomeScan(el) {
    const n = libraryItems().length, busy = S.found.scanning || S.found.refreshing;
    const text = busy ? "Looking for games on this PC" : n ? `Found ${n} game${n === 1 ? "" : "s"}` : "No games found on this PC yet";
    if (!el) return `<span class="wl-meter${busy ? "" : " is-done"}" aria-hidden="true"><i></i></span><span>${text}</span>`;
    el.firstElementChild.classList.toggle("is-done", !busy);
    el.lastElementChild.textContent = text;
  }
  // First start, as a console boots: the gem, one line, the scan, and the
  // one choice that matters, who you are.
  function viewWelcome() {
    if (S.pnFound === undefined) {
      S.pnFound = "";
      api.playniteFind().then((r) => { S.pnFound = r.dir || ""; const n = $("#wl-pn"); if (n) n.hidden = !S.pnFound; }, () => {});
    }
    const ps = S.players || (S.players = JSON.parse(JSON.stringify(cfg().players)));
    const p = ps[0];
    return `<div class="welcome ln-root" role="dialog" aria-modal="true" aria-labelledby="wl-title">
      <span class="wl-gem">${BRAND_MARK}</span>
      <h1 id="wl-title" class="wl-in" style="--i:0">Welcome to Lootnest</h1>
      <p class="wl-lead wl-in" style="--i:1">Every game on this PC, each profile with its own saves.</p>
      <p class="wl-scan wl-in" style="--i:2" id="wl-scan" aria-live="polite">${welcomeScan()}</p>
      <div class="wl-me wl-in ln-player" style="--i:3;--player:${esc(p.color || "#6c9bd2")}">
        <span id="wl-av">${lnAvatar({ ...p, name: welcomeName(p) }, "72")}</span>
        <div class="wl-who">
          <input type="text" class="wl-name" id="wl-name" data-field="pname" data-i="0" value="${/^Player \d$/.test(p.name) ? "" : esc(p.name)}" placeholder="Your name" maxlength="24" aria-label="Your name">
          ${SWATCHES(p, 0)}
        </div>
      </div>
      <div class="wl-go wl-in" style="--i:4"><button class="ln-button ln-button--primary ln-button--lg" data-action="welcome-done">Continue</button><span class="error-text" id="players-error"></span></div>
      <p class="wl-note" id="wl-pn" ${S.pnFound ? "" : "hidden"}>Playnite is on this PC. Bring in its time played later, in Settings.</p>
    </div>`;
  }

  function playerRow(p, i, total) {
    // While editing, a row can be a run of a person in the list (not of
    // itself, not of a run, and not while it has runs of its own).
    const list = S.players || [];
    const par = p.of && list.find((x) => x.id === p.of && x.id !== p.id && !x.of);
    const hasRuns = p.id && list.some((x) => x.of === p.id);
    const people = list.filter((x) => x !== p && !x.of && x.id && x.name.trim());
    const of = `<select class="ln-input player-of" data-field="pof" data-i="${i}" aria-label="Whose profile" ${hasRuns ? "disabled title=\"It has runs of its own\"" : ""}>
        <option value="">A person</option>${people.map((x) => `<option value="${esc(x.id)}" ${par && par.id === x.id ? "selected" : ""}>A run for ${esc(x.name)}</option>`).join("")}</select>`;
    return `<div class="player ln-player${par ? " player-run" : ""}" style="--player:${esc(p.color || "#6c9bd2")}">
      ${lnAvatar({ ...p, name: p.name || "?" }, "32")}
      <input type="text" class="ln-input" data-field="pname" data-i="${i}" value="${esc(p.name)}" placeholder="${par ? par.name + " hardcore" : "Name"}" maxlength="24" aria-label="Name">
      ${of}
      ${par ? `<span class="player-same">${esc(possessive(par.name))} colour</span>` : SWATCHES(p, i)}
      <button class="ln-button ln-button--ghost ln-button--icon ln-button--sm" data-action="remove-player" data-i="${i}" aria-label="Remove ${esc(p.name)}" title="Remove" ${total <= 1 ? "disabled" : ""}>${GI.x}</button>
    </div>`;
  }

  // ------------------------------------------------------------ profiles --
  // Each player's own page, which they arrange themselves: a banner, their
  // picture, a few words, the game they're into, their numbers, activity
  // and collection.
  const SECTIONS = [
    ["now", "Currently playing"],
    ["stats", "Numbers"],
    ["favorites", "Favourite games"],
    ["activity", "Activity"],
    ["collection", "Collection"],
    ["recent", "Played lately"],
  ];
  const sectionsOf = (p) => { const known = SECTIONS.map((s) => s[0]), on = (p.sections || []).filter((s) => known.includes(s)); return on.length ? on : known; };
  const daysOf = (pid) => ((S.d.activity || {}).days || {})[pid] || {};
  const dayKey = (d) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
  const daySum = (day) => Object.values(day || {}).reduce((a, b) => a + b, 0);

  const avatarHTML = (p, cls) => {
    const u = (S.d.avatars || {})[p.id];
    const g = !u && p.avatarGame && itemByKey(p.avatarGame);
    if (g) ensureCover(g);
    const gu = g && (artFor(g, "cover").url || S.icons[coverKey(g)]);
    const inner = u ? `<img src="${u}" alt="">` : gu ? `<img src="${gu}" alt="">` : esc(p.name.trim().charAt(0).toUpperCase());
    return `<span class="avatar ln-player ${cls || ""}" style="--c:${esc(p.color)}" aria-hidden="true">${inner}</span>`;
  };
  const itemByKey = (k) => everyItem().find((it) => recentKey(it) === k || it.id === k);
  const itemAttr = (it) => (it.kind === "game" ? `data-game="${esc(it.id)}"` : `data-key="${esc(it.id)}"`);

  function playerStats(p) {
    const items = everyItem();
    let total = 0, top = null, topT = 0, played = 0;
    items.forEach((it) => { const t = (playtimeOf(it) || {})[p.id] || 0; total += t; if (t) played++; if (t > topT) { top = it; topT = t; } });
    // Their own last time in each game, not whoever played it last.
    const recent = items.map((it) => ({ it, r: { at: lastBy(it, p.id), player: p.id } })).filter((x) => x.r.at)
      .sort((a, b) => new Date(b.r.at) - new Date(a.r.at));
    played = Math.max(played, recent.length);
    const profiles = cfg().games.filter((g) => hasProfile(g.id, p.id)).length;
    // Streaks: days in a row with some play, counted to today (or yesterday).
    const days = daysOf(p.id), keys = Object.keys(days).filter((k) => daySum(days[k]) > 0).sort();
    let best = 0, run = 0, prev = null;
    keys.forEach((k) => { const d = new Date(k + "T12:00:00"); run = prev && (d - prev) / 864e5 < 1.5 ? run + 1 : 1; best = Math.max(best, run); prev = d; });
    let streak = 0;
    for (let d = new Date(), i = 0; i < 400; i++, d.setDate(d.getDate() - 1)) {
      if (daySum(days[dayKey(d)]) > 0) streak++;
      else if (i > 0 || streak > 0) break;
    }
    const act = S.d.activity || {};
    const longest = (act.longest || {})[p.id], sessions = (act.sessions || {})[p.id] || 0;
    const sources = new Set(items.filter((it) => ((playtimeOf(it) || {})[p.id] || 0) > 0 || recent.some((x) => x.it === it)).map((it) => sourceLabel(it.source, it.f)));
    return { total, top, topT, recent, profiles, played, streak, best, longest, sessions, sources };
  }

  // top (null when none): the most played game, whose art stands in when
  // they chose no banner; without it the page has none.
  function bannerHTML(p, top) {
    const y = `background-position:center ${p.bannerY || 0}%`;
    if (!p.banner && top !== undefined) { const u = top && (loadArt(top), artOf(top)); return u ? `<div class="art auto" style="background-image:url('${u}')"></div>` : `<div class="art plain none"></div>`; }
    if (p.banner === "image" && (S.d.banners || {})[p.id]) return `<div class="art" style="background-image:url('${S.d.banners[p.id]}');${y}"></div>`;
    const it = p.banner && p.banner !== "image" && itemByKey(p.banner);
    if (it) { loadArt(it); ensureCover(it); }
    const u = it ? artOf(it) : "";
    return u ? `<div class="art" style="background-image:url('${u}');${y}"></div>` : `<div class="art plain"></div>`;
  }

  function viewPlayer() {
    const p = player(S.playerPage);
    if (!p) { S.view = "players"; return viewPlayers(); }
    const st = playerStats(p);
    const last = st.recent.find((x) => x.it.kind === "game" && !session(x.it.id));
    const secs = sectionsOf(p), main = secs.filter((s) => s !== "stats").map((s) => ppSection(s, p, st)).join("");
    const stats = secs.includes("stats") ? ppSection("stats", p, st) : "";
    const pickFavs = secs.includes("favorites") && libraryItems().length && !(p.favorites || []).some(itemByKey);
    return `<div class="profile-page ln-root ln-player" style="--player:${esc(p.color)}">
      <button class="ln-button ln-button--ghost ln-button--sm gpage-back" data-action="nav" data-view="players">${GI.back}Profiles</button>
      <div class="pp-banner">${bannerHTML(p, st.top || null)}</div>
      <div class="pp-head">${lnAvatar(p, "120")}
        <div class="pp-name"><h1>${esc(p.name)}</h1>${p.tagline ? `<p class="pp-tag">${esc(p.tagline)}</p>` : ""}${p.bio ? `<p class="pp-bio">${esc(p.bio).replace(/\n/g, "<br>")}</p>` : ""}</div>
        <div class="pp-actions">${last ? `<button class="ln-button ln-button--play ln-button--lg" data-action="play-as" data-game="${esc(last.it.id)}" data-player="${esc(p.id)}">${GI.play}Play ${esc(last.it.name)}</button>` : ""}
          <button class="ln-button ln-button--secondary" data-action="edit-profile" data-player="${esc(p.id)}">Edit profile</button>
          ${pickFavs ? `<button class="ln-button ln-button--ghost" data-action="edit-profile" data-player="${esc(p.id)}" data-tab="showcase">Pick favourites</button>` : ""}
          <button class="ln-button ln-button--ghost" data-action="recap" data-player="${esc(p.id)}">Year in games</button>
          <button class="ln-button ln-button--ghost" data-action="export-player" data-player="${esc(p.id)}">Export profile</button></div></div>
      <div class="pp-body">${main || stats ? `${main ? `<div class="pp-main">${main}</div>` : ""}${stats ? `<aside class="pp-side">${stats}</aside>` : ""}` : `<p class="hint">Nothing to show yet: <b>Edit profile</b> chooses what this page shows.</p>`}</div>
    </div>`;
  }

