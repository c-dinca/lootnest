  // --------------------------------------------------------------- actions --
  const actions = {
    nav(el, ev) {
      if (S.view === "edit" && !confirmLeaveEdit()) return;
      const view = el.dataset.view;
      if (view === S.view) { $("#view").scrollTo({ top: 0, behavior: reduced() ? "auto" : "smooth" }); return; }
      if (ev) uiSound("tab");
      renderBar(view);
      swapPage(() => { enterView(view); S.animateView = true; render(); $("#view").scrollTop = 0; });
    },
    "hist-back"() { goHist(true); },
    "hist-fwd"() { goHist(false); },
    "tabs-more"(el) { moreMenu(el); },
    "me-menu"(el) { meMenu(el); },
    "add-run"(el) {
      const per = player(el.dataset.player);
      if (!per) return;
      if (!S.players) S.players = JSON.parse(JSON.stringify(cfg().players));
      const i = S.players.push({ id: "", name: "", color: per.color, of: per.id }) - 1;
      S.dialog = { kind: "profiles" };
      render();
      const inp = $(`[data-field="pname"][data-i="${i}"]`);
      if (inp) inp.focus();
    },
    "side-toggle"() { setFold(!S.folded); },
    "bp-open"() { bpOpen(); },
    "side-group"(el) {
      const g = el.dataset.g;
      if (S.sideClosed.has(g)) S.sideClosed.delete(g); else S.sideClosed.add(g);
      setPref("sideClosed", JSON.stringify([...S.sideClosed]));
      const grp = el.closest(".sg-group");
      grp.classList.toggle("closed");
      el.setAttribute("aria-expanded", !grp.classList.contains("closed"));
      if (!grp.classList.contains("closed")) { const r = grp.querySelector(".sg-rows"); r.classList.add("opening"); r.addEventListener("animationend", () => r.classList.remove("opening"), { once: true }); watchSideThumbs(); }
      $("#side-games").dataset.html = "";
    },
    "tag-new"() { startNewTag([]); },
    "tag-new-for"() { startNewTag(S.tagPick); },
    "tag-menu"(el) { toggleMenu(el, tagMenuHTML(el.dataset.tag)); },
    "tag-pick"() { const m = $("#ctx"); ctxAt(parseFloat(m.style.left) || 0, parseFloat(m.style.top) || 0, tagPickHTML()); },
    "tag-toggle"(el) { if (el.dataset.on) removeFromTag(S.tagPick, el.dataset.tag); else addToTag(S.tagPick, el.dataset.tag); },
    "tag-out"(el) { removeFromTag(S.tagPick, el.dataset.tag); },
    "lib-views"(el) { toggleMenu(el, viewsMenuHTML()); },
    "view-apply"(el) { applyView(el.dataset.name); },
    "view-new"() { viewNaming(""); },
    "view-rename"(el) { viewNaming(el.dataset.name); },
    "view-save"(el) { saveViewName(el.parentNode.querySelector("#view-name")); },
    async "view-delete"(el) {
      const v = viewsSaved().find((x) => x.name === el.dataset.name);
      if (!v) return;
      try { await api.deleteView(v.name); } catch (e) { toast(errText(e), "error"); return; }
      await refresh();
      toast(`Deleted the view ${v.name}. Your games stay as they are.`, "", { label: "Undo", run: () => api.saveView(v, "").then(refresh, (e) => toast(errText(e), "error")) });
    },
    "sel-tags"(el) { S.tagPick = [...S.libSel]; toggleMenu(el, tagPickHTML(), true); },
    "sel-untag"() { removeFromTag([...S.libSel], S.tag); },
    "sel-status"(el) { toggleMenu(el, selStatusHTML(), true); },
    "sel-status-set"(el) { libSetStatus(el.dataset.status); },
    "sel-seat"() { meMenu($("#bar-me")); },
    "sel-fav"() {
      const keys = [...S.libSel];
      if (keyedItems(keys).every((it) => tagsOf(it).includes(FAV))) removeFromTag(keys, FAV); else addToTag(keys, FAV);
    },
    "sel-hide"() { libHide(); },
    "sel-clear"() { libSelClear(); const t = $(`#grid [data-k="${CSS.escape(S.libAnchor)}"] .ln-tile, #grid [data-k="${CSS.escape(S.libAnchor)}"] .lib-row-open`); if (t) t.focus(); },
    "tag-rename"(el) {
      if (S.folded) setFold(false);
      S.sideNew = null; S.sideRename = el.dataset.tag; renderSideGames();
      const i = $("#sg-rename"); if (i) { i.focus(); i.select(); }
    },
    "tag-move"(el) { tagOp({ op: "move", tag: el.dataset.tag, index: +el.dataset.index }); },
    async "tag-delete"(el) {
      const t = el.dataset.tag;
      if (!(await confirmDialog({ title: `Delete ${t}?`, body: "The games stay in your Library.", ok: "Delete collection" }))) return;
      if (!(await tagOp({ op: "delete", tag: t }))) return;
      if (S.tag === t) S.tag = "";
      if (BP.tag === t) BP.tag = "";
      toast(`Deleted ${t}. Its games are still in your Library.`);
    },
    "side-sort"() {
      const order = Object.keys(SIDE_SORTS);
      S.sideSort = order[(order.indexOf(S.sideSort) + 1) % order.length];
      setPref("sideSort", S.sideSort);
      renderSideGames();
      toast(SIDE_SORTS[S.sideSort]);
    },
    "add-game": startAddDialog,
    "add-steam": (el) => addSteam(el.dataset.app),
    "add-exe": addExe,
    play: (el) => { const p = seated(); if (p) actions["play-as"]({ dataset: { game: el.dataset.game, player: p.id } }); else openPicker(el.dataset.game, "play"); },
    seat(el) { const p = el.dataset.player; setSeat(solo() ? p : S.seat === p ? "" : p); },
    "seat-set"(el) { setSeat(el.dataset.player); },
    async "pt-make"(el) {
      const it = itemFor(el);
      if (!it) return;
      const key = recentKey(it), inp = $("#pt-new"), P = S.points[key] || (S.points[key] = {});
      P.busy = true; render();
      try { await api.makeSavePoint(key, inp ? inp.value : ""); } catch (e) { P.busy = false; toast(errText(e), "error"); render(); }
    },
    "tl-more"(el) { const k = el.dataset.key; S.tlMore[k] = (S.tlMore[k] || TL_SHOWN) + TL_SHOWN; render(); },
    async "pt-restore"(el) {
      const it = itemFor(el);
      if (!it) return;
      const key = recentKey(it);
      const ok = await confirmDialog({ title: `Go back to ${el.dataset.name}?`, body: `The saves of ${it.name} become what they were then. The saves as they are now are kept as a save point first, so you can come back to them too.`, ok: "Go back to it" });
      if (!ok) return;
      const P = S.points[key] || (S.points[key] = {});
      P.busy = true; render();
      try { await api.restoreSavePoint(key, el.dataset.id); } catch (e) { P.busy = false; toast(errText(e), "error"); render(); }
    },
    async "pt-branch"(el) {
      const it = itemFor(el);
      if (!it) return;
      const key = recentKey(it);
      const who = el.dataset.player && player(el.dataset.player) ? pname(el.dataset.player) : "this profile";
      const ok = await confirmDialog({ title: `Start a new run from ${el.dataset.name}?`, body: `A new run of ${who} starts with the saves of that moment: a profile of its own, with its own saves from then on. Everyone else's stay as they are.`, ok: "Start the run" });
      if (!ok) return;
      try { const id = await api.branchSavePoint(key, el.dataset.id, ""); await refresh(); toast(`New run: ${pname(id)}. Choose it when you play.`); } catch (e) { toast(errText(e), "error"); }
    },
    "pt-rename"(el) { ptEdit = el.dataset.id; render(); const r = $("#pt-rename"); if (r) { r.focus(); r.select(); } },
    "pt-rename-cancel"() { ptEdit = null; render(); },
    async "pt-rename-save"(el) {
      const it = itemFor(el), inp = $("#pt-rename");
      if (!it || !inp) return;
      const key = recentKey(it);
      try { await api.renameSavePoint(key, el.dataset.id, inp.value); ptEdit = null; api.savePoints(key).catch(() => {}); } catch (e) { toast(errText(e), "error"); }
    },
    async "pt-delete"(el) {
      const it = itemFor(el);
      if (!it) return;
      const key = recentKey(it);
      const ok = await confirmDialog({ title: `Delete ${el.dataset.name}?`, body: "This save point's copy of the saves goes. The game's saves now aren't touched.", ok: "Delete it", danger: true });
      if (!ok) return;
      try { await api.deleteSavePoint(key, el.dataset.id); } catch (e) { toast(errText(e), "error"); }
    },
    async "offline-toggle"(el) {
      const it = itemFor(el);
      if (!it) return;
      const on = el.dataset.on === "1";
      const ok = await confirmDialog(on
        ? { title: `Keep ${it.name} offline?`, body: "Windows Firewall will stop its programs from connecting to the internet, and to other PCs on your network. Online play, updates from inside the game and anything else online stop working until you let it go online again. Windows asks first, as administrator.", ok: "Keep it offline" }
        : { title: `Let ${it.name} go online again?`, body: "Lootnest takes away the Windows Firewall rules it made for this game. Windows asks first, as administrator.", ok: "Let it go online" });
      if (!ok) return;
      try { await api.setOffline(recentKey(it), on); toast("Waiting for Windows' prompt"); } catch (e) { toast(errText(e), "error"); }
    },
    "ach-doctor"(el) {
      const it = itemFor(el);
      if (!it) return;
      const key = recentKey(it);
      delete S.doctor[key];
      const open = S.dialog && S.dialog.kind === "achdoc";
      S.dialog = { kind: "achdoc", ik: it.kind, iid: it.id };
      renderOverlay(); if (!open) focusOverlay();
      api.achDoctor(key, it.appId, docDir(it), it.name).catch((e) => { S.doctor[key] = { error: errText(e) }; renderOverlay(); });
    },
    async "doc-name"(el) {
      const it = itemFor(el);
      if (!it) return;
      try { await api.setNameInGame(recentKey(it), docDir(it), el.dataset.player); toast(`The name in ${it.name} is now ${pname(el.dataset.player)}`); actions["ach-doctor"](el); }
      catch (e) { toast(errText(e), "error"); }
    },
    async "show-log-nav"() {
      S.dialog = null; renderOverlay(); S.setTab = "data"; S.view = "settings"; S.menuFor = null; S.animateView = true;
      try { S.logText = (await api.log()) || "Nothing yet."; } catch (e) { S.logText = errText(e); }
      render(); $("#view").scrollTop = 0;
    },
    "game-tab"(el) { setGameTab(el.dataset.tab); },
    manage(el) { const it = itemFor(el); if (it) { S.dialog = { kind: "manage", ik: it.kind, iid: it.id }; renderOverlay(); focusOverlay(); } },
    "ach-who"(el) { S.achWho[el.dataset.key] = el.dataset.player; render(); },
    "ach-page"(el) { S.achPage = el.dataset.player; render(); },
    "status-who"(el) { S.statusWho[el.dataset.key] = el.dataset.player; render(); },
    async "set-status"(el) {
      const { key, player: pid, status } = el.dataset, cur = shelfAt(pid, key);
      await saveShelf(pid, key, { status: cur.status === status ? "" : status }); // pressing it again clears it
    },
    async rate(el) {
      const { key, player: pid } = el.dataset, n = +el.dataset.stars, cur = shelfAt(pid, key);
      const r = cur.rating === n ? 0 : n; // the current star again clears it
      if (await saveShelf(pid, key, { rating: r })) { const b = $(`.gp-rate [data-stars="${r || n}"]`); if (b) b.focus(); }
    },
    "hours-edit"(el) {
      if (el.getAttribute("aria-expanded") === "true") { actions["hours-cancel"](); return; }
      S.hoursEdit = { key: el.dataset.key, player: el.dataset.player }; render(); const i = $("#gp-hours"); if (i) { i.focus(); i.select(); } },
    "hours-cancel"() { const E = S.hoursEdit; S.hoursEdit = null; render(); if (E) hoursBack(E.player); },
    async "hours-save"() {
      const E = S.hoursEdit, i = $("#gp-hours");
      if (!E || !i) return;
      if (i.value === i.defaultValue) { actions["hours-cancel"](); return; }
      const v = parseFloat(i.value);
      if (!isFinite(v) || v < 0 || v > 100000) { toast("Hours played is a number from 0 to 100,000.", "error"); i.focus(); return; }
      const off = shelfAt(E.player, E.key).offset || 0, shown = ((S.d.playtime || {})[E.key] || {})[E.player] || 0;
      S.hoursEdit = null;
      if (await saveShelf(E.player, E.key, { offset: Math.round(v * 3600) - (shown - off) })) hoursBack(E.player); else S.hoursEdit = E;
    },
    async "hours-reset"() {
      const E = S.hoursEdit; if (!E) return;
      S.hoursEdit = null;
      if (await saveShelf(E.player, E.key, { offset: 0 })) hoursBack(E.player); else S.hoursEdit = E;
    },
    async "tag-add"(el) {
      const key = el.dataset.key, i = $("#tag-new"), t = ((i && i.value) || "").trim();
      if (!t) { if (i) i.focus(); return; }
      const it = itemByKey(key), cur = it ? tagsOf(it) : [];
      try { await api.setTags(key, cur.concat([t])); } catch (e) { toast(errText(e), "error"); return; }
      await refresh();
      const n = $("#tag-new"); if (n) n.focus();
    },
    async "tag-remove"(el) {
      const key = el.dataset.key, it = itemByKey(key);
      try { await api.setTags(key, (it ? tagsOf(it) : []).filter((t) => t !== el.dataset.tag)); } catch (e) { toast(errText(e), "error"); return; }
      if (S.tag === el.dataset.tag && !Object.entries(cfg().tags || {}).some(([k, ts]) => k !== key && ts.includes(el.dataset.tag))) S.tag = "";
      await refresh();
    },
    async "ach-shared"(el) {
      const id = el.dataset.game, shared = el.dataset.on === "1", g = game(id);
      if (shared && !(await confirmDialog({ title: "Share one set of achievements?", body: `Everyone who plays ${g ? g.name : "this game"} will see the same achievements, the ones on the PC now. Each profile's own stay with it, and come back if you change your mind.`, ok: "Share one set" }))) return;
      try { await api.setSharedAchievements(id, shared); } catch (e) { toast(errText(e), "error"); return; }
      await refresh();
      achAsked.delete("game:" + id);
      toast(shared ? "Achievements are shared now." : "Each profile gets its own achievements from the next swap.");
    },
    "open-sheet": (el) => openSheet(el.dataset.game ? "game" : isOwnedKey(el.dataset.key) ? "owned" : "found", el.dataset.game || el.dataset.key, el.closest(".card")),
    "back-library": () => pageBack(),
    "open-player"(el) { S.playerPage = el.dataset.player; S.view = "player"; S.animateView = true; render(); $("#view").scrollTop = 0; },
    "edit-profile"(el) { const p = player(el.dataset.player); if (!p) return; S.dialog = { kind: "profile", p: JSON.parse(JSON.stringify(p)), q: "", tab: el.dataset.tab || "look" }; renderOverlay(); },
    "pe-tab"(el) { S.dialog.tab = el.dataset.tab; renderOverlay(); },
    "pe-set"(el) { S.dialog.p[el.dataset.f] = el.dataset.v; renderOverlay(); },
    "pe-color"(el) { S.dialog.p.color = el.dataset.color; renderOverlay(); },
    "pe-banner"(el) {
      const D = S.dialog, v = el.dataset.v;
      if (v === "image") { if (!(S.d.banners || {})[D.p.id]) return actions["banner-pick"](); D.p.banner = "image"; }
      else if (v === "game") { const f = (D.p.favorites || [])[0] || (libraryItems()[0] && recentKey(libraryItems()[0])) || ""; D.p.banner = D.p.banner && D.p.banner !== "image" && !D.p.banner.startsWith("style:") ? D.p.banner : f; }
      else D.p.banner = v === "plain" ? "style:plain" : "";
      renderOverlay();
    },
    async "banner-pick"() {
      const D = S.dialog;
      const r = await browse("image");
      if (!r.path) { if (r.error) toast(r.error, "error"); return; }
      D.busy = "banner"; renderOverlay();
      try { await api.setBanner(D.p.id, r.path); } catch (e) { D.busy = false; renderOverlay(); toast(errText(e), "error"); }
    },
    async "banner-remove"() {
      try { await api.removeBanner(S.dialog.p.id); if (S.dialog.p.banner === "image") S.dialog.p.banner = ""; await refresh(); } catch (e) { toast(errText(e), "error"); }
    },
    "pe-fav"(el) {
      const f = S.dialog.p.favorites = S.dialog.p.favorites || [];
      const i = f.indexOf(el.dataset.k);
      if (i >= 0) f.splice(i, 1); else if (f.length < 6) f.push(el.dataset.k);
      renderOverlay();
    },
    "pe-fav-move"(el) {
      const f = S.dialog.p.favorites, i = +el.dataset.i, j = i + +el.dataset.d;
      if (j < 0 || j >= f.length) return;
      [f[i], f[j]] = [f[j], f[i]]; renderOverlay();
    },
    "pe-sec"(el) {
      const p = S.dialog.p, on = sectionsOf(p).slice(), i = on.indexOf(el.dataset.k);
      if (i >= 0) { if (on.length > 1) on.splice(i, 1); } else on.push(el.dataset.k);
      p.sections = on; renderOverlay();
    },
    "pe-sec-move"(el) {
      const p = S.dialog.p, on = sectionsOf(p).slice(), i = on.indexOf(el.dataset.k), j = i + +el.dataset.d;
      if (i < 0 || j < 0 || j >= on.length) return;
      [on[i], on[j]] = [on[j], on[i]]; p.sections = on; renderOverlay();
    },
    async "avatar-pick"() {
      const D = S.dialog;
      const r = await browse("image");
      if (!r.path) { if (r.error) toast(r.error, "error"); return; }
      D.busy = "avatar"; renderOverlay();
      try { await api.setAvatar(D.p.id, r.path); } catch (e) { D.busy = false; renderOverlay(); toast(errText(e), "error"); }
    },
    async "avatar-remove"() { try { await api.removeAvatar(S.dialog.p.id); S.dialog.p.avatar = false; await refresh(); } catch (e) { toast(errText(e), "error"); } },
    async "pe-save"() {
      const D = S.dialog, p = { ...D.p };
      p.avatar = !!(S.d.avatars || {})[p.id];
      if (p.banner === "image" && !(S.d.banners || {})[p.id]) p.banner = "";
      p.bannerY = Math.round(+p.bannerY || 0);
      const ps = cfg().players.map((x) => (x.id === p.id ? p : x));
      try { await api.savePlayers(ps); S.dialog = null; S.players = null; await refresh(); toast("Profile saved"); }
      catch (e) { const t = $("#pe-error"); if (t) t.textContent = errText(e); }
    },
    async "gs-add"(el) {
      const key = el.dataset.key, G = S.gsaves[key] || {};
      const r = await browse(el.dataset.kind === "file" ? "file" : "folder");
      if (!r.path) { if (r.error) toast(r.error, "error"); return; }
      const now = G.source === "you" ? (G.locations || []).map((l) => l.path) : [];
      try { await api.setGameSaves(key, [...now, r.tokenized || r.path]); await refresh(); delete S.gsaves[key]; gsAsked.delete(key); render(); }
      catch (e) { toast(errText(e), "error"); }
    },
    async "gs-remove"(el) {
      const key = el.dataset.key, G = S.gsaves[key] || {};
      const rest = (G.locations || []).map((l) => l.path).filter((p) => p !== el.dataset.path);
      try { await api.setGameSaves(key, rest); await refresh(); delete S.gsaves[key]; gsAsked.delete(key); render(); }
      catch (e) { toast(errText(e), "error"); }
    },
    "gs-own"(el) { const G = S.gsaves[el.dataset.key]; if (G) { G.source = "you"; G.locations = []; G.versions = G.versions || []; render(); } },
    "gs-open"(el) { api.open("gameSaves", el.dataset.key).catch((e) => toast(errText(e), "error")); },
    "gs-pcgw"(el) { api.open("pcgw", el.dataset.name).catch((e) => toast(errText(e), "error")); },
    async "gs-now"(el) {
      const G = S.gsaves[el.dataset.key];
      if (G) G.busy = true; render();
      try { await api.gameSaveNow(el.dataset.key); } catch (e) { if (G) G.busy = false; toast(errText(e), "error"); render(); }
    },
    async "gs-restore"(el) {
      const when = new Date(el.dataset.when).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
      const local = !!el.dataset.local;
      const body = local ? `The copy from ${when} goes back into the game's folder, over the saves there now. Those are kept in Lootnest's backups first.`
        : `The saves from ${when} on ${el.dataset.pc} replace the ones on this PC. The saves here now are kept in Lootnest's backups first.`;
      const ok = await confirmDialog({ title: "Restore these saves?", body, ok: "Restore" });
      if (!ok) return;
      const G = S.gsaves[el.dataset.key];
      if (G) G.busy = true; render();
      try { await (local ? api.gameSaveRestoreLocal : api.gameSaveRestore)(el.dataset.key, el.dataset.id); } catch (e) { if (G) G.busy = false; toast(errText(e), "error"); render(); }
    },
    async "find-on"() {
      const s = JSON.parse(JSON.stringify(cfg().settings));
      s.findGames = true;
      try { await api.saveSettings({ settings: s, autostart: S.d.autostart }); await refresh(); setTimeout(() => { const q = $("#find-q"); if (q) q.focus(); }, 30); } catch (e) { toast(errText(e), "error"); }
    },
    "find-open"(el) {
      const appId = +el.dataset.app;
      S.dialog = { kind: "offer", appId, name: el.dataset.name };
      const R = S.offers[appId];
      if (!R || R.error) { delete S.offers[appId]; api.gameOffer(appId, el.dataset.name).catch((e) => { S.offers[appId] = { error: errText(e) }; renderOverlay(); }); }
      renderOverlay(); focusOverlay();
    },
    async "copy-carry"(el) {
      const old = el.dataset.old, nw = el.dataset.new;
      const ok = await confirmDialog({ title: `Carry ${el.dataset.name} over to its new copy?`, body: "Its saves go into the new copy's folder, at the same places (what the new copy has there is kept in backups first). Its playtime, status, collections and settings move to the new copy. The old folder isn't touched.", ok: "Carry over" });
      if (!ok) return;
      S.carried[old] = { busy: true, name: el.dataset.name }; render();
      try { await api.carryOver(old, nw); } catch (e) { S.carried[old] = { error: errText(e) }; toast(errText(e), "error"); render(); }
    },
    async "copy-dismiss"(el) {
      try { await api.dismissCopy(el.dataset.old, el.dataset.new); S.copies = (S.copies || []).filter((o) => !(o.old.key === el.dataset.old && o.new.key === el.dataset.new)); updateNavCount(); render(); } catch (e) { toast(errText(e), "error"); }
    },
    async "copy-delete-old"(el) {
      const ok = await confirmDialog({ title: "Delete the old folder?", body: "Its saves are copied to Lootnest's backups first. Everything else in the old folder is deleted.", ok: "Delete it", danger: true });
      if (!ok) return;
      const C = S.carried[el.dataset.old] || {}; C.busy = true; S.carried[el.dataset.old] = C; render();
      try { await api.deleteOldCopy(el.dataset.old); } catch (e) { C.busy = false; toast(errText(e), "error"); render(); }
    },
    async "upd-keep"(el) { try { await api.keepSavesNow(el.dataset.key); toast("Copying its saves"); } catch (e) { toast(errText(e), "error"); } },
    async "report-problem"() { try { await api.reportProblem(); toast("Copied. Paste it into the new issue."); } catch (e) { toast(errText(e), "error"); } },
    "ach-reveal"(el) { S.revealed.add(el.dataset.k); render(); },
    "test-toast"() {
      const demo = [{ name: "This is how an unlock looks", percent: 4.2 }];
      achPop(demo, "", (seated() || {}).id); unlockSound(demo);
      api.testToast().then(() => toast("A note shows at the bottom right of the screen for five seconds. If none does, Show recent activity says why.")).catch((e) => toast(errText(e), "error")); },
    "upd-another"() { const D = S.dialog; if (D) { Object.assign(D, { path: null, plan: null, check: null, done: null, undone: false, error: "", step: "", full: undefined, itch: false }); renderOverlay(); } },
    "upd-full"(el) { const D = S.dialog; if (D) D.full = el.checked; },
    async "upd-run"() {
      const D = S.dialog;
      if (!D || !D.plan) return;
      D.busy = true; D.step = ""; D.error = ""; renderOverlay();
      try { await api.updateInstaller(D.key, D.path, D.full); } catch (e) { D.busy = false; D.error = errText(e); renderOverlay(); }
    },
    "dl-review"(el) {
      const D = S.dialog = { kind: "update", key: el.dataset.key, name: el.dataset.name, path: el.dataset.path, busy: true };
      renderOverlay(); focusOverlay();
      api.updatePlan(D.key, D.path).catch((e) => { D.busy = false; D.error = errText(e); renderOverlay(); });
    },
    async "dl-recycle"(el) {
      try { await api.recycleDownload(el.dataset.path); S.dl = (S.dl || []).filter((m) => m.path !== el.dataset.path); updateNavCount(); render(); if (S.dialog) renderOverlay(); toast("Moved to the Recycle Bin"); }
      catch (e) { toast(errText(e), "error"); }
    },
    async "dl-ignore"(el) {
      S.dl = (S.dl || []).filter((m) => m.path !== el.dataset.path); updateNavCount(); render();
      try { await api.ignoreDownload(el.dataset.path); } catch (e) { toast(errText(e), "error"); }
    },
    "upd-start"(el) { S.dialog = { kind: "update", key: el.dataset.key, name: el.dataset.name }; renderOverlay(); focusOverlay(); },
    async "upd-pick"(el) {
      const D = S.dialog;
      if (!D || D.kind !== "update") return;
      const r = await browse(el.dataset.kind, "");
      if (r.cancelled || !r.path) return;
      if (r.error) { toast(r.error, "error"); return; }
      Object.assign(D, { path: r.path, plan: null, error: "", busy: true, done: null });
      renderOverlay();
      api.updatePlan(D.key, r.path).catch((e) => { D.busy = false; D.error = errText(e); renderOverlay(); });
    },
    async "upd-apply"() {
      const D = S.dialog;
      if (!D || !D.plan) return;
      D.busy = true; D.error = ""; renderOverlay();
      try { await api.updateApply(D.key, D.path); } catch (e) { D.busy = false; D.error = errText(e); renderOverlay(); }
    },
    async "upd-undo"(el) {
      const ok = await confirmDialog({ title: `Undo the update of ${el.dataset.name}?`, body: "The files it added go, and the ones it replaced come back. The saves stay as they are now.", ok: "Undo" });
      if (!ok) return;
      if (!S.dialog || S.dialog.kind !== "update") S.dialog = { kind: "update", key: el.dataset.key, name: el.dataset.name, busy: true };
      else S.dialog.busy = true;
      renderOverlay();
      try { await api.updateUndo(el.dataset.key, el.dataset.id); } catch (e) { S.dialog.busy = false; S.dialog.error = errText(e); renderOverlay(); }
    },
    async copies(el) {
      const id = el.dataset.game;
      S.dialog = { kind: "copies", gameId: id, list: null }; renderOverlay(); focusOverlay();
      try { const l = await api.gameBackups(id); if (S.dialog && S.dialog.kind === "copies") { S.dialog.list = l || []; renderOverlay(); } } catch (e) { toast(errText(e), "error"); }
    },
    async "copy-put-back"(el) {
      const back = S.dialog && S.dialog.kind === "copies" ? { ...S.dialog } : null;
      const g = game(el.dataset.game), who = pname(el.dataset.owner);
      const loaded = g && S.d.active[g.id] === el.dataset.owner;
      const ok = await confirmDialog({ title: `Put back ${possessive(who)} files from ${el.dataset.when}?`,
        body: loaded ? `${who}'s files are loaded now, so the copy goes into the game's folders. The files there now are copied first.` : `The copy goes into ${possessive(who)} profile, and comes in the next time ${who} plays. ${who}'s profile as it is now is copied first.`, ok: "Put it back" });
      if (back) { S.dialog = back; renderOverlay(); }
      if (!ok) return;
      if (S.dialog) { S.dialog.busy = true; renderOverlay(); }
      try {
        await api.restoreBackup(el.dataset.game, el.dataset.id);
        toast(`${possessive(who)} files from ${el.dataset.when} are back.`);
        const l = await api.gameBackups(el.dataset.game);
        if (S.dialog && S.dialog.kind === "copies") { S.dialog.busy = false; S.dialog.list = l || []; renderOverlay(); }
      } catch (e) { if (S.dialog) { S.dialog.busy = false; renderOverlay(); } toast(errText(e), "error"); }
    },
    async "conflict-ok"(el) { try { await api.dismissConflict(el.dataset.game); await refresh(); } catch (e) { toast(errText(e), "error"); } },
    "upd-filter"(el) { S.updFilter = el.dataset.f === S.updFilter ? "all" : el.dataset.f; render(); },
    "upd-refresh"() { S.updChecking = true; S.copies = null; render(); api.updates().catch((e) => { S.updChecking = false; toast(errText(e), "error"); render(); }); },
    "upd-notes"(el) {
      const r = S.updates && S.updates.games.find((g) => g.key === el.dataset.key);
      const it = r && updItem(r.key);
      if (it) openSheet(it.kind, it.id, null); // the game's page has its news, with the patch notes on top
    },
    "sc-games"(el) {
      const kind = el.dataset.kind, list = (S.saveCheck && S.saveCheck[kind]) || [];
      const title = { gone: "Saves gone from their folders", oneDrive: "Saves OneDrive syncs" }[kind] || "Where saves are kept";
      const m = menuUnder(el, `<div class="ctx-title">${title}</div>${list.map((g) => `<button data-action="open-sheet" ${g.key.startsWith("game:") ? `data-game="${esc(g.key.slice(5))}"` : `data-key="${esc(g.key)}"`}>${esc(g.name)}</button>`).join("")}`);
      const f = m.querySelector("button"); if (f) f.focus({ preventScroll: true });
    },
    "upd-history"(el) { S.dialog = { kind: "updhist", key: el.dataset.key }; renderOverlay(); focusOverlay(); },
    "upd-more"(el) {
      const k = esc(el.dataset.key), n = esc(el.dataset.name);
      const r = S.updates && S.updates.games.find((g) => g.key === el.dataset.key);
      const last = r && (r.applied || []).find((u) => u.state === "applied");
      const m = $("#ctx");
      m.innerHTML = `<div class="ctx-title">${n}</div>
        <button data-action="upd-history" data-key="${k}">Update history${r && r.applied && r.applied.length ? ` (${r.applied.length})` : ""}</button>
        ${last && !last.noUndo ? `<button data-action="upd-undo" data-key="${k}" data-id="${esc(last.id)}" data-name="${n}">Undo the last update</button>` : ""}
        <button data-action="upd-keep" data-key="${k}">Copy its saves first</button><hr>
        <button data-action="upd-open-game" data-key="${k}">Open the game's page</button>`;
      m.hidden = false;
      const b = el.getBoundingClientRect(), rr = m.getBoundingClientRect();
      m.style.left = Math.max(8, Math.min(b.right - rr.width, innerWidth - rr.width - 8)) + "px";
      m.style.top = Math.min(b.bottom + 4, innerHeight - rr.height - 8) + "px";
      const f = m.querySelector("button"); if (f) f.focus({ preventScroll: true });
    },
    "upd-open-game"(el) {
      const [kind, ...rest] = el.dataset.key.split(":"), id = rest.join(":");
      openSheet(kind === "game" ? "game" : "found", id, null);
    },
    "store-retry"() { S.front = null; const r = $("#find-results"); if (r) r.innerHTML = findResults(); },
    "gq-open"(el) { gqOpen(+el.dataset.i); },
    "gq-all"() { gqAll(); },
    "find-store"(el) { api.open("store", el.dataset.url).catch((e) => toast(errText(e), "error")); },
    "find-library"(el) {
      const o = el.dataset.owned || "", [kind, ...rest] = o.split(":"), id = rest.join(":");
      const it = everyItem().find((x) => (kind === "game" ? x.kind === "game" : x.kind === "found") && x.id === id);
      if (!it) { toast("That game isn't in the Library any more.", "error"); return; }
      S.dialog = null; renderOverlay();
      openSheet(it.kind, it.id, null);
    },
    async "savedb-on"() {
      const s = JSON.parse(JSON.stringify(cfg().settings));
      s.saveDb = true;
      try { await api.saveSettings({ settings: s, autostart: S.d.autostart }); toast("Downloading the save locations list"); await refresh(); } catch (e) { toast(errText(e), "error"); }
    },
    async "sync-now"() { S.syncAsked = true; try { await api.syncNow(); } catch (e) { S.syncAsked = false; toast(errText(e), "error"); } },
    async "sync-resolve"(el) {
      try { S.syncAsked = true; await api.syncResolve(el.dataset.game, el.dataset.player, el.dataset.answer); } catch (e) { S.syncAsked = false; toast(errText(e), "error"); }
    },
    "open-download"(el) { api.open("link", el.dataset.url).catch((e) => toast(errText(e), "error")); },
    "backup-use"(el) { saveBackup({ dir: el.dataset.dir, auto: true }).then(() => toast("Saves go to " + el.dataset.dir)); },
    async "backup-pick"() {
      const r = await browse("backupDir");
      if (!r.path) { if (r.error) toast(r.error, "error"); return; }
      const first = !S.backup.dir;
      await saveBackup({ dir: r.path, ...(first ? { auto: true } : {}) });
    },
    "backup-open": () => api.open("backupFolder").catch((e) => toast(errText(e), "error")),
    "copy-path": (el) => api.copy(el.dataset.path).then(() => toast("Copied"), (e) => toast(errText(e), "error")),
    async "backup-now"() { S.backupAsked = true; try { await api.backupNow(); } catch (e) { S.backupAsked = false; toast(errText(e), "error"); } loadBackupInfo(); },
    "backup-pw"(el) { S.dialog = { kind: "backup-pw", on: el.dataset.on === "1" }; renderOverlay(); setTimeout(() => { const f = $("#bk-pw"); if (f) f.focus(); }, 30); },
    async "backup-pw-save"() {
      const D = S.dialog;
      let pw = "";
      if (D.on) {
        pw = $("#bk-pw").value;
        const unlock = S.backup && S.backup.locked;
        if (!unlock && pw !== $("#bk-pw2").value) { $("#bk-error").textContent = "The two passwords aren't the same."; return; }
        if ([...pw].length < 8) { $("#bk-error").textContent = "Use at least 8 characters."; return; }
      }
      D.busy = true; D.error = ""; renderOverlay();
      try { await api.setBackupPassword(pw); } catch (e) { D.busy = false; D.error = errText(e); renderOverlay(); }
    },
    async "restore-pick"(el) {
      const r = await browse("backup");
      if (!r.path) { if (r.error) toast(r.error, "error"); return; }
      S.dialog = { kind: "restore", file: r.path, busy: true, mode: (el && el.dataset.mode) || "restore" }; renderOverlay();
      try { await api.restoreOpen(r.path, ""); } catch (e) { S.dialog.busy = false; S.dialog.info = { error: errText(e) }; renderOverlay(); }
    },
    async "restore-pw"() {
      const D = S.dialog, pw = $("#rs-pw").value;
      if (!pw) return;
      D.busy = true; D.triedPassword = true; D.error = ""; renderOverlay();
      try { await api.restoreOpen(D.file, pw); } catch (e) { D.busy = false; D.error = errText(e); renderOverlay(); }
    },
    async "restore-apply"() {
      const D = S.dialog;
      D.busy = true; D.applying = true; D.error = ""; renderOverlay();
      try { await (D.info && D.info.kind === "player" ? api.importApply() : api.restoreApply()); } catch (e) { D.busy = false; D.applying = false; D.error = errText(e); renderOverlay(); }
    },
    "export-player"(el) { S.dialog = { kind: "export", pid: el.dataset.player }; renderOverlay(); },
    async "export-save"() {
      const D = S.dialog, pw = $("#ex-pw").value;
      if (pw && pw !== $("#ex-pw2").value) { D.error = "The two passwords aren't the same."; renderOverlay(); return; }
      if (pw && [...pw].length < 8) { D.error = "Use at least 8 characters, or no password."; renderOverlay(); return; }
      const r = await browse("exportDir", (S.backup && S.backup.target) || "");
      if (!r.path) { if (r.error) toast(r.error, "error"); return; }
      D.busy = true; D.error = ""; renderOverlay();
      try { await api.exportPlayer(D.pid, r.path, pw); } catch (e) { D.busy = false; D.error = errText(e); renderOverlay(); }
    },
    "restore-close"() { const D = S.dialog; S.dialog = null; renderOverlay(); if (D && !D.done) api.restoreCancel().catch(() => {}); },
    "launch-from-exe"() { const g = S.edit.game; g.launch = { ...g.launch, type: "exe", uri: "" }; render(); },
    density(el) { S.density = el.dataset.size; setPref("density", S.density); render(); },
    "open-news": (el) => api.open("news", el.dataset.url).catch((e) => toast(errText(e), "error")),
    "open-link": (el) => api.open("link", el.dataset.url).catch((e) => toast(errText(e), "error")),
    "read-news"(el) { S.dialog = { kind: "news", key: el.dataset.key, i: +el.dataset.i }; renderOverlay(); setTimeout(() => { const b = $(".reader .btn.quiet.icon"); if (b) b.focus({ preventScroll: true }); }, 30); },
    async "news-on"() {
      const s = JSON.parse(JSON.stringify(cfg().settings));
      s.gameNews = true;
      try { await api.saveSettings({ settings: s, autostart: S.d.autostart }); await refresh(); } catch (e) { toast(errText(e), "error"); }
    },
    // Game details: the quiet link opens Settings, Online, where the switch is.
    "details-on"() { S.setTab = "online"; setPref("setTab", S.setTab); actions.nav({ dataset: { view: "settings" } }); },
    "details-more"(el) {
      const k = el.dataset.k, open = !S.detOpen.has(k), p = $("#det-sum");
      if (open) S.detOpen.add(k); else S.detOpen.delete(k);
      if (p) p.classList.toggle("open", open);
      el.textContent = open ? "Show less" : "Show more"; el.setAttribute("aria-expanded", open);
    },
    "details-edit"(el) { const it = itemFor(el); if (it) { S.dialog = { kind: "details-edit", ik: it.kind, iid: it.id }; renderOverlay(); focusOverlay(); } },
    async "details-save"(el) {
      const it = itemFor(el), D = S.dialog;
      if (!it || !D) return;
      const v = (id) => $("#" + id).value;
      const edits = { summary: v("de-summary"), developer: v("de-developer"), publisher: v("de-publisher"), released: v("de-released"), genres: v("de-genres").split(",") };
      D.busy = true; renderOverlay();
      try { await api.saveDetailsEdit(recentKey(it), edits); closeDialog(false); toast("Details saved"); } catch (e) { D.busy = false; renderOverlay(); toast(errText(e), "error"); }
    },
    async "details-reset"(el) {
      const it = itemFor(el);
      if (!it) return;
      try { await api.saveDetailsEdit(recentKey(it), {}); closeDialog(false); toast("Back to Steam's details"); } catch (e) { toast(errText(e), "error"); }
    },
    "shot-open"(el) { const it = itemFor(el); if (it) { S.dialog = { kind: "shots", ik: it.kind, iid: it.id, i: +el.dataset.i }; renderOverlay(); focusOverlay(); } },
    "capture-open"(el) { api.openCapture(el.dataset.path).catch((e) => toast(errText(e), "error")); },
    "playnite-open": () => openPlaynite(),
    "itch-keys": () => api.open("store", ITCH_KEYS).catch((e) => toast(errText(e), "error")),
    async "itch-connect"() {
      const I = S.itch, k = ($("#s-itchkey") || {}).value || "";
      Object.assign(I, { busy: true, error: "" }); render();
      try { await api.itchConnect(k); } catch (e) { Object.assign(I, { busy: false, error: errText(e) }); render(); }
    },
    async "itch-disconnect"() {
      const ok = await confirmDialog({ title: "Disconnect itch.io?", body: "Lootnest forgets your API key and stops listing your itch.io games. The games you installed from itch.io stay, with their saves.", ok: "Disconnect" });
      if (!ok) return;
      try { await api.itchDisconnect(); Object.assign(S.itch, { updates: [], error: "" }); await refresh(); toast("itch.io is disconnected."); } catch (e) { toast(errText(e), "error"); }
    },
    "itch-refresh"() { S.itch.refreshing = true; render(); api.itchRefresh().catch((e) => { S.itch.refreshing = false; toast(errText(e), "error"); render(); }); },
    async "itch-dir"() {
      const D = S.dialog, r = await browse("itchDir", D.dir);
      if (r.path && S.dialog === D) { D.dir = r.path; D.free = -1; renderOverlay(); api.itchUploads(D.gameId, r.path).catch(() => {}); }
    },
    async "itch-install"(el) {
      const D = S.dialog;
      if (!D || D.kind !== "itch") return;
      D.busy = true; D.error = ""; renderOverlay();
      try { await api.itchInstall({ gameId: D.gameId, uploadId: +el.dataset.upload, parent: D.dir }); } catch (e) { D.busy = false; D.error = errText(e); renderOverlay(); }
    },
    "itch-cancel": () => api.itchCancel().catch(() => {}),
    "itch-check"(el) { Object.assign(S.itch, { checking: true, asked: true }); render(); api.itchCheck(el.dataset.key, true).catch((e) => { S.itch.checking = false; toast(errText(e), "error"); render(); }); },
    async "itch-update"(el) { try { await api.itchUpdate(el.dataset.key); } catch (e) { toast(errText(e), "error"); } },
    async "pn-dir"() {
      const D = S.dialog, r = await browse("playniteDir", D.dir);
      if (r.path && S.dialog === D) { D.dir = r.path; playnitePreview(D); }
      else if (r.error) toast(r.error, "error");
    },
    "pn-unmatched"() { S.dialog.unmatched = !S.dialog.unmatched; renderOverlay(); },
    async "pn-go"() {
      const D = S.dialog;
      D.busy = true; D.error = ""; renderOverlay();
      try { await api.playniteApply(D.dir, D.player, D.opts); } catch (e) { D.busy = false; D.error = errText(e); if (S.dialog === D) renderOverlay(); }
    },
    "captures-unsorted"() { S.dialog = { kind: "unsorted" }; renderOverlay(); focusOverlay(); api.unsortedCaptures().catch((e) => toast(errText(e), "error")); },
    "pic-open"(el) {
      const it = itemFor(el);
      if (!it) return;
      const pics = pagePictures(it), i = Math.max(0, pics.findIndex((p) => p.ref === el.dataset.ref));
      if (!pics.length) return;
      S.dialog = { kind: "shots", ik: it.kind, iid: it.id, i, pics, full: {} };
      renderOverlay(); focusOverlay();
    },
    "shot-step"(el) {
      if (!S.dialog || S.dialog.kind !== "shots") return;
      const a = document.activeElement, label = a && $("#overlay").contains(a) && a.getAttribute("aria-label");
      S.dialog.i += +el.dataset.d; renderOverlay();
      ((label && $(`#overlay [aria-label="${label}"]`)) || $('#overlay [data-action="dialog-cancel"]')).focus();
    },
    "play-as"(el) { openPicker(el.dataset.game, "play"); S.picker.playerId = el.dataset.player; S.picker.from = S.d.active[el.dataset.game] || ""; runPick(); },
    async "play-found"(el) {
      const f = (S.found.games || []).find((x) => x.key === el.dataset.key);
      if (S.dialog && S.dialog.kind === "sheet") closeDialog(false);
      try { await api.launchFound(el.dataset.key); toast(`Starting ${f ? f.name : "the game"}`); } catch (e) { toast(errText(e), "error"); }
    },
    filter(el) { S.filter = el.dataset.filter; render(); },
    async "close-game"(el) {
      const g = game(el.dataset.game), s = g && session(g.id);
      if (!g) return;
      const n = s ? pname(s.playerId) : "";
      const ok = await confirmDialog({ title: `Close ${g.name}?`, body: `The game is asked to quit, as if you closed its window. Anything it hasn't saved yet may be lost.${n ? ` Once it closes, ${n}'s progress goes back to their profile.` : ""}`, ok: "Close the game", cancel: "Keep playing", safe: true });
      if (!ok) return;
      try { await api.closeGame(g.id); toast(`Asked ${g.name} to close`); } catch (e) { toast(errText(e), "error"); }
    },
    who(el) { S.who = S.who === el.dataset.player ? "" : el.dataset.player; render(); },
    async "covers-on"() {
      const s = JSON.parse(JSON.stringify(cfg().settings));
      s.downloadCovers = true;
      try { await api.saveSettings({ settings: s, autostart: S.d.autostart }); S.covers = {}; S.icons = {}; S.art = {}; artWanted.clear(); pendingCovers.clear(); await refresh(); } catch (e) { toast(errText(e), "error"); }
    },
    "covers-later"() { S.hideCoverHint = true; render(); },
    "restart-normal"() { api.restart(false).catch((e) => toast(errText(e), "error")); },
    async "updates-yes"() {
      const s = JSON.parse(JSON.stringify(cfg().settings));
      s.checkUpdates = true; s.autoInstallUpdates = true; s.updatesAsked = true;
      try { await api.saveSettings({ settings: s, autostart: S.d.autostart }); await refresh(); toast("Lootnest keeps itself up to date. You can change this in Settings."); api.checkUpdate().catch(() => {}); } catch (e) { toast(errText(e), "error"); }
    },
    async "updates-no"() {
      const s = JSON.parse(JSON.stringify(cfg().settings));
      s.updatesAsked = true;
      try { await api.saveSettings({ settings: s, autostart: S.d.autostart }); await refresh(); } catch (e) { toast(errText(e), "error"); }
    },
    "update-check"() { S.update = { checking: true }; render(); api.checkUpdate().catch((e) => { S.update = {}; toast(errText(e), "error"); render(); }); },
    async "update-now"() {
      try { await api.installUpdate(); } catch (e) { toast(errText(e), "error"); }
    },
    "update-notes"() { api.open("release").catch((e) => toast(errText(e), "error")); },
    "load-only": (el) => openPicker(el.dataset.game, "load"),
    "play-pick": (el) => openPicker(el.dataset.game, "play"),
    menu(el) { S.menuFor = S.menuFor === el.dataset.game ? null : el.dataset.game; render(); const m = $(".menu button"); if (m) m.focus(); },
    "edit-game": (el) => { if (S.dialog && S.dialog.kind === "sheet") S.dialog = null; editGame(el.dataset.game); },
    async shortcut(el) {
      S.menuFor = null; render();
      try { await api.shortcut(el.dataset.game); toast("Shortcut added to your desktop"); } catch (e) { toast(errText(e), "error"); }
    },
    "open-path": (el) => { S.menuFor = null; render(); api.open("path", el.dataset.path).catch((e) => toast(errText(e), "error")); },
    "open-backups": (el) => { S.menuFor = null; render(); api.open("backups", el.dataset.game).catch((e) => toast(errText(e), "error")); },
    async "remove-game"(el) {
      S.menuFor = null; render();
      const g = game(el.dataset.game);
      const ok = await confirmDialog({ title: `Turn off profiles for ${g.name}?`, body: "The game stays in your library. Everyone's profiles are moved to the “removed” folder in Lootnest's data, and the game keeps the files of whoever played last.", ok: "Turn off profiles" });
      if (!ok) return;
      try { await api.deleteGame(g.id); toast(`Profiles are off for ${g.name}`); await refresh(); loadFound(); } catch (e) { toast(errText(e), "error"); }
    },
    "np-open"() { S.picker.creating = true; S.picker.newError = ""; renderOverlay(); setTimeout(() => { const i = $("#np-name"); if (i) i.focus(); }, 20); },
    "np-cancel"() { S.picker.creating = false; S.picker.newName = ""; S.picker.newError = ""; renderOverlay(); },
    async "np-create"() {
      const P = S.picker, i = $("#np-name"), name = ((i && i.value) || "").trim(), ofSel = $("#np-of");
      if (!P) return;
      P.newName = name;
      P.newOf = (ofSel && ofSel.value) || "";
      const of = player(P.newOf);
      const fail = (t) => { P.newError = t; renderOverlay(); setTimeout(() => { const x = $("#np-name"); if (x) x.focus(); }, 20); };
      if (!name) return fail("Give it a name.");
      if (cfg().players.some((p) => p.name.trim().toLowerCase() === name.toLowerCase())) return fail(`There's already a profile called ${name}.`);
      const used = new Set(cfg().players.map((p) => p.color.toLowerCase()));
      const list = cfg().players.map((p) => ({ ...p })).concat([{ id: "", name, of: of ? of.id : "", color: of ? of.color : S.d.colors.find((c) => !used.has(c.toLowerCase())) || S.d.colors[cfg().players.length % S.d.colors.length] }]);
      try { await api.savePlayers(list); } catch (e) { return fail(errText(e)); }
      await refresh();
      const np = cfg().players.find((p) => p.name === name);
      if (!S.picker || !np) return;
      S.picker.creating = false; S.picker.newName = ""; S.picker.newError = ""; S.picker.newOf = "";
      S.picker.playerId = np.id; S.picker.owner = ""; S.picker.newMode = ""; S.picker.copyFrom = ""; S.picker.from = S.d.active[S.picker.gameId] || "";
      runPick();
    },
    pick(el) { S.picker.playerId = el.dataset.player; S.picker.owner = ""; S.picker.newMode = ""; S.picker.copyFrom = ""; S.picker.from = S.d.active[S.picker.gameId] || ""; runPick(); },
    "answer-owner"(el) { S.picker.owner = el.dataset.player; S.picker.ask = null; runPick(); },
    "answer-new"(el) { S.picker.newMode = el.dataset.mode; S.picker.copyFrom = el.dataset.from || ""; S.picker.ask = null; runPick(); },
    "answer-cloud"(el) { S.picker.cloud = el.dataset.answer; S.picker.ask = null; runPick(); },
    "picker-back"() { S.picker.ask = null; S.picker.owner = ""; S.picker.newMode = ""; render(); },
    "picker-close": closePicker,
    "dialog-ok": () => closeDialog(true),
    async "fix-blocked"() {
      const D = S.dialog;
      if (!D || D.kind !== "blocked" || D.busy) return;
      D.busy = true; renderOverlay();
      try { await api.fixBlocked(D.gameId, D.playerId); } catch (e) { D.busy = false; renderOverlay(); toast(errText(e), "error"); }
    },
    "dialog-cancel": () => closeDialog(false),
    "launch-options"(el) {
      const it = itemFor(el);
      if (!it) return;
      openLaunch(it); renderOverlay(); focusOverlay();
    },
    "lo-remove"(el) { S.dialog.o[el.dataset.list].splice(+el.dataset.i, 1); renderOverlay(); },
    "lo-apps"() {
      const D = S.dialog;
      D.picking = !D.picking; D.apps = null; renderOverlay();
      if (D.picking) api.runningApps().then((l) => { D.apps = l || []; if (S.dialog === D) { renderOverlay(); $(".lo-apps")?.scrollIntoView({ block: "nearest" }); } }, (e) => { D.picking = false; D.error = errText(e); if (S.dialog === D) renderOverlay(); });
    },
    "lo-app-pick"(el) { const D = S.dialog; D.o.closeApps.push({ name: el.dataset.name, force: false }); renderOverlay(); },
    "lo-app-add"() {
      const D = S.dialog, i = $("#lo-app"), name = i.value.trim();
      if (!name) { i.focus(); return; }
      const full = /\.exe$/i.test(name) ? name : name + ".exe";
      if (!D.o.closeApps.some((c) => c.name.toLowerCase() === full.toLowerCase())) D.o.closeApps.push({ name: full, force: false });
      renderOverlay(); const n = $("#lo-app"); if (n) n.focus();
    },
    async "lo-prog"(el) {
      const D = S.dialog, list = el.dataset.list, r = await browse("program");
      if (r.path && S.dialog === D) { D.o[list].push({ path: r.path, args: "" }); renderOverlay(); }
      else if (r.error) toast(r.error, "error");
    },
    async "lo-dir"() {
      const D = S.dialog, it = dialogItem(D), r = await browse("workDir", D.o.workDir || (it ? (it.f ? it.f.dir : it.g.launch.exe) : ""));
      if (r.path && S.dialog === D) { D.o.workDir = r.path; renderOverlay(); }
      else if (r.error) toast(r.error, "error");
    },
    "lo-dir-clear"() { S.dialog.o.workDir = ""; renderOverlay(); },
    async "lo-steam"(el) { try { await api.steamProperties(+el.dataset.app); } catch (e) { toast(errText(e), "error"); } },
    async "lo-save"() {
      const D = S.dialog, it = dialogItem(D);
      if (!it) return;
      D.busy = true; D.error = ""; renderOverlay();
      try {
        await api.saveLaunchOptions(recentKey(it), D.args, D.o);
        await refresh();
        if (it.kind === "found") await loadFound();
        if (S.dialog === D) closeDialog(true);
        toast("Saved");
      } catch (e) { D.busy = false; D.error = errText(e); if (S.dialog === D) renderOverlay(); }
    },
    "game-settings"(el) { if (S.dialog && S.dialog.kind === "sheet") S.dialog = null; openGameSettings(el.dataset.key); },
    async "game-settings-save"() {
      const D = S.dialog;
      if (!D || D.kind !== "game-settings") return;
      const name = $("#gs-name").value, f = (S.found.games || []).find((x) => x.key === D.key);
      D.busy = true; renderOverlay();
      try {
        await api.saveLibraryPref(D.key, name, (f && f.args) || "");
        await loadFound();
        if (S.dialog === D) { S.dialog = null; renderOverlay(); }
        toast("Saved");
      } catch (e) { D.busy = false; renderOverlay(); toast(errText(e), "error"); }
    },
    async "open-game-folder"(el) {
      try { await (el.dataset.game ? api.openGameFolder("game", el.dataset.game) : api.openGameFolder("found", el.dataset.key)); } catch (e) { toast(errText(e), "error"); }
    },
    async "art-set"(el) {
      const it = itemFor(el), kind = el.dataset.kind;
      if (!it) return;
      const r = await browse("art");
      if (r.error) { toast(r.error, "error"); return; }
      if (!r.path) return;
      try { await api.setCustomArt(recentKey(it), kind, r.path); forgetCustom(it); await refresh(); toast(`${cap(artWhat(kind))} changed. Lootnest keeps a copy; your picture stays where it was.`); } catch (e) { toast(errText(e), "error"); }
    },
    async "art-clear"(el) {
      const it = itemFor(el), kind = el.dataset.kind;
      if (!it) return;
      try { await api.clearCustomArt(recentKey(it), kind); forgetCustom(it); await refresh(); toast(`Your ${artWhat(kind)} is removed`); } catch (e) { toast(errText(e), "error"); }
    },
    async "found-shortcut"(el) {
      try { await api.foundShortcut(el.dataset.key); toast("Shortcut added to your desktop"); } catch (e) { toast(errText(e), "error"); }
    },
    async "store-act"(el) {
      const it = itemFor(el), t = it && storeActText(it.source, el.dataset.act, it.name);
      if (it && it.source === "itch" && it.kind === "owned") { openItch(it); return; }
      try { await api.storeAction(el.dataset.key, el.dataset.act); toast(t ? t[2] : "The launcher is opening."); } catch (e) { toast(errText(e), "error"); }
    },
    "remove-pc"(el) {
      const it = itemFor(el);
      if (!it) return;
      const D = S.dialog = { kind: "removepc", key: it.id, name: it.name, plan: null, error: "" };
      renderOverlay(); focusOverlay();
      api.removeGame(it.id, false).catch((e) => { if (S.dialog === D) { D.error = errText(e); renderOverlay(); } });
    },
    async "remove-pc-go"() {
      const D = S.dialog;
      if (!D || D.kind !== "removepc" || D.busy || !D.plan) return;
      D.busy = true; D.error = ""; renderOverlay();
      try { await api.removeGame(D.key, true); } catch (e) { D.busy = false; D.error = errText(e); renderOverlay(); }
    },
    async "steam-verify"(el) {
      try { await api.steamVerify(+el.dataset.app); toast("Steam is checking the game's files. It downloads any that are missing or damaged."); } catch (e) { toast(errText(e), "error"); }
    },
    "move-open"(el) { if (S.dialog && S.dialog.kind === "sheet") S.dialog = null; openMove(+el.dataset.app, el.dataset.name); },
    "st-sort"(el) { S.stSort = el.dataset.col; setPref("stSort", S.stSort); render(); },
    "st-more"(el) { storageMenu(el); },
    "movefolder-open"(el) { const it = itemFor(el); closeContext(); if (it) openMoveFolder(it); },
    "movefolder-pick"(el) { if (S.dialog && S.dialog.kind === "movefolder") { S.dialog.pick = el.dataset.path; renderOverlay(); } },
    async "movefolder-browse"() {
      const D = S.dialog;
      if (!D || D.kind !== "movefolder") return;
      const r = await browse("moveDir", "");
      if (r.error) toast(r.error, "error");
      if (r.path && S.dialog === D) { D.custom = D.pick = r.path; renderOverlay(); }
    },
    async "movefolder-go"() {
      const D = S.dialog;
      if (!D || D.kind !== "movefolder" || !D.pick) return;
      try {
        await api.moveFolderGame(D.key, D.pick);
        if (S.dialog !== D) return; // the move already ended
        if (!S.folderMove) S.folderMove = { key: D.key, done: 0, total: 0 };
        renderOverlay();
      } catch (e) { toast(errText(e), "error"); }
    },
    async "movefolder-cancel"() { try { await api.moveFolderCancel(); } catch (e) { toast(errText(e), "error"); } },
    "move-pick"(el) { if (S.dialog && S.dialog.kind === "move") { S.dialog.pick = el.dataset.lib; renderOverlay(); } },
    async "move-go"() {
      const D = S.dialog;
      if (!D || D.kind !== "move" || !D.pick) return;
      try {
        await api.moveGame(D.appId, D.pick);
        if (S.dialog !== D) return; // already done
        S.moving = Object.assign({ appId: D.appId, name: D.name, step: "", done: 0, total: 0 }, S.moving && S.moving.appId === D.appId ? S.moving : {});
        renderOverlay();
      } catch (e) { toast(errText(e), "error"); }
    },
    scrim(el, ev) { if (ev.target === el && S.dialog && S.dialog.kind !== "welcome") closeDialog(false); },
    "edit-cancel"() { if (!confirmLeaveEdit()) return; S.edit = null; S.view = "library"; S.animateView = true; render(); },
    async "browse-exe"() {
      const r = await browse("exe", S.edit.game.launch.exe);
      if (r.path && S.edit) { S.edit.game.launch.exe = r.tokenized || r.path; render(); }
    },
    "set-proc"(el) { S.edit.game.processName = el.dataset.name; render(); },
    async "add-path"(el) {
      const start = (S.edit.game.paths[0] || {}).path || "";
      const r = await browse(el.dataset.kind, start);
      if (r.error) toast(r.error, "error");
      if (!r.path || !S.edit) return;
      addPath(r.tokenized || r.path);
    },
    "add-sugg"(el) { addPath(el.dataset.path); },
    "remove-path"(el) { S.edit.game.paths = S.edit.game.paths.filter((p) => p._k !== el.dataset.k); render(); },
    pcgw() { api.open("pcgw", S.edit.game.name).catch((e) => toast(errText(e), "error")); },
    "save-game": saveGame,
    swatch(el) {
      S.players[+el.dataset.i].color = el.dataset.color;
      if (!S.dialog || S.dialog.kind !== "welcome") { render(); return; }
      $(".wl-me").style.setProperty("--player", el.dataset.color);
      el.parentElement.querySelectorAll(".swatch").forEach((b) => b.setAttribute("aria-pressed", b === el));
      welcomeLive();
    },
    "add-player"() { const used = new Set(S.players.map((p) => p.color)); S.players.push({ id: "", name: "", color: S.d.colors.find((c) => !used.has(c)) || S.d.colors[0] }); render(); const ins = document.querySelectorAll('[data-field="pname"]'); ins[ins.length - 1].focus(); },
    "remove-player"(el) { S.players.splice(+el.dataset.i, 1); render(); },
    "shelf-all"(el) { S.tag = el.dataset.tag; render(); const t = $(".lib-tools"); if (t) t.scrollIntoView({ block: "start" }); },
    "set-tab"(el) { S.setTab = el.dataset.tab; setPref("setTab", S.setTab); render(); const v = $("#view"); if (v) v.scrollTop = 0; },
    "reset-players"() { S.players = null; render(); },
    "edit-profiles"() { S.dialog = { kind: "profiles" }; renderOverlay(); focusOverlay(); },
    "new-profile"() { S.dialog = { kind: "profiles" }; if (!S.players) S.players = JSON.parse(JSON.stringify(cfg().players)); actions["add-player"](); },
    "save-players": () => savePlayers(false),
    "welcome-done": () => savePlayers(true),
    recap(el) {
      const p = player(el.dataset.player);
      if (!p) return;
      const ys = recapYears(p), now = new Date().getFullYear();
      S.dialog = { kind: "recap", player: p.id, year: ys.includes(now) || !ys.length ? now : ys[0] };
      renderOverlay(); focusOverlay();
    },
    async "recap-save"() {
      const D = S.dialog, p = D && player(D.player);
      if (!D || D.kind !== "recap" || D.busy || !p) return;
      D.busy = true; renderOverlay();
      try {
        let d;
        do { d = D.drawn; await d; } while (d !== D.drawn);
        const cv = $("#recap-canvas");
        if (S.dialog === D && cv) {
          const at = await api.saveRecap(p.name, D.year, cv.toDataURL("image/png"));
          toast(`Saved: ${String(at).split(/[\\/]/).pop()}, in Pictures\\Lootnest.`);
        }
      } catch (e) { toast(errText(e), "error"); }
      D.busy = false;
      if (S.dialog === D) { renderOverlay(); const b = $('[data-action="recap-save"]'); if (b && document.activeElement === document.body) b.focus(); }
    },
    "tonight-spin"() {
      if (S.spinning) return;
      const all = libraryItems(), picks = tonightPicks(all);
      if (picks.length < 2 || !S.tonight) return;
      S.spinning = true;
      const next = tonightPick(picks, undefined, S.tonight.id);
      const done = () => { S.spinning = false; S.tonight.id = next.it.id; render(); };
      const cover = $(".tn-cover"), name = $(".tn-name");
      if (reduced() || !cover || !name) { done(); return; }
      // A short roulette over the games that have a cover, slowing down.
      const withArt = picks.filter((p) => artFor(p.it, "cover").url);
      if (withArt.length < 3) { done(); return; }
      let n = 0;
      const steps = 11, start = Math.floor(Math.random() * withArt.length);
      const tick = () => {
        const p = withArt[(start + n) % withArt.length];
        cover.innerHTML = artImg(artFor(p.it, "cover"), "cover");
        name.textContent = p.it.name;
        cover.classList.remove("tn-flick"); void cover.offsetWidth; cover.classList.add("tn-flick");
        if (++n < steps) setTimeout(tick, 45 + n * n * 3); else setTimeout(done, 160);
      };
      tick();
    },
    "new-run"(el) { S.dialog = { kind: "newrun", gameId: el.dataset.game, person: el.dataset.person }; renderOverlay(); focusOverlay(); },
    "try-save"(el) { const sp = seated(); S.dialog = { kind: "trysave", gameId: el.dataset.game, from: sp && hasProfile(el.dataset.game, sp.id) ? sp.id : "" }; renderOverlay(); focusOverlay(); },
    async "try-browse"(el) {
      const D = S.dialog;
      if (!D || D.kind !== "trysave") return;
      tryKeep(D);
      const r = await browse(el.dataset.kind);
      if (S.dialog !== D) return;
      if (r.error) D.error = r.error;
      else if (r.path) { D.src = r.path; D.error = ""; }
      renderOverlay();
    },
    async "try-go"() {
      const D = S.dialog;
      if (!D || D.kind !== "trysave" || D.busy || !D.src) return;
      tryKeep(D);
      const g = game(D.gameId);
      const froms = cfg().players.filter((p) => hasProfile(D.gameId, p.id));
      const from = player(D.from) || froms[0];
      const loc = ((g && g.paths) || []).filter((p) => !String(p.id || "").startsWith("ach-")).find((l) => l.id === D.loc) || ((g && g.paths) || []).filter((p) => !String(p.id || "").startsWith("ach-"))[0];
      if (!g || !from || !loc) return;
      D.busy = true; D.error = ""; renderOverlay();
      try {
        const id = await api.trySave(g.id, from.id, loc.id, D.src, (D.name || "").trim());
        await refresh();
        S.dialog = null; renderOverlay();
        const run = player(id);
        toast(run ? `${run.name} is ready, with the save you brought. ${possessive(from.name)} own saves are as they were.` : "The run is ready.");
        render();
      } catch (e) { D.busy = false; D.error = errText(e); renderOverlay(); }
    },
    async "new-run-go"() {
      const D = S.dialog;
      if (!D || D.kind !== "newrun" || D.busy) return;
      const per = player(D.person), inp = $("#newrun-name");
      const name = ((inp && inp.value) || "").trim() || per.name + " hardcore";
      if (cfg().players.some((p) => p.name.toLowerCase() === name.toLowerCase())) { D.error = "There's a profile with that name already."; D.name = name; renderOverlay(); return; }
      D.name = (inp && inp.value) || "";
      D.busy = true; renderOverlay();
      try {
        await api.savePlayers([...cfg().players, { id: "", name, color: per.color, of: per.id }]);
        await refresh();
        if (S.dialog !== D) return;
        const run = cfg().players.find((p) => p.name === name && p.of === per.id);
        S.dialog = null; renderOverlay();
        if (run) actions["play-as"]({ dataset: { game: D.gameId, player: run.id } });
      } catch (e) { D.busy = false; D.error = errText(e); renderOverlay(); }
    },
    "open-data": () => api.open("data").catch((e) => toast(errText(e), "error")),
    "steam-setup": (el) => openSteamSetup(el.dataset.game),
    async "steam-recheck"() { S.hooks = null; renderOverlay(); await loadHooks(); if (S.dialog && S.dialog.kind === "steam") renderOverlay(); render(); },
    async "copy-hook"(el) {
      const h = S.hooks && S.hooks.games.find((x) => x.gameId === el.dataset.game);
      if (!h) return;
      try { await api.copy(h.option); S.dialog.copied = h.gameId; renderOverlay(); } catch (e) { toast(errText(e), "error"); }
    },
    "hide-banner"() { S.hideHookBanner = true; render(); toast("Lootnest will finish this the next time Steam is closed."); },
    async "hooks-apply"() {
      S.hooksBusy = true; render(); if (S.dialog) renderOverlay();
      try {
        const onlyLeaving = S.hooks && !S.hooks.games.some((h) => !h.detected && !h.direct);
        const r = await api.applyHooks(true);
        await loadHooks();
        if (r && r.changed && onlyLeaving) toast(r.restarted ? "Steam restarted. It now starts those games by itself." : "Done. Steam now starts those games by itself.");
        else if (r && r.changed) toast(r.restarted ? "Steam restarted. Its Play button now asks who's playing." : "Done. Steam's Play button now asks who's playing.");
      } catch (e) { toast(errText(e), "error"); }
      S.hooksBusy = false; render(); if (S.dialog) renderOverlay();
    },
    sources() { S.sources = null; S.dialog = { kind: "sources" }; renderOverlay(); focusOverlay(); loadSources(); },
    "sources-rescan"() { actions.rescan(); renderOverlay(); },
    rescan() { S.rescanning = new Set((S.found.games || []).map((f) => f.key)); S.found = { scanning: !(S.found.games || []).length, refreshing: true, games: S.found.games || [], groups: S.found.groups || [] }; render(); api.scan(); },
    async "add-all"() {
      const keys = (S.found.games || []).filter((f) => f.ready).map((f) => f.key);
      await addFound(keys);
    },
    async "found-add"(el) {
      const f = (S.found.games || []).find((x) => x.key === el.dataset.key);
      if (!f) return;
      if (S.dialog && S.dialog.kind === "sheet") closeDialog(false);
      if (f.ready) { await addFound([f.key]); return; }
      openEditor({
        isNew: true,
        game: { id: "", name: f.name, launch: f.source === "steam" ? { type: "steam", appId: f.appId, args: f.args || "" } : { type: "exe", exe: f.exe, args: f.args || "" },
          processName: f.processName || "", playerArgs: {}, paths: (f.paths || []).map((p) => ({ id: "", path: p })) },
        inspect: { suggestions: f.suggestions || [], candidates: [] },
      });
    },
    async "found-hide"(el) {
      if (S.dialog && S.dialog.kind === "sheet") closeDialog(false);
      const key = el.dataset.key, f = (S.found.games || []).find((x) => x.key === key);
      // A game from more than one store is hidden with every copy.
      const grp = f && groupOf({ kind: "found", id: key }), keys = grp ? grp.copies.map((c) => c.id) : [key];
      // A game added by hand leaves the Library for good; a scanned one can come back.
      await hideFound(keys, f && !f.added ? `${f.name} is hidden from the Library.` : "");
    },
    "start-with"(el) {
      const it = everyItem().find((x) => x.kind === "found" && x.id === el.dataset.key), grp = it && groupOf(it);
      if (!grp) return;
      const m = menuUnder(el, `<div class="ctx-title">Start with</div>${grp.copies.map((c) => `<button data-action="start-copy" data-group="${esc(grp.key)}" data-key="${esc(c.id)}" role="menuitemradio" aria-checked="${c.id === it.id}" title="${esc(c.f.dir || "")}"><span class="grow">${esc(sourceLabel(c.source))}</span>${c.id === it.id ? GI.tick : ""}</button>`).join("")}`);
      const first = m.querySelector('[aria-checked="true"]') || m.querySelector("button");
      if (first) first.focus({ preventScroll: true });
    },
    async "start-copy"(el) {
      const { group, key } = el.dataset;
      try { await api.setPrimaryCopy(group, key); } catch (e) { toast(errText(e), "error"); return; }
      await loadFound();
      if (S.view === "game" && S.page && S.page.id !== key) openSheet("found", key, null);
    },
    async "copies-apart"(el) {
      const group = el.dataset.group, apart = el.dataset.apart !== "0";
      try { await api.separateCopies(group, apart); } catch (e) { toast(errText(e), "error"); return; }
      await loadFound();
      await refresh();
      if (apart) toast("Each copy has its own entry in the Library now.", "success", { label: "Undo", run: () => actions["copies-apart"]({ dataset: { group, apart: "0" } }) });
    },
    async "found-unhide"(el) {
      try { await api.unhideFound(el.dataset.key); await loadFound(); await refresh(); } catch (e) { toast(errText(e), "error"); }
    },
    async "show-log"() { try { S.logText = (await api.log()) || "Nothing yet."; } catch (e) { S.logText = errText(e); } render(); },
    quit: () => api.quit(),
  };

  async function addFound(keys) {
    S.adding = true; render();
    try {
      const r = await api.addFound(keys);
      if (r.added.length) toast(r.added.length === 1 ? `Profiles are on for ${r.added[0]}` : `Profiles are on for ${r.added.length} games`);
      if (r.failed.length) toast(r.failed.join("\n"), "error");
      await refresh();
      if (S.view === "game" && S.page && S.page.kind === "found" && r.added.length === 1) {
        const g = cfg().games.find((x) => x.name === r.added[0]);
        if (g) { S.page = { kind: "game", id: g.id, cloud: null }; hist.at = here(); }
      }
      await loadFound();
      await loadHooks();
    } catch (e) { toast(errText(e), "error"); }
    S.adding = false; render();
  }

  function addPath(path) {
    const E = S.edit;
    if (E.game.paths.some((p) => p.path.toLowerCase() === path.toLowerCase())) return;
    const p = { id: "", path, _k: "k" + ++seq };
    E.game.paths.push(p);
    render();
    checkPath(p);
  }

  function confirmLeaveEdit() {
    return true; // edits are explicit; leaving discards them
  }

  document.addEventListener("click", (ev) => {
    const el = ev.target.closest("[data-action]");
    if (!el) {
      if (S.menuFor && !ev.target.closest(".menu")) { S.menuFor = null; render(); }
      return;
    }
    if (el.dataset.action !== "menu" && S.menuFor && !el.closest(".menu")) S.menuFor = null;
    const had = !!(S.dialog || S.picker), menu = !$("#ctx").hidden, inMenu = !!el.closest("#ctx");
    if (inMenu) closeContext();
    if (el.closest(".ln-drawer[data-menu]") && el.dataset.action !== "dialog-cancel" && el.dataset.action !== "scrim") { S.dialog = null; renderOverlay(); }
    if (PLAYS.has(el.dataset.action)) uiSound("go");
    const f = actions[el.dataset.action];
    if (f) f(el, ev);
    // One sound for what the click did: Play, a page or tab (in the action),
    // then a dialog, drawer or menu opening or closing.
    const has = !!(S.dialog || S.picker);
    if ((has && !had) || (!menu && !$("#ctx").hidden)) uiSound("open");
    else if ((had && !has) || (menu && !inMenu && $("#ctx").hidden)) uiSound("close");
  });
  const PLAYS = new Set(["play", "play-as", "play-found"]);
  // In the Library, Ctrl+click and Shift+click pick games instead of opening them.
  const libTile = (ev) => !BP.on && ev.target.closest && ev.target.closest("#grid .ln-tile, #grid .lib-row-open");
  document.addEventListener("mousedown", (ev) => { if (ev.shiftKey && libTile(ev)) ev.preventDefault(); }, true); // no text selected
  document.addEventListener("click", (ev) => {
    const t = libTile(ev);
    if (!t) return;
    const k = t.closest("[data-k]").dataset.k;
    if (!ev.ctrlKey && !ev.metaKey && !ev.shiftKey) { S.libAnchor = k; return; }
    ev.preventDefault(); ev.stopPropagation();
    if (ev.shiftKey && S.libSel.size && S.libAnchor) libPickRange(S.libAnchor, k);
    else { libPick([k], !S.libSel.has(k)); S.libAnchor = k; }
    t.focus({ preventScroll: true });
  }, true);

  let findTimer = 0;
  document.addEventListener("input", (ev) => {
    const el = ev.target;
    if (el.id === "side-q") { S.sideQ = el.value; renderSideGames(); return; }
    if (el.dataset.lo && S.dialog && S.dialog.kind === "launch") {
      const D = S.dialog, O = D.o, l = el.dataset.lo;
      if (l === "gameArgs") D.args = el.value;
      else if (l === "args") O[el.dataset.list][+el.dataset.i].args = el.value;
      else if (l === "force") O.closeApps[+el.dataset.i].force = el.checked;
      else if (l === "admin") O.runAsAdmin = el.checked;
      else if (l === "reopen") O.reopenClosed = el.checked;
      return;
    }
    if (S.dialog && S.dialog.kind === "details-edit" && el.id.startsWith("de-")) { (S.dialog.vals ||= {})[el.id] = el.value; return; }
    if (S.dialog && S.dialog.kind === "profile") {
      const f = el.dataset.pe, D = S.dialog;
      if (f) D.p[f] = f === "bannerY" ? +el.value : el.value;
      if (f === "name") { const n = $("#pe-prev-name"); if (n) n.textContent = el.value; }
      else if (f === "tagline") { const n = $("#pe-prev-tag"); if (n) n.textContent = el.value; }
      else if (f === "color") { const d = $(".dialog.pe"); if (d) d.style.setProperty("--c", el.value); document.querySelectorAll('.dialog.pe [style*="--c"]').forEach((a) => a.style.setProperty("--c", el.value)); }
      else if (f === "bannerY") { const a = $(".pe-pbanner .art"); if (a) a.style.backgroundPosition = `center ${el.value}%`; }
      else if (f === "banner" || f === "avatarGame") renderOverlay();
      if (el.id === "pe-q") { D.q = el.value; renderOverlay(); }
      return;
    }
    if (el.id === "gq") {
      S.gq.q = el.value; S.gq.hits = null; S.gq.sel = -1;
      clearTimeout(gqTimer);
      if (el.value.trim() && cfg().settings.findGames) gqTimer = setTimeout(() => { api.findGames(S.gq.q).catch(() => { S.gq.hits = []; renderGq(); }); }, 300);
      renderGq();
      return;
    }
    if (el.id === "find-q") {
      S.find.q = el.value; S.find.hits = null; S.find.error = "";
      clearTimeout(findTimer);
      const r = $("#find-results");
      if (!el.value.trim()) { if (r) r.innerHTML = findResults(); return; }
      if (r) r.innerHTML = findResults();
      findTimer = setTimeout(() => { api.findGames(S.find.q).catch((e) => { S.find.error = errText(e); const r2 = $("#find-results"); if (r2) r2.innerHTML = findResults(); }); }, 350);
      return;
    }
    if (el.id === "lib-q") {
      S.q = el.value;
      const grid = $("#grid");
      if (grid) { grid.innerHTML = gridHTML(); loadCovers(); paintLibSel(); }
      document.querySelectorAll(".lib-recent, .lib-shelves, .lib-all-head").forEach((x) => { x.hidden = !!S.q; }); // a search shows only what it finds
      return;
    }
    if (el.id === "steam-q" && S.dialog) {
      S.dialog.q = el.value;
      const pos = el.selectionStart;
      renderOverlay();
      const q = $("#steam-q"); q.focus(); q.setSelectionRange(pos, pos);
      return;
    }
    const f = el.dataset.field;
    if (!f) return;
    if (f === "pof") {
      // A run of a person takes their colour; back to "a person" keeps it.
      const p = S.players[+el.dataset.i], par = S.players.find((x) => x.id === el.value);
      p.of = par ? par.id : "";
      if (par) p.color = par.color;
      render();
      return;
    }
    if (f === "pname") { S.players[+el.dataset.i].name = el.value; if (S.dialog && S.dialog.kind === "welcome") welcomeLive(); if (S.dialog && S.dialog.kind === "profiles") { const b = $('[data-action="save-players"]'); if (b) b.disabled = false; } return; }
    const E = S.edit;
    if (!E) return;
    const g = E.game;
    if (f === "name") g.name = el.value;
    else if (f === "exe") g.launch.exe = el.value;
    else if (f === "processName") g.processName = el.value;
    else if (f === "steamDirect") g.steamDirect = el.checked;
    else if (f === "args") g.launch.args = el.value;
    else if (f === "parg") { g.playerArgs = g.playerArgs || {}; g.playerArgs[el.dataset.player] = el.value; }
    else if (f === "path") { const p = g.paths.find((x) => x._k === el.dataset.k); if (p) p.path = el.value; }
  });

  document.addEventListener("toggle", (ev) => { if (ev.target.classList && ev.target.classList.contains("ach-ms")) S.achMs = ev.target.open; }, true);
  document.addEventListener("focusout", (ev) => { if (ev.target.id === "gp-note") saveNote(ev.target); });
  document.addEventListener("change", (ev) => {
    const el = ev.target;
    if (el.dataset.setting) { saveSetting(el); return; }
    if (el.dataset.backup === "auto") { saveBackup({ auto: el.checked }); return; }
    if (el.dataset.backup === "sync") { S.saves = null; saveBackup({ sync: el.checked }).then(loadSaves); return; }
    if (el.dataset.backup === "keep") { saveBackup({ keep: parseInt(el.value, 10) || 0 }); return; }
    if (el.id === "lib-sort") { S.sort = el.value; setPref("sort", S.sort); render(); return; }
    if (el.id === "lib-store") { S.store = el.value; render(); return; }
    if (el.id === "lib-tag") { S.tag = el.value; render(); return; }
    if (el.id === "lib-status") { S.status = el.value; render(); return; }
    if (el.id === "lib-who") { S.who = el.value; render(); return; }
    if (el.id === "ach-pick") { S.achPage = el.value; render(); return; }
    if (el.dataset.assign) {
      if (!el.value) return;
      const it = everyItem().find((x) => recentKey(x) === el.value);
      if (S.dialog && S.dialog.kind === "unsorted") S.dialog.focusAt = [...document.querySelectorAll("[data-assign]")].indexOf(el);
      api.assignCapture(el.dataset.assign, el.value).then(() => toast(it ? `Now on ${it.name}'s Timeline` : "Marked as not a game"), (e) => toast(errText(e), "error"));
      return;
    }
    if (S.dialog && S.dialog.kind === "playnite" && (el.id === "pn-player" || el.dataset.pn)) {
      const D = S.dialog;
      if (el.id === "pn-player") { D.player = el.value; playnitePreview(D); } else { D.opts[el.dataset.pn] = el.checked; renderOverlay(); }
      const n = el.id === "pn-player" ? $("#pn-player") : $(`[data-pn="${el.dataset.pn}"]`);
      if (n) n.focus();
      return;
    }
    if (el.id === "itch-up" && S.dialog && S.dialog.kind === "itch") { S.dialog.pick = +el.value; renderOverlay(); const n = $("#itch-up"); if (n) n.focus(); return; }
    if (el.id === "recap-year" && S.dialog && S.dialog.kind === "recap") { S.dialog.year = +el.value; renderOverlay(); return; }
    if ((el.id === "try-from" || el.id === "try-loc") && S.dialog && S.dialog.kind === "trysave") { tryKeep(S.dialog); renderOverlay(); const n = $("#" + el.id); if (n) n.focus(); return; }
    if (el.dataset.field === "path" && S.edit) { const p = S.edit.game.paths.find((x) => x._k === el.dataset.k); if (p) checkPath(p); }
  });

  // The search in the top bar: Ctrl+K anywhere, arrows and Enter in its list.
  document.addEventListener("keydown", (ev) => {
    // Ctrl+B folds the sidebar to a rail of icons, and opens it again.
    if (ev.ctrlKey && !ev.altKey && !ev.shiftKey && ev.key.toLowerCase() === "b") { ev.preventDefault(); setFold(!S.folded); return; }
    // In the sidebar's games, the arrows move from game to game.
    if ((ev.key === "ArrowDown" || ev.key === "ArrowUp") && (ev.target.closest && ((ev.target.closest("#side-games") && !ev.target.matches(".sg-input")) || ev.target.id === "side-q"))) {
      const rows = [...document.querySelectorAll("#side-games .sg-group:not(.closed) .sg-row, #side-games .sg-head, #side-games .sg-new")];
      const i = rows.indexOf(ev.target);
      const n = ev.target.id === "side-q" ? (ev.key === "ArrowDown" ? rows[0] : null) : rows[i + (ev.key === "ArrowDown" ? 1 : -1)];
      if (n) { ev.preventDefault(); n.focus(); n.scrollIntoView({ block: "nearest" }); }
      else if (ev.key === "ArrowUp" && i === 0) { ev.preventDefault(); $("#side-q").focus(); }
      return;
    }
    if (ev.key === "Escape" && ev.target.id === "side-q" && ev.target.value) { ev.preventDefault(); ev.target.value = ""; S.sideQ = ""; renderSideGames(); return; }
    if (ev.key === "Enter" && ev.target.id === "newrun-name") { ev.preventDefault(); actions["new-run-go"](); return; }
    // Enter in a save point's name: make it, or save the new name.
    if (ev.key === "Enter" && (ev.target.id === "pt-new" || ev.target.id === "pt-rename")) {
      ev.preventDefault();
      const b = ev.target.id === "pt-new" ? ev.target.closest(".pt-make").querySelector('[data-action="pt-make"]') : ev.target.closest(".pt-row").querySelector('[data-action="pt-rename-save"]');
      if (b) b.click();
      return;
    }
    if (ev.ctrlKey && !ev.altKey && ev.key.toLowerCase() === "k") {
      ev.preventDefault();
      const q = $("#gq"); if (q) { q.focus(); q.select(); renderGq(); }
      return;
    }
    if (ev.target.id !== "gq") return;
    const n = gqItems().length;
    if (ev.key === "ArrowDown" || ev.key === "ArrowUp") {
      ev.preventDefault();
      if (n) { S.gq.sel = ev.key === "ArrowDown" ? (S.gq.sel + 1) % n : (S.gq.sel <= 0 ? n - 1 : S.gq.sel - 1); renderGq(); }
    } else if (ev.key === "Enter") {
      ev.preventDefault();
      if (S.gq.sel >= 0 && S.gq.sel < n) gqOpen(S.gq.sel); else if (cfg().settings.findGames) gqAll(); else if (n) gqOpen(0);
    } else if (ev.key === "Escape") {
      ev.preventDefault(); ev.stopPropagation();
      gqClose(); ev.target.blur();
    }
  }, true);
  document.addEventListener("focusin", (ev) => { if (ev.target.id === "gq") renderGq(); });
  document.addEventListener("pointerdown", (ev) => { if (!ev.target.closest(".bar-search")) gqClose(); });
  document.addEventListener("keydown", (ev) => {
    if (ev.target.id === "lo-app" && ev.key === "Enter") { ev.preventDefault(); actions["lo-app-add"](); return; }
    if (ev.target.id === "view-name" && ev.key === "Enter") { ev.preventDefault(); saveViewName(ev.target); return; }
    if (ev.target.id === "tag-new" && ev.key === "Enter") { ev.preventDefault(); actions["tag-add"]({ dataset: { key: ev.target.dataset.key } }); return; }
    if (ev.target.id === "gp-note" && ev.key === "Enter") { ev.preventDefault(); saveNote(ev.target); return; }
    if (ev.target.id === "gp-hours" && (ev.key === "Enter" || ev.key === "Escape")) { ev.preventDefault(); actions[ev.key === "Enter" ? "hours-save" : "hours-cancel"](); return; }
    // The stars are one stop for Tab; the arrows move the rating.
    const star = ev.target.closest && ev.target.closest(".gp-rate [data-stars]");
    if (star && ["ArrowLeft", "ArrowRight", "ArrowUp", "ArrowDown"].includes(ev.key)) {
      ev.preventDefault();
      const n = Math.min(5, Math.max(1, +star.dataset.stars + (ev.key === "ArrowRight" || ev.key === "ArrowUp" ? 1 : -1)));
      if (n !== +star.dataset.stars) actions.rate({ dataset: { ...star.dataset, stars: String(n) } });
      return;
    }
    if (!$("#ctx").hidden) {
      if (ev.key === "Escape") { ev.preventDefault(); closeContext(true); uiSound("close"); return; }
      const inMenu = !!(ev.target.closest && ev.target.closest("#ctx"));
      if (inMenu && (ev.key === "ArrowDown" || ev.key === "ArrowUp" || ev.key === "Tab")) {
        ev.preventDefault();
        const b = [...document.querySelectorAll("#ctx button:not(:disabled)")];
        const i = b.indexOf(document.activeElement), down = ev.key === "Tab" ? !ev.shiftKey : ev.key === "ArrowDown";
        if (b.length) b[(i + (down ? 1 : b.length - 1)) % b.length].focus();
        return;
      }
      const seat = inMenu && /^[1-9]$/.test(ev.key) && $('#ctx [data-action="seat-set"]') && grouped()[+ev.key - 1];
      if (seat) { ev.preventDefault(); closeContext(true); setSeat(seat.id); return; }
    }
    // Picking games in the Library: Space picks the focused one, Ctrl+A
    // every game shown, Esc none.
    const inText = ev.target.closest && ev.target.closest("input, textarea, select");
    const tile = !ev.ctrlKey && !ev.altKey && ev.target.closest && ev.target.closest("#grid .ln-tile, #grid .lib-row-open");
    if (S.view === "library" && !S.dialog && !S.picker && !BP.on && !inText) {
      const k = tile && tile.closest("[data-k]").dataset.k;
      if (ev.key === " " && tile) { ev.preventDefault(); libPick([k], !S.libSel.has(k)); S.libAnchor = k; return; }
      if (ev.ctrlKey && !ev.altKey && !ev.shiftKey && ev.key.toLowerCase() === "a" && $("#grid")) { ev.preventDefault(); libPick(gridKeys(), true); return; }
      if (ev.key === "Escape" && S.libSel.size) { ev.preventDefault(); libSelClear(); return; }
    }
    // In the Library's grid, the arrows move from cover to cover; with
    // Shift they pick the games on the way.
    if (tile && ["ArrowLeft", "ArrowRight", "ArrowUp", "ArrowDown", "Home", "End"].includes(ev.key)) {
      ev.preventDefault();
      const grid = $("#grid"), tiles = [...grid.querySelectorAll(".ln-tile, .lib-row-open")], i = tiles.indexOf(tile);
      const cols = S.density === "list" ? 1 : getComputedStyle(grid).gridTemplateColumns.split(" ").length;
      const n = tiles[{ ArrowLeft: i - 1, ArrowRight: i + 1, ArrowUp: i - cols, ArrowDown: i + cols, Home: 0, End: tiles.length - 1 }[ev.key]];
      if (n && n !== tile) {
        n.focus(); n.scrollIntoView({ block: "nearest" }); uiSound("move");
        if (ev.shiftKey) {
          if (!S.libSel.size) S.libAnchor = tile.closest("[data-k]").dataset.k;
          libPickRange(S.libAnchor, n.closest("[data-k]").dataset.k);
        }
      }
      return;
    }
    // Ctrl+F or / finds a game.
    if (!S.picker && !S.dialog && ((ev.ctrlKey && ev.key.toLowerCase() === "f") || (ev.key === "/" && !ev.target.closest("input, textarea, select")))) {
      ev.preventDefault();
      if (S.view !== "library") { S.view = "library"; S.page = null; render(); }
      const q = $("#lib-q"); if (q) { q.focus(); q.select(); }
      return;
    }
    if (S.picker && !S.picker.working) {
      // Naming a new profile: Enter makes it, Esc goes back to the slots.
      if (S.picker.creating && ev.target.id === "np-name") {
        if (ev.key === "Enter") { ev.preventDefault(); actions["np-create"](); }
        else if (ev.key === "Escape") { ev.preventDefault(); actions["np-cancel"](); }
        return;
      }
      if (ev.key === "Escape") { ev.preventDefault(); if (S.picker.ask) actions["picker-back"](); else { closePicker(); uiSound("close"); } return; }
      if (!S.picker.ask && /^[1-9]$/.test(ev.key)) {
        const p = grouped()[+ev.key - 1];
        if (p) { ev.preventDefault(); S.picker.playerId = p.id; S.picker.from = S.d.active[S.picker.gameId] || ""; runPick(); }
      }
      return;
    }
    // The game page's tabs: the arrows, Home and End in their row, and
    // Ctrl+Tab and Ctrl+Shift+Tab anywhere on the page.
    if (ev.target.closest && ev.target.closest(".gp-tabs, .gp-head-tabs") && !ev.ctrlKey && !ev.altKey && ["ArrowLeft", "ArrowRight", "Home", "End"].includes(ev.key)) { ev.preventDefault(); stepTab(ev.key); return; }
    if (ev.ctrlKey && !ev.altKey && ev.key === "Tab" && S.view === "game" && !S.dialog && !S.picker && !BP.on) { ev.preventDefault(); stepTab(ev.shiftKey ? "ArrowLeft" : "ArrowRight"); return; }
    if (S.dialog && S.dialog.kind === "shots" && !ev.altKey && (ev.key === "ArrowLeft" || ev.key === "ArrowRight")) { ev.preventDefault(); actions["shot-step"]({ dataset: { d: ev.key === "ArrowRight" ? 1 : -1 } }); return; }
    if (ev.key === "Escape") {
      if (S.dialog && S.dialog.kind !== "welcome") { closeDialog(false); uiSound("close"); return; }
      if (S.menuFor) { S.menuFor = null; render(); return; }
      if (S.view === "game" && !ev.target.closest("input, textarea, select")) pageBack();
    }
    // Back and Forward, as in a browser: Alt+Left and Alt+Right, and
    // Backspace outside a text field.
    if (ev.altKey && !ev.ctrlKey && !ev.shiftKey && (ev.key === "ArrowLeft" || ev.key === "ArrowRight")) { ev.preventDefault(); goHist(ev.key === "ArrowLeft"); return; }
    if (ev.key === "Delete" && sideRemoveKey(ev)) return;
    if (ev.key === "Backspace" && !ev.altKey && !ev.ctrlKey && !ev.target.closest("input, textarea, select, [contenteditable]")) { ev.preventDefault(); goHist(true); return; }
    if (ev.key === "Enter" && S.dialog && S.dialog.kind === "welcome" && ev.target.dataset.field === "pname") savePlayers(true);
    if (ev.key === "Enter" && ev.target.id === "rs-pw") actions["restore-pw"]();
    if (ev.key === "Enter" && (ev.target.id === "bk-pw2" || ev.target.id === "bk-pw" && S.backup && S.backup.locked)) actions["backup-pw-save"]();
  });

  // block browser shortcuts that make no sense in an app window
  document.addEventListener("keydown", (ev) => {
    if ((ev.ctrlKey && ["p", "f", "g", "r", "u", "s", "+", "-", "=", "0"].includes(ev.key.toLowerCase())) || ev.key === "F5" || ev.key === "F3") ev.preventDefault();
    // F5 looks for games again, as Scan again does.
    if (ev.key === "F5" && !ev.ctrlKey && !ev.altKey && !ev.shiftKey && !S.picker && !S.dialog && !S.found.refreshing) actions.rescan();
  }, true);
  // No page zoom (Ctrl+wheel, pinch) and no browser menu outside text fields.
  document.addEventListener("wheel", (ev) => { if (ev.ctrlKey) ev.preventDefault(); }, { passive: false });
  document.addEventListener("contextmenu", (ev) => {
    if (!ev.target.closest("input, textarea, .selectable")) ev.preventDefault();
  });
  // The mouse's back and forward buttons.
  document.addEventListener("mouseup", (ev) => { if (ev.button === 3 || ev.button === 4) { ev.preventDefault(); goHist(ev.button === 3); } });
  document.addEventListener("dragover", (ev) => ev.preventDefault());
  document.addEventListener("drop", (ev) => ev.preventDefault());

