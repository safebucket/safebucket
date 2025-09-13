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

# ECS Configuration
variable "safebucket_image" {
  description = "Docker image for SafeBucket application"
  type        = string
  default     = "docker.io/safebucket/safebucket:latest"
}

variable "safebucket_cpu" {
  description = "CPU units for SafeBucket task (1024 = 1 vCPU)"
  type        = number
  default     = 512
}

variable "safebucket_memory" {
  description = "Memory in MB for SafeBucket task"
  type        = number
  default     = 1024
}

variable "safebucket_desired_count" {
  description = "Desired number of SafeBucket tasks"
  type        = number
  default     = 1
}

variable "safebucket_min_capacity" {
  description = "Minimum capacity for SafeBucket auto scaling"
  type        = number
  default     = 1
}

variable "safebucket_max_capacity" {
  description = "Maximum capacity for SafeBucket auto scaling"
  type        = number
  default     = 3
}

variable "loki_image" {
  description = "Docker image for Loki"
  type        = string
  default     = "grafana/loki:3.2.1"
}

variable "loki_cpu" {
  description = "CPU units for Loki task (1024 = 1 vCPU)"
  type        = number
  default     = 512
}

variable "loki_memory" {
  description = "Memory in MB for Loki task"
  type        = number
  default     = 1024
}

variable "mailpit_image" {
  description = "Docker image for Mailpit"
  type        = string
  default     = "axllent/mailpit:v1.27.7"
}

variable "mailpit_cpu" {
  description = "CPU units for Mailpit task (1024 = 1 vCPU)"
  type        = number
  default     = 256
}

variable "mailpit_memory" {
  description = "Memory in MB for Mailpit task"
  type        = number
  default     = 512
}

variable "enable_autoscaling" {
  description = "Enable auto scaling for SafeBucket service"
  type        = bool
  default     = false
}

variable "enable_ecs_exec" {
  description = "Enable ECS Exec for debugging"
  type        = bool
  default     = false
}

variable "log_retention_days" {
  description = "CloudWatch logs retention period in days"
  type        = number
  default     = 7
}

# Application Configuration
variable "jwt_secret" {
  description = "JWT secret for application authentication"
  type        = string
  sensitive   = true
}

variable "admin_password" {
  description = "Admin password for SafeBucket application"
  type        = string
  sensitive   = true
}

variable "smtp_sender" {
  description = "SMTP sender email address"
  type        = string
  default     = "notifications@safebucket.io"
}

variable "admin_email" {
  description = "Admin email address for SafeBucket application"
  type        = string
  default     = "admin@safebucket.io"
}