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

// Accept connection's from dial*() and accept*() functions.
func (m *Modem) handleModem() {
	var conn connection
	
	go m.acceptTelnet(callChannel)
	go m.acceptSSH(callChannel)
	go m.clearRingCounter()

	// If we have an incoming call, answer it.  If we have an outgoing call or
	// an answered incoming call, service the connection
	for {
		m.setLineBusy(false)
		conn = <- callChannel
		m.setLineBusy(true)

		if conn.Direction() == INBOUND {
			m.log.Printf("Incomming call from %s", conn.RemoteAddr())
			if !m.answerIncomming(conn) {
				conn.Close()
				continue
			}
			m.conn.Write([]byte("Answered\n\r"))
		} else {
			m.log.Printf("Outgoing call to %s ", m.conn.RemoteAddr())
		}

		// We now have an established connection (either answered or dialed)
		// so service it.
		m.conn = conn
		m.connect_speed = 38400
		m.mode = conn.Mode()
		m.raiseCD()
		m.handleConnection()
	}
}

