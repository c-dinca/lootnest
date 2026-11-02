"""Draws Lootnest's icon: a gold gem resting in a gold nest, on a dark tile.

Flat colours only: the tile, the gold, and a lighter gold for the gem's
crown so its cut still reads at 16 px.
"""
import sys
from PIL import Image, ImageDraw

S = 1024
TILE = (35, 33, 30, 255)
GOLD = (224, 164, 88, 255)
GOLD_LIGHT = (240, 200, 140, 255)


def draw(size):
    out = Image.new("RGBA", (S, S), (0, 0, 0, 0))
    d = ImageDraw.Draw(out)
    d.rounded_rectangle((32, 32, S - 32, S - 32), radius=256, fill=TILE)

    # Nest: the lower half of a thick ring, with round ends.
    w = 104
    box = (224, 256, S - 224, S - 176)
    d.arc(box, start=0, end=180, fill=GOLD, width=w)
    cy = (box[1] + box[3]) // 2
    for x in (box[0] + w // 2, box[2] - w // 2):
        d.ellipse((x - w // 2, cy - w // 2, x + w // 2, cy + w // 2), fill=GOLD)

    # Gem: a cut diamond sitting in the nest; the crown is a lighter gold.
    cx, top, mid, bottom, half = S // 2, 224, 352, 560, 160
    d.polygon([(cx - half, mid), (cx - half // 2, top), (cx + half // 2, top), (cx + half, mid), (cx, bottom)], fill=GOLD)
    d.polygon([(cx - half, mid), (cx - half // 2, top), (cx + half // 2, top), (cx + half, mid)], fill=GOLD_LIGHT)
    return out.resize((size, size), Image.LANCZOS)


if __name__ == "__main__":
    out = sys.argv[1]
    sizes = [16, 20, 24, 32, 40, 48, 64, 128, 256]
    imgs = [draw(s) for s in sizes]
    imgs[-1].save(out, format="ICO", sizes=[(s, s) for s in sizes], append_images=imgs[:-1])
    draw(512).save(out.replace(".ico", ".png"))
