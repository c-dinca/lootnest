  // A game outside Steam without profiles: its saves, kept as versions in
  // the cloud folder (Steam games have Steam Cloud; games with profiles sync
  // each player's saves).
  const gsAsked = new Set();
  // Status and collections: where the game is for one profile (playing,
  // finished, 100%), and which of the user's collections it's in.
  function collectionsBlock(it) {
    const key = recentKey(it), ps = grouped();
    if (!ps.length) return "";
    const seat = seated(), owner = it.g && S.d.active[it.g.id];
    const who = (ps.find((p) => p.id === S.statusWho[key]) || (seat && ps.find((p) => p.id === seat.id)) || ps.find((p) => p.id === owner) || ps[0]).id;
    const st = statusOf(it, who), mine = shelfOf(it, who), r = mine.rating || 0, tags = tagsOf(it), others = allTags().filter((t) => !tags.includes(t));
    const chips = ps.length > 1 ? `<div class="ach-who" role="group" aria-label="Whose status">${ps.map((p) => `<button class="ln-chip ln-player" style="--player:${esc(p.color)}" data-action="status-who" data-key="${esc(key)}" data-player="${esc(p.id)}" aria-pressed="${p.id === who}">${lnAvatar(p, "20")}${esc(parentOf(p) ? runLabel(p) : p.name)}${statusOf(it, p.id) ? `<span class="ln-num">${esc(statusName(statusOf(it, p.id)))}</span>` : ""}</button>`).join("")}</div>` : "";
    return `<section class="gpage-block gp-coll"><div class="ln-section-head"><div><h2>Status</h2></div></div>${chips}
      <div class="ln-segmented gp-status" role="group" aria-label="${esc(possessive(pname(who)))} status">${STATUSES.map(([k, l]) => `<button data-action="set-status" data-key="${esc(key)}" data-player="${esc(who)}" data-status="${k}" aria-pressed="${st === k}">${l}</button>`).join("")}</div>
      <div class="gp-rate" role="radiogroup" aria-label="${esc(possessive(pname(who)))} rating">${[1, 2, 3, 4, 5].map((i) => `<button role="radio" aria-checked="${r === i}" tabindex="${(r || 1) === i ? 0 : -1}"${i <= r ? ' class="on"' : ""} data-action="rate" data-key="${esc(key)}" data-player="${esc(who)}" data-stars="${i}" aria-label="${i} ${i === 1 ? "star" : "stars"}" title="${r === i ? "Clear the rating" : i + (i === 1 ? " star" : " stars")}">${GI.star}</button>`).join("")}</div>
      <input id="gp-note" class="ln-input gp-note" type="text" maxlength="280" autocomplete="off" placeholder="A note, just for you" aria-label="${esc(possessive(pname(who)))} note" value="${esc(mine.note || "")}" data-key="${esc(key)}" data-player="${esc(who)}">
      <div class="ln-section-head ln-section-head--minor"><h2>Collections</h2></div>
      <div class="gp-tags">${tags.map((t) => `<span class="ln-tag gp-tag">${esc(t)}<button data-action="tag-remove" data-key="${esc(key)}" data-tag="${esc(t)}" aria-label="Take it out of ${esc(t)}">${GI.x}</button></span>`).join("") || `<span class="gpage-hint">In no collection yet.</span>`}</div>
      ${tags.length < 8 ? `<div class="gp-tag-add"><input id="tag-new" class="ln-input" type="text" maxlength="40" list="tag-list" autocomplete="off" placeholder="Add to a collection, such as Co-op" data-key="${esc(key)}">
        <datalist id="tag-list">${others.map((t) => `<option value="${esc(t)}"></option>`).join("")}</datalist>
        <button class="ln-button ln-button--secondary ln-button--sm" data-action="tag-add" data-key="${esc(key)}">Add</button></div>` : ""}</section>`;
  }

  // Achievements: read from this PC (Steam's library cache, or the file a
  // Steam-compatible tool keeps next to the game's saves). The latest
  // unlocked first; the whole list on request.
  const achAsked = new Set();
  // Whose achievements a game shows, and the pieces both the page and the
  // panel with the whole list are made of.
  function achFor(it) {
    const key = recentKey(it);
    if (!achAsked.has(key)) {
      achAsked.add(key);
      const dir = it.f ? it.f.dir : it.exe;
      if (it.appId || dir) api.achievements(key, it.appId, dir, it.name).catch(() => {});
    }
    const top = S.ach[key];
    if (!top || !top.total) return null;
    // Each player's own (they go with their profile): whose to show is the
    // one picked here, else whoever's at the PC, else whose files are in.
    const ids = top.players ? cfg().players.map((p) => p.id).filter((id) => top.players[id]) : [];
    const seat = seated(), owner = it.g && S.d.active[it.g.id];
    const who = ids.includes(S.achWho[key]) ? S.achWho[key] : seat && ids.includes(seat.id) ? seat.id : ids.includes(owner) ? owner : ids[0];
    const A = who ? top.players[who] : top;
    const pct = A.total ? Math.round((A.unlocked / A.total) * 100) : 0;
    const whoRow = ids.length ? `<div class="ach-who" role="group" aria-label="Whose achievements">${ids.map((id) => { const p = player(id); return `<button class="ln-chip ln-player" style="--player:${esc(p.color)}" data-action="ach-who" data-key="${esc(key)}" data-player="${esc(id)}" aria-pressed="${id === who}">${lnAvatar(p, "20")}${esc(p.name)}<span class="ln-num">${top.players[id].unlocked}</span></button>`; }).join("")}</div>` : "";
    const g = it.g;
    const note = A.source !== "pc" ? "From Steam, for the account signed in on this PC."
      : ids.length ? "Each profile has its own. They go with it when you swap, like its saves."
      : g && g.sharedAchievements ? "Shared by everyone who plays on this PC." : "Saved on this PC next to the game's saves.";
    const toggle = g && A.source === "pc" ? `<button class="ln-button ln-button--ghost ln-button--sm" data-action="ach-shared" data-game="${esc(g.id)}" data-on="${g.sharedAchievements ? "0" : "1"}">${g.sharedAchievements ? "Give each profile its own" : "Share one set"}</button>` : "";
    const bar = `<div class="ach-bar" role="progressbar" aria-valuemin="0" aria-valuemax="${A.total}" aria-valuenow="${A.unlocked}" aria-label="${A.unlocked} of ${A.total} achievements"><span style="width:${pct}%"></span></div>`;
    const row = (a) => achRow(a, key);
    return { key, who, A, pct, whoRow, note, toggle, bar, row };
  }
  // Achievements by how rare they are, as gamers know them: ultra rare in
  // gold, rare in blue, uncommon in green, common in silver.
  const TIERS = { ultra: "Ultra rare", rare: "Rare", uncommon: "Uncommon", common: "Common" };
  const tierOf = (p) => (!p ? "" : p < 2 ? "ultra" : p < 10 ? "rare" : p < 30 ? "uncommon" : "common");
  const pctText = (p) => (p < 10 ? p.toFixed(1) : Math.round(p)) + "%";
  const achIcon = (a) => `<span class="ach-ico">${a.icon ? `<img src="${esc(a.icon)}" alt="" loading="lazy">` : GI.trophy}${a.unlocked ? "" : `<span class="ach-lock">${GI.lock || ""}</span>`}</span>`;
  function achRow(a, key, sub) {
    const t = a.unlocked || !a.hidden ? tierOf(a.percent) : "";
    const rk = key + "|" + a.id;
    const hid = a.hidden && !a.unlocked && !S.revealed.has(rk);
    const line = sub || (a.unlocked ? (a.at && new Date(a.at).getFullYear() > 2000 ? "Unlocked " + esc(ago(a.at)) : "Unlocked") : hid ? "Hidden until it's unlocked. Click to show it." : esc(a.description || "Not unlocked yet"));
    return `<li class="ach${a.unlocked ? " is-on" : " locked"}${t ? " tier-" + t : ""}${hid ? " is-hidden" : ""}"${hid ? ` data-action="ach-reveal" data-k="${esc(rk)}" role="button" tabindex="0" aria-label="Show this hidden achievement"` : ""}>${achIcon(a)}
      <div><b>${hid ? "Hidden achievement" : esc(a.name)}</b><span>${line}</span>${a.percent && !hid ? `<span class="ach-rar" aria-hidden="true"><i style="width:${Math.max(3, Math.min(100, a.percent))}%"></i></span>` : ""}</div>
      ${a.percent && !hid ? `<span class="ach-tier" title="${esc(pctText(a.percent))} of players have it"><em>${TIERS[tierOf(a.percent)]}</em><b class="ln-num">${pctText(a.percent)}</b></span>` : ""}</li>`;
  }
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
      // Read and none found: for a game from outside the stores, say why.
      if (few || S.ach[recentKey(it)] !== null || !docWorth(it)) return "";
      return `<section class="gpage-block gp-ach">${gpHead("Achievements")}<p>None found on this PC for ${esc(it.name)}.</p>
        <div class="gpage-actions"><button class="ln-button ln-button--secondary ln-button--sm" data-action="ach-doctor" ${itemAttr(it)} aria-haspopup="dialog">Check why</button></div></section>`;
    }
    const { key, who, A, pct } = X;
    const list = few ? A.list.filter((a) => a.unlocked).slice(0, 3) : A.list;
    const acts = few ? `<button class="ln-button ln-button--secondary ln-button--sm" data-action="game-tab" data-tab="achievements">Show all ${A.total}</button>`
      : (docWorth(it) ? `<button class="ln-button ln-button--ghost ln-button--sm" data-action="ach-doctor" ${itemAttr(it)} aria-haspopup="dialog">Check</button>` : "") + X.toggle;
    return `<section class="gpage-block gp-ach"><div class="ln-section-head"><div><h2>${who ? esc(possessive(pname(who))) + " achievements" : "Achievements"}</h2></div>
        <span class="ach-count"><b class="ln-num">${A.unlocked}</b> of <span class="ln-num">${A.total}</span>, ${pct}%</span></div>
      ${X.whoRow}${X.bar}
      ${list.length ? `<ul class="ach-list${few ? "" : " ach-grid"}">${list.map(X.row).join("")}</ul>` : `<p>None unlocked yet.</p>`}
      <div class="gp-ach-foot"><p class="gpage-hint">${X.note}</p>${acts ? `<div class="gpage-actions">${acts}</div>` : ""}</div></section>`;
  }
  // Save points: named copies of a game's saves, on a timeline.
  const pointsAsked = new Set();
  let ptEdit = null; // the save point being renamed
  function pointsFor(it) {
    const key = recentKey(it);
    if (!pointsAsked.has(key)) { pointsAsked.add(key); api.savePoints(key).catch(() => {}); }
    return S.points[key];
  }
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
        + (full && P.profiles ? b("ln-button--ghost", `data-action="pt-branch" ${attrs} data-player="${esc(p.player || "")}"`, "New run from it") : "")
        + (full ? b("ln-button--ghost", `data-action="pt-rename" ${attrs}`, "Rename") + b("ln-button--ghost", `data-action="pt-delete" ${attrs}`, "Delete") : "");
    const pic = p.thumb ? picButton(it, "point:" + p.id, p.thumb, `Picture of ${p.name}`) : "";
    return `<li class="pt-row${p.auto ? " pt-row--auto" : ""}${pic ? " pt-row--pic" : ""}"><span class="pt-dot" aria-hidden="true"></span><div class="pt-main">${name}
      <span class="pt-facts">${who ? `<span>${who}</span>` : ""}${facts.map((x) => `<span>${x}</span>`).join("")}${p.update ? "<span>Made before the game updated</span>" : p.auto ? "<span>Kept before going back</span>" : ""}</span></div>${pic}<div class="pt-actions">${actions}</div></li>`;
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
  function savePointsBlock(it) {
    const P = pointsFor(it);
    if (!P || !P.loaded || P.error) return "";
    const list = P.points || [];
    const make = `<div class="pt-make"><input class="ln-input" id="pt-new" maxlength="60" placeholder="Name it, such as Before the boss" aria-label="Save point name" ${P.busy ? "disabled" : ""}>
      <button class="ln-button ln-button--secondary ln-button--sm" data-action="pt-make" ${itemAttr(it)} ${P.busy ? "disabled" : ""}>${P.busy ? '<span class="ln-spinner" aria-hidden="true"></span>Copying' : "Make a save point"}</button></div>`;
    const hint = cfg().settings.pointHotkeyOff ? "" : " Ctrl+Shift+F5 makes one while you play.";
    return `<section class="gpage-block gp-points"><div class="ln-section-head"><div><h2>Save points</h2></div>${list.length ? `<span class="ach-count"><b class="ln-num">${list.length}</b> kept</span>` : ""}</div>
      <p class="gpage-hint">A copy of the saves as they are now, to come back to. Going back to one keeps the current saves as a point first.${hint}</p>${make}
      ${list.length ? `<ol class="pt-list pt-list--all">${list.map((p) => pointRow(it, p, true, P)).join("")}</ol>` : ""}</section>`;
  }
  // Overview: the newest save point, and the way to the rest.
  function latestPoint(it) {
    const P = S.points[recentKey(it)] || {}, list = P.points || [];
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
      : (G.locations || []).length ? "No copies yet. Lootnest copies its saves when they change and the game is closed." : `Lootnest doesn't know where ${it.name} keeps its saves yet.`;
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
    return `<section class="gpage-block"><div class="ln-list">
      ${gpRow(`data-action="copies" data-game="${esc(g.id)}" aria-haspopup="dialog"`, GI.history, "Copies before swaps", "Files from before each swap")}
      ${gpRow(`data-action="try-save" data-game="${esc(g.id)}" aria-haspopup="dialog"`, GI.flask, "Try a save", "A friend's save, in a new run", { disabled: s || cfg().players.length >= MAX_PROFILES })}</div></section>`;
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
    const tip = C && C.gameBarOff ? "Game Bar captures are off in Windows. Turn them on in Windows Settings, Gaming, Captures. Then Win+Alt+PrtScn takes a screenshot and Win+Alt+R records a clip, and they show here."
      : "Win+Alt+PrtScn takes a screenshot with Game Bar, and Win+Alt+R records a clip. They show here.";
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
    const opts = `<option value="">Choose a game</option><option value="none">Not a game</option>${everyItem().sort((a, b) => a.name.localeCompare(b.name)).map((it) => `<option value="${esc(recentKey(it))}">${esc(it.name)}</option>`).join("")}`;
    const body = !U ? lnInline("", "Reading the captures") : !U.items.length ? "<p>Every capture is matched to a game.</p>"
      : `<ul class="cap-list">${U.items.slice(0, UNSORTED_SHOWN).map((c) => `<li class="cap-row">${capThumb(null, c)}<div class="cap-main"><b>${esc(capWhat(c))}</b><span>${esc(when(c))}</span><span class="cap-file" title="${esc(c.path)}">${esc(c.path.split(/[\\/]/).pop())}</span></div>
          <select class="ln-input" data-assign="${esc(c.path)}" aria-label="${esc("The game of " + capLabel(c))}">${opts}</select></li>`).join("")}</ul>`;
    return drawer({ label: "Unsorted captures", title: "Unsorted captures", sub: U && U.total ? `${U.total} ${U.total === 1 ? "capture" : "captures"} Lootnest couldn't match to a game${U.total > UNSORTED_SHOWN ? `, the newest ${UNSORTED_SHOWN} here` : ", newest first"}` : "", body,
      foot: `<p class="gpage-hint ln-drawer__note">Choosing a game puts the capture on its Timeline. Lootnest only notes it down: the file stays where it is, as it is.</p>` });
  }
  function gameSavesBlock(it) {
    if (it.kind !== "found" || it.source === "steam" || it.appId) return "";
    const head = gpHead("Saves") + oneDriveNote(it, "gpage-hint");
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
        <dd>${btn("ln-button--ghost", `data-action="gs-restore" data-local="1" data-key="${esc(it.id)}" data-id="${esc(v.id)}" data-when="${esc(v.created)}" ${G.busy ? "disabled" : ""}`, "Restore")}</dd></div>`).join("")}</dl>
      <p class="gpage-hint">Lootnest copies ${esc(possessive(it.name))} saves when they change and the game is closed, and keeps the last five. Saves inside the game's own folder are in them too, so reinstalling or updating it can't take them.</p></div>` : "";
    if (G.off === "nocloud") return `<section class="gpage-block">${head}${local}<p>${lv.length ? "To have them on every PC too, keep" : "Keep"} ${esc(possessive(it.name))} saves in your own cloud, with every version to go back to.</p>
      <div class="gpage-actions"><button class="ln-button ln-button--secondary ln-button--sm" data-action="nav" data-view="cloud">Set up Cloud saves</button></div></section>`;
    if (G.off === "locked") return `<section class="gpage-block">${head}<p>The cloud folder has a password. Enter it on <b>Cloud saves</b>.</p>${local}</section>`;
    const s = cfg().settings;
    const choose = `<div class="gpage-actions">${btn("ln-button--secondary", `data-action="gs-add" data-key="${esc(it.id)}" data-kind="folder"`, G.locations.length ? "Add a folder" : "Choose the save folder")}
      ${btn("ln-button--ghost", `data-action="gs-add" data-key="${esc(it.id)}" data-kind="file"`, G.locations.length ? "Add a file" : "Choose a save file")}
      ${btn("ln-button--ghost", `data-action="gs-pcgw" data-name="${esc(it.name)}"`, "Look it up on PCGamingWiki")}</div>`;
    let where;
    if (!G.locations.length) {
      where = `<p>${G.source === "you" ? `Choose where ${esc(it.name)} keeps its saves.` : `Lootnest doesn't know where ${esc(it.name)} keeps its saves yet. Choose the folder, and its saves go to the cloud from then on.`}</p>
        <p class="gpage-hint">Usually in Documents, Saved Games, or AppData (Roaming, Local or LocalLow), in a folder named after the game or its studio. PCGamingWiki's page for the game says where.</p>
        ${choose}
        ${s.saveDb ? "" : `<p class="gpage-hint">Or turn on <b>Find save locations online</b>: Lootnest then uses PCGamingWiki's list of where over 20,000 games keep their saves. <button class="gpage-link" data-action="savedb-on">Turn it on</button></p>`}`;
    } else {
      const yours = G.source === "you";
      where = `${gpHead("On this PC")}<dl class="ln-facts gpage-paths">${G.locations.map((l) => `<div><dt class="selectable" title="${esc(l.resolved)}">${esc(l.path)}</dt><dd><span class="ln-num">${esc(fmtBytes(l.size || 0))}</span>
          ${btn("ln-button--ghost", `data-action="open-path" data-path="${esc(l.path)}"`, "Open")}${yours ? btn("ln-button--ghost", `data-action="gs-remove" data-key="${esc(it.id)}" data-path="${esc(l.path)}"`, "Remove") : ""}</dd></div>`).join("")}</dl>
        <p class="gpage-hint">${yours ? "Chosen by you." : G.source === "list" ? "Found with PCGamingWiki's list of save locations." : "Folders named like the game."}
          ${yours ? "" : `<button class="gpage-link" data-action="gs-own" data-key="${esc(it.id)}">Choose them yourself</button>`}</p>
        ${yours ? choose : ""}`;
    }
    const vs = G.versions || [];
    const inCloud = G.cloudDir ? `<div class="gpage-sub">${gpHead("In " + esc(G.provider || "your cloud folder"))}
      <dl class="ln-facts gpage-paths"><div><dt class="selectable">${esc(G.cloudDir)}</dt><dd>${btn("ln-button--ghost", `data-action="gs-open" data-key="${esc(it.id)}"`, "Open folder")}</dd></div></dl></div>` : "";
    const list = G.locations.length || vs.length ? `<div class="gpage-sub">${gpHead("Versions")}${vs.length ? `<dl class="ln-facts">${vs.map((v, i) => `<div><dt><span><b>${esc(new Date(v.created).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" }))}</b>
        <span class="gpage-hint">${esc(v.computer)}, ${esc(fmtBytes(v.size))}${i === 0 ? ", newest" : ""}</span></span></dt>
        <dd>${btn("ln-button--ghost", `data-action="gs-restore" data-key="${esc(it.id)}" data-id="${esc(v.id)}" data-when="${esc(v.created)}" data-pc="${esc(v.computer)}" ${G.busy ? "disabled" : ""}`, "Restore")}</dd></div>`).join("")}</dl>`
        : `<p class="gpage-hint">No versions yet.${G.off === "disabled" ? "" : " One is made when the saves change and the game is closed."}</p>`}
      ${G.off === "disabled" ? `<p class="gpage-hint">Automatic versions are off on <b>Cloud saves</b>.</p>` : ""}
      ${G.locations.length ? `<div class="gpage-actions">${G.busy ? `<button class="ln-button ln-button--secondary ln-button--sm" aria-busy="true"><span class="ln-spinner" aria-hidden="true"></span>Saving</button>` : btn("ln-button--secondary", `data-action="gs-now" data-key="${esc(it.id)}"`, "Back up now")}</div>` : ""}</div>` : "";
    return `<section class="gpage-block">${head}${where}${inCloud}${list}${local}</section>`;
  }


  // Game details from the Steam store (ps_gameDetails, opt-in): the copy
  // kept on this PC, with the user's own edits over it.
  const SHOTS_SHOWN = 4;
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
    const shots = d.screenshots || [], more = shots.length - SHOTS_SHOWN;
    const thumbs = shots.length ? `<div class="gp-shots">${shots.slice(0, SHOTS_SHOWN).map((s, i) => `<button class="gp-shot" data-action="shot-open" ${itemAttr(it)} data-i="${i}" aria-label="Screenshot ${i + 1} of ${shots.length}" aria-haspopup="dialog">
        <img src="${esc(s.thumb)}" alt="" loading="lazy">${i === SHOTS_SHOWN - 1 && more > 0 ? `<span class="gp-shot-more ln-num" aria-hidden="true">+${more}</span>` : ""}</button>`).join("")}</div>` : "";
    const also = d.from ? `, and ${esc(d.from)}` : "", edits = R.edited ? ", with your edits" : "";
    const from = d.storeUrl ? `From the Steam store${also}${edits}. <button class="gpage-link" data-action="open-link" data-url="${esc(d.storeUrl)}">Open on Steam</button>` : d.from ? `From ${esc(d.from)}${edits}.` : "Your own details.";
    return `<section class="gpage-block gp-det">${head}<div class="gp-det-cols"><div class="gp-det-main">
        ${sum ? `<p class="gp-det-sum${open ? " open" : ""}" id="det-sum">${esc(sum)}</p><button class="gpage-link" data-action="details-more" data-k="${esc(key)}" aria-expanded="${open}" aria-controls="det-sum"${open ? "" : " hidden"}>${open ? "Show less" : "Show more"}</button>` : ""}
        ${gameCtlLine(d)}${thumbs}<p class="gpage-hint">${from}${R.fetching ? " Checking for newer details." : ""}</p></div>
        ${facts.length ? `<dl class="ln-facts gp-det-facts">${facts.map(([k, v]) => `<div><dt>${k}</dt><dd>${v}</dd></div>`).join("")}</dl>` : ""}</div></section>`;
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
      <div><h2>Edit details</h2><p class="sub">Your changes stay on this PC and win over Steam's. Only what you change is kept.</p></div>
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
