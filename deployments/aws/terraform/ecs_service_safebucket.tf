# SafeBucket ECS Service
resource "aws_ecs_service" "safebucket" {
  name                   = "${var.project_name}-${var.environment}-safebucket"
  cluster                = aws_ecs_cluster.safebucket_cluster.id
  task_definition        = aws_ecs_task_definition.safebucket.arn
  desired_count          = var.safebucket_desired_count
  deployment_minimum_healthy_percent = 50
  deployment_maximum_percent         = 200
  enable_execute_command = var.enable_ecs_exec

  # Capacity provider strategy - supports Spot instances for cost savings
  dynamic "capacity_provider_strategy" {
    for_each = var.enable_spot_instances ? [1] : []
    content {
      capacity_provider = "FARGATE_SPOT"
      weight            = var.spot_instance_percentage
      base              = 0
    }
  }

  dynamic "capacity_provider_strategy" {
    for_each = var.enable_spot_instances && var.spot_instance_percentage < 100 ? [1] : []
    content {
      capacity_provider = "FARGATE"
      weight            = 100 - var.spot_instance_percentage
      base              = 0
    }
  }

  # Default to FARGATE if spot instances are disabled
  dynamic "capacity_provider_strategy" {
    for_each = var.enable_spot_instances ? [] : [1]
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

  load_balancer {
    target_group_arn = aws_lb_target_group.safebucket_tg.arn
    container_name   = "safebucket"
    container_port   = 8080
  }

  depends_on = [
    aws_lb_listener.safebucket_listener,
    aws_db_instance.main,
    aws_elasticache_replication_group.main,
    aws_ecs_service.loki,
    aws_ecs_service.mailpit
  ]

  deployment_circuit_breaker {
    enable   = false
    rollback = false
  }

  # Trigger redeployment when needed
  # Change the redeployment_trigger variable value to force update
  triggers = {
    redeployment = var.redeployment_trigger
  }

  tags = {
    Name        = "${var.project_name}-${var.environment}-safebucket-service"
    Environment = var.environment
    Project     = var.project_name
  }
}

# Auto Scaling for SafeBucket Service
resource "aws_appautoscaling_target" "safebucket_target" {
  count              = var.enable_autoscaling ? 1 : 0
  max_capacity       = var.safebucket_max_capacity
  min_capacity       = var.safebucket_min_capacity
  resource_id        = "service/${aws_ecs_cluster.safebucket_cluster.name}/${aws_ecs_service.safebucket.name}"
  scalable_dimension = "ecs:service:DesiredCount"
  service_namespace  = "ecs"

  tags = {
    Name        = "${var.project_name}-${var.environment}-safebucket-scaling-target"
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_appautoscaling_policy" "safebucket_cpu_policy" {
  count              = var.enable_autoscaling ? 1 : 0
  name               = "${var.project_name}-${var.environment}-safebucket-cpu-scaling"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.safebucket_target[0].resource_id
  scalable_dimension = aws_appautoscaling_target.safebucket_target[0].scalable_dimension
  service_namespace  = aws_appautoscaling_target.safebucket_target[0].service_namespace

  target_tracking_scaling_policy_configuration {
    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageCPUUtilization"
    }
    target_value = 70.0
    scale_in_cooldown  = 300
    scale_out_cooldown = 300
  }
}

resource "aws_appautoscaling_policy" "safebucket_memory_policy" {
  count              = var.enable_autoscaling ? 1 : 0
  name               = "${var.project_name}-${var.environment}-safebucket-memory-scaling"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.safebucket_target[0].resource_id
  scalable_dimension = aws_appautoscaling_target.safebucket_target[0].scalable_dimension
  service_namespace  = aws_appautoscaling_target.safebucket_target[0].service_namespace

  target_tracking_scaling_policy_configuration {
    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageMemoryUtilization"
    }
    target_value = 80.0
    scale_in_cooldown  = 300
    scale_out_cooldown = 300
  }
}