package main

import (
	"fmt"
	"time"
	"runtime"
	"code.cloudfoundry.org/bytefmt"
)

func logf(format string, a ...interface{}) {
	logger.Printf(format, a...)
}
func pf(format string, a ...interface{}) {
	serial.Printf(format, a...)
}

type out func(string, ...interface{})

// Debug function
func outputState(debugf out)  {
	
	if onHook() {
		debugf("Hook      : ON HOOK\n")
	} else {
		debugf("Hook      : OFF HOOK\n")	
	}
	debugf("EchoInCmd : %t\n", conf.echoInCmdMode)
	if m.mode == COMMANDMODE {
		debugf( "Mode      : Command\n")
	} else {
		debugf( "Mode      : Data\n")
	}
	debugf("Quiet     : %t\n", conf.quiet)
	debugf("Verbose   : %t\n", conf.verbose)
	debugf("Line Busy : %t\n", getLineBusy())
	debugf("Speed     : %d\n", m.connectSpeed)
	debugf("Volume    : %d\n", conf.speakerVolume)
	debugf("SpkrMode  : %d\n", conf.speakerMode)
	debugf("Last Cmd  : %s\n", m.lastCmd)
	debugf("Last num  : %s\n", m.lastDialed)
	debugf("Phonebook:\n")
	debugf("%s\n", phonebook.String())
	debugf("Cur reg   : %d\n", registers.ShowCurrent())
	debugf("Registers:\n")
	debugf("%s\n", registers.String())
	if netConn != nil {
		sent, recv := netConn.Stats()
		debugf("Connection: %s, tx: %s rx: %s\n", netConn.RemoteAddr(),
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

// AT*... debug command
// Given a string that looks like a "*" debug command, parse & normalize it
func parseDebug(cmd string) (string, int, error) {
	var s string
	var err error
	var reg, val int

	// NOTE: The order of these stanzas is critical.

	if len(cmd) == 1 && cmd[0] == '*' {
		return "*", 1, nil
	}
	
	if  len(cmd) < 2  {
		return "", 0, fmt.Errorf("Bad command: %s", cmd)
	}

	// S? - query selected register
	if cmd[:2] == "*?" {
		s = "*?"
		return s, 2, nil
	}

	// Sn=x - write x to n
	_, err = fmt.Sscanf(cmd, "*%d=%d", &reg, &val)
	if err == nil {
		s = fmt.Sprintf("*%d=%d", reg, val)
		return s, len(s), nil
	}

	// Sn? - query register n
	_, err = fmt.Sscanf(cmd, "*%d?", &reg)
	if err == nil {
		s = fmt.Sprintf("*%d?", reg)
		return s, len(s), nil
	}

	return "", 0, fmt.Errorf("Bad * command: %s", cmd)
}

// Given a parsed register command, execute it.
func debug(cmd string) error {
	var err error
	var reg, val int
	
	// NOTE: The order of these stanzas is critical.

	logger.Printf("cmd = '%s'", cmd)

	if cmd == "*" {
		showState()
		logState()
		return nil
	}

	// *n=x - write x to n
	_, err = fmt.Sscanf(cmd, "*%d=%d", &reg, &val)
	if err != nil {
		logger.Print(err)
		return err
	}

	switch reg {
	case 1:		// Toggle DSR/CTS
		switch val {
		case 1:
			lowerDSR()
			lowerCTS()
		case 2:
			raiseDSR()
			raiseCTS()
		}
	case 2:		// Run ledTest
		ledTest(val)
	case 3:
		for i := 0; i < val; i++ {
			showPins()
			time.Sleep(500 * time.Millisecond)
		}
	case 4:
		phonebook.Write()
	case 8:		// Toggle CD pin val times
		for i := 0; i < val; i++ {
			serial.Println("Toggling CD up")
			raiseCD()
			time.Sleep(2 * time.Second)
			serial.Println("Toggling CD down")
			lowerCD()
			time.Sleep(2 * time.Second)
		}
	case 9:		// Toggle RI pin val times
		for i := 0; i < val; i++ {
			serial.Println("Toggling RI up")
			raiseRI()
			time.Sleep(2 * time.Second)
			serial.Println("Toggling RI down")
			lowerRI()
			time.Sleep(2 * time.Second)
		}
	case 10: 	// Toggle DSR
		for i := 0; i < val; i++ {
			serial.Println("Toggling DSR up")
			raiseDSR()
			time.Sleep(2 * time.Second)
			serial.Println("Toggling DSR down")
			lowerDSR()
			time.Sleep(2 * time.Second)
		}
	case 11: 	// Toggle CTS
		for i := 0; i < val; i++ {
			serial.Println("Toggling CTS up")
			raiseCTS()
			time.Sleep(2 * time.Second)
			serial.Println("Toggling CTS down")
			lowerCTS()
			time.Sleep(2 * time.Second)
		}
	case 99: 		// All output
		for i := 0; i < val; i++ {
			serial.Println("Rasising: CD, RI, DSR, CTS")
			raiseCD()
			raiseRI()
			raiseDSR()
			raiseCTS()
			time.Sleep(2 * time.Second)
			serial.Println("Lowering: CD, RI, DSR, CTS")
			lowerCD()
			lowerRI()
			lowerDSR()
			lowerCTS()
			time.Sleep(2 * time.Second)
		}
	}
	return nil
}
