package main

import (
	"time"
)

var timer *time.Ticker

func guardtime(gt int) time.Duration {
	return time.Duration(float64(gt) * 20) * time.Millisecond
}

func resetGuardCodeTimer(guard_time int) {
	stopGuardCodeTimer()
	gt := guardtime(guard_time)
	logger.Printf("Resetting escape code timer for %v", gt)
	timer = time.NewTicker(gt)
}

func stopGuardCodeTimer() {
	if timer != nil {
		timer.Stop()
	}
}

func startGuardCodeTimer() {
	if timer != nil {
		panic("Can't have more than one Escape Code timer")
	}
	gt := int(registers.Read(REG_ESC_CODE_GUARD_TIME))
	resetGuardCodeTimer(gt)
}

//Consume bytes from the serial port and process or send to remote as
// per conf.mode
func handleSerial() {
	var c, CR, BS, ESC byte
	var s string
	var lastThree [3]byte
	var idx int
	var countAtTick, countAtLastTick uint64
	var waitForOneTick bool

	// Start accepting and processing bytes from the DTE
	countAtTick = 0
	for {

		select {
		case <-timer.C:
			if m.getMode() == COMMANDMODE { // Skip if in COMMAND mode
				continue
			}

			// Look for the command escape sequence
			// (see http://www.messagestick.net/modem/Hayes_Ch1-4.html)
			// Basically:
			//   1s of silence, "+++", 1s of silence.
			// So, count the incoming chars between ticks, saving
			// the previous tick's count.  If you see
			// countAtTick == 3 && CountAtLastTick == 0 && the last
			// three characters are "+++", wait one more tick.  If
			// countAtTick == 0, the guard sequence was detected.

			if countAtTick == 3 && countAtLastTick == 0 &&
				lastThree == escSequence {
				waitForOneTick = true
			} else if waitForOneTick && countAtTick == 0 {
				logger.Print("Escape sequence detected, ",
					"entering command mode")
				m.setMode(COMMANDMODE)
				prstatus(OK)
				s = ""
				continue
			} else {
				waitForOneTick = false
			}
			countAtLastTick = countAtTick
			countAtTick = 0
			continue

		case c = <-serial.channel:
			countAtTick++
		}

		// Syntatic helpers.  Reload each time we loop
		CR  = registers.Read(REG_CR_CH)
		BS  = registers.Read(REG_BS_CH)
		ESC = registers.Read(REG_ESC_CH)

		switch m.getMode() {
		case COMMANDMODE:
			if conf.echoInCmdMode { // Echo back to the DTE
				serial.WriteByte(c)
			}

			// Accumulate chars in s until we read a CR, then process
			// s as a command.

			// 'A/' command, immediately exec.
			switch {
			case  (s == "A" || s == "a") && c == '/':
				serial.Println()
				if m.lastCmd == "" {
					prstatus(ERROR)
				} else {
					err := runCommand(m.lastCmd)
					prstatus(err)
				}
				s = ""

			case c == CR && s != "":
				err := runCommand(s)
				prstatus(err)
				s = ""

			case c == BS && len(s) > 0:
				s = s[0 : len(s)-1]

			case c == CR || c == BS && len(s) == 0:
				// ignore naked CR's & BS if s is already empty

			default:
				s += string(c)
			}

		case DATAMODE:
			// Look for the command escape sequence
			switch c {
			case ESC:
				lastThree[idx] = c
				idx = (idx + 1) % 3
			default: 
				lastThree = [3]byte{' ', ' ', ' '}
				idx = 0
			}
			// Send to remote, blinking the SD LED
			if m.offHook() && m.conn != nil {
				led_SD_on()
				out := make([]byte, 1)
				out[0] = c
				m.conn.Write(out)
				led_SD_off()
			}
		}
	}
}

