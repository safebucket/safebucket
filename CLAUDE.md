# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

SafeBucket is an open-source secure file sharing platform with multi-cloud storage support (AWS S3, GCP Cloud Storage,
MinIO), featuring JWT authentication, OAuth SSO integration, role-based access control, event-driven architecture, and
comprehensive audit trails.

## Architecture Summary

### Backend (Go 1.24)

- **Framework**: Chi v5 HTTP router with composable middleware
- **Database**: GORM with PostgreSQL (Goose SQL migrations in `internal/database/migrations/`)
- **Authentication**: JWT (golang-jwt/jwt/v5) with OAuth2/OIDC providers (Google, Apple via coreos/go-oidc/v3)
- **Authorization**: Custom two-tier RBAC system (platform roles + bucket groups)
- **Storage**: Multi-provider abstraction with AWS S3 SDK v2, GCP Cloud Storage, MinIO Go client v7
- **Messaging**: Watermill event-driven architecture (JetStream, GCP Pub/Sub, AWS SQS)
- **Activity Logging**: Loki integration for audit trails
- **Configuration**: Koanf with hierarchical YAML file + environment variable support
- **Validation**: go-playground/validator with struct tags
- **Password Hashing**: Argon2id for secure credential storage
- **Cache**: Rueidis (Redis/Valkey client) for rate limiting and app identity
- **Logging**: Zap structured logging

### Frontend (React 19 + Vite 7)

- **Build Tool**: Vite 7 with TypeScript 5.9 and React plugin
- **Framework**: React 19 with strict TypeScript
- **Routing**: TanStack Router v1 with file-based routing and type-safe navigation
- **State Management**:
    - Local/Context: React Context with reducer patterns
    - Server State: TanStack Query v5 (5-minute stale time, query invalidation)
- **UI**: Tailwind CSS 4 (Vite plugin) + Radix UI primitives + shadcn/ui patterns
- **Component Styling**: class-variance-authority + tailwind-merge + clsx
- **Forms**: react-hook-form v7 with @hookform/resolvers validation
- **Data Tables**: TanStack Table v8
- **Icons**: Lucide React
- **Date Handling**: date-fns v4 with react-day-picker v9
- **Authentication**: js-cookie for JWT token management
- **I18n**: i18next with browser language detection
- **Testing**: Vitest v4 with jsdom and React Testing Library

## Key Architectural Patterns

### Backend Abstractions

#### 1. Storage Interface (`internal/storage/interfaces.go`)

Cloud-agnostic storage operations with 14 methods:
```go
type IStorage interface {
    PresignedGetObject(path string) (string, error)
    PresignedPostPolicy(path string, size int, metadata map[string]string) (string, map[string]string, error)
    StatObject(path string) (map[string]string, error)
    ListObjects(prefix string, maxKeys int32) ([]string, error)
    RemoveObject(path string) error
    RemoveObjects(paths []string) error
    SetObjectTags(path string, tags map[string]string) error
    GetObjectTags(path string) (map[string]string, error)
    RemoveObjectTags(path string, tagsToRemove []string) error
    EnsureTrashLifecyclePolicy(retentionDays int) error
    MarkFileAsTrashed(objectPath string, metadata models.TrashMetadata) error
    UnmarkFileAsTrashed(objectPath string) error
    IsTrashMarkerPath(path string) (isMarker bool, originalPath string)
    GetBucketName() string
}
```

- **Implementations**: S3Storage (MinIO), GCPStorage, AWSStorage
- **Factory**: `core.NewStorage()` initializes provider from config
- **Trash System**: Lifecycle policies with configurable retention (1-365 days)

#### 2. Messaging Interface (`internal/messaging/interfaces.go`)

Event-driven pub/sub abstraction:
```go
type IPublisher interface {
    Publish(messages ...*message.Message) error
    Close() error
}
type ISubscriber interface {
    Subscribe() <-chan *message.Message
    Close() error
}
```

- **Implementations**: JetStream, GCP Pub/Sub, AWS SQS (via Watermill)
- **Queues**: `notifications`, `object_deletion`, `bucket_events`
- **Factory**: `core.NewMessaging()` creates router with background handlers

#### 3. Activity Logger Interface (`internal/activity/loki.go`)

Audit trail abstraction with Loki backend:

- Structured logs with labels (user_id, action, resource_type, resource_id, bucket_id)
- Query with filters and 30-day retention window
- HTTP-based push via resty client

#### 4. Generic Handler Pattern (`internal/handlers/generic.go`)

Type-safe CRUD operations using Go generics:
```go
type CreateTargetFunc[In any, Out any] func(models.UserClaims, uuid.UUIDs, In) (Out, error)
type ListTargetFunc[Out any] func(models.UserClaims) []Out
type GetOneTargetFunc[Out any] func(models.UserClaims, uuid.UUIDs) (Out, error)
type UpdateTargetFunc[In any] func(models.UserClaims, uuid.UUIDs, In) error
type DeleteTargetFunc func(models.UserClaims, uuid.UUIDs) error
```

- Automatic body extraction from context
- Consistent error handling with custom APIError types
- Used throughout services for DRY CRUD operations

#### 5. RBAC System (`internal/rbac/`)

Two-tier hierarchical authorization:

- **Platform Roles** (global): Admin > User > Guest
- **Bucket Groups** (per-bucket): Owner > Contributor > Viewer
- Group rank comparison: `GetGroupRank(group string) int`
- Middleware: `AuthorizeRole`, `AuthorizeGroup`, `AuthorizeSelfOrAdmin`
- Database-backed membership with GORM

#### 6. Configuration Management (`internal/configuration/config.go`)

Koanf-based hierarchical config:

- **Sources**: YAML file (`config.yaml`, `templates/config.yaml`) + environment variables
- **Env Format**: Double underscore delimiters (e.g., `APP__LOG_LEVEL=debug`)
- **Validation**: Automatic struct validation with go-playground/validator
- **Array Parsing**: Supports complex nested configurations

### Frontend Patterns

#### 1. Component Organization

Feature-based structure:
```
components/
├── feature-name/
│   ├── Component.tsx          # Main component
│   ├── components/            # Sub-components
│   ├── hooks/                 # Custom hooks (useFooData)
│   ├── helpers/               # Types and utilities
│   └── context/               # Context providers
```

- 15+ feature areas: bucket-view, bucket-members, upload, activity-view, auth-view, etc.
- Shared UI in `components/ui/` (shadcn/ui pattern)

#### 2. State Management

- **Context Providers**: SessionProvider, ThemeProvider, UploadProvider, BucketViewProvider, SidebarProvider
- **Custom Hooks**: useBucketViewContext, useUploadContext, useTheme, useDialog, useMobile
- **Server State**: TanStack Query with 5-minute stale time and mutation invalidation
- **Upload State**: Reducer pattern with actions (addUpload, updateProgress, updateStatus)

#### 3. API Layer (`src/lib/api.ts`)

Centralized type-safe fetch:

```typescript
async function fetchApi<T>(endpoint: string, options?: FetchOptions): Promise<T>
```

- Automatic JWT token injection from cookies
- Token refresh on 403 errors with retry mechanism
- Query parameter builder with null/undefined filtering

#### 4. Routing (`src/routes/`)

TanStack Router with file-based routing:

- Type-safe navigation with generated route tree (`routeTree.gen.ts`)
- Router context includes QueryClient for data integration
- Auto code splitting via TanStack router plugin

## Code Conventions

### Backend Conventions

#### Package Structure

```
internal/
├── models/        # Data models (User, Bucket, File) and DTOs
├── services/      # Business logic with Routes() methods returning chi.Router
├── handlers/      # Generic HTTP handler functions
├── middlewares/   # Auth, validation, RBAC, rate limiting, logging
├── storage/       # IStorage implementations
├── messaging/     # IPublisher/ISubscriber implementations
├── activity/      # Activity logger implementation
├── rbac/          # Authorization rules and middleware
├── core/          # Factory functions for abstractions
├── helpers/       # JWT, JSON, validation utilities
├── configuration/ # Config loading and validation
├── database/      # GORM setup and migrations/
└── errors/        # Custom APIError struct
```

#### Naming Conventions

- **Interfaces**: Prefix with `I` (IStorage, IPublisher, ICache)
- **Models**: PascalCase matching table names (User, Bucket, File)
- **Services**: Suffix with `Service` (BucketService, FileService)
- **Constants**: UPPER_SNAKE_CASE in `const.go` files
- **Package Aliases**: Single letter imports (`c` for cache, `h` for helpers, `m` for middlewares)

#### Database Patterns

- **GORM Models**: Embedded `gorm.Model`, UUID primary keys with `gen_random_uuid()`
- **Timestamps**: Automatic created_at, updated_at
- **Soft Deletes**: deleted_at with index
- **Migrations**: Goose SQL migrations in `internal/database/migrations/`, auto-run on startup

#### Error Handling

- Custom `APIError` struct in `internal/errors/common.go`
- HTTP status code mapping
- Use `errors.As()` for type checking

### Frontend Conventions

#### Naming Conventions

- **Components**: PascalCase (BucketView, FileActions)
- **Hooks**: camelCase with `use` prefix (useBucketData, useTheme)
- **Types/Interfaces**: PascalCase, optionally `I` prefix for interfaces
- **Constants**: UPPER_SNAKE_CASE
- **Files**: kebab-case for utilities, PascalCase for components

#### TypeScript Patterns

- Strict mode with noUnusedLocals, noUnusedParameters
- Path alias: `@/*` → `src/*`
- All props and state properly typed
- Union types for enums (BucketViewMode, UploadStatus)

## Development Commands

### Backend
```bash
go run main.go          # Start server (default port 8080)
go test ./...           # Run all tests
go mod tidy             # Clean dependencies
go fmt ./...            # Format code
go vet ./...            # Lint code
```

### Frontend (in `web/` directory)

```bash
npm run dev             # Vite dev server on port 3000 with HMR
npm run build           # Production build (Vite + TypeScript check)
npm run serve           # Serve production build
npm run test            # Vitest unit tests
npm run lint            # ESLint check
npm run prettier        # Check formatting
npm run fixup           # Prettier format all files
```

### Docker

```bash
docker compose up -d    # Start all services locally
```

## Database Schema

### Core Models

- **Users**: email, role (admin/user/guest), provider (oauth/local), hashed_password (Argon2id)
- **Buckets**: name, created_by (user FK)
- **Files**: bucket FK, path, type (file/folder), size
- **Memberships**: user-bucket relationship with group (owner/contributor/viewer)
- **Invites**: email invitation system with challenge codes

All models use UUID primary keys, timestamps (created_at, updated_at), and soft deletes (deleted_at).

## Configuration

### Environment Variables

Use double underscore delimiter for nested keys:

```bash
APP__LOG_LEVEL=debug
APP__DATABASE__HOST=localhost
APP__DATABASE__PORT=5432
APP__STORAGE__PROVIDER=minio
APP__STORAGE__MINIO__ENDPOINT=localhost:9000
```

### Key Configuration Sections

- **App**: Port, log level, environment
- **Database**: Host, port, user, password, name
- **JWT**: Secret, access/refresh token expiration
- **Storage**: Provider (aws/gcp/minio), credentials, bucket
- **Messaging**: Provider (jetstream/gcppubsub/awssqs), configuration
- **Cache**: Redis/Valkey host, port, password, DB
- **Activity**: Loki endpoint
- **OAuth**: Provider configurations (Google, Apple)
- **SMTP**: Email sending configuration

## Security Patterns

- **Authentication**: JWT with 60-minute access token, 10-hour refresh token
- **Authorization**: Two-tier RBAC (platform roles + bucket groups with rank comparison)
- **Passwords**: Argon2id hashing with salt
- **Presigned URLs**: 15-minute expiration for uploads and downloads
- **Rate Limiting**: Redis-based with trusted proxy support
- **CORS**: Configurable allowed origins
- **Input Validation**: Struct tags with go-playground/validator
- **Audit Logging**: All user actions logged to Loki with 30-day retention

## Special Considerations

### Trash System

- Files marked with tags instead of immediate deletion
- Lifecycle policies (1-365 days configurable retention)
- `MarkFileAsTrashed()` and `UnmarkFileAsTrashed()` in IStorage
- All providers implement `EnsureTrashLifecyclePolicy()`

### Event-Driven Architecture

- Three event queues: notifications, object_deletion, bucket_events
- Background goroutines handle events
- Watermill router pattern for pub/sub abstraction

### Multi-Provider Support

All cloud services are abstracted:

- **Storage**: AWS S3, GCP Cloud Storage, MinIO
- **Messaging**: JetStream, GCP Pub/Sub, AWS SQS
- **Cache**: Redis/Valkey via Rueidis

### Frontend Data Flow

- TanStack Query for server state with 5-minute stale time
- Query invalidation after mutations
- Token refresh mechanism in fetchApi on 403 errors
- Context providers for feature-level state (upload, bucket view, session)