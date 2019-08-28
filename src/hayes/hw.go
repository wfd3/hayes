package main

import (
	"time"
)


// Second granularity background tasks.  Currently. this clears the ring counter after 8s and resets the LCD display to 'READY'
// after a 'NO CARRIER' or 'BUSY' response after 10s
// Must be a goroutine
func secondTimer() {
	for range time.Tick(time.Second) {

		// Ring Count reset timer
		last_ring := m.getLastRingTime()
		if !last_ring.IsZero() && time.Since(last_ring) >= (8 * time.Second) {
			logger.Printf("Resetting REG_RING_COUNT")
			registers.Write(REG_RING_COUNT, 0)
			m.resetLastRingTime()
		}

		// LCD reset timer
		if (last_error == NO_CARRIER || last_error == BUSY) && (time.Since(last_error_time) >= (10 * time.Second)) {
			logger.Printf("Resetting LCD")
			lcd.Clear()
			lcd.Printf(1, "READY")
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
// Must be a goroutine
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
			logger.Printf("softReset() error: %s", err)
		}
		prstatus(err)
	}
}

func setupHW() {
	go secondTimer()
	go handlePins()
	go handleDTR()
}
