package main

import (
	"time"
)

var dtrchan chan bool

// Watch a subset of pins and/or config, and act as apropriate
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

		// Check carrier, check CD LED
		if m.dcd || conf.dcdControl {
			raiseCD()
		} else {
			lowerCD()
		}
	}
}

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
		
		// Has the S25 register changed?
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
			logger.Printf("Waiting for DTR to go up at %s",
				now.Format(time.StampMilli))
			continue
		}

		switch lastb {
		case true:	// DTR was up last time we looped
			logger.Printf("DTR down at %s", now.Format(time.StampMilli))
			startDown = now
			lastb = false
			
		case false:	// DTR was down last time we looped
			down := now.Sub(startDown)
			logger.Printf("DTR down for %s", down)
			if down >= S25time {
				logger.Print("Triggering processDTR()")
				dtrchan <- true
				waitForUp = true
			}
		}
	}
}

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
				goOnHook()
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

	dtrchan = make(chan bool)
	go handlePINs()    // Monitor input pins & internal registers
	go handleDTR()
	go processDTR()
}
