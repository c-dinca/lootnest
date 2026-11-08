package core

import (
	"strings"
	"testing"
)

func TestNewsHTML(t *testing.T) {
	in := `[p]Hello [b]everyone[/b] & friends,[/p][p][img src="{STEAM_CLAN_IMAGE}/1/a.png"][/img][/p]` +
		`[h2]Fixes[/h2][list][*]One[*]Two[/list][p]More: [url="https://fanga.me/r/x"]the shop[/url][/p]` +
		`[img]{STEAM_CLAN_LOC_IMAGE}/1/b.jpg[/img][previewyoutube=dQw4w9WgXcQ;full][/previewyoutube]` +
		`[dynamiclink href="https://www.twitch.tv/cdprojektred"][/dynamiclink][color=red]red[/color]`
	got := NewsHTML(in)
	for _, want := range []string{
		`<p>Hello <b>everyone</b> &amp; friends,</p>`,
		`<img loading="lazy" alt="" src="https://clan.akamai.steamstatic.com/images/1/a.png">`,
		`<h4>Fixes</h4><ul><li>One</li><li>Two</li></ul>`,
		`<span class="link" data-action="open-link" data-url="https://fanga.me/r/x">the shop</span>`,
		`src="https://clan.akamai.steamstatic.com/images/1/b.jpg"`,
		`data-url="https://www.youtube.com/watch?v=dQw4w9WgXcQ">Watch the video on YouTube</span>`,
		`data-url="https://www.twitch.tv/cdprojektred">twitch.tv/cdprojektred</span>`,
		`red`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s in\n%s", want, got)
		}
	}
}

func TestNewsHTMLIsSafe(t *testing.T) {
	for _, in := range []string{
		`<script>alert(1)</script>`,
		`[img src="javascript:alert(1)"][/img]`,
		`[img]https://evil.example/x.png[/img]`,
		`[url="javascript:alert(1)"]x[/url]`,
		`[url=https://a.example/"onmouseover="x]x[/url]`,
		`[p onclick="x"]hi[/p]`,
		`[b]unclosed [i]tags`,
	} {
		got := NewsHTML(in)
		low := strings.ToLower(got)
		if strings.Contains(low, "<script") || strings.Contains(low, "javascript:") || strings.Contains(low, "evil.example") ||
			strings.Contains(low, `"onmouseover`) || strings.Contains(low, "onclick") {
			t.Errorf("unsafe output for %q: %s", in, got)
		}
	}
	if got := NewsHTML(`[b]unclosed [i]tags`); got != `<b>unclosed <i>tags</i></b>` {
		t.Errorf("open tags are closed: %s", got)
	}
}

func TestNewsSummaryAndImage(t *testing.T) {
	in := `[p][img src="{STEAM_CLAN_IMAGE}/45164767/607.png"][/img][/p][p]As Halloween draws near, the residents of the town...[/p]`
	if s := newsSummary(in, 240); s != "As Halloween draws near, the residents of the town..." {
		t.Errorf("summary without images: %q", s)
	}
	if s := newsSummary("{STEAM_CLAN_LOC_IMAGE}/45164767/11ea2 It's hard to know", 240); s != "It's hard to know" {
		t.Errorf("bare image placeholders are dropped: %q", s)
	}
	if img := newsImage(in); img != "https://clan.akamai.steamstatic.com/images/45164767/607.png" {
		t.Errorf("first image: %q", img)
	}
}

func TestAllowedPageRequest(t *testing.T) {
	for u, want := range map[string]bool{
		"about:blank":                true,
		"data:image/png;base64,AAAA": true,
		"https://clan.akamai.steamstatic.com/images/1/a.png": true,
		"https://evil.example/x.js":                          false,
		"http://clan.akamai.steamstatic.com/images/1/a.png":  false,
		"file:///C:/Windows/win.ini":                         false,
		"https://steamstatic.com.evil.example/a.png":         false,
	} {
		if got := AllowedPageRequest(u); got != want {
			t.Errorf("AllowedPageRequest(%q) = %v", u, got)
		}
	}
}

func TestPageSource(t *testing.T) {
	var p PageSource
	if p.Trusted("") {
		t.Error("an empty source was trusted")
	}
	page := "data:text/html;charset=utf-8;base64,PGh0bWw+"
	if !p.Trusted(page) {
		t.Fatal("the page's own first call wasn't trusted")
	}
	for src, want := range map[string]bool{
		page:                                   true,
		page + "PHNjcmlwdD4=":                  false,
		"data:text/html;charset=utf-8;base64,": false,
		"about:blank":                          false,
		"https://evil.example/":                false,
		"":                                     false,
	} {
		if got := p.Trusted(src); got != want {
			t.Errorf("Trusted(%q) = %v", src, got)
		}
	}
}
