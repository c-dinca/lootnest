package core

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestSessionLog(t *testing.T) {
	s := &Store{Dir: t.TempDir()}
	at := time.Date(2026, 11, 6, 20, 0, 0, 0, time.UTC)
	add := func(game string, start time.Time, d time.Duration) {
		t.Helper()
		if err := s.AppendSession(SessionRecord{Game: game, Profile: "p1", Start: start, End: start.Add(d), Source: "profile", Build: "b1"}); err != nil {
			t.Fatal(err)
		}
	}
	add("game:a", at, 2*time.Hour)
	add("game:b", at.Add(3*time.Hour), 30*time.Minute)
	add("game:a", at.Add(5*time.Hour), 59*time.Second) // too short
	add("game:a", at.Add(24*time.Hour), 40*time.Minute)

	all, bad, err := s.ReadSessions("", time.Time{})
	if err != nil || bad != 0 || len(all) != 3 {
		t.Fatalf("all = %+v, bad %d, %v", all, bad, err)
	}
	if all[0].Game != "game:a" || all[1].Game != "game:b" || all[2].Seconds != 7200 || all[2].Build != "b1" || all[2].Profile != "p1" {
		t.Fatalf("not newest first, or a field lost: %+v", all)
	}
	a, _, _ := s.ReadSessions("game:a", time.Time{})
	if len(a) != 2 || !a[0].Start.Equal(at.Add(24*time.Hour)) {
		t.Fatalf("game:a = %+v", a)
	}
	if recent, _, _ := s.ReadSessions("game:a", at.Add(time.Hour+59*time.Minute)); len(recent) != 2 {
		t.Fatalf("since: %+v", recent)
	}
	if recent, _, _ := s.ReadSessions("game:a", at.Add(3*time.Hour)); len(recent) != 1 {
		t.Fatalf("since: %+v", recent)
	}
	b, _ := os.ReadFile(s.SessionLogPath())
	if !strings.Contains(string(b), `"start":"2026-11-06T20:00:00Z"`) || strings.Count(string(b), "\n") != 3 {
		t.Fatalf("file = %s", b)
	}
}

func TestSessionLogCorrupt(t *testing.T) {
	s := &Store{Dir: t.TempDir()}
	at := time.Date(2026, 11, 6, 20, 0, 0, 0, time.UTC)
	rec := func(start time.Time) SessionRecord {
		return SessionRecord{Game: "game:a", Start: start, End: start.Add(time.Hour), Source: "library"}
	}
	_ = s.AppendSession(rec(at))
	f, err := os.OpenFile(s.SessionLogPath(), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(`{"game":"game:a","start":"not a time"}` + "\n" + strings.Repeat("x", 100<<10) + "\n\n")
	_ = f.Close()
	_ = s.AppendSession(rec(at.Add(2 * time.Hour)))
	got, bad, err := s.ReadSessions("game:a", time.Time{})
	if err != nil || bad != 2 || len(got) != 2 || !got[0].Start.Equal(at.Add(2*time.Hour)) {
		t.Fatalf("got %+v, bad %d, %v", got, bad, err)
	}
	b, _ := os.ReadFile(s.SessionLogPath())
	if !strings.Contains(string(b), "not a time") {
		t.Fatal("the corrupt line was removed")
	}
}

func TestSessionLogNoTrailingNewline(t *testing.T) {
	s := &Store{Dir: t.TempDir()}
	at := time.Date(2026, 11, 6, 20, 0, 0, 0, time.UTC)
	line := `{"game":"game:a","start":"2026-11-06T20:00:00Z","end":"2026-11-06T21:00:00Z","seconds":3600,"source":"profile"}`
	if err := os.WriteFile(s.SessionLogPath(), []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, bad, _ := s.ReadSessions("game:a", time.Time{}); len(got) != 1 || bad != 0 {
		t.Fatalf("last line without a newline: %+v, bad %d", got, bad)
	}
	if err := s.AppendSession(SessionRecord{Game: "game:a", Start: at.Add(2 * time.Hour), End: at.Add(3 * time.Hour), Source: "profile"}); err != nil {
		t.Fatal(err)
	}
	got, bad, _ := s.ReadSessions("game:a", time.Time{})
	if len(got) != 2 || bad != 0 || got[1].Seconds != 3600 {
		t.Fatalf("after an append: %+v, bad %d", got, bad)
	}
}

func TestSessionLogMissing(t *testing.T) {
	s := &Store{Dir: t.TempDir()}
	if got, bad, err := s.ReadSessions("", time.Time{}); got != nil || bad != 0 || err != nil {
		t.Fatalf("no file: %+v, %d, %v", got, bad, err)
	}
}
