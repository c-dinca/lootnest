package app

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"lootnest/internal/core"
	"lootnest/internal/plat"
)

// ---------------------------------------------------------------------------
// Where the FPS counter's frames come from (core/fps.go explains the
// reader). Lootnest listens itself when Windows lets this sign-in do it
// (an administrator's full rights, or Performance Log Users); otherwise it
// starts the frame reader through its task, which the user allowed once,
// and reads its pipe.
// ---------------------------------------------------------------------------

// frameSource gives the rate of a process, and stops.
type frameSource interface {
	Rate(pid uint32) (int, bool)
	Stop()
	Kind() string
}

// Lootnest listening itself.
type localFrames struct {
	count *core.FrameCounter
	tr    *plat.FrameTrace
}

func (l *localFrames) Rate(pid uint32) (int, bool) { return l.count.Rate(pid, plat.PerfNow()) }
func (l *localFrames) Stop()                       { l.tr.Stop() }
func (l *localFrames) Kind() string                { return "Lootnest" }

// The frame reader's pipe.
type readerFrames struct {
	f     *os.File
	mu    sync.Mutex
	rates map[uint32]int
	at    time.Time
	done  chan struct{}
}

func (r *readerFrames) Rate(pid uint32) (int, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if time.Since(r.at) > 3*time.Second {
		return 0, false
	}
	n, ok := r.rates[pid]
	return n, ok
}

func (r *readerFrames) Stop() {
	r.f.Close() // the reader's next write fails, and it ends
	<-r.done
}

func (r *readerFrames) Kind() string { return "the frame reader" }

var errReaderGone = errors.New("the frame reader stopped")

func (a *App) startFrames() (frameSource, error) {
	if plat.FramesAllowed() {
		count := core.NewFrameCounter(plat.PerfFreq())
		tr, err := plat.StartFrameTrace(count.Add)
		if err == nil {
			return &localFrames{count, tr}, nil
		}
		if !errors.Is(err, plat.ErrFramesNotAllowed) {
			return nil, err
		}
	}
	if !a.frameTaskThere(false) {
		return nil, plat.ErrFramesNotAllowed
	}
	return a.startReader()
}

func schtasksExe() string {
	return filepath.Join(os.Getenv("SystemRoot"), "System32", "schtasks.exe")
}

func frameTaskArg() string { return `"` + core.FrameTaskPath + core.FrameTaskName + `"` }

// frameReaderPath is the reader's program in Program Files.
func frameReaderPath() string {
	pf := os.Getenv("ProgramFiles")
	if pf == "" {
		return ""
	}
	return filepath.Join(pf, core.FrameReaderDir, core.FrameReaderExe)
}

// frameTaskThere says whether the user allowed the reader: its task is
// registered and its program in place. Asked of Windows at most every
// 30 seconds unless fresh.
func (a *App) frameTaskThere(fresh bool) bool {
	fpsRun.mu.Lock()
	if !fresh && time.Since(fpsRun.taskAt) < 30*time.Second {
		there := fpsRun.task
		fpsRun.mu.Unlock()
		return there
	}
	fpsRun.mu.Unlock()
	there := false
	if p := frameReaderPath(); p != "" {
		if _, err := os.Stat(p); err == nil {
			_, err = plat.RunHiddenOutput(schtasksExe(), "/query /tn "+frameTaskArg(), 10*time.Second)
			there = err == nil
		}
	}
	fpsRun.mu.Lock()
	fpsRun.task, fpsRun.taskAt = there, time.Now()
	fpsRun.mu.Unlock()
	return there
}

func (a *App) startReader() (frameSource, error) {
	if out, err := plat.RunHiddenOutput(schtasksExe(), "/run /tn "+frameTaskArg(), 15*time.Second); err != nil {
		a.logf("FPS counter: Task Scheduler didn't start the frame reader: %v %s", err, out)
		return nil, errors.New("Task Scheduler didn't start Lootnest's frame reader")
	}
	name := core.FramePipeName(plat.UserSID())
	var f *os.File
	var err error
	for i := 0; i < 40; i++ { // the task takes a moment to start
		if f, err = os.OpenFile(name, os.O_RDONLY, 0); err == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if f == nil {
		return nil, errors.New("Lootnest's frame reader didn't answer")
	}
	r := &readerFrames{f: f, done: make(chan struct{})}
	first := make(chan error, 1)
	go func() {
		defer close(r.done)
		sc := bufio.NewScanner(f)
		said := false
		for sc.Scan() {
			rates, readerErr, err := core.DecodeFrameReport(sc.Bytes())
			if err == nil && readerErr != "" {
				err = errors.New(readerErr)
			}
			if !said {
				said = true
				first <- err
			}
			if err != nil {
				return
			}
			r.mu.Lock()
			r.rates, r.at = rates, time.Now()
			r.mu.Unlock()
		}
		if !said {
			first <- errReaderGone
		}
	}()
	select {
	case err := <-first:
		if err != nil {
			f.Close()
			<-r.done
			if errors.Is(err, core.ErrFrameReaderOld) {
				fpsRun.mu.Lock()
				fpsRun.old = true
				fpsRun.mu.Unlock()
			}
			return nil, err
		}
	case <-time.After(5 * time.Second):
		f.Close()
		<-r.done
		return nil, errors.New("Lootnest's frame reader said nothing")
	}
	return r, nil
}

// RunFrameReader is `lootnest-frames.exe --frames`, started by the task
// with the account's highest rights: it listens to frames and writes every
// process's rate to the account's pipe twice a second, and ends when
// nobody reads it, or when nobody opens the pipe within 20 seconds. It
// reads nothing from anyone.
func RunFrameReader() int {
	pipe, err := plat.ServeOutboundPipe(core.FramePipeName(plat.UserSID()))
	if err != nil {
		return 1
	}
	defer pipe.Close()
	opened := make(chan struct{})
	go func() {
		select {
		case <-opened:
		case <-time.After(20 * time.Second):
			os.Exit(2)
		}
	}()
	if err := plat.WaitPipeReader(pipe); err != nil {
		return 1
	}
	close(opened)
	count := core.NewFrameCounter(plat.PerfFreq())
	tr, err := plat.StartFrameTrace(count.Add)
	if err != nil {
		pipe.Write(core.EncodeFrameReport(nil, sentence(err)))
		return 1
	}
	defer tr.Stop()
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
	for range t.C {
		if _, err := pipe.Write(core.EncodeFrameReport(count.Rates(plat.PerfNow()), "")); err != nil {
			return 0
		}
	}
	return 0
}
