module lootnest

go 1.26.0

toolchain go1.27.4

require (
	github.com/bodgit/sevenzip v1.6.5
	github.com/dop251/goja v0.0.0-20261116234614-2eded47d7035
	github.com/jchv/go-webview2 v0.0.0-00010101000000-000000000000
	github.com/nwaples/rardecode/v2 v2.4.6
	github.com/syndtr/goleveldb v1.0.1-0.20220721030215-126854af5e6d
	golang.org/x/crypto v0.59.0
	golang.org/x/sys v0.50.0
)

require (
	github.com/andybalholm/brotli v1.2.7 // indirect
	github.com/bodgit/plumbing v1.3.0 // indirect
	github.com/bodgit/windows v1.0.1 // indirect
	github.com/dlclark/regexp2/v2 v2.11.0 // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/pprof v0.0.0-20230207041349-798e818bf904 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/jchv/go-winloader v0.0.0-20250406163304-c1995be93bd1 // indirect
	github.com/klauspost/compress v1.20.2 // indirect
	github.com/pierrec/lz4/v4 v4.1.27 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/stangelandcl/ppmd v0.1.1 // indirect
	github.com/ulikunitz/xz v0.5.15 // indirect
	go4.org v0.0.0-20260112195520-a5071408f32f // indirect
	golang.org/x/text v0.44.0 // indirect
)

replace (
	github.com/jchv/go-webview2 => ./third_party/go-webview2
	github.com/jchv/go-winloader => ./third_party/go-winloader
	golang.org/x/sys => ./third_party/sys
)
