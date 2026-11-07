package core

import (
	"html"
	"net/url"
	"regexp"
	"strings"
)

// ---------------------------------------------------------------------------
// Steam announcements are written in BBCode. NewsHTML turns one into a small,
// safe subset of HTML for Lootnest's reader: every piece of text is escaped,
// only known tags become elements, images load only from Steam's own image
// servers, and links are not links at all but spans the UI opens in the
// browser when clicked (so nothing can navigate Lootnest's window).
// ---------------------------------------------------------------------------

const clanImages = "https://clan.akamai.steamstatic.com/images"

var youtubeID = regexp.MustCompile(`^[A-Za-z0-9_-]{6,20}$`)

var clanPlaceholder = regexp.MustCompile(`\{STEAM_CLAN(?:_LOC)?_IMAGE\}`)

// expandClanImages replaces Steam's image placeholders with the real host.
func expandClanImages(s string) string { return clanPlaceholder.ReplaceAllString(s, clanImages) }

// SafeNewsImage accepts only images on Steam's own servers.
func SafeNewsImage(u string) bool {
	p, err := url.Parse(u)
	if err != nil || p.Scheme != "https" {
		return false
	}
	h := strings.ToLower(p.Hostname())
	return strings.HasSuffix(h, ".steamstatic.com") || h == "steamcdn-a.akamaihd.net" || h == "images.steamusercontent.com" ||
		strings.HasSuffix(h, ".steamusercontent.com")
}

// SafeLink accepts web links (opened in the browser, never in Lootnest).
func SafeLink(u string) string {
	u = strings.TrimSpace(strings.Trim(u, `"'`))
	p, err := url.Parse(u)
	if err != nil || (p.Scheme != "https" && p.Scheme != "http") || p.Host == "" {
		return ""
	}
	return p.String()
}

type bbTag struct {
	name  string // lower case, without "/"
	arg   string // after "=", or the src/href attribute
	close bool
	raw   string
}

var bbAttr = regexp.MustCompile(`(?i)\b(src|href)\s*=\s*"([^"]*)"`)

func parseTag(raw string) (bbTag, bool) {
	inner := raw[1 : len(raw)-1]
	t := bbTag{raw: raw}
	if strings.HasPrefix(inner, "/") {
		t.close = true
		inner = inner[1:]
	}
	if inner == "" {
		return t, false
	}
	name := inner
	if i := strings.IndexAny(inner, "= "); i >= 0 {
		name = inner[:i]
		rest := inner[i:]
		if m := bbAttr.FindStringSubmatch(rest); m != nil {
			t.arg = m[2]
		} else if strings.HasPrefix(rest, "=") {
			t.arg = strings.Trim(strings.TrimSpace(rest[1:]), `"'`)
		}
	}
	t.name = strings.ToLower(name)
	for _, r := range t.name {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '*') {
			return t, false
		}
	}
	return t, true
}

// htmlFor maps the tags Lootnest shows to the element it uses for them.
var htmlFor = map[string]string{
	"p": "p", "b": "b", "i": "i", "u": "u", "s": "s", "strike": "s", "h1": "h3", "h2": "h4", "h3": "h4", "h4": "h5", "h5": "h5",
	"list": "ul", "olist": "ol", "quote": "blockquote", "code": "pre", "table": "table", "tr": "tr", "td": "td", "th": "th",
	"spoiler": "span", "center": "p",
}

// NewsHTML converts a Steam announcement to safe HTML for the reader.
func NewsHTML(bb string) string {
	bb = expandClanImages(strings.ReplaceAll(bb, "\r\n", "\n"))
	var out strings.Builder
	var stack []string // open bbcode tag names
	inLi := false
	text := func(s string) {
		s = html.EscapeString(html.UnescapeString(s))
		out.WriteString(strings.ReplaceAll(s, "\n", "<br>"))
	}
	closeLi := func() {
		if inLi {
			out.WriteString("</li>")
			inLi = false
		}
	}
	writeOpen := func(name string) {
		el := htmlFor[name]
		switch name {
		case "spoiler":
			out.WriteString(`<span class="spoiler">`)
		case "center":
			out.WriteString(`<p class="center">`)
		default:
			out.WriteString("<" + el + ">")
		}
	}
	popTo := func(name string) {
		at := -1
		for i := len(stack) - 1; i >= 0; i-- {
			if stack[i] == name {
				at = i
				break
			}
		}
		if at < 0 {
			return
		}
		for i := len(stack) - 1; i >= at; i-- {
			n := stack[i]
			if n == "list" || n == "olist" {
				closeLi()
			}
			if n == "url" || n == "dynamiclink" {
				out.WriteString("</span>")
			} else if el := htmlFor[n]; el != "" {
				out.WriteString("</" + el + ">")
			}
		}
		stack = stack[:at]
	}

	for len(bb) > 0 {
		i := strings.IndexByte(bb, '[')
		if i < 0 {
			text(bb)
			break
		}
		j := strings.IndexByte(bb[i:], ']')
		if j < 0 {
			text(bb)
			break
		}
		if i > 0 {
			text(bb[:i])
		}
		raw := bb[i : i+j+1]
		bb = bb[i+j+1:]
		t, ok := parseTag(raw)
		if !ok {
			text(raw)
			continue
		}
		switch {
		case t.name == "noparse" && !t.close:
			end := strings.Index(strings.ToLower(bb), "[/noparse]")
			if end < 0 {
				end = len(bb)
			}
			text(bb[:end])
			bb = bb[min(len(bb), end+len("[/noparse]")):]
		case t.name == "img" && !t.close:
			src := t.arg
			if src == "" { // [img]url[/img]
				end := strings.Index(strings.ToLower(bb), "[/img]")
				if end < 0 {
					end = len(bb)
				}
				src = strings.TrimSpace(bb[:end])
				bb = bb[min(len(bb), end+len("[/img]")):]
			}
			if SafeNewsImage(src) {
				out.WriteString(`<img loading="lazy" alt="" src="` + html.EscapeString(src) + `">`)
			}
		case t.name == "img" && t.close:
			// [img src=...][/img]: nothing to close
		case t.name == "previewyoutube" && !t.close:
			id := strings.SplitN(t.arg, ";", 2)[0]
			if youtubeID.MatchString(id) {
				u := "https://www.youtube.com/watch?v=" + id
				out.WriteString(`<span class="link video" data-action="open-link" data-url="` + html.EscapeString(u) + `">Watch the video on YouTube</span>`)
			}
		case t.name == "previewyoutube" && t.close:
		case t.name == "hr":
			if !t.close {
				out.WriteString("<hr>")
			}
		case t.name == "*" && !t.close:
			closeLi()
			out.WriteString("<li>")
			inLi = true
		case (t.name == "url" || t.name == "dynamiclink") && !t.close:
			u := SafeLink(t.arg)
			if u == "" && t.name == "url" { // [url]https://...[/url]
				end := strings.Index(strings.ToLower(bb), "[/url]")
				if end > 0 {
					u = SafeLink(bb[:end])
				}
			}
			if u == "" {
				continue
			}
			out.WriteString(`<span class="link" data-action="open-link" data-url="` + html.EscapeString(u) + `">`)
			if t.name == "dynamiclink" {
				if p, err := url.Parse(u); err == nil {
					text(strings.TrimPrefix(p.Hostname(), "www.") + p.EscapedPath())
				}
			}
			stack = append(stack, t.name)
		case t.close:
			popTo(t.name)
		case htmlFor[t.name] != "":
			writeOpen(t.name)
			stack = append(stack, t.name)
		default:
			// tags Lootnest doesn't show (emoticons, colours, ...): left out
		}
	}
	for len(stack) > 0 {
		popTo(stack[len(stack)-1])
	}
	closeLi()
	return out.String()
}

var (
	bbImageBlock = regexp.MustCompile(`(?is)\[img[^\]]*\].*?\[/img\]|\[img[^\]]*\]`)
	bbVideo      = regexp.MustCompile(`(?is)\[previewyoutube[^\]]*\].*?\[/previewyoutube\]`)
	bareImage    = regexp.MustCompile(`\{STEAM_CLAN(?:_LOC)?_IMAGE\}\S*`)
	firstImage   = regexp.MustCompile(`(?i)\[img(?:\s+src="([^"]+)")?\]([^\[]*)`)
)

// newsSummary is an announcement's opening text, without images or videos.
func newsSummary(bb string, max int) string {
	s := bbImageBlock.ReplaceAllString(bb, " ")
	s = bbVideo.ReplaceAllString(s, " ")
	s = bareImage.ReplaceAllString(s, " ")
	s = plainText(s)
	if r := []rune(s); len(r) > max {
		s = strings.TrimSpace(string(r[:max])) + "…"
	}
	return s
}

// newsImage is the first picture in an announcement, for its thumbnail.
func newsImage(bb string) string {
	m := firstImage.FindStringSubmatch(expandClanImages(bb))
	if m == nil {
		return ""
	}
	src := m[1]
	if src == "" {
		src = strings.TrimSpace(m[2])
	}
	if SafeNewsImage(src) {
		return src
	}
	return ""
}
