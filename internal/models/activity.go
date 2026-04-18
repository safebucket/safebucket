package models

import "reflect"

type IActivityLoggable interface {
	ToActivity() interface{}
}

type Activity struct {
	Message string
	Filter  LogFilter
	Object  interface{}
}

type LogFilter struct {
	Fields    ActivityFields
	Timestamp string
}

type ActivityFields struct {
	Action            string `json:"action"              bleve:"keyword" loki:"label"`
	ObjectType        string `json:"object_type"         bleve:"keyword" loki:"label"`
	UserID            string `json:"user_id"             bleve:"keyword"`
	Domain            string `json:"domain"              bleve:"keyword"`
	BucketID          string `json:"bucket_id"           bleve:"keyword"`
	FileID            string `json:"file_id"             bleve:"keyword"`
	FolderID          string `json:"folder_id"           bleve:"keyword"`
	ShareID           string `json:"share_id"            bleve:"keyword"`
	DeviceID          string `json:"device_id"           bleve:"keyword"`
	BucketMemberEmail string `json:"bucket_member_email" bleve:"keyword"`
	ProviderType      string `json:"provider_type"       bleve:"keyword"`
	ProviderName      string `json:"provider_name"       bleve:"keyword"`
	ChallengeID       string `json:"challenge_id"        bleve:"keyword"`
	MFARequired       string `json:"mfa_required"        bleve:"keyword"`
	InviteID          string `json:"invite_id"           bleve:"keyword"`
	AttemptsLeft      string `json:"attempts_left"       bleve:"keyword"`
	SessionID         string `json:"session_id"          bleve:"keyword"`
}

// ToMap converts non-empty fields to a map keyed by their json tag.
func (f ActivityFields) ToMap() map[string]string {
	m := make(map[string]string)
	v := reflect.ValueOf(f)
	t := v.Type()
	for i := range t.NumField() {
		jsonTag := t.Field(i).Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		val := v.Field(i).String()
		if val != "" {
			m[jsonTag] = val
		}
	}
	return m
}
