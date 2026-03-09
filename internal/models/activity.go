package models

type IActivityLoggable interface {
	ToActivity() interface{}
}

type Activity struct {
	Message string
	Filter  LogFilter
	Object  interface{}
}

type LogFilter struct {
	Fields    map[string]string
	Timestamp string
}
