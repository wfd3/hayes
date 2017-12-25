// +build !arm

package main

// Support for generic hardare (ie, not a Raspberry Pi)

import (
	"runtime"
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

	_PIN_LEN // This needs to be last in the const list
)

type hwPins [_PIN_LEN]bool

var leds hwPins
var pins hwPins

func setupPins() {
	logger.Printf("Simulated Pins enabled on %s/%s\n",
		runtime.GOOS, runtime.GOARCH)

	clearPins()

	// The DTE is always ready
	pins[DTR_PIN] = true
	pins[RTS_PIN] = true
}

func clearPins() {
	for i := range leds {
		leds[i] = false
	}
	for i := range pins {
		pins[i] = false
	}
}

func showPins() string {

	pp := func(n string, p int) string {
		var s string
		if pins[p] {
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

	pl := func(n string, p int) string {
		var s string
		if leds[p] { // LED is on
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
func led_HS_on() {
	leds[HS_LED] = true
}
func led_HS_off() {
	leds[HS_LED] = false
}

func led_AA_on() {
	leds[AA_LED] = true
}
func led_AA_off() {
	leds[AA_LED] = false
}

func led_OH_on() {
	leds[OH_LED] = true
}
func led_OH_off() {
	leds[OH_LED] = false
}

func led_TR_on() {
	leds[TR_LED] = true
}
func led_TR_off() {
	leds[TR_LED] = false
}

func led_SD_on() {
	leds[SD_LED] = true
}
func led_SD_off() {
	leds[SD_LED] = false
}

func led_RD_on() {
	leds[RD_LED] = true
}
func led_RD_off() {
	leds[RD_LED] = false
}

func ledTest(i int) {
	// NOOP
}

// PINs

// RI - Ring Indicator
func raiseRI() {
	pins[RI_PIN] = true
}
func lowerRI() {
	pins[RI_PIN] = false
}
func readRI() bool {
	return pins[RI_PIN]
}

// CD - Carrier Detect
func raiseCD() {
	leds[CD_LED] = true
	pins[CD_PIN] = true
}
func lowerCD() {
	leds[CD_LED] = false
	pins[CD_PIN] = false
}
func readCD() bool {
	return pins[CD_PIN]
}

// DSR - Data Set Ready
func raiseDSR() {
	leds[MR_LED] = true
	pins[DSR_PIN] = true
	logger.Print("raiseDSR()")
}
func lowerDSR() {
	if !conf.dsrControl {
		return
	}
	leds[MR_LED] = false
	pins[DSR_PIN] = false
	logger.Print("lowerDSR()")
}
func readDSR() bool {
	return pins[DSR_PIN]
}

// CTS - Clear to Send
func raiseCTS() {
	leds[CS_LED] = true
	pins[CTS_PIN] = true
	logger.Print("raiseCTS()")
}
func lowerCTS() {
	leds[CS_LED] = true
	pins[CTS_PIN] = false
	logger.Print("lowerCTS()")
}
func readCTS() bool {
	return pins[CTS_PIN]
}

// DTR - Data Terminal Ready (input)
func readDTR() bool {
	// Is the computer ready to send data?
	return pins[DTR_PIN]
}

// RTS - Request to Send (input)
func readRTS() bool {
	// Has the computer requested data be sent?
	return pins[RTS_PIN]
}
