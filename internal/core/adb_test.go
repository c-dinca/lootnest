package core

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	mathbig "math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// A key and its .pub made by Google's own `adb keygen` (platform-tools):
// Lootnest's encoding of the public key must be the same bytes.
func TestADBPublicKeyMatchesADB(t *testing.T) {
	priv, err := os.ReadFile("testdata/adb/adbkey")
	if err != nil {
		t.Fatal(err)
	}
	pub, err := os.ReadFile("testdata/adb/adbkey.pub")
	if err != nil {
		t.Fatal(err)
	}
	k, err := parseADBKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Fields(string(pub))[0]
	got, err := ADBPublicKey(&k.PublicKey, "x")
	if err != nil {
		t.Fatal(err)
	}
	if g := strings.Fields(string(got))[0]; g != want {
		t.Fatalf("public key differs from adb's:\n got %s\nwant %s", g[:40], want[:40])
	}
}

// decodeADBPub reads the RSAPublicKey struct the way adbd does.
func decodeADBPub(t *testing.T, line []byte) *rsa.PublicKey {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(strings.Fields(strings.TrimRight(string(line), "\x00"))[0])
	if err != nil || len(raw) != 4+4+256+256+4 {
		t.Fatalf("bad public key: %v, %d bytes", err, len(raw))
	}
	if binary.LittleEndian.Uint32(raw) != 64 {
		t.Fatal("word count isn't 64")
	}
	be := make([]byte, 256)
	for i := range 64 {
		binary.BigEndian.PutUint32(be[(63-i)*4:], binary.LittleEndian.Uint32(raw[8+i*4:]))
	}
	n := new(mathbig.Int).SetBytes(be)
	// n0inv * n[0] == -1 mod 2^32, as adbd's Montgomery code needs.
	n0inv := binary.LittleEndian.Uint32(raw[4:])
	if n0inv*uint32(new(mathbig.Int).Mod(n, mathbig.NewInt(1<<32)).Uint64()) != 0xffffffff {
		t.Fatal("n0inv is wrong")
	}
	return &rsa.PublicKey{N: n, E: int(binary.LittleEndian.Uint32(raw[8+512:]))}
}

// fakeADBD is enough of a TV's adbd: auth (with a person who allows the
// computer or not), shell commands answered by a table, and sync pushes.
type fakeADBD struct {
	t       *testing.T
	ln      net.Listener
	allow   bool
	mu      sync.Mutex
	known   []*rsa.PublicKey
	prompts int
	shell   func(cmd string) string
	files   map[string][]byte
	cmds    []string
	maxData int
}

func newFakeADBD(t *testing.T, allow bool) *fakeADBD {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeADBD{t: t, ln: ln, allow: allow, files: map[string][]byte{}, maxData: 4096,
		shell: func(string) string { return "" }}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go f.serve(c)
		}
	}()
	t.Cleanup(func() { ln.Close() })
	return f
}

func (f *fakeADBD) addr() string { return f.ln.Addr().String() }

func (f *fakeADBD) serve(c net.Conn) {
	defer c.Close()
	a := &ADB{conn: c}
	m, err := a.read()
	if err != nil || m.cmd != adbCNXN {
		return
	}
	token := make([]byte, 20)
	_, _ = rand.Read(token)
	_ = a.send(adbAUTH, adbAuthToken, 0, token)
	authed := false
	for !authed {
		m, err := a.read()
		if err != nil {
			return
		}
		if m.cmd != adbAUTH {
			continue
		}
		switch m.arg0 {
		case adbAuthSignature:
			f.mu.Lock()
			for _, k := range f.known {
				if rsa.VerifyPKCS1v15(k, crypto.SHA1, token, m.data) == nil {
					authed = true
				}
			}
			f.mu.Unlock()
			if !authed {
				_, _ = rand.Read(token)
				_ = a.send(adbAUTH, adbAuthToken, 0, token)
			}
		case adbAuthPublicKey:
			k := decodeADBPub(f.t, m.data)
			f.mu.Lock()
			f.prompts++
			f.mu.Unlock()
			if !f.allow {
				return // the person said no: adbd drops the connection
			}
			f.mu.Lock()
			f.known = append(f.known, k)
			f.mu.Unlock()
			authed = true
		}
	}
	_ = a.send(adbCNXN, adbVersion, uint32(f.maxData), []byte("device::ro.product.name=sabrina;ro.product.model=Chromecast;"))
	var syncBuf []byte
	var syncFile string
	var local uint32
	for {
		m, err := a.read()
		if err != nil {
			return
		}
		switch m.cmd {
		case adbOPEN:
			local++
			svc := strings.TrimRight(string(m.data), "\x00")
			remote := m.arg0
			_ = a.send(adbOKAY, local, remote, nil)
			if cmd, ok := strings.CutPrefix(svc, "shell:"); ok {
				f.mu.Lock()
				f.cmds = append(f.cmds, cmd)
				f.mu.Unlock()
				out := []byte(f.shell(cmd))
				for len(out) > 0 {
					n := min(len(out), f.maxData)
					_ = a.send(adbWRTE, local, remote, out[:n])
					out = out[n:]
					if ok, _ := a.read(); ok.cmd != adbOKAY {
						return
					}
				}
				_ = a.send(adbCLSE, local, remote, nil)
			}
			syncBuf, syncFile = nil, ""
		case adbWRTE:
			if len(m.data) > f.maxData {
				f.t.Errorf("a write of %d bytes, over the TV's %d", len(m.data), f.maxData)
			}
			_ = a.send(adbOKAY, local, m.arg0, nil)
			syncBuf = append(syncBuf, m.data...)
			for len(syncBuf) >= 8 {
				id, n := string(syncBuf[:4]), int(binary.LittleEndian.Uint32(syncBuf[4:8]))
				if id == "DONE" || id == "QUIT" {
					syncBuf = syncBuf[8:]
					if id == "DONE" {
						resp := make([]byte, 8)
						copy(resp, "OKAY")
						_ = a.send(adbWRTE, local, m.arg0, resp)
						if ok, _ := a.read(); ok.cmd != adbOKAY {
							return
						}
					}
					continue
				}
				if len(syncBuf) < 8+n {
					break
				}
				body := syncBuf[8 : 8+n]
				switch id {
				case "SEND":
					syncFile = strings.SplitN(string(body), ",", 2)[0]
					f.mu.Lock()
					f.files[syncFile] = nil
					f.mu.Unlock()
				case "DATA":
					f.mu.Lock()
					f.files[syncFile] = append(f.files[syncFile], body...)
					f.mu.Unlock()
				}
				syncBuf = syncBuf[8+n:]
			}
		case adbCLSE:
			// The host is done with that stream.
		}
	}
}

func testADBKey(t *testing.T) []byte {
	t.Helper()
	k, err := os.ReadFile("testdata/adb/adbkey")
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestADBAllowOnceThenSignIn(t *testing.T) {
	f := newFakeADBD(t, true)
	key := testADBKey(t)
	a, err := DialADB(context.Background(), f.addr(), key, "Lootnest on PC", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(a.Banner, "Chromecast") {
		t.Fatalf("banner %q", a.Banner)
	}
	a.Close()
	// The second time the TV knows the key: no question.
	a, err = DialADB(context.Background(), f.addr(), key, "Lootnest on PC", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	a.Close()
	if f.prompts != 1 {
		t.Fatalf("the TV asked %d times, want 1", f.prompts)
	}
}

func TestADBNotAllowed(t *testing.T) {
	f := newFakeADBD(t, false)
	_, err := DialADB(context.Background(), f.addr(), testADBKey(t), "Lootnest on PC", 2*time.Second)
	if !errors.Is(err, ErrADBNotAllowed) {
		t.Fatalf("got %v, want ErrADBNotAllowed", err)
	}
}

func TestADBNotListening(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	ln.Close()
	_, err := DialADB(context.Background(), addr, testADBKey(t), "x", time.Second)
	if err == nil || !strings.Contains(err.Error(), "network debugging") {
		t.Fatalf("got %v", err)
	}
}

func TestADBShellLongOutput(t *testing.T) {
	f := newFakeADBD(t, true)
	long := strings.Repeat("line of output\r\n", 2000) // over several writes
	f.shell = func(cmd string) string {
		if cmd == "big" {
			return long
		}
		return "ok\n"
	}
	a, err := DialADB(context.Background(), f.addr(), testADBKey(t), "x", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	out, err := a.Shell("big", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if out != strings.ReplaceAll(long, "\r\n", "\n") {
		t.Fatalf("got %d bytes, want %d", len(out), len(long))
	}
	// A second command on the same connection.
	if out, err := a.Shell("small", 5*time.Second); err != nil || out != "ok\n" {
		t.Fatalf("%q %v", out, err)
	}
}

func TestADBPushAndInstall(t *testing.T) {
	f := newFakeADBD(t, true)
	installed := ""
	f.shell = func(cmd string) string {
		switch {
		case strings.HasPrefix(cmd, "pm install -r /data/local/tmp/lootnest-tv.apk"):
			f.mu.Lock()
			installed = string(f.files["/data/local/tmp/lootnest-tv.apk"])
			f.mu.Unlock()
			return "Performing Streamed Install\nSuccess\n"
		case strings.HasPrefix(cmd, "dumpsys package"):
			return "    versionName=1.0.0\n"
		}
		return ""
	}
	apk := filepath.Join(t.TempDir(), "app.apk")
	body := bytes.Repeat([]byte("0123456789abcdef"), 20000) // 320 KB, many writes
	if err := os.WriteFile(apk, body, 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := DialADB(context.Background(), f.addr(), testADBKey(t), "x", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if err := a.InstallAndroidApp(apk); err != nil {
		t.Fatal(err)
	}
	if installed != string(body) {
		t.Fatalf("the TV got %d bytes, want %d", len(installed), len(body))
	}
	if v := a.AndroidAppVersion(); v != "1.0.0" {
		t.Fatalf("version %q", v)
	}
	var rm bool
	for _, c := range f.cmds {
		rm = rm || strings.HasPrefix(c, "rm -f /data/local/tmp/lootnest-tv.apk")
	}
	if !rm {
		t.Fatal("the copy wasn't removed after the install")
	}
}

func TestADBLaunchRefusesOddInput(t *testing.T) {
	f := newFakeADBD(t, true)
	f.shell = func(cmd string) string {
		return "Starting: Intent { cmp=com.lootnest.tv/com.limelight.LootnestActivity }\n"
	}
	a, err := DialADB(context.Background(), f.addr(), testADBKey(t), "x", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if _, err := a.LaunchAndroidApp("192.168.1.5; reboot", "1234"); err == nil {
		t.Fatal("a host that isn't an address went to the shell")
	}
	if _, err := a.LaunchAndroidApp("192.168.1.5", "12a4"); err == nil {
		t.Fatal("a PIN that isn't four digits went to the shell")
	}
	if _, err := a.LaunchAndroidApp("192.168.1.5", "0420"); err != nil {
		t.Fatal(err)
	}
	last := f.cmds[len(f.cmds)-1]
	if !strings.Contains(last, "-n com.lootnest.tv/com.limelight.LootnestActivity --es host 192.168.1.5 --es pairPin 0420") {
		t.Fatalf("launched with %q", last)
	}
}

func TestInstallAnswer(t *testing.T) {
	for out, want := range map[string]string{
		"Success\n": "",
		"Failure [INSTALL_FAILED_UPDATE_INCOMPATIBLE: Package com.lootnest.tv signatures do not match]": "signed by someone else",
		"Failure [INSTALL_FAILED_INSUFFICIENT_STORAGE]":                                                 "out of space",
		"Failure [INSTALL_FAILED_SOMETHING_NEW]":                                                        "INSTALL_FAILED_SOMETHING_NEW",
		"":                                                                                              "didn't say",
	} {
		err := InstallAnswer(out)
		if want == "" {
			if err != nil {
				t.Errorf("%q: %v", out, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: got %v, want %q", out, err, want)
		}
	}
}

func TestAndroidAppStarted(t *testing.T) {
	log := "11-18 Screen: starting\n11-18 Pair: done\n11-18 Lootnest: starting the app\n11-18 Screen: starting\n"
	if AndroidAppStarted(log) {
		t.Fatal("the last start hasn't reached Lootnest yet")
	}
	if !AndroidAppStarted(log + "11-18 Screen: ready\n") {
		t.Fatal("ready after the last start")
	}
}

func TestADBAddress(t *testing.T) {
	if a, err := ADBAddress(" 192.168.1.40 "); err != nil || a != "192.168.1.40:5555" {
		t.Fatal(a, err)
	}
	if a, err := ADBAddress("192.168.1.40:5556"); err != nil || a != "192.168.1.40:5556" {
		t.Fatal(a, err)
	}
	if _, err := ADBAddress("tv; rm"); err == nil {
		t.Fatal("not an address")
	}
}

var _ = io.EOF
