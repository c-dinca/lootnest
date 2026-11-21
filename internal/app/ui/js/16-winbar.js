  // --------------------------------------------------------------- winbar --
  // On Windows the window has no caption of its own (app/winframe.go): the
  // page holds the strip it's dragged by (the top of the window, and the
  // sidebar's top) and the caption's keys (#winbar). A press there that
  // isn't on something of the page's own moves the window; a double press
  // maximizes or restores it; the top edge resizes. Linux keeps its
  // desktop's own title bar.
  const WB_OWN = "button, a, input, select, textarea, label, summary, [data-action], [role=button], [role=menuitem], [contenteditable], .side-resize, .wb-keys";
  let wbOn = false;
  let wbLast = { t: 0, x: 0, y: 0 }; // the last press on the strip, for a double press
  function paintWinbar(st) {
    const bar = $("#winbar");
    if (!bar || !st) return;
    wbOn = !!st.frame;
    bar.hidden = !wbOn;
    const root = document.documentElement;
    root.classList.toggle("win-chrome", wbOn);
    root.classList.toggle("win-max", !!st.max);
    const max = bar.querySelector('[data-wb="max"]');
    if (max) { const say = st.max ? "Restore" : "Maximize"; max.title = say; max.setAttribute("aria-label", say); }
  }
  function winbarStart() {
    if (typeof window.ps_win !== "function" || (S.d && S.d.platform === "linux")) return;
    api.win("state").then(paintWinbar).catch(() => {});
    const bar = $("#winbar");
    bar.addEventListener("click", (e) => {
      const k = e.target.closest(".wb-key");
      if (k) api.win(k.dataset.wb).then(paintWinbar).catch(() => {});
    });
    // Capture, so a page's own handlers can't swallow the press first.
    document.addEventListener("mousedown", (e) => {
      if (!wbOn || e.button !== 0) return;
      const edge = e.target.closest(".wb-edge");
      if (edge) { e.preventDefault(); api.win(edge.dataset.wb).catch(() => {}); return; }
      if (e.target.closest(WB_OWN)) return;
      const strip = e.clientY < $("#winbar").offsetHeight || e.target.closest(".side-top");
      if (!strip || BP.on || document.querySelector("#bp")) return;
      e.preventDefault();
      // The system's move loop takes the button up, so the page may never
      // count a second press as one: time it here.
      const again = e.timeStamp - wbLast.t < 500 && Math.abs(e.clientX - wbLast.x) < 6 && Math.abs(e.clientY - wbLast.y) < 6;
      wbLast = again ? { t: 0, x: 0, y: 0 } : { t: e.timeStamp, x: e.clientX, y: e.clientY };
      api.win(again ? "max" : "drag").then(paintWinbar).catch(() => {});
    }, true);
    window.addEventListener("blur", () => document.documentElement.classList.add("win-blur"));
    window.addEventListener("focus", () => document.documentElement.classList.remove("win-blur"));
  }
