<h1 align="center">
  <a href="https://safebucket.io"><img src="./assets/safebucket_banner.png" alt="SafeBucket"></a>
</h1>

## Introduction

Safebucket is an open-source secure file sharing platform designed to share files in an easy and secure way, integrating
with different cloud providers. Built for individuals and organizations that need to collaborate on files with robust
security, flexible access controls, and seamless multi-cloud support across AWS S3, Google Cloud Storage, and MinIO.

![SafeBucket Homepage](./assets/homepage.png)

## Why Safebucket?

Safebucket eliminates the complexity of secure file sharing by providing a lightweight, stateless solution that
integrates seamlessly with your existing infrastructure.
Plug in your preferred auth providers and eliminate the need for local logins - your users can share files using their
existing corporate identities.

## Features

- 🔒 **Secure File Sharing**: Create a bucket to start sharing files and folders with colleagues, customers, and teams
- 👥 **Role-Based Access Control**: Fine grained sharing permissions with owner, contributor, and viewer roles
- 🔐 **SSO Integration**: Single sign-on with any/multiple auth providers and manage their sharing capabilities
- 📧 **User Invitation System**: Invite external collaborators via email
- 📊 **Real-Time Activity Tracking**: Monitor file sharing activity with comprehensive audit trails
- ☁️ **Multi-Storage Integration**: Store and share files across AWS S3, GCP Cloud Storage, or MinIO
- 🚀 **Highly Scalable**: Event-driven and cloud native architecture for high-performance operations

## Quick Start

```bash
docker compose up -d
```

## Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the Apache 2.0 - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with ❤️ using Go and React
- UI components by [Radix UI](https://radix-ui.com) and [shadcn/ui](https://ui.shadcn.com)
- Database ORM by [Gorm](https://gorm.io/index.html)
- Database migrations by [Goose](https://github.com/pressly/goose)
- Pub/sub integrations by [Watermill](https://watermill.io)
- Configuration management by [Koanf](https://github.com/knadh/koanf)
- Icons by [Lucide](https://lucide.dev)

## Support

- 🐛 Issues: [GitHub Issues](https://github.com/safebucket/safebucket/issues)

#

# Task: Refactor Trash Management System to Async PATCH-based API

## Context

The bucket object handling system was recently refactored using IDs instead of names for folders and files
We need to update the trash/deletion workflow to align with new patterns while maintaining backward compatibility with
the marker-based trash policy.

## Current State Analysis Needed

Before making changes, please:

1. Locate and analyze the current trash implementation:
    - Current HTTP endpoints for trash operations
    - How files vs folders are currently handled
    - Where marker-based deletion is implemented
    - Current trash policy logic

2. Identify the message queue infrastructure in use

## Requirements

### 1. PATCH-Based API Endpoints

Create/modify endpoints to use PATCH verb for trash operations:

**Endpoint 1: Move to Trash**

```
PATCH /api/v1/buckets/{id}/files/{id}
PATCH /api/v1/buckets/{id}/folders/{id}
```

**Endpoint 2: Restore from Trash**

```
PATCH /api/v1/buckets/{id}/files/{id}
PATCH /api/v1/buckets/{id}/folders/{id}
```

Endpoint 3: Modify the List bucket to use the MDW available here to filtered list fiels based on their
status (https://github.com/safebucket/safebucket/tree/feature/add-path-validation):

```
GET  /api/v1/buckets/{id}
```

### 2. Asynchronous Folder Processing

- **Files**: Handle synchronously (immediate response)
- **Folders**: Process asynchronously via queue
    - Batch size: 1000
    - Use existing queue infrastructure

### 3. Preserve Marker-Based System

- Keep the existing marker deletion mechanism
- Ensure trash policy continues to work with markers
- Document how markers interact with new PATCH workflow

### 4. Race Condition Analysis

Specifically analyze and handle:

- **Concurrent PATCH requests** on same object (trash + restore)
- **Folder deletion while child objects** are being modified
- **Trash policy execution** during active trash/restore operations
- **Queue message ordering** for nested folder structures
- **Optimistic locking** strategy for status transitions

Implement appropriate:

- Database-level locking or versioning
- Idempotency keys for API operations
- Conflict resolution strategy (document chosen approach)

## Implementation Guidelines

1. **Error Handling**:
    - Return 409 Conflict for race conditions
    - Handle queue failures gracefully with retry logic
    - Implement dead letter queue for failed operations
    -

## Deliverables

1. Updated API handlers with PATCH endpoints
2. Async queue worker for folder processing
3. Race condition analysis document with mitigations
4. Updated tests
5. Migration plan (if schema changes needed)
