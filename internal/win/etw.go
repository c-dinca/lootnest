//go:build windows

package win

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ---------------------------------------------------------------------------
// Frames as Windows reports them (Event Tracing for Windows), for the FPS
// counter: a real-time session of Lootnest's own that hears DXGI's and
// Direct3D 9's Present calls and the graphics kernel's presents, each with
// the process that made it and when. Nothing is opened in any game. Starting
// the session takes an administrator or a member of Performance Log Users
// (ErrFramesNotAllowed otherwise), as Xbox Game Bar's counter does.
// ---------------------------------------------------------------------------

var (
	advapi32           = windows.NewLazySystemDLL("advapi32.dll")
	procStartTraceW    = advapi32.NewProc("StartTraceW")
	procControlTraceW  = advapi32.NewProc("ControlTraceW")
	procEnableTraceEx2 = advapi32.NewProc("EnableTraceEx2")
	procOpenTraceW     = advapi32.NewProc("OpenTraceW")
	procProcessTrace   = advapi32.NewProc("ProcessTrace")
	procCloseTrace     = advapi32.NewProc("CloseTrace")
	procQPC            = kernel32.NewProc("QueryPerformanceCounter")
	procQPF            = kernel32.NewProc("QueryPerformanceFrequency")
)

// ErrFramesNotAllowed: Windows refused the frame session; this account is
// neither an administrator here nor in Performance Log Users (yet: a new
// member counts from the next sign-in).
var ErrFramesNotAllowed = errors.New("Windows doesn't let this account read frame timings yet")

// Frame sources, as core.FrameDXGI, FrameD3D9 and FrameKernel.
const (
	FrameDXGI   = 0
	FrameD3D9   = 1
	FrameKernel = 2
)

// The providers and the one or two events of each that are a frame.
var frameProviders = []struct {
	guid windows.GUID
	ids  []uint16
	src  int
}{
	// Microsoft-Windows-DXGI: Present_Start, PresentMultiplaneOverlay_Start.
	{windows.GUID{Data1: 0xCA11C036, Data2: 0x0102, Data3: 0x4A2D, Data4: [8]byte{0xA6, 0xAD, 0xF0, 0x3C, 0xFE, 0xD5, 0xD3, 0xC9}}, []uint16{42, 55}, FrameDXGI},
	// Microsoft-Windows-D3D9: Present_Start.
	{windows.GUID{Data1: 0x783ACA0A, Data2: 0x790E, Data3: 0x4D7F, Data4: [8]byte{0x84, 0x51, 0xAA, 0x85, 0x05, 0x11, 0xC6, 0xB9}}, []uint16{1}, FrameD3D9},
	// Microsoft-Windows-DxgKrnl: Present_Info (every present that reaches
	// the kernel, Vulkan and OpenGL included).
	{windows.GUID{Data1: 0x802EC45A, Data2: 0x1E99, Data3: 0x4B83, Data4: [8]byte{0x99, 0x20, 0x87, 0xC9, 0x82, 0x77, 0xBA, 0x9D}}, []uint16{184}, FrameKernel},
}

// The session's name and an id of its own.
const frameSession = "Lootnest frame counter"

var frameSessionGUID = windows.GUID{Data1: 0x4C4E4650, Data2: 0x5353, Data3: 0x4C4E, Data4: [8]byte{0x8F, 0x2A, 0x16, 0x3D, 0x5C, 0x71, 0x0B, 0xE4}}

// EVENT_TRACE_PROPERTIES (with its WNODE_HEADER), 120 bytes on 64-bit.
type eventTraceProperties struct {
	BufferSize        uint32
	ProviderID        uint32
	HistoricalContext uint64
	TimeStamp         int64
	GUID              windows.GUID
	ClientContext     uint32
	Flags             uint32

	BufferSizeKB        uint32
	MinimumBuffers      uint32
	MaximumBuffers      uint32
	MaximumFileSize     uint32
	LogFileMode         uint32
	FlushTimer          uint32
	EnableFlags         uint32
	AgeLimit            int32
	NumberOfBuffers     uint32
	FreeBuffers         uint32
	EventsLost          uint32
	BuffersWritten      uint32
	LogBuffersLost      uint32
	RealTimeBuffersLost uint32
	LoggerThreadID      uintptr
	LogFileNameOffset   uint32
	LoggerNameOffset    uint32
}

// EVENT_TRACE_LOGFILEW, 448 bytes on 64-bit; only a few fields are used.
type eventTraceLogfile struct {
	LogFileName         *uint16
	LoggerName          *uint16
	CurrentTime         int64
	BuffersRead         uint32
	ProcessTraceMode    uint32
	CurrentEvent        [88]byte  // EVENT_TRACE
	LogfileHeader       [280]byte // TRACE_LOGFILE_HEADER
	BufferCallback      uintptr
	BufferSize          uint32
	Filled              uint32
	EventsLost          uint32
	EventRecordCallback uintptr
	IsKernelTrace       uint32
	Context             uintptr
}

// EVENT_HEADER, the start of EVENT_RECORD: 80 bytes.
type eventHeader struct {
	Size          uint16
	HeaderType    uint16
	Flags         uint16
	EventProperty uint16
	ThreadID      uint32
	ProcessID     uint32
	TimeStamp     int64
	ProviderID    windows.GUID
	ID            uint16
	Version       uint8
	Channel       uint8
	Level         uint8
	Opcode        uint8
	Task          uint16
	Keyword       uint64
	ProcessorTime uint64
	ActivityID    windows.GUID
}

type eventFilterDescriptor struct {
	Ptr  uint64
	Size uint32
	Type uint32
}

type enableTraceParameters struct {
	Version          uint32
	EnableProperty   uint32
	ControlFlags     uint32
	SourceID         windows.GUID
	EnableFilterDesc *eventFilterDescriptor
	FilterDescCount  uint32
}

const (
	wnodeFlagTracedGUID          = 0x00020000
	eventTraceRealTimeMode       = 0x00000100
	processTraceModeRealTime     = 0x00000100
	processTraceModeEventRecord  = 0x10000000
	eventTraceControlStop        = 1
	eventControlCodeEnable       = 1
	traceLevelVerbose            = 5
	eventFilterTypeEventID       = 0x80000200
	enableTraceParametersVersion = 2
	errAlreadyExists             = 183
	errAccessDenied              = 5
)

// newTraceProperties is EVENT_TRACE_PROPERTIES with room for the session's
// name after it, as StartTrace and ControlTrace want.
func newTraceProperties() *eventTraceProperties {
	size := unsafe.Sizeof(eventTraceProperties{}) + 2*(uintptr(len(frameSession))+1) + 64
	buf := make([]uint64, (size+7)/8) // 8-byte aligned
	p := (*eventTraceProperties)(unsafe.Pointer(&buf[0]))
	p.BufferSize = uint32(size)
	p.Flags = wnodeFlagTracedGUID
	p.GUID = frameSessionGUID
	p.ClientContext = 1 // times as QueryPerformanceCounter ticks
	p.LoggerNameOffset = uint32(unsafe.Sizeof(eventTraceProperties{}))
	return p
}

func stopFrameSession(name *uint16) {
	procControlTraceW.Call(0, uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(newTraceProperties())), eventTraceControlStop)
}

// The callback ProcessTrace calls for every event: one for the process,
// passed the current sink (a callback can't be made per session).
var (
	frameSink atomic.Pointer[func(pid uint32, src int, at int64)]
	frameCB   = windows.NewCallback(func(rec *eventHeader) uintptr {
		sink := frameSink.Load()
		if sink == nil || rec == nil {
			return 0
		}
		for i := range frameProviders {
			p := &frameProviders[i]
			if rec.ProviderID != p.guid {
				continue
			}
			for _, id := range p.ids {
				if rec.ID == id {
					(*sink)(rec.ProcessID, p.src, rec.TimeStamp)
					return 0
				}
			}
			return 0
		}
		return 0
	})
)

// FrameTrace is a running frame session.
type FrameTrace struct {
	mu      sync.Mutex
	session uint64
	reader  uint64
	done    chan struct{}
}

var frameTraceOne sync.Mutex // one session at a time

// StartFrameTrace starts the session and calls sink, on the session's own
// thread, for every frame: the process, the source (FrameDXGI...) and the
// frame's QueryPerformanceCounter tick. Keep sink short.
func StartFrameTrace(sink func(pid uint32, src int, at int64)) (*FrameTrace, error) {
	if !frameTraceOne.TryLock() {
		return nil, errors.New("the frame counter is already listening")
	}
	ok := false
	defer func() {
		if !ok {
			frameTraceOne.Unlock()
		}
	}()
	name, _ := windows.UTF16PtrFromString(frameSession)
	var session uint64
	start := func() uintptr {
		p := newTraceProperties()
		p.LogFileMode = eventTraceRealTimeMode
		p.BufferSizeKB = 16
		p.MinimumBuffers = 4
		p.MaximumBuffers = 16
		p.FlushTimer = 1 // seconds: frames arrive at most about a second late
		r, _, _ := procStartTraceW.Call(uintptr(unsafe.Pointer(&session)), uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(p)))
		return r
	}
	r := start()
	if r == errAlreadyExists { // left by a Lootnest that didn't close
		stopFrameSession(name)
		r = start()
	}
	switch r {
	case 0:
	case errAccessDenied:
		return nil, ErrFramesNotAllowed
	default:
		return nil, windows.Errno(r)
	}
	enabled := 0
	var firstErr error
	for _, p := range frameProviders {
		if err := enableFrameProvider(session, p.guid, p.ids); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		enabled++
	}
	if enabled == 0 {
		stopFrameSession(name)
		if firstErr == windows.ERROR_ACCESS_DENIED {
			return nil, ErrFramesNotAllowed
		}
		return nil, firstErr
	}
	frameSink.Store(&sink)
	lf := eventTraceLogfile{LoggerName: name, ProcessTraceMode: processTraceModeRealTime | processTraceModeEventRecord, EventRecordCallback: frameCB}
	h, _, e := procOpenTraceW.Call(uintptr(unsafe.Pointer(&lf)))
	if uint64(h) == ^uint64(0) {
		frameSink.Store(nil)
		stopFrameSession(name)
		return nil, e
	}
	t := &FrameTrace{session: session, reader: uint64(h), done: make(chan struct{})}
	go func() {
		runtime.LockOSThread()
		defer close(t.done)
		handle := t.reader
		procProcessTrace.Call(uintptr(unsafe.Pointer(&handle)), 1, 0, 0) // until stopped
	}()
	ok = true
	return t, nil
}

func enableFrameProvider(session uint64, guid windows.GUID, ids []uint16) error {
	// EVENT_FILTER_EVENT_ID: only these events leave the provider, so the
	// rest of what it can say (a lot, for the kernel) costs nothing.
	filter := make([]byte, 4+2*len(ids))
	filter[0] = 1 // FilterIn
	*(*uint16)(unsafe.Pointer(&filter[2])) = uint16(len(ids))
	for i, id := range ids {
		*(*uint16)(unsafe.Pointer(&filter[4+2*i])) = id
	}
	desc := eventFilterDescriptor{Ptr: uint64(uintptr(unsafe.Pointer(&filter[0]))), Size: uint32(len(filter)), Type: eventFilterTypeEventID}
	params := enableTraceParameters{Version: enableTraceParametersVersion, EnableFilterDesc: &desc, FilterDescCount: 1}
	r, _, _ := procEnableTraceEx2.Call(uintptr(session), uintptr(unsafe.Pointer(&guid)), eventControlCodeEnable, traceLevelVerbose,
		uintptr(^uint64(0)), 0, 0, uintptr(unsafe.Pointer(&params)))
	runtime.KeepAlive(filter)
	if r != 0 {
		return windows.Errno(r)
	}
	return nil
}

// Stop ends the session; safe to call twice.
func (t *FrameTrace) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.session == 0 {
		return
	}
	name, _ := windows.UTF16PtrFromString(frameSession)
	procControlTraceW.Call(uintptr(t.session), uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(newTraceProperties())), eventTraceControlStop)
	procCloseTrace.Call(uintptr(t.reader))
	<-t.done
	frameSink.Store(nil)
	t.session = 0
	frameTraceOne.Unlock()
}

// PerfNow is QueryPerformanceCounter now; PerfFreq its ticks a second.
func PerfNow() int64 {
	var n int64
	procQPC.Call(uintptr(unsafe.Pointer(&n)))
	return n
}

func PerfFreq() int64 {
	var n int64
	procQPF.Call(uintptr(unsafe.Pointer(&n)))
	return n
}

// FramesAllowed says whether this process may start the frame session:
// it runs elevated, or the account is in Performance Log Users in this
// sign-in's token (or an administrator with a full token).
func FramesAllowed() bool {
	tok := windows.GetCurrentProcessToken()
	if tok.IsElevated() {
		return true
	}
	// CheckTokenMembership skips deny-only groups, so an administrator
	// whose token Windows filtered (UAC) counts only once elevated.
	for _, s := range []string{"S-1-5-32-559", "S-1-5-32-544"} {
		sid, err := windows.StringToSid(s)
		if err != nil {
			continue
		}
		if in, err := windows.Token(0).IsMember(sid); err == nil && in {
			return true
		}
	}
	return false
}

// UserSID is the signed-in account's SID ("S-1-5-21-...").
func UserSID() string {
	u, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return ""
	}
	return u.User.Sid.String()
}

// ForegroundPID is the process of the window in front, 0 for none.
func ForegroundPID() uint32 {
	var pid uint32
	if h := foreground(); h != 0 {
		procGetWindowThreadProcessId.Call(h, uintptr(unsafe.Pointer(&pid)))
	}
	return pid
}

var procSHQueryUserNotificationState = shell32.NewProc("SHQueryUserNotificationState")

// ExclusiveFullscreen says whether a Direct3D program holds the screen in
// exclusive fullscreen, where no other window shows.
func ExclusiveFullscreen() bool {
	var state uint32
	if procSHQueryUserNotificationState.Find() != nil {
		return false
	}
	r, _, _ := procSHQueryUserNotificationState.Call(uintptr(unsafe.Pointer(&state)))
	return r == 0 && state == 3 // QUNS_RUNNING_D3D_FULL_SCREEN
}
