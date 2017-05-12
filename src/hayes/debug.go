package hayes

import (
	"fmt"
	"os"
	"sort"
)

var debug bool = false
//var debug bool = true

func debugf(format string, a ...interface{}) {
	if debug {
		format = "# " + format + "\n"
		fmt.Fprintf(os.Stderr, format, a...)
	}
}

func (m *Modem) setupDebug() {
	for i := range m.d {
		m.d[i] = 0
	}
}

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
	fmt.Printf("OnHook   : %t\n", m.onhook)
	fmt.Printf("Echo     : %t\n", m.onhook)
	fmt.Print( "Mode     : ")
	if m.mode == COMMANDMODE {
		fmt.Println("Command")
	} else {
		fmt.Println("Data")
	}
	fmt.Printf("Quiet    : %t\n", m.quiet)
	fmt.Printf("Verbose  : %t\n", m.verbose)
	fmt.Printf("Volume   : %d\n", m.volume)
	fmt.Printf("SpkrMode : %d\n", m.speakermode)
	fmt.Printf("Last Cmd : %v\n", m.lastcmds)
	fmt.Printf("Last num : %s\n", m.lastdialed)
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
func (m *Modem) debug(cmd string) (int) {
	var err error
	var reg, val int
	
	// NOTE: The order of these stanzas is critical.

	// *n=x - write x to n
	_, err = fmt.Sscanf(cmd, "*%d=%d", &reg, &val)
	if err == nil {
		m.d[reg] = val
		if reg == 0 {
			if m.d[0] != 0 {
				debug = true
				debugf("Debugging enabled")
			} else {
				debugf("Debugging disabled")
				debug = false
			}
		}
		return OK
	}

	// *n? - query register n
	_, err = fmt.Sscanf(cmd, "*%d?", &reg)
	if err == nil {
		fmt.Printf("%d\n", m.d[reg])
		return OK
	}

	return ERROR
}
