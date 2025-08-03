# SafeBucket ElastiCache Redis Resources

# Get default VPC information
data "aws_vpc" "default" {
  default = true
}

# Get default subnets
data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

# Security Group for ElastiCache Redis
resource "aws_security_group" "redis" {
  name_prefix = "${var.project_name}-redis-"
  description = "Security group for SafeBucket Redis cluster"
  vpc_id      = data.aws_vpc.default.id

  # Allow inbound Redis traffic from within the VPC
  ingress {
    description = "Redis access from VPC"
    from_port   = 6379
    to_port     = 6379
    protocol    = "tcp"
    cidr_blocks = [data.aws_vpc.default.cidr_block]
  }

  # Allow all outbound traffic
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(local.common_tags, {
    Name = "${var.project_name}-redis-sg"
  })

  lifecycle {
    create_before_destroy = true
  }
}

# ElastiCache Subnet Group
resource "aws_elasticache_subnet_group" "redis" {
  name       = "${var.project_name}-redis-subnet-group"
  subnet_ids = data.aws_subnets.default.ids

  tags = merge(local.common_tags, {
    Name = "${var.project_name}-redis-subnet-group"
  })
}

# ElastiCache Parameter Group for Redis
resource "aws_elasticache_parameter_group" "redis" {
  family = "redis7"
  name   = "${var.project_name}-redis-params"

  # Redis configuration parameters
  parameter {
    name  = "maxmemory-policy"
    value = "allkeys-lru"
  }

  tags = merge(local.common_tags, {
    Name = "${var.project_name}-redis-params"
  })
}


# CloudWatch Log Group for Redis slow logs
resource "aws_cloudwatch_log_group" "redis_slow" {
  name              = "/aws/elasticache/redis/${var.project_name}-slow-log"
  retention_in_days = var.redis_log_retention_days

  tags = merge(local.common_tags, {
    Name = "${var.project_name}-redis-slow-log"
  })
}

# ElastiCache User Group
resource "aws_elasticache_user_group" "redis" {
  engine        = "redis"
  user_group_id = "${var.project_name}-redis-users"
  user_ids      = ["default", aws_elasticache_user.redis_app_user.user_id]

  tags = merge(local.common_tags, {
    Name = "${var.project_name}-redis-user-group"
  })
}

# ElastiCache Application User
resource "aws_elasticache_user" "redis_app_user" {
  user_id       = "${var.project_name}-app-user"
  user_name     = "${var.project_name}-app-user"
  access_string = "on ~* &* +@all"
  engine        = "redis"
  passwords     = var.redis_auth_token_enabled && var.redis_auth_token != null ? [var.redis_auth_token] : null

  tags = merge(local.common_tags, {
    Name = "${var.project_name}-redis-app-user"
  })
}

# ElastiCache Cluster
resource "aws_elasticache_cluster" "redis" {
  cluster_id           = "${var.project_name}-redis-cluster"
  engine               = "redis"
  node_type            = var.redis_node_type
  num_cache_nodes      = var.redis_num_cache_nodes
  parameter_group_name = aws_elasticache_parameter_group.redis.name
  port                 = 6379
  subnet_group_name    = aws_elasticache_subnet_group.redis.name
  security_group_ids   = [aws_security_group.redis.id]
  
  # Maintenance configuration
  maintenance_window   = var.redis_maintenance_window
  snapshot_window      = var.redis_snapshot_window
  snapshot_retention_limit = var.redis_snapshot_retention_limit

  tags = merge(local.common_tags, {
    Name = "${var.project_name}-redis-cluster"
  })

  depends_on = [
    aws_elasticache_parameter_group.redis,
    aws_elasticache_subnet_group.redis
  ]
}