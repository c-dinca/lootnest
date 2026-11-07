package core

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/bodgit/sevenzip"
	"github.com/nwaples/rardecode/v2"
)

// ---------------------------------------------------------------------------
// Updates in .7z and .rar archives, and the versions an update names.
//
// A .7z or .rar is extracted first into a staging folder Lootnest owns, with
// the same path checks as a .zip, and then planned and applied like a
// folder. Multi-part .rar sets (.part1.rar, .part2.rar, ...) are read from
// the first part.
// ---------------------------------------------------------------------------

// NeedsStaging says whether an update must be extracted before it's planned.
func NeedsStaging(src string) bool {
	switch strings.ToLower(filepath.Ext(src)) {
	case ".7z", ".rar":
		return true
	}
	return false
}

// StageUpdate extracts a .zip, .7z or .rar into dst (made empty first) and returns
// dst, to plan as a folder. A password-protected archive can't be read.
func StageUpdate(src, dst string) (string, error) {
	_ = os.RemoveAll(dst)
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return "", err
	}
	var err error
	switch strings.ToLower(filepath.Ext(src)) {
	case ".7z":
		err = stage7z(src, dst)
	case ".rar":
		err = stageRar(src, dst)
	case ".zip":
		err = stageZip(src, dst)
	default:
		err = errors.New("not a .zip, .7z or .rar")
	}
	if err != nil {
		_ = os.RemoveAll(dst)
		return "", err
	}
	return dst, nil
}

func stageFile(dst, name string, r io.Reader) error {
	rel, err := updateRel(name)
	if err != nil {
		return err
	}
	p := filepath.Join(dst, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, r) // the archive checks its own CRC as it's read
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

func stage7z(src, dst string) error {
	r, err := sevenzip.OpenReader(src)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "password") {
			return errors.New("that .7z has a password; Lootnest can't open it")
		}
		return errors.New("that .7z can't be opened; it may be damaged or incomplete")
	}
	defer r.Close()
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("%s: %w", f.Name, err)
		}
		err = stageFile(dst, f.Name, rc)
		rc.Close()
		if err != nil {
			return fmt.Errorf("%s: %w", f.Name, err)
		}
	}
	return nil
}

func stageRar(src, dst string) error {
	r, err := rardecode.OpenReader(src)
	if err != nil {
		return errors.New("that .rar can't be opened; it may be damaged, or a part of it missing")
	}
	defer r.Close()
	for {
		h, err := r.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			if errors.Is(err, rardecode.ErrArchivedFileEncrypted) || strings.Contains(strings.ToLower(err.Error()), "password") {
				return errors.New("that .rar has a password; Lootnest can't open it")
			}
			return fmt.Errorf("reading the .rar: %w", err)
		}
		if h.IsDir {
			continue
		}
		if err := stageFile(dst, h.Name, r); err != nil {
			return fmt.Errorf("%s: %w", h.Name, err)
		}
	}
}

// ------------------------------------------------------------- versions ---

var (
	verRe     = regexp.MustCompile(`(?i)(?:^|[^a-z0-9])v?(\d+(?:\.\d+){1,3}[a-z]?)(?:[^0-9.]|$)`)
	buildRe   = regexp.MustCompile(`(?i)\bbuild[\s._-]*(\d{4,})\b`)
	fromToRe  = regexp.MustCompile(`(?i)v?(\d+(?:\.\d+){1,3}[a-z]?)[\s._]*(?:to|-|–|→|->)[\s._]*v?(\d+(?:\.\d+){1,3}[a-z]?)`)
	buildToRe = regexp.MustCompile(`(?i)build[\s._-]*(\d{4,})[\s._]*(?:to|-|→|->)[\s._]*(?:build[\s._-]*)?(\d{4,})`)
)

// UpdateVersions reads the versions an update's name gives: "v1.0.2 to
// v1.0.3" (from, to), "Update v1.0.3" (to only), or Steam-style build
// numbers ("Build 16251443"). Empty when the name has none.
func UpdateVersions(name string) (from, to string) {
	n := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	n = strings.TrimSuffix(n, ".part1")
	if m := buildToRe.FindStringSubmatch(n); m != nil {
		return "build " + m[1], "build " + m[2]
	}
	if m := fromToRe.FindStringSubmatch(n); m != nil && CompareVersions(m[1], m[2]) < 0 {
		return m[1], m[2]
	}
	if m := buildRe.FindStringSubmatch(n); m != nil {
		return "", "build " + m[1]
	}
	if m := verRe.FindStringSubmatch(n); m != nil {
		return "", m[1]
	}
	return "", ""
}

// FolderVersion is the version a game's folder name gives, such as
// "Game.v1.0.2-GROUP" or "Game Build 16251443", or "".
func FolderVersion(dir string) string {
	n := filepath.Base(dir)
	if m := buildRe.FindStringSubmatch(n); m != nil {
		return "build " + m[1]
	}
	if m := verRe.FindStringSubmatch(n); m != nil {
		return m[1]
	}
	return ""
}

// CompareVersions orders two versions ("1.0.10" after "1.0.9", "build 2"
// after "build 1"): -1, 0 or 1. Versions of different kinds compare equal.
func CompareVersions(a, b string) int {
	ab, bb := strings.HasPrefix(a, "build "), strings.HasPrefix(b, "build ")
	if ab != bb {
		return 0
	}
	a, b = strings.TrimPrefix(a, "build "), strings.TrimPrefix(b, "build ")
	pa, pb := strings.Split(strings.ToLower(a), "."), strings.Split(strings.ToLower(b), ".")
	for i := 0; i < len(pa) || i < len(pb); i++ {
		var x, y string
		if i < len(pa) {
			x = pa[i]
		}
		if i < len(pb) {
			y = pb[i]
		}
		xn, xs := splitNum(x)
		yn, ys := splitNum(y)
		switch {
		case xn != yn:
			if xn < yn {
				return -1
			}
			return 1
		case xs != ys:
			if xs < ys {
				return -1
			}
			return 1
		}
	}
	return 0
}

func splitNum(s string) (int, string) {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	n, _ := strconv.Atoi(s[:i])
	return n, s[i:]
}

// VersionCheck says whether an update fits the copy it goes into.
type VersionCheck struct {
	Have  string `json:"have,omitempty"` // this copy's version, as far as Lootnest knows
	From  string `json:"from,omitempty"` // the version the update is for
	To    string `json:"to,omitempty"`   // the version it brings
	State string `json:"state"`          // "ok", "skips", "older", "same", "unknown"
}

// CheckVersions compares an update's versions with this copy's.
func CheckVersions(have, from, to string) VersionCheck {
	c := VersionCheck{Have: have, From: from, To: to, State: "unknown"}
	switch {
	case have == "" || to == "":
		// nothing to compare
	case CompareVersions(to, have) == 0 && !strings.EqualFold(to, have):
		// different kinds of version: can't tell
	case strings.EqualFold(to, have):
		c.State = "same"
	case CompareVersions(to, have) < 0:
		c.State = "older"
	case from != "" && !strings.EqualFold(from, have) && CompareVersions(from, have) != 0:
		c.State = "skips" // made for another version than this copy's
	default:
		c.State = "ok"
	}
	return c
}

// CopyVersion is this copy's version: the last applied update's, else the
// folder name's.
func CopyVersion(dir string, applied []AppliedUpdate) string {
	for _, u := range applied { // newest first
		if u.State == "applied" && u.To != "" {
			return u.To
		}
	}
	return FolderVersion(dir)
}

func stageZip(src, dst string) error {
	zr, err := zip.OpenReader(src)
	if err != nil {
		return errors.New("that .zip can't be opened; it may be damaged")
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if f.Flags&0x1 != 0 {
			return errors.New("that .zip has a password; extract it first, then choose the folder")
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		err = stageFile(dst, f.Name, rc)
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
