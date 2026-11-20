package app

import _ "embed"

// overlayDLL is the in-game FPS counter (native/overlay, built by
// tools/overlay/build.sh; CI checks it matches its source).
//
//go:embed overlay/lootnest-overlay64.dll
var overlayDLL []byte
