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

////////////////////////////////////////////////////////////////////////////////////

const (
	COMMANDMODE = iota
	DATAMODE
)

const OFFHOOK = false
const ONHOOK = true

//Basic modem struct
type Modem struct {
	mode int
	onhook bool
	echo bool
	speakermode int
	volume int
	verbose bool
	quiet bool
	lastcmds []string
	lastdialed string
	rlock sync.RWMutex	// Lock for registers map (r)
	r map[byte]byte
	curreg int
	conn io.ReadWriteCloser
	serial *serialPort
	pins Pins
	leds Pins
	d [10]int
	connect_speed int
	linebusy bool
	linebusylock sync.RWMutex
	addressbook map[string] *ab_host
	log *log.Logger
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

// Watch a subset of pins and registers and toggle the LED as apropriate
// Must be a goroutine
func (m *Modem) handlePINs() {
	for {
		if m.readDTR() {
			m.led_TR_on()
		} else { 
			if m.getHook() == OFF_HOOK && m.conn != nil {
				// DTE Dropped DTR, hang up the phone if DTR is not
				// reestablished withing S25 * 1/100's of a second
				time.Sleep(time.Duration(m.readReg(REG_DTR_DELAY)) *
					100 * time.Millisecond)
				if !m.readDTR() && m.getHook() == OFF_HOOK &&
					m.conn != nil {
					m.onHook()
				}
			}
			m.led_TR_off()
		}

		if m.connect_speed > 19200 {
			m.led_HS_on()
		} else {
			m.led_HS_off()
		}
			

		// debug
		if m.d[1] == 2 {
			m.raiseDSR()
			m.raiseCTS()
			m.d[1] = 0
		}
		if m.d[1] == 1 {
			m.lowerDSR()
			m.lowerCTS()
			m.d[1] = 0
		}

		if m.d[2] != 0 {
			m.ledTest(m.d[2])
			m.d[2] = 0
		}

		time.Sleep(250 * time.Millisecond)
	}
}

// Catch ^C, reset the HW pins
func (m *Modem) signalHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	for {
		// Block until a signal is received.
		s := <-c
		m.log.Print("Caught signal:", s)
		if s == syscall.SIGINT {
			m.clearPins()
			m.log.Print("Exiting")
			os.Exit(0)
		}
		if s == syscall.SIGQUIT { // TODO: I don't think this works
			pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
		}
	}
}

// Setup/reset modem.  Also ATZ, conveniently.  Leaves RTS & CTS down.
func (m *Modem) reset() (int) {
	m.log.Print("Resetting modem")

	m.onHook()
	m.lowerDSR()
	m.lowerCTS()
	m.lowerRI()

	m.echo = true		// Echo local keypresses
	m.quiet = false		// Modem offers return status
	m.verbose = true	// Text return codes
	m.volume = 1		// moderate volume
	m.speakermode = 1	// on until other modem heard
	m.lastcmds = nil
	m.lastdialed = ""
	m.connect_speed = 0
	m.setLineBusy(false)
	m.setupRegs()
	m.setupDebug()

	m.loadAddressBook()
	return OK
}

// Boot the modem
func (m *Modem) PowerOn() {

	var logger io.Writer
	var err error

	// TODO: should this be here or in the main
	logger = os.Stdout
	if *_flags_logfile != "" {
		logger, err = os.OpenFile(*_flags_logfile,
			os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			panic("Can't open logfile")
		}
	}
	m.log = log.New(logger, "modem: ",
		log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	// TODO: end
	
	m.log.Print("------------ Starting up")
	m.setupPins()	      
	m.reset()	      // Setup modem inital state (or reset initial state)
	m.serial = setupSerialPort(true)
	
	go m.signalHandler()	// Catch signals in a different thread
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
