# SafeBucket AWS Infrastructure Outputs - Minimal Setup

# S3 Outputs
output "s3_bucket_name" {
  description = "Name of the S3 bucket"
  value       = aws_s3_bucket.storage.bucket
}

output "s3_bucket_arn" {
  description = "ARN of the S3 bucket"
  value       = aws_s3_bucket.storage.arn
}

output "s3_bucket_domain_name" {
  description = "Domain name of the S3 bucket"
  value       = aws_s3_bucket.storage.bucket_domain_name
}

# SQS Outputs
output "sqs_s3_events_queue_url" {
  description = "URL of the S3 events SQS queue"
  value       = aws_sqs_queue.s3_events.url
}

output "sqs_s3_events_queue_arn" {
  description = "ARN of the S3 events SQS queue"
  value       = aws_sqs_queue.s3_events.arn
}

output "sqs_s3_events_queue_name" {
  description = "Name of the S3 events SQS queue"
  value       = aws_sqs_queue.s3_events.name
}

output "sqs_notifications_queue_url" {
  description = "URL of the notifications SQS queue"
  value       = aws_sqs_queue.notifications.url
}

output "sqs_notifications_queue_arn" {
  description = "ARN of the notifications SQS queue"
  value       = aws_sqs_queue.notifications.arn
}

output "sqs_notifications_queue_name" {
  description = "Name of the notifications SQS queue"
  value       = aws_sqs_queue.notifications.name
}

# IAM Outputs
output "iam_role_arn" {
  description = "ARN of the SafeBucket application IAM role"
  value       = aws_iam_role.safebucket_app.arn
}

output "iam_role_name" {
  description = "Name of the SafeBucket application IAM role"
  value       = aws_iam_role.safebucket_app.name
}

output "instance_profile_name" {
  description = "Name of the instance profile for EC2"
  value       = aws_iam_instance_profile.safebucket_app.name
}

# Redis Outputs
output "redis_endpoint" {
  description = "Redis cluster endpoint"
  value       = aws_elasticache_cluster.redis.cache_nodes[0].address
}

output "redis_port" {
  description = "Port number for the Redis cluster"
  value       = aws_elasticache_cluster.redis.port
}

output "redis_auth_token_enabled" {
  description = "Whether Redis AUTH token is enabled"
  value       = var.redis_auth_token_enabled
}

output "redis_security_group_id" {
  description = "Security group ID for the Redis cluster"
  value       = aws_security_group.redis.id
}

output "redis_user_group_id" {
  description = "User group ID for Redis authentication"
  value       = aws_elasticache_user_group.redis.user_group_id
}

output "redis_app_user_id" {
  description = "Application user ID for Redis authentication"
  value       = aws_elasticache_user.redis_app_user.user_id
}

# RDS Outputs
output "rds_endpoint" {
  description = "RDS PostgreSQL endpoint"
  value       = aws_db_instance.postgres.endpoint
}

output "rds_port" {
  description = "RDS PostgreSQL port"
  value       = aws_db_instance.postgres.port
}

output "rds_database_name" {
  description = "RDS database name"
  value       = aws_db_instance.postgres.db_name
}

output "rds_username" {
  description = "RDS master username"
  value       = aws_db_instance.postgres.username
  sensitive   = true
}

output "rds_security_group_id" {
  description = "Security group ID for the RDS instance"
  value       = aws_security_group.rds.id
}

output "rds_instance_id" {
  description = "RDS instance identifier"
  value       = aws_db_instance.postgres.id
}

# Configuration Summary
output "infrastructure_summary" {
  description = "Summary of created infrastructure"
  value = {
    s3_bucket              = aws_s3_bucket.storage.bucket
    s3_events_queue        = aws_sqs_queue.s3_events.name
    notifications_queue    = aws_sqs_queue.notifications.name
    iam_role              = aws_iam_role.safebucket_app.name
    redis_endpoint         = aws_elasticache_cluster.redis.cache_nodes[0].address
    redis_port             = aws_elasticache_cluster.redis.port
    environment           = var.environment
    project_name          = var.project_name
    region                = "eu-west-1"
  }
}