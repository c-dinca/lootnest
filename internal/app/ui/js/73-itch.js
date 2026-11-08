  // --------------------------------------------------------------- itch.io --
  // The games in the user's itch.io library (app/itchio.go), with the API key
  // they made: listed under Not installed, installed from a drawer, and on
  // a game Lootnest installed, a check for a newer upload that goes in
  // through the update panel. S.itch: the account's answer, the records of
  // installed games, the updates found, and the download running.
  const itchOn = () => !!cfg().settings.itchUser;
  const ITCH_KEYS = "https://itch.io/user/settings/api-keys";
  const itchStageText = (R) => ({
    download: "Downloading",
    extract: "Unpacking into its folder",
    install: "The installer is running. Finish it, then come back here.",
    check: "Looking for the game it installed",
  }[R.stage] || "Getting ready");
  const itchPct = (R) => (R.total ? Math.min(100, Math.round((R.done / R.total) * 100)) : 0);
  const itchBytes = (R) => (R.stage === "download" && R.total ? `${fmtBytes(R.done)} of ${fmtBytes(R.total)}` : "");
  // Only a download has an amount to fill; the other stages show a moving
  // bar. The status line is announced when the stage changes; the bytes and
  // the bar are patched in place (itchPatch), so focus stays where it is.
  // text: the status line for a stage (GOG's, 74-gog.js, has its own).
  const itchMeter = (R, act = "", text = itchStageText) => {
    const busy = R.stage !== "download";
    return `<div class="itch-run mover" data-key="${esc(R.key)}"><div class="meter${busy ? " busy" : ""}" role="progressbar" aria-label="${esc(text(R))}" aria-valuemin="0" aria-valuemax="100"${busy ? "><i></i>" : ` aria-valuenow="${itchPct(R)}"><i style="width:${itchPct(R)}%"></i>`}</div>
      <p class="hint"><span role="status">${esc(text(R))}</span> <span class="ln-num itch-bytes">${esc(itchBytes(R))}</span>${act}</p></div>`;
  };
  const itchPatch = (n, R) => {
    const m = n.querySelector(".meter"), b = n.querySelector(".itch-bytes");
    if (m && m.firstElementChild) { m.firstElementChild.style.width = itchPct(R) + "%"; m.setAttribute("aria-valuenow", itchPct(R)); }
    if (b) b.textContent = itchBytes(R);
  };

  // Settings, Online: connect with a key, or who it's connected as.
  function itchRows() {
    const I = S.itch, s = cfg().settings;
    const b = (attrs, label, kind = "secondary") => `<button class="ln-button ln-button--${kind} ln-button--sm" ${attrs}>${label}</button>`;
    const refresh = I.refreshing ? `<button class="ln-button ln-button--secondary ln-button--sm" aria-busy="true" disabled><span class="ln-spinner" aria-hidden="true"></span>Refreshing</button>` : b('data-action="itch-refresh"', "Refresh library");
    if (s.itchUser) return `<div class="bk-row"><div><b>Connected as ${esc(s.itchUser)}</b>
        <p>Your itch.io games that aren't installed show in the Library under Not installed. The key is encrypted on this PC and sent only to itch.io.</p>${I.error ? `<p class="error-text">${esc(I.error)}</p>` : ""}</div>
      <div class="set-itch-acts">${refresh}${b('data-action="itch-disconnect"', "Disconnect", "ghost")}</div></div>`;
    return `<div class="bk-row set-wide"><div><label for="s-itchkey"><b>API key</b></label>
        <p>Make one on itch.io, then paste it here.</p>
        <p><button class="gpage-link" data-action="itch-keys">Make a key on itch.io</button></p>${I.error ? `<p class="error-text">${esc(I.error)}</p>` : ""}</div>
      <div class="set-itch-acts"><input id="s-itchkey" class="ln-input" type="password" autocomplete="off" spellcheck="false" placeholder="Paste your key" ${I.busy ? "disabled" : ""}>
        ${I.busy ? `<button class="ln-button ln-button--secondary ln-button--sm" aria-busy="true" disabled><span class="ln-spinner" aria-hidden="true"></span>Connecting</button>` : b('data-action="itch-connect"', "Connect")}</div></div>`;
  }

  function openItch(it) {
    const o = it.o || {};
    const D = S.dialog = { kind: "itch", key: it.id, gameId: +o.storeId, name: it.name, uploads: null, pick: 0, dir: "", target: "", free: -1, dataDrive: "", dataFree: -1, error: "" };
    renderOverlay(); focusOverlay();
    api.itchUploads(D.gameId, "").catch((e) => { D.error = errText(e); renderOverlay(); });
  }
  function viewItchInstall() {
    const D = S.dialog, R = S.itch.run && S.itch.run.key === D.key ? S.itch.run : null;
    const shell = (body, foot) => drawer({ label: `Install ${esc(D.name)}`, title: `Install ${esc(D.name)}`, sub: "From your itch.io library", body, foot });
    const close = `<span class="grow"></span><button class="ln-button ln-button--ghost ln-button--sm" data-action="dialog-cancel">Close</button>`;
    if (R) return shell(itchMeter(R) + `<p class="gpage-hint">You can close this panel; Lootnest keeps going and tells you when the game is in your Library.</p>`,
      `${R.stage === "download" ? `<button class="ln-button ln-button--secondary ln-button--sm" data-action="itch-cancel">Cancel the download</button>` : ""}${close}`);
    if (D.error && !D.uploads) return shell(lnInline("error", esc(D.error)), close);
    if (!D.uploads) return shell(lnInline("", "Asking itch.io for the game's downloads"), close);
    if (!D.uploads.length) return shell(`<p>itch.io has no download of ${esc(D.name)} for Windows that Lootnest can install: a .zip, .7z or .rar, or an installer.</p>`, close);
    const up = D.uploads.find((u) => u.id === D.pick) || D.uploads[0];
    // Room as Go checks it: the download on Lootnest's drive, and for a game
    // Lootnest puts in a folder, the download and the game on that folder's.
    const installer = up.kind === "installer", drive = (D.dir || "").slice(0, 2).toUpperCase(), size = up.size || 0;
    const room = installer ? [[D.dataDrive, D.dataFree, size]] : drive === D.dataDrive ? [[drive, D.free, size * 3]] : [[drive, D.free, size * 3], [D.dataDrive, D.dataFree, size]];
    const short = room.find(([, free, need]) => free >= 0 && free < need + 64 * 2 ** 20);
    const low = !!short;
    const label = (u) => `${u.displayName || u.filename}${u.demo ? " (demo)" : ""}${u.userVersion ? ", " + u.userVersion : ""}`;
    const which = D.uploads.length > 1
      ? `<select id="itch-up" class="ln-input" aria-label="Download">${D.uploads.map((u) => `<option value="${u.id}" ${u.id === up.id ? "selected" : ""}>${esc(label(u))}</option>`).join("")}</select>`
      : `<p><b>${esc(label(up))}</b></p>`;
    const folder = !installer ? `<div class="gpage-sub lo-sec">${gpHead("Folder")}
        <div class="lo-pick"><span class="selectable">${esc(D.target || D.dir)}</span><button class="ln-button ln-button--ghost ln-button--sm" data-action="itch-dir">Change</button></div>
        <p class="gpage-hint">Lootnest makes this folder for the game and never installs into one that has files in it.</p></div>`
      : `<div class="gpage-sub lo-sec">${gpHead("Installer")}<p class="gpage-hint">Lootnest runs the game's own installer, which asks where the game goes. Once it closes, Lootnest finds the game and adds it to your Library.</p></div>`;
    const body = `<div class="gpage-sub lo-sec itch-dl">${gpHead("Download")}${which}
        <dl class="ln-facts"><div><dt>Size</dt><dd class="ln-num">${esc(fmtBytes(size))}</dd></div>
          ${room.map(([drv, free], i) => `<div><dt>Free on ${esc(drv)}${room.length > 1 && i ? ", for the download" : ""}</dt><dd class="ln-num">${free >= 0 ? esc(fmtBytes(free)) : "Not known"}</dd></div>`).join("")}</dl>
        ${low ? `<p class="find-low">There isn't enough room on ${esc(short[0])}: this needs about ${esc(fmtBytes(short[2]))}.</p>` : ""}</div>
      ${folder}${D.error ? lnInline("error", esc(D.error)) : ""}`;
    return shell(body, `<span class="grow"></span><button class="ln-button ln-button--ghost ln-button--sm" data-action="dialog-cancel">Cancel</button>
      <button class="ln-button ln-button--primary ln-button--sm" data-action="itch-install" data-upload="${up.id}" ${low || D.busy ? "disabled" : ""}>Install</button>`);
  }

  // What itch.io and GOG (74-gog.js) share: the record of a Library game
  // Lootnest installed from the store (its own, or the one of the game it
  // was before profiles were turned on), the note on its page, and the
  // events of a download. st.s names the store's part of S, its actions and
  // its calls (S.itch, "itch-check", api.itchCheck).
  const ITCH = {
    s: "itch", label: "itch.io", on: itchOn, text: itchStageText,
    cancelled: (name) => `The download of ${name} stopped. What it had downloaded is deleted.`,
    installed: (key) => api.itchCheck(key, false).catch(() => {}),
  };
  function storeRecKey(st, it) {
    const R = S[st.s].recs;
    return it.kind === "found" ? (R[it.id] ? it.id : "") : Object.keys(R).find((k) => R[k].game === it.id) || "";
  }
  // A game Lootnest installed from the store: its version, the last check,
  // and an update when there's one.
  function storeNote(st, it) {
    const I = S[st.s], k = storeRecKey(st, it), rec = k && I.recs[k];
    if (!rec) return "";
    if (!I.seen.has(k)) { I.seen.add(k); api[st.s + "Check"](k, false).catch(() => {}); }
    const R = I.run && I.run.key === k ? I.run : null, newer = I.updates.find((u) => u.key === k);
    if (R) return `<div class="gpage-note itch-note">${GI.load}${itchMeter(R, R.stage === "download" ? ` <button class="gpage-link" data-action="${st.s}-cancel">Cancel</button>` : "", st.text)}</div>`;
    const checked = rec.checkedAt && new Date(rec.checkedAt).getFullYear() > 1 ? ` Checked ${esc(ago(rec.checkedAt))}.` : "";
    const said = (rec.version ? `Version ${esc(rec.version)}, installed from ${st.label}.` : `Installed from ${st.label}.`) + checked;
    const out = newer ? ` ${newer.version ? `Version ${esc(newer.version)} is out` : "A newer download is out"}, ${esc(fmtBytes(newer.size || 0))}.` : "";
    const act = !st.on() ? "" : newer
      ? `<button class="gpage-link" data-action="${st.s}-update" data-key="${esc(k)}">Update</button>`
      : `<button class="gpage-link" data-action="${st.s}-check" data-key="${esc(k)}" ${I.checking ? "disabled" : ""}>${I.checking ? "Checking" : "Check for an update"}</button>`;
    return `<p class="gpage-note">${GI.load}<span>${said}${out}</span>${act}</p>`;
  }
  // The record the game page shows, and all it shows of it, to redraw the
  // page only when that changes.
  function storePageKey(st) {
    const P = S.view === "game" && S.page, it = P && everyItem().find((x) => x.kind === P.kind && x.id === P.id);
    return it ? storeRecKey(st, it) : "";
  }
  function storePageState(st) {
    const k = storePageKey(st), I = S[st.s];
    return k ? JSON.stringify([k, I.recs[k], I.updates.find((u) => u.key === k), I.checking]) : "";
  }
  // A store's Progress, Updates and Done events; false for any other.
  function storeRunEvent(st, ev) {
    const I = S[st.s], D = S.dialog && S.dialog.kind === st.s ? S.dialog : null, kind = ev.type.slice(st.s.length);
    if (kind === "Progress") {
      const prev = I.run, runs = [...document.querySelectorAll(".itch-run")].filter((n) => n.dataset.key === ev.key);
      I.run = ev;
      if (prev && prev.key === ev.key && prev.stage === ev.stage && prev.file === ev.file && runs.length) { runs.forEach((n) => itchPatch(n, ev)); return true; }
      if (D && D.key === ev.key) renderOverlay();
      if (storePageKey(st) === ev.key) render();
      return true;
    }
    if (kind === "Updates") {
      const was = storePageState(st);
      I.recs = ev.records || {}; I.updates = ev.list || []; I.checking = false;
      if (ev.error && I.asked) toast(ev.error, "error");
      I.asked = false;
      if (storePageState(st) !== was) render();
      return true;
    }
    if (kind !== "Done") return false;
    I.run = null;
    if (D && D.key === ev.key) { D.busy = false; if (ev.error) { D.error = ev.error; renderOverlay(); } else closeDialog(); }
    if (ev.cancelled) toast(st.cancelled(ev.name));
    else if (ev.error) toast(ev.error, "error");
    else if (ev.updKey) {
      // The update is downloaded: the update panel shows what it changes, and nothing changes until it's applied.
      const U = S.dialog = { kind: "update", key: ev.updKey, name: ev.name, path: ev.path, busy: true, [st.s]: true };
      renderOverlay(); focusOverlay();
      api.updatePlan(U.key, U.path).catch((e) => { U.busy = false; U.error = errText(e); renderOverlay(); });
    } else if (ev.libKey) {
      const key = ev.libKey;
      toast(`${ev.name} is installed.`, "", { label: "Open game", run: () => openSheet("found", key) });
      loadFound().then(refresh);
      st.installed(key);
    }
    if (S.view === "game" || S.view === "library") render();
    return true;
  }

  function itchEvent(ev) {
    const I = S.itch, D = S.dialog && S.dialog.kind === "itch" ? S.dialog : null;
    if (ev.type === "itch") {
      I.busy = false; I.error = ev.error || "";
      if (ev.connected) { toast(`Connected to itch.io as ${ev.user}.`); refresh(); }
      else if (S.view === "settings") render();
      return;
    }
    if (ev.type === "itchUploads") {
      if (!D || D.gameId !== ev.gameId) return;
      D.error = ev.error || ""; D.busy = false;
      if (!ev.error) {
        Object.assign(D, { uploads: ev.uploads || [], dir: ev.dir || "", target: ev.target || "", free: ev.free ?? -1, dataDrive: ev.dataDrive || "", dataFree: ev.dataFree ?? -1 });
        if (!D.pick && D.uploads.length) D.pick = D.uploads[0].id;
      }
      renderOverlay();
      return;
    }
    storeRunEvent(ITCH, ev);
  }
