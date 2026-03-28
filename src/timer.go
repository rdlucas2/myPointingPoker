package main

import (
	"fmt"
	"sync"
	"time"
)

type roomTimer struct {
	startTime time.Time
	running   bool
}

var (
	roomTimers    = make(map[string]*roomTimer)
	roomTimersMux sync.Mutex
)

func resetTimer(roomID string) {
	roomTimersMux.Lock()
	rt, exists := roomTimers[roomID]
	if !exists {
		rt = &roomTimer{}
		roomTimers[roomID] = rt
	}
	rt.startTime = time.Now()
	wasRunning := rt.running
	rt.running = true
	roomTimersMux.Unlock()

	if !wasRunning {
		go startRoomTimer(roomID)
	}
}

func startRoomTimer(roomID string) {
	for {
		roomTimersMux.Lock()
		rt, exists := roomTimers[roomID]
		if !exists || !rt.running {
			roomTimersMux.Unlock()
			return
		}
		elapsed := time.Since(rt.startTime)
		roomTimersMux.Unlock()

		formattedTime := formatDurationAsHHMMSS(elapsed)
		sendSSETimeUpdate(roomID, formattedTime)
		time.Sleep(1 * time.Second)
	}
}

func formatDurationAsHHMMSS(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}
