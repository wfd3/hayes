package hayes

// Command Result codes

import (
	"fmt"
)

const (
	OK            = 0
	CONNECT       = 1
	RING          = 2
	NO_CARRIER    = 3
	ERROR         = 4
	CONNECT_1200  = 5
	NO_DIALTONE   = 6
	BUSY          = 7
	NO_ANSWER     = 8
	CONNECT_2400  = 10
	CONNECT_4800  = 11
	CONNECT_9600  = 12
	CONNECT_14400 = 13
	CONNECT_19200 = 14
	CONNECT_38400 = 28
)

var status_codes = map[int]string{
	OK:            "OK",  	
	CONNECT:       "CONNECT",
	RING:          "RING",
	NO_CARRIER:    "NO CARRIER",
	ERROR:         "ERROR",
	CONNECT_1200:  "CONNECT 1200",
	NO_DIALTONE:   "NO DIALTONE",
	BUSY:          "BUSY",
	NO_ANSWER:     "NO ANSWER",
	CONNECT_2400:  "CONNECT 2400",
	CONNECT_4800:  "CONNECT 4800",
	CONNECT_9600:  "CONNECT 9600",
	CONNECT_14400: "CONNECT 14400",
	CONNECT_19200: "CONNECT 19200",
	CONNECT_38400: "CONNECT 38400",
}

// Print command status, subject to quiet mode and verbose mode flags
func (m *Modem) prstatus(code int) {
	if m.quiet {
		return
	}
	if code == CONNECT {
		switch  m.connect_speed {
		case 2400: code = CONNECT_2400
		case 4800: code = CONNECT_4800
		case 9600: code = CONNECT_9600
		case 14400: code = CONNECT_14400
		case 19200: code = CONNECT_19200
		case 38400: code = CONNECT_38400
		}
	}
			
	if m.verbose {
		fmt.Println(status_codes[code])
	} else {
		fmt.Println(code)
	} 
}
