"""Draws Lootnest's icon: the brand mark (a Racing Red ring over two Ice
Grey arcs, the nest) on an Obsidian Black tile. It is the same mark as
the sidebar's and the boot screen's (`BRAND_MARK`, a 20 by 20 SVG), so
the taskbar, the tray and the window share one sign (the owner, 22 Nov
2026).

The arcs and the ring follow the SVG's shapes; the strokes are a little
heavier at small sizes so the mark still reads at 16 px. Drawn at 1024 px
with round caps, then scaled down.
"""
import math
import sys
from PIL import Image, ImageDraw

S = 1024
TILE = (0x11, 0x12, 0x17, 255)       # Obsidian Black
NEST = (0xDF, 0xE5, 0xEB, 255)       # Ice Grey
GEM = (0xEF, 0x23, 0x3C, 255)        # Racing Red

# The SVG's 20-unit box, placed on the tile: the mark spans x 2.75..17.25
# and y 3.75..16.5, so its middle is (10, 10.1).
UNIT = 46          # pixels per SVG unit at 1024
CX, CY = S / 2, S / 2 + 10


def stroke(d, pts, width, fill):
    """A polyline with round joins and caps: discs along it."""
    r = width / 2
    for (x0, y0), (x1, y1) in zip(pts, pts[1:]):
        steps = max(1, int(math.hypot(x1 - x0, y1 - y0) / 2))
        for i in range(steps + 1):
            t = i / steps
            x, y = x0 + (x1 - x0) * t, y0 + (y1 - y0) * t
            d.ellipse((x - r, y - r, x + r, y + r), fill=fill)


def paint(d, size, unit, cx, cy):
    """The mark at unit pixels per SVG unit around (cx, cy), its strokes
    heavier when it will be shown small (size, the final pixels across)."""
    def at(x, y):
        return cx + (x - 10) * unit, cy + (y - 10.1) * unit

    def lower_arc(rx, ry, n=200):
        return [at(10 - rx * math.cos(math.pi * i / n), 10.25 + ry * math.sin(math.pi * i / n)) for i in range(n + 1)]

    w = (2.6 if size <= 24 else 2.2 if size <= 48 else 1.9) * unit
    stroke(d, lower_arc(7.25, 6.25), w, NEST)   # the outer arc
    stroke(d, lower_arc(4.25, 3.5), w, NEST)    # the inner arc
    gx, gy = at(10, 6.25)                       # the ring: r 2.5 around (10, 6.25)
    r = 2.5 * unit
    d.ellipse((gx - r - w / 2, gy - r - w / 2, gx + r + w / 2, gy + r + w / 2), outline=GEM, width=round(w))


def draw(size):
    """The app icon: the mark on the obsidian tile."""
    out = Image.new("RGBA", (S, S), (0, 0, 0, 0))
    d = ImageDraw.Draw(out)
    d.rounded_rectangle((32, 32, S - 32, S - 32), radius=232, fill=TILE)
    paint(d, size, UNIT, CX, CY)
    return out.resize((size, size), Image.LANCZOS)


def mark(size, fill=1.0):
    """The mark alone on a clear ground, filling fill of the square (for
    places that draw their own ground: the TV apps' screens, Android's
    adaptive icon)."""
    out = Image.new("RGBA", (S, S), (0, 0, 0, 0))
    d = ImageDraw.Draw(out)
    unit = S * fill / 17.2   # the mark is about 17.2 units across with its strokes
    paint(d, size, unit, S / 2, S / 2)
    box = out.getbbox()
    if box:  # centre what was drawn
        dx = (S - (box[0] + box[2])) / 2
        dy = (S - (box[1] + box[3])) / 2
        out = out.transform(out.size, Image.AFFINE, (1, 0, -dx, 0, 1, -dy), resample=Image.BICUBIC)
    return out.resize((size, size), Image.LANCZOS)


if __name__ == "__main__":
    out = sys.argv[1]
    sizes = [16, 20, 24, 32, 40, 48, 64, 128, 256]
    imgs = [draw(s) for s in sizes]
    imgs[-1].save(out, format="ICO", sizes=[(s, s) for s in sizes], append_images=imgs[:-1])
    draw(512).save(out.replace(".ico", ".png"))
