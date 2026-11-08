  // --------------------------------------------------- import from Playnite --
  // One way (core/playnite.go): Lootnest reads Playnite's library on this PC
  // and never writes to it. The preview has everything a match could bring
  // in; the switches choose what is.
  const PN_OPTS = [["playtime", "Time played"], ["statuses", "Statuses"], ["favourites", "Favourites"], ["ratings", "Ratings"], ["notes", "Notes"],
    ["categories", "Categories as collections"], ["tags", "Tags as collections"]];
  const PN_SHOWN = 200;
  async function openPlaynite() {
    const who = seated() || people()[0] || cfg().players[0];
    const D = S.dialog = { kind: "playnite", dir: "", player: who ? who.id : "", plan: null, loading: true, error: "", busy: false, unmatched: false,
      opts: { playtime: true, statuses: true, favourites: true, ratings: true, notes: true, categories: false, tags: false } };
    renderOverlay(); focusOverlay();
    try { D.dir = (await api.playniteFind()).dir || ""; } catch (e) { D.error = errText(e); }
    if (S.dialog !== D) return;
    if (D.dir) playnitePreview(D); else { D.loading = false; renderOverlay(); }
  }
  function playnitePreview(D) {
    D.loading = true; D.plan = null; D.error = ""; D.req = D.dir;
    renderOverlay();
    api.playnitePreview(D.dir, D.player).catch((e) => { D.loading = false; D.error = errText(e); if (S.dialog === D) renderOverlay(); });
  }
  // What one game gets, as short marks: +12 h, Finished, 4 stars.
  function pnMarks(c, o) {
    const m = [];
    if (o.playtime && c.add) m.push("+" + hours(c.add));
    else if (o.playtime && c.lastPlayed) m.push("Last played " + new Date(c.lastPlayed).toLocaleDateString(undefined, { dateStyle: "medium" }));
    if (o.statuses && c.status) m.push((STATUSES.find((s) => s[0] === c.status) || [, c.status])[1]);
    if (o.ratings && c.stars) m.push(c.stars === 1 ? "1 star" : c.stars + " stars");
    if (o.favourites && c.favourite) m.push("Favourite");
    if (o.notes && c.note) m.push("Note");
    [...(o.categories ? c.categories || [] : []), ...(o.tags ? c.tags || [] : [])].forEach((t) => m.push(t));
    return m;
  }
  function viewPlaynite() {
    const D = S.dialog, o = D.opts, P = D.plan;
    const btn = (attrs, label, cls = "ln-button--secondary") => `<button class="ln-button ${cls} ln-button--sm" ${attrs}>${label}</button>`;
    const where = `<div class="gpage-sub lo-sec">${gpHead("Playnite's library")}
      <div class="lo-pick"><span class="${D.dir ? "selectable" : "gpage-hint"}">${esc(D.dir || "Lootnest didn't find Playnite in AppData.")}</span>${btn('data-action="pn-dir"', D.dir ? "Choose another" : "Choose Playnite's folder", D.dir ? "ln-button--ghost" : undefined)}</div>
      <p class="gpage-hint">For Playnite portable, choose the folder it's in. Close Playnite first: it keeps its library to itself while it's open.</p></div>`;
    const who = cfg().players.length < 2 ? "" : `<div class="gpage-sub lo-sec">${gpHead("Profile")}
      <select id="pn-player" class="ln-input" aria-label="The profile it's for">${cfg().players.map((p) => `<option value="${esc(p.id)}" ${p.id === D.player ? "selected" : ""}>${esc(p.name)}</option>`).join("")}</select>
      <p class="gpage-hint">Time played, statuses, ratings and notes go to this profile. Favourites and collections are the Library's, for everyone.</p></div>`;
    const opts = `<div class="gpage-sub lo-sec">${gpHead("Bring in")}<div class="toggles pn-opts">${PN_OPTS.map(([k, label]) => `<label class="toggle"><span><b>${label}</b></span>
      <span class="switch"><input type="checkbox" data-pn="${k}" ${o[k] ? "checked" : ""}><span></span></span></label>`).join("")}</div>
      <p class="gpage-hint">Time played is added to what Lootnest counts, once: importing again adds only what's new. A status, rating or note comes in only for a game that has none here.</p></div>`;
    let preview = "";
    const changes = P ? P.changes.map((c) => [c, pnMarks(c, o)]).filter((x) => x[1].length) : [];
    if (D.loading) preview = lnInline("", "Reading Playnite's library");
    else if (P) {
      const n = P.unmatched.length;
      preview = `<p>${P.matched ? `${P.matched} of Playnite's games are in your Library. ${changes.length ? `${changes.length} of them have something to bring in:` : "There's nothing new to bring in for them."}` : "None of Playnite's games are in your Library."}</p>
        ${changes.length ? `<ul class="lo-list">${changes.slice(0, PN_SHOWN).map(([c, m]) => `<li class="lo-item"><div class="lo-main"><b>${esc(c.name)}</b>${c.from !== c.name ? `<span class="gpage-hint">${esc(c.from)} in Playnite</span>` : ""}
          <span class="pn-marks">${m.map((x) => `<span class="ln-tag">${esc(x)}</span>`).join("")}</span></div></li>`).join("")}</ul>
          ${changes.length > PN_SHOWN ? `<p class="gpage-hint">And ${changes.length - PN_SHOWN} more.</p>` : ""}` : ""}
        ${n ? `<p class="gpage-hint">${n} ${n === 1 ? "game isn't" : "games aren't"} on this PC, so ${n === 1 ? "it isn't" : "they aren't"} added: Lootnest only keeps the games that are here.
          <button class="gpage-link" data-action="pn-unmatched" aria-expanded="${D.unmatched}">${D.unmatched ? "Hide" : "Show"}</button></p>
          ${D.unmatched ? `<ul class="set-folders">${P.unmatched.slice(0, PN_SHOWN).map((x) => `<li>${esc(x)}</li>`).join("")}${n > PN_SHOWN ? `<li>And ${n - PN_SHOWN} more</li>` : ""}</ul>` : ""}` : ""}`;
    }
    const body = `${where}${D.dir ? who + opts : ""}
      ${D.dir ? `<div class="gpage-sub lo-sec">${gpHead("What changes")}${preview}</div>` : ""}
      ${D.error ? lnInline("error", esc(D.error)) : ""}`;
    return drawer({ label: "Import from Playnite", title: "Import from Playnite", sub: "Lootnest reads Playnite's library on this PC and never changes it. A copy of Lootnest's own records is kept first.", body,
      foot: `<span class="grow"></span><button class="ln-button ln-button--ghost ln-button--sm" data-action="dialog-cancel" ${D.busy ? "disabled" : ""}>Cancel</button>
        <button class="ln-button ln-button--primary ln-button--sm" data-action="pn-go" ${D.busy || D.loading || !changes.length ? "disabled" : ""}>${D.busy ? "Importing" : "Import"}</button>` });
  }
  function playniteEvent(ev) {
    const D = S.dialog && S.dialog.kind === "playnite" ? S.dialog : null;
    if (ev.type === "playnitePlan") {
      if (!D || ev.player !== D.player || ev.req !== D.req) return;
      D.loading = false; D.error = ev.error || ""; D.plan = ev.plan || null;
      if (ev.dir) D.dir = ev.dir;
      renderOverlay();
      return;
    }
    if (ev.error && !ev.id) {
      if (D) { D.busy = false; D.error = ev.error; renderOverlay(); } else toast(ev.error, "error");
      return;
    }
    if (D) closeDialog(true);
    refresh();
    const n = ev.games || 0, undo = { label: "Undo", run: async () => {
      try { await api.playniteUndo(ev.id); await refresh(); toast("Undone. Lootnest's records are as they were before the import."); } catch (e) { toast(errText(e), "error"); }
    } };
    if (ev.error) toast(ev.error, "error", undo);
    else toast(`Imported ${n} ${n === 1 ? "game" : "games"} from Playnite${ev.seconds ? `, with ${hours(ev.seconds)} played` : ""}.`, "", undo);
  }
