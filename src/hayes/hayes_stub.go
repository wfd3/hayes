// +build !arm

package hayes

// Support for generic hardare (ie, not a Raspberry Pi)

import (
	"runtime"
	"fmt"
	"strings"
)

const (
	HS_LED = iota
	AA_LED 
	RI_LED
	MR_LED
	TR_LED
	RD_LED
	CS_LED
	CD_LED
	SD_LED
	OH_LED

	RI_PIN
	CD_PIN
	DSR_PIN
	CTS_PIN
	DTR_PIN
	RTS_PIN

	_PIN_LEN = RTS_PIN
)
type Pins [_PIN_LEN + 1]bool

func (m *Modem) setupPins() {
	debugf("Simulated Pins enabled on %s/%s\n", runtime.GOOS, runtime.GOARCH)

	m.clearPins()

	// The DTE is always ready
	m.pins[DTR_PIN] = true
	m.pins[RTS_PIN] = true
}

func (m *Modem) clearPins() {
	for i := range m.leds {
		m.leds[i] = false
	}
	for i := range m.pins {
		m.pins[i] = false
	}
}

func (m *Modem) showPins() {

	pp := func (n string, p int) (string) {
		var s string
		if m.pins[p] {
			s = "High"
		} else {
			s = "Low"
		}
		return fmt.Sprintf("%s:[%s] ", n, s)
	}
	s := "PINs: "
	s += pp("CTS", CTS_PIN)
	s += pp("RI", RI_PIN)
	s += pp("CD", CD_PIN)
	s += pp("DSR", DSR_PIN)
	s += pp("RTS", RTS_PIN)
	s += pp("DTR", DTR_PIN)
	fmt.Println(s)

	pl := func (n string, p int) (string) {
		var s string
		if m.leds[p] {	// LED is on
			s = strings.ToUpper(n)
		} else {
			s = strings.ToLower(n)
		}

		s += " "
		return s
	}
	s = "LEDs: [ "
	s += pl("HS", HS_LED)
	s += pl("AA", AA_LED)
	s += pl("RI", RI_LED)
	s += pl("MR", MR_LED)
	s += pl("TR", TR_LED)
	s += pl("RD", RD_LED)
	s += pl("CS", CS_LED)
	s += pl("CD", CD_LED)
	s += pl("SD", SD_LED)
	s += "]"
	fmt.Println(s)

}

// LED functions
func (m *Modem) led_HS_on() {
	m.leds[HS_LED] = true
}
func (m *Modem) led_HS_off() {
	m.leds[HS_LED] = false
}
func (m *Modem) led_MR_on() {
	m.leds[MR_LED] = true
}
func (m *Modem) led_MR_off() {
	m.leds[MR_LED] = false
}
func (m *Modem) led_AA_on() {
	m.leds[AA_LED] = true
}
func (m *Modem) led_AA_off() {
	m.leds[AA_LED] = false
}
func (m *Modem) led_RI_on() {
	m.leds[AA_LED] = true
}
func (m *Modem) led_RI_off() {
	m.leds[AA_LED] = false
}
func(m *Modem) led_OH_on() {
	m.leds[OH_LED] = true
}
func(m *Modem) led_OH_off() {
	m.leds[OH_LED] = false
}
func(m *Modem) led_TR_on() {
	m.leds[TR_LED] = true
}
func(m *Modem) led_TR_off() {
	m.leds[TR_LED] = false
}
func(m *Modem) led_CS_on() {
	m.leds[CS_LED] = true
}
func(m *Modem) led_CS_off() {
	m.leds[CS_LED] = false
}
func (m *Modem) led_SD_on() {
	m.leds[SD_LED] = true
}
func (m *Modem) led_SD_off() {
	m.leds[SD_LED] = false
}
func (m *Modem) led_RD_on() {
	m.leds[RD_LED] = true
}
func (m *Modem) led_RD_off() {
	m.leds[RD_LED] = false
}
func (m *Modem) led_CD_on() {
	m.leds[CD_LED] = true
}
func (m *Modem) led_CD_off() {
	m.leds[CD_LED] = false
}
func (m *Modem) ledTest(i int) {
	// NOOP
}

// PINs

// RI - Ring Indicator
func (m *Modem) raiseRI() {
	m.pins[RI_PIN] = true
}
func (m *Modem) lowerRI() {
	m.pins[RI_PIN] = false
}
func (m *Modem) readRI() (bool) {
	return m.pins[RI_PIN]
}

// CD - Carrier Detect
func (m *Modem) raiseCD() {
	m.pins[CD_PIN] = true
}
func (m *Modem) lowerCD() {
	m.pins[CD_PIN] = false
}
func (m *Modem) readCD() (bool) {
	return m.pins[CD_PIN]
}

// DSR - Data Set Ready
func (m *Modem) raiseDSR() {
	m.pins[DSR_PIN] = true
	debugf("raiseDSR()")
}
func (m *Modem) lowerDSR() {
	m.pins[DSR_PIN] = false
	debugf("lowerDSR()")
}
func (m *Modem) readDSR() (bool) {
	return m.pins[DSR_PIN]
}

// CTS - Clear to Send
func (m *Modem) raiseCTS() {
	m.pins[CTS_PIN] = true
	debugf("raiseCTS()")
}
func (m *Modem) lowerCTS() {
	m.pins[CTS_PIN] = false
	debugf("lowerCTS()")
}
func (m *Modem) readCTS() (bool) {
	return m.pins[CTS_PIN]
}

// DTR - Data Terminal Ready
func (m *Modem) readDTR() (bool) {
	// Is the computer ready to send data?
	return m.pins[DTR_PIN]
}
func (m *Modem) raiseDTR() {
	if !debug {
		panic("Can't raise/lower input pins when not in DEBUG mode")
	}
	m.pins[DTR_PIN] = true
}
func (m *Modem) lowerDTR() {
	if !debug {
		panic("Can't raise/lower input pins when not in DEBUG mode")
	}
	m.pins[DTR_PIN] = false
}

// RTS - Request to Send
func (m *Modem) readRTS() (bool) {
	// Has the computer requested data be sent?
	return m.pins[RTS_PIN]
}
func (m *Modem) raiseRTS() {
	if !debug {
		panic("Can't raise/lower input pins when not in DEBUG mode")
	}
	m.pins[RTS_PIN] = true
}
func (m *Modem) lowerRTS() {
	if !debug {
		panic("Can't raise/lower input pins when not in DEBUG mode")
	}
	m.pins[RTS_PIN] = false
}


