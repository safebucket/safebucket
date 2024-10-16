import { api } from "@/lib/api";

import { IBucketForm } from "@/components/bucket-view/helpers/types";

export const api_createBucket = (body: IBucketForm) =>
  api.post("/buckets", body);
