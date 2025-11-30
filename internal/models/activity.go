package models

// IActivityLoggable defines the interface for models that can be logged to activity logs.
// Implementations should return a minimal struct containing only safe fields for logging.
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
