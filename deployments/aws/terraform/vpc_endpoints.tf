# VPC Endpoints for private subnet connectivity to AWS services

# Secrets Manager VPC Endpoint
resource "aws_vpc_endpoint" "secrets_manager" {
  vpc_id             = data.aws_vpc.default.id
  service_name       = "com.amazonaws.${data.aws_region.current.name}.secretsmanager"
  vpc_endpoint_type  = "Interface"
  subnet_ids         = data.aws_subnets.default.ids
  security_group_ids = [aws_security_group.vpc_endpoints.id]

  private_dns_enabled = true

  tags = merge(local.common_tags, {
    Name = "${var.project_name}-secrets-manager-endpoint"
  })
}

# ECR API VPC Endpoint (for pulling container images)
resource "aws_vpc_endpoint" "ecr_api" {
  vpc_id             = data.aws_vpc.default.id
  service_name       = "com.amazonaws.${data.aws_region.current.name}.ecr.api"
  vpc_endpoint_type  = "Interface"
  subnet_ids         = data.aws_subnets.default.ids
  security_group_ids = [aws_security_group.vpc_endpoints.id]

  private_dns_enabled = true

  tags = merge(local.common_tags, {
    Name = "${var.project_name}-ecr-api-endpoint"
  })
}

# ECR DKR VPC Endpoint (for pulling container images)
resource "aws_vpc_endpoint" "ecr_dkr" {
  vpc_id             = data.aws_vpc.default.id
  service_name       = "com.amazonaws.${data.aws_region.current.name}.ecr.dkr"
  vpc_endpoint_type  = "Interface"
  subnet_ids         = data.aws_subnets.default.ids
  security_group_ids = [aws_security_group.vpc_endpoints.id]

  private_dns_enabled = true

  tags = merge(local.common_tags, {
    Name = "${var.project_name}-ecr-dkr-endpoint"
  })
}

# S3 VPC Endpoint (Gateway type for ECR image layers)
resource "aws_vpc_endpoint" "s3" {
  vpc_id          = data.aws_vpc.default.id
  service_name    = "com.amazonaws.${data.aws_region.current.name}.s3"
  vpc_endpoint_type = "Gateway"
  route_table_ids = data.aws_route_tables.default.ids

  tags = merge(local.common_tags, {
    Name = "${var.project_name}-s3-endpoint"
  })
}

# CloudWatch Logs VPC Endpoint
resource "aws_vpc_endpoint" "logs" {
  vpc_id             = data.aws_vpc.default.id
  service_name       = "com.amazonaws.${data.aws_region.current.name}.logs"
  vpc_endpoint_type  = "Interface"
  subnet_ids         = data.aws_subnets.default.ids
  security_group_ids = [aws_security_group.vpc_endpoints.id]

  private_dns_enabled = true

  tags = merge(local.common_tags, {
    Name = "${var.project_name}-logs-endpoint"
  })
}

# SQS VPC Endpoint
resource "aws_vpc_endpoint" "sqs" {
  vpc_id             = data.aws_vpc.default.id
  service_name       = "com.amazonaws.${data.aws_region.current.name}.sqs"
  vpc_endpoint_type  = "Interface"
  subnet_ids         = data.aws_subnets.default.ids
  security_group_ids = [aws_security_group.vpc_endpoints.id]

  private_dns_enabled = true

  tags = merge(local.common_tags, {
    Name = "${var.project_name}-sqs-endpoint"
  })
}

# Security group for VPC endpoints
resource "aws_security_group" "vpc_endpoints" {
  name_prefix = "${var.project_name}-vpc-endpoints-"
  description = "Security group for VPC endpoints"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    description = "HTTPS from ECS tasks"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    security_groups = [aws_security_group.ecs_tasks.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(local.common_tags, {
    Name = "${var.project_name}-vpc-endpoints-sg"
  })

  lifecycle {
    create_before_destroy = true
  }
}

# Data source for route tables
data "aws_route_tables" "default" {
  vpc_id = data.aws_vpc.default.id
}