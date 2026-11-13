// Lootnest's site: the load screen in the hero (pick a slot and the game
// gets its saves, with a copy made first), the tour of screenshots, and
// the latest version. Nothing moves on its own with reduced motion.
(() => {
  "use strict";
  const reduce = matchMedia("(prefers-reduced-motion: reduce)").matches;
  const wait = (ms) => new Promise((r) => setTimeout(r, reduce ? 0 : ms));
  const seen = (el, fn, threshold = 0.3) => {
    if (!el || !("IntersectionObserver" in window)) return;
    const io = new IntersectionObserver((es) => es.forEach((x) => fn(x.isIntersecting)), { threshold });
    io.observe(el);
  };

  // ------------------------------------------------------------ the swap ---
  const slots = [
    { name: "Player 1", color: "#6C9BD2", save: "Act 3, Lower City", mode: "Balanced", ctl: "Mouse and keyboard", ach: "7 of 54" },
    { name: "Player 1, hardcore", color: "#6C9BD2", save: "Act 2, Moonrise Towers", mode: "Honour Mode", ctl: "Mouse and keyboard", ach: "11 of 54" },
    { name: "Player 2", color: "#E88CA0", save: "Act 1, the Emerald Grove", mode: "Explorer", ctl: "Controller", ach: "2 of 54" },
  ];
  const list = document.querySelector(".slots");
  const status = document.getElementById("status");
  const statusT = document.getElementById("status-t");
  const cells = [...document.querySelectorAll("#files dd")];
  let loaded = 0, cursor = 0, busy = false, queued = -1, touched = false, inView = false, timer = 0;

  const say = (text, on) => { statusT.textContent = text; status.classList.toggle("busy", !!on); };
  const items = () => [...list.querySelectorAll("li[data-p]")];
  const point = (i) => {
    cursor = i;
    items().forEach((li) => li.setAttribute("aria-selected", String(+li.dataset.p === i)));
    list.setAttribute("aria-activedescendant", "slot-" + i);
  };

  async function load(i) {
    if (busy) { queued = i; return; }
    if (i === loaded) return;
    busy = true;
    const from = slots[loaded], to = slots[i];
    say(`Copying ${from.name}'s saves`, true);
    await wait(550);
    say(`Swapping in ${to.name}`, true);
    const changing = cells.filter((dd) => dd.textContent !== to[dd.dataset.k]);
    for (const dd of changing) { dd.classList.remove("in"); dd.classList.add("out"); await wait(60); }
    await wait(200);
    for (const dd of changing) { dd.textContent = to[dd.dataset.k]; dd.classList.remove("out"); dd.classList.add("in"); await wait(60); }
    await wait(240);
    status.style.setProperty("--c", to.color);
    say(`${to.name} is loaded. ${from.name}'s saves were copied first.`, false);
    loaded = i;
    busy = false;
    if (queued >= 0) { const q = queued; queued = -1; if (q !== loaded) load(q); }
    else next();
  }

  // Until someone picks a slot, the screen goes through them on its own.
  function next() {
    clearTimeout(timer);
    if (touched || reduce || !inView) return;
    timer = setTimeout(() => { if (!touched && inView && !busy) { const n = (cursor + 1) % slots.length; point(n); load(n); } }, 3200);
  }
  const take = () => { touched = true; clearTimeout(timer); };

  if (list) {
    list.addEventListener("click", (e) => {
      const li = e.target.closest("li[data-p]");
      if (!li) return;
      take();
      point(+li.dataset.p);
      load(+li.dataset.p);
    });
    list.addEventListener("keydown", (e) => {
      const n = slots.length;
      if (e.key === "ArrowDown" || e.key === "ArrowUp") {
        e.preventDefault();
        take();
        point((cursor + (e.key === "ArrowDown" ? 1 : n - 1)) % n);
        load(cursor);
      } else if (e.key === "Enter" || e.key === " ") {
        e.preventDefault();
        take();
        load(cursor);
      }
    });
    seen(list.closest(".load"), (on) => { inView = on; if (on && !busy) next(); else if (!on) clearTimeout(timer); });
  }

  // ---------------------------------------------------------------- tour ---
  const tour = document.querySelector(".tour");
  if (tour) {
    const tabs = [...tour.querySelectorAll("[role=tab]")];
    const shots = [...tour.querySelectorAll(".tour-stage img")];
    const line = document.getElementById("tour-line");
    const stage = document.getElementById("tour-stage");
    let at = 0, auto = !reduce;
    const show = (i, focus) => {
      at = (i + shots.length) % shots.length;
      tabs.forEach((t, k) => { t.setAttribute("aria-selected", String(k === at)); t.tabIndex = k === at ? 0 : -1; });
      shots.forEach((s, k) => { s.classList.toggle("on", k === at); if (k === at) s.loading = "eager"; });
      shots[(at + 1) % shots.length].loading = "eager";
      line.textContent = shots[at].dataset.line;
      stage.setAttribute("aria-labelledby", tabs[at].id);
      if (focus) tabs[at].focus();
      // restart the progress line under the tab
      const bar = tabs[at].querySelector("i");
      bar.style.animation = "none"; void bar.offsetWidth; bar.style.animation = "";
    };
    const stop = () => { auto = false; tour.classList.remove("playing", "paused"); };
    tabs.forEach((t, k) => {
      t.addEventListener("click", () => { stop(); show(k); });
      t.querySelector("i").addEventListener("animationend", () => { if (auto) show(at + 1); });
    });
    tour.querySelector("[role=tablist]").addEventListener("keydown", (e) => {
      const d = e.key === "ArrowRight" ? 1 : e.key === "ArrowLeft" ? -1 : 0;
      if (e.key === "Home" || e.key === "End") { e.preventDefault(); stop(); show(e.key === "Home" ? 0 : shots.length - 1, true); }
      else if (d) { e.preventDefault(); stop(); show(at + d, true); }
    });
    // Plays only while it's on screen, and waits while the pointer is on it.
    seen(tour, (on) => { if (auto) tour.classList.toggle("playing", on); }, 0.35);
    stage.addEventListener("pointerenter", () => auto && tour.classList.add("paused"));
    stage.addEventListener("pointerleave", () => tour.classList.remove("paused"));
    show(0);
  }

  // -------------------------------------------------- latest version ---
  const ver = document.getElementById("ver");
  if (ver) {
    fetch("https://api.github.com/repos/c-dinca/lootnest/releases/latest", { headers: { Accept: "application/vnd.github+json" } })
      .then((r) => (r.ok ? r.json() : null))
      .then((j) => {
        const tag = j && typeof j.tag_name === "string" ? j.tag_name.replace(/^v/, "") : "";
        if (/^\d+\.\d+\.\d+$/.test(tag)) ver.textContent = `Version ${tag}, in beta.`;
      })
      .catch(() => {});
  }
})();
