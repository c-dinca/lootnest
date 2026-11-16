# The trailer

A 70 second film of Lootnest, 1920x1080 at 60 frames a second, made
entirely from this folder: no editor, no stock footage, no licensed music.
Dev only, like the harness: nothing here is built into the app.

What you see in the windows and on the TV is the real UI. `record.mjs`
opens the design harness's page and records it frame by frame: Playwright's
clock fires the page's timers, and every CSS transition and animation is
paused and stepped by hand, so the launcher's own motion comes out at a
steady 60 frames a second. A drawn pointer follows the cues, with the real
mouse under it, so hover and clicks are the page's own. Profiles are
"Player 1" and "Player 2" (`?anon=1`), as on the website.

Around that footage, `trailer.html` (`trailer.js`, `trailer.css`) is one
page whose every frame is a function of the time: `seek(t)` places
everything. Titles, the swap as a diagram, the keys, the TV and the end
are drawn there, in the launcher's look (DESIGN.md) and its copy voice.
`render.mjs` asks for each frame, screenshots it and hands it to ffmpeg.

The sound is made by `music.mjs`: a small synthesized score at 120 beats a
minute (a bar is two seconds, which is what the film cuts on) and the app's
own interface sounds (Kenney's, CC0, `internal/app/ui/sounds`) at the
moments the picture shows them, mapped as Big Picture maps them.

```bash
node design/harness/fetch-art.mjs    # once: the fixtures' art
node design/harness/build.mjs        # the real UI as one page
node design/trailer/record.mjs       # out/footage: about 15 minutes
node design/trailer/music.mjs        # out/sound.wav
node design/trailer/render.mjs       # out/lootnest-trailer.mp4: about 12 minutes
```

Needs Node, ffmpeg and the Playwright that the harness uses. Everything it
writes goes to `out/`, which isn't committed.

While working on it:

- `PREVIEW=1 node design/trailer/record.mjs [name]` records every fifth
  frame at 1x, to check a take's cues quickly; `SRC=preview` makes
  `render.mjs` use those.
- `node design/trailer/render.mjs sheet 6 17 0.5` writes a contact sheet
  of the film from 6 s to 17 s; `still 12.5 31` writes single frames.
- `FROM=40 LEN=58 NAME=tv.mp4 node design/trailer/render.mjs` renders a
  part, without sound.
- A take's cues are in `record.mjs` (`takes`); where a take plays in the
  film, and every title, is in `trailer.js`. When a cue's time changes,
  the take's offset in `trailer.js` and the sound's cue in `music.mjs`
  move with it.

The film's order: one game, many lives; every game on your PC; a slot for
everyone and the swap; a run of your own; save points; achievements; Big
Picture on the TV; also in it; never; the end.

Game names and art belong to their makers, as the website says.
