package main

import (
	"sync"
	"time"
)


// Basic modem state.  This is ephemeral and can be restored by a
// stored config.  Fields starting with _ must be protected by Modem.lock

// What mode is the modem in?
const (
	COMMANDMODE bool = false
	DATAMODE    bool = true
)

type Modem struct {
	lock          sync.RWMutex
	currentConfig int            // Which stored config are we using
	_mode         bool           // DATA or COMMAND mode
	lastCmd       string         // Last command (for A/ command)
	lastDialed    string         // Last number dialed (for ATDL)
	_connectSpeed int            // What speed did we connect at (0 or 38k)
	_dcd          bool           // Data Carrier Detect -- active connection?
	_lineBusy     bool           // Is the "phone line" busy?
	_hook         bool           // Is the phone on or off hook?
	_lastRingTime time.Time	     // When did the last ring occur? 
	conn          connection     // Current active connection
}

func (m *Modem) setMode(mode bool) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m._mode = mode
}

func (m *Modem) getMode() bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m._mode
}

func (m *Modem) setConnectSpeed(speed int) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m._connectSpeed = speed
}

func (m *Modem) getConnectSpeed() int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m._connectSpeed
}

func (m *Modem) getLineBusy() bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m._lineBusy
}

func (m *Modem) setLineBusy(b bool) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m._lineBusy = b
}

func (m *Modem) goOnHook() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m._hook = ONHOOK
}

func (m *Modem) goOffHook() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m._hook = OFFHOOK
}

func (m *Modem) onHook() bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m._hook == ONHOOK
}

func (m *Modem) offHook() bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m._hook == OFFHOOK
}

func (m *Modem) dcdHigh() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m._dcd = true
}

func (m *Modem) dcdLow() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m._dcd = false
}

func (m *Modem) getdcd() bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m._dcd
}

func (m *Modem) setLastRingTime() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m._lastRingTime = time.Now()
}

func (m *Modem) getLastRingTime() time.Time {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m._lastRingTime
}

func (m *Modem) resetLastRingTime() {
	m.lock.RLock()
	defer m.lock.RUnlock()
	m._lastRingTime = time.Time{}
}
