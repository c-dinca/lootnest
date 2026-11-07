package core

import (
	"bytes"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
)

// Your year in games: the recap is drawn in the window (a canvas) and
// handed over as a PNG data URL, saved on this PC for the user to share.
// Nothing is sent anywhere.

const recapMaxBytes = 25 << 20

var pngMagic = []byte("\x89PNG\r\n\x1a\n")

// DecodePNGDataURL returns the PNG in a "data:image/png;base64," URL, or
// an error for anything else.
func DecodePNGDataURL(s string) ([]byte, error) {
	const prefix = "data:image/png;base64,"
	if !strings.HasPrefix(s, prefix) {
		return nil, errors.New("not a PNG picture")
	}
	if len(s)-len(prefix) > recapMaxBytes*4/3+4 {
		return nil, errors.New("the picture is too big")
	}
	b, err := base64.StdEncoding.DecodeString(s[len(prefix):])
	if err != nil || !bytes.HasPrefix(b, pngMagic) {
		return nil, errors.New("not a PNG picture")
	}
	return b, nil
}

// RecapFileName is the file a profile's recap of a year is saved as,
// such as "Player 1 in 2026.png".
func RecapFileName(profile string, year int) string {
	n := "Lootnest"
	if strings.TrimSpace(profile) != "" {
		n = strings.TrimSpace(safeFileName(profile))
	}
	if r := []rune(n); len(r) > 40 {
		n = string(r[:40])
	}
	return n + " in " + strconv.Itoa(year) + ".png"
}
