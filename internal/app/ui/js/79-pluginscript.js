  // -------------------------------------------------------- script plugins --
  // A script plugin's pages (core/pluginscript.go, app/pluginscript.go). The
  // script describes a page as data: sections of rows, each with text, a
  // lamp, a switch, a select and buttons, and forms for its buttons.
  // Lootnest draws it with its own components and escapes every word, so a
  // plugin brings no HTML, no CSS and no script into the window. A key (a
  // field with `secret`) goes to Go, which keeps it; the script never reads
  // it.
  const PP = { views: {}, asked: {}, errors: {}, waiting: {}, n: 0 };
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
    if (r.form) ppOpenForm(w.plugin, w.page, r.form, r.act || String(r.form).split(":")[0], r.row || "");
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
      const p = ppShown();
      if (p && p.plugin === ev.id && p.id === ev.page && !(document.activeElement && document.activeElement.matches("[data-pp-select]"))) render();
    } else if (ev.type === "pluginChanged") {
      const p = ppShown();
      Object.keys(PP.asked).forEach((k) => { if (k.startsWith(ev.id + "/")) PP.asked[k] = false; });
      if (p && p.plugin === ev.id) askPluginView(p.plugin, p.id, true);
    } else if (ev.type === "pluginActed") {
      const w = PP.waiting[ev.req];
      if (!w) return;
      delete PP.waiting[ev.req];
      ppAnswer(w, ev.error ? { error: ev.error } : ev.result || {});
    }
  }
  // Turning a script plugin on: what it asks is read and allowed first.
  async function allowPlugin(p) {
    if (!await confirmDialog({ title: `Let ${p.name} run?`, body: `It has a script, which Lootnest runs with a time limit and only these ways out. It asks to:`,
      list: p.asks || [], foot: p.key ? `Signed, key ${p.key}.` : "It isn't signed, so Lootnest can't tell who made it.", ok: "Allow and turn on", safe: true })) return false;
    await api.pluginGrant(p.id);
    return true;
  }
