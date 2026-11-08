package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// Updates. Lootnest asks GitHub for its latest release only when the user
// presses "Check now" or turned on "Check for updates automatically".
// ---------------------------------------------------------------------------

// LatestReleaseURL is GitHub's public API for the newest published release.
const LatestReleaseURL = "https://api.github.com/repos/c-dinca/lootnest/releases/latest"

// SetupAsset is the release file the in-app update downloads.
const SetupAsset = "Lootnest-Setup.exe"

// releaseDownloads is where Lootnest's own release files live. An installer
// anywhere else, or one without a SHA-256 to check it against, is never
// offered, whatever the release says.
const releaseDownloads = "https://github.com/c-dinca/lootnest/releases/download/"

// releaseAssets is the same file through GitHub's API (asked for with
// Accept: application/octet-stream). Lootnest falls back to it when the
// download link fails, as github.com's release links can (504s seen on
// 8 November 2026 for this repository only, while the API served the file).
const releaseAssets = "https://api.github.com/repos/c-dinca/lootnest/releases/assets/"

type Release struct {
	Version  string `json:"version"` // "1.4.0"
	Page     string `json:"page"`    // the release's web page ("What's new")
	SetupURL string `json:"-"`
	AssetURL string `json:"-"` // the same file through GitHub's API, a fallback
	SHA256   string `json:"-"` // hex digest of the setup, when GitHub published one
	Size     int64  `json:"-"`
}

// ParseRelease reads GitHub's "latest release" reply.
func ParseRelease(b []byte) (Release, error) {
	var r struct {
		Tag        string `json:"tag_name"`
		Page       string `json:"html_url"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
		Assets     []struct {
			Name   string `json:"name"`
			URL    string `json:"browser_download_url"`
			API    string `json:"url"`
			Size   int64  `json:"size"`
			Digest string `json:"digest"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return Release{}, fmt.Errorf("unexpected reply from GitHub: %w", err)
	}
	if r.Tag == "" || r.Draft || r.Prerelease {
		return Release{}, fmt.Errorf("no published release found")
	}
	rel := Release{Version: strings.TrimPrefix(r.Tag, "v"), Page: r.Page}
	for _, a := range r.Assets {
		if strings.EqualFold(a.Name, SetupAsset) {
			d, ok := strings.CutPrefix(a.Digest, "sha256:")
			if !ok || len(d) != 64 || !strings.HasPrefix(a.URL, releaseDownloads) || a.Size <= 0 {
				continue
			}
			rel.SetupURL, rel.Size, rel.SHA256 = a.URL, a.Size, strings.ToLower(d)
			if strings.HasPrefix(a.API, releaseAssets) {
				rel.AssetURL = a.API
			}
		}
	}
	return rel, nil
}

// LatestRelease asks GitHub for the newest published release.
func LatestRelease(fetch Fetch) (Release, error) {
	b, err := fetch(LatestReleaseURL)
	if err != nil {
		return Release{}, err
	}
	if len(b) == 0 {
		return Release{}, fmt.Errorf("no published release found")
	}
	return ParseRelease(b)
}

// NewerVersion reports whether latest ("1.4.1" or "v1.4.1") is newer than
// current. Versions that don't parse are never newer.
func NewerVersion(current, latest string) bool {
	parse := func(v string) ([3]int, bool) {
		var out [3]int
		parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(v), "v"), ".")
		if len(parts) == 0 || len(parts) > 3 {
			return out, false
		}
		for i, p := range parts {
			n, err := strconv.Atoi(p)
			if err != nil || n < 0 {
				return out, false
			}
			out[i] = n
		}
		return out, true
	}
	c, ok1 := parse(current)
	l, ok2 := parse(latest)
	if !ok1 || !ok2 {
		return false
	}
	for i := 0; i < 3; i++ {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

// VerifyFile checks a downloaded file's size and, when known, its SHA-256.
func VerifyFile(path string, size int64, sha string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return err
	}
	if size > 0 && n != size {
		return fmt.Errorf("the download is incomplete (%d of %d bytes)", n, size)
	}
	if sha != "" && hex.EncodeToString(h.Sum(nil)) != sha {
		return fmt.Errorf("the download doesn't match the release (checksum differs)")
	}
	return nil
}
