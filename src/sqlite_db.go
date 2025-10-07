//go:build !production

package main

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type sqliteDatabase struct {
	db *sql.DB
}

func newDatabase() (Database, error) {
	db, err := sql.Open("sqlite3", "file:pointing.db?cache=shared&mode=rwc")
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	sqliteDB := &sqliteDatabase{db: db}
	if err := sqliteDB.initSchema(); err != nil {
		db.Close()
		return nil, err
	}
	return sqliteDB, nil
}

func (s *sqliteDatabase) initSchema() error {
	createStoriesTable := `
    CREATE TABLE IF NOT EXISTS stories (
        id INTEGER PRIMARY KEY,
        title TEXT
    );`
	if _, err := s.db.Exec(createStoriesTable); err != nil {
		return fmt.Errorf("create stories table: %w", err)
	}

	createPlayersTable := `
    CREATE TABLE IF NOT EXISTS players (
        id TEXT PRIMARY KEY,
        name TEXT UNIQUE,
        points TEXT,
        observer BOOLEAN
    );`
	if _, err := s.db.Exec(createPlayersTable); err != nil {
		return fmt.Errorf("create players table: %w", err)
	}
	return nil
}

func (s *sqliteDatabase) Close() error {
	return s.db.Close()
}

func (s *sqliteDatabase) UpsertStory(title string) error {
	_, err := s.db.Exec("INSERT INTO stories (id, title) VALUES (1, ?) ON CONFLICT(id) DO UPDATE SET title = excluded.title", title)
	if err != nil {
		return fmt.Errorf("upsert story: %w", err)
	}
	return nil
}

func (s *sqliteDatabase) UpsertPlayer(id, name, points string, observer bool) error {
	_, err := s.db.Exec("INSERT INTO players (id, name, points, observer) VALUES (?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET name = excluded.name, points = excluded.points, observer = excluded.observer", id, name, points, observer)
	if err != nil {
		return fmt.Errorf("upsert player: %w", err)
	}
	return nil
}

func (s *sqliteDatabase) GetStory() (Story, error) {
	var story Story
	row := s.db.QueryRow("SELECT title FROM stories ORDER BY id DESC LIMIT 1")
	if err := row.Scan(&story.Title); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return story, nil
		}
		return story, fmt.Errorf("get story: %w", err)
	}
	return story, nil
}

func (s *sqliteDatabase) GetPlayerByID(id string) (Player, error) {
	var player Player
	row := s.db.QueryRow("SELECT id, name, points, observer FROM players WHERE id = ?", id)
	if err := row.Scan(&player.Id, &player.Name, &player.Points, &player.Observer); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return player, fmt.Errorf("no player found with id: %s", id)
		}
		return player, fmt.Errorf("get player by id: %w", err)
	}
	return player, nil
}

func (s *sqliteDatabase) GetPlayers() ([]Player, error) {
	rows, err := s.db.Query("SELECT id, name, points, observer FROM players")
	if err != nil {
		return nil, fmt.Errorf("get players: %w", err)
	}
	defer rows.Close()

	var players []Player
	for rows.Next() {
		var p Player
		if err := rows.Scan(&p.Id, &p.Name, &p.Points, &p.Observer); err != nil {
			return nil, fmt.Errorf("scan player: %w", err)
		}
		players = append(players, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate players: %w", err)
	}
	return players, nil
}

func (s *sqliteDatabase) ClearPlayersPoints() error {
	if _, err := s.db.Exec("UPDATE players SET points = ''"); err != nil {
		return fmt.Errorf("clear player points: %w", err)
	}
	return nil
}

func (s *sqliteDatabase) ClearStory() error {
	if _, err := s.db.Exec("UPDATE stories SET title = ''"); err != nil {
		return fmt.Errorf("clear story: %w", err)
	}
	return nil
}

func (s *sqliteDatabase) DeletePlayer(id string) error {
	if _, err := s.db.Exec("DELETE FROM players WHERE id = ?", id); err != nil {
		return fmt.Errorf("delete player: %w", err)
	}
	return nil
}
