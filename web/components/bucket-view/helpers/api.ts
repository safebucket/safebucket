import { api } from "@/lib/api";

import { IBucket, IBucketForm } from "@/components/bucket-view/helpers/types";

export const api_createBucket = (body: IBucketForm) =>
  api.post<IBucket>("/buckets", body);
