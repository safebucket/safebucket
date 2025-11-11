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
  description = "List of allowed origins for S3 CORS configuration. If empty, will default to ALB DNS and localhost"
  type        = list(string)
  default = []
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

variable "object_deletion_queue_name" {
  description = "Name of the SQS queue for object deletion events (trash expiration, folder restore)"
  type        = string
}

# Redis Configuration
variable "redis_node_type" {
  description = "The instance type for the Redis cache nodes"
  type        = string
  default = "cache.t4g.micro"
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
  default = "db.t4g.micro"
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
  default = 256
}

variable "safebucket_memory" {
  description = "Memory in MB for SafeBucket task"
  type        = number
  default = 512
}

variable "safebucket_architecture" {
  description = "CPU architecture for SafeBucket task (X86_64 or ARM64). ARM64 provides better price/performance."
  type        = string
  default     = "ARM64"
  validation {
    condition = contains(["X86_64", "ARM64"], var.safebucket_architecture)
    error_message = "Architecture must be either X86_64 or ARM64"
  }
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

variable "enable_loki_spot_instances" {
  description = "Enable Fargate Spot instances for Loki service (up to 70% cost savings)"
  type        = bool
  default     = false
}

variable "loki_spot_instance_percentage" {
  description = "Percentage of Loki tasks to run on Spot instances (0-100)"
  type        = number
  default     = 100
  validation {
    condition     = var.loki_spot_instance_percentage >= 0 && var.loki_spot_instance_percentage <= 100
    error_message = "Loki spot instance percentage must be between 0 and 100"
  }
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

variable "enable_mailpit_spot_instances" {
  description = "Enable Fargate Spot instances for Mailpit service (up to 70% cost savings)"
  type        = bool
  default     = false
}

variable "mailpit_spot_instance_percentage" {
  description = "Percentage of Mailpit tasks to run on Spot instances (0-100)"
  type        = number
  default     = 100
  validation {
    condition     = var.mailpit_spot_instance_percentage >= 0 && var.mailpit_spot_instance_percentage <= 100
    error_message = "Mailpit spot instance percentage must be between 0 and 100"
  }
}

variable "enable_autoscaling" {
  description = "Enable auto scaling for SafeBucket service"
  type        = bool
  default     = false
}

variable "enable_spot_instances" {
  description = "Enable Fargate Spot instances for SafeBucket service (up to 70% cost savings, but can be interrupted)"
  type        = bool
  default     = false
}

variable "spot_instance_percentage" {
  description = "Percentage of tasks to run on Spot instances (0-100). Only used if enable_spot_instances is true. 100 = all spot, 0 = all on-demand"
  type        = number
  default     = 100
  validation {
    condition     = var.spot_instance_percentage >= 0 && var.spot_instance_percentage <= 100
    error_message = "Spot instance percentage must be between 0 and 100"
  }
}

variable "enable_ecs_exec" {
  description = "Enable ECS Exec for debugging"
  type        = bool
  default     = false
}

variable "redeployment_trigger" {
  description = "Change this value to force ECS service redeployment (e.g., when Docker image with same tag is updated)"
  type        = string
  default     = "1"
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