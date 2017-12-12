package main

// Simulate the phone line

// Is the phone on or off hook?
const (
	ONHOOK = iota
	OFFHOOK
)

// ATH0
func goOnHook() error {
	m.dcd = false
	m.hookLock.Lock()
	m.hook = ONHOOK
	m.hookLock.Unlock()

	// It's OK to hang up the phone when there's no active network connection.
	// But if there is, close it.
	if netConn != nil {
		logger.Printf("Hanging up on active connection (remote %s)",
			netConn.RemoteAddr())
		netConn.Close()
		netConn = nil
	}

	if err := serial.Flush(); err != nil {
		logger.Printf("serial.Flush(): %s", err)
	}

	m.mode = COMMANDMODE
	m.connectSpeed = 0
	setLineBusy(false)
	led_HS_off()
	led_OH_off()
	return OK
}

// ATH1
func goOffHook() error {
	setLineBusy(true)

	m.hookLock.Lock()
	m.hook = OFFHOOK
	m.hookLock.Unlock()

	led_OH_on()
	return OK
}

func onHook() bool {
	m.hookLock.RLock()
	defer m.hookLock.RUnlock()
	return m.hook == ONHOOK
}

func offHook() bool {
	m.hookLock.RLock()
	defer m.hookLock.RUnlock()
	return m.hook == OFFHOOK
}

// Is the phone line busy?
func getLineBusy() bool {
	m.lineBusyLock.RLock()
	defer m.lineBusyLock.RUnlock()
	return m.lineBusy
}

func setLineBusy(b bool) {
	m.lineBusyLock.Lock()
	defer m.lineBusyLock.Unlock()
	m.lineBusy = b
}

// "Busy" signal.
func checkBusy() bool {
	return offHook() || getLineBusy()
}
