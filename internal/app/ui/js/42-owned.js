  // ------------------------------------------- stores, not installed, remove --
  // A game's store does the installing, uninstalling and repairing
  // (ps_storeAction): Lootnest opens its launcher on the game and says
  // which. Games a store's files on this PC say you have, not installed
  // (ps_ownedGames), are the Library's Not installed. A folder game is
  // removed with its saves copied first (ps_removeGame).
  const isOwnedKey = (k) => String(k || "").startsWith("owned:");
  const ownedItems = () => (S.owned || []).map((o) => ({ kind: "owned", id: o.key, name: o.name, appId: o.appId || 0, source: o.source, exe: "", o }));
  const ownedItem = (key) => ownedItems().find((x) => x.id === key) || null;
  // [what the button says, what happens, the toast once the launcher opens]
  function storeActText(src, act, name) {
    const t = {
      "steam install": ["Install with Steam", "Steam downloads and installs it", `Steam is opening the install for ${name}.`],
      "steam uninstall": ["Uninstall with Steam", "Steam asks first, then removes it", `Steam is asking whether to uninstall ${name}.`],
      "steam verify": ["Verify files with Steam", "Steam replaces missing or damaged files", "Steam is checking the game's files. It downloads any that are missing or damaged."],
      "epic install": ["Install with Epic", "The Epic Games Launcher downloads and installs it", `The Epic Games Launcher is opening the install for ${name}.`],
      "epic uninstall": ["Open your Epic library", "Uninstall it from its menu there", "Your Epic library is opening. Uninstall the game from its menu there."],
      "gog open": ["Open in GOG Galaxy", "Install, uninstall and repair it there", `GOG Galaxy is opening on ${name}.`],
      "ubisoft install": ["Install with Ubisoft Connect", "Ubisoft Connect downloads and installs it", `Ubisoft Connect is opening the install for ${name}.`],
      "ubisoft uninstall": ["Uninstall with Ubisoft Connect", "Ubisoft Connect asks first, then removes it", `Ubisoft Connect is asking whether to uninstall ${name}.`],
      "battlenet open": ["Open in Battle.net", "Install, uninstall and repair it there", `Battle.net is opening on ${name}.`],
    }[src + " " + act];
    return t || (act === "uninstall" ? ["Uninstall in Windows Settings", "Opens Installed apps, where you find the game", `Installed apps is opening in Windows Settings. Find ${name} there.`] : null);
  }
  const storeActs = (it) => (it.kind === "owned" ? ["install"] : (S.found.acts || {})[it.id] || []);
  const storeRow = (it, act, opt) => { const t = storeActText(it.source, act, it.name); return t ? gpRow(`data-action="store-act" ${itemAttr(it)} data-act="${act}"`, act === "uninstall" ? GI.trash : act === "open" ? GI.next : GI.check, t[0], t[1], opt) : ""; };
  const storeItem = (it, act) => { const t = storeActText(it.source, act, it.name); return t ? `<button${act === "uninstall" ? ' class="danger"' : ""} data-action="store-act" ${itemAttr(it)} data-act="${act}">${esc(t[0])}</button>` : ""; };
  // A game in a folder of its own, outside any store: Remove from this PC.
  const removable = (it) => it.kind === "found" && UPDATABLE.includes(it.source) && !it.appId && !it.f.uri && !!it.f.dir;
  // label: the button's text ("" for an icon only); it always names the store.
  function installButton(it, cls, label) {
    const t = storeActText(it.source, "install", it.name);
    return t ? `<button class="ln-button ${cls}" data-action="store-act" ${itemAttr(it)} data-act="install" aria-label="${esc(t[0])}: ${esc(it.name)}" title="${esc(t[0])}">${GI.load}${esc(label)}</button>` : "";
  }

  // A game you own but haven't installed: its art, and Install with its store.
  function ownedView(it) {
    loadArt(it); ensureCover(it);
    const h = artFor(it, "hero"), logo = artFor(it, "logo").url, store = sourceLabel(it.source);
    const art = h.url ? artImg(h, "hero", "gpage-art") : `<div class="gpage-plate" aria-hidden="true"><span>${esc(initial(it.name))}</span></div>`;
    const t = storeActText(it.source, "install", it.name);
    return `<div class="gpage ln-root is-owned${h.url ? "" : " no-art"}"><section class="gpage-hero">${art}
        <button class="ln-button ln-button--ghost ln-button--sm gpage-back" data-action="back-library">${GI.back}${esc(backLabel())}</button>
        <div class="gpage-title"><h1 class="${it.name.length > 22 && !logo ? "long" : ""}">${logo ? `<img class="gpage-logo" src="${logo}" alt="${esc(it.name)}">` : esc(it.name)}</h1>
          <p class="gpage-meta"><span>${esc(store)}</span></p></div></section>
      <div class="gpage-body"><section class="gpage-bar" aria-label="Install">${installButton(it, "ln-button--primary ln-button--lg", t ? t[0] : "")}
        <div class="gpage-state"><b>Not installed</b><span>Found in the files ${esc(store)} keeps on this PC.${t ? ` ${esc(t[1])}.` : ""}</span></div></section></div></div>`;
  }

  // Remove from this PC: what's copied first, then the folder to the Recycle Bin.
  function viewRemovePC(D) {
    const P = D.plan, n = P ? P.saves.length : 0;
    const saves = !P ? (D.error ? "" : `<p class="gpage-hint">Looking for its saves</p>`)
      : n ? `<div class="rm-list"><p class="rm-head">${n === 1 ? "Its save location, copied first" : `Its ${n} save locations, copied first`}</p><ul class="rm-saves">${P.saves.map((s) => `<li><span class="selectable" title="${esc(s.from)}">${esc(s.rel || s.from)}</span><span class="ln-num">${fmtSize(s.size)}</span></li>`).join("")}</ul></div>`
      : `<p class="gpage-hint">Lootnest doesn't know where this game keeps its saves. If they're in its folder, they go to the Recycle Bin with it.</p>`;
    return `<div class="scrim" data-action="scrim"><div class="dialog narrow" role="dialog" aria-modal="true" aria-label="Remove from this PC">
      <div><h2>Remove ${esc(D.name)} from this PC?</h2><p class="sub">${P && !n ? "Lootnest moves the game's folder to the Recycle Bin." : "Lootnest copies its saves to its backups first, then moves the game's folder to the Recycle Bin. Add the game again later, and Lootnest offers its saves back."}</p></div>
      ${P ? `<p class="rm-dir"><span>Goes to the Recycle Bin</span><span class="selectable">${esc(P.dir)}</span><span class="ln-num">${fmtSize(P.size)}</span></p>` : ""}
      ${saves}
      ${P ? `<p class="gpage-hint">If the folder is too big for the Recycle Bin, Windows asks you before deleting it.</p>` : ""}
      <div class="dialog-foot"><span class="error-text grow">${esc(D.error || "")}</span><button class="btn quiet" data-action="dialog-cancel" ${D.busy ? "disabled" : ""}>Cancel</button>
      <button class="btn primary" data-action="remove-pc-go" ${D.busy || !P ? "disabled" : ""}>${D.busy ? (n ? "Copying its saves" : "Removing") : "Remove from this PC"}</button></div></div></div>`;
  }
