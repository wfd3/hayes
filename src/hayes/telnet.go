package hayes

import (
	"net"
)

// Telnet negoitation commands
const (
	IAC = 0377
	DO = 0375
	WILL = 0373
	WONT = 0374
	ECHO = 0001
	LINEMODE = 0042
)

// Implements connection for in- and out-bound telnet
type telnetReadWriteCloser struct {
	direction int
	mode int
	c net.Conn
}

func (m telnetReadWriteCloser) Read(p []byte) (int, error) {
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

	return i, err
}
func (m telnetReadWriteCloser) Write(p []byte) (int, error) {
	return m.c.Write(p)
}
func (m telnetReadWriteCloser) Close() error {
	err := m.c.Close()
	return err
}
func (m telnetReadWriteCloser) Mode() int {
	return m.mode
}
func (m telnetReadWriteCloser) Direction() int {
	return m.direction
}
func (m telnetReadWriteCloser) RemoteAddr() net.Addr {
	return m.c.RemoteAddr()
}
func (m telnetReadWriteCloser) SetMode(mode int) {
	if mode != DATAMODE || mode != COMMANDMODE {
		panic("Bad mode")
	}
	m.mode = mode
}

func (m *Modem) acceptTelnet(channel chan connection) {
	// TODO: Cmdline option for port
	l, err := net.Listen("tcp", ":20000")
	if err != nil {
		m.log.Fatal("Fatal Error: ", err)
	}
	defer l.Close()
	m.log.Print("Listening: tcp/2000")

	for {
		conn, err := l.Accept()
		if err != nil {
			m.log.Print("l.Accept(): %s\n", err)
			continue
		}

		if m.checkBusy() {
			conn.Write([]byte("Busy..."))
			conn.Close()
			continue
		}
		
		// This is a telnet session, negotiate char-at-a-time
		conn.Write([]byte{IAC, DO, LINEMODE, IAC, WILL, ECHO})
		channel <- telnetReadWriteCloser{INBOUND, DATAMODE, conn}
	}
}

func (m *Modem) dialTelnet(remote string) (connection, error) {

	if _, _, err := net.SplitHostPort(remote); err != nil {
		remote += ":23"
	}
	m.log.Print("Connecting to: ", remote)
	conn, err := net.DialTimeout("tcp", remote, __CONNECT_TIMEOUT)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			m.log.Print("net.DialTimeout: Timed out")
		}
		return nil, err
	}
	m.log.Printf("Connected to remote host '%s'", conn.RemoteAddr())
	return telnetReadWriteCloser{OUTBOUND, DATAMODE, conn}, nil
}

