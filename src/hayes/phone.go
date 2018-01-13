package main

// Simulate the phone line

// Is the phone on or off hook?
const (
	ONHOOK = false
	OFFHOOK = true
)

// ATH0
func hangup() error {
	var ret error = OK
	
	m.dcd = false
	lowerDSR()
	m.hook = ONHOOK

	// It's OK to hang up the phone when there's no active network connection.
	// But if there is, close it.
	if m.conn != nil {
		logger.Printf("Hanging up on active connection (remote %s)",
			m.conn.RemoteAddr())
		m.conn.Close()
		ret = NO_CARRIER
	}

	m.mode = COMMANDMODE
	m.connectSpeed = 0
	setLineBusy(false)
	led_HS_off()
	led_OH_off()

       	if err := serial.Flush(); err != nil {
		logger.Printf("serial.Flush(): %s", err)
	}

	return ret
}

// ATH1
func pickup() error {
	setLineBusy(true)
	m.hook = OFFHOOK
	led_OH_on()
	return OK
}

func onHook() bool {
	return m.hook == ONHOOK
}

func offHook() bool {
	return m.hook == OFFHOOK
}

// Is the phone line busy?
func getLineBusy() bool {
	return m.lineBusy
}

func setLineBusy(b bool) {
	m.lineBusy = b
}

// "Busy" signal.
func checkBusy() bool {
	return offHook() || getLineBusy()
}
