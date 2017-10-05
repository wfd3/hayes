package hayes

import (
	"fmt"
	"time"
	"runtime"
	"code.cloudfoundry.org/bytefmt"
)

func (m *Modem) logf(format string, a ...interface{}) {
	m.log.Printf(format, a...)
}
func (m *Modem) pf(format string, a ...interface{}) {
	m.serial.Printf(format, a...)
}

type out func(string, ...interface{})

// Debug function
func (m *Modem) outputState(debugf out)  {
	
	if m.onHook() {
		debugf("Hook      : ON HOOK\n")
	} else {
		debugf("Hook      : OFF HOOK\n")	
	}
	debugf("EchoInCmd : %t\n", m.echoInCmdMode)
	if m.mode == COMMANDMODE {
		debugf( "Mode      : Command\n")
	} else {
		debugf( "Mode      : Data\n")
	}
	debugf("Quiet     : %t\n", m.quiet)
	debugf("Verbose   : %t\n", m.verbose)
	debugf("Line Busy : %t\n", m.getLineBusy())
	debugf("Speed     : %d\n", m.connectSpeed)
	debugf("Volume    : %d\n", m.speakerVolume)
	debugf("SpkrMode  : %d\n", m.speakerMode)
	debugf("Last Cmd  : %s\n", m.lastCmd)
	debugf("Last num  : %s\n", m.lastDialed)
	debugf("Phonebook:\n")
	debugf("%s\n", m.phonebook.String())
	debugf("Cur reg   : %d\n", m.registers.ShowCurrent())
	debugf("Registers:\n")
	debugf("%s\n", m.registers.String())
	if m.conn != nil {
		sent, recv := m.conn.Stats()
		debugf("Connection: %s, tx: %s rx: %s\n", m.conn.RemoteAddr(),
			bytefmt.ByteSize(sent), bytefmt.ByteSize(recv))
	} else {
		debugf("Connection: <Not connected>\n")
	}
	debugf("%s\n", m.showPins())
	debugf("GoRoutines: %d\n", runtime.NumGoroutine())

}

func (m *Modem) showState() {
	m.outputState(m.pf)
}

func (m *Modem) logState() {
	m.outputState(m.logf)
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
func (m *Modem) debug(cmd string) error {
	var err error
	var reg, val int
	
	// NOTE: The order of these stanzas is critical.

	m.log.Printf("cmd = '%s'", cmd)

	if cmd == "*" {
		m.showState()
		m.logState()
		return nil
	}

	// *n=x - write x to n
	_, err = fmt.Sscanf(cmd, "*%d=%d", &reg, &val)
	if err != nil {
		m.log.Print(err)
		return err
	}

	switch reg {
	case 1:		// Toggle DSR/CTS
		switch val {
		case 1:
			m.lowerDSR()
			m.lowerCTS()
		case 2:
			m.raiseDSR()
			m.raiseCTS()
		}
	case 2:		// Run ledTest
		m.ledTest(val)
	case 3:
		for i := 0; i < val; i++ {
			m.showPins()
			time.Sleep(500 * time.Millisecond)
		}
	case 4:
		m.phonebook.Write()
	case 8:		// Toggle CD pin val times
		for i := 0; i < val; i++ {
			m.serial.Println("Toggling CD up")
			m.raiseCD()
			time.Sleep(2 * time.Second)
			m.serial.Println("Toggling CD down")
			m.lowerCD()
			time.Sleep(2 * time.Second)
		}
	case 9:		// Toggle RI pin val times
		for i := 0; i < val; i++ {
			m.serial.Println("Toggling RI up")
			m.raiseRI()
			time.Sleep(2 * time.Second)
			m.serial.Println("Toggling RI down")
			m.lowerRI()
			time.Sleep(2 * time.Second)
		}
	case 10: 	// Toggle DSR
		for i := 0; i < val; i++ {
			m.serial.Println("Toggling DSR up")
			m.raiseDSR()
			time.Sleep(2 * time.Second)
			m.serial.Println("Toggling DSR down")
			m.lowerDSR()
			time.Sleep(2 * time.Second)
		}
	case 11: 	// Toggle CTS
		for i := 0; i < val; i++ {
			m.serial.Println("Toggling CTS up")
			m.raiseCTS()
			time.Sleep(2 * time.Second)
			m.serial.Println("Toggling CTS down")
			m.lowerCTS()
			time.Sleep(2 * time.Second)
		}
	case 99: 		// All output
		for i := 0; i < val; i++ {
			m.serial.Println("Rasising: CD, RI, DSR, CTS")
			m.raiseCD()
			m.raiseRI()
			m.raiseDSR()
			m.raiseCTS()
			time.Sleep(2 * time.Second)
			m.serial.Println("Lowering: CD, RI, DSR, CTS")
			m.lowerCD()
			m.lowerRI()
			m.lowerDSR()
			m.lowerCTS()
			time.Sleep(2 * time.Second)
		}
	}
	return nil
}
