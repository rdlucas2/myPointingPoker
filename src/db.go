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
	roomMutex   sync.Mutex
)

func initDB(isLocal bool) {
	var err error
	if isLocal {
		db, err = sql.Open("sqlite3", ":memory:")
		if err != nil {
			log.Fatal(err)
		}
	} else {
		db, err = sql.Open("sqlite3", "file:pointing.db?cache=shared&mode=rwc")
		if err != nil {
			log.Fatal(err)
		}
	}

	// Enable foreign keys
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		log.Fatal(err)
	}

	createRoomsTable := `
    CREATE TABLE IF NOT EXISTS rooms (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        room_id TEXT UNIQUE NOT NULL
    );`
	_, err = db.Exec(createRoomsTable)
	if err != nil {
		log.Fatal(err)
	}

	createPlayersTable := `
    CREATE TABLE IF NOT EXISTS players (
        id TEXT PRIMARY KEY,
        name TEXT,
        points TEXT,
        observer BOOLEAN
    );`
	_, err = db.Exec(createPlayersTable)
	if err != nil {
		log.Fatal(err)
	}

	createRoomPlayersTable := `
    CREATE TABLE IF NOT EXISTS room_players (
        room_id TEXT NOT NULL,
        player_id TEXT NOT NULL,
        PRIMARY KEY (room_id, player_id),
        FOREIGN KEY (room_id) REFERENCES rooms(room_id) ON DELETE CASCADE,
        FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE
    );`
	_, err = db.Exec(createRoomPlayersTable)
	if err != nil {
		log.Fatal(err)
	}

	createStoriesTable := `
    CREATE TABLE IF NOT EXISTS stories (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        room_id TEXT UNIQUE NOT NULL,
        title TEXT,
        FOREIGN KEY (room_id) REFERENCES rooms(room_id) ON DELETE CASCADE
    );`
	_, err = db.Exec(createStoriesTable)
	if err != nil {
		log.Fatal(err)
	}
}

// Room functions

func ensureRoom(roomID string) error {
	roomMutex.Lock()
	defer roomMutex.Unlock()
	_, err := db.Exec("INSERT OR IGNORE INTO rooms (room_id) VALUES (?)", roomID)
	return err
}

func addPlayerToRoom(roomID string, playerID string) error {
	roomMutex.Lock()
	defer roomMutex.Unlock()
	_, err := db.Exec("INSERT OR IGNORE INTO room_players (room_id, player_id) VALUES (?, ?)", roomID, playerID)
	return err
}

func removePlayerFromRoom(roomID string, playerID string) error {
	roomMutex.Lock()
	defer roomMutex.Unlock()
	_, err := db.Exec("DELETE FROM room_players WHERE room_id = ? AND player_id = ?", roomID, playerID)
	return err
}

func getPlayersByRoom(roomID string) ([]Player, error) {
	var players []Player
	rows, err := db.Query(`
		SELECT p.id, p.name, p.points, p.observer
		FROM players p
		INNER JOIN room_players rp ON p.id = rp.player_id
		WHERE rp.room_id = ?`, roomID)
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

// Story functions (room-scoped)

func upsertStory(roomID string, title string) error {
	storyMutex.Lock()
	defer storyMutex.Unlock()
	_, err := db.Exec(`
		INSERT INTO stories (room_id, title) VALUES (?, ?)
		ON CONFLICT(room_id) DO UPDATE SET title = excluded.title`, roomID, title)
	// If no unique constraint on room_id yet, we need one
	return err
}

func getStoryByRoom(roomID string) Story {
	var story Story
	row := db.QueryRow("SELECT COALESCE(title, '') FROM stories WHERE room_id = ?", roomID)
	err := row.Scan(&story.Title)
	if err != nil {
		return story
	}
	story.RoomID = roomID
	return story
}

func clearStoryByRoom(roomID string) {
	if _, err := db.Exec("UPDATE stories SET title = '' WHERE room_id = ?", roomID); err != nil {
		log.Printf("Error clearing story title: %v", err)
	}
}

// Player functions

func upsertPlayer(id string, name string, points string, observer bool) error {
	playerMutex.Lock()
	defer playerMutex.Unlock()
	_, err := db.Exec("INSERT INTO players (id, name, points, observer) VALUES (?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET name = excluded.name, points = excluded.points, observer = excluded.observer", id, name, points, observer)
	if err != nil {
		return err
	}
	return nil
}

func getPlayerById(id string) (Player, error) {
	var player Player
	row := db.QueryRow("SELECT id, name, points, observer FROM players WHERE id = ?", id)
	err := row.Scan(&player.Id, &player.Name, &player.Points, &player.Observer)
	if err != nil {
		if err == sql.ErrNoRows {
			return player, fmt.Errorf("no player found with id: %s", id)
		}
		return player, err
	}
	return player, nil
}

func clearPlayersPointsByRoom(roomID string) {
	if _, err := db.Exec(`
		UPDATE players SET points = ''
		WHERE id IN (SELECT player_id FROM room_players WHERE room_id = ?)`, roomID); err != nil {
		log.Printf("Error clearing player points: %v", err)
	}
}

func deletePlayer(id string) error {
	_, err := db.Exec("DELETE FROM players WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("error deleting player: %w", err)
	}
	return nil
}

func allPlayersInRoomHavePoints(roomID string) (bool, error) {
	players, err := getPlayersByRoom(roomID)
	if err != nil {
		return false, err
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

func isNameTakenInRoom(roomID string, name string, excludePlayerID string) (bool, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM players p
		INNER JOIN room_players rp ON p.id = rp.player_id
		WHERE rp.room_id = ? AND p.name = ? AND p.id != ?`, roomID, name, excludePlayerID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func getRoomForPlayer(playerID string) (string, error) {
	var roomID string
	row := db.QueryRow("SELECT room_id FROM room_players WHERE player_id = ? LIMIT 1", playerID)
	err := row.Scan(&roomID)
	if err != nil {
		return "", fmt.Errorf("no room found for player: %s", playerID)
	}
	return roomID, nil
}
