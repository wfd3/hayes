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

// Implements io.ReadWriteCloser
type myReadWriter struct {
	in io.Reader
	out io.WriteCloser
	client *ssh.Client
	session *ssh.Session
}

func (m myReadWriter) Read(p []byte) (int, error) {
	return m.in.Read(p)
}

func (m myReadWriter) Write(p []byte) (int, error) {
	return m.out.Write(p)
}

func (m myReadWriter) Close() error {
	// Remember, in is an io.Reader so it doesn't Close()
	err := m.out.Close()
	m.session.Close()
	m.client.Close()
	return err
}

func (m *Modem) dialSSH(remote string) int {
	config := &ssh.ClientConfig{
		User: *_flags_user,
		Auth: []ssh.AuthMethod{
			ssh.Password(*_flags_pw),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Danger?
	}

	client, err := ssh.Dial("tcp", remote, config)
	if err != nil {
		m.log.Fatal("Fatal Error: ssh.Dial(): ", err)
	}

	// Create a session
	session, err := client.NewSession()
	if err != nil {
    		m.log.Print("unable to create session: ", err)
		return ERROR
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
		return ERROR
	}

	// Start remote shell
	send, err :=  session.StdinPipe()
	if err != nil {
		m.log.Print("StdinPipe(): ", err)
		return ERROR
	}
	recv, err := session.StdoutPipe()
	if err != nil {
		m.log.Print("StdoutPipe(): ", err)
		return ERROR
	}

	m.conn = &myReadWriter{recv, send, client, session}
        session.Shell()
	
	return CONNECT
}

func (m *Modem) dialTelnet(remote string) int {

	conn, err := net.DialTimeout("tcp", remote, __CONNECT_TIMEOUT)
	if err != nil {
		return BUSY
	}
	m.conn = conn
	return CONNECT
}

func (m *Modem) dialNumber(remote string) int {
	n := sanitizeNumber(remote)
	host := m.addressbook[n]
	if host == nil {
		return ERROR
	}
	if strings.ToUpper(host.protocol) == "SSH" {
		return m.dialSSH(host.host)
	}
	if strings.ToUpper(host.protocol) == "TELNET" {
		return m.dialTelnet(host.host)
	}

	return ERROR
}

// ATD...
// See http://www.messagestick.net/modem/Hayes_Ch1-1.html on ATD... result codes
func (m *Modem) dial(to string) (int) {
	var ret int

	cmd := to[1]
	if cmd == 'L' {
		return m.dial(m.lastdialed)
	}

	// Now we know the dial command isn't Dial Last (ATDL), save
	// this number as last dialed
	m.lastdialed = to

	// Strip out dial modifiers we don't need.
	r := strings.NewReplacer(",", "","@", "", "W", "", " ", "", "!", "",
		";", "")
	clean_to := r.Replace(to[2:])

	switch cmd {
	case 'H': ret = m.dialTelnet(clean_to)
	case 'E': ret = m.dialSSH(clean_to)
	case 'T', 'P': ret = m.dialNumber(clean_to)
	case 'S':
		index, err := strconv.Atoi(clean_to[1:])
		if err != nil {
			return ERROR
		}
		phone := m.storedNumber(index)
		if phone == "" {
			return ERROR
		}
		ret = m.dialNumber(phone)
	default:
		fmt.Println(clean_to)
		m.log.Printf("Dial mode '%c' not supported\n", cmd)
		return ERROR
	}

	// if we're connected, setup the connected state in the modem, otherwise
	// return whatever error we have.
	if ret != CONNECT {
		return ret
	}

	// Remote answered, setup the modem
	m.raiseCD()
	m.offHook()
	m.connect_speed = 38400
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
