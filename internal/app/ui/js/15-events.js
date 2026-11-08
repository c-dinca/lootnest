  // ---------------------------------------------------------------- events --
  window.__ps = function (ev) {
    switch (ev.type) {
      case "sessions":
        if (!S.d) return;
        S.d.sessions = ev.sessions || [];
        S.d.active = ev.active || {};
        S.pinfo = {}; // a swap or a session's end writes profiles: read again
        achAsked.clear();
        tlAsked.clear();
        S.allAch = null;
        onSessions();
        refresh();
        break;
      case "launch":
        openPicker(ev.gameId, ev.reqId ? "steam" : "play", ev.reqId);
        break;
      case "found":
        loadFound().then(() => { if (S.dialog && S.dialog.kind === "sources") loadSources(); });
        break;
      case "hooks":
        loadHooks().then(() => { render(); if (S.dialog && S.dialog.kind === "steam") renderOverlay(); });
        break;
      case "blocked":
        if (S.picker) { S.picker = null; render(); }
        S.menuFor = null;
        S.dialog = { kind: "blocked", gameId: ev.gameId, playerId: ev.playerId || "" };
        renderOverlay();
        setTimeout(() => { const b = $('[data-action="fix-blocked"]'); if (b) b.focus(); }, 20);
        break;
      case "blockedDone": {
        const D = S.dialog && S.dialog.kind === "blocked" ? S.dialog : null;
        if (ev.error) {
          if (D) { D.busy = false; renderOverlay(); }
          toast(ev.error, "error");
          break;
        }
        if (D) { S.dialog = null; renderOverlay(); }
        refresh();
        loadHooks().then(render);
        const g = game(ev.gameId);
        if (ev.need) openPicker(ev.gameId, "play");
        else toast(`Starting ${g ? g.name : "the game"} for ${pname(ev.playerId)}. Next time, start it from the ${g ? g.name : "game"} (Lootnest) shortcut on your desktop to choose who's playing.`);
        break;
      }
      case "update":
        if (ev.available && !ev.manual && S.update.version !== ev.version) toast(`Lootnest ${ev.version} is out. Update from the sidebar.`);
        S.update = Object.assign({}, ev, { checking: false });
        if (ev.manual && ev.upToDate) toast(`Lootnest ${S.d.version} is the latest version.`);
        if (ev.error) toast(ev.error, "error");
        render();
        break;
      case "updateStep":
        if (ev.error) { S.update.step = ""; toast(ev.error, "error"); } else S.update.step = ev.step;
        render();
        break;
      case "cover":
        pendingCovers.delete(ev.key);
        S.covers[ev.key] = ev.url;
        // Its wide art stood in, so there's no image to fill: draw again.
        if (ev.url && S.art["hero:" + ev.key] && !document.querySelector(`img[data-cover="${CSS.escape(ev.key)}"]`)) render();
        else showCover(ev.key);
        break;
      case "art":
        S.art[ev.key] = ev.url;
        if (S.view === "game" || S.view === "library" || S.view === "home") render();
        break;
      case "achievements":
        S.ach[ev.key] = ev.set || null;
        if (S.view === "game") render();
        break;
      case "bigPicture":
        if (ev.on) bpOpen(); else bpClose();
        break;
      case "bpReturn": // the game started from Big Picture closed: back where it was
        if (BP.on) { renderBP(false); sfx("open"); }
        break;
      case "saveCheck":
        S.saveCheck = ev.check || null;
        if (S.view === "home" || S.view === "game") render();
        break;
      case "timeline":
        S.timeline[ev.key] = ev.items || [];
        if (S.view === "game") render();
        break;
      case "savePoints":
        S.points[ev.key] = Object.assign({}, S.points[ev.key] || {}, { points: ev.points || [], error: ev.error || "", player: ev.player || "", running: !!ev.running, profiles: !!ev.profiles, loaded: true });
        if (S.view === "game") render();
        if (S.dialog && S.dialog.kind === "points") renderOverlay();
        break;
      case "savePointMade": {
        const P = S.points[ev.key] || (S.points[ev.key] = {});
        P.busy = false;
        tlAsked.delete(ev.key);
        if (ev.error) toast(ev.error, "error");
        else if (ev.restored) toast(`Back to ${ev.point ? ev.point.name : "the save point"}. The saves from before are kept as a point.`);
        else if (ev.point) { if (!ev.hotkey) toast(`Save point made: ${ev.point.name}`); if (!ev.native) cue("point", ev.key); }
        if (S.view === "game") render();
        if (S.dialog && S.dialog.kind === "points") renderOverlay();
        break;
      }
      case "offline":
        if (ev.error) toast(ev.error, "error");
        else toast(ev.on ? "Kept offline. Windows Firewall blocks it from connecting." : "It can go online again.");
        refresh();
        break;
      case "achDoctor":
        S.doctor[ev.key] = ev;
        if (S.dialog && S.dialog.kind === "achdoc") renderOverlay();
        break;
      case "achUnlocked":
        // A game running now unlocked something: its page and the
        // Achievements page read it again, and the window says so.
        if (ev.list && ev.list.length) { if (!document.hidden) achPop(ev.list, ev.game, ev.player); if (!ev.native) unlockSound(ev.list); }
        achAsked.delete(ev.key); S.allAch = null;
        if (S.view === "game" || S.view === "achievements") render();
        break;
      case "allAchievements":
        S.allAch = { games: ev.games || [] };
        if (S.view === "achievements") { S.animateView = "kids"; render(); }
        break;
      case "profileInfo":
        S.pinfo[ev.gameId] = ev.info || {};
        if (S.picker && S.picker.gameId === ev.gameId) renderOverlay();
        if (S.view === "game" || S.view === "home") render();
        break;
      case "moveInfo": {
        const D = S.dialog && S.dialog.kind === "move" && S.dialog.appId === ev.info.appId ? S.dialog : null;
        if (!D) break;
        D.info = ev.info;
        const room = (l) => l.free < 0 || l.free >= ev.info.size + MOVE_MARGIN;
        const first = (ev.info.libraries || []).find((l) => !l.current && room(l));
        D.pick = first ? first.path : "";
        renderOverlay();
        break;
      }
      case "moveProgress":
        S.moving = Object.assign(S.moving || {}, ev);
        if (S.dialog && S.dialog.kind === "move") renderOverlay();
        break;
      case "moveDone": {
        S.moving = null;
        if (S.dialog && S.dialog.kind === "move" && S.dialog.appId === ev.appId) { S.dialog = null; renderOverlay(); }
        if (ev.error) toast(`Couldn't move ${ev.name}: ${ev.error}${/[.!?]$/.test(ev.error) ? "" : "."}`, "error");
        else if (ev.leftover) toast(`${ev.name} moved. Some old files were in use and are still in ${ev.leftover}; you can delete that folder.`);
        else toast(`${ev.name} is now in ${ev.lib}.`);
        refresh();
        break;
      }
      case "details":
        S.details[ev.key] = ev;
        if (S.view === "game") render();
        break;
      case "news":
        S.news[ev.key] = ev;
        if (S.view === "game" || S.view === "library" || S.view === "home") render();
        break;
      case "avatar":
      case "banner": {
        const D = S.dialog && S.dialog.kind === "profile" ? S.dialog : null;
        if (D) { D.busy = false; if (!ev.error && ev.type === "banner") D.p.banner = "image"; if (!ev.error && ev.type === "avatar") D.p.avatar = true; }
        if (ev.error) toast(ev.error, "error");
        refresh();
        break;
      }
      case "backup": {
        const D = S.dialog;
        if (ev.password && D && D.kind === "backup-pw") {
          if (ev.error) { D.busy = false; D.error = ev.error; renderOverlay(); }
          else { const on = D.on; S.dialog = null; renderOverlay(); toast(on ? "Saves and backups are encrypted" : "Saves and backups aren't encrypted any more"); S.saves = null; loadBackupInfo(); loadSaves(); api.syncNow().catch(() => {}); }
        }
        if (ev.done && S.backupAsked) { S.backupAsked = false; if (ev.error) toast(ev.error, "error"); else toast("Backed up"); }
        loadBackupInfo();
        break;
      }
      case "findGames": {
        if (ev.q === S.gq.q) { S.gq.hits = ev.hits || []; renderGq(); }
        if (ev.q !== S.find.q) break; // an older search
        S.find.hits = ev.hits || null; S.find.error = ev.error || "";
        const r = $("#find-results"); if (r) r.innerHTML = findResults();
        break;
      }
      case "updates": {
        S.updates = { games: ev.games || [], newsOn: !!ev.newsOn, at: Date.now() }; S.updChecking = false;
        updateNavCount();
        if (S.view === "updates") render();
        break;
      }
      case "newCopies": {
        if (ev.offers) { S.copies = ev.offers; updateNavCount(); if (S.view === "updates" || S.view === "game") render(); }
        else api.newCopies().catch(() => {});
        break;
      }
      case "downloadsFound": {
        const same = JSON.stringify(S.dl) === JSON.stringify(ev.matches || []);
        S.dl = ev.matches || []; updateNavCount();
        if (!same && (S.view === "updates" || S.view === "game")) render();
        break;
      }
      case "carried": {
        const C = S.carried[ev.old] || {};
        C.busy = false;
        if (ev.error) { C.error = ev.error; toast(ev.error, "error"); }
        else if (ev.deleted) { C.deleted = true; toast("The old folder was deleted. Its saves are in Lootnest's backups."); }
        else Object.assign(C, { done: true, saves: ev.saves || [], profile: ev.profile || "", oldThere: !!ev.oldThere, error: "" });
        S.carried[ev.old] = C;
        S.updates = null; api.newCopies().catch(() => {}); api.updates().catch(() => {});
        if (S.view === "updates" || S.view === "game") render();
        break;
      }
      case "updatePlan": {
        const D = S.dialog;
        if (D && D.kind === "update" && D.key === ev.key && D.path === ev.path) {
          if (ev.opening) { D.opening = true; renderOverlay(); break; }
          D.busy = false; D.opening = false; D.plan = ev.plan || null; D.check = ev.check || null; D.error = ev.error || ""; D.dir = ev.dir || ""; D.free = ev.free; renderOverlay();
        }
        break;
      }
      case "updateProgress": {
        const D = S.dialog;
        if (D && D.kind === "update" && D.key === ev.key) { D.step = ev.step; renderOverlay(); }
        break;
      }
      case "updateBroke": {
        toast("The game closed right after starting. If the update broke it, Undo it on the Updates page.", "error");
        S.updates = null; api.updates().catch(() => {});
        break;
      }
      case "updateDone": {
        const D = S.dialog;
        if (ev.kept) { toast(ev.error || "Its saves are copied to Lootnest's backups. You can reinstall it now.", ev.error ? "error" : ""); break; }
        if (D && D.kind === "update" && D.key === ev.key) { D.busy = false; D.error = ev.error || ""; if (!ev.error) { D.done = ev.update; D.undone = !!ev.undone; } renderOverlay(); }
        else if (ev.error) toast(ev.error, "error");
        else toast(ev.undone ? "The update was undone" : "Updated");
        api.downloads().catch(() => {});
        S.updates = null; S.news = {}; api.updates().catch(() => {});
        break;
      }
      case "storeFront": {
        S.front = ev.front || { error: ev.error || "Couldn't reach the store." };
        if (S.view === "find" && !S.find.q.trim()) { const r = $("#find-results"); if (r) r.innerHTML = findResults(); }
        break;
      }
      case "gameOffer": {
        S.offers[ev.appId] = ev;
        if (S.dialog && S.dialog.kind === "offer" && S.dialog.appId === ev.appId) renderOverlay();
        break;
      }
      case "gameSaves": {
        if (ev.info) { S.gsaves[ev.info.key] = ev.info; if (S.view === "game") render(); break; }
        if (ev.error) toast(ev.error, "error");
        else if (ev.unchanged) toast("Nothing new to back up: the saves are the same as the newest version.");
        else if (ev.saved) toast("Saved to the cloud");
        else if (ev.restored) toast(ev.skipped ? `Restored; ${ev.skipped} save location${ev.skipped === 1 ? "" : "s"} this PC doesn't have ${ev.skipped === 1 ? "was" : "were"} left out` : "Restored");
        if (ev.changed) { S.gsaves = {}; if (S.view === "game") render(); }
        if (ev.key) { const G = S.gsaves[ev.key]; if (G) G.busy = false; }
        break;
      }
      case "savesStatus":
        S.saves = ev.status;
        if (S.view === "cloud") render();
        break;
      case "savesChanged":
        if (ev.done && S.syncAsked) {
          S.syncAsked = false;
          if (ev.error) toast(ev.error, "error");
          else toast(ev.up || ev.down ? [ev.up ? `${ev.up} uploaded` : "", ev.down ? `${ev.down} brought down` : ""].filter(Boolean).join(", ") : "Everything is up to date");
        }
        if (S.view === "cloud") loadSaves();
        break;
      case "restoreInfo": {
        const D = S.dialog;
        if (D && D.kind === "restore") { D.busy = false; D.info = ev.info; D.error = ev.info.error || (ev.info.needPassword && D.triedPassword ? "That password doesn't open this backup." : ""); renderOverlay(); setTimeout(() => { const f = $("#rs-pw"); if (f) f.focus(); }, 30); }
        break;
      }
      case "restored": {
        const D = S.dialog;
        if (D && D.kind === "restore") { D.busy = false; D.applying = false; if (ev.ok) { D.done = ev.safety; D.result = ev.import; } else D.error = ev.error; renderOverlay(); }
        refresh();
        break;
      }
      case "exported": {
        const D = S.dialog;
        if (D && D.kind === "export") { D.busy = false; if (ev.error) D.error = ev.error; else D.done = ev.file; renderOverlay(); }
        else if (ev.error) toast(ev.error, "error");
        break;
      }
      case "pickerClosed":
        if (S.picker && S.picker.reqId) { S.picker = null; render(); }
        break;
      case "toast":
        if (ev.kind === "error" && S.picker && S.picker.working) S.picker.failed = true;
        toast(ev.text, ev.kind);
        break;
      case "browse": {
        const w = browseWaiters[ev.reqId];
        if (w) { delete browseWaiters[ev.reqId]; w(ev); }
        break;
      }
    }
  };

  function browse(kind, start) {
    const id = "b" + ++seq;
    return new Promise((resolve) => {
      browseWaiters[id] = resolve;
      api.browse(id, kind, start).catch((e) => { delete browseWaiters[id]; resolve({ error: errText(e) }); });
    });
  }

  async function refresh() {
    try { S.d = await api.state(); } catch (e) { toast(errText(e), "error"); return; }
    render();
  }

