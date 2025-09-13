# Loki ECS Service
resource "aws_ecs_service" "loki" {
  name                              = "${var.project_name}-${var.environment}-loki"
  cluster                          = aws_ecs_cluster.safebucket_cluster.id
  task_definition                  = aws_ecs_task_definition.loki.arn
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
    target_group_arn = aws_lb_target_group.loki_tg.arn
    container_name   = "loki"
    container_port   = 3100
  }

  depends_on = [
    aws_lb_listener.loki_listener
  ]

  deployment_circuit_breaker {
    enable   = false
    rollback = false
  }

  tags = {
    Name        = "${var.project_name}-${var.environment}-loki-service"
    Environment = var.environment
    Project     = var.project_name
  }

  # Temporarily removed ignore_changes to allow task definition updates
  # lifecycle {
  #   ignore_changes = [task_definition]
  # }
}