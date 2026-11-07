//go:build windows

package app

import (
	"encoding/base64"
	"strings"

	"lootnest/internal/win"
)

// Keys the user saves in Settings are kept encrypted for their Windows
// account (DPAPI), as "dpapi:<base64>", never as plain text in config.json.
const secretPrefix = "dpapi:"

// protectSecret encrypts a key typed in Settings; one already encrypted
// (sent back unchanged by the UI) is kept as it is.
func protectSecret(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" || strings.HasPrefix(s, secretPrefix) {
		return s, nil
	}
	b, err := win.Protect([]byte(s))
	if err != nil {
		return "", err
	}
	return secretPrefix + base64.StdEncoding.EncodeToString(b), nil
}

// revealSecret decrypts a saved key ("" if it can't be read, such as a
// config copied from another PC or Windows user). Keys saved in plain text
// by earlier versions are read as they are and encrypted at the next start.
func revealSecret(s string) string {
	enc, ok := strings.CutPrefix(s, secretPrefix)
	if !ok {
		return s
	}
	b, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return ""
	}
	out, err := win.Unprotect(b)
	if err != nil {
		return ""
	}
	return string(out)
}
