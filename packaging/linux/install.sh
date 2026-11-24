#!/bin/sh
# Puts Lootnest in your home folder: the program in ~/.local/bin, the menu
# entry and the icon in ~/.local/share. Nothing needs root, and nothing
# outside your home folder changes.
#
#   ./install.sh            install, or update over an older copy
#   ./install.sh --remove   take the program, menu entry and icon away;
#                           your profiles and saves in ~/.local/share/lootnest stay
set -eu
here=$(cd "$(dirname "$0")" && pwd)
bin="${XDG_BIN_HOME:-$HOME/.local/bin}"
data="${XDG_DATA_HOME:-$HOME/.local/share}"
apps="$data/applications"
icons="$data/icons/hicolor/512x512/apps"

if [ "${1:-}" = "--remove" ]; then
  rm -f "$bin/lootnest" "$apps/lootnest.desktop" "$icons/lootnest.png"
  command -v update-desktop-database >/dev/null 2>&1 && update-desktop-database "$apps" 2>/dev/null || true
  echo "Lootnest is removed. Your profiles and saves are still in $data/lootnest."
  exit 0
fi

mkdir -p "$bin" "$apps" "$icons"
# A copy beside the old one, then a rename, so a running Lootnest keeps its file.
cp "$here/lootnest" "$bin/.lootnest.new"
chmod 755 "$bin/.lootnest.new"
mv -f "$bin/.lootnest.new" "$bin/lootnest"
cp "$here/lootnest.png" "$icons/lootnest.png"
sed "s|^Exec=.*|Exec=$bin/lootnest|" "$here/lootnest.desktop" > "$apps/lootnest.desktop"
command -v update-desktop-database >/dev/null 2>&1 && update-desktop-database "$apps" 2>/dev/null || true
command -v gtk-update-icon-cache >/dev/null 2>&1 && gtk-update-icon-cache -q "$data/icons/hicolor" 2>/dev/null || true
echo "Lootnest is installed: $bin/lootnest, and in your applications menu."
