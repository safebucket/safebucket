export enum ProviderType {
  LOCAL = "local",
  OIDC = "oidc",
}

export interface IProvider {
  id: string;
  name: string;
  type: ProviderType;
}

export type IProvidersResponse = {
  data: Array<IProvider>;
};
