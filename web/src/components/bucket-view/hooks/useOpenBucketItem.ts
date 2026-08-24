import { useNavigate, useParams } from "@tanstack/react-router";

import type { BucketItem } from "@/types/bucket.ts";
import { isFolder } from "@/components/bucket-view/helpers/utils";

export const useOpenBucketItem = () => {
  const { bucketId, folderId } = useParams({
    from: "/_authenticated/buckets/$bucketId/files/{-$folderId}",
  });
  const navigate = useNavigate();

  return (item: BucketItem) => {
    if (isFolder(item)) {
      navigate({
        to: "/buckets/$bucketId/files/{-$folderId}",
        params: { bucketId, folderId: item.id },
      });
      return;
    }
    navigate({
      to: "/buckets/$bucketId/files/{-$folderId}",
      params: { bucketId, folderId },
      search: { preview: item.id },
    });
  };
};
