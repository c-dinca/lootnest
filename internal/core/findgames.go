package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Find games: look up any game, installed or not, with its price in the
// official stores and its system requirements. Opt-in (Settings, Online,
// "Find games"); only the search words or the game's Steam id are sent.
//
//   - the search and the requirements come from the Steam store
//     (storesearch, appdetails), with prices in the user's country;
//   - the prices in other stores, and the lowest price ever, come from
//     CheapShark, which lists authorized stores only (no key resellers),
//     in US dollars.
//
// Lootnest only opens a store's page in the browser. It never buys or
// downloads anything.
// ---------------------------------------------------------------------------

const (
	findSearchURL  = "https://store.steampowered.com/api/storesearch/?l=english&cc=%s&term=%s"
	findDetailsURL = "https://store.steampowered.com/api/appdetails?appids=%d&cc=%s&l=english&filters=basic,price_overview,pc_requirements"
	sharkGameURL   = "https://www.cheapshark.com/api/1.0/games?steamAppID=%d"
	sharkDealsURL  = "https://www.cheapshark.com/api/1.0/games?id=%s"
	sharkStoresURL = "https://www.cheapshark.com/api/1.0/stores"
	// SharkDealURL opens a deal on its store, through CheapShark's link.
	SharkDealURL = "https://www.cheapshark.com/redirect?dealID="
)

// StorePrice is a price in a currency's smallest unit (cents).
type StorePrice struct {
	Currency string `json:"currency"`
	Initial  int    `json:"initial"`
	Final    int    `json:"final"`
	Discount int    `json:"discount,omitempty"` // percent
}

// StoreHit is one search result.
type StoreHit struct {
	AppID int         `json:"appId"`
	Name  string      `json:"name"`
	Image string      `json:"image"` // Steam's 231x87 capsule
	Price *StorePrice `json:"price,omitempty"`
	// Owned is the Library key or game id of the copy on this PC, set by
	// the app.
	Owned string `json:"owned,omitempty"`
}

// Deal is the game's price in one store, in US dollars.
type Deal struct {
	Store   string  `json:"store"`
	Price   float64 `json:"price"`
	Retail  float64 `json:"retail"`
	Savings int     `json:"savings"` // percent off
	URL     string  `json:"url"`
}

// ReqLine is one line of a game's system requirements, such as
// {"Memory", "16 GB RAM"}. A line without a label has only Value.
type ReqLine struct {
	Label string `json:"label,omitempty"`
	Value string `json:"value"`
}

// GameOffer is what Find games shows for one game.
type GameOffer struct {
	AppID       int         `json:"appId"`
	Name        string      `json:"name"`
	Header      string      `json:"header"`
	Free        bool        `json:"free,omitempty"`
	Steam       *StorePrice `json:"steam,omitempty"` // in the user's currency
	Deals       []Deal      `json:"deals"`
	Lowest      float64     `json:"lowest,omitempty"` // US dollars, ever
	LowestAt    time.Time   `json:"lowestAt,omitempty"`
	Minimum     []ReqLine   `json:"minimum"`
	Recommended []ReqLine   `json:"recommended"`
	StoreURL    string      `json:"storeUrl"`
}

var countryCode = regexp.MustCompile(`^[a-z]{2}$`)

// StoreCountry is a two-letter country code for Steam's prices, "us" when
// the one given isn't one.
func StoreCountry(cc string) string {
	cc = strings.ToLower(strings.TrimSpace(cc))
	if !countryCode.MatchString(cc) {
		return "us"
	}
	return cc
}

// SearchStore looks games up by name on the Steam store.
func SearchStore(fetch Fetch, term, cc string) ([]StoreHit, error) {
	term = strings.TrimSpace(term)
	if term == "" {
		return []StoreHit{}, nil
	}
	if len(term) > 100 {
		term = term[:100]
	}
	b, err := fetch(fmt.Sprintf(findSearchURL, StoreCountry(cc), url.QueryEscape(term)))
	if err != nil {
		return nil, err
	}
	var res struct {
		Items []struct {
			Type  string `json:"type"`
			ID    int    `json:"id"`
			Name  string `json:"name"`
			Image string `json:"tiny_image"`
			Price *struct {
				Currency string `json:"currency"`
				Initial  int    `json:"initial"`
				Final    int    `json:"final"`
			} `json:"price"`
		} `json:"items"`
	}
	if err := json.Unmarshal(b, &res); err != nil {
		return nil, fmt.Errorf("unexpected reply from the Steam store: %w", err)
	}
	out := []StoreHit{}
	for _, it := range res.Items {
		if it.Type != "app" || it.ID <= 0 || it.Name == "" {
			continue
		}
		h := StoreHit{AppID: it.ID, Name: it.Name}
		if SafeNewsImage(it.Image) {
			h.Image = it.Image
		}
		if p := it.Price; p != nil && p.Currency != "" {
			h.Price = &StorePrice{Currency: p.Currency, Initial: p.Initial, Final: p.Final, Discount: percentOff(p.Initial, p.Final)}
		}
		out = append(out, h)
	}
	return out, nil
}

func percentOff(initial, final int) int {
	if initial <= 0 || final >= initial {
		return 0
	}
	return (initial - final) * 100 / initial
}

// FindOffer gathers a game's prices and requirements. stores names
// CheapShark's store ids (see SharkStores); a failure there leaves the
// deals empty rather than failing the whole offer.
func FindOffer(fetch Fetch, appID int, cc string, stores map[string]string) (*GameOffer, error) {
	if appID <= 0 {
		return nil, errors.New("unknown game")
	}
	b, err := fetch(fmt.Sprintf(findDetailsURL, appID, StoreCountry(cc)))
	if err != nil {
		return nil, err
	}
	var res map[string]struct {
		Success bool `json:"success"`
		Data    struct {
			Name   string `json:"name"`
			Free   bool   `json:"is_free"`
			Header string `json:"header_image"`
			Price  *struct {
				Currency string `json:"currency"`
				Initial  int    `json:"initial"`
				Final    int    `json:"final"`
				Discount int    `json:"discount_percent"`
			} `json:"price_overview"`
			// An object, or [] when the store has none.
			Req json.RawMessage `json:"pc_requirements"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &res); err != nil {
		return nil, fmt.Errorf("unexpected reply from the Steam store: %w", err)
	}
	r, ok := res[strconv.Itoa(appID)]
	if !ok || !r.Success {
		return nil, errors.New("The Steam store doesn't have this game.")
	}
	d := r.Data
	o := &GameOffer{AppID: appID, Name: d.Name, Free: d.Free, Deals: []Deal{}, Minimum: []ReqLine{}, Recommended: []ReqLine{},
		StoreURL: fmt.Sprintf("https://store.steampowered.com/app/%d/", appID)}
	if SafeNewsImage(d.Header) {
		o.Header = d.Header
	}
	if p := d.Price; p != nil && p.Currency != "" {
		o.Steam = &StorePrice{Currency: p.Currency, Initial: p.Initial, Final: p.Final, Discount: p.Discount}
	}
	var req struct {
		Minimum     string `json:"minimum"`
		Recommended string `json:"recommended"`
	}
	if len(d.Req) > 0 && d.Req[0] == '{' && json.Unmarshal(d.Req, &req) == nil {
		o.Minimum = ParseRequirements(req.Minimum)
		o.Recommended = ParseRequirements(req.Recommended)
	}
	if !d.Free {
		_ = sharkDeals(fetch, appID, stores, o)
	}
	return o, nil
}

func sharkDeals(fetch Fetch, appID int, stores map[string]string, o *GameOffer) error {
	b, err := fetch(fmt.Sprintf(sharkGameURL, appID))
	if err != nil {
		return err
	}
	var games []struct {
		ID string `json:"gameID"`
	}
	if err := json.Unmarshal(b, &games); err != nil || len(games) == 0 || !safeSharkID(games[0].ID) {
		return err
	}
	if b, err = fetch(fmt.Sprintf(sharkDealsURL, games[0].ID)); err != nil {
		return err
	}
	var res struct {
		Lowest struct {
			Price string `json:"price"`
			Date  int64  `json:"date"`
		} `json:"cheapestPriceEver"`
		Deals []struct {
			Store   string `json:"storeID"`
			ID      string `json:"dealID"`
			Price   string `json:"price"`
			Retail  string `json:"retailPrice"`
			Savings string `json:"savings"`
		} `json:"deals"`
	}
	if err := json.Unmarshal(b, &res); err != nil {
		return err
	}
	if p, err := strconv.ParseFloat(res.Lowest.Price, 64); err == nil && p >= 0 {
		o.Lowest = p
		if res.Lowest.Date > 0 {
			o.LowestAt = time.Unix(res.Lowest.Date, 0).UTC()
		}
	}
	for _, d := range res.Deals {
		name := stores[d.Store]
		price, err1 := strconv.ParseFloat(d.Price, 64)
		retail, err2 := strconv.ParseFloat(d.Retail, 64)
		if name == "" || err1 != nil || err2 != nil || d.ID == "" {
			continue // a store CheapShark no longer lists as active
		}
		sav, _ := strconv.ParseFloat(d.Savings, 64)
		o.Deals = append(o.Deals, Deal{Store: name, Price: price, Retail: retail, Savings: int(sav + 0.5), URL: SharkDealURL + url.QueryEscape(unescapeDeal(d.ID))})
	}
	sort.SliceStable(o.Deals, func(i, j int) bool { return o.Deals[i].Price < o.Deals[j].Price })
	return nil
}

// unescapeDeal: CheapShark's deal ids come URL-escaped already.
func unescapeDeal(id string) string {
	if u, err := url.QueryUnescape(id); err == nil {
		return u
	}
	return id
}

var sharkIDRe = regexp.MustCompile(`^[0-9]{1,12}$`)

func safeSharkID(id string) bool { return sharkIDRe.MatchString(id) }

// SharkStores reads CheapShark's active stores, by id.
func SharkStores(fetch Fetch) (map[string]string, error) {
	b, err := fetch(sharkStoresURL)
	if err != nil {
		return nil, err
	}
	var list []struct {
		ID     string `json:"storeID"`
		Name   string `json:"storeName"`
		Active int    `json:"isActive"`
	}
	if err := json.Unmarshal(b, &list); err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, s := range list {
		if s.Active == 1 && s.ID != "" && s.Name != "" {
			out[s.ID] = s.Name
		}
	}
	return out, nil
}

// SafeStoreLink accepts the links Find games opens: the Steam store and
// CheapShark's deal links, and the itch.io page where API keys are made.
func SafeStoreLink(u string) bool {
	p, err := url.Parse(u)
	if err != nil || p.Scheme != "https" {
		return false
	}
	switch strings.ToLower(p.Hostname()) {
	case "store.steampowered.com":
		return strings.HasPrefix(p.Path, "/app/")
	case "www.cheapshark.com":
		return p.Path == "/redirect" && p.Query().Get("dealID") != ""
	case "itch.io": // where the user makes the API key Lootnest asks for
		return p.Path == "/user/settings/api-keys" && p.RawQuery == "" && p.Fragment == ""
	}
	return false
}

var (
	reqItem = regexp.MustCompile(`(?is)<li[^>]*>(.*?)</li>`)
	reqTag  = regexp.MustCompile(`(?s)<[^>]*>`)
	reqBold = regexp.MustCompile(`(?is)^\s*<strong>(.*?)</strong>(.*)$`)
)

// ParseRequirements turns the Steam store's requirements HTML into lines.
func ParseRequirements(s string) []ReqLine {
	out := []ReqLine{}
	for _, m := range reqItem.FindAllStringSubmatch(s, 40) {
		item := m[1]
		var l ReqLine
		if b := reqBold.FindStringSubmatch(item); b != nil {
			l.Label = strings.TrimSuffix(strings.TrimSpace(reqText(b[1])), ":")
			l.Value = reqText(b[2])
		} else {
			l.Value = reqText(item)
		}
		if l.Value == "" {
			continue // "Additional Notes:" with nothing after it
		}
		if len(l.Value) > 300 {
			l.Value = l.Value[:300]
		}
		out = append(out, l)
	}
	return out
}

func reqText(s string) string {
	s = reqTag.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(html.UnescapeString(s)), " ")
}

var reqSize = regexp.MustCompile(`(?i)(\d+(?:[.,]\d+)?)\s*(GB|MB|TB)`)

// ReqBytes reads a size such as "16 GB RAM" or "60 GB available space"
// from a requirement, or 0 when it names none.
func ReqBytes(v string) int64 {
	m := reqSize.FindStringSubmatch(v)
	if m == nil {
		return 0
	}
	n, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", "."), 64)
	if err != nil {
		return 0
	}
	switch strings.ToUpper(m[2]) {
	case "TB":
		n *= 1 << 40
	case "GB":
		n *= 1 << 30
	default:
		n *= 1 << 20
	}
	return int64(n)
}

// PCSpecs is this PC, for comparing with a game's requirements.
type PCSpecs struct {
	CPU     string `json:"cpu"`
	GPU     string `json:"gpu"`
	VRAM    int64  `json:"vram"`    // bytes, 0 when Windows doesn't say
	RAM     int64  `json:"ram"`     // bytes
	Free    int64  `json:"free"`    // bytes free on the drive with the most room
	FreeOn  string `json:"freeOn"`  // that drive, such as "D:"
	Windows string `json:"windows"` // such as "Windows 11"
}

// ---------------------------------------------------------------- store ---

const (
	featuredURL  = "https://store.steampowered.com/api/featuredcategories?l=english&cc=%s"
	sharkBestURL = "https://www.cheapshark.com/api/1.0/deals?sortBy=Deal%20Rating&metacritic=75&onSale=1&pageSize=40"
)

// StoreItem is a game in one of the Store page's rows.
type StoreItem struct {
	AppID int         `json:"appId"`
	Name  string      `json:"name"`
	Image string      `json:"image"` // wide art, 616x353 when Steam has it
	Price *StorePrice `json:"price,omitempty"`
	// For a deal in another store: its name, price in US dollars and link.
	Store string  `json:"store,omitempty"`
	USD   float64 `json:"usd,omitempty"`
	URL   string  `json:"url,omitempty"`
	Score int     `json:"score,omitempty"` // Metacritic
	Owned string  `json:"owned,omitempty"`
}

// StoreFront is what the Store page shows before a search.
type StoreFront struct {
	Daily    *StoreItem  `json:"daily,omitempty"`
	Specials []StoreItem `json:"specials"`
	Top      []StoreItem `json:"top"`
	Deals    []StoreItem `json:"deals"`
}

// FetchStoreFront reads Steam's featured games (the daily deal, specials
// and top sellers) and CheapShark's best deals on well-reviewed games.
// Either half failing leaves its rows empty; both failing is an error.
func FetchStoreFront(fetch Fetch, cc string, stores map[string]string) (*StoreFront, error) {
	f := &StoreFront{Specials: []StoreItem{}, Top: []StoreItem{}, Deals: []StoreItem{}}
	err1 := steamFeatured(fetch, cc, f)
	err2 := sharkBest(fetch, stores, f)
	if err1 != nil && err2 != nil {
		return nil, err1
	}
	return f, nil
}

type steamFeaturedItem struct {
	ID       int    `json:"id"`
	Type     int    `json:"type"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
	Original *int   `json:"original_price"`
	Final    int    `json:"final_price"`
	Discount int    `json:"discount_percent"`
	Large    string `json:"large_capsule_image"`
	Header   string `json:"header_image"`
}

func (it steamFeaturedItem) item() (StoreItem, bool) {
	if it.Type != 0 || it.ID <= 0 || it.Name == "" {
		return StoreItem{}, false
	}
	s := StoreItem{AppID: it.ID, Name: it.Name}
	for _, img := range []string{it.Large, it.Header} {
		if SafeNewsImage(img) {
			s.Image = img
			break
		}
	}
	if it.Currency != "" && it.Original != nil {
		s.Price = &StorePrice{Currency: it.Currency, Initial: *it.Original, Final: it.Final, Discount: it.Discount}
	}
	return s, true
}

func steamFeatured(fetch Fetch, cc string, f *StoreFront) error {
	b, err := fetch(fmt.Sprintf(featuredURL, StoreCountry(cc)))
	if err != nil {
		return err
	}
	var res map[string]json.RawMessage
	if err := json.Unmarshal(b, &res); err != nil {
		return fmt.Errorf("unexpected reply from the Steam store: %w", err)
	}
	type row struct {
		ID    string              `json:"id"`
		Items []steamFeaturedItem `json:"items"`
	}
	read := func(key string) []StoreItem {
		var r row
		out := []StoreItem{}
		if json.Unmarshal(res[key], &r) != nil {
			return out
		}
		for _, it := range r.Items {
			if s, ok := it.item(); ok {
				out = append(out, s)
			}
		}
		return out
	}
	f.Specials, f.Top = read("specials"), read("top_sellers")
	// The daily deal sits in a numbered row with id "cat_dailydeal".
	for k, v := range res {
		if _, err := strconv.Atoi(k); err != nil {
			continue
		}
		var r row
		if json.Unmarshal(v, &r) == nil && r.ID == "cat_dailydeal" && len(r.Items) > 0 {
			if s, ok := r.Items[0].item(); ok {
				f.Daily = &s
			}
		}
	}
	return nil
}

func sharkBest(fetch Fetch, stores map[string]string, f *StoreFront) error {
	b, err := fetch(sharkBestURL)
	if err != nil {
		return err
	}
	var list []struct {
		Title   string `json:"title"`
		Deal    string `json:"dealID"`
		Store   string `json:"storeID"`
		GameID  string `json:"gameID"`
		Sale    string `json:"salePrice"`
		Normal  string `json:"normalPrice"`
		Savings string `json:"savings"`
		Score   string `json:"metacriticScore"`
		SteamID string `json:"steamAppID"`
	}
	if err := json.Unmarshal(b, &list); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, d := range list {
		id, _ := strconv.Atoi(d.SteamID)
		sale, err := strconv.ParseFloat(d.Sale, 64)
		name := stores[d.Store]
		if id <= 0 || err != nil || name == "" || d.Deal == "" || seen[d.GameID] {
			continue // the list is best first: keep each game's best deal only
		}
		seen[d.GameID] = true
		sav, _ := strconv.ParseFloat(d.Savings, 64)
		score, _ := strconv.Atoi(d.Score)
		f.Deals = append(f.Deals, StoreItem{AppID: id, Name: d.Title, Store: name, USD: sale, Score: score,
			Price: &StorePrice{Currency: "USD", Discount: int(sav + 0.5)},
			Image: fmt.Sprintf("https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/%d/header.jpg", id),
			URL:   SharkDealURL + url.QueryEscape(unescapeDeal(d.Deal))})
		if len(f.Deals) == 12 {
			break
		}
	}
	return nil
}
