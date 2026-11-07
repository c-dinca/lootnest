package core

import (
	"errors"
	"strings"
	"testing"
)

const elden = `{"1245620":{"success":true,"data":{"name":"ELDEN RING","is_free":false,
"header_image":"https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/1245620/header.jpg?t=1",
"price_overview":{"currency":"EUR","initial":5999,"final":2999,"discount_percent":50},
"pc_requirements":{"minimum":"<strong>Minimum:</strong><br><ul class=\"bb_ul\"><li>Requires a 64-bit processor and operating system<br></li><li><strong>OS:</strong> Windows 10<br></li><li><strong>Processor:</strong> INTEL CORE I5-8400 or AMD RYZEN 3 3300X<br></li><li><strong>Memory:</strong> 12 GB RAM<br></li><li><strong>Graphics:</strong> NVIDIA GEFORCE GTX 1060 3 GB or AMD RADEON RX 580 4 GB<br></li><li><strong>Storage:</strong> 60 GB available space<br></li><li><strong>Additional Notes:</strong> </li></ul>",
"recommended":"<strong>Recommended:</strong><br><ul class=\"bb_ul\"><li><strong>Memory:</strong> 16 GB RAM &amp; more<br></li></ul>"}}}}`

func fakeFetch(pages map[string]string) Fetch {
	return func(u string) ([]byte, error) {
		for k, v := range pages {
			if strings.Contains(u, k) {
				return []byte(v), nil
			}
		}
		return nil, errors.New("offline")
	}
}

func TestSearchStore(t *testing.T) {
	var asked string
	fetch := func(u string) ([]byte, error) {
		asked = u
		return []byte(`{"total":3,"items":[
{"type":"app","name":"ELDEN RING","id":1245620,"price":{"currency":"EUR","initial":5999,"final":2999},"tiny_image":"https://shared.akamai.steamstatic.com/a/capsule_231x87.jpg"},
{"type":"app","name":"Free Thing","id":5,"tiny_image":"https://evil.example/x.jpg"},
{"type":"sub","name":"A bundle","id":6}]}`), nil
	}
	hits, err := SearchStore(fetch, "  elden ring ", "RO")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(asked, "cc=ro&") || !strings.Contains(asked, "term=elden+ring") {
		t.Fatalf("asked %s", asked)
	}
	if len(hits) != 2 || hits[0].Price == nil || hits[0].Price.Discount != 50 || hits[0].Image == "" {
		t.Fatalf("hits = %+v", hits)
	}
	if hits[1].Price != nil || hits[1].Image != "" {
		t.Fatalf("a free game has no price, and pictures come from Steam only: %+v", hits[1])
	}
	if StoreCountry("x1") != "us" || StoreCountry("DE") != "de" {
		t.Fatal("StoreCountry")
	}
	if h, _ := SearchStore(fetch, "   ", "ro"); len(h) != 0 {
		t.Fatal("empty search")
	}
}

func TestFindOffer(t *testing.T) {
	fetch := fakeFetch(map[string]string{
		"appdetails":        elden,
		"games?steamAppID=": `[{"gameID":"236717","steamAppID":"1245620"}]`,
		"games?id=236717":   `{"cheapestPriceEver":{"price":"29.95","date":1751353695},"deals":[{"storeID":"1","dealID":"NUAl%2BxY%3D","price":"59.99","retailPrice":"59.99","savings":"0.000000"},{"storeID":"23","dealID":"0yH%2B27f%3D","price":"52.69","retailPrice":"59.99","savings":"12.168695"},{"storeID":"99","dealID":"zz","price":"1.00","retailPrice":"59.99","savings":"98"}]}`,
	})
	o, err := FindOffer(fetch, 1245620, "ro", map[string]string{"1": "Steam", "23": "GameBillet"})
	if err != nil {
		t.Fatal(err)
	}
	if o.Name != "ELDEN RING" || o.Steam == nil || o.Steam.Final != 2999 || o.Steam.Currency != "EUR" || o.Header == "" {
		t.Fatalf("offer = %+v", o)
	}
	if len(o.Deals) != 2 || o.Deals[0].Store != "GameBillet" || o.Deals[0].Savings != 12 || o.Lowest != 29.95 || o.LowestAt.Year() != 2025 {
		t.Fatalf("deals = %+v, lowest %v %v", o.Deals, o.Lowest, o.LowestAt)
	}
	if o.Deals[0].URL != SharkDealURL+"0yH%2B27f%3D" || !SafeStoreLink(o.Deals[0].URL) || !SafeStoreLink(o.StoreURL) {
		t.Fatalf("deal links: %s %s", o.Deals[0].URL, o.StoreURL)
	}
	if len(o.Minimum) != 6 || o.Minimum[0].Label != "" || o.Minimum[3] != (ReqLine{"Memory", "12 GB RAM"}) {
		t.Fatalf("minimum = %+v", o.Minimum)
	}
	if len(o.Recommended) != 1 || o.Recommended[0].Value != "16 GB RAM & more" {
		t.Fatalf("recommended = %+v", o.Recommended)
	}

	// CheapShark down: the offer still comes, without other stores.
	o, err = FindOffer(fakeFetch(map[string]string{"appdetails": elden}), 1245620, "ro", nil)
	if err != nil || len(o.Deals) != 0 {
		t.Fatalf("without CheapShark: %+v %v", o, err)
	}
	// No requirements: Steam sends [] instead of an object.
	o, err = FindOffer(fakeFetch(map[string]string{"appdetails": `{"9":{"success":true,"data":{"name":"X","is_free":true,"pc_requirements":[]}}}`}), 9, "ro", nil)
	if err != nil || !o.Free || len(o.Minimum) != 0 {
		t.Fatalf("free game: %+v %v", o, err)
	}
	if _, err := FindOffer(fakeFetch(map[string]string{"appdetails": `{"9":{"success":false}}`}), 9, "ro", nil); err == nil {
		t.Fatal("a game the store doesn't have")
	}
}

func TestSafeStoreLink(t *testing.T) {
	for u, want := range map[string]bool{
		"https://store.steampowered.com/app/570/":            true,
		"https://www.cheapshark.com/redirect?dealID=abc":     true,
		"https://www.cheapshark.com/api/1.0/stores":          false,
		"http://store.steampowered.com/app/570/":             false,
		"https://store.steampowered.com.evil.example/app/1/": false,
		"https://example.com/":                               false,
	} {
		if SafeStoreLink(u) != want {
			t.Errorf("SafeStoreLink(%q) = %v", u, !want)
		}
	}
}

func TestReqBytes(t *testing.T) {
	for v, want := range map[string]int64{"12 GB RAM": 12 << 30, "8192 MB RAM": 8 << 30, "1,5 GB": 3 << 29, "60 GB available space": 60 << 30, "1 TB": 1 << 40, "Windows 10": 0} {
		if got := ReqBytes(v); got != want {
			t.Errorf("ReqBytes(%q) = %d, want %d", v, got, want)
		}
	}
}
