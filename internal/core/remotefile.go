package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ---------------------------------------------------------------------------
// Reading an archive where it lies, on its server, without saving it.
//
// RemoteFile reads any part of a file over https by asking for that range
// (Range requests), so an archive whose list of files is at its end (.zip,
// .7z) can be read for its list, and then for only the files that changed.
// RemoteVolumes hands an archive's parts to a reader that goes through them
// in order (.rar), each as one stream, picking up where it broke off when a
// connection drops.
// ---------------------------------------------------------------------------

const (
	remoteBlocks = 4 // blocks kept
	remoteTries  = 4
)

// remoteBlock is how many bytes are asked for at a time (smaller in tests).
var remoteBlock int64 = 4 << 20

// ErrNoRanges: the server sends only whole files.
var ErrNoRanges = errors.New("the server doesn't send parts of a file")

// RemoteFile is a file on a server, read in parts.
type RemoteFile struct {
	ctx     context.Context
	client  *http.Client
	url     string
	size    int64
	mu      sync.Mutex
	blocks  []remoteBlockData // most recent last
	fetched atomic.Int64
}

type remoteBlockData struct {
	off  int64
	data []byte
}

// OpenRemote asks the server for the file's first byte, to learn its size
// and whether it sends parts. ErrNoRanges when it doesn't.
func OpenRemote(ctx context.Context, client *http.Client, u string) (*RemoteFile, error) {
	pu, err := url.Parse(u)
	if err != nil || (pu.Scheme != "https" && pu.Scheme != "http") {
		return nil, errors.New("that address isn't http or https")
	}
	r := &RemoteFile{ctx: ctx, client: client, url: u}
	resp, err := r.get(0, 0)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusPartialContent {
		return nil, ErrNoRanges
	}
	_, _, total, ok := contentRange(resp.Header.Get("Content-Range"))
	if !ok || total <= 0 {
		return nil, ErrNoRanges
	}
	r.size = total
	return r, nil
}

// Size is the file's size.
func (r *RemoteFile) Size() int64 { return r.size }

// Fetched is how many bytes have come from the server so far.
func (r *RemoteFile) Fetched() int64 { return r.fetched.Load() }

func (r *RemoteFile) get(from, to int64) (*http.Response, error) {
	req, err := http.NewRequestWithContext(r.ctx, "GET", r.url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Range", "bytes="+strconv.FormatInt(from, 10)+"-"+strconv.FormatInt(to, 10))
	req.Header.Set("User-Agent", "Lootnest")
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, netErr("the download's site", err)
	}
	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("the download's site answered %s", strings.ToLower(resp.Status))
	}
	return resp, nil
}

// contentRange reads "bytes 0-99/1000".
func contentRange(s string) (from, to, total int64, ok bool) {
	s, found := strings.CutPrefix(s, "bytes ")
	if !found {
		return 0, 0, 0, false
	}
	span, size, found := strings.Cut(s, "/")
	a, b, found2 := strings.Cut(span, "-")
	if !found || !found2 {
		return 0, 0, 0, false
	}
	var e1, e2, e3 error
	from, e1 = strconv.ParseInt(a, 10, 64)
	to, e2 = strconv.ParseInt(b, 10, 64)
	total, e3 = strconv.ParseInt(size, 10, 64)
	return from, to, total, e1 == nil && e2 == nil && e3 == nil && from <= to
}

// block gives the block that starts at off, from the ones kept or the
// server. r.mu is held.
func (r *RemoteFile) block(off int64) ([]byte, error) {
	for i, b := range r.blocks {
		if b.off == off {
			r.blocks = append(append(r.blocks[:i:i], r.blocks[i+1:]...), b)
			return b.data, nil
		}
	}
	end := min(off+remoteBlock, r.size) - 1
	var lastErr error
	for try := 0; try < remoteTries; try++ {
		if try > 0 {
			select {
			case <-r.ctx.Done():
				return nil, r.ctx.Err()
			case <-time.After(time.Duration(try) * time.Second):
			}
		}
		data, err := r.fetch(off, end)
		if err == nil {
			r.blocks = append(r.blocks, remoteBlockData{off, data})
			if len(r.blocks) > remoteBlocks {
				r.blocks = r.blocks[1:]
			}
			return data, nil
		}
		if r.ctx.Err() != nil {
			return nil, r.ctx.Err()
		}
		lastErr = err
	}
	return nil, lastErr
}

func (r *RemoteFile) fetch(from, to int64) ([]byte, error) {
	resp, err := r.get(from, to)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusPartialContent {
		return nil, ErrNoRanges
	}
	if a, b, _, ok := contentRange(resp.Header.Get("Content-Range")); !ok || a != from || b != to {
		return nil, errors.New("the download's site sent another part than the one asked for")
	}
	data := make([]byte, to-from+1)
	n, err := io.ReadFull(resp.Body, data)
	r.fetched.Add(int64(n))
	if err != nil {
		return nil, netErr("the download's site", err)
	}
	return data, nil
}

// ReadAt reads len(p) bytes at off, from the server as needed.
func (r *RemoteFile) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, errors.New("negative offset")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for n < len(p) {
		at := off + int64(n)
		if at >= r.size {
			return n, io.EOF
		}
		start := at - at%remoteBlock
		b, err := r.block(start)
		if err != nil {
			return n, err
		}
		n += copy(p[n:], b[at-start:])
	}
	return n, nil
}

// RemoteVolumes is an fs.FS of an archive's parts on a server, by file
// name, each opened as one stream from its start.
type RemoteVolumes struct {
	Ctx    context.Context
	Client *http.Client
	URLs   map[string]string // file name -> address
	read   atomic.Int64
}

// Read is how many bytes have come from the server so far.
func (v *RemoteVolumes) Read() int64 { return v.read.Load() }

// Open opens one part by its name.
func (v *RemoteVolumes) Open(name string) (fs.File, error) {
	u, ok := v.URLs[path.Base(name)]
	if !ok {
		for k, x := range v.URLs { // names are matched without regard to case, as on Windows
			if strings.EqualFold(k, path.Base(name)) {
				u, ok = x, true
				break
			}
		}
	}
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	s := &remoteStream{v: v, url: u, name: path.Base(name)}
	if err := s.connect(); err != nil {
		return nil, err
	}
	return s, nil
}

// remoteStream reads one part from its start. A connection that drops is
// opened again where it stopped, when the server sends parts.
type remoteStream struct {
	v     *RemoteVolumes
	url   string
	name  string
	body  io.ReadCloser
	off   int64
	size  int64
	tries int
}

func (s *remoteStream) connect() error {
	req, err := http.NewRequestWithContext(s.v.Ctx, "GET", s.url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Lootnest")
	if s.off > 0 {
		req.Header.Set("Range", "bytes="+strconv.FormatInt(s.off, 10)+"-")
	}
	resp, err := s.v.Client.Do(req)
	if err != nil {
		return netErr("the download's site", err)
	}
	switch {
	case s.off == 0 && resp.StatusCode == http.StatusOK:
		s.size = resp.ContentLength
	case s.off > 0 && resp.StatusCode == http.StatusPartialContent:
		if a, _, total, ok := contentRange(resp.Header.Get("Content-Range")); !ok || a != s.off {
			resp.Body.Close()
			return errors.New("the download's site sent another part than the one asked for")
		} else if s.size <= 0 {
			s.size = total
		}
	default:
		resp.Body.Close()
		if s.off > 0 {
			return ErrNoRanges
		}
		return fmt.Errorf("the download's site answered %s", strings.ToLower(resp.Status))
	}
	s.body = resp.Body
	return nil
}

func (s *remoteStream) Read(p []byte) (int, error) {
	for {
		n, err := s.body.Read(p)
		s.off += int64(n)
		s.v.read.Add(int64(n))
		if n > 0 || err == nil {
			return n, nil // an error comes back on the next call
		}
		if err == io.EOF && (s.size <= 0 || s.off >= s.size) {
			return 0, io.EOF
		}
		// The connection dropped, or ended before the part did.
		if s.v.Ctx.Err() != nil {
			return 0, s.v.Ctx.Err()
		}
		s.body.Close()
		if s.tries++; s.tries > remoteTries {
			return 0, netErr("the download's site", errors.New("the connection kept dropping"))
		}
		select {
		case <-s.v.Ctx.Done():
			return 0, s.v.Ctx.Err()
		case <-time.After(time.Duration(s.tries) * time.Second):
		}
		if cerr := s.connect(); cerr != nil {
			return 0, cerr
		}
	}
}

func (s *remoteStream) Close() error { return s.body.Close() }

func (s *remoteStream) Stat() (fs.FileInfo, error) { return remoteInfo{s.name, s.size}, nil }

type remoteInfo struct {
	name string
	size int64
}

func (i remoteInfo) Name() string       { return i.name }
func (i remoteInfo) Size() int64        { return i.size }
func (i remoteInfo) Mode() fs.FileMode  { return 0o444 }
func (i remoteInfo) ModTime() time.Time { return time.Time{} }
func (i remoteInfo) IsDir() bool        { return false }
func (i remoteInfo) Sys() any           { return nil }
