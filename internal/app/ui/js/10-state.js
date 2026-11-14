  // ---------------------------------------------------------------- state --
  const S = {
    pinfo: {},        // game id -> player id -> {saved, size} of their profile
    ach: {},          // Library key -> achievements read from this PC (null: none)
    doctor: {},       // Library key -> Achievement Doctor's check (achDoctor event)
    sources: null,    // what the last scan saw of each source (the Sources drawer)
    points: {},       // Library key -> its save points (savePoints event)
    timeline: {},     // Library key -> its timeline, newest first (timeline event)
    tlMore: {},       // Library key -> how many timeline entries show
    captures: {},     // Library key -> other apps' captures of it: {items, counts, gameBarOff} (captures event)
    unsorted: null,   // captures matched to no game, and the folders read: {items, total, folders}
    gameTab: {},      // "<kind>:<id>" -> the game page's tab open (Overview when none)
    facts: {},        // Library key -> what its folder says (anti-cheat), from ps_gameFacts
    mods: {},         // Library key -> its mods and what's in its folder (ps_mods), or null while asked
    modsRun: {},      // Library key -> a change to its mods under way: {what, done, total}
    modsFor: {},      // Library key -> whose set its Mods tab shows: a profile id, "" for the game's
    sideQ: "",        // the filter over the sidebar's games
    revealed: new Set(), // hidden achievements shown on request (key|id)
    sideSort: pref("sideSort", "name"), // name, recent or played
    sideShow: pref("sideShow", "all"), // the sidebar's games: all, installed, fav, owned or "p:<profile id>"
    sideSel: new Set(), // Library keys of the sidebar's selected rows (Ctrl and Shift+click)
    sideAnchor: "",   // the row a Shift+click selects from
    sideNew: null,    // the inline New collection field: {keys} of the games it gets
    sideRename: "",   // the collection being renamed in place
    tagPick: [],      // Library keys the Add to collection menu is for
    libSel: new Set(), // Library keys of the games picked in the Library's grid or list
    libAnchor: "",    // the game a Shift+click or Shift+arrow picks from
    sideClosed: (() => { try { return new Set(JSON.parse(pref("sideClosed", "[]"))); } catch (e) { return new Set(); } })(),
    folded: pref("folded", "") === "1", // the sidebar folded to a rail
    achWho: {},       // Library key -> whose achievements are shown
    setTab: pref("setTab", "general"), // the Settings category open
    allAch: null,     // the Achievements page: every game's, null until read
    achPage: "",      // whose the Achievements page shows
    achMs: false,     // the Achievements page's milestones are open
    statusWho: {},    // Library key -> whose status the game page shows
    hoursEdit: null,  // {key, player}: whose hours the game page is correcting
    tag: "",          // the Library's collection filter ("" for all)
    status: "",       // the Library's status filter ("" for any)
    gsaves: {},       // Library key -> cloud saves of a game outside Steam
    svo: null,        // Saves: {recent, games} from ps_savesOverview
    svMore: false,    // Saves lists every recent copy, not the first few
    svOpenRow: "",    // the copy on Saves whose files are open, "key|kind|id"
    svFiles: {},      // each opened copy's files: { files, count } or { loading } or { error }
    svNaming: "",     // the copy being named
    svSort: "last",   // Saves: the Per game column sorted by
    svCloud: false,   // Saves opened for its Cloud saves section
    svBusy: "",       // Saves: the copy being put back from Recent copies
    find: { q: "", hits: null, error: "" }, // Store: the search and what came back
    front: null,      // Store rows before a search: {daily, specials, top, deals} or {error}
    gq: { q: "", hits: null, sel: -1 }, // the search in the top bar
    updates: null,    // the Updates page: {games, newsOn} from ps_updates
    copies: null,     // "Is this a new copy?" offers from ps_newCopies
    dl: null,         // updates found in Downloads, from ps_downloads
    carried: {},      // old key -> the result of carrying it over (for the page)
    offers: {},       // Steam app id -> {offer, pc, owned} or {error}, for Find games
    d: null,          // AppState from Go
    view: "home",
    from: "home",      // the view a game's page goes back to
    selected: null,    // {kind, id} of the game last opened: the Library's colours follow it
    q: "",             // library search
    filter: "all",     // library: "all", "profiles" or "owned" (not installed)
    store: "all",      // library: a store or launcher, or "all"
    who: "",           // library: only the games this player has a profile in or played last
    covers: artStore(), // cover key -> a blob: URL, or "" when there is none (yet)
    icons: artStore(),  // cover key -> the game exe's icon, or "" (shown when there's no cover)
    art: artStore(),    // "hero:<appid>" / "logo:<appid>" -> Steam's wide art and logo, or ""
    custom: {},        // "<Library key>:<file>" -> art the user chose (Manage, Cover and art), or ""
    news: {},          // cover key -> {items, newer, none, error} from Steam's news feed, or {loading: true}
    details: {},       // Library key -> {details, edited, fetching, error} from ps_gameDetails, or {loading: true}
    detOpen: new Set(), // Library keys whose summary is shown in full
    reviews: {},       // "key|kind" -> {loading} or the "reviews" event from ps_gameReviews
    revKind: {},       // Library key -> "all", "positive" or "negative"
    revOpen: new Set(), // review ids shown in full
    page: null,        // the game page: {kind: "game"|"found"|"owned", id, cloud}
    sort: pref("sort", "name"),       // Library order: name | recent | played | rating | size
    density: pref("density", "m"),    // cover size: s | m | l
    ctx: null,         // the right-click menu: {x, y, kind, id}
    update: {},        // {available, version, page, upToDate, checking, error, step}
    animateView: true,  // fade the page in on the next render
    menuFor: null,
    picker: null,     // {gameId, mode, ask, playerId, owner, newMode, working}
    dialog: null,     // {kind, ...}
    edit: null,       // editor state
    players: null,    // players being edited
    logText: "",
    hooks: null,        // Steam launch option status per game
    hideHookBanner: false,
    found: { scanning: true, games: [] },
    owned: null,      // games you have in a store, not installed (owned event); null until asked
    itch: { busy: false, refreshing: false, error: "", recs: {}, updates: [], run: null, checking: false, asked: false, seen: new Set() }, // itch.io, 73-itch.js
    gog: { busy: false, refreshing: false, error: "", recs: {}, updates: [], run: null, checking: false, asked: false, seen: new Set() }, // GOG, 74-gog.js
    adding: false,
    hooksBusy: false,
    moving: null,     // {appId, name, step, done, total} while Game Mover copies a game
    folderMove: null, // {key, done, total, step} while a folder game moves to another drive
    sizes: null,      // ps_sizes: {games: [{key, install, saves, profiles, copies}], drives, counting, safe}
    stSort: pref("stSort", "install"), // the Storage list's order: name or a size
  };
  const browseWaiters = {};
  let seq = 0;

  const $ = (sel, root = document) => root.querySelector(sel);
  // Pictures come from Go as data: URLs, many of them 50 KB of text or more.
  // Kept as blob: URLs instead, the HTML that shows them stays short, so a
  // Library of hundreds of games draws without parsing megabytes of text.
  function artStore() {
    return new Proxy({}, { set(t, k, v) { t[k] = typeof v === "string" && v.length > 4096 && /^data:[^,]*;base64,/.test(v.slice(0, 64)) ? safeBlob(v) : v; return true; } });
  }
  function safeBlob(d) { try { return blobURL(d); } catch (e) { return d; } }
  // Names in the reader's order, case and accents aside. One collator for
  // every sort: localeCompare with options builds one per comparison.
  const byName = new Intl.Collator(undefined, { sensitivity: "base", numeric: true }).compare;
  const esc = (s) => String(s == null ? "" : s).replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
  const errText = (e) => (typeof e === "string" ? e : (e && e.message) || "Something went wrong.");
  const cfg = () => S.d.config;
  const player = (id) => cfg().players.find((p) => p.id === id);
  const game = (id) => cfg().games.find((g) => g.id === id);
  const color = (id) => (player(id) || {}).color || "var(--dim)";
  const pname = (id) => (player(id) || {}).name || "Someone";
  // Runs: a profile can be a run of a person ("Cristian hardcore" of
  // Cristian). It shows under them, with their colour and picture.
  const MAX_PROFILES = 16;
  const parentOf = (p) => (p && p.of && p.of !== p.id && player(p.of)) || null;
  // Just you: one person on this PC (with or without runs of their own).
  // The words about who's at the PC and playing as someone else go away.
  const people = () => cfg().players.filter((p) => !parentOf(p));
  const solo = () => people().length <= 1;
  const personOf = (p) => parentOf(p) || p;
  // "hardcore" for "Cristian hardcore" of Cristian.
  const runLabel = (p) => {
    const par = parentOf(p);
    if (!par) return "";
    const n = p.name.trim();
    return n.toLowerCase().startsWith(par.name.toLowerCase() + " ") ? n.slice(par.name.length).trim() : n;
  };
  // People in order, each followed by their runs.
  function grouped(list) {
    list = list || cfg().players;
    const ids = new Set(list.map((p) => p.id));
    const isRun = (p) => p.of && p.of !== p.id && ids.has(p.of);
    const out = [];
    list.filter((p) => !isRun(p)).forEach((p) => { out.push(p); list.filter((r) => isRun(r) && r.of === p.id).forEach((r) => out.push(r)); });
    list.forEach((p) => { if (!out.includes(p)) out.push(p); });
    return out;
  }
  const session = (gid) => (S.d.sessions || []).find((s) => s.gameId === gid);
  const hasProfile = (gid, pid) => ((S.d.profiles || {})[gid] || []).includes(pid);
  const possessive = (n) => (/s$/i.test(n) ? n + "’" : n + "’s");
  const fmtSize = (b) => (b < 1024 ? b + " B" : b < 1048576 ? (b / 1024).toFixed(0) + " KB" : b < 1073741824 ? (b / 1048576).toFixed(1) + " MB" : (b / 1073741824).toFixed(2) + " GB");
  const short = (name) => (name.length > 22 ? name.slice(0, 20).trim() + "…" : name);

  const icon = {
    dots: ph("dots-three"),
    folder: ph("folder"),
    x: ph("x"),
    back: ph("caret-left"),
    plus: ph("plus"),
    play: ph("play", "fill"),
  };

  // --------------------------------------------------------------- toasts --
  // act: { label, run } adds a button, such as Undo.
  function toast(text, kind, act) {
    const el = document.createElement("div");
    el.className = "ln-toast ln-root" + (kind === "error" ? " ln-toast--error" : " ln-toast--success");
    el.setAttribute("role", kind === "error" ? "alert" : "status");
    el.innerHTML = kind === "error" ? GI.alert : ph("check-circle");
    const p = document.createElement("p");
    p.textContent = text;
    el.appendChild(p);
    if (act) {
      const b = document.createElement("button");
      b.className = "ln-button ln-button--ghost ln-button--sm";
      b.textContent = act.label;
      b.addEventListener("click", () => { el.remove(); act.run(); });
      el.appendChild(b);
    }
    $("#toasts").appendChild(el);
    // The toast stays while the pointer or the keyboard is on it.
    let timer = 0;
    const arm = () => { clearTimeout(timer); timer = setTimeout(() => el.remove(), kind === "error" ? 7000 : act ? 8000 : 3800); };
    const hold = () => clearTimeout(timer);
    const release = () => setTimeout(() => { if (!el.matches(":hover, :focus-within")) arm(); });
    el.addEventListener("pointerenter", hold);
    el.addEventListener("focusin", hold);
    el.addEventListener("pointerleave", release);
    el.addEventListener("focusout", release);
    arm();
  }

