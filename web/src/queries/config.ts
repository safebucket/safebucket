import { queryOptions } from "@tanstack/react-query";
import type { IConfig } from "@/types/app.ts";
import { EnvironmentType } from "@/types/app.ts";

export const configQueryOptions = () =>
  queryOptions({
    queryKey: ["config"],
    queryFn: async (): Promise<IConfig> => {
      const defaultConfig = {
        apiUrl: "http://localhost:8080",
        environment: EnvironmentType.production,
      };

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
