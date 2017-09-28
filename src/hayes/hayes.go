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
	"sync"
	"runtime/pprof"
	"syscall"
)

// What mode is the modem in?
const (
	COMMANDMODE = iota
	DATAMODE
)

var callChannel chan connection

//Basic modem struct
type Modem struct {
	mode int
	echoInCmdMode bool
	speakermode int
	volume int
	verbose bool
	quiet bool
	connectMsgSpeed bool
	busyDetect bool
	extendedResultCodes bool

	dcdControl bool
	dcd bool

	lastcmd string
	lastdialed string
	connect_speed int

	linebusy bool
	linebusylock sync.RWMutex
	hook int
	hooklock sync.RWMutex

	conn connection
	serial *serialPort
	pins Pins
	leds Pins
	phonebook *Phonebook
	registers *Registers
	log *log.Logger
	timer *time.Ticker
}

// Watch a subset of pins and act as apropriate
// Must be a goroutine
func (m *Modem) handlePINs() {
	for {
		if m.readDTR() {
			m.led_TR_on()
		} else { 
			m.goOnHook()
			m.led_TR_off()
		}

		if m.connect_speed > 19200 {
			m.led_HS_on()
		} else {
			m.led_HS_off()
		}

		if m.dcd || m.dcdControl {
			m.raiseCD()
		} else {
			m.lowerCD()
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

	callChannel = make(chan connection, 1)

	m.reset()	      // Setup modem inital state (or reset initial state)
	m.serial = setupSerialPort(*_flags_serialPort, m.registers, m.log)
	
	go m.handleSignals()	// Catch signals in a different thread
	go m.handlePINs()       // Monitor input pins & internal registers
	go m.handleModem()	// Handle in-bound bytes in a seperate goroutine

	// Signal to DTE that we're ready
	time.Sleep(250 * time.Millisecond) // make it look good
	m.raiseDSR()
	m.raiseCTS()

	// Tell user we're ready
	m.log.Print("Modem Ready")
	m.prstatus(OK)

	m.readSerial()		// never returns
}

// Consume bytes from the serial port and process or send to
// remote as per m.mode
func (m *Modem) readSerial() {
	var c byte
	var s string
	var lastThree [3]byte
	var idx int
	var regs *Registers
	var countAtTick uint64
	var countAtLastTick uint64
	var waitForOneTick bool

	charchannel := make(chan byte, 1)
	go m.serial.getChars(charchannel)

	countAtTick = 0
	for {
		select {
		case <- m.timer.C:
			if m.mode == COMMANDMODE { // Skip this if in COMMAND mode
				continue
			}

			// Look for the command escape sequence
			// (see http://www.messagestick.net/modem/Hayes_Ch1-4.html)
			// Basically:
			// 1s of silence, "+++", 1s of silence.
			// So, count the incoming chars between ticks, saving
			// the previous tick's count.  If you see
			// countAtTick == 3 && CountAtLastTick == 0 && the last
			// three characters are "+++", wait one more tick.  If
			// countAtTick == 0, the guard sequence was detected.
			
			if countAtTick == 3 && countAtLastTick == 0 &&
				lastThree == escSequence { 
				waitForOneTick = true
			} else if waitForOneTick && countAtTick == 0 {
				m.mode = COMMANDMODE
				m.prstatus(OK) // signal that we're in command mode
			} else {
				waitForOneTick = false
			}
			countAtLastTick = countAtTick
			countAtTick = 0
			continue

		case c = <- charchannel:
			countAtTick++

		}

		switch m.mode {
		case COMMANDMODE:
			regs = m.registers // Reload regs in case we reset the modem
			if m.echoInCmdMode { // Echo back to the DTE
				m.serial.WriteByte(c)
			}

			// Accumulate chars in s until we read a CR, then process
			// s as a command.

			// 'A/' command, immediately exec.
			if (s == "A" || s == "a") && c == '/' {
				m.serial.Println()
				if m.lastcmd == "" {
					m.prstatus(ERROR)
				} else {
					m.command(m.lastcmd)
				}
				s = ""
			} else if c == regs.Read(REG_LF_CH) && s != "" {
				m.command(s)
				s = ""
			} else if c == regs.Read(REG_BS_CH)  && len(s) > 0 {
				s = s[0:len(s) - 1]
			} else if c == regs.Read(REG_LF_CH)  ||
				c == regs.Read(REG_BS_CH) && len(s) == 0 {
				// ignore naked CR's & BS if s is already empty
			} else {
				s += string(c)
			}

		case DATAMODE:
			// Look for the command escape sequence
			if c != m.registers.Read(REG_ESC_CH) {
				lastThree = [3]byte{' ', ' ', ' '}
				idx = 0
			} else {
				lastThree[idx] = c
				idx = (idx + 1) % 3
			}
			
			// Send to remote
			// TODO: make sure the LED says on long enough
			if m.offHook() && m.conn != nil {
				m.led_SD_on()
				out := make([]byte, 1)
				out[0] = c
				m.conn.Write(out)
				m.led_SD_off()	
			}
		}
	}
}




