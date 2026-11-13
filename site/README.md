# The Lootnest website

Plain HTML, CSS and JS, with no build step. `.github/workflows/site.yml`
publishes this folder to GitHub Pages when it changes on main.

- `index.html`, `style.css`, `site.js`: the page. Two scroll stories, each
  on one sticky stage: profiles and runs, then what's around a game
  (achievements, save points, mods, updates, Big Picture). Scrolling moves
  a clock from 0 to 5 and each actor has keyframes on it (`at()` in
  `site.js`). Nothing moves on its own.
- `saves.html`: how Lootnest handles saves, as a diagram.
- `fonts/`: Tilt Warp, Atkinson Hyperlegible Next and Figtree's digits,
  the app's own fonts (SIL Open Font License 1.1, see
  `assets/THIRD_PARTY_NOTICES.txt`).
- `img/`: the icon, and `game.jpg` for link previews only. The page has
  no screenshots on purpose. No real names anywhere on the site:
  profiles are "Player 1" and "Player 2".

To look at it locally: `python3 -m http.server -d site 8000`, then open
http://localhost:8000 (fonts don't load from `file://`).

Copy follows the app's voice: plain, specific, calm, no hype. Never say
"cracked": games from outside stores, DRM-free, portable.
