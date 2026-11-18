package core

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	mathbig "math/big"
	"net"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// ADB over the network, for Android TV, Google TV and Fire TV (Lootnest
// Link M4a). With the TV's network debugging on (Developer options), its
// adbd listens on port 5555. Lootnest speaks the device side of the ADB
// protocol itself, with no adb.exe: a CNXN handshake signed with
// Lootnest's own RSA key (the TV asks once to allow this computer), then
// one stream at a time for shell commands and the sync service that
// copies the app's package over. See AOSP's adb/protocol.txt and
// adb/SYNC.TXT.
// ---------------------------------------------------------------------------

const (
	adbCNXN = 0x4e584e43
	adbAUTH = 0x48545541
	adbOPEN = 0x4e45504f
	adbOKAY = 0x59414b4f
	adbCLSE = 0x45534c43
	adbWRTE = 0x45545257

	adbVersion = 0x01000001
	adbMaxData = 256 * 1024

	adbAuthToken     = 1
	adbAuthSignature = 2
	adbAuthPublicKey = 3

	// ADBPort is where a TV's adbd listens with network debugging on.
	ADBPort = 5555
)

// ErrADBNotAllowed: the TV didn't allow this computer in time.
var ErrADBNotAllowed = errors.New("the TV didn't allow this computer: when it asks \"Allow USB debugging?\", choose Always allow from this computer, then try again")

type adbMsg struct {
	cmd, arg0, arg1 uint32
	data            []byte
}

// ADB is a connection to one device's adbd.
type ADB struct {
	conn    net.Conn
	mu      sync.Mutex // one stream at a time
	maxData int
	nextID  uint32
	Banner  string // what the device said about itself in CNXN
}

func adbSum(b []byte) uint32 {
	var s uint32
	for _, c := range b {
		s += uint32(c)
	}
	return s
}

func (a *ADB) send(cmd, arg0, arg1 uint32, data []byte) error {
	h := make([]byte, 24)
	binary.LittleEndian.PutUint32(h[0:], cmd)
	binary.LittleEndian.PutUint32(h[4:], arg0)
	binary.LittleEndian.PutUint32(h[8:], arg1)
	binary.LittleEndian.PutUint32(h[12:], uint32(len(data)))
	binary.LittleEndian.PutUint32(h[16:], adbSum(data)) // older adbd checks it, newer ones ignore it
	binary.LittleEndian.PutUint32(h[20:], cmd^0xffffffff)
	if _, err := a.conn.Write(append(h, data...)); err != nil {
		return netErr("the TV", err)
	}
	return nil
}

func (a *ADB) read() (adbMsg, error) {
	h := make([]byte, 24)
	if _, err := io.ReadFull(a.conn, h); err != nil {
		return adbMsg{}, netErr("the TV", err)
	}
	m := adbMsg{cmd: binary.LittleEndian.Uint32(h[0:]), arg0: binary.LittleEndian.Uint32(h[4:]), arg1: binary.LittleEndian.Uint32(h[8:])}
	n := binary.LittleEndian.Uint32(h[12:])
	if binary.LittleEndian.Uint32(h[20:]) != m.cmd^0xffffffff {
		return adbMsg{}, errors.New("the TV sent something that isn't ADB")
	}
	if n > 1<<20 {
		return adbMsg{}, errors.New("the TV sent a message too large for ADB")
	}
	m.data = make([]byte, n)
	if _, err := io.ReadFull(a.conn, m.data); err != nil {
		return adbMsg{}, netErr("the TV", err)
	}
	return m, nil
}

// NewADBKey makes the key Lootnest signs in with, as PEM.
func NewADBKey() ([]byte, error) {
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}), nil
}

func parseADBKey(p []byte) (*rsa.PrivateKey, error) {
	b, _ := pem.Decode(p)
	if b == nil {
		return nil, errors.New("Lootnest's key for Android TVs is damaged")
	}
	if k, err := x509.ParsePKCS1PrivateKey(b.Bytes); err == nil {
		return k, nil
	}
	k, err := x509.ParsePKCS8PrivateKey(b.Bytes) // as adb keygen writes it
	if err != nil {
		return nil, err
	}
	rk, ok := k.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("Lootnest's key for Android TVs isn't an RSA key")
	}
	return rk, nil
}

// ADBPublicKey is the key in the form adbd keeps in adb_keys: Android's
// RSAPublicKey struct (word count, n0inv, n and R^2 mod n as little-endian
// 32-bit words, the exponent), in base64, then a name.
func ADBPublicKey(k *rsa.PublicKey, name string) ([]byte, error) {
	if k.N.BitLen() != 2048 {
		return nil, errors.New("ADB keys are 2048-bit")
	}
	const words = 64
	var b bytes.Buffer
	w := func(v uint32) { _ = binary.Write(&b, binary.LittleEndian, v) }
	r32 := new(mathbig.Int).Lsh(mathbig.NewInt(1), 32)
	n0 := new(mathbig.Int).Mod(k.N, r32)
	n0inv := new(mathbig.Int).ModInverse(n0, r32)
	n0inv.Sub(r32, n0inv) // -1/n[0] mod 2^32
	w(words)
	w(uint32(n0inv.Uint64()))
	le := func(x *mathbig.Int) {
		be := x.FillBytes(make([]byte, words*4))
		for i := words - 1; i >= 0; i-- {
			w(binary.BigEndian.Uint32(be[i*4:]))
		}
	}
	le(k.N)
	rr := new(mathbig.Int).Lsh(mathbig.NewInt(1), 2*2048)
	le(rr.Mod(rr, k.N))
	w(uint32(k.E))
	return []byte(base64.StdEncoding.EncodeToString(b.Bytes()) + " " + name), nil
}

// DialADB connects to adbd at addr and signs in with keyPEM. A TV that
// doesn't know the key yet shows "Allow USB debugging?"; DialADB waits up
// to allow for the person to say yes. name is how the TV lists this
// computer.
func DialADB(ctx context.Context, addr string, keyPEM []byte, name string, allow time.Duration) (*ADB, error) {
	key, err := parseADBKey(keyPEM)
	if err != nil {
		return nil, err
	}
	d := net.Dialer{Timeout: 5 * time.Second}
	c, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("the TV doesn't answer on %s: turn on network debugging (or ADB debugging) in its Developer options (%v)", addr, err)
	}
	a := &ADB{conn: c, maxData: 4096, nextID: 1}
	fail := func(err error) (*ADB, error) { c.Close(); return nil, err }
	_ = c.SetDeadline(time.Now().Add(10 * time.Second))
	if err := a.send(adbCNXN, adbVersion, adbMaxData, []byte("host::\x00")); err != nil {
		return fail(err)
	}
	signed, offered := false, false
	for {
		m, err := a.read()
		if err != nil {
			if offered {
				return fail(ErrADBNotAllowed)
			}
			return fail(err)
		}
		switch m.cmd {
		case adbCNXN:
			_ = c.SetDeadline(time.Time{})
			a.Banner = strings.TrimRight(string(m.data), "\x00")
			if m.arg1 > 0 && m.arg1 < adbMaxData {
				a.maxData = int(m.arg1)
			} else {
				a.maxData = adbMaxData
			}
			return a, nil
		case adbAUTH:
			if m.arg0 != adbAuthToken {
				continue
			}
			if !signed {
				signed = true
				if len(m.data) != 20 {
					return fail(errors.New("the TV asked to sign something ADB doesn't use"))
				}
				sig, err := rsa.SignPKCS1v15(nil, key, crypto.SHA1, m.data)
				if err != nil {
					return fail(err)
				}
				if err := a.send(adbAUTH, adbAuthSignature, 0, sig); err != nil {
					return fail(err)
				}
				continue
			}
			if offered {
				return fail(ErrADBNotAllowed)
			}
			// The TV doesn't know the key: offer it, and wait while the
			// TV asks the person.
			pub, err := ADBPublicKey(&key.PublicKey, name)
			if err != nil {
				return fail(err)
			}
			offered = true
			_ = c.SetDeadline(time.Now().Add(allow))
			if err := a.send(adbAUTH, adbAuthPublicKey, 0, append(pub, 0)); err != nil {
				return fail(err)
			}
		default:
			// Nothing else comes before CNXN.
		}
	}
}

// Close ends the connection.
func (a *ADB) Close() error { return a.conn.Close() }

// stream opens service and calls fn with a writer and a reader for it,
// then closes it. Writes wait for the TV's OKAY, as ADB asks.
func (a *ADB) stream(service string, limit time.Duration, fn func(write func([]byte) error, read func() ([]byte, error)) error) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	_ = a.conn.SetDeadline(time.Now().Add(limit))
	defer a.conn.SetDeadline(time.Time{})
	local := a.nextID
	a.nextID++
	if err := a.send(adbOPEN, local, 0, append([]byte(service), 0)); err != nil {
		return err
	}
	var remote uint32
	closed := false
	pending := [][]byte{} // what the TV wrote while we waited for an OKAY
	next := func() (adbMsg, error) {
		for {
			m, err := a.read()
			if err != nil {
				return m, err
			}
			if m.arg1 != local { // another stream's leftovers
				continue
			}
			return m, nil
		}
	}
	for remote == 0 {
		m, err := next()
		if err != nil {
			return err
		}
		switch m.cmd {
		case adbOKAY:
			remote = m.arg0
		case adbCLSE:
			return fmt.Errorf("the TV refused %q", strings.SplitN(service, ":", 2)[0])
		}
	}
	write := func(b []byte) error {
		for len(b) > 0 {
			if closed {
				return errors.New("the TV closed the connection")
			}
			n := min(len(b), a.maxData)
			if err := a.send(adbWRTE, local, remote, b[:n]); err != nil {
				return err
			}
			b = b[n:]
			for {
				m, err := next()
				if err != nil {
					return err
				}
				if m.cmd == adbOKAY {
					break
				}
				if m.cmd == adbWRTE {
					pending = append(pending, m.data)
					if err := a.send(adbOKAY, local, remote, nil); err != nil {
						return err
					}
					continue
				}
				if m.cmd == adbCLSE {
					closed = true
					return errors.New("the TV closed the connection")
				}
			}
		}
		return nil
	}
	read := func() ([]byte, error) {
		if len(pending) > 0 {
			d := pending[0]
			pending = pending[1:]
			return d, nil
		}
		if closed {
			return nil, io.EOF
		}
		for {
			m, err := next()
			if err != nil {
				return nil, err
			}
			switch m.cmd {
			case adbWRTE:
				if err := a.send(adbOKAY, local, remote, nil); err != nil {
					return nil, err
				}
				return m.data, nil
			case adbCLSE:
				closed = true
				return nil, io.EOF
			}
		}
	}
	err := fn(write, read)
	if !closed {
		_ = a.send(adbCLSE, local, remote, nil)
	}
	return err
}

// Shell runs cmd on the TV and returns what it printed.
func (a *ADB) Shell(cmd string, limit time.Duration) (string, error) {
	var out bytes.Buffer
	err := a.stream("shell:"+cmd, limit, func(_ func([]byte) error, read func() ([]byte, error)) error {
		for {
			b, err := read()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
			if out.Len() < 1<<20 {
				out.Write(b)
			}
		}
	})
	return strings.ReplaceAll(out.String(), "\r\n", "\n"), err
}

// Push copies size bytes from r to remote on the TV with the sync service.
func (a *ADB) Push(r io.Reader, remote string, mode uint32, limit time.Duration) error {
	return a.stream("sync:", limit, func(write func([]byte) error, read func() ([]byte, error)) error {
		req := func(id string, n uint32, body []byte) []byte {
			b := make([]byte, 8, 8+len(body))
			copy(b, id)
			binary.LittleEndian.PutUint32(b[4:], n)
			return append(b, body...)
		}
		spec := fmt.Sprintf("%s,%d", remote, mode)
		if err := write(req("SEND", uint32(len(spec)), []byte(spec))); err != nil {
			return err
		}
		buf := make([]byte, 64*1024)
		for {
			n, err := io.ReadFull(r, buf)
			if n > 0 {
				if werr := write(req("DATA", uint32(n), buf[:n])); werr != nil {
					return werr
				}
			}
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			if err != nil {
				return err
			}
		}
		if err := write(req("DONE", uint32(time.Now().Unix()), nil)); err != nil {
			return err
		}
		var got []byte
		for len(got) < 8 {
			b, err := read()
			if err != nil {
				return fmt.Errorf("the TV didn't say whether the copy arrived (%v)", err)
			}
			got = append(got, b...)
		}
		if string(got[:4]) == "FAIL" {
			n := int(binary.LittleEndian.Uint32(got[4:8]))
			for len(got) < 8+n {
				b, err := read()
				if err != nil {
					break
				}
				got = append(got, b...)
			}
			return fmt.Errorf("the TV refused the copy: %s", string(got[8:min(len(got), 8+n)]))
		}
		if string(got[:4]) != "OKAY" {
			return errors.New("the TV answered the copy with something unknown")
		}
		_ = write(req("QUIT", 0, nil))
		return nil
	})
}

// ADBDevice is what a TV says about itself.
type ADBDevice struct {
	Model, Maker, Name, Android string
}

// Device reads the TV's maker, model, name and Android version.
func (a *ADB) Device() ADBDevice {
	out, _ := a.Shell("getprop ro.product.manufacturer; getprop ro.product.model; settings get global device_name; getprop ro.build.version.release", 15*time.Second)
	l := strings.Split(strings.TrimSpace(out), "\n")
	for len(l) < 4 {
		l = append(l, "")
	}
	clean := func(s string) string {
		s = strings.TrimSpace(s)
		if s == "null" {
			return ""
		}
		return s
	}
	return ADBDevice{Maker: clean(l[0]), Model: clean(l[1]), Name: clean(l[2]), Android: clean(l[3])}
}

// Label is a name for the TV in Lootnest.
func (d ADBDevice) Label() string {
	if d.Name != "" {
		return d.Name
	}
	if d.Model != "" {
		if d.Maker != "" && !strings.Contains(strings.ToLower(d.Model), strings.ToLower(d.Maker)) {
			return d.Maker + " " + d.Model
		}
		return d.Model
	}
	return "Android TV"
}
