export interface IProvider {
  id: string,
  name: string,
}

export interface IProvidersData {
  providers: IProvider[];
  error: string;
  isLoading: boolean;
}

export type IProvidersResponse = {
  data: IProvider[];
};
