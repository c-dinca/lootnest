#!/bin/sh
# Builds the in-game FPS counter (native/overlay/overlay.c, MinHook) into
# internal/app/overlay/lootnest-overlay64.dll with MinGW-w64, the same
# bytes every time, so the committed DLL can be checked against its source:
#
#   tools/overlay/build.sh          build it
#   tools/overlay/build.sh --check  build it elsewhere and compare (CI)
#
# Needs Ubuntu 24.04's MinGW-w64 (gcc-mingw-w64-x86-64: GCC 13.2, binutils
# 2.41.90): another release's makes other bytes. Elsewhere, in Docker:
#
#   docker run --rm -v "$PWD:/src" -w /src ubuntu:24.04 sh -c \
#     'apt-get update -qq && apt-get install -y -qq gcc-mingw-w64-x86-64 && tools/overlay/build.sh'
set -eu
root=$(cd "$(dirname "$0")/../.." && pwd)
# The win32 threads flavour, as Debian and Ubuntu install by default.
cc=${CC:-x86_64-w64-mingw32-gcc-win32}
out="$root/internal/app/overlay/lootnest-overlay64.dll"
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
mh="$root/third_party/minhook"
# Fixed object names and paths: nothing of the build machine in the DLL.
cd "$work"
for f in "$root/native/overlay/overlay.c" "$mh/src/buffer.c" "$mh/src/hook.c" "$mh/src/trampoline.c" "$mh/src/hde/hde64.c"; do
	"$cc" -std=gnu11 -O2 -Wall -Wno-unused-parameter -Wno-cast-function-type \
		-frandom-seed=lootnest -ffile-prefix-map="$root"=. \
		-I"$mh/include" -c "$f" -o "$(basename "$f" .c).o"
done
"$cc" -shared -s -static-libgcc -Wl,--no-insert-timestamp \
	-o lootnest-overlay64.dll overlay.o buffer.o hook.o trampoline.o hde64.o
if [ "${1:-}" = "--check" ]; then
	if ! cmp -s lootnest-overlay64.dll "$out"; then
		echo "internal/app/overlay/lootnest-overlay64.dll isn't what native/overlay builds: run tools/overlay/build.sh" >&2
		"$cc" --version | head -1 >&2
		x86_64-w64-mingw32-ld --version | head -1 >&2
		sha256sum lootnest-overlay64.dll "$out" >&2
		cmp -l lootnest-overlay64.dll "$out" | head -20 >&2 || true
		exit 1
	fi
	echo "lootnest-overlay64.dll matches its source"
else
	cp lootnest-overlay64.dll "$out"
	sha256sum "$out"
fi
