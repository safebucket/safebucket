# SafeBucket AWS Deployment

This directory contains the AWS infrastructure and deployment configuration for SafeBucket, providing a complete cloud storage solution with S3, RDS PostgreSQL, ElastiCache Redis, SQS messaging, and proper IAM security.

## Architecture Overview

The AWS deployment includes:

- **S3 Bucket** - Primary storage backend with event notifications
- **RDS PostgreSQL** - Database backend with automated backups and monitoring
- **ElastiCache Redis Cluster** - Caching and session storage with user authentication
- **SQS Queues** - Event processing and notifications
- **IAM Roles & Policies** - Secure access control
- **Security Groups** - Network-level security
- **CloudWatch Logging** - Monitoring and observability

## Quick Start

### Prerequisites

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [AWS CLI](https://aws.amazon.com/cli/) configured with appropriate permissions
- AWS account with necessary service limits

### 1. Configure Infrastructure

```bash
cd terraform

# Copy and customize the configuration
cp terraform.tfvars.example terraform.tfvars
vim terraform.tfvars
```

### 2. Deploy Infrastructure

```bash
# Initialize Terraform
terraform init

# Review the planned changes
terraform plan

# Apply the configuration
terraform apply
```

### 3. Get Configuration Values

```bash
# View all outputs
terraform output

# Get specific values
terraform output rds_endpoint
terraform output redis_endpoint
terraform output s3_bucket_name
```

## Infrastructure Components

### S3 Storage
- **Bucket**: Primary object storage with AES256 server-side encryption
- **Security**: Public access blocked, CORS configured for web application
- **Events**: Automatic notifications to SQS for object create/delete operations
- **Network**: Private access through application IAM role

### RDS PostgreSQL Database
- **Engine**: PostgreSQL 15.4 with automated backups
- **Storage**: GP3 storage with encryption at rest, auto-scaling enabled
- **Monitoring**: Enhanced monitoring with 60-second intervals, Performance Insights enabled
- **Security**: Private subnets only, VPC security groups
- **Parameters**: Optimized configuration with statement logging and pg_stat_statements

### ElastiCache Redis
- **Type**: Redis 7 cluster with user-based authentication
- **Security**: Private subnets, encrypted at rest and in transit
- **Monitoring**: CloudWatch slow query logging
- **Backup**: Automated snapshots with configurable retention
- **Parameters**: LRU eviction policy, optimized for caching

### Message Queues
- **S3 Events Queue**: Processes S3 object create/delete events
- **Notifications Queue**: General application notifications
- **Configuration**: 14-day message retention, long polling enabled
- **Security**: Resource-specific access policies

### Security & Networking
- **VPC**: Uses default VPC with all available subnets
- **Security Groups**: Minimal access - PostgreSQL (5432), Redis (6379)
- **IAM**: Least-privilege application role for EC2 and ECS
- **Encryption**: All data encrypted at rest and in transit

## Configuration Variables

### Required Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `s3_bucket_name` | S3 bucket name (globally unique) | `safebucket-storage-prod-xyz` |
| `s3_event_queue_name` | SQS queue for S3 events | `safebucket-s3-events-prod` |
| `notification_queue_name` | SQS queue for notifications | `safebucket-notifications-prod` |
| `redis_auth_token` | Redis authentication password (32+ chars) | `secure-password-here` |
| `rds_password` | PostgreSQL database password | `secure-db-password` |

### Optional Variables

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `project_name` | string | `safebucket` | Project identifier for resource naming |
| `environment` | string | `dev` | Environment name (dev, staging, prod) |
| `s3_cors_allowed_origins` | list(string) | `["http://localhost:3000"]` | CORS allowed origins for S3 |
| `redis_node_type` | string | `cache.t3.micro` | ElastiCache instance type |
| `redis_num_cache_nodes` | number | `1` | Number of cache nodes |
| `redis_auth_token_enabled` | bool | `true` | Enable Redis authentication |
| `redis_snapshot_retention_limit` | number | `5` | Backup retention days |
| `redis_snapshot_window` | string | `03:00-05:00` | Daily backup window (UTC) |
| `redis_maintenance_window` | string | `sun:05:00-sun:07:00` | Weekly maintenance window |
| `redis_log_retention_days` | number | `7` | CloudWatch log retention days |
| `rds_instance_class` | string | `db.t3.micro` | RDS instance type |
| `rds_allocated_storage` | number | `20` | Initial storage size (GB) |
| `rds_max_allocated_storage` | number | `100` | Maximum auto-scaling storage (GB) |
| `rds_database_name` | string | `safebucket` | Database name |
| `rds_username` | string | `safebucket` | Database username |
| `rds_backup_retention_period` | number | `7` | Backup retention days |
| `rds_backup_window` | string | `03:00-04:00` | Daily backup window (UTC) |
| `rds_maintenance_window` | string | `sun:04:00-sun:05:00` | Weekly maintenance window |
| `rds_deletion_protection` | bool | `false` | Enable deletion protection |
| `rds_skip_final_snapshot` | bool | `true` | Skip final snapshot on deletion |
| `rds_storage_encrypted` | bool | `true` | Enable storage encryption |

### Database Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `rds_instance_class` | `db.t3.micro` | RDS instance type |
| `rds_allocated_storage` | `20` | Initial storage in GB |
| `rds_max_allocated_storage` | `100` | Maximum auto-scaling storage |
| `rds_database_name` | `safebucket` | Database name |
| `rds_username` | `safebucket` | Database username |
| `rds_backup_retention_period` | `7` | Backup retention days |
| `rds_deletion_protection` | `false` | Enable deletion protection |

### Redis Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `redis_node_type` | `cache.t3.micro` | ElastiCache instance type |
| `redis_num_cache_nodes` | `1` | Number of cache nodes |
| `redis_auth_token_enabled` | `true` | Enable Redis authentication |
| `redis_snapshot_retention_limit` | `5` | Backup retention days |
| `redis_maintenance_window` | `sun:05:00-sun:07:00` | Maintenance window |
| `redis_log_retention_days` | `7` | CloudWatch log retention |

### Environment-Specific Examples

**Development:**
```hcl
project_name = "safebucket"
environment = "dev"
rds_instance_class = "db.t3.micro"
redis_node_type = "cache.t3.micro"
redis_log_retention_days = 3
rds_backup_retention_period = 1
```

**Production:**
```hcl
project_name = "safebucket"
environment = "prod"
rds_instance_class = "db.t3.medium"
redis_node_type = "cache.t3.small"
redis_log_retention_days = 30
rds_backup_retention_period = 30
rds_deletion_protection = true
```

## Outputs

The Terraform configuration provides these key outputs:

### Storage
- `s3_bucket_name` - S3 bucket name
- `s3_bucket_arn` - S3 bucket ARN
- `s3_bucket_domain_name` - S3 bucket domain

### Database
- `rds_endpoint` - PostgreSQL connection endpoint
- `rds_port` - Database port (5432)
- `rds_database_name` - Database name
- `rds_instance_id` - RDS instance identifier

### Cache
- `redis_endpoint` - Redis cluster endpoint
- `redis_port` - Redis port (6379)
- `redis_user_group_id` - User group for authentication
- `redis_app_user_id` - Application user ID

### Messaging
- `sqs_s3_events_queue_url` - S3 events SQS queue URL
- `sqs_notifications_queue_url` - Notifications SQS queue URL

### Security
- `iam_role_arn` - Application IAM role ARN
- `instance_profile_name` - EC2 instance profile name
- `redis_security_group_id` - Redis security group ID
- `rds_security_group_id` - RDS security group ID

### Summary
- `infrastructure_summary` - Complete infrastructure overview

## Application Integration

After deploying the infrastructure, configure SafeBucket with the output values:

```bash
# Set environment variables for SafeBucket
export AWS_REGION="eu-west-1"
export AWS_S3_BUCKET_NAME=$(terraform output -raw s3_bucket_name)

# Database configuration
export DATABASE_HOST=$(terraform output -raw rds_endpoint | cut -d: -f1)
export DATABASE_PORT=$(terraform output -raw rds_port)
export DATABASE_NAME=$(terraform output -raw rds_database_name)
export DATABASE_USER="safebucket"
export DATABASE_PASSWORD="your-rds-password"

# Redis configuration
export REDIS_HOST=$(terraform output -raw redis_endpoint)
export REDIS_PORT=$(terraform output -raw redis_port)
export REDIS_PASSWORD="your-redis-auth-token"

# SQS configuration
export AWS_SQS_S3_EVENTS_URL=$(terraform output -raw sqs_s3_events_queue_url)
export AWS_SQS_NOTIFICATIONS_URL=$(terraform output -raw sqs_notifications_queue_url)
```

## Security Considerations

### Network Security
- All database and cache resources in private subnets (default VPC)
- Security groups restrict access to VPC CIDR blocks only
- No public internet access to RDS or ElastiCache
- Application communicates through IAM roles, not access keys

### Data Protection
- S3 server-side encryption with AES256
- RDS storage encryption enabled by default
- ElastiCache encryption at rest and in transit
- S3 public access blocked at bucket level

### Access Control
- IAM roles with minimal required permissions
- Redis user-based authentication with secure passwords
- Resource-specific access policies for SQS queues
- Support for both EC2 and ECS deployment models

### Authentication & Authorization
- Redis user group with application-specific user
- Database access through username/password
- S3 access through IAM role permissions
- SQS queue policies restrict access to specific resources

## Monitoring and Maintenance

### Automated Backups
- RDS automated backups with 7-day retention (configurable)
- Redis automated snapshots with 5-day retention (configurable)
- S3 versioning enabled for object recovery
- Configurable backup and maintenance windows

### CloudWatch Integration
- Redis slow query logs with configurable retention
- RDS enhanced monitoring with 60-second intervals
- RDS Performance Insights enabled (7-day retention)
- SQS queue metrics and alarms available

### Maintenance Windows
- RDS maintenance: Sunday 4-5 AM UTC (configurable)
- Redis maintenance: Sunday 5-7 AM UTC (configurable)
- Redis snapshot window: 3-5 AM UTC (configurable)
- RDS backup window: 3-4 AM UTC (configurable)

## Cost Optimization

### Development Environment
```hcl
# Minimal resources for development
rds_instance_class = "db.t3.micro"
redis_node_type = "cache.t3.micro"
redis_log_retention_days = 3
rds_backup_retention_period = 1
rds_deletion_protection = false
```

### Production Environment
```hcl
# Scaled resources for production
rds_instance_class = "db.t3.medium"      # or larger based on load
redis_node_type = "cache.t3.small"       # or larger based on load
redis_log_retention_days = 30
rds_backup_retention_period = 30
rds_deletion_protection = true
```

## Troubleshooting

### Common Issues

1. **S3 Bucket Already Exists**
   ```
   Error: BucketAlreadyExists: The requested bucket name is not available
   ```
   Solution: Choose a globally unique bucket name in `terraform.tfvars`

2. **Redis Authentication Issues**
   ```
   Error: Authentication failed
   ```
   Solution: Ensure `redis_auth_token` is set and at least 32 characters

3. **RDS Connection Issues**
   ```
   Error: could not connect to server
   ```
   Solution: Verify security group allows access from your application subnet

4. **Parameter Group Family Error**
   ```
   Error: InvalidParameterValue: CacheParameterGroupFamily redis7.x is not valid
   ```
   Solution: Already fixed - using `redis7` family

5. **VPC Subnet Issues**
   ```
   Error: DB Subnet Group doesn't meet availability zone coverage requirement
   ```
   Solution: Ensure your default VPC has subnets in multiple AZs

### State Management

For production deployments, use remote state storage:

```hcl
terraform {
  backend "s3" {
    bucket = "your-terraform-state-bucket"
    key    = "safebucket/terraform.tfstate"
    region = "eu-west-1"
  }
}
```

## Cleanup

To destroy all infrastructure:

```bash
cd terraform
terraform destroy
```

**⚠️ Warning:** This permanently deletes all data including:
- S3 bucket contents
- PostgreSQL database and all data
- Redis cache data
- All backups and snapshots

Ensure you have backups if needed before destroying infrastructure.

## Support

- Review the [main SafeBucket documentation](../../README.md)
- Check AWS service limits and quotas
- Verify IAM permissions for Terraform operations
- For Terraform-specific issues, see the [terraform/README.md](terraform/README.md)
- Review the [terraform.tfvars.example](terraform/terraform.tfvars.example) for configuration examples
