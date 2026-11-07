# The Lootnest website

Plain HTML, CSS and JS, with no build step. `.github/workflows/site.yml`
publishes this folder to GitHub Pages when it changes on main.

- `index.html`, `style.css`, `site.js`: the page. The hero's swap demo and
  the unlock note's arrival are in `site.js`; both respect reduced motion.
- `fonts/`: Bricolage Grotesque and Figtree, the app's own fonts (SIL Open
  Font License 1.1, see `assets/THIRD_PARTY_NOTICES.txt`).
- `img/`: screenshots of the app from the design harness
  (`design/harness`), at 2x, as JPEG.

To look at it locally: `python3 -m http.server -d site 8000`, then open
http://localhost:8000 (fonts don't load from `file://`).

Copy follows the app's voice: plain, specific, calm, no hype. Never say
"cracked": games from outside stores, DRM-free, portable.
