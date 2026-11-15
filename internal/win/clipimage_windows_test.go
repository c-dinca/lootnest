package win

import (
	"encoding/binary"
	"image"
	"image/color"
	"testing"
)

// A CF_DIB is a 40-byte header, then BGRA rows from the bottom up.
func TestDIBFromImage(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.NRGBA{R: 255, A: 255}) // top left: red
	img.Set(1, 1, color.NRGBA{B: 255, A: 128}) // bottom right: blue, half see-through
	d := DIBFromImage(img)
	le := binary.LittleEndian
	if len(d) != 40+16 || le.Uint32(d[0:]) != 40 || le.Uint32(d[4:]) != 2 || le.Uint32(d[8:]) != 2 || le.Uint16(d[14:]) != 32 {
		t.Fatalf("header %v", d[:40])
	}
	px := d[40:]
	if got := px[8:12]; got[0] != 0 || got[1] != 0 || got[2] != 255 || got[3] != 255 { // top row comes last: red as B G R A
		t.Fatalf("top left %v", got)
	}
	if got := px[4:8]; got[0] != 255 || got[3] != 128 { // bottom row first: blue
		t.Fatalf("bottom right %v", got)
	}
}
