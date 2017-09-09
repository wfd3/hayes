package hayes

import (
	"io"
	"net"
	"time"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
)

const __MAX_RINGS = 15		// How many rings before giving up

var connection chan io.ReadWriteCloser
var last_ring_time time.Time

// Telnet negoitation commands
const (
	IAC = 0377
	DO = 0375
	WILL = 0373
	WONT = 0374
	ECHO = 0001
	LINEMODE = 0042
)

func (m *Modem) acceptSSH() {

	// In the latest version of crypto/ssh (after Go 1.3), the SSH
	// server type has been removed in favour of an SSH connection
	// type. A ssh.ServerConn is created by passing an existing
	// net.Conn and a ssh.ServerConfig to ssh.NewServerConn, in
	// effect, upgrading the net.Conn into an ssh.ServerConn

	config := &ssh.ServerConfig{
		// You may also explicitly allow anonymous client
		// authentication, though anon bash sessions may not
		// be a wise idea
		NoClientAuth: true,
	}

	// You can generate a keypair with 'ssh-keygen -t rsa'
	private_key := "id_rsa"
	privateBytes, err := ioutil.ReadFile(private_key)
	if err != nil {
		m.log.Fatalf("Fatal Error: failed to load private key (%s)\n",
			private_key)
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		m.log.Fatal("Fatal Error: failed to parse private key")
	}

	config.AddHostKey(private)

	// Once a ServerConfig has been configured, connections can be accepted.
	listener, err := net.Listen("tcp", "0.0.0.0:2200")
	if err != nil {
		m.log.Fatal("Fatal Error: ", err)
	}

	// Accept all connections
	var conn ssh.Channel
	var newChannel ssh.NewChannel
	for {
		tcpConn, err := listener.Accept()
		if err != nil {
			m.log.Print("Failed to accept incoming connection (%s)",
				err)
			continue
		}
		// Before use, a handshake must be performed on the
		// incoming net.Conn.
		sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, config)
		if err != nil {
			m.log.Print("Failed to handshake (%s)", err)
			continue
		}
		go ssh.DiscardRequests(reqs)

		m.log.Printf("New SSH connection from %s (%s)\n",
			sshConn.RemoteAddr(), sshConn.ClientVersion())

		for newChannel = range chans {

			if newChannel.ChannelType() != "session" {
				newChannel.Reject(ssh.UnknownChannelType,
					"unknown channel type")
				continue
			} 

			conn, _, err = newChannel.Accept()
			if err != nil {
				m.log.Fatal("Fatal Error: ", err)
			}

			if checkBusy(m, conn) {
				conn.Close()
				continue
			}
			connection <- conn
			break
		}
	}
}
	
func (m *Modem) acceptTelnet() {
	l, err := net.Listen("tcp", ":20000")
	if err != nil {
		m.log.Fatal("Fatal Error: ", err)
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			m.log.Print("l.Accept(): %s\n", err)
			continue
		}

		if checkBusy(m, conn) {
			conn.Close()
			continue
		}
		
		// This is a telnet session, negotiate char-at-a-time
		conn.Write([]byte{IAC, DO, LINEMODE, IAC, WILL, ECHO})

		connection <- conn
	}
}

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
		if buf[0] == IAC {
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

func (m *Modem) answerIncomming(conn io.ReadWriteCloser) bool {
	const __DELAY_MS = 20

	zero := make([]byte, 1)
	zero[0] = 0

	for i := 0; i < __MAX_RINGS; i++ {
		last_ring_time = time.Now()
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
			r := m.registers
			if r.Inc(REG_RING_COUNT) >= r.Read(REG_AUTO_ANSWER) {
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

// "Busy" signal.
func checkBusy(m *Modem, conn io.ReadWriteCloser) bool {
	if m.getHook() == OFF_HOOK || m.getLineBusy() {	
		conn.Write([]byte("BUSY\n\r"))
		return true
	}
	return false
}

func (m *Modem) handleModem() {
	var conn io.ReadWriteCloser

	connection = make(chan io.ReadWriteCloser, 1)
	go m.acceptTelnet()
	go m.acceptSSH()

	// Clear the ring counter if there's been no rings for at least 8 seconds
	last_ring_time = time.Now()
	go func() {		
		for range time.Tick(8 * time.Second) {
			if time.Since(last_ring_time) >= 8 * time.Second {
				m.registers.Write(REG_RING_COUNT, 0) 
			}
		}
	}()

	// If we have an incoming call, answer it.  If we have an outgoing call or
	// an answered incoming call, service the connection
	for {
		conn = nil
		select {
		case conn = <- connection:
			m.log.Print("Incomming call")
		default: 
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

