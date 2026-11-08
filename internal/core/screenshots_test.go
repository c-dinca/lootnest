package core

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestSafeFileName(t *testing.T) {
	for in, want := range map[string]string{
		"Baldur's Gate 3":        "Baldur's Gate 3",
		`AC/DC: Live\Rock?`:      "ACDC LiveRock",
		"Game...":                "Game",
		"Ends in space .  ":      "Ends in space",
		"CON":                    "CON_",
		"lpt1.txt":               "lpt1.txt_",
		"Console":                "Console",
		"<>:|*":                  "Game",
		"Tab\there":              "Tabhere",
		"  Hollow Tale  ":        "Hollow Tale",
		"S.T.A.L.K.E.R. 2...":    "S.T.A.L.K.E.R. 2",
		"Nul":                    "Nul_",
		"COM9":                   "COM9_",
		"Comet":                  "Comet",
		"..":                     "Game",
		"Under the \"Sea\" (v2)": "Under the Sea (v2)",
	} {
		if got := SafeFileName(in); got != want {
			t.Errorf("SafeFileName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSaveNewScreenshotCollisions(t *testing.T) {
	pics := t.TempDir()
	when := time.Date(2026, 11, 8, 21, 5, 9, 0, time.Local)
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	p, err := SaveNewScreenshot(img, pics, "Hollow: Tale.", when)
	want := filepath.Join(pics, "Lootnest", "Screenshots", "Hollow Tale", "2026-11-08 21-05-09.png")
	if err != nil || p != want {
		t.Fatalf("path %q %v, want %q", p, err, want)
	}
	// Pressed again and again within the same second, at the same time.
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			if _, err := SaveNewScreenshot(img, pics, "Hollow: Tale.", when); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	list := Screenshots(ScreenshotsDir(pics, "Hollow: Tale."))
	if len(list) != 5 || !list[0].At.Equal(when) {
		t.Fatalf("list %+v", list)
	}
	for _, n := range []string{"", " (2)", " (3)", " (4)", " (5)"} {
		if fi, err := os.Stat(filepath.Join(filepath.Dir(want), "2026-11-08 21-05-09"+n+".png")); err != nil || fi.Size() == 0 {
			t.Fatalf("suffix %q: %v", n, err)
		}
	}
}

func TestSaveScreenshotPNGLeavesNothingOnError(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "shot.png")
	if err := SaveScreenshotPNG(image.NewRGBA(image.Rect(0, 0, 0, 0)), p); err == nil {
		t.Fatal("an empty picture must not encode")
	}
	if es, _ := os.ReadDir(dir); len(es) != 0 {
		t.Fatalf("left behind %v", es)
	}
}

func TestThumbnailAndNearBlack(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1920, 1080))
	if !IsNearBlack(img) {
		t.Fatal("black")
	}
	for i := range img.Pix {
		if i%4 != 3 {
			img.Pix[i] = uint8(i % 11) // dark noise
		}
	}
	if !IsNearBlack(img) {
		t.Fatal("dark noise is still black")
	}
	for y := 500; y < 540; y++ {
		for x := 900; x < 1000; x++ {
			img.Set(x, y, color.RGBA{200, 40, 40, 255})
		}
	}
	if IsNearBlack(img) {
		t.Fatal("a dark scene with something in it is a picture")
	}
	th := Thumbnail(img, ThumbWidth)
	if th.Rect.Dx() != 320 || th.Rect.Dy() != 180 {
		t.Fatalf("thumb %v", th.Rect)
	}
	small := image.NewRGBA(image.Rect(0, 0, 100, 50))
	if th := Thumbnail(small, ThumbWidth); th.Rect.Dx() != 100 || th.Rect.Dy() != 50 {
		t.Fatalf("a small picture keeps its size: %v", th.Rect)
	}
	// Averages: half white, half black gives grey.
	hw := image.NewRGBA(image.Rect(0, 0, 2, 1))
	hw.Set(0, 0, color.RGBA{255, 255, 255, 255})
	hw.Set(1, 0, color.RGBA{0, 0, 0, 255})
	if c := Thumbnail(hw, 1).RGBAAt(0, 0); c.R != 127 || c.A != 255 {
		t.Fatalf("average %v", c)
	}
}

func TestSavePointPicturesStayOutOfRestores(t *testing.T) {
	live := t.TempDir()
	write(t, filepath.Join(live, "slot1.sav"), "before")
	locs := []SaveLoc{{ID: "loc-a", Live: live, Path: live}}
	root := filepath.Join(t.TempDir(), "points")
	p, err := MakeSavePoint(root, locs, SavePoint{Name: "Gate"}, time.Now(), 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 640, 360))
	if err := SavePointPictures(root, p.ID, img); err != nil {
		t.Fatal(err)
	}
	thumb, err := SavePointFile(root, p.ID, PointThumb)
	if err != nil || !exists(thumb) {
		t.Fatalf("thumb %q %v", thumb, err)
	}
	if _, err := SavePointFile(root, "../"+p.ID, PointPicture); err == nil {
		t.Fatal("ids with a path must be refused")
	}
	if _, err := SavePointFile(root, p.ID, "point.json"); err == nil {
		t.Fatal("only the pictures")
	}
	write(t, filepath.Join(live, "slot1.sav"), "after")
	if err := RestoreSavePoint(root, p.ID, locs); err != nil {
		t.Fatal(err)
	}
	es, _ := os.ReadDir(live)
	if len(es) != 1 || es[0].Name() != "slot1.sav" || read(t, filepath.Join(live, "slot1.sav")) != "before" {
		t.Fatalf("restore wrote %v", es)
	}
	if len(SavePoints(root)) != 1 {
		t.Fatal("the point is still listed")
	}
}
