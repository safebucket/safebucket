# SafeBucket AWS Infrastructure Variables - Minimal Setup

variable "project_name" {
  description = "Name of the project, used for resource naming"
  type        = string
  default     = "safebucket"
}

variable "environment" {
  description = "Environment name (dev, staging, prod)"
  type        = string
  default     = "dev"
}

# S3 Configuration
variable "s3_bucket_name" {
  description = "Name of the S3 bucket for storage"
  type        = string
}

variable "s3_cors_allowed_origins" {
  description = "List of allowed origins for S3 CORS configuration"
  type        = list(string)
  default     = ["http://localhost:3000"]
}

# SQS Configuration
variable "s3_event_queue_name" {
  description = "Name of the SQS queue for S3 events"
  type        = string
}

variable "notification_queue_name" {
  description = "Name of the SQS queue for application notifications"
  type        = string
}

# Redis Configuration
variable "redis_node_type" {
  description = "The instance type for the Redis cache nodes"
  type        = string
  default     = "cache.t3.micro"
}

variable "redis_num_cache_nodes" {
  description = "Number of cache nodes in the Redis cluster"
  type        = number
  default     = 1
}

variable "redis_auth_token_enabled" {
  description = "Enable Redis AUTH token"
  type        = bool
  default     = true
}

variable "redis_auth_token" {
  description = "Redis AUTH token (must be at least 32 characters if enabled)"
  type        = string
  default     = null
  sensitive   = true
}

variable "redis_snapshot_retention_limit" {
  description = "Number of days to retain automatic snapshots"
  type        = number
  default     = 5
}

variable "redis_snapshot_window" {
  description = "Daily time range for automatic snapshots (UTC)"
  type        = string
  default     = "03:00-05:00"
}

variable "redis_maintenance_window" {
  description = "Weekly time range for maintenance (UTC)"
  type        = string
  default     = "sun:05:00-sun:07:00"
}

variable "redis_log_retention_days" {
  description = "Number of days to retain CloudWatch logs"
  type        = number
  default     = 7
}

# RDS Configuration
variable "rds_instance_class" {
  description = "The instance class for the RDS PostgreSQL database"
  type        = string
  default     = "db.t3.micro"
}

variable "rds_allocated_storage" {
  description = "The allocated storage in gigabytes for the RDS instance"
  type        = number
  default     = 20
}

variable "rds_max_allocated_storage" {
  description = "The upper limit to which Amazon RDS can automatically scale the storage"
  type        = number
  default     = 100
}

variable "rds_database_name" {
  description = "The name of the database to create"
  type        = string
  default     = "safebucket"
}

variable "rds_username" {
  description = "Username for the RDS instance"
  type        = string
  default     = "safebucket"
}

variable "rds_password" {
  description = "Password for the RDS instance"
  type        = string
  sensitive   = true
}

variable "rds_backup_retention_period" {
  description = "The days to retain backups for"
  type        = number
  default     = 7
}

variable "rds_backup_window" {
  description = "The daily time range for automated backups (UTC)"
  type        = string
  default     = "03:00-04:00"
}

variable "rds_maintenance_window" {
  description = "The weekly time range for maintenance (UTC)"
  type        = string
  default     = "sun:04:00-sun:05:00"
}

variable "rds_deletion_protection" {
  description = "If the DB instance should have deletion protection enabled"
  type        = bool
  default     = false
}

variable "rds_skip_final_snapshot" {
  description = "Determines whether a final DB snapshot is created before the DB instance is deleted"
  type        = bool
  default     = true
}

variable "rds_storage_encrypted" {
  description = "Specifies whether the DB instance is encrypted"
  type        = bool
  default     = true
}