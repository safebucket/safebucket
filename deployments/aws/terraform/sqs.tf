# SafeBucket SQS Event Queue Resources

# SQS Queue for S3 events
resource "aws_sqs_queue" "s3_events" {
  name                       = var.s3_event_queue_name
  message_retention_seconds  = 1209600 # 14 days
  visibility_timeout_seconds = 30
  receive_wait_time_seconds  = 20

  tags = merge(local.common_tags, {
    Name = "${var.project_name}-s3-events"
  })
}


# SQS Queue for application notifications
resource "aws_sqs_queue" "notifications" {
  name                       = var.notification_queue_name
  message_retention_seconds  = 1209600 # 14 days
  visibility_timeout_seconds = 30
  receive_wait_time_seconds  = 20

  tags = merge(local.common_tags, {
    Name = "${var.project_name}-notifications"
  })
}

# SQS Policy to allow SafeBucket role to access S3 events queue
resource "aws_sqs_queue_policy" "s3_events_safebucket_access" {
  queue_url = aws_sqs_queue.s3_events.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Service = "s3.amazonaws.com"
        }
        Action   = "sqs:SendMessage"
        Resource = aws_sqs_queue.s3_events.arn
        Condition = {
          ArnEquals = {
            "aws:SourceArn" = aws_s3_bucket.main.arn
          }
        }
      },
      {
        Effect = "Allow"
        Principal = {
          AWS = aws_iam_role.safebucket_app.arn
        }
        Action = [
          "sqs:ReceiveMessage",
          "sqs:GetQueueAttributes",
          "sqs:GetQueueUrl"
        ]
        Resource = aws_sqs_queue.s3_events.arn
      }
    ]
  })
}

# SQS Policy to allow SafeBucket role to access notifications queue
resource "aws_sqs_queue_policy" "notifications_safebucket_access" {
  queue_url = aws_sqs_queue.notifications.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          AWS = aws_iam_role.safebucket_app.arn
        }
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