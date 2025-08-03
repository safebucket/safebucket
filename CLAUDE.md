# Claude Context - SafeBucket Project

## Project Overview
SafeBucket is a secure cloud storage management platform with multi-provider support (AWS, GCP, MinIO) featuring user authentication, role-based access control, and real-time activity tracking.

## Architecture Summary

### Backend (Go 1.23)
- **Framework**: Chi HTTP router with middleware composition
- **Database**: GORM with PostgreSQL (supports MySQL, SQLServer, SQLite)
- **Authentication**: JWT-based with OAuth providers (Google, Apple via OIDC)
- **Authorization**: Casbin RBAC with role hierarchy (Admin > User > Guest)
- **Storage**: Multi-provider abstraction (AWS S3, GCP Cloud Storage, MinIO)
- **Messaging**: Event-driven architecture with Watermill (JetStream, GCP Pub/Sub, AWS SQS)
- **Activity Logging**: Loki integration for audit trails
- **Configuration**: Viper-based config management with YAML/ENV support
- **Validation**: go-playground/validator with struct tags
- **Password Hashing**: Argon2id for secure secret storage

### Frontend (Next.js 15/React 18)
- **Framework**: Next.js 15 with TypeScript, Turbopack for dev
- **UI**: Tailwind CSS with Radix UI primitives (@radix-ui/*)
- **Component Library**: shadcn/ui with class-variance-authority
- **State Management**: React Context with reducer patterns + SWR for data fetching
- **Authentication**: Custom JWT handling with js-cookie
- **File Management**: Upload/download with react-hook-form and progress tracking
- **Data Tables**: TanStack Table (@tanstack/react-table)
- **Icons**: Lucide React + React Icons
- **Date Handling**: date-fns with react-day-picker

## Abstraction Layers

### Backend Abstractions

#### 1. **Storage Interface** (`internal/storage/interfaces.go`)
```go
type IStorage interface {
    UploadFile(bucketName, fileName string, file io.Reader) error
    DownloadFile(bucketName, fileName string) (io.Reader, error)
    DeleteFile(bucketName, fileName string) error
    ListFiles(bucketName, prefix string) ([]FileInfo, error)
}
```
- **Implementations**: S3Storage, GCPStorage, MinIOStorage
- **Purpose**: Cloud-agnostic object operations

#### 2. **Messaging Interface** (`internal/messaging/interfaces.go`)
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
- **Implementations**: JetStream, GCP Pub/Sub, AWS SQS
- **Purpose**: Event-driven communication abstraction

#### 3. **Activity Logger Interface** (`internal/activity/interfaces.go`)
```go
type IActivityLogger interface {
    Log(activity ActivityEntry) error
    Query(filters ActivityFilters) ([]ActivityEntry, error)
}
```
- **Implementations**: LokiLogger
- **Purpose**: Audit trail abstraction

#### 4. **Generic Handler Pattern** (`internal/handlers/generic.go`)
Type-safe CRUD operations with generics:
```go
type CreateTargetFunc[In any, Out any] func(models.UserClaims, uuid.UUIDs, In) (Out, error)
type ListTargetFunc[Out any] func(models.UserClaims) []Out
```
- **Purpose**: Consistent HTTP handler patterns with type safety

### Frontend Abstractions

#### 1. **Context Providers Pattern**
- **BucketViewProvider**: File browser state management
- **UploadProvider**: File upload state with reducer pattern
- **SessionProvider**: Authentication state management

#### 2. **Custom Hooks Pattern**
- **Data Fetching**: `useBucketData`, `useActivityData` with SWR
- **State Management**: `useBucketViewContext`, `useUploadContext`
- **Utilities**: `useDialog`, `useMobile`

#### 3. **Component Composition**
- **Compound Components**: DataTable with DataColumnHeader, DataTableRowActions
- **Provider Pattern**: Context providers wrapping feature areas
- **Slot Pattern**: Using @radix-ui/react-slot for flexible composition

## Code Conventions

### Backend Conventions (Go)

#### 1. **Package Structure**
```
internal/
├── models/        # Data models and DTOs
├── services/      # Business logic layer
├── handlers/      # HTTP handlers
├── middlewares/   # HTTP middleware
├── storage/       # Storage implementations
├── messaging/     # Event messaging
├── rbac/          # Authorization rules
├── core/          # Core abstractions
└── helpers/       # Utility functions
```

#### 2. **Naming Conventions**
- **Interfaces**: Prefixed with `I` (e.g., `IStorage`, `IPublisher`)
- **Models**: Struct names match table names (e.g., `User`, `Bucket`)
- **Services**: Suffixed with `Service` (e.g., `BucketService`)
- **Constants**: UPPER_SNAKE_CASE in const.go files
- **Package Aliases**: Single letter for commonly used packages (`c` for cache, `h` for helpers, `m` for middlewares)

#### 3. **Error Handling**
- **Custom Errors**: `internal/errors/common.go` with `APIError` struct
- **Error Wrapping**: Use `errors.As()` for type checking
- **HTTP Status**: Map custom errors to appropriate HTTP status codes

#### 4. **Database Patterns**
- **GORM Models**: Embedded `gorm.Model` for timestamps
- **Relationships**: Proper foreign keys and associations
- **Migrations**: Handled via GORM AutoMigrate
- **Transactions**: Use `tx.Begin()` for multi-table operations

### Frontend Conventions (TypeScript/React)

#### 1. **File Structure**
```
components/
├── feature-name/
│   ├── Component.tsx          # Main component
│   ├── components/           # Sub-components
│   ├── hooks/               # Custom hooks
│   ├── helpers/             # Utilities & types
│   └── context/             # Context providers
```

#### 2. **Naming Conventions**
- **Components**: PascalCase (e.g., `BucketView`, `FileActions`)
- **Hooks**: camelCase with `use` prefix (e.g., `useBucketData`)
- **Types/Interfaces**: PascalCase with `I` prefix for interfaces
- **Constants**: UPPER_SNAKE_CASE
- **Files**: kebab-case for utilities, PascalCase for components

#### 3. **TypeScript Patterns**
- **Strict Types**: All props and state properly typed
- **Generic Components**: Use generics for reusable components
- **Union Types**: For status/mode enums (e.g., `BucketViewMode`)
- **Optional Properties**: Use `?` for optional props

#### 4. **State Management**
- **Local State**: `useState` for component-specific state
- **Shared State**: Context providers for feature-level state
- **Server State**: SWR for data fetching with caching
- **Form State**: react-hook-form for complex forms

## Common Tools & Utilities

### Backend Tools
- **Configuration**: Viper for config file and environment variables
- **Logging**: Zap structured logging with different levels
- **Testing**: Testify for assertions and mocks
- **Validation**: go-playground/validator with struct tags
- **UUID**: Google UUID for unique identifiers
- **Hashing**: Argon2id for password hashing
- **HTTP Client**: resty for external API calls

### Frontend Tools
- **Linting**: ESLint with Prettier integration
- **Code Formatting**: Prettier with Tailwind plugin
- **Build**: Next.js with Turbopack for development
- **Type Checking**: TypeScript with strict mode
- **CSS**: Tailwind CSS with custom configuration
- **Icons**: Lucide React (primary) + React Icons (additional)
- **Forms**: react-hook-form with validation
- **Date/Time**: date-fns for date manipulation

### Development Workflow
- **Package Management**: npm (frontend), go modules (backend)
- **Code Quality**: ESLint + Prettier (frontend), go fmt + go vet (backend)
- **Testing**: Jest/React Testing Library (frontend), Go testing (backend)
- **Git Workflow**: Feature branches with PR reviews
- **Environment**: Docker Compose for local development

## Development Commands
```bash
# Backend
go run main.go                 # Start server
go test ./...                  # Run tests
go mod tidy                    # Clean dependencies
go fmt ./...                   # Format code

# Frontend
npm run dev                    # Development server (Turbopack)
npm run build                  # Production build
npm run lint                   # ESLint check
npm run prettier               # Format check
npm start                      # Production server
```

## Database Schema
- **Users**: Authentication and profile data with OAuth integration
- **Buckets**: Storage container metadata with provider configuration
- **Files**: File metadata with permissions and versioning
- **Invites**: User invitation system with email verification
- **Activities**: Audit log entries with Loki integration
- **Policies**: RBAC permission rules managed by Casbin

## Security Considerations
- **Authentication**: JWT tokens with configurable expiration
- **Authorization**: Casbin RBAC with hierarchical roles
- **CORS**: Configurable origins for cross-origin requests
- **Input Validation**: Struct validation on all endpoints
- **Password Security**: Argon2id hashing with salt
- **File Access**: Bucket-level permissions with user context
- **Audit Logging**: All user actions logged for compliance
- **Environment Variables**: Sensitive data in environment variables

## Current Branch: feature/claude-context-subagents
Working on implementing a subagent system for AI-assisted development workflows.