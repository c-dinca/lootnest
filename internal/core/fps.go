package core

import (
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// ---------------------------------------------------------------------------
// The FPS counter. Windows tells anyone allowed to listen (Event Tracing for
// Windows) each time a program hands a frame to the screen, through DXGI
// (Direct3D 10 to 12), Direct3D 9 or the graphics kernel (Vulkan, OpenGL and
// the rest). Lootnest counts those for the program in front; nothing is put
// into the game and nothing of it is read. win/etw.go listens, FrameCounter
// counts, win/fpsbox.go shows the number.
// ---------------------------------------------------------------------------

// Frame sources: where a frame was heard. A Direct3D 11 game is heard by
// DXGI and by the kernel for the same frame, so each source is counted on
// its own and the counter takes the largest, never the sum.
const (
	FrameDXGI = iota
	FrameD3D9
	FrameKernel  // DxgKrnl Present_Info
	FrameHistory // DxgKrnl PresentHistory_Start, which PresentMon always reads
	frameSources
)

// FrameCounter keeps the last two seconds of frames per process. Times are
// the events' own (QueryPerformanceCounter ticks), since Windows delivers
// them in batches about a second late: the rate is the second before the
// newest frame heard, not before now.
type FrameCounter struct {
	mu    sync.Mutex
	freq  int64 // ticks a second
	procs map[uint32]*frameProc
}

type frameProc struct {
	at   [frameSources][]int64
	last int64
}

// NewFrameCounter counts with a clock of freq ticks a second.
func NewFrameCounter(freq int64) *FrameCounter {
	if freq <= 0 {
		freq = 1
	}
	return &FrameCounter{freq: freq, procs: map[uint32]*frameProc{}}
}

// Add notes one frame of process pid from source src at tick at.
func (c *FrameCounter) Add(pid uint32, src int, at int64) {
	if src < 0 || src >= frameSources || pid == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	p := c.procs[pid]
	if p == nil {
		if len(c.procs) > 256 { // programs that stopped drawing long ago go
			c.pruneLocked(at)
		}
		p = &frameProc{}
		c.procs[pid] = p
	}
	if at > p.last {
		p.last = at
	}
	list := append(p.at[src], at)
	// Keep two seconds. Events from different processors can arrive a
	// little out of order, so this trims by time, not by position alone.
	cut := p.last - 2*c.freq
	i := 0
	for i < len(list) && list[i] < cut {
		i++
	}
	if i > 0 {
		list = append(list[:0], list[i:]...)
	}
	if len(list) > 4096 { // nothing draws 2000 frames a second; a bound all the same
		list = append(list[:0], list[len(list)-4096:]...)
	}
	p.at[src] = list
}

func (c *FrameCounter) pruneLocked(now int64) {
	for pid, p := range c.procs {
		if p.last < now-5*c.freq {
			delete(c.procs, pid)
		}
	}
}

// Rate is pid's frames in the second before its newest frame. ok is false
// when it drew nothing in the 3 seconds before now.
func (c *FrameCounter) Rate(pid uint32, now int64) (fps int, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	p := c.procs[pid]
	if p == nil || p.last == 0 || p.last < now-3*c.freq {
		return 0, false
	}
	from := p.last - c.freq
	for _, list := range p.at {
		n := 0
		for _, t := range list {
			if t > from && t <= p.last {
				n++
			}
		}
		if n > fps {
			fps = n
		}
	}
	return fps, true
}

// Rates is the rate of every process that drew in the 3 seconds before
// now, by pid.
func (c *FrameCounter) Rates(now int64) map[uint32]int {
	c.mu.Lock()
	pids := make([]uint32, 0, len(c.procs))
	for pid := range c.procs {
		pids = append(pids, pid)
	}
	c.mu.Unlock()
	out := map[uint32]int{}
	for _, pid := range pids {
		if fps, ok := c.Rate(pid, now); ok {
			out[pid] = fps
		}
	}
	return out
}

// Forget drops everything counted (the counter was turned off).
func (c *FrameCounter) Forget() {
	c.mu.Lock()
	c.procs = map[uint32]*frameProc{}
	c.mu.Unlock()
}

// FPS corners: where the counter sits on the game's screen.
var FPSCorners = []string{"tl", "tr", "bl", "br"}

// CleanFPSCorner keeps a known corner, else top left.
func CleanFPSCorner(c string) string {
	for _, k := range FPSCorners {
		if c == k {
			return c
		}
	}
	return "tl"
}

// ---------------------------------------------------------------------------
// The frame reader: Windows lets only administrators and Performance Log
// Users listen to frames, and a new member counts only from the next
// sign-in. So the user allows, once with Windows' prompt, a task of
// Task Scheduler that starts Lootnest's own frame reader with the highest
// rights the account has. Lootnest starts it on demand (no further
// prompt); the reader writes each process's rate twice a second to a pipe
// and ends when nobody reads it. Its program is a copy of lootnest.exe in
// Program Files, which only administrators can change, so the task can't
// be used to run something else with those rights.
// ---------------------------------------------------------------------------

// FrameTaskPath and FrameTaskName name the task in Task Scheduler.
const (
	FrameTaskPath = `\Lootnest\`
	FrameTaskName = "Frame counter"
	FrameTaskArg  = "--frames"
	// FrameReaderDir is under %ProgramFiles%.
	FrameReaderDir = `Lootnest\Frame counter`
	FrameReaderExe = "lootnest-frames.exe"
	// FrameProtocol is what reader and app agree on; a reader saying
	// another is from an older Lootnest, and Allow puts the new one in.
	FrameProtocol = 1
)

var sidRe = regexp.MustCompile(`^S-1-[0-9]+(-[0-9]+)+$`)

// FramePipeName is the reader's pipe for one account.
func FramePipeName(userSID string) string {
	return `\\.\pipe\lootnest-frames-` + userSID
}

func psQuoted(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// FrameTaskScript is the PowerShell, run once as administrator, that puts
// the reader in Program Files (a copy of exe) and registers the task for
// the account userSID, run only while that account is signed in, with its
// highest rights. Running it again replaces both.
func FrameTaskScript(exe, userSID string) (string, error) {
	if !sidRe.MatchString(userSID) {
		return "", errors.New("Windows didn't say who you are, so nothing was changed.")
	}
	if exe == "" || strings.ContainsAny(exe, "\r\n") {
		return "", errors.New("Lootnest couldn't find its own program, so nothing was changed.")
	}
	return `$ErrorActionPreference = 'Stop'
$dir = Join-Path $env:ProgramFiles ` + psQuoted(FrameReaderDir) + `
New-Item -ItemType Directory -Force -Path $dir | Out-Null
$reader = Join-Path $dir ` + psQuoted(FrameReaderExe) + `
Copy-Item -LiteralPath ` + psQuoted(exe) + ` -Destination $reader -Force
$user = (New-Object System.Security.Principal.SecurityIdentifier(` + psQuoted(userSID) + `)).Translate([System.Security.Principal.NTAccount]).Value
$action = New-ScheduledTaskAction -Execute $reader -Argument ` + psQuoted(FrameTaskArg) + `
$principal = New-ScheduledTaskPrincipal -UserId $user -LogonType Interactive -RunLevel Highest
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -ExecutionTimeLimit ([TimeSpan]::Zero) -MultipleInstances IgnoreNew
Register-ScheduledTask -TaskPath ` + psQuoted(FrameTaskPath) + ` -TaskName ` + psQuoted(FrameTaskName) + ` -Action $action -Principal $principal -Settings $settings -Force | Out-Null
exit 0
`, nil
}

// FrameUntaskScript takes the task and the reader away again.
func FrameUntaskScript() string {
	return `$ErrorActionPreference = 'SilentlyContinue'
Unregister-ScheduledTask -TaskPath ` + psQuoted(FrameTaskPath) + ` -TaskName ` + psQuoted(FrameTaskName) + ` -Confirm:$false
$dir = Join-Path $env:ProgramFiles ` + psQuoted(FrameReaderDir) + `
Remove-Item -LiteralPath $dir -Recurse -Force
exit 0
`
}

// FrameReport is one line the reader writes: each process's rate.
type FrameReport struct {
	V   int            `json:"v"`
	FPS map[string]int `json:"fps,omitempty"`
	Err string         `json:"err,omitempty"` // the reader couldn't listen
}

// EncodeFrameReport is one line of the pipe.
func EncodeFrameReport(rates map[uint32]int, errText string) []byte {
	r := FrameReport{V: FrameProtocol, Err: errText}
	if len(rates) > 0 {
		r.FPS = make(map[string]int, len(rates))
		for pid, n := range rates {
			r.FPS[strconv.FormatUint(uint64(pid), 10)] = n
		}
	}
	b, _ := json.Marshal(r)
	return append(b, '\n')
}

// DecodeFrameReport reads one line; a line from another protocol, or that
// isn't one, is an error.
func DecodeFrameReport(line []byte) (map[uint32]int, string, error) {
	var r FrameReport
	if err := json.Unmarshal(line, &r); err != nil {
		return nil, "", errors.New("the frame reader said something Lootnest can't read")
	}
	if r.V != FrameProtocol {
		return nil, "", ErrFrameReaderOld
	}
	out := make(map[uint32]int, len(r.FPS))
	for k, n := range r.FPS {
		pid, err := strconv.ParseUint(k, 10, 32)
		if err != nil || n < 0 || n > 100000 {
			continue
		}
		out[uint32(pid)] = n
	}
	return out, r.Err, nil
}

// ErrFrameReaderOld: the reader in Program Files is from another version.
var ErrFrameReaderOld = errors.New("the frame reader is from another version of Lootnest")
