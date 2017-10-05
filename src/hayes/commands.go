package hayes

import (
	"time"
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
	m.connectSpeed = 38400	// We only go fast...
	return CONNECT
}

// ATZ
// Setup/reset modem.  Leaves RTS & CTS down.
func (m *Modem) reset() error {
	var err error = OK

	m.log.Print("Resetting modem")

	// Reset state
	m.goOnHook()
	m.setLineBusy(false)
	m.lowerDSR()
	m.lowerCTS()
	m.lowerRI()
	m.stopTimer()
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
	m.resetRegs()
	m.phonebook = NewPhonebook(*_flags_phoneBook, m.log)
	err = m.phonebook.Load()
	if err != nil {
		m.log.Print(err)
	}

	m.resetTimer()
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
	s += "L" + i(m.speakerVolume)
	s += "M" + i(m.speakerMode)
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
func (m *Modem) processAmpersand(cmd string) error {

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
			status = m.dial(cmd)
		case 'S':
			status = m.registerCmd(cmd)
		case '&':
			status = m.processAmpersand(cmd)
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
