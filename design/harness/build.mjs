// Builds dist/page.html: the real UI (internal/app/ui) put together the way
// app.go's pageHTML does, with stub.js and the art cache in front of app.js.
import fs from "node:fs";
import path from "node:path";
const here = path.dirname(new URL(import.meta.url).pathname);
const ui = path.join(here, "../../internal/app/ui");
let css = fs.readFileSync(path.join(ui, "app.css"), "utf8");
for (const n of ["bricolage-latin", "bricolage-latin-ext"]) {
  css = css.replace("@FONT:" + n + "@", fs.readFileSync(path.join(ui, "fonts", n + ".woff2")).toString("base64"));
}
const art = {};
const cache = path.join(here, ".cache");
if (fs.existsSync(cache)) {
  for (const f of fs.readdirSync(cache)) {
    const m = f.match(/^(\d+)-(cover|hero|logo)\.(jpg|png)$/);
    if (m) art[`${m[1]}-${m[2]}`] = `data:image/${m[3] === "png" ? "png" : "jpeg"};base64,` + fs.readFileSync(path.join(cache, f)).toString("base64");
  }
}
const js = fs.readFileSync(path.join(ui, "app.js"), "utf8");
const stub = fs.readFileSync(path.join(here, "stub.js"), "utf8");
const html = fs.readFileSync(path.join(ui, "index.html"), "utf8")
  .replace("/*@CSS*/", () => css)
  .replace('<script nonce="@NONCE@">/*@JS*/</script>', () =>
    `<script nonce="@NONCE@">window.__ART=${JSON.stringify(art)};</script><script nonce="@NONCE@">${stub}</script><script nonce="@NONCE@">${js}</script>`)
  .replaceAll("@NONCE@", "harness");
fs.mkdirSync(path.join(here, "dist"), { recursive: true });
fs.writeFileSync(path.join(here, "dist", "page.html"), html);
console.log("dist/page.html", Math.round(html.length / 1024), "KB,", Object.keys(art).length, "art files");
