# SafeBucket AWS Deployment

This directory contains the AWS infrastructure and deployment configuration for SafeBucket, providing a complete cloud storage solution with S3, ElastiCache Redis, SQS messaging, and proper IAM security.

## Architecture Overview

The AWS deployment includes:

- **S3 Bucket** - Primary storage backend with event notifications
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
terraform output redis_endpoint
terraform output s3_bucket_name
```

## Infrastructure Components

### S3 Storage
- **Bucket**: Primary object storage with versioning
- **Encryption**: Server-side encryption enabled
- **Events**: Automatic notifications to SQS on object operations
- **CORS**: Configured for web application access

### Redis Cache
- **Type**: ElastiCache Redis cluster
- **Authentication**: User-based with secure password
- **Encryption**: At-rest and in-transit encryption
- **Backup**: Automated snapshots with configurable retention
- **Monitoring**: CloudWatch logging for slow queries

### Message Queues
- **S3 Events Queue**: Processes S3 object create/delete events
- **Notifications Queue**: General application notifications
- **Dead Letter Queues**: Error handling and retry logic

### Security
- **IAM Roles**: Least-privilege access for application components
- **Security Groups**: Network-level access controls
- **Encryption**: All data encrypted at rest and in transit
- **Network**: VPC-isolated resources

## Configuration Variables

### Required Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `project_name` | Project identifier | `safebucket` |
| `environment` | Environment name | `dev`, `staging`, `prod` |
| `s3_bucket_name` | S3 bucket name (globally unique) | `safebucket-storage-prod-xyz` |
| `redis_auth_token` | Redis authentication password | `secure-32-char-password` |

### Redis Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `redis_node_type` | `cache.t3.micro` | ElastiCache instance type |
| `redis_num_cache_nodes` | `1` | Number of cache nodes |
| `redis_auth_token_enabled` | `true` | Enable Redis authentication |
| `redis_snapshot_retention_limit` | `5` | Backup retention days |
| `redis_maintenance_window` | `sun:05:00-sun:07:00` | Maintenance window |

### Environment-Specific Configurations

**Development:**
```hcl
environment = "dev"
redis_node_type = "cache.t3.micro"
redis_num_cache_nodes = 1
redis_log_retention_days = 3
```

**Production:**
```hcl
environment = "prod"
redis_node_type = "cache.t3.small"
redis_num_cache_nodes = 2
redis_log_retention_days = 30
```

## Outputs

The Terraform configuration provides these key outputs:

### Storage
- `s3_bucket_name` - S3 bucket name
- `s3_bucket_arn` - S3 bucket ARN

### Cache
- `redis_endpoint` - Redis cluster endpoint
- `redis_port` - Redis port (6379)
- `redis_user_group_id` - User group for authentication

### Messaging
- `s3_events_queue_url` - S3 events SQS queue URL
- `notifications_queue_url` - Notifications SQS queue URL

### Security
- `iam_role_arn` - Application IAM role ARN
- `redis_security_group_id` - Redis security group ID

## Application Integration

After deploying the infrastructure, configure SafeBucket with the output values:

```bash
# Set environment variables
export AWS_REGION="us-east-1"
export AWS_S3_BUCKET_NAME=$(terraform output -raw s3_bucket_name)
export REDIS_HOST=$(terraform output -raw redis_endpoint)
export REDIS_PORT=$(terraform output -raw redis_port)
export REDIS_PASSWORD="your-redis-auth-token"
export SQS_S3_EVENTS_URL=$(terraform output -raw s3_events_queue_url)
export SQS_NOTIFICATIONS_URL=$(terraform output -raw notifications_queue_url)
```

## Security Considerations

### Network Security
- Redis cluster in private subnets (using default VPC)
- Security groups restrict access to necessary ports only
- No public internet access to cache resources

### Data Protection
- S3 server-side encryption with AWS managed keys
- Redis encryption at rest and in transit
- S3 public access blocked by default

### Access Control
- IAM roles with minimal required permissions
- Redis user-based authentication
- Resource-specific access policies

## Monitoring and Maintenance

### CloudWatch Integration
- Redis slow query logs
- S3 access logs
- SQS queue metrics

### Backup Strategy
- Redis automated snapshots (configurable retention)
- S3 versioning enabled
- Cross-AZ redundancy where applicable

### Maintenance Windows
- Redis maintenance: Sunday 5-7 AM UTC (configurable)
- Snapshot window: 3-5 AM UTC (configurable)

## Cost Optimization

### Development Environment
- Use `cache.t3.micro` for Redis
- Single cache node
- Shorter log retention (3-7 days)
- Minimal backup retention

### Production Environment
- Scale Redis instance type based on load
- Multiple cache nodes for high availability
- Extended log and backup retention
- Enable monitoring and alerting

## Troubleshooting

### Common Issues

1. **S3 Bucket Already Exists**
   ```
   Error: BucketAlreadyExists: The requested bucket name is not available
   ```
   Solution: Choose a globally unique bucket name

2. **Redis Authentication Issues**
   ```
   Error: Authentication failed
   ```
   Solution: Ensure `redis_auth_token` is set and at least 32 characters

3. **Parameter Group Family Error**
   ```
   Error: InvalidParameterValue: CacheParameterGroupFamily redis7.x is not valid
   ```
   Solution: Update to supported family (already fixed to `redis7`)

### State Management

For production, use remote state storage:

```hcl
terraform {
  backend "s3" {
    bucket = "your-terraform-state-bucket"
    key    = "safebucket/terraform.tfstate"
    region = "us-east-1"
  }
}
```

## Cleanup

To destroy all infrastructure:

```bash
cd terraform
terraform destroy
```

**⚠️ Warning:** This permanently deletes all data including S3 bucket contents. Ensure you have backups if needed.

## Support

- Review the [main SafeBucket documentation](../../README.md)
- Check AWS service limits and quotas
- Verify IAM permissions for Terraform operations
- For Terraform-specific issues, see the [terraform/README.md](terraform/README.md)