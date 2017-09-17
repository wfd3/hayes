package hayes

import (
	"time"
)

func (m *Modem) getGuardTime() time.Duration {
	gt := m.registers.Read(REG_ESC_CODE_GUARD_TIME)
	return time.Duration(float64(gt) * 0.02) * time.Second
}

func (m *Modem) resetTimer() {
	m.stopTimer()
	guardTime := m.getGuardTime()

	m.log.Printf("Setting timer for %v", guardTime)
	m.timer = time.NewTicker(guardTime)
}

func (m *Modem) stopTimer() {
	if m.timer != nil {
		m.timer.Stop()
	}
}
