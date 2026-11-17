// Sources and updates, a Lootnest script plugin (docs/PLUGINS.md).
//
// The user adds sources: addresses of JSON files that list downloads. This
// plugin reads them, never trusts them, and keeps the last good copy of
// each. It comes with no source, suggests none and looks nowhere the user
// didn't point it to.
//
// Parts: limits, small helpers, feed formats (adapters), sources and their
// refresh, the Sources page.
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
    const s = v.replace(/[\u0000-\u001f\u007f-\u009f​-‏ -‮⁦-⁩﻿]/g, " ").replace(/\s+/g, " ").trim();
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

  // --------------------------------------------------------------- formats --
  // An adapter reads one feed format: sniff(doc) says whether the document
  // is its kind, parse(doc, now) gives {name, entries, skipped, http}.
  // An entry: {t: title, u: [addresses], d: date in ms or 0, s: bytes or 0}
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
    const e = { t, u, d: dateMs(raw.uploadDate, now), s: sizeBytes(raw.fileSize) };
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
  const feedOf = (id) => { const f = lootnest.storage.get("feed-" + id); return isObj(f) && Array.isArray(f.entries) ? f.entries : null; };

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
    const cached = feedOf(src.id) !== null;
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
    lootnest.storage.set("feed-" + src.id, { entries: feed.entries });
    Object.assign(src, { error: "", format: feed.format, count: feed.entries.length, skipped: feed.skipped, http: feed.http, changed: now,
      etag: clean(r.headers.etag, 200), modified: clean(r.headers["last-modified"], 60) });
    if (!src.name) src.name = feed.name || src.url.replace(/^https?:\/\//, "").split("/")[0];
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
        lootnest.log(`${src.name || src.url}: ${src.error || (got ? plural(src.count, "download") : "no change")}`);
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
  // Called when a source's entries changed (overridden by the parts below).
  let feedsChanged = () => {};

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
    return {
      sections: [
        { title: "Your sources", actions: head, rows: s.list.map((src) => sourceRow(src, now)), empty: "No sources yet. Lootnest comes with none." },
        { title: "Refresh", rows: [{ id: "hours", title: "Look for new entries", select: { value: String(s.hours), options: INTERVALS.map((h) => ({ value: String(h), label: h ? `Every ${h === 1 ? "hour" : h + " hours"}` : "Only when I ask" })) } }] },
      ],
      forms,
    };
  }

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
    select(a) {
      if (a.row !== "hours" || !INTERVALS.includes(+a.value)) return {};
      state().hours = +a.value;
      save();
      planRefresh();
      return {};
    },
  };

  lootnest.on("start", () => { state(); lootnest.schedule("refresh", 5000); });
  lootnest.on("refresh", refreshTick);
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
    return null;
  });
})();
