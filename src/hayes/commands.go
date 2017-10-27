package main

import (
	"time"
	"fmt"
)

// ATZn - 0 == config 0, 1 == config 1
func softReset(i int) error {
	c, r, err := profiles.Switch(i)
	if err != nil {
		return err
	}
	logger.Printf("Switching config/registers")
	factoryReset()
	conf = c
	registers = r

	return nil
}

// AT&F - reset to factory defaults
func factoryReset() error {
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

	
	registers.Reset()
	conf.Reset()

	phonebook = NewPhonebook(*_flags_phoneBook, logger)
	err = phonebook.Load()
	if err != nil {
		logger.Print(err)
	}
	
	resetTimer()
	return err
}

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

// AT&V
// TODO: THis needs to be fixed
func amperV() error {
	serial.Println("ACTIVE PROFILE:")
	serial.Println(conf.String())
	serial.Println(registers)
	serial.Println()

	serial.Println(profiles)
	
	serial.Println("TELEPHONE NUMBERS:")
	serial.Println(phonebook)

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
		conf.dcdControl = false
		return OK
	case "&C1":
		conf.dcdControl = true
		return OK
	case "&F0":
		return factoryReset()
	case "&V0":
		return amperV()
	case "&W0":
		return profiles.writeActive(0)
	case "&W1":
		return profiles.writeActive(1)
	case "&Y0":
		return profiles.setPowerUpConfig(0)
	case "&Y1":
		return profiles.setPowerUpConfig(1)

	}
	return ERROR
}


// process each command
func processSingleCommand(cmd string) error {
	var status error

	switch cmd[0] {
	case 'A':
		status = answer()
	case 'Z':
		var c int
		switch cmd[1] {
		case '0': c = 0
		case '1': c = 1
		}				
		status = softReset(c)
		time.Sleep(250 * time.Millisecond)
		if status == OK {
			raiseDSR()
			raiseCTS()
		}
	case 'E':
		conf.echoInCmdMode = cmd[1] == '0' 
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
		conf.quiet = cmd[1] == '0'
	case 'V':
		conf.verbose = cmd[1] == '0'
	case 'L':
		switch cmd[1] {
		case '0': conf.speakerVolume = 0
		case '1': conf.speakerVolume = 1
		case '2': conf.speakerVolume = 2
		case '3': conf.speakerVolume = 3
		}
	case 'M':
		switch cmd[1] {
		case '0': conf.speakerMode = 0
		case '1': conf.speakerMode = 1
		case '2': conf.speakerMode = 2
		}
	case 'O':
		m.mode = DATAMODE
		status = OK
	case 'W':
		switch cmd[1] {
		case '0': conf.connectMsgSpeed = false
		case '1', '2': conf.connectMsgSpeed = true
		default: status = ERROR
		}
	case 'X':	// Change result codes displayed
		switch cmd[1] {
		case '0':
			conf.extendedResultCodes = false
			conf.busyDetect = false
		case '1', '2':
			conf.extendedResultCodes = true
			conf.busyDetect = false
		case '3', '4', '5', '6', '7':
			conf.extendedResultCodes = true
			conf.busyDetect = true
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

	return status
}

func processCommands(commands []string) error {
	var cmd string 
	var status error
	
	status = OK
	for _, cmd = range commands {
		logger.Printf("Processing: %s", cmd)
		status = processSingleCommand(cmd)
		if status != OK {
			return status
		}
	}
	return status
}
