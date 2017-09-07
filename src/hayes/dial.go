package hayes

import (
	"fmt"
	"strings"
	"strconv"
	"net"
	"io"
	"time"
	"golang.org/x/crypto/ssh"
)

const __CONNECT_TIMEOUT = __MAX_RINGS * 6 * time.Second

// Implements io.ReadWriteCloser, used to convert SSH ssh.Session into
// io.ReadWriteCloser.
type myReadWriteCloser struct {
	in io.Reader
	out io.WriteCloser
	client *ssh.Client
	session *ssh.Session
}
func (m myReadWriteCloser) Read(p []byte) (int, error) {
	return m.in.Read(p)
}
func (m myReadWriteCloser) Write(p []byte) (int, error) {
	return m.out.Write(p)
}
func (m myReadWriteCloser) Close() error {
	// Remember, in is an io.Reader so it doesn't Close()
	err := m.out.Close()
	m.session.Close()
	m.client.Close()
	return err
}

// TODO: user:password entry in dial string?
func (m *Modem) dialSSH(remote string, username string, pw string) (myReadWriteCloser, error) {

	m.log.Printf("Connecting to %s", remote)

	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(pw),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Danger?
		Timeout: time.Duration(__CONNECT_TIMEOUT),
	}

	client, err := ssh.Dial("tcp", remote, config)
	if err != nil {
		m.log.Print("Fatal Error: ssh.Dial(): ", err)
		if err, ok := err.(net.Error); ok && err.Timeout() {
			m.log.Print("ssh.Dial: Timed out")
		}

		return myReadWriteCloser{}, fmt.Errorf("ssh.Dial() failed: ", err)
	}

	// Create a session
	session, err := client.NewSession()
	if err != nil {
    		m.log.Print("unable to create session: ", err)
		return myReadWriteCloser{},
		fmt.Errorf("unable to create session: ", err)
	}

	// Set up terminal modes
	modes := ssh.TerminalModes{
    		ssh.ECHO:          0,     // disable echoing
    		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
    		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}	
	// Request pseudo terminal
	if err := session.RequestPty("xterm", 40, 80, modes); err != nil {
    		m.log.Print("request for pseudo terminal failed: ", err)
		return myReadWriteCloser{},
		fmt.Errorf("request for pty failed: ", err)
	}

	// Start remote shell
	send, err :=  session.StdinPipe()
	if err != nil {
		m.log.Print("StdinPipe(): ", err)
		return myReadWriteCloser{}, fmt.Errorf("session.StdinPipe(): ", err)
	}
	recv, err := session.StdoutPipe()
	if err != nil {
		m.log.Print("StdoutPipe(): ", err)
		return myReadWriteCloser{}, fmt.Errorf("session.StdinOut(): ", err)
	}

        session.Shell()

	m.log.Printf("Connected to remote host '%s', SSH Server version %s",
		client.Conn.RemoteAddr(), client.Conn.ServerVersion())
	
	return myReadWriteCloser{recv, send, client, session}, nil
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

// Using the addressbook mapping, fake out dialing a standard phone number
// (ATDT5551212)
func (m *Modem) dialNumber(remote string) (io.ReadWriteCloser, error) {
	n := sanitizeNumber(remote)
	host := m.addressbook[n]
	if host == nil {
		return nil, fmt.Errorf("number not in address book")
	}
	switch strings.ToUpper(host.protocol) {
	case "SSH":
		return m.dialSSH(host.host, *_flags_user, *_flags_pw)
	case "TELNET":
		return m.dialTelnet(host.host)
	}

	m.log.Printf("Protocol '%s' not supported", host.protocol)
	return nil, fmt.Errorf("Unknown protocol")
}

// ATD...
// See http://www.messagestick.net/modem/Hayes_Ch1-1.html on ATD... result codes
func (m *Modem) dial(to string) error {
	var conn io.ReadWriteCloser
	var err error

	m.offHook()

	cmd := to[1]
	if cmd == 'L' {
		return m.dial(m.lastdialed)
	}

	// Now we know the dial command isn't Dial Last (ATDL), save
	// this number as last dialed
	m.lastdialed = to

	// Strip out dial modifiers we don't need.
	r := strings.NewReplacer(
		",", "",
		"@", "",
		"W", "",
		" ", "",
		"!", "",
		";", "")
	
	clean_to := r.Replace(to[2:])

	switch cmd {
	case 'H':		// Hostname (ATDH hostname)
		m.log.Print("Opening telnet connection to: ", clean_to)
		conn, err = m.dialTelnet(clean_to)
	case 'E':		// Encrypted host (ATDE hostname)
		m.log.Print("Opening SSH connection to: ", clean_to)
		conn, err = m.dialSSH(clean_to, *_flags_user, *_flags_pw)
	case 'T', 'P':		// Fake number from address book (ATDT 5551212)
		m.log.Print("Dialing fake number: ", clean_to)
		conn, err = m.dialNumber(clean_to)
	case 'S':		// Stored number (ATDS3)
		m.log.Print("Dialing stored number: ", clean_to)
		index, err := strconv.Atoi(clean_to[1:])
		if err != nil {
			return ERROR
		}
		phone := m.storedNumber(index)
		if phone == "" {
			m.log.Print("Stored number not found")
			return ERROR
		}
		conn, err = m.dialNumber(phone)
	default:
		fmt.Println(clean_to)
		m.log.Printf("Dial mode '%c' not supported\n", cmd)
		return ERROR
	}

	// if we're connected, setup the connected state in the modem, otherwise
	// return a BUSY result code.
	// TODO: Can we tell the difference between BUSY and NO_ANSWER?
	if err != nil {
		m.onHook()
		return BUSY
	}

	// Remote answered, set connection speed and signal CD.
	m.conn = conn
	m.connect_speed = 38400
	m.raiseCD()

	// Stay in command mode if ; present in the original command string
	if strings.Contains(to, ";") {
		return OK
	}
	m.mode = DATAMODE
	return CONNECT
}

func parseDial(cmd string) (string, int, error) {
	var s string
	var c int
	
	c = 1			// Skip the 'D'
	switch cmd[c] {
	case 'T', 'P':		// Number dialing
		e := strings.LastIndexAny(cmd, "0123456789,;@!")
		if e == -1 {
			return "", 0, fmt.Errorf("Bad phone number: %s", cmd)
		}
		s = fmt.Sprintf("DT%s", cmd[2:e+1])
		return s, len(s), nil
	case 'H', 'E':		// Host Dialing
		s = fmt.Sprintf("D%c%s", cmd[c], cmd[c+1:])
		return s, len(s), nil
	case 'L':		// Dial last number
		s = fmt.Sprintf("DL")
		return s, len(s), nil
	case 'S': 		// Dial stored number
		s = fmt.Sprintf("DS%s", cmd[c+1:])
		return s, len(s), nil
	}

	return "", 0, fmt.Errorf("Bad/unsupported dial command: %s", cmd)
}
