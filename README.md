# safebucket

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
