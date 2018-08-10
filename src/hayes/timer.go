package main

import (
	"time"
)

var timer *time.Ticker

// Timer functions
func resetTimer() {
	stopTimer()
	// REG_ESC_CODE_GUARD_TIME is in 50th's of a second (20ms)
	gt := registers.Read(REG_ESC_CODE_GUARD_TIME)
	guardTime := time.Duration(float64(gt) * 20) * time.Millisecond
		
	logger.Printf("Setting timer for %v", guardTime)
	timer = time.NewTicker(guardTime)
}

func stopTimer() {
	if timer != nil {
		timer.Stop()
	}
}
