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
  region = "eu-west-1" # Ireland
}

# Local variables
locals {
  common_tags = {
    Project     = "SafeBucket"
    Environment = var.environment
    ManagedBy   = "Terraform"
  }
}

