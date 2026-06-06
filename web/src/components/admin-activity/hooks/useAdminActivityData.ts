import { useInfiniteQuery } from "@tanstack/react-query";
import type { AdminActivityFilters } from "@/queries/admin";
import { adminActivityInfiniteQueryOptions } from "@/queries/admin";

export const useAdminActivityData = (filters: AdminActivityFilters = {}) => {
  const {
    data,
    isLoading,
    isFetching,
    refetch,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useInfiniteQuery(adminActivityInfiniteQueryOptions(filters));

  return {
    activities: data?.pages.flatMap((page) => page.data) ?? [],
    isLoading,
    isFetching,
    refetch,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  };
};
