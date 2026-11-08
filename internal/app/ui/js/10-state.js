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
    sideQ: "",        // the filter over the sidebar's games
    revealed: new Set(), // hidden achievements shown on request (key|id)
    sideSort: pref("sideSort", "name"), // name, recent or played
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
    find: { q: "", hits: null, error: "" }, // Store: the search and what came back
    front: null,      // Store rows before a search: {daily, specials, top, deals} or {error}
    gq: { q: "", hits: null, sel: -1 }, // the search in the sidebar
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
    covers: {},        // cover key -> data: URL, or "" when there is none (yet)
    icons: {},         // cover key -> the game exe's icon as a data: URL, or "" (shown when there's no cover)
    art: {},           // "hero:<appid>" / "logo:<appid>" -> Steam's wide art and logo, or ""
    custom: {},        // "<Library key>:<file>" -> art the user chose (Manage, Cover and art), or ""
    news: {},          // cover key -> {items, newer, none, error} from Steam's news feed, or {loading: true}
    details: {},       // Library key -> {details, edited, fetching, error} from ps_gameDetails, or {loading: true}
    detOpen: new Set(), // Library keys whose summary is shown in full
    page: null,        // the game page: {kind: "game"|"found"|"owned", id, cloud}
    sort: pref("sort", "name"),       // Library order: name | recent | played
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
    adding: false,
    hooksBusy: false,
    moving: null,     // {appId, name, step, done, total} while Game Mover copies a game
  };
  const browseWaiters = {};
  let seq = 0;

  const $ = (sel, root = document) => root.querySelector(sel);
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
    dots: '<svg viewBox="0 0 16 16"><circle cx="3" cy="8" r="1.4" fill="currentColor"/><circle cx="8" cy="8" r="1.4" fill="currentColor"/><circle cx="13" cy="8" r="1.4" fill="currentColor"/></svg>',
    folder: '<svg viewBox="0 0 16 16"><path d="M1.75 3.5h4l1.5 1.5h7v7.25a.75.75 0 0 1-.75.75H2.5a.75.75 0 0 1-.75-.75z" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linejoin="round"/></svg>',
    x: '<svg viewBox="0 0 16 16"><path d="M4 4l8 8M12 4l-8 8" stroke="currentColor" stroke-width="1.4" stroke-linecap="round"/></svg>',
    back: '<svg viewBox="0 0 16 16"><path d="M10 3 5 8l5 5" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>',
    plus: '<svg viewBox="0 0 16 16"><path d="M8 3v10M3 8h10" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>',
    play: '<svg viewBox="0 0 16 16" aria-hidden="true"><path d="M5 3.2v9.6a.6.6 0 0 0 .92.5l7.3-4.8a.6.6 0 0 0 0-1L5.92 2.7A.6.6 0 0 0 5 3.2z" fill="currentColor"/></svg>',
  };

  // --------------------------------------------------------------- toasts --
  // act: { label, run } adds a button, such as Undo.
  function toast(text, kind, act) {
    const el = document.createElement("div");
    el.className = "ln-toast ln-root" + (kind === "error" ? " ln-toast--error" : " ln-toast--success");
    el.setAttribute("role", kind === "error" ? "alert" : "status");
    el.innerHTML = kind === "error" ? GI.alert : '<svg class="ln-icon" viewBox="0 0 20 20" aria-hidden="true"><circle cx="10" cy="10" r="6.75"/><path d="M7.25 10.25 9.1 12.1l3.65-3.85"/></svg>';
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

