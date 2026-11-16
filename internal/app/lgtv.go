//go:build windows

package app

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// Add a TV, for LG TVs (Lootnest Link M3, core/webos.go): Lootnest finds
// the TV, takes its developer-mode key with the passphrase the TV shows,
// installs Lootnest for TV on it over SSH, opens it with a PIN, submits
// the same PIN to the streaming engine, and lets the TV play. It keeps the
// TV's key (encrypted, DPAPI) in link.json to extend developer mode while
// the PC is on, since LG ends a session after about 50 hours.

type savedTV struct {
	Name      string    `json:"name"`
	IP        string    `json:"ip"`
	Key       string    `json:"key"`     // "dpapi:..." the developer-mode key, unlocked
	HostKey   string    `json:"hostKey"` // the TV's SSH identity, pinned
	Version   string    `json:"version,omitempty"`
	Installed time.Time `json:"installed,omitzero"`
	Extended  time.Time `json:"extended,omitzero"` // developer mode last extended
	Problem   string    `json:"problem,omitempty"`
	Stream    string    `json:"stream,omitempty"` // the last finished stream, as the TV app wrote it
	StreamAt  time.Time `json:"streamAt,omitzero"`
}

// LGStatus is a saved LG TV as Settings, TV shows it.
type LGStatus struct {
	Name      string    `json:"name"`
	IP        string    `json:"ip"`
	Version   string    `json:"version"`
	Latest    string    `json:"latest"` // the version this Lootnest has for it
	Installed time.Time `json:"installed"`
	Extended  time.Time `json:"extended"`
	Problem   string    `json:"problem,omitempty"`
	// The last finished stream: its numbers, what they point at
	// (core.Stream*) and that in plain words. Nil until one was read.
	Stream     *core.StreamStats `json:"stream,omitempty"`
	StreamKind string            `json:"streamKind,omitempty"`
	StreamSays string            `json:"streamSays,omitempty"`
	StreamAt   time.Time         `json:"streamAt,omitzero"`
}

func lgStatuses(r *linkRecord) []LGStatus {
	out := []LGStatus{}
	if r == nil {
		return out
	}
	for _, t := range r.TVs {
		st := LGStatus{Name: t.Name, IP: t.IP, Version: t.Version, Latest: core.TVAppPinned.Version, Installed: t.Installed, Extended: t.Extended, Problem: t.Problem}
		if _, stats, ok := core.LastStream(t.Stream); ok {
			st.Stream, st.StreamAt = &stats, t.StreamAt
			st.StreamKind, st.StreamSays = core.StreamVerdict(stats)
		}
		out = append(out, st)
	}
	return out
}

// apiLinkFindTVs looks for LG TVs and answers with a "linkTVs" event.
func (a *App) apiLinkFindTVs() {
	go func() {
		tvs, err := core.FindLGTVs(context.Background(), 3*time.Second)
		out := map[string]any{"tvs": tvs}
		if err != nil {
			out["error"] = sentence(err)
		}
		if tvs == nil {
			out["tvs"] = []core.LGTV{}
		}
		a.emit("linkTVs", out)
	}()
}

func randomPIN() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%04d", n.Int64()), nil
}

// apiLinkInstallTV installs Lootnest for TV on the LG TV at ip and pairs it.
func (a *App) apiLinkInstallTV(ip, name, passphrase string) {
	ip, name = strings.TrimSpace(ip), strings.TrimSpace(name)
	if name == "" {
		name = "LG TV"
	}
	if !claimLink("key") {
		a.emit("linkTVDone", map[string]any{"error": "Lootnest is still busy with the streaming engine or another TV (" + linkStep() + "). Try again when it's done."})
		return
	}
	go func() {
		a.logf("TV streaming: Add an LG TV on %s started", ip)
		err := a.installTV(ip, name, passphrase)
		setLinkStep("")
		if err != nil {
			a.logf("TV streaming: Lootnest for TV wasn't installed on %s: %v", ip, err)
			out := map[string]any{"error": sentence(err)}
			var pub *core.PublicNetworkError
			if errors.As(err, &pub) {
				out["public"] = true
			}
			a.emit("linkTVDone", out)
		} else {
			a.logf("TV streaming: Lootnest for TV installed on %q (%s) and paired", name, ip)
			a.emit("linkTVDone", map[string]any{"ok": true, "name": name})
		}
		a.emitLink()
	}()
}

func (a *App) installTV(ip, name, passphrase string) error {
	ctx := context.Background()
	ap, r, err := engine()
	if err != nil {
		return err
	}
	a.linkProgress("key", 0, 0)
	// On a network Windows calls Public, the engine's firewall rules keep
	// the TV out: say so before anything is installed.
	if pc := pcAddressFor(ip); pc != "" {
		cat := networkCategory(pc)
		a.logf("TV streaming: Add an LG TV on %s: this PC is %s on a network Windows calls %q", ip, pc, cat)
		if cat == "Public" {
			return &core.PublicNetworkError{PC: pc}
		}
	}
	// A TV added before: the key Lootnest kept for it, so installing again
	// needs nothing from the TV. A new TV, or a passphrase typed: the key
	// the TV hands out for it.
	var keyPEM []byte
	if passphrase == "" {
		if old := r.tv(ip); old != nil {
			keyPEM = []byte(revealSecret(old.Key))
		}
		if len(keyPEM) == 0 {
			return errors.New("Lootnest has no key for this TV yet. Type the passphrase Developer Mode shows")
		}
		a.logf("TV streaming: Add an LG TV on %s: with the key kept for it", ip)
	} else {
		a.logf("TV streaming: Add an LG TV on %s: unlocking the key", ip)
		k, err := core.FetchTVKey(ctx, ip, passphrase)
		if err != nil {
			return err
		}
		keyPEM = k
	}

	setLinkStep("tvdownload")
	a.logf("TV streaming: Add an LG TV on %s: the app package", ip)
	ipk := filepath.Join(DataDir(), "downloads", "tv", "lootnest-tv-"+core.TVAppPinned.Version+".ipk")
	if err := core.DownloadTVApp(ctx, core.TVAppPinned, ipk, func(done, total int64) { a.linkProgress("tvdownload", done, total) }); err != nil {
		return err
	}

	setLinkStep("copy")
	a.logf("TV streaming: Add an LG TV on %s: connecting", ip)
	a.linkProgress("copy", 0, 0)
	old := r.tv(ip)
	pinned := ""
	if old != nil {
		pinned = old.HostKey
	}
	tv, hostKey, err := core.ConnectTV(ctx, ip, keyPEM, pinned)
	if err != nil && pinned != "" && strings.Contains(err.Error(), "identity changed") {
		// The TV was reset or developer mode turned on again: the key the
		// user just unlocked on it is what's trusted now.
		tv, hostKey, err = core.ConnectTV(ctx, ip, keyPEM, "")
	}
	if err != nil {
		if passphrase == "" {
			// The kept key no longer opens the TV: developer mode was turned
			// on again, or the TV was reset.
			return fmt.Errorf("the key Lootnest kept for this TV no longer opens it (%v). Turn on Key Server in the TV's Developer Mode and type its passphrase", err)
		}
		return err
	}
	defer tv.Close()
	setLinkStep("tvinstall")
	a.logf("TV streaming: Add an LG TV on %s: installing", ip)
	a.linkProgress("tvinstall", 0, 0)
	if err := tv.InstallTVApp(ctx, ipk); err != nil {
		return err
	}

	// Saved now: the app is on the TV, whatever pairing does.
	enc, err := protectSecret(string(keyPEM))
	if err != nil {
		return err
	}
	saved := savedTV{Name: name, IP: ip, Key: enc, HostKey: hostKey, Version: core.TVAppPinned.Version, Installed: time.Now(), Extended: time.Now()}
	r.putTV(saved)
	if err := saveLink(r); err != nil {
		return err
	}

	setLinkStep("tvpair")
	a.logf("TV streaming: Add an LG TV on %s: pairing", ip)
	a.linkProgress("tvpair", 0, 0)
	pin, err := randomPIN()
	if err != nil {
		return err
	}
	pc := tv.LocalIP()
	if pc == "" {
		return errors.New("Lootnest couldn't tell the TV this PC's address")
	}
	before, err := ap.Clients(ctx)
	if err != nil {
		return err
	}
	// The request also goes in a file the TV app reads, for an app that's
	// already open (it gets no new launch parameters) or a TV that drops
	// them; removed once pairing is over either way.
	if err := tv.LeavePairing(fmt.Sprintf("%d", time.Now().UnixNano()), pc, pin); err != nil {
		a.logf("TV streaming: Add an LG TV on %s: the pairing request couldn't be left on the TV: %v", ip, err)
	}
	defer tv.ClearPairing()
	trace, err := tv.LaunchTVApp(map[string]string{"host": pc, "pairPin": pin})
	a.logf("TV streaming: Add an LG TV on %s: opening the app with this PC's address %s: %s", ip, pc, trace)
	if err != nil {
		return err
	}
	// The TV starts pairing a moment after it opens: the engine accepts
	// the PIN only then.
	paired := false
	for i, end := 0, time.Now().Add(75*time.Second); time.Now().Before(end); i++ {
		if i > 0 {
			time.Sleep(2 * time.Second)
		}
		ok, err := ap.Pair(ctx, pin, name)
		if err != nil {
			return err
		}
		if ok {
			paired = true
			break
		}
		// A TV paired before needs no PIN: its app goes straight on to
		// start Lootnest, as its own log says.
		if i%3 == 2 && core.TVAppStarted(tv.TVAppLog()) {
			a.logf("TV streaming: Add an LG TV on %s: the TV was already paired with this PC", ip)
			return nil
		}
	}
	if !paired {
		// Already paired before (the app was installed again) is fine.
		for _, c := range before {
			if c.Name == name {
				return nil
			}
		}
		// Why not: whether the app is open on the TV, and whether the TV
		// reaches the engine on this PC.
		running, known := tv.AppRunning()
		reach := tv.ReachesPC(pc)
		cat := networkCategory(pc)
		a.logf("TV streaming: Add an LG TV on %s: not paired; the app is on the TV's screen: %s; the TV reaches the engine at %s: %q; Windows calls the network %q", ip, map[bool]string{true: "yes", false: "no"}[running]+map[bool]string{true: "", false: " (unknown)"}[known], pc, reach, cat)
		if l := tv.TVAppLog(); l != "" {
			a.logf("TV streaming: the TV app's own log:\n%s", l)
		}
		switch {
		case cat == "Public":
			return &core.PublicNetworkError{PC: pc}
		case known && !running:
			return errors.New("the app is on the TV, but the TV didn't open it. Open Lootnest from the TV's home screen, then install again")
		case reach == "no":
			return errors.New("the TV can't reach the streaming engine on this PC. Check that they're on the same network and that no firewall on this PC blocks Apollo")
		}
		return errors.New("the app is on the TV, but it didn't pair with this PC. Check that the TV and this PC are on the same network, then install again")
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
		if !had[c.UUID] {
			return ap.AllowTV(ctx, c)
		}
	}
	return nil
}

func (r *linkRecord) tv(ip string) *savedTV {
	for i := range r.TVs {
		if r.TVs[i].IP == ip {
			return &r.TVs[i]
		}
	}
	return nil
}

func (r *linkRecord) putTV(t savedTV) {
	if old := r.tv(t.IP); old != nil {
		*old = t
		return
	}
	r.TVs = append(r.TVs, t)
}

// apiLinkForgetLG forgets a saved LG TV: Lootnest stops extending its
// developer mode and drops its key. The app stays on the TV.
func (a *App) apiLinkForgetLG(ip string) {
	go func() {
		if r := loadLink(); r != nil {
			out := r.TVs[:0]
			for _, t := range r.TVs {
				if t.IP != ip {
					out = append(out, t)
				}
			}
			r.TVs = out
			_ = saveLink(r)
		}
		a.emitLink()
	}()
}

// keepDevModes extends each saved LG TV's developer mode every 12 hours
// while Lootnest runs, when the TV is on; a TV that's off is tried again
// within the hour. A failure shows on the TV's row in Settings, TV.
func (a *App) keepDevModes() {
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		a.extendDevModes()
		a.updateTVApps()
		<-t.C
	}
}

var tvUpdating sync.Mutex

// updateTVApps puts the Lootnest for TV this Lootnest comes with on every
// saved TV that has an older one, with the key kept for it, so a new
// version needs nothing done at the TV. It runs at start, every hour and
// a while after TV mode ends; never while this PC streams to a TV, and
// never while the app is what's on the TV's screen (it would be closed
// under the person). It only installs: the app isn't opened.
func (a *App) updateTVApps() {
	a.mu.Lock()
	streaming := a.tv
	a.mu.Unlock()
	want := core.TVAppPinned
	if a.safe || streaming || len(want.SHA256) != 64 || !tvUpdating.TryLock() {
		return
	}
	defer tvUpdating.Unlock()
	r := loadLink()
	if r == nil || len(r.TVs) == 0 {
		return
	}
	changed := false
	for i := range r.TVs {
		t := &r.TVs[i]
		key := revealSecret(t.Key)
		if key == "" {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		func() {
			tv, _, err := core.ConnectTV(ctx, t.IP, []byte(key), t.HostKey)
			if err != nil {
				return // off or away: another time
			}
			defer tv.Close()
			have := tv.TVAppVersion()
			if have == "" {
				return
			}
			if have != t.Version {
				t.Version, changed = have, true
			}
			if !core.NewerVersion(have, want.Version) {
				return
			}
			if on, known := tv.AppRunning(); !known || on {
				a.logf("TV streaming: %s has Lootnest for TV %s and %s is ready; it's open on the TV, so it's left for later", t.Name, have, want.Version)
				return
			}
			if !claimLink("tvinstall") {
				return
			}
			defer setLinkStep("")
			ipk := filepath.Join(DataDir(), "downloads", "tv", "lootnest-tv-"+want.Version+".ipk")
			if err := core.DownloadTVApp(ctx, want, ipk, nil); err != nil {
				a.logf("TV streaming: Lootnest for TV %s couldn't be downloaded for %s: %v", want.Version, t.Name, err)
				return
			}
			if err := tv.InstallTVApp(ctx, ipk); err != nil {
				a.logf("TV streaming: Lootnest for TV %s couldn't be put on %s: %v", want.Version, t.Name, err)
				return
			}
			a.logf("TV streaming: Lootnest for TV on %s went from %s to %s", t.Name, have, want.Version)
			t.Version, t.Installed, changed = want.Version, time.Now(), true
			a.emit("linkTVUpdated", map[string]any{"name": t.Name, "version": want.Version})
		}()
		cancel()
	}
	if changed {
		// Only what this pass learned: the record may have changed meanwhile.
		if now := loadLink(); now != nil {
			for _, t := range r.TVs {
				if cur := now.tv(t.IP); cur != nil {
					cur.Version, cur.Installed = t.Version, t.Installed
				}
			}
			_ = saveLink(now)
		}
		a.emitLink()
	}
}

func (a *App) extendDevModes() {
	r := loadLink()
	if r == nil || len(r.TVs) == 0 {
		return
	}
	changed := false
	for i := range r.TVs {
		tvr := &r.TVs[i]
		if time.Since(tvr.Extended) < 12*time.Hour && tvr.Problem == "" {
			continue
		}
		key := revealSecret(tvr.Key)
		if key == "" {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := func() error {
			tv, _, err := core.ConnectTV(ctx, tvr.IP, []byte(key), tvr.HostKey)
			if err != nil {
				return err
			}
			defer tv.Close()
			tok, err := tv.DevModeToken()
			if err != nil {
				return err
			}
			return core.ExtendDevMode(ctx, tok)
		}()
		cancel()
		switch {
		case err == nil:
			tvr.Extended, tvr.Problem, changed = time.Now(), "", true
			a.logf("TV streaming: developer mode on %q extended", tvr.Name)
		case strings.Contains(err.Error(), "didn't answer"):
			// Off, or away: next hour.
		default:
			if tvr.Problem != sentence(err) {
				tvr.Problem, changed = sentence(err), true
				a.logf("TV streaming: developer mode on %q couldn't be extended: %v", tvr.Name, err)
			}
		}
	}
	if changed {
		_ = saveLink(r)
		a.emitLink()
	}
}

// pcAddressFor is this PC's address on the network that reaches the TV at
// ip, or "" when there's no route. Nothing is sent: a UDP socket only
// picks the route.
func pcAddressFor(ip string) string {
	c, err := net.Dial("udp", net.JoinHostPort(ip, "9"))
	if err != nil {
		return ""
	}
	defer c.Close()
	if u, ok := c.LocalAddr().(*net.UDPAddr); ok {
		return u.IP.String()
	}
	return ""
}

func powershell() string {
	return filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
}

// networkCategory is the Windows category of the network with this PC's
// address pc ("Public", "Private", "DomainAuthenticated"), or "" when
// Windows didn't say.
func networkCategory(pc string) string {
	script, err := core.NetworkCategoryScript(pc)
	if err != nil {
		return ""
	}
	out, err := win.RunHiddenOutput(powershell(), "-NoProfile -NonInteractive -ExecutionPolicy Bypass -EncodedCommand "+core.EncodePowerShell(script), 20*time.Second)
	if err != nil {
		return ""
	}
	return core.NetworkCategory(out)
}

// apiLinkMakePrivate makes the network that reaches the TV at ip Private,
// after Windows' prompt, and answers with a "linkPrivate" event.
func (a *App) apiLinkMakePrivate(ip string) {
	go func() {
		pc := pcAddressFor(strings.TrimSpace(ip))
		script, err := core.MakePrivateScript(pc)
		if err != nil {
			a.emit("linkPrivate", map[string]any{"error": "Lootnest couldn't tell which network reaches the TV."})
			return
		}
		code, err := win.RunElevatedAndWait(a.hwnd, powershell(), "-NoProfile -NonInteractive -ExecutionPolicy Bypass -WindowStyle Hidden -EncodedCommand "+core.EncodePowerShell(script))
		switch {
		case errors.Is(err, win.ErrCancelledByUser):
			a.emit("linkPrivate", map[string]any{"error": "Nothing changed: Windows' prompt was cancelled."})
		case err != nil || code != 0:
			a.logf("TV streaming: the network with %s wasn't made private (code %d, %v)", pc, code, err)
			a.emit("linkPrivate", map[string]any{"error": "Windows didn't change the network. In Windows Settings, Network and internet, open this network and choose Private."})
		default:
			a.logf("TV streaming: the network with %s is private now", pc)
			a.emit("linkPrivate", map[string]any{"ok": true})
		}
	}()
}

// noteStreams reads each saved TV's last finished stream from the TV app's
// log and keeps it: in Lootnest's log, so a report has the numbers even
// when the TV is off by then, and in link.json for Settings, TV. Called
// when TV mode ends, which is when the TV has just written it.
func (a *App) noteStreams() {
	r := loadLink()
	if r == nil || a.safe {
		return
	}
	found := map[string]string{}
	for _, t := range r.TVs {
		key := revealSecret(t.Key)
		if key == "" {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		tv, _, err := core.ConnectTV(ctx, t.IP, []byte(key), t.HostKey)
		cancel()
		if err != nil {
			continue
		}
		line, stats, ok := core.LastStream(tv.TVAppLog())
		wifi := tv.WiFi()
		tv.Close()
		if !ok || line == t.Stream {
			continue
		}
		found[t.IP] = line
		_, says := core.StreamVerdict(stats)
		a.logf("TV streaming: %s: %s", t.Name, line)
		if wifi != "" {
			a.logf("TV streaming: %s: its Wi-Fi link (quality, signal, noise): %s", t.Name, wifi)
		}
		a.logf("TV streaming: %s: %s", t.Name, says)
	}
	if len(found) == 0 {
		return
	}
	if now := loadLink(); now != nil {
		for i := range now.TVs {
			if line, ok := found[now.TVs[i].IP]; ok {
				now.TVs[i].Stream, now.TVs[i].StreamAt = line, time.Now()
			}
		}
		_ = saveLink(now)
	}
	a.emitLink()
}

// tvAppLog reads a saved TV's app log over SSH, or says why it couldn't.
func tvAppLog(t savedTV) string {
	key := revealSecret(t.Key)
	if key == "" {
		return "Lootnest has no key for this TV."
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	tv, _, err := core.ConnectTV(ctx, t.IP, []byte(key), t.HostKey)
	if err != nil {
		return "Couldn't reach the TV: " + err.Error()
	}
	defer tv.Close()
	head := ""
	if v := tv.WebOSVersion(); v != "" {
		head = "webOS: " + v + "\r\n"
	}
	if w := tv.WiFi(); w != "" {
		head += "Wi-Fi link (quality, signal, noise): " + w + "\r\n"
	}
	if l := tv.TVAppLog(); l != "" {
		return head + l
	}
	return head + "The TV app hasn't written a log yet."
}
