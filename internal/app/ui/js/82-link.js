  // -------------------------------------------------------------- tv link --
  // Settings, TV (Lootnest Link M2, app/link.go): Lootnest installs the
  // streaming engine (Apollo), sets it up and pairs TVs, so the engine's own
  // pages never open. Everything slow comes back as an event.
  let linkAsked = false;
  const LINK_STEPS = { key: "Unlocking the TV's key", copy: "Connecting to the TV", install: "Installing Lootnest on the TV", download: "Downloading", install: "Installing. Allow it in Windows' prompt", start: "Starting the streaming engine", configure: "Setting it up", pair: "Pairing the TV", power: "Waiting for Windows' prompt" };
  function askLink() {
    if (linkAsked) return;
    linkAsked = true;
    api.linkStatus().catch(() => {});
  }
  function linkEvent(ev) {
    const L = S.link = S.link || {};
    if (ev.type === "link") { L.status = ev.status; L.asking = false; if (!ev.status.busy) L.step = null; }
    else if (ev.type === "linkStep") L.step = { step: ev.step, done: ev.done, total: ev.total };
    else if (ev.type === "linkDone") { L.step = null; L.error = ev.error || ""; if (ev.ok) toast("TV streaming is set up. Pair a TV next.", "success"); }
    else if (ev.type === "linkTVs") { const D = S.dialog; if (D && D.kind === "lgtv") { D.tvs = ev.tvs || []; D.finding = false; if (!D.ip && D.tvs.length) { D.ip = D.tvs[0].ip; D.name = D.tvs[0].name; } renderOverlay(); } return; }
    else if (ev.type === "linkTVDone") {
      L.step = null;
      const D = S.dialog;
      if (D && D.kind === "lgtv") { D.busy = false; D.error = ev.error || ""; D.done = !!ev.ok; renderOverlay(); }
      if (ev.ok) toast(`Lootnest is on ${ev.name}. Pick it up from the TV's home screen.`, "success");
    }
    else if (ev.type === "linkPaired") {
      L.pairing = false; L.pairError = ev.error || "";
      if (ev.ok) { L.pin = ""; toast(`${ev.name} can play now. Choose Lootnest on it in Moonlight.`, "success"); }
    }
    if (S.view === "settings" && S.setTab === "tv") render();
    if (S.dialog && S.dialog.kind === "lgtv" && ev.type === "linkStep") renderOverlay();
  }
  // Add an LG TV: developer mode on the TV, then Lootnest installs and pairs.
  function openLGTV(ip, name) {
    S.dialog = { kind: "lgtv", tvs: null, finding: true, ip: ip || "", name: name || "", busy: false, error: "", done: false };
    renderOverlay(); focusOverlay();
    api.linkFindTVs().catch(() => { S.dialog.finding = false; S.dialog.tvs = []; renderOverlay(); });
  }
  function viewLGTV() {
    const D = S.dialog, L = S.link || {};
    const close = `<span class="grow"></span><button class="ln-button ln-button--ghost ln-button--sm" data-action="dialog-cancel">${D.done ? "Done" : "Cancel"}</button>`;
    const shell = (body, foot) => drawer({ label: "Add an LG TV", title: "Add an LG TV", sub: "Lootnest installs its TV app and pairs it", body, foot });
    if (D.done) return shell(`<p>Lootnest is on the TV and paired. On the TV, open Lootnest from the home screen.</p>
      <p>Lootnest keeps the TV's developer mode from running out while this PC is on.</p>`, close);
    if (D.busy) {
      const st = L.step || {}, pct = st.total > 0 ? Math.round((st.done / st.total) * 100) : -1;
      return shell(lnInline("", `${esc(LINK_STEPS[st.step] || "Working")}${pct >= 0 ? ` <span class="ln-num">${pct}%</span>` : ""}`), close);
    }
    const tvs = D.tvs || [];
    const found = D.finding ? lnInline("", "Looking for LG TVs on this network")
      : tvs.length ? `<div class="lg-tvs" role="radiogroup" aria-label="TVs found">${tvs.map((t) => `<label class="lg-tv"><input type="radio" name="lg-pick" data-ip="${esc(t.ip)}" data-name="${esc(t.name)}" ${D.ip === t.ip ? "checked" : ""}><span><b>${esc(t.name)}</b><span class="ln-num">${esc(t.ip)}</span></span></label>`).join("")}</div>`
      : `<p>No LG TV answered. Check it's on, then type its address from its network settings.</p>`;
    return shell(`<section class="lg-sec"><h3>On the TV</h3><ol class="steps">
        <li>Open <b>LG Content Store</b> and install <b>Developer Mode</b>.</li>
        <li>Open it and sign in with an LG developer account, free at developer.lge.com.</li>
        <li>Turn on <b>Dev Mode Status</b>. The TV restarts.</li>
        <li>Open Developer Mode again and turn on <b>Key Server</b>. Leave it open.</li></ol></section>
      <section class="lg-sec"><h3>The TV</h3>${found}
        <div class="lg-row"><input id="lg-ip" class="ln-input ln-num" inputmode="decimal" autocomplete="off" placeholder="192.168.1.20" aria-label="The TV's address" value="${esc(D.ip)}">
          <button class="ln-button ln-button--ghost ln-button--sm" data-action="lg-find"${D.finding ? " disabled" : ""}>Look again</button></div>
        <input id="lg-name" class="ln-input" maxlength="40" autocomplete="off" placeholder="Living room TV" aria-label="The TV's name" value="${esc(D.name)}"></section>
      <section class="lg-sec"><h3>Passphrase</h3><p>Developer Mode shows it under Key Server.</p>
        <input id="lg-pass" class="ln-input lg-pass" maxlength="12" autocomplete="off" spellcheck="false" placeholder="A1B2C3" aria-label="Passphrase"></section>
      ${D.error ? `<p class="error-text" role="alert">${esc(D.error)}</p>` : ""}`,
      `<button class="ln-button ln-button--primary ln-button--sm" data-action="lg-install">Install</button>${close}`);
  }
  function linkBusyRow(L) {
    const st = L.step || { step: L.status && L.status.busy };
    const pct = st.total > 0 ? Math.round((st.done / st.total) * 100) : -1;
    return setRow(`<span class="ln-spinner" aria-hidden="true"></span> ${esc(LINK_STEPS[st.step] || "Working")}`, "", pct >= 0 ? `<span class="ln-num link-pct">${pct}%</span>` : "");
  }
  function viewLink() {
    askLink();
    const L = S.link = S.link || {}, st = L.status;
    const card = (rows) => `<div class="bk-card">${rows}</div>`;
    const key = (attrs, label, kind = "secondary") => `<button class="ln-button ln-button--${kind} ln-button--sm" ${attrs}>${label}</button>`;
    if (!st) return lnInline("", "Asking the streaming engine");
    const busy = !!(st.busy || L.step);
    if (!st.pinned) return setSub("Play on a TV", "", card(setRow("Not in this version yet", "TV streaming comes with a later version of Lootnest.")));
    if (!st.set || (busy && !st.info)) {
      const line = st.installed
        ? "Apollo, the streaming engine, is already on this PC. Lootnest takes over its sign-in; its apps and paired devices stay."
        : `Lootnest installs Apollo ${esc(st.version)}, a free, open-source streaming engine, and sets it up. A TV with Moonlight then plays your games here, in Lootnest's TV mode. Windows asks once.`;
      return setSub("Play on a TV", "", card(
        (busy ? linkBusyRow(L) : setRow("Set up TV streaming", line, key('data-action="link-setup"', st.installed ? "Use it" : "Set up", "primary")))
        + (L.error ? `<div class="bk-row"><p class="error-text" role="alert">${esc(L.error)}</p></div>` : "")));
    }
    const info = st.info, on = !st.off && st.running;
    const lamp = (ok, text) => `<span class="ctl-lamp${ok ? " is-on" : ""}">${text}</span>`;
    const engine = setRow(`Apollo${info && info.version ? ` <span class="ln-num">${esc(info.version)}</span>` : ""}`,
      st.off ? "Turned off" : !st.running ? "Not running. Turn it on, or restart the PC." : st.problem ? esc(st.problem) : info ? `Shows on the TV as ${esc(info.name || "Lootnest")}` : "",
      `<span class="link-acts">${lamp(on && !st.problem, on ? (st.problem ? "Not answering" : "On") : "Off")}${busy ? "" : key(`data-action="link-power" data-on="${on ? "0" : "1"}"`, on ? "Turn off" : "Turn on")}</span>`);
    const rows = [engine];
    if (busy) rows.push(linkBusyRow(L));
    if (info && !info.virtualDisplay) rows.push(setRow("Virtual display", "Not working, so the stream uses this PC's own screen. Restart the PC; if it stays, set TV streaming up again."));
    if (info && !info.lootnestApp) rows.push(setRow("TV mode isn't in the engine", "", key('data-action="link-setup"', "Add it")));
    if (info && info.runningApp) rows.push(setRow("Streaming now", "", key('data-action="link-end"', "End the stream")));
    // An LG TV Lootnest installed is one row with its pairing, not two.
    const lgNames = new Set((st.lg || []).map((t) => t.name));
    const tvs = (st.tvs || []).filter((t) => !lgNames.has(t.name));
    const tvRows = tvs.map((t) => setRow(esc(t.name || "TV"), t.connected ? "Streaming now" : "Paired", key(`data-action="link-unpair" data-uuid="${esc(t.uuid)}" data-name="${esc(t.name || "TV")}"`, "Forget", "ghost"))).join("");
    const lgRows = (st.lg || []).map((t) => {
      const c = (st.tvs || []).find((x) => x.name === t.name);
      const state = c ? (c.connected ? "Streaming now" : "Paired") : "Not paired";
      return setRow(esc(t.name), t.problem ? `<p class="error-text">${esc(t.problem)}</p>` : `${state}. Lootnest for TV ${esc(t.version || "")}, developer mode extended ${t.extended ? esc(ago(t.extended) || "just now") : "never"}.`,
      `<span class="link-acts">${key(`data-action="lg-again" data-ip="${esc(t.ip)}" data-name="${esc(t.name)}"`, "Install again", "ghost")}${key(`data-action="lg-forget" data-ip="${esc(t.ip)}" data-name="${esc(t.name)}"${c ? ` data-uuid="${esc(c.uuid)}"` : ""}`, "Forget", "ghost")}</span>`);
    }).join("");
    const addLG = setRow("Add an LG TV", st.tvApp ? "Lootnest installs its TV app on the TV and pairs it. The TV needs developer mode, which takes a few minutes once." : "Comes with a later version of Lootnest.",
      key(`data-action="lg-add"${st.tvApp && on ? "" : " disabled"}`, "Add an LG TV", "primary"));
    const pair = `<div class="bk-row set-wide"><div class="set-wide-head"><b>Another TV with Moonlight</b></div>
      <p>On the TV, open Moonlight and choose ${esc((info && info.name) || "this PC")}. Enter the PIN it shows.</p>
      <div class="link-pair"><input id="link-pin" class="ln-input ln-num" inputmode="numeric" pattern="[0-9]{4}" maxlength="4" autocomplete="off" placeholder="PIN" aria-label="PIN from the TV" value="${esc(L.pin || "")}" ${L.pairing || !on ? "disabled" : ""}>
        <input id="link-name" class="ln-input" maxlength="40" autocomplete="off" placeholder="Living room TV" aria-label="The TV's name" value="${esc(L.tvName || "")}" ${L.pairing || !on ? "disabled" : ""}>
        ${L.pairing ? `<button class="ln-button ln-button--secondary ln-button--sm" aria-busy="true" disabled><span class="ln-spinner" aria-hidden="true"></span>Pairing</button>` : key(`data-action="link-pair"${on ? "" : " disabled"}`, "Pair")}</div>
      ${L.pairError ? `<p class="error-text" role="alert">${esc(L.pairError)}</p>` : ""}</div>`;
    return setSub("Streaming engine", "", card(rows.join("")))
      + setSub("TVs", "", card(lgRows + tvRows + addLG + pair))
      + setSub("Picture", "", card(setRow("In Moonlight's settings on the TV", "1080p at 60 fps and 20 to 30 Mbps, or 4K at 60 fps and 60 Mbps when the TV is on a cable. Lower the bitrate if the picture stutters.")))
      + setSub("Problems", "", card(setRow("Save a report", "The streaming engine's log and Lootnest's, without the engine's password, in one file to attach to a GitHub issue.", key('data-action="link-logs"', "Save"))));
  }
