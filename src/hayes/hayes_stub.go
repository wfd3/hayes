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
		if m.pins[p] {	// LED is on
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
func (m *Modem) led_MR_on() {
	m.pins[MR_LED] = true
}
func (m *Modem) led_MR_off() {
	m.pins[MR_LED] = false
}
func (m *Modem) led_AA_on() {
	m.pins[AA_LED] = true
}
func (m *Modem) led_AA_off() {
	m.pins[AA_LED] = false
}
func(m *Modem) led_OH_on() {
	m.pins[OH_LED] = true
}
func(m *Modem) led_OH_off() {
	m.pins[OH_LED] = false
}
func(m *Modem) led_TR_on() {
	m.pins[TR_LED] = true
}
func(m *Modem) led_TR_off() {
	m.pins[TR_LED] = false
}
func(m *Modem) led_CS_on() {
	m.pins[CS_LED] = true
}
func(m *Modem) led_CS_off() {
	m.pins[CS_LED] = false
}
func (m *Modem) led_SD_on() {
	m.pins[SD_LED] = true
}
func (m *Modem) led_SD_off() {
	m.pins[SD_LED] = false
}
func (m *Modem) led_RD_on() {
	m.pins[RD_LED] = true
}
func (m *Modem) led_RD_off() {
	m.pins[RD_LED] = false
}
func (m *Modem) led_CD_on() {
	m.pins[CD_LED] = true
}
func (m *Modem) led_CD_off() {
	m.pins[CD_LED] = false
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

// RTS - Request to Send
func (m *Modem) readRTS() (bool) {
	// Has the computer requested data be sent?
	return m.pins[RTS_PIN]
}
