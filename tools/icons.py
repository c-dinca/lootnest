#!/usr/bin/env python3
"""Writes internal/app/ui/js/01-icons.js from Phosphor Icons' SVGs.

Lootnest's icons are Phosphor Icons (MIT, assets/THIRD_PARTY_NOTICES.txt):
regular in keys, rows and menus, fill for the current item (the sidebar's
section, a tab, a switch that's on), duotone for empty pages. Only the
icons named below are copied, so the app carries what it uses.

    npm pack @phosphor-icons/core   # in an empty folder
    tar -xzf phosphor-icons-core-*.tgz
    python3 tools/icons.py path/to/package/assets

Add a name to ICONS (Phosphor's name, as on phosphoricons.com) and run it
again; the app calls ph("name"), ph("name", "fill") or ph("name", "duotone").
"""
import pathlib
import re
import sys

ICONS = """
arrow-clockwise arrow-circle-up arrow-left arrow-right arrows-left-right bell camera caret-down caret-left caret-right caret-up
check check-circle clock-counter-clockwise cloud copy desktop dots-six-vertical dots-three download-simple eye-slash flag-pennant
flask floppy-disk folder folder-open funnel game-controller gear-six globe-simple hard-drive heart house image info
list lock magnifying-glass palette pencil-simple play plus power puzzle-piece question rocket-launch shield shield-check
sidebar-simple sliders-horizontal sort-ascending squares-four star storefront television-simple trash trophy user
users users-three vault video-camera warning x
""".split()
# Weights besides regular: fill for whatever can be current or on, duotone
# for the big icon of an empty page.
FILL = set("house squares-four storefront arrow-circle-up trophy vault gear-six heart star play user users users-three television-simple check-circle".split())
DUOTONE = set("vault trophy squares-four arrow-circle-up storefront game-controller folder-open users-three".split())

VERSION = "2.1.1"


def inner(svg: str) -> str:
    """The markup inside <svg>, with nothing a page could run."""
    m = re.search(r"<svg[^>]*>(.*)</svg>", svg, re.S)
    if not m:
        raise SystemExit("not an svg")
    body = m.group(1).strip()
    if re.search(r"<script|on\w+=|href", body, re.I):
        raise SystemExit("unexpected markup in an icon")
    return re.sub(r"\s+", " ", body)


def main() -> None:
    if len(sys.argv) != 2:
        raise SystemExit(__doc__)
    assets = pathlib.Path(sys.argv[1])
    out = {}
    for name in ICONS:
        w = {"regular": inner((assets / "regular" / f"{name}.svg").read_text())}
        if name in FILL:
            w["fill"] = inner((assets / "fill" / f"{name}-fill.svg").read_text())
        if name in DUOTONE:
            w["duotone"] = inner((assets / "duotone" / f"{name}-duotone.svg").read_text())
        out[name] = w
    lines = []
    for name, w in out.items():
        parts = ", ".join(f"{k}: {js(v)}" for k, v in w.items())
        lines.append(f"    {js(name)}: {{ {parts} }},")
    body = "\n".join(lines)
    text = f"""  // ----------------------------------------------------------------- icons --
  // Phosphor Icons {VERSION} (MIT; assets/THIRD_PARTY_NOTICES.txt), written by
  // tools/icons.py: regular for keys, rows and menus, fill for the current
  // item or a thing that's on, duotone for the big icon of an empty page.
  // Sizes come from CSS (--icon-sm, --icon, --icon-lg), never from here.
  const PH = {{
{body}
  }};
  const ph = (name, weight) => {{
    const i = PH[name] || PH.question, d = (weight && i[weight]) || i.regular;
    return `<svg class="ln-icon ph" viewBox="0 0 256 256" aria-hidden="true" focusable="false">${{d}}</svg>`;
  }};
  // Static markup (index.html) asks for icons as <i data-ph="name">; data-two
  // draws both weights, regular and fill, for something that can be current.
  function paintIcons(root) {{
    root.querySelectorAll("i[data-ph]").forEach((el) => {{
      const n = el.dataset.ph;
      el.outerHTML = el.hasAttribute("data-two") ? ph(n).replace('class="ln-icon ph"', 'class="ln-icon ph ph-off"') + ph(n, "fill").replace('class="ln-icon ph"', 'class="ln-icon ph ph-on"') : ph(n, el.dataset.w);
    }});
  }}
"""
    dest = pathlib.Path(__file__).resolve().parent.parent / "internal/app/ui/js/01-icons.js"
    dest.write_text(text)
    print(dest, len(text) // 1024, "KB,", len(out), "icons")


def js(s: str) -> str:
    return '"' + s.replace("\\", "\\\\").replace('"', '\\"') + '"'


if __name__ == "__main__":
    main()
