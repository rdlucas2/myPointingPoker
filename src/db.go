package main

import (
	"log"
	"sync"
)

type Database interface {
	Close() error
	UpsertStory(title string) error
	UpsertPlayer(id, name, points string, observer bool) error
	GetStory() (Story, error)
	GetPlayerByID(id string) (Player, error)
	GetPlayers() ([]Player, error)
	ClearPlayersPoints() error
	ClearStory() error
	DeletePlayer(id string) error
}

var (
	dbClient    Database
	storyMutex  sync.Mutex
	playerMutex sync.Mutex
)

func initDB() {
	var err error
	dbClient, err = newDatabase()
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
}

func closeDB() {
	if dbClient == nil {
		return
	}
	if err := dbClient.Close(); err != nil {
		log.Printf("error closing database: %v", err)
	}
}

func upsertStory(title string) error {
	storyMutex.Lock()
	defer storyMutex.Unlock()
	return dbClient.UpsertStory(title)
}

func upsertPlayer(id string, name string, points string, observer bool) error {
	playerMutex.Lock()
	defer playerMutex.Unlock()
	return dbClient.UpsertPlayer(id, name, points, observer)
}

func getStory() Story {
	story, err := dbClient.GetStory()
	if err != nil {
		log.Printf("error retrieving story: %v", err)
	}
	return story
}

func getPlayerById(id string) (Player, error) {
	return dbClient.GetPlayerByID(id)
}

func getPlayers() ([]Player, error) {
	return dbClient.GetPlayers()
}

func clearPlayersPoints() {
	if err := dbClient.ClearPlayersPoints(); err != nil {
		log.Printf("Error clearing player points: %v", err)
	}
}

func clearStory() {
	if err := dbClient.ClearStory(); err != nil {
		log.Printf("Error clearing story title: %v", err)
	}
}

func deletePlayer(id string) error {
	return dbClient.DeletePlayer(id)
}

func allPlayersHavePoints() (bool, error) {
	players, err := dbClient.GetPlayers()
	if err != nil {
		return false, err
	}
	for _, player := range players {
		if !player.Observer && player.Points == "" {
			return false, nil
		}
	}
	return true, nil
}
