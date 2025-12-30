export interface CreateUserPayload {
  first_name: string;
  last_name: string;
  email: string;
  password: string;
}

export interface TimeSeriesPoint {
  date: string;
  count: number;
}

export interface AdminStatsResponse {
  total_users: number;
  total_buckets: number;
  total_files: number;
  total_folders: number;
  total_storage: number;
  shared_files_per_day: Array<TimeSeriesPoint>;
}

export interface IAdminBucket {
  id: string;
  name: string;
  created_at: string;
  updated_at: string;
  creator: {
    id: string;
    first_name: string;
    last_name: string;
    email: string;
  };
  member_count: number;
  file_count: number;
  size: number;
}
