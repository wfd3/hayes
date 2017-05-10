package hayes

import (
	"fmt"
)

// Register constants
const (
	REG_AUTO_ANSWER    = 0
	REG_RING_COUNT     = 1
	REG_ESC_CH         = 2
	REG_CR_CH          = 4
	REG_BS_CH          = 5
	REG_ESC_CODE_GUARD = 12
	REG_DTR_DELAY      = 30
)

// TODO:  This is ugly
const ACTIVE_REGS_COUNT = 61
var ACTIVE_REGS [ACTIVE_REGS_COUNT]int = [ACTIVE_REGS_COUNT]int {0,1,2,3,4,5,6,7,8,
	9,10,11,12,18,25,26,30,33,36,38,43,44,46,48,49,50,53,55,56,57,63,69,70,71,
	72,73,74,75,76,77,78,79,80,81,82,84,85,86,87,91,92,93,94,95,97,105,108,109,
	110,115,120}

func (m *Modem) setupRegs() {
	for i := 0; i < len(m.r); i++ {
		m.r[i] = 0
	}
	m.r[REG_ESC_CH] = 43	// escape character '+'
	m.r[REG_CR_CH] = 13	// Carriage return character
	m.r[4] = 10 		// Line feed character
	m.r[REG_BS_CH] = 8	// Backspace character
	m.r[6] = 2		// Wait time before blind dialing (seconds)
	m.r[7] = 50		// Wait for carrier after dial (seconds)
	m.r[8] = 2		// Pause time for comma (dial delay) (seconds)
	m.r[9] = 6		// Carrier Detect Response time (tenths of a second)
	m.r[10] = 14		// Delay between Loss of Carrier and hangup (1/10s)
	m.r[11] = 95		// DTMF Tone Duration (milliseconds)
	m.r[REG_ESC_CODE_GUARD] = 50 // Escape code guard time (1/50 second)
	m.r[25] = 5		// Delay to DTR (seconds)
	m.r[26] = 1		// RTS to DTS delay interval (1/100 second)
	m.r[28] = 20		// Delay before force disconnect (seconds)
}

// ATS...
// Given a string that looks like a "S" command, parse & normalize it
func parseRegisters(cmd string) (string, int, error) {
	var s string
	var err error
	var reg, val int

	// NOTE: The order of these stanzas is critical.

	if  len(cmd) < 2  {
		return "", 0, fmt.Errorf("Bad command: %s", cmd)
	}

	// S? - query selected register
	if cmd[:2] == "S?" {
		s = "S?"
		return s, 2, nil
	}

	// Sn=x - write x to n
	_, err = fmt.Sscanf(cmd, "S%d=%d", &reg, &val)
	if err == nil {
		s = fmt.Sprintf("S%d=%d", reg, val)
		return s, len(s), nil
	}

	// Sn? - query register n
	_, err = fmt.Sscanf(cmd, "S%d?", &reg)
	if err == nil {
		s = fmt.Sprintf("S%d?", reg)
		return s, len(s), nil
	}

	// Sn - slect register
	_, err = fmt.Sscanf(cmd, "S%d", &reg)
	if err == nil {
		s = fmt.Sprintf("S%d", reg)
		return s, len(s), nil
	}

	return "", 0, fmt.Errorf("Bad S command: %s", cmd)
}

// Given a parsed register command, execute it.
func (m *Modem) registers(cmd string) (int) {
	var err error
	var reg, val int

	// NOTE: The order of these stanzas is critical.

	// S? - query selected register
	if cmd[:2] == "S?" {
		fmt.Printf("%d\n", m.r[m.curreg])
		return OK
	}

	// Sn=x - write x to n
	_, err = fmt.Sscanf(cmd, "S%d=%d", &reg, &val)
	if err == nil {
		if reg > 255 || reg < 0 {
			debugf("Register index over/underflow: %d", reg)
			return ERROR
		}
		if val > 255 || val < 0 {
			debugf("Register value over/underflow: %d", val)
			return ERROR
		}
		
		m.r[byte(reg)] = byte(val)
		return OK
	}

	// Sn? - query register n
	_, err = fmt.Sscanf(cmd, "S%d?", &reg)
	if err == nil {
		if reg > 255 || reg < 0 {	
			debugf("Register index over/underflow: %d", reg)
			return ERROR
		}
		
		fmt.Printf("%d\n", m.r[byte(reg)])
		return OK
	}

	// Sn - slect register
	_, err = fmt.Sscanf(cmd, "S%d", &reg)
	if err == nil {
		if reg > 255 || reg < 0 {	
			debugf("Register index over/underflow: %d", reg)
			return ERROR
		}
		m.curreg = byte(reg)
		return OK
	}

	if err != nil {
		debugf("registers(): err = %s", err)
	}
	return ERROR
}
