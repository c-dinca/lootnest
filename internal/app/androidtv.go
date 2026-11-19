package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lootnest/internal/core"
)

// Add an Android TV (Lootnest Link M4a, core/androidtv.go): for Android
// TV, Google TV and Fire TV. With network debugging on in the TV's
// Developer options, Lootnest connects over ADB with its own key (the TV
// asks once to allow this computer), installs Lootnest for TV, opens it
// with this PC's address and a PIN, and hands the same PIN to the
// streaming engine, as it does for LG TVs over SSH. Its key is kept
// encrypted (DPAPI) in link.json, so installing again needs nothing done
// at the TV while it still allows this computer.

type savedAndroid struct {
	Name      string    `json:"name"`
	IP        string    `json:"ip"`
	Model     string    `json:"model,omitempty"`
	Android   string    `json:"android,omitempty"`
	Version   string    `json:"version,omitempty"`
	Installed time.Time `json:"installed,omitzero"`
}

// AndroidStatus is a saved Android TV as Settings, TV shows it.
type AndroidStatus struct {
	Name      string    `json:"name"`
	IP        string    `json:"ip"`
	Model     string    `json:"model,omitempty"`
	Android   string    `json:"android,omitempty"`
	Version   string    `json:"version"`
	Latest    string    `json:"latest"`
	Installed time.Time `json:"installed"`
}

// adbTimeToAllow is how long Lootnest waits while the TV asks the person
// to allow this computer.
const adbTimeToAllow = 90 * time.Second

func androidStatuses(r *linkRecord) []AndroidStatus {
	out := []AndroidStatus{}
	if r == nil {
		return out
	}
	for _, t := range r.Androids {
		out = append(out, AndroidStatus{Name: t.Name, IP: t.IP, Model: t.Model, Android: t.Android, Version: t.Version, Latest: core.AndroidAppPinned.Version, Installed: t.Installed})
	}
	return out
}

func (r *linkRecord) android(ip string) *savedAndroid {
	for i := range r.Androids {
		if r.Androids[i].IP == ip {
			return &r.Androids[i]
		}
	}
	return nil
}

func (r *linkRecord) putAndroid(t savedAndroid) {
	if old := r.android(t.IP); old != nil {
		*old = t
		return
	}
	r.Androids = append(r.Androids, t)
}

// adbKey is Lootnest's key for Android TVs, made the first time.
func adbKey(r *linkRecord) ([]byte, error) {
	if k := revealSecret(r.ADBKey); k != "" {
		return []byte(k), nil
	}
	k, err := core.NewADBKey()
	if err != nil {
		return nil, err
	}
	enc, err := protectSecret(string(k))
	if err != nil {
		return nil, err
	}
	r.ADBKey = enc
	if err := saveLink(r); err != nil {
		return nil, err
	}
	return k, nil
}

// adbName is how the TV lists this computer when it asks to allow it.
func adbName() string {
	h, _ := os.Hostname()
	if h == "" {
		h = "this PC"
	}
	return "Lootnest@" + h
}

func (a *App) apiLinkFindAndroidTVs() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		ips := core.FindAndroidTVs(ctx)
		a.logf("TV streaming: %d address(es) on this network answer on ADB's port", len(ips))
		a.emit("linkAndroidFound", map[string]any{"ips": ips})
	}()
}

func (a *App) apiLinkInstallAndroidTV(ip string) {
	ip = strings.TrimSpace(ip)
	if !claimLink("adbconnect") {
		a.emit("linkAndroidDone", map[string]any{"error": "Lootnest is still busy with the streaming engine or another TV (" + linkStep() + "). Try again when it's done."})
		return
	}
	go func() {
		a.logf("TV streaming: Add an Android TV on %s started", ip)
		name, err := a.installAndroidTV(ip)
		setLinkStep("")
		if err != nil {
			a.logf("TV streaming: Lootnest for TV wasn't installed on the Android TV at %s: %v", ip, err)
			out := map[string]any{"error": sentence(err)}
			var pub *core.PublicNetworkError
			if errors.As(err, &pub) {
				out["public"] = true
			}
			if errors.Is(err, core.ErrADBNotAllowed) {
				out["allow"] = true
			}
			a.emit("linkAndroidDone", out)
		} else {
			a.logf("TV streaming: Lootnest for TV installed on %q (%s) and paired", name, ip)
			a.emit("linkAndroidDone", map[string]any{"ok": true, "name": name})
		}
		a.emitLink()
	}()
}

func (a *App) installAndroidTV(ip string) (string, error) {
	ctx := context.Background()
	addr, err := core.ADBAddress(ip)
	if err != nil {
		return "", err
	}
	host := strings.Split(addr, ":")[0]
	ap, r, err := engine()
	if err != nil {
		return "", err
	}
	if pc := pcAddressFor(host); pc != "" {
		cat := networkCategory(pc)
		a.logf("TV streaming: Add an Android TV on %s: this PC is %s on a network Windows calls %q", ip, pc, cat)
		if cat == "Public" {
			return "", &core.PublicNetworkError{PC: pc}
		}
	}
	key, err := adbKey(r)
	if err != nil {
		return "", err
	}

	// The TV asks "Allow USB debugging?" the first time: the drawer says
	// so while Lootnest waits.
	setLinkStep("adbconnect")
	a.linkProgress("adbconnect", 0, 0)
	a.logf("TV streaming: Add an Android TV on %s: connecting; the TV may ask to allow this computer", addr)
	tv, err := core.DialADB(ctx, addr, key, adbName(), adbTimeToAllow)
	if err != nil {
		return "", err
	}
	defer tv.Close()
	dev := tv.Device()
	name := dev.Label()
	a.logf("TV streaming: Add an Android TV on %s: %s %s, Android %s, named %q", addr, dev.Maker, dev.Model, dev.Android, name)

	have := tv.AndroidAppVersion()
	if have != core.AndroidAppPinned.Version {
		setLinkStep("tvdownload")
		a.logf("TV streaming: Add an Android TV on %s: the app package (the TV has %q)", addr, have)
		apk := filepath.Join(DataDir(), "downloads", "tv", "lootnest-tv-android-"+core.AndroidAppPinned.Version+".apk")
		if err := core.DownloadAndroidApp(ctx, core.AndroidAppPinned, apk, func(done, total int64) { a.linkProgress("tvdownload", done, total) }); err != nil {
			return "", err
		}
		setLinkStep("tvinstall")
		a.linkProgress("tvinstall", 0, 0)
		a.logf("TV streaming: Add an Android TV on %s: installing", addr)
		if err := tv.InstallAndroidApp(apk); err != nil {
			return "", err
		}
	}
	saved := savedAndroid{Name: name, IP: host, Model: strings.TrimSpace(dev.Maker + " " + dev.Model), Android: dev.Android, Version: core.AndroidAppPinned.Version, Installed: time.Now()}
	r.putAndroid(saved)
	if err := saveLink(r); err != nil {
		return "", err
	}

	setLinkStep("tvpair")
	a.linkProgress("tvpair", 0, 0)
	pin, err := randomPIN()
	if err != nil {
		return "", err
	}
	pc := tv.LocalIP()
	before, err := ap.Clients(ctx)
	if err != nil {
		return "", err
	}
	trace, err := tv.LaunchAndroidApp(pc, pin)
	a.logf("TV streaming: Add an Android TV on %s: opening the app with this PC's address %s: %s", addr, pc, trace)
	if err != nil {
		return "", err
	}
	paired := false
	for i, end := 0, time.Now().Add(75*time.Second); time.Now().Before(end); i++ {
		if i > 0 {
			time.Sleep(2 * time.Second)
		}
		ok, err := ap.Pair(ctx, pin, name)
		if err != nil {
			return "", err
		}
		if ok {
			paired = true
			break
		}
		// A TV paired before needs no PIN: its app goes straight on.
		if i%3 == 2 && core.AndroidAppStarted(tv.AndroidAppLog()) {
			a.logf("TV streaming: Add an Android TV on %s: the TV was already paired with this PC", addr)
			return name, nil
		}
	}
	if !paired {
		for _, c := range before {
			if c.Name == name {
				return name, nil
			}
		}
		cat := networkCategory(pc)
		a.logf("TV streaming: Add an Android TV on %s: not paired; Windows calls the network %q", addr, cat)
		if l := tv.AndroidAppLog(); l != "" {
			a.logf("TV streaming: the TV app's own log:\n%s", l)
		}
		if cat == "Public" {
			return "", &core.PublicNetworkError{PC: pc}
		}
		return "", errors.New("the app is on the TV, but it didn't pair with this PC. Check that the TV and this PC are on the same network, then install again")
	}
	after, err := ap.Clients(ctx)
	if err != nil {
		return "", err
	}
	had := map[string]bool{}
	for _, c := range before {
		had[c.UUID] = true
	}
	for _, c := range after {
		if !had[c.UUID] {
			return name, ap.AllowTV(ctx, c)
		}
	}
	return name, nil
}

// apiLinkForgetAndroid forgets a saved Android TV. The app stays on the
// TV, and the TV keeps allowing this computer until it's told not to in
// its Developer options.
func (a *App) apiLinkForgetAndroid(ip string) {
	go func() {
		if r := loadLink(); r != nil {
			out := r.Androids[:0]
			for _, t := range r.Androids {
				if t.IP != ip {
					out = append(out, t)
				}
			}
			r.Androids = out
			_ = saveLink(r)
		}
		a.emitLink()
	}()
}

// androidAppLog is a saved Android TV's app log, for Save a report.
func androidAppLog(t savedAndroid) string {
	r := loadLink()
	if r == nil {
		return ""
	}
	key := revealSecret(r.ADBKey)
	if key == "" {
		return ""
	}
	addr, err := core.ADBAddress(t.IP)
	if err != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Never waits for the TV to ask: a TV that no longer allows this
	// computer just isn't read.
	tv, err := core.DialADB(ctx, addr, []byte(key), adbName(), 2*time.Second)
	if err != nil {
		return fmt.Sprintf("(not read: %v)", err)
	}
	defer tv.Close()
	return tv.AndroidAppLog()
}
