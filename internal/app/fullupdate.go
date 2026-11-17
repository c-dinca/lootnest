//go:build windows

package app

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/win"
)

// ---------------------------------------------------------------------------
// Updates a script plugin starts (core/dlqueue.go, core/fullupdate.go). A
// plugin with the "updates" permission may ask, for a game no store keeps
// up to date, that files be downloaded: lootnest.updates.run("start",
// {...}). Everything after that is Lootnest's own code, shown on the
// Updates page:
//
//	download  into <game folder>.lootnest-new\dl, never over the game,
//	          after a check that there's room; pause, resume, cancel
//	unpack    a .zip, .7z or .rar is opened into ...\build
//	ready     a whole build was found: the user reads what's kept from the
//	          old folder and says yes (ps_pluginJobOp "apply")
//	hand      a patch or an installer: handed to the update centre's own
//	          drawer, which shows what it changes (an installer is only
//	          ever run on the user's click there)
//	apply     core.RunFullUpdate: saves copied, folder swapped, checked
//	kept      the old folder stays until the game has run, or for some
//	          days; Roll back until then
//
// Each job is fullupdates\<hash of the game's key>\job.json with its
// log.txt; a full update's own state is state.json beside it, and one cut
// short is put right at the next start (recoverJobs).
// ---------------------------------------------------------------------------

type jobFile struct {
	URL    string `json:"url"`
	Name   string `json:"name"`
	Size   int64  `json:"size,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
}

// updJob is one update a plugin started.
type updJob struct {
	ID      string    `json:"id"`
	Key     string    `json:"key"` // the Library key
	Name    string    `json:"name"`
	Plugin  string    `json:"plugin"`
	Offer   string    `json:"offer"`
	Title   string    `json:"title"`
	Version string    `json:"version,omitempty"`
	Source  string    `json:"source,omitempty"`
	Files   []jobFile `json:"files"`
	Staging string    `json:"staging"`
	Dir     string    `json:"dir"` // the game's folder
	// Mode: "" the whole folder replaced (2.11), "delta" only what changed
	// read in place (deltaupdate.go), "delta-local" the same from a whole
	// download.
	Mode   string `json:"mode,omitempty"`
	Paused bool   `json:"paused,omitempty"` // a delta job stopped while it fetched
	// Phase: download, fetch, unpack, ready, hand, apply, kept, done,
	// failed, rolledback.
	Phase   string    `json:"phase"`
	Step    string    `json:"step,omitempty"` // the full update's state while it's applied
	Error   string    `json:"error,omitempty"`
	Path    string    `json:"path,omitempty"` // for "hand": the patch's folder or the installer
	Kind    string    `json:"kind,omitempty"` // for "hand": patch or installer
	Old     string    `json:"old,omitempty"`  // the old folder, while it's kept
	Started time.Time `json:"started"`
	Applied time.Time `json:"applied,omitempty"`
	Broke   bool      `json:"broke,omitempty"` // the game closed right after its first start
}

// JobView is a job as the page draws it.
type JobView struct {
	updJob
	UpdKey string `json:"updKey"` // the Updates page's key of the game
	Done   int64  `json:"done"`
	Total  int64  `json:"total"`
	Speed  int64  `json:"speed"`
	ETA    int64  `json:"eta"`
	Ready  int    `json:"ready"`
	Count  int    `json:"count,omitempty"` // files compared or fetched so far
	Of     int    `json:"of,omitempty"`
}

const fullKeepDays = 14 // how long an old folder is kept when the game isn't played

var jobs struct {
	sync.Mutex
	m      map[string]*updJob // by job id
	q      *core.DownloadQueue
	loaded bool
	told   time.Time
}

func jobsRoot() string { return filepath.Join(DataDir(), "fullupdates") }

func jobDir(key string) string {
	h := sha1.Sum([]byte(strings.ToLower(key)))
	return filepath.Join(jobsRoot(), hex.EncodeToString(h[:8]))
}

// loadJobs reads the jobs kept on disk, once, and starts the queue.
// jobs is locked.
func (a *App) loadJobs() {
	if jobs.loaded {
		return
	}
	jobs.loaded, jobs.m = true, map[string]*updJob{}
	ents, _ := os.ReadDir(jobsRoot())
	for _, e := range ents {
		b, err := os.ReadFile(filepath.Join(jobsRoot(), e.Name(), "job.json"))
		if err != nil {
			continue
		}
		var j updJob
		if json.Unmarshal(b, &j) == nil && j.ID != "" {
			jobs.m[j.ID] = &j
		}
	}
	jobs.q = core.NewDownloadQueue(filepath.Join(DataDir(), "downloads.json"), core.GuardedClient(false), func() { go a.jobsMoved() })
}

func saveJob(j *updJob) {
	dir := jobDir(j.Key)
	_ = os.MkdirAll(dir, 0o755)
	b, _ := json.MarshalIndent(j, "", "  ")
	tmp := filepath.Join(dir, "job.json.tmp")
	if os.WriteFile(tmp, b, 0o644) == nil {
		_ = os.Rename(tmp, filepath.Join(dir, "job.json"))
	}
}

// jobLog adds a line to a job's log.txt and to Lootnest's own log.
func (a *App) jobLog(j *updJob, format string, args ...any) {
	line := fmt.Sprintf(format, args...)
	a.logf("Update of %s: %s", j.Name, line)
	dir := jobDir(j.Key)
	_ = os.MkdirAll(dir, 0o755)
	if f, err := os.OpenFile(filepath.Join(dir, "log.txt"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
		fmt.Fprintf(f, "[%s] %s\r\n", time.Now().Format("2006-01-02 15:04:05"), line)
		f.Close()
	}
}

func (a *App) emitJobs() { a.emit("pluginJobs", nil) }

// setPhase moves a job on, keeps it, and tells the page and the plugin.
func (a *App) setPhase(j *updJob, phase, errText string) {
	jobs.Lock()
	j.Phase, j.Error = phase, errText
	if phase != "apply" {
		j.Step = ""
	}
	saveJob(j)
	jobs.Unlock()
	a.emitJobs()
	switch phase {
	case "kept", "failed", "rolledback", "done", "ready", "hand":
		a.tellPlugin(j)
	}
}

// tellPlugin calls the plugin's "update" handler with what became of the
// update it started.
func (a *App) tellPlugin(j *updJob) {
	r := a.scriptRun(j.Plugin)
	if r == nil {
		return
	}
	go func() {
		if _, err := r.host.Call("update", map[string]any{"job": j.ID, "offer": j.Offer, "game": j.Key, "state": j.Phase, "error": j.Error, "title": j.Title, "version": j.Version}); err != nil && !errors.Is(err, core.ErrNoHandler) {
			a.logf("Plugin %s: update: %v", r.plugin.Name, err)
		}
	}()
}

// errUpdating is why a game that starts from p can't start now: Lootnest
// is replacing its folder. Else nil.
func (a *App) errUpdating(p string) error {
	jobs.Lock()
	defer jobs.Unlock()
	for _, j := range jobs.m {
		if j.Phase == "apply" && j.Dir != "" && isInside(p, j.Dir) {
			return fmt.Errorf("Lootnest is updating %s. Start it once that's done.", j.Name)
		}
	}
	return nil
}

// jobTarget is the game a job is for, as the update centre sees it.
func (a *App) jobTarget(key string) (updTarget, core.FoundGame, error) {
	f, ok := a.foundByKey(key)
	if !ok {
		return updTarget{}, f, errors.New("that game isn't in the Library any more")
	}
	t, err := a.updTarget("found:" + key)
	if err != nil {
		return t, f, errors.New("a store keeps that game up to date, so Lootnest doesn't")
	}
	return t, f, nil
}

// activeJob is the job in progress for a game, if any. jobs is locked.
func activeJob(key string) *updJob {
	for _, j := range jobs.m {
		if j.Key == key && j.Phase != "done" && j.Phase != "failed" && j.Phase != "rolledback" {
			return j
		}
	}
	return nil
}

type startReq struct {
	Offer   string    `json:"offer"`
	Game    string    `json:"game"`
	Title   string    `json:"title"`
	Version string    `json:"version"`
	Source  string    `json:"source"`
	Size    int64     `json:"size"`
	Files   []jobFile `json:"files"`
}

// pluginUpdate is lootnest.updates.run for the plugin id.
func (a *App) pluginUpdate(id, op string, raw json.RawMessage) (any, error) {
	if op != "start" {
		return nil, errors.New("Lootnest has no such update call")
	}
	var req startReq
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, errors.New("that update request can't be read")
	}
	t, f, err := a.jobTarget(req.Game)
	if err != nil {
		return nil, err
	}
	if err := core.FullUpdateRefused(t.Dir); err != nil {
		return nil, err
	}
	if t.running() {
		return nil, fmt.Errorf("%s is running; close it first", t.Name)
	}
	if len(req.Files) == 0 || len(req.Files) > 300 {
		return nil, errors.New("an update needs 1 to 300 files to download")
	}
	staging := t.Dir + ".lootnest-new"
	var total int64
	names := map[string]bool{}
	files := make([]jobFile, 0, len(req.Files))
	for i, x := range req.Files {
		u, err := url.Parse(strings.TrimSpace(x.URL))
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil {
			return nil, errors.New("Lootnest only downloads over https")
		}
		if u.Scheme == "http" && len(x.SHA256) != 64 {
			return nil, errors.New("Lootnest downloads over plain http only when the source gives the file's SHA-256")
		}
		name := core.SafeDownloadName(x.Name)
		if name == "" {
			name = core.SafeDownloadName(filepath.Base(u.Path))
		}
		if name == "" {
			name = fmt.Sprintf("download-%d", i+1)
		}
		for n := 2; names[strings.ToLower(name)]; n++ {
			name = fmt.Sprintf("%d-%s", n, name)
		}
		names[strings.ToLower(name)] = true
		files = append(files, jobFile{URL: u.String(), Name: name, Size: max(x.Size, 0), SHA256: strings.ToLower(x.SHA256)})
		total += max(x.Size, 0)
	}
	if total == 0 {
		total = max(req.Size, 0)
	}
	_, delta := deltaOK(files)
	// Room for the download, what it unpacks to, and a margin. An archive
	// read in place needs room only for what changed, checked as it's known.
	if need, free := total*22/10, win.DiskFree(filepath.Dir(t.Dir)); !delta && total > 0 && free >= 0 && free < need {
		return nil, fmt.Errorf("there isn't enough room on %s: this update needs about %s (the download, what it unpacks to and a margin) and %s is free",
			strings.ToUpper(filepath.VolumeName(t.Dir)), gb(need), gb(free))
	}
	jobs.Lock()
	a.loadJobs()
	if activeJob(f.Key) != nil {
		jobs.Unlock()
		return nil, fmt.Errorf("an update of %s is already under way; see the Updates page", t.Name)
	}
	j := &updJob{ID: "j" + time.Now().Format("20060102150405"), Key: f.Key, Name: t.Name, Plugin: id, Offer: req.Offer, Title: cutText(req.Title, 300), Version: cutText(req.Version, 60),
		Source: cutText(req.Source, 60), Files: files, Staging: staging, Dir: t.Dir, Phase: "download", Started: time.Now()}
	// A job of this game from before is replaced; its log starts again.
	for k, old := range jobs.m {
		if old.Key == f.Key {
			delete(jobs.m, k)
		}
	}
	_ = os.Remove(filepath.Join(jobDir(f.Key), "log.txt"))
	_ = os.Remove(filepath.Join(jobDir(f.Key), "state.json"))
	jobs.m[j.ID] = j
	saveJob(j)
	q := jobs.q
	jobs.Unlock()
	_ = os.RemoveAll(staging)
	a.jobLog(j, "Started: %s, from %s, %d file(s), into %s", j.Title, j.Source, len(files), staging)
	if delta {
		jobs.Lock()
		j.Mode, j.Phase = "delta", "fetch"
		saveJob(j)
		jobs.Unlock()
		go a.runDelta(j)
		a.emitJobs()
		return map[string]any{"job": j.ID}, nil
	}
	list := make([]core.Download, len(files))
	for i, x := range files {
		list[i] = core.Download{ID: fmt.Sprintf("%s-%d", j.ID, i), Job: j.ID, URL: x.URL, Dst: filepath.Join(staging, "dl", x.Name), Size: x.Size, SHA256: x.SHA256}
	}
	q.Add(list...)
	a.emitJobs()
	return map[string]any{"job": j.ID}, nil
}

func cutText(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

// jobsMoved is the queue saying something changed: a job whose downloads
// are all there goes on to unpacking, one whose download failed says so.
func (a *App) jobsMoved() {
	jobs.Lock()
	a.loadJobs()
	q := jobs.q
	var unpack, failed []*updJob
	for _, j := range jobs.m {
		if j.Phase != "download" {
			continue
		}
		p := q.Job(j.ID)
		switch {
		case p.Failed != "" && p.Ready+1 <= p.Files && j.Error != p.Failed:
			j.Error = p.Failed
			failed = append(failed, j)
		case p.Files > 0 && p.Ready == p.Files:
			j.Phase = "unpack"
			saveJob(j)
			unpack = append(unpack, j)
		}
	}
	tell := time.Since(jobs.told) > 400*time.Millisecond || len(unpack)+len(failed) > 0
	if tell {
		jobs.told = time.Now()
	}
	jobs.Unlock()
	for _, j := range failed {
		a.jobLog(j, "A download failed: %s", j.Error)
		a.notify(j.Name+" couldn't be downloaded", j.Error+" Try again from the Updates page.")
	}
	for _, j := range unpack {
		go a.jobUnpack(j)
	}
	if tell {
		a.emitJobs()
	}
}

// jobUnpack opens what was downloaded and works out what it is: a whole
// build of the game, a patch, or an installer.
func (a *App) jobUnpack(j *updJob) {
	if j.Mode == "delta-local" {
		a.jobLog(j, "Downloaded. Comparing it with the game's files")
		a.deltaFromDisk(j)
		return
	}
	a.jobLog(j, "Downloaded. Opening it")
	a.emitJobs()
	t, _, err := a.jobTarget(j.Key)
	if err != nil {
		a.setPhase(j, "failed", sentence(err))
		return
	}
	dl := filepath.Join(j.Staging, "dl")
	ents, _ := os.ReadDir(dl)
	var archives, exes []string
	for _, e := range ents {
		switch ext := strings.ToLower(filepath.Ext(e.Name())); {
		case e.IsDir():
		case ext == ".zip" || ext == ".7z":
			archives = append(archives, e.Name())
		case ext == ".rar":
			if n := strings.ToLower(e.Name()); !strings.Contains(n, ".part") || strings.Contains(n, ".part1.") || strings.Contains(n, ".part01.") || strings.Contains(n, ".part001.") {
				archives = append(archives, e.Name())
			}
		case ext == ".exe":
			exes = append(exes, e.Name())
		}
	}
	sort.Strings(archives)
	look := dl
	if len(archives) == 1 {
		build := filepath.Join(j.Staging, "build")
		_ = os.RemoveAll(build)
		limit := win.DiskFree(filepath.Dir(j.Staging))
		if limit < 0 {
			limit = 1 << 50
		}
		if _, err := core.StageUpdateMax(filepath.Join(dl, archives[0]), build, limit); err != nil {
			a.jobLog(j, "Couldn't open %s: %v", archives[0], err)
			a.setPhase(j, "failed", "The download couldn't be opened: "+sentence(err))
			return
		}
		look = build
	} else if len(archives) > 1 {
		a.jobLog(j, "It holds %d archives; Lootnest opens one", len(archives))
		a.setPhase(j, "failed", "The download holds several archives, which Lootnest doesn't open by itself. They're in "+dl+".")
		return
	}
	rel, err := filepath.Rel(t.Dir, t.Exe)
	if err != nil || strings.HasPrefix(rel, "..") {
		rel = filepath.Base(t.Exe)
	}
	root, newExe, err := core.FindBuild(look, filepath.ToSlash(rel))
	if err == nil {
		u := &core.FullUpdate{ID: j.ID, Key: j.Key, Name: j.Name, Dir: t.Dir, Exe: filepath.ToSlash(rel), Title: j.Title, Version: j.Version, Source: j.Source,
			Plugin: j.Plugin, Offer: j.Offer, Staging: j.Staging, Build: root, NewExe: newExe, State: core.FUStaged}
		if err := core.SaveFullUpdate(jobDir(j.Key), u); err != nil {
			a.setPhase(j, "failed", sentence(err))
			return
		}
		a.jobLog(j, "It's a whole build (%s). Waiting for you to look at what's kept", newExe)
		a.setPhase(j, "ready", "")
		a.notify(j.Name+" is ready to update", "Open the Updates page to see what's kept and replace the game's folder.")
		return
	}
	// Not a whole build: an installer, or files that go over the game's.
	if len(archives) == 0 && len(exes) > 0 {
		pick := exes[0]
		for _, e := range exes {
			if n := strings.ToLower(e); strings.Contains(n, "setup") || strings.Contains(n, "install") {
				pick = e
				break
			}
		}
		jobs.Lock()
		j.Kind, j.Path = "installer", filepath.Join(dl, pick)
		jobs.Unlock()
		a.jobLog(j, "It's an installer (%s). Lootnest runs it only when you say so", pick)
	} else {
		jobs.Lock()
		j.Kind, j.Path = "patch", look
		jobs.Unlock()
		a.jobLog(j, "It isn't a whole build: its files go over the game's, through the update panel")
	}
	a.setPhase(j, "hand", "")
	a.notify(j.Name+"'s update is downloaded", "Open the Updates page to look at it and apply it.")
}

// apiPluginJobs lists the jobs for the page.
func (a *App) apiPluginJobs() []JobView {
	jobs.Lock()
	defer jobs.Unlock()
	a.loadJobs()
	out := []JobView{}
	for _, j := range jobs.m {
		v := JobView{updJob: *j, UpdKey: "found:" + j.Key}
		if j.Phase == "download" {
			p := jobs.q.Job(j.ID)
			v.Done, v.Total, v.Speed, v.ETA, v.Paused, v.Ready = p.Done, p.Total, p.Speed, p.ETA, p.Paused, p.Ready
			if p.Failed != "" {
				v.Error = p.Failed
			}
		}
		if j.Phase == "fetch" {
			a.deltaView(&v)
			v.Paused = v.Paused || j.Paused
		}
		out = append(out, v)
	}
	sort.Slice(out, func(i, k int) bool { return out[i].Started.After(out[k].Started) })
	return out
}

func (a *App) job(id string) (*updJob, error) {
	jobs.Lock()
	defer jobs.Unlock()
	a.loadJobs()
	if j := jobs.m[id]; j != nil {
		return j, nil
	}
	return nil, errors.New("That update isn't on the list any more.")
}

// keptPaths is fullupdates\keep.json: for each game, the paths the user
// chose to keep last time, ticked again next time.
func loadKeptPaths() map[string][]string {
	out := map[string][]string{}
	if b, err := os.ReadFile(filepath.Join(jobsRoot(), "keep.json")); err == nil {
		_ = json.Unmarshal(b, &out)
	}
	return out
}

// jobSaves are the game's save locations and, of them, the ones inside
// its folder, relative to it.
func (a *App) jobSaves(f core.FoundGame, dir string) (locs []core.SaveLocation, inside []string) {
	locs = withFiles(a.vaultLocs(f))
	for _, l := range core.InFolder(locs, dir) {
		if rel, err := filepath.Rel(dir, l.Resolved); err == nil && !strings.HasPrefix(rel, "..") && rel != "." {
			inside = append(inside, filepath.ToSlash(rel))
		}
	}
	return locs, inside
}

// JobPlan is what applying a ready job would do.
type JobPlan struct {
	*core.FullPlan
	Yours  []string `json:"yours"`  // what the user kept last time
	Saves  int      `json:"saves"`  // save locations copied first
	Folder string   `json:"folder"` // the game's folder
	Old    string   `json:"old"`    // where the old one is kept
	Days   int      `json:"days"`
	// Only what changed (deltaupdate.go).
	Delta   bool  `json:"delta,omitempty"`
	Full    bool  `json:"full,omitempty"` // the whole game: files it lacks are moved aside
	Changed int   `json:"changed,omitempty"`
	Added   int   `json:"added,omitempty"`
	Same    int   `json:"same,omitempty"`
	Removed int   `json:"removed,omitempty"`
	Bytes   int64 `json:"bytes,omitempty"`
}

// apiPluginJobPlan says what a ready job would keep and replace.
func (a *App) apiPluginJobPlan(id string) (*JobPlan, error) {
	j, err := a.job(id)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(j.Mode, "delta") {
		return a.deltaPlanFor(j)
	}
	u, err := core.LoadFullUpdate(jobDir(j.Key))
	if err != nil || u == nil || u.State != core.FUStaged {
		return nil, errors.New("That update isn't waiting to be applied.")
	}
	_, f, err := a.jobTarget(j.Key)
	if err != nil {
		return nil, errors.New(sentence(err))
	}
	locs, inside := a.jobSaves(f, u.Dir)
	yours := loadKeptPaths()[j.Key]
	p, err := core.PlanFullUpdate(u.Dir, u.Build, inside, nil)
	if err != nil {
		return nil, errors.New(sentence(err))
	}
	return &JobPlan{FullPlan: p, Yours: yours, Saves: len(locs), Folder: u.Dir, Old: u.Dir + ".lootnest-old", Days: fullKeepDays}, nil
}

// apiPluginJobOp steers a job: pause, resume, cancel, apply (with the
// paths to keep), rollback, finish (let the old folder go), dismiss.
func (a *App) apiPluginJobOp(id, op string, keep []string) error {
	j, err := a.job(id)
	if err != nil {
		return err
	}
	jobs.Lock()
	q, phase := jobs.q, j.Phase
	delta := strings.HasPrefix(j.Mode, "delta")
	jobs.Unlock()
	switch op {
	case "pause":
		if phase == "fetch" {
			a.deltaPause(j)
		} else {
			q.Pause(id)
		}
	case "resume":
		jobs.Lock()
		j.Error = ""
		jobs.Unlock()
		if phase == "fetch" {
			a.deltaResume(j)
		} else {
			q.Resume(id)
		}
	case "cancel":
		if phase == "apply" || phase == "kept" {
			return errors.New("That update is past cancelling. Roll it back instead.")
		}
		q.Remove(id)
		go func() {
			if phase == "fetch" {
				a.deltaCancel(j)
			}
			_ = os.RemoveAll(j.Staging)
			a.jobLog(j, "Cancelled; what was downloaded is deleted")
			jobs.Lock()
			delete(jobs.m, id)
			_ = os.Remove(filepath.Join(jobDir(j.Key), "job.json"))
			_ = os.Remove(filepath.Join(jobDir(j.Key), "state.json"))
			jobs.Unlock()
			j.Phase = "failed"
			a.tellPlugin(j)
			a.emitJobs()
		}()
	case "apply":
		if phase != "ready" {
			return errors.New("That update isn't waiting to be applied.")
		}
		if delta {
			go a.deltaApply(j, keep)
		} else {
			go a.jobApply(j, keep)
		}
	case "rollback":
		if phase != "kept" {
			return errors.New("The folder from before that update isn't kept any more.")
		}
		if delta {
			go a.deltaRollback(j)
		} else {
			go a.jobRollback(j)
		}
	case "finish":
		if phase != "kept" {
			return errors.New("That update has nothing kept.")
		}
		go a.jobFinish(j, "you asked")
	case "dismiss":
		if phase != "failed" && phase != "rolledback" && phase != "done" && phase != "hand" {
			return errors.New("That update is still under way.")
		}
		go func() {
			if phase == "hand" || phase == "failed" {
				_ = os.RemoveAll(j.Staging)
			}
			jobs.Lock()
			delete(jobs.m, id)
			_ = os.Remove(filepath.Join(jobDir(j.Key), "job.json"))
			jobs.Unlock()
			a.emitJobs()
		}()
	default:
		return errors.New("Lootnest has no such step.")
	}
	a.emitJobs()
	return nil
}

// apiPluginJobLog is a job's log, for the page.
func (a *App) apiPluginJobLog(id string) (string, error) {
	j, err := a.job(id)
	if err != nil {
		return "", err
	}
	b, _ := os.ReadFile(filepath.Join(jobDir(j.Key), "log.txt"))
	return string(b), nil
}

// copySaves writes a copy of a game's saves, named name, where the game's
// page lists its copies (the safety net's store), and returns its id.
func (a *App) copySaves(f core.FoundGame, name string) (string, error) {
	netMu.Lock()
	defer netMu.Unlock()
	locs := withFiles(a.vaultLocs(f))
	if len(locs) == 0 {
		return "", nil // no saves found: nothing to copy
	}
	hash, err := core.SavesHash(locs)
	if err != nil {
		return "", err
	}
	c, key := netStore(), gameSavesKey(f)
	st := loadNet()
	e := st[key]
	v, err := c.SaveGameVersion(key, name, locs, hash, max(e.Keep, netKeep))
	if err != nil {
		return "", err
	}
	e.Hash, e.At, e.Vault, e.Quick = hash, v.Created, true, quickPrint(locs)
	st[key] = e
	saveNet(st)
	return v.ID, nil
}

// fullHooks are the app's side of a full update's steps for job j.
func (a *App) fullHooks(j *updJob, f core.FoundGame) core.FullUpdateHooks {
	steps := map[string]string{core.FUBackup: "Copying your saves", core.FUSwap: "Replacing the game's folder", core.FURestore: "Carrying your files over", core.FUVerify: "Checking the new build"}
	return core.FullUpdateHooks{
		Save: func(u *core.FullUpdate) error {
			jobs.Lock()
			j.Step = steps[u.State]
			saveJob(j)
			jobs.Unlock()
			a.emitJobs()
			return core.SaveFullUpdate(jobDir(j.Key), u)
		},
		Log: func(s string) { a.jobLog(j, "%s", s) },
		Backup: func() (string, error) {
			name := "Before the update"
			if j.Version != "" {
				name += " to " + j.Version
			}
			return a.copySaves(f, name)
		},
		RestoreSaves: func(id string) error {
			// The old folder is back whole, so only saves outside it could have
			// changed, and nothing here writes those: put the copy back only
			// if they differ from it.
			netMu.Lock()
			defer netMu.Unlock()
			locs := withFiles(a.vaultLocs(f))
			hash, err := core.SavesHash(locs)
			if err != nil {
				return err
			}
			for _, v := range netStore().GameVersions(gameSavesKey(f)) {
				if v.ID == id && v.Hash == hash {
					return nil // as they were
				}
			}
			return a.restoreVersion(netStore(), f, id, a.vaultLocs(f))
		},
	}
}

// jobApply runs a ready job's full update, after the user said yes.
func (a *App) jobApply(j *updJob, keep []string) {
	fail := func(err error) {
		a.jobLog(j, "Not applied: %v", err)
		a.emit("toast", map[string]any{"kind": "error", "text": sentence(err)})
		a.emitJobs()
	}
	t, f, err := a.jobTarget(j.Key)
	if err != nil {
		fail(err)
		return
	}
	u, err := core.LoadFullUpdate(jobDir(j.Key))
	if err != nil || u == nil || u.State != core.FUStaged {
		fail(errors.New("that update isn't waiting to be applied"))
		return
	}
	updMu.Lock()
	defer updMu.Unlock()
	if t.running() {
		fail(fmt.Errorf("%s is running; close it first", t.Name))
		return
	}
	out := map[string]any{}
	if !a.modsOffBefore(t, out) {
		fail(errors.New(fmt.Sprint(out["error"])))
		return
	}
	if out["modsOff"] == true {
		a.jobLog(j, "Its mods were taken out first; they're off until you apply them again")
	}
	_, inside := a.jobSaves(f, u.Dir)
	u.Keep = append(append([]string{}, inside...), keep...)
	kept := loadKeptPaths()
	kept[j.Key] = keep
	if b, err := json.MarshalIndent(kept, "", "  "); err == nil {
		_ = os.WriteFile(filepath.Join(jobsRoot(), "keep.json"), b, 0o644)
	}
	jobs.Lock()
	j.Phase, j.Error = "apply", ""
	saveJob(j)
	jobs.Unlock()
	a.emitJobs()
	a.jobLog(j, "Applying %s to %s", j.Title, u.Dir)
	if err := core.RunFullUpdate(u, a.fullHooks(j, f)); err != nil {
		if u.Aside != "" {
			if rerr := win.Recycle(a.hwnd, u.Aside); rerr == nil {
				a.jobLog(j, "The build that failed went to the Recycle Bin")
			}
		}
		a.setPhase(j, "failed", sentence(err)+" The game is as it was.")
		a.notify(j.Name+" wasn't updated", sentence(err)+" The game is as it was.")
		return
	}
	jobs.Lock()
	j.Old, j.Applied = u.Old, time.Now()
	jobs.Unlock()
	_ = os.RemoveAll(j.Staging) // the download, now that its build is in place
	a.setPhase(j, "kept", "")
	a.notify(j.Name+" is updated", "The folder from before is kept until you've played, or "+fmt.Sprint(fullKeepDays)+" days. Roll back from the Updates page.")
	a.startScan()
	go a.watchKept(j)
}

// jobRollback takes the old folder back.
func (a *App) jobRollback(j *updJob) {
	t, f, err := a.jobTarget(j.Key)
	if err == nil && t.running() {
		err = fmt.Errorf("%s is running; close it first", t.Name)
	}
	u, lerr := core.LoadFullUpdate(jobDir(j.Key))
	if err == nil && (lerr != nil || u == nil) {
		err = errors.New("that update's record is gone")
	}
	if err != nil {
		a.emit("toast", map[string]any{"kind": "error", "text": sentence(err)})
		return
	}
	updMu.Lock()
	defer updMu.Unlock()
	h := a.fullHooks(j, f)
	h.Backup = func() (string, error) { return a.copySaves(f, "Before rolling the update back") }
	if err := core.RollbackFullUpdate(u, h); err != nil {
		a.jobLog(j, "Not rolled back: %v", err)
		a.emit("toast", map[string]any{"kind": "error", "text": sentence(err)})
		return
	}
	if u.Aside != "" {
		if err := win.Recycle(a.hwnd, u.Aside); err != nil {
			a.jobLog(j, "The new build stays at %s: %v", u.Aside, err)
		}
	}
	jobs.Lock()
	j.Old = ""
	jobs.Unlock()
	a.jobLog(j, "Rolled back. A copy of the saves as they were is among the game's copies")
	a.setPhase(j, "rolledback", "")
	a.startScan()
}

// jobFinish lets the old folder go, to the Recycle Bin.
func (a *App) jobFinish(j *updJob, why string) {
	if strings.HasPrefix(j.Mode, "delta") {
		a.deltaFinish(j, why)
		return
	}
	u, err := core.LoadFullUpdate(jobDir(j.Key))
	if err != nil || u == nil {
		return
	}
	updMu.Lock()
	defer updMu.Unlock()
	old, err := core.FinishFullUpdate(u, core.FullUpdateHooks{Save: func(x *core.FullUpdate) error { return core.SaveFullUpdate(jobDir(j.Key), x) }})
	if err != nil {
		return
	}
	if old != "" {
		if err := win.Recycle(a.hwnd, old); err != nil {
			a.jobLog(j, "The old folder stays at %s: %v", old, err)
		} else {
			a.jobLog(j, "The old folder went to the Recycle Bin (%s)", why)
		}
	}
	jobs.Lock()
	j.Old = ""
	jobs.Unlock()
	a.setPhase(j, "done", "")
}

// watchKept follows a game whose old folder is kept: once the game has run
// for a minute and closed, or after fullKeepDays, the old folder goes. A
// first start that ends within 20 seconds is told, with Roll back.
func (a *App) watchKept(j *updJob) {
	_, f, err := a.jobTarget(j.Key)
	if err != nil || f.ProcessName == "" {
		return
	}
	name := strings.ToLower(strings.TrimSuffix(f.ProcessName, ".exe"))
	var started time.Time
	ranLong := false
	for {
		time.Sleep(5 * time.Second)
		jobs.Lock()
		phase, applied := j.Phase, j.Applied
		jobs.Unlock()
		if phase != "kept" {
			return
		}
		procs, _ := win.ProcessNames()
		up := procs[name]
		switch {
		case up && started.IsZero():
			started = time.Now()
		case up && time.Since(started) > time.Minute:
			ranLong = true
		case !up && !started.IsZero():
			if ranLong {
				a.jobFinish(j, "the game ran")
				return
			}
			if time.Since(started) < 20*time.Second && !j.Broke {
				jobs.Lock()
				j.Broke = true
				saveJob(j)
				jobs.Unlock()
				a.jobLog(j, "The game closed %s after its first start since the update", time.Since(started).Round(time.Second))
				a.notify(j.Name+" closed right after starting", "It's the first start since the update. If it doesn't work, Roll back puts the old folder back, from the Updates page.")
				a.emitJobs()
			}
			started = time.Time{}
		case !up && !applied.IsZero() && time.Since(applied) > fullKeepDays*24*time.Hour:
			a.jobFinish(j, fmt.Sprintf("%d days passed", fullKeepDays))
			return
		}
	}
}

// recoverJobs runs at start (not in safe mode): a full update cut short is
// put right, downloads go on, and kept folders are watched again.
func (a *App) recoverJobs() {
	jobs.Lock()
	a.loadJobs()
	var list []*updJob
	for _, j := range jobs.m {
		list = append(list, j)
	}
	jobs.Unlock()
	for _, j := range list {
		if strings.HasPrefix(j.Mode, "delta") && j.Phase != "download" && j.Phase != "unpack" {
			a.deltaRecover(j)
			continue
		}
		switch j.Phase {
		case "apply":
			u, err := core.LoadFullUpdate(jobDir(j.Key))
			if err != nil || u == nil {
				a.setPhase(j, "failed", "The update's record couldn't be read. Look at the game's folder before you play.")
				continue
			}
			_, f, _ := a.jobTarget(j.Key)
			changed, err := core.RecoverFullUpdate(u, a.fullHooks(j, f))
			switch {
			case err != nil:
				a.jobLog(j, "Couldn't put it right: %v", err)
				a.setPhase(j, "failed", sentence(err))
				a.notify(j.Name+" needs a look", sentence(err))
			case u.State == core.FUStaged:
				a.jobLog(j, "Lootnest closed before anything moved; the update waits again")
				a.setPhase(j, "ready", "")
			case changed:
				if u.Aside != "" {
					_ = win.Recycle(a.hwnd, u.Aside)
				}
				a.setPhase(j, "failed", u.Error)
				a.notify(j.Name+"'s update was cut short", "The game is back as it was. Start the update again from the Updates page.")
			case u.State == core.FUKept:
				jobs.Lock()
				j.Old = u.Old
				jobs.Unlock()
				a.setPhase(j, "kept", "")
				go a.watchKept(j)
			}
		case "unpack":
			go a.jobUnpack(j)
		case "kept":
			go a.watchKept(j)
		}
	}
	a.jobsMoved()
}
