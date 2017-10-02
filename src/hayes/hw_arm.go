// +build arm

package hayes

import (
	"github.com/stianeikeland/go-rpio"
	"strings"
	"time"
)

type Pins map[int]rpio.Pin

// LED and data pins
const (
	// LEDs - controlled in handleLeds()
	HS_LED  = 2		// Connected at High speed (m.speed > 14400)
	AA_LED  = 3		// Auto Answer configured (m.r[0] > 0)
	TR_LED  = 9		// Terminal Ready (turn on if read(DTR) is high)
	OH_LED  = 27		// Is the modem off hook (m.offHook() == true)

	// Receive and Send Data LEDs.  Manually controlled
	RD_LED  = 10		// Receive Data
	SD_LED  = 22		// Send Data

	// Data Pins
	// A MAX3232 translates these from 0V & 3V to RS232 -/+{3,5,12}V
	CTS_PIN = 12		// Clear To Send pin
	CS_LED  = 11		// Clear To Send

	RI_PIN  = 23		// Ring Indicator pin
	RI_LED  = 4		// Ring Indicator

	CD_PIN  = 24		// Carrier Detect pin
	CD_LED  = 17		// Carrier Detect

	DSR_PIN = 25 		// Data Set Ready pin
	MR_LED  = 5		// Modem Ready

	RTS_PIN = 7		// Request to Send pin (Input)
	DTR_PIN = 16 		// Data Terminal Ready (Input)
)

func (m *Modem) setupPins() {

	if err := rpio.Open(); err != nil {
		m.log.Fatal("Fatal Error: ", err)
	}

	leds := make(Pins)
	pins := make(Pins)

	// LEDs
	leds[HS_LED] = rpio.Pin(HS_LED)
	leds[HS_LED].Output()
	
	leds[AA_LED] = rpio.Pin(AA_LED)
	leds[AA_LED].Output()
	
	leds[RI_LED] = rpio.Pin(RI_LED)
	leds[RI_LED].Output()
	
	leds[MR_LED] = rpio.Pin(MR_LED)
	leds[MR_LED].Output()
	
	leds[TR_LED] = rpio.Pin(TR_LED)
	leds[TR_LED].Output()
	
	leds[RD_LED] = rpio.Pin(RD_LED)
	leds[RD_LED].Output()
	
	leds[CS_LED] = rpio.Pin(CS_LED)
	leds[CS_LED].Output()

	leds[OH_LED] = rpio.Pin(OH_LED)
	leds[OH_LED].Output()

	leds[CD_LED] = rpio.Pin(CD_LED)
	leds[CD_LED].Output()
	
	leds[SD_LED] = rpio.Pin(SD_LED)
	leds[SD_LED].Output()
	

	// Pins
	pins[CTS_PIN] = rpio.Pin(CTS_PIN)
	pins[CTS_PIN].Output()
	
	pins[RI_PIN] = rpio.Pin(RI_PIN)
	pins[RI_PIN].Output()
	
	pins[CD_PIN] = rpio.Pin(CD_PIN)
	pins[CD_PIN].Output()
	
	pins[DSR_PIN] = rpio.Pin(DSR_PIN)
	pins[DSR_PIN].Output()
	
	pins[DTR_PIN] = rpio.Pin(DTR_PIN)
	pins[DTR_PIN].Input()
	
	pins[RTS_PIN] = rpio.Pin(RTS_PIN)
	pins[RTS_PIN].Input()

	m.leds = leds
	m.pins = pins
}

func (m *Modem) clearPins() {
	m.leds[HS_LED].Low()
	m.leds[AA_LED].Low()
	m.leds[RI_LED].Low()
	m.leds[MR_LED].Low()
	m.leds[TR_LED].Low()
	m.leds[RD_LED].Low()
	m.leds[CS_LED].Low()
	m.leds[CD_LED].Low()
	m.leds[SD_LED].Low()
	m.leds[OH_LED].Low()

	m.pins[RI_PIN].Low()
	m.pins[CD_PIN].Low()
	m.pins[DSR_PIN].Low()
	m.pins[CTS_PIN].Low()
	// No need to do RTS and DTR
}

func (m *Modem) showPins() string {
	pp := func (n string, pin rpio.Pin) (string) {
		var s string
		if pin.Read() == rpio.High {	
			s = strings.ToUpper(n)
		} else {
			s = strings.ToLower(n)
		}
		s += " "
		return s
	}

	s := "PINs: ["
	s += pp("CTS", m.pins[CTS_PIN])
	s += pp("RI_", m.pins[RI_PIN])
	s += pp("DCD", m.pins[CD_PIN])
	s += pp("DSR", m.pins[DSR_PIN])
	s += pp("RTS", m.pins[RTS_PIN])
	s += pp("DTR", m.pins[DTR_PIN])
	s += "]"

	s += "\n"

	s += "LEDs: "
	s += pp("HS", m.leds[HS_LED])
	s += pp("AA", m.leds[AA_LED])
	s += pp("RI", m.leds[RI_LED])
	s += pp("CD", m.leds[CD_LED])
	s += pp("OH", m.leds[OH_LED])
	s += pp("MR", m.leds[MR_LED])
	s += pp("CS", m.leds[ CS_LED])
	s += pp("TR", m.leds[TR_LED])
	s += pp("SD", m.leds[SD_LED])
	s += pp("RD", m.leds[RD_LED])
	s += "]"
	return s
}

// Led functions
func (m *Modem) led_HS_on() {
	m.leds[HS_LED].High()
}
func (m *Modem) led_HS_off() {
	m.leds[HS_LED].Low()
}

func (m *Modem) led_AA_on() {
	m.leds[AA_LED].High()
}
func (m *Modem) led_AA_off() {
	m.leds[AA_LED].Low()
}

func(m *Modem) led_OH_on() {
	m.leds[OH_LED].High()
}
func(m *Modem) led_OH_off() {
	m.leds[OH_LED].Low()
}

func(m *Modem) led_TR_on() {
	m.leds[TR_LED].High()
}
func(m *Modem) led_TR_off() {
	m.leds[TR_LED].Low()
}

func (m *Modem) led_SD_on() {
	m.leds[SD_LED].High()
}
func (m *Modem) led_SD_off() {
	m.leds[SD_LED].Low()
}

func (m *Modem) led_RD_on() {
	m.leds[RD_LED].High()
}
func (m *Modem) led_RD_off() {
	m.leds[RD_LED].Low()
}

func (m *Modem) ledTest(round int) {
	var saved_leds map[int]rpio.State

	saved_leds = make(map[int]rpio.State)

	// Turn them all on, wait a bit, turn them all off.
	for i:= range m.leds {
		saved_leds[i] = m.leds[i].Read() // Save current state
		m.leds[i].High()
		time.Sleep(50 * time.Millisecond)
	}
	time.Sleep(500 * time.Millisecond)
	for i:= range m.leds {
		m.leds[i].Low()
		time.Sleep(50 * time.Millisecond)
	}
	time.Sleep(500 * time.Millisecond)
	
	// Randomly (based on how range works) turn on and off round times
	for j := 0; j < round; j++ {
		for i:= range m.leds {
			m.leds[i].High()
			time.Sleep(50 * time.Millisecond)
		}
		time.Sleep(10 * time.Millisecond)
		for i:= range m.leds {
			m.leds[i].Low()
			time.Sleep(50 * time.Millisecond)
		}
	}

	// Restore LED state
	for j := range saved_leds {
		m.leds[j].Write(saved_leds[j])
	}

}

// PINs
//
// This assumes a MAX3232 to do the level conversion between the Pi's
// 0 and 3V low/high and RS-232 +5V/-5V.  So 0V (low) on the Pi is +5V
// (low) on RS-232 and vice versa.

// RI - assert RI and turn on RI light
func (m *Modem) raiseRI() {
	m.leds[RI_LED].High()
	m.pins[RI_PIN].High()
}
func (m *Modem) lowerRI() {
	m.leds[RI_LED].Low()
	m.pins[RI_PIN].Low()
}
func (m *Modem) readRI() (bool) {
	return m.pins[RI_PIN].Read() == rpio.High &&
		m.leds[RI_LED].Read() == rpio.High
}

// CD - assert CD and turn on CD light
func (m *Modem) raiseCD() {
	m.leds[CD_LED].High()
	m.pins[CD_PIN].High()
}
func (m *Modem) lowerCD() {
	m.leds[CD_LED].Low()
	m.pins[CD_PIN].Low()
}
func (m *Modem) readCD() (bool) {
	return m.pins[CD_PIN].Read() == rpio.High &&
		m.leds[CD_LED].Read() == rpio.High
}

// DSR - assert DSR and turn on MR light
func (m *Modem) raiseDSR() {
	m.leds[MR_LED].High()
	m.pins[DSR_PIN].High()
}
func (m *Modem) lowerDSR() {
	m.leds[MR_LED].Low()
	m.pins[DSR_PIN].Low()
}
func (m *Modem) readDSR() (bool) {
	return m.pins[DSR_PIN].Read() == rpio.High &&
		m.pins[MR_LED].Read() == rpio.High
}

// CTS - assert CTS and turn on CS light
func (m *Modem) raiseCTS() {
	m.leds[CS_LED].High()
	m.pins[CTS_PIN].High()
}
func (m *Modem) lowerCTS() {
	m.leds[CS_LED].Low()
	m.pins[CTS_PIN].Low()
}
func (m *Modem) readCTS() (bool) {
	return m.pins[CTS_PIN].Read() == rpio.High &&
		m.leds[CS_LED].Read() == rpio.High
}

// DTR (input)
func (m *Modem) readDTR() (bool) {
	return m.pins[DTR_PIN].Read() == rpio.High
}

// RTS (input)
func (m *Modem) readRTS() (bool) {
	return m.pins[RTS_PIN].Read() == rpio.High
}
	
