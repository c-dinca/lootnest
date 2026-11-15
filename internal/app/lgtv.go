//go:build windows

package app

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"path/filepath"
	"strings"
	"time"

	"lootnest/internal/core"
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
}

// LGStatus is a saved LG TV as Settings, TV shows it.
type LGStatus struct {
	Name      string    `json:"name"`
	IP        string    `json:"ip"`
	Version   string    `json:"version"`
	Installed time.Time `json:"installed"`
	Extended  time.Time `json:"extended"`
	Problem   string    `json:"problem,omitempty"`
}

func lgStatuses(r *linkRecord) []LGStatus {
	out := []LGStatus{}
	if r == nil {
		return out
	}
	for _, t := range r.TVs {
		out = append(out, LGStatus{Name: t.Name, IP: t.IP, Version: t.Version, Installed: t.Installed, Extended: t.Extended, Problem: t.Problem})
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
		return
	}
	go func() {
		err := a.installTV(ip, name, passphrase)
		setLinkStep("")
		if err != nil {
			a.logf("TV streaming: Lootnest for TV wasn't installed on %s: %v", ip, err)
			a.emit("linkTVDone", map[string]any{"error": sentence(err)})
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
	keyPEM, err := core.FetchTVKey(ctx, ip, passphrase)
	if err != nil {
		return err
	}

	setLinkStep("download")
	ipk := filepath.Join(DataDir(), "downloads", "tv", "lootnest-tv-"+core.TVAppPinned.Version+".ipk")
	if err := core.DownloadTVApp(ctx, core.TVAppPinned, ipk, func(done, total int64) { a.linkProgress("download", done, total) }); err != nil {
		return err
	}

	setLinkStep("copy")
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
		return err
	}
	defer tv.Close()
	setLinkStep("install")
	a.linkProgress("install", 0, 0)
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

	setLinkStep("pair")
	a.linkProgress("pair", 0, 0)
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
	if err := tv.LaunchTVApp(map[string]string{"host": pc, "pairPin": pin}); err != nil {
		return err
	}
	// The TV starts pairing a moment after it opens: the engine accepts
	// the PIN only then.
	paired := false
	for end := time.Now().Add(75 * time.Second); time.Now().Before(end); time.Sleep(2 * time.Second) {
		ok, err := ap.Pair(ctx, pin, name)
		if err != nil {
			return err
		}
		if ok {
			paired = true
			break
		}
	}
	if !paired {
		// Already paired before (the app was installed again) is fine.
		for _, c := range before {
			if c.Name == name {
				return nil
			}
		}
		return errors.New("the app is on the TV, but it didn't pair. Open Lootnest on the TV, then use Add a TV with the PIN it shows")
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
		<-t.C
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
