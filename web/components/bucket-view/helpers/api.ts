import { api } from "@/lib/api";

import { IBucket } from "@/components/bucket-view/helpers/types";

export const api_createBucket = (name: string) =>
  api.post<IBucket>("/buckets", { name });
