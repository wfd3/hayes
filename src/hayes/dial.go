package hayes

import (
	"fmt"
	"strings"
	"net"
)

func parseDial(cmd string) (string, int, error) {
	var s string
	var c int

	// TODO: Add hostname:port based dialing
	// TODO: Add phone number -> hostname:port mapping
	
	c = 1			// Skip the 'D'

	if cmd[c] == 'T'  {	// Only support tone (& host) dialing
		e := strings.LastIndexAny(cmd, "0123456789,;@!")
		if e == -1 {
			return "", 0, fmt.Errorf("Bad phone number: %s", cmd)
		}
		s = fmt.Sprintf("DT%s", cmd[2:e+1])
		return s, len(s), nil
	}

	if cmd[c] == 'H' { 	// Host dialing
		s = fmt.Sprintf("DH%s", cmd[c+1:])
		return s, len(s), nil
	}

	if cmd[c] == 'L' {	// dial last number
		s = fmt.Sprintf("DL")
		return s, len(s), nil
	}
	
	return "", 0, fmt.Errorf("Bad/unsupported dial command: %s", cmd)
}


// ATD...
// TODO: See http://www.messagestick.net/modem/Hayes_Ch1-1.html on
// ATD... result codes
func (m *Modem) dial(to string) (int) {
	var err error

	debugf("dial(): %s\n", to)

	if to[1] == 'L' {
		return m.dial(m.lastdialed)
	}

	// Now we know the dial command isn't Dial Last (ATDL), save
	// this number as last dialed
	m.lastdialed = to

	if to[1] == 'H' {
		to = to[2:]
		debugf("Dialing: %s\n", to)
		m.conn, err = net.DialTimeout("tcp", to, __CONNECT_TIMEOUT)
		if err != nil {
			fmt.Printf("# connect error: %s\n", err)
			return BUSY
		}
		m.mode = DATAMODE // TODO: ';' to stay in command mode
		m.offHook()
		return CONNECT_38400
	}

	if to[2] == 'T' {	// TODO: add phone number->host mapping
		return ERROR
	}

	return ERROR
}
