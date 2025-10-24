# Product Requirements Document: Trash/Soft-Delete Feature

**Version:** 1.8
**Last Updated:** 2025-10-23
**Status:** Implementation Complete (Backend + Frontend + Folders + Async + Event Handlers + Soft Delete Fix + React Query Migration + NULL Handling Bug Fixes)
**Owner:** Engineering Team
**Branch:** `add-bucket-trash`

---

## Implementation Status

### ✅ Completed (Backend)
- Database schema updates with migration scripts
- Storage interface and implementations (AWS S3, GCP, MinIO)
- Service layer methods (TrashFile, RestoreFile, ListTrashedFiles, PurgeFile)
- **Folder support** (TrashFolder, RestoreFolder, PurgeFolder) ✨ **V1.3**
- **Async folder operations** (prevents HTTP timeouts on large folders) ✨ **V1.4**
- API endpoints and routing
- RBAC integration with proper permissions
- Activity logging for all trash operations (files + folders)
- Event handlers (trash expiration, folder restore, folder trash, folder purge)
- **Event-driven deletion processing** (ParseBucketDeletionEvents) ✨ **V1.5**
- Download prevention for trashed files
- Build verification: **SUCCESS** ✅

### ✅ Completed (Frontend)
- Trash view UI component with DataTable
- API integration (list trash, restore, purge)
- **React Query (TanStack Query) migration** ✨ **V1.7**
  - Centralized query options in `web/src/queries/bucket.ts`
  - `bucketTrashedFilesQueryOptions` for consistent caching
  - `useTrashActions` hook with mutations and cache invalidation
  - i18n support for success/error messages
- Activity feed support for trash operations (files + folders)
- **Folder activity icons and messages** ✨ **NEW**
- Context menu updates for trash view
- Right-click menu shows restore/purge in trash mode
- Auto-refresh on trash view activation
- Localization (English + French) with folder translations and toast messages
- Download restriction for trashed files
- Build verification: **SUCCESS** ✅

### ✅ Completed (Event Processing) - **V1.5**
- **ParseBucketDeletionEvents** implemented for all providers (AWS SQS, GCP Pub/Sub, NATS/JetStream)
- Event handler integration in HandleBucketEvents
- Multi-provider deletion event parsing (ObjectRemoved, LifecycleExpiration, OBJECT_DELETE)
- Bucket ID extraction from object key paths
- TrashExpiration event processing via callback
- Build verification: **SUCCESS** ✅

### 🚧 Pending (Infrastructure)
- Cloud provider lifecycle policy configuration (AWS/GCP/MinIO)
- Event notification setup for deletion events (AWS S3→SQS, GCP→Pub/Sub, MinIO→NATS)
- Reconciliation job scheduling (optional - for backup cleanup)

### 📝 Implementation Notes
- **✅ Folders Now Supported (V1.3)**: Folders can now be trashed, restored, and purged. All folder contents (files and subfolders) are trashed/restored together with matching timestamps. **V1 limitation has been removed.**
- **✅ Async Folder Operations (V1.4)**: All folder operations (trash, restore, purge) are now asynchronous to prevent HTTP timeouts on large folders. Operations return HTTP 200 immediately and process in background via event system.
- **✅ Event-Driven Deletion (V1.5)**: All three messaging providers (AWS SQS, GCP Pub/Sub, NATS/JetStream) now parse deletion events and trigger trash expiration processing automatically. Supports both manual deletions and lifecycle policy expirations.
- **Timestamp-Based Restore**: When restoring folders, only children with matching `trashed_at` timestamps are restored. This prevents restoring files that were individually trashed after the folder deletion.
- **Batch Processing**: Folder operations process children in batches (`c.BulkActionsLimit`) and requeue events if more items remain.
- **FileStatusRestoring**: New transient status for folders being restored asynchronously.
- **Database is Source of Truth**: Storage tagging is best-effort. Trash state is determined by the database `status` field, not storage tags.
- **Handler Signature Change**: `ListTrashedFiles` returns `[]models.File` directly (no error return) to match the `ListTargetFunc` handler pattern.
- **Frontend API Wrapper**: `api_listTrashedFiles` unwraps the `Page[T]` structure to provide direct array access.
- **Download Protection**: Both client-side (disabled UI) and server-side (403 error) prevent downloading trashed files.

### 📦 Deliverables
- ✅ **Backend** - 24 modified files (21 base + 3 folder event handlers), complete trash API with async folder support
- ✅ **Frontend** - 13 modified files, complete trash UI implementation with folder support
- ✅ **Database Migration** - Forward and rollback scripts included
- ✅ **Implementation Guide** - `docs/IMPLEMENTATION-trash-feature.md`
- ✅ **Build Verification** - Both backend and frontend compile successfully
- ✅ **Localization** - English and French translations (including folder operations)
- ✅ **Async Event Handlers** - FolderRestore, FolderTrash, FolderPurge (V1.4)
- ✅ **Event Parsing Implementations** - ParseBucketDeletionEvents for AWS/GCP/MinIO (V1.5)
- ⏳ **Infrastructure Config** - Lifecycle policies need separate deployment

### 🔗 Related Documentation
- **Implementation Guide**: [`docs/IMPLEMENTATION-trash-feature.md`](./IMPLEMENTATION-trash-feature.md) - Deployment steps, API examples, troubleshooting
- **Database Migration**: [`migrations/001_add_trash_fields.sql`](../migrations/001_add_trash_fields.sql) - Schema changes
- **Event Handlers**:
  - [`internal/events/trash_expiration.go`](../internal/events/trash_expiration.go) - Trash cleanup logic
  - [`internal/events/folder_restore.go`](../internal/events/folder_restore.go) - Async folder restore ✨ V1.4
  - [`internal/events/folder_trash.go`](../internal/events/folder_trash.go) - Async folder trash ✨ V1.4
  - [`internal/events/folder_purge.go`](../internal/events/folder_purge.go) - Async folder purge ✨ V1.4
  - [`internal/events/handler.go`](../internal/events/handler.go) - Event processing with deletion support ✨ V1.5
- **Messaging Implementations**: ✨ **V1.5**
  - [`internal/messaging/types.go`](../internal/messaging/types.go) - BucketDeletionEvent type
  - [`internal/messaging/interfaces.go`](../internal/messaging/interfaces.go) - ISubscriber with ParseBucketDeletionEvents
  - [`internal/messaging/jetstream.go`](../internal/messaging/jetstream.go) - MinIO/NATS deletion event parsing
  - [`internal/messaging/aws.go`](../internal/messaging/aws.go) - AWS S3/SQS deletion event parsing
  - [`internal/messaging/gcp.go`](../internal/messaging/gcp.go) - GCP Pub/Sub deletion event parsing

---

## 1. Executive Summary

### 1.1 Overview
SafeBucket currently implements hard deletion of files, which permanently removes objects from both the database and cloud storage providers (AWS S3, GCP Cloud Storage, MinIO). This PRD defines a trash/soft-delete feature that provides users with a 7-day recovery window before permanent deletion, improving data safety and user experience.

### 1.2 Objectives
- Prevent accidental data loss through a recovery window
- Implement role-based access control for file restoration
- Maintain compatibility with all supported storage providers (AWS, GCP, MinIO)
- Ensure automatic cleanup of expired trash items
- Provide full audit trail for trash and restore operations

### 1.3 Success Metrics
- Zero permanent data loss incidents from accidental deletion
- <100ms API response time for trash/restore operations
- 100% audit coverage for all trash-related actions
- 95%+ user satisfaction with trash feature (post-launch survey)

---

## 2. Background & Context

### 2.1 Current State
SafeBucket exposes S3 buckets directly to end users through secure scoped credentials for GET and PUT operations. The existing DELETE method (`DeleteFile` in `internal/services/bucket.go:368`) performs immediate hard deletion:
- Files: Deleted from database and storage via `Storage.RemoveObject()`
- Folders: Marked as `deleting` status and processed asynchronously

### 2.2 Problem Statement
Users have no recovery mechanism after file deletion, leading to:
- Risk of accidental data loss
- Support burden for recovery requests
- Compliance concerns for regulated industries
- Competitive disadvantage vs. Google Drive, Dropbox, etc.

### 2.3 User Personas
1. **Content Creator (Primary)**: Uploads/manages files, occasionally deletes by mistake
2. **Team Owner (Secondary)**: Manages bucket permissions, needs to restore team members' deletions
3. **Administrator (Tertiary)**: Requires audit trails and bulk management capabilities

---

## 3. Functional Requirements

### 3.1 Core Features

#### 3.1.1 Soft Delete (Trash)
**User Story:** As a user with `erase` permission, I want deleted files to move to trash instead of being permanently deleted, so I can recover from mistakes.

**Acceptance Criteria:**
- When a user deletes a file, it is marked as `trashed` in the database
- Original file remains in storage location with trash metadata tag
- Trashed files are hidden from default file listings
- Trash timestamp and user ID are recorded
- Activity log entry is created with action type `erase`

**Technical Specs:**
- API Endpoint: `DELETE /api/buckets/{id}/files/{fileId}` (existing, behavior changed)
- Required Permission: `rbac.ActionErase` (existing)
- Database Updates:
  ```sql
  UPDATE files SET
    status = 'trashed',
    trashed_at = NOW(),
    trashed_by = {user_id}
  WHERE id = {file_id};
  ```
- Storage Operation: Add object tags `Status=trashed`, `TrashedAt={timestamp}`

#### 3.1.2 File Restoration
**User Story:** As a user with `restore` permission, I want to restore files from trash, so I can recover accidentally deleted content.

**Acceptance Criteria:**
- Users can restore files within 7 days of deletion
- Restored files return to original location and status
- Restoration fails gracefully if target path has naming conflict
- Activity log entry is created with action type `restore`

**Technical Specs:**
- API Endpoint: `POST /api/buckets/{id}/files/{fileId}/restore` (new)
- Required Permission: `rbac.ActionRestore` (new)
- Database Updates:
  ```sql
  UPDATE files SET
    status = 'uploaded',
    trashed_at = NULL,
    trashed_by = NULL
  WHERE id = {file_id} AND status = 'trashed';
  ```
- Storage Operation: Remove `Status=trashed` tag from object

#### 3.1.3 Trash Listing
**User Story:** As a user with `read` permission, I want to view all trashed files in a bucket, so I can decide what to restore.

**Acceptance Criteria:**
- Endpoint returns all files with `status=trashed` and `trashed_at` within 7 days
- Results ordered by trash date (newest first)
- Response includes trashed_by user information
- Includes calculated days remaining until permanent deletion

**Technical Specs:**
- API Endpoint: `GET /api/buckets/{id}/trash` (new)
- Required Permission: `rbac.ActionRead`
- Query:
  ```sql
  SELECT * FROM files
  WHERE bucket_id = {bucket_id}
    AND status = 'trashed'
    AND trashed_at > NOW() - INTERVAL '7 days'
  ORDER BY trashed_at DESC;
  ```

#### 3.1.4 Permanent Delete from Trash (Purge)
**User Story:** As a bucket owner, I want to permanently delete a specific trashed file immediately instead of waiting for automatic cleanup, so I can free up storage space.

**Acceptance Criteria:**
- Only bucket owners/admins can permanently delete from trash
- File must be in trash status
- Confirmation required before execution
- Operation is irreversible
- Activity log entry created with "purge" action

**Technical Specs:**
- API Endpoint: `DELETE /api/buckets/{id}/trash/{fileId}` (new)
- Required Permission: `rbac.ActionPurge` (new, owner/admin only)
- Database Operations:
  ```sql
  -- Verify file is in trash
  SELECT * FROM files
  WHERE id = {file_id}
    AND bucket_id = {bucket_id}
    AND status = 'trashed';

  -- Soft delete (allows activity enrichment)
  DELETE FROM files WHERE id = {file_id};
  ```
- Storage Operations:
  ```go
  // Delete from storage
  storage.RemoveObject(objectPath)

  // Soft delete from database (allows activity enrichment)
  db.Delete(&file)
  ```

#### 3.1.5 Automatic Permanent Deletion
**User Story:** As a system administrator, I want trashed files older than 7 days to be automatically purged, so storage costs are managed.

**Acceptance Criteria:**
- Cloud provider lifecycle policies automatically delete objects after 7 days
- Backend receives deletion events via message queue
- Event handler hard deletes database records
- Logs cleanup metrics (files deleted, events processed)
- Reconciliation process detects and fixes inconsistencies

**Technical Specs:**
- **Lifecycle Policies:** S3/GCP/MinIO delete objects tagged `Status=trashed` after 7 days
- **Event Processing:** Deletion events trigger `TrashExpiration` handler
- **Implementation:** `internal/events/trash_expiration.go`
- **Operations:**
  ```go
  // Event-driven cleanup when lifecycle policy deletes object
  db.Delete(&file)  // Soft delete from database (allows activity enrichment)
  ```

**Reconciliation Strategy:**
- **Daily reconciliation job** compares database trash records with storage objects
- **Orphan Detection:** Database records without corresponding storage objects
- **Cleanup:** Hard delete orphaned database records
- **Alerting:** Notify if orphan count exceeds threshold (>100 files/day)

### 3.2 Role-Based Access Control

#### 3.2.1 Permission Matrix

| Role         | Trash (Erase) | View Trash | Restore | Purge (Hard Delete) |
|--------------|---------------|------------|---------|---------------------|
| Guest        | ❌            | ❌         | ❌      | ❌                  |
| Viewer       | ❌            | ✅         | ❌      | ❌                  |
| Contributor  | ✅            | ✅         | ✅      | ❌                  |
| Owner        | ✅            | ✅         | ✅      | ✅                  |
| Admin        | ✅            | ✅         | ✅      | ✅                  |

#### 3.2.2 RBAC Changes
- **New Action:** `rbac.ActionRestore` added to `internal/rbac/const.go`
- **Existing Action:** `rbac.ActionErase` behavior changes from hard delete to soft delete
- **Optional Action:** `rbac.ActionPurge` for manual permanent deletion (Owner/Admin only)

**Policy Updates:**
```go
// Contributor group gains restore permission
{c.DefaultDomain, groupName, rbac.ResourceBucket.String(), bucket.ID.String(), rbac.ActionRestore.String()}

// Owner group gains purge permission (optional future feature)
{c.DefaultDomain, groupName, rbac.ResourceBucket.String(), bucket.ID.String(), rbac.ActionPurge.String()}
```

### 3.3 Multi-Provider Storage Implementation

**Unified Approach:** All three supported providers (AWS S3, GCP Cloud Storage, MinIO) support object tagging and lifecycle management. This enables a consistent implementation with provider-specific nuances only in the tagging API syntax.

#### 3.3.1 AWS S3
**Method:** Object Tagging + S3 Lifecycle Rules

**Tagging Implementation:**
```go
func (a AWSStorage) TagObjectForTrash(path string, trashedAt time.Time) error {
    _, err := a.storage.PutObjectTagging(context.Background(), &s3.PutObjectTaggingInput{
        Bucket: aws.String(a.BucketName),
        Key:    aws.String(path),
        Tagging: &types.Tagging{
            TagSet: []types.Tag{
                {Key: aws.String("Status"), Value: aws.String("trashed")},
                {Key: aws.String("TrashedAt"), Value: aws.String(trashedAt.Format(time.RFC3339))},
            },
        },
    })
    return err
}
```

**S3 Lifecycle Policy (IaC):**
```json
{
    "Rules": [{
        "Id": "SafeBucket-TrashExpiration",
        "Status": "Enabled",
        "Filter": {
            "Tag": {
                "Key": "Status",
                "Value": "trashed"
            }
        },
        "Expiration": {
            "Days": 7
        },
        "NoncurrentVersionExpiration": {
            "NoncurrentDays": 7
        }
    }]
}
```

**Restore Implementation:**
```go
func (a AWSStorage) RemoveTrashMarker(path string) error {
    _, err := a.storage.DeleteObjectTagging(context.Background(), &s3.DeleteObjectTaggingInput{
        Bucket: aws.String(a.BucketName),
        Key:    aws.String(path),
    })
    return err
}
```

#### 3.3.2 GCP Cloud Storage
**Method:** Custom Metadata + Lifecycle Management

**Implementation:**
```go
func (g GCPStorage) TagObjectForTrash(path string, trashedAt time.Time) error {
    obj := g.bucket.Object(path)
    attrs, err := obj.Attrs(context.Background())
    if err != nil {
        return err
    }

    if attrs.Metadata == nil {
        attrs.Metadata = make(map[string]string)
    }
    attrs.Metadata["status"] = "trashed"
    attrs.Metadata["trashed_at"] = trashedAt.Format(time.RFC3339)

    _, err = obj.Update(context.Background(), storage.ObjectAttrsToUpdate{
        Metadata: attrs.Metadata,
    })
    return err
}
```

**GCP Lifecycle Configuration:**
```json
{
    "lifecycle": {
        "rule": [{
            "action": {"type": "Delete"},
            "condition": {
                "matchesPrefix": ["buckets/"],
                "customTimeBefore": "7d",
                "matchesMetadata": [{"key": "status", "value": "trashed"}]
            }
        }]
    }
}
```

#### 3.3.3 MinIO
**Method:** Object Tagging + Lifecycle Rules (Same as AWS S3)

**Rationale:** MinIO has supported S3-compatible object tagging and tag-based lifecycle management since 2020 (PRs #8880, #9604). This allows for a unified implementation across all providers.

**Tagging Implementation:**
```go
func (m MinIOStorage) TagObjectForTrash(path string, trashedAt time.Time) error {
    return m.client.PutObjectTagging(context.Background(), &minio.PutObjectTaggingOptions{
        Bucket: m.BucketName,
        Object: path,
        Tags: map[string]string{
            "Status":    "trashed",
            "TrashedAt": trashedAt.Format(time.RFC3339),
        },
    })
}
```

**MinIO Lifecycle Configuration (via CLI):**
```bash
# Using MinIO Client (mc)
mc ilm rule add \
  --tags "Status=trashed" \
  --expire-days 7 \
  minio/safebucket
```

**Lifecycle Configuration (Programmatic/IaC):**
```xml
<LifecycleConfiguration>
    <Rule>
        <ID>SafeBucket-TrashExpiration</ID>
        <Status>Enabled</Status>
        <Filter>
            <Tag>
                <Key>Status</Key>
                <Value>trashed</Value>
            </Tag>
        </Filter>
        <Expiration>
            <Days>7</Days>
        </Expiration>
    </Rule>
</LifecycleConfiguration>
```

**Restore Implementation:**
```go
func (m MinIOStorage) RemoveTrashMarker(path string) error {
    return m.client.RemoveObjectTagging(context.Background(), &minio.RemoveObjectTaggingOptions{
        Bucket: m.BucketName,
        Object: path,
    })
}
```

---

## 4. Technical Architecture

### 4.1 Database Schema Changes

**File Model Updates (`internal/models/file.go`):**
```go
type FileStatus string

const (
    FileStatusUploading FileStatus = "uploading"
    FileStatusUploaded  FileStatus = "uploaded"
    FileStatusDeleting  FileStatus = "deleting"
    FileStatusTrashed   FileStatus = "trashed"    // NEW
    FileStatusRestoring FileStatus = "restoring"  // NEW (transient state)
)

type File struct {
    ID        uuid.UUID  `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
    Name      string     `gorm:"not null;default:null" json:"name"`
    Extension string     `gorm:"default:null" json:"extension"`
    Status    FileStatus `gorm:"type:file_status;default:null" json:"status"`
    BucketId  uuid.UUID  `gorm:"type:uuid;" json:"bucket_id"`
    Bucket    Bucket     `json:"-"`
    Path      string     `gorm:"not null;default:/" json:"path"`
    Type      string     `gorm:"not null;default:null" json:"type"`
    Size      int        `gorm:"default:null" json:"size"`

    // NEW FIELDS
    TrashedAt   *time.Time `gorm:"default:null;index" json:"trashed_at,omitempty"`
    TrashedBy   *uuid.UUID `gorm:"type:uuid;default:null" json:"trashed_by,omitempty"`
    TrashedUser User       `gorm:"foreignKey:TrashedBy" json:"-"`

    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
    DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // Hard delete timestamp (after 7 days)
}
```

**Migration SQL:**
```sql
-- Add new enum value
ALTER TYPE file_status ADD VALUE IF NOT EXISTS 'trashed';
ALTER TYPE file_status ADD VALUE IF NOT EXISTS 'restoring';

-- Add new columns
ALTER TABLE files ADD COLUMN trashed_at TIMESTAMP DEFAULT NULL;
ALTER TABLE files ADD COLUMN trashed_by UUID REFERENCES users(id) DEFAULT NULL;

-- Create index for trash queries
CREATE INDEX idx_files_trashed_at ON files(trashed_at) WHERE trashed_at IS NOT NULL;
CREATE INDEX idx_files_status_trashed ON files(status) WHERE status = 'trashed';
```

### 4.2 Storage Interface Updates

**Interface Definition (`internal/storage/interfaces.go`):**
```go
type IStorage interface {
    PresignedGetObject(path string) (string, error)
    PresignedPostPolicy(path string, size int, metadata map[string]string) (string, map[string]string, error)
    StatObject(path string) (map[string]string, error)
    ListObjects(prefix string, maxKeys int32) ([]string, error)
    RemoveObject(path string) error
    RemoveObjects(paths []string) error

    // NEW METHODS
    TagObjectForTrash(path string, trashedAt time.Time) error
    RemoveTrashMarker(path string) error
}
```

### 4.3 Service Layer Implementation

**New Service Methods (`internal/services/bucket.go`):**

```go
// TrashFile moves a file to trash instead of permanent deletion
func (s BucketService) TrashFile(logger *zap.Logger, user models.UserClaims, ids uuid.UUIDs) error {
    bucketId, fileId := ids[0], ids[1]

    file, err := sql.GetFileById(s.DB, bucketId, fileId)
    if err != nil {
        return err
    }

    // Don't allow trashing already-trashed files
    if file.Status == models.FileStatusTrashed {
        return errors.NewAPIError(400, "FILE_ALREADY_TRASHED")
    }

    return s.DB.Transaction(func(tx *gorm.DB) error {
        now := time.Now()

        // Update database metadata
        updates := map[string]interface{}{
            "status":     models.FileStatusTrashed,
            "trashed_at": now,
            "trashed_by": user.UserID,
        }

        if err := tx.Model(&file).Updates(updates).Error; err != nil {
            logger.Error("Failed to update file status to trashed", zap.Error(err))
            return errors.ErrorUpdateFailed
        }

        // Tag object in storage (best effort)
        objectPath := path.Join("buckets", bucketId.String(), file.Path, file.Name)
        if err := s.Storage.TagObjectForTrash(objectPath, now); err != nil {
            logger.Warn("Failed to tag object for trash in storage",
                zap.Error(err),
                zap.String("path", objectPath))
            // Continue - database is source of truth
        }

        // Log activity
        action := models.Activity{
            Message: activity.FileTrashed,
            Filter: activity.NewLogFilter(map[string]string{
                "action":      rbac.ActionErase.String(),
                "bucket_id":   bucketId.String(),
                "file_id":     fileId.String(),
                "domain":      c.DefaultDomain,
                "object_type": rbac.ResourceFile.String(),
                "user_id":     user.UserID.String(),
            }),
        }

        if err := s.ActivityLogger.Send(action); err != nil {
            logger.Error("Failed to log trash activity", zap.Error(err))
            return err
        }

        return nil
    })
}

// RestoreFile recovers a file from trash
func (s BucketService) RestoreFile(logger *zap.Logger, user models.UserClaims, ids uuid.UUIDs) error {
    bucketId, fileId := ids[0], ids[1]

    file, err := sql.GetFileById(s.DB, bucketId, fileId)
    if err != nil {
        return err
    }

    // Validate file is in trash
    if file.Status != models.FileStatusTrashed {
        return errors.NewAPIError(400, "FILE_NOT_IN_TRASH")
    }

    // Check if expired (extra safety check)
    if file.TrashedAt != nil && time.Since(*file.TrashedAt) > 7*24*time.Hour {
        return errors.NewAPIError(410, "FILE_TRASH_EXPIRED")
    }

    // Check for naming conflicts
    var existingFile models.File
    result := s.DB.Where(
        "bucket_id = ? AND name = ? AND path = ? AND status != ?",
        bucketId, file.Name, file.Path, models.FileStatusTrashed,
    ).First(&existingFile)

    if result.RowsAffected > 0 {
        return errors.NewAPIError(409, "FILE_NAME_CONFLICT")
    }

    return s.DB.Transaction(func(tx *gorm.DB) error {
        // Restore in database
        updates := map[string]interface{}{
            "status":     models.FileStatusUploaded,
            "trashed_at": nil,
            "trashed_by": nil,
        }

        if err := tx.Model(&file).Updates(updates).Error; err != nil {
            logger.Error("Failed to restore file status", zap.Error(err))
            return errors.ErrorUpdateFailed
        }

        // Remove trash marker from storage
        objectPath := path.Join("buckets", bucketId.String(), file.Path, file.Name)
        if err := s.Storage.RemoveTrashMarker(objectPath); err != nil {
            logger.Warn("Failed to remove trash tag from storage",
                zap.Error(err),
                zap.String("path", objectPath))
            // Continue - database restoration is most critical
        }

        // Log activity
        action := models.Activity{
            Message: activity.FileRestored,
            Filter: activity.NewLogFilter(map[string]string{
                "action":      rbac.ActionRestore.String(),
                "bucket_id":   bucketId.String(),
                "file_id":     fileId.String(),
                "domain":      c.DefaultDomain,
                "object_type": rbac.ResourceFile.String(),
                "user_id":     user.UserID.String(),
            }),
        }

        if err := s.ActivityLogger.Send(action); err != nil {
            logger.Error("Failed to log restore activity", zap.Error(err))
            return err
        }

        return nil
    })
}

// ListTrashedFiles returns all trashed files for a bucket within 7-day window
func (s BucketService) ListTrashedFiles(logger *zap.Logger, user models.UserClaims, ids uuid.UUIDs) ([]models.File, error) {
    var files []models.File

    cutoffDate := time.Now().Add(-7 * 24 * time.Hour)

    result := s.DB.
        Preload("TrashedUser").
        Where(
            "bucket_id = ? AND status = ? AND trashed_at > ?",
            ids[0],
            models.FileStatusTrashed,
            cutoffDate,
        ).
        Order("trashed_at DESC").
        Find(&files)

    if result.Error != nil {
        logger.Error("Failed to list trashed files", zap.Error(result.Error))
        return nil, errors.NewAPIError(500, "TRASH_LIST_FAILED")
    }

    return files, nil
}

// PurgeFile permanently deletes a file from trash (hard delete)
func (s BucketService) PurgeFile(logger *zap.Logger, user models.UserClaims, ids uuid.UUIDs) error {
    bucketId, fileId := ids[0], ids[1]

    file, err := sql.GetFileById(s.DB, bucketId, fileId)
    if err != nil {
        return err
    }

    // Validate file is in trash
    if file.Status != models.FileStatusTrashed {
        return errors.NewAPIError(400, "FILE_NOT_IN_TRASH")
    }

    return s.DB.Transaction(func(tx *gorm.DB) error {
        objectPath := path.Join("buckets", bucketId.String(), file.Path, file.Name)

        // Delete from storage
        if err := s.Storage.RemoveObject(objectPath); err != nil {
            logger.Warn("Failed to delete file from storage",
                zap.Error(err),
                zap.String("path", objectPath))
            // Continue to database deletion even if storage fails
        }

        // Soft delete from database (allows activity enrichment)
        if err := tx.Delete(&file).Error; err != nil {
            logger.Error("Failed to soft delete file from database", zap.Error(err))
            return errors.ErrorDeleteFailed
        }

        // Log activity
        action := models.Activity{
            Message: activity.FilePurged,
            Filter: activity.NewLogFilter(map[string]string{
                "action":      rbac.ActionPurge.String(),
                "bucket_id":   bucketId.String(),
                "file_id":     fileId.String(),
                "domain":      c.DefaultDomain,
                "object_type": rbac.ResourceFile.String(),
                "user_id":     user.UserID.String(),
            }),
        }

        if err := s.ActivityLogger.Send(action); err != nil {
            logger.Error("Failed to log purge activity", zap.Error(err))
            return err
        }

        return nil
    })
}
```

**Routing Changes:**
```go
func (s BucketService) Routes() chi.Router {
    r := chi.NewRouter()

    // ... existing routes ...

    r.Route("/{id0}", func(r chi.Router) {
        // ... existing bucket routes ...

        // NEW: Trash endpoints
        r.Route("/trash", func(r chi.Router) {
            // List trashed files
            r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionRead, 0)).
                Get("/", handlers.GetListHandler(s.ListTrashedFiles))

            // Purge individual file from trash
            r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionPurge, 0)).
                Delete("/{id1}", handlers.DeleteHandler(s.PurgeFile))
        })

        r.Route("/files/{id1}", func(r chi.Router) {
            // CHANGED: Delete now moves to trash
            r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionErase, 0)).
                Delete("/", handlers.DeleteHandler(s.TrashFile))

            // NEW: Restore endpoint
            r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionRestore, 0)).
                Post("/restore", handlers.UpdateHandler(s.RestoreFile))

            r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionDownload, 0)).
                Get("/download", handlers.GetOneHandler(s.DownloadFile))
        })
    })

    return r
}
```

### 4.4 Event-Driven Trash Expiration

**Architecture:** Instead of polling with cron jobs, trash cleanup leverages cloud-native lifecycle policies and event notifications. When S3/GCP/MinIO automatically deletes objects after 7 days, the backend receives deletion events and updates the database accordingly.

**Trash Expiration Event (`internal/events/trash_expiration.go`):**
```go
package events

import (
    "api/internal/models"
    "encoding/json"
    "path"

    "github.com/ThreeDotsLabs/watermill"
    "github.com/ThreeDotsLabs/watermill/message"
    "github.com/google/uuid"
    "go.uber.org/zap"
)

const TrashExpirationName = "TrashExpiration"
const TrashExpirationPayloadName = "TrashExpirationPayload"

type TrashExpirationPayload struct {
    Type      string    `json:"type"`
    BucketId  uuid.UUID `json:"bucket_id"`
    ObjectKey string    `json:"object_key"`  // e.g., "buckets/{id}/path/file.txt"
}

type TrashExpiration struct {
    Payload TrashExpirationPayload
}

func NewTrashExpirationFromBucketEvent(bucketId uuid.UUID, objectKey string) *TrashExpiration {
    return &TrashExpiration{
        Payload: TrashExpirationPayload{
            Type:      TrashExpirationName,
            BucketId:  bucketId,
            ObjectKey: objectKey,
        },
    }
}

func (e *TrashExpiration) callback(params *EventParams) error {
    zap.L().Info("Processing trash expiration event",
        zap.String("bucket_id", e.Payload.BucketId.String()),
        zap.String("object_key", e.Payload.ObjectKey),
    )

    // Parse object key to extract path and filename
    // Expected format: "buckets/{bucket_id}/path/to/file.ext"
    objectPath := e.Payload.ObjectKey
    if len(objectPath) > len("buckets/"+e.Payload.BucketId.String()+"/") {
        objectPath = objectPath[len("buckets/"+e.Payload.BucketId.String()+"/"):]
    }

    dir := path.Dir(objectPath)
    filename := path.Base(objectPath)

    zap.L().Debug("Parsed object path",
        zap.String("directory", dir),
        zap.String("filename", filename),
    )

    // Find the file in database
    var file models.File
    result := params.DB.Where(
        "bucket_id = ? AND path = ? AND name = ? AND status = ?",
        e.Payload.BucketId,
        dir,
        filename,
        models.FileStatusTrashed,
    ).First(&file)

    if result.Error != nil {
        // File not found or not in trash - might be already cleaned up
        zap.L().Warn("File not found in trash, skipping cleanup",
            zap.String("bucket_id", e.Payload.BucketId.String()),
            zap.String("path", dir),
            zap.String("name", filename),
            zap.Error(result.Error),
        )
        return nil
    }

    // Verify file is actually expired (safety check)
    if file.TrashedAt != nil {
        daysSinceTrashed := time.Since(*file.TrashedAt).Hours() / 24
        if daysSinceTrashed < 7 {
            zap.L().Error("Received expiration event for non-expired file",
                zap.String("file_id", file.ID.String()),
                zap.Float64("days_in_trash", daysSinceTrashed),
            )
            return errors.New("file not yet expired")
        }
    }

    // Soft delete from database (allows activity enrichment)
    if err := params.DB.Delete(&file).Error; err != nil {
        zap.L().Error("Failed to soft delete file from database",
            zap.String("file_id", file.ID.String()),
            zap.Error(err),
        )
        return err
    }

    zap.L().Info("Successfully processed trash expiration",
        zap.String("file_id", file.ID.String()),
        zap.String("name", file.Name),
    )

    return nil
}
```

**Bucket Event Handler Integration (`internal/events/handler.go`):**

Extend the existing `HandleBucketEvents` function to process deletion events:

```go
func HandleBucketEvents(
    subscriber messaging.ISubscriber,
    db *gorm.DB,
    activityLogger activity.IActivityLogger,
    messages <-chan *message.Message,
) {
    for msg := range messages {
        zap.L().Debug("message received", zap.Any("raw_payload", string(msg.Payload)))

        // Parse upload events (existing logic)
        uploadEvents := subscriber.ParseBucketUploadEvents(msg)
        for _, event := range uploadEvents {
            // ... existing upload handling ...
        }

        // Parse deletion events (NEW)
        deletionEvents := subscriber.ParseBucketDeletionEvents(msg)
        for _, event := range deletionEvents {
            // Check if deleted object was in trash
            trashEvent := NewTrashExpirationFromBucketEvent(event.BucketId, event.ObjectKey)

            params := &EventParams{
                DB:             db,
                ActivityLogger: activityLogger,
            }

            if err := trashEvent.callback(params); err != nil {
                zap.L().Error("Failed to process trash expiration", zap.Error(err))
                // Continue processing other events
            }
        }

        msg.Ack()
    }
}
```

**Subscriber Interface Extension (`internal/messaging/interfaces.go`):**

Add method to parse deletion events:

```go
type ISubscriber interface {
    Subscribe() <-chan *message.Message
    Close() error
    ParseBucketUploadEvents(msg *message.Message) []BucketUploadEvent
    ParseBucketDeletionEvents(msg *message.Message) []BucketDeletionEvent  // NEW
}
```

**Deletion Event Type (`internal/messaging/types.go`):**

```go
type BucketDeletionEvent struct {
    BucketId  uuid.UUID `json:"bucket_id"`
    ObjectKey string    `json:"object_key"`
    EventName string    `json:"event_name"`  // e.g., "s3:ObjectRemoved:Delete"
}
```

### 4.5 Reconciliation System

**Purpose:** Ensure database and storage consistency when events are lost or delayed.

**Scheduler Integration:**

Add reconciliation to existing scheduler infrastructure:

```go
// In main.go or scheduler setup
import (
    "api/internal/jobs"
    "github.com/robfig/cron/v3"
)

func setupScheduledJobs(db *gorm.DB, storage storage.IStorage) {
    c := cron.New()

    // Run reconciliation daily at 02:00 UTC (offset from peak hours)
    reconciliation := jobs.NewTrashReconciliationJob(db, storage)
    c.AddFunc("0 2 * * *", func() {
        if err := reconciliation.Run(); err != nil {
            zap.L().Error("Trash reconciliation failed", zap.Error(err))
        }
    })

    c.Start()
    zap.L().Info("Scheduled jobs started")
}
```

**Monitoring:**
- Metric: `trash_reconciliation_orphans_found`: Count of orphaned records cleaned per run
- Alert: `OrphanedTrashRecordsHigh` if count >100 in single run
- Dashboard: Graph showing orphan count over time to detect event delivery degradation

### 4.6 Asynchronous Folder Operations (V1.4)

**Architecture:** Folder operations (trash, restore, purge) use the same asynchronous event pattern as `ObjectDeletion` to prevent HTTP timeouts when processing large folders with thousands of files. Operations return immediately while background workers process children in batches.

#### 4.6.1 FolderRestore Event (`internal/events/folder_restore.go`)

**Event Structure:**
```go
type FolderRestorePayload struct {
    Type     string
    BucketId uuid.UUID
    FolderId uuid.UUID
    UserId   uuid.UUID
}
```

**Processing Logic:**
1. **Validation**: Verify folder is in `restoring` status
2. **Expiration Check**: Ensure folder not expired (>7 days in trash)
3. **Conflict Check**: Verify no naming conflicts at restore location
4. **Batch Restore**: Restore children in batches (`c.BulkActionsLimit`)
5. **Timestamp Filter**: Only restore children with matching `trashed_at`
6. **Requeue**: If more children remain, return error to trigger retry
7. **Activity Log**: Log `FOLDER_RESTORED` when complete

**Error Handling:**
- If expired: Reverts folder status back to `trashed`
- If conflict detected: Reverts folder status back to `trashed`
- Storage tag removal failures are logged but non-blocking

#### 4.6.2 FolderTrash Event (`internal/events/folder_trash.go`)

**Event Structure:**
```go
type FolderTrashPayload struct {
    Type      string
    BucketId  uuid.UUID
    FolderId  uuid.UUID
    UserId    uuid.UUID
    TrashedAt time.Time
}
```

**Processing Logic:**
1. **Verify Status**: Check folder is in `trashed` status
2. **Find Children**: Query all children using `path LIKE` pattern
3. **Batch Trash**: Update children in batches (`c.BulkActionsLimit`)
4. **Storage Tagging**: Tag all children for lifecycle cleanup
5. **Requeue**: If more children remain, return error to trigger retry
6. **Activity Log**: Log `FOLDER_TRASHED` when complete

**Features:**
- All children get same `trashed_at` timestamp as parent folder
- Enables timestamp-based restore logic
- Best-effort storage tagging (failures logged)

#### 4.6.3 FolderPurge Event (`internal/events/folder_purge.go`)

**Event Structure:**
```go
type FolderPurgePayload struct {
    Type     string
    BucketId uuid.UUID
    FolderId uuid.UUID
    UserId   uuid.UUID
}
```

**Processing Logic:**
1. **Verify Status**: Check folder is in `trashed` status
2. **Find Children**: Query all children (no status filter)
3. **Batch Delete**: Soft delete children in batches
4. **Storage Cleanup**: Batch delete from storage
5. **Requeue**: If more children remain, return error to trigger retry
6. **Delete Folder**: Once all children purged, delete folder itself
7. **Activity Log**: Log `FOLDER_PURGED` when complete

**Safety:**
- Uses soft delete to allow activity enrichment to still find records
- Storage deletion failures are logged but don't block database cleanup
- Only accessible to users with `ActionPurge` permission (Owner/Admin)

#### 4.6.4 Service Layer Integration

**Performance Benefits:**
- ✅ No HTTP timeouts for folders with 1000+ files
- ✅ Reduced database lock contention (batch transactions)
- ✅ Better user experience (immediate response)
- ✅ Automatic retry via event requeuing
- ✅ Scalable for large folder hierarchies

### 4.7 Activity Logging

**New Activity Messages (`internal/activity/const.go`):**
```go
const (
    // ... existing messages ...
    FileDeleted    = "File deleted"     // Existing
    FileTrashed    = "File moved to trash"     // NEW
    FileRestored   = "File restored from trash" // NEW
    FilePurged     = "File permanently deleted" // NEW (future use)
)
```

---

## 5. API Specification

### 5.1 Trash File (Soft Delete)
**Endpoint:** `DELETE /api/buckets/{bucketId}/files/{fileId}`

**Request:**
```http
DELETE /api/buckets/550e8400-e29b-41d4-a716-446655440000/files/123e4567-e89b-12d3-a456-426614174000 HTTP/1.1
Authorization: Bearer {jwt_token}
```

**Response (200 OK):**
```json
{
    "success": true,
    "message": "File moved to trash"
}
```

**Error Responses:**
- `400 FILE_ALREADY_TRASHED`: File is already in trash
- `404 FILE_NOT_FOUND`: File does not exist
- `403 FORBIDDEN`: User lacks `erase` permission

### 5.2 Restore File
**Endpoint:** `POST /api/buckets/{bucketId}/files/{fileId}/restore`

**Request:**
```http
POST /api/buckets/550e8400-e29b-41d4-a716-446655440000/files/123e4567-e89b-12d3-a456-426614174000/restore HTTP/1.1
Authorization: Bearer {jwt_token}
```

**Response (200 OK):**
```json
{
    "success": true,
    "message": "File restored successfully",
    "data": {
        "id": "123e4567-e89b-12d3-a456-426614174000",
        "name": "document.pdf",
        "status": "uploaded",
        "path": "/documents",
        "size": 2048576,
        "created_at": "2025-10-13T14:30:00Z",
        "updated_at": "2025-10-20T10:15:00Z"
    }
}
```

**Error Responses:**
- `400 FILE_NOT_IN_TRASH`: File is not in trash state
- `409 FILE_NAME_CONFLICT`: Another file exists at target location
- `410 FILE_TRASH_EXPIRED`: File exceeded 7-day retention
- `403 FORBIDDEN`: User lacks `restore` permission

### 5.3 List Trash
**Endpoint:** `GET /api/buckets/{bucketId}/trash`

**Request:**
```http
GET /api/buckets/550e8400-e29b-41d4-a716-446655440000/trash HTTP/1.1
Authorization: Bearer {jwt_token}
```

**Response (200 OK):**
```json
{
    "success": true,
    "data": [
        {
            "id": "123e4567-e89b-12d3-a456-426614174000",
            "name": "old-report.pdf",
            "extension": "pdf",
            "status": "trashed",
            "path": "/reports/2025",
            "type": "file",
            "size": 1024000,
            "trashed_at": "2025-10-15T09:20:00Z",
            "trashed_by": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
            "trashed_user": {
                "id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
                "email": "user@example.com",
                "name": "John Doe"
            },
            "days_until_deletion": 2,
            "created_at": "2025-09-01T10:00:00Z"
        }
    ]
}
```

---

## 6. Frontend Requirements

### 6.1 UI Components

#### 6.1.1 File List Updates
**Location:** `web/components/bucket-view/`

**Changes:**
- Add trash icon badge to trashed files (if shown)
- Default filter: Hide files with `status=trashed`
- Add toggle: "Show trashed files" checkbox
- Visual indicator: Grayed out/strikethrough for trashed items in list

#### 6.1.2 Trash View Page
**Location:** `web/routes/buckets/{bucketId}/trash.tsx` (new)

**Components:**
- **Header:** "Trash - {Bucket Name}"
- **Empty State:** "No items in trash" with illustration
- **File List Table:**
  - Columns: Name, Size, Deleted By, Deleted Date, Days Remaining, Actions
  - Sort by: Deleted Date (desc default)
  - Bulk actions: Restore selected
- **Countdown Timer:** Visual indicator of days/hours until permanent deletion
- **Search/Filter:** Filter by name, date range

#### 6.1.3 Context Menu Actions
**Location:** File row right-click menu / action dropdown

**Changes:**
- **Default State:** "Delete" → "Move to Trash" (icon change to trash can)
- **Trash State:** Show "Restore" action (conditional on `restore` permission)

#### 6.1.4 Confirmation Dialogs
**Delete Confirmation:**
```
Title: Move to Trash?
Body: "{filename}" will be moved to trash and permanently deleted after 7 days. You can restore it before then.
Actions: [Cancel] [Move to Trash]
```

**Restore Confirmation:**
```
Title: Restore File?
Body: "{filename}" will be restored to {path}
Actions: [Cancel] [Restore]
```

### 6.2 State Management

**BucketViewProvider Updates:**
```typescript
interface BucketViewState {
    // ... existing state ...
    showTrashedFiles: boolean;      // NEW
    trashedFiles: File[];           // NEW
    loadingTrash: boolean;          // NEW
}

// New actions
type BucketViewAction =
    | { type: 'SET_SHOW_TRASHED'; payload: boolean }
    | { type: 'SET_TRASHED_FILES'; payload: File[] }
    | { type: 'TRASH_FILE_SUCCESS'; payload: string }    // file ID
    | { type: 'RESTORE_FILE_SUCCESS'; payload: string }  // file ID
    // ... existing actions
```

**Custom Hooks:**
```typescript

```

### 6.3 Routing Updates

**New Route:**
```typescript
// web/routes/buckets.$bucketId.trash.tsx
import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/buckets/$bucketId/trash')({
    component: TrashView,
    loader: async ({ params }) => {
        // Prefetch trash data
        return queryClient.ensureQueryData({
            queryKey: ['trash', params.bucketId],
            queryFn: () => api.get(`/buckets/${params.bucketId}/trash`),
        })
    },
})
```

**Navigation Update:**
```typescript
// Add to bucket sidebar navigation
<NavItem
    to="/buckets/$bucketId/trash"
    params={{ bucketId }}
    icon={<TrashIcon />}
>
    Trash
</NavItem>
```

---

## 7. Testing Requirements

### 7.1 Backend Tests

#### 7.1.1 Unit Tests
**File:** `internal/services/bucket_test.go`



#### 7.1.2 Integration Tests
```go
func TestTrashWorkflow(t *testing.T) {
    // End-to-end: Upload → Trash → List Trash → Restore → Verify
    // End-to-end: Upload → Trash → Wait 7 days (mock) → Verify cleanup
}

func TestStorageProviderTrash(t *testing.T) {
    // Test AWS S3 tagging implementation
    // Test GCP metadata implementation
    // Test MinIO tagging implementation (unified with AWS)
}
```

### 7.2 Frontend Tests

#### 7.2.1 Component Tests
**File:** `web/components/bucket-view/__tests__/TrashView.test.tsx`

```typescript
describe('TrashView', () => {
    it('displays trashed files with countdown', async () => {
        // Render component with mock data
        // Verify file list renders
        // Verify countdown calculation is correct
    });

    it('handles restore action', async () => {
        // Click restore button
        // Verify confirmation dialog appears
        // Confirm action
        // Verify API call and optimistic update
    });

    it('shows empty state when no trashed files', () => {
        // Render with empty trash
        // Verify empty state message
    });

    it('filters trash by search query', async () => {
        // Enter search term
        // Verify filtered results
    });
});
```

#### 7.2.2 E2E Tests
**File:** `e2e/trash-workflow.spec.ts`

```typescript
test('complete trash workflow', async ({ page }) => {
    // Login
    await login(page);

    // Upload file
    await uploadFile(page, 'test.pdf');

    // Delete (trash) file
    await page.click('[data-testid="file-menu"]');
    await page.click('[data-testid="trash-action"]');
    await page.click('[data-testid="confirm-trash"]');

    // Navigate to trash
    await page.click('[data-testid="trash-nav"]');
    await expect(page.locator('text=test.pdf')).toBeVisible();

    // Restore file
    await page.click('[data-testid="restore-action"]');
    await page.click('[data-testid="confirm-restore"]');

    // Verify file restored
    await page.goto('/buckets/test-bucket');
    await expect(page.locator('text=test.pdf')).toBeVisible();
});
```

### 7.3 Load Testing

**Scenario:** Bulk trash cleanup job performance
- **Setup:** 100,000 files in trash (expired)
- **Target:** Complete cleanup in <10 minutes
- **Metrics:** DB query time, storage API calls, memory usage

---

## 8. Security Considerations

### 8.1 Authentication & Authorization
- All trash endpoints require valid JWT authentication
- Permissions validated via Casbin RBAC before operations
- Trash listing respects bucket-level permissions

### 8.2 Data Integrity
- Database is source of truth for trash state
- Storage tagging failures are logged but non-blocking
- Restore operations validate no naming conflicts

### 8.3 Audit Trail
- All trash/restore actions logged to Loki
- Activity logs include user ID, timestamp, action type
- Logs retained per company retention policy

### 8.4 Privacy & Compliance
- Permanent deletion after 7 days complies with GDPR "right to erasure"
- Trash retention period configurable for regulatory requirements
- Admin purge action available for immediate compliance needs (future)

---

## 9. Performance Requirements

### 9.1 API Response Times
- Trash file: <100ms (p95)
- Restore file: <150ms (p95)
- List trash: <200ms for <1000 files (p95)

### 9.2 Background Job Performance
- Cleanup job: Process 10,000 files/minute
- Total runtime: <30 minutes for 100k files

### 9.3 Database Performance
- Trash queries use `idx_files_trashed_at` index
- Cleanup query uses composite index on (status, trashed_at)

---

## 10. Monitoring & Alerting

### 10.1 Metrics
- `trash_operations_total{action="trash|restore"}`: Counter of operations
- `trash_cleanup_files_deleted`: Gauge of files cleaned up per run
- `trash_cleanup_duration_seconds`: Histogram of cleanup job runtime
- `trash_api_duration_seconds`: Histogram of API response times

### 10.2 Alerts
- **Critical:** Cleanup job fails for >24 hours
- **Warning:** Trash restore rate >50% (may indicate UX issue)
- **Info:** Trash size >100k files (storage cost concern)

### 10.3 Dashboards
- Trash operations over time (line chart)
- Top users by trash volume (bar chart)
- Average time to restore (gauge)
- Storage space in trash (pie chart)

---

## 11. Migration & Rollout Plan

### 11.1 Phase 1: Database Migration (Week 1)
**Tasks:**
- Deploy database schema changes
- Run migration to add `trashed_at`, `trashed_by` columns
- Create indexes
- Verify schema in staging

**Rollback:** Drop columns if issues detected

### 11.2 Phase 2: Backend API Deployment (Week 2)
**Tasks:**
- Deploy storage interface updates (AWS, GCP, MinIO)
- Deploy service layer changes
- Deploy RBAC policy updates
- Enable feature flag `TRASH_ENABLED=false` (soft launch)

**Testing:**
- Smoke tests in staging
- Load test cleanup job

**Rollback:** Revert code deployment, feature flag off

### 11.3 Phase 3: Lifecycle Policy & Event Configuration (Week 2)

**Tasks:**

**AWS S3:**
- Deploy S3 Lifecycle Policy via Terraform/CloudFormation:
  ```hcl
  resource "aws_s3_bucket_lifecycle_configuration" "trash_expiration" {
    bucket = aws_s3_bucket.safebucket.id

    rule {
      id     = "trash-expiration"
      status = "Enabled"

      filter {
        tag {
          key   = "Status"
          value = "trashed"
        }
      }

      expiration {
        days = 7
      }
    }
  }
  ```

- Configure S3 Event Notifications:
  ```hcl
  resource "aws_s3_bucket_notification" "trash_deletion" {
    bucket = aws_s3_bucket.safebucket.id

    queue {
      queue_arn     = aws_sqs_queue.bucket_events.arn
      events        = ["s3:ObjectRemoved:Delete", "s3:LifecycleExpiration:Delete"]
      filter_prefix = "buckets/"
    }
  }
  ```

**GCP Cloud Storage:**
- Deploy GCP Lifecycle Policy via Terraform:
  ```hcl
  resource "google_storage_bucket" "safebucket" {
    name     = "safebucket"
    location = "US"

    lifecycle_rule {
      condition {
        age                = 7
        matches_prefix     = ["buckets/"]
        matches_metadata = {
          status = "trashed"
        }
      }
      action {
        type = "Delete"
      }
    }
  }
  ```

- Configure Pub/Sub Notifications:
  ```hcl
  resource "google_storage_notification" "trash_deletion" {
    bucket         = google_storage_bucket.safebucket.name
    payload_format = "JSON_API_V1"
    topic          = google_pubsub_topic.bucket_events.id
    event_types    = ["OBJECT_DELETE"]
  }
  ```

**MinIO:**
- Deploy Lifecycle Policy via MinIO Client:
  ```bash
  mc ilm rule add \
    --tags "Status=trashed" \
    --expire-days 7 \
    --transition-days 0 \
    minio/safebucket
  ```


**Testing:**
- Tag test object as `Status=trashed`
- Wait 7 days (or modify test rule to 1 minute for testing)
- Verify lifecycle policy deletes object
- Verify event is received by backend
- Verify database record is hard deleted

**Rollback:**
- Remove lifecycle policies via Terraform
- Remove event notification configurations

### 11.4 Phase 4: Event Handler Deployment (Week 2-3) - ✅ **COMPLETED (V1.5)**

**Tasks:** ✅ **ALL COMPLETE**
- ✅ Implement `ParseBucketDeletionEvents()` in subscriber implementations:
  - ✅ AWS SQS subscriber (`internal/messaging/aws.go`)
  - ✅ GCP Pub/Sub subscriber (`internal/messaging/gcp.go`)
  - ✅ NATS/JetStream subscriber for MinIO (`internal/messaging/jetstream.go`)
- ✅ Deploy trash expiration event handler (existing: `internal/events/trash_expiration.go`)
- ✅ Register `TrashExpiration` event in event registry (existing: `internal/events/registry.go`)
- ✅ Update event routing to process deletion events (`internal/events/handler.go`)
- ⏳ Test event processing in staging with synthetic events (PENDING - requires infrastructure)

**Implementation Details (V1.5):**
- **BucketDeletionEvent Type**: Added to `internal/messaging/types.go` with fields: BucketId, ObjectKey, EventName
- **ISubscriber Interface**: Extended with `ParseBucketDeletionEvents(*message.Message) []BucketDeletionEvent`
- **Event Parsing**: All three providers parse deletion events and extract bucket ID from object key path
- **HandleBucketEvents**: Updated to process both upload and deletion events in same message loop
- **Event Types Supported**:
  - AWS/MinIO: `ObjectRemoved:*`, `LifecycleExpiration:*`
  - GCP: `OBJECT_DELETE`

**Testing:** ⏳ **PENDING INFRASTRUCTURE**
- ⏳ Send mock deletion event to event queue
- ⏳ Verify TrashExpiration handler processes event
- ⏳ Verify database record is deleted
- ⏳ Check logs for proper error handling

**Build Verification:** ✅ **PASSED**
- ✅ `go build` successful (101MB binary)
- ✅ `go vet` passed for messaging package
- ✅ `go vet` passed for events package

**Rollback:**
- Remove event handler registration
- Events will accumulate in queue but won't be processed

### 11.5 Phase 5: Frontend Deployment (Week 3)
**Tasks:**
- Deploy trash UI components
- Deploy routing updates
- Enable feature flag `TRASH_ENABLED=true` for internal users (dogfooding)

**Testing:**
- Internal team beta testing
- Gather feedback on UX

**Rollback:** Feature flag off, hide UI elements

### 11.6 Phase 6: Full Production Rollout (Week 4)
**Tasks:**
- Enable `TRASH_ENABLED=true` for all users
- Verify event-driven cleanup is functioning:
  - Check event queue has active consumers
  - Monitor deletion event processing metrics
  - Verify no message backlog in event queues
- Monitor metrics dashboard
- Publish changelog/docs

**Success Criteria:**
- <0.1% error rate on trash endpoints
- Zero data loss incidents
- Event processing latency <5 seconds
- Positive user feedback

**Rollback:**
- Disable feature flag
- Pause lifecycle policy execution (if necessary)

---

## 12. Documentation Requirements

### 12.1 User-Facing Documentation
- **Help Center Article:** "How Trash Works in SafeBucket"
- **FAQ:** "How long do deleted files stay in trash?"
- **Video Tutorial:** Trash and restore workflow (2 min)

### 12.2 Developer Documentation
- **API Reference:** OpenAPI/Swagger spec updates
- **Architecture Diagram:** Trash flow visualization
- **Runbook:** Troubleshooting trash cleanup job failures

### 12.3 Internal Documentation
- **Deployment Guide:** Step-by-step rollout checklist
- **Incident Response:** "Trash deleted too early" playbook
- **Monitoring Guide:** Dashboard interpretation

---

## 13. Success Criteria

### 13.1 Launch Metrics (30 days post-rollout)
- ✅ 90%+ of deletes use trash (vs. accidental permanent deletes pre-feature)
- ✅ <5% restore rate (indicates users confident in deletion)
- ✅ Zero P0/P1 incidents related to trash
- ✅ API p95 latency <200ms

### 13.2 User Satisfaction
- ✅ NPS score >40 for trash feature
- ✅ <10 support tickets/month about trash confusion

### 13.3 Technical Health
- ✅ 99.9% uptime for trash endpoints
- ✅ Cleanup job success rate >99.5%
- ✅ Storage cost increase <2% (due to 7-day retention)

---

## 14. Future Enhancements

### 14.1 V2 Features (Potential Future Additions)
- ~~**Frontend UI Components:** Complete trash view page, context menus, restore actions~~ ✅ **IMPLEMENTED**
- ~~**Folder Trash Support:** Handle folder deletion/restore as a unit~~ ✅ **IMPLEMENTED in V1.3**
- **Configurable Retention:** Allow owners to set 1-30 day retention
- **Trash Size Limits:** Auto-purge oldest items if trash exceeds quota
- **Trash Analytics:** Dashboard showing trash usage trends
- **Global Trash View:** See all trashed files across all buckets
- **Bulk Operations:** Batch restore/purge multiple files/folders at once
- **Trash Search/Filter:** Search within trash, filter by date/size/user/type
- **Reconciliation Job:** Automated cleanup of orphaned database records (currently manual)

### 14.2 Technical Debt
- Implement storage provider health checks for trash operations
- Add retry logic for failed storage tag operations
- Add pagination to trash listing endpoint (currently returns all results)
- Implement error metrics for storage tagging failures
- Add integration tests for lifecycle policy + event handler workflow

---

## 15. Open Questions

1. **Q:** Should trash count against user storage quota?
   **A:** **DEFERRED** - Recommend YES to incentivize cleanup, but needs product decision. Currently trash does not count against quota.

2. **Q:** What happens if user loses `restore` permission after file trashed?
   **A:** **DECISION MADE** - Permissions are checked at restore time. If user lost restore permission, they cannot restore. Owner/admin can still restore via `ActionPurge` permission.

3. **Q:** Should trash be per-bucket or per-user view?
   **A:** **IMPLEMENTED** - Per-bucket (endpoint: `GET /api/buckets/{id}/trash`). Global trash view deferred to V2.

4. **Q:** How to handle file version conflicts during restore if versioning added later?
   **A:** **OUT OF SCOPE** - Defer to versioning PRD. Current implementation checks for naming conflicts and returns 409 error.

5. **Q:** Should folders be supported in trash?
   **A:** ✅ **IMPLEMENTED in V1.3** - Folders are now fully supported. Folders and all their contents can be trashed, restored, and purged. Recursive operations handle all descendants with timestamp-based restore logic to prevent unintended file restoration.

---

## 16. Approval & Sign-Off

| Stakeholder       | Role                | Status  | Date       |
|-------------------|---------------------|---------|------------|
| Engineering Lead  | Technical Review    | Pending | -          |
| Product Manager   | Requirements Review | Pending | -          |
| Security Team     | Security Review     | Pending | -          |
| DevOps Lead       | Infrastructure      | Pending | -          |
| UX Designer       | UI/UX Review        | Pending | -          |

---

## 17. Implementation Details

### 17.1 Actual Implementation Deviations

#### Database Model (`internal/models/file.go`)
**As Specified:**
```go
TrashedAt   *time.Time `gorm:"default:null;index" json:"trashed_at,omitempty"`
TrashedBy   *uuid.UUID `gorm:"type:uuid;default:null" json:"trashed_by,omitempty"`
TrashedUser User       `gorm:"foreignKey:TrashedBy" json:"-"`
```

**As Implemented:**
```go
TrashedAt   *time.Time `gorm:"default:null;index" json:"trashed_at,omitempty"`
TrashedBy   *uuid.UUID `gorm:"type:uuid;default:null" json:"trashed_by,omitempty"`
TrashedUser User       `gorm:"foreignKey:TrashedBy" json:"trashed_user,omitempty"`
```
**Reason:** Changed JSON field from `-` to `trashed_user,omitempty` to include user details in API responses.

#### API Response Format
**List Trash Endpoint:**
Returns `models.Page[models.File]` structure (consistent with other list endpoints), not raw array.

**Actual Response:**
```json
{
  "data": [
    {
      "id": "uuid",
      "name": "file.txt",
      "status": "trashed",
      "trashed_at": "2025-10-21T10:00:00Z",
      "trashed_by": "user-uuid",
      "trashed_user": {
        "id": "user-uuid",
        "email": "user@example.com"
      }
    }
  ]
}
```

#### Storage Implementation - MinIO
**As Specified in PRD:**
```go
s.storage.PutObjectTagging(context.Background(), &minio.PutObjectTaggingOptions{
    Bucket: s.BucketName,
    Object: path,
    Tags: map[string]string{...},
})
```

**As Implemented:**
```go
tags, err := tags.MapToObjectTags(tagMap)
if err != nil {
    return err
}
err = s.storage.PutObjectTagging(context.Background(), s.BucketName, path, tags, minio.PutObjectTaggingOptions{})
```
**Reason:** MinIO SDK requires `*tags.Tags` type, not raw map. Used `tags.MapToObjectTags()` helper.

### 17.2 Files Modified

#### Backend (Go) - 11 files
1. `internal/models/file.go` - File model with trash fields
2. `internal/storage/interfaces.go` - Storage interface updates
3. `internal/storage/aws.go` - AWS S3 tagging implementation
4. `internal/storage/gcp.go` - GCP metadata implementation
5. `internal/storage/s3.go` - MinIO tagging implementation
6. `internal/services/bucket.go` - Service methods and routing
7. `internal/rbac/const.go` - RBAC actions (pre-existing)
8. `internal/rbac/groups/contributor.go` - Contributor permissions
9. `internal/rbac/groups/owner.go` - Owner permissions
10. `internal/activity/constants.go` - Activity log messages
11. `internal/events/registry.go` - Event registration

#### New Files - 4 files
12. `internal/events/trash_expiration.go` - Trash expiration event handler
13. `migrations/001_add_trash_fields.sql` - Database migration
14. `migrations/001_add_trash_fields_rollback.sql` - Rollback script
15. `docs/IMPLEMENTATION-trash-feature.md` - Implementation guide

### 17.3 Quick Start Guide

#### 1. Deploy Database Changes
```bash
# Run migration
psql -U safebucket -d safebucket_db -f migrations/001_add_trash_fields.sql

# Verify
psql -U safebucket -d safebucket_db -c "\d files"
# Should show: trashed_at, trashed_by columns
```

#### 2. Deploy Backend
```bash
# Build
go build -o safebucket-api main.go

# Verify build
./safebucket-api --version

# Deploy (method depends on your infrastructure)
```

#### 3. Configure Cloud Lifecycle Policies

**AWS S3 (Terraform):**
```hcl
resource "aws_s3_bucket_lifecycle_configuration" "trash" {
  bucket = aws_s3_bucket.safebucket.id
  rule {
    id     = "trash-expiration"
    status = "Enabled"
    filter {
      tag { key = "Status"; value = "trashed" }
    }
    expiration { days = 7 }
  }
}
```

**GCP (Terraform):**
```hcl
resource "google_storage_bucket" "safebucket" {
  lifecycle_rule {
    condition {
      age = 7
      matches_metadata = { status = "trashed" }
    }
    action { type = "Delete" }
  }
}
```

**MinIO (CLI):**
```bash
mc ilm rule add --tags "Status=trashed" --expire-days 7 minio/safebucket
```

#### 4. Test the Feature
```bash
# 1. Upload a file
curl -X POST http://localhost:8080/api/buckets/{id}/files \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"test.txt", "path":"/", "type":"file", "size":100}'

# 2. Delete (trash) the file
curl -X DELETE http://localhost:8080/api/buckets/{id}/files/{fileId} \
  -H "Authorization: Bearer $TOKEN"

# 3. List trash
curl -X GET http://localhost:8080/api/buckets/{id}/trash \
  -H "Authorization: Bearer $TOKEN"

# 4. Restore the file
curl -X POST http://localhost:8080/api/buckets/{id}/files/{fileId}/restore \
  -H "Authorization: Bearer $TOKEN"
```

## 18. Appendix

### 18.1 References
- [AWS S3 Lifecycle Documentation](https://docs.aws.amazon.com/AmazonS3/latest/userguide/object-lifecycle-mgmt.html)
- [GCP Cloud Storage Lifecycle](https://cloud.google.com/storage/docs/lifecycle)
- [MinIO Object Lifecycle Management](https://min.io/docs/minio/linux/administration/object-management/object-lifecycle-management.html)
- [MinIO Object Tagging Support](https://github.com/minio/minio/pull/8880)
- [GORM Soft Delete](https://gorm.io/docs/delete.html#Soft-Delete)
- [SafeBucket CLAUDE.md](../CLAUDE.md)

### 18.2 Glossary
- **Hard Delete:** Permanent removal from database and storage
- **Soft Delete:** Marking record as deleted while retaining data
- **Trash:** Temporary storage for deleted files with recovery option
- **Purge:** Permanent deletion from trash (manual or automatic)
- **Restore:** Recover file from trash to original state

### 18.3 Frontend Implementation Details

#### Files Modified (10 files)

**1. API Integration (`web/src/components/bucket-view/helpers/api.ts`)**
- Added `api_listTrashedFiles` - Fetches trash with Page unwrapping
- Added `api_restoreFile` - POST restore endpoint
- Added `api_purgeFile` - DELETE purge endpoint

**2. Trash View Component (`web/src/components/bucket-view/components/BucketTrashView.tsx`)**
- Full DataTable implementation with trash-specific columns
- Columns: Name, Size, Deleted At, Deleted By, Status, Actions
- Empty state with trash icon
- Retention notice banner
- Three-dots menu with restore/purge actions

**3. Trash Actions Hook (`web/src/components/bucket-view/hooks/useTrashActions.ts`)**
- TanStack Query integration for data fetching
- Mutation hooks for restore and purge operations
- Query invalidation on success
- Toast notifications for user feedback

**4. Bucket View (`web/src/components/bucket-view/BucketView.tsx`)**
- Integrated BucketTrashView in view components map
- Passed trash actions to trash view

**5. View Options (`web/src/components/bucket-view/components/BucketViewOptions.tsx`)**
- Added query client for cache invalidation
- Auto-refresh trash data on view switch
- Auto-refresh activity data on view switch

**6. Activity Types (`web/src/types/activity.ts`)**
- Added `FILE_TRASHED` enum value
- Added `FILE_RESTORED` enum value
- Added `FILE_PURGED` enum value

**7. Activity Constants (`web/src/components/activity-view/helpers/constants.ts`)**
- Added icon mapping for FILE_TRASHED (orange trash icon)
- Added icon mapping for FILE_RESTORED (blue restore icon)
- Added icon mapping for FILE_PURGED (red delete icon)

**8. Localization - English (`web/src/locales/en.json`)**
- Trash view labels (already present)
- Activity messages for trash operations (already present)

**9. Localization - French (`web/src/locales/fr.json`)**
- Added French translations for trash activities
- `file_trashed`, `file_restored`, `file_purged` messages

**10. File Actions (`web/src/components/FileActions/FileActions.tsx`)**
- Added `trashMode` prop for context-aware menus
- Added `onRestore` and `onPermanentDelete` callbacks
- Conditional rendering: trash actions vs standard actions
- Download button disabled for trashed files

**11. DataTable Component (`web/src/components/common/components/DataTable/DataTable.tsx`)**
- Added trash mode props
- Pass-through of trash actions to FileActions

**12. Backend Service (`internal/services/bucket.go`)**
- Added download prevention for trashed files (403 error)

#### Key Features Implemented

**Context-Aware Menus**
- Right-click in trash view shows: Restore, Delete Permanently
- Right-click in normal view shows: Download, Share, New Folder, Delete
- Download option disabled if file status is "trashed"

**Data Refresh Strategy**
- Trash view automatically refreshes when user clicks trash icon
- Activity view refreshes when user clicks activity icon
- Mutations invalidate relevant query caches

**Error Handling**
- Backend returns 403 if attempting to download trashed file
- Frontend disables download button preventively
- Toast notifications for all error states

**Localization**
- Full English/French support
- Activity feed shows localized trash operation messages
- UI labels localized in both languages

### 18.4 Event-Driven Deletion Processing (V1.5)

#### Overview
V1.5 implements the critical missing piece for automatic trash cleanup: parsing and processing deletion events from cloud provider lifecycle policies. When S3, GCP, or MinIO automatically deletes objects after 7 days, the backend now receives these events and performs database cleanup.

#### Files Modified (6 files)

**1. `internal/messaging/types.go`**
Added `BucketDeletionEvent` struct:
```go
type BucketDeletionEvent struct {
    BucketId  string `json:"bucket_id"`
    ObjectKey string `json:"object_key"` // Full path: buckets/{id}/path/file.ext
    EventName string `json:"event_name"` // e.g., "s3:ObjectRemoved:Delete"
}
```

**2. `internal/messaging/interfaces.go`**
Extended `ISubscriber` interface:
```go
type ISubscriber interface {
    Subscribe() <-chan *message.Message
    Close() error
    ParseBucketUploadEvents(*message.Message) []BucketUploadEvent
    ParseBucketDeletionEvents(*message.Message) []BucketDeletionEvent  // NEW
}
```

**3. `internal/messaging/jetstream.go` (MinIO/NATS)**
Implemented `ParseBucketDeletionEvents`:
- Handles `s3:ObjectRemoved:*` events (manual deletion)
- Handles `s3:LifecycleExpiration:*` events (automatic cleanup)
- Extracts bucket ID from object key path using string manipulation
- Returns array of `BucketDeletionEvent` for processing

**4. `internal/messaging/aws.go` (AWS S3)**
Implemented `ParseBucketDeletionEvents`:
- Handles `ObjectRemoved:*` events
- Handles `LifecycleExpiration:*` events
- Safe string prefix checking to avoid out-of-bounds errors
- Manually parses object key path to extract bucket UUID

**5. `internal/messaging/gcp.go` (GCP Cloud Storage)**
Implemented `ParseBucketDeletionEvents`:
- Handles `OBJECT_DELETE` event type
- Checks both `objectId` and `name` metadata fields
- Extracts bucket ID from object key path
- GCP-specific metadata handling

**6. `internal/events/handler.go`**
Updated `HandleBucketEvents` function:
- Processes both upload and deletion events in same message loop
- Calls `subscriber.ParseBucketDeletionEvents(msg)` for each message
- Creates `TrashExpiration` events from deletion events
- Processes via `TrashExpiration.callback()` to hard delete from database
- Continues processing even if individual events fail

#### Event Flow

```
Cloud Lifecycle Policy Deletes Object (after 7 days)
    ↓
Event Notification → Message Queue (SQS/Pub/Sub/JetStream)
    ↓
HandleBucketEvents() receives message
    ↓
ParseBucketDeletionEvents() extracts deletion events
    ↓
NewTrashExpirationFromBucketEvent() creates TrashExpiration event
    ↓
TrashExpiration.callback() processes event
    ↓
Validates file is in trash status
    ↓
Checks expiration (safety: must be >7 days)
    ↓
Hard deletes file from database (Unscoped().Delete())
```

#### Event Types Supported

**AWS S3 / MinIO:**
- `s3:ObjectRemoved:Delete` - Manual deletion via API
- `s3:ObjectRemoved:DeleteMarkerCreated` - Versioned bucket deletion
- `s3:LifecycleExpiration:Delete` - Lifecycle policy deletion
- `s3:LifecycleExpiration:DeleteMarkerCreated` - Lifecycle versioned deletion

**GCP Cloud Storage:**
- `OBJECT_DELETE` - Object deletion events

#### Key Implementation Details

**Bucket ID Extraction:**
All implementations extract the bucket UUID from the object key path:
- Expected format: `buckets/{uuid}/path/to/file.ext`
- Splits path by `/` delimiter
- Validates bucket ID exists before processing
- Logs warning if extraction fails

**Error Handling:**
- Malformed events are logged and acknowledged (not requeued)
- Missing bucket IDs are logged as warnings
- Individual event failures don't stop message processing
- TrashExpiration callback errors are logged but continue processing

**Safety Checks:**
- Validates file is in `trashed` status before deletion
- Confirms file is actually expired (>7 days in trash)
- Handles missing files gracefully (might be already cleaned up)
- Uses `Unscoped().Delete()` for true hard deletion

**Performance:**
- Non-blocking: errors in one event don't affect others
- Efficient: processes all events in single message
- Scalable: handles batch deletion events from lifecycle policies

#### Testing Status

**Build Verification:** ✅ **PASSED**
- `go build` successful (101MB binary)
- `go vet` passed for messaging package
- `go vet` passed for events package
- No compilation errors or warnings

**Integration Testing:** ⏳ **PENDING**
- Requires infrastructure setup (lifecycle policies + event notifications)
- See Phase 3 (Section 11.3) for configuration steps
- Testing with synthetic events recommended before production

### 18.5 Folder Trash Implementation (V1.3 → V1.4)

#### Backend Methods

**1. TrashFolder** (`internal/services/bucket.go:726-768`) - **✨ Now Async (V1.4)**
```go
func (s BucketService) TrashFolder(logger *zap.Logger, user models.UserClaims, ids uuid.UUIDs) error
```
- **Synchronous part**: Sets folder status to `trashed` immediately, returns HTTP 200
- **Asynchronous part**: `FolderTrash` event processes children in batches
- All items marked with same `trashed_at` timestamp
- Tags all objects in storage for lifecycle cleanup
- Batch processing prevents HTTP timeouts
- Events requeue if more children remain
- Logs `FOLDER_TRASHED` activity when complete

**2. RestoreFolder** (`internal/services/bucket.go:770-819`) - **✨ Now Async (V1.4)**
```go
func (s BucketService) RestoreFolder(logger *zap.Logger, user models.UserClaims, ids uuid.UUIDs) error
```
- **Synchronous part**: Validates folder, checks conflicts, sets status to `restoring`, returns HTTP 200
- **Asynchronous part**: `FolderRestore` event processes restoration in batches
- Uses timestamp matching to prevent restoring individually trashed files
- Checks for naming conflicts before starting
- Validates 7-day expiration
- Removes trash tags from storage
- Logs `FOLDER_RESTORED` activity when complete

**3. PurgeFolder** (`internal/services/bucket.go:822-849`) - **✨ Now Async (V1.4)**
```go
func (s BucketService) PurgeFolder(logger *zap.Logger, user models.UserClaims, ids uuid.UUIDs) error
```
- **Synchronous part**: Validates folder is in trash, triggers event, returns HTTP 200
- **Asynchronous part**: `FolderPurge` event processes deletion in batches
- Permanently deletes folder and all contents
- Hard deletes from database and storage
- Batch operations for efficiency
- Only accessible to owners (ActionPurge)
- Logs `FOLDER_PURGED` activity when complete

#### Modified Files (Folders Implementation)
**Backend:**
- `internal/services/bucket.go` - Added 3 new methods (TrashFolder, RestoreFolder, PurgeFolder) - **Updated to async in V1.4**
- `internal/activity/constants.go` - Added FolderTrashed, FolderRestored, FolderPurged
- `internal/events/folder_restore.go` - **NEW in V1.4** - Async restore event handler
- `internal/events/folder_trash.go` - **NEW in V1.4** - Async trash event handler
- `internal/events/folder_purge.go` - **NEW in V1.4** - Async purge event handler
- `internal/events/registry.go` - **Updated in V1.4** - Registered new event types

**Frontend:**
- `web/src/types/activity.ts` - Added folder activity enum values
- `web/src/components/activity-view/helpers/constants.ts` - Added folder icon mappings
- `web/src/locales/en.json` - Added folder activity messages (English)
- `web/src/locales/fr.json` - Added folder activity messages (French)

#### Key Design Decisions

**Asynchronous Processing (V1.4):**
- All folder operations return HTTP 200 immediately
- Background event system processes children in batches
- Batch size controlled by `c.BulkActionsLimit`
- Events automatically requeue if more items remain
- Prevents HTTP timeouts on large folders (thousands of files)
- Similar pattern to existing `ObjectDeletion` event

**Timestamp-Based Restore:**
- Only restores children with matching `trashed_at` timestamp
- Prevents restoring files individually trashed after folder deletion
- Example: Trash folder at 10:00, restore it, trash individual file at 11:00, restore folder again → file stays in trash

**FileStatusRestoring (V1.4):**
- New transient status for folders being restored asynchronously
- Prevents concurrent operations on the same folder
- Reverts to `trashed` status if restore fails (e.g., expiration, conflict)

**Recursive Operations:**
- Uses SQL `LIKE` pattern matching on file paths
- Batch processing in transactions
- Batch storage operations for efficiency

**Storage Tagging:**
- Best-effort tagging (failures logged, not blocking)
- Database is source of truth
- Enables cloud lifecycle policies

### 18.6 Change Log
| Version | Date       | Author  | Changes              |
|---------|------------|---------|----------------------|
| 1.0     | 2025-10-20 | Claude  | Initial draft        |
| 1.1     | 2025-10-21 | Claude  | Backend implementation complete - Added implementation status, actual deviations, quick start guide |
| 1.2     | 2025-10-22 | Claude  | Frontend implementation complete - Added UI components, activity feed integration, context menus, localization, download restrictions |
| 1.3     | 2025-10-22 | Claude  | Folder support implementation - Added TrashFolder, RestoreFolder, PurgeFolder methods, folder activity logging, frontend support, localization |
| 1.4     | 2025-10-22 | Claude  | Async folder operations - Converted all folder operations to asynchronous processing with event handlers (FolderRestore, FolderTrash, FolderPurge), prevents HTTP timeouts on large folders, batch processing with automatic requeuing |
| 1.5     | 2025-10-22 | Claude  | Event-driven deletion processing - Implemented ParseBucketDeletionEvents() for all messaging providers (AWS SQS, GCP Pub/Sub, NATS/JetStream), updated HandleBucketEvents to process deletion events, added BucketDeletionEvent type, extended ISubscriber interface. Phase 4 (Event Handler Deployment) now complete. |
| 1.6     | 2025-10-22 | Claude  | **Soft delete fix for activity enrichment** - Changed all hard deletes (`Unscoped().Delete()`) to soft deletes (`Delete()`) in PurgeFile, FolderPurge, and TrashExpiration handlers. Storage objects are still hard deleted, but database records are now soft deleted. This allows activity enrichment to continue working by finding deleted records using `Unscoped()` queries. Fixes issue where activity enrichment failed after files were permanently removed from database. |
| 1.7     | 2025-10-22 | Claude  | **React Query migration** - Migrated trash feature to TanStack Query (React Query) for consistent data fetching patterns. Added `bucketTrashedFilesQueryOptions` to centralized query options in `web/src/queries/bucket.ts`. Updated `useTrashActions` hook to use centralized query options with proper cache invalidation. Added i18n support for success/error toast messages in English and French. Removed deprecated `api_listTrashedFiles` function. All trash data fetching now follows established codebase patterns with automatic caching and background refetching. |
| 1.8     | 2025-10-23 | Claude  | **Critical bug fixes for NULL status handling and event routing** - Fixed three critical bugs: (1) GetBucket query excluding trashed items with NULL-aware filter (`status IS NULL OR status != 'trashed'`), (2) Event router missing FolderRestore and TrashExpiration event mappings causing fatal errors, (3) FolderRestore conflict check excluding normal folders with NULL status. All fixes handle SQL NULL semantics correctly for folder status field. Files modified: `internal/services/bucket.go:211`, `internal/core/event_router.go:53`, `internal/events/folder_restore.go:107`. Folder restore with child file restoration now fully functional. |

---

**END OF DOCUMENT**