  // --------------------------------------------------- big picture: screens --
  // What Big Picture shows and how the focus moves in it. A screen is a
  // list of zones (the bar, a row, a grid, a column of keys), a zone a list
  // of things to choose; Left and Right move in a row, Up and Down between
  // zones, and each zone remembers where the focus was. Nothing scrolls by
  // itself: rows and the screen slide (transform only) to keep the focus in
  // place. A redraw changes only what differs, so nothing blinks or loses
  // its place when art or a session arrives.
  const bpScreenKey = () => (BP.settings ? "settings" : BP.choose ? "choose:" + BP.choose : BP.detail && BP.points ? "points:" + BP.detail : BP.detail ? "detail:" + BP.detail : "tab:" + BP.tab + (BP.tab === "library" ? ":" + BP.tag : ""));
  const bpMem = (k) => BP.mem[k] || (BP.mem[k] = { focus: "", rows: {} });
  const bpSub = () => !!(BP.detail || BP.choose || BP.settings);
  function bpArt(it, wide) {
    loadArt(it); ensureCover(it);
    return artFor(it, wide ? "hero" : "cover");
  }
  const bpWaiting = (it) => !!it.appId && S.covers[coverKey(it)] === undefined;
  function bpTile(it, row) {
    const a = bpArt(it, false), s = it.g && session(it.g.id);
    return `<button class="bp-tile" data-bp data-bpid="t:${esc(row)}:${esc(bpKey(it))}" data-k="${esc(bpKey(it))}" data-go="open" data-hint="Open">
      <span class="bp-art${!a.url && bpWaiting(it) ? " is-waiting" : ""}">${a.url ? artImg(a, "cover") : typeCover(it, "", { icon: true })}</span>
      ${s ? `<span class="bp-live">${s.phase === "playing" ? "Playing" : esc(sessionText(s))}</span>` : ""}<span class="bp-name">${esc(it.name)}</span></button>`;
  }
  // A collection too small for a row of its own: one wide tile with its first three covers and its name.
  function bpTagTile(t, list) {
    const covers = list.slice(0, 3).map((it) => {
      const a = bpArt(it, false);
      return `<span class="bp-tag-cover">${a.url ? artImg(a, "cover") : plate(S.icons[coverKey(it)], "", "")}</span>`;
    }).join("");
    return `<button class="bp-tile bp-tile--tag" data-bp data-bpid="c:${esc(t)}" data-go="tag" data-tag="${esc(t)}" data-hint="Open">
      <span class="bp-art">${covers}<span class="bp-tag-name"><b>${esc(t)}</b>${list.length} ${list.length === 1 ? "game" : "games"}</span></span></button>`;
  }
  const bpRowHTML = (id, title, inner, lead = "") => `<section class="bp-row" data-block>${lead}<h2>${esc(title)}</h2><div class="bp-track" data-zone="${esc(id)}" data-live>${inner}</div></section>`;
  const bpRow = (id, title, list) => (list.length ? bpRowHTML(id, title, list.map((it) => bpTile(it, id)).join("")) : "");
  const bpGhosts = (n) => Array.from({ length: n }, () => '<span class="bp-tile bp-tile--ghost" aria-hidden="true"><span class="bp-art is-waiting"></span></span>').join("");
  const bpEmpty = (title, line, key) => `<div class="bp-empty" data-block><h1>${esc(title)}</h1>${line ? `<p>${esc(line)}</p>` : ""}${key ? `<div class="bp-keys" data-zone="empty">${key}</div>` : ""}</div>`;
  // The game in focus, said large: what's going on with it, its logo or
  // name, and its facts. On Home it follows the focus; a game's page has
  // the stores and the controller line too.
  function bpHero(it, page) {
    if (!it) return "";
    const key = recentKey(it), t = totalPlayed(it) || steamTime(it), seat = seated();
    const top = S.ach[key], A = top && top.players && seat && top.players[seat.id] ? top.players[seat.id] : top;
    const s = it.g && session(it.g.id), owner = it.g && player(S.d.active[it.g.id]);
    const kick = s ? sessionText(s) : owner ? `${possessive(owner.name)} files are loaded` : "";
    const pts = ((S.points[key] || {}).points || []).length;
    const det = page && cfg().settings.gameDetails && (S.details[key] || {}).details;
    const logo = artFor(it, "logo").url;
    const facts = [page ? esc(storeNames(it)) : "", playedAt(it) ? `Played <b>${esc(ago(new Date(playedAt(it))))}</b>` : "", t ? `<b>${esc(hours(t))}</b> played` : "",
      A && A.total ? `<b>${A.unlocked} of ${A.total}</b> achievements` : "", pts ? `<b>${pts}</b> save point${pts === 1 ? "" : "s"}` : "", det && CTL_SAYS[det.controller] ? esc(CTL_SAYS[det.controller]) : ""].filter(Boolean);
    return `${kick ? `<p class="bp-kick"><span class="ln-lamp ln-lamp--on"></span>${esc(kick)}</p>` : ""}
      ${logo ? `<img class="bp-logo" src="${logo}" alt="${esc(it.name)}">` : `<h1 class="bp-title">${esc(it.name)}</h1>`}
      <p class="bp-facts">${facts.map((x) => `<span>${x}</span>`).join("")}</p>`;
  }
  // TV mode's apps: Spotify, YouTube's TV interface and Stremio, when the
  // PC has them (app/tv.go apiMediaApps).
  const APP_ICONS = { spotify: "spotify-logo", youtube: "youtube-logo", stremio: "popcorn" };
  function bpAppsRow() {
    if (!BP.apps) { BP.apps = []; api.mediaApps().then((l) => { BP.apps = l || []; if (BP.apps.length && BP.on && BP.tab === "home") renderBP(false); }).catch(() => {}); }
    if (!BP.apps.length) return "";
    return bpRowHTML("apps", "Apps", BP.apps.map((a) =>
      `<button class="bp-tile bp-tile--app" data-bp data-bpid="app:${esc(a.id)}" data-go="app" data-app="${esc(a.id)}" data-name="${esc(a.name)}" data-hint="Open"><span class="bp-art bp-app--${esc(a.id)}">${ph(APP_ICONS[a.id] || "television-simple", "fill")}</span><span class="bp-name">${esc(a.name)}</span></button>`).join(""));
  }
  function bpApp(id, name) {
    const key = "app:" + id;
    sfx("go");
    if (BP.tv && BP.playing === key) { api.tvResume().catch(() => {}); return; }
    if (BP.tv) { BP.launch = { key, name, art: null, phase: "starting", at: Date.now() }; drawLaunch(); launchWatch(); }
    api.launchApp(id).catch((e) => { tvReturn(); toast(errText(e), "error"); });
  }
  const bpPerson = (p, id, go, line, on) => `<button class="bp-person ln-player${on ? " is-seated" : ""}" style="--player:${esc(p.color)}" data-bp data-bpid="${id}:${esc(p.id)}" data-go="${go}" data-p="${esc(p.id)}">${lnAvatar(p, "96")}<b>${esc(p.name)}</b><span>${esc(line)}</span></button>`;
  // The screen on show, as the inside of its sliding sheet.
  function bpScreen() {
    const all = libraryItems();
    if (BP.settings) return bpSettingsScreen();
    if (BP.choose) {
      const it = bpItem(BP.choose);
      return `<div class="bp-center" data-block><p class="bp-kick">${esc(it ? it.name : "")}</p><h1>Who's playing?</h1>
        <div class="bp-people" data-zone="people" data-grid>${cfg().players.map((p) => bpPerson(p, "who", "play-as", it && it.g && S.d.active[it.g.id] === p.id ? "Files loaded" : it && it.g && hasProfile(it.g.id, p.id) ? "Has a profile" : "New profile")).join("")}</div></div>`;
    }
    if (BP.detail) {
      const it = bpItem(BP.detail);
      if (!it) { BP.detail = null; return bpScreen(); }
      if (BP.points) return bpPointsScreen(it);
      const X = achFor(it); // reads its achievements if it hasn't yet
      pointsFor(it);
      const seat = seated(), as = it.g && seat, k = esc(bpKey(it));
      return `<section class="bp-page" data-block><div class="bp-hero in">${bpHero(it, true)}</div>
        <div class="bp-keys" data-zone="keys">
          <button class="bp-key ${as ? `bp-key--play ln-player" style="--player:${esc(seat.color)}"` : 'bp-key--primary"'} data-bp data-bpid="play" data-go="play" data-k="${k}" data-hint="Play">${GI.play}${as ? `Play as ${esc(seat.name)}` : "Play"}</button>
          ${it.g && cfg().players.length > 1 ? `<button class="bp-key" data-bp data-bpid="playas" data-go="choose" data-k="${k}">Play as someone else</button>` : ""}
          <button class="bp-key" data-bp data-bpid="points" data-go="points" data-k="${k}" data-hint="Open">Save points</button>
          ${bpRunning(it) ? `<button class="bp-key" data-bp data-bpid="close" data-go="close-game" data-k="${k}">Close the game</button>` : ""}</div></section>
        ${X ? bpRowHTML("ach", `Achievements, ${X.A.unlocked} of ${X.A.total}`, bpAchCards(X.A.list)) : ""}`;
    }
    if (!all.length && (BP.tab === "library" || BP.tab === "home")) {
      if (S.found.scanning) return `${bpRowHTML("wait", "Looking for your games", bpGhosts(9))}${bpRowHTML("wait2", "", bpGhosts(9))}`;
      return bpEmpty("No games yet", "Lootnest lists the games on this PC once it has looked for them.", `<button class="bp-key" data-bp data-bpid="empty:menu" data-go="menu">Open the menu</button>`);
    }
    if (BP.tab === "library") {
      const tags = allTags().filter((t) => all.some((it) => tagsOf(it).includes(t)));
      const list = BP.tag ? all.filter((it) => tagsOf(it).includes(BP.tag)) : all;
      const chip = (t, l) => `<button class="bp-chip" data-bp data-bpid="chip:${esc(t)}" data-go="chip" data-tag="${esc(t)}" aria-pressed="${BP.tag === t}" data-hint="Show">${esc(l)}</button>`;
      return `<section class="bp-lib" data-block><div class="bp-lib-head"><div class="bp-track bp-chips" data-zone="chips" data-live>${chip("", "All games")}${tags.map((t) => chip(t, t)).join("")}</div><span class="bp-count ln-num">${list.length} ${list.length === 1 ? "game" : "games"}</span></div>
        <div class="bp-grid" data-zone="grid" data-grid>${list.map((it) => bpTile(it, "grid")).join("")}</div></section>`;
    }
    if (BP.tab === "profiles") {
      const seat = seated(), alone = solo();
      return `<div class="bp-center" data-block><h1>${BP.who ? "Who's playing?" : alone ? "Playing as" : "Who's here?"}</h1>
        <div class="bp-people" data-zone="people" data-grid>${cfg().players.map((p) => bpPerson(p, "seat", "seat", seat && seat.id === p.id ? (alone ? "Playing" : "Sitting here") : alone ? "Play as this" : "Sit down", seat && seat.id === p.id)).join("")}</div></div>`;
    }
    if (BP.tab === "achievements") {
      const rows = (S.allAch ? S.allAch.games : []).map((g) => ({ g, it: itemByKey(g.key) })).filter((r) => r.it);
      if (!S.allAch) return `<section class="bp-lib" data-block><div class="bp-cards">${Array.from({ length: 6 }, () => '<span class="bp-card bp-card--ghost is-waiting" aria-hidden="true"></span>').join("")}</div></section>`;
      if (!rows.length) return bpEmpty("No achievements yet", "They show here once a game on this PC has some.", "");
      const who = (seated() || cfg().players[0] || {}).id;
      const mine = rows.map((r) => ({ ...r, u: r.g.players && r.g.players[who] ? r.g.players[who].unlocked : r.g.unlocked }));
      const sum = mine.reduce((n, r) => n + r.u, 0), of = mine.reduce((n, r) => n + r.g.total, 0);
      return `<section class="bp-lib" data-block><div class="bp-lib-head"><h1 class="bp-h1"><span class="ln-num">${sum}</span> of ${of} unlocked</h1><span class="bp-count ln-num">${rows.length} ${rows.length === 1 ? "game" : "games"}</span></div>
        <div class="bp-cards" data-zone="ach" data-grid>${mine.map(({ g, it, u }) => {
        const pct = Math.round((u / g.total) * 100), a = bpArt(it, false);
        return `<button class="bp-card" data-bp data-bpid="a:${esc(bpKey(it))}" data-k="${esc(bpKey(it))}" data-go="open" data-focus="ach:0" data-hint="Open"><span class="bp-card-art">${a.url ? artImg(a, "cover") : ""}</span><span class="bp-card-text"><b>${esc(g.name)}</b><span>${u} of ${g.total}</span><span class="bp-meter"><i style="width:${pct}%"></i></span></span><span class="ach-gring">${ring(pct, 64)}</span></button>`; }).join("")}</div></section>`;
    }
    const recent = recentItems(all, false).slice(0, 12);
    const tags = {};
    all.forEach((it) => tagsOf(it).forEach((t) => (tags[t] = tags[t] || []).push(it)));
    const names = allTags().filter((t) => tags[t]); // the sidebar's order
    const big = names.filter((t) => tags[t].length >= 5).slice(0, 3);
    const small = names.filter((t) => !big.includes(t));
    const first = recent.length ? recent : all.slice(0, 12);
    return `${bpRowHTML("recent", recent.length ? "Continue playing" : "Your games", first.map((it) => bpTile(it, "recent")).join("") + (S.found.scanning ? bpGhosts(5) : ""), '<div class="bp-hero" data-hero data-own></div>')}
      ${bpAppsRow()}
      ${big.map((t) => bpRow("tag:" + t, t, tags[t])).join("")}
      ${small.length ? bpRowHTML("tags", "Collections", small.map((t) => bpTagTile(t, tags[t])).join("")) : ""}
      ${recent.length ? bpRow("all", "Your games", all.slice(0, 40)) : ""}`;
  }
  // A game's achievements as a row to look through: the ones unlocked
  // first, the rarest of them leading, then what's left.
  function bpAchCards(list) {
    const rank = (a) => (a.unlocked ? 0 : 1000) + (a.percent || 100);
    return [...list].sort((a, b) => rank(a) - rank(b)).slice(0, 40).map((a, i) => {
      const hid = a.hidden && !a.unlocked, t = a.unlocked ? tierOf(a.percent) : "";
      const line = hid ? "Not unlocked yet" : a.unlocked ? (a.percent ? `${a.percent < 10 ? a.percent.toFixed(1) : Math.round(a.percent)}% of players` : "Unlocked") : "Not unlocked yet";
      return `<div class="bp-ach${a.unlocked ? "" : " is-locked"}${t ? " tier-" + t : ""}" tabindex="-1" data-bp data-bpid="ach:${i}" data-hint="">${achIcon(a)}<span><b>${hid ? "Hidden achievement" : esc(a.name)}</b><span>${esc(line)}</span></span></div>`;
    }).join("");
  }
  // A game's save points: make one now, or go back to one. Going back asks
  // first, and keeps the saves as they are now as a point of their own.
  function bpPointsScreen(it) {
    const P = pointsFor(it), k = esc(bpKey(it)), list = (P && P.points) || [];
    const head = `<p class="bp-kick">${esc(it.name)}</p><h1>Save points</h1>`;
    if (!P || !P.loaded) return `<section class="bp-list" data-block>${head}<div class="bp-opts">${'<span class="bp-opt bp-opt--ghost is-waiting" aria-hidden="true"></span>'.repeat(4)}</div></section>`;
    const make = `<button class="bp-opt" data-bp data-bpid="pt:make" data-go="pt-make" data-k="${k}"${P.busy ? " disabled" : ""} data-hint="Make">${ph("plus")}<b>${P.busy ? "Working on it" : "Make a save point now"}</b></button>`;
    const rows = list.map((p) => {
      const when = new Date(p.created).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
      const facts = [when, p.playtime ? hours(p.playtime) + " played" : "", p.player && player(p.player) ? pname(p.player) : ""].filter(Boolean).join(", ");
      return `<button class="bp-opt" data-bp data-bpid="pt:${esc(p.id)}" data-go="pt-restore" data-k="${k}" data-id="${esc(p.id)}" data-name="${esc(p.name)}"${P.busy ? " disabled" : ""} data-hint="Go back to it"><b>${esc(p.name)}</b><span>${esc(facts)}</span></button>`;
    }).join("");
    return `<section class="bp-list" data-block>${head}${P.error ? `<p class="bp-note">${esc(P.error)}</p>` : !list.length ? '<p class="bp-note">None yet. A save point is a copy of this game\'s saves as they are now, to come back to.</p>' : ""}
      <div class="bp-opts" data-zone="points" data-col>${make}${rows}</div></section>`;
  }
  // Settings, made for a remote: the parts on the left, big rows on the
  // right that OK changes, and a line under them saying what the row in
  // focus does. Only settings Lootnest already has.
  const BP_SIGNS = [["", "Automatic"], ["xbox", "Xbox"], ["playstation", "PlayStation"], ["nintendo", "Nintendo"]];
  const BP_MOTION = [["", "Full"], ["less", "Less"], ["off", "Off"]];
  function bpSettingsParts() {
    const s = cfg().settings, next = (list, v) => list[(list.findIndex((x) => x[0] === v) + 1) % list.length][0], label = (list, v) => (list.find((x) => x[0] === v) || list[0])[1];
    const flip = (key) => () => bpSet(key, !s[key]);
    const parts = [
      { id: "sound", name: "Sound", opts: [
        { id: "move", name: "Sounds as you move", on: BP.sounds, say: "A click as the focus moves, and one for Select and Back.", run: () => { BP.sounds = !BP.sounds; setPref("bpSounds", BP.sounds ? "1" : "0"); renderBP(); } },
        { id: "cues", name: "Sounds for unlocks, save points and swaps", on: !s.soundsOff, say: "Heard over a game too. Never for a game with an anti-cheat.", run: flip("soundsOff") }] },
      { id: "picture", name: "Picture and motion", opts: [
        { id: "motion", name: "Motion", value: label(BP_MOTION, s.motion || ""), say: "Full moves and fades everything. Less keeps only the fades. Off changes screens at once.", run: () => bpSet("motion", next(BP_MOTION, s.motion || "")) },
        { id: "trailers", name: "Trailers behind the game in focus", on: BP.trailers, say: s.gameDetails ? "A game's trailer plays behind it, without sound, once the focus has rested on it. A moving picture costs the stream to a TV more than a still one." : "Needs Game details, which is turned on in Lootnest's Settings on the PC, under Online.", run: () => { BP.trailers = !BP.trailers; setPref("bpTrailers", BP.trailers ? "1" : "0"); if (!BP.trailers) bpTrailerStop(); renderBP(); } },
        { id: "start", name: "Open in Big Picture when Lootnest starts", on: !!s.openBigPicture, say: "For a PC that is mostly used from the sofa.", run: flip("openBigPicture") }] },
      { id: "controller", name: "Controller", opts: [
        { id: "signs", name: "Button signs", value: label(BP_SIGNS, BP.signs), say: "Automatic follows the controller in your hands. A controller plugged into the TV reaches the PC as an Xbox one, so choose yours here if the signs are wrong.", run: () => { BP.signs = next(BP_SIGNS, BP.signs); setPref("bpSigns", BP.signs); renderBP(); } }] },
      { id: "play", name: "While you play", opts: [
        { id: "toasts", name: "Show achievements as they unlock", on: !s.achToastsOff, say: "A small note over the game, in the profile's colour. Never over a game with an anti-cheat.", run: flip("achToastsOff") }] },
      { id: "about", name: "About", opts: [
        { id: "version", name: "Lootnest", value: [S.d.version, S.d.stage].filter(Boolean).join(" "), say: "Everything else is in Settings in Lootnest's window on the PC.", run: null }] },
    ];
    if (BP.tv) parts[4].opts.push({ id: "leave", name: "Leave TV mode", value: "", say: "Ends the stream to the TV. Lootnest stays open on the PC.", run: () => { api.tvExit().catch(() => {}); tvClose(); } });
    return parts;
  }
  function bpSet(key, value) {
    const s = JSON.parse(JSON.stringify(cfg().settings));
    s[key] = value;
    api.saveSettings({ settings: s, autostart: S.d.autostart }).then(refresh, (e) => { toast(errText(e), "error"); refresh(); });
  }
  function bpSettingsScreen() {
    const parts = bpSettingsParts(), cur = parts.find((x) => x.id === BP.setCat) || parts[0];
    return `<section class="bp-set" data-block><div><h1>Settings</h1>
      <div class="bp-cats" data-zone="cats" data-col data-right="opts">${parts.map((x) => `<button class="bp-cat" data-bp data-bpid="cat:${x.id}" data-cat="${x.id}" data-go="cat" aria-pressed="${x === cur}" data-hint="Open">${esc(x.name)}</button>`).join("")}</div></div>
      <div class="bp-set-main"><div class="bp-opts" data-zone="opts" data-col data-left="cats">${cur.opts.map((o) =>
        `<button class="bp-opt" data-bp data-bpid="opt:${cur.id}:${o.id}" data-go="set" data-opt="${cur.id}:${o.id}" data-say="${esc(o.say)}" data-hint="${o.run ? (o.on === undefined && o.value ? "Change" : o.on === undefined ? "Select" : o.on ? "Turn off" : "Turn on") : ""}"><b>${esc(o.name)}</b>${o.on === undefined ? `<span class="bp-val">${esc(o.value || "")}</span>` : `<span class="bp-val">${o.on ? "On" : "Off"}</span><span class="bp-sw${o.on ? " on" : ""}"></span>`}</button>`).join("")}</div>
        <p class="bp-say" data-own></p></div></section>`;
  }
  // The menu over the screen: what Start opens, and what Back on Home asks.
  // A question (BP.ask) uses the same sheet, with the safe answer in focus.
  function bpMenuHTML() {
    if (BP.ask) {
      return `<div class="bp-dlg" role="alertdialog" aria-modal="true" aria-label="${esc(BP.ask.title)}"><h1>${esc(BP.ask.title)}</h1><p>${esc(BP.ask.body)}</p>
        <div class="bp-dlg-keys" data-zone="ask" data-col>
          <button class="bp-key" data-bp data-bpid="ask:no" data-go="ask-no">Cancel</button>
          <button class="bp-key" data-bp data-bpid="ask:yes" data-go="ask-yes">${esc(BP.ask.ok)}</button></div></div>`;
    }
    const seat = seated(), leave = BP.menuWhy === "leave";
    return `<div class="bp-dlg" role="dialog" aria-modal="true" aria-label="Menu"><h1>${leave ? (BP.tv ? "Leave TV mode?" : "Leave Big Picture?") : "Menu"}</h1>
      ${BP.tv && leave ? "<p>Leaving ends the stream to the TV. Lootnest stays open on the PC.</p>" : ""}
      <div class="bp-dlg-keys" data-zone="menu" data-col>
        <button class="bp-key" data-bp data-bpid="menu:back" data-go="menu-close">${leave ? "Stay" : "Back to games"}</button>
        ${cfg().players.length > 1 && !leave ? `<button class="bp-key" data-bp data-bpid="menu:who" data-go="menu-who">${seat ? lnAvatar(seat, "32") : ""}${solo() ? "Switch profile" : "Who's playing"}</button>` : ""}
        ${leave ? "" : `<button class="bp-key" data-bp data-bpid="menu:settings" data-go="menu-settings">${ph("gear-six")}Settings</button>`}
        <button class="bp-key" data-bp data-bpid="menu:exit" data-go="${BP.tv ? "tv-exit" : "shut"}">${BP.tv ? "Leave TV mode" : "Leave Big Picture"}</button></div></div>`;
  }
  // How a new screen arrives: a tab from its side, a game's page from
  // further in, the way back from further out.
  function bpEnter(from, to) {
    const tab = (k) => (k.startsWith("tab:") ? BP_TABS.findIndex((t) => t[0] === k.split(":")[1]) : -1);
    const depth = (k) => (k.startsWith("tab:") ? 0 : k.startsWith("detail:") ? 1 : 2);
    if (!from) return "fade";
    const a = tab(from), b = tab(to);
    if (a >= 0 && b >= 0) return a === b ? "fade" : b > a ? "right" : "left";
    return depth(to) < depth(from) ? "out" : "in";
  }
  // Makes a what b is, touching only what differs. An element marked
  // data-live keeps the style a slide gave it; one marked data-own keeps its
  // children (the hero on Home, filled as the focus moves).
  const bpPlainKids = (el) => [...el.childNodes].every((n) => n.nodeType === 1 || !n.nodeValue.trim());
  function bpMorph(a, b) {
    if (a.isEqualNode(b)) return;
    const own = a.dataset.own !== undefined;
    if (a.tagName !== b.tagName || a.dataset.bpid !== b.dataset.bpid || (!own && (!a.children.length || a.children.length !== b.children.length || !bpPlainKids(a) || !bpPlainKids(b)))) { a.replaceWith(b); return; }
    const live = a.dataset.live !== undefined || own;
    for (const at of [...a.attributes]) if (!b.hasAttribute(at.name) && !(live && (at.name === "style" || at.name === "class" || at.name === "data-k"))) a.removeAttribute(at.name);
    for (const at of [...b.attributes]) if (a.getAttribute(at.name) !== at.value && !(own && at.name === "class")) a.setAttribute(at.name, at.value);
    if (own) return;
    const as = [...a.children], bs = [...b.children]; // a replaced child leaves b, so the lists are taken first
    for (let i = 0; i < as.length; i++) bpMorph(as[i], bs[i]);
  }
  const bpOver = () => !!(BP.menu || BP.ask);
  const bpScope = () => (bpOver() ? $("#bp .bp-over") : $("#bp"));
  // Asks before something that changes files; Back says no.
  function bpAsk(q) {
    BP.ask = q; BP.askFrom = BP.focus; BP.focus = "ask:no";
    sfx("open");
    renderBP();
  }
  function bpAnswer(yes) {
    const q = BP.ask;
    if (!q) return;
    BP.ask = null; BP.focus = BP.askFrom || "";
    sfx(yes ? "go" : "back");
    renderBP();
    if (yes) q.run();
  }
  const bpFind = (id) => { const sc = bpScope(); return id && sc ? [...sc.querySelectorAll("[data-bp]")].find((e) => e.dataset.bpid === id && !e.closest(".leaving")) : null; };
  let bpShown = "", bpFam = null, bpHintHTML = "";
  function renderBP() {
    let el = $("#bp");
    if (!el) { el = document.createElement("div"); el.id = "bp"; el.className = "bp ln-root"; el.setAttribute("role", "application"); el.setAttribute("aria-label", "Big Picture"); document.body.appendChild(el); }
    let main = el.querySelector(".bp-main");
    if (!main) {
      el.innerHTML = `<div class="bp-bd" aria-hidden="true"><img class="bp-bd-l" alt=""><img class="bp-bd-l" alt=""><div class="bp-bd-scrim"></div><div class="bp-bd-deep"></div></div>
      <header class="bp-bar" data-zone="bar"><span class="bp-mark">${BRAND_MARK}</span>
        <nav class="bp-tabs"><span class="bp-ind" aria-hidden="true"></span><span class="bp-lb"></span>${BP_TABS.map(([k, l]) => `<button class="bp-tab" data-bp data-bpid="tab:${k}" data-go="tab" data-t="${k}">${l}</button>`).join("")}<span class="bp-rb"></span></nav>
        <span class="bp-bar-end"><span class="bp-seat"></span><time class="bp-clock ln-num"></time>
        <button class="bp-round" data-bp data-bpid="tab:menu" data-go="menu" aria-label="Menu" data-hint="Open">${ph("list")}</button></span></header>
      <main class="bp-main"></main>
      <div class="bp-fade" aria-hidden="true"></div><footer class="bp-foot"></footer><div class="bp-over" hidden></div>`;
      main = el.querySelector(".bp-main");
      bpShown = ""; bpFam = null; bpHintHTML = ""; bpBgUrl = null;
    }
    // The bar: who's at the PC, the time, the tab on show.
    const seat = seated(), sub = bpSub();
    const seatHTML = seat ? `<span class="ln-player" style="--player:${esc(seat.color)}">${lnAvatar(seat, "32")}</span>${esc(seat.name)}` : "";
    const seatEl = el.querySelector(".bp-seat");
    if (seatEl.innerHTML !== seatHTML) seatEl.innerHTML = seatHTML;
    const fam = pad.fam + "|" + BP.signs;
    if (bpFam !== fam) { bpFam = fam; el.querySelector(".bp-lb").innerHTML = pad.fam ? glyph("LB") : ""; el.querySelector(".bp-rb").innerHTML = pad.fam ? glyph("RB") : ""; bpHintHTML = ""; }
    el.querySelector(".bp-clock").textContent = new Date().toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" });
    el.querySelector(".bp-bar").classList.toggle("is-sub", sub);
    el.querySelectorAll(".bp-tab").forEach((t) => { t.setAttribute("aria-current", String(BP.tab === t.dataset.t)); t.tabIndex = sub ? -1 : 0; });
    el.querySelector(".bp-round").tabIndex = sub ? -1 : 0;
    slideInd(el.querySelector(".bp-ind"), el.querySelector('.bp-tab[aria-current="true"]'));
    el.classList.toggle("is-dim", !sub && BP.tab !== "home");
    // The screen: a new one arrives over the one that leaves; the same one
    // is brought up to date in place.
    const key = bpScreenKey(), html = bpScreen();
    let scr = main.querySelector(".bp-screen:not(.leaving)");
    const changed = !scr || scr.dataset.key !== key;
    if (changed || scr.bpHTML !== html) {
      const neu = document.createElement("div");
      neu.className = "bp-screen"; neu.dataset.key = key; neu.innerHTML = `<div class="bp-scroll" data-live>${html}</div>`; neu.bpHTML = html;
      if (changed) {
        if (scr) { const old = scr; old.classList.add("leaving"); old.inert = true; setTimeout(() => old.remove(), 240); }
        neu.dataset.enter = bpEnter(bpShown, key);
        main.appendChild(neu); scr = neu; bpShown = key;
        scr.querySelectorAll(".bp-track").forEach((t) => bpShift(t, -(BP.sx[key + "|" + t.dataset.zone] || 0), 0, true));
      } else {
        bpMorph(scr.firstElementChild, neu.firstElementChild); scr.bpHTML = html;
        // What was replaced whole goes back where it had slid to.
        scr.querySelectorAll(".bp-track").forEach((t) => { if (!t.style.transform) bpShift(t, -(BP.sx[key + "|" + t.dataset.zone] || 0), 0, true); });
        if (!scr.firstElementChild.style.transform) bpShift(scr.firstElementChild, 0, -(scr.bpY || 0), true);
      }
    }
    const over = el.querySelector(".bp-over"), mh = bpOver() ? bpMenuHTML() : "";
    if (over.bpHTML !== mh) { over.bpHTML = mh; over.innerHTML = mh; over.hidden = !mh; }
    scr.inert = !!mh; el.querySelector(".bp-bar").inert = !!mh;
    // The focus: where it was, else where this screen last had it, else its first thing.
    // A place asked for that isn't drawn yet (a game's achievements, still
    // being read) takes the focus when it comes, unless the focus has moved.
    const want = BP.want && !bpOver() && bpFind(BP.want);
    if (want) { BP.focus = BP.want; BP.want = ""; }
    let f = bpFind(BP.focus);
    if (!f && !bpOver()) f = bpFind(bpMem(key).focus) || scr.querySelector("[data-bp]") || el.querySelector('.bp-tab[aria-current="true"]');
    if (!f) f = bpScope().querySelector("[data-bp]");
    if (f && !S.dialog && !S.picker) { if (document.activeElement !== f) f.focus({ preventScroll: true }); bpFocused(f, changed); }
  }
  function bpSoon() { if (BP.timer) return; BP.timer = requestAnimationFrame(() => { BP.timer = 0; if (BP.on) renderBP(); }); }
  // What follows the focus: the place it's kept, the slide that brings it
  // into view, the art behind, the hero on Home, the prompts.
  function bpFocused(f, jump) {
    const id = f.dataset.bpid || "";
    BP.focus = id;
    const scr = f.closest(".bp-screen"), zone = f.closest("[data-zone]");
    if (scr && zone) { const m = bpMem(scr.dataset.key); m.focus = id; m.rows[zone.dataset.zone] = id; }
    bpPlace(f, jump);
    bpAmbient(f, jump);
    bpHints(f);
    const say = scr && scr.querySelector(".bp-say");
    if (say && f.dataset.say !== undefined && say.textContent !== f.dataset.say) say.textContent = f.dataset.say;
  }
  const bpTopIn = (el, root) => { let y = 0; for (let n = el; n && n !== root; n = n.offsetParent) y += n.offsetTop; return y; };
  function bpShift(el, x, y, jump) {
    const to = `translate3d(${Math.round(x)}px, ${Math.round(y)}px, 0)`;
    if (el.style.transform === to) return;
    if (jump) el.style.transition = "none";
    el.style.transform = to;
    if (jump) { void el.offsetWidth; el.style.transition = ""; }
  }
  // A row slides so the focus sits at its start (the last screenful stays
  // put); the screen slides so the row in focus is under the bar, a grid's
  // line in the middle. Rows above the one in focus fade out.
  function bpPlace(f, jump) {
    const scr = f.closest(".bp-screen") || $("#bp .bp-screen:not(.leaving)");
    if (!scr) return;
    const sc = scr.firstElementChild, inScr = scr.contains(f);
    const track = inScr && f.closest(".bp-track");
    if (track) {
      const padX = parseFloat(getComputedStyle(track).paddingLeft) || 0, last = track.lastElementChild;
      const x = Math.max(0, Math.min(f.offsetLeft - padX, last.offsetLeft + last.offsetWidth + padX - track.clientWidth));
      bpShift(track, -x, 0, jump); BP.sx[scr.dataset.key + "|" + track.dataset.zone] = x;
    }
    const blocks = [...sc.querySelectorAll("[data-block]")], grid = inScr && f.closest("[data-grid], [data-col]");
    const bi = inScr ? blocks.indexOf(f.closest("[data-block]")) : 0;
    let y = 0;
    const padTop = parseFloat(getComputedStyle(sc).paddingTop) || 0;
    if (grid) y = bpTopIn(f, sc) - (scr.clientHeight - f.offsetHeight) / 2;
    else if (bi > 0) y = bpTopIn(blocks[bi], sc) - padTop;
    y = Math.max(0, Math.min(y, sc.scrollHeight - scr.clientHeight));
    bpShift(sc, 0, -y, jump); scr.bpY = y;
    // A row that has gone up under the bar fades out.
    blocks.forEach((b, i) => b.classList.toggle("is-above", !grid && i < bi && bpTopIn(b, sc) + b.offsetHeight - y <= padTop));
    $("#bp").classList.toggle("is-deep", y > 0);
  }
  // The art behind follows the game in focus, once the focus rests: its
  // wide art fades in over the last, and Home's hero says the game.
  let bpAmbT = 0, bpBgUrl = null, bpBgFront = 0;
  function bpAmbient(f, now) {
    clearTimeout(bpAmbT);
    const run = () => {
      const el = $("#bp");
      if (!el) return;
      const hero = el.querySelector(".bp-screen:not(.leaving) [data-hero]");
      const k = (f && f.isConnected && f.dataset.k) || BP.detail || BP.choose || (hero && (hero.dataset.k || (hero.parentElement.querySelector("[data-k]") || { dataset: {} }).dataset.k)) || BP.artKey || "";
      const it = k && bpItem(k);
      if (!it) return;
      BP.artKey = k;
      const wide = bpArt(it, true), url = wide.url || bpArt(it, false).url || "";
      if (url !== bpBgUrl) {
        bpBgUrl = url;
        const ls = el.querySelectorAll(".bp-bd-l"), next = ls[bpBgFront ^ 1], was = ls[bpBgFront];
        const show = () => { if (bpBgUrl !== url) return; next.classList.toggle("on", !!url); was.classList.remove("on"); bpBgFront ^= 1; };
        if (!url) show();
        else { next.src = url; (next.decode ? next.decode() : Promise.resolve()).then(show, show); }
      }
      bpTrailer(el.classList.contains("is-deep") || el.classList.contains("is-dim") || bpOver() || BP.settings || BP.points ? null : it);
      if (hero) {
        const html = bpHero(it, false);
        if (hero.bpHTML !== html) {
          const other = hero.dataset.k !== k;
          hero.bpHTML = html; hero.innerHTML = html; hero.dataset.k = k;
          if (other) { hero.classList.remove("in"); void hero.offsetWidth; }
          hero.classList.add("in");
        }
      }
    };
    if (now) run(); else bpAmbT = setTimeout(run, 180);
  }
  // A trailer behind the game in focus (Settings, Picture and motion; off
  // unless turned on, and only with Game details on, where trailers come
  // from). It starts two seconds after the focus rests on a game, without
  // sound, fades in over the art once it plays, and goes when the focus
  // moves on, a game starts, or motion is turned down.
  let bpVid = null; // { key, timer, el, hls }
  function bpTrailerStop() {
    const v = bpVid;
    bpVid = null;
    if (!v) return;
    clearTimeout(v.timer);
    if (!v.el) return;
    v.el.classList.remove("on");
    setTimeout(() => { if (v.hls) v.hls.destroy(); v.el.remove(); }, 700);
  }
  function bpTrailer(it) {
    const key = it ? recentKey(it) : "";
    if (bpVid && bpVid.key === key) return; // already playing, waiting, or it has none
    bpTrailerStop();
    if (!key || !BP.trailers || !moving() || !cfg().settings.gameDetails || BP.launch || (S.d.sessions || []).some((x) => x.phase === "playing")) return;
    const v = (bpVid = { key });
    const start = (tries) => {
      v.timer = 0;
      if (bpVid !== v || !BP.on) return;
      if (!S.details[key]) { S.details[key] = { loading: true }; api.gameDetails(key).catch(() => {}); }
      const R = S.details[key] || {}, d = R.details, m = d && (d.movies || []).find((x) => x.hls), host = $("#bp .bp-bd");
      if (R.loading && tries < 15) { v.timer = setTimeout(() => start(tries + 1), 700); return; } // its details are on their way
      if (!m || !host) return;
      const el = document.createElement("video");
      el.className = "bp-bd-v"; el.muted = true; el.loop = true; el.playsInline = true; el.preload = "auto";
      v.el = el;
      host.insertBefore(el, host.querySelector(".bp-bd-scrim"));
      el.addEventListener("playing", () => { if (bpVid === v) el.classList.add("on"); });
      loadHls().then((Hls) => {
        if (bpVid !== v) return;
        if (Hls && Hls.isSupported()) {
          const h = new Hls({ enableWorker: false, capLevelToPlayerSize: true });
          h.on(Hls.Events.ERROR, (_, e) => { if (e.fatal && bpVid === v) bpTrailerStop(); });
          h.loadSource(m.hls); h.attachMedia(el); v.hls = h;
        } else if (el.canPlayType("application/vnd.apple.mpegurl")) el.src = m.hls;
        el.play().catch(() => {});
      }).catch(() => { if (bpVid === v) bpTrailerStop(); });
    };
    v.timer = setTimeout(() => start(0), 2000);
  }
  // The prompts at the bottom say what the buttons do where the focus is.
  // The menu has one sign, its key at the end of the bar; Start opens it too.
  function bpHints(f) {
    const it = f && f.dataset.k && bpItem(f.dataset.k);
    const list = f && f.dataset.hint === "" ? [] : [["A", (f && f.dataset.hint) || "Select"]];
    if (it && it.g && cfg().players.length > 1 && !BP.choose && !BP.points && !bpOver()) list.push(["Y", "Play as someone else"]);
    if (it && bpRunning(it) && !BP.points && !bpOver()) list.push(["X", "Close the game"]);
    list.push(["B", BP.ask ? "Cancel" : BP.menu ? "Close" : "Back"]);
    const html = list.map(([g, t]) => `<span>${glyph(g)}${esc(t)}</span>`).join("");
    if (html !== bpHintHTML) { bpHintHTML = html; const foot = $("#bp .bp-foot"); if (foot) foot.innerHTML = html; }
  }
  // Moving the focus. In a row, to the next thing; in a grid, along its
  // lines; out of a zone, to the one above or below, at the thing that had
  // the focus there last, else the nearest across.
  const bpCols = (items) => { const top = items[0].offsetTop; let n = 0; while (n < items.length && items[n].offsetTop === top) n++; return n || 1; };
  function bpInto(zone, cur, dir) {
    const items = [...zone.querySelectorAll("[data-bp]")];
    if (zone.dataset.zone === "bar") return zone.querySelector('.bp-tab[aria-current="true"]') || items[0];
    const grid = zone.dataset.grid !== undefined;
    const kept = !grid && bpMem(bpScreenKey()).rows[zone.dataset.zone], had = kept && items.find((e) => e.dataset.bpid === kept);
    if (had) return had;
    const on = !grid && zone.querySelector('[aria-pressed="true"]');
    if (on) return on;
    let pool = items;
    if (grid) { const cols = bpCols(items); pool = dir === "up" ? items.slice(Math.floor((items.length - 1) / cols) * cols) : items.slice(0, cols); }
    const r = cur.getBoundingClientRect(), cx = r.left + r.width / 2;
    let best = pool[0], bd = Infinity;
    for (const e of pool) { const q = e.getBoundingClientRect(), d = Math.abs(q.left + q.width / 2 - cx); if (d < bd) { bd = d; best = e; } }
    return best;
  }
  function bpMove(dir) {
    const scope = bpScope();
    if (!scope) return;
    const a = document.activeElement, cur = a && a.closest && a.closest("[data-bp]");
    if (!cur || !scope.contains(cur) || cur.closest(".leaving")) { renderBP(); return; }
    const zone = cur.closest("[data-zone]"), items = [...zone.querySelectorAll("[data-bp]")], i = items.indexOf(cur);
    const horiz = dir === "left" || dir === "right", step = dir === "right" || dir === "down" ? 1 : -1;
    const bar = zone.dataset.zone === "bar";
    let next = null, leave = false;
    if (zone.dataset.grid !== undefined) {
      const cols = bpCols(items), j = i + step * (horiz ? 1 : cols);
      if (horiz) next = j >= 0 && j < items.length && Math.floor(j / cols) === Math.floor(i / cols) ? items[j] : null;
      else if (j >= 0 && j < items.length) next = items[j];
      else if (dir === "down" && Math.floor(i / cols) < Math.floor((items.length - 1) / cols)) next = items[items.length - 1];
      else leave = true;
    } else if (zone.dataset.col !== undefined) {
      if (!horiz) next = items[i + step] || null;
      else { const z = zone.dataset[dir] && scope.querySelector(`[data-zone="${zone.dataset[dir]}"]`); if (z && z.querySelector("[data-bp]")) next = bpInto(z, cur, dir); } // the column beside it
    }
    else if (horiz) next = items[i + step] || null;
    else leave = true;
    if (leave) {
      const zones = [...scope.querySelectorAll("[data-zone]")].filter((z) => !z.closest(".leaving") && z.querySelector("[data-bp]") && !(bpSub() && z.dataset.zone === "bar"));
      const z = zones[zones.indexOf(zone) + step];
      if (z) next = bar ? bpFind(bpMem(bpScreenKey()).focus) || bpInto(z, cur, dir) : bpInto(z, cur, dir);
    }
    if (!next) {
      sfx("edge");
      if (moving()) { const d = 10 * step; cur.animate({ translate: ["0 0", horiz ? `${d}px 0` : `0 ${d}px`, "0 0"] }, { duration: 200, easing: "ease-out" }); }
      return;
    }
    BP.want = "";
    next.focus({ preventScroll: true });
    sfx("move");
  }
  function bpTab(step) {
    if (bpSub() || bpOver()) return;
    const i = BP_TABS.findIndex((t) => t[0] === BP.tab);
    bpTabTo(BP_TABS[(i + step + BP_TABS.length) % BP_TABS.length][0]);
  }
  // Another tab: the focus stays on the bar if it was there, else it goes
  // where that screen last had it.
  function bpTabTo(t) {
    if (t === BP.tab && !bpSub()) return;
    const a = document.activeElement, inBar = !!(a && a.closest && a.closest(".bp-bar"));
    BP.tab = t; BP.tag = ""; BP.tagFrom = ""; BP.detail = null; BP.points = false; BP.choose = null; BP.who = false; BP.settings = false;
    BP.focus = inBar ? "tab:" + t : bpMem(bpScreenKey()).focus;
    sfx("tab");
    renderBP();
  }
  function bpPlay(it, pid) {
    if (!it) return;
    sfx("go");
    // The game TV mode is playing: Play goes back to it.
    if (BP.tv && BP.playing && sameGame(it, BP.playing)) { api.tvResume().catch(() => {}); return; }
    if (it.kind === "found") { tvLaunch(it); actions["play-found"]({ dataset: { key: it.id } }); return; }
    const p = pid || (seated() || {}).id;
    if (!p) { BP.choose = bpKey(it); BP.focus = ""; renderBP(); return; }
    BP.choose = null; BP.focus = BP.detail ? "play" : BP.from;
    renderBP();
    tvLaunch(it);
    actions["play-as"]({ dataset: { game: it.id, player: p } });
  }
  const bpRunning = (it) => { const s = it && it.g && session(it.g.id); return !!s && s.phase === "playing"; };
  function bpCloseGame(k) {
    const it = bpItem(k);
    if (!bpRunning(it)) return;
    sfx("go");
    actions["close-game"]({ dataset: { game: it.g.id } });
  }
  function bpX() {
    const el = document.activeElement;
    bpCloseGame((el && el.dataset && el.dataset.k) || BP.detail);
  }
  function bpSelect(el) {
    if (!el) return;
    const go = el.dataset.go;
    if (el.animate && moving()) el.animate({ scale: ["1", "0.95", "1"] }, { duration: 180, easing: "ease-out" });
    if (go === "open") { BP.detail = el.dataset.k; BP.points = false; BP.from = el.dataset.bpid; BP.focus = "play"; BP.want = el.dataset.focus || ""; sfx("go"); renderBP(); }
    else if (go === "menu-settings") { BP.menu = false; BP.settings = true; BP.setFrom = BP.menuFrom; BP.focus = ""; sfx("go"); renderBP(); }
    else if (go === "cat") { el.focus({ preventScroll: true }); bpMove("right"); }
    else if (go === "set") {
      const [c, o] = el.dataset.opt.split(":"), part = bpSettingsParts().find((x) => x.id === c), opt = part && part.opts.find((x) => x.id === o);
      if (opt && opt.run) { sfx("toggle"); opt.run(); }
    }
    else if (go === "points") { BP.points = true; BP.focus = ""; sfx("go"); renderBP(); }
    else if (go === "pt-make") bpPoint(el.dataset.k, "");
    else if (go === "pt-restore") {
      const it = bpItem(el.dataset.k), id = el.dataset.id;
      if (it) bpAsk({ title: `Go back to ${el.dataset.name}?`, body: `The saves of ${it.name} become what they were then. The saves as they are now are kept as a save point first, so you can come back to them too.`, ok: "Go back to it", run: () => bpPoint(el.dataset.k, id) });
    }
    else if (go === "ask-no") bpAnswer(false);
    else if (go === "ask-yes") bpAnswer(true);
    else if (go === "play") bpPlay(bpItem(el.dataset.k));
    else if (go === "app") bpApp(el.dataset.app, el.dataset.name);
    else if (go === "tag") { BP.tab = "library"; BP.tag = el.dataset.tag; BP.tagFrom = el.dataset.bpid; BP.focus = ""; sfx("go"); renderBP(); }
    else if (go === "chip") { if (BP.tag !== el.dataset.tag) { BP.tag = el.dataset.tag; BP.tagFrom = ""; BP.focus = el.dataset.bpid; sfx("tab"); renderBP(); } }
    else if (go === "choose") { BP.choose = el.dataset.k; BP.focus = ""; sfx("go"); renderBP(); }
    else if (go === "close-game") bpCloseGame(el.dataset.k);
    else if (go === "play-as") bpPlay(bpItem(BP.choose), el.dataset.p);
    else if (go === "seat") {
      actions.seat({ dataset: { player: el.dataset.p } }); sfx("go");
      if (BP.who) { BP.who = false; BP.tab = "home"; BP.focus = ""; }
      renderBP();
    }
    else if (go === "menu") bpMenu(true);
    else if (go === "menu-close") bpMenu(false);
    else if (go === "menu-who") { BP.menu = false; BP.tab = "profiles"; BP.who = true; BP.detail = null; BP.settings = false; BP.choose = null; BP.focus = ""; sfx("go"); renderBP(); }
    else if (go === "tv-exit") { api.tvExit().catch(() => {}); tvClose(); }
    else if (go === "shut") { BP.menu = false; bpShut(); }
    else if (go === "tab") {
      if (bpSub() || BP.tab !== el.dataset.t) bpTabTo(el.dataset.t);
      else { el.focus({ preventScroll: true }); bpMove("down"); } // Select on the tab on show goes into its screen
    }
    else if (go === "sound") { BP.sounds = !BP.sounds; setPref("bpSounds", BP.sounds ? "1" : "0"); sfx("toggle"); toast(BP.sounds ? "Sounds on" : "Sounds off"); }
  }
  // Makes a save point of a game's saves (id empty), or goes back to one.
  async function bpPoint(k, id) {
    const it = bpItem(k);
    if (!it) return;
    const key = recentKey(it), P = S.points[key] || (S.points[key] = {});
    P.busy = true; renderBP();
    try { if (id) await api.restoreSavePoint(key, id); else await api.makeSavePoint(key, ""); }
    catch (e) { P.busy = false; toast(errText(e), "error"); renderBP(); }
  }
  // Back, one step at a time: what's on top closes, a game's page goes back
  // to its cover, a screen goes up to the bar, the bar goes Home, and Home
  // asks before leaving.
  function bpBack() {
    BP.want = "";
    if (BP.ask) { bpAnswer(false); return; }
    if (BP.menu) { bpMenu(false); return; }
    const a = document.activeElement, inBar = !!(a && a.closest && a.closest(".bp-bar"));
    if (BP.settings) {
      if (a && a.closest && a.closest('[data-zone="opts"]')) BP.focus = "cat:" + (BP.setCat || "sound");
      else { BP.settings = false; BP.focus = BP.setFrom || ""; }
    }
    else if (BP.choose) { BP.choose = null; BP.focus = BP.detail ? "play" : BP.from; }
    else if (BP.detail && BP.points) { BP.points = false; BP.focus = "points"; }
    else if (BP.detail) { BP.focus = BP.from; BP.detail = null; }
    else if (BP.tab === "library" && BP.tagFrom) { BP.focus = BP.tagFrom; BP.tab = "home"; BP.tag = ""; BP.tagFrom = ""; }
    else if (!inBar) BP.focus = "tab:" + BP.tab;
    else if (BP.tab !== "home" && !BP.who) { BP.tab = "home"; BP.tag = ""; BP.focus = "tab:home"; }
    else { bpMenu(true, "leave"); return; }
    sfx("back");
    renderBP();
  }
  function bpY() {
    const el = document.activeElement;
    const k = (el && el.dataset && el.dataset.k) || BP.detail;
    const it = k && bpItem(k);
    if (it && it.g && cfg().players.length > 1 && !BP.choose && !BP.points && !bpOver()) { if (!BP.detail) BP.from = el.dataset.bpid || ""; BP.choose = k; BP.focus = ""; sfx("go"); renderBP(); }
  }
  document.addEventListener("click", (ev) => {
    const el = BP.on && ev.target.closest("#bp [data-go]");
    if (!el) return;
    ev.preventDefault(); ev.stopPropagation();
    bpSelect(el);
  }, true);
  document.addEventListener("focusin", (ev) => {
    const f = BP.on && ev.target.closest && ev.target.closest("#bp [data-bp]");
    if (!f || f.closest(".leaving")) return;
    bpFocused(f, false);
    if (f.dataset.cat && f.dataset.cat !== (BP.setCat || "sound")) { BP.setCat = f.dataset.cat; renderBP(); } // Settings: the part in focus shows its rows
    // Resting on another tab shows its screen, as on a TV's home.
    if (f.classList.contains("bp-tab") && f.dataset.t !== BP.tab && !bpSub() && !bpOver()) bpTabTo(f.dataset.t);
  });
  // A mouse wheel moves the focus, since nothing here scrolls by itself.
  let bpWheelAt = 0;
  document.addEventListener("wheel", (ev) => {
    if (!BP.on || S.picker || S.dialog || !ev.target.closest || !ev.target.closest("#bp")) return;
    const now = performance.now(), dx = Math.abs(ev.deltaX) > Math.abs(ev.deltaY);
    if (now - bpWheelAt < 140) return;
    bpWheelAt = now;
    bpMove(dx ? (ev.deltaX > 0 ? "right" : "left") : ev.deltaY > 0 ? "down" : "up");
  }, { passive: true });
  window.addEventListener("resize", () => { const f = BP.on && document.activeElement && document.activeElement.closest && document.activeElement.closest("#bp [data-bp]"); if (f) bpPlace(f, true); });
  document.addEventListener("keydown", (ev) => {
    if (ev.key === "F11") { ev.preventDefault(); if (BP.on) bpClose(); else bpOpen(); return; }
    if (!BP.on || S.picker || S.dialog) return;
    const k = ev.key;
    const dirs = { ArrowLeft: "left", ArrowRight: "right", ArrowUp: "up", ArrowDown: "down" };
    if (dirs[k]) { ev.preventDefault(); bpMove(dirs[k]); }
    else if (k === "Enter" || k === " ") { ev.preventDefault(); bpSelect(document.activeElement.closest && document.activeElement.closest("#bp [data-go]")); }
    else if (k === "Escape" || k === "Backspace") { ev.preventDefault(); bpBack(); }
    else if (k === "q" || k === "PageUp") { ev.preventDefault(); bpTab(-1); }
    else if (k === "e" || k === "PageDown") { ev.preventDefault(); bpTab(1); }
    else if (k === "y") { ev.preventDefault(); bpY(); }
    else if (k === "x") { ev.preventDefault(); bpX(); }
    else if (k === "F5") ev.preventDefault();
    ev.stopImmediatePropagation();
  }, true);
  setInterval(() => { const c = BP.on && $("#bp .bp-clock"); if (c) c.textContent = new Date().toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" }); }, 15000);

