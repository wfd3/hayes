package main

import (
	"time"
)

// Simulate the phone line

// Is the phone on or off hook?
const (
	ONHOOK = false
	OFFHOOK = true
)

// How many rings before giving up
const __MAX_RINGS = 10

// How long to wait for the remote to answer.  6 seconds is the default
// ring-silence time
const __CONNECT_TIMEOUT = __MAX_RINGS * 6 * time.Second

// ATH0
func hangup() error {
	var ret error = OK
	
	m.dcdLow()
	lowerDSR()
	m.goOnHook()

	// It's OK to hang up the phone when there's no active network connection.
	// But if there is, close it.
	if m.conn != nil {
		logger.Printf("Hanging up on active connection (remote %s)",
			m.conn.RemoteAddr())
		m.conn.Close()
		ret = NO_CARRIER
	}

	m.setMode(COMMANDMODE)
	m.setConnectSpeed(0)
	m.setLineBusy(false)
	led_HS_off()
	led_OH_off()

       	if err := serial.Flush(); err != nil {
		logger.Printf("serial.Flush(): %s", err)
	}

	return ret
}

// ATH1
// Note that this will execute in a different context than answerIncoming()
func pickup() error {
	m.setLineBusy(true)
	m.goOffHook()
	led_OH_on()
	return OK
}

// "Busy" signal.
func checkBusy() bool {
	return m.offHook() || m.getLineBusy()
}

// Answer an incomming call.
func answerIncomming(conn connection) bool {
	const __DELAY_MS = 20

	zero := make([]byte, 1)

	r := registers
	for i := 0; i < __MAX_RINGS; i++ {
		m.setLastRingTime()
		conn.Write([]byte("Ringing...\n\r"))
		logger.Print("Ringing")
		if m.offHook() { // computer has issued 'ATA'
			goto answered
		}

		// Simulate the "2-4" pattern for POTS ring signal (2
		// seconds of high voltage ring signal, 4 seconds
		// of silence)

		// Ring for 2s
		d := 0
		raiseRI()
		for m.onHook() && d < 2000 {
			if _, err := conn.Write(zero); err != nil {
				goto no_answer
			}
			time.Sleep(__DELAY_MS * time.Millisecond)
			d += __DELAY_MS
			if m.offHook() { // computer has issued 'ATA'
				goto answered
			}
		}
		lowerRI()

		// By verification, the Hayes Ultra 96 displays the
		// "RING" text /after/ the RI signal is lowered.  Do
		// this here so we behave the same.
		serial.Println(RING)
		lcd.Printf(1, "RING %2d", i)
		lcd.Printf(2, "<%s", conn.RemoteAddr())


		// If Auto Answer is enabled and we've exceeded the
		// configured number of rings to wait before
		// answering, answer the call.  We do this here before
		// the 4s delay as I think it feels more correct.
		ringCount := r.Inc(REG_RING_COUNT)
		aaCount := r.Read(REG_AUTO_ANSWER)
		if aaCount > 0 {
			if ringCount >= aaCount {
				logger.Print("Auto answering")
				lcd.Printf(1, "Auto-answering")
				answer()
			}
		}

		// Silence for 4s
		d = 0
		for m.onHook() && d < 4000 {
			// Test for closed connection
			if _, err := conn.Write(zero); err != nil {
				goto no_answer
			}

			time.Sleep(__DELAY_MS * time.Millisecond)
			d += __DELAY_MS
			if m.offHook() { // computer has issued 'ATA'
				goto answered
			}
		}
	}

no_answer:
	// At this point we've not answered and have timed out, or the
	// caller hung up before we answered.
	logger.Print("No answer")
	conn.Write([]byte("No answer, closing connection\n\r"))
	lowerRI()
	prstatus(NO_ANSWER)
	return false

answered:
	// if we're here, the computer answered.
	logger.Print("Answered")
	conn.Write([]byte("Answered\n\r"))
	registers.Write(REG_RING_COUNT, 0)
	lowerRI()
	return true
}
