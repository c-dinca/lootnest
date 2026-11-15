package core

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// lockedKey is an RSA key in the old encrypted PEM form webOS's Key Server
// serves.
func lockedKey(t *testing.T, pass string) (*rsa.PrivateKey, []byte) {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	//nolint:staticcheck // the TV's own format
	block, err := x509.EncryptPEMBlock(rand.Reader, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(k), []byte(pass), x509.PEMCipherAES128)
	if err != nil {
		t.Fatal(err)
	}
	return k, pem.EncodeToMemory(block)
}

func TestUnlockTVKey(t *testing.T) {
	_, locked := lockedKey(t, "A1B2C3")
	if _, err := UnlockTVKey(locked, "WRONG1"); !errors.Is(err, ErrTVPassphrase) {
		t.Fatalf("wrong passphrase: %v", err)
	}
	open, err := UnlockTVKey(locked, " A1B2C3 ")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ssh.ParsePrivateKey(open); err != nil {
		t.Fatalf("unlocked key doesn't parse: %v", err)
	}
}

func TestFetchTVKeyNeedsAnAddress(t *testing.T) {
	if _, err := FetchTVKey(context.Background(), "tv.example.com", "x"); err == nil {
		t.Fatal("a name was fetched")
	}
}

func TestLooksLikeLG(t *testing.T) {
	if !LooksLikeLG("WebOS/1.5 UPnP/1.0 webOSTV/1.0", "") || !LooksLikeLG("", "http://192.168.1.5:1914/lge/description.xml") {
		t.Fatal("an LG answer wasn't seen")
	}
	if LooksLikeLG("Linux/3.10 UPnP/1.0 Sonos/70", "http://192.168.1.9:1400/xml/device_description.xml") {
		t.Fatal("a speaker was taken for a TV")
	}
}

func TestInstallOutcome(t *testing.T) {
	ok := `{"returnValue":true,"subscribed":true}
{"details":{"state":"installing","progress":30},"returnValue":true}
{"details":{"state":"installed"},"returnValue":true}`
	if err := InstallOutcome(strings.NewReader(ok)); err != nil {
		t.Fatal(err)
	}
	bad := `{"returnValue":true}
{"details":{"state":"install failed","errorCode":-5},"returnValue":true,"errorText":"not enough space"}`
	if err := InstallOutcome(strings.NewReader(bad)); err == nil || !strings.Contains(err.Error(), "not enough space") {
		t.Fatalf("failure: %v", err)
	}
	if err := InstallOutcome(strings.NewReader(`{"returnValue":false,"errorText":"Denied"}`)); err == nil {
		t.Fatal("a refusal passed")
	}
	if err := InstallOutcome(strings.NewReader("")); err == nil {
		t.Fatal("silence passed")
	}
}

func TestValidDevModeToken(t *testing.T) {
	for _, tok := range []string{"ABCdef0123456789", "a1b2c3d4-e5f6_7.8"} {
		if !ValidDevModeToken(tok) {
			t.Errorf("%q refused", tok)
		}
	}
	for _, tok := range []string{"", "short", "abc&def=1234567", "abc/../12345678", strings.Repeat("a", 300)} {
		if ValidDevModeToken(tok) {
			t.Errorf("%q accepted", tok)
		}
	}
}

// fakeTV is an SSH server that answers the commands Lootnest runs as a TV
// in developer mode does.
type fakeTV struct {
	mu       sync.Mutex
	files    map[string][]byte
	commands []string
	quiet    bool // the install service ends without an outcome
	checks   int  // test -d answers yes from this check on (quiet only)
}

func (f *fakeTV) serve(t *testing.T, ln net.Listener, user ssh.PublicKey) {
	hostKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	signer, _ := ssh.NewSignerFromKey(hostKey)
	cfg := &ssh.ServerConfig{PublicKeyCallback: func(c ssh.ConnMetadata, k ssh.PublicKey) (*ssh.Permissions, error) {
		if c.User() == "prisoner" && string(k.Marshal()) == string(user.Marshal()) {
			return nil, nil
		}
		return nil, errors.New("no")
	}}
	cfg.AddHostKey(signer)
	for {
		nc, err := ln.Accept()
		if err != nil {
			return
		}
		go func() {
			_, chans, reqs, err := ssh.NewServerConn(nc, cfg)
			if err != nil {
				return
			}
			go ssh.DiscardRequests(reqs)
			for nch := range chans {
				ch, creqs, _ := nch.Accept()
				go func() {
					defer ch.Close()
					for req := range creqs {
						if req.Type != "exec" {
							_ = req.Reply(false, nil)
							continue
						}
						cmd := string(req.Payload[4:])
						_ = req.Reply(true, nil)
						f.mu.Lock()
						f.commands = append(f.commands, cmd)
						f.mu.Unlock()
						switch {
						case strings.Contains(cmd, "cat > "):
							b, _ := io.ReadAll(ch)
							f.mu.Lock()
							f.files["ipk"] = b
							f.mu.Unlock()
						case strings.Contains(cmd, "appInstallService"):
							if f.quiet {
								_, _ = io.WriteString(ch, "{\"returnValue\":true,\"subscribed\":true}\n")
							} else {
								_, _ = io.WriteString(ch, "{\"returnValue\":true}\n{\"details\":{\"state\":\"installed\"},\"returnValue\":true}\n")
							}
						case strings.HasPrefix(cmd, "test -d"):
							f.mu.Lock()
							f.checks--
							yes := f.checks <= 0
							f.mu.Unlock()
							if yes {
								_, _ = io.WriteString(ch, "yes\n")
							}
						case strings.Contains(cmd, "applicationManager/launch"):
							_, _ = io.WriteString(ch, "{\"returnValue\":true}\n")
						case strings.Contains(cmd, "devmode_enabled"):
							_, _ = io.WriteString(ch, "TOKEN1234abcd\n")
						}
						_, _ = ch.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
						return
					}
				}()
			}
		}()
	}
}

func TestTVInstallAndLaunch(t *testing.T) {
	k, _ := rsa.GenerateKey(rand.Reader, 2048)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)})
	pub, _ := ssh.NewPublicKey(&k.PublicKey)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	f := &fakeTV{files: map[string][]byte{}}
	go f.serve(t, ln, pub)

	// ConnectTV dials port 9922; the test dials the fake's port through a
	// TV value made by hand from the same handshake.
	tv, host, err := connectTVAt(context.Background(), "127.0.0.1", ln.Addr().String(), keyPEM, "")
	if err != nil {
		t.Fatal(err)
	}
	defer tv.Close()
	if len(host) != 64 {
		t.Fatalf("host key not pinned: %q", host)
	}
	if _, _, err := connectTVAt(context.Background(), "127.0.0.1", ln.Addr().String(), keyPEM, strings.Repeat("0", 64)); err == nil {
		t.Fatal("a TV with another identity was trusted")
	}

	ipk := filepath.Join(t.TempDir(), "app.ipk")
	_ = os.WriteFile(ipk, []byte("ipk bytes"), 0o644)
	if err := tv.InstallTVApp(context.Background(), ipk); err != nil {
		t.Fatal(err)
	}
	if string(f.files["ipk"]) != "ipk bytes" {
		t.Fatalf("copied %q", f.files["ipk"])
	}
	if err := tv.LaunchTVApp(map[string]string{"host": "192.168.1.20", "pairPin": "4821"}); err != nil {
		t.Fatal(err)
	}
	tok, err := tv.DevModeToken()
	if err != nil || tok != "TOKEN1234abcd" {
		t.Fatalf("token %q %v", tok, err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	var launch string
	for _, c := range f.commands {
		if strings.Contains(c, "applicationManager/launch") {
			launch = c
		}
	}
	if !strings.Contains(launch, `"id":"com.lootnest.tv"`) || !strings.Contains(launch, `"pairPin":"4821"`) {
		t.Fatalf("launch: %s", launch)
	}
}

func TestExtendDevMode(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Path + "?" + r.URL.RawQuery
		_, _ = io.WriteString(w, `{"result":"success","errorCode":"200","errorMsg":"GNL"}`)
	}))
	defer srv.Close()
	old := lgDevSiteURL
	lgDevSiteURL = srv.URL
	defer func() { lgDevSiteURL = old }()
	if err := ExtendDevMode(context.Background(), "TOKEN1234abcd"); err != nil {
		t.Fatal(err)
	}
	if got != "/secure/ResetDevModeSession.dev?sessionToken=TOKEN1234abcd" {
		t.Fatalf("asked %s", got)
	}
	if err := ExtendDevMode(context.Background(), "bad token&x=1"); err == nil {
		t.Fatal("a bad token was sent")
	}
}

func TestDownloadTVAppNeedsPin(t *testing.T) {
	if err := DownloadTVApp(context.Background(), TVAppRelease{URL: "https://example.com/x.ipk", SHA256: strings.Repeat("a", 64)}, t.TempDir()+"/x.ipk", nil); !errors.Is(err, ErrTVAppNotPinned) {
		t.Fatal(err)
	}
}

func TestTVAppPinned(t *testing.T) {
	r := TVAppPinned
	if len(r.SHA256) != 64 || r.URL != "https://github.com/c-dinca/lootnest-tv-webos/releases/download/v"+r.Version+"/com.lootnest.tv_"+r.Version+"_arm.ipk" {
		t.Fatalf("%+v", r)
	}
}

func TestTVInstallQuiet(t *testing.T) {
	k, _ := rsa.GenerateKey(rand.Reader, 2048)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)})
	pub, _ := ssh.NewPublicKey(&k.PublicKey)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	// The service says only that it subscribed; the app is there at the
	// second check.
	f := &fakeTV{files: map[string][]byte{}, quiet: true, checks: 2}
	go f.serve(t, ln, pub)
	tv, _, err := connectTVAt(context.Background(), "127.0.0.1", ln.Addr().String(), keyPEM, "")
	if err != nil {
		t.Fatal(err)
	}
	defer tv.Close()
	ipk := filepath.Join(t.TempDir(), "app.ipk")
	_ = os.WriteFile(ipk, []byte("ipk"), 0o644)
	if err := tv.InstallTVApp(context.Background(), ipk); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	lastCheck, rm := -1, -1
	for i, c := range f.commands {
		if strings.HasPrefix(c, "test -d") {
			lastCheck = i
		}
		if strings.HasPrefix(c, "rm -f") {
			rm = i
		}
	}
	if lastCheck < 0 || rm < lastCheck {
		t.Fatalf("the package was removed before the app was there: %q", f.commands)
	}
}

func TestInstallOutcomeSaysWhat(t *testing.T) {
	err := InstallOutcome(strings.NewReader("{\"returnValue\":true,\"subscribed\":true}\nluna-send-pub: permission denied\n"))
	var q *QuietInstallError
	if !errors.As(err, &q) || !strings.Contains(q.Said, "permission denied") {
		t.Fatalf("%v", err)
	}
}

func TestProbeLGPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	got := probeLGPort(context.Background(), []net.IP{net.IPv4(127, 0, 0, 0).To4()}, port, 200*time.Millisecond)
	if len(got) != 1 || got[0] != "127.0.0.1" {
		t.Fatalf("%v", got)
	}
}
