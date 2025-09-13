# SafeBucket IAM Resources

# IAM Role for SafeBucket Application
resource "aws_iam_role" "safebucket_app" {
  name = "${var.project_name}-application-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = [
            "ec2.amazonaws.com",
            "ecs-tasks.amazonaws.com"
          ]
        }
      }
    ]
  })

  tags = local.common_tags
}

# IAM Policy for SafeBucket Application
resource "aws_iam_role_policy" "safebucket_app" {
  name = "${var.project_name}-application-policy"
  role = aws_iam_role.safebucket_app.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        # S3 permissions for signed URLs and bucket operations
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject",
          "s3:DeleteObject",
          "s3:ListBucket",
          "s3:GetObjectAttributes"
        ]
        Resource = [
          aws_s3_bucket.main.arn,
          "${aws_s3_bucket.main.arn}/*"
        ]
      },
      {
        # SQS permissions for S3 events queue (read only)
        Effect = "Allow"
        Action = [
          "sqs:ReceiveMessage",
          "sqs:GetQueueAttributes",
          "sqs:GetQueueUrl"
        ]
        Resource = aws_sqs_queue.s3_events.arn
      },
      {
        # SQS permissions for notifications queue (publish and read)
        Effect = "Allow"
        Action = [
          "sqs:SendMessage",
          "sqs:ReceiveMessage",
          "sqs:GetQueueAttributes",
          "sqs:GetQueueUrl"
        ]
        Resource = aws_sqs_queue.notifications.arn
      }
    ]
  })
}

# Instance Profile for EC2 instances
resource "aws_iam_instance_profile" "safebucket_app" {
  name = "${var.project_name}-instance-profile"
  role = aws_iam_role.safebucket_app.name

  tags = local.common_tags
}