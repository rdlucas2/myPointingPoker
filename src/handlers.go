package main

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/google/uuid"
)

// Per-room visibility state
var roomVisibility = make(map[string]bool)

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	t, err := template.ParseFiles(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = t.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("404 - Page Not Found"))
}

// getRoomID extracts the room_id cookie. Returns empty string if not set.
func getRoomID(r *http.Request) string {
	cookie, err := r.Cookie("room_id")
	if err != nil || cookie.Value == "" {
		return ""
	}
	return cookie.Value
}

// getUniqueID extracts the unique_id cookie. Returns empty string if not set.
func getUniqueID(r *http.Request) string {
	cookie, err := r.Cookie("unique_id")
	if err != nil || cookie.Value == "" {
		return ""
	}
	return cookie.Value
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		notFoundHandler(w, r)
		return
	}

	base := BasePageData{
		Title:       "Start",
		Header:      "My pointing app!",
		Description: "The homepage for my pointing poker app",
	}

	data := IndexPageData{
		BasePageData: base,
	}
	renderTemplate(w, "templates/pages/index.html", data)
}

func playerHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		name := r.FormValue("name")
		roomID := r.FormValue("room")

		if name == "" || roomID == "" {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `<script>alert("Name and Room ID are required.");window.location.href = "/";</script>`)
			return
		}

		// Get or create unique ID
		var uniqueID string
		cookie, err := r.Cookie("unique_id")
		if err != nil || cookie.Value == "" {
			uniqueID = uuid.New().String()
		} else {
			uniqueID = cookie.Value
		}

		// Set cookies
		http.SetCookie(w, &http.Cookie{
			Name:  "unique_id",
			Value: uniqueID,
			Path:  "/",
		})
		http.SetCookie(w, &http.Cookie{
			Name:  "room_id",
			Value: roomID,
			Path:  "/",
		})

		// Ensure room exists first
		if err := ensureRoom(roomID); err != nil {
			http.Error(w, "Failed to create room", http.StatusInternalServerError)
			return
		}

		// Check for duplicate name within the room
		taken, checkErr := isNameTakenInRoom(roomID, name, uniqueID)
		if checkErr != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if taken {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `<script>alert("Name already taken in this room, use a new name.");window.location.href = "/";</script>`)
			return
		}

		// Create player
		createErr := upsertPlayer(uniqueID, name, "", false)
		if createErr != nil {
			http.Error(w, "Failed to create player", http.StatusInternalServerError)
			return
		}

		// Add player to room
		if err := addPlayerToRoom(roomID, uniqueID); err != nil {
			http.Error(w, "Failed to join room", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<script>window.location.href = "/pointing";</script>`)
	case "GET":
		uniqueID := getUniqueID(r)
		if uniqueID == "" {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}
		player, playerErr := getPlayerById(uniqueID)
		if playerErr != nil {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}
		data := PlayerPartial{
			Player: player,
		}
		renderTemplate(w, "templates/partials/player.html", data)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func pointingHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		uniqueID := getUniqueID(r)
		roomID := getRoomID(r)
		if uniqueID == "" || roomID == "" {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}

		story := getStoryByRoom(roomID)
		sendSSEStoryUpdate(roomID, story)

		player, playerErr := getPlayerById(uniqueID)
		if playerErr != nil {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}
		sendSSEPlayerUpdate(roomID, player)

		players, playersErr := getPlayersByRoom(roomID)
		if playersErr != nil {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}
		sendSSEPlayersUpdate(roomID, players)
	case "GET":
		uniqueID := getUniqueID(r)
		roomID := getRoomID(r)
		if uniqueID == "" || roomID == "" {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}

		// Verify the player exists
		_, playerErr := getPlayerById(uniqueID)
		if playerErr != nil {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}

		base := BasePageData{
			Title:       "Pointing",
			Header:      "My pointing poker!",
			Description: "The homepage for my pointing poker app",
		}
		data := PointingPageData{
			BasePageData: base,
			RoomID:       roomID,
		}
		players, playersErr := getPlayersByRoom(roomID)
		if playersErr != nil {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}
		sendSSEPlayersUpdate(roomID, players)
		renderTemplate(w, "templates/pages/pointing.html", data)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func pointingEventsHandler(w http.ResponseWriter, r *http.Request) {
	roomID := getRoomID(r)
	if roomID == "" {
		http.Error(w, "Room ID required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	clientChan := make(chan SSEMessage)
	addSSEClient(roomID, clientChan)

	defer func() {
		removeSSEClient(roomID, clientChan)
		close(clientChan)
	}()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-clientChan:
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", msg.Type, msg.Content)
			w.(http.Flusher).Flush()
		}
	}
}

func playersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		roomID := getRoomID(r)
		if roomID == "" {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}
		players, playersErr := getPlayersByRoom(roomID)
		if playersErr != nil {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}
		playersReady, _ := allPlayersInRoomHavePoints(roomID)
		if playersReady {
			roomVisibility[roomID] = true
		}
		visible := roomVisibility[roomID]
		data := PlayersTablePartial{
			Players: players,
			Visible: visible,
			RoomID:  roomID,
		}
		sendSSEVisibilityUpdate(roomID, visible)
		renderTemplate(w, "templates/partials/player-table-data.html", data)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func storyHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		roomID := getRoomID(r)
		if roomID == "" {
			http.Error(w, "Room ID required", http.StatusBadRequest)
			return
		}
		title := r.FormValue("title")
		err := upsertStory(roomID, title)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		story := getStoryByRoom(roomID)
		sendSSEStoryUpdate(roomID, story)
		fmt.Fprintf(w, "%s", title)
	case "GET":
		roomID := getRoomID(r)
		if roomID == "" {
			http.Error(w, "Room ID required", http.StatusBadRequest)
			return
		}
		story := getStoryByRoom(roomID)
		data := StoryPartial{
			Story: story,
		}
		renderTemplate(w, "templates/partials/story.html", data)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func pointsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		uniqueID := getUniqueID(r)
		roomID := getRoomID(r)
		if uniqueID == "" || roomID == "" {
			http.Error(w, "Unauthorized", http.StatusBadRequest)
			return
		}
		points := r.FormValue("points")
		player, playerErr := getPlayerById(uniqueID)
		if playerErr != nil {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}
		updateErr := upsertPlayer(player.Id, player.Name, points, player.Observer)
		if updateErr != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		players, err := getPlayersByRoom(roomID)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		sendSSEPlayersUpdate(roomID, players)
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func clearHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		roomID := getRoomID(r)
		if roomID == "" {
			http.Error(w, "Room ID required", http.StatusBadRequest)
			return
		}
		clearPlayersPointsByRoom(roomID)
		clearStoryByRoom(roomID)
		story := getStoryByRoom(roomID)
		sendSSEStoryUpdate(roomID, story)
		players, playersErr := getPlayersByRoom(roomID)
		if playersErr != nil {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}
		roomVisibility[roomID] = false
		sendSSEVisibilityUpdate(roomID, false)
		go sendSSEPlayersUpdate(roomID, players)
		resetTimer(roomID)
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func visibilityHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	roomID := getRoomID(r)
	if roomID == "" {
		http.Error(w, "Room ID required", http.StatusBadRequest)
		return
	}
	roomVisibility[roomID] = !roomVisibility[roomID]
	sendSSEVisibilityUpdate(roomID, roomVisibility[roomID])
	w.WriteHeader(http.StatusOK)
}

func observerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	roomID := getRoomID(r)
	if roomID == "" {
		http.Error(w, "Room ID required", http.StatusBadRequest)
		return
	}
	playerId := r.FormValue("playerId")
	player, playerErr := getPlayerById(playerId)
	if playerErr != nil {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	player.Observer = !player.Observer
	updateErr := upsertPlayer(player.Id, player.Name, player.Points, player.Observer)
	if updateErr != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	players, err := getPlayersByRoom(roomID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	sendSSEPlayersUpdate(roomID, players)
	w.WriteHeader(http.StatusOK)
}

func removeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	roomID := getRoomID(r)
	if roomID == "" {
		http.Error(w, "Room ID required", http.StatusBadRequest)
		return
	}
	playerId := r.FormValue("playerId")
	removePlayerFromRoom(roomID, playerId)
	err := deletePlayer(playerId)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	players, err := getPlayersByRoom(roomID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	sendSSEPlayersUpdate(roomID, players)
	w.WriteHeader(http.StatusOK)
}

func resetTimerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	roomID := getRoomID(r)
	if roomID == "" {
		http.Error(w, "Room ID required", http.StatusBadRequest)
		return
	}
	resetTimer(roomID)
	w.WriteHeader(http.StatusOK)
}
