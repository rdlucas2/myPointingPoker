package main

type BasePageData struct {
	Title       string
	Header      string
	Description string
}

type IndexPageData struct {
	BasePageData BasePageData
}

type PointingPageData struct {
	BasePageData  BasePageData
	CurrentPlayer string
	RoomID        string
}

type Story struct {
	Id     int
	Title  string
	RoomID string
}

type Player struct {
	Id       string
	Name     string
	Points   string
	Observer bool
}

type PlayersTablePartial struct {
	Players []Player
	Visible bool
	RoomID  string
}

type PlayerPartial struct {
	Player Player
}

type StoryPartial struct {
	Story Story
}

type SSEMessage struct {
	Type    string
	Content string
}
