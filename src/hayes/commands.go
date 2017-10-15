package main

import (
	"time"
	"fmt"
)

// ATA
func answer() error {
	if offHook() {
		logger.Print("Can't answer, line off hook already")
		return ERROR
	}
	
	goOffHook()

	// Simulate Carrier Detect delay
	delay := time.Duration(registers.Read(REG_CARRIER_DETECT_RESPONSE_TIME))
	delay = delay * 100 * time.Millisecond
	time.Sleep(delay)
	m.dcd = true
	m.mode = DATAMODE
	m.connectSpeed = 38400	// We only go fast...
	return CONNECT
}

// ATZ
// Setup/reset modem.  Leaves RTS & CTS down.
func reset() error {
	var err error = OK

	logger.Print("Resetting modem")

	// Reset state
	goOnHook()
	setLineBusy(false)
	lowerDSR()
	lowerCTS()
	lowerRI()
	stopTimer()
	m.dcd = false
	m.lastCmd = ""
	m.lastDialed = ""
	m.connectSpeed = 0

	// Reset Config
	m.echoInCmdMode = true  // Echo local keypresses
	m.quiet = false		// Modem offers return status
	m.verbose = true	// Text return codes
	m.speakerVolume = 1	// moderate volume
	m.speakerMode = 1	// on until other modem heard
	m.busyDetect = true
	m.extendedResultCodes = true
	m.dcdControl = false	
	m.connectMsgSpeed = true
	registers.Reset()
	phonebook = NewPhonebook(*_flags_phoneBook, logger)
	err = phonebook.Load()
	if err != nil {
		logger.Print(err)
	}

	resetTimer()
	return err
}

// AT&V
func amperV() error {
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
	s += "L" + i(m.speakerVolume)
	s += "M" + i(m.speakerMode)
	s += "Q" + b(m.quiet)
	s += "V" + b(m.verbose)
	s += "W" + b(m.connectMsgSpeed)
	s += "X" + x(m.extendedResultCodes, m.busyDetect)
	s += "&C" + b(m.dcdControl)
	s += "\n"
	s += registers.String()
	serial.Println(s)
	return OK
}

// AT&...
// Only support &V and &C for now
func processAmpersand(cmd string) error {

	logger.Print(cmd)
	if cmd[:2] == "&Z" {
		var s string
		var i int
		_ , err := fmt.Sscanf(cmd, "&Z%d=%s", &i, &s)
		if err != nil {
			logger.Print(err)
			return err
		}
		if s[0] == 'D' || s[0] == 'd' { // Extension
			return phonebook.Delete(i)
		}
		return phonebook.Add(i, s)
	}

	switch cmd {
	case "&C0":
		m.dcdControl = false
		return OK
	case "&C1":
		m.dcdControl = true
		return OK
	case "&V0": return amperV()
	}
	return ERROR
}


// process each command
func processCommands(commands []string) error {
	var status error
	var cmd string

	status = OK
	for _, cmd = range commands {
		logger.Printf("Processing: %s", cmd)
		switch cmd[0] {
		case 'A':
			status = answer()
		case 'Z':
			status = reset()
			time.Sleep(250 * time.Millisecond)
			raiseDSR()
			raiseCTS()
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
				status = goOnHook()
			} else if cmd[1] == '1' {
				status = goOffHook()
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
			case '0': m.speakerVolume = 0
			case '1': m.speakerVolume = 1
			case '2': m.speakerVolume = 2
			case '3': m.speakerVolume = 3
			}
		case 'M':
			switch cmd[1] {
			case '0': m.speakerMode = 0
			case '1': m.speakerMode = 1
			case '2': m.speakerMode = 2
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
			status = dial(cmd)
		case 'S':
			status = registerCmd(cmd)
		case '&':
			status = processAmpersand(cmd)
		case '*':
			status = debug(cmd)
		default:
			status = ERROR
		}
		if status != OK {
			break
		}
	}
	return status
}
