package core

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ---------------------------------------------------------------------------
// Plugins, phase 3: signatures. A plugin may carry signature.json at its
// top, an Ed25519 key and its signature over PluginDigest. The key is
// pinned per plugin id at its first signed install (plugins.json, "keys"),
// trust on first use as with SSH host keys: no network, no authority.
// ---------------------------------------------------------------------------

const pluginSigFile = "signature.json"

type pluginSignature struct {
	Key string `json:"key"`
	Sig string `json:"sig"`
}

var ErrPluginSignature = errors.New("the signature doesn't match its files, so something changed them after the author signed it")

// PluginKeyChange refuses a version signed with another key than the one
// pinned. Pinned and Key are fingerprints.
type PluginKeyChange struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Pinned string `json:"pinned"`
	Key    string `json:"key"`
	what   string
}

func (e *PluginKeyChange) Error() string {
	return fmt.Sprintf("%s signed with a different key (%s) than the one you installed (%s)", e.what, e.Key, e.Pinned)
}

// PluginDigest is what a plugin's signature signs: a line
// "path\tsha256\n" for every file but signature.json, sorted, with
// forward slashes.
func PluginDigest(dir string) ([]byte, error) {
	var lines []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		rel = filepath.ToSlash(rel)
		if strings.EqualFold(rel, pluginSigFile) {
			return nil
		}
		if !d.Type().IsRegular() {
			return errors.New("it has a link (" + rel + ")")
		}
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return err
		}
		lines = append(lines, rel+"\t"+hex.EncodeToString(h.Sum(nil))+"\n")
		return nil
	})
	sort.Strings(lines)
	return []byte(strings.Join(lines, "")), err
}

// KeyFingerprint is how a key is shown: the first 16 hex digits of its
// SHA-256, in groups of four.
func KeyFingerprint(key []byte) string {
	sum := sha256.Sum256(key)
	h := strings.ToUpper(hex.EncodeToString(sum[:8]))
	return h[0:4] + " " + h[4:8] + " " + h[8:12] + " " + h[12:16]
}

// pluginKey is the key a plugin is signed with, once the signature checks
// out against its files; nil for an unsigned plugin.
func pluginKey(dir string) (ed25519.PublicKey, error) {
	b, err := os.ReadFile(filepath.Join(dir, pluginSigFile))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var s pluginSignature
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, errors.New("signature.json can't be read: " + shortErr(err))
	}
	key, err1 := base64.StdEncoding.DecodeString(s.Key)
	sig, err2 := base64.StdEncoding.DecodeString(s.Sig)
	if err1 != nil || err2 != nil || len(key) != ed25519.PublicKeySize || len(sig) != ed25519.SignatureSize {
		return nil, errors.New("signature.json doesn't hold an Ed25519 key and signature")
	}
	d, err := PluginDigest(dir)
	if err != nil {
		return nil, err
	}
	if !ed25519.Verify(key, d, sig) {
		return nil, ErrPluginSignature
	}
	return key, nil
}

// VerifyPlugin checks a plugin's signature against its files, and returns
// its key's fingerprint; signed is false, with no error, for a plugin with
// no signature.json.
func VerifyPlugin(dir string) (fingerprint string, signed bool, err error) {
	key, err := pluginKey(dir)
	if err != nil || key == nil {
		return "", false, err
	}
	return KeyFingerprint(key), true, nil
}

// SignPlugin writes signature.json for a plugin's folder, for plugin
// authors (cmd/plugsign).
func SignPlugin(dir string, priv ed25519.PrivateKey) error {
	d, err := PluginDigest(dir)
	if err != nil {
		return err
	}
	pub := priv.Public().(ed25519.PublicKey)
	return writeJSONAtomic(filepath.Join(dir, pluginSigFile), pluginSignature{
		Key: base64.StdEncoding.EncodeToString(pub),
		Sig: base64.StdEncoding.EncodeToString(ed25519.Sign(priv, d)),
	})
}

// checkPluginPin compares a plugin's key with the one pinned for its id
// (base64, "" for none). what names the version and its verb in the
// reason ("this version is").
func checkPluginPin(m *PluginManifest, pinned string, key ed25519.PublicKey, what string) error {
	if pinned == "" {
		return nil
	}
	pin, _ := base64.StdEncoding.DecodeString(pinned)
	if key == nil {
		return fmt.Errorf("%s not signed, but the version you have is signed with key %s", what, KeyFingerprint(pin))
	}
	if !key.Equal(ed25519.PublicKey(pin)) {
		return &PluginKeyChange{ID: m.ID, Name: m.Name, Pinned: KeyFingerprint(pin), Key: KeyFingerprint(key), what: what}
	}
	return nil
}
