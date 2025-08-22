import { queryOptions } from "@tanstack/react-query";
import type { IProvidersResponse } from "@/types/auth_providers.ts";
import { api } from "@/lib/api.ts";

export const authProvidersQueryOptions = () =>
  queryOptions({
    queryKey: ["auth", "providers"],
    queryFn: () => api.get<IProvidersResponse>("/auth/providers"),
    select: (data) => data.data,
  });
