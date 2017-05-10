// +build arm

package hayes

import (
	"github.com/stianeikeland/go-rpio"
	"fmt"
	"strings"
)

type Pins  [28]rpio.Pin	// TODO:  this is ugly
const (
	HIGH = true
	LOW  = false
)

// LED and data pins
const (
	// LEDs
	HS_LED  = 2		// Physical pin 3
	AA_LED  = 3		// Physical pin 5
	RI_LED  = 4		// Physical pin 7
	MR_LED  = 5		// Physical pin 29
	TR_LED  = 9		// Physical pin 21
	RD_LED  = 10		// Physical pin 19
	CS_LED  = 11		// Physical pin 23
	CD_LED  = 17		// Physical pin 11
	SD_LED  = 22		// Physical pin 15
	OH_LED  = 27		// Physical pin 13
	// Data Pins
	CTS_PIN = 12		// Physical pin 32 - Output
	DTR_PIN = 16 		// Physical pin 36 - Input
	RI_PIN  = 23		// Physical pin 16 - Output
	CD_PIN  = 24		// Physical pin 18 - Output
	DSR_PIN = 25 		// Physical pin 22 - Output
	RTS_PIN = 7		// Physical pin 26 - Input
)


func (m *Modem) setupPins() {

	if err := rpio.Open(); err != nil {
		panic(err)
	}

	// LEDs
	m.pins[HS_LED] = rpio.Pin(HS_LED)
	m.pins[HS_LED].Output()
	
	m.pins[AA_LED] = rpio.Pin(AA_LED)
	m.pins[AA_LED].Output()
	
	m.pins[RI_LED] = rpio.Pin(RI_LED)
	m.pins[RI_LED].Output()
	
	m.pins[MR_LED] = rpio.Pin(MR_LED)
	m.pins[MR_LED].Output()
	
	m.pins[TR_LED] = rpio.Pin(TR_LED)
	m.pins[TR_LED].Output()
	
	m.pins[RD_LED] = rpio.Pin(RD_LED)
	m.pins[RD_LED].Input()
	
	m.pins[CS_LED] = rpio.Pin(CS_LED)
	m.pins[CS_LED].Output()

	m.pins[OH_LED] = rpio.Pin(OH_LED)
	m.pins[OH_LED].Output()

	m.pins[CD_LED] = rpio.Pin(CD_LED)
	m.pins[CD_LED].Output()
	
	m.pins[SD_LED] = rpio.Pin(SD_LED)
	m.pins[SD_LED].Output()
	

	// Pins
	m.pins[CTS_PIN] = rpio.Pin(CTS_PIN)
	m.pins[CTS_PIN].Output()
	
	m.pins[DTR_PIN] = rpio.Pin(DTR_PIN)
	m.pins[DTR_PIN].Input()
	
	m.pins[RI_PIN] = rpio.Pin(RI_PIN)
	m.pins[RI_PIN].Output()
	
	m.pins[CD_PIN] = rpio.Pin(CD_PIN)
	m.pins[CD_PIN].Output()
	
	m.pins[DSR_PIN] = rpio.Pin(DSR_PIN)
	m.pins[DSR_PIN].Output()
	
	m.pins[RTS_PIN] = rpio.Pin(RTS_PIN)
	m.pins[RTS_PIN].Input()
}

func (m *Modem) clearPins() {
	m.pins[HS_LED].Low()
	m.pins[AA_LED].Low()
	m.pins[RI_LED].Low()
	m.pins[MR_LED].Low()
	m.pins[TR_LED].Low()
	m.pins[RD_LED].Low()
	m.pins[CS_LED].Low()
	m.pins[CD_LED].Low()
	m.pins[SD_LED].Low()
	m.pins[OH_LED].Low()

	m.pins[RI_PIN].Low()
	m.pins[CD_PIN].Low()
	m.pins[DSR_PIN].Low()
	m.pins[CTS_PIN].Low()
}

func (m *Modem) showPins() {

	pp := func (n string, p rpio.Pin) (string) {
		var state string
		if p.Read() == rpio.High {
			state = "High"
		} else {
			state = "Low"
		}
		return fmt.Sprintf("%s:[%s] ", n, state)
	}
	s := "PINs: "
	s += pp("CTS", CTS_PIN)
	s += pp("RI", RI_PIN)
	s += pp("CD", CD_PIN)
	s += pp("DSR", DSR_PIN)
	s += pp("RTS", RTS_PIN)
	s += pp("DTR", DTR_PIN)
	fmt.Println(s)

	pl := func (n string, p rpio.Pin) (string) {
		if p.Read() == rpio.High {	// LED is on
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

// Pin functions
func (m *Modem) led_MR_on() {
	m.pins[MR_LED].High()
}
func (m *Modem) led_MR_off() {
	m.pins[MR_LED].Low()
}
func (m *Modem) led_AA_on() {
	m.pins[AA_LED].High()
}
func (m *Modem) led_AA_off() {
	m.pins[AA_LED].Low()
}
func(m *Modem) led_OH_on() {
	m.pins[OH_LED].High()
}
func(m *Modem) led_OH_off() {
	m.pins[OH_LED].Low()
}
func(m *Modem) led_TR_on() {
	m.pins[TR_LED].High()
}
func(m *Modem) led_TR_off() {
	m.pins[TR_LED].Low()
}
func(m *Modem) led_CS_on() {
	m.pins[CS_LED].High()
}
func(m *Modem) led_CS_off() {
	m.pins[CS_LED].Low()
}
func (m *Modem) led_SD_on() {
	m.pins[SD_LED].High()
}
func (m *Modem) led_SD_off() {
	m.pins[SD_LED].Low()
}
func (m *Modem) led_RD_on() {
	m.pins[RD_LED].High()
}
func (m *Modem) led_RD_off() {
	m.pins[RD_LED].Low()
}
func (m *Modem) led_CD_on() {
	m.pins[CD_LED].High()
}
func (m *Modem) led_CD_off() {
	m.pins[CD_LED].Low()
}

// PINs

// RI
func (m *Modem) raiseRI() {
	m.pins[RI_PIN].High()
}
func (m *Modem) lowerRI() {
	m.pins[RI_PIN].Low()
}
func (m *Modem) readRI() (bool) {
	return m.pins[RI_PIN].Read() == rpio.High
}

// CD
func (m *Modem) raiseCD() {
	m.pins[CD_PIN].High()
}
func (m *Modem) lowerCD() {
	m.pins[CD_PIN].Low()
}
func (m *Modem) readCD() (bool) {
	return m.pins[CD_PIN].Read() == rpio.High
}

// DSR
func (m *Modem) raiseDSR() {
	m.pins[DSR_PIN].High()
}
func (m *Modem) lowerDSR() {
	m.pins[DSR_PIN].Low()
}
func (m *Modem) readDSR() (bool) {
	return m.pins[DSR_PIN].Read() == rpio.High
}

// CTS
func (m *Modem) raiseCTS() {
	m.pins[CTS_PIN].High()
}
func (m *Modem) lowerCTS() {
	m.pins[CTS_PIN].Low()
}
func (m *Modem) readCTS() (bool) {
	return m.pins[CTS_PIN].Read() == rpio.High
}

// DTR
func (m *Modem) readDTR() (bool) {
	return m.pins[DTR_PIN].Read() == rpio.High
	// Debugging
	// return m.r[200] == 1
}

// RTS
func (m *Modem) readRTS() (bool) {
	return m.pins[RTS_PIN].Read() == rpio.High
	// Debugging
	// return m.r[200] == 1
}
