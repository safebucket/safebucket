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
  retention_in_days = var.log_retention_days
  tags = {
    Name        = "${var.project_name}-${var.environment}-safebucket-logs"
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_cloudwatch_log_group" "loki_logs" {
  name              = "/ecs/${var.project_name}-${var.environment}-loki"
  retention_in_days = var.log_retention_days
  tags = {
    Name        = "${var.project_name}-${var.environment}-loki-logs"
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_cloudwatch_log_group" "mailpit_logs" {
  name              = "/ecs/${var.project_name}-${var.environment}-mailpit"
  retention_in_days = var.log_retention_days
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

  ingress {
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "SafeBucket application port"
  }

  ingress {
    from_port   = 3100
    to_port     = 3100
    protocol    = "tcp"
    cidr_blocks = [data.aws_vpc.default.cidr_block]
    description = "Loki HTTP API"
  }

  ingress {
    from_port   = 8025
    to_port     = 8025
    protocol    = "tcp"
    cidr_blocks = [data.aws_vpc.default.cidr_block]
    description = "Mailpit web UI"
  }

  ingress {
    from_port   = 1025
    to_port     = 1025
    protocol    = "tcp"
    cidr_blocks = [data.aws_vpc.default.cidr_block]
    description = "Mailpit SMTP"
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

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "HTTP"
  }

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "HTTPS"
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

resource "aws_lb_target_group" "loki_tg" {
  name        = "${var.project_name}-${var.environment}-loki"
  port        = 3100
  protocol    = "HTTP"
  vpc_id      = data.aws_vpc.default.id
  target_type = "ip"

  health_check {
    enabled             = true
    healthy_threshold   = 2
    unhealthy_threshold = 3
    timeout             = 5
    interval            = 30
    path                = "/ready"
    matcher             = "200"
    port                = "traffic-port"
    protocol            = "HTTP"
  }

  tags = {
    Name        = "${var.project_name}-${var.environment}-loki-tg"
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

# Internal Load Balancer for Loki and Mailpit
resource "aws_lb" "internal_alb" {
  name               = "${var.project_name}-${var.environment}-internal-alb"
  internal           = true
  load_balancer_type = "application"
  security_groups    = [aws_security_group.internal_alb.id]
  subnets            = data.aws_subnets.default.ids

  tags = {
    Name        = "${var.project_name}-${var.environment}-internal-alb"
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_security_group" "internal_alb" {
  name        = "${var.project_name}-${var.environment}-internal-alb"
  description = "Security group for Internal Application Load Balancer"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    from_port       = 3100
    to_port         = 3100
    protocol        = "tcp"
    security_groups = [aws_security_group.ecs_tasks.id]
    description     = "Loki from ECS tasks"
  }

  ingress {
    from_port       = 8025
    to_port         = 8025
    protocol        = "tcp"
    security_groups = [aws_security_group.ecs_tasks.id]
    description     = "Mailpit Web UI from ECS tasks"
  }

  ingress {
    from_port       = 1025
    to_port         = 1025
    protocol        = "tcp"
    security_groups = [aws_security_group.ecs_tasks.id]
    description     = "Mailpit SMTP from ECS tasks"
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
    description = "All outbound traffic"
  }

  tags = {
    Name        = "${var.project_name}-${var.environment}-internal-alb"
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_lb_listener" "loki_listener" {
  load_balancer_arn = aws_lb.internal_alb.arn
  port              = "3100"
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.loki_tg.arn
  }
}

resource "aws_lb_listener" "mailpit_web_listener" {
  load_balancer_arn = aws_lb.internal_alb.arn
  port              = "8025"
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.mailpit_tg.arn
  }
}

# ECS Task Definitions
resource "aws_ecs_task_definition" "safebucket" {
  family                   = "${var.project_name}-${var.environment}-safebucket"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = var.safebucket_cpu
  memory                   = var.safebucket_memory
  execution_role_arn       = aws_iam_role.ecs_execution_role.arn
  task_role_arn            = aws_iam_role.ecs_task_role.arn

  container_definitions = jsonencode([
    {
      name      = "safebucket"
      image     = var.safebucket_image
      essential = true
      
      portMappings = [
        {
          containerPort = 8080
          protocol      = "tcp"
        }
      ]

      environment = [
        {
          name  = "APP__API_URL"
          value = "http://${aws_lb.safebucket_alb.dns_name}"
        },
        {
          name  = "APP__WEB_URL"
          value = "http://${aws_lb.safebucket_alb.dns_name}"
        },
        {
          name  = "APP__PORT"
          value = "8080"
        },
        {
          name  = "APP__STATIC_FILES__ENABLED"
          value = "true"
        },
        {
          name  = "APP__STATIC_FILES__DIRECTORY"
          value = "web/dist"
        },
        {
          name  = "APP__ADMIN_EMAIL"
          value = var.admin_email
        },
        {
          name  = "APP__ALLOWED_ORIGINS"
          value = "http://${aws_lb.safebucket_alb.dns_name}"
        },
        {
          name  = "APP__TRUSTED_PROXIES"
          value = join(",", data.aws_subnets.default.ids)
        },
        {
          name  = "DATABASE__HOST"
          value = split(":", aws_db_instance.main.endpoint)[0]
        },
        {
          name  = "DATABASE__PORT"
          value = "5432"
        },
        {
          name  = "DATABASE__USER"
          value = var.rds_username
        },
        {
          name  = "DATABASE__NAME"
          value = var.rds_database_name
        },
        {
          name  = "DATABASE__SSLMODE"
          value = "require"
        },
        {
          name  = "CACHE__TYPE"
          value = "redis"
        },
        {
          name  = "CACHE__REDIS__HOSTS"
          value = "${aws_elasticache_replication_group.main.primary_endpoint_address}:6379"
        },
        {
          name  = "CACHE__REDIS__TLS_ENABLED"
          value = "true"
        },
        {
          name  = "CACHE__REDIS__TLS_SERVER_NAME"
          value = aws_elasticache_replication_group.main.primary_endpoint_address
        },
        {
          name  = "STORAGE__TYPE"
          value = "aws"
        },
        {
          name  = "STORAGE__AWS__BUCKET_NAME"
          value = aws_s3_bucket.main.bucket
        },
        {
          name  = "STORAGE__AWS__SQS_NAME"
          value = aws_sqs_queue.s3_events.name
        },
        {
          name  = "EVENTS__TYPE"
          value = "aws"
        },
        {
          name  = "EVENTS__AWS__SQS_NAME"
          value = aws_sqs_queue.s3_events.name
        },
        {
          name  = "EVENTS__AWS__BUCKET_NAME"
          value = "zbraaaa"
        },
        {
          name  = "AWS_REGION"
          value = data.aws_region.current.name
        },
        {
          name  = "NOTIFIER__TYPE"
          value = "smtp"
        },
        {
          name  = "NOTIFIER__SMTP__HOST"
          value = aws_lb.internal_alb.dns_name
        },
        {
          name  = "NOTIFIER__SMTP__PORT"
          value = "1025"
        },
        {
          name  = "NOTIFIER__SMTP__USERNAME"
          value = ""
        },
        {
          name  = "NOTIFIER__SMTP__PASSWORD"
          value = ""
        },
        {
          name  = "NOTIFIER__SMTP__SENDER"
          value = var.smtp_sender
        },
        {
          name  = "NOTIFIER__SMTP__ENABLE_TLS"
          value = "false"
        },
        {
          name  = "NOTIFIER__SMTP__SKIP_VERIFY_TLS"
          value = "true"
        },
        {
          name  = "ACTIVITY__TYPE"
          value = "loki"
        },
        {
          name  = "ACTIVITY__ENDPOINT"
          value = "http://${aws_lb.internal_alb.dns_name}:3100"
        },// TODO: Fix
        {
          name  = "ACTIVITY__LOKI__ENDPOINT"
          value = "http://${aws_lb.internal_alb.dns_name}:3100"
        },
        {
          name  = "AUTH__PROVIDERS__KEYS"
          value = "local"
        },
        {
          name  = "AUTH__PROVIDERS__LOCAL__NAME"
          value = "local"
        },
        {
          name  = "AUTH__PROVIDERS__LOCAL__TYPE"
          value = "local"
        }
      ]

      secrets = [
        {
          name      = "APP__JWT_SECRET"
          valueFrom = aws_secretsmanager_secret.jwt_secret.arn
        },
        {
          name      = "APP__ADMIN_PASSWORD"
          valueFrom = aws_secretsmanager_secret.admin_password.arn
        },
        {
          name      = "DATABASE__PASSWORD"
          valueFrom = aws_secretsmanager_secret.db_password.arn
        },
        {
          name      = "CACHE__REDIS__PASSWORD"
          valueFrom = aws_secretsmanager_secret.redis_auth_token.arn
        }
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = aws_cloudwatch_log_group.safebucket_logs.name
          awslogs-region        = data.aws_region.current.name
          awslogs-stream-prefix = "ecs"
        }
      }
    }
  ])

  tags = {
    Name        = "${var.project_name}-${var.environment}-safebucket-task"
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_ecs_task_definition" "loki" {
  family                   = "${var.project_name}-${var.environment}-loki"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = var.loki_cpu
  memory                   = var.loki_memory
  execution_role_arn       = aws_iam_role.ecs_execution_role.arn
  task_role_arn            = aws_iam_role.loki_task_role.arn

  container_definitions = jsonencode([
    {
      name      = "loki"
      image     = var.loki_image
      essential = true
      
      portMappings = [
        {
          containerPort = 3100
          protocol      = "tcp"
        }
      ]

      command = [
        "-config.expand-env=true",
        "-config.file=/tmp/loki-config.yaml"
      ]

      entryPoint = [
        "sh",
        "-c",
        <<-EOT
          cat > /tmp/loki-config.yaml << 'EOF'
${local.loki_config}
EOF
          exec /usr/bin/loki -config.expand-env=true -config.file=/tmp/loki-config.yaml
        EOT
      ]

      environment = [
        {
          name  = "LOKI_S3_BUCKET"
          value = aws_s3_bucket.loki.bucket
        },
        {
          name  = "AWS_REGION"
          value = data.aws_region.current.name
        }
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = aws_cloudwatch_log_group.loki_logs.name
          awslogs-region        = data.aws_region.current.name
          awslogs-stream-prefix = "ecs"
        }
      }

      healthCheck = {
        command = [
          "CMD-SHELL",
          "wget --no-verbose --tries=1 --spider http://localhost:3100/ready || exit 1"
        ]
        interval    = 30
        timeout     = 5
        retries     = 3
        startPeriod = 60
      }
    }
  ])


  tags = {
    Name        = "${var.project_name}-${var.environment}-loki-task"
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_ecs_task_definition" "mailpit" {
  family                   = "${var.project_name}-${var.environment}-mailpit"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = var.mailpit_cpu
  memory                   = var.mailpit_memory
  execution_role_arn       = aws_iam_role.ecs_execution_role.arn
  task_role_arn            = aws_iam_role.ecs_task_role.arn

  container_definitions = jsonencode([
    {
      name      = "mailpit"
      image     = var.mailpit_image
      essential = true
      
      portMappings = [
        {
          containerPort = 8025
          protocol      = "tcp"
        },
        {
          containerPort = 1025
          protocol      = "tcp"
        }
      ]

      environment = [
        {
          name  = "MP_MAX_MESSAGES"
          value = "500"
        },
        {
          name  = "MP_DATABASE"
          value = "/tmp/mailpit.db"
        },
        {
          name  = "MP_SMTP_AUTH_ALLOW_INSECURE"
          value = "true"
        }
      ]


      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = aws_cloudwatch_log_group.mailpit_logs.name
          awslogs-region        = data.aws_region.current.name
          awslogs-stream-prefix = "ecs"
        }
      }

      healthCheck = {
        command = [
          "CMD-SHELL",
          "wget --no-verbose --tries=1 --spider http://localhost:8025/ || exit 1"
        ]
        interval    = 30
        timeout     = 5
        retries     = 3
        startPeriod = 30
      }
    }
  ])


  tags = {
    Name        = "${var.project_name}-${var.environment}-mailpit-task"
    Environment = var.environment
    Project     = var.project_name
  }
}