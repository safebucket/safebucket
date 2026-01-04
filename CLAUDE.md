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
    MarkAsTrashed(objectPath string, model interface{}) error      // Polymorphic - handles File or Folder
    UnmarkAsTrashed(objectPath string, model interface{}) error    // Polymorphic - handles File or Folder
    IsTrashMarkerPath(path string) (isMarker bool, originalPath string)
    GetBucketName() string
}
```

- **Implementations**: S3Storage (MinIO), GCPStorage, AWSStorage
- **Factory**: `core.NewStorage()` initializes provider from config
- **Storage Structure**:
  - Active files: `buckets/{bucket_id}/{file_id}`
  - Trash markers (files): `trash/{bucket_id}/files/{file_id}`
  - Trash markers (folders): `trash/{bucket_id}/folders/{folder_id}`
- **Trash System**: Marker-based lifecycle policies with configurable retention (1-365 days)

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

#### 7. Query Parameter Validation (`internal/middlewares/query_validator.go`)

Reflection-based query parameter parsing and validation:

```go
// Middleware
ValidateQuery[T any](next http.Handler) http.Handler

// Usage in routes
r.With(m.ValidateQuery[models.FileListQueryParams]).
    Get("/files", handlers.GetListHandler(s.ListFiles))
```

- Parses URL query parameters into structs using reflection
- Validates using go-playground/validator tags
- Stores validated params in request context
- Supports string, int, bool, pointer types
- Retrieved via `helpers.GetQueryParams[T](context)`

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
- **Buckets**: name, created_by (user FK), has many files and folders
- **Files** (`internal/models/file.go`):
  - Fields: name, extension, size, bucket_id (FK), folder_id (nullable FK)
  - Status: uploading, uploaded, deleting, trashed, restoring, deleted
  - Trash metadata: Uses GORM soft delete (deleted_at) for timestamp, deleted_by (FK to users) for audit trail
  - **Storage path**: `buckets/{bucket_id}/{file_id}` (UUID-based, not path-based)
- **Folders** (`internal/models/folder.go`):
  - Fields: name, bucket_id (FK), folder_id (nullable self-referencing FK for parent)
  - Status: uploaded, trashed, restoring, deleted
  - Trash metadata: Uses GORM soft delete (deleted_at) for timestamp, deleted_by (FK to users) for audit trail
  - **Self-referencing**: NULL folder_id = root level, otherwise nested
- **Memberships**: user-bucket relationship with group (owner/contributor/viewer)
- **Invites**: email invitation system with challenge codes

All models use UUID primary keys, timestamps (created_at, updated_at), and soft deletes (deleted_at).

### File Status Lifecycle

```
uploading → uploaded ⇄ trashed → deleted (soft delete)
                ↓
            deleting (transition state)
```

- **uploading**: File record created, awaiting storage upload
- **uploaded**: File successfully stored and active
- **trashed**: File in trash bin (marker created, subject to lifecycle policy)
- **restoring**: Transition state during folder restore operations
- **deleted**: File permanently removed (soft delete for audit trail)

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
- **JWT Audience Validation**: All JWT token parsing functions validate the `aud` (audience) claim to enforce strict token type boundaries and prevent token type confusion attacks:
  - `ParseAccessToken` (`internal/helpers/token.go:53-79`): Only accepts tokens with `aud: "app:*"`
  - `ParseMFAToken` (`internal/helpers/token.go:164-186`): Only accepts tokens with `aud: "auth:mfa"`
  - `ParseRefreshToken` (`internal/helpers/token.go:100-118`): Only accepts tokens with `aud: "auth:refresh"`
  - **Security Boundary**: MFA tokens cannot access protected API endpoints; they can only be used for MFA verification at `/api/v1/auth/mfa/verify`
  - **Enforcement**: The `Authenticate` middleware (`internal/middlewares/authenticator.go:22`) calls `ParseAccessToken`, which rejects any token without the `"app:*"` audience
- **Authorization**: Two-tier RBAC (platform roles + bucket groups with rank comparison)
- **Passwords**: Argon2id hashing with salt
- **Presigned URLs**: 15-minute expiration for uploads and downloads
- **Rate Limiting**: Redis-based with trusted proxy support
- **CORS**: Configurable allowed origins
- **Input Validation**: Struct tags with go-playground/validator
- **Audit Logging**: All user actions logged to Loki with 30-day retention

## Special Considerations

### Trash System (PATCH-based with Marker Lifecycle)

**Architecture**: Marker-based soft delete with lifecycle policy automation

#### API Endpoints

**Files** (`internal/services/bucket_file.go`):

```
POST   /buckets/{id}/files              - Upload file (accepts folder_id)
PATCH  /buckets/{id}/files/{id}         - Trash or restore (sync)
  Body: { "status": "trashed" }         - Move to trash
  Body: { "status": "uploaded" }        - Restore from trash
DELETE /buckets/{id}/files/{id}         - Purge permanently (sync)
GET    /buckets/{id}/files/{id}/download - Download file
```

**Folders** (`internal/services/bucket_folder.go`):

```
POST   /buckets/{id}/folders            - Create folder (accepts folder_id for nesting)
PATCH  /buckets/{id}/folders/{id}       - Update name OR trash/restore (async)
  Body: { "name": "new_name" }          - Rename folder
  Body: { "status": "trashed" }         - Move to trash (async)
  Body: { "status": "uploaded" }        - Restore from trash (async)
DELETE /buckets/{id}/folders/{id}       - Purge permanently (async)
```

**Buckets** (with query filtering):

```
GET /buckets/{id}                       - List files and folders (default: status=uploaded)
GET /buckets/{id}?status=trashed        - List trashed items
```

#### Storage Marker System

**Marker Paths**:

- Files: `trash/{bucket_id}/files/{file_id}`
- Folders: `trash/{bucket_id}/folders/{folder_id}`

**Lifecycle Flow**:

1. **Trash**: Creates empty marker object in trash prefix
2. **Lifecycle Policy**: Deletes markers after retention period (1-365 days)
3. **Event Triggered**: Marker deletion fires `TrashExpiration` event
4. **Cleanup**: Event handler deletes actual file and soft-deletes DB record

**Methods**:

- `MarkAsTrashed(objectPath, model)` - Creates trash marker (polymorphic: File or Folder)
- `UnmarkAsTrashed(objectPath, model)` - Removes trash marker (restore operation)
- `IsTrashMarkerPath(path)` - Detects if deletion event is for a marker

#### Processing Strategy

**Files**: Synchronous (immediate response)

- Database status update with row-level locking
- Marker creation/deletion in same transaction
- Direct storage operations

**Folders**: Asynchronous via event queue (handles children recursively)

- Parent folder status updated immediately
- Event published to background queue
- Worker processes children in batches
- Prevents timeout on large folder structures

#### Race Condition Prevention

**Database-Level Protection**:

```go
// Row-level locking
tx.Clauses(clause.Locking{Strength: "UPDATE"}).
  Where("id = ? AND bucket_id = ?", id, bucketID).
  First(&file)

// Atomic status transitions
tx.Model(&file).
  Where("status = ?", models.FileStatusUploaded).  // Only if uploaded
  Updates(map[string]interface{}{
    "status": models.FileStatusTrashed,
  })

// Optimistic concurrency check
if result.RowsAffected == 0 {
  return errors.NewAPIError(409, "INVALID_FILE_STATUS_TRANSITION")
}
```

**Valid Status Transitions**:

- `uploaded → trashed` (trash operation)
- `trashed → uploaded` (restore operation)
- `trashed → deleted` (purge operation)
- Invalid transitions return 409 Conflict

**Transaction Rollback**:

- Storage operations inside DB transactions
- Marker creation failure rolls back DB changes
- Ensures consistency between DB and storage

#### Event Handlers (`internal/events/`)

- **FolderTrash**: Recursively trash folder tree, update all descendant statuses
- **FolderRestore**: Recursively restore, check for naming conflicts at each level
- **FolderPurge**: Permanently delete folder and all contents from storage and DB
- **TrashExpiration**: Handle lifecycle policy deletions, distinguish restore vs expiry

#### Query Parameter Filtering

Status-based filtering via middleware:

```go
// Example: Get only trashed items
GET /buckets/{id}?status=trashed

// Frontend query
const { files, folders } = await api.get(`/buckets/${id}?status=trashed`)
```

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
- **Files and Folders**: Separate arrays returned from bucket queries
- **Type Safety**: `IFile` type (has `extension`, `folder_id`) vs `IFolder` type (has `folder_id` for parent)

## Important Development Patterns

### Working with Files and Folders

**Always use IDs, never paths**:

```go
// ✅ Correct - ID-based
filePath := path.Join("buckets", bucketID.String(), fileID.String())

// ❌ Incorrect - Path-based (old system)
filePath := path.Join("buckets", bucketID.String(), userPath, fileName)
```

**Folder operations are async**:

```go
// Files - synchronous, immediate response
func (s BucketFileService) TrashFile(...) error

// Folders - asynchronous, event-driven
func (s BucketFolderService) TrashFolder(...) error {
    // Update parent status immediately
    // Trigger async event for children
    event := events.NewFolderTrash(...)
    event.Trigger()
}
```

**Frontend must handle separate arrays**:

```typescript
// ✅ Correct - Separate arrays
const { files, folders } = bucket;
folders.forEach(folder => renderFolder(folder));
files.forEach(file => renderFile(file));

// ❌ Incorrect - Old unified approach
const items = bucket.files; // files used to contain both
items.forEach(item => {
    if (item.type === FileType.folder) { ... } // FileType removed
});
```

### Status Transitions

**Always validate before transition**:

```go
// Check current status before changing
if file.Status != models.FileStatusUploaded {
    return errors.NewAPIError(409, "INVALID_FILE_STATUS_TRANSITION")
}
```

**Use atomic WHERE clauses**:

```go
// Only update if current status matches expected
result := tx.Model(&file).
    Where("status = ?", models.FileStatusUploaded).
    Update("status", models.FileStatusTrashed)

if result.RowsAffected == 0 {
    // Status was changed by concurrent request
    return errors.NewAPIError(409, "CONFLICT")
}
```

### Storage Operations

**Polymorphic trash methods**:

```go
// Works for both files and folders
err := s.Storage.MarkAsTrashed(objectPath, file)   // Pass File model
err := s.Storage.MarkAsTrashed(objectPath, folder) // Pass Folder model
```

**Transaction safety**:

```go
return s.DB.Transaction(func(tx *gorm.DB) error {
    // Update database
    if err := tx.Model(&file).Updates(...).Error; err != nil {
        return err
    }

    // Update storage (rollback on failure)
    if err := s.Storage.MarkAsTrashed(path, file); err != nil {
        return err // Automatic rollback
    }

    return nil
})
```

### Query Parameter Filtering

**Use ValidateQuery middleware**:

```go
type FileListQueryParams struct {
    Status  *string `json:"status" validate:"omitempty,oneof=uploaded trashed"`
    Limit   *int    `json:"limit"  validate:"omitempty,min=1,max=1000"`
}

r.With(m.ValidateQuery[FileListQueryParams]).
    Get("/files", handlers.GetListHandler(s.ListFiles))
```

**Retrieve in handler**:

```go
func (s Service) ListFiles(user models.UserClaims, ids uuid.UUIDs) ([]models.File, error) {
    params, err := helpers.GetQueryParams[FileListQueryParams](r.Context())
    if err != nil {
        // Handle error
    }

    // Use params
    if params.Status != nil && *params.Status == "trashed" {
        // Filter trashed files
    }
}
```

### Frontend TypeScript Patterns

**Type guards for files vs folders**:

```typescript
// Check if item is a folder
function isFolder(item: IFile | IFolder): item is IFolder {
    return 'folder_id' in item && !('extension' in item);
}

// Or handle separately from API
const { files, folders } = bucket;
```

**Query filtering**:

```typescript
// Get trashed items
const { files, folders } = await api.get<IBucket>(
    `/buckets/${bucketId}?status=trashed`
);

// TanStack Query hook
export const bucketTrashedFilesQueryOptions = (bucketId: string) =>
  queryOptions({
    queryKey: ["buckets", bucketId, "trash"],
    queryFn: async () => {
      const response = await api.get<IBucket>(
        `/buckets/${bucketId}?status=trashed`
      );
      return {
        files: response.files || [],
        folders: response.folders || [],
      };
    },
  });
```

### Migration Guide (Old → New)

**File Upload**:

```typescript
// Old
createFile({
    name: "doc.pdf",
    type: FileType.file,
    path: "/folder1/subfolder"
});

// New
createFile({
    name: "doc.pdf",
    folder_id: "uuid-of-parent-folder"
});
```

**Folder Creation**:

```typescript
// Old
createFolder({
    name: "New Folder",
    type: FileType.folder,
    path: "/parent/path"
});

// New
createFolder({
    name: "New Folder",
    folder_id: "uuid-of-parent-folder" // null for root
});
```

**Trash Operations**:

```typescript
// Old
deleteFile(fileId); // Immediate delete

// New
patchFile(fileId, { status: "trashed" }); // Trash (soft delete)
deleteFile(fileId);                       // Purge (permanent)
```