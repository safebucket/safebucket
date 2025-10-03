import { queryOptions } from "@tanstack/react-query";
import type { IConfig } from "@/types/app.ts";

export const configQueryOptions = (defaultConfig: IConfig) =>
  queryOptions({
    queryKey: ["config"],
    queryFn: async (): Promise<IConfig> => {
      try {
        const response = await fetch("/config.json");
        if (!response.ok) {
          return defaultConfig;
        }
        return await response.json();
      } catch (error) {
        return defaultConfig;
      }
    },
    staleTime: Infinity,
  });
