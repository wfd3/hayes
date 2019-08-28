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
	lcdm "github.com/wfd3/lcd"
)

var m Modem
var conf *Config
var registers *Registers
var phonebook *Phonebook
var profiles *storedProfiles
var serial *serialPort
var callChannel chan connection
var lcd *lcdm.Lcd

// Catch ^C, reset the HW pins
// Must be a goroutine
func handleSignals() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGQUIT)

	for {
		// Block until a signal is received.
		s := <-c
		logger.Printf("Caught signal: %s", s)
		switch s {
		case syscall.SIGINT:
			clearPins()
			shutdownLCD()
			logger.Print("Exiting")
			os.Exit(0)

		case syscall.SIGQUIT:
			logState()
		}
	}
}

func setupLCD() {
	lcd = lcdm.NewLcd(2, 16)
	if flags.lcd {
		err := lcd.EnableHW()
		if err != nil {
			logger.Fatal(err)
		}
		lcd.On()
		lcd.BacklightOn()
		lcd.Clear()
		lcd.SetPosition(1,1)
		lcd.Centerf(1, "RetroHayes 1.0")
	}
}

func shutdownLCD() {
	if flags.lcd {
		lcd.Clear()
		lcd.BacklightOff()
		lcd.Off()
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
	setupLCD()
	
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
