// Builds dist/page.html: the real UI (internal/app/ui) put together the way
// app.go's pageHTML does, with stub.js and the art cache in front of app.js.
import fs from "node:fs";
import path from "node:path";
const here = path.dirname(new URL(import.meta.url).pathname);
const ui = path.join(here, "../../internal/app/ui");
// Like app.go: system.css, then app.css, with the fonts inlined.
const fonts = ["bricolage-latin", "bricolage-latin-ext", "figtree-latin", "figtree-latin-ext"];
const withFonts = (css) => fonts.reduce((c, n) => c.replace("@FONT:" + n + "@", fs.readFileSync(path.join(ui, "fonts", n + ".woff2")).toString("base64")), css);
const system = fs.readFileSync(path.join(ui, "system.css"), "utf8");
const css = withFonts(system + "\n" + fs.readFileSync(path.join(ui, "app.css"), "utf8"));
const art = {};
const cache = path.join(here, ".cache");
if (fs.existsSync(cache)) {
  for (const f of fs.readdirSync(cache)) {
    const m = f.match(/^(\d+)-(cover|hero|logo)\.(jpg|png)$/);
    if (m) art[`${m[1]}-${m[2]}`] = `data:image/${m[3] === "png" ? "png" : "jpeg"};base64,` + fs.readFileSync(path.join(cache, f)).toString("base64");
  }
}
// Like app.go: the sounds inlined into app.js as {name: base64}.
const sounds = Object.fromEntries(fs.readdirSync(path.join(ui, "sounds")).filter((f) => f.endsWith(".ogg"))
  .map((f) => [f.slice(0, -4), fs.readFileSync(path.join(ui, "sounds", f)).toString("base64")]));
const js = fs.readFileSync(path.join(ui, "app.js"), "utf8").replace("/*@SOUNDS*/{}", () => JSON.stringify(sounds));
const stub = fs.readFileSync(path.join(here, "stub.js"), "utf8");
const html = fs.readFileSync(path.join(ui, "index.html"), "utf8")
  .replace("/*@CSS*/", () => css)
  .replace('<script nonce="@NONCE@">/*@JS*/</script>', () =>
    `<script nonce="@NONCE@">window.__ART=${JSON.stringify(art)};</script><script nonce="@NONCE@">${stub}</script><script nonce="@NONCE@">${js}</script>`)
  .replaceAll("@NONCE@", "harness");
fs.mkdirSync(path.join(here, "dist"), { recursive: true });
fs.writeFileSync(path.join(here, "dist", "page.html"), html);
// The kitchen sink: every system component in every state (dev only).
const ks = `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Kitchen sink · Lootnest</title><style>${withFonts(system)}\n${fs.readFileSync(path.join(here, "kitchen-sink.css"), "utf8")}</style></head>
<body>${fs.readFileSync(path.join(here, "icons.svg"), "utf8")}<script>window.__ART=${JSON.stringify(art)};</script>
<script>${fs.readFileSync(path.join(here, "kitchen-sink.js"), "utf8")}</script></body></html>`;
fs.writeFileSync(path.join(here, "dist", "kitchen-sink.html"), ks);
console.log("dist/page.html", Math.round(html.length / 1024), "KB,", Object.keys(art).length, "art files");
