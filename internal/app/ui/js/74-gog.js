  // ------------------------------------------------------------------- GOG --
  // The games in the user's GOG library (app/gog.go), signed in as GOG Galaxy
  // does: listed under Not installed, installed from a drawer with GOG's
  // offline installer, and on a game Lootnest installed, a check for another
  // version that goes in through the update panel. S.gog is shaped as S.itch.
  const gogOn = () => !!cfg().settings.gogUser;
  const gogStageText = (R) => ({
    download: R.files > 1 ? `Downloading file ${R.file} of ${R.files}:` : "Downloading",
    install: "GOG's installer is running and shows its own progress. Windows may ask first.",
    check: "Looking for the game it installed",
  }[R.stage] || "Getting ready");
  const GOG = {
    s: "gog", label: "GOG", on: gogOn, text: gogStageText,
    cancelled: (name) => `The download of ${name} is paused. Install it again to pick up where it left off.`,
    installed: () => api.gogRecords().catch(() => {}),
  };

  // Settings, Online: sign in by pasting the address GOG ends on, or who's signed in.
  function gogRows() {
    const I = S.gog, s = cfg().settings;
    const b = (attrs, label, kind = "secondary") => `<button class="ln-button ln-button--${kind} ln-button--sm" ${attrs}>${label}</button>`;
    const spin = (label) => `<button class="ln-button ln-button--secondary ln-button--sm" aria-busy="true" disabled><span class="ln-spinner" aria-hidden="true"></span>${label}</button>`;
    const err = I.error ? `<p class="error-text">${esc(I.error)}</p>` : "";
    if (s.gogUser) return `<div class="bk-row"><div><b>Signed in as ${esc(s.gogUser)}</b>
        <p>Your GOG games that aren't installed show in the Library under Not installed. The sign-in is encrypted on this PC and sent only to GOG.</p>${err}</div>
      <div class="set-itch-acts">${I.refreshing ? spin("Refreshing") : b('data-action="gog-refresh"', "Refresh library")}${b('data-action="gog-signout"', "Sign out", "ghost")}</div></div>`;
    return `<div class="bk-row set-wide"><div><b>Sign in</b>
        <p>GOG's sign-in page opens in your browser. Once you've signed in, GOG shows a nearly empty page: copy its whole address from the address bar and paste it here.</p>
        <p><button class="gpage-link" data-action="gog-login">Sign in to GOG</button></p></div>
      <div class="set-itch-acts"><input id="s-gogcode" class="ln-input" aria-label="The address GOG showed after you signed in" type="text" autocomplete="off" spellcheck="false" placeholder="Paste the address, https://embed.gog.com/on_login_success..." ${I.error ? 'aria-describedby="s-gogcode-err"' : ""} ${I.busy ? "disabled" : ""}>
        ${I.busy ? spin("Signing in") : b('data-action="gog-signin"', "Finish signing in")}${I.error ? `<p id="s-gogcode-err" class="error-text">${esc(I.error)}</p>` : ""}</div></div>`;
  }

  function openGog(it) {
    const D = S.dialog = { kind: "gog", key: it.id, id: +it.o.storeId, name: it.name, installers: null, pick: "", dir: "", target: "", free: -1, dataDrive: "", dataFree: -1, kept: 0, error: "" };
    renderOverlay(); focusOverlay();
    api.gogInstallers(D.id, "").catch((e) => { D.error = errText(e); renderOverlay(); });
  }
  const gogLabel = (x) => `${x.languageName || x.language}${x.version ? ", version " + x.version : ""}`;
  function viewGogInstall() {
    const D = S.dialog, R = S.gog.run && S.gog.run.key === D.key ? S.gog.run : null;
    const shell = (body, foot) => drawer({ label: `Install ${esc(D.name)}`, title: `Install ${esc(D.name)}`, sub: "From your GOG library", body, foot });
    const close = `<span class="grow"></span><button class="ln-button ln-button--ghost ln-button--sm" data-action="dialog-cancel">Close</button>`;
    if (R) return shell(itchMeter(R, "", gogStageText) + `<p class="gpage-hint">You can close this panel; Lootnest keeps going and tells you when the game is in your Library.</p>`,
      `${R.stage === "download" ? `<button class="ln-button ln-button--secondary ln-button--sm" data-action="gog-cancel">Cancel the download</button>` : ""}${close}`);
    if (D.error && !D.installers) return shell(lnInline("error", esc(D.error)), close);
    if (!D.installers) return shell(lnInline("", "Asking GOG for the game's installers"), close);
    if (!D.installers.length) return shell(`<p>GOG has no offline installer of ${esc(D.name)} for Windows.</p>`, close);
    const x = D.installers.find((i) => i.id === D.pick) || D.installers[0], size = x.size || 0, parts = (x.files || []).length;
    // Room as Go checks it: the game on its folder's drive, the rest of the download on Lootnest's.
    const drive = (D.dir || "").slice(0, 2).toUpperCase(), rest = Math.max(0, size - (D.kept || 0));
    const room = drive === D.dataDrive ? [[drive, D.free, size * 2.5 + rest]] : [[drive, D.free, size * 2.5], [D.dataDrive, D.dataFree, rest]];
    const short = room.find(([, free, need]) => free >= 0 && free < need + 64 * 2 ** 20);
    const which = D.installers.length > 1
      ? `<select id="gog-in" class="ln-input" aria-label="Language">${D.installers.map((i) => `<option value="${esc(i.id)}" ${i.id === x.id ? "selected" : ""}>${esc(gogLabel(i))}</option>`).join("")}</select>`
      : `<p><b>${esc(gogLabel(x))}</b></p>`;
    const body = `<div class="gpage-sub lo-sec itch-dl">${gpHead("Installer")}${which}
        <dl class="ln-facts"><div><dt>Size</dt><dd class="ln-num">${esc(fmtBytes(size))}${parts > 1 ? `, in ${parts} files` : ""}</dd></div>
          ${D.kept ? `<div><dt>Already downloaded</dt><dd class="ln-num">${esc(fmtBytes(D.kept))}</dd></div>` : ""}
          ${room.map(([drv, free], i) => `<div><dt>Free on ${esc(drv)}${room.length > 1 && i ? ", for the download" : ""}</dt><dd class="ln-num">${free >= 0 ? esc(fmtBytes(free)) : "Not known"}</dd></div>`).join("")}</dl>
        ${short ? `<p class="find-low">There isn't enough room on ${esc(short[0])}: this needs about ${esc(fmtBytes(short[2]))}.</p>` : ""}</div>
      <div class="gpage-sub lo-sec">${gpHead("Folder")}
        <div class="lo-pick"><span class="selectable">${esc(D.target || D.dir)}</span><button class="ln-button ln-button--ghost ln-button--sm" data-action="gog-dir">Change</button></div>
        <p class="gpage-hint">Lootnest downloads GOG's installer, then runs it into this folder without questions. It never installs into a folder that has files in it.</p></div>
      ${D.error ? lnInline("error", esc(D.error)) : ""}`;
    return shell(body, `<span class="grow"></span><button class="ln-button ln-button--ghost ln-button--sm" data-action="dialog-cancel">Cancel</button>
      <button class="ln-button ln-button--primary ln-button--sm" data-action="gog-install" data-installer="${esc(x.id)}" ${short || D.busy ? "disabled" : ""}>${D.kept ? "Continue installing" : "Install"}</button>`);
  }

  function gogEvent(ev) {
    const I = S.gog, D = S.dialog && S.dialog.kind === "gog" ? S.dialog : null;
    if (ev.type === "gog") {
      I.busy = false; I.error = ev.error || "";
      if (ev.signedIn) { toast(`Signed in to GOG as ${ev.user}.`); refresh(); }
      else if (ev.signedOut) { toast(ev.error, "error"); refresh(); }
      else if (S.view === "settings") render();
      return;
    }
    if (ev.type === "gogInstallers") {
      if (!D || D.id !== ev.id) return;
      D.error = ev.error || ""; D.busy = false;
      if (!ev.error) {
        Object.assign(D, { installers: ev.installers || [], dir: ev.dir || "", target: ev.target || "", free: ev.free ?? -1, dataDrive: ev.dataDrive || "", dataFree: ev.dataFree ?? -1, kept: ev.kept || 0 });
        if (!D.pick) D.pick = ev.pick || "";
      }
      renderOverlay();
      return;
    }
    storeRunEvent(GOG, ev);
  }
