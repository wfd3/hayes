package hayes

import (
	"time"
)

const __MAX_RINGS = 15		// How many rings before giving up
var last_ring_time time.Time

// Pass bytes from the remote dialer to the serial port (for now,
// stdout) as long as we're offhook, we're in DATA MODE and we have
// valid carrier (m.comm != nil)
func (m *Modem) handleConnection() {

	buf := make([]byte, 1)

	for {
		if m.getHook() == ON_HOOK {
			m.log.Print("ON_HOOK")
			break
		}
		if _, err := m.conn.Read(buf); err != nil {// TODO: timeout
			m.log.Print("m.conn.Read(): ", err)
			// carrier lost
			break
		}

		// Tell the telnet server we won't comply.
		// TODO: Clean this up
		if m.conn.Type() == TELNET && buf[0] == IAC {
			cmd := make([]byte, 2)
			if _, err := m.conn.Read(cmd); err != nil {
				m.log.Print("m.conn.Read(): ", err)
				break;
			}
			m.log.Print("Telnet negotiation command: %v", cmd)
			m.conn.Write([]byte{IAC, WONT, cmd[1]})
			continue
		}
		

		// Send the byte to the DTE
		if m.mode == DATAMODE {
			// TODO: try 'go m.blinkRD()'
			m.led_RD_on()
			m.serial.Write(buf) 
			m.led_RD_off()
		}
	}
	
	// If we're here, we lost "carrier" somehow.
	m.log.Print("Lost carrier")
	m.prstatus(NO_CARRIER)
	m.onHook()
	if m.conn != nil {
		m.conn.Close() // just to be safe?
	}
}

func (m *Modem) answerIncomming(conn connection) bool {
	const __DELAY_MS = 20

	zero := make([]byte, 1)
	zero[0] = 0

	r := m.registers
	for i := 0; i < __MAX_RINGS; i++ {
		last_ring_time = time.Now()
		r.Inc(REG_RING_COUNT)
		m.prstatus(RING)
		conn.Write([]byte("Ringing...\n\r"))
		if m.getHook() == OFF_HOOK { // computer has issued 'ATA' 
			m.conn = conn
			conn = nil
			goto answered
		}
		
		// Simulate the "2-4" pattern for POTS ring signal (2
		// seconds of high voltage ring signal, 4 seconds
		// of silence)
		
		// Ring for 2s
		d := 0
		m.raiseRI()
		for m.getHook() == ON_HOOK  && d < 2000 {
			if _, err := conn.Write(zero); err != nil {
				goto no_answer
			}
			time.Sleep(__DELAY_MS * time.Millisecond)
			d += __DELAY_MS
			if m.getHook() == OFF_HOOK { // computer has issued 'ATA' 
				m.conn = conn
				conn = nil
				goto answered
			}
		}
		m.lowerRI()
		
		// If Auto Answer if enabled and we've exceeded the
		// configured number of rings to wait before
		// answering, answer the call.  We do this here before
		// the 4s delay as I think it feels more correct.
		if m.registers.Read(REG_AUTO_ANSWER) > 0 {
			if r.Read(REG_RING_COUNT) >= r.Read(REG_AUTO_ANSWER) {
				m.answer()
			}
		}
		
		// Silence for 4s
		d = 0
		for m.getHook() == ON_HOOK && d < 4000 {
			// Test for closed connection
			if _, err := conn.Write(zero); err != nil {
				goto no_answer
			}
			
			time.Sleep(__DELAY_MS * time.Millisecond)
			d += __DELAY_MS
			if m.getHook() == OFF_HOOK { // computer has issued 'ATA' 
				goto answered
			}
		}
	}
	
no_answer:
	// At this point we've not answered and have timed out, or the
	// caller hung up before we answered.
	if m.getHook() == ON_HOOK {	
		conn.Close()
	}
	m.lowerRI()
	return false
	
answered:
	// if we're here, the computer answered.
	m.registers.Write(REG_RING_COUNT, 0)
	m.lowerRI()
	return true
}

// Clear the ring counter after 8s
// Must be a goroutine
func (m *Modem) clearRingCounter() {
	var delay time.Duration = 8 * time.Second

	for _ = range time.Tick(delay) {
		if time.Since(last_ring_time) >= delay &&
			m.registers.Read(REG_RING_COUNT) != 0 {
			m.registers.Write(REG_RING_COUNT, 0)
			m.log.Print("Cleared ring count")
		}
	}
}

func (m *Modem) handleModem() {
	var conn connection

	connChannel := make(chan connection, 1)
	go m.acceptTelnet(connChannel)
	go m.acceptSSH(connChannel)
	last_ring_time = time.Now()
	go m.clearRingCounter()


	// If we have an incoming call, answer it.  If we have an outgoing call or
	// an answered incoming call, service the connection
	for {
		conn = nil
		select {
		case conn = <- connChannel:
			m.log.Print("Incomming call")
		}

		// Answer if incoming call (m.conn == nil, conn != nil)
		if conn != nil {
			if m.answerIncomming(conn) {
				// if we're here, the computer answered.
				m.conn = conn
				m.conn.Write([]byte("Answered\n\r"))
			}
		}

		// We now have an established connection (either answered or dialed)
		// so service it.
		if m.conn != nil {
			m.log.Print("Setting Line Busy, serving connection")
			m.setLineBusy(true)
			m.handleConnection()
			m.setLineBusy(false)
		}
	}
}

