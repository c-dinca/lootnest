# Lootnest Link: technical spec

Nov 15, 2026, Cristian Dinca

## Goal and scope

Lootnest Link lets someone play the games on their Windows PC on any TV,
with a controller, inside the Lootnest interface. They install nothing
except Lootnest on the PC and one Lootnest app on the TV. Lootnest
orchestrates existing open-source streaming components; it does not
reimplement video streaming.

### Target experience

1. In Lootnest settings, the user turns on TV streaming. Lootnest installs
   and configures the streaming engine (one Windows admin prompt).
2. The user clicks Add a TV. Lootnest finds the TV on the network, installs
   Lootnest for TV on it (from the TV's app store where possible, otherwise
   through developer mode) and pairs it automatically.
3. On the couch, the user opens Lootnest for TV. The PC wakes if asleep,
   Lootnest's TV mode appears, the user picks who is playing, then a game.
   Quitting the game returns to Lootnest TV mode.

### In scope for v1

- Host: Windows 10/11 with an NVIDIA, AMD or Intel GPU that has a hardware
  encoder.
- Client: any TV. Native apps for Android TV / Google TV / Fire TV, LG webOS
  and Samsung Tizen, plus a browser fallback for other smart TVs (see TV
  platforms).
- Network: same LAN only, one TV streaming at a time.

### Non-goals for v1

- Streaming over the internet (possible later through Tailscale).
- A custom video encoder or streaming protocol.
- Several TVs streaming different games at once.
- TVs with no smart platform and no browser (they need a cheap streaming
  stick or an HDMI cable).
- Custom hardware.

## Architecture

Lootnest stays the only app the user sees on the PC. It drives a bundled
streaming engine through the engine's local API, and each TV runs a thin
client that only decodes video and sends controller input back.

The streamed picture is always Lootnest's own TV mode or the game it
launched, so profiles, save swapping and achievements work exactly as on the
desktop. The browser client gets its stream from a relay on the PC instead
of a native app.

## Phase 1: Lootnest TV mode

Phase 1 adds a fullscreen, controller-first TV mode to the existing Lootnest
frontend, launched with a command-line flag so a streaming host can start
it. It ships value on its own: it works with stock Apollo + Moonlight TV
before any of Phase 2 or 3 exists.

### Entry point

- `lootnest.exe --tv` opens Lootnest in a borderless fullscreen WebView2
  window on the primary display. During a stream, Apollo makes its virtual
  display the primary one, so TV mode lands on the TV.
- Single instance: if Lootnest is already running, the second launch tells
  the running instance over a named pipe to open and focus TV mode, then
  exits.
- The `--tv` process must stay alive for the whole session. When it exits,
  the streaming host treats the app as closed and ends the stream.

### Screens

1. Who's playing: large profile tiles, using the existing "who's at the PC"
   seat logic.
2. Home: rows for Continue playing (this profile's recent games), All games
   and Collections.
3. Game page: cover, this profile's playtime and achievements, the Play
   button.
4. Launching: "Swapping saves for <profile>…", then hand-off to the game.
5. Menu: switch profile, exit TV mode.

### Input

- Read controllers with the Gamepad API (`navigator.getGamepads()`), polled
  in `requestAnimationFrame`, using the standard mapping.
- Mapping: D-pad or left stick moves focus, A selects, B goes back, Start
  opens the menu, LB/RB switch rows or tabs. On hold, repeat after 400 ms,
  then every 120 ms. Stick deadzone 0.5.
- Keyboard fallback (arrows, Enter, Esc) for desktop testing.
- Write a small focus manager: on each move, pick the nearest focusable
  element in that direction by on-screen position. Do not rely on browser
  spatial navigation.
- WebView2 only receives gamepad input while its window has focus. TV mode
  must take focus at launch and again after every game.

### Launching a game from TV mode

- Reuse the existing pipeline unchanged: profile swap, launch, wait for
  exit, swap back and backup.
- Hide TV mode while the game runs. When the game exits, restore TV mode and
  give it focus.
- Games that open a store launcher first (EA app, Ubisoft Connect) need a
  mouse. Mark them in TV mode with a "may need mouse" badge for v1.

### 10-foot UI rules

- Design at 1920x1080 for a viewer about 3 m away, scaling with the
  viewport.
- Minimum text size 24 px at 1080p.
- The focused item always shows a visible focus ring and scales to 1.05.
- Keep 5% safe margins on every edge, since TVs crop the picture edge
  (overscan).
- No hover-only actions. Every screen works with D-pad, A and B alone.

### Exit test for Phase 1

In a manually installed Apollo, add an app whose command runs
`lootnest.exe --tv`. From the stock Moonlight client on the TV (Moonlight TV
on LG), a full loop must work with only a controller: pick a profile, launch
a game, play, quit, back in TV mode.

## Phase 2: Bundled streaming engine

Phase 2 ships Apollo inside the Lootnest installer and drives it from a new
Go package, `streamhost`, so the user never opens Apollo's own UI. Apollo is
chosen over Sunshine for v1 because its built-in virtual display (SudoVDA)
matches the TV's resolution automatically, which matters on multi-monitor
PCs.

### Engine choice and fallback

- Pin one tested Apollo release and check its SHA-256 before installing.
- Hide the engine behind a Go interface (`Engine`: Install, Start, Stop,
  Health, SetApp, SubmitPIN, ListClients, Logs), so Sunshine plus a separate
  virtual display driver can replace Apollo later. Apollo releases slowly
  and its author has said it will eventually drift away from stock Moonlight
  compatibility.
- Both engines expose the same local web API (Sunshine API docs), so most of
  `streamhost` is shared.

### Install and first run

1. The user turns on TV streaming. Lootnest runs the bundled engine
   installer elevated (one admin prompt). It also installs the virtual
   display driver.
2. Lootnest generates a random admin username and password and sets them
   with `POST /api/password`, which needs no login on first setup. It stores
   them in Windows Credential Manager.
3. Lootnest writes the engine config: host name "Lootnest, <PC name>",
   hardware encoder on auto, admin web UI reachable from this PC only.
4. Lootnest registers its single streamed app (table below) and confirms the
   engine answers.

### API calls Lootnest makes

Base URL `https://localhost:47990/api`, Basic auth with the stored
credentials. The certificate is self-signed: pin its fingerprint on first
contact instead of disabling TLS checks. Newer engine versions protect POST
and DELETE calls against CSRF; fetch a token with `GET /api/csrf-token` when
the pinned version requires it.

| Purpose | Call | When |
| --- | --- | --- |
| Set admin credentials | `POST /api/password` | First run |
| Register the Lootnest app | `POST /api/apps` (name Lootnest, cmd `"<install dir>\lootnest.exe" --tv`, cover PNG, virtual display on) | First run and after each update |
| Health check | `GET /api/apps` | Lootnest start, settings page |
| Complete pairing | `POST /api/pin` with `{"pin","name"}` | Adding a TV |
| List paired TVs | `GET /api/clients/list` | Settings page |
| End a stuck stream | `POST /api/apps/close` | User action |
| Restart engine | `POST /api/restart` | After config changes |
| Diagnostics | `GET /api/logs` | "Export diagnostics" button |

### Pairing

- In GameStream pairing the TV generates a 4-digit PIN and the host submits
  it.
- Phase 2 (stock Moonlight client): Lootnest shows "Enter the PIN shown on
  your TV" and submits it with `POST /api/pin`.
- Phases 3 and 4 (Lootnest for TV): Lootnest generates the PIN itself and
  passes it to the TV app at install or launch, then submits the same PIN,
  so nobody types anything.
- Apollo gives the first paired client full permissions and later clients
  only viewing and app-list rights. Lootnest must make sure every TV it
  pairs gets controller, keyboard and mouse input. How to set this through
  the API is an open question.

### Settings page in Lootnest

- Engine status: installed, running, version, detected encoder.
- Paired TVs, with a Remove button.
- Stream quality presets that map to recommended client settings: 1080p60
  at 30 Mbps, 4K60 at 60 Mbps.
- Export diagnostics (engine logs + Lootnest logs, credentials stripped).
- Turning TV streaming off stops the engine. Uninstalling Lootnest
  uninstalls it.

## TV platforms

Lootnest Link targets every TV through four native clients and one browser
client, all following the same client contract. Each native client is a
fork of an existing GPL v3 Moonlight port, so Lootnest never writes a video
decoder.

| TV platform | Typical brands | Lootnest client based on | How it gets on the TV | Phase |
| --- | --- | --- | --- | --- |
| LG webOS | LG | Moonlight TV | Developer mode, installed by Lootnest; later LG Content Store | 3 |
| Android TV / Google TV | Sony, TCL, Philips, Xiaomi, some Hisense, Chromecast with Google TV | Moonlight Android | Google Play, no developer mode | 4a |
| Fire TV | Fire TV sticks, Fire TV Edition TVs | Same Android build | Amazon Appstore | 4a |
| Samsung Tizen 5.5+ (2020 models and newer) | Samsung | Moonlight for Tizen (WebAssembly port) | Developer mode, installed by Lootnest; later Samsung store | 4b |
| Any other smart TV with a modern browser | Hisense VIDAA, Panasonic, Philips Saphi and others | Lootnest web client served by the PC over WebRTC | Open a URL, nothing to install | 4c |
| Apple TV (box) | Apple | Moonlight iOS/tvOS | App Store | Later |
| No smart platform, or no browser and no client (for example Roku) | | None | A cheap streaming stick or an HDMI cable | Out of scope |

### Rollout order

1. LG webOS first, because the development TV is an LG.
2. Android TV, Google TV and Fire TV next: the largest reach, and real app
   stores remove developer mode.
3. Samsung Tizen: the biggest TV brand, but sideloading needs developer
   mode and a Samsung certificate. Apps2Samsung shows the install can be
   automated from a PC.
4. Browser client last, as the catch-all for every remaining smart TV.

### Shared client contract

Every Lootnest for TV client, native or browser, must:

1. Launch straight into Lootnest: skip Moonlight's host and app lists and
   start the host app named Lootnest.
2. Wake the PC: if the PC is offline, send Wake-on-LAN to its saved MAC
   address and show "Waking your PC…" for up to 60 s.
3. Pair without typing: find Lootnest on the LAN over mDNS
   (`_lootnest._tcp`) and send its name and pairing PIN to Lootnest's LAN
   pairing endpoint. Lootnest shows "Allow <TV name>?" on the PC, then
   submits the PIN to the engine. When Lootnest installed the app itself
   (webOS, Tizen), it can pre-approve that TV.
4. Show controller-friendly error screens: PC unreachable, not paired,
   stream ended, each with Retry or Pair again.
5. Use Lootnest's quality presets (codec, resolution, bitrate) instead of
   its own settings screens.
6. Quit the stream when Start + Select are held for 2 s.
7. Report its version to Lootnest, so Lootnest can warn when a TV app is out
   of date.

## Phase 3: Lootnest for TV on LG webOS (reference client)

Phase 3 builds the first native Lootnest for TV client on LG webOS, the
platform of the development TV. It implements the shared client contract
(TV platforms section) once, end to end, so the other platforms in Phase 4
copy a proven design. The fork lives in its own public repository,
`lootnest-tv-webos`, because the upstream Moonlight TV is GPL v3.

### Changes to the fork

Keep every change in as few files as possible so upstream fixes can still
be merged.

1. Branding: app id `com.lootnest.tv`, name, icon, splash screen.
2. Direct launch: skip the host list. Read the saved PC (address, UUID,
   MAC). If the PC is offline, send a Wake-on-LAN packet and show "Waking
   your PC…" for up to 60 s. Then start the app named Lootnest directly.
3. Pairing from launch parameters: when Lootnest launches the app with
   `{"pairPin":"1234","host":"192.168.1.20"}`, the app pairs using that PIN.
   Lootnest submits the same PIN to the engine, so the user types nothing.
4. Error screens for PC unreachable, not paired and stream ended, each with
   Retry or Pair again, all usable with a controller.
5. Defaults from Lootnest's quality preset: HEVC, 1080p60 or 4K60, matching
   bitrate.
6. Quit stream: hold Start + Select for 2 s to return to the app's start
   screen.

### "Add a TV" wizard in Lootnest

1. Find LG TVs on the LAN (SSDP discovery), with manual IP entry as
   fallback.
2. Walk the user through the TV side: install the Developer Mode app from
   the LG Content Store, sign in with a free LG developer account, turn on
   Dev Mode (the TV reboots), then turn on Key Server and read out the
   passphrase on screen.
3. Download the TV's SSH key from `http://<TV_IP>:9991/webos_rsa`, unlock it
   with the passphrase, and connect over SSH on port 9922 as user
   `prisoner` (webOS Brew setup guide).
4. Upload the lootnest-tv IPK over SFTP and install it the way
   `ares-install` does. Implement this in Go (`golang.org/x/crypto/ssh` +
   `pkg/sftp`), using the open-source webOS Dev Manager and ares-cli code as
   the reference for the exact calls.
5. Launch the app with pairing parameters, then submit the PIN to the
   engine (Phase 2).
6. Save the TV in Lootnest's settings: name, IP, install date, developer
   mode status.

### Developer mode expiry

LG's developer mode session runs out after about 50 hours unless extended,
and the app can stop working when it lapses. Lootnest should extend the
session automatically while the PC is on, the way webOS Dev Manager does,
and warn the user when it could not. The long-term fix is publishing the
webOS client in the LG Content Store, which removes developer mode
entirely.

## Phase 4: Every other TV

Phase 4 brings the same contract to Android-based TVs, Samsung and, through
a browser client, every remaining smart TV. Each client reuses the Phase 3
design and Lootnest's existing PC-side pieces unchanged.

### 4a. Android TV, Google TV and Fire TV

- Fork Moonlight Android into `lootnest-tv-android`, package
  `com.lootnest.tv`. One codebase builds for Android TV, Google TV and Fire
  TV, and also covers Android boxes such as Nvidia Shield and Chromecast.
- Add what TV app stores require: a launcher banner, full D-pad navigation,
  no touch-only screens.
- Publish on Google Play (TV track) and the Amazon Appstore, so users
  install it like any other app.
- "Add a TV" on these platforms: Lootnest shows a QR code for the store
  listing, the user installs the app, then pairing runs through the LAN
  pairing endpoint.

### 4b. Samsung Tizen

- Fork Moonlight for Tizen, a WebAssembly port that needs Tizen 5.5 or newer
  (2020 models onward), into `lootnest-tv-tizen`.
- "Add a TV" walks the user through Samsung's developer mode, with the PC's
  IP entered on the TV. Lootnest then installs the signed app package
  (.wgt) over Samsung's Smart Development Bridge (sdb).
- Signing needs a Samsung certificate tied to the TV. Study how
  Apps2Samsung automates this before choosing between porting the steps to
  Go and bundling a helper tool.

### 4c. Browser client (any other smart TV)

- Lootnest serves a TV page from the PC, for example
  `http://lootnest.local:<port>/tv`, also shown as a QR code and a short IP
  address in Lootnest.
- A relay turns the GameStream stream into WebRTC for the browser.
  Moonlight-Web proves the approach: it embeds Moonlight's core library,
  relays to the browser over WebRTC and decodes with WebCodecs.
- Options for the relay, cheapest first: switch the engine to Vibepollo, an
  Apollo fork that added browser streaming at `/webrtc` in version 1.14.0;
  bundle Moonlight-Web as a second helper; or write a relay in Go with
  `pion/webrtc`. Check each license first.
- The TV page starts with a compatibility check (WebRTC, WebCodecs or a
  video fallback, Gamepad API). If the TV's browser fails it, the page says
  so and recommends a streaming stick instead of a broken stream.
- Expect a less smooth stream than native clients. Vibepollo's own notes say
  browser streaming is not intended to match native Moonlight.

### Later: Apple TV

Fork Moonlight for iOS/tvOS and publish it on the App Store. This needs a
paid Apple developer account, so it waits until there is demand.

## Security, licensing and distribution

The engine's admin API and the TV's SSH key are the two sensitive pieces.
Both stay on the PC, encrypted, and are never exposed beyond the local
network.

### Security

- The engine's admin web UI and API answer only from this PC. Sunshine's own
  docs warn never to expose port 47990 to the internet.
- Engine credentials and the TV SSH key are stored with Windows Credential
  Manager or DPAPI, never written to logs, and stripped from diagnostics
  exports.
- Pin the engine's self-signed certificate on first contact.
- Firewall rules for streaming apply to the Private network profile only.
- Lootnest itself runs without admin rights. Elevation happens only during
  engine install and update.
- The settings page lists every paired TV. Removing one revokes its pairing.

### Licensing (not legal advice: check before release)

- Apollo and Sunshine are GPL v3. Lootnest runs the engine as a separate
  program and talks to it over HTTP. Ship its license text and a link to the
  exact source of the pinned version in Lootnest's third-party notices.
- lootnest-tv is a fork of GPL v3 code, so it must be published as open
  source under GPL v3.
- Lootnest's own core and the planned one-time Supporter tier are not
  affected by either.
- Use the names Moonlight, Sunshine and Apollo only in attribution, never in
  Lootnest's branding.

### Distribution

- Download the engine when the user turns on TV streaming, not inside the
  base installer. This keeps the installer small. Fetch the pinned release
  from its official GitHub releases and check the SHA-256 hash in Lootnest's
  code.
- Build the lootnest-tv IPK in GitHub Actions and publish it as a GitHub
  release. Lootnest fetches it the same way, hash-pinned.
- Code-sign Lootnest. Expect some antivirus warnings because of the virtual
  display driver and controller input injection, and document the fix in
  the FAQ.

## Risks and open questions

The largest risks are outside Lootnest's code: the engine's upkeep, TV
developer modes and app store review. Each has a fallback, so none blocks
the project.

| Risk | Impact | Mitigation |
| --- | --- | --- |
| Apollo slows down or drifts from stock Moonlight compatibility | Engine updates stall | Engine interface; Sunshine + separate virtual display driver as fallback |
| Developer mode friction on LG and Samsung (expiry, certificates) | TV app stops working | Automatic renewal; store submission; Android TV first for store installs |
| LG, Samsung or Google Play rejects the app | No store install on that platform | Keep sideloading; browser client as catch-all |
| WebView2 loses focus and stops receiving controller input | Couch user is stuck | Focus manager re-focuses TV mode after every game; test on every release |
| Store launchers (EA app, Ubisoft Connect) need a mouse | Some games can't start from a controller | "May need mouse" badge; Moonlight's mouse mode |
| Slow TV network (Wi-Fi, 100 Mbps Ethernet ports on many TVs) | Stutter, blocky picture | Presets cap bitrate; network test in the TV app |
| Antivirus flags the virtual display driver or input injection | Install fails or is blocked | Code signing; FAQ entry |
| GPL obligations missed | Legal and reputation damage | Separate public repos for every fork; third-party notices in Lootnest |

### Open questions

- [ ] Which silent-install flags does the pinned Apollo installer accept?
- [ ] Can per-client input permissions in Apollo be set through its API, or
      only through its config file?
- [ ] Exactly which calls do ares-install and webOS Dev Manager make to
      install an app and renew developer mode? (Read their source.)
- [ ] How much of Samsung's certificate and sdb install flow can Lootnest
      automate?
- [ ] Which licenses do Moonlight-Web and Vibepollo use, and do they allow
      bundling?
- [ ] Do the LG and Samsung stores accept a GameStream client?
- [ ] Which TV browsers pass the 4c compatibility check? (Build a test
      matrix.)
- [ ] How is Lootnest's LAN pairing endpoint protected beyond the "Allow"
      prompt on the PC?

## Milestones

Build one milestone at a time, and only start the next once the current one
passes its test on a real TV. M1 alone already lets you play from the
couch.

| Milestone | Delivers | Rough size | Done when |
| --- | --- | --- | --- |
| M1 | TV mode (`--tv`) | ~1 week | Full loop on the LG TV with stock Apollo + Moonlight TV and only a controller: pick profile, launch, play, quit, back in TV mode |
| M2 | Bundled engine (`streamhost`) | 1 to 2 weeks | On a clean Windows PC, one toggle installs and configures the engine; stock Moonlight pairs from Lootnest's PIN screen; Apollo's own UI is never opened |
| M3 | webOS client + "Add a TV" | 2 to 4 weeks | A fresh LG TV in developer mode goes from Add a TV to playing with no typing beyond the dev-mode passphrase; Wake-on-LAN works |
| M4a | Android TV / Google TV / Fire TV client | 2 to 3 weeks | Installed from a store build, pairs through the Allow prompt, passes the same loop as M1 |
| M4b | Samsung Tizen client | 2 to 4 weeks | Same loop on a Samsung TV, installed by Lootnest |
| M4c | Browser client | 2 to 3 weeks | Compatibility check runs on at least 3 TV brands; same loop works on every TV that passes |

### Working on it with Claude Code

1. This file is the spec: `docs/lootnest-link-spec.md`.
2. One session per milestone. Each fork (`lootnest-tv-webos`, `-android`,
   `-tizen`) is its own repo with its own sessions.
3. Start each session with a prompt like this:

   > Read docs/lootnest-link-spec.md. Implement milestone M1 only. First
   > explore the existing frontend and launch pipeline, then propose a plan
   > that lists the files you will change. Wait for my approval before
   > writing code. Add tests where the code allows it. Stop when M1's "Done
   > when" criterion is met, and list anything in the spec that turned out
   > wrong or unclear.

4. Turn each open question into a GitHub issue, and answer it before the
   milestone that depends on it.
