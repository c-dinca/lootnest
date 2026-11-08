// plugsign signs Lootnest plugins (docs/PLUGINS.md, Signing).
//
//	go run ./cmd/plugsign keygen              write plugsign.key (private) here
//	go run ./cmd/plugsign sign <dir> [key]    write <dir>/signature.json with plugsign.key, or key
package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"lootnest/internal/core"
)

func main() {
	var err error
	switch {
	case len(os.Args) == 2 && os.Args[1] == "keygen":
		err = keygen()
	case (len(os.Args) == 3 || len(os.Args) == 4) && os.Args[1] == "sign":
		key := "plugsign.key"
		if len(os.Args) == 4 {
			key = os.Args[3]
		}
		err = sign(os.Args[2], key)
	default:
		err = errors.New("usage: plugsign keygen | plugsign sign <plugin folder> [private key file]")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// keygen never replaces a key file: a lost private key can't sign the
// next version with the same key.
func keygen() error {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return err
	}
	w, err := os.OpenFile("plugsign.key", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("%v; keygen never replaces a key", err)
	}
	_, err = w.WriteString(base64.StdEncoding.EncodeToString(priv) + "\n")
	if cerr := w.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	fmt.Println("Wrote plugsign.key (keep it private, never commit it). Key", core.KeyFingerprint(pub))
	return nil
}

func sign(dir, keyFile string) error {
	b, err := os.ReadFile(keyFile)
	if err != nil {
		return err
	}
	priv, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(b)))
	if err != nil || len(priv) != ed25519.PrivateKeySize {
		return errors.New(keyFile + " isn't a private key from plugsign keygen")
	}
	if _, err := core.CheckPluginDir(dir); err != nil {
		return fmt.Errorf("Lootnest would refuse this plugin: %w", err)
	}
	if err := core.SignPlugin(dir, ed25519.PrivateKey(priv)); err != nil {
		return err
	}
	fmt.Println("Signed", dir, "with key", core.KeyFingerprint(ed25519.PrivateKey(priv).Public().(ed25519.PublicKey)))
	return nil
}
