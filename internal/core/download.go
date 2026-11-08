package core

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// netErr drops the address from a request's error: it can hold a key or a
// token.
func netErr(store string, err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		err = ue.Err
	}
	if errors.Is(err, context.Canceled) {
		return err
	}
	return fmt.Errorf("couldn't reach %s (%w)", store, err)
}

// downloader fetches a store's files for the user (itch.io, GOG).
type downloader struct {
	client *http.Client
	store  string                              // named in errors
	status func(code int, status string) error // the store's reading of an HTTP status
	trace  func(*url.URL)
	keep   bool // a download cut short keeps its .part
}

// get saves the file at u to dst, through dst+".part": only over https,
// checked against size and md5 when they're known. A .part an earlier try
// left goes on from where it stopped when the server answers the Range
// request with 206; otherwise the download starts over. A download that
// fails its checks leaves nothing behind, and one cut short (a refusal, a
// lost connection or a cancel) keeps its .part only with keep.
func (d downloader) get(ctx context.Context, u, dst string, size int64, sum string, progress func(done, total int64)) (err error) {
	pu, err := url.Parse(u)
	if err != nil || pu.Scheme != "https" {
		return errors.New("Lootnest only downloads over https")
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	part, cut := dst+".part", false
	defer func() {
		if err != nil && !(d.keep && cut) {
			_ = os.Remove(part)
		}
	}()
	var have int64
	if st, e := os.Stat(part); e == nil && st.Mode().IsRegular() {
		if have = st.Size(); size > 0 && have >= size {
			have = 0
		}
	}
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return fmt.Errorf("%s sent a download address Lootnest can't read", d.store)
	}
	if have > 0 {
		req.Header.Set("Range", "bytes="+strconv.FormatInt(have, 10)+"-")
	}
	if d.trace != nil {
		d.trace(pu)
	}
	resp, err := d.client.Do(req)
	if err != nil {
		cut = true
		return netErr(d.store, err)
	}
	defer resp.Body.Close()
	if have > 0 && resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		resp.Body.Close()
		if err := os.Remove(part); err != nil {
			return err
		}
		return d.get(ctx, u, dst, size, sum, progress)
	}
	if err := d.status(resp.StatusCode, resp.Status); err != nil {
		cut = true
		return err
	}
	h := md5.New()
	flags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	if have > 0 && resp.StatusCode == http.StatusPartialContent && strings.HasPrefix(resp.Header.Get("Content-Range"), "bytes "+strconv.FormatInt(have, 10)+"-") {
		pf, err := os.Open(part)
		if err != nil {
			return err
		}
		_, err = io.Copy(h, pf)
		pf.Close()
		if err != nil {
			return err
		}
		flags = os.O_WRONLY | os.O_APPEND
	} else {
		have = 0
	}
	total := size
	var body io.Reader = resp.Body
	if size > 0 {
		body = io.LimitReader(body, size-have+1)
	} else if resp.ContentLength >= 0 {
		total = have + resp.ContentLength
	}
	f, err := os.OpenFile(part, flags, 0o644)
	if err != nil {
		return err
	}
	w := io.MultiWriter(f, h)
	done := have
	buf := make([]byte, 1<<20)
	for {
		n, rerr := body.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				f.Close()
				return werr
			}
			done += int64(n)
			if progress != nil {
				progress(done, total)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			f.Close()
			cut = true
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return netErr(d.store, rerr)
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	if size > 0 && done != size {
		return fmt.Errorf("the download from %s isn't the size %s lists. Try again", d.store, d.store)
	}
	if sum != "" && !strings.EqualFold(hex.EncodeToString(h.Sum(nil)), sum) {
		return fmt.Errorf("the download from %s doesn't match its checksum. Try again", d.store)
	}
	return os.Rename(part, dst)
}

// downloadPath is where a store's file is downloaded to: Lootnest's own
// downloads\<store>\<id> folder, under a name safe on Windows.
func downloadPath(dataDir, store string, id int64, filename string) string {
	name := filepath.Base(strings.ReplaceAll(filename, `\`, "/"))
	name = strings.TrimLeft(safeFileName(name), ". ")
	if name == "" || windowsDevice(name) {
		name = "download" + filepath.Ext(name)
	}
	return filepath.Join(dataDir, "downloads", store, strconv.FormatInt(id, 10), name)
}
