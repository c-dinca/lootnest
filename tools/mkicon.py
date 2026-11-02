"""Draws Lootnest's icon: a gem resting in a nest, on a rounded tile.

The nest is a thick bowl-shaped arc and the gem a cut diamond, both in the
two player colours (blue to pink) so the mark keeps the old palette.
"""
import sys
from PIL import Image, ImageDraw

S = 1024
BLUE = (79, 140, 255)
PINK = (255, 93, 143)


def gradient(c1, c2):
    """Left-to-right gradient from c1 to c2, full size."""
    g = Image.new("RGBA", (S, 1))
    for x in range(S):
        t = x / (S - 1)
        g.putpixel((x, 0), tuple(int(c1[i] + (c2[i] - c1[i]) * t) for i in range(3)) + (255,))
    return g.resize((S, S))


def draw(size):
    out = Image.new("RGBA", (S, S), (0, 0, 0, 0))
    ImageDraw.Draw(out).rounded_rectangle((40, 40, S - 40, S - 40), radius=230, fill=(28, 34, 48, 255))

    # Nest: the lower half of a thick ring.
    nest = Image.new("L", (S, S), 0)
    d = ImageDraw.Draw(nest)
    box = (200, 300, S - 200, S - 170)
    d.arc(box, start=10, end=170, fill=255, width=110)
    out.paste(gradient(BLUE, PINK), (0, 0), nest)

    # Gem: a cut diamond sitting in the nest.
    cx, top, mid, bottom, half = S // 2, 250, 400, 650, 190
    gem = Image.new("L", (S, S), 0)
    ImageDraw.Draw(gem).polygon([(cx - half, mid), (cx - half // 2, top), (cx + half // 2, top), (cx + half, mid), (cx, bottom)], fill=255)
    out.paste(gradient((140, 180, 255), (255, 150, 185)), (0, 0), gem)
    # Facets: a lighter crown so the gem reads at small sizes.
    crown = Image.new("L", (S, S), 0)
    ImageDraw.Draw(crown).polygon([(cx - half, mid), (cx - half // 2, top), (cx + half // 2, top), (cx + half, mid)], fill=110)
    out.paste(Image.new("RGBA", (S, S), (255, 255, 255, 255)), (0, 0), crown)
    return out.resize((size, size), Image.LANCZOS)


if __name__ == "__main__":
    out = sys.argv[1]
    sizes = [16, 20, 24, 32, 40, 48, 64, 128, 256]
    imgs = [draw(s) for s in sizes]
    imgs[-1].save(out, format="ICO", sizes=[(s, s) for s in sizes], append_images=imgs[:-1])
    draw(512).save(out.replace(".ico", ".png"))
