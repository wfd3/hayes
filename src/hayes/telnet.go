package main

import (
	"fmt"
	"log"
	"net"
)

// Telnet negoitation 
const (
	// COMMANDS
	IAC      byte = 255
	DONT     byte = 254
	DO       byte = 253
	WONT     byte = 252
	WILL     byte = 251
	SB       byte = 250
	GA       byte = 249
	EL       byte = 248
	EC       byte = 247
	AYT      byte = 246
	AO       byte = 245
	IP       byte = 244
	BRK      byte = 243
	DM       byte = 242
	NOP      byte = 241
	SE       byte = 240

	// OPTIONS
	ECHO     byte = 1
	SGA      byte = 3
	STATUS   byte = 5
	TIMINGMK byte = 6
	TERM     byte = 24
	WINSIZE  byte = 31
	TERMSPD  byte = 32
	REMFLOW  byte = 33
	LINEMODE byte = 34
	ENVVAR   byte = 36
)
var decodeMap map[byte]string = map[byte]string {
	IAC: "IAC ",
	DONT: "DONT ",
	DO: "DO ",
	WONT: "WONT ",
	WILL: "WILL ",
	SB: "SB ",
	GA: "GA ",
	EL: "EL ",
	EC: "EC ",
	AYT: "AYT ",
	AO: "AO ",
	IP: "IP ",
	BRK: "BRK ",
	DM: "DM ",
	NOP: "NOP ",
	SE: "SE ",
	ECHO: "ECHO ",
	SGA: "SGA ",
	STATUS: "STATUS ",
	TIMINGMK: "TIMINGMK ",
	TERM: "TERM ",
	WINSIZE: "WINSIZE ",
	TERMSPD: "TERMSPD ",
	REMFLOW: "REMFLOW ",
	LINEMODE: "LINEMODE ",
	ENVVAR: "ENVVAR ",
}

// Implements connection for in- and out-bound telnet
type telnetReadWriteCloser struct {
	direction int
	mode int
	c net.Conn
	sent uint64
	recv uint64
}


func decode(b byte) string {
	s, ok := decodeMap[b]
	if !ok {
		return fmt.Sprintf("%d", b)
	}
	return s
}

func (m *telnetReadWriteCloser) command(p []byte) (i int, err error) {
	if p[0] != IAC {
		return 0, nil
	}

	s := ""
	done := false

	for !done {
		s += decode(p[0])

		switch p[0] {
		case IAC:
			m.c.Read(p)
			if p[0] == IAC { // Two in a row, it's just ASCII 255
				done = true
				break
			}
			
		case NOP, DM, BRK, IP, AO, AYT, EC, EL, GA, SE:
			logger.Printf("Ignoring: %s", s)			

		case SB:
			// Comsume options until we read a final SE
			for p[0] != SE {
				i, err = m.c.Read(p)
				s += decode(p[0])
			}
			logger.Printf("Ignoring: %s", s)

		case WILL:
			m.c.Read(p)
			s += decode(p[0])
			logger.Printf("Reading: %s", s)
			m.c.Write([]byte{IAC, DONT, p[0]})
			logger.Printf("Sending: IAC DONT %s", decode(p[0]))

		case DO:
			m.c.Read(p)
			s += decode(p[0])
			logger.Printf("Reading: %s", s)
			m.c.Write([]byte{IAC, WONT, p[0]})
			logger.Printf("Sending: IAC WONT %s", decode(p[0]))
			
		case DONT:
			m.c.Read(p)
			s += decode(p[0])
			logger.Printf("Reading: %s", s)
			m.c.Write([]byte{IAC, WONT, p[0]})			
			logger.Printf("Sending: IAC WONT %s", decode(p[0]))			
		case WONT:
			m.c.Read(p)
			s += decode(p[0])
			logger.Printf("Reading: %s", s)
			m.c.Write([]byte{IAC, DONT, p[0]})
			logger.Printf("Sending: IAC DONT %s", decode(p[0]))
			
		default:
			done = true
		}
	}

	return i, err
}

func (m *telnetReadWriteCloser) Read(p []byte) (int, error) {
	i, err := m.c.Read(p)

	// If it's a telnet command, process it
	if p[0] == IAC {
		i, err = m.command(p) 
	}
	m.recv += uint64(i)
	return i, err
}
func (m *telnetReadWriteCloser) Write(p []byte) (int, error) {
	i, err := m.c.Write(p)
	if err != nil {
		logger.Print(err)
	}
	m.sent += uint64(i)
	return i, err
}
func (m *telnetReadWriteCloser) Close() error {
	logger.Printf("Closing telnet connection to %s", m.RemoteAddr())
	return m.c.Close()
}
func (m *telnetReadWriteCloser) Mode() int {
	return m.mode
}
func (m *telnetReadWriteCloser) Direction() int {
	return m.direction
}
func (m *telnetReadWriteCloser) RemoteAddr() net.Addr {
	return m.c.RemoteAddr()
}
func (m *telnetReadWriteCloser) SetMode(mode int) {
	if mode != DATAMODE || mode != COMMANDMODE {
		panic("Bad mode")
	}
	m.mode = mode
}
func (m *telnetReadWriteCloser) Stats() (uint64, uint64) {
	return m.sent, m.recv
}

func acceptTelnet(channel chan connection, busy busyFunc, log *log.Logger,
	ok chan error) {

	port := fmt.Sprintf(":%d", *_flags_telnetPort)
	l, err := net.Listen("tcp", port)
	if err != nil {
		log.Print("Fatal Error: ", err)
		ok <- err
		return
	}
	defer l.Close()
	log.Printf("Listening: telnet tcp/%s", port)
	ok <- nil
	
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Print("l.Accept(): %s\n", err)
			continue
		}

		if busy() {
			conn.Write([]byte("Busy...\n\r"))
			conn.Close()
			continue
		}
		
		// This is a telnet session, negotiate char-at-a-time
		// and turn off local echo
		conn.Write([]byte{IAC, DO, LINEMODE}) // You go into linemode
		conn.Write([]byte{IAC, DONT, ECHO}) // You don't echo locally
		conn.Write([]byte{IAC, WILL, ECHO}) // I'll echo to you

		channel <- &telnetReadWriteCloser{INBOUND, DATAMODE, conn, 0, 0}
	}
}

func dialTelnet(remote string, log *log.Logger) (connection, error) {

	if _, _, err := net.SplitHostPort(remote); err != nil {
		remote += ":23"
	}
	conn, err := net.DialTimeout("tcp", remote, __CONNECT_TIMEOUT)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			log.Print("net.DialTimeout: Timed out")
		}
		return nil, err
	}

	log.Printf("Connected to %s", conn.RemoteAddr())
	return &telnetReadWriteCloser{OUTBOUND, DATAMODE, conn, 0, 0}, nil
}

