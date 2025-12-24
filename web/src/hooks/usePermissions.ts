/**
 * Permission hooks for React components
 *
 * This file provides React hooks that integrate the permission system
 * with TanStack Query for data fetching and React useMemo for performance.
 */

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import type {
  Action,
  BucketGroup,
  PermissionCheck,
  Resource,
} from "@/types/permissions";
import type { IBucketMember } from "@/components/bucket-view/helpers/types";
import { useSession } from "@/hooks/useAuth";
import { bucketMembersQueryOptions } from "@/queries/bucket";
import { canPerformBucketAction, getBucketGroupRank } from "@/lib/permissions";

/**
 * Get current user's bucket-level permissions
 *
 * @param bucketId - ID of the bucket to check permissions for
 * @returns Permission helpers and user's bucket group
 *
 * @example
 * const { isOwner, isContributor, can } = useBucketPermissions(bucketId);
 *
 * if (isOwner) {
 *   // Show owner-only UI
 * }
 *
 * const canDelete = can(Action.Delete, Resource.File);
 * if (!canDelete.allowed) {
 *   console.log(canDelete.reason); // "Requires contributor permission"
 * }
 */
export function useBucketPermissions(bucketId: string | undefined) {
  const session = useSession();

  const { data: members, isLoading } = useQuery({
    ...bucketMembersQueryOptions(bucketId!),
    enabled: !!bucketId && !!session?.email,
  });

  const membership = useMemo<IBucketMember | undefined>(() => {
    if (!session?.email || !members) return undefined;
    const member = members.find((m) => m.email === session.email);
    return member?.status === "active" ? member : undefined;
  }, [members, session?.email]);

  const userGroup = membership?.group as BucketGroup | undefined;

  const can = useMemo(
    () =>
      (action: Action, resource: Resource): PermissionCheck => {
        // If loading, deny but don't show reason (prevents flickering)
        if (isLoading) {
          return { allowed: false };
        }

        if (session?.role === "admin") {
          return { allowed: true };
        }

        return canPerformBucketAction(action, resource, userGroup);
      },
    [userGroup, session?.role, isLoading],
  );

  const isOwner = userGroup === "owner";
  const isContributor = userGroup === "contributor" || isOwner;
  const isViewer = !!userGroup;
  const isMember = !!membership;

  return {
    userGroup,
    membership,
    isLoading,
    isMember,
    isOwner,
    isContributor,
    isViewer,
    can,
    groupRank: userGroup ? getBucketGroupRank(userGroup) : 0,
  };
}
