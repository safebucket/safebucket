# ECS Task Execution Role
resource "aws_iam_role" "ecs_execution_role" {
  name = "${var.project_name}-${var.environment}-ecs-execution-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ecs-tasks.amazonaws.com"
        }
      }
    ]
  })

  tags = {
    Name        = "${var.project_name}-${var.environment}-ecs-execution-role"
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_iam_role_policy_attachment" "ecs_execution_role_policy" {
  role       = aws_iam_role.ecs_execution_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

# Policy for accessing secrets
resource "aws_iam_policy" "ecs_secrets_policy" {
  name        = "${var.project_name}-${var.environment}-ecs-secrets-policy"
  description = "Policy for ECS tasks to access Secrets Manager"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue"
        ]
        Resource = [
          aws_secretsmanager_secret.jwt_secret.arn,
          aws_secretsmanager_secret.admin_password.arn,
          aws_secretsmanager_secret.db_password.arn,
          aws_secretsmanager_secret.redis_auth_token.arn
        ]
      }
    ]
  })

  tags = {
    Name        = "${var.project_name}-${var.environment}-ecs-secrets-policy"
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_iam_role_policy_attachment" "ecs_execution_secrets_policy" {
  role       = aws_iam_role.ecs_execution_role.name
  policy_arn = aws_iam_policy.ecs_secrets_policy.arn
}

# ECS Task Role (for application permissions)
resource "aws_iam_role" "ecs_task_role" {
  name = "${var.project_name}-${var.environment}-ecs-task-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ecs-tasks.amazonaws.com"
        }
      }
    ]
  })

  tags = {
    Name        = "${var.project_name}-${var.environment}-ecs-task-role"
    Environment = var.environment
    Project     = var.project_name
  }
}

# Policy for SafeBucket application
resource "aws_iam_policy" "safebucket_task_policy" {
  name        = "${var.project_name}-${var.environment}-safebucket-task-policy"
  description = "Policy for SafeBucket ECS task"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          # Object operations
          "s3:GetObject",
          "s3:PutObject",
          "s3:DeleteObject",
          "s3:DeleteObjects",
          "s3:HeadObject",

          # Tagging operations (required for file metadata)
          "s3:GetObjectTagging",
          "s3:PutObjectTagging",

          # List operations
          "s3:ListBucket",

          # Lifecycle operations (CRITICAL - required for trash retention)
          # Both old and new API actions are included for compatibility
          "s3:GetBucketLifecycleConfiguration",
          "s3:PutBucketLifecycleConfiguration",
          "s3:GetLifecycleConfiguration",
          "s3:PutLifecycleConfiguration"
        ]
        Resource = [
          aws_s3_bucket.main.arn,
          "${aws_s3_bucket.main.arn}/*"
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "sqs:SendMessage",
          "sqs:ReceiveMessage",
          "sqs:DeleteMessage",
          "sqs:GetQueueAttributes",
          "sqs:GetQueueUrl"
        ]
        Resource = [
          aws_sqs_queue.s3_events.arn,
          aws_sqs_queue.notifications.arn,
          aws_sqs_queue.object_deletion.arn
        ]
      }
    ]
  })

  tags = {
    Name        = "${var.project_name}-${var.environment}-safebucket-task-policy"
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_iam_role_policy_attachment" "safebucket_task_policy" {
  role       = aws_iam_role.ecs_task_role.name
  policy_arn = aws_iam_policy.safebucket_task_policy.arn
}

# Loki Task Role
resource "aws_iam_role" "loki_task_role" {
  name = "${var.project_name}-${var.environment}-loki-task-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ecs-tasks.amazonaws.com"
        }
      }
    ]
  })

  tags = {
    Name        = "${var.project_name}-${var.environment}-loki-task-role"
    Environment = var.environment
    Project     = var.project_name
  }
}

# Policy for Loki to access S3 storage
resource "aws_iam_policy" "loki_task_policy" {
  name        = "${var.project_name}-${var.environment}-loki-task-policy"
  description = "Policy for Loki ECS task to access S3 storage"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject",
          "s3:DeleteObject",
          "s3:ListBucket"
        ]
        Resource = [
          aws_s3_bucket.loki.arn,
          "${aws_s3_bucket.loki.arn}/*"
        ]
      }
    ]
  })

  tags = {
    Name        = "${var.project_name}-${var.environment}-loki-task-policy"
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_iam_role_policy_attachment" "loki_task_policy" {
  role       = aws_iam_role.loki_task_role.name
  policy_arn = aws_iam_policy.loki_task_policy.arn
}

# Policy for ECS Exec (debugging)
resource "aws_iam_policy" "ecs_exec_policy" {
  count       = var.enable_ecs_exec ? 1 : 0
  name        = "${var.project_name}-${var.environment}-ecs-exec-policy"
  description = "Policy for ECS Exec functionality"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ssmmessages:CreateControlChannel",
          "ssmmessages:CreateDataChannel",
          "ssmmessages:OpenControlChannel",
          "ssmmessages:OpenDataChannel"
        ]
        Resource = "*"
      }
    ]
  })

  tags = {
    Name        = "${var.project_name}-${var.environment}-ecs-exec-policy"
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_iam_role_policy_attachment" "ecs_exec_policy" {
  count      = var.enable_ecs_exec ? 1 : 0
  role       = aws_iam_role.ecs_task_role.name
  policy_arn = aws_iam_policy.ecs_exec_policy[0].arn
}