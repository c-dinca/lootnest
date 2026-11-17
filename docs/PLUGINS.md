# Plugins

There are two kinds of plugin. A **data plugin** can add games to the Library, suggest save locations, fill in game details and change colours; nothing in it is run. A **script plugin** has one JavaScript file that Lootnest runs inside itself, and it can only do what its `plugin.json` asks for and you allow (see [script](#script-a-plugin-with-code)). No plugin holds a program, and Lootnest comes with none installed.

Install one in Settings, Plugins, **Install from file**. A plugin can be signed by its author (see Signing). Lootnest keeps it in `plugins\<id>` in its data folder, and the version it replaced in `plugins\.old\<id>\<version>` for **Roll back**. In safe mode every plugin is off.

## The package

A `.zip` with `plugin.json` at its top:

```json
{
  "id": "retroarch",
  "name": "RetroArch playlists",
  "version": "1.0.0",
  "author": "Your name",
  "description": "Every game in your RetroArch playlists, in the Library.",
  "provides": ["sources"]
}
```

- `id`: 3 to 40 lowercase letters, digits or hyphens. A plugin with the same id replaces the one installed, once the new one checks out. Ids starting `ln-` are kept for Lootnest's own themes.
- `version`: three numbers, such as `1.2.0`.
- `provides`: one or more of `sources`, `saves`, `metadata`, `theme` and `script`.

Lootnest refuses the whole plugin, and says why, when:

- a file isn't `.json`, `.ini`, `.vdf`, `.yaml`, `.yml`, `.txt`, `.png`, `.jpg` or `.webp`, or, in a script plugin, its one `.js` (at most 2 MB);
- a file's path leaves the plugin's folder, or is a link;
- a file is over 8 MB, all of them are over 32 MB, or there are more than 500;
- `plugin.json` doesn't check out, or something it provides can't be read.

## sources: games from other apps

One file per source in `sources/*.json` (up to 20). Lootnest reads them each time it looks for games, read-only, and adds each game whose folder and `.exe` are on the PC. Its store shows as the source's `name`.

```json
{
  "name": "RetroArch",
  "kind": "json",
  "path": "%APPDATA%\\RetroArch\\playlists\\*.lpl",
  "list": "items",
  "fields": { "name": "label", "exe": "path", "dir": "", "steamId": "" }
}
```

- `kind`: `folder`, `json`, `ini`, `vdf` or `registry`.
- `path`: starts with `%APPDATA%`, `%LOCALAPPDATA%`, `%USERPROFILE%`, `%PROGRAMDATA%`, `%PROGRAMFILES%` or `%PROGRAMFILES(X86)%`, then folders inside it; no drive and no `..`. The file's name may have `*` or `?`. For `registry`, it's a key under `HKCU\` or `HKLM\SOFTWARE\`.
- `list`: where the games are. For `json` and `vdf`, a dotted path of keys (`Library.Games`) to an array of objects or an object of objects; empty for the top. For `ini`, the start of the sections' names; empty for every section.
- `fields`: which field gives the game's `name`, its `exe`, its folder (`dir`) and its Steam app id (`steamId`). An `exe` that isn't a full path is inside `dir`; with no `exe`, Lootnest picks the main `.exe` in `dir`. For `ini`, no `name` uses the section's name. For `registry`, every subkey is a game and the fields name its values.
- `folder`: every folder in `path` is a game, with its main `.exe`; `fields` aren't used.

## saves: save locations, as suggestions

`saves/*.yaml` in the [Ludusavi manifest's](https://github.com/mtkennerly/ludusavi-manifest) format (Windows paths only):

```yaml
"Super Metroid":
  files:
    "<winAppData>/RetroArch/saves/Super Metroid.srm":
      tags:
        - save
```

These show among a game's suggested save locations, marked **From** and the plugin's name, after the ones from PCGamingWiki. They're never used by themselves: only when you choose one.

## metadata: game details

`metadata/*.json`, keyed by Steam app id or by the game's name (letters and digits, any case):

```json
{
  "Super Metroid": { "summary": "Samus explores planet Zebes.", "genres": ["Action", "Adventure"], "developer": "Nintendo", "publisher": "Nintendo", "released": "1994" }
}
```

They fill in what Steam's details leave empty, and show with Game details off too. Your own edits win over both. Nothing is fetched from anywhere.

## theme: colours

`theme.json`, with new values for Lootnest's colour tokens:

```json
{
  "name": "Deep sea",
  "tokens": { "--bg-ground": "#0d1520", "--bg-raised": "#1b2a3d", "--accent": "oklch(0.70 0.15 220)", "--fg-on-accent": "#0d1520" }
}
```

Only these tokens can be set, each to a hex (`#rgb`, `#rrggbb`), `rgb()` or `oklch()` colour; only the lines may be see-through:

| Tokens | What they colour |
|---|---|
| `--bg-well`, `--bg-ground`, `--bg-surface`, `--bg-raised`, `--bg-active` | Wells and inputs, the window, panels, raised controls, selected rows |
| `--fg-1`, `--fg-2`, `--fg-3` | Text, secondary text, dates and counts |
| `--accent`, `--accent-hover`, `--fg-on-accent` | The primary button and marks, it on hover, the text on it |
| `--edge-1`, `--edge-2`, `--edge-3` | Hairlines, control edges, control edges on hover |

Sizes, fonts, spacing, the glass layers and anything else stay Lootnest's, and there's no raw CSS, so a theme can't hide a safety notice.

A theme is refused, with the pair that fails, unless every pair passes WCAG 2 contrast, with the theme's colours and Lootnest's own for the rest:

- `--fg-1`, `--fg-2` and `--fg-3` on `--bg-well`, `--bg-ground`, `--bg-surface`, `--bg-raised`, and on the sidebar's, the play dock's and the menus' glass over `--bg-ground`: 4.5:1;
- `--fg-1` and `--fg-2` on `--bg-active`: 4.5:1;
- `--fg-on-accent` on `--accent` and `--accent-hover`: 4.5:1;
- Lootnest's error, warning and done colours (`--danger`, `--warning`, `--success`, which a theme can't change) on `--bg-well`, `--bg-ground`, `--bg-surface` and `--bg-raised`: 4.5:1;
- `--accent` on `--bg-ground` and `--bg-surface`: 3:1, as it's never text.

Each profile chooses its theme (Edit profile, Look, Theme); it shows while that profile is at the PC. A run without one wears its person's.

## Signing

A plugin may be signed, so a later version can only come from whoever made the first one. Unsigned plugins install as before; Settings, Plugins shows each as **Signed, key 1A2B 3C4D 5E6F 7A8B** or **Not signed**.

A signed plugin has `signature.json` at its top:

```json
{ "key": "<Ed25519 public key, base64, 32 bytes>", "sig": "<signature, base64, 64 bytes>" }
```

The signature is over a list with one line per file, every file but `signature.json`: its path inside the plugin with forward slashes, a tab, its SHA-256 in lowercase hex, and a newline (`theme.json\t9f86...\n`), sorted, as UTF-8. The key's fingerprint, shown in Lootnest, is the first 16 hex digits of the key's SHA-256, in groups of four.

There's no central authority and nothing goes online: Lootnest trusts a key the first time it sees it, as SSH does with a server's key, and keeps it for that plugin's id in `plugins.json`.

- A signature that doesn't match the files is refused: something changed them after the author signed it. A plugin whose files change once installed is turned off, with the reason, the next time Lootnest reads it (a scan, Settings, Plugins, or a start).
- A new version signed with another key is refused, with both fingerprints, unless you choose **Install with the new key**. That can mean a new author, or someone else's copy.
- A signed version replacing an unsigned one pins its key from then on, and says so.
- An unsigned version can't replace a signed one, so a signature can't be dropped without you noticing.
- **Roll back** is offered only for a kept version signed with the pinned key, so not for one with an old key or none.
- **Remove** forgets the key, so the next install of that id is a first one again.

To sign, from a copy of this repository (needs Go):

```
go run ./cmd/plugsign keygen
go run ./cmd/plugsign sign path\to\my-plugin
```

`keygen` writes `plugsign.key`, the private key (keep it safe, and never commit or share it), in the folder you run it from, and prints its fingerprint. It never replaces a key that's there: with a lost key, your next version needs your users to accept a new one. `sign` checks the folder the way Lootnest does, then writes its `signature.json`; give it another key file as a third argument. Sign last, after every other change, then zip the folder's contents.

## script: a plugin with code

```json
{
  "id": "my-plugin",
  "name": "My plugin",
  "version": "1.0.0",
  "provides": ["script"],
  "script": "main.js",
  "permissions": ["net", "library", "updates", "notify"],
  "secrets": [{ "name": "service", "host": "api.example.com", "label": "Example API token" }],
  "pages": [{ "id": "main", "title": "My plugin" }]
}
```

The script is JavaScript (ES2020 or so, no modules), run by an engine inside Lootnest. It isn't a browser or Node: there's no `window`, `fetch`, `require`, files or processes, only the `lootnest` object. Nothing of the plugin's is ever started as a program.

**It asks first.** A script plugin is off when installed. Turning it on shows what it asks to do, in words, with its signature, and it runs only once you allow that. A new version that asks for the same keeps running; one that asks for anything else is off until you allow it again. In safe mode no script runs.

| In `permissions` | What the script may do |
|---|---|
| (always) | Keep its own data (`lootnest.storage`), draw its pages, write to Lootnest's log, plan a later call |
| `net` | `lootnest.net.fetch`: http and https requests to the public internet, never this PC or its network; each has a size and time limit and is logged |
| `library` | `lootnest.library.games()`: each game's key, name, where it came from, Steam id, its folder's name, its version when Lootnest can read one, and what the launcher that installed it recorded about its download |
| `updates` | `lootnest.updates.offer(list)`: the script's whole list of updates, each `{id, game, title, version, have, source, size, date, note, status, busy, actions: [{id, label}]}`, where `game` is a game's key. They show on the Updates page and on the game's page, only for games in the Library that no store keeps up to date. Downloading and replacing a game's folder is Lootnest's own code, started by you from such a row, with the saves copied first |
| `notify` | `lootnest.notify(text)`: a notice |

**Keys.** A service's token is listed in `secrets` with the one site it's for. You type it into a form Lootnest draws; it's encrypted for your Windows user and the script never reads it. `lootnest.net.fetch(url, { secret: "service" })` sends it as `Authorization: Bearer`, only over https and only to that site.

**The calls.**

- `lootnest.on(name, fn)`: a handler. Lootnest calls `start` when the script starts, `view({page})` to draw a page, `action({id, row, value, on, values, secrets})` for a page's buttons and forms, `library()` after it looked for games again, `offer({id, action})` for a button of an update the script offered, and any name you gave `schedule`. Only JSON goes in and out. A handler has 3 minutes; one that takes longer is stopped.
- `lootnest.schedule(name, ms, arg)`: calls the handler `name` after `ms`. One pending call per name; `ms` under 0 cancels it.
- `lootnest.storage.get(key)`, `.set(key, value)`, `.remove(key)`: JSON values in `plugin-data\<id>` in Lootnest's data folder, at most 64 MB each and 512 MB in all. They stay when the plugin is removed.
- `lootnest.net.fetch(url, {method, headers, body, form, maxBytes, timeout, secret})` waits and gives `{status, headers, text}`. `maxBytes` is 8 MB unless set, 64 MB at most; `timeout` is 30 seconds unless set.
- `lootnest.ui.changed()`: asks Lootnest to draw the plugin's open page again.
- `lootnest.log(text)`, `lootnest.hash(text)` (SHA-256 in hex), `lootnest.secrets.has(name)`, `lootnest.plugin.id`.

**Pages.** Each page in `pages` is a category in Settings. `view` returns data and Lootnest draws it with its own components: no HTML, CSS or script of the plugin's reaches the window.

```js
lootnest.on("view", () => ({
  sections: [{
    title: "Things",
    actions: [{ id: "add", label: "Add", kind: "primary", form: "add" }],
    empty: "Nothing yet",
    rows: [{
      id: "t1", title: "A thing", sub: "a second line", lamp: "on",        // on, warn, bad, off, busy
      tags: ["3 items"], note: "Checked 5 min ago", warn: "", error: "",
      toggle: { on: true },                                               // action "toggle" with {row, on}
      select: { value: "6", options: [{ value: "6", label: "Six" }] },    // action "select" with {row, value}
      actions: [{ id: "remove", label: "Remove", confirm: { title: "Remove it?", body: "", ok: "Remove" } }],
    }],
  }],
  forms: { add: { title: "Add a thing", ok: "Add", fields: [{ id: "name", label: "Name", type: "text" }, { id: "token", label: "Token", secret: "service" }] } },
}));
lootnest.on("action", (a) => (a.id === "add" && !a.values.name ? { error: "Give it a name." } : { toast: "Done." }));
```

A field's `type` is `text`, `url`, `number`, `select` or `check`. An action's answer may hold `error` (shown, and a form stays open), `toast`, or `form` (the id of a form to open next).

The engine has a time limit but no memory limit, which is one reason a script plugin is never on by itself. Install only ones whose author you trust; signing (below) tells you a later version is from the same author.

## What plugins can't do

No programs of any kind: no DLLs, .NET, PowerShell, Python or executables, and a data plugin has no code at all. A script reaches only the calls above: it can't read or write files outside its own data, start programs, read keys, reach this PC or its network, or depend on another plugin. Saves and game folders are only ever changed by Lootnest's own code, with a copy first, when you ask.
