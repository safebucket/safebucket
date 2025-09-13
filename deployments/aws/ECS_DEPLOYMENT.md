# SafeBucket ECS Deployment

Complete AWS ECS deployment with 3 services: SafeBucket application, Loki (logging), and Mailpit (SMTP).

## Architecture

- **ECS Fargate Cluster** with 3 services
- **Application Load Balancer** for public access
- **Internal Load Balancer** for service communication
- **RDS PostgreSQL** database with TLS encryption
- **ElastiCache Redis** with TLS and AUTH token
- **S3 buckets** for storage (SafeBucket data + Loki storage)
- **SQS queues** for event messaging
- **Secrets Manager** for sensitive data
- **VPC Endpoints** for secure AWS service communication

## Services

1. **SafeBucket** (512 CPU, 1024 MB) - Port 8080
2. **Loki** (512 CPU, 1024 MB) - Port 3100
3. **Mailpit** (256 CPU, 512 MB) - Ports 8025, 1025

## Prerequisites

1. **AWS CLI configured** with appropriate permissions
2. **Terraform installed** (version 1.0+)
3. **Docker image built and pushed** to ECR/Docker Hub

## Deployment

1. **Configure variables:**
   ```bash
   cd deployments/aws/terraform
   cp terraform.tfvars.example terraform.tfvars
   # Edit terraform.tfvars with your values
   ```

2. **Required variables:**
   ```hcl
   # Storage
   s3_bucket_name = "safebucket-storage-dev-your-unique-suffix"

   # Messaging
   s3_event_queue_name = "safebucket-s3-events-dev"
   notification_queue_name = "safebucket-notifications-dev"

   # Security
   redis_auth_token = "your-32-char-redis-auth-token-here"
   rds_password = "your-secure-database-password-here"
   jwt_secret = "your-jwt-secret-32-chars-minimum-here"
   admin_password = "your-admin-password-here-12342"

   # Container Image
   safebucket_image = "your-account.dkr.ecr.region.amazonaws.com/safebucket:latest"
   # OR use Docker Hub: "docker.io/your-org/safebucket:latest"
   ```

3. **Optional variables:**
   ```hcl
   # Scaling
   enable_autoscaling = true
   safebucket_max_capacity = 5

   # Resources
   safebucket_cpu = 1024
   safebucket_memory = 2048
   loki_cpu = 1024
   loki_memory = 2048

   # Environment
   environment = "prod"
   s3_cors_allowed_origins = ["https://your-domain.com"]
   ```

4. **Deploy:**
   ```bash
   terraform init
   terraform plan -var-file="terraform.tfvars"
   terraform apply -var-file="terraform.tfvars"
   ```

## Access

- **Application:** `http://<alb-dns-name>`
- **Loki:** `http://<internal-alb-dns-name>:3100` 
- **Mailpit:** `http://<internal-alb-dns-name>:8025`

Get URLs: `terraform output alb_dns_name` and `terraform output internal_alb_dns_name`

## Infrastructure Components

### Networking
- **VPC**: Uses default VPC and subnets
- **Security Groups**:
  - ECS tasks security group (ports 8080, 3100, 8025, 1025)
  - ALB security group (ports 80, 443)
  - Internal ALB security group (service-to-service communication)

### Storage & Database
- **RDS PostgreSQL**:
  - Instance class: `db.t3.micro` (configurable)
  - Storage: 20GB with auto-scaling to 100GB
  - Encrypted at rest, TLS in transit
  - Automated backups (7 days retention)
- **ElastiCache Redis**:
  - Node type: `cache.t3.micro` (configurable)
  - TLS encryption enabled
  - AUTH token authentication
  - Automatic snapshots (5 days retention)
- **S3 Buckets**:
  - SafeBucket data storage with event notifications
  - Loki log storage bucket
  - Versioning and encryption enabled

### Messaging & Events
- **SQS Queues**:
  - S3 events queue for file operations
  - Application notifications queue
  - Dead letter queues for failed messages

## Environment Variables (Auto-configured)

SafeBucket service receives these environment variables automatically:

### Application Config
- `APP__API_URL`, `APP__WEB_URL`: Load balancer DNS
- `APP__PORT`: 8080
- `APP__STATIC_FILES__ENABLED`: true
- `APP__ADMIN_EMAIL`: From variables
- `APP__ALLOWED_ORIGINS`: Load balancer DNS

### Database Config
- `DATABASE__HOST`: RDS endpoint
- `DATABASE__PORT`: 5432
- `DATABASE__USER`, `DATABASE__NAME`: From variables
- `DATABASE__SSLMODE`: require

### Cache Config (Redis with TLS)
- `CACHE__TYPE`: redis
- `CACHE__REDIS__HOSTS`: ElastiCache endpoint:6379
- `CACHE__REDIS__TLS_ENABLED`: true
- `CACHE__REDIS__TLS_SERVER_NAME`: ElastiCache endpoint

### Storage Config
- `STORAGE__TYPE`: aws
- `STORAGE__AWS__BUCKET_NAME`: S3 bucket name
- `STORAGE__AWS__SQS_NAME`: S3 events queue

### Events & Messaging
- `EVENTS__TYPE`: aws
- `EVENTS__AWS__SQS_NAME`: S3 events queue
- `NOTIFIER__TYPE`: smtp (via Mailpit)
- `NOTIFIER__SMTP__HOST`: Internal ALB DNS

### Activity Logging
- `ACTIVITY__TYPE`: loki
- `ACTIVITY__LOKI__ENDPOINT`: Internal Loki endpoint

### Secrets (from AWS Secrets Manager)
- `APP__JWT_SECRET`: JWT signing key
- `APP__ADMIN_PASSWORD`: Admin user password
- `DATABASE__PASSWORD`: RDS password
- `CACHE__REDIS__PASSWORD`: Redis AUTH token

## Post-Deployment

After successful deployment:

1. **Get application URL:**
   ```bash
   terraform output alb_dns_name
   ```

2. **Login to SafeBucket:**
   - URL: `http://<alb-dns-name>`
   - Username: `admin@safebucket.io` (or your `admin_email`)
   - Password: Your `admin_password` from terraform.tfvars

3. **Access internal services:**
   ```bash
   # Get internal ALB DNS
   terraform output internal_alb_dns_name

   # Loki: http://<internal-alb-dns>:3100
   # Mailpit: http://<internal-alb-dns>:8025
   ```

## Monitoring & Logging

### CloudWatch Logs
- **SafeBucket:** `/ecs/safebucket-{env}-safebucket`
- **Loki:** `/ecs/safebucket-{env}-loki`
- **Mailpit:** `/ecs/safebucket-{env}-mailpit`

### Health Checks
- **SafeBucket:** `GET /` (returns 200)
- **Loki:** `GET /ready` (returns 200)
- **Mailpit:** `GET /` (returns 200)

### Monitoring Tools
- **ECS Console:** Monitor service health, CPU/memory, task logs
- **CloudWatch Metrics:** CPU, memory, network, custom metrics
- **Load Balancer:** Target group health, request metrics

## Troubleshooting

### Common Issues

1. **Tasks not starting:**
   ```bash
   # Check ECS service events
   aws ecs describe-services --cluster safebucket-dev-cluster --services safebucket-dev-safebucket

   # Check task logs
   aws logs get-log-events --log-group-name /ecs/safebucket-dev-safebucket --log-stream-name <stream-name>
   ```

2. **Database connection issues:**
   - Verify RDS security group allows connections from ECS tasks
   - Check DATABASE_* environment variables in task definition
   - Ensure RDS is in same VPC as ECS tasks

3. **Redis connection issues:**
   - Verify ElastiCache security group configuration
   - Check TLS settings match between infrastructure and app config
   - Validate AUTH token in Secrets Manager

4. **Load balancer health checks failing:**
   - Check target group health in EC2 console
   - Verify health check path and expected response code
   - Ensure security groups allow ALB to reach tasks

### Debugging Commands

```bash
# ECS service status
aws ecs describe-services --cluster <cluster-name> --services <service-name>

# Task details
aws ecs describe-tasks --cluster <cluster-name> --tasks <task-arn>

# CloudWatch logs
aws logs describe-log-streams --log-group-name <log-group>
aws logs get-log-events --log-group-name <log-group> --log-stream-name <stream>

# Secrets Manager
aws secretsmanager get-secret-value --secret-id <secret-name>

# RDS endpoint
aws rds describe-db-instances --db-instance-identifier <db-identifier>
```

### ECS Exec (if enabled)
```bash
# Connect to running SafeBucket container
aws ecs execute-command --cluster safebucket-dev-cluster \
  --task <task-id> --container safebucket --interactive --command "/bin/sh"
```

## Scaling

### Manual Scaling
```bash
# Update desired count
aws ecs update-service --cluster <cluster> --service <service> --desired-count 3
```

### Auto Scaling (if enabled)
- **CPU-based scaling:** Target 70% average CPU utilization
- **Memory-based scaling:** Target 80% average memory utilization
- **Scale-out cooldown:** 300 seconds
- **Scale-in cooldown:** 300 seconds

## Security Considerations

- **Network:** Services communicate via internal load balancer
- **Encryption:** TLS for Redis, RDS, and S3
- **Secrets:** All sensitive data in AWS Secrets Manager
- **IAM:** Minimal required permissions for each service
- **VPC:** Uses security groups for network isolation

## Cleanup

```bash
terraform destroy -var-file="terraform.tfvars"
```

**Note:** This will destroy all resources including data in RDS and S3. Make sure to backup any important data first.