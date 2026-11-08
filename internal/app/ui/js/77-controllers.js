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
  const ctlRow = (fam, name, meta, note) => `<div class="bk-row ctl-pad">${padFaces(fam)}<div><b>${esc(name)}</b>${meta.length ? `<p>${meta.map(esc).join(", ")}</p>` : ""}${note ? `<p>${esc(note)}</p>` : ""}</div></div>`;
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
    if (slots.length) mine.push(setRow("Xbox-style (XInput) slots in use", slots.map((x) => `Slot ${x.slot}: ${x.wireless ? "wireless " : ""}${esc(x.kind)}`).join(", ")));
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
      const fam = padFamily(g.id), meta = [FAMILY[fam], g.mapping === "standard" ? "" : "its own layout"];
      const twin = fam === "xbox" && web.some(realPad) && (copies ? "Ignored: a copy another app made" : "Ignored while the other pad is in use");
      return ctlRow(fam, nameOf(g.id), meta.filter(Boolean), twin);
    });
    const card = (rows) => `<div class="bk-card">${rows}</div>`;
    const col = (title, rows, none) => setSub(title, "", card(rows.length ? rows.join("") : setRow(none, "")));
    const pads = !mine.length && !seen.length ? setSub("On this PC", "", card(C.failed ? setRow("Windows didn't answer", "Open this page again to ask once more.") : setRow("No controller connected", "")))
      : `<div class="ctl-cols">${col("On this PC", mine, C.failed ? "Windows didn't answer" : "Windows lists none")}${col("What Lootnest sees", seen, "Press a button on your controller")}</div>`;
    const tools = C.translators.length ? setSub("Apps that change your controllers", "", card(C.translators.map((t) =>
      setRow(esc(t.name), esc(t.does), `<span class="ctl-lamp${t.running ? " is-on" : ""}">${t.running ? "Running now" : "Installed"}</span>`)).join(""))
      + C.notes.map((n) => `<p class="ctl-note">${GI.alert}<span>${esc(n)}</span></p>`).join("")) : "";
    const det = cfg().settings.gameDetails ? "" : setSub("Games", "", card(setRow("Controller support on game pages", "", `<button class="ln-button ln-button--secondary ln-button--sm" data-action="details-on">Turn on Game details</button>`)));
    return `<div class="ctl-page">${pads + tools + det}</div>`;
  }

  // What a game expects, from its Steam details or the user's edit: one
  // line on its page, with the faces of the pad in use. The note on a copy
  // of the pad reads S.ctl only if Settings, Controllers has asked already.
  const CTL_SAYS = { full: "Full controller support", partial: "Partial controller support", none: "Keyboard and mouse only" };
  function gameCtlLine(d) {
    const says = CTL_SAYS[d.controller];
    if (!says) return "";
    const g = padsNow(), fam = g.length && d.controller !== "none" ? padFamily((g.find(realPad) || g[0]).id) : "";
    const twice = d.controller === "full" && S.ctl && S.ctl.copiers || [];
    return `<div class="ctl-pad gp-ctl">${fam ? padFaces(fam) : ""}<div><b>${says}</b>
      ${d.controller !== "none" && (d.pads || []).includes("playstation") ? "<p>PlayStation controllers too</p>" : ""}
      ${twice.length ? `<p>${esc(andList(twice))} ${twice.length > 1 ? "are" : "is"} running: this game may see your pad twice.</p>` : ""}</div></div>`;
  }
