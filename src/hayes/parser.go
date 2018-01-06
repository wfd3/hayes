package main

import (
	"fmt"
	"strings"
)

// Helper function to parse non-complex AT commands (everthing except ATS.., ATD...)
func parse(cmd string, opts string) (string, int, error) {

	cmd = strings.ToUpper(cmd)
	if len(cmd) == 1 {
		return cmd + "0", 1, nil
	}

	if strings.ContainsAny(cmd[1:2], opts) {
		return cmd[:2], 2, nil
	}

	logger.Printf("Bad command: %s", cmd)
	return "", 0, fmt.Errorf("Bad command: %s", cmd)
}

// ATS...
// Given a string that looks like a "S" command, parse & normalize it
func parseRegisters(cmd string) (string, int, error) {
	var s string
	var err error
	var reg, val int

	// NOTE: The order of these stanzas is critical.

	if len(cmd) < 2 {
		return "", 0, fmt.Errorf("Bad command: %s", cmd)
	}

	c := strings.ToUpper(cmd)

	// S? - query selected register
	if c[:2] == "S?" {
		s = "S?"
		return s, 2, nil
	}

	// Sn=x - write x to n
	_, err = fmt.Sscanf(c, "S%d=%d", &reg, &val)
	if err == nil {
		s = fmt.Sprintf("S%d=%d", reg, val)
		return s, len(s), nil
	}

	// Sn? - query register n
	_, err = fmt.Sscanf(c, "S%d?", &reg)
	if err == nil {
		s = fmt.Sprintf("S%d?", reg)
		return s, len(s), nil
	}

	// Sn - slect register
	_, err = fmt.Sscanf(c, "S%d", &reg)
	if err == nil {
		s = fmt.Sprintf("S%d", reg)
		return s, len(s), nil
	}

	return "", 0, fmt.Errorf("Bad S command: %s", cmd)
}

// parse AT&...
func parseAmpersand(cmdstr string) (string, int, error) {
	var opts string

	c := strings.ToUpper(cmdstr[1:2])[0]

	// AT&A, AT&B, AT&D, AT&G, AT&J, AT&K, AT&L, AT&M, AT&O, AT&Q,
	// AT&R, AT&S, AT&T, AT&U, AT&X

	switch c {
	case 'F', 'V':
		opts = "0"
	case 'A', 'B', 'C', 'J', 'L', 'R', 'S', 'U', 'W', 'Y':
		opts = "01"
	case 'G', 'X', 'P':
		opts = "012"
	case 'D':
		opts = "0123"
	case 'O', 'K', 'M':
		opts = "01234"
	case 'Q':
		opts = "05689"
	case 'T':
		opts = "0123456789"
	case 'Z':
		var idx int
		var str string
		var err error

		switch cmdstr[1] { // username/passwd could be case-sensitive
		case 'Z':
			_, err = fmt.Sscanf(cmdstr, "&Z%d=%s", &idx, &str)
		case 'z':
			_, err = fmt.Sscanf(cmdstr, "&z%d=%s", &idx, &str)
		default:
			err = fmt.Errorf("Badly formated &Z command: %s", cmdstr)
		}

		if err != nil {
			logger.Print("ERROR: ", err)
			return "", 0, err
		}
		s := fmt.Sprintf("&Z%d=%s", idx, str)
		return s, len(s), nil
	default:
		logger.Printf("Unknown &cmd: %s", cmdstr)
		return "", 0, ERROR
	}

	s, i, err := parse(cmdstr[1:], opts)
	s = "&" + s
	i++
	return s, i, err
}

// +++
func parseCommand(cmdstring string) ([]string, error) {
	var commands []string
	var s, opts, cmd string
	var i, c int
	var status error
	var err error

	// Process here is to parse the entire command string into
	// discrete commands, then execute those discrete commands in
	// the order they were given to us.  This makes syntax
	// checking/failures happen before any commands are executed
	// which is, if I recall correctly, how this works in the real
	// hardware.  Note that the command codes ("DT", "X", etc.)
	// all must be upper case for the rest of the parsing system
	// to work, but the entire command string should be left as it
	// was handed to us.  This is so that we can embed passwords
	// in the extended dial command (ATDE, specifically).

	if len(cmdstring) < 2 {
		logger.Print("Cmd too short: ", cmdstring)
		return nil, ERROR
	}

	if strings.ToUpper(cmdstring[:2]) != "AT" {
		logger.Print("Malformed command: ", cmdstring)
		return nil, ERROR
	}

	logger.Printf("command: %s", cmdstring)

	cmd = cmdstring[2:] // Skip the 'AT'
	c = 0

	commands = nil
	status = OK
	f := strings.ToUpper(cmd)
	for c < len(cmd) && status == OK {
		switch f[c] {
		case 'P', 'T':
			s = f[c:1]
			i = 1
			err = nil
		case 'D':
			s, i, err = parseDial(cmd[c:])
		case 'S':
			s, i, err = parseRegisters(cmd[c:])
		case '*': // Custom debug registers
			s, i, err = parseDebug(cmd[c:])
		case '&':
			s, i, err = parseAmpersand(cmd)
		case 'A', '!':
			opts = "0"
			s, i, err = parse(cmd[c:], opts)
		case 'E', 'H', 'Q', 'V', 'Z':
			opts = "01"
			s, i, err = parse(cmd[c:], opts)
		case 'M', 'W':
			opts = "012"
			s, i, err = parse(cmd[c:], opts)
		case 'L':
			opts = "0123"
			s, i, err = parse(cmd[c:], opts)
		case 'O':
			opts = "O"
			s, i, err = parse(cmd[c:], opts)
		case 'X':
			opts = "01234567"
			s, i, err = parse(cmd[c:], opts)
		case 'I':
			opts = "012345"
			s, i, err = parse(cmd[c:], opts)

		// faked out commands
		case 'Y', 'C':
			opts = "01"
			s, i, err = parse(cmd[c:], opts)
		case 'N', 'B':
			opts = "012345"
			s, i, err = parse(cmd[c:], opts)

		default:
			logger.Printf("Unknown command: %s", cmd)
			return nil, ERROR
		}

		if err != nil {
			return nil, ERROR
		}
		commands = append(commands, s)
		c += i
	}

	logger.Printf("Command array: %+v", commands)

	return commands, nil
}

func runCommand(cmdstring string) error {
	var err error
	if strings.ToUpper(cmdstring) == "AT" {
		m.lastCmd = "AT"
		return OK
	}

	commands, err := parseCommand(cmdstring)
	if err != nil {
		return err
	}

	err = processCommands(commands)

	if err == OK || err == CONNECT {
		logger.Printf("Saving command string '%s'", cmdstring)
		m.lastCmd = cmdstring
	}
	return err
}
