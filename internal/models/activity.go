package models

type Activity struct {
	Message string
	Filter  LogFilter
}

type LogFilter struct {
	Fields    map[string]string
	Timestamp string
}
