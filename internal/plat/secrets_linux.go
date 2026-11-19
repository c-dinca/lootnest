package plat

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"lootnest/internal/core"
	"lootnest/internal/desk"
)

// Keys the user gives Lootnest (a Steam Web API key, store tokens) are kept
// encrypted, as DPAPI does on Windows: AES-256-GCM with a key Lootnest keeps
// in the desktop's keyring (the Secret Service: GNOME Keyring, KWallet).
// Without a keyring the key is a file only this user can read, in
// Lootnest's data folder.

const (
	secretMagic   = "LNX1"
	secretKeyName = "lootnest-secrets"
)

var (
	secretMu  sync.Mutex
	secretKey []byte
)

// keyStore reaches where the key is kept; tests replace it.
var keyStore = struct {
	lookup func() (string, error)
	store  func(string) error
	file   func() string
}{
	lookup: func() (string, error) { return desk.KeyLookup(secretKeyName) },
	store:  func(v string) error { return desk.KeyStore(secretKeyName, "Lootnest", v) },
	file: func() string {
		return filepath.Join(core.XDGDataHome(home(), os.Getenv), "lootnest", "secret.key")
	},
}

// key is the encryption key: from the keyring, else from the key file,
// made the first time.
func key() ([]byte, error) {
	secretMu.Lock()
	defer secretMu.Unlock()
	if secretKey != nil {
		return secretKey, nil
	}
	decode := func(s string) []byte {
		b, err := base64.StdEncoding.DecodeString(s)
		if err != nil || len(b) != 32 {
			return nil
		}
		return b
	}
	v, kerr := keyStore.lookup()
	if kerr == nil && v != "" {
		if k := decode(v); k != nil {
			secretKey = k
			return k, nil
		}
	}
	file := keyStore.file()
	if b, err := os.ReadFile(file); err == nil {
		if k := decode(string(bytes.TrimSpace(b))); k != nil {
			secretKey = k
			return k, nil
		}
	}
	if kerr == nil && v != "" {
		return nil, errors.New("the keyring holds a key Lootnest can't read")
	}
	k := make([]byte, 32)
	if _, err := rand.Read(k); err != nil {
		return nil, err
	}
	enc := base64.StdEncoding.EncodeToString(k)
	if kerr != nil || keyStore.store(enc) != nil {
		if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
			return nil, err
		}
		f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return nil, err
		}
		_, werr := f.WriteString(enc + "\n")
		if cerr := f.Close(); werr == nil {
			werr = cerr
		}
		if werr != nil {
			return nil, werr
		}
	}
	secretKey = k
	return k, nil
}

// Protect encrypts b for this user.
func Protect(b []byte) ([]byte, error) {
	k, err := key()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(k)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	out := append([]byte(secretMagic), nonce...)
	return gcm.Seal(out, nonce, b, []byte(secretMagic)), nil
}

// Unprotect decrypts what Protect made.
func Unprotect(b []byte) ([]byte, error) {
	if !bytes.HasPrefix(b, []byte(secretMagic)) {
		return nil, errors.New("not something Lootnest encrypted on Linux")
	}
	k, err := key()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(k)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	rest := b[len(secretMagic):]
	if len(rest) < gcm.NonceSize() {
		return nil, errors.New("too short")
	}
	return gcm.Open(nil, rest[:gcm.NonceSize()], rest[gcm.NonceSize():], []byte(secretMagic))
}
