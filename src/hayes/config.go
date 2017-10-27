package main

import (
	"fmt"
)

// Configuration
type Config struct {            
	echoInCmdMode bool       
	speakerMode int          
	speakerVolume int        
	verbose bool             
	quiet bool               
	connectMsgSpeed bool     
	busyDetect bool          
	extendedResultCodes bool 
	dcdControl bool          
}

func (c *Config) Reset() {
	c.echoInCmdMode = true  // Echo local keypresses
	c.quiet = false		// Modem offers return status
	c.verbose = true	// Text return codes
	c.speakerVolume = 2	// moderate volume
	c.speakerMode = 1	// on until other modem heard
	c.busyDetect = true
	c.extendedResultCodes = true
	c.dcdControl = false	
	c.connectMsgSpeed = true
}

func (c *Config) String() string {
	b := func(p bool) (string) {
		if p {
			return"1 "
		} 
		return "0 "
	};
	i := func(p int) (string) {
		return fmt.Sprintf("%d ", p)
	};
	x := func(r, b bool) (string) {
		if (r == false && b == false) {
			return "0 "
		}
		if (r == true && b == false) {
			return "1 "
		}
		if (r == true && b == true) {
			return "7 "
		}
		return "0 "
	};

	s := "E" + b(c.echoInCmdMode)
	s += "F1 "		// For Hayes 1200 compatability 
	s += "L" + i(c.speakerVolume)
	s += "M" + i(c.speakerMode)
	s += "Q" + b(c.quiet)
	s += "V" + b(c.verbose)
	s += "W" + b(c.connectMsgSpeed)
	s += "X" + x(c.extendedResultCodes, c.busyDetect)
	s += "&C" + b(c.dcdControl)

	return s
}
