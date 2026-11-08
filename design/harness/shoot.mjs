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
  ["game-tab-ach", "", [...game('data-game="bg3"'), click('.gp-tabs [data-tab="achievements"]')]],
  ["achievements", "", [nav("achievements")]],
  ["updates", "", [nav("updates")]],
  ["store", "", [nav("find")]],
  ["bp", "", [click('[data-action="bp-open"]')]],
  ["bp-400", "games=400", [click('[data-action="bp-open"]')]],
  ["players-edit", "", [nav("players"), click('[data-action="edit-profiles"]')]],
  ["cloud", "", [nav("cloud")]],
  ["cloud-setup", "cloud=0", [nav("cloud")]],
  ["cloud-error", "state=error", [nav("cloud")]],
  ["settings", "", [nav("settings")]],
  ["settings-online", "", [nav("settings"), click('[data-action="set-tab"][data-tab="online"]')]],
  ["settings-games", "", [nav("settings"), click('[data-action="set-tab"][data-tab="games"]')]],
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
