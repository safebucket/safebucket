<h1 align="center">
  <a href="https://safebucket.io"><img src="./assets/safebucket_banner.png" alt="OpenCTI"></a>
</h1>

## Introduction

SafeBucket is an open-source secure file sharing platform designed to share files in an easy and secure way, integrating with different cloud providers. Built for individuals and organizations that need to collaborate on files with robust security, flexible access controls, and seamless multi-cloud support across AWS S3, Google Cloud Storage, and MinIO.

### Core Capabilities

- **Secure File Sharing**: Share files and folders with colleagues, clients, and teams through secure bucket-based collaboration
- **Multi-Provider Integration**: Store and share files across AWS S3, GCP Cloud Storage, and MinIO without vendor lock-in
- **Role-Based Access Control**: Granular sharing permissions with owner, contributor, and viewer roles using Casbin RBAC
- **User Invitation System**: Invite collaborators via email with secure role-based access to shared buckets
- **Real-Time Activity Tracking**: Monitor file sharing activity with comprehensive audit trails via Loki integration
- **Modern Sharing Interface**: Intuitive React-based dashboard with drag-and-drop uploads, file previews, and activity monitoring

### Key Features

- **Easy File Sharing**: Create secure buckets and invite users to collaborate on files with flexible permission levels
- **Cloud-Agnostic Storage**: Share files from any supported storage provider without workflow changes
- **Secure by Design**: JWT authentication, OAuth integration (Google, Apple), and Argon2id password hashing
- **Collaborative Workflows**: Real-time file sharing with upload progress tracking and activity notifications
- **Privacy Focused**: Complete control over who can access shared files with detailed audit logging
- **Developer-Friendly**: RESTful API for building custom file sharing integrations

SafeBucket simplifies secure file sharing by providing an intuitive platform that combines the convenience of modern collaboration tools with the security and flexibility of self-hosted, multi-cloud storage solutions.

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
