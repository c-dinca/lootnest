  // ----------------------------------------------------------- controllers --
  // Settings, Controllers: the pads Windows lists and the apps that change
  // how games see them (ps_controllers, read only), beside what the Gamepad
  // API shows Lootnest and Big Picture. Asked when the category opens and
  // when a pad connects or goes; never in the background.
  let ctlAsked = false;
  function askControllers() {
    if (ctlAsked) return;
    ctlAsked = true;
    const done = (r) => { S.ctl = r; if (S.view === "settings" && S.setTab === "controllers") render(); };
    api.controllers().then(done).catch(() => done({ pads: [], xinput: [], translators: [], notes: [], failed: true }));
  }
  const ctlStale = () => { ctlAsked = false; if (S.view === "settings" && S.setTab === "controllers") askControllers(); };
  window.addEventListener("gamepadconnected", ctlStale);
  window.addEventListener("gamepaddisconnected", ctlStale);

  // The face buttons as the pad's family labels them, in a small diamond.
  const padFaces = (fam) => `<span class="ctl-faces" aria-hidden="true">${["Y", "X", "B", "A"].map((b) => {
    const v = PADS[fam][b];
    return `<span class="ctl-g ctl-g--${b.toLowerCase()}">${PAD_SHAPES[v] ? gsvg(PAD_SHAPES[v]) : esc(v)}</span>`;
  }).join("")}</span>`;
  const ctlRow = (fam, name, meta, note) => `<div class="ctl-pad">${padFaces(fam)}<div><b>${esc(name)}</b>${meta.length ? `<p>${meta.map(esc).join(", ")}</p>` : ""}${note ? `<p>${esc(note)}</p>` : ""}</div></div>`;
  const BUS = { usb: "USB", bluetooth: "Bluetooth" };
  const FAMILY = { xbox: "Xbox buttons", playstation: "PlayStation buttons", nintendo: "Nintendo buttons", generic: "Buttons by position" };

  function viewControllers() {
    askControllers();
    const C = S.ctl;
    if (!C) return lnInline("", "Asking Windows about your controllers");
    const web = padsNow(), copies = C.pads.some((p) => p.virtual);
    const mine = C.pads.map((p) => ctlRow(p.family, p.name, [
      p.virtual ? `Virtual, made by ${p.madeBy || "an app on this PC"}` : "",
      BUS[p.bus] || "",
      p.slot ? `Xbox-style (XInput) slot ${p.slot}` : p.xinput ? "Xbox-style (XInput)" : "",
    ].filter(Boolean)));
    const slots = C.xinput.filter((x) => !C.pads.some((p) => p.slot === x.slot));
    if (slots.length) mine.push(`<div class="bk-row"><div><b>Xbox-style (XInput) slots in use</b><p>${slots.map((x) => `Slot ${x.slot}: ${x.wireless ? "wireless " : ""}${esc(x.kind)}`).join(". ")}.</p></div></div>`);
    // The name Windows gave the same pad, so the two columns agree.
    const xpads = C.pads.filter((p) => p.xinput);
    const nameOf = (id) => {
      const m = /vendor: ?([0-9a-f]{4}) product: ?([0-9a-f]{4})/i.exec(id);
      const p = m && C.pads.find((p) => p.vid === parseInt(m[1], 16) && p.pid === parseInt(m[2], 16));
      if (p) return p.name;
      if (/xinput/i.test(id)) return xpads.length === 1 ? xpads[0].name : "Xbox-style (XInput) controller";
      return id.replace(/\s*\(.*$/, "") || "Controller";
    };
    const seen = web.map((g) => {
      const fam = padFamily(g.id), meta = [FAMILY[fam], g.mapping === "standard" ? "" : "its own layout, so some buttons may be elsewhere"];
      const twin = fam === "xbox" && web.some(realPad) && (copies ? "Lootnest ignores this one while the other pad is in use: it's a copy another app made." : "Lootnest ignores this one while the other pad is in use, in case it's a copy another app made.");
      return ctlRow(fam, nameOf(g.id), meta.filter(Boolean), twin);
    });
    const pads = !mine.length && !seen.length ? setSub("On this PC", C.failed ? "Windows didn't answer. Open this page again to ask once more." : "Connect a controller. Lootnest reads what Windows says about it and never changes its settings.", "")
      : `<div class="ctl-cols">${setSub("On this PC", C.failed ? "Windows didn't answer. Open this page again to ask once more." : "What Windows lists. Lootnest only reads it, and never sends anything to a controller.", mine.length ? `<div class="bk-card">${mine.join("")}</div>` : `<p class="set-note">Windows lists no controller.</p>`)}
        ${setSub("What Lootnest sees", "What Lootnest and Big Picture see right now. A controller shows here once you press one of its buttons.", seen.length ? `<div class="bk-card">${seen.join("")}</div>` : `<p class="set-note">None yet. Press a button on your controller.</p>`)}</div>`;
    const tools = C.translators.length ? setSub("Apps that change your controllers", "Each makes games see your pad differently.", `<div class="bk-card">${C.translators.map((t) =>
      `<div class="bk-row"><div><b>${esc(t.name)}</b><p>${esc(t.does)}</p></div><span class="ln-tag">${t.running ? "Running now" : "Installed"}</span></div>`).join("")}</div>
      ${C.notes.map((n) => `<p class="ctl-note">${esc(n)}</p>`).join("")}`) : "";
    return `<div class="ctl-page">${pads + tools}</div>`;
  }
