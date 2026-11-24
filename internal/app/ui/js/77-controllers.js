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
  const ctlRow = (fam, name, meta, note, keys = "", title = "", after = "") => `<div class="bk-row ctl-pad">${padFaces(fam)}<div>${title || `<b>${esc(name)}</b>`}${meta.length ? `<p>${meta.map(esc).join(", ")}</p>` : ""}${note ? `<p>${esc(note)}</p>` : ""}${after}</div>${keys ? `<span class="ctl-keys">${keys}</span>` : ""}</div>`;
  // A real pad's own name (Config.PadNames by Pad.ID), and a key that makes
  // it vibrate so it's clear which one is being named (app/padbuzz.go).
  // S.padNaming is the pad whose name is being typed, S.padBuzz the one
  // buzzing now.
  function ctlKeys(p) {
    if (p.virtual || !p.id) return "";
    const id = esc(p.id), buzzing = S.padBuzz === p.id;
    const saving = (battOf(p.id) || {}).saver;
    const buzz = p.buzz && !saving ? `<button class="ln-button ln-button--secondary ln-button--sm" data-action="pad-buzz" data-id="${id}"${buzzing ? " disabled" : ""}>${ph("vibrate")}<span>${buzzing ? "Vibrating" : "Vibrate"}</span></button>` : "";
    if (S.padNaming === p.id) return `${buzz}<button class="ln-button ln-button--primary ln-button--sm" data-action="pad-name-save" data-id="${id}">Save</button>`;
    return `${buzz}<button class="ln-button ln-button--ghost ln-button--sm ln-button--icon" data-action="pad-name" data-id="${id}" aria-label="Name this controller" title="Name this controller">${ph("pencil-simple")}</button>`;
  }
  // A controller's battery (app/padbattery.go): the latest reading, from
  // ps_controllers and the "padBattery" event. An Xbox pad says only a
  // rough level.
  const battOf = (id) => (S.batt || (S.ctl && S.ctl.batteries) || []).find((b) => b.id === id);
  function battIcon(b) {
    if (b.charging) return ph("battery-charging");
    if (b.full) return ph("battery-full");
    const p = b.percent;
    return ph(p <= 10 ? "battery-warning" : p <= 25 ? "battery-low" : p <= 55 ? "battery-medium" : p <= 85 ? "battery-high" : "battery-full");
  }
  const BATT_LEVEL = { empty: "Almost empty", low: "Low", medium: "Half", full: "Full" };
  function battWords(b) {
    if (b.full) return "Charged";
    const n = b.level ? BATT_LEVEL[b.level] : `${b.percent}%`;
    return b.charging ? `${n}, charging` : n;
  }
  const battLow = (b) => !b.charging && !b.full && b.percent >= 0 && b.percent <= 20;
  const battChip = (b, cls = "ctl-batt") => b ? `<span class="${cls}${battLow(b) ? " is-low" : ""}${b.charging ? " is-charging" : ""}" title="${esc(battWords(b) + (b.saver ? ", battery saver on" : ""))}">${battIcon(b)}<span class="ln-num">${esc(battWords(b))}</span>${b.saver ? `<span class="ctl-batt-saver">Saver</span>` : ""}</span>` : "";
  const battLine = (b) => b ? `<p class="ctl-batt-line">${battChip(b)}</p>` : "";
  const padTitle = (p) => S.padNaming === p.id
    ? `<input class="ln-input ctl-name-in" id="ctl-name-in" type="text" maxlength="32" value="${esc(p.given || "")}" placeholder="Such as Mihai's" data-id="${esc(p.id)}" aria-label="A name for this controller">`
    : "";
  const BUS = { usb: "USB", bluetooth: "Bluetooth" };
  const SAVER_MODES = [["", "Off"], ["low", "When a controller runs low"], ["on", "Always"]];
  const FAMILY = { xbox: "Xbox buttons", playstation: "PlayStation buttons", nintendo: "Nintendo buttons", generic: "Buttons by position" };

  function viewControllers() {
    askControllers();
    const C = S.ctl;
    const linux = S.d.platform === "linux", os = linux ? "Linux" : "Windows";
    if (!C) return lnInline("", `Asking ${os} about your controllers`);
    const web = padsNow(), copies = C.pads.some((p) => p.virtual);
    const mine = C.pads.map((p) => ctlRow(p.family, p.given || p.name, [
      p.given || S.padNaming === p.id ? p.name : "",
      p.virtual ? `Virtual, made by ${p.madeBy || "an app on this PC"}` : "",
      BUS[p.bus] || "",
      p.slot ? `Xbox-style (XInput) slot ${p.slot}` : p.xinput ? "Xbox-style (XInput)" : "",
    ].filter(Boolean), "", ctlKeys(p), padTitle(p), battLine(battOf(p.id))));
    for (const b of (S.batt || C.batteries || []).filter((b) => !C.pads.some((p) => p.id === b.id))) mine.push(ctlRow("xbox", b.name, [], "", "", "", battLine(b)));
    const slots = C.xinput.filter((x) => !C.pads.some((p) => p.slot === x.slot));
    if (slots.length) mine.push(setRow("Xbox-style (XInput) slots in use", slots.map((x) => `Slot ${x.slot}: ${x.wireless ? "wireless " : ""}${esc(x.kind)}`).join(", ")));
    // The name the system gave the same pad, so the two columns agree.
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
    const pads = !mine.length && !seen.length ? setSub("On this PC", "", card(C.failed ? setRow(`${os} didn't answer`, "Open this page again to ask once more.") : setRow("No controller connected", "")))
      : `<div class="ctl-cols">${col("On this PC", mine, C.failed ? `${os} didn't answer` : `${os} lists none`)}${col("What Lootnest sees", seen, "Press a button on your controller")}</div>`;
    const tools = C.translators.length ? setSub("Apps that change your controllers", "", card(C.translators.map((t) =>
      setRow(esc(t.name), esc(t.does), `<span class="ctl-lamp${t.running ? " is-on" : ""}">${t.running ? "Running now" : "Installed"}</span>`)).join(""))
      + C.notes.map((n) => `<p class="ctl-note">${GI.alert}<span>${esc(n)}</span></p>`).join("")) : "";
    // A PlayStation pad as an Xbox one while a game runs (Settings.PadAsXbox,
    // app/padbridge.go): only with the ViGEmBus driver already on the PC.
    const on = !!cfg().settings.padAsXbox;
    const xbox = setSub("In games", "", card(sw("padAsXbox", "Show a PlayStation controller to games as an Xbox one",
      C.vigem ? "While a game runs, for games that only know Xbox controllers: each DualSense or DualShock 4 becomes an Xbox controller of its own. A game Steam says knows PlayStation controllers gets them as they are; a game's Launch options can say otherwise."
        : "Needs the ViGEmBus driver, which isn't on this PC. DS4Windows and Lootnest's TV streaming set-up install it.", on)
      + (on && C.bridge ? setRow("Now", esc(C.bridge.charAt(0).toUpperCase() + C.bridge.slice(1))) : "")));
    // Batteries: reading them, and the battery saver (Settings.PadBatteryOff,
    // PadSaver; core/padbattery.go).
    const st = cfg().settings, reading = !st.padBatteryOff;
    const saverSel = `<select id="s-pad-saver" class="ln-input set-input" data-setting="padSaver"${reading ? "" : " disabled"}>${SAVER_MODES.map(([v, l]) => `<option value="${v}"${(st.padSaver || "") === v ? " selected" : ""}>${l}</option>`).join("")}</select>`;
    const battery = setSub("Battery", "", card(sw("padBatteryOff", "Show how charged controllers are",
      "Warns when one runs low and says when it's charged. A DualSense or DualShock 4 on Bluetooth is asked for its full reports to say it, as Steam does.", reading, true)
      + setRow("Battery saver", "Turns off a PlayStation controller's light, and a DualSense's vibration, until it's switched off. Lootnest's Vibrate key waits too.", saverSel)));
    const det = cfg().settings.gameDetails ? "" : setSub("Games", "", card(setRow("Controller support on game pages", "", `<button class="ln-button ln-button--secondary ln-button--sm" data-action="details-on">Turn on Game details</button>`)));
    // On Linux the pads are read from the kernel (plat/pads_linux.go): no
    // ViGEmBus there, and batteries aren't read yet.
    if (linux) return `<div class="ctl-page">${pads + tools + det}</div>`;
    return `<div class="ctl-page">${pads + battery + xbox + tools + det}</div>`;
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
