# Loki ECS Task Definition
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