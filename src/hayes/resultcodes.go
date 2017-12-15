package main

// Command Result codes

import (
	"fmt"
	"strings"
)

type MError struct {
	code byte
	text string
}

var (
	OK             error = nil
	CONNECT        error = NewMerror(1, "CONNECT")
	RING           error = NewMerror(2, "RING\n") // RING has CR/LF after it.
	NO_CARRIER     error = NewMerror(3, "NO CARRIER")
	ERROR          error = NewMerror(4, "ERROR")
	CONNECT_1200   error = NewMerror(5, "CONNECT 1200")
	NO_DIALTONE    error = NewMerror(6, "NO DIALTONE")
	BUSY           error = NewMerror(7, "BUSY")
	NO_ANSWER      error = NewMerror(8, "NO ANSWER")
	CONNECT_2400   error = NewMerror(10, "CONNECT 2400")
	CONNECT_4800   error = NewMerror(11, "CONNECT 4800")
	CONNECT_9600   error = NewMerror(12, "CONNECT 9600")
	CONNECT_14400  error = NewMerror(13, "CONNECT 14400")
	CONNECT_19200  error = NewMerror(14, "CONNECT 19200")
	CONNECT_57600  error = NewMerror(18, "CONNECT 57600")
	CONNECT_7200   error = NewMerror(24, "CONNECT 7200")
	CONNECT_12000  error = NewMerror(25, "CONNECT 12000")
	CONNECT_38400  error = NewMerror(28, "CONNECT 38400")
	CONNECT_300    error = NewMerror(40, "CONNECT 300")
	CONNECT_115200 error = NewMerror(87, "CONNECT 115200")
)

func NewMerror(c byte, s string) error {
	return &MError{c, s}
}

func speedToResult(speed int) error {
	switch speed {
	case 300:
		return CONNECT_300
	case 1200:
		return CONNECT_1200
	case 2400:
		return CONNECT_2400
	case 4800:
		return CONNECT_4800
	case 7200:
		return CONNECT_7200
	case 9600:
		return CONNECT_9600
	case 12000:
		return CONNECT_12000
	case 14400:
		return CONNECT_14400
	case 19200:
		return CONNECT_19200
	case 38400:
		return CONNECT_38400
	case 57600:
		return CONNECT_57600
	case 115200:
		return CONNECT_115200
	default:
		return CONNECT
	}
}

func (e *MError) Error() string {

	if conf.quiet {
		logger.Printf("Quiet mode, status: %s", e)
		return ""
	}

	if e == CONNECT && conf.connectMsgSpeed {
		me := speedToResult(m.connectSpeed)
		return me.Error()
	}

	if e == BUSY && !conf.busyDetect {
		e = nil
	}

	if (e == NO_DIALTONE || e == NO_ANSWER) && !conf.extendedResultCodes {
		e = nil
	}

	var s string
	switch conf.verbose {
	case true:
		if e != nil {
			s = fmt.Sprintf("%s", e.text)
		} else {
			s = "OK"
		}
	case false:
		if e != nil {
			s = fmt.Sprintf("%d", e.code)
		} else {
			s = "0"
		}
	}
	
	logentry := fmt.Sprintf("Result Code: %s", s)
        logger.Print(strings.Replace(logentry, "\n", "", -1))

	return s
}

// This is needed because nil errors are "OK", but Prinln(OK) can't work.
// I'm starting to think overloading error as result codes is a massive mistake.
func prstatus(e error) {
	if e == nil {
		switch conf.verbose {
		case true:  serial.Println("OK")
		case false: serial.Println("0")
		}
	} else {
		
		// If the underlying type isn't MError, log it and print a
		// generic ERROR
		if _, ok := e.(*MError); !ok {
			logger.Printf("Error not MError: %s", e.Error())
			e = ERROR
		}
		serial.Println(e)
	}
}	
