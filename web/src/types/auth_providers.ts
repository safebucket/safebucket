export interface IProvider {
  id: string;
  name: string;
}

export type IProvidersResponse = {
  data: Array<IProvider>;
};
