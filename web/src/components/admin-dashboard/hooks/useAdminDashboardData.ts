import { useQuery } from "@tanstack/react-query";
import { adminStatsQueryOptions } from "@/queries/admin";

export const useAdminDashboardData = (days: number = 90) => {
  const { data: stats, isLoading } = useQuery(adminStatsQueryOptions(days));

  return {
    stats,
    isLoading,
  };
};
