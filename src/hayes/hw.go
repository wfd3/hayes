package main

import (
	"time"
)

var dtrchan chan bool // "Interrupts" for DTR/S25 interactions

// Watch a subset of pins and/or config, and act as apropriate. 
// Must be a goroutine
func handlePINs() {
	t := time.Tick(100 * time.Millisecond)
	for range t {
		// Check connect speed, set HS LED
		switch {
		case m.connectSpeed > 19200:
			led_HS_on()
		default:
			led_HS_off()
		}

		// Check carrier, set CD LED
		if conf.dcdPinned { // DCD is pinned high
			raiseCD()
		} else {
			switch m.dcd { // DCD is set by m.dcd
			case true:  raiseCD()
			case false: lowerCD()
			}
		}

		// Check dsrPinned
		if conf.dsrPinned { // DSR is pinned high
			raiseDSR() 
		} else {	// DSR high between DCD and Hangup
			switch  m.dcd {
			case true:  raiseDSR()
			case false: lowerDSR()
			}
		}
	}
}

// Handles DTR behavior as specified by &D and S25
func handleDTR() {
	var d byte
	var lastb, waitForUp bool
	var startDown time.Time
	var S25time time.Duration

	lastb = false
	waitForUp = true
	startDown = time.Now()

	t := time.Tick(5 * time.Millisecond)
	for now := range t { 
		dt := registers.Read(REG_DTR_DETECTION_TIME)
		if d != dt {
			d = dt
			// REG_DTR_DETECTION_TIME is in 1/100ths of a second (10ms)
			S25time = time.Duration(float64(d) * 10 ) * time.Millisecond
			logger.Printf("DTR detection window: %s", S25time)
		}
		
		if readDTR() {
			led_TR_on()
			if lastb == false {
				logger.Printf("DTR up at %s, down for %s total",
					now.Format(time.StampMilli),
					now.Sub(startDown))
			}
			lastb = true
			waitForUp = false
			continue
		}

		//
		// We know that DTR is down from here.
		//
		
		if waitForUp {	// Wait for DTR to have cycled
			continue
		}

		switch lastb {
		case true:	// DTR was up last time we looped
			logger.Printf("DTR down at %s", now.Format(time.StampMilli))
			startDown = now
			lastb = false
			
		case false:	// DTR was down last time we looped
			down := now.Sub(startDown)
			if down >= S25time {
				logger.Print("Triggering processDTR()")
				waitForUp = true
				dtrchan <- true
			}
		}
	}
}

// "Interrupt Handler" for DTR
func processDTR() {
	// If DTR is down, do what conf.dtr says:
	for {
		<-dtrchan
		switch conf.dtr {
		case 0:	// Do nothing, make sure LED is correct
			logger.Print("DTR Toggled, &D0")
			led_TR_off()
			
		case 1:
			led_TR_on()
			logger.Print("DTR toggeled, &D1")
			if m.mode == DATAMODE {
				m.mode = COMMANDMODE
				prstatus(OK)
			}
			
		case 2:
			logger.Print("DTR toggled, &D2")
			led_TR_off()
			if offHook() {
				status := goOnHook()
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
}

func setupHW() {
	go handlePINs()    // Monitor input pins & internal registers
	dtrchan = make(chan bool)
	go handleDTR()	   // Watch DTR, sent "interrupts"
	go processDTR()	   // Catch DTR "interrupts"
}
