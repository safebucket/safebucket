# Secrets Manager for sensitive configuration
resource "aws_secretsmanager_secret" "jwt_secret" {
  name                    = "${var.project_name}-${var.environment}-jwt-secret"
  description             = "JWT secret for SafeBucket application"
  recovery_window_in_days = 0

  tags = {
    Name        = "${var.project_name}-${var.environment}-jwt-secret"
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_secretsmanager_secret_version" "jwt_secret" {
  secret_id     = aws_secretsmanager_secret.jwt_secret.id
  secret_string = var.jwt_secret
}

resource "aws_secretsmanager_secret" "admin_password" {
  name                    = "${var.project_name}-${var.environment}-admin-password"
  description             = "Admin password for SafeBucket application"
  recovery_window_in_days = 0

  tags = {
    Name        = "${var.project_name}-${var.environment}-admin-password"
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_secretsmanager_secret_version" "admin_password" {
  secret_id     = aws_secretsmanager_secret.admin_password.id
  secret_string = var.admin_password
}

resource "aws_secretsmanager_secret" "db_password" {
  name                    = "${var.project_name}-${var.environment}-db-password"
  description             = "Database password for SafeBucket"
  recovery_window_in_days = 0

  tags = {
    Name        = "${var.project_name}-${var.environment}-db-password"
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_secretsmanager_secret_version" "db_password" {
  secret_id     = aws_secretsmanager_secret.db_password.id
  secret_string = var.rds_password
}

resource "aws_secretsmanager_secret" "redis_auth_token" {
  name                    = "${var.project_name}-${var.environment}-redis-auth-token"
  description             = "Redis authentication token for SafeBucket"
  recovery_window_in_days = 0

  tags = {
    Name        = "${var.project_name}-${var.environment}-redis-auth-token"
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_secretsmanager_secret_version" "redis_auth_token" {
  secret_id     = aws_secretsmanager_secret.redis_auth_token.id
  secret_string = var.redis_auth_token
}