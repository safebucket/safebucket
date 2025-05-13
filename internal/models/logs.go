package models

type LogMessage struct {
	Message string
	Filter  LogFilter
}

type LogFilter struct {
	Fields    map[string]string
	Timestamp string
}
