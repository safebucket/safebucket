# SafeBucket S3 Storage Resources

# S3 Bucket for SafeBucket storage
resource "aws_s3_bucket" "storage" {
  bucket = var.s3_bucket_name

  tags = merge(local.common_tags, {
    Name = "${var.project_name}-storage"
  })
}

# S3 Bucket security settings
resource "aws_s3_bucket_public_access_block" "storage" {
  bucket                  = aws_s3_bucket.storage.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# S3 Bucket encryption
resource "aws_s3_bucket_server_side_encryption_configuration" "storage" {
  bucket = aws_s3_bucket.storage.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# S3 Bucket CORS configuration
resource "aws_s3_bucket_cors_configuration" "storage" {
  bucket = aws_s3_bucket.storage.id

  cors_rule {
    allowed_headers = ["*"]
    allowed_methods = ["GET", "POST", "PUT", "DELETE"]
    allowed_origins = var.s3_cors_allowed_origins
    expose_headers  = []
    max_age_seconds = 3000
  }
}

# S3 Bucket notification configuration
resource "aws_s3_bucket_notification" "storage" {
  bucket = aws_s3_bucket.storage.id

  queue {
    queue_arn = aws_sqs_queue.s3_events.arn
    events    = ["s3:ObjectCreated:*", "s3:ObjectRemoved:*"]
    id        = "safebucket-s3-events"
  }

  depends_on = [aws_sqs_queue_policy.s3_events_safebucket_access]
}