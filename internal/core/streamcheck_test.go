package core

import (
	"strings"
	"testing"
)

const streamLog = `[10:00:01] Screen: ready
[10:02:01] Stream: ended after 30 s, H265 1920x1080 at 60. Frames a second: 50.0 arrived, 50.0 shown. The network lost 300 of 1800 frames (16.67%), in 30 of those seconds, 10 in the worst. A frame took the PC 4.8 ms, the TV 1.1 ms to put together and 6.0 ms to hand to its decoder. Round trip 12 ms on average, 12 at most, varying by up to 4
[10:20:01] Stream: ended after 600 s, H265 1920x1080 at 60. Frames a second: 59.2 arrived, 59.2 shown. The network lost 480 of 36000 frames (1.33%), in 120 of those seconds, 4 in the worst. A frame took the PC 4.1 ms, the TV 0.9 ms to put together and 5.1 ms to hand to its decoder. Round trip 12 ms on average, 14 at most, varying by up to 4
[10:20:02] Picture: fewer bits`

func TestLastStream(t *testing.T) {
	line, s, ok := LastStream(streamLog)
	if !ok || !strings.HasPrefix(line, "[10:20:01] Stream: ended after 600 s") {
		t.Fatalf("line = %q, ok = %v", line, ok)
	}
	want := StreamStats{Seconds: 600, Format: "H265", Width: 1920, Height: 1080, FPS: 60, Arrived: 59.2, Shown: 59.2, LostPct: 1.33, BadSecs: 120, HostMs: 4.1, DecoderMs: 5.1, RoundTrip: 12, RoundMax: 14, Jitter: 4}
	if s != want {
		t.Fatalf("stats = %+v", s)
	}
	if _, _, ok := LastStream("Screen: ready\nStream: ended before a second of picture"); ok {
		t.Fatal("a log with no finished stream gave one")
	}
}

func TestStreamVerdict(t *testing.T) {
	base := StreamStats{Seconds: 600, FPS: 60, Arrived: 60, Shown: 60, HostMs: 4, DecoderMs: 5, RoundTrip: 6, Jitter: 2}
	with := func(f func(*StreamStats)) StreamStats { s := base; f(&s); return s }
	for _, c := range []struct {
		s    StreamStats
		kind string
		has  string
	}{
		{with(func(s *StreamStats) { s.Seconds = 20 }), StreamTooShort, "too short"},
		{with(func(s *StreamStats) { s.LostPct = 1.33; s.BadSecs = 40 }), StreamNetwork, "lost 1.3% of the picture, in 40 of 600 seconds"},
		{with(func(s *StreamStats) { s.BadSecs = 150 }), StreamNetwork, "in 150 of 600"},
		{with(func(s *StreamStats) { s.RoundTrip = 35; s.Jitter = 20 }), StreamNetwork, "35 ms there and back, uneven by up to 20 ms"},
		{with(func(s *StreamStats) { s.Shown = 40 }), StreamTV, "showed 40 of the 60 frames"},
		{with(func(s *StreamStats) { s.HostMs = 18.5 }), StreamPC, "18.5 ms for each frame"},
		{base, StreamClean, "6 ms there and back, 4 ms on the PC, 5 ms into the TV's decoder"},
	} {
		kind, says := StreamVerdict(c.s)
		if kind != c.kind || !strings.Contains(says, c.has) {
			t.Errorf("%+v: %s %q, want %s with %q", c.s, kind, says, c.kind, c.has)
		}
	}
}
