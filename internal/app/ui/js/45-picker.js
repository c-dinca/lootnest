  // ---------------------------------------------------------------- picker --
  function openPicker(gameId, mode, reqId) {
    const g = game(gameId);
    if (!g) { toast("That game isn't in Lootnest anymore.", "error"); if (reqId) api.cancelRequest(reqId); return; }
    if (S.picker && S.picker.reqId && S.picker.reqId !== reqId) api.cancelRequest(S.picker.reqId);
    S.menuFor = null;
    if (S.dialog && S.dialog.kind !== "welcome") S.dialog = null;
    S.picker = { gameId, mode: mode || "play", reqId: reqId || "" };
    render();
    setTimeout(() => { const pk = $(".picker"); if (pk) pk.focus(); }, 30); // keyboard 1-9 works without favouring anyone
  }

  function closePicker() {
    if (S.picker && S.picker.reqId && !S.picker.working) api.cancelRequest(S.picker.reqId);
    S.picker = null;
    render();
  }

  // A slot's line: when they last played it (whose files are in shows on
  // the slot's label), else whether they have a profile yet.
  function slotState(gid, pid) {
    const at = lastBy({ kind: "game", id: gid }, pid), i = profileInfo(gid, pid);
    if (at) return "Played " + ago(at);
    if (i.saved) return "Saved " + ago(i.saved);
    return S.d.active[gid] === pid ? "Files loaded now" : hasProfile(gid, pid) ? "Has saves here" : "First time";
  }

  // A new person from the picker. It starts fresh or from a copy of whose
  // files are in, as any first-timer does. Runs are made in their person's
  // column (newRunRow).
  function newProfileSlot(P) {
    if (P.creating !== "person") return `<button class="slot slot-new" data-action="np-open">
        <span class="slot-new-plus" aria-hidden="true">${GI.plus}</span>
        <span class="name">New profile</span><span class="state">Someone new</span></button>`;
    return `<div class="slot slot-new editing" role="group" aria-label="New profile">
        <span class="slot-new-plus" aria-hidden="true">${GI.plus}</span>
        <label class="name" for="np-name">New profile</label>
        <input id="np-name" class="ln-input" type="text" maxlength="24" autocomplete="off" spellcheck="false" placeholder="Their name" value="${esc(P.newName || "")}">
        <span class="error-text" id="np-error" role="alert">${esc(P.newError || "")}</span>
        <div class="slot-new-actions"><button class="ln-button ln-button--ghost ln-button--sm" data-action="np-cancel">Cancel</button><button class="ln-button ln-button--primary ln-button--sm" data-action="np-create">${P.mode === "load" ? "Make it" : "Make it and play"}</button></div></div>`;
  }
  // Naming a new run, in its person's column.
  function newRunRow(P, per) {
    return `<div class="pk-run pk-run--edit" role="group" aria-label="New run for ${esc(per.name)}">
        <input id="np-name" class="ln-input" type="text" maxlength="24" autocomplete="off" spellcheck="false" placeholder="Its name" value="${esc(P.newName || "")}" aria-label="The run's name">
        <span class="error-text" id="np-error" role="alert">${esc(P.newError || "")}</span>
        <div class="slot-new-actions"><button class="ln-button ln-button--ghost ln-button--sm" data-action="np-cancel">Cancel</button><button class="ln-button ln-button--primary ln-button--sm" data-action="np-create">${P.mode === "load" ? "Make it" : "Make it and play"}</button></div></div>`;
  }
  // A person's column: their own profile as the big key, then their runs of
  // this game under it, a list that scrolls when it's long, and New run.
  // The numbers 1 to 9 follow pickOrder: people first, then runs.
  function pickCol(P, g, per, order, i) {
    const runs = runsOf(per, g.id), at = (p) => order.indexOf(p);
    const kbd = (p, cls) => at(p) >= 0 && at(p) < 9 ? `<kbd class="${cls}" aria-hidden="true">${at(p) + 1}</kbd>` : "";
    const state = (p) => S.d.active[g.id] === p.id ? `<span class="slot-in">In the game now</span>` : esc(slotState(g.id, p.id));
    const inn = (p) => (S.d.active[g.id] === p.id ? " in" : "");
    const head = `<button class="slot ln-player${inn(per)}" style="--player:${esc(per.color)}" data-action="pick" data-player="${esc(per.id)}">
          ${kbd(per, "slot-kbd")}${lnAvatar(per, "120")}<span class="name">${esc(per.name)}</span><span class="state">${state(per)}</span></button>`;
    const row = (r) => `<button class="pk-run${inn(r)}" data-action="pick" data-player="${esc(r.id)}" aria-label="${esc(runName(r))}">
          <span class="pk-run-mark" aria-hidden="true"></span><span class="pk-run-text"><b>${esc(runLabel(r))}</b><span>${state(r)}</span></span>${kbd(r, "pk-kbd")}</button>`;
    const add = P.creating === per.id ? newRunRow(P, per)
      : canNewRun(g.id) ? `<button class="pk-run pk-run--new" data-action="np-run" data-person="${esc(per.id)}">${GI.plus}<span class="pk-run-text"><b>New run</b></span></button>` : "";
    const list = runs.length || add ? `<div class="pk-runs" role="group" aria-label="${esc(possessive(per.name))} runs">
          <span class="pk-runs-head">${runs.length === 1 ? "1 run" : runs.length ? runs.length + " runs" : "Runs"}</span>
          <div class="pk-runs-list${runs.length + (add ? 1 : 0) > 5 ? " is-long" : ""}">${runs.map(row).join("")}${add}</div></div>` : "";
    return `<div class="pk-col ln-player" style="--player:${esc(per.color)};--i:${i}">${head}${list}</div>`;
  }

  function viewPicker() {
    const P = S.picker, g = game(P.gameId);
    if (!g) return "";
    if (P.working) {
      const to = player(P.playerId), from = P.from && P.from !== P.playerId && player(P.from);
      const s = session(g.id);
      const m = s && s.phase === "mods" && s.mods, line = m ? `${esc(sessionText(s))}${m.total ? `, ${m.done} of ${m.total} files` : ""}` : s && s.phase === "launching" ? `Starting ${esc(g.name)}` : P.mode !== "load" ? `Loading ${esc(whoseOf(P.playerId))} files and starting ${esc(g.name)}` : `Loading ${esc(whoseOf(P.playerId))} files`;
      return `<div class="picker ln-root" role="dialog" aria-label="${esc(line)}"><div class="picker-head"><div></div></div>
        <div class="swap" aria-live="polite">
          <div class="swap-bay">${from ? `<div class="swap-out" aria-hidden="true">${lnAvatar(from, "120")}</div>` : ""}<div class="swap-in">${to ? lnAvatar(to, "120") : ""}</div></div>
          <div class="swap-text"><b>${esc(pperson(P.playerId))}</b>${player(P.playerId) ? runTag(player(P.playerId)) : ""}<p><span class="ln-spinner" aria-hidden="true"></span>${line}</p>
            ${from ? `<p class="swap-safe">${GI.copy}A copy of ${esc(whoseOf(from.id))} files was made first.</p>` : `<p class="swap-safe">${GI.copy}A copy is made before every swap.</p>`}</div>
        </div><div></div></div>`;
    }
    if (P.ask === "owner") {
      return `<div class="picker"><div class="picker-head"><div></div><button class="ln-button ln-button--ghost" data-action="picker-back">${GI.back}Back</button></div>
        <div class="question"><div><h2>Whose files are in ${esc(g.name)} right now?</h2>
        <p>The settings and saves on this PC become theirs.</p></div>
        <div class="choices">
          ${playersFor(g.id).map((p) => `<button class="choice" data-action="answer-owner" data-player="${esc(p.id)}"><span class="dot" style="background:${esc(p.color)}"></span><span><b>${esc(whoseOf(p.id))}</b></span></button>`).join("")}
          <button class="choice" data-action="answer-owner" data-player="none"><span class="dot"></span><span><b>Nobody's, or not sure</b><span>Kept as a backup</span></span></button>
        </div></div><div></div></div>`;
    }
    if (P.ask === "cloud") {
      const C = P.conflict || {}, n = pname(P.playerId), other = C.computer || "another PC";
      return `<div class="picker"><div class="picker-head"><div></div><button class="ln-button ln-button--ghost" data-action="picker-back">${GI.back}Back</button></div>
        <div class="question"><div><h2>${esc(possessive(n))} ${esc(g.name)} saves changed on two PCs</h2>
        <p>Here and on ${esc(other)}${C.changed ? " (" + esc(ago(C.changed)) + ")" : ""}. The one you don't choose is kept in backups.</p></div>
        <div class="choices">
          <button class="choice" data-action="answer-cloud" data-answer="here"><span class="dot" style="background:${esc(color(P.playerId))}"></span><span><b>Keep this PC's saves</b></span></button>
          <button class="choice" data-action="answer-cloud" data-answer="cloud"><span class="dot"></span><span><b>Use the saves from ${esc(other)}</b></span></button>
        </div></div><div></div></div>`;
    }
    if (P.ask === "newMode") {
      const owner = P.owner && P.owner !== "none" ? P.owner : S.d.active[g.id];
      const ownerName = owner && player(owner) ? whoseOf(owner) + " files" : "the current files";
      const me = player(P.playerId), run = me && parentOf(me);
      const ask = run ? `${whoseOf(P.playerId)} run starts here` : `${pname(P.playerId)} hasn't played ${g.name} yet`;
      return `<div class="picker"><div class="picker-head"><div></div><button class="ln-button ln-button--ghost" data-action="picker-back">${GI.back}Back</button></div>
        <div class="question"><div><h2>${esc(ask)}</h2>
        <p>The files on this PC now are kept.</p></div>
        <div class="choices">
          <button class="choice" data-action="answer-new" data-mode="fresh"><span class="dot" style="background:${esc(color(P.playerId))}"></span><span><b>Start fresh</b><span>Game defaults, no saves</span></span></button>
          <button class="choice" data-action="answer-new" data-mode="copy"><span class="dot"${owner && player(owner) ? ` style="background:${esc(color(owner))}"` : ""}></span><span><b>Start from a copy of ${esc(ownerName)}</b></span></button>
          ${playersFor(g.id).filter((p) => p.id !== owner && p.id !== P.playerId && hasProfile(g.id, p.id)).map((p) => `<button class="choice" data-action="answer-new" data-mode="copy" data-from="${esc(p.id)}"><span class="dot" style="background:${esc(p.color)}"></span><span><b>Start from a copy of ${esc(whoseOf(p.id))} files</b></span></button>`).join("")}
        </div></div><div></div></div>`;
    }
    const order = pickOrder(g.id), folks = order.filter((p) => !parentOf(p));
    const many = order.length > 1 ? "1 to " + Math.min(order.length, 9) : "1";
    return `<div class="picker ln-root" role="dialog" tabindex="-1" aria-label="Who's playing ${esc(g.name)}?">
      <div class="picker-head"><div><h1>Who's playing ${esc(g.name)}?</h1>${P.mode === "steam" ? `<p class="hint">Steam is starting the game.</p>` : P.mode === "load" ? `<p class="hint">Loads their files without starting the game.</p>` : ""}</div>
        <button class="ln-button ln-button--ghost" data-action="picker-close">${P.mode === "steam" ? "Don't start" : "Cancel"}</button></div>
      <div class="slots pk-people">
        ${folks.map((per, i) => pickCol(P, g, per, order, i)).join("")}
        ${canNewPerson() ? newProfileSlot(P) : ""}
      </div>
      <div class="picker-foot"><span>Press ${many}, or Esc ${P.mode === "steam" ? "to not start the game" : "to cancel"}</span><span>${esc(g.name)}</span></div>
    </div>`;
  }

  async function runPick() {
    const P = S.picker;
    const key = document.querySelector(`.slot[data-player="${CSS.escape(P.playerId)}"]`);
    if (key) key.classList.add("picked"); // the key stays lit until the swap shows
    const req = { gameId: P.gameId, playerId: P.playerId, firstOwner: P.owner || "", newProfile: P.newMode || "", copyFrom: P.copyFrom || "", reqId: P.reqId || "", cloud: P.cloud || "" };
    try {
      const res = await (P.mode === "steam" ? api.steamPick(req) : P.mode === "play" ? api.play(req) : api.load(req));
      if (!S.picker) return;
      if (res && res.need) { S.picker.ask = res.need; S.picker.conflict = res.cloud || null; render(); return; }
      S.picker.working = true;
      render();
    } catch (e) {
      toast(errText(e), "error");
      if (S.picker) { S.picker.ask = null; S.picker.working = false; render(); }
    }
  }

  function onSessions() {
    const P = S.picker;
    if (!P || !P.working) return;
    const s = session(P.gameId);
    // The files are in once the session is past loading (or gone, unless it failed).
    if (!P.swapped && (s ? s.phase !== "loading" : !P.failed)) { P.swapped = true; cue("swap", "game:" + P.gameId); }
    // The picker stays up ("Starting …") until the game is actually running.
    if (!s || s.phase === "playing") { S.picker = null; return; }
    render();
  }

