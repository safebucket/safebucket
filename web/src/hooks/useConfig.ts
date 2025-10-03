import { useSuspenseQuery } from "@tanstack/react-query";
import type { IConfig } from "@/types/app.ts";
import { EnvironmentType } from "@/types/app.ts";
import { configQueryOptions } from "@/queries/config.ts";
import { router } from "@/main.tsx";

const defaultConfig: IConfig = {
  apiUrl: "http://localhost:8080",
  environment: EnvironmentType.production,
};

/**
 * Custom hook to access the application config
 * Uses React Query for caching and automatic refetching
 * This hook suspends while loading (use within a Suspense boundary)
 */
export function useConfig() {
  const { data } = useSuspenseQuery(configQueryOptions(defaultConfig));
  return data;
}

/**
 * Synchronously get the cached config from React Query
 * Returns default config if not yet loaded (during initial render)
 */
export function getConfigSync(): IConfig {
  const queryClient = router.options.context.queryClient;
  const config = queryClient.getQueryData<IConfig>(
    configQueryOptions(defaultConfig).queryKey,
  );

  return config ?? defaultConfig;
}

/**
 * Get the API URL from the cached config
 * @returns The full API URL with /api/v1 path
 */
export function getApiUrl(): string {
  const config = getConfigSync();
  return `${config.apiUrl}/api/v1`;
}
