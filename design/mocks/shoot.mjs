// Screenshots the direction mocks: node design/mocks/shoot.mjs [name]
// Needs the art cache: node design/harness/fetch-art.mjs
import { createRequire } from "node:module";
const { chromium } = createRequire(import.meta.url)("playwright");
import path from "node:path";
const here = path.dirname(new URL(import.meta.url).pathname);
const names = process.argv[2] ? [process.argv[2]] : ["high-score", "save-slot", "couch"];
const sizes = [[1440, 900, 1], [1280, 720, 1.5], [1920, 1080, 1]];
const b = await chromium.launch();
for (const n of names) {
  for (const [w, h, s] of sizes) {
    const ctx = await b.newContext({ viewport: { width: w, height: h }, deviceScaleFactor: s });
    const p = await ctx.newPage();
    p.on("pageerror", (e) => console.log(n, e.message));
    await p.goto("file://" + path.join(here, n + ".html"));
    await p.waitForLoadState("networkidle");
    await p.waitForTimeout(600);
    const file = `${n}-${w}x${h}${s !== 1 ? "@" + s + "x" : ""}.jpg`;
    await p.screenshot({ path: path.join(here, file), type: "jpeg", quality: 85 });
    await ctx.close();
    console.log(file);
  }
}
await b.close();
