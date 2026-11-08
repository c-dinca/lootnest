package core

import (
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Screenshots: pictures of a running game, taken from outside it (win's
// GDI capture), kept in Pictures\Lootnest\Screenshots\<game>, never in the
// game's folder. A save point made while its game runs keeps one too, as
// point.png and thumb.png beside its point.json, outside the files it puts
// back.
// ---------------------------------------------------------------------------

const (
	PointPicture = "point.png"
	PointThumb   = "thumb.png"
	ThumbWidth   = 320
	shotStamp    = "2006-01-02 15-04-05"
)

var reservedNames = regexp.MustCompile(`(?i)^(con|prn|aux|nul|com[0-9]|lpt[0-9])(\..*)?$`)

// SafeFileName makes name usable as one file or folder name on Windows: the
// characters Windows forbids are dropped, trailing dots and spaces too, and
// a reserved device name (CON, LPT1) gets an underscore.
func SafeFileName(name string) string {
	var b strings.Builder
	for _, r := range name {
		if r >= 32 && !strings.ContainsRune(`<>:"/\|?*`, r) {
			b.WriteRune(r)
		}
	}
	s := strings.TrimSpace(b.String())
	if r := []rune(s); len(r) > 100 {
		s = string(r[:100])
	}
	s = strings.TrimRight(s, ". ")
	if s == "" {
		return "Game"
	}
	if reservedNames.MatchString(s) {
		s += "_"
	}
	return s
}

// ScreenshotsDir is where a game's screenshots go.
func ScreenshotsDir(picturesDir, gameName string) string {
	return filepath.Join(picturesDir, "Lootnest", "Screenshots", SafeFileName(gameName))
}

// SaveNewScreenshot writes img as a screenshot taken at when and returns its
// path: named by the time, then " (2)" and on if a picture of that second is
// already there. The name is taken (created empty, only if it's free) before
// the picture is moved onto it, so two pictures of one second never replace
// each other.
func SaveNewScreenshot(img image.Image, picturesDir, gameName string, when time.Time) (string, error) {
	dir, base := ScreenshotsDir(picturesDir, gameName), when.Format(shotStamp)
	tmp, err := encodeTemp(img, dir)
	if err != nil {
		return "", err
	}
	for n := 1; ; n++ {
		p := filepath.Join(dir, base+".png")
		if n > 1 {
			p = filepath.Join(dir, fmt.Sprintf("%s (%d).png", base, n))
		}
		f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err == nil {
			f.Close()
			if err = os.Rename(tmp, p); err != nil {
				_ = os.Remove(p)
			}
		}
		if err != nil {
			_ = os.Remove(tmp)
			return "", err
		}
		return p, nil
	}
}

// SaveScreenshotPNG writes img to path as a PNG, through a temporary file,
// so a failure leaves nothing half-written.
func SaveScreenshotPNG(img image.Image, path string) error {
	tmp, err := encodeTemp(img, filepath.Dir(path))
	if err != nil {
		return err
	}
	if err = os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
	}
	return err
}

// encodeTemp writes img as a PNG to a new temporary file in dir.
func encodeTemp(img image.Image, dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	f, err := os.CreateTemp(dir, ".lootnest-*"+tmpSuffix)
	if err != nil {
		return "", err
	}
	err = png.Encode(f, img)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

func toRGBA(img image.Image) *image.RGBA {
	if m, ok := img.(*image.RGBA); ok && m.Rect.Min == (image.Point{}) {
		return m
	}
	b := img.Bounds()
	m := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(m, m.Rect, img, b.Min, draw.Src)
	return m
}

// Thumbnail shrinks img to maxW pixels wide (never wider than it is),
// averaging the pixels each one covers.
func Thumbnail(img image.Image, maxW int) *image.RGBA {
	src := toRGBA(img)
	sw, sh := src.Rect.Dx(), src.Rect.Dy()
	if sw <= maxW || sw == 0 {
		return src
	}
	w, h := maxW, max(1, int(math.Round(float64(sh)*float64(maxW)/float64(sw))))
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		y0, y1 := y*sh/h, max((y+1)*sh/h, y*sh/h+1)
		for x := range w {
			x0, x1 := x*sw/w, max((x+1)*sw/w, x*sw/w+1)
			var sum [4]int
			for sy := y0; sy < y1; sy++ {
				row := src.Pix[sy*src.Stride:]
				for sx := x0; sx < x1; sx++ {
					for c := range 4 {
						sum[c] += int(row[sx*4+c])
					}
				}
			}
			n, o := (y1-y0)*(x1-x0), y*out.Stride+x*4
			for c := range 4 {
				out.Pix[o+c] = uint8(sum[c] / n)
			}
		}
	}
	return out
}

// IsNearBlack says whether a picture is all but black, as a capture of a
// game in exclusive fullscreen often is.
func IsNearBlack(img image.Image) bool {
	m := toRGBA(img)
	w, h := m.Rect.Dx(), m.Rect.Dy()
	step := max(1, int(math.Sqrt(float64(w*h)/250000)))
	for y := 0; y < h; y += step {
		row := m.Pix[y*m.Stride:]
		for x := 0; x < w; x += step {
			if row[x*4] > 16 || row[x*4+1] > 16 || row[x*4+2] > 16 {
				return false
			}
		}
	}
	return true
}

// MakeThumb writes a thumbnail of the PNG at src to dst.
func MakeThumb(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	img, err := png.Decode(f)
	f.Close()
	if err != nil {
		return err
	}
	return SaveScreenshotPNG(Thumbnail(img, ThumbWidth), dst)
}

// GameShot is one picture in a game's screenshots folder.
type GameShot struct {
	Name string    // the file's name
	At   time.Time // when it was taken: from its name, else when it was last written
}

var shotName = regexp.MustCompile(`^(\d{4}-\d\d-\d\d \d\d-\d\d-\d\d)(?: \(\d+\))?\.png$`)

// Screenshots lists the PNG pictures in dir, newest first.
func Screenshots(dir string) []GameShot {
	es, _ := os.ReadDir(dir)
	var out []GameShot
	for _, e := range es {
		n := e.Name()
		if e.IsDir() || strings.HasPrefix(n, ".") || !strings.EqualFold(filepath.Ext(n), ".png") {
			continue
		}
		s := GameShot{Name: n}
		if m := shotName.FindStringSubmatch(n); m != nil {
			s.At, _ = time.ParseInLocation(shotStamp, m[1], time.Local)
		}
		if s.At.IsZero() {
			if fi, err := e.Info(); err == nil {
				s.At = fi.ModTime()
			}
		}
		out = append(out, s)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At.After(out[j].At) })
	return out
}

// ShotThumbsDir keeps the thumbnails of a Library game's screenshots, by
// the screenshot's file name, out of the user's Pictures.
func (s *Store) ShotThumbsDir(gameKey string) string {
	return filepath.Join(s.Dir, "screenshots", gameFileKey(gameKey))
}

// SavePointPictures keeps a picture of the game beside a save point, with
// its thumbnail. RestoreSavePoint never puts these back: only the point's
// files folder is.
func SavePointPictures(root, id string, img image.Image) error {
	_, dir, err := loadPoint(root, id)
	if err != nil {
		return err
	}
	if err := SaveScreenshotPNG(img, filepath.Join(dir, PointPicture)); err != nil {
		return err
	}
	return SaveScreenshotPNG(Thumbnail(img, ThumbWidth), filepath.Join(dir, PointThumb))
}

// SavePointFile is the path of a file beside a point (its picture or
// thumbnail), for an id the caller can't vouch for.
func SavePointFile(root, id, name string) (string, error) {
	if name != PointPicture && name != PointThumb {
		return "", errors.New("unknown file")
	}
	_, dir, err := loadPoint(root, id)
	return filepath.Join(dir, name), err
}
