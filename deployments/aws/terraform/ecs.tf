# ECS Cluster Configuration
resource "aws_ecs_cluster" "safebucket_cluster" {
  name = "${var.project_name}-${var.environment}-cluster"

  setting {
    name  = "containerInsights"
    value = "enabled"
  }

  tags = {
    Name        = "${var.project_name}-${var.environment}-cluster"
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_ecs_cluster_capacity_providers" "cluster_capacity_providers" {
  cluster_name       = aws_ecs_cluster.safebucket_cluster.name
  capacity_providers = ["FARGATE", "FARGATE_SPOT"]

  default_capacity_provider_strategy {
    base              = 1
    weight            = 100
    capacity_provider = "FARGATE"
  }
}

# CloudWatch Log Groups
resource "aws_cloudwatch_log_group" "safebucket_logs" {
  name              = "/ecs/${var.project_name}-${var.environment}-safebucket"
  retention_in_days = lookup(local.log_retention_days, var.environment, 7)
  tags = {
    Name        = "${var.project_name}-${var.environment}-safebucket-logs"
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_cloudwatch_log_group" "loki_logs" {
  name              = "/ecs/${var.project_name}-${var.environment}-loki"
  retention_in_days = lookup(local.log_retention_days, var.environment, 7)
  tags = {
    Name        = "${var.project_name}-${var.environment}-loki-logs"
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_cloudwatch_log_group" "mailpit_logs" {
  name              = "/ecs/${var.project_name}-${var.environment}-mailpit"
  retention_in_days = lookup(local.log_retention_days, var.environment, 7)
  tags = {
    Name        = "${var.project_name}-${var.environment}-mailpit-logs"
    Environment = var.environment
    Project     = var.project_name
  }
}

# Security Groups for ECS Services
resource "aws_security_group" "ecs_tasks" {
  name        = "${var.project_name}-${var.environment}-ecs-tasks"
  description = "Security group for ECS tasks"
  vpc_id      = data.aws_vpc.default.id

  # SafeBucket application port - only accessible from ALB
  ingress {
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    security_groups = [aws_security_group.alb.id]
    description = "SafeBucket application port from ALB only"
  }

  # Loki HTTP API - internal VPC access only (Service Discovery)
  ingress {
    from_port   = 3100
    to_port     = 3100
    protocol    = "tcp"
    cidr_blocks = [data.aws_vpc.default.cidr_block]
    description = "Loki HTTP API (Service Discovery - VPC only)"
  }

  # Mailpit Web UI - only accessible from ALB
  ingress {
    from_port   = 8025
    to_port     = 8025
    protocol    = "tcp"
    security_groups = [aws_security_group.alb.id]
    description = "Mailpit Web UI from ALB only"
  }

  # Mailpit SMTP - internal VPC access only (Service Discovery)
  ingress {
    from_port   = 1025
    to_port     = 1025
    protocol    = "tcp"
    cidr_blocks = [data.aws_vpc.default.cidr_block]
    description = "Mailpit SMTP (Service Discovery - VPC only)"
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
    description = "All outbound traffic"
  }

  tags = {
    Name        = "${var.project_name}-${var.environment}-ecs-tasks"
    Environment = var.environment
    Project     = var.project_name
  }
}

# Application Load Balancer
resource "aws_lb" "safebucket_alb" {
  name               = "${var.project_name}-${var.environment}-alb"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.alb.id]
  subnets            = data.aws_subnets.default.ids

  enable_deletion_protection = var.environment == "prod" ? true : false

  tags = {
    Name        = "${var.project_name}-${var.environment}-alb"
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_security_group" "alb" {
  name        = "${var.project_name}-${var.environment}-alb"
  description = "Security group for Application Load Balancer"
  vpc_id      = data.aws_vpc.default.id

  # HTTP port for SafeBucket application
  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "HTTP - SafeBucket application"
  }

  # Mailpit Web UI port
  ingress {
    from_port   = 8025
    to_port     = 8025
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "Mailpit Web UI"
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
    description = "All outbound traffic"
  }

  tags = {
    Name        = "${var.project_name}-${var.environment}-alb"
    Environment = var.environment
    Project     = var.project_name
  }
}

# Target Groups
resource "aws_lb_target_group" "safebucket_tg" {
  name        = "${var.project_name}-${var.environment}-safebucket"
  port        = 8080
  protocol    = "HTTP"
  vpc_id      = data.aws_vpc.default.id
  target_type = "ip"

  health_check {
    enabled             = true
    healthy_threshold   = 2
    unhealthy_threshold = 3
    timeout             = 5
    interval            = 30
    path                = "/"
    matcher             = "200"
    port                = "traffic-port"
    protocol            = "HTTP"
  }

  tags = {
    Name        = "${var.project_name}-${var.environment}-safebucket-tg"
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_lb_target_group" "mailpit_tg" {
  name        = "${var.project_name}-${var.environment}-mailpit"
  port        = 8025
  protocol    = "HTTP"
  vpc_id      = data.aws_vpc.default.id
  target_type = "ip"

  health_check {
    enabled             = true
    healthy_threshold   = 2
    unhealthy_threshold = 3
    timeout             = 5
    interval            = 30
    path                = "/"
    matcher             = "200"
    port                = "traffic-port"
    protocol            = "HTTP"
  }

  tags = {
    Name        = "${var.project_name}-${var.environment}-mailpit-tg"
    Environment = var.environment
    Project     = var.project_name
  }
}

# Load Balancer Listeners
resource "aws_lb_listener" "safebucket_listener" {
  load_balancer_arn = aws_lb.safebucket_alb.arn
  port              = "80"
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.safebucket_tg.arn
  }
}

resource "aws_lb_listener" "mailpit_web_listener" {
  load_balancer_arn = aws_lb.safebucket_alb.arn
  port              = "8025"
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.mailpit_tg.arn
  }
}

