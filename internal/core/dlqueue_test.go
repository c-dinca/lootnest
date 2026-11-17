package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func waitQueue(t *testing.T, what string, ok func() bool) {
	t.Helper()
	for end := time.Now().Add(10 * time.Second); !ok(); time.Sleep(10 * time.Millisecond) {
		if time.Now().After(end) {
			t.Fatalf("never happened: %s", what)
		}
	}
}

func TestDownloadQueue(t *testing.T) {
	big := bytes.Repeat([]byte("0123456789abcdef"), 64<<10) // 1 MB
	sum := sha256.Sum256(big)
	var mu sync.Mutex
	var ranges []string
	gate := make(chan struct{})
	slow := true
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/big.zip":
			mu.Lock()
			ranges = append(ranges, r.Header.Get("Range"))
			hold := slow
			mu.Unlock()
			if hold {
				// Half of it, then wait until the test lets go or the client leaves.
				w.Header().Set("Content-Length", "1048576")
				w.Header().Set("Accept-Ranges", "bytes")
				w.Write(big[:len(big)/2])
				w.(http.Flusher).Flush()
				select {
				case <-gate:
				case <-r.Context().Done():
				}
				return
			}
			http.ServeContent(w, r, "big.zip", time.Time{}, bytes.NewReader(big))
		case "/small.bin":
			w.Write([]byte("small"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	defer close(gate)

	dir := t.TempDir()
	file := filepath.Join(dir, "downloads.json")
	q := NewDownloadQueue(file, srv.Client(), nil)
	t.Cleanup(q.Close)
	dst := filepath.Join(dir, "staging", "big.zip")
	q.Add(Download{ID: "a", Job: "j1", URL: srv.URL + "/big.zip", Dst: dst, Size: int64(len(big)), SHA256: hex.EncodeToString(sum[:])},
		Download{ID: "b", Job: "j1", URL: srv.URL + "/small.bin", Dst: filepath.Join(dir, "staging", "small.bin")},
		Download{ID: "c", Job: "j2", URL: srv.URL + "/gone.bin", Dst: filepath.Join(dir, "staging", "gone.bin")})

	// One at a time: the second waits while the first runs.
	waitQueue(t, "half of the first", func() bool { return q.Job("j1").Done >= int64(len(big)/2) })
	if l := q.List(); l[0].State != "active" || l[1].State != "queued" || l[2].State != "queued" {
		t.Fatalf("while the first runs: %+v", l)
	}
	// Pause keeps what arrived; the other update's download goes on.
	q.Pause("j1")
	waitQueue(t, "the other job's turn", func() bool { return q.List()[2].State == "failed" })
	if p := q.Job("j1"); !p.Paused || p.Ready != 0 || p.Files != 2 {
		t.Fatalf("paused: %+v", p)
	}
	if st, err := os.Stat(dst + ".part"); err != nil || st.Size() != int64(len(big)/2) {
		t.Fatalf("the part after a pause: %v %v", st, err)
	}
	if p := q.Job("j2"); !strings.Contains(p.Failed, "not found") {
		t.Fatalf("a missing file: %+v", p)
	}

	// The queue outlives a restart, and Resume asks for the rest only.
	q.Close()
	q2 := NewDownloadQueue(file, srv.Client(), nil)
	t.Cleanup(q2.Close)
	if l := q2.List(); len(l) != 3 || l[0].State != "paused" || l[0].Done != int64(len(big)/2) {
		t.Fatalf("after a restart: %+v", l)
	}
	mu.Lock()
	slow = false
	mu.Unlock()
	q2.Resume("j1")
	waitQueue(t, "both files", func() bool { return q2.Job("j1").Ready == 2 })
	got, err := os.ReadFile(dst)
	if err != nil || !bytes.Equal(got, big) {
		t.Fatalf("the resumed file differs: %v", err)
	}
	mu.Lock()
	if len(ranges) != 2 || ranges[0] != "" || ranges[1] != "bytes=524288-" {
		t.Errorf("ranges asked: %q", ranges)
	}
	mu.Unlock()
	if isFile(dst + ".part") {
		t.Error("a part was left beside the finished file")
	}

	// A file that isn't what the feed said is refused and leaves nothing.
	bad := filepath.Join(dir, "staging", "bad.zip")
	q2.Add(Download{ID: "d", Job: "j3", URL: srv.URL + "/small.bin", Dst: bad, SHA256: strings.Repeat("0", 64)})
	waitQueue(t, "the checksum refusal", func() bool { return q2.Job("j3").Failed != "" })
	if p := q2.Job("j3"); !strings.Contains(p.Failed, "checksum") || isFile(bad) || isFile(bad+".part") {
		t.Fatalf("a wrong checksum: %+v", p)
	}
	// http only for a file whose checksum is known; anything else never.
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("small")) }))
	defer plain.Close()
	small := sha256.Sum256([]byte("small"))
	q3 := NewDownloadQueue(filepath.Join(dir, "plain.json"), GuardedClient(true), nil)
	t.Cleanup(q3.Close)
	q3.Add(Download{ID: "e", Job: "j4", URL: "ftp://files.example.com/x", Dst: filepath.Join(dir, "staging", "x")},
		Download{ID: "f", Job: "j5", URL: plain.URL + "/s.bin", Dst: filepath.Join(dir, "staging", "plain-unchecked.bin")},
		Download{ID: "g", Job: "j6", URL: plain.URL + "/s.bin", Dst: filepath.Join(dir, "staging", "plain-checked.bin"), SHA256: hex.EncodeToString(small[:])})
	waitQueue(t, "the plain downloads", func() bool { return q3.Job("j4").Failed != "" && q3.Job("j5").Failed != "" && q3.Job("j6").Ready == 1 })
	if a, b := q3.Job("j4"), q3.Job("j5"); !strings.Contains(a.Failed, "https") || !strings.Contains(b.Failed, "https") {
		t.Fatalf("ftp and unchecked http: %+v %+v", a, b)
	}

	// Remove takes an update off the queue; finished files stay.
	q2.Remove("j1")
	q2.Remove("j3")
	if l := q2.List(); len(l) != 1 || l[0].Job != "j2" || !isFile(dst) {
		t.Fatalf("after Remove: %+v", l)
	}
}

// A download that's removed while it runs stops and leaves no part.
func TestDownloadQueueRemoveActive(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100000")
		w.Write(make([]byte, 1000))
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer srv.Close()
	dir := t.TempDir()
	q := NewDownloadQueue(filepath.Join(dir, "downloads.json"), srv.Client(), nil)
	t.Cleanup(q.Close)
	dst := filepath.Join(dir, "f.bin")
	q.Add(Download{ID: "a", Job: "j", URL: srv.URL, Dst: dst})
	waitQueue(t, "its start", func() bool { return q.Job("j").Done >= 1000 })
	q.Remove("j")
	waitQueue(t, "the part to go", func() bool { return !isFile(dst + ".part") })
	if len(q.List()) != 0 {
		t.Fatalf("left: %+v", q.List())
	}
}
