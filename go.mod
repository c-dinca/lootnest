module lootnest

go 1.24

toolchain go1.27.3

require (
	github.com/jchv/go-webview2 v0.0.0-00010101000000-000000000000
	github.com/syndtr/goleveldb v1.0.1-0.20220721030215-126854af5e6d
	golang.org/x/sys v0.33.0
)

require (
	github.com/golang/snappy v0.0.4 // indirect
	github.com/jchv/go-winloader v0.0.0-20250406163304-c1995be93bd1 // indirect
)

replace (
	github.com/jchv/go-webview2 => ./third_party/go-webview2
	github.com/jchv/go-winloader => ./third_party/go-winloader
	golang.org/x/sys => ./third_party/sys
)
