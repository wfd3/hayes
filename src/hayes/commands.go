package hayes

import (
	"time"
	"strings"
	"fmt"
)

// ATH0
func (m *Modem) onHook() (int) {
	m.lowerCD()

	// It's OK to hang up the phone when there's no active network connection.
	// But if there is, close it.
	if m.conn != nil {
		m.conn.Close()
		m.conn = nil
	}

	m.onhook = true
	m.mode = COMMANDMODE
	m.connect_speed = 0
	m.led_HS_off()
	m.led_OH_off()
	return OK
}

const ON_HOOK = true
const OFF_HOOK = false

// ATH1
func (m *Modem) offHook() int {
	m.onhook = OFF_HOOK
	m.led_OH_on()
	return OK
}

func (m *Modem) getHook() bool {
	return m.onhook
}

// ATA
func (m *Modem) answer() (int) {
	if !m.getLineBusy()  || !m.getHook() {
		return ERROR
	}
	
	m.offHook()
	time.Sleep(400 * time.Millisecond) // Simulate Carrier Detect delay
	m.raiseCD()
	m.mode = DATAMODE
	m.connect_speed = 38400	// We only go fast...
	return CONNECT
}

// process each command
func (m *Modem) processCommands(commands []string) (int) {
	var status int
	var cmd string

	m.log.Printf("entering PC: %v\n", commands)
	status = OK
	for _, cmd = range commands {
		m.log.Printf("Processing: %s", cmd)
		switch cmd[0] {
		case '/':
			status = m.processCommands(m.lastcmds) 
		case 'A':
			status = m.answer()
		case 'Z':
			status = m.reset()
			time.Sleep(250 * time.Millisecond)
			m.raiseDSR()
			m.raiseCTS()
		case 'E':
			if cmd[1] == '0' {
				m.echo = false
			} else {
				m.echo = true
			}
		case 'F':	// Online Echo mode, F1 assumed for backwards
			        // compatability after Hayes 1200
			status = OK 
		case 'H':
			if cmd[1] == '0' { 
				status = m.onHook()
			} else if cmd[1] == '1' {
				status = m.offHook()
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
		case 'X':
			m.printState()
			status = OK
		case 'D':
			status = m.dial(cmd)
		case 'S':
			status = m.registers(cmd)
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

	if len(cmd) == 1 {
		if cmd[0] == '/' {
			// '/' is special as it's the only true one char command
			return "/", 1, nil 
		}
		return cmd + "0", 1, nil
	} 

	if strings.ContainsAny(cmd[1:2], opts) {
		return cmd[:2],  2, nil
	}

	return "", 0, fmt.Errorf("Bad command: %s", cmd)
}

// +++ 
func (m *Modem) command(cmd string) {
	var commands []string
	var s, opts string
	var i int
	var status int
	var err error

	// Process here is to parse the entire command string into
	// discrete commands, then execute those discrete commands in
	// the order they were given to us.  This makes syntax
	// checking/failures happen before any commands are executed
	// which is, if I recall correctly, how this works in the real
	// hardware

	cmd = strings.ToUpper(cmd)
	m.log.Print("command: ", cmd)
	
	if len(cmd) < 2  || (!(cmd[0] == 'A' && cmd[1] == 'T')) {
		m.prstatus(ERROR)
		return
	}
	
	cmd = cmd[2:] 		// Skip the 'AT'
	c := 0

	commands = nil
	status = OK
	savecmds := true
	for  c < len(cmd) && status == OK {
		switch (cmd[c]) {
		case 'D':
			s, i, err = parseDial(cmd[c:])
			if err != nil {
				m.prstatus(ERROR)
				return
			} 
			commands = append(commands, s)
			c += i
			continue
		case 'S':
			s, i, err = parseRegisters(cmd[c:])
			if err != nil {
				m.prstatus(ERROR)
				return
			}
			commands = append(commands, s)
			c += i
			continue
		case '*': 	// Custom debug registers
			s, i, err = parseDebug(cmd[c:])
			if err != nil {
				m.prstatus(ERROR)
				return
			}
			commands = append(commands, s)
			c += i
			continue
		case '/':
			opts = ""
			savecmds = false
		case 'A':
			opts = "0"
		case 'E', 'H', 'Q', 'V', 'Z':
			opts = "01"
		case 'L':
			opts = "0123"
		case 'M':
			opts = "012"
		case 'O':
			opts = "O"
		case 'X':
			opts = "01234"
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
	
	status = m.processCommands(commands)
	m.prstatus(status)

	if savecmds {
		m.lastcmds = commands
	}
}
