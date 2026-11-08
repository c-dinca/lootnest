# Plugins

A Lootnest plugin is data, never code. It can add games to the Library, suggest save locations, fill in game details and change colours. Nothing in a plugin is run or loaded as code, and a plugin can't write saves or game folders, start programs or go online.

Install one in Settings, Plugins, **Install from file**. Lootnest keeps it in `plugins\<id>` in its data folder, and the version it replaced in `plugins\.old\<id>\<version>` for **Roll back**. In safe mode every plugin is off.

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

- `id`: 3 to 40 lowercase letters, digits or hyphens. A plugin with the same id replaces the one installed, once the new one checks out.
- `version`: three numbers, such as `1.2.0`.
- `provides`: one or more of `sources`, `saves`, `metadata` and `theme`.

Lootnest refuses the whole plugin, and says why, when:

- a file isn't `.json`, `.ini`, `.vdf`, `.yaml`, `.yml`, `.txt`, `.png`, `.jpg` or `.webp`;
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

Only these tokens can be set, each to a hex (`#rgb`, `#rrggbb`) or `oklch()` colour; only the lines may be see-through:

| Tokens | What they colour |
|---|---|
| `--bg-well`, `--bg-ground`, `--bg-surface`, `--bg-raised`, `--bg-active` | Wells and inputs, the window, panels, raised controls, selected rows |
| `--fg-1`, `--fg-2`, `--fg-3` | Text, secondary text, dates and counts |
| `--accent`, `--accent-hover`, `--fg-on-accent` | The primary button and marks, it on hover, the text on it |
| `--edge-1`, `--edge-2`, `--edge-3` | Hairlines, control edges, control edges on hover |

Sizes, fonts, spacing, the glass layers and anything else stay Lootnest's, and there's no raw CSS, so a theme can't hide a safety notice.

A theme is refused, with the pair that fails, unless every pair passes WCAG 2 contrast, with the theme's colours and Lootnest's own for the rest:

- `--fg-1`, `--fg-2` and `--fg-3` on `--bg-well`, `--bg-ground`, `--bg-surface`, `--bg-raised`, and on the sidebar's, the play bar's and the menus' glass over `--bg-ground`: 4.5:1;
- `--fg-1` and `--fg-2` on `--bg-active`: 4.5:1;
- `--fg-on-accent` on `--accent` and `--accent-hover`: 4.5:1;
- Lootnest's error, warning and done colours (`--danger`, `--warning`, `--success`, which a theme can't change) on `--bg-well`, `--bg-ground`, `--bg-surface` and `--bg-raised`: 4.5:1;
- `--accent` on `--bg-ground` and `--bg-surface`: 3:1, as it's never text.

Each profile chooses its theme (Edit profile, Look, Theme); it shows while that profile is at the PC. A run without one wears its person's.

## What plugins can't do

No code of any kind: no scripts, DLLs, .NET, PowerShell or Python. A plugin can't write saves or game folders, start programs, download or go online, or depend on another plugin. Online metadata, signatures and script plugins are later phases in [ROADMAP.md](ROADMAP.md), section 1.
