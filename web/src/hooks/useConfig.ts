import { useSuspenseQuery } from "@tanstack/react-query";
import type { IConfig } from "@/types/app.ts";
import { configQueryOptions, defaultConfig } from "@/queries/config.ts";
import { router } from "@/main.tsx";

export function useConfig() {
  const { data } = useSuspenseQuery(configQueryOptions());
  return data;
}

export function getConfigSync(): IConfig {
  const queryClient = router.options.context.queryClient;
  const config = queryClient.getQueryData<IConfig>(
    configQueryOptions().queryKey,
  );

  return config ?? defaultConfig;
}

export function getApiUrl(): string {
  const config = getConfigSync();
  return `${config.apiUrl}/api/v1`;
}
