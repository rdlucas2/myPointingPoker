package main

import (
	"database/sql"
	"log"
	"net/http"
)

var db *sql.DB

func main() {
	initDB(false)
	defer db.Close()
	// Define HTTP routes here
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/pointing", pointingHandler)
	http.HandleFunc("/pointing-events", pointingEventsHandler)
	http.HandleFunc("/story", storyHandler)
	http.HandleFunc("/player", playerHandler)
	http.HandleFunc("/players", playersHandler)
	http.HandleFunc("/points", pointsHandler)
	http.HandleFunc("/clear", clearHandler)
	http.HandleFunc("/visibility", visibilityHandler)
	http.HandleFunc("/observer", observerHandler)
	http.HandleFunc("/remove", removeHandler)
	http.HandleFunc("/reset-timer", resetTimerHandler)

	// Define the port on which to listen
	port := "8080"
	log.Printf("Starting server on port %s", port)

	// Start the HTTP server
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
