package hayes

import (
	"net"
	"io"
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
	
func (m *Modem) acceptTelnet(channel chan io.ReadWriteCloser) {
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

		if checkBusy(m, conn) {
			conn.Close()
			continue
		}
		
		// This is a telnet session, negotiate char-at-a-time
		conn.Write([]byte{IAC, DO, LINEMODE, IAC, WILL, ECHO})
		channel <- conn
	}
}


func (m *Modem) dialTelnet(remote string) (io.ReadWriteCloser, error) {

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
	return conn, nil
}

