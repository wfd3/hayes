package hayes

import (
	"io"
	"net"
	"time"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"sync"
)

var connection chan io.ReadWriteCloser
var last_ring_time time.Time
var ringing bool
var ringlock sync.RWMutex

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
	privateBytes, err := ioutil.ReadFile("id_rsa")
	if err != nil {
		panic("Failed to load private key (./id_rsa)")
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		panic("Failed to parse private key")
	}

	config.AddHostKey(private)

	// Once a ServerConfig has been configured, connections can be accepted.
	listener, err := net.Listen("tcp", "0.0.0.0:2200")
	if err != nil {
		fmt.Printf("Failed to listen on 2200 (%s)", err)
		panic(err)
	}

	// Accept all connections
	var conn ssh.Channel
	var newChannel ssh.NewChannel
	for {
		tcpConn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Failed to accept incoming connection (%s)", err)
			continue
		}
		// Before use, a handshake must be performed on the
		// incoming net.Conn.
		sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, config)
		if err != nil {
			fmt.Printf("Failed to handshake (%s)", err)
			continue
		}
		go ssh.DiscardRequests(reqs)

		fmt.Printf("New SSH connection from %s (%s)\n",
			sshConn.RemoteAddr(), sshConn.ClientVersion())

		for newChannel = range chans {

			if newChannel.ChannelType() != "session" {
				newChannel.Reject(ssh.UnknownChannelType,
					"unknown channel type")
				continue
			} 

			conn, _, err = newChannel.Accept()
			if err != nil {
				panic(err)
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
		panic(err)
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			debugf("l.Accept(): %s\n", err)
			continue
		}

		if checkBusy(m, conn) {
			conn.Close()
			continue
		}
		
		// This is a telnet session, negotiate char-at-a-time
		const (
			IAC = 0377
			DO = 0375
			WILL = 0373
			ECHO = 0001
			LINEMODE = 0042
		)
		conn.Write([]byte{IAC, DO, LINEMODE, IAC, WILL, ECHO})

		connection <- conn
	}
}

// Pass bytes from the remote dialer to the serial port (for now,
// stdout) as long as we're offhook, we're in DATA MODE and we have
// valid carrier (m.comm != nil)
func (m *Modem) handleConnection() {

	buf := make([]byte, 1)

	for !m.onhook {
		if _, err := m.conn.Read(buf); err != nil {//timeout
			debugf("m.conn.Read(): %s", err)
			// carrier lost
			break
		}
		m.led_RD_on()
		if m.mode == DATAMODE {
			fmt.Printf("%s", string(buf)) //  Send to DTE
		}
		m.led_RD_off()
	}
	
	// If we're here, we lost "carrier" somehow.
	m.led_RD_off()
	m.prstatus(NO_CARRIER)
	m.onHook()
	if m.conn != nil {
		m.conn.Close() // just to be safe?
	}
}

func (m *Modem) answerIncomming(conn io.ReadWriteCloser) bool {
	zero := make([]byte, 1)
	zero[0] = 0

	for i := 0; i < __MAX_RINGS; i++ {
		last_ring_time = time.Now()
		m.prstatus(RING)
		conn.Write([]byte("Ringing...\n\r"))
		if !m.onhook { // computer has issued 'ATA' 
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
		for m.onhook  && d < 2000 {
			if _, err := conn.Write(zero); err != nil {
				goto no_answer
			}
			time.Sleep(__DELAY_MS * time.Millisecond)
			d += __DELAY_MS
			if !m.onhook { // computer has issued 'ATA' 
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
		if m.readReg(REG_AUTO_ANSWER) > 0 {
			if m.incReg(REG_RING_COUNT) >=
				m.readReg(REG_AUTO_ANSWER) {
				m.answer()
			}
		}
		
		// Silence for 4s
		d = 0
		for m.onhook && d < 4000 {
			// Test for closed connection
			if _, err := conn.Write(zero); err != nil {
				goto no_answer
			}
			
			time.Sleep(__DELAY_MS * time.Millisecond)
			d += __DELAY_MS
			if !m.onhook { // computer has issued 'ATA' 
				goto answered
			}
		}
	}
	
no_answer:
	// At this point we've not answered and have timed out, or the
	// caller hung up before we answered.
	if m.onhook {	
		conn.Close()
	}
	m.lowerRI()
	return false
	
answered:
	// if we're here, the computer answered.
	m.writeReg(REG_RING_COUNT, 0)
	m.lowerRI()
	return true
}

// "Busy" signal.
func checkBusy(m *Modem, conn io.ReadWriteCloser) bool {
	if !m.onhook || getRinging() {	
		conn.Write([]byte("BUSY\n\r"))
		return true
	}
	return false
}

func getRinging() bool {
	ringlock.RLock()
	defer ringlock.RUnlock()
	fmt.Printf("Ringing: %t\n", ringing)
	return ringing
}	

func setRinging(b bool) {
	ringlock.Lock()
	defer ringlock.Unlock()
	ringing = b
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
				m.writeReg(REG_RING_COUNT, 0) 
			}
		}
	}()

	setRinging(false)
	for {
		fmt.Println("Waiting for a conneciton from channel")
		conn = <- connection

		setRinging(true)
		if m.answerIncomming(conn) {
			// if we're here, the computer answered.
			m.conn = conn
			m.conn.Write([]byte("Answered\n\r"))
			go m.handleConnection()
		}
		setRinging(false)
	}
}

