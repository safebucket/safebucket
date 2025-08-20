interface Config {
  apiUrl: string;
  environment: string;
}

let config: Config | null = null;

export async function loadConfig(): Promise<Config> {
  if (config) return config;

  try {
    const response = await fetch("/config.json");
    if (!response.ok) {
      throw new Error(`Failed to load config: ${response.status}`);
    }
    config = await response.json();

    if (!config) throw new Error(`Failed to load config: ${response.status}`);

    return config;
  } catch (error) {
    console.warn("Failed to load config, using defaults:", error);
    config = {
      apiUrl: window.location.origin,
      environment: "development",
    };
    return config;
  }
}

export async function getApiUrl(): Promise<string> {
  const cfg = await loadConfig();
  return `${cfg.apiUrl}/api/v1`;
}
