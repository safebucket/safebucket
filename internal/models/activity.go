package models

type Activity struct {
	Message string
	Filter  LogFilter
	Object  interface{}
}

type LogFilter struct {
	Fields    map[string]string
	Timestamp string
}
