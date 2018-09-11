package main

import (
	"code.cloudfoundry.org/bytefmt"
	"net"
	"time"
)

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


// Pass bytes from the remote dialer to the serial port (for now,
// stdout) as long as we're offhook, we're in DATA MODE and we have
// valid carrier (m.comm != nil)
func serviceConnection() {
	var t time.Time
	var timeout time.Duration

	logger.Printf("Servicing connection with remote %s", m.conn.RemoteAddr())

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
		if err := m.conn.SetDeadline(t); err != nil {
			logger.Printf("conn.SetDeadline(): %s", err)
			return
		}
		
		if _, err := m.conn.Read(buf); err != nil { // Remote hung up or ...
			nerr, ok := err.(net.Error)	    // we timed out.
			switch {
			case ok && nerr.Timeout():
				logger.Printf("conn.Read(): triggered S30 timeout: %s",
					timeout)
			case ok && nerr.Temporary():
				logger.Printf("conn.Read(): temporary errory: %s",
				err)
				continue // Really? TODO
			default: 
				logger.Print("conn.Read(): ", err)
			}
			return
		}

		if m.dcd == false {
			logger.Print("conn.Read(): No carrier at network read")
			return
		}

		if onHook() {
			logger.Print("conn.Read(): On hook at network read")
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
	startAcceptingCalls()

	// Wait for a connection.  If it's an incoming call, answer
	// it.  If it's an outgoing call or an answered incoming call,
	// service it
	var conn connection
	for {
		lowerDSR()
		lowerCTS()
		setLineBusy(false)

		conn = <-callChannel

		setLineBusy(true)
		raiseDSR()
		raiseCTS()

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
		m.conn = conn
		m.mode = conn.Mode()
		m.connectSpeed = 38400
		m.dcd = true	// Force DCD "up" here.
		time.Sleep(250 * time.Millisecond)
		serviceConnection()

		if m.dcd == true { // User didn't hang up, so print status
			serial.Printf("\n")
			prstatus(NO_CARRIER)
		}
		sent, recv := m.conn.Stats()
		conn.Close()
		m.conn = nil
		hangup()
		logger.Printf("Connection closed, sent %s recv %s",
			bytefmt.ByteSize(sent), bytefmt.ByteSize(recv))

	}
}
