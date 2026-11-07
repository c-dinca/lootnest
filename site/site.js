// Lootnest's site: the swap demo in the hero, the toast over the
// Achievements screenshot, and the latest version under the download.
(() => {
  "use strict";
  const reduce = matchMedia("(prefers-reduced-motion: reduce)").matches;
  const wait = (ms) => new Promise((r) => setTimeout(r, reduce ? 0 : ms));

  // ------------------------------------------------------------ the swap ---
  const profiles = [
    { name: "Cristian", color: "#6C9BD2", save: "Act 3, Lower City", mode: "Balanced", ctl: "Mouse, sensitivity 1.2", res: "2560 × 1440, Ultra", ach: "7 of 54" },
    { name: "Cristian hardcore", color: "#6C9BD2", save: "Act 2, Moonrise Towers", mode: "Honour Mode", ctl: "Mouse, sensitivity 1.2", res: "2560 × 1440, Ultra", ach: "11 of 54" },
    { name: "Mihai", color: "#E88CA0", save: "Act 1, the Emerald Grove", mode: "Explorer", ctl: "Controller", res: "1920 × 1080, High", ach: "2 of 54" },
  ];
  const who = document.querySelector(".demo-who");
  const status = document.getElementById("status");
  const statusT = document.getElementById("status-t");
  const copies = document.getElementById("copies-line");
  const cells = [...document.querySelectorAll("#files dd")];
  let loaded = 0, busy = false, copyCount = 0;

  const say = (text, on) => {
    statusT.textContent = text;
    status.classList.toggle("busy", !!on);
  };

  async function swapTo(i) {
    if (busy || i === loaded) return;
    busy = true;
    const from = profiles[loaded], to = profiles[i];
    who.querySelectorAll("button").forEach((b) => {
      b.setAttribute("aria-checked", String(+b.dataset.p === i));
      b.disabled = true;
    });
    say(`Saving ${from.name}'s files back to their profile`, true);
    await wait(700);
    copyCount++;
    copies.innerHTML = `<span>${copyCount}</span> ${copyCount === 1 ? "copy" : "copies"} kept`;
    const n = copies.firstChild;
    requestAnimationFrame(() => n.classList.add("bump"));
    say("Copy made and read back", true);
    await wait(650);
    say(`Loading ${to.name}'s files`, true);
    for (const dd of cells) {
      const v = to[dd.dataset.k];
      if (dd.textContent === v) continue;
      dd.classList.remove("in"); dd.classList.add("out");
      await wait(90);
    }
    await wait(240);
    for (const dd of cells) {
      if (!dd.classList.contains("out")) continue;
      dd.textContent = to[dd.dataset.k];
      dd.classList.remove("out"); dd.classList.add("in");
      await wait(70);
    }
    await wait(300);
    status.style.setProperty("--c", to.color);
    say(`${to.name}'s files are loaded. Ready to play.`, false);
    loaded = i;
    who.querySelectorAll("button").forEach((b) => (b.disabled = false));
    busy = false;
  }

  if (who) {
    who.addEventListener("click", (e) => {
      const b = e.target.closest("button[data-p]");
      if (b) swapTo(+b.dataset.p);
    });
    who.addEventListener("keydown", (e) => {
      if (!["ArrowRight", "ArrowLeft", "ArrowDown", "ArrowUp"].includes(e.key)) return;
      e.preventDefault();
      const n = profiles.length;
      const next = (loaded + (e.key === "ArrowRight" || e.key === "ArrowDown" ? 1 : n - 1)) % n;
      who.querySelector(`[data-p="${next}"]`).focus();
      swapTo(next);
    });
    // One swap on its own, so the page shows what Lootnest does before anyone
    // clicks. Only once, and only while the demo is on screen.
    const io = new IntersectionObserver((es) => {
      if (es.some((x) => x.isIntersecting)) {
        io.disconnect();
        setTimeout(() => { if (loaded === 0 && !busy) swapTo(1); }, reduce ? 0 : 1400);
      }
    }, { threshold: 0.6 });
    io.observe(who.closest(".demo"));
  }

  // ------------------------------------------------- the toast arriving ---
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

  // ------------------------------------------------------- the header ---
  const top = document.querySelector(".top");
  const onScroll = () => top.classList.toggle("scrolled", scrollY > 8);
  addEventListener("scroll", onScroll, { passive: true });
  onScroll();

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
