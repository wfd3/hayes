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

// AT&V
// TODO: THis needs to be fixed
func amperV() error {
	serial.Println("ACTIVE PROFILE:")
	serial.Println(profile.Active())

	serial.Printf("\nSTORED PROFILE \n")
	serial.Println("profile.String()")

	serial.Println("\nTELEPHONE NUMBERS:")
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
			conf = profile.Switch(c) // TODO: correct?
			raiseDSR()
			raiseCTS()
		}
		case 'E':
		if cmd[1] == '0' {
			conf.echoInCmdMode = false
		} else {
			conf.echoInCmdMode = true
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
			conf.quiet = true
		} else {
			conf.quiet = false
		}
	case 'V':
		if cmd[1] == '0' {
			conf.verbose = true
		} else {
			conf.verbose = false
		}
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
