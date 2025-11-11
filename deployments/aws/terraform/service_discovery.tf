# AWS Cloud Map Service Discovery Configuration
# Enables internal DNS-based service discovery for ECS services

# Private DNS namespace for internal service discovery
resource "aws_service_discovery_private_dns_namespace" "internal" {
  name        = "${var.project_name}.local"
  description = "Private DNS namespace for internal service discovery"
  vpc         = data.aws_vpc.default.id

  tags = {
    Name        = "${var.project_name}-${var.environment}-service-discovery"
    Environment = var.environment
    Project     = var.project_name
  }
}

# Service Discovery for Loki
resource "aws_service_discovery_service" "loki" {
  name = "loki"

  dns_config {
    namespace_id = aws_service_discovery_private_dns_namespace.internal.id

    dns_records {
      ttl  = 10
      type = "A"
    }

    routing_policy = "MULTIVALUE"
  }

  health_check_custom_config {
    failure_threshold = 1
  }

  tags = {
    Name        = "${var.project_name}-${var.environment}-loki-discovery"
    Environment = var.environment
    Project     = var.project_name
  }
}

# Service Discovery for Mailpit
resource "aws_service_discovery_service" "mailpit" {
  name = "mailpit"

  dns_config {
    namespace_id = aws_service_discovery_private_dns_namespace.internal.id

    dns_records {
      ttl  = 10
      type = "A"
    }

    routing_policy = "MULTIVALUE"
  }

  health_check_custom_config {
    failure_threshold = 1
  }

  tags = {
    Name        = "${var.project_name}-${var.environment}-mailpit-discovery"
    Environment = var.environment
    Project     = var.project_name
  }
}