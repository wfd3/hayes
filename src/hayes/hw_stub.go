// +build !arm

package hayes

// Support for generic hardare (ie, not a Raspberry Pi)

import (
	"strings"
	"runtime"
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

	_PIN_LEN		// This needs to be last in the const list
)
type Pins [_PIN_LEN]bool

func (m *Modem) setupPins() {
	m.log.Printf("Simulated Pins enabled on %s/%s\n",
		runtime.GOOS, runtime.GOARCH)

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

func (m *Modem) showPins() string {

	pp := func (n string, p int) (string) {
		var s string
		if m.pins[p] {
			s = strings.ToUpper(n)
		} else {
			s = strings.ToLower(n)
		}
		s += " " 
		return s
	}
	s := "PINs: ["
	s += pp("CTS", CTS_PIN)
	s += pp("RI ", RI_PIN)
	s += pp("CD ", CD_PIN)
	s += pp("DSR", DSR_PIN)
	s += pp("RTS", RTS_PIN)
	s += pp("DTR", DTR_PIN)
	s += "]\n"

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
	s += "LEDs: [ "
	s += pl("HS", HS_LED)
	s += pl("AA", AA_LED)
	s += pl("RI", RI_LED)
	s += pl("CD", CD_LED)
	s += pl("OH", OH_LED)
	s += pl("SD", SD_LED)
	s += pl("RD", RD_LED)
	s += pl("TR", TR_LED)
	s += pl("CS", CS_LED)
	s += pl("MR", MR_LED)
	s += "]"
	return s
}

// LED functions
func (m *Modem) led_HS_on() {
	m.leds[HS_LED] = true
}
func (m *Modem) led_HS_off() {
	m.leds[HS_LED] = false
}

func (m *Modem) led_AA_on() {
	m.leds[AA_LED] = true
}
func (m *Modem) led_AA_off() {
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
	m.leds[CD_LED] = true
	m.pins[CD_PIN] = true
}
func (m *Modem) lowerCD() {
	m.leds[CD_LED] = false
	m.pins[CD_PIN] = false
}
func (m *Modem) readCD() (bool) {
	return m.pins[CD_PIN] && m.leds[CD_LED]
}

// DSR - Data Set Ready
func (m *Modem) raiseDSR() {
	m.leds[MR_LED] = true
	m.pins[DSR_PIN] = true
	m.log.Print("raiseDSR()")
}
func (m *Modem) lowerDSR() {
	m.leds[MR_LED] = false
	m.pins[DSR_PIN] = false
	m.log.Print("lowerDSR()")
}
func (m *Modem) readDSR() (bool) {
	return m.pins[DSR_PIN] && m.leds[MR_LED]
}

// CTS - Clear to Send
func (m *Modem) raiseCTS() {
	m.leds[CS_LED] = true
	m.pins[CTS_PIN] = true
	m.log.Print("raiseCTS()")
}
func (m *Modem) lowerCTS() {
	m.leds[CS_LED] = true
	m.pins[CTS_PIN] = false
	m.log.Print("lowerCTS()")
}
func (m *Modem) readCTS() (bool) {
	return m.pins[CTS_PIN] && m.leds[CS_LED]
}

// DTR - Data Terminal Ready (input)
func (m *Modem) readDTR() (bool) {
	// Is the computer ready to send data?
	return m.pins[DTR_PIN]
}

// RTS - Request to Send (input)
func (m *Modem) readRTS() (bool) {
	// Has the computer requested data be sent?
	return m.pins[RTS_PIN]
}
