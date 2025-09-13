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
  region     = "eu-west-1"
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
}

