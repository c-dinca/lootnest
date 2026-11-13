# The Lootnest website

Plain HTML, CSS and JS, with no build step. `.github/workflows/site.yml`
publishes this folder to GitHub Pages when it changes on main.

- `index.html`, `style.css`, `site.js`: the page. The hero's slot demo
  (it goes through the slots on its own until someone picks one) and the
  tour of screenshots are in `site.js`; neither moves with reduced motion.
- `saves.html`: how Lootnest handles saves, as a diagram.
- `fonts/`: Tilt Warp, Atkinson Hyperlegible Next and Figtree's digits,
  the app's own fonts (SIL Open Font License 1.1, see
  `assets/THIRD_PARTY_NOTICES.txt`).
- `img/`: screenshots of the app from the design harness
  (`design/harness`, `?anon=1&runs=1`), at 2x, as JPEG. No real names
  anywhere on the site: profiles are "Player 1" and "Player 2".

To look at it locally: `python3 -m http.server -d site 8000`, then open
http://localhost:8000 (fonts don't load from `file://`).

Copy follows the app's voice: plain, specific, calm, no hype. Never say
"cracked": games from outside stores, DRM-free, portable.
