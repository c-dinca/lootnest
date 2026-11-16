package core

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Lootnest for TV writes one line when a stream ends, with what it counted
// (lootnest_stream_end in the TV app). Lootnest on the PC reads it back so
// the numbers are kept here, and says in plain words where the delay was.

// StreamStats is one finished stream as the TV counted it.
type StreamStats struct {
	Seconds   int     `json:"seconds"`
	Format    string  `json:"format"` // "H265", "H264"
	Width     int     `json:"width"`
	Height    int     `json:"height"`
	FPS       int     `json:"fps"`
	Arrived   float64 `json:"arrived"`   // frames a second that reached the TV
	Shown     float64 `json:"shown"`     // frames a second the TV put on screen
	LostPct   float64 `json:"lostPct"`   // frames lost on the network, percent
	BadSecs   int     `json:"badSecs"`   // seconds with a lost frame
	HostMs    float64 `json:"hostMs"`    // the PC's time for a frame
	DecoderMs float64 `json:"decoderMs"` // the TV's time to hand a frame over
	RoundTrip int     `json:"roundTrip"` // ms, on average
	RoundMax  int     `json:"roundMax"`
	Jitter    int     `json:"jitter"`
}

var streamLine = regexp.MustCompile(`Stream: ended after (\d+) s, (\S+) (\d+)x(\d+) at (\d+)\. Frames a second: ([\d.]+) arrived, ([\d.]+) shown\. The network lost \d+ of \d+ frames \(([\d.]+)%\), in (\d+) of those seconds, \d+ in the worst\. A frame took the PC ([\d.]+) ms, the TV [\d.]+ ms to put together and ([\d.]+) ms to hand to its decoder\. Round trip (\d+) ms on average, (\d+) at most, varying by up to (\d+)`)

// LastStream finds the newest finished stream in the TV app's log: the line
// as written and its numbers. ok is false when the log has none.
func LastStream(log string) (line string, s StreamStats, ok bool) {
	lines := strings.Split(log, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		m := streamLine.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		n := func(i int) int { v, _ := strconv.Atoi(m[i]); return v }
		f := func(i int) float64 { v, _ := strconv.ParseFloat(m[i], 64); return v }
		s = StreamStats{
			Seconds: n(1), Format: m[2], Width: n(3), Height: n(4), FPS: n(5),
			Arrived: f(6), Shown: f(7), LostPct: f(8), BadSecs: n(9),
			HostMs: f(10), DecoderMs: f(11), RoundTrip: n(12), RoundMax: n(13), Jitter: n(14),
		}
		return strings.TrimSpace(lines[i]), s, true
	}
	return "", StreamStats{}, false
}

// Stream verdicts: what a finished stream's numbers point at.
const (
	StreamTooShort = "short"   // under 45 s: not enough to judge
	StreamNetwork  = "network" // frames lost or a slow, uneven network
	StreamTV       = "tv"      // the TV shows fewer frames than arrive
	StreamPC       = "pc"      // the PC is slow to draw and encode
	StreamClean    = "clean"   // none of those: the rest is the TV and the game
)

// StreamVerdict says which part of a stream held it back, and a line in
// plain words with the numbers that say so.
func StreamVerdict(s StreamStats) (kind, says string) {
	switch {
	case s.Seconds < 45:
		return StreamTooShort, "The last stream was too short to judge."
	case s.LostPct >= 1 || (s.Seconds > 0 && s.BadSecs*5 >= s.Seconds):
		return StreamNetwork, fmt.Sprintf("The network lost %s%% of the picture, in %d of %d seconds. A lighter Quality on the TV, its 5 GHz Wi-Fi or a cable will help most.", trimFloat(s.LostPct), s.BadSecs, s.Seconds)
	case s.RoundTrip >= 20 || s.Jitter >= 15:
		return StreamNetwork, fmt.Sprintf("The network takes %d ms there and back, uneven by up to %d ms. The TV's 5 GHz Wi-Fi or a cable will help most.", s.RoundTrip, s.Jitter)
	case s.Arrived > 0 && s.Shown < 0.9*s.Arrived:
		return StreamTV, fmt.Sprintf("The TV showed %s of the %s frames a second that arrived. A smaller Size on the TV will help.", trimFloat(s.Shown), trimFloat(s.Arrived))
	case s.HostMs >= 12:
		return StreamPC, fmt.Sprintf("The PC took %s ms for each frame. Capping the game at %d frames a second leaves the graphics card room to send them.", trimFloat(s.HostMs), s.FPS)
	}
	return StreamClean, fmt.Sprintf("Nothing was lost: %d ms there and back, %s ms on the PC, %s ms into the TV's decoder. The delay that's left is the TV's own picture processing and the game's V-Sync.", s.RoundTrip, trimFloat(s.HostMs), trimFloat(s.DecoderMs))
}

func trimFloat(f float64) string {
	return strings.TrimSuffix(strings.TrimSuffix(strconv.FormatFloat(f, 'f', 1, 64), "0"), ".")
}
