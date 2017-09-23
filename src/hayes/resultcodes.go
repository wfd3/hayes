package hayes

// Command Result codes

import (
	"fmt"
)

type MError struct {
	code byte
	text string
}

func NewMerror(c byte, s string) error {
	return &MError{c, s}
}

var OK error            = nil
var CONNECT error       = NewMerror(1,  "CONNECT")
var RING error          = NewMerror(2,  "RING")
var NO_CARRIER error    = NewMerror(3,  "NO CARRIER")
var ERROR error         = NewMerror(4,  "ERROR")
var CONNECT_1200 error  = NewMerror(5,  "CONNECT 1200")
var NO_DIALTONE error   = NewMerror(6,  "NO DIALTONE")
var BUSY error          = NewMerror(7,  "BUSY")
var NO_ANSWER error     = NewMerror(8,  "NO ANSWER")
var CONNECT_2400 error  = NewMerror(10, "CONNECT 2400")
var CONNECT_4800 error  = NewMerror(11, "CONNECT 4800")
var CONNECT_9600 error  = NewMerror(12, "CONNECT 9600")
var CONNECT_14400 error = NewMerror(13, "CONNECT 14400")
var CONNECT_19200 error = NewMerror(14, "CONNECT 19200")
var CONNECT_57600 error = NewMerror(18, "CONNECT 57600")
var CONNECT_7200 error  = NewMerror(24, "CONNECT 7200")
var CONNECT_12000 error = NewMerror(25, "CONNECT 12000")
var CONNECT_38400 error = NewMerror(28, "CONNECT 38400")
var CONNECT_300 error   = NewMerror(40, "CONNECT 300")
var CONNECT_115200 error= NewMerror(87, "CONNECT 115200")

func speedToResult(speed int) error {
	switch speed {
	case 300:    return CONNECT_300
	case 1200:   return CONNECT_1200
	case 2400:   return CONNECT_2400
	case 4800:   return CONNECT_4800
	case 7200:   return CONNECT_7200
	case 9600:   return CONNECT_9600
	case 12000:  return CONNECT_12000
	case 14400:  return CONNECT_14400
	case 19200:  return CONNECT_19200
	case 38400:  return CONNECT_38400
	case 57600:  return CONNECT_57600
	case 115200: return CONNECT_115200
	default:     return CONNECT
	}
}

func (e *MError) Error() string {
	if e == nil {
		return "OK\n"
	}
	return fmt.Sprintf("%s\n",e.text)
}

func (e *MError) Code() string {
	if e == nil {
		return "0\n"
	}
	return fmt.Sprintf("%d\n", e.code)
}

// Print command status, subject to quiet mode and verbose mode flags
func (m *Modem) prstatus(e error) {
	var ok bool
	var merr *MError
	
	if m.quiet {
		return
	}

	if e == CONNECT && m.connectMsgSpeed {
		e = speedToResult(m.connect_speed)
	}

	if e == BUSY && !m.busyDetect {
		e = OK
	}

	if (e == NO_DIALTONE || e == NO_ANSWER) && !m.extendedResultCodes {
		e = OK
	}
	
	if e != nil { // nil is "OK", so that's OK.  
		merr, ok = e.(*MError)
		if !ok {
			m.log.Print("Underlying error: ", e)
			m.prstatus(ERROR)
			return
		}
	}
	
	if !m.verbose {
		m.serial.Write([]byte(merr.Code()))
	} else {
		m.serial.Write([]byte(merr.Error()))
	} 
}
