package core

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif" // pictures people pick for their profile
	"image/jpeg"
	"image/png"
	"os"
)

// AvatarSize is the side of the square profile picture Lootnest keeps.
const AvatarSize = 256

// A banner picture is kept at BannerW x BannerH (3:1), as a JPEG.
const (
	BannerW = 1500
	BannerH = 500
)

// MakeAvatar turns a picture file into a square PNG for a profile: the
// largest centred square, scaled to AvatarSize.
func MakeAvatar(path string) ([]byte, error) {
	img, err := fitPicture(path, AvatarSize, AvatarSize)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// MakeBanner turns a picture file into a profile banner: the largest
// centred 3:1 strip, scaled to BannerW x BannerH, as a JPEG.
func MakeBanner(path string) ([]byte, error) {
	img, err := fitPicture(path, BannerW, BannerH)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 86}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// fitPicture reads a picture file and returns its largest centred w:h part
// scaled to w x h. Files over 20 MB or 8000 px are refused before they're
// decoded.
func fitPicture(path string, w, h int) (*image.NRGBA, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if st.Size() > 20<<20 {
		return nil, errors.New("That picture is too big; pick one under 20 MB.")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return nil, errors.New("That file isn't a picture Lootnest can read (PNG, JPEG or GIF).")
	}
	if cfg.Width > 8000 || cfg.Height > 8000 || cfg.Width < 16 || cfg.Height < 16 {
		return nil, errors.New("Pick a picture between 16 and 8000 pixels on each side.")
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, errors.New("That file isn't a picture Lootnest can read (PNG, JPEG or GIF).")
	}
	src := image.NewNRGBA(img.Bounds())
	draw.Draw(src, src.Bounds(), img, img.Bounds().Min, draw.Src)
	r := src.Bounds()
	// The largest centred part with the w:h shape.
	cw, ch := r.Dx(), r.Dx()*h/w
	if ch > r.Dy() {
		cw, ch = r.Dy()*w/h, r.Dy()
	}
	cw, ch = max(cw, 1), max(ch, 1)
	x0, y0 := r.Min.X+(r.Dx()-cw)/2, r.Min.Y+(r.Dy()-ch)/2
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	// Box filter: each output pixel averages the source pixels it covers.
	for y := 0; y < h; y++ {
		sy0, sy1 := y0+y*ch/h, y0+(y+1)*ch/h
		if sy1 <= sy0 {
			sy1 = sy0 + 1
		}
		for x := 0; x < w; x++ {
			sx0, sx1 := x0+x*cw/w, x0+(x+1)*cw/w
			if sx1 <= sx0 {
				sx1 = sx0 + 1
			}
			var rs, gs, bs, as, n uint64
			for sy := sy0; sy < sy1; sy++ {
				i := src.PixOffset(sx0, sy)
				for sx := sx0; sx < sx1; sx++ {
					px := src.Pix[i : i+4 : i+4]
					rs, gs, bs, as, n = rs+uint64(px[0]), gs+uint64(px[1]), bs+uint64(px[2]), as+uint64(px[3]), n+1
					i += 4
				}
			}
			out.SetNRGBA(x, y, color.NRGBA{uint8(rs / n), uint8(gs / n), uint8(bs / n), uint8(as / n)})
		}
	}
	return out, nil
}
