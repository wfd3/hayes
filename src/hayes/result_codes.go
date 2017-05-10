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
	NO_CARRIER:    "NO_CARRIER",
	ERROR:         "ERROR",
	CONNECT_1200:  "CONNECT_1200",
	NO_DIALTONE:   "NO_DIALTONE",
	BUSY:          "BUSY",
	NO_ANSWER:     "NO_ANSWER",
	CONNECT_2400:  "CONNECT_2400",
	CONNECT_4800:  "CONNECT_4800",
	CONNECT_9600:  "CONNECT_9600",
	CONNECT_14400: "CONNECT_14400",
	CONNECT_19200: "CONNECT_19200",
	CONNECT_38400: "CONNECT_38400",
}

// Print command status, subject to quiet mode and verbose mode flags
func (m *Modem) prstatus(status int) {
	if m.quiet {
		return
	}
	if m.verbose {
		fmt.Println(status_codes[status])
	} else {
		fmt.Println(status)
	} 
}
