# SafeBucket AWS Infrastructure Outputs - Minimal Setup

# S3 Outputs
output "s3_bucket_name" {
  description = "Name of the S3 bucket"
  value       = aws_s3_bucket.main.bucket
}

output "s3_bucket_arn" {
  description = "ARN of the S3 bucket"
  value       = aws_s3_bucket.main.arn
}

output "s3_bucket_domain_name" {
  description = "Domain name of the S3 bucket"
  value       = aws_s3_bucket.main.bucket_domain_name
}

output "s3_loki_bucket_name" {
  description = "Name of the Loki S3 bucket"
  value       = aws_s3_bucket.loki.bucket
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

output "ecs_execution_role_arn" {
  description = "ARN of the ECS execution role"
  value       = aws_iam_role.ecs_execution_role.arn
}

output "ecs_task_role_arn" {
  description = "ARN of the ECS task role"
  value       = aws_iam_role.ecs_task_role.arn
}

# Redis Outputs
output "redis_endpoint" {
  description = "Redis cluster endpoint"
  value       = aws_elasticache_replication_group.main.primary_endpoint_address
}

output "redis_port" {
  description = "Port number for the Redis cluster"
  value       = 6379
}

output "redis_auth_token_enabled" {
  description = "Whether Redis AUTH token is enabled"
  value       = var.redis_auth_token_enabled
}

output "redis_security_group_id" {
  description = "Security group ID for the Redis cluster"
  value       = aws_security_group.redis.id
}

# RDS Outputs
output "rds_endpoint" {
  description = "RDS PostgreSQL endpoint"
  value       = aws_db_instance.main.endpoint
}

output "rds_port" {
  description = "RDS PostgreSQL port"
  value       = aws_db_instance.main.port
}

output "rds_database_name" {
  description = "RDS database name"
  value       = aws_db_instance.main.db_name
}

output "rds_username" {
  description = "RDS master username"
  value       = aws_db_instance.main.username
  sensitive   = true
}

output "rds_security_group_id" {
  description = "Security group ID for the RDS instance"
  value       = aws_security_group.rds.id
}

output "rds_instance_id" {
  description = "RDS instance identifier"
  value       = aws_db_instance.main.id
}

# ECS Outputs
output "ecs_cluster_name" {
  description = "Name of the ECS cluster"
  value       = aws_ecs_cluster.safebucket_cluster.name
}

output "ecs_cluster_arn" {
  description = "ARN of the ECS cluster"
  value       = aws_ecs_cluster.safebucket_cluster.arn
}

output "alb_dns_name" {
  description = "DNS name of the Application Load Balancer"
  value       = aws_lb.safebucket_alb.dns_name
}

output "internal_alb_dns_name" {
  description = "DNS name of the internal Application Load Balancer"
  value       = aws_lb.internal_alb.dns_name
}

output "safebucket_service_name" {
  description = "Name of the SafeBucket ECS service"
  value       = aws_ecs_service.safebucket.name
}

output "loki_service_name" {
  description = "Name of the Loki ECS service"
  value       = aws_ecs_service.loki.name
}

output "mailpit_service_name" {
  description = "Name of the Mailpit ECS service"
  value       = aws_ecs_service.mailpit.name
}


# Configuration Summary
output "infrastructure_summary" {
  description = "Summary of created infrastructure"
  value = {
    s3_bucket              = aws_s3_bucket.main.bucket
    s3_loki_bucket         = aws_s3_bucket.loki.bucket
    s3_events_queue        = aws_sqs_queue.s3_events.name
    notifications_queue    = aws_sqs_queue.notifications.name
    iam_role              = aws_iam_role.safebucket_app.name
    redis_endpoint         = aws_elasticache_replication_group.main.primary_endpoint_address
    redis_port             = 6379
    rds_endpoint           = aws_db_instance.main.endpoint
    ecs_cluster            = aws_ecs_cluster.safebucket_cluster.name
    alb_dns_name           = aws_lb.safebucket_alb.dns_name
    internal_alb_dns_name  = aws_lb.internal_alb.dns_name
    environment           = var.environment
    project_name          = var.project_name
    region                = data.aws_region.current.name
  }
}