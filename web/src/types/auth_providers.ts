export enum ProviderType {
  LOCAL = "local",
  OIDC = "oidc",
  LDAP = "ldap",
}

export interface IProvider {
  id: string;
  name: string;
  type: ProviderType;
  domains: Array<string>;
}

export type IProvidersResponse = {
  data: Array<IProvider>;
};
