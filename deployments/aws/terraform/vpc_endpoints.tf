# VPC Endpoints for AWS services
# Note: Using S3 Gateway endpoint only (free tier)
# Interface endpoints removed as ECS tasks have public IPs and use Docker Hub for images

# S3 VPC Endpoint (Gateway type - FREE, improves S3 performance)
resource "aws_vpc_endpoint" "s3" {
  vpc_id          = data.aws_vpc.default.id
  service_name    = "com.amazonaws.${data.aws_region.current.name}.s3"
  vpc_endpoint_type = "Gateway"
  route_table_ids = data.aws_route_tables.default.ids

  tags = merge(local.common_tags, {
    Name = "${var.project_name}-s3-endpoint"
  })
}

# Data source for route tables
data "aws_route_tables" "default" {
  vpc_id = data.aws_vpc.default.id
}