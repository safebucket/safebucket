# Create Loki configuration as a local file that will be baked into the task definition

locals {
  loki_config = <<-EOT
auth_enabled: false

server:
  http_listen_port: 3100
  grpc_listen_port: 9096
  log_level: info

common:
  storage:
    s3:
      bucketnames: ${aws_s3_bucket.loki.bucket}
      region: ${data.aws_region.current.name}
      s3forcepathstyle: false
  replication_factor: 1
  ring:
    instance_addr: 127.0.0.1
    kvstore:
      store: inmemory

ingester:
  chunk_idle_period: 30m
  chunk_retain_period: 1m
  max_chunk_age: 1h
  chunk_target_size: 1048576
  wal:
    enabled: false

query_range:
  results_cache:
    cache:
      embedded_cache:
        enabled: true
        max_size_mb: 512
        ttl: 24h

schema_config:
  configs:
    - from: 2024-01-01
      store: tsdb
      object_store: s3
      schema: v13
      index:
        prefix: loki_index_
        period: 24h

storage_config:
  tsdb_shipper:
    active_index_directory: /loki/tsdb-index
    cache_location: /loki/tsdb-cache
    cache_ttl: 24h
    index_gateway_client:
      server_address: 127.0.0.1:9095
  aws:
    bucketnames: ${aws_s3_bucket.loki.bucket}
    region: ${data.aws_region.current.name}
    s3forcepathstyle: false

compactor:
  working_directory: /tmp/compactor
  compaction_interval: 1h
  retention_enabled: true
  retention_delete_delay: 2h
  delete_request_store: s3

limits_config:
  allow_structured_metadata: true
  discover_service_name: [ domain obj_type action ]

ruler:
  storage:
    type: local
    local:
      directory: /tmp/rules
  rule_path: /tmp/rules
  ring:
    kvstore:
      store: inmemory
  enable_api: true

analytics:
  reporting_enabled: false
  EOT
}