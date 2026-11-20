//go:build !windows

package app

// RunNexusWindow has no window of Lootnest's own on this system yet; the
// app opens the page in the browser instead (app/nexus.go).
func RunNexusWindow(arg string) int { return 1 }

// nexusOwnWindow says whether Lootnest shows Nexus in a window of its own.
const nexusOwnWindow = false
