package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// ---------------------------------------------------------------------------
// Nexus Mods (the owner, 20 Nov 2026: mods found and added without leaving
// Lootnest, Nexus as the source). Browsing reads Nexus's public GraphQL API
// (v2), which answers without a key: games, mods, a mod's page, its files
// and what it needs. Lootnest never downloads a mod itself: the user
// downloads the file on nexusmods.com (in Lootnest's window for the site,
// or their browser), it lands in their Downloads folder, and Lootnest
// finds it there by the mod's id in its name and offers to add it to the
// game's mods. Signing in and one-click downloads need Lootnest registered
// with Nexus (docs/ROADMAP.md, Mods 5). Adult mods are never listed.
// ---------------------------------------------------------------------------

// NexusGraphQL is Nexus's public API address.
const NexusGraphQL = "https://api.nexusmods.com/v2/graphql"

// NexusPost sends one request body to addr and returns the answer's body.
type NexusPost func(addr string, body []byte) ([]byte, error)

// NexusGame is a game on Nexus.
type NexusGame struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Domain string `json:"domain"`
	Mods   int    `json:"mods"`
}

// NexusMod is one mod in a list.
type NexusMod struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Author       string    `json:"author,omitempty"`
	Version      string    `json:"version,omitempty"`
	Summary      string    `json:"summary,omitempty"`
	Category     string    `json:"category,omitempty"`
	Endorsements int       `json:"endorsements"`
	Downloads    int       `json:"downloads"`
	Thumb        string    `json:"thumb,omitempty"`
	Picture      string    `json:"picture,omitempty"`
	Updated      time.Time `json:"updated"`
}

// NexusFile is one file of a mod. Category is Nexus's: MAIN, OPTIONAL,
// UPDATE, MISCELLANEOUS or OLD_VERSION (archived and removed ones are left
// out).
type NexusFile struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version,omitempty"`
	Category    string `json:"category"`
	Description string `json:"description,omitempty"`
	Size        int64  `json:"size"`
	Date        int64  `json:"date"` // Unix seconds; also the end of the downloaded file's name
	Primary     bool   `json:"primary,omitempty"`
}

// NexusNeed is something a mod needs: another mod on Nexus, or a program
// from elsewhere (External, with its address).
type NexusNeed struct {
	ModID    int    `json:"modId,omitempty"`
	Name     string `json:"name"`
	Notes    string `json:"notes,omitempty"`
	URL      string `json:"url,omitempty"`
	External bool   `json:"external,omitempty"`
}

// NexusModPage is a mod's page: its details, its description as text,
// what it needs and its files, newest first in each category.
type NexusModPage struct {
	NexusMod
	Description string      `json:"description,omitempty"`
	Needs       []NexusNeed `json:"needs"`
	Files       []NexusFile `json:"files"`
	Adult       bool        `json:"adult,omitempty"`
}

// NexusSearchSize is how many mods one page of a search holds.
const NexusSearchSize = 24

// nexusQuery sends a GraphQL query and reads its data into out.
func nexusQuery(post NexusPost, query string, vars map[string]any, out any) error {
	body, err := json.Marshal(map[string]any{"query": query, "variables": vars})
	if err != nil {
		return err
	}
	b, err := post(NexusGraphQL, body)
	if err != nil {
		return err
	}
	var res struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message    string `json:"message"`
			Extensions struct {
				Code string `json:"code"`
			} `json:"extensions"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(b, &res); err != nil {
		return fmt.Errorf("Nexus Mods answered something Lootnest can't read")
	}
	if len(res.Errors) > 0 {
		e := res.Errors[0]
		if strings.HasSuffix(e.Extensions.Code, "NOT_FOUND") || strings.HasSuffix(strings.ToLower(e.Message), "not found") {
			return errNexusNone
		}
		return fmt.Errorf("Nexus Mods said: %s", res.Errors[0].Message)
	}
	if len(res.Data) == 0 || string(res.Data) == "null" {
		return errNexusNone
	}
	return json.Unmarshal(res.Data, out)
}

var errNexusNone = errors.New("not on Nexus Mods")

// nexusDomain is the domain Nexus most likely gives a game of this name:
// its letters and digits in lower case ("Baldur's Gate 3" is baldursgate3).
func nexusDomain(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if r < utf8.RuneSelf && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// nexusPlain is a game's name for comparing: words of letters and digits.
func nexusPlain(name string) string {
	name = strings.NewReplacer("™", "", "®", "", "©", "", "'", "", "’", "").Replace(name)
	return strings.Join(nameTokens(name), " ")
}

type nexusGameNode struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	DomainName string `json:"domainName"`
	ModCount   int    `json:"modCount"`
}

func (n nexusGameNode) game() *NexusGame {
	return &NexusGame{ID: n.ID, Name: n.Name, Domain: n.DomainName, Mods: n.ModCount}
}

// NexusFindGame is the game on Nexus with this name, or nil when Nexus has
// none: first by the domain its name would have, then by the name itself,
// then by its words, where only a game with the same words counts.
func NexusFindGame(post NexusPost, name string) (*NexusGame, error) {
	plain := nexusPlain(name)
	if plain == "" {
		return nil, nil
	}
	if d := nexusDomain(name); d != "" {
		var one struct {
			Game *nexusGameNode `json:"game"`
		}
		err := nexusQuery(post, `query G($d: String!) { game(domainName: $d) { id name domainName modCount } }`, map[string]any{"d": d}, &one)
		if err == nil && one.Game != nil && one.Game.ID > 0 {
			return one.Game.game(), nil
		}
		if err != nil && !errors.Is(err, errNexusNone) {
			return nil, err
		}
	}
	clean := strings.Join(strings.Fields(strings.NewReplacer("™", "", "®", "", "©", "").Replace(name)), " ")
	for _, op := range []string{"EQUALS", "MATCHES"} {
		var many struct {
			Games struct {
				Nodes []nexusGameNode `json:"nodes"`
			} `json:"games"`
		}
		q := `query G($n: String!) { games(filter: {name: [{value: $n, op: ` + op + `}]}, count: 10) { nodes { id name domainName modCount } } }`
		if err := nexusQuery(post, q, map[string]any{"n": clean}, &many); err != nil && !errors.Is(err, errNexusNone) {
			return nil, err
		}
		var best *nexusGameNode
		for i, g := range many.Games.Nodes {
			if nexusPlain(g.Name) == plain && (best == nil || g.ModCount > best.ModCount) {
				best = &many.Games.Nodes[i]
			}
		}
		if best != nil {
			return best.game(), nil
		}
	}
	// By its words, as "The Witcher® 3: Wild Hunt" is found.
	games, err := NexusFindGames(post, clean)
	if err != nil {
		return nil, err
	}
	for _, g := range games {
		if nexusPlain(g.Name) == plain {
			return &g, nil
		}
	}
	return nil, nil
}

// NexusFindGames lists Nexus's games whose names hold every word of q,
// those with the most mods first: for choosing one by hand.
func NexusFindGames(post NexusPost, q string) ([]NexusGame, error) {
	words := nameTokens(q)
	if len(words) == 0 {
		return []NexusGame{}, nil
	}
	if len(words) > 6 {
		words = words[:6]
	}
	parts := make([]map[string]any, len(words))
	for i, w := range words {
		parts[i] = map[string]any{"name": []map[string]any{{"value": w, "op": "WILDCARD"}}}
	}
	var res struct {
		Games struct {
			Nodes []nexusGameNode `json:"nodes"`
		} `json:"games"`
	}
	err := nexusQuery(post, `query G($f: GamesSearchFilter) { games(filter: $f, count: 20, sort: {downloads: {direction: DESC}}) { nodes { id name domainName modCount } } }`,
		map[string]any{"f": map[string]any{"op": "AND", "filter": parts}}, &res)
	if err != nil && !errors.Is(err, errNexusNone) {
		return nil, err
	}
	out := []NexusGame{}
	for _, g := range res.Games.Nodes {
		if g.ID > 0 && nexusDomainOK(g.DomainName) {
			out = append(out, *g.game())
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Mods > out[j].Mods })
	return out, nil
}

var nexusDomainRe = regexp.MustCompile(`^[a-z0-9]{1,64}$`)

func nexusDomainOK(d string) bool { return nexusDomainRe.MatchString(d) }

// NexusSorts are the orders a search can take; the first is the default.
var NexusSorts = []string{"endorsements", "downloads", "updated", "new", "name"}

type nexusModNode struct {
	ModID        int    `json:"modId"`
	Name         string `json:"name"`
	Author       string `json:"author"`
	Version      string `json:"version"`
	Summary      string `json:"summary"`
	Endorsements int    `json:"endorsements"`
	Downloads    int    `json:"downloads"`
	ThumbnailURL string `json:"thumbnailUrl"`
	PictureURL   string `json:"pictureUrl"`
	UpdatedAt    string `json:"updatedAt"`
	AdultContent bool   `json:"adultContent"`
	Status       string `json:"status"`
	ModCategory  *struct {
		Name string `json:"name"`
	} `json:"modCategory"`
}

const nexusModFields = `modId name author version summary endorsements downloads thumbnailUrl pictureUrl updatedAt adultContent status modCategory { name }`

func (n nexusModNode) mod() NexusMod {
	m := NexusMod{ID: n.ModID, Name: oneLine(n.Name), Author: oneLine(n.Author), Version: oneLine(n.Version), Summary: nexusSummary(n.Summary),
		Endorsements: n.Endorsements, Downloads: n.Downloads}
	if SafeNexusImage(n.ThumbnailURL) {
		m.Thumb = n.ThumbnailURL
	}
	if SafeNexusImage(n.PictureURL) {
		m.Picture = n.PictureURL
	}
	if n.ModCategory != nil {
		m.Category = oneLine(n.ModCategory.Name)
	}
	m.Updated, _ = time.Parse(time.RFC3339, n.UpdatedAt)
	return m
}

func oneLine(s string) string { return strings.Join(strings.Fields(html.UnescapeString(s)), " ") }

// NexusSearch is one page of a game's mods: those whose names hold every
// word of q (all of them when q is empty), in the order sort names (one
// of NexusSorts), from offset; and how many there are in all.
func NexusSearch(post NexusPost, gameID int, q, order string, offset int) ([]NexusMod, int, error) {
	if gameID <= 0 {
		return nil, 0, errors.New("unknown game")
	}
	f := map[string]any{
		"op":           "AND",
		"gameId":       []map[string]any{{"value": strconv.Itoa(gameID), "op": "EQUALS"}},
		"adultContent": []map[string]any{{"value": false, "op": "EQUALS"}},
	}
	words := nameTokens(q)
	if len(words) > 8 {
		words = words[:8]
	}
	if len(words) > 0 {
		parts := make([]map[string]any, len(words))
		for i, w := range words {
			parts[i] = map[string]any{"name": []map[string]any{{"value": w, "op": "WILDCARD"}}}
		}
		f["filter"] = parts
	}
	key, dir := "endorsements", "DESC"
	switch order {
	case "downloads":
		key = "downloads"
	case "updated":
		key = "updatedAt"
	case "new":
		key = "createdAt"
	case "name":
		key, dir = "name", "ASC"
	}
	if offset < 0 {
		offset = 0
	}
	var res struct {
		Mods struct {
			TotalCount int            `json:"totalCount"`
			Nodes      []nexusModNode `json:"nodes"`
		} `json:"mods"`
	}
	query := `query S($f: ModsFilter, $s: [ModsSort!], $o: Int, $c: Int) { mods(filter: $f, sort: $s, offset: $o, count: $c) { totalCount nodes { ` + nexusModFields + ` } } }`
	err := nexusQuery(post, query, map[string]any{"f": f, "s": []map[string]any{{key: map[string]any{"direction": dir}}}, "o": offset, "c": NexusSearchSize}, &res)
	if err != nil && !errors.Is(err, errNexusNone) {
		return nil, 0, err
	}
	out := []NexusMod{}
	for _, n := range res.Mods.Nodes {
		if n.ModID > 0 && !n.AdultContent && (n.Status == "" || n.Status == "published") {
			out = append(out, n.mod())
		}
	}
	return out, res.Mods.TotalCount, nil
}

// NexusModInfo is a mod's page, or nil when Nexus has no such mod or it
// isn't one Lootnest shows (adult, hidden or removed).
func NexusModInfo(post NexusPost, gameID, modID int) (*NexusModPage, error) {
	if gameID <= 0 || modID <= 0 {
		return nil, errors.New("unknown mod")
	}
	var res struct {
		Mod *struct {
			nexusModNode
			Description     string `json:"description"`
			ModRequirements *struct {
				Nexus struct {
					Nodes []struct {
						ModID    json.RawMessage `json:"modId"`
						ModName  string          `json:"modName"`
						URL      string          `json:"url"`
						Notes    string          `json:"notes"`
						External bool            `json:"externalRequirement"`
					} `json:"nodes"`
				} `json:"nexusRequirements"`
			} `json:"modRequirements"`
		} `json:"mod"`
		Files []struct {
			FileID      int             `json:"fileId"`
			Name        string          `json:"name"`
			Version     string          `json:"version"`
			Category    string          `json:"category"`
			Description string          `json:"description"`
			Size        json.RawMessage `json:"sizeInBytes"`
			Date        int64           `json:"date"`
			Primary     int             `json:"primary"`
		} `json:"modFiles"`
	}
	query := `query M($g: ID!, $m: ID!) { mod(gameId: $g, modId: $m) { ` + nexusModFields + ` description
	  modRequirements { nexusRequirements { nodes { modId modName url notes externalRequirement } } } }
	  modFiles(gameId: $g, modId: $m) { fileId name version category description sizeInBytes date primary } }`
	err := nexusQuery(post, query, map[string]any{"g": strconv.Itoa(gameID), "m": strconv.Itoa(modID)}, &res)
	if errors.Is(err, errNexusNone) || err == nil && res.Mod == nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	n := res.Mod
	if n.AdultContent || n.Status != "" && n.Status != "published" {
		return nil, nil
	}
	p := &NexusModPage{NexusMod: n.mod(), Description: NexusText(n.Description), Needs: []NexusNeed{}, Files: []NexusFile{}}
	if n.ModRequirements != nil {
		for _, r := range n.ModRequirements.Nexus.Nodes {
			need := NexusNeed{Name: oneLine(r.ModName), Notes: oneLine(r.Notes), External: r.External}
			need.ModID, _ = strconv.Atoi(strings.Trim(string(r.ModID), `"`))
			if r.External || need.ModID <= 0 {
				need.External, need.ModID = true, 0
				need.URL = SafeLink(r.URL)
			}
			if need.Name != "" {
				p.Needs = append(p.Needs, need)
			}
		}
	}
	for _, f := range res.Files {
		cat := strings.ToUpper(f.Category)
		if f.FileID <= 0 || cat == "REMOVED" || cat == "ARCHIVED" || cat == "DELETED" {
			continue
		}
		size, _ := strconv.ParseInt(strings.Trim(string(f.Size), `"`), 10, 64)
		p.Files = append(p.Files, NexusFile{ID: f.FileID, Name: oneLine(f.Name), Version: oneLine(f.Version), Category: cat,
			Description: NexusText(f.Description), Size: size, Date: f.Date, Primary: f.Primary != 0})
	}
	rank := map[string]int{"MAIN": 0, "UPDATE": 1, "OPTIONAL": 2, "MISCELLANEOUS": 3, "OLD_VERSION": 4}
	sort.SliceStable(p.Files, func(i, j int) bool {
		a, b := p.Files[i], p.Files[j]
		ra, ok := rank[a.Category]
		if !ok {
			ra = 3
		}
		rb, ok := rank[b.Category]
		if !ok {
			rb = 3
		}
		if ra != rb {
			return ra < rb
		}
		if a.Primary != b.Primary {
			return a.Primary
		}
		return a.Date > b.Date
	})
	return p, nil
}

var (
	nexusBreak  = regexp.MustCompile(`(?i)<br\s*/?>|\[/?(?:line|hr)\]`)
	nexusDrop   = regexp.MustCompile(`(?is)\[(img|youtube|video|spoiler)(?:=[^\]]*)?\].*?\[/(?:img|youtube|video|spoiler)\]`)
	nexusURL    = regexp.MustCompile(`(?is)\[url(?:=[^\]]*)?\](.*?)\[/url\]`)
	nexusBB     = regexp.MustCompile(`(?i)\[/?(?:b|i|u|s|h[1-6]|strike|noparse|list|olist|\*|quote|code|table|tr|th|td|url|size|color|colour|font|center|left|right|justify|heading|indent|sub|sup|email|ol|ul|li)(?:=[^\]]*)?\]`)
	nexusLongNL = regexp.MustCompile(`\n{3,}`)
)

// nexusTextMax is how much of a description Lootnest shows.
const nexusTextMax = 6000

// NexusText turns a mod's BBCode description (with Nexus's <br /> lines
// and escaped characters) into plain text with paragraphs. Pictures and
// videos are left out; a link keeps its words.
func NexusText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = nexusBreak.ReplaceAllString(s, "\n")
	s = nexusDrop.ReplaceAllString(s, "")
	s = nexusURL.ReplaceAllString(s, "$1")
	s = nexusBB.ReplaceAllString(s, "")
	s = htmlTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = strings.Map(func(r rune) rune {
		switch {
		case r == 0xFEFF || r == 0x200B || r == 0x200C || r == 0x200D:
			return -1
		case r == '\n' || r == '\t':
			return r
		case unicode.IsControl(r):
			return -1
		}
		return r
	}, s)
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(spaceRun.ReplaceAllString(l, " "))
	}
	s = strings.TrimSpace(nexusLongNL.ReplaceAllString(strings.Join(lines, "\n"), "\n\n"))
	if utf8.RuneCountInString(s) > nexusTextMax {
		r := []rune(s)[:nexusTextMax]
		s = strings.TrimSpace(string(r)) + "…"
	}
	return s
}

// nexusSummary is a list's one-paragraph summary.
func nexusSummary(s string) string {
	return strings.Join(strings.Fields(NexusText(s)), " ")
}

// SafeNexusImage accepts only pictures on Nexus's own image servers.
func SafeNexusImage(u string) bool {
	p, err := url.Parse(u)
	if err != nil || p.Scheme != "https" || p.User != nil || p.Port() != "" {
		return false
	}
	h := strings.ToLower(p.Hostname())
	return h == "staticdelivery.nexusmods.com" || h == "images.nexusmods.com"
}

// NexusPage says whether u is a page of nexusmods.com itself (the site,
// its sign-in and its files' pages), which Lootnest's window for the site
// may show.
func NexusPage(u string) bool {
	p, err := url.Parse(u)
	if err != nil || p.Scheme != "https" || p.User != nil || p.Port() != "" {
		return false
	}
	h := strings.ToLower(p.Hostname())
	return h == "nexusmods.com" || strings.HasSuffix(h, ".nexusmods.com")
}

// NexusModURL is a mod's page on nexusmods.com.
func NexusModURL(domain string, modID int) string {
	if !nexusDomainOK(domain) || modID <= 0 {
		return ""
	}
	return fmt.Sprintf("https://www.nexusmods.com/%s/mods/%d", domain, modID)
}

// NexusFileURL is the page on nexusmods.com where a mod's file is
// downloaded (Manual download).
func NexusFileURL(domain string, modID, fileID int) string {
	base := NexusModURL(domain, modID)
	if base == "" || fileID <= 0 {
		return ""
	}
	return fmt.Sprintf("%s?tab=files&file_id=%d", base, fileID)
}

// ---------------------------------------------------------------------------
// The downloaded file. Nexus names a file it serves
// "<name>-<mod id>-<version>-<upload time>.<ext>" (the version's dots as
// dashes, the time in Unix seconds), and a browser adds " (1)" to a name
// it already has. A file in Downloads that came after the user asked for
// one is matched to that ask by the mod id and the upload time in its
// name; by the mod id alone when the time differs (another file of the
// same mod); and, when only one ask is open, by being the one archive that
// arrived since.
// ---------------------------------------------------------------------------

// NexusWait is a file the user went to download from Nexus.
type NexusWait struct {
	Key    string    `json:"key"`  // the Library game it's for
	Game   string    `json:"game"` // its name
	Domain string    `json:"domain"`
	ModID  int       `json:"modId"`
	FileID int       `json:"fileId"`
	Mod    string    `json:"mod"`  // the mod's name
	File   string    `json:"file"` // the file's name
	Date   int64     `json:"date"` // the file's upload time
	Since  time.Time `json:"since"`
}

// NexusArrival is a file in Downloads matched to a NexusWait.
type NexusArrival struct {
	NexusWait
	Path  string    `json:"path"`
	Name  string    `json:"name"`
	Size  int64     `json:"size"`
	At    time.Time `json:"at"`
	Guess bool      `json:"guess,omitempty"` // matched only by arriving while it was the one ask open
}

var (
	browserCopy = regexp.MustCompile(` \(\d{1,3}\)$`)
	nexusName   = regexp.MustCompile(`-(\d{1,7})(?:-[0-9A-Za-z]+)*-(\d{9,11})$`)
)

// nexusStem is a file's name without its extension or a browser's " (1)".
func nexusStem(name string) string {
	return browserCopy.ReplaceAllString(strings.TrimSuffix(name, filepath.Ext(name)), "")
}

// NexusFileName says whether a file's name has the shape of one Nexus
// serves, and the mod id it most likely names.
func NexusFileName(name string) (int, bool) {
	m := nexusName.FindStringSubmatch(nexusStem(name))
	if m == nil {
		return 0, false
	}
	id, _ := strconv.Atoi(m[1])
	return id, id > 0
}

// NexusArrivals looks at the top of dir for archives that came after the
// asks in waits and matches each ask to at most one of them (the newest
// that fits). A file still downloading is left until it's done.
func NexusArrivals(dir string, waits []NexusWait) []NexusArrival {
	if len(waits) == 0 {
		return nil
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	type file struct {
		name string
		size int64
		at   time.Time
	}
	var files []file
	oldest := waits[0].Since
	for _, w := range waits {
		if w.Since.Before(oldest) {
			oldest = w.Since
		}
	}
	for _, e := range ents {
		if e.IsDir() || downloading(e.Name()) {
			continue
		}
		switch downloadKind(e.Name(), false) {
		case "zip", "7z", "rar":
		default:
			continue
		}
		info, err := e.Info()
		if err != nil || info.Size() == 0 || info.ModTime().Before(oldest.Add(-10*time.Second)) {
			continue
		}
		files = append(files, file{e.Name(), info.Size(), info.ModTime()})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].at.After(files[j].at) })
	used := map[string]bool{}
	out := []NexusArrival{}
	take := func(w NexusWait, f file, guess bool) {
		used[f.name] = true
		out = append(out, NexusArrival{NexusWait: w, Path: filepath.Join(dir, f.name), Name: f.name, Size: f.size, At: f.at, Guess: guess})
	}
	matched := map[int]bool{}
	// By the mod and the file's upload time first, then by the mod.
	for pass := 0; pass < 2; pass++ {
		for i, w := range waits {
			if matched[i] {
				continue
			}
			for _, f := range files {
				if used[f.name] || f.at.Before(w.Since.Add(-10*time.Second)) {
					continue
				}
				stem := nexusStem(f.name)
				mod := strings.Contains(stem, fmt.Sprintf("-%d-", w.ModID))
				if pass == 0 && mod && w.Date > 0 && strings.HasSuffix(stem, fmt.Sprintf("-%d", w.Date)) || pass == 1 && mod {
					take(w, f, false)
					matched[i] = true
					break
				}
			}
		}
	}
	if len(waits) == 1 && !matched[0] {
		for _, f := range files {
			if _, nexus := NexusFileName(f.name); !used[f.name] && !f.at.Before(waits[0].Since) && !nexus {
				take(waits[0], f, true)
				break
			}
		}
	}
	return out
}

// nexusSignIn are the sign-in pages Nexus's own sign-in sends to (with
// Google, Apple, Discord or Steam), which its window shows too.
var nexusSignIn = map[string]bool{
	"accounts.google.com": true, "appleid.apple.com": true, "discord.com": true, "steamcommunity.com": true,
}

// NexusWindowGo says what Lootnest's window for Nexus does with a page it
// is about to show: "show" it (nexusmods.com, its file servers, the
// sign-in pages), "nxm" for a Mod Manager Download link (not taken: that
// needs Lootnest registered with Nexus), "browser" for any other web page,
// which opens in the user's browser, or "refuse".
func NexusWindowGo(u string) string {
	l := strings.ToLower(strings.TrimSpace(u))
	if strings.HasPrefix(l, "nxm:") {
		return "nxm"
	}
	if l == "about:blank" {
		return "show"
	}
	p, err := url.Parse(u)
	if err != nil || p.Host == "" || p.User != nil {
		return "refuse"
	}
	h := strings.ToLower(p.Hostname())
	switch {
	case p.Scheme == "https" && p.Port() == "" && (NexusPage(u) || h == "nexus-cdn.com" || strings.HasSuffix(h, ".nexus-cdn.com") || nexusSignIn[h]):
		return "show"
	case p.Scheme == "https" || p.Scheme == "http":
		return "browser"
	}
	return "refuse"
}

// NexusWindowArg is what `lootnest.exe --nexus` takes: the game's domain,
// the mod and the file ("<domain>/<mod>/<file>").
func NexusWindowArg(domain string, modID, fileID int) string {
	return fmt.Sprintf("%s/%d/%d", domain, modID, fileID)
}

// NexusWindowURL is the page the window opens for its argument: the
// file's, else the mod's; "" for anything else.
func NexusWindowURL(arg string) string {
	parts := strings.Split(arg, "/")
	if len(parts) != 3 {
		return ""
	}
	mod, err1 := strconv.Atoi(parts[1])
	file, err2 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil {
		return ""
	}
	if file > 0 {
		return NexusFileURL(parts[0], mod, file)
	}
	return NexusModURL(parts[0], mod)
}
