  // -------------------------------------------------------------- tv link --
  // Settings, TV (Lootnest Link M2, app/link.go): Lootnest installs the
  // streaming engine (Apollo), sets it up and pairs TVs, so the engine's own
  // pages never open. Everything slow comes back as an event.
  let linkAsked = false;
  const LINK_STEPS = { download: "Downloading the streaming engine", install: "Installing. Allow it in Windows' prompt", start: "Starting the streaming engine", configure: "Setting it up", pair: "Pairing the TV", power: "Waiting for Windows' prompt" };
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
    else if (ev.type === "linkPaired") {
      L.pairing = false; L.pairError = ev.error || "";
      if (ev.ok) { L.pin = ""; toast(`${ev.name} can play now. Choose Lootnest on it in Moonlight.`, "success"); }
    }
    if (S.view === "settings" && S.setTab === "tv") render();
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
    const tvs = st.tvs || [];
    const tvRows = tvs.map((t) => setRow(esc(t.name || "TV"), t.connected ? "Streaming now" : "Paired", key(`data-action="link-unpair" data-uuid="${esc(t.uuid)}" data-name="${esc(t.name || "TV")}"`, "Forget", "ghost"))).join("");
    const pair = `<div class="bk-row set-wide"><div class="set-wide-head"><b>Add a TV</b></div>
      <p>On the TV, open Moonlight and choose ${esc((info && info.name) || "this PC")}. Enter the PIN it shows.</p>
      <div class="link-pair"><input id="link-pin" class="ln-input ln-num" inputmode="numeric" pattern="[0-9]{4}" maxlength="4" autocomplete="off" placeholder="PIN" aria-label="PIN from the TV" value="${esc(L.pin || "")}" ${L.pairing || !on ? "disabled" : ""}>
        <input id="link-name" class="ln-input" maxlength="40" autocomplete="off" placeholder="Living room TV" aria-label="The TV's name" value="${esc(L.tvName || "")}" ${L.pairing || !on ? "disabled" : ""}>
        ${L.pairing ? `<button class="ln-button ln-button--primary ln-button--sm" aria-busy="true" disabled><span class="ln-spinner" aria-hidden="true"></span>Pairing</button>` : key(`data-action="link-pair"${on ? "" : " disabled"}`, "Pair", "primary")}</div>
      ${L.pairError ? `<p class="error-text" role="alert">${esc(L.pairError)}</p>` : ""}</div>`;
    return setSub("Streaming engine", "", card(rows.join("")))
      + setSub("TVs", "", card(tvRows + pair))
      + setSub("Picture", "", card(setRow("In Moonlight's settings on the TV", "1080p at 60 fps and 20 to 30 Mbps, or 4K at 60 fps and 60 Mbps when the TV is on a cable. Lower the bitrate if the picture stutters.")))
      + setSub("Problems", "", card(setRow("Save a report", "The streaming engine's log and Lootnest's, without the engine's password, in one file to attach to a GitHub issue.", key('data-action="link-logs"', "Save"))));
  }
