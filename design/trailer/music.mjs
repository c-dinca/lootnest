// The trailer's sound, made here so nothing has to be licensed: a small
// synthesized score at 120 beats a minute (a bar is two seconds, which is
// what trailer.js cuts on), and the app's own interface sounds (Kenney's,
// CC0, internal/app/ui/sounds) at the moments the picture shows them.
//   node design/trailer/music.mjs   writes out/sound.wav
//
// The film's music is "Voltaic" (the owner's pick, 16 Nov 2026), a piece
// somebody else made, under the hits and interface sounds made here.
// TRACK=<name> tries another one and writes out/sound-<name>.wav;
// TRACK=score is the score synthesized here. The pieces below are
// Kevin MacLeod's (incompetech.com), licensed CC BY 4.0, so the film must
// credit them wherever it is shown (README). They are at 120 beats a minute
// on the film's own grid; "cut" says which second of the piece plays from
// which second of the film, in whole bars, so its lifts land on the cuts.
// Any other file: TRACK=/path/to/piece.mp3 CUT="0:10,24:14" (film:piece).
import fs from "node:fs";
import path from "node:path";
import { execFileSync, spawnSync } from "node:child_process";
const here = path.dirname(new URL(import.meta.url).pathname), out = path.join(here, "out");
const SOUNDS = path.join(here, "..", "..", "internal", "app", "ui", "sounds");
fs.mkdirSync(out, { recursive: true });
const SR = 48000, LEN = 70, N = SR * LEN, BEAT = 0.5, BAR = 2;
const buf = () => [new Float32Array(N), new Float32Array(N)];
const hz = (m) => 440 * Math.pow(2, (m - 69) / 12);
let seed = 7; const rnd = () => ((seed = (seed * 1664525 + 1013904223) >>> 0) / 4294967296) * 2 - 1;

// The chords, a bar each: Am9, Fmaj7, Cadd9, G6. The film ends on C.
const CH = [
  { bass: 33, notes: [57, 60, 64, 71] },
  { bass: 29, notes: [53, 57, 60, 64] },
  { bass: 36, notes: [52, 55, 60, 62] },
  { bass: 31, notes: [50, 55, 59, 62] },
];
const TRACKS = {
  // the quiet stabs under the hook, the groove from the Library, one bar's breath for the keys, the heavy drums on the TV
  "long-time-coming": { file: "Long Time Coming.mp3", cut: [[0, 10], [24, 14], [26, 26]] },
  // drums that keep growing; a breath before the TV
  stormfront: { file: "Stormfront.mp3", cut: [[0, 10], [32, 38]] },
  // the same lift twice: at the Library and at the TV
  voltaic: { file: "Voltaic.mp3", cut: [[0, 58], [38, 62]] },
};
const FILM = "voltaic", TRACK = process.env.TRACK || FILM;
const T = TRACK === "score" ? null : TRACKS[TRACK] ? { name: TRACK, ...TRACKS[TRACK] } : { name: path.basename(TRACK).replace(/\.[^.]+$/, "").toLowerCase().replace(/[^a-z0-9]+/g, "-"), path: TRACK, cut: (process.env.CUT || "0:0").split(",").map((c) => c.split(":").map(Number)) };
const chordAt = (bar) => (bar >= 33 ? CH[2] : bar === 32 ? CH[1] : CH[(((bar - 3) % 4) + 4) % 4]);
// What plays when (seconds): the film's parts.
const drums = (t) => (t >= 6 && t < 24) || (t >= 26 && t < 62);
const full = (t) => (t >= 26 && t < 58);
const big = (t) => t >= 40 && t < 58;

// ------------------------------------------------------------------ pad --
const pad = buf();
if (!T) for (let bar = 0; bar < LEN / BAR; bar++) {
  const c = chordAt(bar), t0 = bar * BAR, a = Math.max(0, Math.floor((t0 - 0.25) * SR)), b = Math.min(N, Math.floor((t0 + BAR + 0.35) * SR));
  for (const m of [c.bass + 12, ...c.notes]) {
    for (const [det, pan] of [[-0.07, 0.25], [0.07, 0.75], [0.02, 0.5]]) {
      const f = hz(m + det), inc = f / SR; let ph = Math.abs(rnd());
      for (let i = a; i < b; i++) {
        const t = i / SR - t0, env = Math.min(1, (t + 0.25) / 0.45) * Math.min(1, (BAR + 0.35 - t) / 0.55);
        ph += inc; if (ph >= 1) ph -= 1;
        const v = (2 * ph - 1) * env * 0.05;
        pad[0][i] += v * (1 - pan); pad[1][i] += v * pan;
      }
    }
  }
}
// two one-pole lowpasses, opening as the film grows
if (!T) for (const ch of pad) {
  let y1 = 0, y2 = 0;
  for (let i = 0; i < N; i++) {
    const t = i / SR, open = t < 6 ? 0.35 + 0.1 * (t / 6) : big(t) ? 1 : t >= 62 && t < 65 ? 0.4 : t >= 65 ? 0.9 : 0.7;
    const fc = (420 + 900 * open) * (1 + 0.25 * Math.sin(t * 0.9)), k = 1 - Math.exp((-2 * Math.PI * fc) / SR);
    y1 += k * (ch[i] - y1); y2 += k * (y1 - y2); ch[i] = y2;
  }
}

// ---------------------------------------------------------------- plucks --
// A pluck is a few harmonics, the higher ones dying sooner: no filter needed.
const pl = buf();
function pluck(t0, m, gain, pan = 0.5, len = 0.9, bright = 1) {
  const f = hz(m), a = Math.floor(t0 * SR), b = Math.min(N, a + Math.floor(len * SR));
  for (let h = 1; h <= 7; h++) {
    if (f * h > 9000) break;
    const w = (2 * Math.PI * f * h) / SR, amp = (gain / h) * Math.pow(bright, h - 1), dec = (3.2 + h * 2.6) / SR;
    for (let i = a; i < b; i++) { const n = i - a, v = Math.sin(w * n) * amp * Math.exp(-dec * n) * Math.min(1, n / 96); pl[0][i] += v * (1 - pan); pl[1][i] += v * pan; }
  }
}
if (!T) for (let bar = 0; bar < LEN / BAR; bar++) {
  const c = chordAt(bar), t0 = bar * BAR, n = c.notes;
  if (t0 < 6) { pluck(t0, n[2] + 12, 0.16, 0.4, 1.8, 0.7); pluck(t0 + 0.75, n[3] + 12, 0.11, 0.62, 1.6, 0.7); pluck(t0 + 1.5, n[1] + 12, 0.09, 0.5, 1.4, 0.7); continue; }
  if (t0 >= 62 && t0 < 65) continue; // the nevers: quiet
  if (t0 >= 66) { [0, 1, 2, 3].forEach((k) => pluck(t0 + k * 0.06, n[k] + 12, 0.14, 0.3 + k * 0.13, 2.6, 0.8)); continue; }
  // eighths, then sixteenths once the film is going
  const steps = full(t0) ? 16 : 8, order = [0, 2, 1, 3, 2, 3, 1, 2];
  for (let s = 0; s < steps; s++) {
    const t = t0 + (s * BAR) / steps, k = order[s % 8], up = (s % 8 === 3 || s % 8 === 5) && big(t0) ? 24 : 12;
    if (t >= 24 && t < 26 && s % 2) continue;
    pluck(t, n[k] + up, (s % 4 === 0 ? 0.13 : 0.085) * (full(t0) ? 0.9 : 1), 0.5 + 0.3 * Math.sin(s * 1.7), 0.5, big(t0) ? 0.92 : 0.8);
  }
}
// a ping-pong echo on the plucks, a dotted eighth apart
if (!T) {
  const d = Math.floor(0.375 * SR), fb = 0.4, wet = 0.34, eL = new Float32Array(N), eR = new Float32Array(N);
  for (let i = d; i < N; i++) { eL[i] = pl[1][i - d] * 0.9 + eR[i - d] * fb; eR[i] = pl[0][i - d] * 0.5 + eL[i - d] * fb; }
  for (let i = 0; i < N; i++) { pl[0][i] += eL[i] * wet; pl[1][i] += eR[i] * wet; }
}

// ------------------------------------------------------------------ bass --
const bass = buf();
if (!T) for (let bar = 0; bar < LEN / BAR; bar++) {
  const c = chordAt(bar), t0 = bar * BAR, f = hz(c.bass), w = (2 * Math.PI * f) / SR;
  const hits = t0 < 6 || (t0 >= 62 && t0 < 65) || t0 >= 66 ? [[0, BAR]] : Array.from({ length: 8 }, (_, s) => [s * 0.25, 0.25]);
  for (const [at, len] of hits) {
    if (t0 + at >= 24 && t0 + at < 26 && at % 1) continue;
    const a = Math.floor((t0 + at) * SR), b = Math.min(N, a + Math.floor(len * SR)), long = len > 1;
    for (let i = a; i < b; i++) {
      const n = i - a, env = Math.min(1, n / 200) * (long ? Math.min(1, (b - i) / 4000) * 0.34 : Math.exp(-n / (0.16 * SR)));
      const v = (Math.sin(w * i) + 0.35 * Math.sin(2 * w * i) + 0.12 * Math.sin(3 * w * i)) * env * 0.3;
      bass[0][i] += v; bass[1][i] += v;
    }
  }
}

// ----------------------------------------------------------------- drums --
const dr = buf(), hits = buf(), kicks = [];
function kick(t0, g = 1) {
  kicks.push(t0);
  const a = Math.floor(t0 * SR); let ph = 0;
  for (let n = 0; n < 0.4 * SR && a + n < N; n++) {
    const t = n / SR; ph += (2 * Math.PI * (46 + 120 * Math.exp(-t / 0.028))) / SR;
    const v = (Math.sin(ph) * Math.exp(-t / 0.15) + (n < 120 ? rnd() * 0.25 * (1 - n / 120) : 0)) * 0.62 * g;
    dr[0][a + n] += v; dr[1][a + n] += v;
  }
}
function noise(t0, len, gain, hp, pan = 0.5, shape = (k) => Math.exp(-k * 6), to = dr) {
  const a = Math.floor(t0 * SR), n1 = Math.floor(len * SR); let last = 0, lp = 0;
  for (let n = 0; n < n1 && a + n < N; n++) {
    const x = rnd(); lp += hp * (x - lp); const v = (hp >= 1 ? x - last : x - lp) * gain * shape(n / n1); last = x;
    to[0][a + n] += v * (1 - pan) * 2; to[1][a + n] += v * pan * 2;
  }
}
function clap(t0, g) { for (const d of [0, 0.011, 0.023]) noise(t0 + d, 0.11, g, 0.35, 0.5, (k) => Math.exp(-k * 9)); }
if (!T) for (let t = 0; t < LEN; t += BEAT) {
  if (!drums(t)) continue;
  const beat = Math.round(t / BEAT) % 4;
  kick(t, t >= 58 ? 0.9 : 1);
  if (t < 58) noise(t + 0.25, 0.05, 0.09, 1, 0.6); // hat on the off-beat
  if (full(t)) { noise(t + 0.125, 0.03, 0.04, 1, 0.35); noise(t + 0.375, 0.03, 0.045, 1, 0.7); }
  if (t >= 12 && t < 58 && (beat === 1 || beat === 3)) clap(t, 0.11);
  if (big(t) && beat === 3) noise(t + 0.25, 0.22, 0.06, 1, 0.5, (k) => Math.exp(-k * 3)); // an open hat
}
// hits where the film turns, and air rushing into them
function hit(t0, g = 1) {
  const a = Math.floor(t0 * SR); let ph = 0;
  for (let n = 0; n < 1.6 * SR && a + n < N; n++) { const t = n / SR; ph += (2 * Math.PI * (38 + 60 * Math.exp(-t / 0.05))) / SR; const v = Math.sin(ph) * Math.exp(-t / 0.5) * 0.7 * g; hits[0][a + n] += v; hits[1][a + n] += v; }
  noise(t0, 1.2, 0.16 * g, 0.2, 0.5, (k) => Math.exp(-k * 5), hits);
}
function rush(t1, len, g = 1) { // a band of noise that climbs and stops at t1
  const a = Math.floor((t1 - len) * SR), n1 = Math.floor(len * SR); let lo = 0, bp = 0;
  for (let n = 0; n < n1 && a + n < N; n++) {
    if (a + n < 0) continue;
    const k = n / n1, fc = 300 + 5200 * k * k, q = 1 - Math.exp((-2 * Math.PI * fc) / SR), x = rnd();
    lo += q * (x - lo); bp += q * (lo - bp); const v = (lo - bp) * 0.5 * g * k * k, pan = 0.5 + 0.35 * Math.sin(k * 5);
    hits[0][a + n] += v * (1 - pan) * 2; hits[1][a + n] += v * pan * 2;
  }
}
[[6, 1], [26, 0.7], [40, 1], [65, 1]].forEach(([t, g]) => hit(t, g));
[[62, 0.5], [63, 0.45], [64, 0.45], [2.5, 0.35], [24, 0.4]].forEach(([t, g]) => hit(t, g));
[[6, 1.6, 1], [10.9, 0.9, 0.5], [17.6, 0.5, 0.5], [24, 0.7, 0.6], [26, 0.5, 0.5], [32, 0.5, 0.5], [40, 1.2, 1], [49.6, 0.6, 0.4], [51.8, 0.6, 0.4], [58, 0.8, 0.7], [62, 2.2, 0.8], [65, 0.9, 0.9], [2.5, 0.5, 0.4], [4.85, 0.55, 0.5]].forEach(([t, l, g]) => rush(t, l, g));

// ------------------------------------------------- the app's own sounds --
const fx = buf(), cache = {};
function sample(name) {
  if (!cache[name]) { const raw = execFileSync("ffmpeg", ["-loglevel", "error", "-i", path.join(SOUNDS, name + ".ogg"), "-f", "f32le", "-ac", "2", "-ar", String(SR), "-"], { maxBuffer: 1 << 26 }); cache[name] = new Float32Array(raw.buffer, raw.byteOffset, raw.length / 4); }
  return cache[name];
}
function play(t0, name, gain = 0.5, pan = 0.5) {
  const s = sample(name), a = Math.floor(t0 * SR);
  for (let n = 0; n < s.length / 2 && a + n < N; n++) { fx[0][a + n] += s[2 * n] * gain * (1 - pan) * 2; fx[1][a + n] += s[2 * n + 1] * gain * pan * 2; }
}
const S = { move: "tick_001", tab: "select_007", go: "select_002", open: "maximize_008", unlock: "confirmation_002", rare: "glass_001", point: "drop_002", swap: "confirmation_001", key: "click_004" };
const CUES = [
  [2.5, "rare", 0.3], [4.85, "rare", 0.45],
  // the Library and Who's playing
  [7, "tab", 0.5], [11, "open", 0.5], [13.55, "go", 0.5], [15.45, "swap", 0.6],
  // the swap
  [18.5, "point", 0.3, 0.4], [18.58, "point", 0.3, 0.5], [18.66, "point", 0.3, 0.6],
  [19.5, "move", 0.4, 0.4], [19.6, "move", 0.4, 0.35], [19.7, "move", 0.4, 0.3], [20.5, "move", 0.4, 0.7], [20.6, "move", 0.4, 0.6], [20.7, "move", 0.4, 0.5], [21.25, "swap", 0.5],
  [22.2, "move", 0.4, 0.7], [22.29, "move", 0.4, 0.7], [22.38, "move", 0.4, 0.7],
  // Ctrl+Shift+F5 and the Saves tab
  [24.72, "key", 0.6, 0.35], [24.97, "key", 0.6, 0.5], [25.22, "key", 0.7, 0.65], [25.32, "point", 0.6],
  [26.5, "tab", 0.45], [27.85, "key", 0.35], ...Array.from({ length: 15 }, (_, i) => [28 + i * 0.052, "move", 0.14]), [29.5, "go", 0.5], [29.92, "point", 0.6],
  // achievements
  [32.6, "tab", 0.45], [34, "unlock", 0.7, 0.62], [34.05, "rare", 0.4, 0.62], [35.65, "key", 0.35], [37.3, "open", 0.5], [39, "go", 0.45],
  // the TV
  [40.7, "move", 0.5], [41.3, "move", 0.5], [42, "go", 0.5], [42.9, "move", 0.5], [43.4, "move", 0.5], [43.9, "move", 0.5], [45, "open", 0.5],
  [46.5, "move", 0.5], [46.95, "move", 0.5], [47.4, "move", 0.5], [48.2, "move", 0.5], [48.9, "go", 0.5], [49.05, "swap", 0.55],
  [51.5, "open", 0.5], [52.8, "move", 0.5], [53.6, "go", 0.5], [54, "point", 0.55], [55.2, "move", 0.5], [56, "move", 0.5], [56.6, "move", 0.5],
  // the end
  [67, "rare", 0.5], [67.7, "go", 0.4],
];
for (const [t, name, gain, pan] of CUES) play(t, S[name], gain, pan);

// ------------------------------------------------------------------- mix --
const L = new Float32Array(N), R = new Float32Array(N);
if (!T) {
  // The music breathes with the kick, and steps back a little under the UI's sounds.
  let ki = 0, since = 9;
  kicks.sort((a, b) => a - b);
  for (let i = 0; i < N; i++) {
    const t = i / SR;
    while (ki < kicks.length && kicks[ki] <= t) { since = 0; ki++; }
    since += 1 / SR;
    const duck = 1 - 0.55 * Math.exp(-since / 0.11);
    const lvl = t < 6 ? 0.85 : t >= 24 && t < 26 ? 0.7 : t >= 62 && t < 65 ? 0.75 : 1;
    const end = Math.min(1, t / 0.4) * Math.min(1, Math.max(0, (LEN - 0.15 - t) / 2.2));
    for (const [o, c] of [[L, 0], [R, 1]]) {
      const music = (pad[c][i] * 0.9 + pl[c][i] * 1.9 + bass[c][i] * 0.8) * duck * lvl + (dr[c][i] + hits[c][i]) * 0.9;
      o[i] = Math.tanh((music * 0.8 + fx[c][i] * 1.2) * 1.15) * end;
    }
  }
} else {
  // Somebody's piece, cut on its bars, under the hits and the UI's sounds.
  const file = T.path || path.join(out, "music", T.file);
  if (!fs.existsSync(file)) { fs.mkdirSync(path.dirname(file), { recursive: true }); execFileSync("curl", ["-fsSL", "-A", "Mozilla/5.0", "-o", file, "https://incompetech.com/music/royalty-free/mp3-royaltyfree/" + encodeURIComponent(T.file)]); }
  const raw = execFileSync("ffmpeg", ["-loglevel", "error", "-i", file, "-f", "f32le", "-ac", "2", "-ar", String(SR), "-"], { maxBuffer: 1 << 30 });
  const src = new Float32Array(raw.buffer, raw.byteOffset, raw.length / 4), frames = src.length / 2, X = 0.025 * SR;
  const at = (seg, i, c) => { const n = Math.round((seg[1] - seg[0]) * SR) + i; return n >= 0 && n < frames ? src[2 * n + c] : 0; };
  const ext = buf(); let e2 = 0;
  for (let i = 0; i < N; i++) {
    const t = i / SR; let k = 0; while (k + 1 < T.cut.length && T.cut[k + 1][0] <= t) k++;
    const into = i - T.cut[k][0] * SR, x = k && into < X ? into / X : 1; // a short crossfade over each cut
    for (const c of [0, 1]) { const v = at(T.cut[k], i, c) * x + (x < 1 ? at(T.cut[k - 1], i, c) * (1 - x) : 0); ext[c][i] = v; e2 += v * v; }
  }
  const lvl = 0.15 / Math.sqrt(e2 / (2 * N)); let env = 0; const rel = Math.exp(-1 / (0.2 * SR));
  for (let i = 0; i < N; i++) {
    const t = i / SR, end = Math.min(1, t / 0.25) * Math.min(1, Math.max(0, (LEN - 0.1 - t) / 2.6));
    env = Math.max(Math.abs(fx[0][i]), Math.abs(fx[1][i]), env * rel);
    const duck = 1 - 0.4 * Math.min(1, env / 0.18);
    for (const [o, c] of [[L, 0], [R, 1]]) o[i] = Math.tanh((ext[c][i] * lvl * duck + hits[c][i] * 0.8 + fx[c][i] * 1.25) * 1.1) * end;
  }
}
if (process.env.STEMS) for (const [name, b] of [["pad", pad], ["plucks", pl], ["bass", bass], ["drums", dr], ["ui", fx]]) for (const [from, to] of [[1, 5], [8, 22], [42, 56]]) { let e = 0; for (let i = from * SR; i < to * SR; i++) e += b[0][i] * b[0][i]; console.log(name, from + "-" + to, (10 * Math.log10(e / ((to - from) * SR))).toFixed(1), "dB"); }
let peak = 0; for (let i = 0; i < N; i++) peak = Math.max(peak, Math.abs(L[i]), Math.abs(R[i]));
const g = 0.89 / peak, wav = Buffer.alloc(44 + N * 4);
wav.write("RIFF", 0); wav.writeUInt32LE(36 + N * 4, 4); wav.write("WAVEfmt ", 8); wav.writeUInt32LE(16, 16); wav.writeUInt16LE(1, 20); wav.writeUInt16LE(2, 22);
wav.writeUInt32LE(SR, 24); wav.writeUInt32LE(SR * 4, 28); wav.writeUInt16LE(4, 32); wav.writeUInt16LE(16, 34); wav.write("data", 36); wav.writeUInt32LE(N * 4, 40);
for (let i = 0; i < N; i++) { wav.writeInt16LE(Math.round(Math.max(-1, Math.min(1, L[i] * g)) * 32767), 44 + i * 4); wav.writeInt16LE(Math.round(Math.max(-1, Math.min(1, R[i] * g)) * 32767), 46 + i * 4); }
const name = TRACK === FILM ? "sound.wav" : `sound-${T ? T.name : "score"}.wav`;
fs.writeFileSync(path.join(out, name), wav);
// as loud as the web expects, about -14 LUFS, with the peaks held
const said = spawnSync("ffmpeg", ["-hide_banner", "-nostats", "-i", path.join(out, name), "-af", "ebur128", "-f", "null", "-"], { maxBuffer: 1 << 26 }).stderr.toString();
const I = +(said.match(/I:\s+(-?[\d.]+) LUFS(?![\s\S]*I:\s)/) || [0, -14])[1], up = Math.max(-6, Math.min(6, -14 - I));
if (Math.abs(up) > 0.3) { const tmp = path.join(out, "sound-tmp.wav"); execFileSync("ffmpeg", ["-y", "-loglevel", "error", "-i", path.join(out, name), "-af", `volume=${up.toFixed(2)}dB,alimiter=limit=0.8:attack=3:release=60:level=disabled`, tmp]); fs.renameSync(tmp, path.join(out, name)); }
console.log("out/" + name, LEN, "s,", I, "LUFS before, moved by", up.toFixed(1), "dB");
