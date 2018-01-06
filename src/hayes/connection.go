package main

import (
	"code.cloudfoundry.org/bytefmt"
	"net"
	"time"
)

var last_ring_time time.Time

type busyFunc func() bool

// Is the network connection inbound or outbound
const (
	INBOUND = iota
	OUTBOUND
)

// Interface specification for a connection
type connection interface {
	Read(p []byte) (int, error)
	Write(p []byte) (int, error)
	Close() error
	RemoteAddr() net.Addr
	Direction() int // INBOUND or OUTBOUND
	Mode() bool      // What command mode to be in after connection
	SetMode(bool)
	Stats() (uint64, uint64)
	String() string
	SetDeadline(t time.Time) error
}

func answerIncomming(conn connection) bool {
	const __DELAY_MS = 20

	zero := make([]byte, 1)
	zero[0] = 0

	r := registers
	for i := 0; i < __MAX_RINGS; i++ {
		last_ring_time = time.Now()
		conn.Write([]byte("Ringing...\n\r"))
		logger.Print("Ringing")
		if offHook() { // computer has issued 'ATA'
			netConn = conn
			conn = nil
			goto answered
		}

		// Simulate the "2-4" pattern for POTS ring signal (2
		// seconds of high voltage ring signal, 4 seconds
		// of silence)

		// Ring for 2s
		d := 0
		raiseRI()
		for onHook() && d < 2000 {
			if _, err := conn.Write(zero); err != nil {
				goto no_answer
			}
			time.Sleep(__DELAY_MS * time.Millisecond)
			d += __DELAY_MS
			if offHook() { // computer has issued 'ATA'
				netConn = conn
				conn = nil
				goto answered
			}
		}
		lowerRI()

		// By verification, the Hayes Ultra 96 displays the
		// "RING" text /after/ the RI signal is lowered.  So
		// do this here so we behave the same.
		serial.Println(RING)

		// If Auto Answer is enabled and we've exceeded the
		// configured number of rings to wait before
		// answering, answer the call.  We do this here before
		// the 4s delay as I think it feels more correct.
		ringCount := r.Inc(REG_RING_COUNT)
		aaCount := r.Read(REG_AUTO_ANSWER)
		if aaCount > 0 {
			if ringCount >= aaCount {
				logger.Print("Auto answering")
				answer()
			}
		}

		// Silence for 4s
		d = 0
		for onHook() && d < 4000 {
			// Test for closed connection
			if _, err := conn.Write(zero); err != nil {
				goto no_answer
			}

			time.Sleep(__DELAY_MS * time.Millisecond)
			d += __DELAY_MS
			if offHook() { // computer has issued 'ATA'
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
	return false

answered:
	// if we're here, the computer answered.
	logger.Print("Answered")
	conn.Write([]byte("Answered\n\r"))
	registers.Write(REG_RING_COUNT, 0)
	lowerRI()
	return true
}

func startAcceptingCalls() {
	started_ok := make(chan error)

	if flags.skipTelnet {
		logger.Print("Telnet server not started by command line flag")
	} else {
		go acceptTelnet(callChannel, checkBusy, logger, started_ok)
		if err := <-started_ok; err != nil {
			logger.Printf("Telnet server failed to start: %s", err)
		} else {
			logger.Print("Telnet server started")
		}
	}


	if flags.skipSSH {
		logger.Print("SSH server not started by command line flag")
	} else {
		go acceptSSH(callChannel, flags.privateKey, checkBusy, logger,
			started_ok)
		if err := <-started_ok; err != nil {
			logger.Printf("SSH server failed to start: %s", err)
		} else {
			logger.Print("SSH server started")
		}
	}
}


// Clear the ring counter after 8s
// Must be a goroutine
func clearRingCounter() {
	var delay time.Duration = 8 * time.Second

	for _ = range time.Tick(delay) {
		if time.Since(last_ring_time) >= delay &&
			registers.Read(REG_RING_COUNT) != 0 {
			registers.Write(REG_RING_COUNT, 0)
			logger.Print("Cleared ring count")
		}
	}
}

// Pass bytes from the remote dialer to the serial port (for now,
// stdout) as long as we're offhook, we're in DATA MODE and we have
// valid carrier (m.comm != nil)
func serviceConnection() {
	var t time.Time
	var timeout time.Duration

	logger.Printf("Servicing connection with remote %s", netConn.RemoteAddr())

	buf := make([]byte, 1)
	for {
		// If S30 is non-zero, set a timeout
		b := registers.Read(REG_INACTIVITY_TIMER)
		timeout = time.Duration(b) * 10 * time.Second
		if timeout == time.Duration(0) {
			t = time.Time{}
		} else {
			t = time.Now().Add(timeout)
		}
		if err := netConn.SetDeadline(t); err != nil {
			logger.Printf("netConn.SetDeadline(): %s", err)
			return
		}
		
		if _, err := netConn.Read(buf); err != nil { // carrier lost
			nerr, ok := err.(net.Error)
			switch {
			case ok && nerr.Timeout():
				logger.Printf("netConn.Read(): S30 timeout: %s",
					timeout)
			case ok && nerr.Temporary():
				logger.Printf("netConn.Read(): temporary errory")
				continue
			default: 
				logger.Print("netConn.Read(): ", err)
			}
			return
		}

		if m.dcd == false {
			logger.Print("netConn.Read(): No carrier at network read")
			return
		}

		if onHook() {
			logger.Print("netConn.Read(): On hook at network read")
			return
		}

		// Send the byte to the DTE, blink the RD LED
		if m.mode == DATAMODE {
			led_RD_on()
			serial.Write(buf)
			led_RD_off()
		}
	}
}

// Accept connection's from dial*() and accept*() functions.
func handleCalls() {
	go clearRingCounter()
	startAcceptingCalls()

	// Wait for a connection.  If it's an incoming call, answer
	// it.  If it's an outgoing call or an answered incoming call,
	// service it
	var conn connection
	for {
		conn = <-callChannel
		setLineBusy(true)

		switch conn.Direction() {
		case INBOUND:
			logger.Printf("Incomming call from %s", conn.RemoteAddr())
			if !answerIncomming(conn) {
				conn.Close()
				continue
			}
		case OUTBOUND:
			logger.Printf("Outgoing call to %s ", conn.RemoteAddr())
		}

		// We now have an established connection (either answered or dialed)
		// so service it.
		netConn = conn
		m.mode = conn.Mode()
		m.connectSpeed = 38400
		m.dcd = true	// Force DCD "up" here.
		serviceConnection()

		// If we're here, we lost "carrier" somehow.
		sent, recv := conn.Stats()
		goOnHook()
		setLineBusy(false)
		serial.Printf("\n")
		prstatus(NO_CARRIER)
		logger.Printf("Connection closed, sent %s recv %s",
			bytefmt.ByteSize(sent), bytefmt.ByteSize(recv))

	}
}
