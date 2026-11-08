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
        ${(p.provides || []).length ? `<p class="pl-kinds">${p.provides.map((k) => `<span class="ln-tag">${PLUGIN_KINDS[k] || esc(k)}</span>`).join("")}</p>` : ""}
        ${p.problem ? `<p class="error-text">${esc(p.problem)}</p>` : ""}</div>
      <div class="pl-acts">${p.older && !safe ? b(`data-action="plugin-rollback" data-id="${esc(p.id)}"`, `Roll back to ${esc(p.older)}`) : ""}
        ${b(`data-action="plugin-remove" data-id="${esc(p.id)}"`, "Remove")}
        <span class="switch"><input type="checkbox" data-plugin-on="${esc(p.id)}" aria-label="${esc(p.name)} on" ${p.on && !safe ? "checked" : ""} ${safe || p.problem ? "disabled" : ""}><span></span></span></div></div>`).join("");
    const install = `<div class="bk-row pl-row"><div><b>Add a plugin</b><p>A plugin is a .zip. Lootnest checks every file in it first and installs it only if it holds nothing but data.</p>${S.pluginError ? `<p class="error-text">${esc(S.pluginError)}</p>` : ""}</div>
      <button class="ln-button ln-button--secondary ln-button--sm" data-action="plugin-install" ${safe ? "disabled" : ""}>Install from file</button></div>`;
    const note = safe ? "Lootnest is in safe mode, so every plugin is off. Restart it normally to use them." : "";
    return setSub("Installed", note, `<div class="bk-card">${install}${rows || `<div class="bk-row"><div><p>No plugins yet.</p></div></div>`}</div>`);
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
  function applyTheme() {
    const p = seated(), id = p && (p.theme || (parentOf(p) || {}).theme), t = id && (S.d.themes || {})[id];
    const next = t ? JSON.stringify(t.tokens) : "";
    if (next === themeOn) return;
    const root = document.documentElement.style;
    if (themeOn) Object.keys(JSON.parse(themeOn)).forEach((k) => root.removeProperty(k));
    if (t) Object.entries(t.tokens).forEach(([k, v]) => { if (k.startsWith("--")) root.setProperty(k, v); });
    themeOn = next;
  }
