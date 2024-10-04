package models

type Page[T any] struct {
	Data []T `json:"data"`
}

type Error struct {
	Status int      `json:"status"`
	Error  []string `json:"error"`
}
