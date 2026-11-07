package core

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestMakeAvatar(t *testing.T) {
	dir := t.TempDir()
	src := image.NewNRGBA(image.Rect(0, 0, 900, 600)) // wide: the centre square is kept
	for y := 0; y < 600; y++ {
		for x := 0; x < 900; x++ {
			c := color.NRGBA{0, 0, 255, 255}
			if x >= 150 && x < 750 {
				c = color.NRGBA{255, 0, 0, 255}
			}
			src.SetNRGBA(x, y, c)
		}
	}
	p := filepath.Join(dir, "me.png")
	f, _ := os.Create(p)
	_ = png.Encode(f, src)
	f.Close()
	b, err := MakeAvatar(p)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil || img.Bounds().Dx() != AvatarSize || img.Bounds().Dy() != AvatarSize {
		t.Fatalf("a %d px square PNG: %v %v", AvatarSize, img.Bounds(), err)
	}
	if r, _, bl, _ := img.At(5, 5).RGBA(); r>>8 != 255 || bl != 0 {
		t.Fatal("the centre square of the picture is kept")
	}
	if err := os.WriteFile(filepath.Join(dir, "x.png"), []byte("not a picture"), 0o644); err == nil {
		if _, err := MakeAvatar(filepath.Join(dir, "x.png")); err == nil {
			t.Fatal("a file that isn't a picture is refused")
		}
	}
}
