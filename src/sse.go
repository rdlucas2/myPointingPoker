package main

import (
	"encoding/json"
	"fmt"
	"sync"
)

var (
	sseClients = make(map[chan SSEMessage]struct{})
	sseMutex   sync.Mutex
)

func sendSSEStoryUpdate(story Story) {
	sseMutex.Lock()
	defer sseMutex.Unlock()
	s, err := json.Marshal(story)
	if err != nil {
		fmt.Println(err)
		return
	}
	msg := SSEMessage{Type: "update_story", Content: string(s)}
	for clientChan := range sseClients {
		select {
		case clientChan <- msg:
		default:
		}
	}
}

func sendSSEPlayersUpdate(players []Player) {
	sseMutex.Lock()
	defer sseMutex.Unlock()
	p, err := json.Marshal(players)
	if err != nil {
		fmt.Println(err)
		return
	}
	msg := SSEMessage{Type: "update_players", Content: string(p)}
	for clientChan := range sseClients {
		select {
		case clientChan <- msg:
		default:
		}
	}
}

func sendSSEPlayerUpdate(player Player) {
	sseMutex.Lock()
	defer sseMutex.Unlock()
	p, err := json.Marshal(player)
	if err != nil {
		fmt.Println(err)
		return
	}
	msg := SSEMessage{Type: "update_players", Content: string(p)}
	for clientChan := range sseClients {
		select {
		case clientChan <- msg:
		default:
		}
	}
}

func sendSSEVisibilityUpdate() {
	sseMutex.Lock()
	defer sseMutex.Unlock()
	visibilityJson := fmt.Sprintf("{\"visible\": %t}", pointsColumnVisible)
	msg := SSEMessage{Type: "update_visibility", Content: visibilityJson}
	for clientChan := range sseClients {
		select {
		case clientChan <- msg:
		default:
		}
	}
}

func sendSSETimeUpdate(formattedTimer string) {
	sseMutex.Lock()
	defer sseMutex.Unlock()
	msg := SSEMessage{Type: "update_timer", Content: formattedTimer}
	for clientChan := range sseClients {
		select {
		case clientChan <- msg:
		default:
		}
	}
}
