package main

import (
	"code.cloudfoundry.org/bytefmt"
	"fmt"
	"runtime"
)

func logf(format string, a ...interface{}) {
	logger.Printf(format, a...)
}
func pf(format string, a ...interface{}) {
	serial.Printf(format, a...)
}

type out func(string, ...interface{})

// Debug function
func outputState(debugf out) {

	debugf("Modem state:\n")
	debugf(" currentconfig: %d\n", m.currentConfig)
	switch m.mode {
	case COMMANDMODE:
		debugf(" mode         : COMMAND\n")
	case DATAMODE:
		debugf(" mode         : DATA\n")
	}
	debugf(" lastCmd      : %s\n", m.lastCmd)
	debugf(" lastDialed   : %s\n", m.lastDialed)
	debugf(" connectSpeed : %d\n", m.connectSpeed)
	debugf(" dcd          : %t\n", m.dcd)
	debugf(" lineBusy     : %t\n", getLineBusy())
	debugf(" onHook       : %t\n", onHook())

	debugf("Config:\n")
	debugf(" echoInCmdMode : %t\n", conf.echoInCmdMode)
	debugf(" speakerMode   : %d\n", conf.speakerMode)
	debugf(" speakerVolume : %d\n", conf.speakerVolume)
	debugf(" verbose       : %t\n", conf.verbose)
	debugf(" quiet         : %t\n", conf.quiet)
	debugf(" connctMsgSpeed: %t\n", conf.connectMsgSpeed)
	debugf(" busyDetect    : %t\n", conf.busyDetect)
	debugf(" extResultCodes: %t\n", conf.extendedResultCodes)
	debugf(" dcdPinned     : %t\n", conf.dcdPinned)
	debugf(" dsrPinned     : %t\n", conf.dsrPinned)
	debugf(" dtr           : %d\n", conf.dtr)

	debugf("Phonebook:\n")
	debugf("%s\n", phonebook.String())

	debugf("Registers:\n")
	debugf("Curent register: %d\n", registers.ShowCurrent())
	debugf("%s\n", registers.String())

	if m.conn != nil {
		sent, recv := m.conn.Stats()
		debugf("Connection: %s, tx: %s rx: %s\n", m.conn.RemoteAddr(),
			bytefmt.ByteSize(sent), bytefmt.ByteSize(recv))
	} else {
		debugf("Connection: <Not connected>\n")
	}
	
	debugf("%s\n", showPins())
	debugf("GoRoutines: %d\n", runtime.NumGoroutine())
}

func showState() {
	outputState(pf)
}

func logState() {
	outputState(logf)
}

// Given a parsed register command, execute it.
func debug(cmd string) error {
	logger.Printf("cmd = '%s'", cmd)

	switch {
	case cmd == "*":
		showState()
		logState()
	case cmd == "*ledtest":
		ledTest(5)
	default:
		return fmt.Errorf("Bad debug command: %s", cmd)
	}

	return nil
}


// AT*... debug command
// Given a string that looks like a "*" debug command, parse & normalize it
func parseDebug(cmd string) (string, int, error) {

	logger.Printf("parseDebug(): %s", cmd)

	// Naked AT*
	if len(cmd) == 1 && cmd[0] == '*' {
		return "*", 1, nil
	}

	// Too short
	if len(cmd) < 2 {
		return "", 0, fmt.Errorf("Bad command: %s", cmd)
	}

	// Doesn't start with debug character
	if cmd[0] != '*' {
		return "", 0, fmt.Errorf("Bad command: %s", cmd)
	}

	// It's OK!
	return cmd, len(cmd), nil
}
