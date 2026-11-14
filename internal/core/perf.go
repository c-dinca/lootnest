package core

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// Will it run here? A game's requirements name a processor and a graphics
// card (often two: "GTX 1060 or RX 580"). Lootnest gives each a rough score,
// and gives this PC's the same way, then says how the game should run:
// above the recommended requirements, at them, at the minimum, or below.
//
// The scores are estimates on one scale per kind, close to well-known
// benchmark results (graphics: 3DMark Time Spy graphics; processors: a
// multi-core CPU mark, in thousands). They're only good for "faster or
// slower, and by how much", which is all a verdict needs. A name Lootnest
// doesn't know gives no score, and then no verdict for that part.
// ---------------------------------------------------------------------------

// gpuScores are graphics cards by model, as normalized by gpuKey.
var gpuScores = map[string]int{
	// NVIDIA GeForce
	"gt 710": 250, "gt 730": 400, "gt 1030": 1100, "gtx 650": 900, "gtx 650 ti": 1200, "gtx 660": 1500, "gtx 660 ti": 1800, "gtx 670": 2000, "gtx 680": 2200,
	"gtx 750": 1300, "gtx 750 ti": 1600, "gtx 760": 1900, "gtx 770": 2300, "gtx 780": 2900, "gtx 780 ti": 3400,
	"gtx 950": 2000, "gtx 960": 2400, "gtx 970": 3400, "gtx 980": 4000, "gtx 980 ti": 5100, "gtx titan x": 5200,
	"gtx 1050": 1800, "gtx 1050 ti": 2300, "gtx 1060": 4100, "gtx 1060 3gb": 3900, "gtx 1060 6gb": 4200, "gtx 1070": 5900, "gtx 1070 ti": 6700, "gtx 1080": 7300, "gtx 1080 ti": 9700,
	"gtx 1630": 1900, "gtx 1650": 3500, "gtx 1650 super": 4800, "gtx 1660": 5300, "gtx 1660 super": 6000, "gtx 1660 ti": 6200,
	"rtx 2060": 7500, "rtx 2060 super": 8800, "rtx 2070": 9100, "rtx 2070 super": 10400, "rtx 2080": 11000, "rtx 2080 super": 11700, "rtx 2080 ti": 14000,
	"rtx 3050": 6200, "rtx 3060": 8800, "rtx 3060 ti": 11500, "rtx 3070": 13500, "rtx 3070 ti": 14500, "rtx 3080": 17500, "rtx 3080 ti": 19500, "rtx 3090": 20000, "rtx 3090 ti": 21500,
	"rtx 4060": 10500, "rtx 4060 ti": 13500, "rtx 4070": 17500, "rtx 4070 super": 20500, "rtx 4070 ti": 22500, "rtx 4070 ti super": 24000, "rtx 4080": 28000, "rtx 4080 super": 28500, "rtx 4090": 36000,
	"rtx 5050": 9500, "rtx 5060": 12500, "rtx 5060 ti": 15000, "rtx 5070": 22000, "rtx 5070 ti": 27500, "rtx 5080": 32000, "rtx 5090": 48000,
	// AMD Radeon
	"hd 7750": 1000, "hd 7770": 1300, "hd 7850": 1900, "hd 7870": 2300, "hd 7950": 2700, "hd 7970": 3100,
	"r7 260x": 1500, "r7 370": 2000, "r9 270": 2100, "r9 270x": 2300, "r9 280": 2800, "r9 280x": 3100, "r9 285": 3000, "r9 290": 3900, "r9 290x": 4200, "r9 380": 3100, "r9 380x": 3400, "r9 390": 4400, "r9 390x": 4700, "r9 fury": 4700, "r9 fury x": 5000,
	"rx 460": 1900, "rx 470": 3700, "rx 480": 4200, "rx 550": 1200, "rx 560": 2100, "rx 570": 4000, "rx 580": 4400, "rx 590": 4800,
	"rx vega 56": 6000, "rx vega 64": 6800, "vega 56": 6000, "vega 64": 6800, "radeon vii": 8500,
	"rx 5500 xt": 5000, "rx 5600 xt": 7400, "rx 5700": 8300, "rx 5700 xt": 9200,
	"rx 6400": 3700, "rx 6500 xt": 4400, "rx 6600": 8100, "rx 6600 xt": 9500, "rx 6650 xt": 9900, "rx 6700": 10800, "rx 6700 xt": 12500, "rx 6750 xt": 13200, "rx 6800": 15500, "rx 6800 xt": 18000, "rx 6900 xt": 20000, "rx 6950 xt": 21500,
	"rx 7600": 10800, "rx 7600 xt": 11000, "rx 7700 xt": 16800, "rx 7800 xt": 19800, "rx 7900 gre": 21500, "rx 7900 xt": 26500, "rx 7900 xtx": 30000,
	"rx 9060 xt": 14500, "rx 9070": 25000, "rx 9070 xt": 29000,
	// Intel
	"arc a310": 3000, "arc a380": 4300, "arc a580": 10000, "arc a750": 12500, "arc a770": 13500, "arc b570": 12500, "arc b580": 14500,
	"hd graphics 4000": 300, "hd graphics 520": 550, "hd graphics 530": 600, "hd graphics 620": 650, "hd graphics 630": 650,
	"uhd graphics 620": 650, "uhd graphics 630": 700, "uhd graphics 730": 700, "uhd graphics 770": 900, "iris xe": 1600, "arc graphics": 3500,
	// Graphics built into AMD processors
	"vega 8": 1100, "vega 11": 1300, "radeon 680m": 2400, "radeon 760m": 2400, "radeon 780m": 2900, "radeon 880m": 3200, "radeon 890m": 3600, "radeon graphics": 1500,
}

var (
	gpuNoise = regexp.MustCompile(`(?i)\(r\)|\(tm\)|®|™|nvidia|geforce|amd|ati|radeon(?: \(tm\))?|intel|series|graphics card|with \d+ ?gb.*|\d+ ?gb(?: vram| of vram| ram)?|vram|desktop|gpu`)
	gpuSpace = regexp.MustCompile(`[\s/-]+`)
	vramRe   = regexp.MustCompile(`(?i)(\d+)\s*gb`)
	arcDisc  = regexp.MustCompile(`arc [ab]\d`)
	intelIGP = regexp.MustCompile(`(u?hd) graphics (\d{3,4})`)
	amdIGP   = regexp.MustCompile(`radeon\s*(?:\(tm\)\s*)?(\d{3}m)`)
	vegaIGP  = regexp.MustCompile(`vega\s*(\d+)`)
	gpuJoin  = regexp.MustCompile(`\b(gtx|rtx|gt|rx|hd|r[79])(\d)`)
	gpuTier  = regexp.MustCompile(`(\d)(ti|xt|xtx|gre|super)\b`)
	bareNum  = regexp.MustCompile(`^\d{3,4}`)
	tmMarks  = regexp.MustCompile(`\(r\)|\(tm\)|®|™`)
	orSplit  = regexp.MustCompile(`(?i)\s+or\s+|\s*/\s*(?:amd|nvidia|intel)|\s*,\s*(?:amd|nvidia|intel)|\s*\|\s*`)
)

// gpuKeys turns one graphics card's name into the keys it may be listed
// under, most specific first: "NVIDIA GeForce GTX 1060 3 GB" gives
// "gtx 1060 3gb" and "gtx 1060".
func gpuKeys(name string) []string {
	l := strings.Join(strings.Fields(strings.ToLower(tmMarks.ReplaceAllString(name, " "))), " ")
	var out []string
	vram := ""
	if m := vramRe.FindStringSubmatch(l); m != nil {
		vram = m[1] + "gb"
	}
	// Integrated graphics keep their words ("uhd graphics 630", "radeon 780m").
	for _, k := range []string{"iris xe", "arc graphics"} {
		if strings.Contains(l, k) && !arcDisc.MatchString(l) {
			out = append(out, k)
		}
	}
	if m := intelIGP.FindStringSubmatch(l); m != nil {
		out = append(out, m[1]+" graphics "+m[2])
	}
	if m := amdIGP.FindStringSubmatch(l); m != nil {
		out = append(out, "radeon "+m[1])
	}
	if m := vegaIGP.FindStringSubmatch(l); m != nil && !strings.Contains(l, "rx vega") && !strings.Contains(l, "vega 56") && !strings.Contains(l, "vega 64") {
		out = append(out, "vega "+m[1])
	}
	if strings.Contains(l, "radeon vii") {
		out = append(out, "radeon vii")
	}
	k := gpuNoise.ReplaceAllString(l, " ")
	k = strings.TrimSpace(gpuSpace.ReplaceAllString(k, " "))
	k = strings.ReplaceAll(k, "laptop", "")
	k = strings.ReplaceAll(k, "max q", "")
	k = strings.TrimSuffix(strings.TrimSpace(k), " graphics")
	k = strings.Join(strings.Fields(k), " ")
	// "RX5700XT", "GTX1060" and "RTX 3060Ti" written together.
	k = gpuJoin.ReplaceAllString(k, "$1 $2")
	k = gpuTier.ReplaceAllString(k, "$1 $2")
	if vram != "" {
		out = append(out, k+" "+vram)
	}
	out = append(out, k)
	if strings.Contains(l, "radeon") && !strings.Contains(k, "rx") && bareNum.MatchString(k) {
		out = append(out, "rx "+k) // "Radeon 580" means the RX 580
	}
	if strings.Contains(l, "radeon graphics") || strings.Contains(l, "radeon(tm) graphics") {
		out = append(out, "radeon graphics")
	}
	return out
}

// GPUScore is a graphics card's rough score, or 0 for a name Lootnest
// doesn't know. A laptop's version of a card scores about a quarter less.
func GPUScore(name string) int {
	l := strings.ToLower(name)
	for _, k := range gpuKeys(name) {
		if s, ok := gpuScores[k]; ok {
			if strings.Contains(l, "laptop") || strings.Contains(l, "mobile") || strings.Contains(l, "max-q") {
				s = s * 3 / 4
			}
			return s
		}
	}
	return 0
}

// ---------------------------------------------------------------- CPUs ---

// Intel Core i3/i5/i7/i9 by generation, and Ryzen 3/5/7/9 by series, in
// thousands of CPU mark points, for the desktop chip of that tier.
var (
	intelCore = map[string]map[int]float64{
		"i3": {2: 2.2, 3: 2.4, 4: 3.2, 5: 3.4, 6: 4.1, 7: 4.4, 8: 6.1, 9: 6.6, 10: 8.7, 11: 9.5, 12: 14, 13: 14.5, 14: 15},
		"i5": {2: 4.1, 3: 4.6, 4: 5.3, 5: 5.5, 6: 5.6, 7: 6.2, 8: 9.2, 9: 9.5, 10: 12, 11: 17, 12: 19.5, 13: 25, 14: 25.5},
		"i7": {1: 3.5, 2: 5.3, 3: 6.3, 4: 7, 5: 7.5, 6: 8.1, 7: 9.6, 8: 13.5, 9: 14.5, 10: 18.5, 11: 24, 12: 34, 13: 46, 14: 53},
		"i9": {9: 18, 10: 23.5, 11: 25, 12: 41, 13: 59, 14: 60},
	}
	ryzen = map[string]map[int]float64{
		"3": {1: 6.3, 2: 6.7, 3: 12, 4: 11, 5: 15, 7: 20, 8: 20},
		"5": {1: 12.4, 2: 13.2, 3: 17.8, 4: 16.4, 5: 21.9, 7: 28.5, 8: 25, 9: 30},
		"7": {1: 15.4, 2: 17.6, 3: 22.6, 4: 20, 5: 28, 7: 36, 8: 31, 9: 37},
		"9": {3: 32.5, 5: 42, 7: 57, 9: 66},
	}
	coreUltra = map[string]float64{"5": 43, "7": 58, "9": 67}
	oldCPUs   = map[string]float64{
		"fx 4300": 3.5, "fx 6300": 4.2, "fx 6350": 4.5, "fx 8300": 5.6, "fx 8320": 5.8, "fx 8350": 6, "fx 9590": 7,
		"phenom ii x4": 2.5, "phenom ii x6": 3.5, "athlon ii x4": 2, "athlon x4": 2.8, "athlon 200ge": 3.4, "athlon 3000g": 3.6,
		"core 2 duo": 1.3, "core 2 quad": 2.9, "q6600": 2.9, "q9550": 3.4, "pentium g4560": 3.3, "pentium gold g5400": 3.6,
		"ryzen 7 5800x3d": 28.5, "ryzen 7 7800x3d": 34.5, "ryzen 7 9800x3d": 40, "ryzen 9 5950x": 46, "ryzen 9 7950x": 63, "ryzen 9 9950x": 66,
	}
	intelRe = regexp.MustCompile(`(?i)\bi([3579])[\s-]*(\d{4,5})([a-z]{0,2})\b`)
	i7OldRe = regexp.MustCompile(`(?i)\bi([3579])[\s-]*(\d{3})([a-z]{0,2})\b`)
	ultraRe = regexp.MustCompile(`(?i)\bultra\s*([579])\s*(\d{3})([a-z]{0,2})\b`)
	ryzenRe = regexp.MustCompile(`(?i)\bryzen\s*([3579])\s*(?:pro\s*)?(\d)(\d{3})([a-z0-9]{0,4})\b`)
)

// CPUScore is a processor's rough score (thousands of CPU mark points), or
// 0 for a name Lootnest doesn't know. Laptop chips (U, H) score less.
func CPUScore(name string) float64 {
	l := strings.ToLower(strings.Join(strings.Fields(tmMarks.ReplaceAllString(name, " ")), " "))
	l = strings.ReplaceAll(l, "-", " ")
	for k, v := range oldCPUs {
		if strings.Contains(l, k) {
			return v
		}
	}
	if m := ultraRe.FindStringSubmatch(l); m != nil {
		s := coreUltra[m[1]]
		if m[2][0] == '1' { // Core Ultra 1xx: laptop chips
			s *= 0.45
		}
		return s * mobile(m[3])
	}
	if m := intelRe.FindStringSubmatch(l); m != nil {
		gen, _ := strconv.Atoi(m[2][:len(m[2])-3])
		if s, ok := intelCore["i"+m[1]][gen]; ok {
			return s * mobile(m[3])
		}
		return 0
	}
	if m := i7OldRe.FindStringSubmatch(l); m != nil { // first generation, "i7-920"
		if s, ok := intelCore["i"+m[1]][1]; ok {
			return s
		}
		return 0
	}
	if m := ryzenRe.FindStringSubmatch(l); m != nil {
		series, _ := strconv.Atoi(m[2])
		if s, ok := ryzen[m[1]][series]; ok {
			suffix := m[4]
			if strings.HasPrefix(suffix, "x") {
				s *= 1.04
			}
			return s * mobile(suffix)
		}
	}
	return 0
}

// mobile scales a laptop chip's score by its suffix.
func mobile(suffix string) float64 {
	switch {
	case strings.HasPrefix(suffix, "u"):
		return 0.6
	case strings.HasPrefix(suffix, "h"), strings.HasPrefix(suffix, "p"):
		return 0.85
	case suffix == "t":
		return 0.8
	}
	return 1
}

// ------------------------------------------------------------- verdict ---

// The ways a game may run here, best first.
const (
	RunsHigh        = "high"        // well above the recommended
	RunsRecommended = "recommended" // at the recommended
	RunsMinimum     = "minimum"     // at the minimum: low settings
	RunsStruggles   = "struggles"   // a little under the minimum
	RunsBelow       = "below"       // well under the minimum
	RunsUnknown     = ""            // nothing to compare
)

var runsRank = map[string]int{RunsHigh: 5, RunsRecommended: 4, RunsMinimum: 3, RunsStruggles: 2, RunsBelow: 1}

// PartVerdict is how one part of this PC compares with the requirements.
type PartVerdict struct {
	Part  string  `json:"part"` // "Graphics", "Processor", "Memory", "Storage"
	Have  float64 `json:"have"` // this PC's score (or bytes)
	Min   float64 `json:"min"`  // the minimum's (0: not known)
	Rec   float64 `json:"rec"`  // the recommended's (0: not known)
	Runs  string  `json:"runs"`
	Known bool    `json:"known"` // both this PC's part and a requirement were recognized
}

// RunVerdict is the whole answer: the weakest part decides.
type RunVerdict struct {
	Runs  string        `json:"runs"`
	Parts []PartVerdict `json:"parts"`
	// Limit is the part that holds the game back, when one does.
	Limit string `json:"limit,omitempty"`
}

func rate(have, min, rec float64) string {
	switch {
	case have <= 0 || (min <= 0 && rec <= 0):
		return RunsUnknown
	case rec > 0 && have >= rec*1.5:
		return RunsHigh
	case rec > 0 && have >= rec*0.95:
		return RunsRecommended
	case min > 0 && have >= min*0.95:
		return RunsMinimum
	case min <= 0 && rec > 0 && have >= rec*0.6:
		return RunsMinimum
	case min > 0 && have >= min*0.75:
		return RunsStruggles
	}
	return RunsBelow
}

// alternativesScore is the weakest of the parts a requirement names ("GTX
// 1060 or RX 580": either is enough), or 0 when none is known.
func alternativesScore(req string, score func(string) float64) float64 {
	parts := orSplit.Split(req, -1)
	var known []float64
	for _, p := range parts {
		if s := score(p); s > 0 {
			known = append(known, s)
		}
	}
	if len(known) == 0 {
		return 0
	}
	sort.Float64s(known)
	return known[0]
}

// WillItRun compares this PC with a game's requirements.
func WillItRun(pc PCSpecs, minimum, recommended []ReqLine) RunVerdict {
	val := func(list []ReqLine, label string) string {
		for _, l := range list {
			if strings.EqualFold(l.Label, label) {
				return l.Value
			}
		}
		return ""
	}
	gpu := func(s string) float64 { return float64(GPUScore(s)) }
	part := func(name string, have float64, score func(string) float64) PartVerdict {
		p := PartVerdict{Part: name, Have: have,
			Min: score(val(minimum, name)), Rec: score(val(recommended, name))}
		p.Runs = rate(p.Have, p.Min, p.Rec)
		p.Known = p.Runs != RunsUnknown
		return p
	}
	size := func(s string) float64 { return float64(ReqBytes(s)) }
	v := RunVerdict{Parts: []PartVerdict{
		part("Graphics", gpu(pc.GPU), func(s string) float64 { return alternativesScore(s, gpu) }),
		part("Processor", CPUScore(pc.CPU), func(s string) float64 { return alternativesScore(s, CPUScore) }),
		part("Memory", float64(pc.RAM), size),
		part("Storage", float64(pc.Free), size),
	}}
	// Space and memory are enough or not: "high" means nothing for them.
	for i := 2; i < 4; i++ {
		if v.Parts[i].Runs == RunsHigh {
			v.Parts[i].Runs = RunsRecommended
		}
	}
	// Graphics and the processor say how well it runs. Memory and space
	// only count when they're short of the recommended.
	worst := 0
	for i, p := range v.Parts {
		r := runsRank[p.Runs]
		if !p.Known || (i >= 2 && r >= runsRank[RunsRecommended]) {
			continue
		}
		if worst == 0 || r < worst {
			worst, v.Runs, v.Limit = r, p.Runs, p.Part
		}
	}
	if worst == 0 && (v.Parts[2].Known || v.Parts[3].Known) {
		worst, v.Runs = runsRank[RunsRecommended], RunsRecommended // only memory and space known, and both enough
	}
	// Graphics and the processor decide how well it runs; if neither is
	// known, memory and space alone can't say "high".
	if !v.Parts[0].Known && !v.Parts[1].Known && v.Runs != RunsUnknown && runsRank[v.Runs] >= runsRank[RunsMinimum] {
		v.Runs, v.Limit = RunsUnknown, ""
	}
	if worst >= runsRank[RunsRecommended] {
		v.Limit = ""
	}
	return v
}

// RunPlan is what a game's page shows under Will it run: a score out of 100
// and the settings Lootnest would start with. An estimate from the score
// tables, never a measurement.
type RunPlan struct {
	Score      int    `json:"score"`      // 0 to 100; 0 when Lootnest can't tell
	Preset     string `json:"preset"`     // "Ultra", "High", "Medium", "Low", "Lowest", or ""
	Resolution string `json:"resolution"` // such as "1080p"
	FPS        string `json:"fps"`        // such as "60"
	Upscaling  bool   `json:"upscaling"`  // DLSS, FSR or XeSS helps
}

// PlanRun turns a verdict into a score and starting settings. The score is
// the weaker of graphics and processor against the recommended (70 at it,
// 100 at half again), else against the minimum (40 at it); memory short of
// the recommended caps it.
func PlanRun(v RunVerdict) RunPlan {
	score := -1.0
	for _, p := range v.Parts {
		if !p.Known || (p.Part != "Graphics" && p.Part != "Processor") {
			continue
		}
		var s float64
		switch {
		case p.Rec > 0:
			s = 70 * p.Have / p.Rec
		case p.Min > 0:
			s = 40 * p.Have / p.Min
		default:
			continue
		}
		if score < 0 || s < score {
			score = s
		}
	}
	if score < 0 {
		return RunPlan{}
	}
	for _, p := range v.Parts {
		if p.Part == "Memory" && p.Known {
			switch p.Runs {
			case RunsMinimum:
				score = math.Min(score, 55)
			case RunsStruggles, RunsBelow:
				score = math.Min(score, 30)
			}
		}
	}
	n := int(math.Round(math.Max(1, math.Min(100, score))))
	switch {
	case n >= 90:
		return RunPlan{n, "Ultra", "1440p", "60+", false}
	case n >= 70:
		return RunPlan{n, "High", "1080p", "60", false}
	case n >= 50:
		return RunPlan{n, "Medium", "1080p", "45 to 60", false}
	case n >= 35:
		return RunPlan{n, "Low", "1080p", "30 to 45", true}
	case n >= 20:
		return RunPlan{n, "Lowest", "720p", "30", true}
	}
	return RunPlan{Score: n}
}
