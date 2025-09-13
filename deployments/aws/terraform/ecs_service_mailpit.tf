# Mailpit ECS Service
resource "aws_ecs_service" "mailpit" {
  name                              = "${var.project_name}-${var.environment}-mailpit"
  cluster                          = aws_ecs_cluster.safebucket_cluster.id
  task_definition                  = aws_ecs_task_definition.mailpit.arn
  desired_count                    = 1
  deployment_minimum_healthy_percent = 0
  deployment_maximum_percent         = 100
  enable_execute_command            = var.enable_ecs_exec

  capacity_provider_strategy {
    capacity_provider = "FARGATE"
    weight           = 100
  }

  network_configuration {
    security_groups  = [aws_security_group.ecs_tasks.id]
    subnets          = data.aws_subnets.default.ids
    assign_public_ip = true
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.mailpit_tg.arn
    container_name   = "mailpit"
    container_port   = 8025
  }

  depends_on = [
    aws_lb_listener.mailpit_web_listener
  ]

  deployment_circuit_breaker {
    enable   = false
    rollback = false
  }

  tags = {
    Name        = "${var.project_name}-${var.environment}-mailpit-service"
    Environment = var.environment
    Project     = var.project_name
  }
}