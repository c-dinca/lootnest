package core

import (
	"net/url"
	"strings"
	"testing"
)

func TestSimilarGPUs(t *testing.T) {
	got := SimilarGPUs("AMD Radeon RX 6700 XT", 2)
	if len(got) != 3 || got[0] != "AMD Radeon RX 6700 XT" {
		t.Fatalf("got %v", got)
	}
	for _, g := range got[1:] {
		if s := GPUScore(g); s < 10600 || s > 14400 {
			t.Fatalf("%s (%d) isn't close to an RX 6700 XT", g, s)
		}
	}
	if gpuSteamName("rtx 4070 ti super") != "NVIDIA GeForce RTX 4070 Ti SUPER" || gpuSteamName("rx 7900 xtx") != "AMD Radeon RX 7900 XTX" || gpuSteamName("arc a770") != "" {
		t.Fatal("steam names")
	}
	if got := SimilarGPUs("Some Unknown GPU", 2); len(got) != 1 {
		t.Fatalf("an unknown card is asked about alone: %v", got)
	}
}

func TestFetchPeers(t *testing.T) {
	var asked []string
	fetch := func(u string) ([]byte, error) {
		q, _ := url.QueryUnescape(u)
		asked = append(asked, q)
		if strings.Contains(q, "RTX 3070") {
			return []byte(`{"response":{"query_summary":{"total_positive":8,"total_reviews":10},"reviews":[
				{"author":{"playtime_at_review":600},"review":"Great story. Runs at 90 fps on high settings with no stutter.","voted_up":true,"hardware":{"cpu_name":"AMD Ryzen 5 5600X 6-Core Processor   ","adapter_description":"NVIDIA GeForce RTX 3070","system_ram":"32768"}},
				{"author":{"playtime_at_review":120},"review":"Crashes every hour.","voted_up":false,"hardware":{"adapter_description":"NVIDIA GeForce RTX 3070","system_ram":16384}},
				{"author":{"playtime_at_review":60},"review":"Fun with friends, 60 fps","voted_up":true,"hardware":{}}]}}`), nil
		}
		return []byte(`{"response":{"query_summary":{"total_positive":3,"total_reviews":4},"reviews":[
			{"author":{"playtime_at_review":300},"review":"Solid 75fps at 1440p.","voted_up":true,"hardware":{"adapter_description":"AMD Radeon RX 6700 XT"}}]}}`), nil
	}
	r, err := FetchPeers(fetch, 1086940, []string{"AMD Radeon RX 6700 XT", "NVIDIA GeForce RTX 3070"})
	if err != nil {
		t.Fatal(err)
	}
	if len(asked) != 2 || !strings.Contains(asked[0], `"hardware_gpu":"AMD Radeon RX 6700 XT"`) || !strings.Contains(asked[0], `"day_range":0`) {
		t.Fatalf("asked %v", asked)
	}
	if r.Count != 14 || r.Positive != 11 || r.Read != 4 || r.Perf != 4 || r.PerfBad != 1 {
		t.Fatalf("report %+v", r)
	}
	if r.FPS != 75 || r.FPSCount != 3 || r.Hours != 5 {
		t.Fatalf("fps %d of %d, hours %v", r.FPS, r.FPSCount, r.Hours)
	}
	q := r.Quotes[1]
	if q.CPU != "AMD Ryzen 5 5600X 6-Core Processor" || q.RAM != 32 || !strings.Contains(q.Text, "90 fps") || strings.Contains(q.Text, "Great story") {
		t.Fatalf("quote %+v", q)
	}
	if len(r.Quotes) != 3 || r.Quotes[2].RAM != 16 {
		t.Fatalf("quotes without the writer's card are left out; a RAM size given as a number: %+v", r.Quotes)
	}
}
