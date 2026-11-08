  // Correcting a profile's hours for a game: the difference is kept apart
  // from the sessions Lootnest recorded (ShelfEntry.offset), so Reset brings
  // the recorded time back.
  function hoursEditor(key, p, t) {
    const off = shelfAt(p.id, key).offset || 0, rec = Math.max(0, t - off);
    return `<div class="gp-hours"><label for="gp-hours">Hours played</label>
      <input id="gp-hours" class="ln-input ln-num" type="number" min="0" max="100000" step="0.5" value="${Math.round(t / 360) / 10}">
      <button class="ln-button ln-button--secondary ln-button--sm" data-action="hours-save">Save</button>
      <button class="ln-button ln-button--ghost ln-button--sm" data-action="hours-cancel">Cancel</button>
      <p class="gpage-hint">${rec ? `Lootnest recorded ${esc(hours(rec))}.` : "Lootnest hasn't recorded any time yet."}${off ? ` <button class="gpage-link" data-action="hours-reset">Reset</button>` : ""}</p></div>`;
  }
  // "Saved yesterday, 24 MB": when the game last wrote this player's
  // profile, and how much it holds.
  function saved(g, pid) {
    const i = g ? profileInfo(g.id, pid) : {};
    return i.saved ? `<span class="gp-person-saved">Saved ${esc(ago(i.saved))}, ${esc(fmtBytes(i.size))}</span>` : "";
  }
  // Everyone who plays this game: their time, when they played, and whose
  // files are loaded.
  function playersPanel(it) {
    const pt = playtimeOf(it), key = recentKey(it);
    const editing = (p) => !!S.hoursEdit && S.hoursEdit.key === key && S.hoursEdit.player === p.id;
    const g = it.g, owner = g && S.d.active[g.id];
    const ps = cfg().players;
    if (!ps.length) return "";
    const total = ps.reduce((a, p) => a + (pt[p.id] || 0), 0), st = g && steamTime(it);
    const split = total ? `<div class="gp-split" aria-hidden="true">${ps.filter((p) => pt[p.id]).map((p) => `<span class="ln-player" style="--player:${esc(p.color)};flex:${pt[p.id]}"></span>`).join("")}</div>` : "";
    return `<section class="gpage-block gp-players"><div class="ln-section-head"><div><h2>Profiles</h2></div></div>${split}<div class="gp-people">${ps.map((p) => {
      const last = lastBy(it, p.id);
      const sub = last ? `Played ${ago(last)}` : g && hasProfile(g.id, p.id) ? "Has saves here" : pt[p.id] ? "" : "Not played yet";
      return `<div class="gp-person ln-player" style="--player:${esc(p.color)}">
        <div class="gp-person-top">${lnAvatar(p, "32")}<div class="gp-person-name"><b>${esc(p.name)}${owner === p.id ? ' <span class="ln-tag">Files loaded</span>' : ""}</b>${sub ? `<span>${esc(sub)}</span>` : ""}${saved(g, p.id)}</div>
        <button class="gp-person-time ln-num${pt[p.id] ? "" : " is-zero"}" data-action="hours-edit" data-key="${esc(key)}" data-player="${esc(p.id)}" aria-label="${esc(possessive(p.name))} hours played: ${pt[p.id] ? esc(hours(pt[p.id])) : "none"}. Correct them" title="Correct the hours" aria-expanded="${editing(p)}">${pt[p.id] ? esc(hours(pt[p.id])) : "0 h"}</button></div>
        ${editing(p) ? hoursEditor(key, p, pt[p.id] || 0) : ""}</div>`;
    }).join("")}</div>${g ? `<dl class="ln-facts gp-facts">${st ? `<div><dt>Steam account</dt><dd class="ln-num">${esc(hours(st))}</dd></div>` : ""}<div><dt>Copies before swaps</dt><dd><button class="gpage-link" data-action="copies" data-game="${esc(g.id)}" aria-haspopup="dialog">See them</button></dd></div></dl>` : ""}</section>`;
  }
  function viewProfileEditor() {
    const D = S.dialog, p = D.p, tab = D.tab || "look";
    const tabs = [["look", "Look"], ["about", "About"], ["showcase", "Showcase"], ["page", "Page"]];
    const gameOptions = (sel, none) => `<option value="">${none}</option>${libraryItems().map((it) => `<option value="${esc(recentKey(it))}" ${sel === recentKey(it) ? "selected" : ""}>${esc(it.name)}</option>`).join("")}`;
    let body = "";
    if (tab === "look") {
      const hasPic = !!(S.d.avatars || {})[p.id], hasBanner = !!(S.d.banners || {})[p.id];
      const kind = !p.banner ? "auto" : p.banner === "image" ? "image" : p.banner.startsWith("style:") ? "plain" : "game";
      body = `
      <div class="field"><span class="label">Picture</span><div class="pe-row">
        <button class="btn" data-action="avatar-pick" ${D.busy ? "disabled" : ""}>${D.busy === "avatar" ? "Reading the picture" : hasPic ? "Change picture" : "Choose a picture"}</button>
        ${hasPic ? `<button class="btn quiet" data-action="avatar-remove">Remove picture</button>` : `<select data-pe="avatarGame" aria-label="Or a game's cover">${gameOptions(p.avatarGame || "", "Or use a game's cover")}</select>`}</div></div>
      <div class="field"><span class="label">Colour</span><div class="pe-row"><div class="swatches">${S.d.colors.map((c) => `<button class="swatch" style="background:${c}" data-action="pe-color" data-color="${c}" aria-pressed="${c.toLowerCase() === (p.color || "").toLowerCase()}" aria-label="${c}"></button>`).join("")}</div>
        <label class="custom-color"><input type="color" value="${esc((p.color || "#6c9bd2").toLowerCase())}" data-pe="color" aria-label="Any colour"><span>Any colour</span></label></div></div>
      <div class="field"><span class="label">Banner</span>
        <div class="segt">${[["auto", "Most played"], ["plain", "Colour"], ["image", "Your picture"], ["game", "Game art"]].map(([v, l]) => `<button data-action="pe-banner" data-v="${v}" aria-pressed="${kind === v}">${l}</button>`).join("")}</div>
        ${kind === "game" ? `<select data-pe="banner" aria-label="Game">${gameOptions(p.banner, "Choose a game")}</select>` : ""}
        ${kind === "image" ? `<div class="pe-row"><button class="btn" data-action="banner-pick" ${D.busy ? "disabled" : ""}>${D.busy === "banner" ? "Reading the picture" : "Choose another picture"}</button><button class="btn quiet" data-action="banner-remove">Remove picture</button></div>` : ""}
        ${kind === "image" || kind === "game" ? `<label class="range"><span>Position</span><input type="range" min="0" max="100" step="1" value="${p.bannerY || 0}" data-pe="bannerY"></label>` : ""}
        <span class="hint">A picture is cropped to a wide strip (3:1) and kept on this PC.</span></div>`;
    } else if (tab === "about") {
      body = `
      <div class="field"><label for="pe-name">Name</label><input id="pe-name" type="text" maxlength="24" value="${esc(p.name)}" data-pe="name"></div>
      <div class="field"><label for="pe-tag">Tagline</label><input id="pe-tag" type="text" maxlength="80" value="${esc(p.tagline || "")}" placeholder="Optional" data-pe="tagline"></div>
      <div class="field"><label for="pe-bio">About <span class="hint">up to 300 characters</span></label><textarea id="pe-bio" rows="4" maxlength="300" placeholder="Optional" data-pe="bio">${esc(p.bio || "")}</textarea></div>
      <div class="field"><label for="pe-now">Currently playing</label><select id="pe-now" data-pe="now">${gameOptions(p.now || "", "None")}</select>
        <input type="text" maxlength="80" value="${esc(p.nowNote || "")}" placeholder="A note (optional)" data-pe="nowNote" aria-label="A word on it" style="margin-top:8px"></div>`;
    } else if (tab === "showcase") {
      const fav = p.favorites || [];
      const m = nameMatcher(D.q), games = libraryItems().filter((it) => m(it.name));
      body = `
      <div class="field"><span class="label">Favourite games <span class="hint">${fav.length} of 6, in this order</span></span>
        ${fav.length ? `<ol class="fav-order">${fav.map((k, i) => { const it = itemByKey(k); return `<li><span>${esc(it ? it.name : "A game no longer here")}</span>
          <button class="btn quiet icon" data-action="pe-fav-move" data-i="${i}" data-d="-1" aria-label="Earlier" ${i === 0 ? "disabled" : ""}>↑</button>
          <button class="btn quiet icon" data-action="pe-fav-move" data-i="${i}" data-d="1" aria-label="Later" ${i === fav.length - 1 ? "disabled" : ""}>↓</button>
          <button class="btn quiet icon" data-action="pe-fav" data-k="${esc(k)}" aria-label="Remove">${icon.x}</button></li>`; }).join("")}</ol>` : ""}
        <input type="search" id="pe-q" placeholder="Find a game" value="${esc(D.q || "")}" autocomplete="off">
        <div class="pe-games">${games.map((it) => { const k = recentKey(it), on = fav.includes(k); return `<button class="pe-game${on ? " on" : ""}" data-action="pe-fav" data-k="${esc(k)}" aria-pressed="${on}" ${!on && fav.length >= 6 ? "disabled" : ""}><span class="check"></span>${esc(it.name)}</button>`; }).join("")}</div></div>`;
    } else {
      const on = sectionsOf(p);
      const order = [...on, ...SECTIONS.map((s) => s[0]).filter((s) => !on.includes(s))];
      body = `<p class="sub" style="margin:0">What your page shows, top to bottom.</p><ul class="sec-list">${order.map((k) => {
        const s = SECTIONS.find((x) => x[0] === k), i = on.indexOf(k), shown = i >= 0;
        return `<li class="${shown ? "" : "off"}"><button class="sec-toggle" data-action="pe-sec" data-k="${k}" aria-pressed="${shown}" ${shown && on.length === 1 ? "disabled" : ""}><span class="check"></span><span><b>${s[1]}</b><span>${s[2]}</span></span></button>
          ${shown ? `<button class="btn quiet icon" data-action="pe-sec-move" data-k="${k}" data-d="-1" aria-label="Higher" ${i === 0 ? "disabled" : ""}>↑</button><button class="btn quiet icon" data-action="pe-sec-move" data-k="${k}" data-d="1" aria-label="Lower" ${i === on.length - 1 ? "disabled" : ""}>↓</button>` : ""}</li>`;
      }).join("")}</ul>`;
    }
    return `<div class="scrim" data-action="scrim"><div class="dialog pe" role="dialog" aria-modal="true" aria-label="Edit ${esc(p.name)}'s profile" style="--c:${esc(p.color)}">
      <div class="pe-preview"><div class="pe-pbanner">${bannerHTML(p, playerStats(p).top || null)}</div>${avatarHTML(p, "lg")}
        <div class="pe-pname"><b id="pe-prev-name">${esc(p.name)}</b><span id="pe-prev-tag">${esc(p.tagline || "")}</span></div>
        <button class="btn quiet icon" data-action="dialog-cancel" aria-label="Close">${icon.x}</button></div>
      <div class="segt pe-tabs" role="tablist">${tabs.map(([k, l]) => `<button role="tab" aria-selected="${k === tab}" data-action="pe-tab" data-tab="${k}">${l}</button>`).join("")}</div>
      <div class="pe-body">${body}</div>
      <div class="dialog-foot"><span class="error-text grow" id="pe-error"></span><button class="btn quiet" data-action="dialog-cancel">Cancel</button><button class="btn primary" data-action="pe-save">Save profile</button></div>
    </div></div>`;
  }

  // Players: the household, a row per person (who, their numbers, recent
  // covers) with their runs under them.
  function viewPlayers() {
    const ps = S.players || (S.players = JSON.parse(JSON.stringify(cfg().players)));
    const dirty = JSON.stringify(ps) !== JSON.stringify(cfg().players);
    const card = (p) => {
      const st = playerStats(p);
      const last = st.recent[0];
      // Their games: the latest first, then the ones they've played or have
      // saves in, so a row is never empty while they have games. As many as
      // a wide window fits; the row clips the rest.
      const seen = new Set(), mine = [...st.recent.map((x) => x.it),
        ...libraryItems().filter((it) => playtimeOf(it)[p.id]).sort((a, b) => playtimeOf(b)[p.id] - playtimeOf(a)[p.id]),
        ...libraryItems().filter((it) => it.kind === "game" && hasProfile(it.id, p.id))]
        .filter((it) => !seen.has(recentKey(it)) && seen.add(recentKey(it))).slice(0, 24);
      const covers = mine.map((it) => { ensureCover(it); const c = artFor(it, "cover").url; return `<span class="pcard-cover" title="${esc(it.name)}"${c ? ` style="background-image:url('${c}')"` : ""}></span>`; }).join("");
      return `<button class="pcard ln-player" data-action="open-player" data-player="${esc(p.id)}" style="--player:${esc(p.color)}">
        <span class="pcard-banner">${bannerHTML(p)}</span>
        <span class="pcard-who">${lnAvatar(p, "72")}<span class="pcard-name"><b>${esc(p.name)}</b><span>${p.tagline ? esc(p.tagline) : last ? `${esc(last.it.name)}, ${esc(ago(last.r.at))}` : st.top ? `Most played: ${esc(st.top.name)}` : "No games played yet"}</span></span></span>
        <span class="pcard-nums"><span><b class="ln-num">${st.total ? esc(hours(st.total)) : "0 h"}</b>played</span><span><b class="ln-num">${st.played}</b>${st.played === 1 ? "game" : "games"}</span><span><b class="ln-num">${st.profiles}</b>with saves</span></span>
        ${covers ? `<span class="pcard-covers" aria-hidden="true">${covers}</span>` : ""}</button>`;
    };
    // Each person, with their runs listed under them.
    const people = cfg().players.filter((p) => !parentOf(p));
    const cards = people.map((p) => {
      const runs = cfg().players.filter((r) => parentOf(r) === p);
      return `<div class="pgroup">${card(p)}${runs.length ? `<ul class="pruns" aria-label="${esc(possessive(p.name))} runs">${runs.map((r) => { const st = playerStats(r); return `<li><button class="prun ln-player" data-action="open-player" data-player="${esc(r.id)}" style="--player:${esc(r.color)}">${lnAvatar(r, "32")}<span><b>${esc(runLabel(r))}</b><span>Run, ${st.total ? esc(hours(st.total)) : "0 h"} played</span></span></button></li>`; }).join("")}</ul>` : ""}</div>`;
    }).join("");
    const n = cfg().players.length;
    return `<div class="players-page ln-root">
      <header class="lib-head"><div class="lib-title"><h1>Profiles</h1><span class="lib-meta">${n} of ${MAX_PROFILES}</span></div>
        <div class="lib-actions"><button class="ln-button ln-button--ghost" data-action="restore-pick" data-mode="import" title="A profile someone exported from Lootnest on another PC: their saves, picture and play time">Import a profile</button>
          <button class="ln-button ln-button--ghost" data-action="edit-profiles" aria-haspopup="dialog">${GI.edit}Names and colours${dirty ? '<span class="ln-tag">Not saved</span>' : ""}</button>
          ${ps.length < MAX_PROFILES ? `<button class="ln-button ln-button--primary" data-action="new-profile" aria-haspopup="dialog">${GI.plus}New profile</button>` : ""}</div>
        <p>Each profile has its own settings, saves and achievements in every game. Make one for everyone who plays, or a few for yourself, one per run.</p></header>
      <div class="pcards">${cards}</div></div>`;
  }
  // Names and colours, in a panel: every profile at once, saved together.
  function viewProfilesDrawer() {
    const ps = S.players || (S.players = JSON.parse(JSON.stringify(cfg().players)));
    const dirty = JSON.stringify(ps) !== JSON.stringify(cfg().players);
    return drawer({ label: "Names and colours", title: "Names and colours", wide: true,
      sub: "A profile's colour marks its files, its Play button and its games everywhere in Lootnest. A run takes its person's colour.",
      body: `<div class="players">${grouped(ps).map((p) => playerRow(p, ps.indexOf(p), ps.length)).join("")}</div>
        ${ps.length < MAX_PROFILES ? `<button class="ln-button ln-button--secondary" data-action="add-player">${GI.plus}Add profile</button>` : `<p class="gpage-hint">Lootnest keeps up to ${MAX_PROFILES} profiles.</p>`}`,
      foot: `<span class="error-text players-grow" id="players-error"></span>
        ${dirty ? `<button class="ln-button ln-button--ghost" data-action="reset-players">Discard changes</button>` : ""}
        <button class="ln-button ln-button--primary" data-action="save-players" ${dirty ? "" : "disabled"}>Save profiles</button>` });
  }

  async function savePlayers(welcome) {
    const ps = S.players.map((p, i) => ({ ...p, name: (p.name || "").trim() || (welcome ? "Player " + (i + 1) : "") }));
    const removed = cfg().players.filter((o) => !ps.some((p) => p.id === o.id));
    const inDrawer = S.dialog && S.dialog.kind === "profiles";
    if (removed.length && !welcome) {
      const ok = await confirmDialog({ title: `Remove ${removed.map((p) => p.name).join(" and ")}?`, body: "Their profiles are moved to the “removed” folder inside Lootnest's data folder, so nothing is deleted for good.", ok: "Remove" });
      // Back to the panel either way, with the changes still in it.
      if (inDrawer) { S.dialog = { kind: "profiles" }; renderOverlay(); }
      if (!ok) return;
    }
    try {
      await api.savePlayers(ps);
      S.players = null;
      if (welcome || inDrawer) S.dialog = null;
      await refresh();
      if (!welcome) toast("Profiles saved");
    } catch (e) { const t = $("#players-error"); if (t) t.textContent = errText(e); }
  }

