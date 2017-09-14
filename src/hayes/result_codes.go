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
var CONNECT_38400 error = NewMerror(28, "CONNECT 38400")

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
	
	if e != nil { // nil is "OK", so that's OK.  
		merr, ok = e.(*MError)
		if !ok {
			m.log.Print("Called prstatus with an error not of MError: ",
				e)
			m.prstatus(ERROR)
			return
		}
	}
	if m.quiet {
		return
	}

	// TODO: Add support for ATXn return codes.
	if m.verbose {
		m.serial.Write([]byte(merr.Error()))
	} else {
		m.serial.Write([]byte(merr.Code()))
	} 
}
