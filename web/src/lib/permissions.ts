/**
 * Permission utility functions
 *
 * This file contains pure functions for permission checking logic.
 * Mirrors backend RBAC system: internal/rbac/group_checker.go
 */

import type {
  BucketGroup,
  PermissionCheck,
  PlatformRole,
} from "@/types/permissions";
import { Action, Resource } from "@/types/permissions";

/**
 * Get hierarchical rank of platform role
 * Higher rank = more permissions
 *
 * @param role - Platform role to get rank for
 * @returns Numeric rank (admin=3, user=2, guest=1)
 */
export function getPlatformRoleRank(role: PlatformRole): number {
  switch (role) {
    case "admin":
      return 3;
    case "user":
      return 2;
    case "guest":
      return 1;
    default:
      return 0;
  }
}

/**
 * Get hierarchical rank of bucket group
 * Mirrors backend: internal/rbac/group_checker.go GetGroupRank()
 * Higher rank = more permissions
 *
 * @param group - Bucket group to get rank for
 * @returns Numeric rank (owner=3, contributor=2, viewer=1)
 */
export function getBucketGroupRank(group: BucketGroup): number {
  switch (group) {
    case "owner":
      return 3;
    case "contributor":
      return 2;
    case "viewer":
      return 1;
    default:
      return 0;
  }
}

/**
 * Check if user's platform role meets or exceeds required role
 *
 * @param userRole - User's current platform role
 * @param requiredRole - Minimum required platform role
 * @returns True if user role rank >= required role rank
 *
 * @example
 * hasPlatformRole("admin", "user") // true (admin >= user)
 * hasPlatformRole("guest", "user") // false (guest < user)
 */
export function hasPlatformRole(
  userRole: PlatformRole,
  requiredRole: PlatformRole,
): boolean {
  return getPlatformRoleRank(userRole) >= getPlatformRoleRank(requiredRole);
}

/**
 * Check if user's bucket group meets or exceeds required group
 * Mirrors backend: internal/rbac/group_checker.go HasGroup()
 *
 * @param userGroup - User's current bucket group
 * @param requiredGroup - Minimum required bucket group
 * @returns True if user group rank >= required group rank
 *
 * @example
 * hasBucketGroup("owner", "contributor") // true (owner >= contributor)
 * hasBucketGroup("viewer", "contributor") // false (viewer < contributor)
 */
export function hasBucketGroup(
  userGroup: BucketGroup,
  requiredGroup: BucketGroup,
): boolean {
  return getBucketGroupRank(userGroup) >= getBucketGroupRank(requiredGroup);
}

/**
 * Permission matrix for bucket-level actions
 * Maps (action, resource) -> required bucket group
 *
 * null = action not valid for this resource
 * string = minimum bucket group required
 */
const BUCKET_PERMISSION_MATRIX: Record<string, BucketGroup | null> = {
  // Owner-only actions
  "update:bucket": "owner",
  "delete:bucket": "owner",
  "grant:member": "owner",

  // Contributor+ actions
  "create:file": "contributor",
  "create:folder": "contributor",
  "delete:file": "contributor",
  "delete:folder": "contributor",
  "restore:file": "contributor",
  "restore:folder": "contributor",
  "purge:file": "contributor",
  "purge:folder": "contributor",
  "update:file": "contributor",
  "update:folder": "contributor",

  // Viewer+ actions (all authenticated bucket members)
  "download:file": "viewer",
};

/**
 * Check if user can perform action on resource in bucket
 *
 * @param action - Action to perform
 * @param resource - Resource to act on
 * @param userGroup - User's bucket group (undefined if not a member)
 * @returns Permission check result with allowed boolean and optional reason
 *
 * @example
 * canPerformBucketAction(Action.Delete, Resource.Bucket, "owner")
 * // { allowed: true }
 *
 * canPerformBucketAction(Action.Delete, Resource.Bucket, "viewer")
 * // { allowed: false, reason: "Requires owner permission" }
 */
export function canPerformBucketAction(
  action: Action,
  resource: Resource,
  userGroup: BucketGroup | undefined,
): PermissionCheck {
  const key = `${action}:${resource}`;
  const requiredGroup = BUCKET_PERMISSION_MATRIX[key];

  // No group requirement = not a valid bucket action
  if (requiredGroup === null) {
    return {
      allowed: false,
      reason: "Invalid action for this resource",
    };
  }

  // No user group = no membership
  if (!userGroup) {
    return {
      allowed: false,
      reason: "You are not a member of this bucket",
    };
  }

  // Check if user's group meets requirement
  const allowed = hasBucketGroup(userGroup, requiredGroup);

  return {
    allowed,
    reason: allowed ? undefined : `Requires ${requiredGroup} permission`,
  };
}

/**
 * Platform-level permission checks
 *
 * @param action - Action to perform
 * @param resource - Resource to act on
 * @param userRole - User's platform role
 * @returns Permission check result
 *
 * @example
 * canPerformPlatformAction(Action.Create, Resource.Bucket, "user")
 * // { allowed: true }
 *
 * canPerformPlatformAction(Action.Create, Resource.Bucket, "guest")
 * // { allowed: false, reason: "Guest users cannot create buckets" }
 */
export function canPerformPlatformAction(
  action: Action,
  resource: Resource,
  userRole: PlatformRole,
): PermissionCheck {
  // Only create bucket has platform-level restriction
  if (action === Action.Create && resource === Resource.Bucket) {
    const allowed = hasPlatformRole(userRole, "user");
    return {
      allowed,
      reason: allowed ? undefined : "Guest users cannot create buckets",
    };
  }

  // Admin override for all actions
  if (userRole === "admin") {
    return { allowed: true };
  }

  // Default: allow (backend will enforce actual authorization)
  return { allowed: true };
}
