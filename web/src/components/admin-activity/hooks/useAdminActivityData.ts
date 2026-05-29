import { useQuery } from "@tanstack/react-query";
import type { AdminActivityFilters } from "@/queries/admin";
import { adminActivityQueryOptions } from "@/queries/admin";

export const useAdminActivityData = (filters: AdminActivityFilters = {}) => {
  const {
    data: activities,
    isLoading,
    isFetching,
    refetch,
  } = useQuery(adminActivityQueryOptions(filters));

  return {
    activities: activities ?? [],
    isLoading,
    isFetching,
    refetch,
  };
};
