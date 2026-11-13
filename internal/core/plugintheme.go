package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// A plugin's theme (theme.json): new values for Lootnest's colour tokens,
// never raw CSS. Only ThemeTokens can be set, only to hex, rgb() or oklch()
// colours, and every pair of text and background the app uses must pass
// WCAG AA with them (themePairs), or the theme is refused with the pair
// that fails. A theme is chosen per profile.
// ---------------------------------------------------------------------------

// ThemeTokens are the tokens a theme may set, with Lootnest's own values
// (system.css; a test keeps them the same).
var ThemeTokens = map[string]string{
	"--bg-well":      "#0f1015",
	"--bg-ground":    "#111217",
	"--bg-surface":   "#181a21",
	"--bg-raised":    "#1f222b",
	"--bg-active":    "#282b35",
	"--fg-1":         "#DFE5EB",
	"--fg-2":         "#a6aeb9",
	"--fg-3":         "#8c94a0",
	"--accent":       "#EF233C",
	"--accent-hover": "#f43b52",
	"--fg-on-accent": "#1a0a0d",
	"--edge-1":       "rgb(255 255 255 / 0.07)",
	"--edge-2":       "rgb(255 255 255 / 0.12)",
	"--edge-3":       "rgb(255 255 255 / 0.22)",
}

// themeGlass are the layers a theme can't change, over the ground; text on
// them is checked too, so a light theme can't put dark text on the dark
// glass (the sidebar, the play dock, menus).
var themeGlass = [][2]string{
	{"the sidebar", "rgb(17 18 23 / 0.46)"},
	{"the play dock", "rgb(17 18 23 / 0.62)"},
	{"menus and dialogs", "rgb(24 26 33 / 0.94)"},
}

// themeFixed are text colours a theme can't change (errors, warnings,
// done), checked on the theme's backgrounds.
var themeFixed = map[string]string{
	"--danger":  "#ff8a7a",
	"--warning": "#f2c14e",
	"--success": "#7be08f",
}

var tokenWords = map[string]string{
	"--bg-well": "wells and inputs", "--bg-ground": "the window", "--bg-surface": "panels", "--bg-raised": "raised controls",
	"--bg-active": "selected rows", "--fg-1": "text", "--fg-2": "secondary text", "--fg-3": "dates and counts",
	"--accent": "the accent", "--accent-hover": "the accent on hover", "--fg-on-accent": "text on the accent",
	"--danger": "error text", "--warning": "warnings", "--success": "done marks",
}

// themePairs: text, background, and the contrast it needs (4.5:1 for text,
// 3:1 for the accent, which is never text, against the window and panels).
var themePairs = func() [][3]string {
	var out [][3]string
	for _, fg := range []string{"--fg-1", "--fg-2", "--fg-3"} {
		for _, bg := range []string{"--bg-well", "--bg-ground", "--bg-surface", "--bg-raised"} {
			out = append(out, [3]string{fg, bg, "4.5"})
		}
		for _, g := range themeGlass {
			out = append(out, [3]string{fg, g[0], "4.5"})
		}
	}
	for _, fg := range []string{"--danger", "--warning", "--success"} {
		for _, bg := range []string{"--bg-well", "--bg-ground", "--bg-surface", "--bg-raised"} {
			out = append(out, [3]string{fg, bg, "4.5"})
		}
	}
	return append(out, [3]string{"--fg-1", "--bg-active", "4.5"}, [3]string{"--fg-2", "--bg-active", "4.5"},
		[3]string{"--fg-on-accent", "--accent", "4.5"}, [3]string{"--fg-on-accent", "--accent-hover", "4.5"},
		[3]string{"--accent", "--bg-ground", "3"}, [3]string{"--accent", "--bg-surface", "3"})
}()

type Theme struct {
	Name   string            `json:"name"`
	Tokens map[string]string `json:"tokens"`
}

// PluginTheme reads and checks a plugin's theme.json.
func PluginTheme(dir string) (*Theme, error) {
	b, err := os.ReadFile(filepath.Join(dir, "theme.json"))
	if err != nil {
		return nil, errors.New("it provides a theme but has no theme.json")
	}
	var t Theme
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, errors.New("theme.json can't be read: " + shortErr(err))
	}
	t.Name = strings.Join(strings.Fields(t.Name), " ")
	if t.Name == "" || len([]rune(t.Name)) > 60 {
		return nil, errors.New("its theme's name must be 1 to 60 characters")
	}
	if err := CheckTheme(t.Tokens); err != nil {
		return nil, err
	}
	return &t, nil
}

// CheckTheme refuses a theme that sets anything but a colour token, or
// whose colours fail a contrast pair.
func CheckTheme(tokens map[string]string) error {
	if len(tokens) == 0 {
		return errors.New("its theme sets no colours")
	}
	cols := map[string]rgba{}
	for _, m := range []map[string]string{ThemeTokens, themeFixed} {
		for k, v := range m {
			cols[k], _ = parseColor(v)
		}
	}
	for k, v := range tokens {
		if _, ok := ThemeTokens[k]; !ok {
			return fmt.Errorf("its theme sets %s, which a theme can't change; only colour tokens such as --bg-ground, --fg-1 and --accent", k)
		}
		c, err := parseColor(v)
		if err != nil {
			return fmt.Errorf("its theme's %s isn't a hex, rgb() or oklch() colour", k)
		}
		if c.a < 1 && !strings.HasPrefix(k, "--edge-") {
			return fmt.Errorf("its theme's %s must be a solid colour", k)
		}
		cols[k] = c
	}
	for _, g := range themeGlass {
		c, _ := parseColor(g[1])
		cols[g[0]] = c.over(cols["--bg-ground"])
	}
	for _, p := range themePairs {
		need, _ := strconv.ParseFloat(p[2], 64)
		if r := contrast(cols[p[0]], cols[p[1]]); r < need {
			on := tokenWords[p[1]]
			if on == "" {
				on = p[1]
			}
			return fmt.Errorf("its theme's %s (%s) on %s is %.1f:1; it needs %s:1", tokenWords[p[0]], p[0], on, math.Floor(r*10)/10, p[2])
		}
	}
	return nil
}

// rgba is a colour in sRGB, 0 to 1.
type rgba struct{ r, g, b, a float64 }

var (
	hexColor   = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)
	rgbColor   = regexp.MustCompile(`^rgba?\(\s*([0-9.]+)[\s,]+([0-9.]+)[\s,]+([0-9.]+)\s*(?:[/,]\s*([0-9.]+%?)\s*)?\)$`)
	oklchColor = regexp.MustCompile(`^oklch\(\s*([0-9.]+%?)\s+([0-9.]+)\s+([0-9.]+)(?:deg)?\s*(?:/\s*([0-9.]+%?)\s*)?\)$`)
)

func parseColor(s string) (rgba, error) {
	s = strings.TrimSpace(s)
	if m := hexColor.FindStringSubmatch(s); m != nil {
		h := m[1]
		if len(h) == 3 {
			h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
		}
		n, _ := strconv.ParseUint(h, 16, 32)
		return rgba{float64(n>>16&255) / 255, float64(n>>8&255) / 255, float64(n&255) / 255, 1}, nil
	}
	if m := rgbColor.FindStringSubmatch(strings.ToLower(s)); m != nil {
		ch := func(v string) (float64, bool) {
			f, err := strconv.ParseFloat(v, 64)
			return f / 255, err == nil && f <= 255
		}
		r, ok1 := ch(m[1])
		g, ok2 := ch(m[2])
		b, ok3 := ch(m[3])
		a, ok4 := 1.0, true
		if m[4] != "" {
			v, div := m[4], 1.0
			if strings.HasSuffix(v, "%") {
				v, div = strings.TrimSuffix(v, "%"), 100
			}
			f, err := strconv.ParseFloat(v, 64)
			a, ok4 = f/div, err == nil && f/div <= 1
		}
		if !ok1 || !ok2 || !ok3 || !ok4 {
			return rgba{}, errors.New("not a colour")
		}
		return rgba{r, g, b, a}, nil
	}
	m := oklchColor.FindStringSubmatch(strings.ToLower(s))
	if m == nil {
		return rgba{}, errors.New("not a colour")
	}
	num := func(v string, pct float64) (float64, bool) {
		div := 1.0
		if strings.HasSuffix(v, "%") {
			v, div = strings.TrimSuffix(v, "%"), pct
		}
		f, err := strconv.ParseFloat(v, 64)
		return f / div, err == nil
	}
	l, ok1 := num(m[1], 100)
	c, ok2 := num(m[2], 1)
	h, ok3 := num(m[3], 1)
	a, ok4 := 1.0, true
	if m[4] != "" {
		a, ok4 = num(m[4], 100)
	}
	if !ok1 || !ok2 || !ok3 || !ok4 || l > 1 || c > 0.5 || h > 360 || a > 1 {
		return rgba{}, errors.New("not a colour")
	}
	return oklchRGB(l, c, h, a), nil
}

// oklchRGB converts OKLCH to sRGB, clipped to the sRGB gamut.
func oklchRGB(l, c, h, alpha float64) rgba {
	a, b := c*math.Cos(h*math.Pi/180), c*math.Sin(h*math.Pi/180)
	l_ := math.Pow(l+0.3963377774*a+0.2158037573*b, 3)
	m_ := math.Pow(l-0.1055613458*a-0.0638541728*b, 3)
	s_ := math.Pow(l-0.0894841775*a-1.2914855480*b, 3)
	lin := [3]float64{
		4.0767416621*l_ - 3.3077115913*m_ + 0.2309699292*s_,
		-1.2684380046*l_ + 2.6097574011*m_ - 0.3413193965*s_,
		-0.0041960863*l_ - 0.7034186147*m_ + 1.7076147010*s_,
	}
	var out [3]float64
	for i, v := range lin {
		v = min(max(v, 0), 1)
		if v <= 0.0031308 {
			out[i] = 12.92 * v
		} else {
			out[i] = 1.055*math.Pow(v, 1/2.4) - 0.055
		}
	}
	return rgba{out[0], out[1], out[2], alpha}
}

// over is c drawn over an opaque bottom, as the browser blends it.
func (c rgba) over(bottom rgba) rgba {
	mix := func(x, y float64) float64 { return c.a*x + (1-c.a)*y }
	return rgba{mix(c.r, bottom.r), mix(c.g, bottom.g), mix(c.b, bottom.b), 1}
}

func (c rgba) luminance() float64 {
	lin := func(v float64) float64 {
		if v <= 0.04045 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(c.r) + 0.7152*lin(c.g) + 0.0722*lin(c.b)
}

// contrast is WCAG 2's contrast ratio of two opaque colours.
func contrast(a, b rgba) float64 {
	x, y := a.luminance(), b.luminance()
	return (max(x, y) + 0.05) / (min(x, y) + 0.05)
}
