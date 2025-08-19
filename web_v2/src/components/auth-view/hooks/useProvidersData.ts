import { fetchApi } from "@/lib/api";
import useSWR from "swr";

import type {
  IProvidersData,
  IProvidersResponse,
} from "@/components/auth-view/types/providers";

export const useProvidersData = (): IProvidersData => {
  const { data, error, isLoading } = useSWR(
    "/auth/providers",
    fetchApi<IProvidersResponse>,
  );

  return {
    providers: data ? data.data : [],
    error,
    isLoading,
  };
};
