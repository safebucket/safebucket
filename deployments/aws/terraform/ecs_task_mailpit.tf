# Mailpit ECS Task Definition
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