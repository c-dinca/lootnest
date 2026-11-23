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
from PIL import Image, ImageDraw, ImageOps

S = 1024
TILE = (0x11, 0x12, 0x17, 255)       # Obsidian Black
NEST = (0xDF, 0xE5, 0xEB, 255)       # Ice Grey
GEM = (0xEF, 0x23, 0x3C, 255)        # Racing Red

# The SVG's 20-unit box: arcs around (10, 10.25), the outer one 7.25 by
# 6.25, the inner 4.25 by 3.5, the ring r 2.5 around (10, 6.25), strokes
# 1.6. Large icons keep those shapes with a slightly heavier stroke. Small
# ones need room the SVG doesn't have at 16 to 48 px: a stroke at least
# 1.5 px, a gap of at least 1.1 px between the arcs and 1.8 px between
# the ring and the arcs' ends, so the inner arc narrows and the ring rises (the
# owner, 23 Nov 2026: the ring sat in the lines).
TILE_FILL = 0.94   # the rounded tile, of the square


def geometry(up):
    """The mark's shapes in SVG units for up pixels per unit."""
    w = max(1.9, 1.5 / up)
    g = max(1.4, 1.1 / up)
    ro, ryo = 7.25, 6.25
    ri = min(4.25, ro - w - g)
    ryi = ri * 3.5 / 4.25
    r = max(2.5, w / 2 + 0.7 / up)       # the ring's hole stays open
    # The ring as low as the SVG's, unless that brings it near the inner
    # arc's ends: red against grey blurs together sooner than grey against
    # grey, so it keeps more room (1.8 px at least).
    near = r + w + max(g, 1.8 / up)
    cy = min(6.25, 10.25 - math.sqrt(max(0.0, near * near - ri * ri)))
    return dict(w=w, ro=ro, ryo=ryo, ri=ri, ryi=ryi, r=r, cy=cy)


def extent(geo):
    """Width and height of the mark, strokes included, and its middle."""
    h = geo["w"] / 2
    x0, x1 = 10 - geo["ro"] - h, 10 + geo["ro"] + h
    y0 = geo["cy"] - geo["r"] - h
    y1 = 10.25 + geo["ryo"] + h
    return x1 - x0, y1 - y0, (x0 + x1) / 2, (y0 + y1) / 2


def fit(size, fill):
    """The shapes and pixels per unit so the mark fills fill of size."""
    up = size * fill / 17
    for _ in range(4):
        geo = geometry(up)
        w, h, _, _ = extent(geo)
        up = size * fill / max(w, h)
    return geometry(up), up


def stroke(d, pts, width, fill):
    """A polyline with round joins and caps: discs along it."""
    r = width / 2
    for (x0, y0), (x1, y1) in zip(pts, pts[1:]):
        steps = max(1, int(math.hypot(x1 - x0, y1 - y0) / 2))
        for i in range(steps + 1):
            t = i / steps
            x, y = x0 + (x1 - x0) * t, y0 + (y1 - y0) * t
            d.ellipse((x - r, y - r, x + r, y + r), fill=fill)


def paint(d, geo, unit):
    """The mark at unit pixels per SVG unit, centred on the S square."""
    _, _, mx, my = extent(geo)

    def at(x, y):
        return S / 2 + (x - mx) * unit, S / 2 + (y - my) * unit

    def lower_arc(rx, ry, n=240):
        return [at(10 - rx * math.cos(math.pi * i / n), 10.25 + ry * math.sin(math.pi * i / n)) for i in range(n + 1)]

    w = geo["w"] * unit
    stroke(d, lower_arc(geo["ro"], geo["ryo"]), w, NEST)
    stroke(d, lower_arc(geo["ri"], geo["ryi"]), w, NEST)
    gx, gy = at(10, geo["cy"])
    ro = geo["r"] * unit + w / 2
    ri = geo["r"] * unit - w / 2
    d.ellipse((gx - ro, gy - ro, gx + ro, gy + ro), fill=GEM)
    d.ellipse((gx - ri, gy - ri, gx + ri, gy + ri), fill=(0, 0, 0, 0))


def render(size, fill, tile):
    out = Image.new("RGBA", (S, S), (0, 0, 0, 0))
    d = ImageDraw.Draw(out)
    if tile:
        m = S * (1 - TILE_FILL) / 2
        d.rounded_rectangle((m, m, S - m, S - m), radius=232, fill=TILE)
    geo, up = fit(size, fill)
    mark_layer = Image.new("RGBA", (S, S), (0, 0, 0, 0))
    paint(ImageDraw.Draw(mark_layer), geo, up * S / size)
    out.alpha_composite(mark_layer)
    # The drawing calls round half pixels one way, which left the mark a
    # touch off centre once scaled down: keep the left half and mirror it.
    left = out.crop((0, 0, S // 2, S))
    out.paste(ImageOps.mirror(left), (S // 2, 0))
    return out.resize((size, size), Image.LANCZOS)


def draw(size):
    """The app icon: the mark on the obsidian tile, a little larger on
    small icons."""
    return render(size, 0.80 if size <= 24 else 0.76 if size <= 48 else 0.72, True)


def mark(size, fill=1.0):
    """The mark alone on a clear ground, filling fill of the square (for
    places that draw their own ground: the TV apps' screens, Android's
    adaptive icon)."""
    return render(size, fill * 0.96, False)


if __name__ == "__main__":
    out = sys.argv[1]
    # Every size Windows asks for at 100% to 200%, so none is scaled.
    sizes = [16, 20, 24, 30, 32, 36, 40, 48, 60, 64, 72, 80, 96, 128, 256]
    imgs = [draw(s) for s in sizes]
    imgs[-1].save(out, format="ICO", sizes=[(s, s) for s in sizes], append_images=imgs[:-1])
    draw(512).save(out.replace(".ico", ".png"))
