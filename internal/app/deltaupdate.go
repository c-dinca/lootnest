package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/plat"
)

// ---------------------------------------------------------------------------
// Updates that take only what changed (core/delta.go). A job whose download
// is one archive (.zip, .7z, or a .rar in parts) is read where it lies:
// Lootnest compares each of its files with the game's and keeps only the
// ones that differ, in <folder>.lootnest-new, never saving the archive.
// Applying moves files on the game's drive (the old ones into
// <folder>.lootnest-old), so it needs no room beyond the changed files.
// A server that won't send parts of a file gets the whole download first
// ("delta-local"), and the archive goes once it's read.
// ---------------------------------------------------------------------------

type deltaRun struct {
	cancel  context.CancelFunc
	pause   bool // stopped by Pause, not Cancel
	step    string
	done    int
	of      int
	fetched func() int64
	total   int64
	mark    time.Time
	markN   int64
	speed   int64
}

var deltas struct {
	sync.Mutex
	m map[string]*deltaRun // by job id
}

var rarPartRe = regexp.MustCompile(`(?i)\.part0*(\d+)\.rar$`)
var rarOldRe = regexp.MustCompile(`(?i)\.r(\d\d)$`)

// deltaFirst says whether a job's files are one archive Lootnest can read
// in place, and which file it starts from.
func deltaFirst(files []jobFile) (string, bool) {
	if len(files) == 1 {
		switch strings.ToLower(filepath.Ext(files[0].Name)) {
		case ".zip", ".7z", ".rar":
			return files[0].Name, true
		}
		return "", false
	}
	first, lowest := "", 1<<30
	for _, f := range files {
		n := strings.ToLower(f.Name)
		switch {
		case rarPartRe.MatchString(n):
			var k int
			fmt.Sscan(rarPartRe.FindStringSubmatch(n)[1], &k)
			if k < lowest {
				first, lowest = f.Name, k
			}
		case strings.HasSuffix(n, ".rar"):
			first, lowest = f.Name, -1 // the old naming: name.rar, name.r00, ...
		case rarOldRe.MatchString(n):
		default:
			return "", false
		}
	}
	return first, first != ""
}

// deltaOK: Lootnest reads the archive in place only over https; plain
// http downloads whole, to check the file's SHA-256 the source gave.
func deltaOK(files []jobFile) (string, bool) {
	for _, f := range files {
		if !strings.HasPrefix(f.URL, "https://") {
			return "", false
		}
	}
	return deltaFirst(files)
}

func (a *App) deltaState(id string) *deltaRun {
	deltas.Lock()
	defer deltas.Unlock()
	if deltas.m == nil {
		return nil
	}
	return deltas.m[id]
}

// runDelta reads a job's archive against the game and stages what changed.
// It goes on from what an earlier try staged.
func (a *App) runDelta(j *updJob) {
	ctx, cancel := context.WithCancel(context.Background())
	run := &deltaRun{cancel: cancel, step: "start", mark: time.Now()}
	deltas.Lock()
	if deltas.m == nil {
		deltas.m = map[string]*deltaRun{}
	}
	if old := deltas.m[j.ID]; old != nil {
		deltas.Unlock()
		cancel()
		return // already running
	}
	deltas.m[j.ID] = run
	deltas.Unlock()
	defer func() {
		cancel()
		deltas.Lock()
		delete(deltas.m, j.ID)
		deltas.Unlock()
		a.emitJobs()
	}()
	jobs.Lock()
	j.Paused, j.Error = false, ""
	saveJob(j)
	jobs.Unlock()
	a.emitJobs()

	t, _, err := a.jobTarget(j.Key)
	if err != nil {
		a.setPhase(j, "failed", sentence(err))
		return
	}
	rel, err := filepath.Rel(t.Dir, t.Exe)
	if err != nil || strings.HasPrefix(rel, "..") {
		rel = filepath.Base(t.Exe)
	}
	first, _ := deltaFirst(j.Files)
	client := core.GuardedClient(false)
	var src core.DeltaSource
	switch strings.ToLower(filepath.Ext(first)) {
	case ".zip", ".7z":
		url := j.Files[0].URL
		rf, err := core.OpenRemote(ctx, client, url)
		if errors.Is(err, core.ErrNoRanges) {
			a.jobLog(j, "The download's site sends only whole files, so it's downloaded first, then read")
			a.deltaWhole(j)
			return
		}
		if err != nil {
			a.deltaStopped(j, run, err)
			return
		}
		src = core.DeltaSource{Name: first, At: rf, Size: rf.Size()}
		run.fetched, run.total = rf.Fetched, rf.Size()
	default:
		urls := map[string]string{}
		var size int64
		for _, f := range j.Files {
			urls[f.Name] = f.URL
			size += f.Size
		}
		rv := &core.RemoteVolumes{Ctx: ctx, Client: client, URLs: urls}
		src = core.DeltaSource{Name: first, FS: rv}
		run.fetched, run.total = rv.Read, size
	}
	a.deltaStage(ctx, j, run, t, rel, src)
}

// deltaStage compares and stages, then makes the job ready or says why not.
func (a *App) deltaStage(ctx context.Context, j *updJob, run *deltaRun, t updTarget, exeRel string, src core.DeltaSource) {
	vol := strings.ToUpper(filepath.VolumeName(t.Dir))
	opts := core.DeltaOptions{
		ExeRel: exeRel,
		Room: func(need int64) error {
			if free := plat.DiskFree(filepath.Dir(t.Dir)); free >= 0 && free < need+(1<<30) {
				return fmt.Errorf("there isn't enough room on %s: the changed files need about %s and %s is free", vol, gb(need+(1<<30)), gb(free))
			}
			return nil
		},
		Progress: func(p core.DeltaProgress) {
			deltas.Lock()
			run.step, run.done, run.of = p.Step, p.Done, p.Of
			if run.fetched != nil {
				if el := time.Since(run.mark); el >= 2*time.Second {
					n := run.fetched()
					run.speed = int64(float64(n-run.markN) / el.Seconds())
					run.mark, run.markN = time.Now(), n
				}
			}
			deltas.Unlock()
			a.jobsTick()
		},
	}
	a.jobLog(j, "Comparing %s with the game's files, to take only what changed", src.Name)
	plan, err := core.StageDelta(ctx, src, t.Dir, j.Staging, opts)
	if err != nil {
		a.deltaStopped(j, run, err)
		return
	}
	var got int64
	if run.fetched != nil {
		got = run.fetched()
	}
	a.jobLog(j, "%d file(s) changed or new (%s), %d the same (%s); %s downloaded. %s",
		len(plan.Changed), gb(plan.Bytes), plan.Same, gb(plan.SameBytes), gb(got),
		map[bool]string{true: "It's the whole game.", false: "It's a patch: only its files go in."}[plan.Full])
	if len(plan.Changed) == 0 {
		_ = os.RemoveAll(j.Staging)
		a.setPhase(j, "failed", "Every file in it is already the same as the game's: nothing to update.")
		return
	}
	a.setPhase(j, "ready", "")
	a.notify(j.Name+" is ready to update", fmt.Sprintf("%d file(s) changed. Open the Updates page to look at it.", len(plan.Changed)))
}

// deltaStopped: a pause keeps what's staged; anything else fails the job.
func (a *App) deltaStopped(j *updJob, run *deltaRun, err error) {
	deltas.Lock()
	paused := run.pause
	deltas.Unlock()
	if errors.Is(err, context.Canceled) {
		if paused {
			jobs.Lock()
			j.Paused = true
			saveJob(j)
			jobs.Unlock()
			a.jobLog(j, "Paused; what's taken so far is kept")
		}
		return
	}
	a.jobLog(j, "Stopped: %v", err)
	jobs.Lock()
	j.Paused, j.Error = true, sentence(err)
	saveJob(j)
	jobs.Unlock()
	a.notify(j.Name+" couldn't be updated", sentence(err)+" Try again from the Updates page.")
}

var jobsTickAt struct {
	sync.Mutex
	t time.Time
}

// jobsTick tells the page, at most a few times a second.
func (a *App) jobsTick() {
	jobsTickAt.Lock()
	ok := time.Since(jobsTickAt.t) > 400*time.Millisecond
	if ok {
		jobsTickAt.t = time.Now()
	}
	jobsTickAt.Unlock()
	if ok {
		a.emitJobs()
	}
}

// deltaWhole downloads the whole archive first (a server that won't send
// parts of a file), then reads it from the disk.
func (a *App) deltaWhole(j *updJob) {
	jobs.Lock()
	j.Mode, j.Phase = "delta-local", "download"
	saveJob(j)
	q := jobs.q
	jobs.Unlock()
	list := make([]core.Download, len(j.Files))
	for i, x := range j.Files {
		list[i] = core.Download{ID: fmt.Sprintf("%s-%d", j.ID, i), Job: j.ID, URL: x.URL, Dst: filepath.Join(j.Staging, "dl", x.Name), Size: x.Size, SHA256: x.SHA256}
	}
	q.Add(list...)
	a.emitJobs()
}

// deltaFromDisk reads an archive the whole of which was downloaded, then
// deletes it.
func (a *App) deltaFromDisk(j *updJob) {
	t, _, err := a.jobTarget(j.Key)
	if err != nil {
		a.setPhase(j, "failed", sentence(err))
		return
	}
	rel, err := filepath.Rel(t.Dir, t.Exe)
	if err != nil || strings.HasPrefix(rel, "..") {
		rel = filepath.Base(t.Exe)
	}
	dl := filepath.Join(j.Staging, "dl")
	first, _ := deltaFirst(j.Files)
	src := core.DeltaSource{Name: first, FS: os.DirFS(dl)}
	if ext := strings.ToLower(filepath.Ext(first)); ext == ".zip" || ext == ".7z" {
		f, err := os.Open(filepath.Join(dl, first))
		if err != nil {
			a.setPhase(j, "failed", sentence(err))
			return
		}
		defer f.Close()
		st, _ := f.Stat()
		src = core.DeltaSource{Name: first, At: f, Size: st.Size()}
	}
	jobs.Lock()
	j.Phase = "fetch"
	saveJob(j)
	jobs.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	run := &deltaRun{cancel: cancel, mark: time.Now()}
	deltas.Lock()
	if deltas.m == nil {
		deltas.m = map[string]*deltaRun{}
	}
	deltas.m[j.ID] = run
	deltas.Unlock()
	defer func() {
		deltas.Lock()
		delete(deltas.m, j.ID)
		deltas.Unlock()
	}()
	a.deltaStage(ctx, j, run, t, rel, src)
	if f, ok := src.At.(*os.File); ok {
		f.Close()
	}
	_ = os.RemoveAll(dl) // the archive's read: its room comes back
}

// deltaPause, deltaResume and deltaCancel steer a job while it compares
// and fetches.
func (a *App) deltaPause(j *updJob) {
	if r := a.deltaState(j.ID); r != nil {
		deltas.Lock()
		r.pause = true
		deltas.Unlock()
		r.cancel()
	}
}

func (a *App) deltaResume(j *updJob) {
	if a.deltaState(j.ID) == nil {
		go a.runDelta(j)
	}
}

func (a *App) deltaCancel(j *updJob) {
	if r := a.deltaState(j.ID); r != nil {
		r.cancel()
	}
	for i := 0; i < 50 && a.deltaState(j.ID) != nil; i++ {
		time.Sleep(100 * time.Millisecond)
	}
	_ = os.RemoveAll(j.Staging)
}

// deltaView fills a job's progress while it compares and fetches.
func (a *App) deltaView(v *JobView) {
	r := a.deltaState(v.ID)
	if r == nil {
		v.Paused = true
		return
	}
	deltas.Lock()
	defer deltas.Unlock()
	v.Step = r.step
	v.Count, v.Of = r.done, r.of
	if r.fetched != nil {
		v.Done = r.fetched()
	}
	v.Total, v.Speed = r.total, r.speed
}

// deltaPlanFor is what the dialog before an update shows.
func (a *App) deltaPlanFor(j *updJob) (*JobPlan, error) {
	plan, err := core.LoadDeltaPlan(j.Staging)
	if err != nil {
		return nil, errors.New("That update isn't waiting to be applied.")
	}
	_, f, err := a.jobTarget(j.Key)
	if err != nil {
		return nil, errors.New(sentence(err))
	}
	locs, inside := a.jobSaves(f, j.Dir)
	extras, err := core.DeltaExtras(j.Dir, j.Staging, plan)
	if err != nil {
		return nil, errors.New(sentence(err))
	}
	var shown []core.KeepItem
	for _, e := range extras {
		if !underAny(e.Rel, inside) {
			shown = append(shown, e)
		}
	}
	keep := []core.KeepItem{}
	for _, s := range inside {
		keep = append(keep, core.KeepItem{Rel: s, Why: "saves", Size: core.Size(filepath.Join(j.Dir, filepath.FromSlash(s)))})
	}
	yours, remembered := loadKeptPaths()[j.Key]
	if !remembered {
		for _, e := range shown {
			yours = append(yours, e.Rel) // all kept unless the user says otherwise
		}
	}
	removed, _ := core.DeltaRemovals(j.Dir, j.Staging, plan, append(append([]string{}, inside...), relsOf(shown)...))
	p := &JobPlan{FullPlan: &core.FullPlan{Keep: keep, Extra: shown}, Yours: yours, Saves: len(locs), Folder: j.Dir, Old: j.Dir + ".lootnest-old", Days: fullKeepDays,
		Delta: true, Full: plan.Full, Same: plan.Same, Removed: len(removed), Bytes: plan.Bytes}
	if p.Extra == nil {
		p.Extra = []core.KeepItem{}
	}
	for _, c := range plan.Changed {
		if c.New {
			p.Added++
		} else {
			p.Changed++
		}
	}
	return p, nil
}

func relsOf(items []core.KeepItem) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Rel
	}
	return out
}

func underAny(rel string, roots []string) bool {
	lr := strings.ToLower(rel)
	for _, r := range roots {
		lrr := strings.ToLower(r)
		if lr == lrr || strings.HasPrefix(lrr, lr+"/") || strings.HasPrefix(lr, lrr+"/") {
			return true
		}
	}
	return false
}

// deltaApply moves the staged files in, after the user said yes. keep is
// what they ticked to keep of what only their folder has.
func (a *App) deltaApply(j *updJob, keep []string) {
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
	plan, err := core.LoadDeltaPlan(j.Staging)
	if err != nil {
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
	kept := loadKeptPaths()
	kept[j.Key] = keep
	_ = os.MkdirAll(jobsRoot(), 0o755)
	if b, err := json.MarshalIndent(kept, "", "  "); err == nil {
		_ = os.WriteFile(filepath.Join(jobsRoot(), "keep.json"), b, 0o644)
	}
	jobs.Lock()
	j.Phase, j.Error, j.Step = "apply", "", "Copying your saves"
	saveJob(j)
	jobs.Unlock()
	a.emitJobs()
	name := "Before the update"
	if j.Version != "" {
		name += " to " + j.Version
	}
	if id, err := a.copySaves(f, name); err != nil {
		a.setPhase(j, "ready", "")
		fail(errors.New("nothing was changed: the saves couldn't be copied first (" + err.Error() + ")"))
		return
	} else if id != "" {
		a.jobLog(j, "Saves copied (%s)", id)
	}
	_, inside := a.jobSaves(f, j.Dir)
	remove, err := core.DeltaRemovals(j.Dir, j.Staging, plan, append(inside, keep...))
	if err != nil {
		a.setPhase(j, "ready", "")
		fail(err)
		return
	}
	exe := plan.NewExe
	if exe == "" {
		if r, err := filepath.Rel(t.Dir, t.Exe); err == nil && !strings.HasPrefix(r, "..") {
			exe = filepath.ToSlash(r)
		}
	}
	jobs.Lock()
	j.Step = "Replacing the changed files"
	saveJob(j)
	jobs.Unlock()
	a.emitJobs()
	keepDir := j.Dir + ".lootnest-old"
	a.jobLog(j, "Applying: %d file(s) in, %d moved aside to %s", len(plan.Changed), len(remove), keepDir)
	rec, err := core.ApplyDelta(j.Dir, j.Staging, keepDir, exe, plan, remove, func(d *core.DeltaApplied) error { return core.SaveDeltaApplied(jobDir(j.Key), d) })
	if err != nil {
		state := "applying"
		if rec != nil {
			state = rec.State
		}
		a.jobLog(j, "Not applied (%s): %v", state, err)
		if state == "undone" {
			a.setPhase(j, "ready", "")
		} else {
			a.setPhase(j, "failed", sentence(err))
		}
		a.notify(j.Name+" wasn't updated", sentence(err))
		a.emit("toast", map[string]any{"kind": "error", "text": sentence(err)})
		return
	}
	jobs.Lock()
	j.Old, j.Applied = keepDir, time.Now()
	jobs.Unlock()
	_ = os.RemoveAll(j.Staging) // everything staged is in place now
	a.jobLog(j, "Done. The old versions of %d file(s) are kept in %s", len(rec.Replaced)+len(rec.Removed), keepDir)
	a.setPhase(j, "kept", "")
	a.notify(j.Name+" is updated", "The old files are kept until you've played, or "+fmt.Sprint(fullKeepDays)+" days. Roll back from the Updates page.")
	a.startScan()
	go a.watchKept(j)
}

// deltaRollback puts the old files back, after copying the saves.
func (a *App) deltaRollback(j *updJob) {
	t, f, err := a.jobTarget(j.Key)
	if err == nil && t.running() {
		err = fmt.Errorf("%s is running; close it first", t.Name)
	}
	rec, lerr := core.LoadDeltaApplied(jobDir(j.Key))
	if err == nil && lerr != nil {
		err = errors.New("that update's record is gone")
	}
	if err != nil {
		a.emit("toast", map[string]any{"kind": "error", "text": sentence(err)})
		return
	}
	updMu.Lock()
	defer updMu.Unlock()
	if _, err := a.copySaves(f, "Before rolling the update back"); err != nil {
		a.emit("toast", map[string]any{"kind": "error", "text": "Nothing was changed: the saves couldn't be copied first (" + err.Error() + ")."})
		return
	}
	if err := core.RevertDelta(rec, func(d *core.DeltaApplied) error { return core.SaveDeltaApplied(jobDir(j.Key), d) }); err != nil {
		a.jobLog(j, "Not rolled back: %v", err)
		a.emit("toast", map[string]any{"kind": "error", "text": sentence(err)})
		return
	}
	if err := plat.Recycle(a.hwnd, j.Staging); err != nil {
		a.jobLog(j, "The new files stay in %s: %v", j.Staging, err)
	}
	jobs.Lock()
	j.Old = ""
	jobs.Unlock()
	a.jobLog(j, "Rolled back. A copy of the saves as they were is among the game's copies")
	a.setPhase(j, "rolledback", "")
	a.startScan()
}

// deltaFinish lets the kept old files go, to the Recycle Bin.
func (a *App) deltaFinish(j *updJob, why string) {
	rec, err := core.LoadDeltaApplied(jobDir(j.Key))
	if err != nil || rec.State != "applied" {
		return
	}
	updMu.Lock()
	defer updMu.Unlock()
	if _, err := os.Lstat(rec.Keep); err == nil {
		if err := plat.Recycle(a.hwnd, rec.Keep); err != nil {
			a.jobLog(j, "The old files stay in %s: %v", rec.Keep, err)
		} else {
			a.jobLog(j, "The old files went to the Recycle Bin (%s)", why)
		}
	}
	rec.State = "finished"
	_ = core.SaveDeltaApplied(jobDir(j.Key), rec)
	jobs.Lock()
	j.Old = ""
	jobs.Unlock()
	a.setPhase(j, "done", "")
}

// deltaRecover puts right a delta job Lootnest closed in the middle of.
func (a *App) deltaRecover(j *updJob) {
	switch j.Phase {
	case "fetch":
		if !j.Paused {
			go a.runDelta(j) // goes on from what's staged
		}
	case "apply":
		rec, err := core.LoadDeltaApplied(jobDir(j.Key))
		if err != nil {
			a.setPhase(j, "ready", "") // nothing was moved yet
			return
		}
		switch rec.State {
		case "applying":
			a.jobLog(j, "Lootnest closed while files were being moved; putting them back")
			if err := core.RevertDelta(rec, func(d *core.DeltaApplied) error { return core.SaveDeltaApplied(jobDir(j.Key), d) }); err != nil {
				a.setPhase(j, "failed", "An update was cut short and some files couldn't be put back: "+sentence(err)+" Look at the game's folder before you play.")
				a.notify(j.Name+" needs a look", sentence(err))
				return
			}
			a.setPhase(j, "failed", "The update was cut short. The game is back as it was.")
			a.notify(j.Name+"'s update was cut short", "The game is back as it was. Start the update again from the Updates page.")
		case "applied":
			jobs.Lock()
			j.Old = rec.Keep
			jobs.Unlock()
			a.setPhase(j, "kept", "")
			go a.watchKept(j)
		default:
			a.setPhase(j, "ready", "")
		}
	case "kept":
		go a.watchKept(j)
	}
}
