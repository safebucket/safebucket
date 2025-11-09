# SafeBucket AWS Deployment Guide

## Table of Contents
- [Overview](#overview)
- [Architecture](#architecture)
- [Prerequisites](#prerequisites)
- [Critical IAM Permissions](#critical-iam-permissions)
- [Configuration](#configuration)
- [Deployment Steps](#deployment-steps)
- [Pre-Deployment Checklist](#pre-deployment-checklist)
- [Post-Deployment Validation](#post-deployment-validation)
- [Known Issues](#known-issues)
- [Troubleshooting](#troubleshooting)
- [Security Considerations](#security-considerations)
- [Additional Resources](#additional-resources)

---

## Overview

This guide provides comprehensive instructions for deploying SafeBucket on AWS using Terraform. The infrastructure includes:

- **ECS Fargate**: Serverless container deployment for SafeBucket application
- **RDS PostgreSQL**: Managed database with multi-AZ support
- **ElastiCache Redis**: In-memory cache with TLS encryption
- **S3**: Object storage with lifecycle policies for trash retention
- **SQS**: Event-driven messaging for S3 notifications
- **Secrets Manager**: Secure credential storage
- **CloudWatch Logs**: Centralized logging with environment-specific retention
- **VPC**: Isolated network with public/private subnets and NAT Gateway
- **Application Load Balancer**: HTTPS traffic distribution

**Deployment Time**: ~15-20 minutes
**Estimated Monthly Cost**: $150-300 (varies by usage and environment)

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                          AWS Cloud                               │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                         VPC                                │  │
│  │                                                            │  │
│  │  ┌─────────────┐      ┌──────────────┐                   │  │
│  │  │   Public    │      │   Private    │                   │  │
│  │  │   Subnet    │      │   Subnet     │                   │  │
│  │  │             │      │              │                   │  │
│  │  │    ALB      │─────▶│  ECS Fargate │                  │  │
│  │  │             │      │  (SafeBucket)│                   │  │
│  │  └─────────────┘      │              │                   │  │
│  │                       │              │                   │  │
│  │                       │  RDS (Postgres)                  │  │
│  │                       │  ElastiCache (Redis)             │  │
│  │                       └──────────────┘                   │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌────────────┐  ┌────────────┐  ┌────────────────────────┐   │
│  │  S3 Bucket │  │ SQS Queues │  │   Secrets Manager      │   │
│  │  (Storage) │  │ (Events)   │  │   (Credentials)        │   │
│  └────────────┘  └────────────┘  └────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

---

## Prerequisites

### Required Tools
- **Terraform** >= 1.0.0
- **AWS CLI** >= 2.0.0
- **Docker** (for local testing)
- **Git**

### AWS Account Requirements
- AWS account with administrative access
- AWS CLI configured with credentials (`aws configure`)
- Sufficient service quotas for:
  - VPC (1 VPC, 4 subnets)
  - ECS (2 Fargate tasks minimum)
  - RDS (1 PostgreSQL instance)
  - ElastiCache (1 Redis cluster)
  - S3 (2 buckets)
  - SQS (2 queues)

### Domain and SSL
- **Domain Name**: Required for production deployment
- **SSL Certificate**: ACM certificate ARN for HTTPS (can be created via Terraform or manually)
- **Route 53** (optional): For DNS management

### Permissions
Your AWS user/role must have permissions to create:
- IAM roles and policies
- VPC and networking resources
- ECS clusters, services, and task definitions
- RDS databases
- ElastiCache clusters
- S3 buckets and lifecycle policies
- SQS queues
- Secrets Manager secrets
- CloudWatch Log Groups
- Application Load Balancers

---

## Critical IAM Permissions

The SafeBucket application requires specific IAM permissions to function correctly. **Missing any of these will cause deployment failures.**

### S3 Permissions (CRITICAL)

```terraform
# Object operations
"s3:GetObject"
"s3:PutObject"
"s3:DeleteObject"
"s3:DeleteObjects"      # Required for batch delete operations
"s3:HeadObject"          # Required for efficient metadata checks
"s3:ListBucket"

# Tagging operations (CRITICAL - required for file metadata)
"s3:GetObjectTagging"    # Application will fail without this
"s3:PutObjectTagging"    # Application will fail without this

# Lifecycle operations (CRITICAL - application startup blocker)
"s3:GetBucketLifecycleConfiguration"  # Application crashes without this
"s3:PutBucketLifecycleConfiguration"  # Application crashes without this
```

**Why These Are Critical:**
- **Lifecycle Permissions**: The application calls `EnsureTrashLifecyclePolicy()` at startup (internal/core/storage.go:25). Without these permissions, the application will exit with a fatal error.
- **Tagging Permissions**: Used for file metadata operations and trash marking (internal/storage/aws.go:195-261). Operations will fail with Access Denied errors.
- **HeadObject**: Used for efficient metadata retrieval without downloading objects (internal/storage/aws.go:284-290).
- **DeleteObjects**: Required for batch delete operations.

### SQS Permissions

```terraform
"sqs:SendMessage"        # Publish notifications
"sqs:ReceiveMessage"     # Consume S3 events
"sqs:DeleteMessage"      # REQUIRED - prevents message reprocessing
"sqs:GetQueueAttributes"
"sqs:GetQueueUrl"
```

**Note**: Both SQS queues (s3_events and notifications) require all these permissions.

### IAM Configuration Files

There are TWO IAM configuration files in this deployment:

1. **ecs_iam.tf** - Modern ECS-specific IAM roles (RECOMMENDED)
   - `ecs_execution_role`: For ECS task execution (pulling images, accessing secrets)
   - `ecs_task_role`: For application runtime permissions (S3, SQS access)

2. **iam.tf** - Legacy IAM role for EC2/ECS compatibility
   - `safebucket_app`: General application role
   - Kept for backward compatibility

**Both files have been updated with all required permissions.**

---

## Configuration

### 1. Copy Example Configuration

```bash
cd deployments/aws/terraform
cp terraform.tfvars.example terraform.tfvars
```

### 2. Edit terraform.tfvars

```hcl
# Basic Configuration
project_name = "safebucket"
environment  = "prod"  # Options: dev, staging, prod
aws_region   = "us-east-1"

# Domain Configuration (REQUIRED for HTTPS)
domain_name = "your-domain.com"
acm_certificate_arn = "arn:aws:acm:us-east-1:123456789012:certificate/xxxxx"

# Database Configuration
db_instance_class = "db.t3.small"  # dev: db.t3.micro, prod: db.t3.small or larger
db_allocated_storage = 20
db_max_allocated_storage = 100

# Redis Configuration
elasticache_node_type = "cache.t3.micro"  # Adjust based on workload

# ECS Configuration
ecs_task_cpu    = "512"   # 0.5 vCPU (dev: 256, prod: 512-1024)
ecs_task_memory = "1024"  # 1 GB (dev: 512, prod: 1024-2048)
```

### 3. Environment-Specific Settings

The deployment automatically configures environment-specific settings:

| Setting | Dev | Staging | Production |
|---------|-----|---------|------------|
| **CloudWatch Log Retention** | 3 days | 7 days | 90 days |
| **RDS Multi-AZ** | No | No | Yes (recommended) |
| **RDS Backup Retention** | 1 day | 3 days | 7-30 days |
| **Minimum Task Count** | 1 | 1 | 2+ |

These are configured in `main.tf`:

```terraform
# Environment-specific log retention periods (in days)
log_retention_days = {
  dev     = 3
  staging = 7
  prod    = 90
}
```

### 4. Secrets Configuration

The following secrets are created in AWS Secrets Manager and must be populated:

```bash
# After initial deployment, update secrets with actual values
aws secretsmanager update-secret \
  --secret-id safebucket-prod-jwt-secret \
  --secret-string "your-secure-random-string"

aws secretsmanager update-secret \
  --secret-id safebucket-prod-admin-password \
  --secret-string "your-admin-password"

aws secretsmanager update-secret \
  --secret-id safebucket-prod-db-password \
  --secret-string "your-database-password"

aws secretsmanager update-secret \
  --secret-id safebucket-prod-redis-auth-token \
  --secret-string "your-redis-auth-token"
```

**Security Note**: Use strong, randomly generated values for all secrets. Consider using a password manager or:

```bash
# Generate secure random strings
openssl rand -base64 32
```

---

## Deployment Steps

### Step 1: Initialize Terraform

```bash
cd deployments/aws/terraform
terraform init
```

**Expected Output:**
```
Initializing modules...
Initializing the backend...
Initializing provider plugins...
Terraform has been successfully initialized!
```

### Step 2: Review Planned Changes

```bash
terraform plan
```

**Review carefully:**
- Number of resources to be created (~40-50 resources)
- IAM roles and policies
- Networking configuration
- Database settings
- Cost estimates

### Step 3: Deploy Infrastructure

```bash
terraform apply
```

**Deployment Time:** 15-20 minutes

**Critical Steps During Apply:**
1. VPC and networking resources created (2-3 min)
2. Security groups configured
3. RDS database provisioned (10-12 min) - **Longest step**
4. ElastiCache cluster created (5-7 min)
5. ECS cluster and services launched
6. Load balancer configured
7. S3 buckets and SQS queues created

### Step 4: Populate Secrets

```bash
# Use the script or manual commands from "Secrets Configuration" section above
./scripts/populate-secrets.sh  # If you create this helper script
```

### Step 5: Deploy Application Container

```bash
# Push your SafeBucket Docker image to ECR
aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin <account-id>.dkr.ecr.us-east-1.amazonaws.com

docker build -t safebucket:latest .
docker tag safebucket:latest <account-id>.dkr.ecr.us-east-1.amazonaws.com/safebucket:latest
docker push <account-id>.dkr.ecr.us-east-1.amazonaws.com/safebucket:latest
```

### Step 6: Verify Deployment

```bash
# Get Load Balancer DNS
terraform output alb_dns_name

# Check ECS service status
aws ecs describe-services \
  --cluster safebucket-prod \
  --services safebucket \
  --query 'services[0].{status:status,running:runningCount,desired:desiredCount}'

# View application logs
aws logs tail /ecs/safebucket-prod --follow
```

---

## Pre-Deployment Checklist

Before deploying to production, verify:

### Infrastructure Readiness
- [ ] AWS credentials configured and tested (`aws sts get-caller-identity`)
- [ ] Terraform version >= 1.0.0 (`terraform version`)
- [ ] terraform.tfvars configured with correct values
- [ ] ACM certificate created and validated for your domain
- [ ] Service quotas sufficient for deployment
- [ ] Cost estimates reviewed and approved

### Security Configuration
- [ ] All secrets generated with strong random values
- [ ] IAM permissions reviewed (both ecs_iam.tf and iam.tf)
- [ ] VPC CIDR ranges don't conflict with existing networks
- [ ] Security group rules follow least privilege principle
- [ ] S3 bucket encryption enabled (default in config)
- [ ] RDS encryption enabled (default in config)
- [ ] Redis TLS enabled (default in config)

### Application Configuration
- [ ] Docker image built and tagged
- [ ] ECR repository created
- [ ] Environment variables verified in ecs_task_safebucket.tf
- [ ] Database migration strategy planned
- [ ] Backup and disaster recovery plan documented

### Monitoring and Logging
- [ ] CloudWatch Log Groups will be created automatically
- [ ] Log retention periods appropriate for environment
- [ ] Alerts configured (optional, can be added post-deployment)

---

## Post-Deployment Validation

After deployment completes, run these tests:

### 1. Application Health Check

```bash
# Get ALB DNS name
ALB_DNS=$(terraform output -raw alb_dns_name)

# Test health endpoint
curl https://${ALB_DNS}/health

# Expected response: {"status": "ok"}
```

### 2. Verify S3 Lifecycle Policies

```bash
# Check that lifecycle policies are configured correctly
BUCKET_NAME=$(terraform output -raw s3_bucket_name)

aws s3api get-bucket-lifecycle-configuration --bucket ${BUCKET_NAME}

# Expected output should include:
# 1. "abort-incomplete-multipart-uploads" rule (1 day)
# 2. Application-managed trash expiration rule (configured at startup)
```

**Expected Lifecycle Rules (Both Application-Managed):**
- **Trash Expiration**: Files in `trash/` prefix expire after `TRASH_RETENTION_DAYS`
- **Incomplete Multipart Uploads**: Automatically aborted after 1 day (saves storage costs)

**Why Application-Managed?**
Both lifecycle policies are created by the application at startup (not Terraform) because:
- Consistent across all storage providers (AWS S3, MinIO, GCP)
- Single source of truth for lifecycle configuration
- No Terraform/Application conflicts or drift
- Easier to update retention periods without infrastructure changes

**If this fails**, check application logs for lifecycle permission errors.

### 3. Test File Operations

```bash
# Upload a test file
curl -X POST https://${ALB_DNS}/api/files/upload \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "file=@test.txt"

# Download the file
curl https://${ALB_DNS}/api/files/download/{file_id} \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -o downloaded.txt

# Move to trash
curl -X DELETE https://${ALB_DNS}/api/files/{file_id} \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# Verify trash marker created in S3
aws s3 ls s3://${BUCKET_NAME}/trash/ --recursive
```

### 4. Verify SQS Event Processing

```bash
# Check SQS queue metrics
aws cloudwatch get-metric-statistics \
  --namespace AWS/SQS \
  --metric-name NumberOfMessagesSent \
  --dimensions Name=QueueName,Value=safebucket-prod-s3-events \
  --start-time $(date -u -d '5 minutes ago' +%Y-%m-%dT%H:%M:%S) \
  --end-time $(date -u +%Y-%m-%dT%H:%M:%S) \
  --period 300 \
  --statistics Sum

# Check application logs for event processing
aws logs tail /ecs/safebucket-prod --follow --filter-pattern "S3 event"
```

### 5. Database Connectivity

```bash
# View application logs for database connection
aws logs tail /ecs/safebucket-prod --follow --filter-pattern "database"

# Should see: "Successfully connected to database"
# Should NOT see: "Failed to connect" or "Access denied"
```

### 6. Redis Connectivity

```bash
# View application logs for Redis connection
aws logs tail /ecs/safebucket-prod --follow --filter-pattern "redis"

# Should see: "Connected to Redis"
# Should NOT see: "Redis connection failed"
```

### 7. Complete Testing Checklist

Use this comprehensive checklist from INFRASTRUCTURE_ALIGNMENT_ANALYSIS.md:

- [ ] Application starts without lifecycle policy errors
- [ ] Can upload files to S3
- [ ] Can download files from S3
- [ ] Can delete files (move to trash)
- [ ] Trash markers are created correctly
- [ ] Lifecycle policy is applied to bucket
- [ ] SQS events are received and processed
- [ ] File metadata (tags) can be read/written
- [ ] Batch delete operations work
- [ ] No IAM permission denied errors in logs

---

## Known Issues

### 1. Duplicate IAM Configurations

**Issue**: Both `iam.tf` and `ecs_iam.tf` exist with similar IAM roles.

**Status**: ✅ Both files have been updated with identical permissions

**Details**:
- **ecs_iam.tf** - Modern ECS-specific roles (recommended for ECS deployments)
- **iam.tf** - Legacy role supporting both EC2 and ECS

**Impact**: LOW - Both configurations are now production-ready with all required permissions

**Recommendation**:
- For new deployments, use `ecs_iam.tf` (already configured in ECS task definitions)
- Consider removing `iam.tf` in future cleanup
- No action required for current deployments

### 2. Initial Secrets Are Placeholders

**Issue**: Secrets created by Terraform have placeholder values

**Status**: ⚠️ MANUAL ACTION REQUIRED

**Resolution**: Update secrets after deployment using AWS CLI or Console (see "Secrets Configuration" section)

---

## Troubleshooting

### Application Won't Start - Lifecycle Policy Error

**Symptom**: Application logs show:
```
FATAL: Failed to configure trash lifecycle policy: AccessDenied
```

**Cause**: Missing S3 lifecycle permissions

**Fix**:
1. Verify IAM role has `s3:GetBucketLifecycleConfiguration` and `s3:PutBucketLifecycleConfiguration`
2. Check `ecs_iam.tf` lines 114-116
3. Re-run `terraform apply` if permissions were added

**Prevention**: This should not occur with current configuration - all permissions are included.

### Access Denied on File Upload

**Symptom**: File upload fails with 403 Access Denied

**Possible Causes**:
1. Missing `s3:PutObjectTagging` permission
2. Missing `s3:PutObject` permission
3. Incorrect bucket name in configuration

**Fix**:
```bash
# Check current IAM policy
aws iam get-role-policy \
  --role-name safebucket-prod-ecs-task-role \
  --policy-name safebucket-prod-safebucket-task-policy

# Verify S3 bucket name in ECS task definition
aws ecs describe-task-definition \
  --task-definition safebucket-prod \
  --query 'taskDefinition.containerDefinitions[0].environment[?name==`STORAGE__AWS__BUCKET_NAME`]'
```

### SQS Messages Not Being Processed

**Symptom**: Messages accumulate in SQS queue but aren't processed

**Possible Causes**:
1. Missing `sqs:DeleteMessage` permission (messages reprocess)
2. Application not subscribed to queue
3. Invalid message format

**Fix**:
```bash
# Check queue attributes
aws sqs get-queue-attributes \
  --queue-url $(terraform output -raw sqs_s3_events_url) \
  --attribute-names All

# View dead letter queue (if configured)
aws sqs receive-message \
  --queue-url $(terraform output -raw sqs_dlq_url)

# Check application logs for processing errors
aws logs tail /ecs/safebucket-prod --follow --filter-pattern "ERROR"
```

### RDS Connection Timeout

**Symptom**: Application can't connect to database

**Possible Causes**:
1. Security group rules blocking traffic
2. Incorrect database endpoint
3. Wrong credentials in Secrets Manager

**Fix**:
```bash
# Verify security group rules allow ECS -> RDS traffic
aws ec2 describe-security-groups \
  --group-ids $(terraform output -raw rds_security_group_id)

# Test connectivity from ECS task
aws ecs execute-command \
  --cluster safebucket-prod \
  --task <task-id> \
  --command "nc -zv <rds-endpoint> 5432" \
  --interactive

# Verify database endpoint in task definition
aws ecs describe-task-definition \
  --task-definition safebucket-prod \
  --query 'taskDefinition.containerDefinitions[0].environment[?name==`DATABASE__HOST`]'
```

### High CloudWatch Costs

**Symptom**: Unexpected CloudWatch costs

**Cause**: Log retention set too high for dev/staging environments

**Fix**: Adjust log retention in `main.tf`:
```terraform
log_retention_days = {
  dev     = 1   # Reduce to 1 day for development
  staging = 3   # Reduce to 3 days for staging
  prod    = 90  # Keep 90 days for production
}
```

---

## Security Considerations

### Secrets Management
- ✅ All secrets stored in AWS Secrets Manager (encrypted at rest)
- ✅ ECS task execution role has least privilege access to secrets
- ⚠️ Rotate secrets regularly (recommended: every 90 days)
- ⚠️ Enable automatic secret rotation for database credentials

### Network Security
- ✅ RDS and ElastiCache deployed in private subnets (no public access)
- ✅ Security groups follow least privilege principle
- ✅ VPC endpoints for private AWS service access (optional enhancement)
- ⚠️ Enable VPC Flow Logs for network traffic analysis (optional)

### Data Encryption
- ✅ S3 buckets encrypted at rest (AES-256)
- ✅ RDS encrypted at rest (KMS)
- ✅ ElastiCache TLS in-transit encryption enabled
- ✅ ALB HTTPS with ACM certificate
- ⚠️ Consider customer-managed KMS keys for additional control

### IAM Security
- ✅ Separate execution role (image pulling) and task role (runtime permissions)
- ✅ Least privilege permissions for S3, SQS, and other services
- ✅ No hardcoded credentials in code or configuration
- ⚠️ Enable CloudTrail for IAM activity auditing

### Audit and Compliance
- ✅ CloudWatch Logs for application activity
- ✅ S3 server access logging available (optional)
- ✅ RDS enhanced monitoring available (optional)
- ⚠️ Enable AWS Config for compliance tracking
- ⚠️ Implement CloudWatch Alarms for security events

### Backup and Disaster Recovery
- ✅ RDS automated backups enabled
- ✅ S3 versioning available (optional)
- ⚠️ Configure backup retention based on compliance requirements
- ⚠️ Test disaster recovery procedures regularly
- ⚠️ Consider cross-region replication for critical data

---

## Additional Resources

### Documentation
- **External Docs**: https://docs.safebucket.io/docs/getting-started/aws-deployment
- **Security Review**: See `SECURITY_REVIEW_ISSUES.md` for comprehensive security audit
- **Infrastructure Alignment**: See `INFRASTRUCTURE_ALIGNMENT_ANALYSIS.md` for detailed infrastructure analysis

### Terraform Resources Created

This deployment creates approximately 45-50 resources:

**Networking** (15 resources):
- 1 VPC
- 4 Subnets (2 public, 2 private)
- 2 Route Tables
- 1 Internet Gateway
- 1 NAT Gateway
- 5 Security Groups
- Various route table associations

**Compute** (8 resources):
- 1 ECS Cluster
- 2 ECS Task Definitions (SafeBucket, Loki)
- 2 ECS Services
- 1 Application Load Balancer
- 1 Target Group
- 1 Listener

**Storage & Data** (7 resources):
- 2 S3 Buckets (main storage, Loki logs)
- 1 RDS PostgreSQL Instance
- 1 RDS Subnet Group
- 1 ElastiCache Redis Cluster
- 1 ElastiCache Subnet Group
- 1 ElastiCache Parameter Group

**Messaging** (2 resources):
- 2 SQS Queues (s3_events, notifications)

**Secrets** (4 resources):
- 4 Secrets Manager Secrets

**IAM** (6 resources):
- 2 IAM Roles (execution, task)
- 4 IAM Policies

**Logging** (3 resources):
- 3 CloudWatch Log Groups

### Cost Optimization Tips

**Development Environment**:
- Use `db.t3.micro` for RDS
- Use `cache.t3.micro` for Redis
- Set ECS desired count to 1
- Reduce log retention to 1-3 days
- Use spot instances for ECS (optional)

**Production Environment**:
- Right-size RDS and Redis based on actual usage
- Enable RDS autoscaling for storage
- Use Reserved Instances or Savings Plans for predictable workloads
- Implement S3 Intelligent-Tiering for storage cost optimization
- Monitor and set CloudWatch billing alarms

**S3 Cost Optimization** (Already Implemented):
- ✅ Incomplete multipart uploads automatically aborted after 1 day (prevents orphaned part accumulation)
- ✅ Trash files automatically expire based on `TRASH_RETENTION_DAYS` configuration
- Consider enabling S3 Storage Lens for deeper cost insights
- Enable S3 bucket metrics for monitoring storage growth

### Support and Contributing

For issues or questions:
- GitHub Issues: https://github.com/safebucket/safebucket/issues
- Documentation: https://docs.safebucket.io
- Security Issues: Email security@safebucket.io

---

## Changelog

### Version 1.1.0 (2025-11-08)
- ✅ Added critical S3 lifecycle permissions (fixes deployment blocker)
- ✅ Added S3 object tagging permissions (fixes file operations)
- ✅ Added S3 HeadObject permission (improves performance)
- ✅ Added S3 DeleteObjects permission (enables batch operations)
- ✅ Added SQS DeleteMessage permission (prevents message reprocessing)
- ✅ Updated both iam.tf and ecs_iam.tf with identical permissions
- ✅ Added environment-specific log retention configuration
- ✅ Fixed race condition in folder deletion (trash_expiration.go)
- ✅ Added input validation for lifecycle retention days (s3.go)
- ✅ Added S3 lifecycle policy to abort incomplete multipart uploads after 1 day (cost optimization)
- ✅ Comprehensive security review completed (SECURITY_REVIEW_ISSUES.md)
- ✅ Infrastructure alignment analysis completed (INFRASTRUCTURE_ALIGNMENT_ANALYSIS.md)

### Version 1.0.0 (Initial)
- Initial Terraform configuration for AWS deployment
