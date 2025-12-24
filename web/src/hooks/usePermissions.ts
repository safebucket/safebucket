/**
 * Permission hooks for React components
 *
 * This file provides React hooks that integrate the permission system
 * with TanStack Query for data fetching and React useMemo for performance.
 */

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import type { BucketGroup } from "@/types/bucket";
import type { IBucketMember } from "@/components/bucket-view/helpers/types";
import { useSession } from "@/hooks/useAuth";
import { bucketMembersQueryOptions } from "@/queries/bucket";

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

  const isOwner = userGroup === "owner";
  const isContributor = userGroup === "contributor" || isOwner;
  const isViewer = !!userGroup;

  return {
    isLoading,
    isOwner,
    isContributor,
    isViewer,
  };
}
