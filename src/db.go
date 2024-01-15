package main

import (
	"database/sql"
	"fmt"
	"log"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

var (
	storyMutex  sync.Mutex
	playerMutex sync.Mutex
)

func initDB(isLocal bool) {
	var err error
	if isLocal {
		db, err = sql.Open("sqlite3", ":memory:") // Use in-memory database for testing
		if err != nil {
			log.Fatal(err)
		}
	} else {
		db, err = sql.Open("sqlite3", "file:pointing.db?cache=shared&mode=rwc")
		if err != nil {
			log.Fatal(err)
		}
	}

	createStoriesTable := `
    CREATE TABLE IF NOT EXISTS stories (
        id INTEGER PRIMARY KEY,
        title TEXT
    );`
	_, err = db.Exec(createStoriesTable)
	if err != nil {
		log.Fatal(err)
	}

	createPlayersTable := `
    CREATE TABLE IF NOT EXISTS players (
        id TEXT PRIMARY KEY,
		name TEXT UNIQUE,
		points TEXT,
		observer BOOLEAN
    );`
	_, err = db.Exec(createPlayersTable)
	if err != nil {
		log.Fatal(err)
	}
}

func upsertStory(title string) error {
	storyMutex.Lock()
	defer storyMutex.Unlock()
	// Assuming 'id' is the primary key and always has the same value like 1 for your single story
	_, err := db.Exec("INSERT INTO stories (id, title) VALUES (1, ?) ON CONFLICT(id) DO UPDATE SET title = excluded.title", title)
	if err != nil {
		return err
	}
	return nil
}

func upsertPlayer(id string, name string, points string, observer bool) error {
	playerMutex.Lock()
	defer playerMutex.Unlock()
	_, err := db.Exec("INSERT INTO players (id, name, points, observer) VALUES (?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET name = excluded.name, points = excluded.points, observer = excluded.observer", id, name, points, observer)
	if err != nil {
		return err
	}
	return nil
}

func getStory() Story {
	var story Story
	row := db.QueryRow("SELECT title FROM stories ORDER BY id DESC LIMIT 1")
	err := row.Scan(&story.Title)
	if err != nil {
		return story
	}
	return story
}

func getPlayerById(id string) (Player, error) {
	var player Player

	// Query the database for the player with the given id
	row := db.QueryRow("SELECT id, name, points, observer FROM players WHERE id = ?", id)

	// Scan the result into the Player struct
	err := row.Scan(&player.Id, &player.Name, &player.Points, &player.Observer)
	if err != nil {
		if err == sql.ErrNoRows {
			// No rows were returned - player with the given id does not exist
			return player, fmt.Errorf("no player found with id: %s", id)
		}
		// Some other error occurred during the query
		return player, err
	}

	return player, nil
}

func getPlayers() ([]Player, error) {
	// Slice to hold the players
	var players []Player

	rows, err := db.Query("SELECT id, name, points, observer FROM players")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var p Player
		err := rows.Scan(&p.Id, &p.Name, &p.Points, &p.Observer)
		if err != nil {
			return nil, err
		}
		players = append(players, p)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return players, nil
}

func clearPlayersPoints() {
	if _, err := db.Exec("UPDATE players SET points = ''"); err != nil {
		log.Printf("Error clearing player points: %v", err)
		return
	}
}

func clearStory() {
	if _, err := db.Exec("UPDATE stories SET title = ''"); err != nil {
		log.Printf("Error clearing story title: %v", err)
		return
	}
}

func deletePlayer(id string) error {
	_, err := db.Exec("DELETE FROM players WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("error deleting player: %w", err)
	}
	return nil
}

func allPlayersHavePoints() (bool, error) {
	players, playersErr := getPlayers()
	if playersErr != nil {
		return false, playersErr
	}
	var nonobservers []Player
	for _, player := range players {
		if !player.Observer {
			nonobservers = append(nonobservers, player)
		}
	}
	for _, p := range nonobservers {
		if p.Points == "" {
			return false, nil
		}
	}
	return true, nil
}
