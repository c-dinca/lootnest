//go:build windows

package app

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// Lootnest Link M2 (docs/lootnest-link-spec.md): Lootnest installs the
// streaming engine (Apollo, core/apollo.go), signs it in, names it,
// registers TV mode as its one app and pairs TVs, so the user never opens
// the engine's own pages. Windows asks once, to install it. What Lootnest
// keeps is link.json in its data folder: the engine's user, its password
// encrypted for this Windows user (DPAPI) and its certificate's
// fingerprint. None of it goes to the page.

type linkRecord struct {
	User      string    `json:"user"`
	Pass      string    `json:"pass"` // "dpapi:..."
	Cert      string    `json:"cert,omitempty"`
	Version   string    `json:"version,omitempty"`
	Installed time.Time `json:"installed,omitzero"`
	Off       bool      `json:"off,omitempty"` // turned off: the service is stopped
	TVs       []savedTV `json:"tvs,omitempty"` // LG TVs Lootnest installed its app on (lgtv.go)
}

var (
	linkMu   sync.Mutex
	linkBusy string // the step running now, "" when idle
)

func linkPath() string { return filepath.Join(DataDir(), "link.json") }

func loadLink() *linkRecord {
	b, err := os.ReadFile(linkPath())
	if err != nil {
		return nil
	}
	var r linkRecord
	if json.Unmarshal(b, &r) != nil || r.User == "" {
		return nil
	}
	return &r
}

func saveLink(r *linkRecord) error {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	tmp := linkPath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, linkPath())
}

func apolloDir() string { return filepath.Join(win.ProgramFiles(), "Apollo") }

func apolloInstalled() bool {
	_, err := os.Stat(filepath.Join(apolloDir(), "sunshine.exe"))
	return err == nil
}

// engine is a signed-in client for the engine, from link.json.
func engine() (*core.Apollo, *linkRecord, error) {
	r := loadLink()
	if r == nil {
		return nil, nil, errors.New("TV streaming isn't set up")
	}
	pass := revealSecret(r.Pass)
	if pass == "" {
		return nil, r, errors.New("Lootnest can't read the streaming engine's password on this Windows account. Set TV streaming up again")
	}
	ap, err := core.NewApollo(core.ApolloAPI, r.User, pass, r.Cert)
	return ap, r, err
}

// randomSecret is letters and digits only, so it passes through a command
// line untouched.
func randomSecret(n int) (string, error) {
	const abc = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	out := make([]byte, n)
	for i := range out {
		k, err := rand.Int(rand.Reader, big.NewInt(int64(len(abc))))
		if err != nil {
			return "", err
		}
		out[i] = abc[k.Int64()]
	}
	return string(out), nil
}

// LinkStatus is what Settings, TV shows.
type LinkStatus struct {
	Pinned    bool                `json:"pinned"` // this Lootnest can install the engine
	Version   string              `json:"version"`
	Installed bool                `json:"installed"`
	Set       bool                `json:"set"` // Lootnest set it up
	Off       bool                `json:"off"`
	Running   bool                `json:"running"`
	Busy      string              `json:"busy,omitempty"`
	Info      *core.ApolloInfo    `json:"info,omitempty"`
	TVs       []core.ApolloClient `json:"tvs"`
	LG        []LGStatus          `json:"lg"`    // LG TVs with Lootnest for TV
	TVApp     bool                `json:"tvApp"` // this Lootnest can install the TV app
	Problem   string              `json:"problem,omitempty"`
}

func linkStep() string {
	linkMu.Lock()
	defer linkMu.Unlock()
	return linkBusy
}

// apiLinkStatus answers with a "link" event.
func (a *App) apiLinkStatus() {
	go a.emitLink()
}

func (a *App) emitLink() {
	st := LinkStatus{Pinned: len(core.ApolloPinned.SHA256) == 64, Version: core.ApolloPinned.Version, Installed: apolloInstalled(), Busy: linkStep(), TVs: []core.ApolloClient{}}
	_, st.Running = win.ServiceState(core.ApolloService)
	r := loadLink()
	st.Set = r != nil
	st.Off = r != nil && r.Off
	st.LG, st.TVApp = lgStatuses(r), len(core.TVAppPinned.SHA256) == 64
	if r != nil && st.Running && st.Busy == "" {
		if ap, _, err := engine(); err != nil {
			st.Problem = sentence(err)
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			info, err := ap.Info(ctx)
			if err == nil {
				st.Info = &info
				if tvs, err := ap.Clients(ctx); err == nil {
					st.TVs = tvs
				}
			} else {
				st.Problem = sentence(err)
			}
			cancel()
		}
	}
	a.emit("link", map[string]any{"status": st})
}

func (a *App) linkProgress(step string, done, total int64) {
	a.emit("linkStep", map[string]any{"step": step, "done": done, "total": total})
}

// claimLink marks the engine busy with a step, once at a time.
func claimLink(step string) bool {
	linkMu.Lock()
	defer linkMu.Unlock()
	if linkBusy != "" {
		return false
	}
	linkBusy = step
	return true
}

func setLinkStep(step string) {
	linkMu.Lock()
	linkBusy = step
	linkMu.Unlock()
}

// apiLinkSetup turns TV streaming on: download the pinned engine, install
// it with one Windows prompt, sign it in, name it and add TV mode as its
// app. An engine already installed and signed in by Lootnest is only set
// again; one installed some other way gets Lootnest's sign-in.
func (a *App) apiLinkSetup() {
	if !claimLink("download") {
		return
	}
	go func() {
		err := a.linkSetup()
		setLinkStep("")
		if err != nil {
			a.logf("TV streaming: set-up stopped: %v", err)
			a.emit("linkDone", map[string]any{"error": sentence(err)})
		} else {
			a.logf("TV streaming: set up (Apollo %s)", core.ApolloPinned.Version)
			a.emit("linkDone", map[string]any{"ok": true})
		}
		a.emitLink()
	}()
}

func (a *App) linkSetup() error {
	ctx := context.Background()
	r := loadLink()
	// Set up before by Lootnest, still there and answering: only the
	// settings and the app again.
	if r != nil && apolloInstalled() {
		if ap, _, err := engine(); err == nil && reachSoon(ctx, ap, 5*time.Second) == nil {
			if _, _, err := ap.Apps(ctx); err == nil { // signed in
				setLinkStep("configure")
				return a.configureEngine(ctx, r, ap)
			}
		}
	}

	installer := ""
	if !apolloInstalled() {
		installer = filepath.Join(DataDir(), "downloads", "apollo", "Apollo-"+core.ApolloPinned.Version+".exe")
		a.linkProgress("download", 0, 0)
		if err := core.DownloadApollo(ctx, core.ApolloPinned, installer, func(done, total int64) { a.linkProgress("download", done, total) }); err != nil {
			return err
		}
	}

	// Lootnest's own sign-in for the engine, saved before the engine has
	// it, so a set-up cut short can be finished.
	pass, err := randomSecret(28)
	if err != nil {
		return err
	}
	enc, err := protectSecret(pass)
	if err != nil {
		return err
	}
	var tvs []savedTV
	if r != nil {
		tvs = r.TVs // the LG TVs stay
	}
	r = &linkRecord{User: "lootnest", Pass: enc, Version: core.ApolloPinned.Version, Installed: time.Now(), TVs: tvs}
	if err := saveLink(r); err != nil {
		return err
	}
	creds := filepath.Join(os.TempDir(), "lootnest-engine-"+fmt.Sprint(time.Now().UnixNano())+".json")
	b, _ := json.Marshal(map[string]string{"user": r.User, "pass": pass})
	if err := os.WriteFile(creds, b, 0o600); err != nil {
		return err
	}
	defer os.Remove(creds)

	setLinkStep("install")
	a.linkProgress("install", 0, 0)
	ps := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	script := core.ApolloInstallScript(installer, apolloDir(), creds)
	code, err := win.RunElevatedAndWait(a.hwnd, ps, "-NoProfile -NonInteractive -ExecutionPolicy Bypass -WindowStyle Hidden -EncodedCommand "+core.EncodePowerShell(script))
	switch {
	case errors.Is(err, win.ErrCancelledByUser):
		return errors.New("nothing was installed: Windows' prompt was cancelled")
	case err != nil:
		return fmt.Errorf("the streaming engine couldn't be installed: %w", err)
	case code == 10:
		return errors.New("the streaming engine's installer stopped with an error")
	case code == 11:
		return errors.New("the streaming engine wasn't where its installer puts it")
	case code == 12:
		return errors.New("the streaming engine didn't take Lootnest's sign-in")
	case code != 0:
		return fmt.Errorf("installing the streaming engine stopped (code %d)", code)
	}

	setLinkStep("start")
	a.linkProgress("start", 0, 0)
	ap, _, err := engine()
	if err != nil {
		return err
	}
	if err := reachSoon(ctx, ap, 90*time.Second); err != nil {
		return errors.New("the streaming engine was installed but didn't start. Restart the PC and set it up again")
	}
	setLinkStep("configure")
	return a.configureEngine(ctx, r, ap)
}

// reachSoon waits for the engine to answer.
func reachSoon(ctx context.Context, ap *core.Apollo, wait time.Duration) error {
	end := time.Now().Add(wait)
	for {
		err := ap.Reachable(ctx)
		if err == nil || errors.Is(err, core.ErrApolloCert) || time.Now().After(end) {
			return err
		}
		time.Sleep(2 * time.Second)
	}
}

// configureEngine names the engine, keeps its pages to this PC, adds or
// updates TV mode as its app, pins its certificate and restarts it.
func (a *App) configureEngine(ctx context.Context, r *linkRecord, ap *core.Apollo) error {
	a.linkProgress("configure", 0, 0)
	if err := ap.SetConfig(ctx, core.ApolloSettings(os.Getenv("COMPUTERNAME"))); err != nil {
		return err
	}
	if err := a.registerApp(ctx, ap); err != nil {
		return err
	}
	r.Cert, r.Off = ap.CertPin(), false
	if err := saveLink(r); err != nil {
		return err
	}
	if err := ap.Restart(ctx); err != nil {
		return err
	}
	time.Sleep(3 * time.Second)
	return reachSoon(ctx, ap, 60*time.Second)
}

// registerApp adds TV mode as the engine's app, or points the one there at
// this Lootnest.exe.
func (a *App) registerApp(ctx context.Context, ap *core.Apollo) error {
	apps, _, err := ap.Apps(ctx)
	if err != nil {
		return err
	}
	uuid := ""
	if app := core.LootnestAppIn(apps); app != nil {
		uuid, _ = app["uuid"].(string)
	}
	return ap.SaveApp(ctx, core.LootnestApp(a.exePath, "", uuid))
}

// linkAtStart keeps TV mode's app pointing at this Lootnest.exe after an
// update or a move, once the engine answers.
func (a *App) linkAtStart() {
	go a.keepDevModes()
	r := loadLink()
	if r == nil || r.Off || !apolloInstalled() {
		return
	}
	ap, _, err := engine()
	if err != nil {
		return
	}
	ctx := context.Background()
	if reachSoon(ctx, ap, 2*time.Minute) != nil {
		return
	}
	apps, current, err := ap.Apps(ctx)
	if err != nil {
		a.logf("TV streaming: the engine didn't list its apps: %v", err)
		return
	}
	// Settings a newer Lootnest adds (the TV's display as the main one)
	// reach an engine set up before, once no stream runs.
	if cfg, err := ap.Config(ctx); err == nil && current == "" {
		if want := core.ApolloSettings(os.Getenv("COMPUTERNAME")); core.ApolloSettingsDiffer(cfg, want) {
			if err := ap.SetConfig(ctx, want); err != nil {
				a.logf("TV streaming: the engine's settings couldn't be brought up to date: %v", err)
			} else if err := ap.Restart(ctx); err != nil {
				a.logf("TV streaming: the engine didn't restart with its new settings: %v", err)
			} else {
				a.logf("TV streaming: the engine's settings are up to date; the TV's display is the main one while it streams")
				time.Sleep(3 * time.Second)
				if reachSoon(ctx, ap, 60*time.Second) != nil {
					return
				}
			}
		}
	}
	app := core.LootnestAppIn(apps)
	if want := core.LootnestApp(a.exePath, "", ""); app == nil || app["cmd"] != want["cmd"] {
		if err := a.registerApp(ctx, ap); err != nil {
			a.logf("TV streaming: couldn't update TV mode's app: %v", err)
		} else {
			a.logf("TV streaming: TV mode's app now starts %s", a.exePath)
		}
	}
}

// apiLinkPair pairs the TV showing this PIN, names it, and lets it play.
func (a *App) apiLinkPair(pin, name string) {
	pin, name = strings.TrimSpace(pin), strings.TrimSpace(name)
	if name == "" {
		name = "TV"
	}
	if !claimLink("pair") {
		return
	}
	go func() {
		err := func() error {
			ap, _, err := engine()
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			before, err := ap.Clients(ctx)
			if err != nil {
				return err
			}
			ok, err := ap.Pair(ctx, pin, name)
			if err != nil {
				return err
			}
			if !ok {
				return errors.New("no TV was waiting with that PIN. On the TV, choose this PC in Moonlight and enter the PIN it shows now")
			}
			after, err := ap.Clients(ctx)
			if err != nil {
				return err
			}
			had := map[string]bool{}
			for _, c := range before {
				had[c.UUID] = true
			}
			for _, c := range after {
				if !had[c.UUID] || (c.Name == name && c.Perm != core.ApolloTVPerm) {
					return ap.AllowTV(ctx, c)
				}
			}
			return nil
		}()
		setLinkStep("")
		if err != nil {
			a.emit("linkPaired", map[string]any{"error": sentence(err)})
		} else {
			a.logf("TV streaming: paired %q", name)
			a.emit("linkPaired", map[string]any{"ok": true, "name": name})
		}
		a.emitLink()
	}()
}

// apiLinkUnpair forgets a TV.
func (a *App) apiLinkUnpair(uuid string) {
	go a.linkCall(func(ctx context.Context, ap *core.Apollo) error { return ap.Unpair(ctx, uuid) }, "Forgot the TV.")
}

// apiLinkEnd ends a stream that's stuck.
func (a *App) apiLinkEnd() {
	go a.linkCall(func(ctx context.Context, ap *core.Apollo) error { return ap.CloseApp(ctx) }, "The stream ended.")
}

func (a *App) linkCall(f func(context.Context, *core.Apollo) error, done string) {
	ap, _, err := engine()
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		err = f(ctx, ap)
		cancel()
	}
	if err != nil {
		a.emit("toast", map[string]any{"kind": "error", "text": sentence(err)})
	} else {
		a.emit("toast", map[string]any{"text": done})
	}
	a.emitLink()
}

// apiLinkPower turns the engine's service on or off (Windows asks first).
func (a *App) apiLinkPower(on bool) {
	if !claimLink("power") {
		return
	}
	go func() {
		err := func() error {
			r := loadLink()
			if r == nil {
				return errors.New("TV streaming isn't set up")
			}
			ps := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
			code, err := win.RunElevatedAndWait(a.hwnd, ps, "-NoProfile -NonInteractive -ExecutionPolicy Bypass -WindowStyle Hidden -EncodedCommand "+core.EncodePowerShell(core.ApolloServiceScript(on)))
			switch {
			case errors.Is(err, win.ErrCancelledByUser):
				return errors.New("nothing changed: Windows' prompt was cancelled")
			case err != nil:
				return err
			case code != 0:
				return fmt.Errorf("the streaming engine's service didn't change (code %d)", code)
			}
			r.Off = !on
			return saveLink(r)
		}()
		setLinkStep("")
		if err != nil {
			a.emit("toast", map[string]any{"kind": "error", "text": sentence(err)})
		}
		a.emitLink()
	}()
}

// apiLinkLogs saves the engine's log and Lootnest's for a report, with the
// engine's password taken out, and shows the file in Explorer.
func (a *App) apiLinkLogs() {
	go func() {
		var b strings.Builder
		ap, r, err := engine()
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			var logs string
			logs, err = ap.Logs(ctx)
			cancel()
			b.WriteString("== Streaming engine (Apollo) ==\r\n" + logs + "\r\n")
		}
		if err != nil {
			b.WriteString("== Streaming engine (Apollo) ==\r\nCouldn't read its log: " + err.Error() + "\r\n")
		}
		if lb, err := os.ReadFile(filepath.Join(DataDir(), "lootnest.log")); err == nil {
			b.WriteString("\r\n== Lootnest ==\r\n" + string(lb))
		}
		out := b.String()
		if r != nil {
			if p := revealSecret(r.Pass); p != "" {
				out = strings.ReplaceAll(out, p, "[password]")
			}
		}
		dst := filepath.Join(DataDir(), "tv-diagnostics.txt")
		if err := os.WriteFile(dst, []byte(out), 0o600); err != nil {
			a.emit("toast", map[string]any{"kind": "error", "text": sentence(err)})
			return
		}
		_ = win.RevealInExplorer(dst)
	}()
}
