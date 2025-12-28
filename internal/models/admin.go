package models

// AdminStatsQueryParams represents query parameters for admin stats endpoint.
type AdminStatsQueryParams struct {
	Days int `json:"days" validate:"omitempty,oneof=30 90 180"`
}

// AdminStatsResponse contains platform-wide statistics for admin dashboard.
type AdminStatsResponse struct {
	TotalUsers           int64             `json:"total_users"`
	TotalBuckets         int64             `json:"total_buckets"`
	TotalFiles           int64             `json:"total_files"`
	TotalFolders         int64             `json:"total_folders"`
	TotalStorageBytes    int64             `json:"total_storage"`
	RoleDistribution     []RoleCount       `json:"role_distribution"`
	ProviderDistribution []ProviderCount   `json:"provider_distribution"`
	SharedFiles          []TimeSeriesPoint `json:"shared_files"`
}

// RoleCount represents the count of users per role.
type RoleCount struct {
	Role  string `json:"role"`
	Count int64  `json:"count"`
}

// ProviderCount represents the count of users per auth provider.
type ProviderCount struct {
	Provider string `json:"provider"`
	Count    int64  `json:"count"`
}

// TimeSeriesPoint represents a data point in a time series chart.
type TimeSeriesPoint struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}
