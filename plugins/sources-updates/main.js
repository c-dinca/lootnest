// Sources and updates, a Lootnest script plugin (docs/PLUGINS.md).
//
// The user adds sources: addresses of JSON files that list downloads. This
// plugin reads them, never trusts them, and keeps the last good copy of
// each. It comes with no source, suggests none and looks nowhere the user
// didn't point it to.
//
// Parts: limits, small helpers, titles and versions, feed formats
// (adapters), sources and their refresh, matching and detection, the
// Sources page.
(function () {
  "use strict";

  // ---------------------------------------------------------------- limits --
  const LIMITS = {
    feedBytes: 64 << 20, // the most a feed may weigh
    feedSeconds: 120,
    entries: 200000,
    title: 300,
    uris: 20,
    uri: 4096,
    sources: 50,
    name: 60,
    url: 2000,
    version: 40,
    changelog: 2000,
  };
  const HOUR = 3600 * 1000;
  const INTERVALS = [1, 3, 6, 12, 24, 0]; // hours; 0 is only when asked

  // --------------------------------------------------------------- helpers --
  const isObj = (v) => v !== null && typeof v === "object" && !Array.isArray(v);
  // Text from outside: one line, no control characters, at most max long.
  function clean(v, max) {
    if (typeof v !== "string") return "";
    const s = v.replace(/[\u0000-\u001f\u007f-\u009f\u200b-\u200f\u2028-\u202e\u2066-\u2069\ufeff]/g, " ").replace(/\s+/g, " ").trim();
    return s.length > max ? s.slice(0, max).trim() : s;
  }
  // 12345 as "12,345" (the engine has no locales).
  const num = (n) => String(Math.round(n)).replace(/\B(?=(\d{3})+$)/g, ",");
  const plural = (n, one, many) => `${num(n)} ${n === 1 ? one : many || one + "s"}`;
  function ago(ms, now) {
    const m = Math.max(0, Math.round((now - ms) / 60000));
    if (m < 1) return "just now";
    if (m < 60) return `${m} min ago`;
    const h = Math.round(m / 60);
    if (h < 48) return `${h} h ago`;
    return `${Math.round(h / 24)} days ago`;
  }
  // "45.2 GB" or a number of bytes, as bytes; 0 when it can't be read.
  function sizeBytes(v) {
    if (typeof v === "number") return Number.isFinite(v) && v > 0 && v < 1e15 ? Math.round(v) : 0;
    if (typeof v !== "string") return 0;
    const m = /^\s*(\d{1,7}(?:[.,]\d{1,3})?)\s*(b|bytes|kb|kib|mb|mib|gb|gib|tb|tib)\s*$/i.exec(v);
    if (!m) return 0;
    const unit = { b: 1, bytes: 1, kb: 1024, kib: 1024, mb: 1048576, mib: 1048576, gb: 1073741824, gib: 1073741824, tb: 1099511627776, tib: 1099511627776 }[m[2].toLowerCase()];
    return Math.round(parseFloat(m[1].replace(",", ".")) * unit);
  }
  function sizeText(n) {
    if (!n) return "";
    if (n >= 1073741824) return `${(n / 1073741824).toFixed(n >= 10737418240 ? 0 : 1)} GB`;
    if (n >= 1048576) return `${Math.round(n / 1048576)} MB`;
    return `${Math.max(1, Math.round(n / 1024))} KB`;
  }
  // A date from a feed as Unix milliseconds, or 0: it must parse and be
  // between 1990 and two days from now.
  function dateMs(v, now) {
    if (typeof v !== "string" || v.length > 40) return 0;
    const t = Date.parse(v);
    return Number.isFinite(t) && t > 631152000000 && t < now + 2 * 24 * HOUR ? t : 0;
  }

  // An address from a feed or the user: {scheme, host} when it's one of the
  // kinds allowed, else null. No spaces, no user or password part.
  const MAGNET = /^magnet:\?(?:[^\s]*&)?xt=urn:bt(?:ih:(?:[0-9a-f]{40}|[a-z2-7]{32})|mh:[0-9a-f]{68})(?:&[^\s]*)?$/i;
  const WEB = /^(https?):\/\/((?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,24}|\d{1,3}(?:\.\d{1,3}){3})(?::\d{1,5})?(?:[/?#][^\s]*)?$/i;
  function readURI(v, max) {
    if (typeof v !== "string" || v.length > max || /[\u0000- \u007f]/.test(v)) return null;
    if (MAGNET.test(v)) return { scheme: "magnet", host: "" };
    const m = WEB.exec(v);
    return m ? { scheme: m[1].toLowerCase(), host: m[2].toLowerCase() } : null;
  }
  // The info hash of a magnet link, lowercase, or "".
  function infoHash(uri) {
    const m = /xt=urn:btih:([0-9a-f]{40}|[a-z2-7]{32})/i.exec(uri || "");
    return m ? m[1].toLowerCase() : "";
  }

  // ---------------------------------------------------------------- titles --
  // normTitle gives a title's words for matching: lowercase, without
  // bracketed tags, what follows a "+", versions, builds, dates, edition and
  // release words or punctuation, and with roman numerals as digits.
  // "Starfall Tactics II: Gold Edition v1.2.3 + 5 DLCs [MULTi12]" and
  // "starfall tactics 2" both give ["starfall", "tactics", "2"].
  const NORM = 1; // raised when normTitle changes, so kept feeds are read again
  const ROMAN = { i: 1, ii: 2, iii: 3, iv: 4, v: 5, vi: 6, vii: 7, viii: 8, ix: 9, x: 10, xi: 11, xii: 12, xiii: 13, xiv: 14, xv: 15, xvi: 16, xvii: 17, xviii: 18, xix: 19, xx: 20 };
  const DROP = new Set(["the", "a", "an", "of", "and", "edition", "deluxe", "ultimate", "complete", "goty", "gold", "premium", "digital", "collectors", "collector", "standard",
    "definitive", "enhanced", "anniversary", "bundle", "repack", "dlc", "dlcs", "bonus", "update", "updates", "patch", "hotfix", "incl", "including", "ost", "soundtrack",
    "build", "version", "pc", "windows", "win64", "x64", "portable"]);
  function normTitle(title) {
    let s = String(title || "").toLowerCase();
    try { s = s.normalize("NFKD").replace(/[\u0300-\u036f]/g, ""); } catch (e) { /* as it is */ }
    s = s.replace(/[\[({][^\])}]*[\])}]/g, " ")
      .replace(/\s\+.*$/, " ")
      .replace(/\bgame of the year\b/g, " ")
      .replace(/\b(?:20\d{2}[.\-/_]\d{1,2}[.\-/_]\d{1,2}|\d{1,2}[.\-/_]\d{1,2}[.\-/_]20\d{2})\b/g, " ")
      .replace(/(^|[^a-z0-9])v?\.?\d+(?:\.\d+)+[a-z]?\s*(?:to|-+>|\u2192)\s*v?\.?\d+(?:\.\d+)+[a-z]?(?![a-z0-9])/g, "$1 ")
      .replace(/(^|[^a-z0-9])v\.?\s?\d+(?:\.\d+)*[a-z]?(?![a-z0-9])/g, "$1 ")
      .replace(/\bbuild[\s._#-]*\d+\b/g, " ").replace(/\bb\d{5,}\b/g, " ")
      .replace(/\b\d+(?:\.\d+){2,}[a-z]?\b/g, " ")
      .replace(/\bmulti\s?\d+\b/g, " ")
      .replace(/&/g, " and ").replace(/['\u2019`]/g, "").replace(/[^a-z0-9]+/g, " ");
    const out = [];
    for (const w of s.split(" ")) {
      if (!w || DROP.has(w)) continue;
      out.push(ROMAN[w] ? String(ROMAN[w]) : w);
    }
    return out;
  }
  // How alike two titles' words are, 0 to 1: the share of words they have in
  // common. Titles whose numbers differ ("2" against "3") are never alike.
  function titleScore(a, b) {
    if (!a.length || !b.length) return 0;
    const sa = new Set(a), sb = new Set(b);
    let both = 0;
    for (const t of sa) if (sb.has(t)) both++;
    const nums = (set) => [...set].filter((t) => /^\d+$/.test(t)).sort().join(",");
    const score = both / (sa.size + sb.size - both);
    return nums(sa) === nums(sb) ? score : Math.min(score, 0.3);
  }
  const AUTO = 0.9, ASK = 0.6; // link by itself; ask the user; under that, nothing

  // -------------------------------------------------------------- versions --
  // cmpSem orders "1.0.10" after "1.0.9" and "1.2b" after "1.2a".
  function cmpSem(a, b) {
    const pa = String(a).toLowerCase().split("."), pb = String(b).toLowerCase().split(".");
    for (let i = 0; i < Math.max(pa.length, pb.length); i++) {
      const x = /^(\d*)(.*)$/.exec(pa[i] || "0"), y = /^(\d*)(.*)$/.exec(pb[i] || "0");
      const d = (+x[1] || 0) - (+y[1] || 0);
      if (d) return d < 0 ? -1 : 1;
      if (x[2] !== y[2]) return x[2] < y[2] ? -1 : 1;
    }
    return 0;
  }
  // titleVersion reads what a title says about its version, by a chain:
  // "v1.0 to v1.1" (a patch: from and sem), "v1.2.3" or a bare "1.2.3",
  // "Build 12345", and a date. {} when it says nothing.
  function titleVersion(title) {
    let t = String(title || "");
    const out = {};
    let m = /(?:^|[^a-z0-9.])(20\d{2})[.\-/_](\d{1,2})[.\-/_](\d{1,2})(?![\d.])/i.exec(t);
    let ymd = m ? [+m[1], +m[2], +m[3]] : null;
    if (!m && (m = /(?:^|[^a-z0-9.])(\d{1,2})[.\-/_](\d{1,2})[.\-/_](20\d{2})(?![\d.])/i.exec(t))) ymd = [+m[3], +m[2], +m[1]];
    if (ymd && ymd[1] >= 1 && ymd[1] <= 12 && ymd[2] >= 1 && ymd[2] <= 31) {
      out.date = Date.UTC(ymd[0], ymd[1] - 1, ymd[2]);
      t = t.replace(m[0], " ");
    }
    m = /(?:^|[^a-z0-9.])v?\.?(\d+(?:\.\d+)+[a-z]?)\s*(?:to|-+>|\u2192)\s*v?\.?(\d+(?:\.\d+)+[a-z]?)(?![\d.])/i.exec(t);
    if (m && cmpSem(m[1], m[2]) < 0) { out.from = m[1].toLowerCase(); out.sem = m[2].toLowerCase(); }
    if (!out.sem) {
      m = /(?:^|[^a-z0-9])v\.?\s?(\d+(?:\.\d+){1,4}[a-z]?)(?![\d.a-z])/i.exec(t) || /(?:^|[^a-z0-9.])(\d+\.\d+\.\d+(?:\.\d+)?[a-z]?)(?![\d.a-z])/i.exec(t);
      if (m) out.sem = m[1].toLowerCase();
    }
    m = /\bbuild[\s._#-]*(\d{3,})(?!\d)/i.exec(t) || /(?:^|[^a-z0-9])b(\d{5,})(?![\da-z])/i.exec(t);
    if (m) out.build = +m[1];
    return out;
  }
  // -1, 0 or 1 when two versions are of a kind that compares (both have a
  // version number, else both a build, else both a date in the title), and
  // null when they don't.
  function compareVer(a, b) {
    if (a.sem && b.sem) return cmpSem(a.sem, b.sem);
    if (a.build && b.build) return Math.sign(a.build - b.build);
    if (a.date && b.date) return Math.sign(a.date - b.date);
    return null;
  }
  const verText = (v) => (v.sem ? v.sem : v.build ? "Build " + v.build : v.date ? new Date(v.date).toISOString().slice(0, 10) : "");
  // The version a feed's entry carries: its own field in the richer format,
  // else what its title says.
  function entryVersion(e) {
    const v = titleVersion(e.t);
    if (e.v) {
      if (/^\d+(?:\.\d+)*[a-z]?$/i.test(e.v)) v.sem = e.v.toLowerCase();
      else Object.assign(v, titleVersion("v" + e.v), titleVersion(e.v));
    }
    return v;
  }

  // --------------------------------------------------------------- formats --
  // An adapter reads one feed format: sniff(doc) says whether the document
  // is its kind, parse(doc, now) gives {name, entries, skipped, http}.
  // An entry: {t: title, k: its words for matching, u: [addresses], d: date
  // in ms or 0, s: bytes or 0}
  // and, where the format has them, a: Steam app id, v: version, h: SHA-256,
  // c: changelog. Unknown fields are ignored; nothing in a feed is run,
  // opened or used as a path.
  function readEntry(raw, now, rich, out) {
    if (!isObj(raw)) return false;
    const t = clean(raw.title, LIMITS.title);
    if (!t || !Array.isArray(raw.uris)) return false;
    const u = [];
    for (let i = 0; i < raw.uris.length && i < 200 && u.length < LIMITS.uris; i++) {
      const r = readURI(raw.uris[i], LIMITS.uri);
      if (!r) continue;
      if (r.scheme === "http") out.http++;
      u.push(raw.uris[i]);
    }
    if (!u.length) return false;
    const e = { t, k: normTitle(t).join(" "), u, d: dateMs(raw.uploadDate, now), s: sizeBytes(raw.fileSize) };
    if (rich) {
      if (Number.isInteger(raw.steamAppId) && raw.steamAppId > 0 && raw.steamAppId < 2147483647) e.a = raw.steamAppId;
      const v = clean(raw.version, LIMITS.version);
      if (v) e.v = v;
      if (typeof raw.sha256 === "string" && /^[0-9a-f]{64}$/i.test(raw.sha256)) e.h = raw.sha256.toLowerCase();
      const c = clean(raw.changelog, LIMITS.changelog);
      if (c) e.c = c;
    }
    out.entries.push(e);
    return true;
  }
  function readDownloads(doc, now, rich) {
    if (!Array.isArray(doc.downloads)) throw new Error("it has no list of downloads");
    if (doc.downloads.length > LIMITS.entries) throw new Error(`it lists more than ${num(LIMITS.entries)} downloads`);
    const out = { name: clean(doc.name, LIMITS.name), entries: [], skipped: 0, http: 0 };
    for (const raw of doc.downloads) if (!readEntry(raw, now, rich, out)) out.skipped++;
    if (!out.entries.length) throw new Error(doc.downloads.length ? "none of its downloads could be read" : "it lists no downloads");
    return out;
  }
  const ADAPTERS = [
    // Lootnest's own, richer format: the Hydra one plus "lootnest": 1 at
    // the top and, per download, steamAppId, version, changelog, sha256.
    { id: "lootnest", name: "Lootnest", sniff: (doc) => doc.lootnest === 1, parse: (doc, now) => readDownloads(doc, now, true) },
    // The format Hydra Launcher's sources use: {name, downloads: [{title,
    // uris, uploadDate, fileSize}]}.
    { id: "hydra", name: "Hydra-compatible", sniff: (doc) => Array.isArray(doc.downloads), parse: (doc, now) => readDownloads(doc, now, false) },
  ];
  // parseFeed reads a feed's text. It throws, with words the user can
  // read, when the feed can't be used.
  function parseFeed(text, now) {
    let doc;
    try { doc = JSON.parse(text); } catch (e) { throw new Error("it isn't JSON"); }
    if (!isObj(doc)) throw new Error("it isn't a list of downloads");
    const ad = ADAPTERS.find((a) => a.sniff(doc));
    if (!ad) throw new Error("it isn't in a format this plugin reads");
    const out = ad.parse(doc, now);
    out.format = ad.id;
    return out;
  }

  // --------------------------------------------------------------- sources --
  // "sources" in storage: {notice, hours, list: [{id, name, url, on, format,
  // added, checked, changed, count, skipped, http, etag, modified, error}]}.
  // "feed-<id>": {entries} of the last good copy.
  let ST = null;
  function state() {
    if (ST) return ST;
    const s = lootnest.storage.get("sources");
    ST = isObj(s) ? s : {};
    if (!Array.isArray(ST.list)) ST.list = [];
    if (!INTERVALS.includes(ST.hours)) ST.hours = 6;
    return ST;
  }
  const save = () => lootnest.storage.set("sources", state());
  const sourceBy = (id) => state().list.find((s) => s.id === id);
  // What a source is called outside its own row on the Sources page: its
  // name, else its site. Never its whole address, which may hold a key in
  // its path or query and must not reach a log, a notice or another page.
  const srcSite = (url) => { const m = /^https?:\/\/([^\/?#:@]+)(?::\d+)?(?:[\/?#]|$)/i.exec(String(url || "")); return m ? m[1].toLowerCase() : "a source"; };
  const srcLabel = (src) => src.name || srcSite(src.url);
  // A source's kept entries, or null. A copy kept by an older normTitle gets
  // its words again.
  function feedOf(id) {
    const f = lootnest.storage.get("feed-" + id);
    if (!isObj(f) || !Array.isArray(f.entries)) return null;
    if (f.norm !== NORM) {
      for (const e of f.entries) e.k = normTitle(e.t).join(" ");
      lootnest.storage.set("feed-" + id, { norm: NORM, entries: f.entries });
    }
    return f.entries;
  }
  const hasFeed = (id) => lootnest.storage.get("feed-" + id) !== null;

  // The name and address the user typed, checked. Throws what's wrong.
  function readSource(values, except) {
    const name = clean(values && values.name, LIMITS.name), url = typeof (values && values.url) === "string" ? values.url.trim() : "";
    const r = readURI(url, LIMITS.url);
    if (!r || r.scheme === "magnet") throw new Error("Type the address of a JSON file, starting with https://");
    if (state().list.some((s) => s.id !== except && s.url === url)) throw new Error("That source is already on the list.");
    return { name, url, plain: r.scheme === "http" };
  }

  // refreshSource asks for one source's feed and keeps it when it checks
  // out. A failure keeps the last good copy and says why.
  function refreshSource(src) {
    const now = Date.now(), headers = { Accept: "application/json" };
    const cached = hasFeed(src.id);
    if (cached && src.etag) headers["If-None-Match"] = src.etag;
    if (cached && src.modified) headers["If-Modified-Since"] = src.modified;
    src.checked = now;
    let r;
    try {
      r = lootnest.net.fetch(src.url, { headers, maxBytes: LIMITS.feedBytes, timeout: LIMITS.feedSeconds });
    } catch (e) {
      src.error = clean(String(e && e.message || e), 200);
      return false;
    }
    if (r.status === 304 && cached) { src.error = ""; return false; }
    if (r.status !== 200) { src.error = `Its site answered ${r.status}`; return false; }
    let feed;
    try { feed = parseFeed(r.text, now); } catch (e) {
      src.error = "Not used: " + clean(String(e && e.message || e), 200);
      return false;
    }
    lootnest.storage.set("feed-" + src.id, { norm: NORM, entries: feed.entries });
    Object.assign(src, { error: "", format: feed.format, count: feed.entries.length, skipped: feed.skipped, http: feed.http, changed: now,
      etag: clean(r.headers.etag, 200), modified: clean(r.headers["last-modified"], 60) });
    if (!src.name) src.name = feed.name || srcSite(src.url);
    return true;
  }

  // Which source is due: on, and not asked within the interval. force is a
  // set of ids asked for by hand.
  const forced = new Set();
  function nextDue(now) {
    const s = state();
    let best = null;
    for (const src of s.list) {
      if (!src.on) continue;
      const due = forced.has(src.id) ? 0 : !s.hours ? (src.checked ? Infinity : 0) : (src.checked || 0) + s.hours * HOUR;
      if (due <= now && (!best || due < best.due)) best = { src, due };
    }
    return best && best.src;
  }
  // The "refresh" handler: one source per call, so the page's calls never
  // wait behind a long round, then the next call is scheduled.
  function refreshTick() {
    const now = Date.now(), src = nextDue(now);
    try {
      if (src) {
        forced.delete(src.id);
        const got = refreshSource(src);
        save();
        lootnest.log(`${srcLabel(src)}: ${src.error || (got ? plural(src.count, "download") : "no change")}`);
        if (got) feedsChanged();
        lootnest.ui.changed();
      }
    } finally {
      planRefresh(); // whatever happened, the next round is planned
    }
  }
  function planRefresh() {
    const s = state(), now = Date.now();
    if (nextDue(now)) return lootnest.schedule("refresh", 200);
    if (!s.hours) return lootnest.schedule("refresh", -1);
    let at = Infinity;
    for (const src of s.list) if (src.on) at = Math.min(at, (src.checked || 0) + s.hours * HOUR);
    lootnest.schedule("refresh", at === Infinity ? -1 : Math.max(1000, at - now));
  }
  // Called when a source's entries changed: look for updates again soon.
  const feedsChanged = () => lootnest.schedule("detect", 300);

  // ------------------------------------------------------------- detection --
  // "links" in storage, by Library key: {seen: when this plugin first saw
  // the game, base: {title, date, from} the installed version as far as it's
  // known, manual: [words] entries linked by hand, yes: [words] and no:
  // [words] answers to "is this the same game?", skip: [entry ids] ignored
  // versions, mute: [source ids] ignored sources}.
  let LK = null;
  function links() {
    if (!LK) { const l = lootnest.storage.get("links"); LK = isObj(l) ? l : {}; }
    return LK;
  }
  const linkOf = (key) => { const L = links(); if (!isObj(L[key])) L[key] = {}; return L[key]; };
  const saveLinks = () => lootnest.storage.set("links", links());
  // Games no store keeps up to date: the ones an update can go into.
  const UPDATABLE = ["folder", "added", "shortcut", "hydra", ""];
  const updatable = (g) => UPDATABLE.includes(g.source || "") && !!g.folder;
  const entryId = (e) => lootnest.hash(e.t + "|" + e.u[0]).slice(0, 16);

  // The installed version of a game: what this PC says (its folder's name
  // or a version file) before the stored baseline; the exe's own version
  // only when it continues the baseline's numbering, since many exes carry
  // their engine's version.
  function installed(g, L) {
    const base = L.base || {}, bv = base.title ? titleVersion(base.title) : {};
    let v = bv, from = base.from || "";
    const local = g.version ? (/^build /.test(g.version) ? { build: +g.version.slice(6) } : { sem: g.version.toLowerCase() }) : null;
    if (local && (local.sem || local.build)) { v = Object.assign({}, bv, local); from = "pc"; }
    else if (g.exeVersion && bv.sem && g.exeVersion.split(".")[0] === bv.sem.split(".")[0] && cmpSem(g.exeVersion, bv.sem) > 0) { v = Object.assign({}, bv, { sem: g.exeVersion }); from = "exe"; }
    return { v, date: base.date || L.seen || 0, from };
  }
  // Whether an entry is newer than what's installed: by version when the
  // two compare, else by the entry's upload date against the baseline's.
  // A patch ("v1.0 to v1.1") counts only for the version it starts from.
  function isNewer(ev, e, inst) {
    if (ev.from) return !!inst.v.sem && cmpSem(inst.v.sem, ev.from) === 0;
    const c = compareVer(ev, inst.v);
    if (c !== null) return c > 0;
    return !!e.d && !!inst.date && e.d > inst.date + HOUR;
  }
  // Of two entries, the later one: by version when they compare, else date.
  function later(a, b) {
    const c = compareVer(a.ver, b.ver);
    return (c !== null ? c : Math.sign((a.e.d || 0) - (b.e.d || 0))) >= 0 ? a : b;
  }

  let OFFERS = [];   // {id, game, name, src, e, ver, have}
  let PENDING = [];  // {game, name, k, title, src} matches to ask about
  let KEYS = new Map(); // words -> {title, src: source name} of every entry kept, for linking by hand
  let GAMES = [];

  // detect reads the Library and every source that's on, links games to
  // entries and lists the entries newer than what's installed.
  function detect() {
    const now = Date.now(), L = links();
    GAMES = lootnest.library.games().filter(updatable);
    const srcs = state().list.filter((x) => x.on);
    const wantHash = new Map(); // info hash or address -> game key, for Hydra's records
    for (const g of GAMES) {
      const l = linkOf(g.key);
      if (!l.seen) l.seen = now;
      const o = g.origin;
      if (o && o.uri) wantHash.set(infoHash(o.uri) || o.uri, g.key);
    }
    const byKey = new Map(), byToken = new Map(), byApp = new Map(), origin = new Map();
    KEYS = new Map();
    for (const src of srcs) {
      const entries = feedOf(src.id) || [];
      for (const e of entries) {
        const hit = { src, e };
        if (e.a) { if (!byApp.has(e.a)) byApp.set(e.a, []); byApp.get(e.a).push(hit); }
        if (wantHash.size) for (const u of e.u) { const h = infoHash(u) || u; if (wantHash.has(h) && !origin.has(wantHash.get(h))) origin.set(wantHash.get(h), e); }
        if (!e.k) continue;
        let list = byKey.get(e.k);
        if (!list) {
          byKey.set(e.k, list = []);
          KEYS.set(e.k, { title: e.t, src: srcLabel(src) });
          for (const t of new Set(e.k.split(" "))) { if (!byToken.has(t)) byToken.set(t, []); byToken.get(t).push(e.k); }
        }
        list.push(hit);
      }
    }
    OFFERS = []; PENDING = [];
    for (const g of GAMES) {
      const l = linkOf(g.key), o = g.origin;
      // The baseline: what Hydra downloaded, by its address in a source,
      // else by its folder's name; an update applied here or a link made by
      // hand stays.
      if (!l.base || l.base.from === "hydra" || l.base.from === "seen") {
        const oe = origin.get(g.key);
        if (oe) l.base = { title: oe.t, date: oe.d || (o && o.time) || l.seen, from: "hydra" };
        else if (o && (o.folder || o.time)) l.base = { title: o.folder || "", date: o.time || l.seen, from: "hydra" };
        else l.base = { title: "", date: l.seen, from: "seen" };
      }
      if (active()[g.key] && !active()[g.key].kept) continue; // its update is under way
      const inst = installed(g, l), words = normTitle(g.name), mine = words.join(" ");
      const hits = new Map(), asked = new Set(); // words of entries -> how sure; the ones to ask about
      for (const k of l.manual || []) if (byKey.has(k)) hits.set(k, 1);
      if (byKey.has(mine)) hits.set(mine, 1);
      // The words of entries that share this game's rarest words.
      const rare = [...new Set(words)].filter((t) => byToken.has(t)).sort((a, b) => byToken.get(a).length - byToken.get(b).length).slice(0, 2);
      for (const t of rare) for (const k of byToken.get(t)) {
        if (hits.has(k) || asked.has(k)) continue;
        const sc = titleScore(words, k.split(" "));
        if (sc >= AUTO || (sc >= ASK && (l.yes || []).includes(k))) hits.set(k, sc);
        else if (sc >= ASK && !(l.no || []).includes(k) && PENDING.length < 50) {
          const info = KEYS.get(k);
          asked.add(k);
          PENDING.push({ game: g.key, name: g.name, k, title: info.title, src: info.src });
        }
      }
      const cands = [];
      for (const k of hits.keys()) if (!(l.no || []).includes(k) || (l.manual || []).includes(k)) cands.push(...byKey.get(k));
      if (g.steamId && byApp.has(g.steamId)) for (const h of byApp.get(g.steamId)) if (!cands.includes(h)) cands.push(h);
      const best = new Map(); // source id -> its latest newer entry
      for (const h of cands) {
        if ((l.mute || []).includes(h.src.id)) continue;
        const ver = entryVersion(h.e);
        if (!isNewer(ver, h.e, inst)) continue;
        const id = entryId(h.e);
        if ((l.skip || []).includes(id)) continue;
        const c = { id: h.src.id + "." + id, eid: id, game: g.key, name: g.name, src: h.src, e: h.e, ver, have: verText(inst.v) };
        best.set(h.src.id, best.has(h.src.id) ? later(best.get(h.src.id), c) : c);
      }
      for (const c of best.values()) OFFERS.push(c);
    }
    // Links of games that left the Library are kept: they may come back.
    saveLinks();
    publish(true);
    lootnest.ui.changed();
  }
  const OFFER_ACTIONS = [{ id: "update", label: "Update" }, { id: "ignore-version", label: "Ignore this version" }, { id: "ignore-source", label: "Ignore this source for this game" }];
  const MAGNET_ACTION = { id: "magnet", label: "Open in your torrent app" };
  function offerActions(o) {
    if (prep()[o.id]) return [{ id: "stop", label: "Stop waiting" }];
    return o.e.u.some((x) => /^magnet:/i.test(x)) ? [...OFFER_ACTIONS, MAGNET_ACTION] : OFFER_ACTIONS;
  }
  // publish tells Lootnest the offers, and says so once for each new one.
  function publish(tell) {
    lootnest.updates.offer(OFFERS.slice(0, 500).map((o) => ({
      id: o.id, game: o.game, title: o.e.t, version: verText(o.ver), have: o.have, source: srcLabel(o.src), size: o.e.s || 0,
      date: o.e.d ? new Date(o.e.d).toISOString() : "", note: o.ver.from ? `A patch for ${o.ver.from}` : o.e.c || "", actions: offerActions(o),
      status: prep()[o.id] ? prep()[o.id].status || "Asking Real-Debrid" : "", busy: !!prep()[o.id],
    })));
    if (!tell) return;
    const told = lootnest.storage.get("told"), seen = new Set(Array.isArray(told) ? told : []);
    const fresh = OFFERS.filter((o) => !seen.has(o.id));
    if (!fresh.length) return;
    for (const o of fresh) seen.add(o.id);
    lootnest.storage.set("told", [...seen].slice(-2000));
    const f = fresh[0], v = verText(f.ver);
    lootnest.notify(fresh.length === 1 ? `${f.name}${v ? " " + v : ""} is listed by ${f.src.name || "a source"}` : `${plural(fresh.length, "update")} for your games are listed by your sources`);
  }
  // An offer's buttons, from the Updates page.
  function offerAction(a) {
    const o = OFFERS.find((x) => x.id === (a && a.id));
    if (!o) return { error: "That update isn't listed any more." };
    const l = linkOf(o.game);
    if (a.action === "ignore-version") {
      l.skip = [...(l.skip || []), o.eid].slice(-200);
      OFFERS = OFFERS.filter((x) => x !== o);
    } else if (a.action === "ignore-source") {
      l.mute = [...new Set([...(l.mute || []), o.src.id])];
      OFFERS = OFFERS.filter((x) => !(x.game === o.game && x.src.id === o.src.id));
    } else if (a.action === "update") {
      return startUpdate(o);
    } else if (a.action === "stop") {
      delete prep()[o.id];
      lootnest.storage.set("prep", prep());
      publish(false);
      return { toast: "Stopped waiting. What Real-Debrid fetched stays in your account there." };
    } else if (a.action === "magnet") {
      const m = o.e.u.find((x) => /^magnet:/i.test(x));
      if (!m) return { error: "This download has no magnet link." };
      try { lootnest.updates.run("magnet", { uri: m }); } catch (e) { return { error: clean(String(e && e.message || e), 300) }; }
      return { toast: "Opened in your torrent app. When it's in Downloads, Lootnest offers it on the Updates page." };
    } else return { error: "This plugin has no such action." };
    saveLinks();
    publish(false);
    lootnest.ui.changed();
    return { toast: a.action === "ignore-version" ? "Ignored. A later version will show again." : `Ignored ${o.src.name || "that source"} for ${o.name}.` };
  }
  // ---------------------------------------------------------------- updates --
  // Starting an update: this plugin only finds the addresses to download.
  // Lootnest downloads them into a staging folder, copies the saves, and
  // replaces the game's folder once the user has seen what's kept.
  // "active" in storage, by Library key: the update under way for a game
  // ({offer, title, date, version}), so its baseline can be set when it's
  // in place. "prep": downloads Real-Debrid is still fetching.
  let ACTIVE = null, PREP = null;
  const active = () => { if (!ACTIVE) { const a = lootnest.storage.get("active"); ACTIVE = isObj(a) ? a : {}; } return ACTIVE; };
  const prep = () => { if (!PREP) { const a = lootnest.storage.get("prep"); PREP = isObj(a) ? a : {}; } return PREP; };
  const ARCHIVE = /\.(?:zip|7z|rar|exe)(?:[?#].*)?$/i;
  const fileName = (u) => { try { return decodeURIComponent(u.split(/[?#]/)[0].split("/").pop() || ""); } catch (e) { return ""; } };
  const isMagnet = (u) => /^magnet:/i.test(u);

  // begin hands the files to Lootnest. It refuses with words when there's
  // no room, the game is running, or the game isn't one it may update.
  function begin(o, files) {
    try {
      lootnest.updates.run("start", { offer: o.id, game: o.game, title: o.e.t, version: verText(o.ver), source: srcLabel(o.src), size: o.e.s || 0, files });
    } catch (e) {
      return { error: clean(String(e && e.message || e), 300) };
    }
    active()[o.game] = { offer: o.id, title: o.e.t, date: o.e.d || Date.now(), version: verText(o.ver) };
    lootnest.storage.set("active", active());
    OFFERS = OFFERS.filter((x) => x.game !== o.game);
    publish(false);
    return { toast: `Downloading ${o.name}. It shows on the Updates page.` };
  }
  function startUpdate(o) {
    if (active()[o.game]) return { error: "An update of that game is already under way." };
    const u = o.e.u, rd = lootnest.secrets.has("realdebrid");
    // A file over https goes straight to Lootnest; plain http only with the
    // file's SHA-256 from the source.
    const file = u.find((x) => /^https:/i.test(x) && ARCHIVE.test(x)) || (o.e.h && u.find((x) => /^http:/i.test(x) && ARCHIVE.test(x)));
    if (file) return begin(o, [{ url: file, name: fileName(file), size: o.e.s || 0, sha256: o.e.h || "" }]);
    const magnet = u.find(isMagnet), page = u.find((x) => /^https:/i.test(x));
    if (rd && (magnet || page)) return rdStart(o, magnet || page);
    if (page) return begin(o, [{ url: page, name: fileName(page), size: o.e.s || 0, sha256: o.e.h || "" }]);
    if (magnet) return { error: "This download is a magnet link. Add your Real-Debrid token on the Sources page, or use Open in your torrent app and let Lootnest find it in Downloads." };
    return { error: "This download has no address Lootnest can use." };
  }
  // Lootnest tells what became of an update it ran.
  function updateState(a) {
    const act = active()[a.game];
    if (!act || (a.offer && act.offer !== a.offer)) return;
    const l = linkOf(a.game);
    if (a.state === "kept") {
      l.prev = l.base || null;
      l.base = { title: act.title, date: act.date, from: "update" };
    } else if (a.state === "rolledback") {
      if (l.prev) l.base = l.prev;
      delete l.prev;
    }
    if (a.state === "kept" || a.state === "rolledback" || a.state === "failed" || a.state === "done") {
      if (a.state !== "kept") delete active()[a.game];
      else active()[a.game].kept = true;
      lootnest.storage.set("active", active());
      saveLinks();
      lootnest.schedule("detect", 300);
    }
  }

  // ----------------------------------------------------------- Real-Debrid --
  // The user's own Real-Debrid account turns a magnet link, or a link to a
  // file host, into https links. The token is kept by Lootnest and goes
  // only to api.real-debrid.com; this script never reads it.
  const RD = "https://api.real-debrid.com/rest/1.0";
  function rdCall(method, path, form) {
    const r = lootnest.net.fetch(RD + path, { method, form, secret: "realdebrid", timeout: 60, maxBytes: 4 << 20 });
    let body = null;
    try { body = r.text ? JSON.parse(r.text) : null; } catch (e) { body = null; }
    if (r.status === 401) throw new Error("Real-Debrid refused the token. Add it again on the Sources page.");
    if (r.status === 403) throw new Error("Real-Debrid refused: " + clean((body && body.error) || "the account may not be premium", 120).replace(/_/g, " "));
    if (r.status < 200 || r.status >= 300) throw new Error("Real-Debrid answered " + r.status + (body && body.error ? ": " + clean(body.error, 120).replace(/_/g, " ") : ""));
    return body;
  }
  // One of the user's torrents there with this info hash, or a new one.
  function rdTorrent(magnet) {
    const hash = infoHash(magnet);
    if (hash) {
      const list = rdCall("GET", "/torrents?limit=100");
      const have = (Array.isArray(list) ? list : []).find((t) => t && typeof t.hash === "string" && t.hash.toLowerCase() === hash && !["error", "dead", "virus", "magnet_error"].includes(t.status));
      if (have && typeof have.id === "string") return have.id;
    }
    const added = rdCall("POST", "/torrents/addMagnet", { magnet });
    if (!added || typeof added.id !== "string" || !/^[A-Za-z0-9]{1,40}$/.test(added.id)) throw new Error("Real-Debrid didn't take the magnet link.");
    return added.id;
  }
  // Its links as files for Lootnest: each is unrestricted to an https
  // address, checked like any address from outside.
  function rdFiles(links) {
    const files = [];
    for (const link of links.slice(0, 300)) {
      if (!readURI(link, LIMITS.uri)) continue;
      const f = rdCall("POST", "/unrestrict/link", { link });
      const r = f && readURI(f.download, LIMITS.uri);
      if (!r || r.scheme !== "https") throw new Error("Real-Debrid gave an address Lootnest can't use.");
      files.push({ url: f.download, name: clean(f.filename, 200), size: Number.isFinite(f.filesize) && f.filesize > 0 ? f.filesize : 0 });
    }
    if (!files.length) throw new Error("Real-Debrid has no files for that download.");
    return files;
  }
  function rdStart(o, uri) {
    try {
      if (!isMagnet(uri)) return begin(o, rdFiles([uri]));
      const id = rdTorrent(uri);
      prep()[o.id] = { tid: id, game: o.game, at: Date.now(), status: "Asking Real-Debrid" };
      lootnest.storage.set("prep", prep());
      publish(false);
      lootnest.schedule("rd", 500);
      return { toast: "Real-Debrid is getting it ready. The download starts by itself." };
    } catch (e) {
      return { error: clean(String(e && e.message || e), 300) };
    }
  }
  const RD_WAIT = { magnet_conversion: "Real-Debrid is reading the magnet link", waiting_files_selection: "Real-Debrid is choosing the files", queued: "Waiting in Real-Debrid's queue",
    downloading: "Real-Debrid is fetching it", compressing: "Real-Debrid is packing it", uploading: "Real-Debrid is finishing it" };
  // The "rd" handler: looks at each download Real-Debrid is preparing, and
  // hands the ready ones to Lootnest.
  function rdTick() {
    const P = prep();
    let waiting = false;
    for (const id of Object.keys(P)) {
      const p = P[id], o = OFFERS.find((x) => x.id === id);
      const drop = (why) => { delete P[id]; if (why) lootnest.notify(why); };
      if (!o) { drop(""); continue; }
      if (Date.now() - p.at > 48 * HOUR) { drop(`${o.name}: Real-Debrid didn't finish in two days, so Lootnest stopped waiting.`); continue; }
      try {
        const info = rdCall("GET", "/torrents/info/" + encodeURIComponent(p.tid));
        const st = info && info.status;
        if (st === "waiting_files_selection") { rdCall("POST", "/torrents/selectFiles/" + encodeURIComponent(p.tid), { files: "all" }); p.status = RD_WAIT[st]; waiting = true; }
        else if (st === "downloaded") {
          const r = begin(o, rdFiles(Array.isArray(info.links) ? info.links : []));
          drop(r.error ? `${o.name}: ${r.error}` : "");
        } else if (RD_WAIT[st]) { p.status = RD_WAIT[st] + (st === "downloading" && Number.isFinite(info.progress) ? `, ${Math.round(info.progress)}%` : ""); waiting = true; }
        else drop(`${o.name}: Real-Debrid couldn't get it (${clean(String(st || "no answer"), 40).replace(/_/g, " ")}).`);
      } catch (e) {
        p.fails = (p.fails || 0) + 1;
        p.status = clean(String(e && e.message || e), 120);
        if (p.fails >= 5) drop(`${o.name}: ${p.status}`); else waiting = true;
      }
    }
    lootnest.storage.set("prep", P);
    publish(false);
    if (waiting) lootnest.schedule("rd", 8000);
  }

  // ----------------------------------------------------------------- pages --
  const NOTICE = {
    title: "Sources are yours to choose",
    body: "Lootnest comes with no sources and doesn't check what a source lists. What you add, what you download from it, and whether you may under the law and the publisher's terms, is up to you.",
    ok: "I understand",
  };
  const sourceForm = (title, ok, src) => ({
    title, ok,
    fields: [
      { id: "url", label: "Address of the JSON file", type: "url", value: src ? src.url : "", placeholder: "https://" },
      { id: "name", label: "Name", type: "text", value: src ? src.name : "", placeholder: "Leave empty to use the source's own", max: LIMITS.name },
    ],
  });
  function sourceRow(src, now) {
    const tags = [];
    if (src.count) tags.push(plural(src.count, "download"));
    if (src.format) tags.push((ADAPTERS.find((a) => a.id === src.format) || {}).name || src.format);
    const notes = [];
    if (src.skipped) notes.push(`${plural(src.skipped, "entry", "entries")} couldn't be read and ${src.skipped === 1 ? "was" : "were"} left out`);
    if (/^http:/i.test(src.url)) notes.push("Its address isn't https, so what it lists can be changed on the way");
    else if (src.http) notes.push(`${plural(src.http, "address", "addresses")} in it ${src.http === 1 ? "isn't" : "aren't"} https`);
    return {
      id: src.id,
      title: src.name || src.url,
      sub: src.url,
      lamp: !src.on ? "off" : src.error ? "bad" : forced.has(src.id) || !src.checked ? "busy" : notes.length ? "warn" : "on",
      tags,
      note: src.checked ? `Refreshed ${ago(src.checked, now)}` : src.on ? "Not refreshed yet" : "",
      warn: notes.join(". "),
      error: src.error || "",
      toggle: { on: !!src.on, label: `${src.name || src.url} on` },
      actions: [
        { id: "refresh", label: "Refresh" },
        { id: "edit", label: "Edit", form: "edit:" + src.id },
        { id: "remove", label: "Remove", confirm: { title: `Remove ${src.name || "this source"}?`, body: "Its list goes from Lootnest. Games you have stay as they are.", ok: "Remove" } },
      ],
    };
  }
  function sourcesView() {
    const s = state(), now = Date.now(), forms = { add: sourceForm("Add a source", "Add") };
    for (const src of s.list) forms["edit:" + src.id] = sourceForm("Edit the source", "Save", src);
    const add = s.notice ? { id: "add", label: "Add a source", kind: "primary", form: "add" } : { id: "notice", label: "Add a source", kind: "primary", confirm: NOTICE };
    const head = [add];
    if (s.list.some((x) => x.on)) head.push({ id: "refresh-all", label: "Refresh all" });
    const sections = [{ title: "Your sources", actions: head, rows: s.list.map((src) => sourceRow(src, now)), empty: "No sources yet. Lootnest comes with none." }];
    if (PENDING.length) sections.push({
      title: "Is this the same game?", note: "A source lists something with a name close to a game of yours.",
      rows: PENDING.slice(0, 20).map((p, i) => ({ id: "m" + i, title: p.name, sub: p.title, tags: [p.src],
        actions: [{ id: "match-yes", label: "Yes, the same", kind: "secondary" }, { id: "match-no", label: "No" }] })),
    });
    if (s.list.length) {
      const L = links(), rows = [];
      for (const g of GAMES) {
        const l = L[g.key] || {}, parts = [];
        if ((l.manual || []).length) parts.push("Linked by hand");
        if ((l.skip || []).length) parts.push(`${plural(l.skip.length, "version")} ignored`);
        if ((l.mute || []).length) parts.push(`${plural(l.mute.length, "source")} ignored`);
        if (!parts.length) continue;
        const acts = [];
        if ((l.manual || []).length) acts.push({ id: "unlink", label: "Unlink" });
        if ((l.skip || []).length || (l.mute || []).length) acts.push({ id: "unignore", label: "Stop ignoring" });
        rows.push({ id: "g:" + g.key, title: g.name, tags: parts, actions: acts });
      }
      forms.link = { title: "Link a game to a download", ok: "Find", body: "For a game whose name in your sources is too different to match by itself.", fields: [
        { id: "game", label: "Game", type: "select", value: "", options: GAMES.slice().sort((a, b) => (a.name.toLowerCase() < b.name.toLowerCase() ? -1 : 1)).slice(0, 500).map((g) => ({ value: g.key, label: g.name })) },
        { id: "words", label: "Words from the download's title", type: "text", max: 100 }] };
      if (PICK) forms.pick = { title: `Which one is ${PICK.name}?`, ok: "Link", fields: [{ id: "k", label: "Downloads in your sources", type: "select", value: PICK.options[0].value, options: PICK.options }] };
      if (GAMES.length) sections.push({ title: "Games", actions: [{ id: "link", label: "Link a game", form: "link" }], rows, empty: "Games are matched to your sources by name. Nothing here needs you." });
    }
    const rd = lootnest.secrets.has("realdebrid");
    forms.rd = { title: "Your Real-Debrid token", ok: "Save", body: "Lootnest keeps it encrypted on this PC and sends it only to api.real-debrid.com. This plugin never reads it.",
      fields: [{ id: "token", label: "API token", secret: "realdebrid", hint: "From your Real-Debrid account, under API token." }] };
    sections.push({ title: "Real-Debrid", rows: [{ id: "rd", title: rd ? (s.rdUser ? `Connected as ${s.rdUser}` : "Connected") : "Not connected", lamp: rd ? "on" : "off",
      sub: "Your own account there turns magnet links and file-host links into downloads. Without it, those open in your torrent app.",
      actions: rd ? [{ id: "rd-set", label: "Change the token", form: "rd" }, { id: "rd-clear", label: "Disconnect", confirm: { title: "Disconnect Real-Debrid?", body: "Lootnest forgets the token. Your account there isn't changed.", ok: "Disconnect" } }]
        : [{ id: "rd-set", label: "Add your token", kind: "secondary", form: "rd" }] }] });
    sections.push({ title: "Refresh", rows: [{ id: "hours", title: "Look for new entries", select: { value: String(s.hours), options: INTERVALS.map((h) => ({ value: String(h), label: h ? `Every ${h === 1 ? "hour" : h + " hours"}` : "Only when I ask" })) } }] });
    return { sections, forms };
  }
  let PICK = null; // {game, name, options} while a game is being linked by hand

  const ACTIONS = {
    notice() { state().notice = Date.now(); save(); return { form: "add" }; },
    add(a) {
      const s = state();
      if (!s.notice) return { error: "Read the notice first." };
      if (s.list.length >= LIMITS.sources) return { error: `That's the most sources (${LIMITS.sources}).` };
      const v = readSource(a.values);
      const src = { id: lootnest.hash(v.url + "|" + Date.now()).slice(0, 12), name: v.name, url: v.url, on: true, added: Date.now() };
      s.list.push(src);
      save();
      forced.add(src.id);
      planRefresh();
      return { toast: "Added. Reading it now." };
    },
    edit(a) {
      const src = sourceBy(a.row);
      if (!src) return { error: "That source is gone." };
      const v = readSource(a.values, src.id);
      if (v.url !== src.url) {
        Object.assign(src, { url: v.url, etag: "", modified: "", checked: 0, count: 0, skipped: 0, http: 0, error: "", format: "" });
        lootnest.storage.remove("feed-" + src.id);
        forced.add(src.id);
        feedsChanged();
      }
      src.name = v.name || src.name;
      save();
      planRefresh();
      return {};
    },
    remove(a) {
      const s = state(), i = s.list.findIndex((x) => x.id === a.row);
      if (i < 0) return {};
      const [src] = s.list.splice(i, 1);
      lootnest.storage.remove("feed-" + src.id);
      save();
      feedsChanged();
      return { toast: `Removed ${src.name || "the source"}.` };
    },
    toggle(a) {
      const src = sourceBy(a.row);
      if (!src) return {};
      src.on = !!a.on;
      save();
      if (src.on) forced.add(src.id);
      planRefresh();
      feedsChanged();
      return {};
    },
    refresh(a) {
      const src = sourceBy(a.row);
      if (!src || !src.on) return { error: "Turn the source on first." };
      forced.add(src.id);
      planRefresh();
      return {};
    },
    "refresh-all"() {
      for (const src of state().list) if (src.on) forced.add(src.id);
      planRefresh();
      return {};
    },
    // The token went to Lootnest with the form; this only checks it works.
    "rd-set"(a) {
      if (!(a.secrets || {}).realdebrid) return { error: "Paste the token." };
      try {
        const u = rdCall("GET", "/user");
        state().rdUser = clean(u && u.username, 60);
        save();
        return { toast: state().rdUser ? `Real-Debrid is connected as ${state().rdUser}.` : "Real-Debrid is connected." };
      } catch (e) {
        lootnest.secrets.remove("realdebrid");
        return { error: clean(String(e && e.message || e), 300) };
      }
    },
    "rd-clear"() {
      lootnest.secrets.remove("realdebrid");
      delete state().rdUser;
      save();
      return { toast: "Real-Debrid is disconnected." };
    },
    "match-yes"(a) { return answerMatch(a, "yes"); },
    "match-no"(a) { return answerMatch(a, "no"); },
    // Linking by hand: the words typed are looked for in every kept title.
    link(a) {
      const g = GAMES.find((x) => x.key === (a.values || {}).game), words = normTitle((a.values || {}).words);
      if (!g) return { error: "Choose a game." };
      if (!words.length) return { error: "Type a word or two from the download's title." };
      const options = [];
      for (const [k, info] of KEYS) {
        const have = k.split(" ");
        if (words.every((w) => have.includes(w))) options.push({ value: k, label: `${info.title.slice(0, 90)} (${info.src})` });
        if (options.length >= 30) break;
      }
      if (!options.length) return { error: "None of your sources lists a download with those words." };
      PICK = { game: g.key, name: g.name, options };
      return { form: "pick", act: "pick" };
    },
    pick(a) {
      const k = (a.values || {}).k;
      if (!PICK || !PICK.options.some((o) => o.value === k)) return { error: "Choose one." };
      const l = linkOf(PICK.game);
      l.manual = [...new Set([...(l.manual || []), k])].slice(-10);
      l.no = (l.no || []).filter((x) => x !== k);
      PICK = null;
      saveLinks();
      detect();
      return { toast: "Linked." };
    },
    unlink(a) {
      const l = links()[String(a.row).slice(2)];
      if (l) { delete l.manual; saveLinks(); detect(); }
      return {};
    },
    unignore(a) {
      const l = links()[String(a.row).slice(2)];
      if (l) { delete l.skip; delete l.mute; saveLinks(); detect(); }
      return {};
    },
    select(a) {
      if (a.row !== "hours" || !INTERVALS.includes(+a.value)) return {};
      state().hours = +a.value;
      save();
      planRefresh();
      return {};
    },
  };

  function answerMatch(a, what) {
    const p = PENDING[+String(a.row).slice(1)];
    if (!p) return {};
    const l = linkOf(p.game);
    l[what] = [...new Set([...(l[what] || []), p.k])].slice(-50);
    saveLinks();
    detect();
    return {};
  }

  lootnest.on("start", () => { state(); lootnest.schedule("refresh", 5000); lootnest.schedule("detect", 1500); if (Object.keys(prep()).length) lootnest.schedule("rd", 4000); });
  lootnest.on("refresh", refreshTick);
  lootnest.on("detect", detect);
  lootnest.on("library", () => lootnest.schedule("detect", 300));
  lootnest.on("offer", offerAction);
  lootnest.on("update", updateState);
  lootnest.on("rd", rdTick);
  lootnest.on("view", (a) => (a && a.page === "sources" ? sourcesView() : { sections: [] }));
  lootnest.on("action", (a) => {
    const f = a && Object.prototype.hasOwnProperty.call(ACTIONS, a.id) ? ACTIONS[a.id] : null;
    if (!f) return { error: "This plugin has no such action." };
    try { return f(a) || {}; } catch (e) { return { error: clean(String(e && e.message || e), 300) }; }
  });
  // For tests: the pure parts, called by name with JSON.
  lootnest.on("test", (a) => {
    if (a.what === "feed") { try { return parseFeed(a.text, a.now || Date.now()); } catch (e) { return { error: e.message }; } }
    if (a.what === "uri") return readURI(a.uri, LIMITS.uri);
    if (a.what === "size") return sizeBytes(a.size);
    if (a.what === "hash") return infoHash(a.uri);
    if (a.what === "label") return srcLabel(a.src);
    if (a.what === "norm") return normTitle(a.title);
    if (a.what === "score") return titleScore(normTitle(a.a), normTitle(a.b));
    if (a.what === "version") return titleVersion(a.title);
    if (a.what === "compare") return compareVer(titleVersion(a.a), titleVersion(a.b));
    if (a.what === "installed") return installed(a.game, a.link || {});
    if (a.what === "detect") { detect(); return { offers: OFFERS.map((o) => ({ id: o.id, game: o.game, title: o.e.t, version: verText(o.ver), have: o.have })), pending: PENDING }; }
    return null;
  });
})();
