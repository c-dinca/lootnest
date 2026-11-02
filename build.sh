#!/usr/bin/env bash
# Builds dist/Lootnest-Setup.exe (and the bare dist/Lootnest.exe) for Windows x64.
# Requires Go 1.24+ and Python 3 (Pillow optional, only to redraw the icon).
# Runs on Linux, macOS or Windows (Git Bash). Version: $VERSION, else the VERSION file.
set -euo pipefail
cd "$(dirname "$0")"
VERSION="${VERSION:-$(cat VERSION)}"
export GOOS=windows GOARCH=amd64 CGO_ENABLED=0
LDFLAGS="-H windowsgui -s -w"

# Redraw the icon when Pillow is available; otherwise use the committed one.
if python3 -c "import PIL" 2>/dev/null; then
  python3 tools/mkicon.py assets/lootnest.ico
fi
python3 tools/mkres.py cmd/lootnest/rsrc_windows_amd64.syso assets/lootnest.ico "Lootnest" "Lootnest" "$VERSION" Lootnest.exe
python3 tools/mkres.py cmd/setup/rsrc_windows_amd64.syso assets/lootnest.ico "Lootnest" "Lootnest Setup" "$VERSION" Lootnest-Setup.exe

GOOS= GOARCH= go test ./internal/core/

mkdir -p dist cmd/setup/payload
go build -trimpath -ldflags "$LDFLAGS -X lootnest/internal/app.Version=$VERSION" -o dist/Lootnest.exe ./cmd/lootnest
python3 tools/checkpe.py dist/Lootnest.exe >/dev/null

cp dist/Lootnest.exe cmd/setup/payload/Lootnest.exe
cp assets/THIRD_PARTY_NOTICES.txt cmd/setup/payload/THIRD_PARTY_NOTICES.txt
go build -trimpath -ldflags "$LDFLAGS -X main.Version=$VERSION" -o dist/Lootnest-Setup.exe ./cmd/setup
python3 tools/checkpe.py dist/Lootnest-Setup.exe >/dev/null

ls -la dist
