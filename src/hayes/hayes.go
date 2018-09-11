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

var m Modem
var conf *Config
var registers *Registers
var phonebook *Phonebook
var profiles *storedProfiles
var serial *serialPort
var callChannel chan connection

// Catch ^C, reset the HW pins
// Must be a goroutine
func handleSignals() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGQUIT)

	for {
		// Block until a signal is received.
		s := <-c
		logger.Print("Caught signal: ", s)
		switch s {
		case syscall.SIGINT:
			clearPins()
			logger.Print("Exiting")
			os.Exit(0)

		case syscall.SIGQUIT:
			logState()
		}
	}
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
	conf = &Config{}
	registers = NewRegisters()
	factoryReset()

	// Setup the "hardware"
	setupHW()

	// Setup the comms channels and handle inbound/outbound comms
	callChannel = make(chan connection)
	go handleCalls()

	time.Sleep(500 * time.Millisecond)

	// Tell user & DTE we're ready
	raiseDSR()
	raiseCTS()
	logger.Print("Modem Ready")
	prstatus(OK)

	handleSerial()	        // never returns
}
