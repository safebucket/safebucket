# SafeBucket AWS Terraform Module

This Terraform module provisions all the necessary AWS infrastructure for SafeBucket deployment, including S3 storage, RDS PostgreSQL, ElastiCache Redis, SQS queues, and S3 event notifications.

## Features

- ✅ **Complete Infrastructure**: VPC, subnets, security groups, and all AWS services
- ✅ **S3 Event Notifications**: Automatic S3 events to SQS for object create/delete
- ✅ **Security**: Encrypted storage, private subnets, least-privilege IAM
- ✅ **Scalability**: Auto-scaling storage, configurable instance sizes
- ✅ **Monitoring**: CloudWatch integration, enhanced monitoring options
- ✅ **Backup**: Automated RDS backups with configurable retention

## Quick Start

### 1. Prerequisites

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [AWS CLI](https://aws.amazon.com/cli/) configured with appropriate permissions
- AWS account with necessary service limits

### 2. Configure Variables

```bash
# Copy the example variables file
cp terraform.tfvars.example terraform.tfvars

# Edit the variables file with your configuration
nano terraform.tfvars
```

### 3. Deploy Infrastructure

```bash
# Initialize Terraform
terraform init

# Review the planned changes
terraform plan

# Apply the configuration
terraform apply
```

### 4. Get Configuration Values

```bash
# Output all environment variables for SafeBucket
terraform output environment_variables

# Get specific outputs
terraform output database_endpoint
terraform output s3_bucket_name
terraform output redis_endpoint
```

## Required Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `s3_bucket_name` | S3 bucket name (must be globally unique) | `safebucket-storage-prod` |
| `sqs_queue_name` | Main SQS queue name | `safebucket-events-prod` |
| `s3_event_queue_name` | S3 events SQS queue name | `safebucket-s3-events-prod` |
| `database_password` | PostgreSQL database password | `your-secure-password` |

## Optional Configuration

### VPC Configuration

**New VPC (default):**
```hcl
create_vpc = true
vpc_cidr   = "10.0.0.0/16"
```

**Existing VPC:**
```hcl
create_vpc                  = false
existing_vpc_id             = "vpc-xxxxxxxxx"
existing_private_subnet_ids = ["subnet-xxxxxxxx", "subnet-yyyyyyyy"]
```

### Instance Sizing

**Development:**
```hcl
db_instance_class = "db.t3.micro"
cache_node_type   = "cache.t3.micro"
```

**Production:**
```hcl
db_instance_class = "db.t3.medium"
cache_node_type   = "cache.t3.small"
```

### S3 Event Configuration

```hcl
enable_s3_events = true
s3_event_types   = [
  "s3:ObjectCreated:*",
  "s3:ObjectRemoved:*"
]
```

## Outputs

The module provides all necessary configuration values as outputs:

### Infrastructure Outputs
- `vpc_id`, `private_subnet_ids`, `public_subnet_ids`
- `application_security_group_id`, `database_security_group_id`
- `s3_bucket_name`, `s3_bucket_arn`
- `database_endpoint`, `redis_endpoint`
- `sqs_events_queue_url`, `sqs_s3_events_queue_url`

### Application Configuration
- `environment_variables` - Complete environment configuration for SafeBucket

## Security Features

### Network Security
- Private subnets for database and cache
- Security groups with minimal required access
- No public access to RDS or ElastiCache

### Data Protection
- S3 server-side encryption enabled
- RDS storage encryption enabled
- ElastiCache encryption at rest and in transit
- S3 public access blocked

### IAM Security
- Least-privilege IAM role for application
- Resource-specific permissions
- Support for EC2 and ECS deployment models

## Monitoring and Backup

### Automated Backups
- RDS automated backups (7-day retention default)
- Configurable backup and maintenance windows
- S3 versioning enabled by default

### Monitoring Options
- CloudWatch metrics enabled
- Optional RDS Performance Insights
- Optional RDS Enhanced Monitoring

## Cost Optimization

### Development Environment
```hcl
environment                = "dev"
db_instance_class          = "db.t3.micro"
cache_node_type            = "cache.t3.micro"
enable_performance_insights = false
enable_enhanced_monitoring  = false
backup_retention_period    = 1
```

### Production Environment
```hcl
environment                = "prod"
db_instance_class          = "db.t3.medium"
cache_node_type            = "cache.t3.small"
enable_performance_insights = true
enable_enhanced_monitoring  = true
backup_retention_period    = 30
enable_deletion_protection  = true
```

## Usage with SafeBucket

After deploying the infrastructure, configure SafeBucket with the output values:

```bash
# Get environment variables
terraform output -json environment_variables > safebucket.env

# Example output values
export AWS_REGION="us-east-1"
export AWS_S3_BUCKET_NAME="safebucket-storage-prod"
export DATABASE_HOST="safebucket-postgres.abc123.us-east-1.rds.amazonaws.com"
export REDIS_HOST="safebucket-redis.abc123.cache.amazonaws.com"
# ... additional variables
```

## Cleanup

To destroy all created resources:

```bash
# Destroy infrastructure
terraform destroy

# Confirm when prompted
```

**Warning:** This will permanently delete all data including the S3 bucket contents and database. Ensure you have backups if needed.

## Troubleshooting

### Common Issues

1. **S3 Bucket Name Already Exists**
   ```
   Error: Error creating S3 bucket: BucketAlreadyExists
   ```
   Solution: Use a unique bucket name in `terraform.tfvars`

2. **Insufficient Permissions**
   ```
   Error: AccessDenied: User is not authorized to perform: iam:CreateRole
   ```
   Solution: Ensure your AWS credentials have necessary IAM permissions

3. **VPC Limit Exceeded**
   ```
   Error: VpcLimitExceeded: The maximum number of VPCs has been reached
   ```
   Solution: Use an existing VPC by setting `create_vpc = false`

### State Management

For production deployments, use remote state storage:

```hcl
terraform {
  backend "s3" {
    bucket = "your-terraform-state-bucket"
    key    = "safebucket/terraform.tfstate"
    region = "us-east-1"
  }
}
```

## Support

- Check the [main README](../README.md) for manual setup instructions
- Review AWS documentation for service-specific configurations
- Verify your AWS service limits and quotas