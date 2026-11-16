package core

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// LG webOS TVs (Lootnest Link M3, docs/lootnest-link-spec.md): Lootnest
// puts Lootnest for TV (c-dinca/lootnest-tv-webos, a fork of Moonlight
// TV) on a TV in developer mode, the way LG's own ares-install and the
// webOS Dev Manager do: the TV's key from its Key Server (port 9991),
// unlocked with the passphrase the TV shows, then SSH as "prisoner" on
// port 9922, the package copied into /media/developer/temp and installed
// through the TV's appInstallService. The TV is found on the network with
// SSDP, as LG's second-screen apps find it.

// TVAppID is Lootnest for TV's id on the TV.
const TVAppID = "com.lootnest.tv"

// TVAppRelease is the Lootnest for TV package Lootnest installs, from its
// own GitHub releases, checked against the SHA-256 kept here.
type TVAppRelease struct {
	Version string
	URL     string
	SHA256  string
}

// TVAppPinned is the package Lootnest installs on LG TVs. A new release of
// c-dinca/lootnest-tv-webos is tested with Lootnest first, then its address
// and SHA-256 change here.
// How long Lootnest waits for the TV app to close and to open, and how
// often it looks. tvCloseWait is a pause after it's gone, before it opens
// again.
var (
	tvCloseLimit = 10 * time.Second
	tvOpenLimit  = 8 * time.Second
	tvPoll       = 500 * time.Millisecond
	tvCloseWait  = time.Second
	tvLunaLimit  = 20 * time.Second
)

var TVAppPinned = TVAppRelease{
	Version: "1.6.1",
	URL:     "https://github.com/c-dinca/lootnest-tv-webos/releases/download/v1.6.1/com.lootnest.tv_1.6.1_arm.ipk",
	SHA256:  "8db8519c38c0979befe19a0c23243086ac45cf3c004c483ba2d9d138fdebddb0",
}

// ErrTVAppNotPinned stops an install Lootnest has no checksum for.
var ErrTVAppNotPinned = errors.New("this version of Lootnest doesn't have the TV app's checksum yet, so it won't install it")

// DownloadTVApp saves the pinned package to dst, checked against its SHA-256.
func DownloadTVApp(ctx context.Context, rel TVAppRelease, dst string, progress func(done, total int64)) error {
	if len(rel.SHA256) != 64 || !strings.HasPrefix(rel.URL, "https://github.com/c-dinca/lootnest-tv-webos/releases/download/") {
		return ErrTVAppNotPinned
	}
	if ok, _ := fileSHA256Is(dst, rel.SHA256); ok {
		return nil
	}
	return DownloadApollo(ctx, ApolloRelease{Version: rel.Version, URL: rel.URL, SHA256: rel.SHA256}, dst, progress)
}

// LGTV is a TV found on the network.
type LGTV struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
}

// lgSSDPTargets are what LG TVs answer to: their second-screen service,
// and the DIAL and media renderer services they all have.
var lgSSDPTargets = []string{"urn:lge-com:service:webos-second-screen:1", "urn:dial-multiscreen-org:service:dial:1"}

// FindLGTVs asks the network for LG TVs with SSDP for wait, and reads each
// one's name from its description. A TV that doesn't answer isn't listed;
// the user can type its address.
func FindLGTVs(ctx context.Context, wait time.Duration) ([]LGTV, error) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{})
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	group := &net.UDPAddr{IP: net.IPv4(239, 255, 255, 250), Port: 1900}
	for _, st := range lgSSDPTargets {
		msg := "M-SEARCH * HTTP/1.1\r\nHOST: 239.255.255.250:1900\r\nMAN: \"ssdp:discover\"\r\nMX: 2\r\nST: " + st + "\r\n\r\n"
		for range 2 {
			_, _ = conn.WriteToUDP([]byte(msg), group)
		}
	}
	seen := map[string]string{} // ip -> description address
	end := time.Now().Add(wait)
	buf := make([]byte, 4096)
	for time.Now().Before(end) && ctx.Err() == nil {
		_ = conn.SetReadDeadline(end)
		n, from, err := conn.ReadFromUDP(buf)
		if err != nil {
			break
		}
		loc, server := ssdpHeaders(buf[:n])
		if !LooksLikeLG(server, loc) {
			continue
		}
		if _, ok := seen[from.IP.String()]; !ok {
			seen[from.IP.String()] = loc
		}
	}
	var out []LGTV
	for ip, loc := range seen {
		out = append(out, LGTV{Name: lgName(ctx, loc, ip), IP: ip})
	}
	if len(out) == 0 {
		// Windows Firewall often drops the TVs' answers (they come from
		// another address than the one asked). Every webOS TV listens on
		// port 3000 for LG's second-screen apps: try each address of this
		// PC's home networks for it.
		for _, ip := range probeLGPort(ctx, localSubnets(), 3000, 600*time.Millisecond) {
			out = append(out, LGTV{Name: "LG TV (" + ip + ")", IP: ip})
		}
	}
	return out, nil
}

// localSubnets are the /24 networks of this PC's private IPv4 addresses.
func localSubnets() []net.IP {
	var out []net.IP
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		n, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		ip := n.IP.To4()
		if ip == nil || !ip.IsPrivate() {
			continue
		}
		out = append(out, net.IPv4(ip[0], ip[1], ip[2], 0).To4())
	}
	return out
}

// probeLGPort returns the addresses in these /24 networks that accept a
// connection on port.
func probeLGPort(ctx context.Context, nets []net.IP, port int, wait time.Duration) []string {
	var (
		mu  sync.Mutex
		out []string
		wg  sync.WaitGroup
	)
	sem := make(chan struct{}, 64)
	for _, base := range nets {
		for host := 1; host < 255; host++ {
			ip := net.IPv4(base[0], base[1], base[2], byte(host)).String()
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer func() { <-sem; wg.Done() }()
				d := net.Dialer{Timeout: wait}
				c, err := d.DialContext(ctx, "tcp", net.JoinHostPort(ip, fmt.Sprint(port)))
				if err != nil {
					return
				}
				c.Close()
				mu.Lock()
				out = append(out, ip)
				mu.Unlock()
			}()
		}
	}
	wg.Wait()
	return out
}

func ssdpHeaders(b []byte) (location, server string) {
	for _, line := range strings.Split(string(b), "\r\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		switch strings.ToUpper(strings.TrimSpace(k)) {
		case "LOCATION":
			location = strings.TrimSpace(v)
		case "SERVER":
			server = strings.TrimSpace(v)
		}
	}
	return
}

// LooksLikeLG says whether an SSDP answer comes from an LG webOS TV.
func LooksLikeLG(server, location string) bool {
	s := strings.ToLower(server)
	return strings.Contains(s, "webos") || strings.Contains(s, "lge") || strings.Contains(strings.ToLower(location), "lge")
}

func lgName(ctx context.Context, loc, ip string) string {
	u, err := url.Parse(loc)
	if err != nil || u.Scheme != "http" || u.Hostname() != ip {
		return "LG TV (" + ip + ")"
	}
	c, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(c, "GET", loc, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "LG TV (" + ip + ")"
	}
	defer resp.Body.Close()
	var d struct {
		Device struct {
			FriendlyName string `xml:"friendlyName"`
		} `xml:"device"`
	}
	if xml.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&d) != nil || strings.TrimSpace(d.Device.FriendlyName) == "" {
		return "LG TV (" + ip + ")"
	}
	return strings.TrimSpace(d.Device.FriendlyName)
}

// ErrTVKeyServer is a TV that doesn't give its key: Key Server is off, or
// the TV isn't in developer mode.
var ErrTVKeyServer = errors.New("the TV didn't give Lootnest its key. In the Developer Mode app on the TV, turn on Key Server")

// ErrTVPassphrase is a passphrase that doesn't open the TV's key.
var ErrTVPassphrase = errors.New("that passphrase doesn't match. Type the one the Developer Mode app shows, in capitals")

// FetchTVKey gets the TV's developer-mode key from its Key Server and
// unlocks it with the passphrase the TV shows. It returns the key in PEM,
// unlocked, for Lootnest to keep (encrypted) and use again.
func FetchTVKey(ctx context.Context, ip, passphrase string) ([]byte, error) {
	if net.ParseIP(ip) == nil {
		return nil, errors.New("that isn't a TV's address")
	}
	c, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(c, "GET", "http://"+net.JoinHostPort(ip, "9991")+"/webos_rsa", nil)
	resp, err := (&http.Client{Transport: &http.Transport{Proxy: nil}}).Do(req)
	if err != nil {
		return nil, ErrTVKeyServer
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, ErrTVKeyServer
	}
	pemKey, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return nil, ErrTVKeyServer
	}
	return UnlockTVKey(pemKey, passphrase)
}

// UnlockTVKey checks the key opens with the passphrase and returns it as
// an unlocked PEM.
func UnlockTVKey(pemKey []byte, passphrase string) ([]byte, error) {
	key, err := ssh.ParseRawPrivateKeyWithPassphrase(pemKey, []byte(strings.TrimSpace(passphrase)))
	if err != nil {
		var missing *ssh.PassphraseMissingError
		if errors.As(err, &missing) {
			// Some TVs serve it unlocked.
			if key, err = ssh.ParseRawPrivateKey(pemKey); err != nil {
				return nil, ErrTVPassphrase
			}
		} else {
			return nil, ErrTVPassphrase
		}
	}
	block, err := ssh.MarshalPrivateKey(key, "")
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(block), nil
}

// TV is an SSH session with a TV in developer mode.
type TV struct {
	IP     string
	client *ssh.Client
}

// ConnectTV opens SSH to the TV with its developer-mode key. The TV's own
// host key is pinned the first time (hostKey "") and returned by HostKey.
func ConnectTV(ctx context.Context, ip string, pemKey []byte, hostKey string) (*TV, string, error) {
	return connectTVAt(ctx, ip, net.JoinHostPort(ip, "9922"), pemKey, hostKey)
}

func connectTVAt(ctx context.Context, ip, addr string, pemKey []byte, hostKey string) (*TV, string, error) {
	signer, err := ssh.ParsePrivateKey(pemKey)
	if err != nil {
		return nil, "", err
	}
	seen := ""
	cfg := &ssh.ClientConfig{
		User: "prisoner",
		Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: func(_ string, _ net.Addr, key ssh.PublicKey) error {
			sum := sha256.Sum256(key.Marshal())
			seen = hex.EncodeToString(sum[:])
			if hostKey != "" && seen != hostKey {
				return errors.New("the TV's identity changed since Lootnest last talked to it")
			}
			return nil
		},
		Timeout: 10 * time.Second,
	}
	d := net.Dialer{Timeout: 10 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, "", errors.New("the TV didn't answer. Check it's on, in developer mode, and on this network")
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, cfg)
	if err != nil {
		conn.Close()
		return nil, "", fmt.Errorf("the TV refused Lootnest (%w)", err)
	}
	return &TV{IP: ip, client: ssh.NewClient(c, chans, reqs)}, seen, nil
}

// Close ends the session.
func (t *TV) Close() error { return t.client.Close() }

// LocalIP is this PC's address on the network the TV is on.
func (t *TV) LocalIP() string {
	if a, ok := t.client.LocalAddr().(*net.TCPAddr); ok {
		return a.IP.String()
	}
	return ""
}

// run runs a command on the TV and returns what it printed.
func (t *TV) run(cmd string, stdin io.Reader) (string, error) {
	return t.runFor(cmd, stdin, time.Minute)
}

// runFor runs a command with a time limit: a TV that stops answering
// ends the session instead of holding Lootnest forever.
func (t *TV) runFor(cmd string, stdin io.Reader, limit time.Duration) (string, error) {
	s, err := t.client.NewSession()
	if err != nil {
		return "", err
	}
	defer s.Close()
	var out, errb bytes.Buffer
	s.Stdout, s.Stderr, s.Stdin = &out, &errb, stdin
	done := make(chan error, 1)
	go func() { done <- s.Run(cmd) }()
	select {
	case err = <-done:
	case <-time.After(limit):
		s.Close()
		return out.String(), fmt.Errorf("the TV didn't finish %q in %s", strings.Fields(cmd)[0], limit)
	}
	if err != nil {
		return out.String(), fmt.Errorf("%v: %s", err, strings.TrimSpace(errb.String()))
	}
	return out.String(), nil
}

// luna calls a webOS service with luna-send-pub and returns its first
// answer. luna-send-pub ends without calling anything when its input closes
// first, as an SSH command's does when it's given none, so the input is
// held open until the answer has come, or limit has passed.
func (t *TV) luna(uri, payload string, limit time.Duration) (string, error) {
	s, err := t.client.NewSession()
	if err != nil {
		return "", err
	}
	defer s.Close()
	hold, err := s.StdinPipe()
	if err != nil {
		return "", err
	}
	defer hold.Close()
	pipe, err := s.StdoutPipe()
	if err != nil {
		return "", err
	}
	// -i, as the install that works: it calls once and waits on its input.
	if err := s.Start("luna-send-pub -i " + shQuote(uri) + " " + shQuote(payload) + " 2>&1"); err != nil {
		return "", err
	}
	lines, quit := make(chan string), make(chan struct{})
	defer close(quit)
	go func() {
		defer close(lines)
		sc := bufio.NewScanner(pipe)
		for sc.Scan() {
			select {
			case lines <- sc.Text():
			case <-quit:
				return
			}
		}
	}()
	var out strings.Builder
	deadline := time.After(limit)
	for {
		select {
		case line, ok := <-lines:
			if !ok {
				return out.String(), nil
			}
			out.WriteString(line + "\n")
			if strings.Contains(line, "returnValue") {
				return out.String(), nil
			}
		case <-deadline:
			return out.String(), fmt.Errorf("the TV didn't answer %s in %s", uri, limit)
		}
	}
}

// shQuote is a POSIX shell single-quoted string.
func shQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

const tvTempDir = "/media/developer/temp"

// InstallTVApp copies the package to the TV and installs it, as
// ares-install does.
func (t *TV) InstallTVApp(ctx context.Context, ipk string) error {
	f, err := os.Open(ipk)
	if err != nil {
		return err
	}
	defer f.Close()
	remote := tvTempDir + "/lootnest-tv.ipk"
	if _, err := t.run("mkdir -p "+tvTempDir+" && cat > "+shQuote(remote), f); err != nil {
		return fmt.Errorf("couldn't copy the app to the TV (%w)", err)
	}
	defer t.run("rm -f "+shQuote(remote), nil) //nolint:errcheck
	// The copy arrived whole?
	if st, err := f.Stat(); err == nil {
		got, _ := t.run("wc -c < "+shQuote(remote), nil)
		if n, perr := strconv.ParseInt(strings.TrimSpace(got), 10, 64); perr != nil || n != st.Size() {
			return fmt.Errorf("the app didn't arrive whole on the TV (%s of %d bytes)", strings.TrimSpace(got), st.Size())
		}
	}
	// As LG's ares-install (and webOS Homebrew's ares-cli-rs) do: an
	// interactive luna-send-pub subscribed to the install, its input held
	// open, since it stops at the end of its input before it calls anything
	// (what 2.2.1 and 2.2.2 ran into: the session's input was closed, or no
	// -i). Its answers say "installed" or why not.
	payload, _ := json.Marshal(map[string]any{"id": "com.ares.defaultName", "ipkUrl": remote, "subscribe": true})
	sess, err := t.client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()
	hold, err := sess.StdinPipe()
	if err != nil {
		return err
	}
	defer hold.Close()
	out, err := sess.StdoutPipe()
	if err != nil {
		return err
	}
	if err := sess.Start("luna-send-pub -i 'luna://com.webos.appInstallService/dev/install' " + shQuote(string(payload)) + " 2>&1"); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- InstallOutcome(out) }()
	select {
	case err = <-done:
	case <-time.After(3 * time.Minute):
		err = &QuietInstallError{Said: "nothing for three minutes"}
	case <-ctx.Done():
		return ctx.Err()
	}
	var quiet *QuietInstallError
	if !errors.As(err, &quiet) {
		return err
	}
	// It ended without an outcome: the install may still go on in the
	// background, so look for the app for a while before the package (which
	// it reads) is removed.
	for end := time.Now().Add(90 * time.Second); ; time.Sleep(3 * time.Second) {
		if t.HasTVApp() {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if time.Now().After(end) {
			where, _ := t.run("command -v luna-send-pub || echo missing", nil)
			return fmt.Errorf("the TV didn't install the app. It said: %s (luna-send-pub: %s)", quiet.Said, strings.TrimSpace(where))
		}
	}
}

var installState = regexp.MustCompile(`"state"\s*:\s*"([^"]+)"`)

// InstallOutcome reads appInstallService's answers until the install ends.
func InstallOutcome(r io.Reader) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	var last []string
	for sc.Scan() {
		line := sc.Text()
		if t := strings.TrimSpace(line); t != "" {
			if last = append(last, t); len(last) > 3 {
				last = last[1:]
			}
		}
		if strings.Contains(line, `"returnValue":false`) || strings.Contains(line, `"returnValue": false`) {
			return fmt.Errorf("the TV refused the app: %s", tvErrorText(line))
		}
		m := installState.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		switch st := strings.ToLower(m[1]); {
		case st == "installed":
			return nil
		case strings.Contains(st, "fail") || strings.Contains(st, "error"):
			return fmt.Errorf("the TV couldn't install the app: %s", tvErrorText(line))
		}
	}
	said := strings.Join(last, " ")
	if said == "" {
		said = "nothing"
	}
	if len(said) > 300 {
		said = said[:300]
	}
	return &QuietInstallError{Said: said}
}

// QuietInstallError is the install service ending without an outcome;
// Said is the last of what it printed.
type QuietInstallError struct{ Said string }

func (e *QuietInstallError) Error() string {
	return "the TV stopped answering while it installed the app"
}

// HasTVApp says whether Lootnest for TV is installed on the TV.
func (t *TV) HasTVApp() bool {
	out, _ := t.run("test -d "+shQuote("/media/developer/apps/usr/palm/applications/"+TVAppID)+" && echo yes", nil)
	if strings.TrimSpace(out) == "yes" {
		return true
	}
	payload, _ := json.Marshal(map[string]any{"id": TVAppID})
	out, _ = t.luna("luna://com.webos.applicationManager/getAppInfo", string(payload), tvLunaLimit)
	return strings.Contains(out, `"returnValue":true`) || strings.Contains(out, `"returnValue": true`)
}

// TVAppVersion is the version of Lootnest for TV on the TV, from the
// app's own appinfo.json; "" when it isn't there or can't be read.
func (t *TV) TVAppVersion() string {
	out, err := t.run("cat "+shQuote("/media/developer/apps/usr/palm/applications/"+TVAppID+"/appinfo.json"), nil)
	if err != nil {
		return ""
	}
	return tvAppVersionIn(out)
}

func tvAppVersionIn(appinfo string) string {
	var m struct {
		Version string `json:"version"`
	}
	if json.Unmarshal([]byte(appinfo), &m) != nil {
		return ""
	}
	return strings.TrimSpace(m.Version)
}

func tvErrorText(line string) string {
	var m map[string]any
	if json.Unmarshal([]byte(line), &m) == nil {
		for _, k := range []string{"errorText", "message", "errorMessage"} {
			if s, ok := m[k].(string); ok && s != "" {
				return s
			}
		}
	}
	if len(line) > 200 {
		line = line[:200]
	}
	return line
}

// LaunchTVApp opens Lootnest for TV with its launch parameters. The
// trace says what the TV answered at each step, for the log.
func (t *TV) LaunchTVApp(params map[string]string) (trace string, err error) {
	var tr []string
	note := func(f string, a ...any) { tr = append(tr, fmt.Sprintf(f, a...)) }
	defer func() { trace = strings.Join(tr, "; ") }()
	// An app that's already open never reads new launch parameters, so it
	// is closed first, and Lootnest waits until it's gone: a launch while
	// it closes is lost. Closing one that isn't open only fails.
	closing, _ := json.Marshal(map[string]any{"id": TVAppID})
	closed := false
	for _, uri := range []string{"luna://com.webos.service.applicationManager/closeByAppId", "luna://com.webos.applicationManager/closeByAppId"} {
		out, _ := t.luna(uri, string(closing), tvLunaLimit)
		note("close: %s", short(out))
		if !refused(out) && strings.Contains(out, "returnValue") {
			closed = true
			break
		}
	}
	// A TV that can't close it: the app open on it takes the pairing from
	// the request LeavePairing wrote instead.
	for end := time.Now().Add(tvCloseLimit); closed; time.Sleep(tvPoll) {
		running, known := t.AppRunning()
		if !known || !running {
			note("running before: %s", yesNo(running, known))
			break
		}
		if time.Now().After(end) {
			note("still running after %s", tvCloseLimit)
			break
		}
	}
	time.Sleep(tvCloseWait)
	payload, _ := json.Marshal(map[string]any{"id": TVAppID, "params": params})
	launch := func() error {
		out, err := t.luna("luna://com.webos.applicationManager/launch", string(payload), tvLunaLimit)
		note("launch: %s", short(out))
		if err != nil {
			return fmt.Errorf("the TV didn't open the app (%w)", err)
		}
		if refused(out) {
			return fmt.Errorf("the TV didn't open the app: %s", tvErrorText(strings.TrimSpace(out)))
		}
		return nil
	}
	if err := launch(); err != nil {
		return "", err
	}
	// Once more if it didn't open.
	for end := time.Now().Add(tvOpenLimit); time.Now().Before(end); time.Sleep(tvPoll) {
		if running, known := t.AppRunning(); !known || running {
			note("running after: %s", yesNo(running, known))
			return "", nil
		}
	}
	note("not running after %s, launching again", tvOpenLimit)
	return "", launch()
}

// refused is a luna answer with returnValue false, however it's spaced.
func refused(out string) bool {
	for _, line := range strings.Split(out, "\n") {
		var m map[string]any
		if json.Unmarshal([]byte(strings.TrimSpace(line)), &m) == nil {
			if v, ok := m["returnValue"].(bool); ok && !v {
				return true
			}
		}
	}
	return false
}

func short(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 160 {
		s = s[:160] + "..."
	}
	if s == "" {
		return "nothing"
	}
	return s
}

func yesNo(v, known bool) string {
	switch {
	case !known:
		return "unknown"
	case v:
		return "yes"
	}
	return "no"
}

// AppRunning says whether Lootnest for TV is the app on the TV's screen;
// known is false when the TV doesn't say. (A developer session's ps
// doesn't list developer apps.)
func (t *TV) AppRunning() (running, known bool) {
	out, err := t.luna("luna://com.webos.applicationManager/getForegroundAppInfo", "{}", tvLunaLimit)
	if err != nil || refused(out) {
		return false, false
	}
	for _, line := range strings.Split(out, "\n") {
		var m map[string]any
		if json.Unmarshal([]byte(strings.TrimSpace(line)), &m) == nil {
			if id, ok := m["appId"].(string); ok {
				return id == TVAppID, true
			}
		}
	}
	return false, false
}

// LeavePairing writes lootnest-pair.json, the pairing request the TV app
// reads at start and while it waits (it gets no new launch parameters
// when it's already open).
func (t *TV) LeavePairing(id, host, pin string) error {
	b, _ := json.Marshal(map[string]string{"id": id, "host": host, "pairPin": pin})
	_, err := t.run("mkdir -p "+tvTempDir+" && cat > "+shQuote(tvTempDir+"/lootnest-pair.json"), bytes.NewReader(b))
	return err
}

// ClearPairing removes the pairing request.
func (t *TV) ClearPairing() {
	_, _ = t.run("rm -f "+shQuote(tvTempDir+"/lootnest-pair.json"), nil)
}

// TVAppStarted says whether the TV app's log shows it went on to start
// Lootnest (so it was paired) after its last launch.
func TVAppStarted(log string) bool {
	last := strings.LastIndex(log, "Launched ")
	if last < 0 {
		last = 0
	}
	rest := log[last:]
	return strings.Contains(rest, "Screen: starting") || strings.Contains(rest, "Screen: ready")
}

// WebOSVersion is the TV's webOS release, from /etc/starfish-release, or "".
func (t *TV) WebOSVersion() string {
	out, _ := t.run("cat /etc/starfish-release 2>/dev/null | head -n 1", nil)
	if v := strings.TrimSpace(out); v != "" {
		return v
	}
	// A developer session can't read that file on every TV; the system
	// property service answers any app.
	out, _ = t.luna("luna://com.webos.service.tv.systemproperty/getSystemInfo", `{"keys":["modelName","sdkVersion","firmwareVersion"]}`, tvLunaLimit)
	return tvSystemInfoIn(out)
}

// tvSystemInfoIn reads the model and versions from getSystemInfo's answer:
// "OLED55C9PLA, webOS 4.9.0, firmware 05.30.25", or "".
func tvSystemInfoIn(answer string) string {
	i := strings.Index(answer, "{")
	if i < 0 {
		return ""
	}
	var info struct {
		Model    string `json:"modelName"`
		SDK      string `json:"sdkVersion"`
		Firmware string `json:"firmwareVersion"`
	}
	if json.NewDecoder(strings.NewReader(answer[i:])).Decode(&info) != nil {
		return ""
	}
	var parts []string
	if info.Model != "" {
		parts = append(parts, info.Model)
	}
	if info.SDK != "" {
		parts = append(parts, "webOS "+info.SDK)
	}
	if info.Firmware != "" {
		parts = append(parts, "firmware "+info.Firmware)
	}
	return strings.Join(parts, ", ")
}

// WiFi is the TV's wireless link as Linux counts it (/proc/net/wireless:
// the interface, link quality, signal and noise), or "" on a cable or when
// the TV doesn't say.
func (t *TV) WiFi() string {
	out, _ := t.run("tail -n +3 /proc/net/wireless 2>/dev/null | head -n 1", nil)
	return strings.Join(strings.Fields(out), " ")
}

// TVAppLog is the end of the TV app's own log of its steps, or "".
func (t *TV) TVAppLog() string {
	out, _ := t.run("tail -n 30 "+shQuote(tvTempDir+"/lootnest-tv.log")+" 2>/dev/null", nil)
	return strings.TrimSpace(out)
}

// ReachesPC asks the TV whether it reaches the streaming engine at the PC
// address pc: "yes", "no", or "" when the TV can't tell.
func (t *TV) ReachesPC(pc string) string {
	ip, err := plainIP(pc)
	if err != nil {
		return ""
	}
	url := shQuote("http://" + ip + ":47989/serverinfo")
	out, err := t.run("if command -v wget >/dev/null 2>&1; then wget -q -T 5 -O - "+url+" 2>/dev/null | grep -c -i '<root'; elif command -v curl >/dev/null 2>&1; then curl -s -m 5 "+url+" 2>/dev/null | grep -c -i '<root'; else echo x; fi", nil)
	_ = err // grep -c exits 1 on no match
	n, perr := strconv.Atoi(strings.TrimSpace(out))
	switch {
	case perr != nil:
		return ""
	case n > 0:
		return "yes"
	}
	return "no"
}

// DevModeToken reads the developer-mode session token the TV keeps, which
// LG's site extends.
func (t *TV) DevModeToken() (string, error) {
	out, err := t.run("cat /var/luna/preferences/devmode_enabled", nil)
	tok := strings.TrimSpace(out)
	if err != nil || !ValidDevModeToken(tok) {
		return "", errors.New("the TV has no developer mode session to extend")
	}
	return tok, nil
}

// ValidDevModeToken is the session token's shape: letters, digits and a
// few signs, never anything that would change the address it goes in.
func ValidDevModeToken(tok string) bool {
	if len(tok) < 8 || len(tok) > 256 {
		return false
	}
	for _, r := range tok {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.') {
			return false
		}
	}
	return true
}

// lgDevSiteURL is LG's developer site, where a developer-mode session is
// extended (as the webOS Dev Manager does).
var lgDevSiteURL = "https://developer.lge.com"

// ExtendDevMode resets the TV's developer-mode session to its full length
// (about 50 hours) on LG's site.
func ExtendDevMode(ctx context.Context, token string) error {
	if !ValidDevModeToken(token) {
		return errors.New("the TV's developer mode token isn't valid")
	}
	c, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(c, "GET", lgDevSiteURL+"/secure/ResetDevModeSession.dev?sessionToken="+url.QueryEscape(token), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("couldn't reach LG's developer site (%w)", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("LG's developer site answered %s", resp.Status)
	}
	var r struct {
		Result    string `json:"result"`
		ErrorCode string `json:"errorCode"`
		ErrorMsg  string `json:"errorMsg"`
	}
	if json.Unmarshal(b, &r) == nil && r.Result != "" && !strings.EqualFold(r.Result, "success") {
		return fmt.Errorf("LG's developer site didn't extend it: %s", strings.TrimSpace(r.ErrorMsg))
	}
	return nil
}
