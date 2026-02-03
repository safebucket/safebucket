import { queryOptions } from "@tanstack/react-query";
import type { IConfig } from "@/types/app.ts";
import { EnvironmentType } from "@/types/app.ts";

export const defaultConfig: IConfig = {
  apiUrl: "http://localhost:8080",
  environment: EnvironmentType.production,
  requiresUploadConfirmation: false,
};

export const configQueryOptions = () =>
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
