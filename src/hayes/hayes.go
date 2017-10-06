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
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"sync"
	"syscall"
	"time"
)

// What mode is the modem in?
const (
	COMMANDMODE = iota
	DATAMODE
)

var callChannel chan connection

//Basic modem struct
type Modem struct {
	// Configuration
	echoInCmdMode bool
	speakerMode int
	speakerVolume int
	verbose bool
	quiet bool
	connectMsgSpeed bool
	busyDetect bool
	extendedResultCodes bool
	dcdControl bool
	phonebook *Phonebook
	registers *Registers

	// State
	mode int
	lastCmd string
	lastDialed string
	connectSpeed int
	dcd bool
	lineBusy bool
	lineBusyLock sync.RWMutex
	hook int
	hookLock sync.RWMutex

	// I/O
	conn connection
	serial *serialPort
	charchannel chan byte
	pins Pins
	leds Pins
	log *log.Logger
	timer *time.Ticker
}

func setupLogging() *log.Logger {
	var err error
	
	logger := os.Stderr
	if *_flags_logfile != "" {
		logger, err = os.OpenFile(*_flags_logfile,
			os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			panic("Can't open logfile")
		}
	}
	return log.New(logger, "modem: ",
		log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
}


// Watch a subset of pins and act as apropriate
// Must be a goroutine
func (m *Modem) handlePINs() {

	for {
		// TODO: Do I need to support DTR state changes (&Q & &D)?
		if m.readDTR() {
			m.led_TR_on()
		} else {
			if m.offHook() {
				m.goOnHook()
			}
			m.led_TR_off()
		}

		if m.connectSpeed > 19200 {
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
	signal.Notify(c, syscall.SIGINT, syscall.SIGUSR1, syscall.SIGUSR2,
		syscall.SIGQUIT)

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

		case syscall.SIGUSR2:
			m.logState()

		case syscall.SIGQUIT:
			m.logState()
		}
	}
}

// Timer functions
func (m *Modem) resetTimer() {
	m.stopTimer()
	gt := m.registers.Read(REG_ESC_CODE_GUARD_TIME)
	guardTime := time.Duration(float64(gt) * 0.02) * time.Second

	m.log.Printf("Setting timer for %v", guardTime)
	m.timer = time.NewTicker(guardTime)
}

func (m *Modem) stopTimer() {
	if m.timer != nil {
		m.timer.Stop()
	}
}

// Boot the modem
func (m *Modem) PowerOn() {

	flag.Parse()
	
	m.log = setupLogging()
	m.log.Print("------------ Starting up")
	m.log.Printf("Cmdline: %s", strings.Join(os.Args, " "))

	m.registers = NewRegisters()
	m.setupPins()
	callChannel = make(chan connection, 1)
	m.reset()	      // Setup modem inital state (or reset initial state)
	m.charchannel = make(chan byte, 1)
	m.serial = setupSerialPort(*_flags_serialPort, *_flags_serialSpeed,
		m.charchannel, m.registers, m.log)
	
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

	countAtTick = 0
	for {
		regs = m.registers // Reload regs in case we reset the modem

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
				m.log.Print("Escape sequence detected, ", 
					"entering command mode")
				m.mode = COMMANDMODE
				m.prstatus(OK) // signal that we're in command mode
			} else {
				waitForOneTick = false
			}
			countAtLastTick = countAtTick
			countAtTick = 0
			continue

		case c = <- m.charchannel:
			countAtTick++
		}

		switch m.mode {
		case COMMANDMODE:
			if m.echoInCmdMode { // Echo back to the DTE
				m.serial.WriteByte(c)
			}

			// Accumulate chars in s until we read a CR, then process
			// s as a command.

			// 'A/' command, immediately exec.
			if (s == "A" || s == "a") && c == '/' {
				m.serial.Println()
				if m.lastCmd == "" {
					m.prstatus(ERROR)
				} else {
					m.command(m.lastCmd)
				}
				s = ""
			} else if c == regs.Read(REG_CR_CH) && s != "" {
				m.command(s)
				s = ""
			} else if c == regs.Read(REG_BS_CH)  && len(s) > 0 {
				s = s[0:len(s) - 1]
			} else if c == regs.Read(REG_CR_CH)  ||
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
			
			// Send to remote, blinking the SD LED
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




