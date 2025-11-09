# SafeBucket Configuration Variables

This document provides a comprehensive reference for all configuration variables in SafeBucket, including their file
format, environment variable equivalents, default values, and descriptions.

## Configuration Overview

SafeBucket uses a hierarchical configuration system that supports:

- **YAML file configuration** with automatic search paths
- **Environment variable overrides** using double underscore (`__`) delimiters
- **Validation** with automatic startup failure on invalid configuration
- **Multi-provider abstractions** for storage, caching, and messaging

## Configuration Variables Reference

| File Variable                                   | Environment Variable                                | Default Value | Required | Example Value                                        | Description                                       |
|-------------------------------------------------|-----------------------------------------------------|---------------|----------|------------------------------------------------------|---------------------------------------------------|
| **App Configuration**                           |
| `app.log_level`                                 | `APP__LOG_LEVEL`                                    | `info`        | ❌        | `info`                                               | Set the app log level                             |
| `app.admin_email`                               | `APP__ADMIN_EMAIL`                                  | -             | ✅        | `admin@safebucket.io`                                | Administrator email address (must be valid email) |
| `app.admin_password`                            | `APP__ADMIN_PASSWORD`                               | -             | ✅        | `SecurePassword123!`                                 | Administrator password                            |
| `app.api_url`                                   | `APP__API_URL`                                      | -             | ✅        | `http://localhost:8080`                              | API base URL                                      |
| `app.allowed_origins`                           | `APP__ALLOWED_ORIGINS`                              | -             | ✅        | `["http://localhost:3000", "http://127.0.0.1:3000"]` | CORS allowed origins array                        |
| `app.jwt_secret`                                | `APP__JWT_SECRET`                                   | -             | ✅        | `6n5o+dFncio8gQA4jt7pUJrJz92WrqD25zXAa8ashxA`        | JWT signing secret                                |
| `app.port`                                      | `APP__PORT`                                         | `8080`        | ❌        | `3000`                                               | Server port (80-65535)                            |
| `app.web_url`                                   | `APP__WEB_URL`                                      | -             | ✅        | `http://localhost:3000`                              | Frontend web URL                                  |
| `app.trusted_proxies`                           | `APP__TRUSTED_PROXIES`                              | -             | ✅        | `["127.0.0.1", "::1"]`                               | Trusted proxy IPs array                           |
| `app.static_files.enabled`                      | `APP__STATIC_FILES__ENABLED`                        | `true`        | ❌        | `false`                                              | Enable static file serving                        |
| `app.static_files.directory`                    | `APP__STATIC_FILES__DIRECTORY`                      | `web/dist`    | ❌        | `public/build`                                       | Static files directory path                       |
| **Database Configuration**                      |
| `database.host`                                 | `DATABASE__HOST`                                    | -             | ✅        | `localhost`                                          | Database hostname                                 |
| `database.port`                                 | `DATABASE__PORT`                                    | `5432`        | ❌        | `5433`                                               | Database port (80-65535)                          |
| `database.user`                                 | `DATABASE__USER`                                    | -             | ✅        | `safebucket`                                         | Database username                                 |
| `database.password`                             | `DATABASE__PASSWORD`                                | -             | ✅        | `mySecretPassword`                                   | Database password                                 |
| `database.name`                                 | `DATABASE__NAME`                                    | -             | ✅        | `safebucket_prod`                                    | Database name                                     |
| `database.sslmode`                              | `DATABASE__SSLMODE`                                 | -             | ❌        | `require`                                            | SSL connection mode                               |
| **Auth Configuration**                          |
| `auth.providers.{name}.name`                    | `AUTH__PROVIDERS__{NAME}__NAME`                     | -             | ✅*       | `Google`                                             | Auth provider display name                        |
| `auth.providers.{name}.type`                    | `AUTH__PROVIDERS__{NAME}__TYPE`                     | -             | ✅*       | `oidc` or `local`                                    | Auth provider type                                |
| `auth.providers.{name}.client_id`               | `AUTH__PROVIDERS__{NAME}__CLIENT_ID`                | -             | ✅*       | `123456789.apps.googleusercontent.com`               | OAuth client ID (OIDC only)                       |
| `auth.providers.{name}.client_secret`           | `AUTH__PROVIDERS__{NAME}__CLIENT_SECRET`            | -             | ✅*       | `GOCSPX-abcdef123456`                                | OAuth client secret (OIDC only)                   |
| `auth.providers.{name}.issuer`                  | `AUTH__PROVIDERS__{NAME}__ISSUER`                   | -             | ✅*       | `https://accounts.google.com`                        | OIDC issuer URL (OIDC only)                       |
| `auth.providers.{name}.sharing.enabled`         | `AUTH__PROVIDERS__{NAME}__SHARING__ENABLED`         | `true`        | ❌        | `false`                                              | Enable domain sharing                             |
| `auth.providers.{name}.sharing.allowed_domains` | `AUTH__PROVIDERS__{NAME}__SHARING__ALLOWED_DOMAINS` | -             | ❌        | `["example.com", "company.org"]`                     | Allowed domains for sharing                       |
| **Cache Configuration**                         |
| `cache.type`                                    | `CACHE__TYPE`                                       | -             | ✅        | `redis`                                              | Cache type: `redis` or `valkey`                   |
| `cache.redis.hosts`                             | `CACHE__REDIS__HOSTS`                               | -             | ✅*       | `["localhost:6379", "redis-2:6379"]`                 | Redis host addresses array                        |
| `cache.redis.password`                          | `CACHE__REDIS__PASSWORD`                            | -             | ❌        | `redisPassword123`                                   | Redis password                                    |
| `cache.redis.tls_enabled`                       | `CACHE__REDIS__TLS_ENABLED`                         | `false`       | ❌        | `true`                                               | Enable TLS encryption for Redis connections       |
| `cache.redis.tls_server_name`                   | `CACHE__REDIS__TLS_SERVER_NAME`                     | -             | ❌        | `redis-cluster.amazonaws.com`                        | TLS server name for Redis certificate validation  |
| `cache.valkey.hosts`                            | `CACHE__VALKEY__HOSTS`                              | -             | ✅*       | `["localhost:6380"]`                                 | Valkey host addresses array                       |
| `cache.valkey.password`                         | `CACHE__VALKEY__PASSWORD`                           | -             | ❌        | `valkeySecret`                                       | Valkey password                                   |
| `cache.valkey.tls_enabled`                      | `CACHE__VALKEY__TLS_ENABLED`                        | `false`       | ❌        | `true`                                               | Enable TLS encryption for Valkey connections      |
| `cache.valkey.tls_server_name`                  | `CACHE__VALKEY__TLS_SERVER_NAME`                    | -             | ❌        | `valkey-cluster.amazonaws.com`                       | TLS server name for Valkey certificate validation |
| **Storage Configuration**                       |
| `storage.type`                                  | `STORAGE__TYPE`                                     | -             | ✅        | `minio`                                              | Storage provider: `minio`, `gcp`, or `aws`        |
| **MinIO Storage**                               |
| `storage.minio.bucket_name`                     | `STORAGE__MINIO__BUCKET_NAME`                       | -             | ✅*       | `safebucket`                                         | MinIO bucket name                                 |
| `storage.minio.endpoint`                        | `STORAGE__MINIO__ENDPOINT`                          | -             | ✅*       | `localhost:9000`                                     | MinIO internal endpoint URL                       |
| `storage.minio.external_endpoint`               | `STORAGE__MINIO__EXTERNAL_ENDPOINT`                 | -             | ✅*       | `http://localhost:9000`                              | MinIO external endpoint URL (for browser CSP)     |
| `storage.minio.client_id`                       | `STORAGE__MINIO__CLIENT_ID`                         | -             | ✅*       | `minio-root-user`                                    | MinIO access key                                  |
| `storage.minio.client_secret`                   | `STORAGE__MINIO__CLIENT_SECRET`                     | -             | ✅*       | `minio-root-password`                                | MinIO secret key                                  |
| `storage.minio.type`                            | `STORAGE__MINIO__TYPE`                              | -             | ✅*       | `jetstream`                                          | Event type: `jetstream`                           |
| `storage.minio.jetstream.topic_name`            | `STORAGE__MINIO__JETSTREAM__TOPIC_NAME`             | -             | ✅*       | `safebucket:notifications`                           | JetStream topic name                              |
| `storage.minio.jetstream.host`                  | `STORAGE__MINIO__JETSTREAM__HOST`                   | -             | ✅*       | `localhost`                                          | JetStream host                                    |
| `storage.minio.jetstream.port`                  | `STORAGE__MINIO__JETSTREAM__PORT`                   | -             | ✅*       | `4222`                                               | JetStream port                                    |
| **GCP Cloud Storage**                           |
| `storage.gcp.bucket_name`                       | `STORAGE__GCP__BUCKET_NAME`                         | -             | ✅*       | `my-gcp-bucket`                                      | GCP Storage bucket name                           |
| `storage.gcp.project_id`                        | `STORAGE__GCP__PROJECT_ID`                          | -             | ✅*       | `my-gcp-project-123`                                 | GCP project ID                                    |
| `storage.gcp.topic_name`                        | `STORAGE__GCP__TOPIC_NAME`                          | -             | ✅*       | `safebucket-events`                                  | GCP Pub/Sub topic name for storage events         |
| `storage.gcp.subscription_name`                 | `STORAGE__GCP__SUBSCRIPTION_NAME`                   | -             | ✅*       | `safebucket-storage-sub`                             | GCP Pub/Sub subscription name for storage events  |
| **AWS S3 Storage**                              |
| `storage.aws.bucket_name`                       | `STORAGE__AWS__BUCKET_NAME`                         | -             | ✅*       | `my-s3-bucket`                                       | S3 bucket name                                    |
| `storage.aws.external_endpoint`                 | `STORAGE__AWS__EXTERNAL_ENDPOINT`                   | -             | ❌        | `https://my-bucket.s3.us-east-1.amazonaws.com`       | S3 external endpoint URL (for browser CSP)        |
| `storage.aws.sqs_name`                          | `STORAGE__AWS__SQS_NAME`                            | -             | ✅*       | `safebucket-s3-events`                               | SQS queue name for S3 events                      |
| **Events Configuration**                        |
| `events.type`                                   | `EVENTS__TYPE`                                      | -             | ✅        | `jetstream`                                          | Event system: `jetstream`, `gcp`, or `aws`        |
| **JetStream Events**                            |
| `events.jetstream.topic_name`                   | `EVENTS__JETSTREAM__TOPIC_NAME`                     | -             | ✅*       | `safebucket:notifications`                           | JetStream topic name                              |
| `events.jetstream.host`                         | `EVENTS__JETSTREAM__HOST`                           | -             | ✅*       | `localhost`                                          | JetStream host                                    |
| `events.jetstream.port`                         | `EVENTS__JETSTREAM__PORT`                           | -             | ✅*       | `4222`                                               | JetStream port                                    |
| **GCP Pub/Sub Events**                          |
| `events.gcp.project_id`                         | `EVENTS__GCP__PROJECT_ID`                           | -             | ✅*       | `my-gcp-project-123`                                 | GCP project ID for Pub/Sub                        |
| `events.gcp.topic_name`                         | `EVENTS__GCP__TOPIC_NAME`                           | -             | ✅*       | `safebucket-events`                                  | GCP Pub/Sub topic name                            |
| `events.gcp.subscription_name`                  | `EVENTS__GCP__SUBSCRIPTION_NAME`                    | -             | ✅*       | `safebucket-events-sub`                              | GCP Pub/Sub subscription name                     |
| **AWS SQS Events**                              |
| `events.aws.name`                               | `EVENTS__AWS__NAME`                                 | -             | ✅*       | `safebucket-notifications`                           | SQS queue name for application events             |
| **Notifier Configuration**                      |
| `notifier.type`                                 | `NOTIFIER__TYPE`                                    | -             | ✅        | `smtp`                                               | Notifier type: `smtp`                             |
| `notifier.smtp.host`                            | `NOTIFIER__SMTP__HOST`                              | -             | ✅*       | `smtp.gmail.com`                                     | SMTP server host                                  |
| `notifier.smtp.port`                            | `NOTIFIER__SMTP__PORT`                              | -             | ✅*       | `587`                                                | SMTP server port                                  |
| `notifier.smtp.username`                        | `NOTIFIER__SMTP__USERNAME`                          | -             | ❌        | `notifications@example.com`                          | SMTP username                                     |
| `notifier.smtp.password`                        | `NOTIFIER__SMTP__PASSWORD`                          | -             | ❌        | `app-password-123`                                   | SMTP password                                     |
| `notifier.smtp.sender`                          | `NOTIFIER__SMTP__SENDER`                            | -             | ✅*       | `notifications@safebucket.io`                        | Email sender address                              |
| `notifier.smtp.enable_tls`                      | `NOTIFIER__SMTP__ENABLE_TLS`                        | `true`        | ❌        | `false`                                              | Enable TLS encryption                             |
| `notifier.smtp.skip_verify_tls`                 | `NOTIFIER__SMTP__SKIP_VERIFY_TLS`                   | `false`       | ❌        | `true`                                               | Skip TLS certificate verification                 |
| **Activity Configuration**                      |
| `activity.type`                                 | `ACTIVITY__TYPE`                                    | -             | ✅        | `loki`                                               | Activity logger: `loki`                           |
| `activity.loki.endpoint`                        | `ACTIVITY__LOKI__ENDPOINT`                          | -             | ✅        | `http://localhost:3100`                              | Loki endpoint URL                                 |

## Special Environment Variables

| Environment Variable    | Description                                                      |
|-------------------------|------------------------------------------------------------------|
| `CONFIG_FILE_PATH`      | Override default config file search paths                        |
| `AUTH__PROVIDERS__KEYS` | Comma-separated list of OAuth provider names for dynamic loading |

## Configuration Loading Process

1. **File Discovery**: Searches in order:
    - `CONFIG_FILE_PATH` environment variable
    - `./config.yaml` (current directory)
    - `templates/config.yaml` (templates directory)

2. **Environment Override**: Environment variables take precedence over file values

3. **Array Processing**: String arrays can be specified as:
    - Comma-separated: `"value1,value2,value3"`
    - Space-separated: `"value1 value2 value3"`
    - Bracketed: `"[value1,value2,value3]"`

4. **Provider Loading**: OAuth providers are dynamically loaded based on `AUTH__PROVIDERS__KEYS`

## Example Configuration

### YAML File (`config.yaml`)

```yaml
app:
  api_url: http://localhost:8080
  web_url: http://localhost:3000
  admin_email: admin@example.com
  admin_password: SecurePassword123
  jwt_secret: your-jwt-secret-key
  port: 8080
  allowed_origins:
    - http://localhost:3000
    - http://127.0.0.1:3000
  trusted_proxies:
    - 127.0.0.1
    - ::1

database:
  host: localhost
  port: 5432
  user: safebucket
  password: password
  name: safebucket
  sslmode: disable

cache:
  type: redis
  redis:
    hosts:
      - localhost:6379
    password: ""
    tls_enabled: false
    tls_server_name: ""

storage:
  type: minio
  minio:
    bucket_name: safebucket
    endpoint: localhost:9000
    client_id: minio
    client_secret: minio123

events:
  type: jetstream
  jetstream:
    topic_name: safebucket:notifications
    host: localhost
    port: 4222
```

### AWS S3 Configuration
```yaml
storage:
  type: aws
  aws:
    bucket_name: safebucket-prod
    sqs_name: safebucket-s3-events

events:
  type: aws
  aws:
    name: safebucket-notifications

# Note: AWS credentials should be provided via:
# - AWS credentials file (~/.aws/credentials)
# - EC2 instance profile
# - Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
# - IAM roles for service accounts (IRSA) in EKS
```

### GCP Cloud Storage Configuration
```yaml
storage:
  type: gcp
  gcp:
    bucket_name: safebucket-prod
    project_id: my-gcp-project-123
    topic_name: safebucket-storage-events
    subscription_name: safebucket-storage-sub

events:
  type: gcp
  gcp:
    project_id: my-gcp-project-123
    topic_name: safebucket-events
    subscription_name: safebucket-events-sub

# Note: GCP credentials should be provided via:
# - GOOGLE_APPLICATION_CREDENTIALS environment variable pointing to service account JSON
# - GCP metadata service (on GCE instances)
# - Workload Identity (on GKE)
```

### AWS ElastiCache with TLS Configuration
```yaml
cache:
  type: redis
  redis:
    hosts:
      - safebucket-cache.amazonaws.com:6379
    password: your-redis-auth-token
    tls_enabled: true
    tls_server_name: safebucket-cache.amazonaws.com

storage:
  type: aws
  aws:
    bucket_name: safebucket-prod
    sqs_name: safebucket-s3-events

events:
  type: aws
  aws:
    name: safebucket-notifications

# Note: AWS credentials and region should be provided via IAM roles
```

### Valkey with TLS Configuration
```yaml
cache:
  type: valkey
  valkey:
    hosts:
      - valkey.example.com:6380
    password: valkey-secure-password
    tls_enabled: true
    tls_server_name: valkey.example.com
```

### Environment Variables

```bash
# App Configuration
export APP__LOG_LEVEL=info
export APP__API_URL="http://localhost:8080"
export APP__WEB_URL="http://localhost:3000"
export APP__ADMIN_EMAIL="admin@example.com"
export APP__ADMIN_PASSWORD="SecurePassword123"
export APP__JWT_SECRET="your-jwt-secret-key"

# Database
export DATABASE__HOST="localhost"
export DATABASE__PASSWORD="password"

# Auth Providers
export AUTH__PROVIDERS__KEYS="local,google,authelia"

# Local Auth Provider (for development)
export AUTH__PROVIDERS__LOCAL__NAME="Local"
export AUTH__PROVIDERS__LOCAL__TYPE="local"

# Google OAuth Provider
export AUTH__PROVIDERS__GOOGLE__NAME="Google"
export AUTH__PROVIDERS__GOOGLE__TYPE="oidc"
export AUTH__PROVIDERS__GOOGLE__CLIENT_ID="your-google-client-id"
export AUTH__PROVIDERS__GOOGLE__CLIENT_SECRET="your-google-client-secret"
export AUTH__PROVIDERS__GOOGLE__ISSUER="https://accounts.google.com"
export AUTH__PROVIDERS__GOOGLE__SHARING__ENABLED="true"

# Authelia OIDC Provider
export AUTH__PROVIDERS__AUTHELIA__NAME="Authelia"
export AUTH__PROVIDERS__AUTHELIA__TYPE="oidc"
export AUTH__PROVIDERS__AUTHELIA__CLIENT_ID="safebucket"
export AUTH__PROVIDERS__AUTHELIA__CLIENT_SECRET="your-authelia-secret"
export AUTH__PROVIDERS__AUTHELIA__ISSUER="https://auth.local"
export AUTH__PROVIDERS__AUTHELIA__SHARING__ENABLED="true"

# AWS Configuration
export STORAGE__TYPE="aws"
export STORAGE__AWS__BUCKET_NAME="safebucket-prod"
export STORAGE__AWS__SQS_NAME="safebucket-s3-events"
export EVENTS__TYPE="aws"
export EVENTS__AWS__NAME="safebucket-notifications"

# AWS Credentials (optional - can use IAM roles)
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_REGION="us-east-1"

# GCP Configuration
export STORAGE__TYPE="gcp"
export STORAGE__GCP__BUCKET_NAME="safebucket-prod"
export STORAGE__GCP_STORAGE__PROJECT_ID="my-gcp-project-123"
export STORAGE__GCP__TOPIC_NAME="safebucket-storage-events"
export STORAGE__GCP__SUBSCRIPTION_NAME="safebucket-storage-sub"
export EVENTS__TYPE="gcp"
export EVENTS__GCP__PROJECT_ID="my-gcp-project-123"
export EVENTS__GCP__TOPIC_NAME="safebucket-events"
export EVENTS__GCP__SUBSCRIPTION_NAME="safebucket-events-sub"

# GCP Credentials
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account.json"
export GOOGLE_CLOUD_PROJECT="my-gcp-project-123"

# Cache Configuration (Redis/Valkey)
export CACHE__TYPE="redis"
export CACHE__REDIS__HOSTS="localhost:6379,redis-2:6379"
export CACHE__REDIS__PASSWORD="redis-auth-token"
export CACHE__REDIS__TLS_ENABLED="false"
export CACHE__REDIS__TLS_SERVER_NAME=""

# AWS ElastiCache with TLS (Production)
export CACHE__TYPE="redis"
export CACHE__REDIS__HOSTS="safebucket-cache.amazonaws.com:6379"
export CACHE__REDIS__PASSWORD="your-redis-auth-token"
export CACHE__REDIS__TLS_ENABLED="true"
export CACHE__REDIS__TLS_SERVER_NAME="safebucket-cache.amazonaws.com"

# Valkey Configuration
export CACHE__TYPE="valkey"
export CACHE__VALKEY__HOSTS="localhost:6380"
export CACHE__VALKEY__PASSWORD="valkey-password"
export CACHE__VALKEY__TLS_ENABLED="true"
export CACHE__VALKEY__TLS_SERVER_NAME="valkey.example.com"
```

## Requirement Legend

- ✅ **Required**: Field must be set or application will fail to start
- ❌ **Optional**: Field has a default value or is not mandatory
- ✅* **Conditionally Required**: Required only if parent type match this provider

## Notes

- **Required fields** must be set or the application will fail to start with validation errors
- **Environment variables** use double underscores (`__`) as delimiters to represent nested YAML structure
- **Array fields** support multiple input formats for flexibility
- **Provider names** in OAuth configuration should be uppercase in environment variables
- **Validation** occurs at startup with detailed error messages for invalid configurations