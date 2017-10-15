package main

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
	"log/syslog"
	"os"
	"os/signal"
	"path"
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

}

var m Modem
var registers *Registers
var phonebook *Phonebook
var netConn connection
var serial *serialPort

var timer *time.Ticker
var charchannel chan byte
var logger *log.Logger

func setupLogging() *log.Logger {
	var err error

	flags := log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile
	
	if *_flags_syslog {
		logger, err := syslog.NewLogger(syslog.LOG_CRIT, flags)
		if err != nil {
			panic("Can't open syslog")
		}
		return logger
	}

	logger := os.Stderr	// default to StdErr
	if *_flags_logfile != "" {
		logger, err = os.OpenFile(*_flags_logfile,
			os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			panic("Can't open logfile")
		}
	}
	prefix := path.Base(os.Args[0]) + ": "
	return log.New(logger, prefix, flags)

}

// Watch a subset of pins and act as apropriate
// Must be a goroutine
func handlePINs() {

	for {
		// TODO: Do I need to support DTR state changes (&Q & &D)?
		if readDTR() {
			led_TR_on()
		} else {
			if offHook() {
				goOnHook()
			}
			led_TR_off()
		}

		if m.connectSpeed > 19200 {
			led_HS_on()
		} else {
			led_HS_off()
		}

		if m.dcd || m.dcdControl {
			raiseCD()
		} else {
			lowerCD()
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// Catch ^C, reset the HW pins
// Must be a goroutine
func handleSignals() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGUSR1, syscall.SIGUSR2,
		syscall.SIGQUIT)

	for {
		// Block until a signal is received.
		s := <-c
		logger.Print("Caught signal: ", s)
		switch s {
		case syscall.SIGINT:
			clearPins()
			logger.Print("Exiting")
			os.Exit(0)

		case syscall.SIGUSR1:
			pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)

		case syscall.SIGUSR2:
			logState()

		case syscall.SIGQUIT:
			logState()
		}
	}
}

// Timer functions
func resetTimer() {
	stopTimer()
	gt := registers.Read(REG_ESC_CODE_GUARD_TIME)
	guardTime := time.Duration(float64(gt) * 0.02) * time.Second

	logger.Printf("Setting timer for %v", guardTime)
	timer = time.NewTicker(guardTime)
}

func stopTimer() {
	if timer != nil {
		timer.Stop()
	}
}

// Consume bytes from the serial port and process or send to
// remote as per m.mode
func readSerial() {
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
		regs = registers // Reload regs in case we reset the modem

		select {
		case <- timer.C:
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
				logger.Print("Escape sequence detected, ", 
					"entering command mode")
				m.mode = COMMANDMODE
				prstatus(OK) // signal that we're in command mode
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
			if m.echoInCmdMode { // Echo back to the DTE
				serial.WriteByte(c)
			}

			// Accumulate chars in s until we read a CR, then process
			// s as a command.

			// 'A/' command, immediately exec.
			if (s == "A" || s == "a") && c == '/' {
				serial.Println()
				if m.lastCmd == "" {
					prstatus(ERROR)
				} else {
					command(m.lastCmd)
				}
				s = ""
			} else if c == regs.Read(REG_CR_CH) && s != "" {
				command(s)
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
			if c != registers.Read(REG_ESC_CH) {
				lastThree = [3]byte{' ', ' ', ' '}
				idx = 0
			} else {
				lastThree[idx] = c
				idx = (idx + 1) % 3
			}
			
			// Send to remote, blinking the SD LED
			if offHook() && netConn != nil {
				led_SD_on()
				out := make([]byte, 1)
				out[0] = c
				netConn.Write(out)
				led_SD_off()	
			}
		}
	}
}

// Boot the modem
func main() {
	flag.Parse()
	
	logger = setupLogging()
	logger.Print("------------ Starting up")
	logger.Printf("Cmdline: %s", strings.Join(os.Args, " "))

	registers = NewRegisters()
	callChannel = make(chan connection)
	charchannel = make(chan byte)
	serial = setupSerialPort(*_flags_serialPort, *_flags_serialSpeed,
		charchannel, logger)
	serial.Chars(registers.Read(REG_BS_CH), registers.Read(REG_CR_CH),
		registers.Read(REG_LF_CH))

	setupPins()

	reset()	      // Setup modem inital state (or reset initial state)
	
	go handleSignals()	// Catch signals in a different thread
	go handlePINs()         // Monitor input pins & internal registers
	go handleModem()	// Handle in-bound bytes in a seperate goroutine

	// Signal to DTE that we're ready
	time.Sleep(250 * time.Millisecond) // make it look good
	raiseDSR()
	raiseCTS()

	// Tell user we're ready
	logger.Print("Modem Ready")
	prstatus(OK)

	readSerial()		// never returns
}


