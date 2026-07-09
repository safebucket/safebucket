import { useQuery } from "@tanstack/react-query";
import { adminSettingsQueryOptions } from "@/queries/admin";

export function useAdminSettings() {
  return useQuery(adminSettingsQueryOptions());
}
