package core

import (
	"errors"
	"regexp"
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
	FrameKernel
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

// PerfLogUsersSID is the built-in Performance Log Users group, the same in
// every language of Windows. Its members may listen to Windows' frame
// events without being administrators, as Xbox Game Bar's counter does.
const PerfLogUsersSID = "S-1-5-32-559"

var sidRe = regexp.MustCompile(`^S-1-[0-9]+(-[0-9]+)+$`)

// PerfLogUsersScript is the PowerShell, run once as administrator, that
// puts the user (by SID) into Performance Log Users. Being a member
// already is not a failure. Add-LocalGroupMember first; the older ADSI way
// where that module is missing, with the group's name in this Windows'
// language.
func PerfLogUsersScript(userSID string) (string, error) {
	if !sidRe.MatchString(userSID) {
		return "", errors.New("Windows didn't say who you are, so nothing was changed.")
	}
	return `$ErrorActionPreference = 'Stop'
$group = New-Object System.Security.Principal.SecurityIdentifier('` + PerfLogUsersSID + `')
$user = '` + userSID + `'
try {
  Add-LocalGroupMember -SID $group -Member $user
} catch {
  if ($_.FullyQualifiedErrorId -like 'MemberExists*') { exit 0 }
  try {
    $name = $group.Translate([System.Security.Principal.NTAccount]).Value.Split('\')[-1]
    ([ADSI]('WinNT://./' + $name + ',group')).Add('WinNT://' + $user)
  } catch {
    $inner = $_.Exception.InnerException
    if ($inner -and $inner.ErrorCode -eq -2147023518) { exit 0 }
    exit 2
  }
}
exit 0
`, nil
}
