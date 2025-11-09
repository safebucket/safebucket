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
          # Object operations
          "s3:GetObject",
          "s3:PutObject",
          "s3:DeleteObject",
          "s3:DeleteObjects",
          "s3:HeadObject",
          "s3:GetObjectAttributes",

          # Tagging operations (required for file metadata)
          "s3:GetObjectTagging",
          "s3:PutObjectTagging",

          # List operations
          "s3:ListBucket",

          # Lifecycle operations (CRITICAL - required for trash retention)
          "s3:GetBucketLifecycleConfiguration",
          "s3:PutBucketLifecycleConfiguration"
        ]
        Resource = [
          aws_s3_bucket.main.arn,
          "${aws_s3_bucket.main.arn}/*"
        ]
      },
      {
        # SQS permissions for S3 events queue (read and delete)
        Effect = "Allow"
        Action = [
          "sqs:ReceiveMessage",
          "sqs:DeleteMessage",
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
          "sqs:DeleteMessage",
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