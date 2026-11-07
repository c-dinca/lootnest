// Screenshots the kitchen sink into design/system/: node design/harness/shoot-ks.mjs
import { createRequire } from "node:module";
const { chromium } = createRequire(import.meta.url)("playwright");
import fs from "node:fs";
import path from "node:path";
const here = path.dirname(new URL(import.meta.url).pathname);
const out = path.join(here, "..", "system");
fs.mkdirSync(out, { recursive: true });
const b = await chromium.launch();
for (const [w, h, s, motion] of [[1440, 900, 1, "no-preference"], [1280, 720, 1.5, "no-preference"]]) {
  const ctx = await b.newContext({ viewport: { width: w, height: h }, deviceScaleFactor: s, reducedMotion: "reduce" });
  const p = await ctx.newPage();
  p.on("pageerror", (e) => console.log("error:", e.message));
  await p.goto("file://" + path.join(here, "dist", "kitchen-sink.html"));
  await p.waitForTimeout(800);
  const file = `kitchen-sink-${w}x${h}${s !== 1 ? "@" + s + "x" : ""}.jpg`;
  await p.screenshot({ path: path.join(out, file), type: "jpeg", quality: 85, fullPage: true });
  console.log(file);
  await ctx.close();
}
await b.close();
