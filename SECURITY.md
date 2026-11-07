# Security

Lootnest runs on your PC with your Windows account's rights, copies game
files in and out of place, starts games and can update itself. This is how
it keeps that safe.

## Reporting a problem

Please report security problems privately through GitHub: **Security >
Report a vulnerability** on this repository. Don't open a public issue.

## What Lootnest does to stay safe

**Your files**
- Every swap is backed up first, files are written to a temporary name and
  then moved into place, so a crash never leaves half-written saves.
- Folders too broad to swap are refused: drive roots, your user folder,
  AppData itself, Windows, Program Files, Steam's own folders and
  Lootnest's data folder (`Env.CheckSafe`).
- Game Mover copies and checks every file before anything is removed.

**The window**
- The interface is one local page, built into `Lootnest.exe`. Nothing is
  loaded from the internet to run it: no remote scripts, fonts or styles.
- A Content-Security-Policy allows only that page's own script (with a
  nonce made at each start); injected scripts and inline handlers don't run.
- The window can't navigate anywhere or fetch anything: every request it
  makes goes through an allowlist (Lootnest's page, and pictures on Steam's
  image servers for news). Everything else gets an empty 403.
- Text from games, scans and the internet is always escaped before it's
  shown. Steam news is converted to a small set of safe HTML elements in
  Go (`core.NewsHTML`); links in it open in your browser, never in Lootnest.
- Developer tools and the browser's context menu are off.

**Going online**
- Only for features you turn on, listed in the README's Privacy section.
- Links Lootnest opens are checked first: Steam news pages only for news,
  and web links only (`https`/`http`) from inside posts.
- Updates come only from this repository's GitHub releases. The installer
  must be one of the release's own files and must match the SHA-256 GitHub
  lists for it; anything else is refused before it runs.

**Keys you save**
- A SteamGridDB API key is encrypted for your Windows account with DPAPI
  before it's written to `config.json`; another user or PC can't read it.

**Backups**
- A backup is one file in the folder you choose; Lootnest never uploads it.
  It holds every player's profiles, the save files in place, profile
  pictures and Lootnest's own lists, and is first written as `.partial`,
  then renamed, so a sync app never picks up half a file.
- With a password, it's encrypted with AES-256-GCM, in 1 MiB chunks, each
  bound to its position and to whether it's the last one, so a changed,
  reordered or cut-off file is refused instead of restored. The key comes
  from the password through PBKDF2-SHA256 (600,000 rounds, random salt).
  Only that key is kept (DPAPI-encrypted, for automatic backups), never
  the password.
- Restoring checks every file name first: only Lootnest's own files and
  folders, no paths outside them, no device names, within size and count
  limits. Games in the backup are checked like ones you add, and their
  folders again before every swap. Everything on the PC is saved to
  `backups\before-restore` before anything is replaced.

**Cloud saves**
- Each player's saves for a game are one file in the cloud folder, with
  the same encryption as backups when the folder has a password. The
  folder's `key.json` holds the password's salt and an encrypted check, so
  every PC makes the same key from the same password and a wrong one is
  refused; the password itself is never stored.
- Saves that come down are checked like a restore (names, sizes, counts)
  and only go into the save locations this PC's game already has. The
  files in place are backed up before they're replaced, and when saves
  changed on two PCs, the one not kept is saved in that game's backups.

**Building**
- Lootnest is built with the current Go release (the `toolchain` line in
  `go.mod`), and every build runs `govulncheck`, which fails the build on a
  known vulnerability in Go or in the code Lootnest uses.

## What it doesn't protect against

- Anything already running as your Windows user can read and change
  Lootnest's files, like it can any other program's.
- Lootnest isn't code-signed yet, so Windows SmartScreen warns on first
  install.
