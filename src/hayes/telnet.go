package hayes

import (
	"net"
	"fmt"
	"log"
)

// Telnet negoitation commands
const (
	IAC      byte = 0377
	DO       byte = 0375
	WILL     byte = 0373
	WONT     byte = 0374
	ECHO     byte = 0001
	LINEMODE byte = 0042
)

// Implements connection for in- and out-bound telnet
type telnetReadWriteCloser struct {
	direction int
	mode int
	c net.Conn
	sent uint64
	recv uint64
}

func (m *telnetReadWriteCloser) Read(p []byte) (int, error) {
	i, err := m.c.Read(p)

	// Tell the telnet server we won't comply.
	if  p[0] == IAC {
		cmd := make([]byte, 2)
		if _, err := m.Read(cmd); err != nil {
			return 0, err
		}
		m.Write([]byte{IAC, WONT, cmd[1]})
		i, err = m.Read(p)
	}
	m.recv += uint64(i)
	return i, err
}
func (m *telnetReadWriteCloser) Write(p []byte) (int, error) {
	i, err := m.c.Write(p)
	m.sent += uint64(i)
	return i, err
}
func (m *telnetReadWriteCloser) Close() error {
	err := m.c.Close()
	return err
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

	port := ":" + fmt.Sprintf("%d", *_flags_telnetPort)
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
		conn.Write([]byte{IAC, DO, LINEMODE, IAC, WILL, ECHO})
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

	return &telnetReadWriteCloser{OUTBOUND, DATAMODE, conn, 0, 0}, nil
}

