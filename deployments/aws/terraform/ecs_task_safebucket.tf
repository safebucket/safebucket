# SafeBucket ECS Task Definition
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
          value = aws_s3_bucket.main.bucket
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