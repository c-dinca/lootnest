// Lootnest's site: the load screen in the hero (choose a slot and the
// game's folder gets its files, with a copy made first), the unlock note
// over the Achievements screenshot, and the latest version.
(() => {
  "use strict";
  const reduce = matchMedia("(prefers-reduced-motion: reduce)").matches;
  const wait = (ms) => new Promise((r) => setTimeout(r, reduce ? 0 : ms));

  // ------------------------------------------------------------ the swap ---
  const slots = [
    { name: "Player 1", color: "#6C9BD2", save: "Act 3, Lower City", mode: "Balanced", ctl: "Mouse, sensitivity 1.2", res: "2560 × 1440, Ultra", ach: "7 of 54" },
    { name: "Player 1, hardcore", color: "#6C9BD2", save: "Act 2, Moonrise Towers", mode: "Honour Mode", ctl: "Mouse, sensitivity 1.2", res: "2560 × 1440, Ultra", ach: "11 of 54" },
    { name: "Player 2", color: "#E88CA0", save: "Act 1, the Emerald Grove", mode: "Explorer", ctl: "Controller", res: "1920 × 1080, High", ach: "2 of 54" },
  ];
  const list = document.querySelector(".slots");
  const status = document.getElementById("status");
  const statusT = document.getElementById("status-t");
  const copies = document.getElementById("copies");
  const cells = [...document.querySelectorAll("#files dd")];
  let loaded = 0, cursor = 0, busy = false, made = 0, queued = -1;

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
    say(`Saving ${from.name}'s files back to their slot`, true);
    await wait(650);
    made++;
    copies.innerHTML = `<b>${made}</b> ${made === 1 ? "copy" : "copies"} made before swapping, kept in Lootnest's backups.`;
    requestAnimationFrame(() => copies.firstChild.classList.add("bump"));
    say("Copy made and read back", true);
    await wait(600);
    say(`Putting ${to.name}'s files in the game's folder`, true);
    const changing = cells.filter((dd) => dd.textContent !== to[dd.dataset.k]);
    for (const dd of changing) { dd.classList.remove("in"); dd.classList.add("out"); await wait(70); }
    await wait(220);
    for (const dd of changing) { dd.textContent = to[dd.dataset.k]; dd.classList.remove("out"); dd.classList.add("in"); await wait(60); }
    await wait(280);
    status.style.setProperty("--c", to.color);
    say(`${to.name}'s files are in the game's folder. Ready to play.`, false);
    loaded = i;
    busy = false;
    if (queued >= 0) { const q = queued; queued = -1; if (q !== loaded) load(q); }
  }

  if (list) {
    list.addEventListener("click", (e) => {
      const li = e.target.closest("li[data-p]");
      if (!li) return;
      point(+li.dataset.p);
      load(+li.dataset.p);
    });
    list.addEventListener("keydown", (e) => {
      const n = slots.length;
      if (e.key === "ArrowDown" || e.key === "ArrowUp") {
        e.preventDefault();
        point((cursor + (e.key === "ArrowDown" ? 1 : n - 1)) % n);
        load(cursor);
      } else if (e.key === "Enter" || e.key === " ") {
        e.preventDefault();
        load(cursor);
      }
    });
    // One swap on its own, so the page shows what Lootnest does before
    // anyone touches it: once, and only while the screen is in view.
    const io = new IntersectionObserver((es) => {
      if (es.some((x) => x.isIntersecting)) {
        io.disconnect();
        setTimeout(() => { if (loaded === 0 && !busy && cursor === 0) { point(1); load(1); } }, reduce ? 0 : 1600);
      }
    }, { threshold: 0.3 });
    io.observe(list.closest(".load"));
  }

  // ------------------------------------------------- the note arriving ---
  const toast = document.querySelector(".toast");
  if (toast && "IntersectionObserver" in window) {
    toast.style.opacity = "0";
    const io = new IntersectionObserver((es) => {
      if (es.some((x) => x.isIntersecting)) {
        io.disconnect();
        setTimeout(() => { toast.style.opacity = ""; toast.classList.add("show"); }, reduce ? 0 : 500);
      }
    }, { threshold: 0.5 });
    io.observe(toast.parentElement);
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
