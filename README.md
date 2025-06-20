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
    project_id: atomic-kit-462909-u2
    subscription_name: safebucket-sub
    topic_name: safebucket-notifications
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
    project_id: atomic-kit-462909-u2
    subscription_name: safebucket-mail-notifs-sub
    topic_name: safebucket-mail-notifs
```
