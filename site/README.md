# The Lootnest website

Plain HTML, CSS and JS, with no build step. `.github/workflows/site.yml`
publishes this folder to GitHub Pages when it changes on main.

- `index.html`, `style.css`, `site.js`: the page. Two scroll stories, each
  played in a Lootnest window drawn in HTML and CSS (`story1` and `story2`
  in `site.js`, the `.app` and `a-` rules in `style.css`): profiles and
  runs (Library, a game's page, Play, Who's playing, the swap, New run),
  then what's around a game (an unlock over the game, a save point,
  Mods, Updates, Big Picture). Scrolling moves a clock from 0 to 5; the
  pointer follows keyframes on it (`at()`), the rest are states it
  switches, so scrolling back undoes them. Under 700px wide the window is
  640 wide, without its sidebar. The window uses the site's palette,
  which is where the launcher is heading.
- `saves.html`: how Lootnest handles saves, as a diagram.
- `fonts/`: Tilt Warp, Atkinson Hyperlegible Next and Figtree's digits,
  the app's own fonts (SIL Open Font License 1.1, see
  `assets/THIRD_PARTY_NOTICES.txt`).
- `img/`: the icon; `og.jpg`, the link preview (a shot of the window);
  `art/`, game covers and wide art as small WebP, made from the design
  harness's art cache (`design/harness/.cache`). No real names anywhere
  on the site: profiles are "Player 1" and "Player 2".

To look at it locally: `python3 -m http.server -d site 8000`, then open
http://localhost:8000 (fonts don't load from `file://`).

Copy follows the app's voice: plain, specific, calm, no hype. Never say
"cracked": games from outside stores, DRM-free, portable.
