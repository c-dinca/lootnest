// The kitchen sink: every component of internal/app/ui/system.css in every
// state, on one page. Dev only (build.mjs writes dist/kitchen-sink.html);
// nothing here ships in Lootnest.exe.
(function () {
  const ART = window.__ART || {};
  const art = (id, kind) => ART[`${id}-${kind}`] || "";
  const P = {
    sam: { name: "Sam", color: "#D9738E" },
    alex: { name: "Alex", color: "#6C9BD2" },
    ana: { name: "Ana", color: "#5FB49C" },
    dan: { name: "Dan", color: "#C9A35F" },
  };
  const i = (id) => `<svg class="ln-icon" aria-hidden="true"><use href="#i-${id}"/></svg>`;
  const av = (p, size = "") => `<span class="ln-avatar ln-player${size ? " ln-avatar--" + size : ""}" style="--player:${p.color}" aria-hidden="true">${p.name[0]}</span>`;
  const states = ["default", "hover", "active", "focus", "disabled", "loading"];
  const label = (s) => s[0].toUpperCase() + s.slice(1);

  function button(cls, text, icon, state, player) {
    const attrs = [];
    if (["hover", "active", "focus"].includes(state)) attrs.push(`data-force="${state}"`);
    if (state === "disabled") attrs.push("disabled");
    if (state === "loading") attrs.push('aria-busy="true"');
    const style = player ? ` style="--player:${player.color}"` : "";
    const spin = state === "loading" ? '<span class="ln-spinner" aria-hidden="true"></span>' : "";
    return `<button class="ln-button ${cls}${player ? " ln-player" : ""}" type="button"${style} ${attrs.join(" ")}>${spin}${icon ? i(icon) : ""}${text}</button>`;
  }

  const buttonRows = [
    ["Primary", "ln-button--primary", "Scan again", ""],
    ["Secondary", "ln-button--secondary", "Load someone's files", ""],
    ["Ghost", "ln-button--ghost", "Library", "back"],
    ["Destructive", "ln-button--danger", "Turn off profiles", "power"],
    ["Play as a player", "ln-button--play ln-button--lg", "Play as Sam", "play", P.sam],
    ["", "ln-button--play ln-button--lg", "Play as Alex", "play", P.alex],
    ["Small", "ln-button--secondary ln-button--sm", "Switch", "switch"],
  ];

  function tile({ id, title, store, owners = [], loaded, badge, none, state }) {
    const force = state && state !== "default" ? ` data-force="${state}"` : "";
    const cover = none
      ? `<div class="ln-tile__none"><b>${title}</b><span>${store}</span></div>`
      : `<img src="${art(id, "cover")}" alt="">`;
    const bars = owners.length
      ? `<div class="ln-tile__owners">${owners.map((p) => `<span class="ln-player" style="--player:${p.color}"${p === loaded ? " data-loaded" : ""}></span>`).join("")}</div>`
      : "";
    const meta = loaded
      ? `<span class="ln-mark ln-player" style="--player:${loaded.color}"></span>${loaded.name}'s files`
      : owners.length ? "Profiles on" : store;
    return `<button class="ln-tile" type="button"${force}>
      <div class="ln-tile__cover">${cover}${bars}${badge || ""}</div>
      <span class="ln-tile__title">${title}</span>
      <span class="ln-tile__meta">${meta}</span></button>`;
  }
  const tileLoading = `<div class="ln-tile" aria-busy="true"><div class="ln-tile__cover ln-skeleton"></div><span class="ln-skeleton" style="height:14px;width:80%"></span><span class="ln-skeleton" style="height:12px;width:50%"></span></div>`;

  const sec = (title, note, body) => `<section class="ks-sec">
    <div class="ln-section-head"><div><h2>${title}</h2>${note ? `<p>${note}</p>` : ""}</div></div>${body}</section>`;

  const swatch = (name, v, note) => `<div class="ks-sw"><span style="background:var(${v})"></span><b>${name}</b><code>${v}</code>${note ? `<em>${note}</em>` : ""}</div>`;

  const html = `
<div class="ln-backdrop" aria-hidden="true"><img src="${art(730, "hero")}" alt=""></div>
<main class="ks ln-root">
  <header class="ks-head">
    <h1>Lootnest design system</h1>
    <p>Every component in every state, from <code>internal/app/ui/system.css</code>. The background is Counter-Strike 2's art, blurred, as on a game page, so the glass shows what it does.</p>
  </header>

  ${sec("Colour", "The website's: obsidian, Ice Grey text, Racing Red for the gem and one primary key, no section colours. Glass only over art. Player colours are brought to the same lightness, so text on them always passes AA.", `
    <div class="ks-swatches">
      ${swatch("Shell", "--shell")}${swatch("Well", "--bg-well")}${swatch("Ground", "--bg-ground")}${swatch("Surface", "--bg-surface")}${swatch("Raised", "--bg-raised")}${swatch("Active", "--bg-active")}
      ${swatch("Text", "--fg-1", "14.7:1")}${swatch("Secondary", "--fg-2", "8.4:1")}${swatch("Tertiary", "--fg-3", "6.1:1")}${swatch("Disabled", "--fg-disabled")}
      ${swatch("Racing Red", "--accent")}${swatch("Lamp", "--lamp")}${swatch("Danger", "--danger", "8.4:1")}${swatch("Warning", "--warning", "10.4:1")}
    </div>
    <div class="ks-players">${Object.values(P).map((p) => `<div class="ks-pl ln-player" style="--player:${p.color}"><span class="ks-pl-fill">${p.name}</span><code>${p.color} → fill</code></div>`).join("")}
      <div class="ks-pl ln-player" style="--player:#B8E34A"><span class="ks-pl-fill">Custom</span><code>#B8E34A → fill</code></div></div>`)}

  ${sec("Type", "Tilt Warp for page titles, a game's name, the Play key and big numbers; Atkinson Hyperlegible Next for everything else, with Figtree's plain zero. All SIL Open Font License 1.1, embedded in the exe.", `
    <div class="ks-type">
      <div><span class="ks-k">64 / hero</span><span style="font:400 64px/1 var(--font-title)">Counter-Strike 2</span></div>
      <div><span class="ks-k">34 / page</span><span style="font:400 var(--text-page)/1 var(--font-title)">Library</span></div>
      <div><span class="ks-k">26 / score</span><span style="font:400 var(--text-score)/1 var(--font-title)">412 h, 10 days</span></div>
      <div><span class="ks-k">16 / label</span><span class="ln-label" style="justify-self:start">Continue playing</span></div>
      <div><span class="ks-k">18 / value</span><span class="ln-num" style="font:750 var(--fs-18)/1.1 var(--font-ui)">44 h, 22 h, 2026</span></div>
      <div><span class="ks-k">14 / body</span><span style="font:500 var(--fs-14)/1.5 var(--font-ui)">Before a game starts, Lootnest swaps the right person's files into place.</span></div>
      <div><span class="ks-k">13 / secondary</span><span style="font:500 var(--fs-13)/1.5 var(--font-ui);color:var(--fg-2)">A copy is made before every swap.</span></div>
      <div><span class="ks-k">12 / label</span><span style="font:600 var(--fs-12)/1.5 var(--font-ui);color:var(--fg-2)">Steam account</span></div>
    </div>`)}

  ${sec("Shape and depth", "Radius by role. Depth by light: keys carry a fine top light, panels a 1px rim; only floating layers cast a shadow.", `
    <div class="ks-shapes">
            <div class="ks-shape" style="border-radius:var(--radius-tag)"><b>4px</b><span>Tags, keyboard keys</span></div>
      <div class="ks-shape" style="border-radius:var(--radius-key);box-shadow:var(--key-light)"><b>10px</b><span>Keys (Play 12px)</span></div>
      <div class="ks-shape" style="border-radius:var(--radius-cover)"><b>8px</b><span>Covers</span></div>
      <div class="ks-shape" style="border-radius:var(--radius-panel);box-shadow:var(--lip-panel)"><b>14px</b><span>Panels, toasts</span></div>
      <div class="ks-shape" style="border-radius:var(--radius-round);width:96px"><b>Round</b><span>Avatars, pills</span></div>
      <div class="ks-shape ks-glass" style="background:var(--glass);border-radius:var(--radius-glass)"><b>Glass</b><span>The sidebar</span></div>
      <div class="ks-shape ks-glass" style="background:var(--glass-bar)"><b>Dock</b><span>The play dock</span></div>
      <div class="ks-shape ks-glass" style="background:var(--glass-panel)"><b>Panel</b><span>On art</span></div>
      <div class="ks-shape ks-glass" style="background:var(--glass-overlay);box-shadow:var(--shadow-overlay)"><b>Overlay</b><span>Toasts, dialogs</span></div>
    </div>`)}

  ${sec("Buttons", "One primary per page. When the action is for a player, it takes that player's colour instead.", `
    <div class="ks-grid" style="--cols:${states.length}">
      <span></span>${states.map((s) => `<span class="ks-col">${label(s)}</span>`).join("")}
      ${buttonRows.map(([name, cls, text, icon, player]) => `<span class="ks-row-label">${name}</span>${states.map((s) => `<div>${button(cls, text, icon, s, player)}</div>`).join("")}`).join("")}
    </div>`)}

  ${sec("Players", "Avatars, the chips you pick someone with, and tags.", `
    <div class="ks-flow">
      ${av(P.sam, "20")}${av(P.sam)}${av(P.alex, "32")}${av(P.ana, "48")}
      <span class="ln-avatar ln-avatar--72 ln-player" style="--player:${P.dan.color}" aria-hidden="true">D</span>
    </div>
    <div class="ks-grid" style="--cols:6">
      <span></span>${["Default", "Hover", "Active", "Focus", "Loaded", "Disabled"].map((s) => `<span class="ks-col">${s}</span>`).join("")}
      <span class="ks-row-label">Chip</span>
      ${["", "hover", "active", "focus", "pressed", "disabled"].map((s) => `<div><button class="ln-chip ln-player" type="button" style="--player:${P.alex.color}"${s === "pressed" ? ' aria-pressed="true"' : ""}${s === "disabled" ? " disabled" : ""}${["hover", "active", "focus"].includes(s) ? ` data-force="${s}"` : ""}>${av(P.alex)}Alex</button></div>`).join("")}
    </div>
    <div class="ks-flow">
      <span class="ln-tag">Seated</span>
      <span class="ln-tag ln-tag--player ln-player" style="--player:${P.sam.color}">Playing</span>
      <span class="ln-tag">Steam</span>
      <span class="ln-tag">Profiles on</span>
      <span class="ks-inline"><span class="ln-mark ln-player" style="--player:${P.sam.color}"></span>Sam's files</span>
    </div>`)}

  ${sec("Game tiles", "The cover is the content. A colour code along the bottom shows who has a profile; the tall bar is whose files are loaded.", `
    <div class="ks-tiles">
      <div>${tile({ id: 730, title: "Counter-Strike 2", store: "Steam", owners: [P.alex, P.sam], loaded: P.sam })}<span class="ks-cap">Profiles on, Sam loaded</span></div>
      <div>${tile({ id: 1086940, title: "Baldur's Gate 3", store: "Steam", owners: [P.sam, P.alex, P.ana], loaded: P.ana, state: "hover" })}<span class="ks-cap">Hover</span></div>
      <div>${tile({ id: 1426210, title: "It Takes Two", store: "Steam", owners: [P.alex, P.sam], loaded: P.alex, state: "focus" })}<span class="ks-cap">Focus</span></div>
      <div>${tile({ id: 1091500, title: "Cyberpunk 2077", store: "Steam", owners: [P.sam], loaded: P.sam, badge: `<span class="ln-tile__badge ln-tag ln-tag--player ln-player" style="--player:${P.sam.color}">Playing</span>` })}<span class="ks-cap">Playing</span></div>
      <div>${tile({ id: 250900, title: "The Binding of Isaac: Rebirth", store: "Steam" })}<span class="ks-cap">Long title, profiles off</span></div>
      <div>${tile({ title: "Phantom Blade Zero", store: "Game folder", none: true })}<span class="ks-cap">No art</span></div>
      <div>${tileLoading}<span class="ks-cap">Loading</span></div>
    </div>`)}

  ${sec("Metadata", "Facts you scan: the value first, then what it is.", `
    <dl class="ln-meta">
      <div class="ln-meta__item"><dd class="ln-num">44<small>h</small></dd><dt><span class="ln-mark ln-player" style="--player:${P.sam.color}"></span>Sam</dt></div>
      <div class="ln-meta__item"><dd class="ln-num">22<small>h</small></dd><dt><span class="ln-mark ln-player" style="--player:${P.alex.color}"></span>Alex</dt></div>
      <div class="ln-meta__item"><dd class="ln-num">412<small>h</small></dd><dt>Steam account</dt></div>
      <div class="ln-meta__item"><dd class="ln-meta__text">5 hours ago</dd><dt>Alex played last</dt></div>
      <div class="ln-meta__item"><dd class="ln-meta__text">Steam</dd><dt>Store</dt></div>
    </dl>
    <div class="ks-two">
      <div><div class="ln-section-head ln-section-head--minor"><h2>Last played</h2></div>
        <dl class="ln-facts">
          <div><dt>${av(P.sam, "20")}Sam</dt><dd>Yesterday</dd></div>
          <div><dt>${av(P.alex, "20")}Alex</dt><dd>5 hours ago</dd></div>
        </dl></div>
      <div><div class="ln-section-head ln-section-head--minor"><h2>Loading</h2></div>
        <dl class="ln-meta" aria-busy="true">
          <div class="ln-meta__item"><dd><span class="ln-skeleton" style="display:block;height:20px;width:48px"></span></dd><dt><span class="ln-skeleton" style="display:block;height:12px;width:56px"></span></dt></div>
          <div class="ln-meta__item"><dd><span class="ln-skeleton" style="display:block;height:20px;width:48px"></span></dd><dt><span class="ln-skeleton" style="display:block;height:12px;width:56px"></span></dt></div>
        </dl></div>
      <div><div class="ln-section-head ln-section-head--minor"><h2>Empty</h2></div>
        <dl class="ln-meta"><div class="ln-meta__item"><dd class="ln-meta__text">Nobody yet</dd><dt>Time played</dt></div></dl></div>
    </div>`)}

  ${sec("Section labels and action lists", "A section's title is plain Tilt Warp; groups of actions take a small title and hairline rows.", `
    <div class="ln-section-head" data-sec="library"><div><h2>Library<span class="ln-count ln-num">13</span></h2></div>${button("ln-button--secondary ln-button--sm", "Add game", "plus", "default")}</div>
    <div class="ks-flow">${[["home", "Home"], ["store", "Store"], ["updates", "Updates"], ["ach", "Achievements"], ["saves", "Saves"], ["profiles", "Profiles"], ["neutral", "Settings"]].map(([k, t]) => `<span class="ln-label" data-sec="${k}">${t}</span>`).join("")}</div>
    <div class="ks-two">
      <div><div class="ln-section-head ln-section-head--minor"><h2>Profiles</h2></div>
        <div class="ln-list">
          <button class="ln-row" type="button">${i("load")}<span><b>Load someone's files</b><span>Without starting the game</span></span></button>
          <button class="ln-row" type="button" data-force="hover">${i("ask")}<span><b>Ask who's playing in Steam</b><span>When the game starts from Steam's Play button</span></span></button>
          <button class="ln-row" type="button" data-force="focus">${i("edit")}<span><b>Edit game</b><span>How it starts and what profiles swap</span></span></button>
          <button class="ln-row" type="button" disabled>${i("drive")}<span><b>Move to another drive</b><span>Not while the game is running</span></span></button>
        </div></div>
      <div><div class="ln-section-head ln-section-head--minor"><h2>Danger</h2></div>
        <div class="ln-list"><button class="ln-row ln-row--danger" type="button">${i("power")}<span><b>Turn off profiles</b></span></button></div>
        <p class="ks-note">Rows: default, hover, focus, disabled.</p></div>
    </div>`)}

  ${sec("Segmented control", "For a choice between views, like who the Library shows. Arrow keys move between options.", `
    <div class="ks-flow">
      <div class="ln-segmented" role="radiogroup" aria-label="Show">
        <button type="button" role="radio" aria-checked="true" tabindex="0">Everyone <span class="ln-count ln-num">13</span></button>
        <button type="button" role="radio" aria-checked="false" tabindex="-1">Mine <span class="ln-count ln-num">6</span></button>
        <button type="button" role="radio" aria-checked="false" tabindex="-1">With profiles <span class="ln-count ln-num">3</span></button>
      </div>
      <div class="ln-segmented" role="radiogroup" aria-label="Sizes">
        <button type="button" role="radio" aria-checked="false" tabindex="-1" data-force="hover">Small</button>
        <button type="button" role="radio" aria-checked="true" tabindex="0" data-force="focus">Medium</button>
        <button type="button" role="radio" aria-checked="false" tabindex="-1" disabled>Large</button>
      </div>
    </div>
    <p class="ks-note">Second control: hover, selected with focus, disabled.</p>`)}

  ${sec("Switches", "Off is a dark slot; on is lit green, like a power LED.", `
    <div class="ks-flow">
      <label class="switch"><input type="checkbox" aria-label="Off"><span></span></label>
      <label class="switch"><input type="checkbox" checked aria-label="On"><span></span></label>
      <label class="switch"><input type="checkbox" disabled aria-label="Off, disabled"><span></span></label>
      <label class="switch"><input type="checkbox" checked disabled aria-label="On, disabled"><span></span></label>
    </div>
    <p class="ks-note">Off, on, off disabled, on disabled.</p>`)}

  ${sec("Tabs", "For the parts of one page, like a game's Overview and Achievements. Pills in a well, the current one lit; arrow keys move between tabs.", `
    <div class="ln-tabs" role="tablist" aria-label="Hollow Knight">
      <button type="button" role="tab" aria-selected="true" tabindex="0">Overview</button>
      <button type="button" role="tab" aria-selected="false" tabindex="-1" data-force="hover">Achievements</button>
      <button type="button" role="tab" aria-selected="false" tabindex="-1" data-force="focus">Saves</button>
      <button type="button" role="tab" aria-selected="false" tabindex="-1">Timeline</button>
    </div>
    <p class="ks-note">Selected, hover, focus, default.</p>`)}

  ${sec("Toasts", "Short, specific, and the same words as the action that caused them.", `
    <div class="ks-toasts">
      <div class="ln-toast ln-player" style="--player:${P.sam.color}">${av(P.sam)}<p>Sam's files are loaded<span>Counter-Strike 2. A copy of Alex's was made first.</span></p>${button("ln-button--ghost ln-button--sm", "Open backups", "", "default")}</div>
      <div class="ln-toast ln-toast--success">${i("ok")}<p>Saved to the cloud<span>Hollow Knight, 2 files</span></p></div>
      <div class="ln-toast">${i("info")}<p>Lootnest 1.16.0 is out. Update from the sidebar.</p></div>
      <div class="ln-toast ln-toast--error">${i("alert")}<p>Couldn't move Cyberpunk 2077<span>D: has 12 GB free and the game needs 64 GB.</span></p>${button("ln-button--ghost ln-button--sm", "Choose another drive", "", "default")}</div>
    </div>`)}

  ${sec("Empty, loading and error", "Each one says what is going on and what to do next.", `
    <div class="ks-states">
      <div class="ln-glass ln-state">${i("games")}<div><h3>No games found yet</h3><p>Lootnest looks in Steam, Epic, GOG, Xbox, EA, Ubisoft, Battle.net, Riot, Amazon, Rockstar, itch.io, game folders and your shortcuts. Add a game by hand if yours isn't found.</p>
        <div class="ln-actions">${button("ln-button--primary", "Add a game", "plus", "default")}${button("ln-button--secondary", "Scan again", "", "default")}</div></div></div>
      <div class="ln-glass ln-state"><span class="ln-spinner" aria-hidden="true"></span><div><h3>Looking for games on this PC</h3><p>Steam and Epic are done. Checking GOG, Xbox and your game folders.</p></div></div>
      <div class="ln-glass ln-state ln-state--error">${i("offline")}<div><h3>Couldn't reach the internet</h3><p>Covers and news need a connection. Your games, profiles and saves work offline.</p>
        <div class="ln-actions">${button("ln-button--secondary", "Try again", "", "default")}</div></div></div>
      <div class="ks-inline-states">
        <div class="ln-state ln-state--inline"><span class="ln-spinner" aria-hidden="true"></span><div><h3>Looking for its saves</h3><p>Checking the usual places and PCGamingWiki's list</p></div></div>
        <div class="ln-state ln-state--inline ln-state--error">${i("alert")}<div><h3>Couldn't read the save folder</h3><p>Another program has it open. Close the game, then try again.</p></div></div>
      </div>
    </div>`)}
</main>`;

  document.body.insertAdjacentHTML("beforeend", html);

  // Segmented controls: arrow keys move the choice, like radio buttons.
  document.querySelectorAll(".ln-segmented").forEach((g) => {
    g.addEventListener("keydown", (e) => {
      if (!["ArrowLeft", "ArrowRight"].includes(e.key)) return;
      const opts = [...g.querySelectorAll("button:not(:disabled)")];
      const at = opts.indexOf(document.activeElement);
      const next = opts[(at + (e.key === "ArrowRight" ? 1 : opts.length - 1)) % opts.length];
      opts.forEach((o) => { o.setAttribute("aria-checked", String(o === next)); o.tabIndex = o === next ? 0 : -1; });
      next.focus();
      e.preventDefault();
    });
    g.addEventListener("click", (e) => {
      const b = e.target.closest("button");
      if (!b || b.disabled) return;
      g.querySelectorAll("button").forEach((o) => { o.setAttribute("aria-checked", String(o === b)); o.tabIndex = o === b ? 0 : -1; });
    });
  });
})();
