package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"unicode"
)

func supportedProtocol(proto string) bool {
	switch strings.ToUpper(proto) {
	case "TELNET", "SSH":
		return true
	default:
		return false
	}
}

// Using the phonebook mapping, fake out dialing a standard phone number
// (ATDT5551212)
func dialNumber(phone string) (connection, error) {

	host, protocol, username, password, err := phonebook.Lookup(phone)
	if err != nil {
		return nil, err
	}

	logger.Printf("Dialing address book entry: %+v", host)

	if !supportedProtocol(protocol) {
		return nil, fmt.Errorf("Unsupported protocol '%s'", protocol)
	}

	switch strings.ToUpper(protocol) {
	case "SSH":
		return dialSSH(host, logger, username, password)
	case "TELNET":
		return dialTelnet(host, logger)
	}
	return nil, fmt.Errorf("Unknown protocol")
}

func dialStoredNumber(idxstr string) (connection, error) {

	index, err := strconv.Atoi(idxstr)
	if err != nil {
		logger.Print(err)
		return nil, err
	}

	phone, err := phonebook.LookupStoredNumber(index)
	if err != nil {
		logger.Print("Error: ", err)
		return nil, ERROR // We want ATDS to return ERROR.
	}
	logger.Print("-- phone number ", phone)
	return dialNumber(phone)
}

// Returns host|username|password
func splitATDE(cmd string) (string, string, string, error) {
	s := strings.Split(cmd, "|")
	if len(s) != 3 {
		return "", "", "", fmt.Errorf("Malformated ATDE command")
	}
	return s[0], s[1], s[2], nil
}

// ATD command (ATD, ATDT, ATDP, ATDL and the extensions ATDH (host) and ATDE (SSH)
// See http://www.messagestick.net/modem/Hayes_Ch1-1.html on ATD... result codes
func dial(to string) error {
	var conn connection
	var err error
	var clean_to string

	goOffHook()

	cmd := to[1]
	if cmd == 'L' {
		return dial(m.lastDialed)
	}

	// Now we know the dial command isn't Dial Last (ATDL), save
	// this number as last dialed
	m.lastDialed = to

	// Strip out dial modifiers we don't need.
	r := strings.NewReplacer(
		",", "",
		"@", "",
		"W", "",
		" ", "",
		"!", "",
		";", "")

	// Is this ATD<number>?  If so, dial it
	if unicode.IsDigit(rune(cmd)) {
		clean_to = r.Replace(to[1:])
		conn, err = dialNumber(clean_to)
	} else { // ATD<modifier>

		clean_to = r.Replace(to[2:])

		switch cmd {
		case 'H': // Hostname (ATDH hostname)
			logger.Print("Opening telnet connection to: ", clean_to)
			conn, err = dialTelnet(clean_to, logger)
		case 'E': // Encrypted host (ATDE hostname)
			logger.Print("Opening SSH connection to: ", clean_to)
			host, user, pw, e := splitATDE(clean_to)
			if e != nil {
				logger.Print(e)
				conn = nil
				err = e
			} else {
				conn, err = dialSSH(host, logger, user, pw)
			}
		case 'T', 'P': // Fake number from address book (ATDT 5551212)
			logger.Print("Dialing fake number: ", clean_to)
			conn, err = dialNumber(clean_to)
		case 'S': // Stored number (ATDS3)
			conn, err = dialStoredNumber(clean_to[1:])
		default:
			logger.Printf("Dial mode '%c' not supported\n", cmd)
			goOnHook()
			err = fmt.Errorf("Dial mode '%c' not supported", cmd)
		}
	}

	// if we're connected, setup the connected state in the modem,
	// otherwise return a BUSY or NO_ANSWER result code.
	if err != nil {
		goOnHook()
		if err == ERROR {
			return ERROR
		}
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return NO_ANSWER
		}
		return BUSY
	}

	// By default, conn.Mode() will return DATAMODE here.
	// Override and stay in command mode if ; present in the
	// original command string
	err = CONNECT
	m.connectSpeed = 38400  // We only go fast...
	if strings.Contains(to, ";") {
		conn.SetMode(COMMANDMODE)
		err = OK
	}

	// Remote answered, hand off conneciton to handleModem()
	callChannel <- conn
	return err
}

func parseDial(cmd string) (string, int, error) {
	var s string
	var c int

	if len(cmd) <= 1 {
		return "", 0, fmt.Errorf("Bad/unsupported dial command: %s", cmd)
	}

	c = 1 // Skip the 'D'

	// Parse 'ATD555555'
	if unicode.IsDigit(rune(cmd[c])) {
		e := strings.LastIndexAny(cmd, "0123456789,;@!")
		if e == -1 {
			return "", 0, fmt.Errorf("Bad phone number: %s", cmd)
		}
		s = fmt.Sprintf("D%s", cmd[1:e+1])
		return s, len(s), nil
	}

	switch cmd[c] {
	case 'T', 't', 'P', 'p': // Number dialing
		e := strings.LastIndexAny(cmd, "0123456789,;@!")
		if e == -1 {
			return "", 0, fmt.Errorf("Bad phone number: %s", cmd)
		}
		s = fmt.Sprintf("DT%s", cmd[2:e+1])
		return s, len(s), nil
	case 'H', 'h':
		s = fmt.Sprintf("DH%s", cmd[c+1:])
		return s, len(s), nil
	case 'E', 'e': // Host Dialing
		s = fmt.Sprintf("DE%s", cmd[c+1:])
		return s, len(s), nil
	case 'L', 'l': // Dial last number
		s = fmt.Sprintf("DL")
		return s, len(s), nil
	case 'S', 's': // Dial stored number
		s = fmt.Sprintf("DS%s", cmd[c+1:])
		return s, len(s), nil
	}

	return "", 0, fmt.Errorf("Bad/unsupported dial command: %s", cmd)
}
