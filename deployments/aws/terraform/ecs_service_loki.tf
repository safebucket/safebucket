# Loki ECS Service
resource "aws_ecs_service" "loki" {
  name                   = "${var.project_name}-${var.environment}-loki"
  cluster                = aws_ecs_cluster.safebucket_cluster.id
  task_definition        = aws_ecs_task_definition.loki.arn
  desired_count          = 1
  deployment_minimum_healthy_percent = 0
  deployment_maximum_percent         = 100
  enable_execute_command = var.enable_ecs_exec

  # Capacity provider strategy - supports Spot instances for cost savings
  dynamic "capacity_provider_strategy" {
    for_each = var.enable_loki_spot_instances ? [1] : []
    content {
      capacity_provider = "FARGATE_SPOT"
      weight            = var.loki_spot_instance_percentage
      base              = 0
    }
  }

  dynamic "capacity_provider_strategy" {
    for_each = var.enable_loki_spot_instances && var.loki_spot_instance_percentage < 100 ? [1] : []
    content {
      capacity_provider = "FARGATE"
      weight            = 100 - var.loki_spot_instance_percentage
      base              = 0
    }
  }

  # Default to FARGATE if spot instances are disabled
  dynamic "capacity_provider_strategy" {
    for_each = var.enable_loki_spot_instances ? [] : [1]
    content {
      capacity_provider = "FARGATE"
      weight            = 100
      base              = 1
    }
  }

  network_configuration {
    security_groups  = [aws_security_group.ecs_tasks.id]
    subnets          = data.aws_subnets.default.ids
    assign_public_ip = true
  }

  # Service Discovery for internal communication
  service_registries {
    registry_arn = aws_service_discovery_service.loki.arn
  }

  depends_on = [
    aws_service_discovery_service.loki
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