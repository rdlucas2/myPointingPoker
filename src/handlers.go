package main

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/google/uuid"
)

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

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		notFoundHandler(w, r)
		return
	}

	var uniqueID string
	cookie, err := r.Cookie("unique_id")
	if err != nil || cookie.Value == "" {
		uniqueID = uuid.New().String()
		upsertPlayer(uniqueID, "", "", false)
		http.SetCookie(w, &http.Cookie{
			Name:  "unique_id",
			Value: uniqueID,
			Path:  "/",
		})
	} else {
		uniqueID = cookie.Value
		getPlayerById(uniqueID)
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
		cookie, cookieErr := r.Cookie("unique_id")
		if cookieErr != nil {
			http.Error(w, "Player id not found", http.StatusBadRequest)
			return
		}
		uniqueID := cookie.Value
		createErr := upsertPlayer(uniqueID, name, "", false)
		if createErr != nil {
			// http.Error(w, createErr.Error(), http.StatusInternalServerError)
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `<script>confirm("Name already taken, use a new name.")</script>`)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<script>window.location.href = "/pointing";</script>`)
	case "GET":
		cookie, cookieErr := r.Cookie("unique_id")
		if cookieErr != nil {
			http.Error(w, "Player id not found", http.StatusBadRequest)
			return
		}
		uniqueID := cookie.Value
		player, playerErr := getPlayerById(uniqueID)
		if playerErr != nil {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
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
		cookie, cookieErr := r.Cookie("unique_id")
		if cookieErr != nil {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}

		story := getStory()
		sendSSEStoryUpdate(story)

		player, playerErr := getPlayerById(cookie.Value)
		if playerErr != nil {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}
		sendSSEPlayerUpdate(player)

		players, playersErr := getPlayers()
		if playersErr != nil {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}
		sendSSEPlayersUpdate(players)
	case "GET":
		_, cookieErr := r.Cookie("unique_id")
		if cookieErr != nil {
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
		}
		players, playersErr := getPlayers()
		if playersErr != nil {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}
		sendSSEPlayersUpdate(players)
		renderTemplate(w, "templates/pages/pointing.html", data)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func pointingEventsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	clientChan := make(chan SSEMessage)
	sseMutex.Lock()
	sseClients[clientChan] = struct{}{}
	sseMutex.Unlock()

	defer func() {
		sseMutex.Lock()
		delete(sseClients, clientChan)
		sseMutex.Unlock()
		close(clientChan)
	}()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-clientChan:
			if msg.Type == "update_story" {
				fmt.Fprintf(w, "event: update_story\ndata: %s\n\n", msg.Content)
			} else if msg.Type == "update_player" {
				fmt.Fprintf(w, "event: update_player\ndata: %s\n\n", msg.Content)
			} else if msg.Type == "update_players" {
				fmt.Fprintf(w, "event: update_players\ndata: %s\n\n", msg.Content)
			} else if msg.Type == "update_visibility" {
				fmt.Fprintf(w, "event: update_visibility\ndata: %s\n\n", msg.Content)
			} else if msg.Type == "update_timer" {
				fmt.Fprintf(w, "event: update_timer\ndata: %s\n\n", msg.Content)
			} else {
				fmt.Fprintf(w, "event: unknown_type\ndata: %s\n\n", msg.Content)
			}
			w.(http.Flusher).Flush()
		}
	}
}

func playersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		players, playersErr := getPlayers()
		if playersErr != nil {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		}
		playersReady, _ := allPlayersHavePoints()
		if playersReady {
			pointsColumnVisible = true
		}
		data := PlayersTablePartial{
			Players: players,
			Visible: pointsColumnVisible,
		}
		sendSSEVisibilityUpdate()
		renderTemplate(w, "templates/partials/player-table-data.html", data)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func storyHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		title := r.FormValue("title")
		err := upsertStory(title)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		story := getStory()
		sendSSEStoryUpdate(story)
		fmt.Fprintf(w, "%s", title)
	case "GET":
		story := getStory()
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
		points := r.FormValue("points")
		cookie, cookieErr := r.Cookie("unique_id")
		if cookieErr != nil {
			http.Error(w, "Unauthorized", http.StatusBadRequest)
			return
		}
		uniqueID := cookie.Value
		player, playerErr := getPlayerById(uniqueID)
		if playerErr != nil {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		}
		updateErr := upsertPlayer(player.Id, player.Name, points, player.Observer)
		if updateErr != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		players, err := getPlayers()
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		sendSSEPlayersUpdate(players)
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func clearHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		clearPlayersPoints()
		clearStory()
		story := getStory()
		sendSSEStoryUpdate(story)
		players, playersErr := getPlayers()
		if playersErr != nil {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}
		resetTimer()
		pointsColumnVisible = false
		sendSSEPlayersUpdate(players)
		sendSSEVisibilityUpdate()
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

var pointsColumnVisible bool = false

func visibilityHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	pointsColumnVisible = !pointsColumnVisible
	sendSSEVisibilityUpdate()
	w.WriteHeader(http.StatusOK)
}

func observerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	playerId := r.FormValue("playerId")
	player, playerErr := getPlayerById(playerId)
	if playerErr != nil {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}
	player.Observer = !player.Observer
	updateErr := upsertPlayer(player.Id, player.Name, player.Points, player.Observer)
	if updateErr != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	players, err := getPlayers()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	sendSSEPlayersUpdate(players)
	w.WriteHeader(http.StatusOK)
}

func removeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	playerId := r.FormValue("playerId")
	err := deletePlayer(playerId)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	players, err := getPlayers()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	sendSSEPlayersUpdate(players)
	w.WriteHeader(http.StatusOK)
}

func resetTimerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	resetTimer()
	w.WriteHeader(http.StatusOK)
}
