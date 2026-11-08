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
        loadFound().then(() => {
          if (S.owned && !S.found.refreshing) api.ownedGames().catch(() => {}); // an install or uninstall may be done
          if (S.dialog && S.dialog.kind === "sources") loadSources();
        });
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
      case "captures":
        S.captures[ev.key] = { items: ev.items || [], counts: ev.counts || {}, gameBarOff: !!ev.gameBarOff };
        if (S.view === "game") render();
        break;
      case "capturesIndexed": // asked again, so what's shown stays until the answer
        capAsked.clear();
        if (S.view === "game") render();
        if (S.unsorted || unsortedAsked) api.unsortedCaptures().catch(() => {});
        break;
      case "itch":
      case "itchUploads":
      case "itchProgress":
      case "itchUpdates":
      case "itchDone":
        itchEvent(ev);
        break;
      case "gog":
      case "gogInstallers":
      case "gogProgress":
      case "gogUpdates":
      case "gogDone":
        gogEvent(ev);
        break;
      case "playnitePlan":
      case "playniteDone":
        playniteEvent(ev);
        break;
      case "unsortedCaptures":
        S.unsorted = { items: ev.items || [], total: ev.total || 0, folders: ev.folders || [] };
        {
          // In the drawer, focus stays on the row it was on, or the one that
          // took the place of a row just given a game.
          const D = S.dialog && S.dialog.kind === "unsorted" ? S.dialog : null, sels = () => [...document.querySelectorAll("[data-assign]")];
          const at = !D ? -1 : D.focusAt != null ? D.focusAt : sels().indexOf(document.activeElement);
          if (S.view === "settings") render(); else if (D) renderOverlay();
          if (D) D.focusAt = null;
          if (at >= 0) (sels()[Math.min(at, sels().length - 1)] || $('.ln-drawer [data-action="dialog-cancel"]')).focus();
        }
        break;
      case "savePoints":
        S.points[ev.key] = Object.assign({}, S.points[ev.key] || {}, { points: ev.points || [], error: ev.error || "", player: ev.player || "", running: !!ev.running, profiles: !!ev.profiles, loaded: true });
        if (S.view === "game") render();
        break;
      case "savePointMade": {
        const P = S.points[ev.key] || (S.points[ev.key] = {});
        P.busy = false;
        tlAsked.delete(ev.key);
        if (ev.error) toast(ev.error, "error");
        else if (ev.restored) toast(`Back to ${ev.point ? ev.point.name : "the save point"}. The saves from before are kept as a point.`);
        else if (ev.point) { if (!ev.hotkey) toast(`Save point made: ${ev.point.name}`); if (!ev.native) cue("point", ev.key); }
        if (S.view === "game") render();
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
      case "sizes":
        S.sizes = ev;
        if ((S.view === "settings" && S.setTab === "storage") || (S.view === "library" && S.sort === "size") || S.view === "game") render();
        if (S.dialog && S.dialog.kind === "movefolder") renderOverlay();
        break;
      case "folderMoveProgress":
        S.folderMove = ev;
        if (S.dialog && S.dialog.kind === "movefolder") renderOverlay();
        break;
      case "folderMoveDone": {
        S.folderMove = null;
        if (S.dialog && S.dialog.kind === "movefolder" && S.dialog.key === ev.key) { S.dialog = null; renderOverlay(); }
        if (ev.error) toast(`Couldn't move ${ev.name}: ${ev.error}`, "error");
        else if (ev.cancelled) toast(`${ev.name} stays where it was.`);
        else {
          if (S.page && S.page.kind === "found" && S.page.id === ev.key) S.page.id = ev.newKey;
          toast(ev.leftover ? `${ev.name} is now in ${ev.to}. Its old folder couldn't go to the Recycle Bin and is still in ${ev.leftover}.` : `${ev.name} is now in ${ev.to}. Its old folder is in the Recycle Bin.`);
        }
        loadFound().then(refresh); // the page's new key is in the Library first
        break;
      }
      case "modPlan": {
        const D = S.dialog;
        if (D && D.kind === "modplan" && D.key === ev.key && D.path === ev.path) { Object.assign(D, { busy: false, mod: ev.mod || null, plan: ev.plan || null, refused: ev.refused || "", error: ev.error || "" }); renderOverlay(); }
        break;
      }
      case "modsProgress": {
        const R = (S.modsRun[ev.key] = Object.assign(S.modsRun[ev.key] || {}, { done: ev.done, total: ev.total }));
        const box = [...document.querySelectorAll(".gp-mods")].find((e) => e.dataset.key === ev.key);
        const bar = box && box.querySelector(".meter i"), line = box && box.querySelector(".mover [role=status]");
        if (bar) bar.style.width = `${R.total ? Math.round((R.done / R.total) * 100) : 0}%`;
        if (line) line.textContent = modsRunText(R);
        break;
      }
      case "modsDone": {
        delete S.modsRun[ev.key];
        const r = ev.result || {};
        if (ev.error) toast(ev.error, "error");
        else toast((ev.what === "apply" ? "The mods are in the game's folder." : ev.what === "remove" ? "The mod is out of the game's folder and removed." : "The game is back as it was.")
          + (r.changed ? ` ${r.changed} file${r.changed === 1 ? "" : "s"} had changed since the mods went in; Lootnest kept ${r.changed === 1 ? "it" : "them"}.` : ""),
          "", r.changed ? { label: "Show", run: () => api.open("modsChanged", ev.key).catch((e) => toast(errText(e), "error")) } : undefined);
        loadMods(ev.key);
        break;
      }
      case "modsFinished":
        S.mods = {};
        if (S.view === "game") render();
        break;
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
        S.updates = { games: ev.games || [], newsOn: !!ev.newsOn, at: Date.now() }; S.updChecking = S.updAsking = false;
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
      case "owned": {
        S.owned = ev.games || [];
        S.owned.forEach((o) => {
          const k = "n:" + o.name.toLowerCase();
          if (o.cover && !S.covers[k]) S.covers[k] = o.cover;
          if (o.hero && !S.art["hero:" + k]) S.art["hero:" + k] = o.hero;
        });
        if (S.itch.refreshing && !ev.itchReading) { S.itch.refreshing = false; S.itch.error = ev.itchError || ""; if (S.view === "settings") render(); if (!ev.itchError) toast("Lootnest read your itch.io library again."); }
        if (S.gog.refreshing && !ev.gogReading) { S.gog.refreshing = false; S.gog.error = ev.gogError || ""; if (S.view === "settings") render(); if (!ev.gogError) toast("Lootnest read your GOG library again."); }
        renderSideGames();
        if (S.view === "library" || S.view === "game" && S.page && S.page.kind === "owned") render();
        break;
      }
      case "removePlan": {
        const D = S.dialog;
        if (D && D.kind === "removepc" && D.key === ev.key) { D.plan = ev.plan || null; D.error = ev.error || ""; renderOverlay(); }
        break;
      }
      case "gameRemoved": {
        const D = S.dialog && S.dialog.kind === "removepc" && S.dialog.key === ev.key ? S.dialog : null;
        if (ev.error) {
          if (D) { D.busy = false; D.error = ev.error; renderOverlay(); } else toast(ev.error, "error");
          break;
        }
        if (D) { S.dialog = null; renderOverlay(); }
        if (S.view === "game" && S.page && S.page.id === ev.key) pageBack();
        toast(`${ev.name} is in the Recycle Bin. Its saves are copied in Lootnest's backups.`);
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
        if (!ev.error) { api.itchRecords().catch(() => {}); api.gogRecords().catch(() => {}); } // a record from itch.io or GOG may have changed
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

