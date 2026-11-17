  // -------------------------------------------------------- script plugins --
  // A script plugin's pages (core/pluginscript.go, app/pluginscript.go). The
  // script describes a page as data: sections of rows, each with text, a
  // lamp, a switch, a select and buttons, and forms for its buttons.
  // Lootnest draws it with its own components and escapes every word, so a
  // plugin brings no HTML, no CSS and no script into the window. A key (a
  // field with `secret`) goes to Go, which keeps it; the script never reads
  // it.
  const PP = { views: {}, asked: {}, errors: {}, waiting: {}, next: {}, n: 0 };
  const ppKey = (id, page) => id + "/" + page;
  const ppTab = (p) => "pp:" + p.plugin + ":" + p.id;
  const ppShown = () => (S.view === "settings" ? (S.d.pluginPages || []).find((p) => ppTab(p) === S.setTab) : null);
  function askPluginView(id, page, again) {
    const k = ppKey(id, page);
    if (PP.asked[k] && !again) return;
    PP.asked[k] = true;
    api.pluginView(id, page).then((v) => { if (v && !PP.views[k]) { PP.views[k] = v; if (ppShown()) render(); } }).catch(() => { PP.asked[k] = false; });
  }
  const PP_KINDS = ["primary", "secondary", "ghost", "danger"];
  function ppButton(p, a, row) {
    const kind = PP_KINDS.includes(a.kind) ? a.kind : row ? "ghost" : "secondary";
    return `<button class="ln-button ln-button--${kind} ln-button--sm" data-action="pp-act" data-plugin="${esc(p.plugin)}" data-page="${esc(p.id)}" data-act="${esc(a.id)}" data-row="${esc(row || "")}"${a.form || a.confirm ? ' aria-haspopup="dialog"' : ""}>${esc(a.label)}</button>`;
  }
  const PP_LAMPS = { on: "ln-lamp ln-lamp--on", warn: "ln-lamp ln-lamp--warn", bad: "ln-lamp ln-lamp--bad", off: "ln-lamp" };
  function ppRow(p, r) {
    const at = `data-plugin="${esc(p.plugin)}" data-page="${esc(p.id)}" data-row="${esc(r.id || "")}"`;
    const lamp = r.lamp === "busy" ? '<span class="ln-spinner" aria-label="Working"></span>' : PP_LAMPS[r.lamp] ? `<span class="${PP_LAMPS[r.lamp]}" aria-hidden="true"></span>` : "";
    const tags = (r.tags || []).map((t) => `<span class="ln-tag">${esc(t)}</span>`).join("") + (r.note ? `<span>${esc(r.note)}</span>` : "");
    const sel = r.select && Array.isArray(r.select.options) ? `<select class="ln-input set-input" data-pp-select ${at} aria-label="${esc(r.title)}">${r.select.options.map((o) =>
      `<option value="${esc(o.value)}"${String(o.value) === String(r.select.value) ? " selected" : ""}>${esc(o.label)}</option>`).join("")}</select>` : "";
    const sw = r.toggle ? `<span class="switch"><input type="checkbox" data-pp-toggle ${at} aria-label="${esc(r.toggle.label || r.title)}" ${r.toggle.on ? "checked" : ""}><span></span></span>` : "";
    return `<div class="bk-row pl-row pp-row"><div>
        <b class="pp-title">${lamp}<span>${esc(r.title)}</span></b>
        ${r.sub ? `<p class="selectable pp-sub">${esc(r.sub)}</p>` : ""}
        ${tags ? `<p class="pl-key">${tags}</p>` : ""}
        ${r.warn ? `<p class="pp-warn">${esc(r.warn)}</p>` : ""}
        ${r.error ? `<p class="error-text">${esc(r.error)}</p>` : ""}</div>
      <div class="pl-acts">${(r.actions || []).map((a) => ppButton(p, a, r.id || "")).join("")}${sel}${sw}</div></div>`;
  }
  function viewPluginPage(p) {
    const k = ppKey(p.plugin, p.id), V = PP.views[k];
    askPluginView(p.plugin, p.id);
    if (!V) return PP.errors[k] ? setSub("", "", `<div class="bk-card"><div class="bk-row"><div><b>${esc(p.name)} couldn't draw this page</b><p class="error-text">${esc(PP.errors[k])}</p></div></div></div>`) : lnInline("", "Asking " + esc(p.name));
    return (V.note ? `<p class="set-note">${esc(V.note)}</p>` : "") + (V.sections || []).map((s) => `<section class="set-sub">
        <div class="ln-section-head"><div><h2>${esc(s.title || "")}</h2>${s.note ? `<p>${esc(s.note)}</p>` : ""}</div><div class="pl-acts">${(s.actions || []).map((a) => ppButton(p, a, "")).join("")}</div></div>
        <div class="bk-card">${(s.rows || []).map((r) => ppRow(p, r)).join("") || `<div class="bk-row"><div><b>${esc(s.empty || "Nothing here yet")}</b></div></div>`}</div></section>`).join("")
      + `<p class="set-note pp-by">This page is drawn by the plugin ${esc(p.name)}.</p>`;
  }
  // The action a button stands for, found again in the page's last view.
  function ppFind(plugin, page, row, act) {
    const V = PP.views[ppKey(plugin, page)];
    for (const s of (V && V.sections) || []) {
      if (!row) { const a = (s.actions || []).find((x) => x.id === act); if (a) return a; continue; }
      const r = (s.rows || []).find((x) => x.id === row), a = r && (r.actions || []).find((x) => x.id === act);
      if (a) return a;
    }
    return null;
  }
  // Sends an action to the plugin's script; the answer is the event
  // pluginActed with the same req.
  function ppSend(plugin, page, o, D) {
    const req = "pp" + (++PP.n);
    PP.waiting[req] = { plugin, page, D };
    api.pluginAction(plugin, Object.assign({ req, page }, o)).catch((e) => { delete PP.waiting[req]; ppAnswer({ plugin, page, D }, { error: errText(e) }); });
  }
  function ppAnswer(w, r) {
    const D = w.D && S.dialog === w.D ? w.D : null;
    if (r.error) {
      if (D) { D.busy = false; D.error = r.error; renderOverlay(); } else toast(r.error, "error");
      return;
    }
    if (D) { S.dialog = null; renderOverlay(); }
    if (r.toast) toast(r.toast);
    if (!r.form) return;
    // A form the plugin made for this answer comes with the page's next view.
    const V = PP.views[ppKey(w.plugin, w.page)], next = { form: r.form, act: r.act || String(r.form).split(":")[0], row: r.row || "" };
    if (V && V.forms && V.forms[r.form]) ppOpenForm(w.plugin, w.page, next.form, next.act, next.row);
    else PP.next[ppKey(w.plugin, w.page)] = next;
  }
  function ppOpenForm(plugin, page, form, act, row) {
    const V = PP.views[ppKey(plugin, page)], F = V && V.forms && V.forms[form];
    if (!F) return;
    S.dialog = { kind: "ppform", plugin, page, form, act, row, error: "", busy: false };
    renderOverlay();
    setTimeout(() => { const i = $("#overlay [data-pp-field], #overlay [data-pp-secret]"); if (i) i.focus(); }, 20);
  }
  function ppFormHTML(D) {
    const V = PP.views[ppKey(D.plugin, D.page)], F = V && V.forms && V.forms[D.form];
    if (!F) return "";
    const dis = D.busy ? "disabled" : "";
    const field = (f) => {
      const id = "pp-f-" + esc(f.id), at = f.secret ? `data-pp-secret="${esc(f.secret)}"` : `data-pp-field="${esc(f.id)}"`;
      let input;
      if (f.type === "select") input = `<select id="${id}" class="ln-input" ${at} ${dis}>${(f.options || []).map((o) => `<option value="${esc(o.value)}"${String(o.value) === String(f.value) ? " selected" : ""}>${esc(o.label)}</option>`).join("")}</select>`;
      else if (f.type === "check") return `<label class="toggle"><span><b>${esc(f.label)}</b>${f.hint ? `<p>${esc(f.hint)}</p>` : ""}</span><span class="switch"><input type="checkbox" ${at} ${f.value ? "checked" : ""} ${dis}><span></span></span></label>`;
      else input = `<input id="${id}" class="ln-input" type="${f.secret || f.type === "password" ? "password" : f.type === "number" ? "number" : "text"}" ${at} value="${f.secret ? "" : esc(f.value == null ? "" : f.value)}"
          placeholder="${esc(f.placeholder || "")}" maxlength="${Math.min(+f.max || 2000, 4000)}" autocomplete="off" spellcheck="false" ${f.type === "url" ? 'inputmode="url"' : ""} ${dis}>`;
      return `<div class="pp-field"><label for="${id}">${esc(f.label)}</label>${input}${f.hint ? `<p>${esc(f.hint)}</p>` : ""}</div>`;
    };
    return `<div class="scrim" data-action="scrim"><form class="dialog narrow" role="dialog" aria-modal="true" aria-label="${esc(F.title)}" data-pp-form novalidate>
      <div><h2>${esc(F.title)}</h2>${F.body ? `<p class="sub">${esc(F.body)}</p>` : ""}</div>
      ${(F.fields || []).map(field).join("")}
      <div class="dialog-foot"><span class="error-text grow" role="alert">${esc(D.error || "")}</span><button type="button" class="btn quiet" data-action="dialog-cancel" ${dis}>Cancel</button>
      <button type="submit" class="btn primary" ${dis}>${esc(F.ok || "OK")}</button></div></form></div>`;
  }
  document.addEventListener("submit", (ev) => {
    if (!ev.target.matches || !ev.target.matches("[data-pp-form]")) return;
    ev.preventDefault();
    const D = S.dialog;
    if (!D || D.kind !== "ppform" || D.busy) return;
    const values = {}, secrets = {};
    ev.target.querySelectorAll("[data-pp-field]").forEach((i) => { values[i.dataset.ppField] = i.type === "checkbox" ? i.checked : i.value; });
    ev.target.querySelectorAll("[data-pp-secret]").forEach((i) => { if (i.value) secrets[i.dataset.ppSecret] = i.value; });
    D.busy = true; D.error = "";
    ppSend(D.plugin, D.page, { id: D.act, row: D.row, values, secrets }, D);
  });
  document.addEventListener("change", (ev) => {
    const el = ev.target, d = el.dataset || {};
    if (d.ppToggle !== undefined) ppSend(d.plugin, d.page, { id: "toggle", row: d.row, on: el.checked });
    else if (d.ppSelect !== undefined) ppSend(d.plugin, d.page, { id: "select", row: d.row, value: el.value });
  });
  // What Go says about plugins' pages (15-events.js passes these on).
  function pluginEvent(ev) {
    if (ev.type === "pluginPages") {
      if (!S.d) return;
      S.d.pluginPages = ev.pages || [];
      if (S.view === "settings") { if (/^pp:/.test(S.setTab) && !ppShown()) S.setTab = "plugins"; render(); }
    } else if (ev.type === "pluginView") {
      const k = ppKey(ev.id, ev.page);
      if (ev.error) PP.errors[k] = ev.error; else { PP.views[k] = ev.view; delete PP.errors[k]; }
      const nx = PP.next[k];
      if (nx && !ev.error) { delete PP.next[k]; if (!S.dialog) ppOpenForm(ev.id, ev.page, nx.form, nx.act, nx.row); }
      const p = ppShown();
      if (p && p.plugin === ev.id && p.id === ev.page && !(document.activeElement && document.activeElement.matches("[data-pp-select]"))) render();
    } else if (ev.type === "pluginChanged") {
      const p = ppShown();
      Object.keys(PP.asked).forEach((k) => { if (k.startsWith(ev.id + "/")) PP.asked[k] = false; });
      if (p && p.plugin === ev.id) askPluginView(p.plugin, p.id, true);
    } else if (ev.type === "pluginOffers") {
      askOffers(true);
    } else if (ev.type === "pluginJobs") {
      askJobs(true);
    } else if (ev.type === "pluginActed") {
      const w = PP.waiting[ev.req];
      if (!w) return;
      delete PP.waiting[ev.req];
      ppAnswer(w, ev.error ? { error: ev.error } : ev.result || {});
    }
  }
  // Updates that script plugins offer for games in the Library
  // (ps_pluginOffers): rows on the Updates page, a count in the nav and a
  // line on the game's page. The words are the plugin's, drawn as text.
  let offersAsked = false;
  function askOffers(again) {
    if (offersAsked && !again) return;
    offersAsked = true;
    api.pluginOffers().then((l) => { S.plugOffers = l || []; updateNavCount(); if (S.view === "updates" || S.view === "game") render(); }).catch(() => { offersAsked = false; });
  }
  // An offer's first action is its key; the others are quiet links.
  const offerAt = (o, a) => `data-action="pp-offer" data-plugin="${esc(o.plugin)}" data-offer="${esc(o.id)}" data-act="${esc(a.id)}"`;
  const offerKey = (o, lead) => { const a = !o.busy && (o.actions || [])[0]; return a ? `<button class="ln-button ${lead ? "ln-button--primary" : "ln-button--secondary"} ln-button--sm" ${offerAt(o, a)} aria-label="${esc(a.label)} ${esc(o.name)}">${esc(a.label)}</button>` : ""; };
  const offerRest = (o) => (o.busy ? "" : (o.actions || []).slice(1).map((a) => `<button class="upd-note" ${offerAt(o, a)}>${esc(a.label)}</button>`).join(""));
  const offerFacts = (o) => [o.size ? fmtBytes(o.size) : "", o.date && !isNaN(new Date(o.date)) ? day(o.date) : ""].filter(Boolean).join(", ");
  function offerCard(o, lead) {
    const facts = offerFacts(o);
    const meta = `<p class="upd-meta"><span>From ${esc(o.source || o.pluginName)}</span><span class="upd-why" title="${esc(o.title)}">${esc(o.title)}</span>${facts ? `<span>${esc(facts)}</span>` : ""}</p>
      ${o.status ? `<p class="upd-meta pp-offer-status">${o.busy ? '<span class="ln-spinner" aria-hidden="true"></span>' : ""}<span>${esc(o.status)}</span></p>` : o.note ? `<p class="upd-meta"><span>${esc(o.note)}</span></p>` : ""}
      ${offerRest(o) ? `<p class="upd-meta">${offerRest(o)}</p>` : ""}`;
    return updRow({ key: o.key, name: o.name }, { k: "new", text: "Newer version listed" }, meta, updVer(o.have || "", o.version ? ["Listed", o.version] : null), offerKey(o, lead));
  }
  // An update under way hides the offers for its game.
  const liveOffers = () => (S.plugOffers || []).filter((o) => !(S.plugJobs || []).some((j) => j.updKey === o.key && !["done", "rolledback", "failed"].includes(j.phase)));
  function offersSection() {
    const jobs = S.plugJobs || [], offers = liveOffers();
    if (!jobs.length && !offers.length) return "";
    return labelSec("From plugins", jobs.length + offers.length, `<ol class="ln-rows upd-cards pp-offers">${jobs.map(jobCard).join("")}${offers.map((o, i) => offerCard(o, i === 0 && !jobs.length)).join("")}</ol>`);
  }

  // Updates a plugin started (ps_pluginJobs, app/fullupdate.go): downloaded
  // into a staging folder, opened, and applied by Lootnest once the user has
  // seen what's kept. The steps show where each is.
  let jobsAsked = false;
  function askJobs(again) {
    if (jobsAsked && !again) return;
    jobsAsked = true;
    api.pluginJobs().then((l) => {
      S.plugJobs = l || [];
      updateNavCount();
      if (S.view === "updates" || S.view === "game") { if (!S.dialog) render(); else if (!ppJobsLater) ppJobsLater = setTimeout(() => { ppJobsLater = 0; if (S.view === "updates" || S.view === "game") render(); }, 600); }
    }).catch(() => { jobsAsked = false; });
  }
  let ppJobsLater = 0;
  const JOB_STEPS = ["Download", "Open", "Copy saves", "Replace", "Check"];
  // Only what changed: the archive is read where it lies (deltaupdate.go).
  const DELTA_STEPS = ["Compare", "Get what changed", "Copy saves", "Replace"];
  const isDelta = (j) => /^delta/.test(j.mode || "");
  function jobAt(j) {
    if (isDelta(j)) {
      if (j.phase === "download") return 1;
      if (j.phase === "fetch") return j.step === "fetch" ? 1 : 0;
      if (j.phase === "ready" || j.phase === "hand") return 2;
      if (j.phase === "apply") return /saves/i.test(j.step || "") ? 2 : 3;
      return j.phase === "kept" || j.phase === "done" ? 4 : -1;
    }
    if (j.phase === "download") return 0;
    if (j.phase === "unpack") return 1;
    if (j.phase === "ready" || j.phase === "hand") return 2;
    if (j.phase === "apply") return /saves/i.test(j.step || "") ? 2 : /Check/.test(j.step || "") ? 4 : 3;
    return j.phase === "kept" || j.phase === "done" ? 5 : -1;
  }
  const jobSteps = (j) => { const at = jobAt(j); return at < 0 ? "" : `<ol class="pp-steps" aria-label="Steps">${(isDelta(j) ? DELTA_STEPS : JOB_STEPS).map((s, i) => `<li class="${i < at ? "is-done" : i === at && j.phase !== "ready" && j.phase !== "hand" ? "is-now" : ""}"${i === at ? ' aria-current="step"' : ""}>${s}</li>`).join("")}</ol>`; };
  const mins = (s) => (s < 90 ? "a minute" : s < 5400 ? `${Math.round(s / 60)} minutes` : `${Math.round(s / 3600)} hours`);
  const jobBtn = (j, op, label, cls) => `<button class="${cls || "upd-note"}" data-action="pp-job" data-job="${esc(j.id)}" data-op="${op}"${op === "plan" || op === "log" ? ' aria-haspopup="dialog"' : ""}>${label}</button>`;
  const jobKey = (j, op, label, primary) => jobBtn(j, op, label, `ln-button ${primary ? "ln-button--primary" : "ln-button--secondary"} ln-button--sm`);
  // What a job says, its key and its quiet links, by phase.
  function jobParts(j) {
    const log = jobBtn(j, "log", "Show the log"), cancel = jobBtn(j, "cancel", "Cancel");
    switch (j.phase) {
      case "download": {
        const bar = j.total ? `<progress class="pp-bar" max="${j.total}" value="${Math.min(j.done, j.total)}" aria-label="Downloaded"></progress>` : "";
        if (j.error) return { text: j.error, bad: true, key: jobKey(j, "resume", "Try again"), rest: cancel + log };
        if (j.paused) return { text: `Paused at ${fmtBytes(j.done)}${j.total ? ` of ${fmtBytes(j.total)}` : ""}`, bar, key: jobKey(j, "resume", "Resume"), rest: cancel };
        return { text: `Downloading: ${fmtBytes(j.done)}${j.total ? ` of ${fmtBytes(j.total)}` : ""}${j.speed ? `, ${fmtBytes(j.speed)} a second` : ""}${j.eta ? `, about ${mins(j.eta)} left` : ""}`, bar,
          key: jobKey(j, "pause", "Pause"), rest: cancel };
      }
      case "fetch": {
        const bar = j.of ? `<progress class="pp-bar" max="${j.of}" value="${Math.min(j.count || 0, j.of)}" aria-label="${j.step === "fetch" ? "Files taken" : "Files compared"}"></progress>` : "";
        const got = j.done ? `${fmtBytes(j.done)} downloaded` : "";
        if (j.error) return { text: j.error, bad: true, key: jobKey(j, "resume", "Try again"), rest: cancel + log };
        if (j.paused) return { text: `Paused${got ? `, ${got}` : ""}. What's taken so far is kept.`, bar, key: jobKey(j, "resume", "Resume"), rest: cancel };
        const what = j.step === "fetch"
          ? `Taking the changed files${j.of ? `: ${j.count || 0} of ${j.of}` : ""}`
          : `Comparing with your files${j.of ? `: ${j.count || 0} of ${j.of}` : ""}`;
        return { text: `${what}${got ? `, ${got}` : ""}${j.speed ? `, ${fmtBytes(j.speed)} a second` : ""}`, bar, busy: !j.of,
          key: jobKey(j, "pause", "Pause"), rest: cancel };
      }
      case "unpack": return { text: "Opening the download", busy: true, key: "", rest: "" };
      case "ready": return { text: isDelta(j) ? "Only the changed files were taken. Nothing in the game's folder has changed yet." : "Downloaded and opened. Nothing in the game's folder has changed yet.", key: jobKey(j, "plan", "Review and update", true), rest: cancel + log };
      case "hand": return { text: j.kind === "installer" ? "It's an installer. Lootnest runs it only when you say so, after copying your saves." : "It isn't a whole game: its files go over the game's, with each one it replaces kept.",
        key: jobKey(j, "hand", "Open the update", true), rest: jobBtn(j, "dismiss", "Delete the download") + log };
      case "apply": return { text: j.step || "Updating", busy: true, key: "", rest: "" };
      case "kept": return isDelta(j)
        ? { text: j.broke ? "The game closed right after its first start. Roll back puts the old files back." : "Updated. The files it replaced are kept until you've played, or for 14 days.", warn: j.broke,
          key: jobKey(j, "rollback", "Roll back"), rest: jobBtn(j, "finish", "Remove the old files now") + log }
        : { text: j.broke ? "The game closed right after its first start. Roll back puts the folder from before back." : "Updated. The folder from before is kept until you've played, or for 14 days.", warn: j.broke,
          key: jobKey(j, "rollback", "Roll back"), rest: jobBtn(j, "finish", "Remove the old folder now") + log };
      case "failed": return { text: j.error || "The update failed. The game is as it was.", bad: true, key: jobKey(j, "dismiss", "Dismiss"), rest: log };
      case "rolledback": return { text: "Rolled back. The game is as it was before the update.", key: jobKey(j, "dismiss", "Dismiss"), rest: log };
      default: return { text: "Updated.", key: jobKey(j, "dismiss", "Dismiss"), rest: log };
    }
  }
  function jobCard(j) {
    const p = jobParts(j), st = j.phase === "failed" || ((j.phase === "download" || j.phase === "fetch") && j.error) ? { k: "low", text: "Needs a look" } : j.phase === "kept" || j.phase === "done" ? { k: "ok", text: "Updated" }
      : j.phase === "rolledback" ? { k: "dim", text: "Rolled back" } : j.phase === "ready" || j.phase === "hand" ? { k: "new", text: "Ready" } : { k: "new", text: "Updating" };
    const meta = `<p class="upd-meta"><span>From ${esc(j.source || j.pluginName || "a plugin")}</span><span class="upd-why" title="${esc(j.title)}">${esc(j.title)}</span></p>
      ${jobSteps(j)}
      <p class="upd-meta pp-job-text${p.bad ? " is-bad" : p.warn ? " is-warn" : ""}">${p.busy ? '<span class="ln-spinner" aria-hidden="true"></span>' : ""}<span>${esc(p.text)}</span></p>${p.bar || ""}
      ${p.rest ? `<p class="upd-meta">${p.rest}</p>` : ""}`;
    return updRow({ key: j.updKey, name: j.name }, st, meta, j.version ? updVer("", ["Updating to", j.version]).replace(/<div><span>Your copy<\/span>[\s\S]*?<\/div>\s*<svg[\s\S]*?<\/svg>/, "") : "<div></div>", p.key);
  }
  function jobBanner(it) {
    const j = it.kind === "found" && (S.plugJobs || []).find((x) => x.updKey === "found:" + it.id && x.phase !== "done");
    if (!j) return "";
    const p = jobParts(j);
    return banner(j.phase === "kept" ? "Updated" : j.phase === "failed" ? "The update failed" : j.phase === "rolledback" ? "Rolled back" : "Updating", esc(p.text),
      p.key + `<button class="ln-button ln-button--ghost ln-button--sm" data-action="nav" data-view="updates">More</button>`, p.busy, p.bad ? "warn" : "");
  }
  // The dialog before a folder is replaced: what's copied, what's carried
  // over, what else could be, and where the old folder stays.
  function ppJobHTML(D) {
    const j = (S.plugJobs || []).find((x) => x.id === D.job), P = D.plan;
    if (!j) return "";
    const item = (k) => `<li><span class="selectable">${esc(k.rel)}</span><span class="ln-num">${esc(fmtBytes(k.size || 0))}</span></li>`;
    const files = (n) => `${n} ${n === 1 ? "file" : "files"}`;
    const extras = (P) => P.extra.length ? `<div class="pp-keep"><b>Only in your folder. Keep any?</b><ul>${P.extra.map((k) => `<li><label><input type="checkbox" data-pp-keep="${esc(k.rel)}" ${(P.yours || []).includes(k.rel) ? "checked" : ""} ${D.busy ? "disabled" : ""}><span class="selectable">${esc(k.rel)}</span></label><span class="ln-num">${esc(fmtBytes(k.size || 0))}</span></li>`).join("")}</ul></div>` : "";
    const delta = (P) => `
      <dl class="ln-facts">
        <div><dt>Your saves</dt><dd>${P.saves ? `${P.saves} ${P.saves === 1 ? "place" : "places"} copied first` : "None found to copy"}</dd></div>
        <div><dt>What changes</dt><dd>${[P.changed ? `${files(P.changed)} replaced` : "", P.added ? `${files(P.added)} new` : ""].filter(Boolean).join(", ")}, ${esc(fmtBytes(P.bytes || 0))}</dd></div>
        ${P.same ? `<div><dt>Already the same</dt><dd>${files(P.same)}, not downloaded</dd></div>` : ""}
        ${P.full && P.removed ? `<div><dt>Not in the new version</dt><dd>${files(P.removed)} moved aside</dd></div>` : ""}
        <div><dt>The files it replaces</dt><dd>Kept until you've played, or ${P.days} days</dd></div>
      </dl>
      ${P.keep.length ? `<div class="pp-keep"><b>Left as they are</b><ul>${P.keep.map(item).join("")}</ul></div>` : ""}
      ${extras(P)}
      <p class="sub">${P.full ? "Only the files listed above change" : "It's a patch: only its files go in, the rest of the folder isn't touched"}. Every file replaced or moved aside goes to <span class="selectable">${esc(P.old)}</span>, so Roll back puts the game back as it was.</p>`;
    const body = D.error && !P ? `<p class="error-text">${esc(D.error)}</p>` : !P ? lnInline("", "Looking at what's in the folder") : P.delta ? delta(P) : `
      <dl class="ln-facts">
        <div><dt>Your saves</dt><dd>${P.saves ? `${P.saves} ${P.saves === 1 ? "place" : "places"} copied first` : "None found to copy"}</dd></div>
        <div><dt>The folder from before</dt><dd>Kept until you've played, or ${P.days} days</dd></div>
      </dl>
      ${P.keep.length ? `<div class="pp-keep"><b>Carried into the new folder</b><ul>${P.keep.map(item).join("")}</ul></div>` : ""}
      ${extras(P)}
      <p class="sub">Everything else in <span class="selectable">${esc(P.folder)}</span> is replaced by the new build. Roll back puts the old folder back.</p>`;
    return `<div class="scrim" data-action="scrim"><div class="dialog" role="dialog" aria-modal="true" aria-label="Update ${esc(j.name)}">
      <div><h2>Update ${esc(j.name)}?</h2><p class="sub">${esc(j.title)}${j.source ? `, from ${esc(j.source)}` : ""}.</p></div>
      ${body}
      <div class="dialog-foot"><span class="error-text grow" role="alert">${P ? esc(D.error || "") : ""}</span><button class="btn quiet" data-action="dialog-cancel" ${D.busy ? "disabled" : ""}>Cancel</button>
      <button class="btn primary" data-action="pp-job-apply" ${D.busy || !P ? "disabled" : ""}>${P && P.delta ? "Update the game" : "Replace the game's folder"}</button></div></div></div>`;
  }
  const ppLogHTML = (D) => `<div class="scrim" data-action="scrim"><div class="dialog" role="dialog" aria-modal="true" aria-label="The update's log">
      <div><h2>${esc(D.name)}</h2><p class="sub">Every step of this update.</p></div><div class="log selectable" tabindex="0">${esc(D.text || "Nothing yet.")}</div>
      <div class="dialog-foot"><button class="btn primary" data-action="dialog-cancel">Close</button></div></div></div>`;
  async function jobOp(id, op) {
    const j = (S.plugJobs || []).find((x) => x.id === id);
    if (!j) return;
    try {
      if (op === "plan") {
        const D = S.dialog = { kind: "ppjob", job: id, plan: null, error: "", busy: false };
        renderOverlay();
        try { D.plan = await api.pluginJobPlan(id); } catch (e) { D.error = errText(e); }
        if (S.dialog === D) renderOverlay();
      } else if (op === "log") {
        S.dialog = { kind: "pplog", name: j.name, text: await api.pluginJobLog(id) };
        renderOverlay();
      } else if (op === "hand") {
        // The update centre's own panel shows what it changes; nothing changes until it's applied there.
        const U = S.dialog = { kind: "update", key: j.updKey, name: j.name, path: j.path, busy: true };
        renderOverlay(); focusOverlay();
        api.updatePlan(U.key, U.path).catch((e) => { U.busy = false; U.error = errText(e); renderOverlay(); });
      } else {
        const ask = { cancel: [`Cancel the update of ${j.name}?`, "What was downloaded is deleted. The game stays as it is.", "Cancel the update"],
          rollback: [`Roll ${j.name} back?`, "Your saves are copied first, then the folder from before the update goes back. The new build goes to the Recycle Bin.", "Roll back"],
          finish: [`Remove ${j.name}'s old folder?`, "It goes to the Recycle Bin, and Roll back won't be offered any more.", "Remove it"] }[op];
        if (ask && !await confirmDialog({ title: ask[0], body: ask[1], ok: ask[2], cancel: op === "cancel" ? "Keep going" : "Cancel", safe: true })) return;
        await api.pluginJobOp(id, op, []);
      }
    } catch (e) { toast(errText(e), "error"); }
  }
  // On a game's page: the first offer for it.
  function offerBanner(it) {
    const o = it.kind === "found" && (S.plugOffers || []).find((x) => x.key === "found:" + it.id);
    if (!o) return "";
    const facts = offerFacts(o);
    return banner("Update available", `${esc(o.title)}, from ${esc(o.source || o.pluginName)}${facts ? `, ${esc(facts)}` : ""}${o.status ? `. ${esc(o.status)}` : ""}`, offerKey(o, false) + `<button class="ln-button ln-button--ghost ln-button--sm" data-action="nav" data-view="updates">More</button>`, o.busy);
  }
  // Turning a script plugin on: what it asks is read and allowed first.
  async function allowPlugin(p) {
    if (!await confirmDialog({ title: `Let ${p.name} run?`, body: `It has a script, which Lootnest runs with a time limit and only these ways out. It asks to:`,
      list: p.asks || [], foot: p.key ? `Signed, key ${p.key}.` : "It isn't signed, so Lootnest can't tell who made it.", ok: "Allow and turn on", safe: true })) return false;
    await api.pluginGrant(p.id);
    return true;
  }
