# The rework's reference

`reference.html` is the look every phase of `docs/REWORK.md` follows:
direction A of three mockups (13 Nov 2026), with a sidebar. Open it in a
browser; it's interactive (covers, Play, the profile at the foot of the
sidebar, the tabs, Library, Saves, Settings, T for an unlock toast).

- `art/`: covers, wide art and logos as small WebP, from the harness's
  art cache. They belong to the games' makers; dev only.
- `src/`: what `reference.html` was built from (`base.css` with fonts,
  `data.js`, the page). The fonts are inlined from `internal/app/ui/fonts`.

The mockup isn't the app's code: build the real screens from the app's
own components and tokens, using this for the look, the layout and the
motion.
