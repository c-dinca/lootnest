// Lootnest design harness: stands in for the Go side (window.ps_* and the
// __ps events) so the real UI runs in a plain browser with fixture data.
// Scenario from the URL: ?games=13|0|400&players=1|2|6&state=normal|loading|offline|error|playing&cloud=0|1&first=1&hidden=1&art=bright|missing|partial&boot=<ms>&bp=1&pad=ds5|ds4|xbox|switch|generic (comma-separated for more than one)
// ART is filled in by build.mjs: { "<appId>-cover|hero|logo": dataURL }.
(function () {
  const q = new URLSearchParams(location.search);
  const N = q.get("games") || "13", P = +(q.get("players") || 2), STATE = q.get("state") || "normal";
  const CLOUD = q.get("cloud") !== "0", FIRST = q.get("first") === "1";
  // ?boot=<ms>: the splash shows, and the first state comes after <ms>.
  // Without it the splash is taken out, so screenshots never catch it.
  const BOOT = +(q.get("boot") || 0);
  if (!BOOT) document.getElementById("boot")?.remove();
  // ?hidden=1: two games hidden from the Library.
  const HIDDEN = q.get("hidden") === "1" ? { "steam:1245620": "Elden Ring", "dir:d:\\games\\old racer": "Old Racer" } : {};
  const now = Date.now(), h = 3600e3, d = 24 * h;
  const iso = (ms) => new Date(now - ms).toISOString();
  // Updates found in Downloads (ps_downloads).
  let DL = [
    { key: "found:dir:d:\\games\\hydra\\hollow tale", game: "Hollow Tale", path: "C:\\Users\\you\\Downloads\\Hollow Tale v1.3", name: "Hollow Tale v1.3", kind: "folder",
      size: 812e6, modTime: iso(2 * h), check: { have: "1.2", to: "1.3", state: "ok" }, newer: true },
    { key: "found:dir:d:\\games\\phantom blade zero", game: "Phantom Blade Zero", path: "C:\\Users\\you\\Downloads\\Phantom.Blade.Zero.Update.v1.0.3.part1.rar", name: "Phantom.Blade.Zero.Update.v1.0.3.part1.rar", kind: "rar",
      size: 2.1e9, modTime: iso(26 * h), check: { to: "1.0.3", state: "unknown" }, newer: false },
  ];
  const ART = window.__ART || {};
  // ?art=bright: a near-white hero for Baldur's Gate 3, our own gradient, to check text over light art.
  if (q.get("art") === "bright") ART["1086940-hero"] = "data:image/svg+xml," + encodeURIComponent('<svg xmlns="http://www.w3.org/2000/svg" width="1920" height="620"><defs><linearGradient id="g" x2="1" y2="1"><stop stop-color="oklch(0.97 0 0)"/><stop offset="1" stop-color="oklch(0.9 0.02 240)"/></linearGradient></defs><rect width="100%" height="100%" fill="url(#g)"/></svg>');
  // ?art=missing: no game has any art. ?art=partial: Hollow Tale has only
  // wide art, Onimusha only a logo, and Phantom Blade Zero a cover of the user's own.
  const ARTQ = q.get("art");
  const art = (id, k) => (ARTQ === "missing" ? "" : ART[id + "-" + k] || "");
  // Custom art (ps_setCustomArt): "<Library key>:<kind>" -> data: URL. The
  // picture is our own, drawn here.
  const MINE = "data:image/svg+xml;base64," + btoa('<svg xmlns="http://www.w3.org/2000/svg" width="600" height="900"><defs><linearGradient id="g" x2="0" y2="1"><stop stop-color="oklch(0.42 0.09 160)"/><stop offset="1" stop-color="oklch(0.18 0.04 250)"/></linearGradient></defs><rect width="100%" height="100%" fill="url(#g)"/><circle cx="300" cy="330" r="150" fill="oklch(0.85 0.08 90)" opacity=".85"/><path d="M0 700 200 520 330 640 450 540 600 680V900H0z" fill="oklch(0.12 0.02 250)"/></svg>');
  const CUSTOM = ARTQ === "partial" ? { "dir:d:\\games\\phantom blade zero:cover": MINE } : {};
  const customIndex = () => {
    const out = {};
    Object.keys(CUSTOM).forEach((k) => { const i = k.lastIndexOf(":"), key = k.slice(0, i), kind = k.slice(i + 1); (out[key] = out[key] || {})[kind] = `custom-${kind}.png`; });
    return out;
  };

  const PLAYERS = [
    { id: "p1", name: "Cristian", color: "#6C9BD2" },
    { id: "p2", name: "Mihai", color: "#D9738E" },
    { id: "p3", name: "Ana", color: "#5FB49C" },
    { id: "p4", name: "Dan", color: "#C9A35F" },
    { id: "p5", name: "Ioana", color: "#9C88D0" },
    { id: "p6", name: "Radu", color: "#D98A5F" },
  ].slice(0, P);
  // ?anon=1: "Player 1", "Player 2" and so on, for screenshots made public
  // (the website), with nobody's name in them.
  const ANON = q.get("anon") === "1";
  if (ANON) PLAYERS.forEach((p, i) => { p.name = "Player " + (i + 1); });
  // ?runs=1: two runs of the first profile ("Cristian hardcore", "Cristian 100%").
  if (q.get("runs") === "1") PLAYERS.push({ id: "r1", name: PLAYERS[0].name + " hardcore", color: "#6C9BD2", of: "p1" }, { id: "r2", name: PLAYERS[0].name + " 100%", color: "#6C9BD2", of: "p1" });
  const ids = PLAYERS.map((p) => p.id);
  const some = (n) => ids.slice(0, Math.min(n, ids.length));

  // The 13 games. Three have profiles (config games), ten are scan results.
  const steamGame = (id, name, appId, proc, players) => ({
    id, name, processName: proc, launch: { type: "steam", appId },
    paths: [{ id: "d1", path: `{STEAMUSER}\\${appId}\\local\\cfg` }], playerArgs: {}, _players: players,
  });
  let games = [
    steamGame("cs2", "Counter-Strike 2", 730, "cs2", some(2)),
    steamGame("ittakestwo", "It Takes Two", 1426210, "ItTakesTwo", some(2)),
    steamGame("bg3", "Baldur's Gate 3", 1086940, "bg3", some(6)),
  ];
  let found = [
    { key: "steam:1091500", name: "Cyberpunk 2077", source: "steam", appId: 1091500, ready: true },
    { key: "steam:413150", name: "Stardew Valley", source: "steam", appId: 413150, ready: true },
    { key: "steam:250900", name: "The Binding of Isaac: Rebirth", source: "steam", appId: 250900, ready: true },
    { key: "steam:1326470", name: "Sons Of The Forest", source: "steam", appId: 1326470, _art: 242760 }, // mismatched art
    { key: "dir:c:\\games\\hades ii", name: "Hades II", source: "epic", exe: "C:\\Games\\Hades II\\Hades2.exe", _art: 1145350 },
    { key: "dir:c:\\gog games\\hollow knight", name: "Hollow Knight", source: "gog", exe: "C:\\GOG Games\\Hollow Knight\\hollow_knight.exe", _art: 367520 },
    { key: "dir:c:\\xboxgames\\forza horizon 5", name: "Forza Horizon 5", source: "xbox", exe: "", uri: "shell:AppsFolder\\x_8wekyb3d8bbwe!Game", _art: 1551360 },
    { key: "dir:d:\\games\\gtav", name: "Grand Theft Auto V", source: "rockstar", exe: "D:\\Games\\GTAV\\GTA5.exe", _art: 271590 },
    { key: "dir:d:\\games\\phantom blade zero", name: "Phantom Blade Zero", source: "folder", exe: "D:\\Games\\Phantom Blade Zero\\PBZ.exe" }, // no art
    { key: "dir:d:\\games\\hydra\\hollow tale", name: "Hollow Tale", source: "hydra", steamId: 777, exe: "D:\\Games\\Hydra\\Hollow Tale\\HollowTale.exe", ...(ARTQ === "partial" ? { _art: 413150, _only: "hero" } : {}) }, // no art (?art=partial: wide art only)
    { key: "dir:d:\\games\\onimusha", name: "Onimusha: Way of the Sword", source: "added", added: true, exe: "D:\\Games\\Onimusha\\Onimusha.exe", ...(ARTQ === "partial" ? { _art: 1086940, _only: "logo" } : {}) }, // no art (?art=partial: a logo only)
    // The same game from two stores (core.GroupCopies): two Steam and Epic pairs, and GOG with a folder copy.
    { key: "steam:1145350", name: "Hades II", source: "steam", appId: 1145350, ready: true },
    { key: "dir:c:\\epic games\\cyberpunk 2077", name: "Cyberpunk 2077", source: "epic", exe: "C:\\Epic Games\\Cyberpunk 2077\\bin\\x64\\Cyberpunk2077.exe", _art: 1091500 },
    { key: "dir:d:\\games\\hollow knight", name: "Hollow Knight", source: "folder", exe: "D:\\Games\\Hollow Knight\\hollow_knight.exe", _art: 367520 },
  ];
  // As core.GroupCopies would group them; ps_separateCopies takes one out.
  let GROUPS = [
    { key: "steam:1145350", keys: ["steam:1145350", "dir:c:\\games\\hades ii"], primary: "dir:c:\\games\\hades ii" },
    { key: "steam:1091500", keys: ["steam:1091500", "dir:c:\\epic games\\cyberpunk 2077"], primary: "steam:1091500" },
    { key: "name:hollowknight", keys: ["dir:c:\\gog games\\hollow knight", "dir:d:\\games\\hollow knight"], primary: "dir:c:\\gog games\\hollow knight" },
  ];
  const SEPARATE = new Set();
  if (N === "0") { games = []; found = []; GROUPS = []; }
  if (N === "400") {
    const pool = [730, 1091500, 1426210, 413150, 367520, 1086940, 1145350, 1551360, 271590, 250900, 0, 0];
    const words = ["Iron", "Hollow", "Neon", "Last", "Crimson", "Quiet", "Broken", "Wild", "Northern", "Paper", "Deep", "Silver"];
    const nouns = ["Harbor", "Kingdom", "Signal", "Orchard", "Frontier", "Engine", "Garden", "Tides", "Circuit", "Lantern"];
    for (let i = 0; i < 390; i++) {
      const name = `${words[i % words.length]} ${nouns[(i * 7) % nouns.length]}${i % 9 === 0 ? ": The Long Way Back Home Edition" : i % 4 === 0 ? " " + (i % 5 + 2) : ""}`;
      found.push({ key: "dir:x\\" + i, name, source: ["steam", "epic", "gog", "folder", "xbox"][i % 5], exe: "x.exe", _art: pool[i % pool.length] });
    }
  }
  found.forEach((f) => { f.dir = f.dir || "D:\\Games\\" + f.name; f.paths = f.paths || []; f.suggestions = []; f.processName = f.processName || "game"; });

  const active = {}, profiles = {}, recent = {}, playtime = {}, shelf = {}, steamTime = {};
  games.forEach((g, i) => {
    profiles[g.id] = g._players;
    active[g.id] = g._players[g._players.length > 1 ? 1 : 0];
    // The last player, plus each player's own last time (players).
    const last = g._players[i % g._players.length];
    recent["game:" + g.id] = { at: iso((i + 1) * 5 * h), player: last,
      players: Object.fromEntries(g._players.map((p, j) => [p, iso((i + 1) * 5 * h + (p === last ? 0 : (j + 1) * 26 * h))])) };
    playtime["game:" + g.id] = Object.fromEntries(g._players.map((p, j) => [p, (j + 1) * (i + 2) * 11 * h / 1000]));
    delete g._players;
  });
  recent["steam:1091500"] = { at: iso(2 * d), player: "p1" };
  recent["dir:c:\\games\\hades ii"] = { at: iso(3 * d), player: "p1" };
  playtime["steam:1145350"] = { p1: 12 * 3600 }; playtime["dir:c:\\games\\hades ii"] = { p1: 7.5 * 3600 };
  playtime["dir:c:\\epic games\\cyberpunk 2077"] = { p1: 16 * 3600 };
  recent["dir:d:\\games\\onimusha"] = { at: iso(43 * 60e3) };
  recent["steam:413150"] = { at: iso(9 * d) };
  if (ARTQ === "partial") recent["dir:d:\\games\\phantom blade zero"] = { at: iso(20 * 60e3) }; // on the shelves, wide art made from its cover
  steamTime["steam:1091500"] = 84 * 3600; steamTime["game:cs2"] = 412 * 3600; steamTime["steam:250900"] = 133 * 3600;
  shelf.p1 = { "game:cs2": { status: "playing", rating: 4, updated: iso(d) }, "steam:1091500": { status: "playing", updated: iso(2 * d) }, "game:ittakestwo": { status: "finished", rating: 5, note: "Best co-op game", updated: iso(20 * d) } };
  const sessions = STATE === "playing" && games[0] ? [{ gameId: "cs2", playerId: ids[ids.length > 1 ? 1 : 0], phase: "playing", since: iso(40 * 60e3) }] : [];
  const days = {};
  // Most of a year, across a few games, for the profile page and the recap.
  const dayGames = ["game:cs2", "game:bg3", "game:bg3", "steam:1091500", "game:ittakestwo", "steam:413150", "game:bg3"];
  for (let i = 0; i < 300; i++) if (Math.sin(i * 1.7) > 0.2) { const t = new Date(now - i * d); days[t.toISOString().slice(0, 10)] = { [dayGames[(i * 5) % dayGames.length]]: Math.round(Math.abs(Math.sin(i)) * 9000) + 1200 }; }

  const settings = {
    startMinimized: false, closeToTray: true, hideWhilePlaying: true, watchOutsideLaunches: true, notifications: true,
    backupsToKeep: 10, startTimeoutSeconds: 180, steamAccountId: "", autoSteamHooks: true, downloadCovers: true, checkUpdates: true,
    gameNews: true, autoInstallUpdates: true, updatesAsked: true, gameSaves: CLOUD, saveDb: CLOUD, findGames: q.get("find") !== "0",
    gameDetails: q.get("details") !== "0", // ?details=0: Game details off
    backupDir: CLOUD ? "C:\\Users\\cris\\OneDrive" : "", backupAuto: CLOUD, backupKeep: 5, cloudSync: CLOUD,
  };
  // As core.WithOffsets: each profile's correction on top, never below 0.
  const withOffsets = () => {
    const out = JSON.parse(JSON.stringify(playtime));
    Object.entries(shelf).forEach(([pid, gs]) => Object.entries(gs).forEach(([k, e]) => { if (e.offset) { out[k] = out[k] || {}; out[k][pid] = Math.max(0, (out[k][pid] || 0) + e.offset); } }));
    return out;
  };
  let FIRSTP = FIRST && [{ id: "p1", name: "Cris", color: "#6C9BD2" }]; // until Continue saves it
  const state = () => ({
    version: "1.45.0", stage: "beta", safe: q.get("safe") === "1", firstRun: FIRST, loadError: STATE === "error" ? "config.json was unreadable and was renamed to config.json.broken: unexpected end of JSON input" : "",
    config: { version: 2, players: FIRSTP || PLAYERS, games: FIRST ? [] : games, settings, libraryPrefs: {}, addedGames: [], ignoredFound: Object.keys(HIDDEN), tags: TAGS, offline: OFFLINE },
    conflicts: { cs2: { at: iso(2 * h), player: "p1", files: ["{STEAMUSER}/730/local/cfg/config.cfg"] } },
    active, sessions, profiles, steam: { found: true, path: "C:\\Program Files (x86)\\Steam", accounts: ["cristian_d"], accountInUse: "cristian_d" },
    autostart: true, dataDir: "C:\\Users\\cris\\AppData\\Local\\Lootnest", colors: ["#6C9BD2", "#D9738E", "#5FB49C", "#C9A35F", "#9C88D0", "#D98A5F"],
    sources: { cs2: "steam", ittakestwo: "steam", bg3: "steam" }, recent, playtime: withOffsets(), steamTime, avatars: {}, banners: {}, shelf,
    customArt: customIndex(),
    activity: { days: { p1: days }, longest: { p1: { key: "game:cs2", seconds: 6 * 3600, at: iso(30 * d) } }, sessions: { p1: 142 } },
  });

  const OFFLINE = {};
  const DETEDITS = {};
  const detailsEv = (key) => {
    const appId = key === "game:bg3" ? 1086940 : key === "game:cs2" ? 730 : key === "game:ittakestwo" ? 1426210 : ((found.find((f) => f.key === key) || {}).appId || 0);
    const e = DETEDITS[key];
    let det = null;
    if (appId === 1086940) det = {
      summary: "Gather your party and return to the Forgotten Realms in a tale of fellowship and betrayal, sacrifice and survival, and the lure of absolute power. Mysterious abilities are awakening inside you, drawn from a mind flayer parasite planted in your brain. Resist, and turn darkness against itself. Or embrace corruption, and become ultimate evil.",
      genres: ["Adventure", "RPG", "Strategy"], developers: ["Larian Studios"], publishers: ["Larian Studios"], released: "3 Aug, 2023",
      screenshots: [1086940, 1091500, 1145350, 1426210, 1551360, 367520, 413150, 271590].map((id) => ({ thumb: art(id, "hero"), full: art(id, "hero") })),
      reviewScore: "Overwhelmingly Positive", reviewPercent: 96, reviewCount: 612345, metacritic: 96, storeUrl: "https://store.steampowered.com/app/1086940/", fetched: iso(4 * d),
    };
    else if (appId) det = { summary: "A short summary of the game from the Steam store.", genres: ["Action"], developers: ["A studio"], publishers: ["A publisher"], released: "1 Jan, 2020",
      reviewScore: "Very Positive", reviewPercent: 92, reviewCount: 12345, storeUrl: `https://store.steampowered.com/app/${appId}/`, fetched: iso(d) };
    if (e) {
      det = Object.assign({}, det || {});
      if (e.summary !== undefined) det.summary = e.summary;
      if (e.developer !== undefined) det.developers = [e.developer];
      if (e.publisher !== undefined) det.publishers = [e.publisher];
      if (e.released !== undefined) det.released = e.released;
      if (e.genres) det.genres = e.genres.map((g) => g.trim()).filter(Boolean);
    }
    return { type: "details", key, details: det, edited: !!e };
  };
  const POINTS = {};

  const TAGS = { "game:cs2": ["Co-op", "Shooters"], "game:ittakestwo": ["Co-op", "For the kids"], "steam:413150": ["Co-op", "Weekend", "For the kids"],
    "steam:1091500": ["Weekend"], "dir:c:\\games\\hades ii": ["Weekend"], "dir:c:\\gog games\\hollow knight": ["Weekend"], "game:bg3": ["Co-op", "Weekend"] };
  const emit = (ev, delay) => setTimeout(() => window.__ps && window.__ps(ev), delay || 40);
  const ok = (v) => Promise.resolve(v);
  const never = () => new Promise(() => {});
  const offline = () => Promise.reject("Couldn't reach the internet. Check your connection, then try again.");
  const appOf = (appId, name) => {
    if (appId) { const f = found.find((x) => x.appId === appId); return f && f._art ? f._art : appId; }
    const f = found.find((x) => x.name === name);
    return f && f._art;
  };
  const gameArt = (appId, name, kind) => {
    const f = !appId && found.find((x) => x.name === name);
    return f && f._only && f._only !== kind ? "" : art(appOf(appId, name), kind);
  };
  const NEWS = [
    { title: "Patch 2.3: new co-op mode and 40 fixes", date: iso(2 * d), update: true, summary: "Play the campaign with a friend, plus balance changes across every class.", url: "https://store.steampowered.com/news/1", body: "<p>Patch notes.</p>", image: art(1091500, "hero") },
    { title: "Developer update: the road to 3.0", date: iso(12 * d), update: false, summary: "What we're working on for the next big update.", url: "https://store.steampowered.com/news/2", body: "<p>Roadmap.</p>" },
    { title: "Hotfix 2.2.1", date: iso(20 * d), update: true, summary: "Fixes a crash when loading older saves.", url: "https://store.steampowered.com/news/3", body: "<p>Hotfix.</p>", image: art(1086940, "hero") },
    { title: "Patch 2.2: photo mode", date: iso(34 * d), update: true, summary: "Photo mode, new difficulty settings and many fixes.", url: "https://store.steampowered.com/news/4", body: "<p>Patch notes.</p>" },
    { title: "Community spotlight", date: iso(41 * d), update: false, summary: "Your best screenshots from the last month.", url: "https://store.steampowered.com/news/5", body: "<p>Spotlight.</p>", image: art(1245620, "hero") },
    { title: "Patch 2.1: performance", date: iso(55 * d), update: true, summary: "Faster loading and smoother frame times.", url: "https://store.steampowered.com/news/6", body: "<p>Patch notes.</p>" },
    { title: "Patch 2.0: the big one", date: iso(80 * d), update: true, summary: "A new act, new classes and a reworked inventory.", url: "https://store.steampowered.com/news/7", body: "<p>Patch notes.</p>" },
  ];

  const api = {
    // ?bp=1: "Open in Big Picture when Lootnest starts" is on.
    ps_ready: () => ok(q.get("bp") === "1" ? { bigPicture: true } : {}),
    ps_restart: () => ok(null),
    ps_closeGame: () => ok(),
    // The last scan, per source: a mix of found, not installed, no games and an error.
    ps_sources: () => ok({ at: iso(2 * 60e3 + 5e3), sources: [
      ["steam", 7, "ok", "", "C:\\Program Files (x86)\\Steam"], ["epic", 0, "info", "Not installed"],
      ["gog", 0, "info", "Installed, no games yet", "C:\\Program Files (x86)\\GOG Galaxy"], ["xbox", 1, "ok", "", "D:\\XboxGames"],
      ["ea", 0, "info", "Not installed"], ["ubisoft", 0, "info", "Installed, no games yet"], ["battlenet", 0, "info", "Not installed"],
      ["riot", 0, "info", "Not installed"], ["amazon", 0, "info", "Not installed"], ["rockstar", 0, "info", "Not installed"],
      ["itch", 0, "info", "Not installed"], ["hydra", 0, "warn", "Couldn't read its library: leveldb: manifest corrupted (field 'comparer'): missing [file=MANIFEST-000004]", "C:\\Users\\you\\AppData\\Roaming\\hydralauncher\\hydra-db"],
      ["folder", 3, "ok", ""], ["shortcut", 1, "ok", ""],
    ].map(([source, found, state, reason, path]) => ({ source, found, installed: reason !== "Not installed", state, reason, path: path || "" })) }),
    ps_state: () => (STATE === "loading" || BOOT ? new Promise((r) => setTimeout(() => r(state()), BOOT || 60)) : ok(state())),
    ps_found: () => ok(STATE === "loading" ? { scanning: true, refreshing: true, games: [], groups: [] } : { scanning: false, refreshing: false, games: found, hidden: HIDDEN, groups: GROUPS.filter((g) => !SEPARATE.has(g.key)) }),
    ps_setPrimaryCopy: (group, key) => { GROUPS.forEach((g) => { if (g.key === group) g.primary = key; }); return ok(); },
    ps_separateCopies: (group, apart) => { if (apart) SEPARATE.add(group); else SEPARATE.delete(group); return ok(); },
    ps_cover: (appId, name) => (STATE === "offline" ? offline() : ok(gameArt(appId, name, "cover"))),
    ps_art: (appId, name, key, kind) => (STATE === "offline" ? ok("") : ok(gameArt(appId, name, kind))),
    ps_customArt: (key, kind) => ok(CUSTOM[key + ":" + kind] || ""),
    ps_setCustomArt: (key, kind) => { CUSTOM[key + ":" + kind] = kind === "cover" ? MINE : art(1086940, kind); return ok(); },
    ps_clearCustomArt: (key, kind) => { delete CUSTOM[key + ":" + kind]; return ok(); },
    // Onimusha has an icon in its exe; the other games have none.
    ps_gameIcon: (exe) => ok(/onimusha/i.test(exe) ? "data:image/svg+xml," + encodeURIComponent('<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32"><rect x="2" y="2" width="28" height="28" rx="6" fill="#7a1f2b"/><path d="M9 23 16 7l7 16-7-4z" fill="#e8c36a"/></svg>') : ""),
    ps_gameNews: (key) => { if (STATE === "offline") emit({ type: "news", key, error: "Couldn't reach Steam's news feed.", items: [] }); else if (STATE !== "loading") emit({ type: "news", key, items: NEWS, newer: /phantom/i.test(key) ? { title: "Phantom Blade Zero - Patch Notes (Version 1.0302256)", date: iso(3 * d), update: true } : undefined }); return ok(); },
    // Game details: Baldur's Gate 3 has everything, other Steam games a
    // summary and facts, games without a Steam id none (or the user's edits).
    ps_gameDetails: (key) => {
      if (STATE === "offline") emit({ type: "details", key, error: "Couldn't reach the Steam store. Lootnest tries again next time you open this page." });
      else if (STATE !== "loading") emit(detailsEv(key));
      return ok();
    },
    ps_saveDetailsEdit: (key, e) => { if (Object.keys(e).length) DETEDITS[key] = e; else delete DETEDITS[key]; emit(detailsEv(key)); return ok(); },
    ps_cloudStatus: () => ok({ state: "overlap", count: 1 }),
    ps_steamSetup: () => ok({ games: [], steamRunning: false }),
    ps_backupInfo: () => ok({
      sync: CLOUD, locked: false, dir: settings.backupDir, target: CLOUD ? settings.backupDir + "\\Lootnest" : "", auto: CLOUD, keep: 5, encrypted: false, computer: "COCONUT",
      status: STATE === "error" ? { error: "The backup folder (C:\\Users\\cris\\OneDrive) isn't there. Is the drive connected, and the cloud app running?", errorAt: iso(60e3), last: iso(3 * d), size: 48e6 } : { last: iso(25 * 60e3), size: 48211000 },
      folders: [{ name: "OneDrive", path: "C:\\Users\\cris\\OneDrive" }, { name: "Google Drive", path: "G:\\My Drive" }],
    }),
    ps_savesStatus: () => { if (STATE !== "loading") emit({ type: "savesStatus", status: STATE === "error" ? { error: "The cloud folder (C:\\Users\\cris\\OneDrive) isn't there.", games: [], players: [] } : { games: [{ gameId: "cs2", playerId: "p1", state: "same", computer: "COCONUT", changed: iso(h) }, { gameId: "cs2", playerId: ids[1] || "p1", state: "conflict", computer: "LAPTOP", changed: iso(3 * h), here: iso(2 * h) }, { gameId: "bg3", playerId: "p1", state: "down", computer: "LAPTOP", changed: iso(3 * h) }], players: [] } }); return ok(); },
    ps_findGames: (term) => {
      const cap = (id) => `https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/${id}/capsule_231x87.jpg`;
      const all = [
        { appId: 1245620, name: "ELDEN RING", image: cap(1245620), price: { currency: "EUR", initial: 5999, final: 2999, discount: 50 } },
        { appId: 2622380, name: "ELDEN RING NIGHTREIGN", image: cap(2622380), price: { currency: "EUR", initial: 3999, final: 3999 } },
        { appId: 1091500, name: "Cyberpunk 2077", image: cap(1091500), price: { currency: "EUR", initial: 5999, final: 2999, discount: 50 }, owned: "found:steam:1091500" },
        { appId: 570, name: "Dota 2", image: cap(570) },
      ];
      setTimeout(() => {
        if (STATE === "offline") emit({ type: "findGames", q: term, error: "Couldn't reach the Steam store. Check the connection and try again." });
        else emit({ type: "findGames", q: term, hits: all.filter((h) => h.name.toLowerCase().includes(term.trim().toLowerCase().split(" ")[0]) || term.trim().length < 2) });
      }, 150);
      return ok();
    },
    ps_storeFront: () => {
      const hdr = (id) => `https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/${id}/header.jpg`;
      const eur = (i, f) => ({ currency: "EUR", initial: i, final: f, discount: i > f ? Math.round((i - f) * 100 / i) : 0 });
      const g = (appId, name, p, extra) => Object.assign({ appId, name, image: hdr(appId), price: p }, extra || {});
      setTimeout(() => {
        if (STATE === "offline") { emit({ type: "storeFront", error: "Couldn't reach the Steam store. Check the connection and try again." }); return; }
        // Phantom Blade Zero's picture 404s.
        emit({ type: "storeFront", front: {
          daily: g(1145360, "Hades", eur(2450, 612)),
          specials: [g(1245620, "ELDEN RING", eur(5999, 2999)), g(1086940, "Baldur's Gate 3", eur(5999, 4799)), g(413150, "Stardew Valley", eur(1399, 979)), g(1091500, "Cyberpunk 2077", eur(5999, 2999), { owned: "found:steam:1091500" }),
            g(367520, "Hollow Knight", eur(1499, 749)), g(1426210, "It Takes Two", eur(3999, 1599)), g(242760, "The Forest", eur(1699, 424)), g(250900, "The Binding of Isaac: Rebirth", eur(1499, 449))],
          deals: [g(632470, "Disco Elysium - The Final Cut", { currency: "USD", discount: 80 }, { store: "IndieGala", usd: 7.89, score: 97, url: "https://www.cheapshark.com/redirect?dealID=a" }),
            g(49520, "Borderlands 2", { currency: "USD", discount: 90 }, { store: "Steam", usd: 1.99, score: 89, url: "https://www.cheapshark.com/redirect?dealID=b" }),
            g(115800, "Owlboy", { currency: "USD", discount: 90 }, { store: "GOG", usd: 1.99, score: 88, url: "https://www.cheapshark.com/redirect?dealID=c" }),
            g(1551360, "Forza Horizon 5", { currency: "USD", discount: 50 }, { store: "Steam", usd: 29.99, score: 92, url: "https://www.cheapshark.com/redirect?dealID=d" }),
            g(271590, "Grand Theft Auto V Legacy", { currency: "USD", discount: 50 }, { store: "Fanatical", usd: 14.99, score: 96, url: "https://www.cheapshark.com/redirect?dealID=e" }),
            g(1145350, "Hades II", { currency: "USD", discount: 20 }, { store: "GreenManGaming", usd: 23.99, score: 95, url: "https://www.cheapshark.com/redirect?dealID=f" })],
          top: [g(4115450, "Phantom Blade Zero", eur(5999, 5999), { image: "missing-header.jpg" }), g(570, "Dota 2", null), g(730, "Counter-Strike 2", null, { owned: "game:cs2" }),
            g(1086940, "Baldur's Gate 3", eur(5999, 5999)), g(1245620, "ELDEN RING", eur(5999, 2999)), g(2358720, "Black Myth: Wukong", eur(5999, 5999)),
            g(1091500, "Cyberpunk 2077", eur(5999, 2999)), g(413150, "Stardew Valley", eur(1399, 1399)), g(1145360, "Hades", eur(2450, 612)), g(271590, "Grand Theft Auto V Legacy", eur(2999, 1499))],
        } });
      }, 150);
      return ok();
    },
    ps_gameOffer: (appId, name) => {
      const req = (rec) => [{ value: "Requires a 64-bit processor and operating system" }, { label: "OS", value: rec ? "Windows 10/11" : "Windows 10" },
        { label: "Processor", value: rec ? "INTEL CORE I7-8700K or AMD RYZEN 5 3600X" : "INTEL CORE I5-8400 or AMD RYZEN 3 3300X" },
        { label: "Memory", value: rec ? "16 GB RAM" : "12 GB RAM" }, { label: "Graphics", value: rec ? "NVIDIA GEFORCE GTX 1070 8 GB or AMD RADEON RX VEGA 56 8 GB" : "NVIDIA GEFORCE GTX 1060 3 GB or AMD RADEON RX 580 4 GB" },
        { label: "DirectX", value: "Version 12" }, { label: "Storage", value: "60 GB available space" }];
      const offer = { appId, name, header: `https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/${appId}/header.jpg`, free: appId === 570,
        steam: appId === 570 ? null : { currency: "EUR", initial: 5999, final: 2999, discount: 50 },
        deals: appId === 570 ? [] : [{ store: "GameBillet", price: 26.69, retail: 59.99, savings: 56, url: "https://www.cheapshark.com/redirect?dealID=a" }, { store: "Fanatical", price: 27.99, retail: 59.99, savings: 53, url: "https://www.cheapshark.com/redirect?dealID=b" },
          { store: "GOG", price: 59.99, retail: 59.99, savings: 0, url: "https://www.cheapshark.com/redirect?dealID=c" }],
        lowest: appId === 570 ? 0 : 23.99, lowestAt: iso(120 * d), minimum: req(false), recommended: req(true), storeUrl: `https://store.steampowered.com/app/${appId}/` };
      const weak = appId === 2622380, mid = appId === 1086940;
      const runs = (g, c) => ({ runs: [g, c].includes("below") ? "below" : [g, c].includes("minimum") ? "minimum" : [g, c].includes("recommended") ? "recommended" : "high",
        limit: g !== "high" && g !== "recommended" ? "Graphics" : "", parts: [
          { part: "Graphics", have: weak ? 2300 : mid ? 4200 : 13500, min: 3900, rec: 5900, runs: g, known: true },
          { part: "Processor", have: 29.1, min: 9.2, rec: 13.5, runs: c, known: true },
          { part: "Memory", have: 16 * 2 ** 30, min: 12 * 2 ** 30, rec: 16 * 2 ** 30, runs: "recommended", known: true },
          { part: "Storage", have: 412 * 2 ** 30, min: 60 * 2 ** 30, rec: 60 * 2 ** 30, runs: "recommended", known: true }] });
      const verdict = appId === 570 ? { runs: "", parts: [{ part: "Graphics", have: 13500 }, { part: "Processor", have: 29.1 }, {}, {}] } : weak ? runs("below", "high") : mid ? runs("minimum", "high") : runs("high", "high");
      setTimeout(() => emit({ type: "gameOffer", appId, owned: appId === 1091500 ? "found:steam:1091500" : "", offer, verdict,
        pc: { cpu: "AMD Ryzen 7 5800X 8-Core Processor", gpu: "NVIDIA GeForce RTX 3070", vram: 8 * 2 ** 30, ram: 16 * 2 ** 30, free: 412 * 2 ** 30, freeOn: "D:", windows: "Windows 11" } }), 200);
      return ok();
    },
    ps_gameSaves: (key) => {
      if (STATE === "loading") return ok();
      const f = found.find((x) => x.key === key) || {};
      const info = { key, locations: [], versions: [], source: "" };
      if (/phantom blade/i.test(f.name || "")) info.local = [{ id: "n1", created: iso(3 * h), computer: "COCONUT", size: 2.3e6 }, { id: "n2", created: iso(2 * d), computer: "COCONUT", size: 2.1e6 }];
      if (f.source === "steam") info.off = "steam";
      else if (!CLOUD) info.off = "nocloud";
      else if (STATE === "error") info.error = "The cloud folder (C:\\Users\\cris\\OneDrive) isn't there. Is the drive connected, and the cloud app running?";
      else if (f.name === "Onimusha: Way of the Sword") info.source = "";
      else Object.assign(info, { source: "list", provider: "OneDrive", cloudDir: "C:\\Users\\cris\\OneDrive\\Lootnest\\saves\\name-" + (f.name || "").toLowerCase().replace(/[^a-z0-9]+/g, "-"),
        locations: [{ path: "%APPDATA%\\" + (f.name || "Game").replace(/\W+/g, "") + "\\Saves", resolved: "C:\\x", size: 18.4e6 }],
        versions: [{ id: "a", created: iso(h), computer: "COCONUT", size: 5.1e6 }, { id: "b", created: iso(2 * d), computer: "LAPTOP", size: 4.9e6 }] });
      emit({ type: "gameSaves", info });
      return ok();
    },
    ps_log: () => ok("10:42 Launched Counter-Strike 2 for Mihai\n10:20 Saved Cristian's files for It Takes Two"),
    ps_setTags: (key, tags) => { if (tags.length) TAGS[key] = tags; else delete TAGS[key]; return ok(); },
    ps_setShelf: (pid, key, patch) => { shelf[pid] = shelf[pid] || {}; const e = { ...shelf[pid][key], ...patch, updated: new Date().toISOString() }; if (e.status || e.rating || e.note || e.offset) shelf[pid][key] = e; else delete shelf[pid][key]; return ok(); },
    // Saving profiles gives new ones an id, as Go does.
    ps_savePlayers: (list) => { FIRSTP = null; PLAYERS.splice(0, PLAYERS.length, ...list.map((p, i) => ({ ...p, id: p.id || "p" + (Date.now() % 100000) + i }))); return ok(); },
    // A first-timer is asked how to start, as in Go.
    ps_play: (req) => ok(!req.newProfile && !(profiles[req.gameId] || []).includes(req.playerId) && req.gameId in profiles ? { need: "newMode" } : {}),
    // Each player's profile: saved a while ago, a few MB.
    ps_profileInfo: (gid) => { if (STATE !== "loading") emit({ type: "profileInfo", gameId: gid, info: Object.fromEntries((profiles[gid] || []).map((p, j) => [p, { saved: iso((j + 1) * 22 * h), size: (j + 1) * 7.3 * 1048576 }])) }); return ok(); },
    // Achievements: CS2 from Steam's cache, a folder game from a file on this PC.
    ps_achievements: (key) => {
      const mk = (n, got, names) => names.slice(0, n).map((nm, i) => ({ id: "ACH_" + i, name: nm, description: "Do the thing number " + (i + 1), unlocked: i < got, at: i < got ? iso((i + 1) * 9 * h) : undefined, percent: [62.4, 31, 12.8, 4.2, 0.9, 48, 22, 7.5][i % 8], hidden: i === n - 1 }));
      const names = ["First blood", "The Fool", "Clean sweep", "Night owl", "Against all odds", "Pacifist", "Collector", "Completionist", "Speed demon", "Old friends", "Last one standing", "Secret ending"];
      const set = key === "game:cs2" ? { appId: 730, source: "steam", total: 12, unlocked: 5, list: mk(12, 5, names) }
        : key === "dir:d:\\games\\onimusha" ? null
        : key.startsWith("dir:") ? { appId: 1091500, source: "pc", total: 12, unlocked: 3, list: mk(12, 3, names) }
        : key === "game:bg3" ? { appId: 1086940, source: "pc", total: 12, unlocked: 7, list: mk(12, 7, names),
          players: Object.fromEntries((profiles.bg3 || []).map((p, i) => [p, { source: "pc", total: 12, unlocked: [7, 2, 11, 4, 0, 9][i % 6], list: mk(12, [7, 2, 11, 4, 0, 9][i % 6], names) }])) } : null;
      if (STATE !== "loading") emit({ type: "achievements", key, set });
      return ok();
    },
    ps_testToast: () => ok(),
    ps_gameBackups: (id) => ok([
      { id: "20261107-201500", when: iso(40 * 60e3), owner: "p2", size: 15e6 },
      { id: "20261107-181000", when: iso(3 * h), owner: "p1", size: 7.3e6 },
      { id: "20261106-210000", when: iso(d), owner: "p2", size: 14e6 },
    ]),
    ps_restoreBackup: () => ok({ owner: "p1", where: "profile" }),
    ps_dismissConflict: () => ok(),
    ps_gameFacts: (dir) => ok({ antiCheat: /onimusha/i.test(dir) ? "Easy Anti-Cheat" : "" }),
    ps_bigPicture: () => ok(),
    ps_saveSettings: (r) => { Object.assign(settings, r.settings); return ok(); },
    ps_saveCheck: () => {
      setTimeout(() => emit({ type: "saveCheck", check: { profiles: 4, net: 3, points: 1, cloud: false, gone: [{ key: "dir:d:\\games\\hydra\\hollow tale", name: "Hollow Tale" }],
        unknown: [{ key: "dir:d:\\games\\onimusha", name: "Onimusha: Way of the Sword" }], unknownN: 1,
        ...(q.has("onedrive") ? { oneDriveIn: "Documents", oneDrive: [{ key: "game:bg3", name: "Baldur's Gate 3" }, { key: "dir:d:\\games\\phantom blade zero", name: "Phantom Blade Zero" }] } : { oneDrive: [] }) } }), 120);
      return ok();
    },
    ps_savePoints: (key) => {
      if (!POINTS[key]) POINTS[key] = key === "game:bg3" ? [
        { id: "20261107-214500", name: "Before the update", created: iso(10 * 60e3), player: "p1", playtime: 313000, unlocked: 21, size: 9e7, locs: ["a"], auto: true, update: true },
        { id: "20261107-201500", name: "Before the Netherbrain", created: iso(40 * 60e3), player: "p1", playtime: 312000, unlocked: 21, size: 9e7, locs: ["a"], thumb: art(1091500, "hero") },
        { id: "20261107-195000", name: "Before going back to Act 2 start", created: iso(70 * 60e3), player: "p1", playtime: 310000, unlocked: 20, size: 9e7, locs: ["a"], auto: true },
        { id: "20261104-210000", name: "Act 2 start", created: iso(3 * d), player: "p1", playtime: 250000, unlocked: 14, size: 8e7, locs: ["a"], thumb: art(1086940, "hero") },
        { id: "20261101-180000", name: "Honour mode, gate", created: iso(6 * d), player: "p2", playtime: 90000, unlocked: 6, size: 6e7, locs: ["a"] },
      ] : [];
      setTimeout(() => emit({ type: "savePoints", key, points: POINTS[key], player: "p1", running: false, profiles: key.startsWith("game:") }), 80);
      return ok();
    },
    // A week of Baldur's Gate 3: sessions, save points, unlocks, an update.
    ps_gameTimeline: (key) => {
      const p2 = ids[1] || "p1";
      const ev = (kind, ago, o) => ({ kind, at: iso(ago), ...(kind === "session" ? { start: iso(ago + o.seconds * 1e3) } : {}), ...o });
      const items = key !== "game:bg3" ? [] : [
        ev("point", 10 * 60e3, { name: "Before the update", profile: "p1" }),
        ev("update", 12 * 60e3, { name: "Patch8_Hotfix2.zip", version: "4.1.1.6" }),
        ev("session", 30 * 60e3, { profile: "p1", seconds: 4800 }),
        ev("shot", 32 * 60e3, { name: "2026-11-08 20-58-12.png", thumb: art(1145350, "hero") }),
        ev("point", 40 * 60e3, { name: "Before the Netherbrain", profile: "p1" }),
        ev("shot", 50 * 60e3, { name: "2026-11-08 20-40-03.png", thumb: art(1426210, "hero") }),
        ev("unlock", 55 * 60e3, { name: "Descent From Avernus", profile: "p1" }),
        ev("point", 70 * 60e3, { name: "Before going back to Act 2 start", profile: "p1" }),
        ev("session", d + 2 * h, { profile: p2, seconds: 2700 }),
        ev("unlock", d + 2.5 * h, { name: "Hag Hunter", profile: p2 }),
        ev("session", d + 5 * h, { profile: "p1", seconds: 7380 }),
        ev("unlock", d + 6 * h, { name: "Relic Hunter", profile: "p1" }),
        ev("session", 2 * d + 3 * h, { profile: "p1", seconds: 3900 }),
        ev("session", 3 * d + 1 * h, { profile: "p1", seconds: 10800 }),
        ev("point", 3 * d + 2 * h, { name: "Act 2 start", profile: "p1" }),
        ev("unlock", 3 * d + 3 * h, { name: "Goblin Hunter", profile: "p1" }),
        ev("unlock", 3 * d + 4 * h, { name: "The Plot Thickens", profile: "p1" }),
        ev("session", 4 * d + 2 * h, { profile: p2, seconds: 1500 }),
        ev("session", 5 * d + 1 * h, { profile: "p1", seconds: 5400 }),
        ev("update", 5 * d + 3 * h, { name: "BG3_Patch8.7z" }),
        ev("session", 6 * d + 2 * h, { profile: p2, seconds: 6600 }),
        ev("point", 6 * d + 3 * h, { name: "Honour mode, gate", profile: p2 }),
        ev("unlock", 6 * d + 4 * h, { name: "Roleplayer", profile: p2 }),
        ev("session", 6 * d + 6 * h, { profile: "p1", seconds: 900 }),
        ev("session", 7 * d + 1 * h, { profile: "p1", seconds: 3000 }),
        ev("unlock", 7 * d + 2 * h, { name: "Act One", profile: "p1" }),
      ];
      setTimeout(() => emit({ type: "timeline", key, items }), 90);
      return ok();
    },
    // Full-size pictures for the lightbox: the harness has no larger art.
    ps_picture: (key, ref) => ok(ref.startsWith("point:") ? ((POINTS[key] || []).find((p) => "point:" + p.id === ref) || {}).thumb || "" : art(1145350, "hero")),
    ps_makeSavePoint: (key, name) => {
      const p = { id: "20261107-213000", name: name || "Save point, 7 Nov, 21:30", created: new Date().toISOString(), player: "p1", playtime: 313000, unlocked: 21, size: 9e7, locs: ["a"] };
      setTimeout(() => { POINTS[key] = [p, ...(POINTS[key] || [])]; emit({ type: "savePointMade", key, point: p }); api.ps_savePoints(key); }, 400);
      return ok();
    },
    ps_restoreSavePoint: (key, id) => { setTimeout(() => emit({ type: "savePointMade", key, restored: id, point: (POINTS[key] || []).find((p) => p.id === id) }), 400); return ok(); },
    ps_deleteSavePoint: (key, id) => { POINTS[key] = (POINTS[key] || []).filter((p) => p.id !== id); api.ps_savePoints(key); return ok(); },
    ps_branchSavePoint: () => ok("p1"),
    ps_setOffline: (key, on) => {
      setTimeout(() => { if (on) OFFLINE[key] = ["D:\\Games\\Onimusha\\Onimusha.exe", "D:\\Games\\Onimusha\\CrashReporter.exe"]; else delete OFFLINE[key]; emit({ type: "offline", key, on }); }, 300);
      return ok();
    },
    ps_achDoctor: (key) => {
      setTimeout(() => emit({ type: "achDoctor", key, steps: [
        { state: "ok", title: "It's Steam game 2134500", detail: "Lootnest found its Steam id from steam_appid.txt, or a tool's settings, next to the game." },
        { state: "ok", title: "Goldberg (GSE) is next to the game", detail: "Lootnest reads the achievement files it keeps.", path: "D:\\Games\\Onimusha" },
        { state: "warn", title: "No achievement file yet", detail: "Tools usually write it when the first achievement unlocks. Lootnest looked in 17 places, such as C:\\Users\\cris\\AppData\\Roaming\\Goldberg SteamEmu Saves\\2134500\\achievements.json." },
        { state: "warn", title: "Achievements show by their ids", detail: "For their names, descriptions and pictures, add your free Steam Web API key in Settings, Online." },
      ], nameInGame: { name: "goldberg", settings: "D:\\Games\\Onimusha\\steam_settings" } }), 250);
      return ok();
    },
    ps_allAchievements: () => {
      const rec = (n, off) => Array.from({ length: n }, (_, i) => ({ id: "A" + i, name: ["First blood", "The Fool", "Clean sweep", "Night owl", "Against all odds", "Pacifist"][i % 6], unlocked: true, at: iso((i + off) * 7 * h), percent: [48.2, 22.5, 7.4, 1.3, 63, 12][(i + off) % 6] }));
      const rar = (list) => list.map((a, i) => ({ ...a, id: "R" + i })).sort((x, y) => x.percent - y.percent);
      const named = (n) => [{ name: "Legend of the Gate", percent: 0.8 }, { name: "Ace", percent: 3.1 }, { name: "Speed demon", percent: 6.4 }, { name: "Collector", percent: 18 }].slice(0, n).map((x, i) => ({ id: "S" + i, unlocked: true, ...x }));
      const games = [
        { key: "game:cs2", name: "Counter-Strike 2", source: "steam", total: 167, unlocked: 61, recent: rec(4, 1), rarest: named(2) },
        { key: "game:bg3", name: "Baldur's Gate 3", source: "pc", total: 54, unlocked: 21, recent: rec(3, 2), players: Object.fromEntries((profiles.bg3 || []).map((p, i) => [p, { unlocked: [21, 54, 9][i % 3], recent: rec(3, i + 2), rarest: rar(rec(3, i + 2)).slice(0, 2) }])) },
        { key: "steam:1091500", name: "Cyberpunk 2077", source: "steam", total: 57, unlocked: 57, recent: rec(2, 5), rarest: named(4).slice(2) },
      ];
      if (STATE !== "loading") emit({ type: "allAchievements", games });
      return ok();
    },
    // A run to try a save in: a profile of the person, as Go makes it.
    ps_saveRecap: (name, year) => ok("C:\\Users\\you\\Pictures\\Lootnest\\" + name + " in " + year + ".png"),
    ps_trySave: (gid, from, loc, src, name) => {
      const base = PLAYERS.find((p) => p.id === from) || PLAYERS[0];
      const per = PLAYERS.find((p) => p.id === base.of) || base;
      const id = "p" + (Date.now() % 100000);
      PLAYERS.push({ id, name: name || [...per.name].slice(0, 17).join("").trim() + " trying", color: per.color, of: per.id });
      (profiles[gid] = profiles[gid] || []).push(id);
      return ok(id);
    },
    ps_browse: (id, kind) => {
      if (kind === "installer") emit({ type: "browse", reqId: id, path: "C:\\Users\\cris\\Downloads\\setup-pbz-update-1.0.3.exe" });
      else if (kind === "trySaveFile" || kind === "trySaveDir") emit({ type: "browse", reqId: id, path: kind === "trySaveFile" ? "C:\\Users\\cris\\Downloads\\BG3 Act 3 save from Mihnea.zip" : "C:\\Users\\cris\\Downloads\\BG3 save" });
      else if (kind === "art") emit({ type: "browse", reqId: id, path: "C:\\Users\\cris\\Pictures\\My cover.png" });
      else if (kind === "update" || kind === "updateDir") emit({ type: "browse", reqId: id, path: kind === "update" ? "C:\\Users\\cris\\Downloads\\PBZ.Update.v1.0.2.to.v1.0.3.rar" : "C:\\Users\\cris\\Downloads\\PBZ Update" });
      else emit({ type: "browse", reqId: id, cancelled: true });
      return ok();
    },
    ps_updates: () => {
      setTimeout(() => emit({ type: "updates", newsOn: true, games: N === "0" ? [] : [
        { key: "found:dir:d:\\games\\phantom blade zero", name: "Phantom Blade Zero", source: "hydra", dir: "D:\\Games\\Phantom.Blade.Zero-InsaneRamZes", changed: iso(9 * d),
          newer: { title: "Phantom Blade Zero - Patch Notes (Version 1.0302256)", date: iso(3 * d), update: true }, applied: [] },
        { key: "found:dir:d:\\games\\hydra\\hollow tale", name: "Hollow Tale", source: "hydra", dir: "D:\\Games\\Hydra\\Hollow Tale", changed: iso(2 * d),
          version: "1.2", applied: [{ id: "20261105-201500", source: "HollowTale-1.1-to-1.2.zip", at: iso(2 * d), added: ["a"], replaced: ["b", "c"], state: "applied", from: "1.1", to: "1.2" }, { id: "20261101-101500", source: "HollowTale-1.0-to-1.1.zip", at: iso(6 * d), added: [], replaced: ["b"], state: "applied", from: "1.0", to: "1.1" }] },
        { key: "found:dir:d:\\games\\onimusha", name: "Onimusha: Way of the Sword", source: "added", dir: "D:\\Games\\Onimusha", changed: iso(20 * d), noNews: true, applied: [] },
      ] }), 120);
      return ok();
    },
    ps_newCopies: () => {
      setTimeout(() => emit({ type: "newCopies", offers: [{ old: { key: "dir:d:\\games\\phantom.blade.zero.v1.0.1", name: "Phantom Blade Zero", dir: "D:\\Games\\Phantom.Blade.Zero.v1.0.1" },
        new: { key: "dir:d:\\games\\phantom blade zero", name: "Phantom Blade Zero", dir: "D:\\Games\\Phantom Blade Zero" }, oldGone: false, saves: ["ProjectPBZ/Saved"], profiles: false }] }), 100);
      return ok();
    },
    ps_carryOver: (oldKey) => { setTimeout(() => emit({ type: "carried", old: oldKey, saves: [{ rel: "ProjectPBZ/Saved", size: 7e7, replaced: true }], oldThere: true }), 500); return ok(); },
    ps_deleteOldCopy: (oldKey) => { setTimeout(() => emit({ type: "carried", old: oldKey, deleted: true }), 400); return ok(); },
    ps_updatePlan: (key, path) => {
      if (/\.exe$/i.test(path)) { setTimeout(() => emit({ type: "updatePlan", key, path, dir: "D:\\Games\\Phantom.Blade.Zero.v1.0.1", free: 412 * 2 ** 30, plan: { kind: "installer", source: path, name: "setup-pbz-update-1.0.3.exe", gameSize: 61 * 2 ** 30, free: 412 * 2 ** 30 } }), 150); return ok(); }
      if (/\.(rar|7z)$/i.test(path)) setTimeout(() => emit({ type: "updatePlan", key, path, opening: true }), 50);
      setTimeout(() => emit({ type: "updatePlan", key, path, dir: "D:\\Games\\Phantom.Blade.Zero.v1.0.1", free: 412 * 2 ** 30,
        check: { have: "1.0.1", from: "1.0.2", to: "1.0.3", state: "skips" }, plan: {
        source: path, kind: /\.zip$/i.test(path) ? "zip" : "folder", skipped: "PBZ Update v1.0302256", added: 3, replace: 12, bytes: 2.4 * 2 ** 30, kept: 2.1 * 2 ** 30,
        where: [{ path: "ProjectPBZ", files: 14 }, { path: "", files: 1 }], readme: ["README.txt"] } }), 200);
      return ok();
    },
    ps_updateApply: (key) => { setTimeout(() => emit({ type: "updateDone", key, update: { id: "20261107-190000", source: "PBZ-Update-v1.0302256.zip", added: ["a", "b", "c"], replaced: new Array(12).fill("x"), state: "applied" } }), 600); return ok(); },
    ps_updateInstaller: (key, path, full) => {
      ["copying", "installer", "checking"].forEach((step, i) => setTimeout(() => emit({ type: "updateProgress", key, step }), 100 + i * 500));
      setTimeout(() => emit({ type: "updateDone", key, update: { id: "20261107-191000", kind: "installer", source: "setup-pbz-update-1.0.3.exe", added: ["a", "b"], replaced: new Array(40).fill("x"), state: "applied", noUndo: !full } }), 1700);
      return ok();
    },
    ps_updateUndo: (key) => { setTimeout(() => emit({ type: "updateDone", key, undone: true, update: { id: "x", source: "HollowTale-1.2.zip", added: [], replaced: [], state: "undone" } }), 400); return ok(); },
    ps_downloads: () => { setTimeout(() => emit({ type: "downloadsFound", matches: DL }), 80); return ok(); },
    ps_ignoreDownload: (path) => { DL = DL.filter((m) => m.path !== path); setTimeout(() => emit({ type: "downloadsFound", matches: DL }), 80); return ok(); },
    ps_recycleDownload: (path) => { DL = DL.filter((m) => m.path !== path); setTimeout(() => emit({ type: "downloadsFound", matches: DL }), 80); return ok(); },
    ps_steamGames: () => ok([{ appId: 1091500, name: "Cyberpunk 2077", installDir: "Cyberpunk 2077" }]),
  };
  // Everything else answers quietly (the harness never changes anything).
  const handler = { get: (t, k) => t[k] || (() => (STATE === "offline" && /check|news|cover/i.test(k) ? offline() : ok({}))) };
  const all = new Proxy(api, handler);
  Object.keys(api).forEach((k) => { window[k] = api[k]; });
  // Functions the UI binds by name but the fixtures don't cover.
  ["ps_play", "ps_load", "ps_steamPick", "ps_savePlayers", "ps_saveSettings", "ps_saveGame", "ps_deleteGame", "ps_addFound", "ps_scan", "ps_open",
    "ps_shortcut", "ps_foundShortcut", "ps_launchFound", "ps_setShelf", "ps_syncNow", "ps_syncResolve", "ps_backupNow", "ps_setBackup", "ps_checkUpdate",
    "ps_setGameSaves", "ps_gameSaveNow", "ps_gameSaveRestore", "ps_gameSaveRestoreLocal", "ps_restoreOpen", "ps_restoreApply", "ps_restoreCancel", "ps_exportPlayer", "ps_importApply",
    "ps_inspectSteam", "ps_inspectExe", "ps_checkPath", "ps_cancelRequest", "ps_ignoreFound", "ps_unhideFound", "ps_foundGame", "ps_addLibraryGame", "ps_saveLibraryPref",
    "ps_setAvatar", "ps_removeAvatar", "ps_setBanner", "ps_removeBanner", "ps_setBackupPassword", "ps_moveInfo", "ps_moveGame", "ps_steamVerify",
    "ps_openGameFolder", "ps_applyHooks", "ps_fixBlocked", "ps_copy", "ps_installUpdate", "ps_quit"].forEach((k) => { if (!window[k]) window[k] = all[k]; });
  // ?pad=ds5,xbox: fake connected controllers, for the glyphs (a DualSense and its Xbox twin here).
  // Press with __harness.pads[i].buttons[b].pressed = true.
  const PAD_IDS = {
    ds5: "DualSense Wireless Controller (STANDARD GAMEPAD Vendor: 054c Product: 0ce6)",
    ds4: "Wireless Controller (STANDARD GAMEPAD Vendor: 054c Product: 09cc)",
    xbox: "Xbox 360 Controller (XInput STANDARD GAMEPAD)",
    switch: "Pro Controller (STANDARD GAMEPAD Vendor: 057e Product: 2009)",
    generic: "USB Gamepad (STANDARD GAMEPAD Vendor: 0079 Product: 0006)",
  };
  const pads = (q.get("pad") || "").split(",").filter((k) => PAD_IDS[k]).map((k, index) =>
    ({ id: PAD_IDS[k], index, connected: true, mapping: "standard", axes: [0, 0, 0, 0], buttons: Array.from({ length: 18 }, () => ({ pressed: false, value: 0 })) }));
  if (pads.length) {
    navigator.getGamepads = () => pads;
    setTimeout(() => pads.forEach((gamepad) => window.dispatchEvent(Object.assign(new Event("gamepadconnected"), { gamepad }))));
  }
  window.__harness = { players: PLAYERS, found, games, pads };
})();
