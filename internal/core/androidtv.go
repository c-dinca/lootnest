package core

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Lootnest for TV on Android TV, Google TV and Fire TV (Lootnest Link
// M4a). The app is c-dinca/lootnest-tv-android, a fork of Moonlight
// Android (GPL v3) whose own screen pairs from what Lootnest passes it and
// opens the host app named Lootnest. Lootnest installs it over ADB
// (adb.go), as it does over SSH on LG TVs.
// ---------------------------------------------------------------------------

// AndroidTVPackage is the app's package name.
const AndroidTVPackage = "com.lootnest.tv"

// androidTVActivity is the app's own screen, the one Lootnest opens.
const androidTVActivity = AndroidTVPackage + "/com.limelight.LootnestActivity"

// AndroidTVLogTag is what the app's log lines are tagged with.
const AndroidTVLogTag = "LootnestTV"

// AndroidAppPinned is the package Lootnest installs on Android TVs. A new
// release of c-dinca/lootnest-tv-android is tried with Lootnest first,
// then its address and SHA-256 change here.
var AndroidAppPinned = TVAppRelease{
	Version: "1.0.1",
	URL:     androidAppReleases + "v1.0.1/com.lootnest.tv_1.0.1.apk",
	SHA256:  "171f5d1ec78b07900e6ab55079faf60fcb5a2181d9fbfc06bc495ce73024c4fb",
}

const androidAppReleases = "https://github.com/c-dinca/lootnest-tv-android/releases/download/"

// AndroidAppReady says whether this Lootnest can install the app.
func AndroidAppReady() bool {
	return len(AndroidAppPinned.SHA256) == 64 && strings.HasPrefix(AndroidAppPinned.URL, androidAppReleases)
}

// DownloadAndroidApp fetches the pinned package into dst, checked against
// its SHA-256.
func DownloadAndroidApp(ctx context.Context, rel TVAppRelease, dst string, progress func(done, total int64)) error {
	if len(rel.SHA256) != 64 || !strings.HasPrefix(rel.URL, androidAppReleases) {
		return ErrTVAppNotPinned
	}
	if ok, _ := fileSHA256Is(dst, rel.SHA256); ok {
		return nil
	}
	return DownloadApollo(ctx, ApolloRelease{Version: rel.Version, URL: rel.URL, SHA256: rel.SHA256}, dst, progress)
}

// FindAndroidTVs returns the addresses on this PC's networks that answer
// on ADB's port. Nothing is asked of them: connecting would make the TV
// ask to allow this computer.
func FindAndroidTVs(ctx context.Context) []string {
	return probeLGPort(ctx, localSubnets(), ADBPort, 700*time.Millisecond)
}

// ADBAddress is ip with ADB's port, unless it names one.
func ADBAddress(ip string) (string, error) {
	ip = strings.TrimSpace(ip)
	if _, _, err := net.SplitHostPort(ip); err == nil {
		return ip, nil
	}
	if net.ParseIP(ip) == nil {
		return "", errors.New("that isn't a TV's address: type it as four numbers, such as 192.168.1.40")
	}
	return net.JoinHostPort(ip, fmt.Sprint(ADBPort)), nil
}

// InstallAndroidApp copies the package at apk to the TV and installs it
// over what's there, keeping the app's data.
func (a *ADB) InstallAndroidApp(apk string) error {
	f, err := os.Open(apk)
	if err != nil {
		return err
	}
	defer f.Close()
	const tmp = "/data/local/tmp/lootnest-tv.apk"
	if err := a.Push(f, tmp, 0o644, 5*time.Minute); err != nil {
		return err
	}
	defer a.Shell("rm -f "+tmp, 15*time.Second)
	out, err := a.Shell("pm install -r "+tmp+" 2>&1", 3*time.Minute)
	if err != nil {
		return err
	}
	return InstallAnswer(out)
}

var pmFailure = regexp.MustCompile(`Failure \[([A-Z_]+)[^\]]*\]`)

// InstallAnswer reads what `pm install` printed.
func InstallAnswer(out string) error {
	if strings.Contains(out, "Success") {
		return nil
	}
	m := pmFailure.FindStringSubmatch(out)
	if m == nil {
		out = strings.TrimSpace(out)
		if out == "" {
			return errors.New("the TV didn't say whether it installed the app")
		}
		return fmt.Errorf("the TV didn't install the app: %s", out)
	}
	switch m[1] {
	case "INSTALL_FAILED_UPDATE_INCOMPATIBLE":
		return errors.New("the TV has a Lootnest for TV signed by someone else. Uninstall it on the TV, then try again")
	case "INSTALL_FAILED_VERSION_DOWNGRADE":
		return errors.New("the TV has a newer Lootnest for TV than this Lootnest knows. Update Lootnest on this PC")
	case "INSTALL_FAILED_INSUFFICIENT_STORAGE":
		return errors.New("the TV is out of space for the app. Remove an app on the TV, then try again")
	case "INSTALL_FAILED_USER_RESTRICTED", "INSTALL_FAILED_VERIFICATION_FAILURE":
		return errors.New("the TV refused apps from outside its store. Allow installs over USB or from unknown sources in its Developer options, then try again")
	case "INSTALL_FAILED_OLDER_SDK":
		return errors.New("this TV's Android is too old for the app (it needs Android 5 or newer)")
	}
	return fmt.Errorf("the TV didn't install the app (%s)", m[1])
}

var versionNameLine = regexp.MustCompile(`versionName=(\S+)`)

// AndroidAppVersion is the installed app's version, "" when it isn't
// installed.
func (a *ADB) AndroidAppVersion() string {
	out, _ := a.Shell("dumpsys package "+AndroidTVPackage+" | grep versionName", 20*time.Second)
	if m := versionNameLine.FindStringSubmatch(out); m != nil {
		return m[1]
	}
	return ""
}

var pinDigits = regexp.MustCompile(`^[0-9]{4}$`)

// LaunchAndroidApp opens the app's screen, closing it first so it reads
// what it's given: this PC's address and, when pairing, the PIN Lootnest
// also hands the engine.
func (a *ADB) LaunchAndroidApp(host, pin string) (string, error) {
	if net.ParseIP(host) == nil {
		return "", errors.New("this PC's address on the TV's network isn't known")
	}
	cmd := "am start -S -n " + androidTVActivity + " --es host " + host
	if pin != "" {
		if !pinDigits.MatchString(pin) {
			return "", errors.New("a pairing PIN is four digits")
		}
		cmd += " --es pairPin " + pin
	}
	out, err := a.Shell(cmd+" 2>&1", 30*time.Second)
	out = strings.TrimSpace(out)
	if err != nil {
		return out, err
	}
	if strings.Contains(out, "Error") {
		return out, fmt.Errorf("the TV didn't open the app: %s", out)
	}
	return out, nil
}

// AndroidAppLog is the app's recent log lines, newest last.
func (a *ADB) AndroidAppLog() string {
	out, _ := a.Shell("logcat -d -v time -s "+AndroidTVLogTag+":V | tail -n 200", 20*time.Second)
	return strings.TrimSpace(out)
}

// AndroidAppStarted says whether the app's log shows it reached the
// streaming engine's app list after its last start.
func AndroidAppStarted(log string) bool {
	i := strings.LastIndex(log, "Screen: starting")
	if i < 0 {
		return false
	}
	return strings.Contains(log[i:], "Lootnest: starting") || strings.Contains(log[i:], "Screen: ready")
}

// LocalIP is this PC's address on the TV's network.
func (a *ADB) LocalIP() string {
	if t, ok := a.conn.LocalAddr().(*net.TCPAddr); ok {
		return t.IP.String()
	}
	return ""
}
