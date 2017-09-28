package hayes

// Simulate the phone line

// Is the phone on or off hook?
const (
	ONHOOK = iota 
	OFFHOOK
)

// ATH0
func (m *Modem) goOnHook() error {
	m.dcd = false

	// It's OK to hang up the phone when there's no active network connection.
	// But if there is, close it.
	if m.conn != nil {
		m.conn.Close()
		m.conn = nil
	}

	m.hookLock.Lock()
	m.hook = ONHOOK
	m.hookLock.Unlock()

	m.mode = COMMANDMODE
	m.connectSpeed = 0
	m.setLineBusy(false)
	m.led_HS_off()
	m.led_OH_off()
	return OK
}

// ATH1
func (m *Modem) goOffHook() error {
	m.setLineBusy(true)

	m.hookLock.Lock()
	m.hook = OFFHOOK
	m.hookLock.Unlock()

	m.led_OH_on()
	return OK
}

func (m *Modem) onHook() bool {
	m.hookLock.RLock()
	defer m.hookLock.RUnlock()
	return m.hook == ONHOOK
}

func (m *Modem) offHook() bool {
	m.hookLock.RLock()
	defer m.hookLock.RUnlock()
	return m.hook == OFFHOOK
}

// Is the phone line busy?
func (m *Modem) getLineBusy() bool {
	m.lineBusyLock.RLock()
	defer m.lineBusyLock.RUnlock()
	return m.lineBusy
}	

func (m *Modem) setLineBusy(b bool) {
	m.lineBusyLock.Lock()
	defer m.lineBusyLock.Unlock()
	m.lineBusy = b
}

// "Busy" signal.
func (m *Modem) checkBusy() bool {
	return  m.offHook() || m.getLineBusy()
}

