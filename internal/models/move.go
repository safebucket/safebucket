package models

import "github.com/google/uuid"

type MoveBody struct {
	FileIDs             uuid.UUIDs `json:"file_ids"              validate:"omitempty,max=100,unique,dive,uuid"`
	FolderIDs           uuid.UUIDs `json:"folder_ids"            validate:"omitempty,max=100,unique,dive,uuid"`
	DestinationFolderID OptionalID `json:"destination_folder_id"`
}

type MoveResponse struct {
	MovedFiles     int `json:"moved_files"`
	MovedFolders   int `json:"moved_folders"`
	UnchangedItems int `json:"unchanged_items"`
}
