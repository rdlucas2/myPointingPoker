package main

import (
	"encoding/json"
	"fmt"
	"sync"
)

// Room-scoped SSE clients: roomID -> set of client channels
var (
	sseRoomClients = make(map[string]map[chan SSEMessage]struct{})
	sseMutex       sync.Mutex
)

func addSSEClient(roomID string, clientChan chan SSEMessage) {
	sseMutex.Lock()
	defer sseMutex.Unlock()
	if sseRoomClients[roomID] == nil {
		sseRoomClients[roomID] = make(map[chan SSEMessage]struct{})
	}
	sseRoomClients[roomID][clientChan] = struct{}{}
}

func removeSSEClient(roomID string, clientChan chan SSEMessage) {
	sseMutex.Lock()
	defer sseMutex.Unlock()
	if clients, ok := sseRoomClients[roomID]; ok {
		delete(clients, clientChan)
		if len(clients) == 0 {
			delete(sseRoomClients, roomID)
		}
	}
}

func broadcastToRoom(roomID string, msg SSEMessage) {
	sseMutex.Lock()
	defer sseMutex.Unlock()
	clients, ok := sseRoomClients[roomID]
	if !ok {
		return
	}
	for clientChan := range clients {
		select {
		case clientChan <- msg:
		default:
		}
	}
}

func sendSSEStoryUpdate(roomID string, story Story) {
	s, err := json.Marshal(story)
	if err != nil {
		fmt.Println(err)
		return
	}
	broadcastToRoom(roomID, SSEMessage{Type: "update_story", Content: string(s)})
}

func sendSSEPlayersUpdate(roomID string, players []Player) {
	p, err := json.Marshal(players)
	if err != nil {
		fmt.Println(err)
		return
	}
	broadcastToRoom(roomID, SSEMessage{Type: "update_players", Content: string(p)})
}

func sendSSEPlayerUpdate(roomID string, player Player) {
	p, err := json.Marshal(player)
	if err != nil {
		fmt.Println(err)
		return
	}
	broadcastToRoom(roomID, SSEMessage{Type: "update_players", Content: string(p)})
}

func sendSSEVisibilityUpdate(roomID string, visible bool) {
	visibilityJson := fmt.Sprintf("{\"visible\": %t}", visible)
	broadcastToRoom(roomID, SSEMessage{Type: "update_visibility", Content: visibilityJson})
}

func sendSSETimeUpdate(roomID string, formattedTimer string) {
	broadcastToRoom(roomID, SSEMessage{Type: "update_timer", Content: formattedTimer})
}
