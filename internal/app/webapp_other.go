//go:build !windows

package app

// RunWebApp has no window to show a site in yet on this system
// (docs/ROADMAP.md, section 9), so it ends at once with 1, as it does on
// Windows when WebView2 can't start.
func RunWebApp(id string, on uintptr) int { return 1 }
