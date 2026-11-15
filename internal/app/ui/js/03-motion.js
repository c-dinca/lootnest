  // --------------------------------------------------------------- motion --
  // Springs, as Motion (motion.dev) and the platforms' own UI use them: a
  // damped spring (mass 1) set by how much it bounces and how long it
  // takes to look done, sampled into a CSS linear() easing so CSS
  // transitions, keyframes and the Web Animations API all move the same
  // way. Opacity never takes a bouncing spring (it would flicker past 0
  // or 1); it fades with --ease-out.
  //   stiffness = (2π / visual)²,  damping = 2 (1 − bounce) √stiffness
  function spring(bounce, visual) {
    const k = (2 * Math.PI / visual) ** 2, w0 = Math.sqrt(k), z = 1 - bounce;
    const x = z < 1
      ? (t) => { const wd = w0 * Math.sqrt(1 - z * z); return 1 - Math.exp(-z * w0 * t) * (Math.cos(wd * t) + ((z * w0) / wd) * Math.sin(wd * t)); }
      : (t) => 1 - Math.exp(-w0 * t) * (1 + w0 * t);
    // Done once it stays within 0.1% of the end.
    let end = visual;
    for (let t = 0; t < 4; t += 0.004) if (Math.abs(1 - x(t)) > 0.001) end = t;
    const n = 56, pts = [];
    for (let i = 0; i <= n; i++) pts.push(i === n ? 1 : +x((end * i) / n).toFixed(4));
    return { ease: `linear(${pts.join(", ")})`, ms: Math.round(end * 1000) };
  }
  // The four springs Lootnest moves with.
  const SPRINGS = {
    press: spring(0, 0.12),   // a key going down: no give
    snappy: spring(0.12, 0.28), // hover, a cover lifting, a pill gliding
    bouncy: spring(0.32, 0.42), // something that lands: a dialog, a toast, a switch, a count
    soft: spring(0, 0.5),     // a page or a panel arriving
  };
  (function setSprings() {
    const r = document.documentElement.style;
    Object.entries(SPRINGS).forEach(([k, v]) => { r.setProperty(`--spring-${k}`, v.ease); r.setProperty(`--spring-${k}-dur`, v.ms + "ms"); });
  })();
  // Covers lean toward the pointer, with a light where it rests: the
  // Library's covers and its wide shelf art. Mouse only, Motion on Full.
  (function coverTilt() {
    const TILT = "ln-tile__cover, lib-cap-art".split(", ").map((c) => "." + c).join(", ");
    let raf = 0, at = null, on = null;
    const reset = (el) => ["--tx", "--ty", "--lx", "--ly"].forEach((k) => el.style.removeProperty(k));
    document.addEventListener("pointermove", (e) => {
      const host = e.pointerType === "mouse" && e.target.closest ? e.target.closest(".ln-tile, .lib-cap-open") : null;
      const art = host && host.querySelector(TILT);
      if (on && on !== art) { reset(on); on = null; }
      if (!art || !moving()) return;
      on = art; at = e;
      if (raf) return;
      raf = requestAnimationFrame(() => {
        raf = 0;
        if (!on || !at) return;
        const r = on.getBoundingClientRect(), x = (at.clientX - r.left) / r.width, y = (at.clientY - r.top) / r.height;
        const deg = parseFloat(getComputedStyle(document.documentElement).getPropertyValue("--tilt")) || 7;
        on.style.setProperty("--tx", ((x - 0.5) * 2 * deg).toFixed(2) + "deg");
        on.style.setProperty("--ty", (-(y - 0.5) * 2 * deg).toFixed(2) + "deg");
        on.style.setProperty("--lx", (x * 100).toFixed(1) + "%");
        on.style.setProperty("--ly", (y * 100).toFixed(1) + "%");
      });
    }, { passive: true });
  })();
  // Content that replaces a skeleton comes up in reading order instead of
  // popping in: the first blocks in view, one after another.
  function revealIn(root, sel, max = 14) {
    if (!root || !moving()) return;
    const vh = innerHeight;
    [...root.querySelectorAll(sel)].filter((el) => { const r = el.getBoundingClientRect(); return r.height && r.top < vh && r.bottom > 0; })
      .slice(0, max).forEach((el, i) => {
        el.style.setProperty("--i", i);
        el.classList.add("reveal");
        el.addEventListener("animationend", () => el.classList.remove("reveal"), { once: true });
      });
  }
