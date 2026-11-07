# Design harness

Runs the real Lootnest UI (`internal/app/ui`) in a plain browser, with the Go
side replaced by fixtures, so every page and state can be opened and
screenshotted without Windows. Dev only: nothing here is built into the app
(`go:embed` only takes `internal/app/ui`).

```bash
node design/harness/fetch-art.mjs   # once: Steam art for the fixtures, into .cache/ (not committed)
node design/harness/build.mjs       # dist/page.html, put together like app.go's pageHTML
node design/harness/shoot.mjs before [filter]   # screenshots into design/before/
```

Open `dist/page.html` in a browser to click around. The scenario comes from
the URL: `?games=13|0|400&players=1|2|6&state=normal|loading|offline|error|playing&cloud=0|1&first=1`.

- `stub.js` defines every `window.ps_*` call and emits the `__ps` events the
  Go side would, from fixture data (13 games: 3 with profiles, the others
  from Steam, Epic, GOG, Xbox, Rockstar, a game folder and one added by hand;
  one with no art, one with another game's art, one long title).
- Nothing the UI asks for changes anything: saves, swaps and settings calls
  answer without doing anything.
