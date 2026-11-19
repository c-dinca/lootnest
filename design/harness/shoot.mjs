// Screenshots every page and state into design/<out>/ (default: before).
// node design/harness/build.mjs && node design/harness/shoot.mjs [out] [filter]
import { createRequire } from "node:module";
const { chromium } = createRequire(import.meta.url)("playwright"); // the preinstalled one
import fs from "node:fs";
import path from "node:path";
const here = path.dirname(new URL(import.meta.url).pathname);
const out = path.join(here, "..", process.argv[2] || "before");
const only = process.argv[3] || "";
fs.mkdirSync(out, { recursive: true });
const page = "file://" + path.join(here, "dist", "page.html");

const click = (sel) => `(() => { const e = document.querySelector(${JSON.stringify(sel)}); if (!e) throw new Error("no " + ${JSON.stringify(sel)}); e.click(); })()`;
const key = (k) => `document.dispatchEvent(new KeyboardEvent("keydown", { key: ${JSON.stringify(k)}, bubbles: true }))`;
const nav = (v) => click(`[data-view="${v}"]`);
const game = (attr) => [nav("library"), click(`[data-action="open-sheet"][${attr}]`)];

// name, query, steps (JS run in order, with a pause after each)
const shots = [
  ["home", "", []],
  ["home-6players", "players=6", []],
  ["home-1player", "players=1", []],
  ["home-empty", "games=0", []],
  ["home-loading", "state=loading", []],
  ["home-error", "state=error", []],
  ["first-run", "first=1&games=0", []],
  ["library", "", [nav("library")]],
  ["library-400", "games=400", [nav("library")]],
  ["library-empty", "games=0", [nav("library")]],
  ["side-installed", "", [nav("library"), `new Promise((r) => setTimeout(() => { const g = document.querySelector('#side-games [data-g="installed"]'); g.scrollIntoView({ block: "start" }); r(); }, 300))`]],
  ["library-sources", "", [nav("library"), click('[data-action="sources"]')]],
  ["library-sources-dir", "", [nav("library"), click('[data-action="sources"]'), click('[data-action="gamedir-add"]'), `new Promise((r) => setTimeout(() => { document.querySelectorAll(".ln-drawer *").forEach((e) => { if (e.scrollHeight > e.clientHeight + 4) e.scrollTop = e.scrollHeight; }); r(); }, 400))`]],
  ["library-scanning", "state=loading", [nav("library")]],
  ["library-offline", "state=offline", [nav("library")]],
  ["game-on", "", game('data-game="cs2"')],
  ["game-on-6players", "players=6", game('data-game="bg3"')],
  ["game-bright", "art=bright", game('data-game="bg3"')],
  ["game-on-playing", "state=playing", game('data-game="cs2"')],
  ["game-off-steam", "", game('data-key="steam:1091500"')],
  ["game-off-noart", "", game('data-key="dir:d:\\\\games\\\\phantom blade zero"')],
  ["game-off-nosaves", "", game('data-key="dir:d:\\\\games\\\\onimusha"')],
  ["game-off-longtitle", "", game('data-key="steam:250900"')],
  ["game-off-mismatched", "", game('data-key="steam:1326470"')],
  ["game-off-offline", "state=offline", game('data-key="steam:1091500"')],
  ["picker", "", [nav("library"), click('[data-action="play"][data-game="cs2"]')]],
  ["picker-6players", "players=6", [nav("library"), click('[data-action="play"][data-game="bg3"]')]],
  ["players", "", [nav("players")]],
  ["players-6", "players=6", [nav("players")]],
  ["player-profile", "", [nav("players"), click('[data-action="open-player"]')]],
  ["game-manage", "", [...game('data-game="cs2"'), click('[data-action="manage"]')]],
  ["game-ach-why", "", [...game('data-key="dir:d:\\\\games\\\\onimusha"'), click('[data-action="game-tab"][data-tab="achievements"]')]],
  ["game-ach-private", "why=private", [...game('data-key="dir:d:\\\\games\\\\onimusha"'), click('[data-action="game-tab"][data-tab="achievements"]')]],
  ["game-manage-hydra", "kept=1", [...game('data-key="dir:d:\\\\games\\\\hydra\\\\hollow tale"'), click('[data-action="manage"]')]],
  ["game-move-hydra", "kept=1", [...game('data-key="dir:d:\\\\games\\\\hydra\\\\hollow tale"'), click('[data-action="manage"]'), click('[data-action="movefolder-open"]')]],
  ["game-tab-ach", "", [...game('data-game="bg3"'), click('.gp-tabs [data-tab="achievements"]')]],
  ["achievements", "", [nav("achievements")]],
  ["updates", "", [nav("updates")]],
  ["store", "", [nav("find")]],
  ["bp", "", [click('[data-action="bp-open"]')]],
  ["bp-400", "games=400", [click('[data-action="bp-open"]')]],
  ["tv-who", "tv=1", []],
  ["tv-solo", "tv=1&players=1", []],
  ["tv-row2", "tv=1&players=1", [key("ArrowDown"), key("ArrowDown")]],
  ["tv-game", "tv=1&players=1", [key("ArrowRight"), key("Enter")]],
  ["tv-game-ach", "tv=1&players=1", [key("ArrowRight"), key("ArrowRight"), key("ArrowRight"), key("Enter"), key("ArrowDown"), key("ArrowRight")]],
  ["tv-points", "tv=1&players=1", [key("ArrowRight"), key("ArrowRight"), key("ArrowRight"), key("Enter"), key("ArrowRight"), key("Enter"), key("ArrowDown")]],
  ["tv-points-ask", "tv=1&players=1", [key("ArrowRight"), key("ArrowRight"), key("ArrowRight"), key("Enter"), key("ArrowRight"), key("Enter"), key("ArrowDown"), key("Enter")]],
  ["tv-ds5", "tv=1&players=1&pad=ds5", [key("ArrowRight"), key("Enter")]],
  ["tv-launch", "tv=1&players=1", ['window.__ps({ type: "tvGame", key: "game:bg3", phase: "starting" })']],
  ["bp-game-menu", "players=1", [click('[data-action="bp-open"]'), 'window.__ps({ type: "tvGame", key: "bg3", phase: "playing" })', 'window.__ps({ type: "tvMenu", on: true, key: "bg3", shot: (document.querySelector("#bp .bp-bd-l[src]") || {}).src || "" })']],
  ["tv-home-menu", "tv=1&players=1", ['window.__ps({ type: "tvGame", key: "game:bg3", phase: "playing" })', 'window.__ps({ type: "tvMenu", on: true, key: "game:bg3" })']],
  ["tv-library", "tv=1&players=1", [key("e"), key("ArrowDown"), key("ArrowRight")]],
  ["ctx-pad", "", [nav("library"), `new Promise((r) => { const c = document.querySelector('#grid [data-key], #grid [data-game]'); const b = c.getBoundingClientRect(); c.dispatchEvent(new MouseEvent("contextmenu", { bubbles: true, clientX: b.x + 40, clientY: b.y + 40 })); setTimeout(r, 300); })`]],
  ["tv-fresh", "tv=1&players=1&fresh=1", []],
  ["tv-fresh-400", "tv=1&players=1&fresh=1&games=400", [key("End")]],
  ["tv-library-kbm", "tv=1&players=1", [key("e"), key("ArrowRight"), key("ArrowRight"), key("ArrowRight"), key("ArrowRight"), key("ArrowRight"), key("ArrowRight"), key("ArrowRight"), key("ArrowRight"), key("ArrowRight"), key("ArrowRight")]],
  ["tv-library-400", "tv=1&players=1&games=400", [key("e"), key("ArrowDown"), key("ArrowDown"), key("ArrowDown")]],
  ["tv-achievements", "tv=1&players=1", [key("e"), key("e")]],
  ["tv-achievements-in", "tv=1&players=1", [key("e"), key("e"), key("ArrowDown"), key("ArrowDown"), "0", key("Enter"), key("ArrowDown"), key("ArrowDown"), key("ArrowDown")]],
  ["tv-achievements-loading", "tv=1&players=1&state=loading", [key("e"), key("e")]],
  ["tv-leave", "tv=1&players=1", [key("Escape"), key("Escape")]],
  ["tv-menu", "tv=1&players=1", [key("ArrowUp"), key("ArrowLeft"), key("ArrowRight"), key("ArrowRight"), key("ArrowRight"), key("ArrowRight"), key("Enter")]],
  ["tv-settings", "tv=1&players=1", [key("ArrowUp"), key("ArrowRight"), key("ArrowRight"), key("ArrowRight"), key("ArrowRight"), key("Enter"), key("ArrowDown"), key("Enter"), key("ArrowDown"), key("ArrowRight")]],
  ["tv-empty", "tv=1&players=1&games=0", []],
  ["tv-scanning", "tv=1&players=1&state=loading", []],
  ["players-edit", "", [nav("players"), click('[data-action="edit-profiles"]')]],
  ["saves", "", [nav("saves")]],
  ["saves-calm", "saves=calm", [nav("saves")]],
  ["saves-empty", "games=0", [nav("saves")]],
  ["cloud-setup", "cloud=0", [nav("saves")]],
  ["cloud-error", "state=error", [nav("saves")]],
  ["settings", "", [nav("settings")]],
  ["settings-online", "", [nav("settings"), click('[data-action="set-tab"][data-tab="online"]')]],
  ["settings-games", "", [nav("settings"), click('[data-action="set-tab"][data-tab="games"]')]],
  ["settings-plugins", "plugins=1", [nav("settings"), click('[data-action="set-tab"][data-tab="plugins"]')]],
  ["settings-tv-android", "link=set&lg=1&atv=1", [nav("settings"), click('[data-action="set-tab"][data-tab="tv"]')]],
  ["atv-add", "link=set", [nav("settings"), click('[data-action="set-tab"][data-tab="tv"]'), click('[data-action="atv-add"]')]],
  ["atv-allow", "link=set&atvrun=allow", [nav("settings"), click('[data-action="set-tab"][data-tab="tv"]'), click('[data-action="atv-add"]'), click('input[name="atv-pick"]'), click('[data-action="atv-install"]')]],
  ["atv-busy", "link=set", [nav("settings"), click('[data-action="set-tab"][data-tab="tv"]'), click('[data-action="atv-add"]'), click('input[name="atv-pick"]'), click('[data-action="atv-install"]')]],
  ["settings-plugins-allow", "plugins=1", [nav("settings"), click('[data-action="set-tab"][data-tab="plugins"]'), click('[data-plugin-on="update-notices"]')]],
  ["sources", "sources=1", [nav("settings"), click('[data-action="set-tab"][data-tab="pp:update-notices:sources"]')]],
  ["update-notices", "sources=1", [nav("updates")]],
  ["sources-game", "sources=1", [nav("updates"), ...game('data-key="dir:d:\\\\games\\\\onimusha"')]],
  ...["download", "paused", "ready", "hand", "apply", "kept", "failed"].map((s) => ["srcupd-" + s, "sources=1&srcupd=" + s, [nav("updates")]]),
  ["srcupd-review", "sources=1&srcupd=ready", [nav("updates"), click('[data-action="pp-job"][data-op="plan"]')]],
  ...["compare", "fetch", "fetchpaused", "deltaready", "deltakept"].map((s) => ["srcupd-" + s, "sources=1&srcupd=" + s, [nav("updates")]]),
  ["srcupd-deltareview", "sources=1&srcupd=deltaready", [nav("updates"), click('[data-action="pp-job"][data-op="plan"]')]],
  ["srcupd-log", "sources=1&srcupd=kept", [nav("updates"), click('[data-action="pp-job"][data-op="log"]')]],
  ["srcupd-game", "sources=1&srcupd=download", [nav("updates"), ...game('data-key="dir:d:\\\\games\\\\onimusha"')]],
  ["sources-error", "sources=error", [nav("settings"), click('[data-action="set-tab"][data-tab="pp:update-notices:sources"]')]],
  ["sources-empty", "sources=empty", [nav("settings"), click('[data-action="set-tab"][data-tab="pp:update-notices:sources"]')]],
  ["sources-notice", "sources=empty", [nav("settings"), click('[data-action="set-tab"][data-tab="pp:update-notices:sources"]'), click('[data-action="pp-act"][data-act="notice"]')]],
  ["sources-add", "sources=1", [nav("settings"), click('[data-action="set-tab"][data-tab="pp:update-notices:sources"]'), click('[data-action="pp-act"][data-act="add"]')]],
  ["add-game", "", [nav("library"), click('[data-action="add-game"]')]],
  ["me-menu", "", [click("#bar-me")]],
  ["me-menu-runs", "runs=1&players=6", [click("#bar-me")]],
  ["me-menu-solo", "players=1", [click("#bar-me")]],
  ["side-folded", "", ['document.dispatchEvent(new KeyboardEvent("keydown", { key: "b", ctrlKey: true }))']],
];
const sizes = [[1024, 700, 1], [1280, 800, 1], [1440, 900, 1], [1920, 1080, 1]];
const extra = [["library", 900, 700, 1], ["game-on", 1280, 720, 1], ["game-on", 2560, 1440, 1], ["game-on", 1280, 720, 1.5], ["library", 2560, 1440, 1], ["home", 1280, 720, 1.5]];

const b = await chromium.launch();
const errors = [];
async function shoot(name, query, steps, w, h, scale, file) {
  const ctx = await b.newContext({ viewport: { width: w, height: h }, deviceScaleFactor: scale });
  const p = await ctx.newPage();
  p.on("pageerror", (e) => errors.push(`${file}: ${e.message}`));
  await p.goto(page + (query ? "?" + query : ""));
  await p.waitForTimeout(900);
  for (const s of steps) { await p.evaluate(s); await p.waitForTimeout(700); }
  await p.waitForTimeout(400);
  await p.screenshot({ path: path.join(out, file.replace(/\.png$/, ".jpg")), type: "jpeg", quality: 85 });
  await ctx.close();
}
for (const [name, query, steps] of shots) {
  if (only && !name.includes(only)) continue;
  for (const [w, h, s] of sizes) {
    try { await shoot(name, query, steps, w, h, s, `${name}-${w}x${h}.png`); } catch (e) { errors.push(`${name}: ${e.message}`); }
  }
}
for (const [name, w, h, s] of extra) {
  if (only && !name.includes(only)) continue;
  const def = shots.find((x) => x[0] === name);
  try { await shoot(name, def[1], def[2], w, h, s, `${name}-${w}x${h}${s !== 1 ? "@" + s + "x" : ""}.png`); } catch (e) { errors.push(`${name}: ${e.message}`); }
}
await b.close();
console.log(fs.readdirSync(out).length, "screenshots in", out);
if (errors.length) console.log("problems:\n" + errors.join("\n"));
