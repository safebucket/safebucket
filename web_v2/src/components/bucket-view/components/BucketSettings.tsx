import { BucketDeletion } from "./BucketDeletion";
import { BucketInformation } from "./BucketInformation";
import type { FC } from "react";

import type { IBucket } from "@/components/bucket-view/helpers/types";
import { BucketMembers } from "@/components/bucket-members/BucketMembers.tsx";

interface IBucketSettingsProps {
  bucket: IBucket;
}

export const BucketSettings: FC<IBucketSettingsProps> = ({ bucket }) => {
  return (
    <div className="mx-auto">
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-5">
        <div className="space-y-6 lg:col-span-2">
          <BucketInformation bucket={bucket} />
          <BucketDeletion bucket={bucket} />
        </div>

        <div className="lg:col-span-3">
          <BucketMembers bucket={bucket} />
        </div>
      </div>
    </div>
  );
};
