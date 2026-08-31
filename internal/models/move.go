package models

import "github.com/google/uuid"

const (
	MaxMoveItems = 100
)

type MoveBody struct {
	FileIDs             uuid.UUIDs `json:"file_ids"              validate:"unique,max=100"`
	FolderIDs           uuid.UUIDs `json:"folder_ids"            validate:"unique,max=100"`
	DestinationFolderID OptionalID `json:"destination_folder_id"`
}

type MoveResponse struct {
	MovedFiles     int `json:"moved_files"`
	MovedFolders   int `json:"moved_folders"`
	UnchangedItems int `json:"unchanged_items"`
}
