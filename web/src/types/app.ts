export enum EnvironmentType {
  development = "development",
  production = "production",
}

export interface IConfig {
  apiUrl: string;
  environment: EnvironmentType;
}
