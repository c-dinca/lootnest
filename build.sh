#!/usr/bin/env bash
# Builds dist/Lootnest-Setup.exe (and the bare dist/Lootnest.exe) for Windows x64.
# Requires Go 1.24+ and Python 3 (Pillow optional, only to redraw the icon).
# Runs on Linux, macOS or Windows (Git Bash). Version: $VERSION, else the VERSION file.
#
#   ./build.sh          both, as below
#   ./build.sh app      dist/Lootnest.exe only
#   ./build.sh setup    dist/Lootnest-Setup.exe from the dist/Lootnest.exe already
#                       there, so CI can sign the app before it goes into the setup
#   ./build.sh plugins  dist/Lootnest-plugin-<id>.zip for each folder in plugins/,
#                       with plugin.json at the zip's top, ready for Install from file
set -euo pipefail
cd "$(dirname "$0")"
VERSION="${VERSION:-$(cat VERSION)}"
STAGE="${1:-all}"
export GOOS=windows GOARCH=amd64 CGO_ENABLED=0
LDFLAGS="-H windowsgui -s -w"

case "$STAGE" in
  all|app|setup|plugins) ;;
  *) echo "usage: $0 [app|setup|plugins]" >&2; exit 2 ;;
esac

if [[ "$STAGE" == plugins ]]; then
  mkdir -p dist
  # Each plugin's files, sorted, with fixed times, so the same folder makes
  # the same zip. Lootnest checks every file again when it's installed.
  python3 - <<'PY'
import json, os, zipfile
for pid in sorted(os.listdir("plugins")):
    src = os.path.join("plugins", pid)
    if not os.path.isfile(os.path.join(src, "plugin.json")):
        continue
    if json.load(open(os.path.join(src, "plugin.json")))["id"] != pid:
        raise SystemExit(f"plugins/{pid}: its id isn't its folder's name")
    out = os.path.join("dist", f"Lootnest-plugin-{pid}.zip")
    with zipfile.ZipFile(out, "w", zipfile.ZIP_DEFLATED) as z:
        for root, dirs, files in os.walk(src):
            dirs.sort()
            for f in sorted(files):
                p = os.path.join(root, f)
                info = zipfile.ZipInfo(os.path.relpath(p, src).replace(os.sep, "/"), (2026, 1, 1, 0, 0, 0))
                info.compress_type = zipfile.ZIP_DEFLATED
                info.external_attr = 0o644 << 16
                z.writestr(info, open(p, "rb").read())
    print(out)
PY
  ls -la dist
  exit 0
fi

if [[ "$STAGE" != setup ]]; then
  # Redraw the icon when Pillow is available; otherwise use the committed one.
  if python3 -c "import PIL" 2>/dev/null; then
    python3 tools/mkicon.py assets/lootnest.ico
  fi
  python3 tools/mkres.py cmd/lootnest/rsrc_windows_amd64.syso assets/lootnest.ico "Lootnest" "Lootnest" "$VERSION" Lootnest.exe

  GOOS= GOARCH= go test ./internal/core/

  mkdir -p dist
  go build -trimpath -ldflags "$LDFLAGS -X lootnest/internal/app.Version=$VERSION" -o dist/Lootnest.exe ./cmd/lootnest
  python3 tools/checkpe.py dist/Lootnest.exe >/dev/null
fi

if [[ "$STAGE" != app ]]; then
  if [[ ! -f dist/Lootnest.exe ]]; then
    echo "dist/Lootnest.exe is missing: run '$0 app' first" >&2
    exit 1
  fi
  python3 tools/mkres.py cmd/setup/rsrc_windows_amd64.syso assets/lootnest.ico "Lootnest" "Lootnest Setup" "$VERSION" Lootnest-Setup.exe
  mkdir -p cmd/setup/payload
  cp dist/Lootnest.exe cmd/setup/payload/Lootnest.exe
  cp assets/THIRD_PARTY_NOTICES.txt cmd/setup/payload/THIRD_PARTY_NOTICES.txt
  go build -trimpath -ldflags "$LDFLAGS -X main.Version=$VERSION" -o dist/Lootnest-Setup.exe ./cmd/setup
  python3 tools/checkpe.py dist/Lootnest-Setup.exe >/dev/null
fi

if [[ "$STAGE" == all ]]; then
  "$0" plugins >/dev/null
fi

ls -la dist
