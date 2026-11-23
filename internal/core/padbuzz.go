package core

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
	"strings"
	"unicode/utf8"
)

// Telling controllers apart and naming them (the owner, 23 Nov 2026: give
// each pad a name, and make the one being named buzz so it's clear which
// is which). Names are kept by the pad's device as Windows knows it
// (Pad.ID): over Bluetooth that's the pad itself, over USB the pad in that
// port, since Windows names a USB pad without a serial number by where
// it's plugged in.

// PSRumble is the output report that sets a PlayStation pad's two motors
// (strong is the left, weak the right; 0 stops them), size bytes long as
// Windows asks (HIDP_CAPS.OutputReportByteLength: the rest stays zero).
// seq is the DualSense's Bluetooth sequence number. Only the motors are
// marked valid, so the lightbar, the player lights and the triggers stay
// as they are. Layouts as in Linux's hid-playstation and SDL:
//
//	DualSense USB 0x02: flags at 1 and 2, right motor 3, left 4, flags2 at 39
//	DualSense BT 0x31: seq<<4 at 1, tag 0x10 at 2, then USB's from 3, CRC at 74
//	DualShock 4 USB 0x05: flags at 1, right motor 4, left 5
//	DualShock 4 BT 0x11: 0xC0 at 1, flags at 3, right 6, left 7, CRC at 74
//
// The Bluetooth CRC is CRC-32 over 0xA2 (the output report's header) and
// the report's first 74 bytes, little-endian.
func PSRumble(dualsense, bluetooth bool, size int, seq, strong, weak uint8) ([]byte, error) {
	need := 32
	switch {
	case bluetooth:
		need = 78
	case dualsense:
		need = 48
	}
	if size < need || size > 1024 {
		return nil, errors.New("the controller's output reports aren't the size Lootnest knows")
	}
	b := make([]byte, size)
	// DualSense: 0x01 compatible vibration, 0x02 haptics as rumble; 0x04
	// in flags2 is the same on firmware 2.24 and later (older ignore it).
	at := 0
	switch {
	case dualsense && !bluetooth:
		b[0], at = 0x02, 1
	case dualsense:
		b[0], b[1], b[2], at = 0x31, (seq&0x0F)<<4, 0x10, 3
	case !bluetooth:
		b[0] = 0x05
		b[1], b[4], b[5] = 0x01, weak, strong // 0x01: the motors are valid
	default:
		b[0], b[1] = 0x11, 0xC0 // HID report with a CRC
		b[3], b[6], b[7] = 0x01, weak, strong
	}
	if dualsense {
		b[at], b[at+2], b[at+3], b[at+38] = 0x03, weak, strong, 0x04
	}
	if bluetooth {
		h := crc32.NewIEEE()
		h.Write([]byte{0xA2})
		h.Write(b[:74])
		binary.LittleEndian.PutUint32(b[74:78], h.Sum32())
	}
	return b, nil
}

// PadNameMax is the longest name a controller takes, in characters.
const PadNameMax = 32

// padNamesMax is how many controllers keep a name.
const padNamesMax = 64

// CleanPadName trims a controller's name; "" takes the name away.
func CleanPadName(name string) (string, error) {
	name = strings.Join(strings.Fields(name), " ")
	if utf8.RuneCountInString(name) > PadNameMax {
		return "", errors.New("A controller's name fits 32 characters.")
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7F {
			return "", errors.New("A controller's name is one line of text.")
		}
	}
	return name, nil
}

// SetPadName names a controller by its id (Pad.ID), or takes its name
// away with "". At most padNamesMax pads keep a name.
func SetPadName(names map[string]string, id, name string) (map[string]string, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" || len(id) > 400 {
		return names, errors.New("Lootnest can't tell which controller that is.")
	}
	name, err := CleanPadName(name)
	if err != nil {
		return names, err
	}
	out := map[string]string{}
	for k, v := range names {
		out[k] = v
	}
	if name == "" {
		delete(out, id)
	} else {
		if _, had := out[id]; !had && len(out) >= padNamesMax {
			return names, errors.New("Lootnest keeps names for 64 controllers. Take one away first.")
		}
		out[id] = name
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}
