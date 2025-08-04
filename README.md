<h1 align="center">
  <a href="https://safebucket.io"><img src="./assets/safebucket_banner.png" alt="OpenCTI"></a>
</h1>

## Introduction

Safebucket is an open-source secure cloud storage management platform designed to unify multi-cloud storage operations under a single interface. Built for organizations that need robust file management across AWS S3, Google Cloud Storage, and MinIO with enterprise-grade security and compliance features.

### Core Capabilities

- **Multi-Provider Storage**: Seamlessly manage files across AWS S3, GCP Cloud Storage, and MinIO through a unified API
- **Role-Based Access Control**: Granular permissions with owner, contributor, and viewer roles using Casbin RBAC
- **Real-Time Activity Tracking**: Comprehensive audit trails with Loki integration for compliance and monitoring  
- **User Invitation System**: Email-based user onboarding with challenge-response authentication
- **Modern Web Interface**: React-based dashboard with file browser, upload progress tracking, and activity monitoring
- **Event-Driven Architecture**: Real-time notifications via JetStream, GCP Pub/Sub, or AWS SQS

### Key Features

- **Cloud-Agnostic**: Switch between storage providers without changing your workflow
- **Secure by Design**: JWT authentication, OAuth integration (Google, Apple), and Argon2id password hashing
- **Developer-Friendly**: RESTful API with comprehensive documentation and modular architecture
- **Scalable**: An event-driven messaging system handles high-throughput operations
- **Auditable**: Complete activity logging for regulatory compliance and security monitoring

SafeBucket addresses the complexity of managing distributed cloud storage by providing a centralized platform that maintains security, observability, and ease of use across multiple cloud providers.


### Storage configuration

#### Minio

```yaml
storage:
  type: minio
  minio:
    bucket_name: safebucket
    endpoint: localhost:9000
    client_id: minio-root-user
    client_secret: minio-root-password
    type: jetstream
    jetstream:
      topic_name: safebucket:notifications
      host: localhost
      port: 4222
```

#### GCP

```yaml
storage:
  type: gcp
  gcp:
    bucket_name: safebucket-gcp
    project_id: project-id
    subscription_name: safebucket-bucket-events-sub
    topic_name: safebucket-bucket-events
```

```
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/gcs.json
```

#### AWS

```yaml
storage:
  type: aws
  aws:
    bucket_name: safebucket
    sqs_name: safebucket-sqs
```

```
export AWS_ACCESS_KEY_ID=access_key
export AWS_SECRET_ACCESS_KEY=secret_access_key
export AWS_REGION=region
```

### Events configuration

#### Jetstream

```yaml
events:
  type: jetstream
  jetstream:
    topic_name: safebucket:notifications
    host: localhost
    port: 4222
```

#### GCP

```yaml
events:
  type: gcp
  gcp:
    project_id: project-id
    subscription_name: safebucket-notifications-sub
    topic_name: safebucket-notifications
```

#### AWS

```yaml
events:
  type: aws
  aws:
    region: region
    account_id: account_id
    sqs_name: safebucket-sqs
```
