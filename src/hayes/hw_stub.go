// +build !arm

package main

// Support for generic hardare (ie, not a Raspberry Pi)

import (
	"runtime"
	"strings"
	"sync"
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
var lock sync.RWMutex

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

	lock.RLock()
	defer lock.RUnlock()
	
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
	lock.Lock()
	leds[HS_LED] = true
	lock.Unlock()
}
func led_HS_off() {
	lock.Lock()
	leds[HS_LED] = false
	lock.Unlock()
}

func led_AA_on() {
	lock.Lock()
	leds[AA_LED] = true
	lock.Unlock()
}
func led_AA_off() {
	lock.Lock()
	leds[AA_LED] = false
	lock.Unlock()
}

func led_OH_on() {
	lock.Lock()
	leds[OH_LED] = true
	lock.Unlock()
}
func led_OH_off() {
	lock.Lock()
	leds[OH_LED] = false
	lock.Unlock()
}

func led_TR_on() {
	lock.Lock()
	leds[TR_LED] = true
	lock.Unlock()
}
func led_TR_off() {
	lock.Lock()
	leds[TR_LED] = false
	lock.Unlock()
}

func led_SD_on() {
	lock.Lock()
	leds[SD_LED] = true
	lock.Unlock()
}
func led_SD_off() {
	lock.Lock()
	leds[SD_LED] = false
	lock.Unlock()
}

func led_RD_on() {
	lock.Lock()
	leds[RD_LED] = true
	lock.Unlock()
}
func led_RD_off() {
	lock.Lock()
	leds[RD_LED] = false
	lock.Unlock()
}

func ledTest(i int) {
	// NOOP
}

// PINs

// RI - Ring Indicator
func raiseRI() {
	lock.Lock()
	pins[RI_PIN] = true
	lock.Unlock()
}
func lowerRI() {
	lock.Lock()
	pins[RI_PIN] = false
	lock.Unlock()
}
func readRI() bool {
	lock.RLock()
	defer lock.RUnlock()
	return pins[RI_PIN]
}

// CD - Carrier Detect
func raiseCD() {
	lock.Lock()
	leds[CD_LED] = true	
	pins[CD_PIN] = true
	lock.Unlock()

}
func lowerCD() {
	lock.Lock()
	leds[CD_LED] = false
	pins[CD_PIN] = false
	lock.Unlock()
}
func readCD() bool {
	lock.RLock()
	defer lock.RUnlock()
	return pins[CD_PIN]
}

// DSR - Data Set Ready
func raiseDSR() {
	lock.Lock()
	leds[MR_LED] = true
	pins[DSR_PIN] = true
	lock.Unlock()
}
func lowerDSR() {
	lock.Lock()
	leds[MR_LED] = false
	pins[DSR_PIN] = false
	lock.Unlock()
}
func readDSR() bool {
	lock.RLock()
	defer lock.RUnlock()
	return pins[DSR_PIN]
}

// CTS - Clear to Send
func raiseCTS() {
	lock.Lock()
	leds[CS_LED] = true
	pins[CTS_PIN] = true
	lock.Unlock()
}
func lowerCTS() {
	lock.Lock()
	leds[CS_LED] = true
	pins[CTS_PIN] = false	
	lock.Unlock()
}
func readCTS() bool {
	lock.RLock()
	defer lock.RUnlock()
	return pins[CTS_PIN]
}

// DTR - Data Terminal Ready (input)
func readDTR() bool {
	// Is the computer ready to send data?
	lock.RLock()
	defer lock.RUnlock()
	return pins[DTR_PIN]
}

// RTS - Request to Send (input)
func readRTS() bool {
	// Has the computer requested data be sent?
	lock.RLock()
	defer lock.RUnlock()
	return pins[RTS_PIN]
}
