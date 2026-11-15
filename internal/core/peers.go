package core

import (
	"encoding/json"
	"errors"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// How a game runs for players with a PC like this one, from Steam's own
// reviews: since 2025 a review can carry its writer's hardware, and
// IUserReviewsService filters by the graphics card's exact name (no key).
// Lootnest asks for this PC's card and the two closest in its score table,
// only when the user asked for it; only the cards' names are sent.
//
// It's what people wrote, not a measurement: the share who recommend the
// game, how many mention performance, the frame rates they name when
// enough of them do, and a few of their words with their hardware.
// ---------------------------------------------------------------------------

// PeerQuote is one review that talks about how the game runs.
type PeerQuote struct {
	Up    bool    `json:"up"`
	Text  string  `json:"text"`
	GPU   string  `json:"gpu"`
	CPU   string  `json:"cpu,omitempty"`
	RAM   int     `json:"ram,omitempty"` // GB
	Hours float64 `json:"hours"`
}

// PeerReport is what reviews from players with the same or a close card say.
type PeerReport struct {
	GPUs     []string    `json:"gpus"`               // the cards asked about, this PC's first
	Count    int         `json:"count"`              // reviews with one of them
	Positive int         `json:"positive"`           // of those, recommending
	Hours    float64     `json:"hours"`              // median hours played when written
	Perf     int         `json:"perf"`               // read reviews that talk about performance
	PerfBad  int         `json:"perfBad"`            // of those, not recommending
	Read     int         `json:"read"`               // reviews read for the above
	FPS      int         `json:"fps,omitempty"`      // median of the frame rates named, when 3 or more
	FPSCount int         `json:"fpsCount,omitempty"` // how many named one
	Quotes   []PeerQuote `json:"quotes"`
}

var (
	fpsMention  = regexp.MustCompile(`(?i)\b(\d{2,3})\s*(?:-|to)?\s*(?:\d{2,3}\s*)?(?:fps|frames)\b`)
	perfMention = regexp.MustCompile(`(?i)\b(?:fps|frame ?rate|frames|stutter\w*|lag\w*|crash\w*|perform\w*|optimi[sz]\w*|smooth\w*|runs?\s+(?:great|well|fine|bad|badly|poorly)|settings|ultra|high settings|medium|low settings|1080p|1440p|4k|dlss|fsr|vram)\b`)
)

// gpuSteamName is a card in gpuScores as Steam writes it ("rtx 3070" ->
// "NVIDIA GeForce RTX 3070"), or "" for one it names differently.
func gpuSteamName(key string) string {
	up := strings.ToUpper(key)
	switch {
	case strings.HasPrefix(key, "rtx ") || strings.HasPrefix(key, "gtx ") || strings.HasPrefix(key, "gt "):
		return "NVIDIA GeForce " + strings.Replace(strings.Replace(up, " SUPER", " SUPER", 1), " TI", " Ti", 1)
	case strings.HasPrefix(key, "rx ") && !strings.HasPrefix(key, "rx vega"):
		return "AMD Radeon " + up
	}
	return ""
}

// SimilarGPUs is this PC's card as Steam writes it, then up to n cards
// closest to it in score (within 15%), as Steam writes them.
func SimilarGPUs(name string, n int) []string {
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		return nil
	}
	out := []string{name}
	mine := float64(GPUScore(name))
	if mine <= 0 {
		return out
	}
	type c struct {
		name string
		d    float64
	}
	var near []c
	for k, s := range gpuScores {
		sn := gpuSteamName(k)
		if sn == "" || strings.EqualFold(sn, name) || strings.Contains(k, "gb") {
			continue
		}
		if d := math.Abs(float64(s)-mine) / mine; d <= 0.15 {
			near = append(near, c{sn, d})
		}
	}
	sort.Slice(near, func(i, j int) bool {
		return near[i].d < near[j].d || (near[i].d == near[j].d && near[i].name < near[j].name)
	})
	for i := 0; i < len(near) && i < n; i++ {
		out = append(out, near[i].name)
	}
	return out
}

// FetchPeers asks Steam's reviews for each card in gpus and sums them up.
func FetchPeers(fetch Fetch, appID int, gpus []string) (PeerReport, error) {
	rep := PeerReport{GPUs: gpus, Quotes: []PeerQuote{}}
	if appID <= 0 || len(gpus) == 0 {
		return rep, errors.New("unknown game")
	}
	var hours []float64
	var fps []int
	asked := 0
	for _, g := range gpus {
		b, err := fetch(reviewsQuery(map[string]any{"appid": appID, "filter": 0, "languages": []string{"english"}, "review_type": 0,
			"purchase_type": 1, "num_per_page": 100, "day_range": 0, "hardware_gpu": g}))
		if err != nil {
			continue
		}
		asked++
		var res struct {
			Response *struct {
				Summary struct {
					Positive int `json:"total_positive"`
					Total    int `json:"total_reviews"`
				} `json:"query_summary"`
				Reviews []struct {
					Author struct {
						AtReview float64 `json:"playtime_at_review"`
					} `json:"author"`
					Review   string `json:"review"`
					Up       bool   `json:"voted_up"`
					Hardware struct {
						CPU string          `json:"cpu_name"`
						GPU string          `json:"adapter_description"`
						RAM json.RawMessage `json:"system_ram"` // MB, a string or a number
					} `json:"hardware"`
				} `json:"reviews"`
			} `json:"response"`
		}
		if json.Unmarshal(b, &res) != nil || res.Response == nil {
			continue
		}
		rep.Count += res.Response.Summary.Total
		rep.Positive += res.Response.Summary.Positive
		for _, r := range res.Response.Reviews {
			rep.Read++
			hours = append(hours, r.Author.AtReview/60)
			text, _ := ReviewText(r.Review)
			for _, m := range fpsMention.FindAllStringSubmatch(text, 3) {
				if n, _ := strconv.Atoi(m[1]); n >= 15 && n <= 500 {
					fps = append(fps, n)
				}
			}
			if !perfMention.MatchString(text) {
				continue
			}
			rep.Perf++
			if !r.Up {
				rep.PerfBad++
			}
			if len(rep.Quotes) < 6 && strings.TrimSpace(r.Hardware.GPU) != "" {
				mb, _ := strconv.ParseFloat(strings.Trim(string(r.Hardware.RAM), `"`), 64)
				rep.Quotes = append(rep.Quotes, PeerQuote{Up: r.Up, Text: perfSnippet(text), GPU: strings.TrimSpace(r.Hardware.GPU),
					CPU: strings.Join(strings.Fields(r.Hardware.CPU), " "), RAM: int(math.Round(mb / 1024)), Hours: math.Round(hours[len(hours)-1]*10) / 10})
			}
		}
	}
	if asked == 0 {
		return rep, errors.New("Steam didn't answer")
	}
	if len(hours) > 0 {
		sort.Float64s(hours)
		rep.Hours = math.Round(hours[len(hours)/2]*10) / 10
	}
	if len(fps) >= 3 {
		sort.Ints(fps)
		rep.FPS, rep.FPSCount = fps[len(fps)/2], len(fps)
	}
	return rep, nil
}

// perfSnippet is the sentence around the first mention of performance,
// at most about 220 characters.
func perfSnippet(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	loc := perfMention.FindStringIndex(text)
	if loc == nil {
		loc = []int{0, 0}
	}
	start := strings.LastIndexAny(text[:loc[0]], ".!?") + 1
	end := len(text)
	if i := strings.IndexAny(text[loc[1]:], ".!?"); i >= 0 {
		end = loc[1] + i + 1
	}
	s := strings.TrimSpace(text[start:end])
	if r := []rune(s); len(r) > 220 {
		s = strings.TrimSpace(string(r[:219])) + "…"
	}
	return s
}
