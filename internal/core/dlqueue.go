package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// The download queue: files a script plugin's update asked for, fetched one
// at a time into a staging folder, never over a game. Each can be paused
// and resumed (its .part stays, and the server is asked for the rest) or
// removed. The list is kept in downloads.json, so a queue outlives a
// restart: what was running then waits again.
// ---------------------------------------------------------------------------

// Download is one file in the queue.
type Download struct {
	ID     string `json:"id"`
	Job    string `json:"job"` // the update it belongs to
	URL    string `json:"url"`
	Dst    string `json:"dst"`
	Size   int64  `json:"size,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
	// State: queued, active, paused, done or failed.
	State string `json:"state"`
	Done  int64  `json:"done"`
	Total int64  `json:"total,omitempty"`
	Speed int64  `json:"speed,omitempty"` // bytes a second, while active
	Error string `json:"error,omitempty"`
}

// JobProgress sums one update's downloads.
type JobProgress struct {
	Files, Ready int
	Done, Total  int64
	Speed        int64
	ETA          int64 // seconds, 0 when it can't be told
	Paused       bool
	Failed       string
}

// DownloadQueue runs downloads one at a time.
type DownloadQueue struct {
	mu      sync.Mutex
	items   []*Download
	file    string
	client  *http.Client
	changed func()
	cancel  context.CancelFunc
	active  string
	wake    chan struct{}
	saved   time.Time
	closed  bool
	stopped chan struct{}
}

// GuardedClient is an http client that follows at most five redirects, only
// to http and https, and never reaches this PC or its network (allowLocal
// is for tests).
func GuardedClient(allowLocal bool) *http.Client {
	return &http.Client{
		Transport: &http.Transport{Proxy: nil, DialContext: guardedDial(allowLocal), TLSHandshakeTimeout: 15 * time.Second,
			ResponseHeaderTimeout: 60 * time.Second, MaxIdleConns: 4, IdleConnTimeout: 30 * time.Second},
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects")
			}
			if r.URL.Scheme != "https" && r.URL.Scheme != "http" {
				return errors.New("a redirect left http")
			}
			return nil
		},
	}
}

// NewDownloadQueue reads the queue kept in file and starts its worker.
// changed is called, not too often, when anything in it moves.
func NewDownloadQueue(file string, client *http.Client, changed func()) *DownloadQueue {
	q := &DownloadQueue{file: file, client: client, changed: changed, wake: make(chan struct{}, 1), stopped: make(chan struct{})}
	if b, err := os.ReadFile(file); err == nil {
		_ = json.Unmarshal(b, &q.items)
	}
	for _, d := range q.items {
		if d.State == "active" {
			d.State = "queued" // Lootnest closed while it ran
		}
		d.Speed = 0
	}
	go q.run()
	q.kick()
	return q
}

func (q *DownloadQueue) kick() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}
	select {
	case q.wake <- struct{}{}:
	default:
	}
}

// Close stops the queue: what's running is cut short and waits, with its
// .part, for the next start.
func (q *DownloadQueue) Close() {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return
	}
	q.closed = true
	if q.cancel != nil {
		q.cancel()
	}
	close(q.wake)
	q.mu.Unlock()
	<-q.stopped
}

// save writes the list. q.mu is held.
func (q *DownloadQueue) save() {
	q.saved = time.Now()
	_ = os.MkdirAll(filepath.Dir(q.file), 0o755)
	_ = writeJSONAtomic(q.file, q.items)
}

func (q *DownloadQueue) tell() {
	if q.changed != nil {
		q.changed()
	}
}

// Add queues downloads. Each needs an id, an address and where it goes.
func (q *DownloadQueue) Add(list ...Download) {
	q.mu.Lock()
	for _, d := range list {
		d := d
		d.State, d.Done, d.Speed, d.Error = "queued", 0, 0, ""
		q.items = append(q.items, &d)
	}
	q.save()
	q.mu.Unlock()
	q.kick()
	q.tell()
}

// List is a copy of the queue, in order.
func (q *DownloadQueue) List() []Download {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]Download, len(q.items))
	for i, d := range q.items {
		out[i] = *d
	}
	return out
}

// Job sums the downloads of one update.
func (q *DownloadQueue) Job(job string) JobProgress {
	q.mu.Lock()
	defer q.mu.Unlock()
	var p JobProgress
	paused := 0
	for _, d := range q.items {
		if d.Job != job {
			continue
		}
		p.Files++
		p.Done += d.Done
		p.Total += max(d.Total, d.Size, d.Done)
		p.Speed += d.Speed
		switch d.State {
		case "done":
			p.Ready++
		case "paused":
			paused++
		case "failed":
			if p.Failed == "" {
				p.Failed = d.Error
			}
		}
	}
	p.Paused = paused > 0 && paused+p.Ready == p.Files
	if p.Speed > 0 && p.Total > p.Done {
		p.ETA = (p.Total - p.Done) / p.Speed
	}
	return p
}

// Pause stops an update's downloads, keeping what arrived.
func (q *DownloadQueue) Pause(job string) {
	q.mu.Lock()
	for _, d := range q.items {
		if d.Job == job && (d.State == "queued" || d.State == "active") {
			if d.ID == q.active && q.cancel != nil {
				q.cancel()
			}
			d.State, d.Speed = "paused", 0
		}
	}
	q.save()
	q.mu.Unlock()
	q.tell()
}

// Resume queues an update's paused and failed downloads again; each goes on
// from where it stopped when its server allows.
func (q *DownloadQueue) Resume(job string) {
	q.mu.Lock()
	for _, d := range q.items {
		if d.Job == job && (d.State == "paused" || d.State == "failed") {
			d.State, d.Error = "queued", ""
		}
	}
	q.save()
	q.mu.Unlock()
	q.kick()
	q.tell()
}

// Remove takes an update's downloads off the queue and deletes what
// arrived of the ones that didn't finish. Finished files stay for whoever
// asked for them.
func (q *DownloadQueue) Remove(job string) {
	q.mu.Lock()
	var keep []*Download
	for _, d := range q.items {
		if d.Job != job {
			keep = append(keep, d)
			continue
		}
		if d.ID == q.active && q.cancel != nil {
			q.cancel()
		}
		if d.State != "done" && d.ID != q.active {
			_ = os.Remove(d.Dst + ".part") // the running one's goes when it stops
		}
	}
	q.items = keep
	q.save()
	q.mu.Unlock()
	q.tell()
}

func (q *DownloadQueue) next() *Download {
	for _, d := range q.items {
		if d.State == "queued" {
			return d
		}
	}
	return nil
}

func (q *DownloadQueue) has(d *Download) bool {
	for _, x := range q.items {
		if x == d {
			return true
		}
	}
	return false
}

func (q *DownloadQueue) run() {
	defer close(q.stopped)
	for range q.wake {
		for {
			q.mu.Lock()
			d := q.next()
			if d == nil || q.closed {
				q.mu.Unlock()
				break
			}
			ctx, cancel := context.WithCancel(context.Background())
			d.State, q.active, q.cancel = "active", d.ID, cancel
			id, u, dst, size, sum := d.ID, d.URL, d.Dst, d.Size, d.SHA256
			q.save()
			q.mu.Unlock()
			q.tell()

			// Speed over the last few seconds.
			var mark time.Time
			var markDone int64
			var lastTell time.Time
			dl := downloader{client: q.client, store: "the source", keep: true, plain: true, status: func(code int, status string) error {
				if code == http.StatusOK || code == http.StatusPartialContent {
					return nil
				}
				return fmt.Errorf("the source's site answered %s", strings.ToLower(status))
			}}
			err := dl.get(ctx, u, dst, size, sum, func(done, total int64) {
				now := time.Now()
				q.mu.Lock()
				if d.ID == id {
					d.Done, d.Total = done, total
					if mark.IsZero() {
						mark, markDone = now, done
					} else if el := now.Sub(mark); el >= 2*time.Second {
						d.Speed = int64(float64(done-markDone) / el.Seconds())
						mark, markDone = now, done
					}
					if now.Sub(q.saved) > 10*time.Second {
						q.save()
					}
				}
				q.mu.Unlock()
				if now.Sub(lastTell) > 500*time.Millisecond {
					lastTell = now
					q.tell()
				}
			})
			cancel()
			q.mu.Lock()
			q.active, q.cancel = "", nil
			d.Speed = 0
			switch {
			case !q.has(d):
				_ = os.Remove(dst + ".part") // removed while it ran
			case q.closed:
				d.State = "queued"
			case err == nil:
				d.State = "done"
				if st, e := os.Stat(dst); e == nil {
					d.Done, d.Total = st.Size(), st.Size()
				}
			case d.State == "paused":
				// Pause cancelled it; its .part stays.
			default:
				d.State, d.Error = "failed", sentenceOf(err)
			}
			q.save()
			q.mu.Unlock()
			q.tell()
		}
	}
}

// sentenceOf makes an error read as a sentence.
func sentenceOf(err error) string {
	s := strings.TrimSpace(err.Error())
	if s == "" {
		return s
	}
	s = strings.ToUpper(s[:1]) + s[1:]
	if !strings.HasSuffix(s, ".") {
		s += "."
	}
	return s
}
