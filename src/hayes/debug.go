package main

import (
	"code.cloudfoundry.org/bytefmt"
	"fmt"
	"net"
	"runtime"
	"strings"
	"time"
)

func logf(format string, a ...interface{}) {
	out := fmt.Sprintf(format, a...)
	out = strings.Replace(out, "\n", "; ", -1)
	out = strings.TrimRight(out, "; ")
	logger.Print(out)
}
func pf(format string, a ...interface{}) {
	serial.Printf(format, a...)
}

type out func(string, ...interface{})

// Debug function
func outputState(debugf out) {

	debugf("Modem state:\n")
	debugf(" currentconfig: %d\n", m.currentConfig)
	switch m.getMode() {
	case COMMANDMODE:
		debugf(" mode         : COMMAND\n")
	case DATAMODE:
		debugf(" mode         : DATA\n")
	}
	debugf(" lastCmd      : %s\n", m.lastCmd)
	debugf(" lastDialed   : %s\n", m.lastDialed)
	debugf(" connectSpeed : %d\n", m.getConnectSpeed())
	debugf(" dcd          : %t\n", m.getdcd())
	debugf(" lineBusy     : %t\n", m.getLineBusy())
	debugf(" onHook       : %t\n", m.onHook())

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

	debugf("Curent register: %d\n", registers.ShowCurrent())
	debugf("Registers: %s\n", registers.String())

	debugf("Phonebook: %s\n", phonebook.String())

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

// Show the user what our current network status is.
func networkStatus() {
	serial.Println("LISTENING ON:")
	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		for _, a := range addrs {
			ip, _, _ := net.ParseCIDR(a.String())
			if !ip.IsMulticast() && !ip.IsLoopback() &&
				!ip.IsUnspecified() && !ip.IsLinkLocalUnicast() {
				serial.Printf("  Interface %s: %s\n", i.Name, ip)
			}
		}
	}
	serial.Println("ACTIVE PROTOCOLS:")
	if flags.telnet {
		serial.Printf("  Telnet (%d)\n", flags.telnetPort)
	}
	if flags.ssh {
		serial.Printf("  SSH (%d)\n", flags.sshdPort)
	}

	serial.Println("ACTIVE CONNECTION:")
	if m.conn != nil {
		serial.Printf("  %s\n", m.conn)
	} else {
		serial.Println("  NONE")
	}
		
}

func toggleRS232() {
	serial.Println("Toggling RS232 lines")
	serial.Printf("Current Pin Status: %s\n", showPins())
	for i :=0; i<5; i++ {
		raiseCD();
		time.Sleep(250 * time.Millisecond)
		lowerCD()
		time.Sleep(250 * time.Millisecond)
	}
	serial.Printf("Current Pin Status: %s\n", showPins())	
}

func help() {
	serial.Println("Debug commands:")
	serial.Println("AT*        - show internal state")
	serial.Println("AT*network - show network status")
	serial.Println("AT*ledtest - run the LED test")
	serial.Println("AT*help    - this help")
	serial.Println("AT*232     - toggle RS232 lines")
}

// Given a parsed register command, execute it.
func debug(cmd string) error {
	logger.Printf("cmd = '%s'", cmd)

	switch {
	case cmd == "*":
		showState()
		logState()
	case cmd == "*help":
		help()
	case cmd == "*ledtest":
		ledTest(5)
	case cmd == "*network":
		networkStatus()
	case cmd == "*232":
		toggleRS232()
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
