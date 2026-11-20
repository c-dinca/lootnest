  // ------------------------------------------------------------ bp music --
  // Big Picture's music, as a console's home screen has: a calm, slow piece
  // made here as it plays (WebAudio oscillators, nothing recorded or
  // downloaded, so no licence to carry). Four chords, eight seconds each,
  // held by soft pads over a low root, with a few bell notes from the chord
  // falling now and then into an echo. It plays while Big Picture shows and
  // nothing else should be heard: never while a game or an app runs or
  // starts, under a trailer, behind the Home menu over a game, with the
  // window hidden, or with Lootnest's sounds off. It fades in and out.
  // Big Picture's Settings, Sound and music, turns it off and sets how loud.
  const MUSIC_VOL = { 1: 0.05, 2: 0.09, 3: 0.14 };
  // D major's colours: Dmaj9, Bm9, Gmaj7(#11), A6sus, as MIDI notes.
  const MUSIC_CHORDS = [
    { root: 38, pad: [62, 66, 69, 73, 76], bells: [78, 81, 85, 88] },
    { root: 35, pad: [62, 66, 69, 73, 74], bells: [74, 78, 81, 86] },
    { root: 31, pad: [59, 62, 66, 73, 74], bells: [73, 78, 79, 83] },
    { root: 33, pad: [62, 64, 66, 69, 71], bells: [76, 78, 81, 83] },
  ];
  const MUSIC_BAR = 8; // seconds a chord lasts
  const hz = (n) => 440 * Math.pow(2, (n - 69) / 12);
  const M = { out: null, verb: null, echo: null, on: false, vol: 0, shift: 0, next: 0, chord: 0, bellAt: 0, timer: 0 };
  // Every sound of the music goes through out (its volume, faded), part of
  // it through a long hall (verb) and an echo.
  function musicGraph() {
    const ctx = audioCtx();
    if (M.out) return ctx;
    M.out = ctx.createGain(); M.out.gain.value = 0;
    M.out.connect(ctx.destination);
    // A hall: three seconds of noise falling away, a little different in each ear.
    const len = Math.floor(ctx.sampleRate * 3), ir = ctx.createBuffer(2, len, ctx.sampleRate);
    for (let c = 0; c < 2; c++) { const d = ir.getChannelData(c); for (let i = 0; i < len; i++) d[i] = (Math.random() * 2 - 1) * Math.pow(1 - i / len, 3.2); }
    M.verb = ctx.createConvolver(); M.verb.buffer = ir;
    const wet = ctx.createGain(); wet.gain.value = 0.55;
    M.verb.connect(wet).connect(M.out);
    // The bells' echo: 420 ms, each repeat softer and duller.
    M.echo = ctx.createDelay(1); M.echo.delayTime.value = 0.42;
    const back = ctx.createGain(); back.gain.value = 0.38;
    const dull = ctx.createBiquadFilter(); dull.type = "lowpass"; dull.frequency.value = 2400;
    M.echo.connect(dull).connect(back).connect(M.echo);
    dull.connect(M.verb); dull.connect(M.out);
    return ctx;
  }
  // One voice: an oscillator (or two, a little apart) through a soft filter,
  // rising over attack and falling away over release.
  function musicVoice(ctx, at, f, { type = "sine", detune = 0, gain, attack, hold, release, cut = 1800, to = [M.out, M.verb] }) {
    const g = ctx.createGain(), lp = ctx.createBiquadFilter();
    lp.type = "lowpass"; lp.frequency.value = cut; lp.Q.value = 0.3;
    g.gain.setValueAtTime(0, at);
    g.gain.linearRampToValueAtTime(gain, at + attack);
    g.gain.setValueAtTime(gain, at + attack + hold);
    g.gain.exponentialRampToValueAtTime(0.0001, at + attack + hold + release);
    lp.connect(g); to.forEach((n) => g.connect(n));
    const end = at + attack + hold + release + 0.05;
    (detune ? [-detune, detune] : [0]).forEach((d) => {
      const o = ctx.createOscillator();
      o.type = type; o.frequency.value = f; o.detune.value = d;
      o.connect(lp); o.start(at); o.stop(end);
    });
  }
  // A chord from at: its pads overlap the next chord's by a few seconds.
  // Each game in focus gives the piece its own key, a few steps up or down
  // from D, so moving between games is heard as well as seen. It changes at
  // the next chord, never in the middle of one.
  function musicShift() {
    const k = BP.artKey || "";
    let h = 0;
    for (let i = 0; i < k.length; i++) h = (h * 31 + k.charCodeAt(i)) | 0;
    return k ? (Math.abs(h) % 5) - 2 : 0;
  }
  function musicChord(ctx, at, c) {
    const sh = (M.shift = musicShift());
    c = { root: c.root + sh, pad: c.pad.map((n) => n + sh), bells: c.bells };
    c.pad.forEach((n, i) => musicVoice(ctx, at + i * 0.12, hz(n), { type: i % 2 ? "triangle" : "sine", detune: 7, gain: 0.05, attack: 3, hold: MUSIC_BAR - 3, release: 4.5, cut: 1500 }));
    musicVoice(ctx, at, hz(c.root), { gain: 0.11, attack: 2.5, hold: MUSIC_BAR - 2.5, release: 3, cut: 400, to: [M.out] });
  }
  // A bell: struck, then fading for two seconds or so, into the echo.
  function musicBell(ctx, at, n) {
    musicVoice(ctx, at, hz(n), { gain: 0.045, attack: 0.008, hold: 0, release: 2.4, cut: 5000, to: [M.out, M.verb, M.echo] });
    musicVoice(ctx, at, hz(n + 12), { gain: 0.012, attack: 0.008, hold: 0, release: 1.1, cut: 6000, to: [M.out, M.echo] });
  }
  // Whether the music should be heard now.
  function musicWanted() {
    if (!BP.on || !BP.music || document.hidden || cfg().settings.soundsOff) return false;
    if (BP.launch || BP.playing || BP.tvMenu) return false;
    if ((S.d.sessions || []).some((s) => s.phase === "playing")) return false;
    const v = document.querySelector("#bp video.bp-bd-v");
    return !(v && !v.paused);
  }
  // Every half second: fade in or out as it should, and schedule what comes next.
  function musicTick() {
    const want = musicWanted();
    if (!want && !M.on) return;
    let ctx;
    try { ctx = musicGraph(); } catch (e) { return; }
    const now = ctx.currentTime, vol = MUSIC_VOL[BP.musicVol] || MUSIC_VOL[2];
    if (want !== M.on) {
      M.on = want;
      M.out.gain.cancelScheduledValues(now);
      M.out.gain.setValueAtTime(M.out.gain.value, now);
      M.out.gain.linearRampToValueAtTime(want ? vol : 0, now + (want ? 2.5 : 1.2));
      M.vol = vol;
      if (want && M.next < now) { M.next = now + 0.1; M.bellAt = now + 2; }
    } else if (want && M.vol !== vol) { // the volume was changed in Settings
      M.vol = vol;
      M.out.gain.cancelScheduledValues(now);
      M.out.gain.setValueAtTime(M.out.gain.value, now);
      M.out.gain.linearRampToValueAtTime(vol, now + 0.6);
    }
    if (!M.on) return;
    while (M.next < now + 1) {
      musicChord(ctx, M.next, MUSIC_CHORDS[M.chord]);
      M.chord = (M.chord + 1) % MUSIC_CHORDS.length;
      M.next += MUSIC_BAR;
    }
    while (M.bellAt < now + 1) {
      const c = MUSIC_CHORDS[(M.chord + MUSIC_CHORDS.length - 1) % MUSIC_CHORDS.length];
      musicBell(ctx, M.bellAt, c.bells[Math.floor(Math.random() * c.bells.length)] + (M.shift || 0));
      M.bellAt += 1.6 + Math.random() * 2.6;
    }
  }
  function musicStart() { if (!M.timer) M.timer = setInterval(musicTick, 500); musicTick(); }
  document.addEventListener("visibilitychange", () => { if (M.timer) musicTick(); });
