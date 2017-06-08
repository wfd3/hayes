// +build arm

package hayes

import (
	"github.com/stianeikeland/go-rpio"
	"fmt"
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
	OH_LED  = 27		// Is the modem off hook (m.offhook == true)

	// Receive and Send Data LEDs.  Manually controlled
	RD_LED  = 10		// Receive Data
	SD_LED  = 22		// Send Data

	// Data Pins
	// These pin pairs are required to drive an L293.  The _LED pin asserts the
	// desired output (High or Low), the _PIN pin is the inverse to control the
	// poliarity int he L293.
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

	// Control Pin -- when asserted, the L293 will switch
	// polarity. When deasserted, the L239 will generate 0C
	CONTROL = 26	
	
)

func (m *Modem) setupPins() {

	if err := rpio.Open(); err != nil {
		panic(err)
	}

	m.leds = make(Pins)
	m.pins = make(Pins)

	// LEDs
	m.leds[HS_LED] = rpio.Pin(HS_LED)
	m.leds[HS_LED].Output()
	
	m.leds[AA_LED] = rpio.Pin(AA_LED)
	m.leds[AA_LED].Output()
	
	m.leds[RI_LED] = rpio.Pin(RI_LED)
	m.leds[RI_LED].Output()
	
	m.leds[MR_LED] = rpio.Pin(MR_LED)
	m.leds[MR_LED].Output()
	
	m.leds[TR_LED] = rpio.Pin(TR_LED)
	m.leds[TR_LED].Output()
	
	m.leds[RD_LED] = rpio.Pin(RD_LED)
	m.leds[RD_LED].Output()
	
	m.leds[CS_LED] = rpio.Pin(CS_LED)
	m.leds[CS_LED].Output()

	m.leds[OH_LED] = rpio.Pin(OH_LED)
	m.leds[OH_LED].Output()

	m.leds[CD_LED] = rpio.Pin(CD_LED)
	m.leds[CD_LED].Output()
	
	m.leds[SD_LED] = rpio.Pin(SD_LED)
	m.leds[SD_LED].Output()
	

	// Pins
	m.pins[CTS_PIN] = rpio.Pin(CTS_PIN)
	m.pins[CTS_PIN].Output()
	
	m.pins[RI_PIN] = rpio.Pin(RI_PIN)
	m.pins[RI_PIN].Output()
	
	m.pins[CD_PIN] = rpio.Pin(CD_PIN)
	m.pins[CD_PIN].Output()
	
	m.pins[DSR_PIN] = rpio.Pin(DSR_PIN)
	m.pins[DSR_PIN].Output()
	
	m.pins[DTR_PIN] = rpio.Pin(DTR_PIN)
	m.pins[DTR_PIN].Input()
	
	m.pins[RTS_PIN] = rpio.Pin(RTS_PIN)
	m.pins[RTS_PIN].Input()

	// Control pin for L293
	m.pins[CONTROL] = rpio.Pin(CONTROL)
	m.pins[CONTROL].Output()
	m.pins[CONTROL].High()

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
	// m.pins[RTS_PIN].Low()
	// m.pins[DTR_PIN].Low()

	m.pins[CONTROL].Low()
}

func (m *Modem) showPins() {

	pp := func (n string, b bool) (string) {
		var state string
		if b {
			state = "High"
		} else {
			state = "Low"
		}
		return fmt.Sprintf("%s:[%s] ", n, state)
	}
	s := "PINs: "
	s += pp("CTS", m.readCTS())
	s += pp("RI", m.readRI())
	s += pp("CD", m.readCD())
	s += pp("DSR", m.readDSR())
	s += pp("RTS", m.readRTS())
	s += pp("DTR", m.readDTR())
	fmt.Println(s)

	pl := func (n string, p int) (string) {
		if m.leds[p].Read() == rpio.High {	// LED is on
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
	s += pl("CD", CD_LED)
	s += pl("OH", OH_LED)
	
	s += pl("MR", MR_LED)
	s += pl("CS", CS_LED)
	s += pl("TR", TR_LED)
	s += pl("SD", SD_LED)
	s += pl("RD", RD_LED)
	s += "]"
	fmt.Println(s)
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

	// First turn off the L293
	m.pins[CONTROL].Low()

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

	// Turn the L293 back on
	m.pins[CONTROL].High()
}

// PINs

// RI - assert RI and turn on RI light
func (m *Modem) raiseRI() {
	m.leds[RI_LED].High()
	m.pins[RI_PIN].Low()
}
func (m *Modem) lowerRI() {
	m.leds[RI_LED].Low()
	m.pins[RI_PIN].High()
}
func (m *Modem) readRI() (bool) {
	return m.pins[RI_PIN].Read() == rpio.Low &&
		m.leds[RI_LED].Read() == rpio.High
}

// CD - assert CD and turn on CD light
func (m *Modem) raiseCD() {
	m.leds[CD_LED].High()
	m.pins[CD_PIN].Low()
}
func (m *Modem) lowerCD() {
	m.leds[CD_LED].Low()
	m.pins[CD_PIN].High()
}
func (m *Modem) readCD() (bool) {
	return m.pins[CD_PIN].Read() == rpio.Low &&
		m.leds[CD_LED].Read() == rpio.High
}

// DSR - assert DSR and turn on MR light
func (m *Modem) raiseDSR() {
	m.leds[MR_LED].High()
	m.pins[DSR_PIN].Low()
}
func (m *Modem) lowerDSR() {
	m.leds[MR_LED].Low()
	m.pins[DSR_PIN].High()
}
func (m *Modem) readDSR() (bool) {
	return m.pins[DSR_PIN].Read() == rpio.Low &&
		m.pins[MR_LED].Read() == rpio.High
}

// CTS - assert CTS and turn on CS light
func (m *Modem) raiseCTS() {
	m.leds[CS_LED].High()
	m.pins[CTS_PIN].Low()
}
func (m *Modem) lowerCTS() {
	m.leds[CS_LED].Low()
	m.pins[CTS_PIN].High()
}
func (m *Modem) readCTS() (bool) {
	return m.pins[CTS_PIN].Read() == rpio.Low &&
		m.leds[CS_LED].Read() == rpio.High
}

// DTR
func (m *Modem) raiseDTR() {
	if !debug {
		panic("Can't raise input pins on this platform")
	}
}
func (m *Modem) lowerDTR() {
	if !debug {
		panic("Can't raise input pins on this platform")
	}
}
func (m *Modem) readDTR() (bool) {
	return m.pins[DTR_PIN].Read() == rpio.High
}

// RTS
func (m *Modem) raiseRTS() {
	if !debug {
		panic("Can't raise input pins on this platform")
	}
}
func (m *Modem) lowerRTS() {
	if !debug {
		panic("Can't raise input pins on this platform")
	}
}
func (m *Modem) readRTS() (bool) {
	return m.pins[RTS_PIN].Read() == rpio.High
}
	
