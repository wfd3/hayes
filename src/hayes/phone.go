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

	m.hooklock.Lock()
	m.hook = ONHOOK
	m.hooklock.Unlock()

	m.mode = COMMANDMODE
	m.connect_speed = 0
	m.setLineBusy(false)
	m.led_HS_off()
	m.led_OH_off()
	return OK
}

// ATH1
func (m *Modem) goOffHook() error {
	m.setLineBusy(true)

	m.hooklock.Lock()
	m.hook = OFFHOOK
	m.hooklock.Unlock()

	m.led_OH_on()
	return OK
}

func (m *Modem) onHook() bool {
	m.hooklock.RLock()
	defer m.hooklock.RUnlock()
	return m.hook == ONHOOK
}

func (m *Modem) offHook() bool {
	m.hooklock.RLock()
	defer m.hooklock.RUnlock()
	return m.hook == OFFHOOK
}

// Is the phone line busy?
func (m *Modem) getLineBusy() bool {
	m.linebusylock.RLock()
	defer m.linebusylock.RUnlock()
	return m.linebusy
}	

func (m *Modem) setLineBusy(b bool) {
	m.linebusylock.Lock()
	defer m.linebusylock.Unlock()
	m.linebusy = b
}

// "Busy" signal.
func (m *Modem) checkBusy() bool {
	return  m.offHook() || m.getLineBusy()
}

