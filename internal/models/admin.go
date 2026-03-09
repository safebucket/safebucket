package models

type AdminStatsQueryParams struct {
	Days int `json:"days" validate:"omitempty,oneof=30 90 180"`
}

type AdminStatsResponse struct {
	TotalUsers        int64             `json:"total_users"`
	TotalBuckets      int64             `json:"total_buckets"`
	TotalFiles        int64             `json:"total_files"`
	TotalFolders      int64             `json:"total_folders"`
	TotalStorageBytes int64             `json:"total_storage"`
	SharedFilesPerDay []TimeSeriesPoint `json:"shared_files_per_day"`
}

type TimeSeriesPoint struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}
