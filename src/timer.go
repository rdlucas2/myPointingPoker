package main

import (
	"fmt"
	"time"
)

var (
	startTime    time.Time
	timerRunning bool
)

func resetTimer() {
	startTime = time.Now()
	if !timerRunning {
		startTimer()
	}
}

func startTimer() {
	timerRunning = true
	go func() {
		for timerRunning {
			elapsedTime := time.Since(startTime)
			formattedTime := formatDurationAsHHMMSS(elapsedTime)
			sendSSETimeUpdate(formattedTime)
			time.Sleep(1 * time.Second)
		}
	}()
}

func formatDurationAsHHMMSS(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}
