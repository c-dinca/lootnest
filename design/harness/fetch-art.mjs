// Downloads the Steam art the fixtures use into .cache/ (not committed).
// Run once: node design/harness/fetch-art.mjs
import fs from "node:fs";
import path from "node:path";
const dir = path.join(path.dirname(new URL(import.meta.url).pathname), ".cache");
fs.mkdirSync(dir, { recursive: true });
const ids = [730, 1091500, 1426210, 413150, 367520, 1086940, 1145350, 1551360, 271590, 250900, 242760];
const kinds = {
  cover: (id) => `https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/${id}/library_600x900_2x.jpg`,
  hero: (id) => `https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/${id}/library_hero.jpg`,
  logo: (id) => `https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/${id}/logo.png`,
};
for (const id of ids) {
  for (const [k, url] of Object.entries(kinds)) {
    const out = path.join(dir, `${id}-${k}.${k === "logo" ? "png" : "jpg"}`);
    if (fs.existsSync(out)) continue;
    const r = await fetch(url(id));
    if (!r.ok) { console.log("missing", id, k, r.status); continue; }
    fs.writeFileSync(out, Buffer.from(await r.arrayBuffer()));
    console.log("ok", id, k);
  }
}
