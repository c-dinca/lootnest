//go:build linux

package webkit

import (
	"os"
	"runtime"
	"testing"
	"time"
)

func init() { runtime.LockOSThread() }

// windowResult is what the window check saw; it runs in TestMain, on the
// main thread GTK needs (tests run on other goroutines).
var windowResult struct {
	ran   bool
	err   error
	got   map[string]string
	calls int

	shown, hidden bool // Visible after Show, then after Hide
}

func TestMain(m *testing.M) {
	if os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != "" {
		checkWindow()
	}
	os.Exit(m.Run())
}

// checkWindow opens a hidden window whose page calls bound functions, tries
// to navigate away and puts a frame in it that tries to reach Go.
func checkWindow() {
	dir, err := os.MkdirTemp("", "webkit-test")
	if err != nil {
		windowResult.err = err
		return
	}
	defer os.RemoveAll(dir)
	windowResult.ran = true
	got := map[string]string{}
	windowResult.got = got
	var v *WebView
	v, err = New(Options{Title: "test", Width: 400, Height: 300, Hidden: true, DataDir: dir,
		AllowMessage: func(uri string) bool { windowResult.calls++; return uri == BaseURI }})
	if err != nil {
		windowResult.err = err
		return
	}
	_ = v.Bind("ps_echo", func(s string, n int) (string, error) { return s + "!", nil })
	_ = v.Bind("ps_note", func(k, val string) {
		got[k] = val
		if len(got) == 3 {
			// The window's state, kept for any goroutine: shown, then hidden.
			v.Show()
			go func() {
				time.Sleep(300 * time.Millisecond)
				windowResult.shown = v.Visible()
				v.Dispatch(v.Hide)
				time.Sleep(300 * time.Millisecond)
				windowResult.hidden = !v.Visible()
				v.Terminate()
			}()
		}
	})
	_ = v.Bind("ps_frame", func() { got["frame"] = "reached Go" })
	v.SetHtml(`<!doctype html><body><script>
	  ps_echo("hi", 2).then(function (r) { ps_note("echo", r); });
	  ps_echo("wrong").catch(function (e) { ps_note("mismatch", String(e)); });
	  var f = document.createElement("iframe");
	  f.srcdoc = "<script>window.webkit.messageHandlers.lootnest.postMessage(JSON.stringify({id: 99, method: 'ps_frame', params: []}))<\/script>";
	  document.body.appendChild(f);
	  setTimeout(function () {
	    setTimeout(function () { ps_note("where", location.href); }, 1500);
	    try { location.href = "https://example.com/"; } catch (e) {}
	  }, 500);
	</script>`)
	go func() {
		time.Sleep(15 * time.Second)
		v.Terminate()
	}()
	v.Run()
}

// TestWindow needs a display (CI runs it under xvfb-run).
func TestWindow(t *testing.T) {
	r := windowResult
	if !r.ran {
		t.Skip("no display")
	}
	if r.err != nil {
		t.Fatal(r.err)
	}
	if r.got["echo"] != "hi!" {
		t.Errorf("a bound function's answer: %q", r.got["echo"])
	}
	if r.got["mismatch"] != "function arguments mismatch" {
		t.Errorf("a wrong call is refused: %q", r.got["mismatch"])
	}
	if r.got["where"] != BaseURI {
		t.Errorf("the page can't navigate away: %q", r.got["where"])
	}
	if r.got["frame"] != "" {
		t.Error("a frame reached Go")
	}
	if r.calls == 0 {
		t.Error("AllowMessage is asked")
	}
	if !r.shown || !r.hidden {
		t.Errorf("Visible: %v once shown, %v once hidden", r.shown, !r.hidden)
	}
}
