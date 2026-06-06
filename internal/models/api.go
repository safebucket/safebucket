package models

type Page[T any] struct {
	Data       []T     `json:"data"`
	NextCursor *string `json:"next_cursor,omitempty"`
}

type Error struct {
	Status int      `json:"status"`
	Error  []string `json:"error"`
}
