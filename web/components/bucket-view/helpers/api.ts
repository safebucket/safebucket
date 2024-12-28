import { api } from "@/lib/api";

import { IBucket, IShareWith } from "@/components/bucket-view/helpers/types";

export const api_createBucket = (name: string, share_with: IShareWith[]) =>
  api.post<IBucket>("/buckets", { name, share_with });
