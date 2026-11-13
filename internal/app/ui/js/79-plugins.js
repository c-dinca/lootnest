  // --------------------------------------------------------------- plugins --
  // Plugins, a Settings category (ps_plugins, core/plugins.go): data only,
  // installed from a .zip. A theme plugin's colours, checked in Go, show
  // while a profile that chose it is seated (applyTheme, from renderSeat).
  const PLUGIN_KINDS = { sources: "Adds games", saves: "Suggests save locations", metadata: "Fills in game details", theme: "Changes colours" };
  let pluginsAsked = false;
  function askPlugins(force) {
    if (pluginsAsked && !force) return;
    pluginsAsked = true;
    api.plugins().then((r) => { S.plugins = r; if (S.view === "settings") render(); }).catch(() => { pluginsAsked = false; });
  }
  function viewPlugins() {
    askPlugins();
    const P = S.plugins, safe = S.d.safe;
    if (!P) return lnInline("", "Reading the plugins");
    const b = (attrs, label) => `<button class="ln-button ln-button--ghost ln-button--sm" ${attrs}>${label}</button>`;
    const rows = P.plugins.map((p) => `<div class="bk-row pl-row"><div>
        <b>${esc(p.name)}${p.version ? ` <span class="set-opt"><span class="ln-num">${esc(p.version)}</span>${p.author ? `, by ${esc(p.author)}` : ""}</span>` : ""}</b>
        ${p.description ? `<p>${esc(p.description)}</p>` : ""}
        ${p.problem ? "" : `<p class="pl-key"><span>${p.key ? `Signed, key <span class="ln-num">${esc(p.key)}</span>` : "Not signed"}</span>${(p.provides || []).map((k) => `<span class="ln-tag">${PLUGIN_KINDS[k] || esc(k)}</span>`).join("")}</p>`}
        ${p.problem ? `<p class="error-text">${esc(p.problem)}</p>` : ""}</div>
      <div class="pl-acts">${p.older && !safe ? b(`data-action="plugin-rollback" data-id="${esc(p.id)}"`, `Roll back to ${esc(p.older)}`) : ""}
        ${b(`data-action="plugin-remove" data-id="${esc(p.id)}"`, "Remove")}
        <span class="switch"><input type="checkbox" data-plugin-on="${esc(p.id)}" aria-label="${esc(p.name)} on" ${p.on && !safe ? "checked" : ""} ${safe || p.problem ? "disabled" : ""}><span></span></span></div></div>`).join("");
    const install = `<div class="bk-row pl-row"><div><b>Install a plugin</b><p>A .zip of data only. Every file is checked first.</p>${S.pluginError ? `<p class="error-text">${esc(S.pluginError)}</p>` : ""}</div>
      <button class="ln-button ln-button--secondary ln-button--sm" data-action="plugin-install" ${safe ? "disabled" : ""}>Install from file</button></div>`;
    const note = safe ? "Safe mode: every plugin is off." : "";
    return setSub("Installed", note, `<div class="bk-card">${install}${rows || `<div class="bk-row"><div><b>No plugins yet</b></div></div>`}</div>`);
  }
  // An install's answer (ps_installPlugin): a version signed with another
  // key than the one pinned is asked about first, Cancel by default, and
  // only the key shown installs.
  async function pluginInstalled(path, r, newKey) {
    if (r.keyChange) {
      const k = r.keyChange;
      if (!await confirmDialog({ title: `A new key for ${k.name}`, body: "Signed with a different key than the one you installed: a new author, or someone else's copy.",
        facts: [["The key you installed", k.pinned], ["This version's key", k.key]], ok: "Install with the new key", safe: true })) return;
      return pluginInstalled(path, await api.installPluginNewKey(path, k.key), true);
    }
    const m = r.plugin;
    await pluginsDone(`Installed ${m.name} ${m.version}.${m.pinnedNow ? ` It's signed now, with key ${m.key}, and later versions must have the same key.` : newKey ? ` Later versions must have its new key, ${m.key}.` : ""}`);
  }
  async function pluginsDone(text) {
    S.pluginError = "";
    S.details = {};
    askPlugins(true);
    await refresh();
    if (text) toast(text);
  }
  // The seated profile's theme (a run without one wears its person's) on
  // :root, as Go sent it; none is Lootnest's own.
  let themeOn = "";
  // Lootnest's own themes, beside any a plugin brings; chosen per profile
  // like those. Their ids start "ln-", which plugins can't use. Arcade is
  // the indigo look of 1.93 to 1.96; High contrast is black with white
  // text and solid panels.
  const BUILTIN_THEMES = {
    "ln-arcade": { name: "Arcade", tokens: {
      "--bg-well": "oklch(0.175 0.036 282)", "--bg-ground": "oklch(0.205 0.040 282)", "--bg-surface": "oklch(0.250 0.044 282)",
      "--bg-raised": "oklch(0.295 0.046 282)", "--bg-active": "oklch(0.345 0.048 282)", "--shell": "oklch(0.165 0.034 282)",
      "--fg-2": "oklch(0.830 0.024 280)", "--fg-3": "oklch(0.730 0.030 280)",
      "--edge-1": "oklch(0.80 0.06 282 / 0.12)", "--edge-2": "oklch(0.80 0.06 282 / 0.20)", "--edge-3": "oklch(0.85 0.06 282 / 0.38)",
      "--glass": "oklch(0.205 0.040 282 / 0.5)", "--glass-strong": "oklch(0.205 0.040 282 / 0.66)", "--glass-overlay": "oklch(0.250 0.044 282 / 0.95)",
      "--glass-panel-page": "oklch(0.250 0.044 282 / 0.6)" } },
    "ln-contrast": { name: "High contrast", tokens: {
      "--bg-well": "#000000", "--bg-ground": "#000000", "--bg-surface": "#0b0b0d", "--bg-raised": "#18181c", "--bg-active": "#2a2a30", "--shell": "#000000",
      "--fg-1": "#ffffff", "--fg-2": "#e8ebf0", "--fg-3": "#cdd3db",
      "--edge-1": "rgb(255 255 255 / 0.32)", "--edge-2": "rgb(255 255 255 / 0.5)", "--edge-3": "rgb(255 255 255 / 0.75)",
      "--glass": "#000000", "--glass-strong": "#000000", "--glass-overlay": "#0b0b0d", "--glass-panel-page": "#0b0b0d", "--glass-edge": "rgb(255 255 255 / 0.4)",
      "--blur-glass": "none", "--room-dim": "blur(36px) brightness(0.35)", "--room-dim-page": "blur(48px) brightness(0.25)" } },
  };
  const allThemes = () => ({ ...BUILTIN_THEMES, ...(S.d.themes || {}) });
  function applyTheme() {
    const p = seated(), id = p && (p.theme || (parentOf(p) || {}).theme), t = id && allThemes()[id];
    const next = t ? JSON.stringify(t.tokens) : "";
    if (next === themeOn) return;
    const root = document.documentElement.style;
    if (themeOn) Object.keys(JSON.parse(themeOn)).forEach((k) => root.removeProperty(k));
    if (t) Object.entries(t.tokens).forEach(([k, v]) => { if (k.startsWith("--")) root.setProperty(k, v); });
    themeOn = next;
  }
