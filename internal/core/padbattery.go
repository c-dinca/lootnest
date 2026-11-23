package core

// A controller's battery, a warning when it runs low, and a battery saver
// (the owner, 23 Nov 2026). A PlayStation pad says its charge and whether
// it's charging in its full input reports (a Bluetooth one sends them once
// it's asked for a feature report, as Steam and DS4Windows do); an Xbox pad
// says a rough level through XInput. Layouts as in Linux's hid-playstation.

// PadBattery is a controller's charge as last read.
type PadBattery struct {
	Percent  int    `json:"percent"`  // 0 to 100; -1 when the pad doesn't say
	Level    string `json:"level"`    // empty, low, medium or full, for an Xbox pad's rough level
	Charging bool   `json:"charging"` // on a cable and filling up
	Full     bool   `json:"full"`     // on a cable and full
	Wired    bool   `json:"wired"`    // runs on its cable, no battery to speak of
}

// Low is where Lootnest warns, and Empty where it warns once more.
const (
	BatteryLow   = 20
	BatteryEmpty = 10
	batteryAgain = 30 // over this, or charging, the warnings may come again
)

// psStatusAt is where a full input report keeps the battery byte.
func psStatusAt(dualsense, bluetooth bool, id byte) int {
	switch {
	case dualsense && !bluetooth && id == 0x01:
		return 53
	case dualsense && bluetooth && id == 0x31:
		return 54
	case !dualsense && !bluetooth && id == 0x01:
		return 30
	case !dualsense && bluetooth && id == 0x11:
		return 32
	}
	return -1 // a Bluetooth pad's short report carries no battery
}

// PSBattery reads the battery from one input report of a PlayStation pad.
// ok is false for a report that doesn't carry it (a Bluetooth pad's short
// one: Windows pads every report to the longest, so the length says
// nothing) and for a status the pad marks as an error.
func PSBattery(dualsense, bluetooth bool, r []byte) (PadBattery, bool) {
	if len(r) == 0 {
		return PadBattery{}, false
	}
	at := psStatusAt(dualsense, bluetooth, r[0])
	if at < 0 || len(r) <= at {
		return PadBattery{}, false
	}
	s := r[at]
	data := int(s & 0x0F)
	pct := min(data*10+5, 100)
	if dualsense {
		switch s >> 4 {
		case 0x0:
			return PadBattery{Percent: pct}, true
		case 0x1:
			return PadBattery{Percent: pct, Charging: true}, true
		case 0x2:
			return PadBattery{Percent: 100, Full: true}, true
		}
		return PadBattery{}, false // a voltage, temperature or charging error
	}
	if s&0x10 == 0 { // no cable
		return PadBattery{Percent: pct}, true
	}
	switch {
	case data < 10:
		return PadBattery{Percent: pct, Charging: true}, true
	case data <= 11:
		return PadBattery{Percent: 100, Full: true}, true
	}
	return PadBattery{}, false
}

// XInputBattery reads XInputGetBatteryInformation's type and level. ok is
// false for a pad that isn't there or doesn't say.
func XInputBattery(typ, level byte) (PadBattery, bool) {
	switch typ {
	case 0x00, 0xFF: // disconnected, unknown
		return PadBattery{}, false
	case 0x01: // on its cable
		return PadBattery{Percent: -1, Wired: true}, true
	}
	b := PadBattery{Level: [...]string{"empty", "low", "medium", "full"}[level&3]}
	b.Percent = [...]int{5, 20, 55, 100}[level&3]
	return b, true
}

// BatteryWarn is what to say about a pad's battery now, given what was
// said before (warned: 0 nothing, 1 low, 2 almost empty) and the reading
// before (was, with seen false the first time): "low", "empty", "charged"
// or "", and what's been said from now on. A pad that's charging, or back
// over 30%, may be warned about again.
func BatteryWarn(was, now PadBattery, seen bool, warned int) (string, int) {
	if now.Wired || now.Percent < 0 {
		return "", 0
	}
	if now.Charging || now.Full || now.Percent > batteryAgain {
		if seen && was.Charging && now.Full {
			return "charged", 0
		}
		return "", 0
	}
	switch {
	case now.Percent <= BatteryEmpty && warned < 2:
		return "empty", 2
	case now.Percent <= BatteryLow && warned < 1:
		return "low", 1
	}
	return "", warned
}

// SaverWanted says whether a pad's battery saver should be on: mode "on"
// always, "low" while it's low and not charging (once on, until it charges
// or is back over 30%), "" never.
func SaverWanted(mode string, b PadBattery, was bool) bool {
	switch mode {
	case "on":
		return true
	case "low":
		if b.Wired || b.Percent < 0 || b.Charging || b.Full {
			return false
		}
		if was {
			return b.Percent <= batteryAgain
		}
		return b.Percent <= BatteryLow
	}
	return false
}

// PSSaver is the output report that turns a PlayStation pad's battery
// saver on or off. On: the light bar and the player lights go out, and a
// DualSense's haptics are put to sleep (its power save control, which
// games don't set, so it holds until the pad is turned off). Off: haptics
// wake and the light bar is Sony's blue again, until a game sets it.
// size and seq as for PSRumble.
func PSSaver(dualsense, bluetooth bool, size int, seq uint8, on bool) ([]byte, error) {
	b, err := PSRumble(dualsense, bluetooth, size, seq, 0, 0)
	if err != nil {
		return nil, err
	}
	blue := byte(0x40)
	if on {
		blue = 0
	}
	if dualsense {
		at := 1
		if bluetooth {
			at = 3
		}
		b[at], b[at+2], b[at+3], b[at+38] = 0, 0, 0, 0 // leave the motors to the game
		// valid_flag1: 0x02 power save control, 0x04 light bar, 0x10 player lights.
		b[at+1] = 0x02 | 0x04
		if on {
			b[at+1] |= 0x10
			b[at+9] = 0x04 // haptics asleep
		}
		b[at+43], b[at+44], b[at+45], b[at+46] = 0, 0, 0, blue
	} else {
		at := 1
		if bluetooth {
			at = 3
		}
		b[at] = 0x02 // only the light bar is valid
		b[at+3], b[at+4] = 0, 0
		b[at+5], b[at+6], b[at+7] = 0, 0, blue
	}
	if bluetooth {
		psCRC(b)
	}
	return b, nil
}
