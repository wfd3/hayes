package hayes

import (
	"fmt"
	"sort"
	"time"
)

func (m *Modem) printRegs() {
	var s string
	var i []int
	
	for f := range m.r {
		i = append(i, int(f))
	}
	sort.Ints(i)

	fmt.Println("Registers:")
	for _, f := range i {
		s += fmt.Sprintf("S%02d:%03d ", f, m.r[byte(f)])
		if (len(s) + 6) >80  {
			fmt.Println(s)
			s = ""
		}
	}
	fmt.Println(s)
}

// Debug function
func (m *Modem) printState() {
	fmt.Printf("Hook     : ")
	if m.getHook() == ON_HOOK {
		fmt.Println("ON HOOK")
	} else {
		fmt.Println("OFF HOOK")
	}
	fmt.Printf("Echo     : %t\n", m.echo)
	fmt.Print( "Mode     : ")
	if m.mode == COMMANDMODE {
		fmt.Println("Command")
	} else {
		fmt.Println("Data")
	}
	fmt.Printf("Quiet    : %t\n", m.quiet)
	fmt.Printf("Verbose  : %t\n", m.verbose)
	fmt.Printf("Line Busy: %t\n", m.getLineBusy())
	fmt.Printf("Speed    : %d\n", m.connect_speed)
	fmt.Printf("Volume   : %d\n", m.volume)
	fmt.Printf("SpkrMode : %d\n", m.speakermode)
	fmt.Printf("Last Cmd : %v\n", m.lastcmds)
	fmt.Printf("Last num : %s\n", m.lastdialed)
	m.printAddressBook()
	fmt.Printf("Cur reg  : %d\n", m.curreg)
	m.printRegs()
	fmt.Printf("Connection: %v\n", m.conn)
	m.showPins()
}


// AT*... debug command
// Given a string that looks like a "*" debug command, parse & normalize it
func parseDebug(cmd string) (string, int, error) {
	var s string
	var err error
	var reg, val int

	// NOTE: The order of these stanzas is critical.

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

	// *n=x - write x to n
	_, err = fmt.Sscanf(cmd, "*%d=%d", &reg, &val)
	if err == nil {
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
		case 8:		// Toggle CD pin val times
			for i := 0; i < val; i++ {
				fmt.Println("Toggling CD up")
				m.raiseCD()
				time.Sleep(2 * time.Second)
				fmt.Println("Toggling CD down")
				m.lowerCD()
				time.Sleep(2 * time.Second)
			}
		case 9:		// Toggle RI pin val times
			for i := 0; i < val; i++ {
				fmt.Println("Toggling RI up")
				m.raiseRI()
				time.Sleep(2 * time.Second)
				fmt.Println("Toggling RI down")
				m.lowerRI()
				time.Sleep(2 * time.Second)
			}
		case 10: 	// Toggle DSR
			for i := 0; i < val; i++ {
				fmt.Println("Toggling DSR up")
				m.raiseDSR()
				time.Sleep(2 * time.Second)
				fmt.Println("Toggling DSR down")
				m.lowerDSR()
				time.Sleep(2 * time.Second)
			}
		case 11: 	// Toggle CTS
			for i := 0; i < val; i++ {
				fmt.Println("Toggling CTS up")
				m.raiseCTS()
				time.Sleep(2 * time.Second)
				fmt.Println("Toggling CTS down")
				m.lowerCTS()
				time.Sleep(2 * time.Second)
			}
		case 99: 		// All output
			for i := 0; i < val; i++ {
				fmt.Println("Rasising: CD, RI, DSR, CTS")
				m.raiseCD()
				m.raiseRI()
				m.raiseDSR()
				m.raiseCTS()
				time.Sleep(2 * time.Second)
				fmt.Println("Lowering: CD, RI, DSR, CTS")
				m.lowerCD()
				m.lowerRI()
				m.lowerDSR()
				m.lowerCTS()
				time.Sleep(2 * time.Second)
			}
		}
		return OK
	}

	return ERROR
}
