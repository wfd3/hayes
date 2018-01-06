// +build arm

package main

import (
	"github.com/stianeikeland/go-rpio"
	"strings"
	"time"
)

// This assumes the MAX3232 does NOT do the level conversion between the Pi's
// 0 and 3V low/high and RS-232 +5V/-5V.  So a "low" pin here is a High RPi output
// vice versa.
//
// So note that the LED pins are normal, and the control pins (RTS, CTS, etc.)
// are backwards (eg, pin.Low() means RS232 High and pin.High() means RS232 Low)

type hwPins map[int]rpio.Pin

var leds hwPins
var pins hwPins

// LED and data pins
const (
	// LEDs - controlled in handleLeds()
	HS_LED = 2  // Connected at High speed (conf.speed > 14400)
	AA_LED = 3  // Auto Answer configured (conf.r[0] > 0)
	TR_LED = 9  // Terminal Ready (turn on if read(DTR) is high)
	OH_LED = 27 // Is the modem off hook (m.offHook() == true)

	// Receive and Send Data LEDs.  Manually controlled
	RD_LED = 10 // Receive Data
	SD_LED = 22 // Send Data

	// Data Pins
	// A MAX3232 translates these from 0V & 3V to RS232 -/+{3,5,12}V
	CTS_PIN = 12 // Clear To Send pin
	CS_LED  = 11 // Clear To Send LED

	RI_PIN = 23 // Ring Indicator pin
	RI_LED = 4  // Ring Indicator LED

	CD_PIN = 24 // Carrier Detect pin
	CD_LED = 17 // Carrier Detect LED

	DSR_PIN = 25 // Data Set Ready pin
	MR_LED  = 5  // Modem Ready LED

	RTS_PIN = 7  // Request to Send pin (Input)
	DTR_PIN = 16 // Data Terminal Ready (Input)
)

func setupPins() {

	logger.Print("Setting up RPi pins")
	if err := rpio.Open(); err != nil {
		logger.Fatal("Fatal Error: ", err)
	}

	// LEDs
	leds = make(hwPins)

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
	pins = make(hwPins)

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

}

func clearPins() {
	leds[HS_LED].Low()
	leds[AA_LED].Low()
	leds[RI_LED].Low()
	leds[MR_LED].Low()
	leds[TR_LED].Low()
	leds[RD_LED].Low()
	leds[CS_LED].Low()
	leds[CD_LED].Low()
	leds[SD_LED].Low()
	leds[OH_LED].Low()

	pins[RI_PIN].High()
	pins[CD_PIN].High()
	pins[DSR_PIN].High()
	pins[CTS_PIN].High()
	// No need to do RTS and DTR
}

func showPins() string {
	pp := func(n string, pin rpio.Pin, up rpio.State) string {
		var s string
		if pin.Read() == up {
			s = strings.ToUpper(n)
		} else {
			s = strings.ToLower(n)
		}
		s += " "
		return s
	}

	s := "PINs: ["
	s += pp("CTS", pins[CTS_PIN], rpio.Low)
	s += pp("RI_", pins[RI_PIN], rpio.Low)
	s += pp("DCD", pins[CD_PIN], rpio.Low)
	s += pp("DSR", pins[DSR_PIN], rpio.Low)
	s += pp("RTS", pins[RTS_PIN], rpio.Low)
	s += pp("DTR", pins[DTR_PIN], rpio.Low)
	s += "]"

	s += "\n"

	s += "LEDs: "
	s += pp("HS", leds[HS_LED], rpio.High)
	s += pp("AA", leds[AA_LED], rpio.High)
	s += pp("RI", leds[RI_LED], rpio.High)
	s += pp("CD", leds[CD_LED], rpio.High)
	s += pp("OH", leds[OH_LED], rpio.High)
	s += pp("MR", leds[MR_LED], rpio.High)
	s += pp("CS", leds[CS_LED], rpio.High)
	s += pp("TR", leds[TR_LED], rpio.High)
	s += pp("SD", leds[SD_LED], rpio.High)
	s += pp("RD", leds[RD_LED], rpio.High)
	s += "]"
	return s
}

// Led functions
func led_HS_on() {
	leds[HS_LED].High()
}
func led_HS_off() {
	leds[HS_LED].Low()
}

func led_AA_on() {
	leds[AA_LED].High()
}
func led_AA_off() {
	leds[AA_LED].Low()
}

func led_OH_on() {
	leds[OH_LED].High()
}
func led_OH_off() {
	leds[OH_LED].Low()
}

func led_TR_on() {
	leds[TR_LED].High()
}
func led_TR_off() {
	leds[TR_LED].Low()
}

func led_SD_on() {
	leds[SD_LED].High()
}
func led_SD_off() {
	leds[SD_LED].Low()
}

func led_RD_on() {
	leds[RD_LED].High()
}
func led_RD_off() {
	leds[RD_LED].Low()
}

func ledTest(round int) {
	var saved_leds map[int]rpio.State

	saved_leds = make(map[int]rpio.State)

	// Turn them all on, wait a bit, turn them all off.
	for i := range leds {
		saved_leds[i] = leds[i].Read() // Save current state
		leds[i].High()
		time.Sleep(50 * time.Millisecond)
	}
	time.Sleep(500 * time.Millisecond)
	for i := range leds {
		leds[i].Low()
		time.Sleep(50 * time.Millisecond)
	}
	time.Sleep(500 * time.Millisecond)

	// Randomly (based on how range works) turn on and off round times
	for j := 0; j < round; j++ {
		for i := range leds {
			leds[i].High()
			time.Sleep(50 * time.Millisecond)
		}
		time.Sleep(10 * time.Millisecond)
		for i := range leds {
			leds[i].Low()
			time.Sleep(50 * time.Millisecond)
		}
	}

	// Restore LED state
	for j := range saved_leds {
		leds[j].Write(saved_leds[j])
	}

}

// PINs

// RI - assert RI and turn on RI light
func raiseRI() {
	leds[RI_LED].High()
	pins[RI_PIN].Low()
}
func lowerRI() {
	leds[RI_LED].Low()
	pins[RI_PIN].High()
}
func readRI() bool {
	return pins[RI_PIN].Read() == rpio.Low
}

// CD - assert CD and turn on CD light
func raiseCD() {
	leds[CD_LED].High()
	pins[CD_PIN].Low()
}
func lowerCD() {
	leds[CD_LED].Low()
	pins[CD_PIN].High()
}
func readCD() bool {
	return pins[CD_PIN].Read() == rpio.Low
}

// DSR - assert DSR and turn on MR light
func raiseDSR() {
	leds[MR_LED].High()
	pins[DSR_PIN].Low()
}

func lowerDSR() {
	leds[MR_LED].Low()
	pins[DSR_PIN].High()
}
func readDSR() bool {
	return pins[DSR_PIN].Read() == rpio.Low
}

// CTS - assert CTS and turn on CS light
func raiseCTS() {
	leds[CS_LED].High()
	pins[CTS_PIN].Low()
}
func lowerCTS() {
	leds[CS_LED].Low()
	pins[CTS_PIN].High()
}
func readCTS() bool {
	return pins[CTS_PIN].Read() == rpio.Low
}

// DTR (input)
func readDTR() bool {
	return pins[DTR_PIN].Read() == rpio.Low
}

// RTS (input)
func readRTS() bool {
	return pins[RTS_PIN].Read() == rpio.Low
}
