// Renders the trailer: every frame of trailer.html, screenshotted and
// handed to ffmpeg, then the sound from music.mjs laid under it.
//
//   node design/harness/fetch-art.mjs && node design/harness/build.mjs
//   node design/trailer/record.mjs          the real UI, frame by frame (out/footage)
//   node design/trailer/music.mjs           the sound (out/sound.wav)
//   node design/trailer/render.mjs          out/lootnest-trailer.mp4
//
//   node design/trailer/render.mjs still 12.5 31 44   single frames into out/stills
//   node design/trailer/render.mjs sheet 6 17 0.5     a contact sheet from 6 s to 17 s, every 0.5 s
//   SRC=preview ...                                   with record.mjs's PREVIEW frames
import { createRequire } from "node:module";
const { chromium } = createRequire(import.meta.url)("playwright");
import fs from "node:fs";
import path from "node:path";
import http from "node:http";
import { spawn, execFileSync } from "node:child_process";
const here = path.dirname(new URL(import.meta.url).pathname), root = path.join(here, "..", "..");
const out = path.join(here, "out");
fs.mkdirSync(out, { recursive: true });

// The icons the trailer draws: Phosphor's, read from the app's own set.
const src = fs.readFileSync(path.join(root, "internal", "app", "ui", "js", "01-icons.js"), "utf8");
const a = src.indexOf("const PH = {"), b = src.indexOf("\n  };", a);
fs.writeFileSync(path.join(out, "icons.js"), "window.PH = " + src.slice(a + "const PH = ".length, b + 4) + "\n");

// Fonts don't load from file://, so the repository is served to the page.
const TYPES = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".jpg": "image/jpeg", ".png": "image/png", ".webp": "image/webp", ".woff2": "font/woff2" };
const server = http.createServer((req, res) => {
  const p = path.join(root, decodeURIComponent(req.url.split("?")[0]));
  if (!p.startsWith(root) || !fs.existsSync(p) || fs.statSync(p).isDirectory()) { res.writeHead(404); res.end(); return; }
  res.writeHead(200, { "content-type": TYPES[path.extname(p)] || "application/octet-stream", "cache-control": "max-age=3600" });
  fs.createReadStream(p).pipe(res);
});
await new Promise((r) => server.listen(0, "127.0.0.1", r));
const url = `http://127.0.0.1:${server.address().port}/design/trailer/trailer.html${process.env.SRC === "preview" ? "?src=preview" : ""}`;

// A browser for each page: one browser draws all its pages on one core.
const browsers = [];
async function open() {
  const browser = await chromium.launch(); browsers.push(browser);
  const p = await (await browser.newContext({ viewport: { width: 1920, height: 1080 }, deviceScaleFactor: 1 })).newPage();
  p.on("pageerror", (e) => console.log("page:", e.message));
  await p.goto(url);
  await p.evaluate("window.ready");
  return p;
}
const [mode, ...args] = process.argv.slice(2);

if (mode === "still" || mode === "sheet") {
  const p = await open(), dir = path.join(out, "stills");
  fs.mkdirSync(dir, { recursive: true });
  let times = args.map(Number);
  if (mode === "sheet") { const [from, to, step] = times; times = []; for (let t = from; t <= to + 1e-6; t += step || 0.5) times.push(+t.toFixed(3)); }
  const files = [];
  for (const t of times) {
    await p.evaluate((t) => window.seek(t), t);
    const f = path.join(dir, `t${t.toFixed(2).padStart(6, "0")}.jpg`);
    await p.screenshot({ path: f, type: "jpeg", quality: 88 });
    files.push(f);
  }
  if (mode === "sheet") {
    const cols = Math.min(4, files.length), rows = Math.ceil(files.length / cols), sheet = path.join(out, `sheet-${args[0]}-${args[1]}.jpg`);
    const list = path.join(out, "sheet.txt");
    fs.writeFileSync(list, files.map((f) => `file '${f}'\nduration 1\n`).join(""));
    execFileSync("ffmpeg", ["-y", "-loglevel", "error", "-f", "concat", "-safe", "0", "-i", list, "-vf", `scale=640:-1,tile=${cols}x${rows}`, "-frames:v", "1", sheet]);
    console.log(sheet);
  } else console.log(files.join("\n"));
} else {
  // The whole film. Four pages take turns at the frames; each frame goes
  // to ffmpeg in order.
  const LEN = +(process.env.LEN || 0) || 70, FROM = +(process.env.FROM || 0), FPS = 60;
  const first = Math.round(FROM * FPS), last = Math.round(LEN * FPS), N = 4;
  const silent = path.join(out, "silent.mp4"), final = path.join(out, process.env.NAME || "lootnest-trailer.mp4");
  const ff = spawn("ffmpeg", ["-y", "-loglevel", "error", "-f", "image2pipe", "-framerate", String(FPS), "-c:v", "png", "-i", "-",
    "-c:v", "libx264", "-preset", "slow", "-crf", "17", "-pix_fmt", "yuv420p", "-color_primaries", "bt709", "-color_trc", "bt709", "-colorspace", "bt709", "-movflags", "+faststart", silent], { stdio: ["pipe", "inherit", "inherit"] });
  const done = new Map(); let next = first, want = first, writing = false;
  const pages = await Promise.all(Array.from({ length: N }, open));
  async function write() { // frames go to the encoder in order, from one place
    if (writing) return; writing = true;
    while (done.has(want)) { const png = done.get(want); done.delete(want); want++; if (!ff.stdin.write(png)) await new Promise((r) => ff.stdin.once("drain", r)); }
    writing = false;
  }
  const t0 = Date.now();
  await Promise.all(pages.map(async (p) => {
    for (;;) {
      const f = next++; if (f >= last) return;
      while (f - want > 40) await new Promise((r) => setTimeout(r, 20)); // not too far ahead of the encoder
      await p.evaluate((t) => window.seek(t), f / FPS);
      done.set(f, await p.screenshot({ type: "png" }));
      await write();
      if (f % 300 === 0) console.log(`${(f / FPS).toFixed(0)} s of ${LEN}, ${((Date.now() - t0) / 1000).toFixed(0)} s in`);
    }
  }));
  while (writing) await new Promise((r) => setTimeout(r, 20));
  await write();
  ff.stdin.end();
  await new Promise((r) => ff.on("close", r));
  const sound = path.join(out, "sound.wav");
  if (fs.existsSync(sound) && !FROM) execFileSync("ffmpeg", ["-y", "-loglevel", "error", "-i", silent, "-i", sound, "-c:v", "copy", "-c:a", "aac", "-b:a", "256k", "-shortest", "-movflags", "+faststart", final]);
  else fs.copyFileSync(silent, final);
  console.log(final);
}
await Promise.all(browsers.map((b) => b.close()));
server.close();
