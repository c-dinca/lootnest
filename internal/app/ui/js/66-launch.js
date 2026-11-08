  // ------------------------------------------------------- launch options --
  // How Lootnest starts a game, and the apps it closes and programs it starts
  // around play (core/launchopts.go). Only programs the user names; nothing
  // is put into the game.
  const launchHow = (it) => (it.kind === "found" ? (it.f.appId ? "steam" : it.f.uri ? "launcher" : "exe") : it.g.launch.type === "uri" ? "launcher" : it.g.launch.type);
  function launchRow(it) {
    return gpRow(`data-action="launch-options" ${itemAttr(it)}`, GI.launch, "Launch options", "");
  }
  function openLaunch(it) {
    const o = (cfg().launchOpts || {})[recentKey(it)] || {}, copy = (l) => (l || []).map((x) => ({ ...x }));
    S.dialog = { kind: "launch", ik: it.kind, iid: it.id, args: (it.kind === "found" ? it.f.args : it.g.launch.args) || "", apps: null, picking: false, busy: false, error: "",
      o: { workDir: o.workDir || "", runAsAdmin: !!o.runAsAdmin, closeApps: copy(o.closeApps), startBefore: copy(o.startBefore), startAfter: copy(o.startAfter),
        reopenClosed: (o.closeApps || []).length ? !!o.reopenClosed : true } };
  }
  function viewLaunch() {
    const D = S.dialog, it = dialogItem(D);
    if (!it) return "";
    const O = D.o, how = launchHow(it), appId = it.kind === "found" ? it.f.appId : it.g.launch.appId;
    const base = (p) => p.split(/[\\/]/).pop();
    const rm = (list, i, what) => `<button class="ln-button ln-button--ghost ln-button--icon ln-button--sm" data-action="lo-remove" data-list="${list}" data-i="${i}" aria-label="Remove ${esc(what)}">${GI.x}</button>`;
    const btn = (attrs, label, cls = "ln-button--secondary") => `<button class="ln-button ${cls} ln-button--sm" ${attrs}>${label}</button>`;
    const progs = (list) => `${O[list].length ? `<ul class="lo-list">${O[list].map((p, i) => `<li class="lo-item"><div class="lo-main"><b>${esc(base(p.path))}</b><span class="gpage-hint selectable">${esc(p.path)}</span>
        <input class="ln-input" type="text" data-lo="args" data-list="${list}" data-i="${i}" value="${esc(p.args || "")}" placeholder="Options, if it needs any" spellcheck="false" aria-label="Options for ${esc(base(p.path))}"></div>${rm(list, i, base(p.path))}</li>`).join("")}</ul>` : ""}
      <div class="gpage-actions">${btn(`data-action="lo-prog" data-list="${list}"`, "Add a program to start")}</div>`;
    const args = `<div class="field"><label for="lo-args">Options</label><input id="lo-args" class="ln-input" type="text" data-lo="gameArgs" value="${esc(D.args)}" placeholder="${how === "steam" ? "-novid -high" : "-windowed"}" spellcheck="false">`;
    const start = how === "exe" ? `${args}</div>
        <div class="field"><span class="label">Start in</span><div class="lo-pick"><span class="${O.workDir ? "selectable" : "gpage-hint"}">${esc(O.workDir || "The game's own folder")}</span>
          ${btn('data-action="lo-dir"', O.workDir ? "Change" : "Choose a folder", "ln-button--ghost")}${O.workDir ? btn('data-action="lo-dir-clear"', "Use the game's folder", "ln-button--ghost") : ""}</div></div>
        <label class="toggle"><span><b>Run as administrator</b><p>Gives the game full rights over this PC.</p></span>
          <span class="switch"><input type="checkbox" data-lo="admin" ${O.runAsAdmin ? "checked" : ""}><span></span></span></label>`
      : how === "steam" ? `${args}</div>
        <div class="gpage-actions">${btn(`data-action="lo-steam" data-app="${esc(appId)}"`, "Open its properties in Steam")}</div>`
      : `<p class="gpage-hint">Its launcher starts this game. Set options there.</p>`;
    const own = String((it.kind === "found" ? it.f.processName : it.g.processName) || "").toLowerCase().replace(/\.exe$/, "");
    const have = new Set(O.closeApps.map((c) => c.name.toLowerCase()));
    const offer = (D.apps || []).filter((a) => !have.has(a.name.toLowerCase()) && a.name.toLowerCase().replace(/\.exe$/, "") !== own);
    const picker = !D.picking ? "" : !D.apps ? lnInline("", "Looking for open apps")
      : offer.length ? `<div class="ln-list lo-apps" role="group" aria-label="Open apps">${offer.map((a) => gpRow(`data-action="lo-app-pick" data-name="${esc(a.name)}"`, GI.plus, esc(a.name), `<span class="selectable">${esc(a.path)}</span>`)).join("")}</div>`
      : `<p class="gpage-hint">No other app has a window open</p>`;
    const close = `${O.closeApps.length ? `<ul class="lo-list">${O.closeApps.map((c, i) => `<li class="lo-item"><div class="lo-main"><b>${esc(c.name)}</b></div>
        <label class="lo-force"><span>Force</span><span class="switch"><input type="checkbox" data-lo="force" data-i="${i}" ${c.force ? "checked" : ""} aria-label="Force ${esc(c.name)} to close"><span></span></span></label>${rm("closeApps", i, c.name)}</li>`).join("")}</ul>
        <p class="gpage-hint">Force ends an app after 10 seconds without asking it, losing anything unsaved.</p>` : ""}
      <div class="lo-add">${btn('data-action="lo-apps"', D.picking ? "Hide open apps" : "Add an open app")}<input id="lo-app" class="ln-input" type="text" placeholder="Or type a name, such as Discord.exe" spellcheck="false" aria-label="An app's name">${btn('data-action="lo-app-add"', "Add", "ln-button--ghost")}</div>
      ${picker}`;
    const around = it.kind === "found" && !own ? `<div class="gpage-sub lo-sec">${gpHead("Before and after you play")}<p class="gpage-hint">Not for this game: Lootnest can't tell when it closes.</p></div>`
      : `<div class="gpage-sub lo-sec">${gpHead("Before you play")}${close}
        ${progs("startBefore")}</div>
      <div class="gpage-sub lo-sec">${gpHead("After you play")}${progs("startAfter")}
        ${O.closeApps.length ? `<label class="toggle"><span><b>Open the apps it closed again</b></span>
          <span class="switch"><input type="checkbox" data-lo="reopen" ${O.reopenClosed ? "checked" : ""}><span></span></span></label>` : ""}</div>`;
    const body = `${S.d.safe ? `<p class="gpage-hint">Safe mode: nothing runs before or after play.</p>` : ""}
      <div class="gpage-sub lo-sec">${gpHead("How it starts")}${start}</div>
      ${around}
      ${D.error ? lnInline("error", esc(D.error)) : ""}`;
    return drawer({ label: `Launch options for ${esc(it.name)}`, title: "Launch options", sub: esc(it.name), body,
      foot: `<span class="grow"></span><button class="ln-button ln-button--ghost ln-button--sm" data-action="dialog-cancel" ${D.busy ? "disabled" : ""}>Cancel</button>
        <button class="ln-button ln-button--primary ln-button--sm" data-action="lo-save" ${D.busy ? "disabled" : ""}>Save</button>` });
  }
