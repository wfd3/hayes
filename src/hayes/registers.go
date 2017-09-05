package hayes

import (
	"fmt"
)

// Register constants
// TODO: Fill this out as per the manual
const (
	// Do auto answer (0 == false, 1 == true) - default 0
	REG_AUTO_ANSWER    = iota // 0
	// AA Ring counter (read only)
	REG_RING_COUNT		  // 1
	// Escape sequence character ASCII code.  Default '+'
	REG_ESC_CH		  // 2
	// ASCII value of the line terminating character.  Defualt ASCII 13 (<CR>)
	REG_CR_CH		  // 3
	// ASCII value of the line feed character.  Default ASCII 10 (<LF>)
	REG_LF_CH		  // 4
	// ASCII value of the backspace character.  Default is ASCII 8 (<BS>)
	REG_BS_CH		  // 5
	// determines how long the modem waits after going off-hook before it dials
	REG_BLIND_DIAL_WAIT	  // 6
	// time delay between dialing and responding to an incoming carrier signal
	REG_WAIT_FOR_CARRIER_AFTER_DIAL // 7
	// duration of the delay generated by the comma (,) dial modifier
	REG_COMMA_DELAY				  // 8 seconds a
	// carrier signal must be present for the modem to recognize
	// it and issue a carrier detect
	REG_CARRIER_DETECT_RESPONSE_TIME	  // 9
	// time between loss of remote carrier and local modem disconnect (0.1s)
	REG_DELAY_BETWEEN_LOST_CARRIER_AND_HANGUP // 10
	// uration and spacing of tones in multi-frequency tone dialing
	REG_MULTIFREQ_TONE_DURATION		  // 11 delay required
	// prior to and following the escape sequence.  In 1/50's of a
	// second.  Factory default is 50 (1 second)
	REG_ESC_CODE_GUARD_TIME			  // 12
	REG_UNUSED_13
	REG_UNUSED_14
	REG_UNUSED_15
	REG_UNUSED_16	
	REG_UNUSED_17
	// Duration of the modem's diagnostic tests, in seconds.
	// Factory default is 0
	REG_MODEM_TEST_TIMER	// 18
	REG_UNUSED_19
	REG_UNUSED_20
	REG_UNUSED_21
	REG_UNUSED_22
	REG_UNUSED_23
	REG_UNUSED_24
	REG_DTR_DETECTION	// 25
	REG_RTS_TO_CTS_INTERVAL	// 26
	REG_UNUSED_27
	REG_UNUSED_28
	REG_UNUSED_29
	REG_DTR_DELAY		// 30
	REG_UNUSED_31
	REG_UNUSED_32
	REG_AFT_OPTIONS		// 33
	REG_UNUSED_34
	REG_UNUSED_35
	REGNEGOTIATION_FAILURE_TREATMENT // 36
	REG_DCE_LINE_SPEED		 // 37
	REG_DELAY_BEFORE_FORCED_HANGUP	 // 38 - seconds
)

func (m *Modem) setupRegs() {

	m.curreg = 0		// current register selected (from ATSn)
	m.r = make(map[byte]byte)

	m.rlock.Lock()
	defer m.rlock.Unlock()

	// Defaults
	// auto-answer? 0 == false, 1 == true
	m.r[REG_AUTO_ANSWER] = 0
	// AA Ring count
	m.r[REG_RING_COUNT] = 0
	// escape character '+'
	m.r[REG_ESC_CH] = 43	
	// Carriage return character
	m.r[REG_CR_CH] = 13	
	// Line feed character
	m.r[REG_LF_CH] = 10 	
	// Backspace character
	m.r[REG_BS_CH] = 8	
	// Wait time before blind dialing (seconds)
	m.r[REG_BLIND_DIAL_WAIT] = 2 
	// Wait for carrier after dial (seconds)
	m.r[REG_WAIT_FOR_CARRIER_AFTER_DIAL] = 50
	// Pause time for comma (dial delay) (seconds)
	m.r[REG_COMMA_DELAY] = 2
	// Carrier Detect Response time (1/10s)
	m.r[REG_CARRIER_DETECT_RESPONSE_TIME] = 6 
	// Delay between Loss of Carrier and hangup (1/10s)
	m.r[REG_DELAY_BETWEEN_LOST_CARRIER_AND_HANGUP] = 14
	// DTMF Tone Duration (milliseconds)
	m.r[REG_MULTIFREQ_TONE_DURATION] = 95
	// Escape code guard time (1/50 second)
	m.r[REG_ESC_CODE_GUARD_TIME] = 50 
	// Delay to DTR (seconds)
	m.r[REG_DTR_DETECTION] = 5		
	// RTS to DTS delay interval (1/100 second)
	m.r[REG_RTS_TO_CTS_INTERVAL] = 1 
	// Delay before force disconnect (seconds)
	m.r[REG_DELAY_BEFORE_FORCED_HANGUP] = 20
	m.r[REG_DTR_DELAY] = 0
	m.r[REG_DELAY_BEFORE_FORCED_HANGUP] = 20
}


// Note the locks here.
func (m *Modem) readReg(reg byte) byte {
	m.rlock.RLock()
	defer m.rlock.RUnlock()
	return m.r[reg]
}

func (m *Modem) writeReg(reg, val byte) {
	m.rlock.Lock()
	defer m.rlock.Unlock()
	m.r[reg] = val
}

func (m *Modem) incReg(reg byte) byte {
	m.rlock.RLock()
	m.rlock.Lock()
	defer m.rlock.Unlock()
	m.r[reg]++
	return m.r[reg]
}

// Given a parsed register command, execute it.
func (m *Modem) registers(cmd string) (int) {
	var err error
	var reg, val int

	// NOTE: The order of these stanzas is critical.

	// S? - query selected register
	if cmd[:2] == "S?" {
		fmt.Printf("%d\n", m.readReg(m.curreg))
		return OK
	}

	// Sn=x - write x to n
	_, err = fmt.Sscanf(cmd, "S%d=%d", &reg, &val)
	if err == nil {
		if reg > 255 || reg < 0 {
			m.log.Printf("Register index over/underflow: %d", reg)
			return ERROR
		}
		if val > 255 || val < 0 {
			m.log.Printf("Register value over/underflow: %d", val)
			return ERROR
		}
		m.writeReg(byte(reg), byte(val))
		if reg == REG_AUTO_ANSWER { // Turn on AA led
			if val == 0 {
				m.led_AA_off()
			} else {
				m.led_AA_on()
			}
		}
		return OK
	}

	// Sn? - query register n
	_, err = fmt.Sscanf(cmd, "S%d?", &reg)
	if err == nil {
		if reg > 255 || reg < 0 {	
			m.log.Printf("Register index over/underflow: %d", reg)
			return ERROR
		}
		
		fmt.Printf("%d\n", m.readReg(byte(reg)))
		return OK
	}

	// Sn - slect register
	_, err = fmt.Sscanf(cmd, "S%d", &reg)
	if err == nil {
		if reg > 255 || reg < 0 {	
			m.log.Printf("Register index over/underflow: %d", reg)
			return ERROR
		}
		m.curreg = byte(reg)
		return OK
	}

	if err != nil {
		m.log.Printf("registers(): err = %s", err)
	}
	return ERROR
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

