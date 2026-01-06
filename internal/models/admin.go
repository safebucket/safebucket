package models

// AdminStatsQueryParams represents query parameters for admin stats endpoint.
type AdminStatsQueryParams struct {
	Days int `json:"days" validate:"omitempty,oneof=30 90 180"`
}

// AdminStatsResponse contains platform-wide statistics for admin dashboard.
type AdminStatsResponse struct {
	TotalUsers        int64             `json:"total_users"`
	TotalBuckets      int64             `json:"total_buckets"`
	TotalFiles        int64             `json:"total_files"`
	TotalFolders      int64             `json:"total_folders"`
	TotalStorageBytes int64             `json:"total_storage"`
	SharedFilesPerDay []TimeSeriesPoint `json:"shared_files_per_day"`
}

// TimeSeriesPoint represents a data point in a time series chart.
type TimeSeriesPoint struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}
