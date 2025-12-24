/**
 * Permission system type definitions
 *
 * This file defines the core types for the permission system, including:
 * - Action and Resource enums (matching backend RBAC)
 * - Platform roles and bucket groups
 * - Permission check result types
 */

/**
 * Actions that can be performed on resources
 * Matches backend: internal/rbac/const.go
 */
export enum Action {
  Create = "create",
  Delete = "delete",
  Download = "download",
  Restore = "restore",
  Update = "update",
  Grant = "grant", // For member management
  Purge = "purge", // For permanent deletion
}

/**
 * Resources that actions can be performed on
 * Matches backend: internal/rbac/const.go
 */
export enum Resource {
  Bucket = "bucket",
  File = "file",
  Folder = "folder",
  Member = "member",
}

/**
 * Platform-level roles (global permissions)
 * From Session type
 */
export type PlatformRole = "admin" | "user" | "guest";

/**
 * Bucket-level groups (per-bucket permissions)
 * From IBucketMember type
 */
export type BucketGroup = "owner" | "contributor" | "viewer";

/**
 * Result of a permission check
 */
export interface PermissionCheck {
  allowed: boolean;
  reason?: string; // Human-readable reason for denial (for tooltips)
}
