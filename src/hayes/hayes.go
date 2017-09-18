package hayes

//
// Pretend to be a Hayes modem.
//
// References:
// - Hayes command/error documentation:
//    http://www.messagestick.net/modem/hayes_modem.html#Introduction
// - Sounds: https://en.wikipedia.org/wiki/Precise_Tone_Plan
// - RS232: https://en.wikipedia.org/wiki/RS-232
// - Serial Programming: https://en.wikibooks.org/wiki/Serial_Programming
// - Raspberry PI lib: github.com/stianeikeland/go-rpio
//

import (
	"os"
	"os/signal"
	"log"
	"time"
	"io"
	"sync"
	"runtime/pprof"
	"syscall"
)

// What mode is the modem in?
const (
	COMMANDMODE = iota
	DATAMODE
)

// Is the phone on or off hook?
const (
	OFFHOOK bool = false
	ONHOOK  bool = true
)

// Is the network connection SSH or Telnet?
const (
	TELNET = iota
	SSH
)

//Basic modem struct
type Modem struct {
	echoInCmdMode bool
	speakermode int
	volume int
	verbose bool
	quiet bool

	mode int
	onhook bool
	lastcmd string
	lastdialed string
	connect_speed int
	linebusy bool
	linebusylock sync.RWMutex

	conn io.ReadWriteCloser
	serial *serialPort
	pins Pins
	leds Pins
	addressbook *Addressbook
	registers *Registers
	log *log.Logger
	timer *time.Ticker
}

// Is the phone line busy?
func (m *Modem) getLineBusy() bool {
	m.linebusylock.RLock()
	defer m.linebusylock.RUnlock()
	return m.linebusy
}	

func (m *Modem) setLineBusy(b bool) {
	m.linebusylock.Lock()
	defer m.linebusylock.Unlock()
	m.linebusy = b
}

// Watch a subset of pins and act as apropriate
// Must be a goroutine
func (m *Modem) handlePINs() {
	for {
		if m.readDTR() {
			m.led_TR_on()
		} else { 
			if m.getHook() == OFF_HOOK && m.conn != nil {
				m.onHook()
			}
			m.led_TR_off()
		}

		if m.connect_speed > 19200 {
			m.led_HS_on()
		} else {
			m.led_HS_off()
		}
			
		time.Sleep(250 * time.Millisecond)
	}
}

// Catch ^C, reset the HW pins
// Must be a goroutine
func (m *Modem) handleSignals() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGUSR1)

	for {
		// Block until a signal is received.
		s := <-c
		m.log.Print("Caught signal: ", s)
		switch s {
		case syscall.SIGINT:
			m.clearPins()
			m.log.Print("Exiting")
			os.Exit(0)

		case syscall.SIGUSR1:
			pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
		}
	}
}

// Boot the modem
func (m *Modem) PowerOn(log *log.Logger) {

	m.log = log
	m.log.Print("------------ Starting up")
	m.registers = NewRegisters()
	m.setupPins()	      
	m.reset()	      // Setup modem inital state (or reset initial state)
	m.serial = setupSerialPort(*_flags_console, m.registers, m.log)
	
	go m.handleSignals()	// Catch signals in a different thread
	go m.handlePINs()       // Monitor input pins & internal registers
	go m.handleModem()	// Handle in-bound bytes in a seperate goroutine

	// Signal to DTE that we're ready
	time.Sleep(250 * time.Millisecond) // make it look good
	m.raiseDSR()
	m.raiseCTS()

	// Tell user we're ready
	m.prstatus(OK)
	m.log.Print("Modem Ready")

	m.readSerial()		// never returns
}


