package core

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	_ "image/gif" // pictures people pick for their profile
	_ "image/jpeg"
	"image/png"
	"os"
)

// AvatarSize is the side of the square profile picture Lootnest keeps.
const AvatarSize = 256

// MakeAvatar turns a picture file into a square PNG for a profile: the
// largest centred square, scaled to AvatarSize. Files over 20 MB or 8000 px
// are refused before they're decoded.
func MakeAvatar(path string) ([]byte, error) {
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
	r := img.Bounds()
	side := min(r.Dx(), r.Dy())
	x0, y0 := r.Min.X+(r.Dx()-side)/2, r.Min.Y+(r.Dy()-side)/2
	out := image.NewNRGBA(image.Rect(0, 0, AvatarSize, AvatarSize))
	// Box filter: each output pixel averages the source pixels it covers.
	for y := 0; y < AvatarSize; y++ {
		sy0, sy1 := y0+y*side/AvatarSize, y0+(y+1)*side/AvatarSize
		if sy1 <= sy0 {
			sy1 = sy0 + 1
		}
		for x := 0; x < AvatarSize; x++ {
			sx0, sx1 := x0+x*side/AvatarSize, x0+(x+1)*side/AvatarSize
			if sx1 <= sx0 {
				sx1 = sx0 + 1
			}
			var rs, gs, bs, as, n uint64
			for sy := sy0; sy < sy1; sy++ {
				for sx := sx0; sx < sx1; sx++ {
					c := color.NRGBAModel.Convert(img.At(sx, sy)).(color.NRGBA)
					rs, gs, bs, as, n = rs+uint64(c.R), gs+uint64(c.G), bs+uint64(c.B), as+uint64(c.A), n+1
				}
			}
			out.SetNRGBA(x, y, color.NRGBA{uint8(rs / n), uint8(gs / n), uint8(bs / n), uint8(as / n)})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
