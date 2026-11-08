//go:build windows

package win

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var procPlaySoundW = windows.NewLazySystemDLL("winmm.dll").NewProc("PlaySoundW")

// PlaySound plays a WAV held in memory without waiting for it; a new sound
// stops the one before. b must outlive the sound, as embedded data does.
func PlaySound(b []byte) {
	const (
		sndAsync     = 0x0001
		sndNoDefault = 0x0002
		sndMemory    = 0x0004
	)
	if len(b) == 0 {
		return
	}
	procPlaySoundW.Call(uintptr(unsafe.Pointer(&b[0])), 0, sndMemory|sndAsync|sndNoDefault)
}
