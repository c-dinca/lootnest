// Lootnest design harness: stands in for the Go side (window.ps_* and the
// __ps events) so the real UI runs in a plain browser with fixture data.
// Scenario from the URL: ?games=13|0|400&players=1|2|6&state=normal|loading|offline|error|playing&cloud=0|1&first=1
// ART is filled in by build.mjs: { "<appId>-cover|hero|logo": dataURL }.
(function () {
  const q = new URLSearchParams(location.search);
  const N = q.get("games") || "13", P = +(q.get("players") || 2), STATE = q.get("state") || "normal";
  const CLOUD = q.get("cloud") !== "0", FIRST = q.get("first") === "1";
  const now = Date.now(), h = 3600e3, d = 24 * h;
  const iso = (ms) => new Date(now - ms).toISOString();
  const ART = window.__ART || {};
  const art = (id, k) => ART[id + "-" + k] || "";

  const PLAYERS = [
    { id: "p1", name: "Cristian", color: "#6C9BD2" },
    { id: "p2", name: "Mihai", color: "#D9738E" },
    { id: "p3", name: "Ana", color: "#5FB49C" },
    { id: "p4", name: "Dan", color: "#C9A35F" },
    { id: "p5", name: "Ioana", color: "#9C88D0" },
    { id: "p6", name: "Radu", color: "#D98A5F" },
  ].slice(0, P);
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
    { key: "dir:d:\\games\\onimusha", name: "Onimusha: Way of the Sword", source: "added", added: true, exe: "D:\\Games\\Onimusha\\Onimusha.exe" }, // no art
  ];
  if (N === "0") { games = []; found = []; }
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
  recent["dir:d:\\games\\onimusha"] = { at: iso(43 * 60e3) };
  recent["steam:413150"] = { at: iso(9 * d) };
  steamTime["steam:1091500"] = 84 * 3600; steamTime["game:cs2"] = 412 * 3600; steamTime["steam:250900"] = 133 * 3600;
  shelf.p1 = { "game:cs2": { status: "playing", rating: 4, updated: iso(d) }, "steam:1091500": { status: "playing", updated: iso(2 * d) }, "game:ittakestwo": { status: "finished", rating: 5, note: "Best co-op game", updated: iso(20 * d) } };
  const sessions = STATE === "playing" && games[0] ? [{ gameId: "cs2", playerId: ids[ids.length > 1 ? 1 : 0], phase: "playing", since: iso(40 * 60e3) }] : [];
  const days = {};
  for (let i = 0; i < 120; i++) if (Math.sin(i * 1.7) > 0.2) { const t = new Date(now - i * d); days[t.toISOString().slice(0, 10)] = { "game:cs2": Math.round(Math.abs(Math.sin(i)) * 9000) }; }

  const settings = {
    startMinimized: false, closeToTray: true, hideWhilePlaying: true, watchOutsideLaunches: true, notifications: true,
    backupsToKeep: 10, startTimeoutSeconds: 180, steamAccountId: "", autoSteamHooks: true, downloadCovers: true, checkUpdates: true,
    gameNews: true, autoInstallUpdates: true, updatesAsked: true, gameSaves: CLOUD, saveDb: CLOUD,
    backupDir: CLOUD ? "C:\\Users\\cris\\OneDrive" : "", backupAuto: CLOUD, backupKeep: 5, cloudSync: CLOUD,
  };
  const state = () => ({
    version: "1.15.1", firstRun: FIRST, loadError: STATE === "error" ? "config.json was unreadable and was renamed to config.json.broken: unexpected end of JSON input" : "",
    config: { version: 2, players: FIRST ? [{ id: "p1", name: "Player 1", color: "#6C9BD2" }, { id: "p2", name: "Player 2", color: "#D9738E" }] : PLAYERS, games: FIRST ? [] : games, settings, libraryPrefs: {}, addedGames: [] },
    active, sessions, profiles, steam: { found: true, path: "C:\\Program Files (x86)\\Steam", accounts: ["cristian_d"], accountInUse: "cristian_d" },
    autostart: true, dataDir: "C:\\Users\\cris\\AppData\\Local\\Lootnest", colors: ["#6C9BD2", "#D9738E", "#5FB49C", "#C9A35F", "#9C88D0", "#D98A5F"],
    sources: { cs2: "steam", ittakestwo: "steam", bg3: "steam" }, recent, playtime, steamTime, avatars: {}, banners: {}, shelf,
    activity: { days: { p1: days }, longest: { p1: { key: "game:cs2", seconds: 6 * 3600, at: iso(30 * d) } }, sessions: { p1: 142 } },
  });

  const emit = (ev, delay) => setTimeout(() => window.__ps && window.__ps(ev), delay || 40);
  const ok = (v) => Promise.resolve(v);
  const never = () => new Promise(() => {});
  const offline = () => Promise.reject("Couldn't reach the internet. Check your connection, then try again.");
  const appOf = (appId, name) => {
    if (appId) { const f = found.find((x) => x.appId === appId); return f && f._art ? f._art : appId; }
    const f = found.find((x) => x.name === name);
    return f && f._art;
  };
  const NEWS = [
    { title: "Patch 2.3: new co-op mode and 40 fixes", date: iso(2 * d), update: true, summary: "Play the campaign with a friend, plus balance changes across every class.", url: "https://store.steampowered.com/news/1", body: "<p>Patch notes.</p>", image: art(1091500, "hero") },
    { title: "Developer update: the road to 3.0", date: iso(12 * d), update: false, summary: "What we're working on for the next big update.", url: "https://store.steampowered.com/news/2", body: "<p>Roadmap.</p>" },
  ];

  const api = {
    ps_ready: () => ok({}),
    ps_state: () => (STATE === "loading" ? new Promise((r) => setTimeout(() => r(state()), 60)) : ok(state())),
    ps_found: () => ok(STATE === "loading" ? { scanning: true, refreshing: true, games: [] } : { scanning: false, refreshing: false, games: found }),
    ps_cover: (appId, name, key) => (STATE === "offline" ? offline() : ok(art(appOf(appId, name), "cover"))),
    ps_art: (appId, name, key, kind) => (STATE === "offline" ? ok("") : ok(art(appOf(appId, name), kind))),
    ps_gameIcon: () => ok(""),
    ps_gameNews: (key) => { if (STATE === "offline") emit({ type: "news", key, error: "Couldn't reach Steam's news feed.", items: [] }); else if (STATE !== "loading") emit({ type: "news", key, items: NEWS }); return ok(); },
    ps_cloudStatus: () => ok({ state: "overlap", count: 1 }),
    ps_steamSetup: () => ok({ games: [], steamRunning: false }),
    ps_backupInfo: () => ok({
      sync: CLOUD, locked: false, dir: settings.backupDir, target: CLOUD ? settings.backupDir + "\\Lootnest" : "", auto: CLOUD, keep: 5, encrypted: false, computer: "COCONUT",
      status: STATE === "error" ? { error: "The backup folder (C:\\Users\\cris\\OneDrive) isn't there. Is the drive connected, and the cloud app running?", errorAt: iso(60e3), last: iso(3 * d), size: 48e6 } : { last: iso(25 * 60e3), size: 48211000 },
      folders: [{ name: "OneDrive", path: "C:\\Users\\cris\\OneDrive" }, { name: "Google Drive", path: "G:\\My Drive" }],
    }),
    ps_savesStatus: () => { if (STATE !== "loading") emit({ type: "savesStatus", status: STATE === "error" ? { error: "The cloud folder (C:\\Users\\cris\\OneDrive) isn't there.", games: [], players: [] } : { games: [{ gameId: "cs2", playerId: "p1", state: "same", computer: "COCONUT", changed: iso(h) }, { gameId: "cs2", playerId: ids[1] || "p1", state: "conflict", computer: "LAPTOP", changed: iso(d) }, { gameId: "bg3", playerId: "p1", state: "down", computer: "LAPTOP", changed: iso(3 * h) }], players: [] } }); return ok(); },
    ps_gameSaves: (key) => {
      if (STATE === "loading") return ok();
      const f = found.find((x) => x.key === key) || {};
      const info = { key, locations: [], versions: [], source: "" };
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
    // Each player's profile: saved a while ago, a few MB.
    ps_profileInfo: (gid) => { if (STATE !== "loading") emit({ type: "profileInfo", gameId: gid, info: Object.fromEntries((profiles[gid] || []).map((p, j) => [p, { saved: iso((j + 1) * 22 * h), size: (j + 1) * 7.3 * 1048576 }])) }); return ok(); },
    ps_browse: (id) => { emit({ type: "browse", reqId: id, cancelled: true }); return ok(); },
    ps_steamGames: () => ok([{ appId: 1091500, name: "Cyberpunk 2077", installDir: "Cyberpunk 2077" }]),
  };
  // Everything else answers quietly (the harness never changes anything).
  const handler = { get: (t, k) => t[k] || (() => (STATE === "offline" && /check|news|cover/i.test(k) ? offline() : ok({}))) };
  const all = new Proxy(api, handler);
  Object.keys(api).forEach((k) => { window[k] = api[k]; });
  // Functions the UI binds by name but the fixtures don't cover.
  ["ps_play", "ps_load", "ps_steamPick", "ps_savePlayers", "ps_saveSettings", "ps_saveGame", "ps_deleteGame", "ps_addFound", "ps_scan", "ps_open",
    "ps_shortcut", "ps_foundShortcut", "ps_launchFound", "ps_setShelf", "ps_syncNow", "ps_syncResolve", "ps_backupNow", "ps_setBackup", "ps_checkUpdate",
    "ps_setGameSaves", "ps_gameSaveNow", "ps_gameSaveRestore", "ps_restoreOpen", "ps_restoreApply", "ps_restoreCancel", "ps_exportPlayer", "ps_importApply",
    "ps_inspectSteam", "ps_inspectExe", "ps_checkPath", "ps_cancelRequest", "ps_ignoreFound", "ps_foundGame", "ps_addLibraryGame", "ps_saveLibraryPref",
    "ps_setAvatar", "ps_removeAvatar", "ps_setBanner", "ps_removeBanner", "ps_setBackupPassword", "ps_moveInfo", "ps_moveGame", "ps_steamVerify",
    "ps_openGameFolder", "ps_applyHooks", "ps_fixBlocked", "ps_copy", "ps_installUpdate", "ps_quit"].forEach((k) => { if (!window[k]) window[k] = all[k]; });
  window.__harness = { players: PLAYERS, found, games };
})();
