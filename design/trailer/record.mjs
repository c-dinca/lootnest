// Records the real UI (the design harness's page) frame by frame, so the
// trailer shows the launcher's own motion at a steady 60 frames a second.
// Time is ours: Playwright's clock fires timers and animation frames, and
// every CSS transition and animation is paused and stepped by hand. The
// pointer is a drawn one that follows the cues, with the real mouse under
// it so hover and clicks are the page's own.
//
//   node design/harness/build.mjs
//   node design/trailer/record.mjs [filter]        every take, or those whose name has the filter
//   PREVIEW=1 node design/trailer/record.mjs ...   every 5th frame at 1x, to check the cues
import { createRequire } from "node:module";
const { chromium } = createRequire(import.meta.url)("playwright");
import fs from "node:fs";
import path from "node:path";
const here = path.dirname(new URL(import.meta.url).pathname);
const PREVIEW = !!process.env.PREVIEW;
const out = path.join(here, "out", PREVIEW ? "preview" : "footage");
const page = "file://" + path.join(here, "..", "harness", "dist", "page.html");
const only = process.argv[2] || "";
const FPS = 60, DT = 1000 / FPS;

const q = (sel) => `document.querySelector(${JSON.stringify(sel)})`;
const has = (sel, text) => `[...document.querySelectorAll(${JSON.stringify(sel)})].find((e) => e.textContent.includes(${JSON.stringify(text)}))`;
const key = (k) => `document.dispatchEvent(new KeyboardEvent("keydown", { key: ${JSON.stringify(k)}, bubbles: true }))`;
const press = (sel) => `${q(sel)}.click()`;
const ev = (o) => `window.__ps(${JSON.stringify(o)})`;
const A = "anon=1&runs=1";
const WIN = { w: 1440, h: 810, scale: 2 }, TV = { w: 1920, h: 1080, scale: 1 };
const bg3 = [press('[data-view="library"]'), press('[data-action="open-sheet"][data-game="bg3"]')];

// A take: the page's query, its size, what to do before the first frame
// (each step waits), then cues [ms, what, ...] and the length in ms.
//   move  target (an expression giving an element, or [x, y]), ms, [fx, fy] inside the element
//   click                  at the pointer
//   key   name             a keydown on the document
//   js    code
//   type  text, ms a letter   into the field in focus
export const takes = [
  { name: "win-lib", q: A, ...WIN, at: [820, 560], len: 11600,
    pre: ['document.querySelectorAll("[data-action=home-pick]")[3].click()'],
    cues: [
      [500, "move", q('.side [data-view="library"]'), 650], [1250, "click"],
      [2000, "move", has(".view .lib-cap-open", "It Takes Two"), 700, [0.5, 0.4]],
      [3100, "move", has(".view .lib-cap-open", "Cyberpunk 2077"), 650, [0.4, 0.45]],
      [4200, "move", has(".view .ln-tile", "Baldur's Gate 3"), 750, [0.5, 0.4]],
      [5250, "click"],
      [6900, "move", '[...document.querySelectorAll(\'.view [data-action="play-pick"]\')].pop()', 700], [7800, "click"],
      [8800, "move", has(".slot", "hardcore"), 650, [0.5, 0.55]], [9700, "click"],
    ] },
  { name: "win-saves", q: A, ...WIN, at: [560, 470], len: 6400,
    pre: bg3,
    cues: [
      [300, "move", q('.gp-tabs [data-tab="saves"]'), 500], [900, "click"],
      [1700, "move", q(".gpage-body input"), 450, [0.3, 0.5]], [2250, "click"],
      [2400, "type", "Before the boss", 52],
      [3350, "move", has(".gpage-body button", "Make a save point"), 420], [3900, "click"],
      [5000, "move", has(".gpage-body button", "Go back to it"), 600],
    ] },
  { name: "win-ach", q: A, ...WIN, at: [900, 420], len: 9000,
    pre: bg3,
    cues: [
      [300, "move", q('.gp-tabs [data-tab="achievements"]'), 500], [900, "click"],
      [2300, "js", ev({ type: "achUnlocked", key: "game:bg3", game: "Baldur's Gate 3", player: "p2", native: true, list: [{ name: "Against all odds", percent: 0.9, description: "" }] })],
      [3300, "move", has(".view [data-action=ach-who]", "hardcore"), 550], [3950, "click"],
      [4900, "move", has(".view [data-action=ach-card]", "Against all odds"), 600, [0.3, 0.5]], [5600, "click"],
      [6500, "move", q(".achc-card"), 500, [0.62, 0.6]], [7300, "click"],
    ] },
  { name: "tv-main", q: A + "&tv=1", ...TV, len: 11600, pre: [],
    cues: [
      [700, "key", "ArrowRight"], [1300, "key", "ArrowLeft"], [2000, "key", "Enter"],
      [2900, "key", "ArrowRight"], [3400, "key", "ArrowRight"], [3900, "key", "ArrowRight"], [5000, "key", "Enter"],
      [6500, "key", "ArrowDown"], [6950, "key", "ArrowRight"], [7400, "key", "ArrowRight"], [8200, "key", "ArrowUp"],
      [8900, "key", "Enter"],
    ] },
  { name: "tv-menu", q: A + "&tv=1&players=1", ...TV, len: 5200,
    pre: [key("ArrowRight"), key("ArrowRight"), key("ArrowRight"), ev({ type: "tvGame", key: "game:bg3", phase: "playing" })],
    cues: [
      [200, "js", 'window.__ps({ type: "tvMenu", on: true, key: "game:bg3", shot: (document.querySelector("#bp .bp-bd-l[src]") || {}).src || "" })'],
      [1500, "key", "ArrowDown"], [2300, "key", "Enter"], [3700, "key", "ArrowDown"],
    ] },
  { name: "tv-apps", q: A + "&tv=1&players=1", ...TV, len: 3600, pre: [key("ArrowRight")],
    cues: [[400, "key", "ArrowDown"], [1200, "key", "ArrowRight"], [1800, "key", "ArrowRight"]] },
];

// Steps every animation of the page by dt. One seen for the first time is paused where it is.
const STEP = `(() => { const seen = new WeakMap(); window.__step = (dt) => { for (const a of document.getAnimations()) { let t = seen.get(a); if (t === undefined) { t = +a.currentTime || 0; try { a.pause(); } catch (e) {} } else t += dt; seen.set(a, t); let end = Infinity; try { end = a.effect.getComputedTiming().endTime; } catch (e) {} if (t >= end) { try { a.finish(); } catch (e) {} } else { try { a.currentTime = t; } catch (e) {} } } }; })()`;
// The drawn pointer and the ring a click leaves.
const POINTER = `(() => { const c = document.createElement("div"); c.id = "tr-ptr"; c.style.cssText = "position:fixed;left:0;top:0;z-index:2147483647;pointer-events:none;width:26px;height:26px;transform-origin:3px 2px;will-change:transform";
  c.innerHTML = '<svg viewBox="0 0 26 26" width="26" height="26"><path d="M3.5 2.2v18.4l4.9-4.6 3.3 7.4 3-1.3-3.3-7.3 6.7-.3z" fill="#fff" stroke="#111217" stroke-width="1.5" stroke-linejoin="round"/></svg>';
  const st = document.createElement("style"); st.textContent = "@keyframes tr-ring{from{transform:translate(-50%,-50%) scale(.3);opacity:.7}to{transform:translate(-50%,-50%) scale(1);opacity:0}}.tr-ring{position:fixed;z-index:2147483646;pointer-events:none;width:64px;height:64px;border-radius:50%;border:2px solid rgba(223,229,235,.9);animation:tr-ring 480ms cubic-bezier(.2,.7,.2,1) forwards}";
  document.head.appendChild(st); document.body.appendChild(c);
  window.__ptr = (x, y, down) => { c.style.transform = "translate(" + (x - 3) + "px," + (y - 2) + "px) scale(" + (down ? 0.86 : 1) + ")"; };
  window.__ring = (x, y) => { const r = document.createElement("div"); r.className = "tr-ring"; r.style.left = x + "px"; r.style.top = y + "px"; document.body.appendChild(r); r.addEventListener("animationend", () => r.remove()); }; })()`;
const ease = (p) => (p < 0.5 ? 4 * p * p * p : 1 - Math.pow(-2 * p + 2, 3) / 2);

async function record(b, tk) {
  const dir = path.join(out, tk.name);
  fs.rmSync(dir, { recursive: true, force: true });
  fs.mkdirSync(dir, { recursive: true });
  const ctx = await b.newContext({ viewport: { width: tk.w, height: tk.h }, deviceScaleFactor: PREVIEW ? 1 : tk.scale });
  const p = await ctx.newPage();
  p.on("pageerror", (e) => console.log(tk.name, e.message));
  await p.clock.install({ time: new Date("2026-11-14T20:20:00") });
  await p.goto(page + "?" + tk.q);
  await p.waitForTimeout(1200);
  for (const s of tk.pre || []) { await p.evaluate(s); await p.waitForTimeout(900); }
  await p.clock.runFor(15000); // what the page still had to hear (news, updates) arrives before the take, not during it
  await p.waitForTimeout(600);
  await p.addStyleTag({ content: ".gp-stage .ln-spinner, .gp-stage [class*=spin] { display: none !important; }" }); // a trailer that never loads here
  await p.evaluate(STEP);
  const ptr = !!tk.at;
  let pos = tk.at || [0, 0], mv = null, down = false, upAt = -1, typing = null;
  if (ptr) { await p.evaluate(POINTER); await p.mouse.move(pos[0], pos[1]); await p.evaluate(`window.__ptr(${pos[0]}, ${pos[1]}, false)`); await p.waitForTimeout(400); }
  await p.clock.pauseAt(new Date(await p.evaluate("Date.now()") + 1000));
  await p.evaluate("window.__step(0)");
  const cues = [...tk.cues].sort((a, b) => a[0] - b[0]);
  const n = Math.round(tk.len / DT);
  for (let i = 0; i < n; i++) {
    const t = i * DT;
    while (cues.length && cues[0][0] <= t) {
      const [at, what, a, b2, c] = cues.shift();
      if (what === "key") await p.evaluate(key(a));
      else if (what === "js") await p.evaluate(a);
      else if (what === "click") { down = true; upAt = t + 110; await p.mouse.down(); await p.evaluate(`window.__ring(${pos[0]}, ${pos[1]})`); }
      else if (what === "type") typing = { text: a, per: b2, t0: t, done: 0 };
      else if (what === "move") {
        let to = a;
        if (typeof a === "string") {
          const r = await p.evaluate(`(() => { const e = ${a}; if (!e) return null; const r = e.getBoundingClientRect(); return [r.left, r.top, r.width, r.height]; })()`);
          if (!r) { console.log(tk.name, "nothing for", a.slice(0, 90)); continue; }
          const f = c || [0.5, 0.5];
          to = [r[0] + r[2] * f[0], r[1] + r[3] * f[1]];
        }
        mv = { from: pos, to, t0: t, dur: b2 };
      }
    }
    if (mv) {
      const k = Math.min(1, (t - mv.t0) / mv.dur), e = ease(k);
      const dx = mv.to[0] - mv.from[0], dy = mv.to[1] - mv.from[1], bow = Math.sin(Math.PI * e) * 0.05; // a slight arc, as a hand makes
      pos = [mv.from[0] + dx * e - dy * bow, mv.from[1] + dy * e + dx * bow];
      await p.mouse.move(pos[0], pos[1]);
      if (k >= 1) mv = null;
    }
    if (down && t >= upAt) { down = false; await p.mouse.up(); }
    if (typing) {
      const want = Math.min(typing.text.length, Math.floor((t - typing.t0) / typing.per) + 1);
      while (typing.done < want) await p.keyboard.type(typing.text[typing.done++]);
      if (typing.done >= typing.text.length) typing = null;
    }
    if (ptr) await p.evaluate(`window.__ptr(${pos[0]}, ${pos[1]}, ${down})`);
    await p.clock.runFor(DT);
    await p.evaluate(`window.__step(${DT})`);
    if (PREVIEW && i % 5) continue;
    await p.screenshot({ path: path.join(dir, `f${String(i).padStart(5, "0")}.jpg`), type: "jpeg", quality: PREVIEW ? 70 : 93 });
  }
  await ctx.close();
  console.log(tk.name, n, "frames");
}

const b = await chromium.launch();
const todo = takes.filter((tk) => !only || only.split(",").some((o) => tk.name.includes(o)));
// Two at a time: each take has its own page and its own clock.
for (let i = 0; i < todo.length; i += 2) await Promise.all(todo.slice(i, i + 2).map((tk) => record(b, tk)));
await b.close();
