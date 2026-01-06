import { useQuery } from "@tanstack/react-query";
import { adminActivityQueryOptions } from "@/queries/admin";

export const useAdminActivityData = () => {
  const {
    data: activities,
    isLoading,
    isFetching,
    refetch,
  } = useQuery(adminActivityQueryOptions());

  return {
    activities: activities ?? [],
    isLoading,
    isFetching,
    refetch,
  };
};
