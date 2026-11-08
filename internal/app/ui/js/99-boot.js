  // ------------------------------------------------------------------ boot --
  // The splash (#boot) fades out once the first page is drawn, after at
  // least 600 ms so it doesn't blink, and after 4 s whatever happens.
  function bootDone() {
    const b = $("#boot");
    if (!b || b.classList.contains("gone")) return;
    b.addEventListener("transitionend", () => b.remove(), { once: true });
    setTimeout(() => b.remove(), 400);
    b.classList.add("gone");
  }
  setTimeout(bootDone, 4000);
  async function boot() {
    try {
      S.d = await api.state();
    } catch (e) {
      setTimeout(boot, 150); // bindings not injected yet
      return;
    }
    if (S.d.loadError) toast(S.d.loadError, "error");
    applyFold(false); applySideWidth();
    if (S.seat) api.setSeat(S.seat).catch(() => {});
    if (S.d.firstRun) { S.players = JSON.parse(JSON.stringify(cfg().players)); S.dialog = { kind: "welcome" }; }
    render();
    setTimeout(bootDone, Math.max(0, 600 - performance.now()));
    loadHooks().then(() => { if (!S.picker) render(); });
    loadFound();
    api.itchRecords().catch(() => {}); // which games came from itch.io or GOG; asks neither
    api.gogRecords().catch(() => {});
    setTimeout(() => { if (!S.updates) api.updates().catch(() => {}); api.downloads().catch(() => {}); }, 8000); // the Updates count in the sidebar
    let dlAsked = 0; // and on coming back to the window, at most every 30 s
    window.addEventListener("focus", () => { if (Date.now() - dlAsked > 30000) { dlAsked = Date.now(); api.downloads().catch(() => {}); } });
    if (S.d.firstRun) setTimeout(() => { const i = $('[data-field="pname"]'); if (i) i.focus(); }, 50);
    try {
      const r = await api.ready();
      if (r && r.launch) openPicker(r.launch, r.reqId ? "steam" : "play", r.reqId);
      else if (r && r.bigPicture) bpOpen();
    } catch (e) { /* ignore */ }
  }
  boot();
})();
