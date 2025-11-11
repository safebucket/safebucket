# SafeBucket AWS Infrastructure - Minimal Setup
# Base configuration and shared resources

terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# AWS Provider configuration
provider "aws" {
  region = "eu-west-1"
}

# Data sources
data "aws_region" "current" {}

data "aws_vpc" "default" {
  default = true
}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

# Local variables
locals {
  common_tags = {
    Project     = "SafeBucket"
    Environment = var.environment
    ManagedBy   = "Terraform"
  }

  # Environment-specific log retention periods (in days)
  log_retention_days = {
    dev     = 3
    staging = 7
    prod    = 90
  }

  # Dynamic CORS origins - use custom list if provided, otherwise use ALB DNS and localhost
  cors_allowed_origins = length(var.s3_cors_allowed_origins) > 0 ? var.s3_cors_allowed_origins : [
    "http://${aws_lb.safebucket_alb.dns_name}",
    "http://localhost:3000"
  ]

  # S3 external endpoint for CSP (presigned URLs)
  s3_external_endpoint = "https://${aws_s3_bucket.main.bucket}.s3.${data.aws_region.current.name}.amazonaws.com"
}

