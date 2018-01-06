package main

import (
	"fmt"
)

// Configuration
type Config struct {
	echoInCmdMode       bool
	speakerMode         int
	speakerVolume       int
	verbose             bool
	quiet               bool
	connectMsgSpeed     bool
	busyDetect          bool
	extendedResultCodes bool
	dcdPinned           bool
	dsrPinned           bool
	dtr                 int
}

func (c *Config) Reset() {
	c.echoInCmdMode = true // Echo local keypresses
	c.quiet = false        // Modem offers return status
	c.verbose = true       // Text return codes
	c.speakerVolume = 2    // moderate volume
	c.speakerMode = 1      // on until other modem heard
	c.busyDetect = true
	c.extendedResultCodes = true
	c.dcdPinned = true	// if true, DCD if fixed 'on'
	c.connectMsgSpeed = true
	c.dsrPinned = true	// if true, DSR is fixed 'on'
	c.dtr = 0
}

func (c *Config) String() string {
	b := func(p bool) string {
		if p {
			return "1 "
		}
		return "0 "
	}
	i := func(p int) string {
		return fmt.Sprintf("%d ", p)
	}
	x := func(r, b bool) string {
		if r == false && b == false {
			return "0 "
		}
		if r == true && b == false {
			return "1 "
		}
		if r == true && b == true {
			return "7 "
		}
		return "0 "
	}

	str := "B16 B1 B41 B60 "
	str += "E" + b(c.echoInCmdMode)
	str += "F1 " // For Hayes 1200 compatability
	str += "L" + i(c.speakerVolume)
	str += "M" + i(c.speakerMode)
	str += "N1 "
	str += "Q" + b(c.quiet)
	str += "V" + b(c.verbose)
	str += "W" + b(c.connectMsgSpeed)
	str += "X" + x(c.extendedResultCodes, c.busyDetect)
	str += "Y0 "
	str += "&A0 "
	str += "&C" + b(c.dcdPinned)
	str += "&D" + i(c.dtr)
	str += "&G0 "
	str += "&J0 "
	str += "&K3 "
	str += "&Q5 "
	str += "&R0 "
	str += "&S" + b(c.dsrPinned)
	str += "&T4 "
	str += "&U0 "
	str += "&X4 "

	return lineWrap(str, 80)
}
