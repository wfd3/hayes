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
	"strings"
	"sync"
	"syscall"
	"time"
)

// What mode is the modem in?
const (
	COMMANDMODE bool = false
	DATAMODE    bool = true
)

// How many rings before giving up
const __MAX_RINGS = 15

// How long to wait for the remote to answer.  6 seconds is the default
// ring-silence time
const __CONNECT_TIMEOUT = __MAX_RINGS * 6 * time.Second

// Basic modem state.  This is ephemeral.
// TODO - I think the mutexes around lineBusy and hook are not required.
type Modem struct {
	currentConfig int             // Which stored config are we using
	mode          bool             // DATA or COMMAND mode
	lastCmd       string          // Last command (for A/ command)
	lastDialed    string          // Last number dialed (for ATDL)
	connectSpeed  int             // What speed did we connect at (0 or 38k)
	dcd           bool            // Data Carrier Detect -- active connection?
	lineBusy      bool            // Is the "phone line" busy --
				      // accepting or dialing?
	hook          bool            // Is the phone on or off hook?
	lineBusyLock  sync.RWMutex
	hookLock      sync.RWMutex
	conn          connection
}

var m Modem
var conf Config
var registers Registers
var phonebook *Phonebook
var profiles *storedProfiles
var serial *serialPort
var timer *time.Ticker
var charchannel chan byte
var callChannel chan connection

// Catch ^C, reset the HW pins
// Must be a goroutine
func handleSignals() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGUSR1, syscall.SIGQUIT)

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
	// REG_ESC_CODE_GUARD_TIME is in 50th of a second (20ms)
	guardTime := time.Duration(float64(gt) * 20) * time.Millisecond
		
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
func handleSerial() {
	var c byte
	var s string
	var lastThree [3]byte
	var idx int
	var countAtTick uint64
	var countAtLastTick uint64
	var waitForOneTick bool
	var status error
	
	// Tell user & DTE we're ready
	raiseCTS()
	raiseDSR()
	logger.Print("Modem Ready")
	prstatus(OK)

	// Start accepting and processing bytes from the DTE
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
				prstatus(OK)
				s = ""
				continue
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
					status = ERROR
				} else {
					status = runCommand(m.lastCmd)
				}
				prstatus(status)
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
			if offHook() && m.conn != nil {
				led_SD_on()
				out := make([]byte, 1)
				out[0] = c
				m.conn.Write(out)
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

	// Setup the GPIO and serial port hardware
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

	setupHW()		// Setup the "hardware"
	go handleSignals()	// Catch signals in a different thread
	go handleCalls()        // Handle in-bound bytes in a seperate goroutine
	handleSerial()	        // never returns
}
