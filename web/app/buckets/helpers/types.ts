export interface Bucket {
  id: string;
  name: string;
  files: object[];
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface IBucketsData {
  buckets: Bucket[];
  error: string;
  isLoading: boolean;
}
