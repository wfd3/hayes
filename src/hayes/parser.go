package main

import (
	"fmt"
	"strings"
	"time"
)

// Helper function to parse non-complex AT commands (everthing except ATS.., ATD...)
func parse(cmd string, opts string) (string, int, error) {

	cmd = strings.ToUpper(cmd)
	if len(cmd) == 1 {
		return cmd + "0", 1, nil
	}

	if strings.ContainsAny(cmd[1:2], opts) {
		return cmd[:2],  2, nil
	}

	return "", 0, fmt.Errorf("Bad command: %s", cmd)
}

func parseAmpersand(cmdstr string) (string, int, error) {
	var opts string

	switch strings.ToUpper(cmdstr[:2]) {
	case "&F":
		opts = "0"
	case "&V":
		opts = "0"
	case "&C":
		opts = "01"
	case "&Z":
		var idx int
		var str string
		var err error
		
		switch cmdstr[1] {
		case 'Z': _, err = fmt.Sscanf(cmdstr, "&Z%d=%s", &idx, &str)
		case 'z': _, err = fmt.Sscanf(cmdstr, "&z%d=%s", &idx, &str)
		default: err = fmt.Errorf("Badly formated &Z command: ", cmdstr)
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


	
	if len(cmdstring) < 2  {
		logger.Print("Cmd too short: ", cmdstring)
		return nil, ERROR
	}

	if strings.ToUpper(cmdstring[:2]) != "AT" {
		logger.Print("Malformed command: ", cmdstring)
		return nil, ERROR
	}

	logger.Printf("command: %s", cmdstring)

	cmd = cmdstring[2:] 		// Skip the 'AT'
	c = 0

	commands = nil
	status = OK
	for  c < len(cmd) && status == OK {
		switch (cmd[c]) {
		case 'D', 'd':
			s, i, err = parseDial(cmd[c:])
		case 'S', 's':
			s, i, err = parseRegisters(cmd[c:])
		case '*': 	// Custom debug registers
			s, i, err = parseDebug(cmd[c:])
		case '&':
			s, i, err = parseAmpersand(cmd)
		case 'A', 'a':
			opts = "0"
			s, i, err = parse(cmd[c:], opts)
		case 'E', 'e', 'H', 'h', 'Q', 'q', 'V', 'v', 'Z', 'z':
			opts = "01"
			s, i, err = parse(cmd[c:], opts)
		case 'L', 'l':
			opts = "0123"
			s, i, err = parse(cmd[c:], opts)
		case 'M', 'm', 'W', 'w':
			opts = "012"
			s, i, err = parse(cmd[c:], opts)
		case 'O', 'o':
			opts = "O"
			s, i, err = parse(cmd[c:], opts)
		case 'X', 'x':
			opts = "01234567"
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
		return nil
	}

	err = processCommands(commands)
	time.Sleep(500 * time.Millisecond) // Simulate command delay

	if err == OK || err == CONNECT {
		logger.Printf("Saving command string '%s'", cmdstring)
		m.lastCmd = cmdstring
	}
	return err
}
