// Checks the window's script for syntax errors: internal/app/ui/js/*.js
// joined in name order, as app.go joins them (the parts aren't valid alone).
import fs from "node:fs";
import path from "node:path";
import vm from "node:vm";
const dir = path.join(path.dirname(new URL(import.meta.url).pathname), "../internal/app/ui/js");
const files = fs.readdirSync(dir).filter((f) => f.endsWith(".js")).sort();
const js = files.map((f) => fs.readFileSync(path.join(dir, f), "utf8")).join("");
try {
  new vm.Script(js, { filename: "ui/js (joined)" });
} catch (e) {
  let line = +(e.stack.match(/^ui\/js \(joined\):(\d+)/) || [])[1];
  for (const f of files) {
    const n = fs.readFileSync(path.join(dir, f), "utf8").split("\n").length - 1;
    if (line <= n) { console.error(`${f}:${line}`); break; }
    line -= n;
  }
  console.error(e.stack.split("\n").slice(0, 5).join("\n"));
  process.exit(1);
}
console.log(`ok: ${files.length} files, ${js.split("\n").length - 1} lines`);
