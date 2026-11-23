package app

import (
	"strings"
	"testing"
)

// WebView2 shows nothing at all for a page over NavigateToString's limit
// (2.23.0's empty window), so the page stays well under it.
func TestPageFitsNavigateToString(t *testing.T) {
	h := pageHTML()
	if len(h) > maxPageHTML {
		t.Fatalf("the page is %d bytes; keep it under %d (move big parts out, as the script is)", len(h), maxPageHTML)
	}
	if strings.Contains(h, "/*@JS*/") || !strings.Contains(h, `ps_vendor("app")`) {
		t.Fatal("the page has no loader for its script")
	}
	if !strings.Contains(appScript(), "function boot()") || strings.Contains(appScript(), "/*@SOUNDS*/{}") {
		t.Fatal("the script isn't whole, or has no sounds")
	}
	t.Logf("page %d bytes, script %d", len(h), len(appScript()))
}
