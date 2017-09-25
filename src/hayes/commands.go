package hayes

import (
	"time"
	"strings"
	"fmt"
)

// ATA
func (m *Modem) answer() error {
	if m.offHook() {
		m.log.Print("Can't answer, line off hook already")
		return ERROR
	}
	
	m.goOffHook()

	// Simulate Carrier Detect delay
	delay := time.Duration(m.registers.Read(REG_CARRIER_DETECT_RESPONSE_TIME))
	delay = delay * 100 * time.Millisecond
	time.Sleep(delay)
	m.dcd = true
	m.mode = DATAMODE
	m.connect_speed = 38400	// We only go fast...
	return CONNECT
}

// ATZ
// Setup/reset modem.  Leaves RTS & CTS down.
func (m *Modem) reset() error {
	var err error = OK

	m.log.Print("Resetting modem")

	m.goOnHook()
	m.setLineBusy(false)
	m.lowerDSR()
	m.lowerCTS()
	m.lowerRI()
	m.stopTimer()

	m.echoInCmdMode = true  // Echo local keypresses
	m.quiet = false		// Modem offers return status
	m.verbose = true	// Text return codes
	m.volume = 1		// moderate volume
	m.speakermode = 1	// on until other modem heard
	m.lastcmd = ""
	m.lastdialed = ""
	m.connect_speed = 0
	m.connectMsgSpeed = true
	m.busyDetect = true
	m.extendedResultCodes = true
	m.dcdControl = false
	m.resetRegs()
	m.resetTimer()
	m.phonebook = NewPhonebook(*_flags_phoneBook, m.log)
	err = m.phonebook.Load()
	if err != nil {
		m.log.Print(err)
	}

	return err
}

// AT&V
func (m *Modem) amperV() error {
	b := func(p bool) (string) {
		if p {
			return"1 "
		} 
		return "0 "
	};
	i := func(p int) (string) {
		return fmt.Sprintf("%d ", p)
	};
	x := func(r, b bool) (string) {
		if (r == false && b == false) {
			return "0 "
		}
		if (r == true && b == false) {
			return "1 "
		}
		if (r == true && b == true) {
			return "7 "
		}
		return "0 "
	};

	var s string
	s += "E" + b(m.echoInCmdMode)
	s += "F1 "		// For Hayes 1200 compatability 
	s += "L" + i(m.volume)
	s += "M" + i(m.speakermode)
	s += "Q" + b(m.quiet)
	s += "V" + b(m.verbose)
	s += "W" + b(m.connectMsgSpeed)
	s += "X" + x(m.extendedResultCodes, m.busyDetect)
	s += "&C" + b(m.dcdControl)
	s += "\n"
	s += m.registers.String()
	m.serial.Println(s)
	return OK
}

// AT&...
// Only support &V and &C for now
func (m *Modem) ampersand(cmd string) error {

	m.log.Print(cmd)
	if cmd[:2] == "&Z" {
		var s string
		var i int
		_ , err := fmt.Sscanf(cmd, "&Z%d=%s", &i, &s)
		if err != nil {
			m.log.Print(err)
			return err
		}
		if s[0] == 'D' || s[0] == 'd' { // Extension
			return m.phonebook.Delete(i)
		}
		return m.phonebook.Add(i, s)
	}

	switch cmd {
	case "&C0":
		m.dcdControl = false
		return OK
	case "&C1":
		m.dcdControl = true
		return OK
	case "&V0": return m.amperV()
	}
	return ERROR
}

// process each command
func (m *Modem) processCommands(commands []string) error {
	var status error
	var cmd string

	m.log.Printf("entering PC: %+v\n", commands)
	status = OK
	for _, cmd = range commands {
		m.log.Printf("Processing: %s", cmd)
		switch cmd[0] {
		case 'A':
			status = m.answer()
		case 'Z':
			status = m.reset()
			time.Sleep(250 * time.Millisecond)
			m.raiseDSR()
			m.raiseCTS()
		case 'E':
			if cmd[1] == '0' {
				m.echoInCmdMode = false
			} else {
				m.echoInCmdMode = true
			}
		case 'F':	// Online Echo mode, F1 assumed for backwards
			        // compatability after Hayes 1200
			status = OK 
		case 'H':
			if cmd[1] == '0' { 
				status = m.goOnHook()
			} else if cmd[1] == '1' {
				status = m.goOffHook()
			} else {
				status = ERROR
			}
		case 'Q':
			if cmd[1] == '0' {
				m.quiet = true
			} else {
				m.quiet = false
			}
		case 'V':
			if cmd[1] == '0' {
				m.verbose = true
			} else {
				m.verbose = false
			}
		case 'L':
			switch cmd[1] {
			case '0': m.volume = 0
			case '1': m.volume = 1
			case '2': m.volume = 2
			case '3': m.volume = 3
			}
		case 'M':
			switch cmd[1] {
			case '0': m.speakermode = 0
			case '1': m.speakermode = 1
			case '2': m.speakermode = 2
			}
		case 'O':
			m.mode = DATAMODE
			status = OK
		case 'W':
			switch cmd[1] {
			case '0': m.connectMsgSpeed = false
			case '1', '2': m.connectMsgSpeed = true
			}
		case 'X':	// Change result codes displayed
			switch cmd[1] {
			case '0':
				m.extendedResultCodes = false
				m.busyDetect = false
			case '1', '2':
				m.extendedResultCodes = true
				m.busyDetect = false
			case '3', '4', '5', '6', '7':
				m.extendedResultCodes = true
				m.busyDetect = true
			}
		case 'D':
			status = m.dial(cmd)
		case 'S':
			status = m.registerCmd(cmd)
		case '&':
			status = m.ampersand(cmd)
		case '*':
			status = m.debug(cmd)
		default:
			status = ERROR
		}
		if status != OK {
			break
		}
	}
	return status
}

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

func (m *Modem) parseAmpersand(cmdstr string) (string, int, error) {
	var opts string

	switch strings.ToUpper(cmdstr[:2]) {
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
			m.log.Print("ERROR: ", err)
			return "", 0, err
		}
		s := fmt.Sprintf("&Z%d=%s", idx, str)
		return s, len(s), nil

	default:
		m.log.Printf("Unknown &cmd: %s", cmdstr)
		return "", 0, ERROR
	}
	
	s, i, err := parse(cmdstr[1:], opts)
	s = "&" + s
	i++
	return s, i, err
}

// +++ 
func (m *Modem) command(cmdstring string) {
	var commands []string
	var s, opts string
	var i int
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


	if strings.ToUpper(cmdstring) == "AT" {
		m.prstatus(OK)
		return
	}
	
	if len(cmdstring) < 2  {
		m.log.Print("Cmd too short: ", cmdstring)
		m.prstatus(ERROR)
		return
	}

	if strings.ToUpper(cmdstring[:2]) != "AT" {
		m.log.Print("Malformed command: ", cmdstring)
		m.prstatus(ERROR)
		return
	}

	m.log.Print("command: ", cmdstring)

	cmd := cmdstring[2:] 		// Skip the 'AT'
	c := 0

	commands = nil
	status = OK
	savecmds := true
	for  c < len(cmd) && status == OK {
		switch (cmd[c]) {
		case 'D', 'd':
			s, i, err = parseDial(cmd[c:])
			if err != nil {
				m.prstatus(err)
				return
			}
			commands = append(commands, s)
			c += i
			continue
		case 'S', 's':
			s, i, err = parseRegisters(cmd[c:])
			if err != nil {
				m.prstatus(err)
				return
			}
			commands = append(commands, s)
			c += i
			continue
		case '*': 	// Custom debug registers
			s, i, err = parseDebug(cmd[c:])
			if err != nil {
				m.prstatus(err)
				return
			}
			commands = append(commands, s)
			c += i
			continue
		case '&':
			s, i, err = m.parseAmpersand(cmd)
			if err != nil {
				m.prstatus(err)
				return
			}
			commands = append(commands, s)
			c += i
			continue
		case 'A', 'a':
			opts = "0"
		case 'E', 'e', 'H', 'h', 'Q', 'q', 'V', 'v', 'Z', 'z':
			opts = "01"
		case 'L', 'l':
			opts = "0123"
		case 'M', 'm', 'W', 'w':
			opts = "012"
		case 'O', 'o':
			opts = "O"
		case 'X', 'x':
			opts = "01234567"
		default:
			m.log.Printf("Unknown command: %s", cmd)
			m.prstatus(ERROR)
			return
		}
		s, i, err = parse(cmd[c:], opts)
		if err != nil {
			m.prstatus(ERROR)
			return
		}
		commands = append(commands, s)
		c += i
	}

	m.log.Print("Command array: %+v", commands)
	status = m.processCommands(commands)
	m.prstatus(status)

	if savecmds && status == OK {
		m.lastcmd = cmdstring
	}
}
