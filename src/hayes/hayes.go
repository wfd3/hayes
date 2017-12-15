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

// How many rings before giving up
const __MAX_RINGS = 15

// How long to wait for the remote to ansewer
const __CONNECT_TIMEOUT = __MAX_RINGS * 6 * time.Second

//Basic modem state
type Modem struct {
	currentConfig int
	mode          int
	lastCmd       string
	lastDialed    string
	connectSpeed  int
	dcd           bool
	lineBusy      bool
	lineBusyLock  sync.RWMutex
	hook          int
	hookLock      sync.RWMutex
}

var m Modem
var conf Config
var registers Registers
var phonebook *Phonebook
var profiles *storedProfiles
var netConn connection
var serial *serialPort

var timer *time.Ticker
var charchannel chan byte

var callChannel chan connection

// Watch a subset of pins and act as apropriate
// Must be a goroutine
func handlePINs() {

	for {
		// TODO: Do I need to support DTR state changes (&Q & &D)?
		// http://www.messagestick.net/modem/Hayes_Ch1-1.html
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

		if m.dcd || conf.dcdControl {
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
	guardTime := time.Duration(float64(gt)*0.02) * time.Second

	logger.Printf("Setting timer for %v", guardTime)
	timer = time.NewTicker(guardTime)
}

func stopTimer() {
	if timer != nil {
		timer.Stop()
	}
}

// Consume bytes from the serial port and process or send to remote as
// per conf.mode
func readSerial() {
	var c byte
	var s string
	var lastThree [3]byte
	var idx int
	var countAtTick uint64
	var countAtLastTick uint64
	var waitForOneTick bool
	var status error

	countAtTick = 0
	for {

		select {
		case <-timer.C:
			if m.mode == COMMANDMODE { // Skip if in COMMAND mode
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

		case c = <-charchannel:
			countAtTick++
		}

		switch m.mode {
		case COMMANDMODE:
			if conf.echoInCmdMode { // Echo back to the DTE
				serial.WriteByte(c)
			}

			// Accumulate chars in s until we read a CR, then process
			// s as a command.

			// 'A/' command, immediately exec.
			if (s == "A" || s == "a") && c == '/' {
				serial.Println()
				if m.lastCmd == "" {
					serial.Println(ERROR)
				} else {
					status = runCommand(m.lastCmd)
					prstatus(status)
				}
				s = ""
			} else if c == registers.Read(REG_CR_CH) && s != "" {
				status = runCommand(s)
				prstatus(status)
				s = ""
			} else if c == registers.Read(REG_BS_CH) && len(s) > 0 {
				s = s[0 : len(s)-1]
			} else if c == registers.Read(REG_CR_CH) ||
				c == registers.Read(REG_BS_CH) && len(s) == 0 {
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
	initFlags()

	logger = setupLogging()
	logger.Print("------------ Starting up")
	logger.Printf("Cmdline: %s", strings.Join(os.Args, " "))

	// Setup the comms channels
	callChannel = make(chan connection)
	charchannel = make(chan byte)

	// Setup the hardware
	setupPins()
	serial = setupSerialPort(flags.serialPort, flags.serialSpeed,
		charchannel, logger)

	// Setup modem inital state
	registers = NewRegisters()
	conf.Reset()
	registers.Reset()

	// If there's stored profiles, load them and make the active one active.
	profiles, _ = newStoredProfiles()
	factoryReset()
	if profiles.PowerUpConfig != -1 {
		softReset(profiles.PowerUpConfig)
	}

	// Setup the helper tasks
	go handleSignals() // Catch signals in a different thread
	go handlePINs()    // Monitor input pins & internal registers
	go handleModem()   // Handle in-bound bytes in a seperate goroutine

	// Signal to DTE that we're ready
	time.Sleep(250 * time.Millisecond) // make it look good
	raiseDSR()
	raiseCTS()

	// Tell user we're ready
	logger.Print("Modem Ready")
	prstatus(OK)

	readSerial() // never returns
}
