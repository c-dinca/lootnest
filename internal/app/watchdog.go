package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"lootnest/internal/plat"
)

// The window's watchdog (the owner, 24 Nov 2026: Lootnest showed "Not
// Responding" as a game closed). Every few seconds it asks the window's
// thread to answer; when it hasn't for uiHangAfter, every goroutine's stack
// goes to a file in hangs\ and the window thread's to the log, so the next
// report says what the window was waiting on. Nothing changes in how
// Lootnest runs.

const (
	uiPing      = 3 * time.Second
	uiHangAfter = 8 * time.Second
	hangsKept   = 5
)

func hangsDir() string { return filepath.Join(DataDir(), "hangs") }

func (a *App) watchUI() {
	for {
		time.Sleep(uiPing)
		if a.w == nil {
			continue
		}
		done := make(chan struct{})
		asked := time.Now()
		a.ui(func() { close(done) })
		select {
		case <-done:
			continue
		case <-time.After(uiHangAfter):
		}
		// A dispatch can also wait out a menu or a window being dragged,
		// which drop thread messages; only a window Windows calls hung
		// counts (on systems that can't tell, an unanswered dispatch).
		if !plat.WindowHung(a.hwnd) && runtime.GOOS == "windows" {
			continue
		}
		file := a.noteHang()
		<-done
		a.logf("The window answers again after %s (what it was doing: %s)", time.Since(asked).Round(time.Second), file)
	}
}

// noteHang writes every goroutine's stack to a new file in hangs\ and the
// window thread's (the stacks that go through the WebView) to the log.
func (a *App) noteHang() string {
	buf := make([]byte, 4<<20)
	buf = buf[:runtime.Stack(buf, true)]
	dir := hangsDir()
	_ = os.MkdirAll(dir, 0o755)
	file := filepath.Join(dir, "hang-"+time.Now().Format("20060102-150405")+".txt")
	_ = os.WriteFile(file, []byte(fmt.Sprintf("Lootnest %s %s, the window didn't answer for %s\r\n\r\n%s", Version, Stage, uiHangAfter, buf)), 0o644)
	pruneHangs(dir)
	a.logf("ERROR The window stopped answering for %s. Its thread:\r\n%s", uiHangAfter, uiStacks(string(buf)))
	return file
}

// uiStacks picks the goroutines that run on the window's thread out of a
// full dump: those inside the WebView's message loop or a bound call.
func uiStacks(dump string) string {
	var out []string
	for _, g := range strings.Split(dump, "\n\n") {
		if strings.Contains(g, "go-webview2") || strings.Contains(g, "internal/webkit") || strings.Contains(g, "locked to thread") {
			out = append(out, g)
		}
	}
	if len(out) == 0 {
		return "(not found in the dump)"
	}
	s := strings.Join(out, "\n\n")
	if len(s) > 16<<10 {
		s = s[:16<<10] + "\n..."
	}
	return strings.ReplaceAll(s, "\n", "\r\n")
}

// pruneHangs keeps the newest hangsKept files.
func pruneHangs(dir string) {
	es, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var names []string
	for _, e := range es {
		if strings.HasPrefix(e.Name(), "hang-") && strings.HasSuffix(e.Name(), ".txt") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for len(names) > hangsKept {
		_ = os.Remove(filepath.Join(dir, names[0]))
		names = names[1:]
	}
}

// lastHang is the newest hang file written in the past two days, for a
// report, or "".
func lastHang() string {
	es, err := os.ReadDir(hangsDir())
	if err != nil {
		return ""
	}
	best := ""
	for _, e := range es {
		if strings.HasPrefix(e.Name(), "hang-") && e.Name() > best {
			best = e.Name()
		}
	}
	if best == "" {
		return ""
	}
	p := filepath.Join(hangsDir(), best)
	if st, err := os.Stat(p); err != nil || time.Since(st.ModTime()) > 48*time.Hour {
		return ""
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	if len(b) > 64<<10 {
		b = b[:64<<10]
	}
	return string(b)
}
