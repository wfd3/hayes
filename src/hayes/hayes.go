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
	"syscall"
	"time"
)

// What mode is the modem in?
const (
	COMMANDMODE bool = false
	DATAMODE    bool = true
)

// Basic modem state.  This is ephemeral.
type Modem struct {
	currentConfig int             // Which stored config are we using
	mode          bool            // DATA or COMMAND mode
	lastCmd       string          // Last command (for A/ command)
	lastDialed    string          // Last number dialed (for ATDL)
	connectSpeed  int             // What speed did we connect at (0 or 38k)
	dcd           bool            // Data Carrier Detect -- active connection?
	lineBusy      bool            // Is the "phone line" busy?
	hook          bool            // Is the phone on or off hook?
	conn          connection      // Current active connection
}

var m Modem
var conf Config
var registers Registers
var phonebook *Phonebook
var profiles *storedProfiles
var serial *serialPort
var timer *time.Ticker
var callChannel chan connection
var last_ring_time time.Time

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
	// REG_ESC_CODE_GUARD_TIME is in 50th's of a second (20ms)
	gt := registers.Read(REG_ESC_CODE_GUARD_TIME)
	guardTime := time.Duration(float64(gt) * 20) * time.Millisecond
		
	logger.Printf("Setting timer for %v", guardTime)
	timer = time.NewTicker(guardTime)
}

func stopTimer() {
	if timer != nil {
		timer.Stop()
	}
}

// ATZn - 0 == config 0, 1 == config 1
func softReset(i int) error {
	logger.Printf("Switching config/registers")
	if err := profiles.Switch(i); err != nil {
		return err
	}
	return nil
}

// AT&F - reset to factory defaults
func factoryReset() error {
	err := OK
	logger.Print("Resetting modem")

	// Reset state
	hangup()
	setLineBusy(false)
	lowerDSR()
	lowerCTS()
	lowerRI()
	stopTimer()
	m = Modem{}

	registers.Reset()
	conf.Reset()
	profiles, _ = newStoredProfiles()
	profiles.Switch(profiles.PowerUpConfig)

	phonebook = NewPhonebook(flags.phoneBook, logger)
	err = phonebook.Load()
	if err != nil {
		logger.Print(err)
	}

	resetTimer()

	raiseCTS()
	raiseDSR()
	return err
}

// Boot the modem
func main() {
	initFlags()

	logger = setupLogging()
	logger.Print("------------ Starting up")
	logger.Printf("Cmdline: %s", strings.Join(os.Args, " "))
	
	// Setup the GPIO and serial port hardware
	setupPins()
	serial = setupSerialPort(flags.serialPort, flags.serialSpeed)

	go handleSignals()	// Catch signals in a different thread

	// Setup modem inital state
	registers = NewRegisters()
	factoryReset()

	// Setup the "hardware"
	setupHW()

	// Setup the comms channels and handle inbound/outbound comms
	callChannel = make(chan connection)
	go handleCalls()    

	// Tell user & DTE we're ready
	raiseDSR()
	raiseCTS()
	logger.Print("Modem Ready")
	prstatus(OK)

	handleSerial()	        // never returns
}
