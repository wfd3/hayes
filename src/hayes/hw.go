package main

import (
	"time"
)


// Clear the ring counter after 8s
// Must be a goroutine
func clearRingCounter() {
	delay := 8 * time.Second
	for range time.Tick(delay) {
		if time.Since(m.getLastRingTime()) >= delay {
			registers.Write(REG_RING_COUNT, 0)
		}
	}
}

func clearLCDNoCarier() {
	delay := 10 * time.Second
	for range time.Tick(delay) {
		if last_error == NO_CARRIER &&
			time.Since(last_error_time) >= delay {
			lcd.Printf(1, "OK")
			last_error = OK
			last_error_time = time.Now()
		}
	}
}

// Watch a subset of pins and/or config, and act as apropriate. 
// Must be a goroutine
func handlePins() {

	for range time.Tick(250 * time.Millisecond) {

		// Check connect speed, set HS LED
		switch {
		case m.getConnectSpeed() > 19200:
			led_HS_on()
		default:
			led_HS_off()
		}
		
		// Check carrier, set CD LED
		if conf.dcdPinned { // DCD is pinned high
			raiseCD()
		} else {
			switch m.getdcd() { // DCD is set by m.dcd
			case true:  raiseCD()
			case false: lowerCD()
			}
		}
		
		// Check dsrPinnedd
		if conf.dsrPinned { // DSR is pinned high
			raiseDSR() 
		} 
	}
}

// Handles DTR behavior as specified by &D and S25
func handleDTR() {
	var d byte
	var wasUp, waitForUp bool
	var startDown time.Time
	var S25time time.Duration

	wasUp = false
	waitForUp = true
	startDown = time.Now()

	for now := range time.Tick(5 * time.Millisecond) {

		// First, see if the DTR detection time has changed
		dt := registers.Read(REG_DTR_DETECTION_TIME)
		if d != dt {
			d = dt
			// REG_DTR_DETECTION_TIME is in 1/100ths of a second (10ms)
			S25time = time.Duration(float64(d) * 10 ) * time.Millisecond
			logger.Printf("DTR detection window: %s", S25time)
		}

		if readDTR() {
			if !wasUp {
				logger.Printf("DTR up, down for %s total",
					now.Sub(startDown))
			}
			wasUp = true
			waitForUp = false
			led_TR_on()
			continue
		}

		//
		// We know that DTR is down from here.
		//
		
		if waitForUp {	// Wait for DTR to have cycled
			continue
		}

		switch wasUp {
		case true:	// DTR was up last time we looped
			logger.Print("DTR down")
			startDown = now
			wasUp = false
			
		case false:	// DTR was down last time we looped
			down := now.Sub(startDown)
			if down >= S25time {
				logger.Print("Triggering processDTR()")
				waitForUp = true
				processDTR()
			}
		}
	}
}

// If DTR is down, do what conf.dtr says:
func processDTR() {
	switch conf.dtr {
	case 0:	// Do nothing, make sure LED is correct
		logger.Print("DTR Toggled, &D0")
		led_TR_off()
		
	case 1:
		led_TR_on()
		logger.Print("DTR toggeled, &D1")
		if m.getMode() == DATAMODE {
			m.setMode(COMMANDMODE)
			prstatus(OK)
		}
		
	case 2:
		logger.Print("DTR toggled, &D2")
		led_TR_off()
		if m.offHook() {
			status := hangup()
			prstatus(status)
		}
		
	case 3:	// Reset modem
		logger.Print("DTR toggled, &D3") 
		err := softReset(m.currentConfig)
		if err != nil {
			logger.Print("softReset() error: %s", err)
		}
		prstatus(err)
	}
}

func setupHW() {
	go clearRingCounter()
	go handlePins()
	go handleDTR()
	go clearLCDNoCarier()
}
