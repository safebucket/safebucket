import React, { FC } from "react";

import { UserPlus } from "lucide-react";

import { AddMembers } from "@/components/add-members/components/AddMembers";
import { IBucket } from "@/components/bucket-view/helpers/types";
import { useBucketMembers } from "@/components/bucket-view/hooks/useBucketMembers";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

interface IBucketMembersProps {
  bucket: IBucket;
}

export const BucketMembers: FC<IBucketMembersProps> = ({ bucket }) => {
  const {
    shareWith,
    setShareWith,
    setAllMembers,
    hasChanges,
    handleUpdateMembers,
    currentUserEmail,
    currentUserName,
  } = useBucketMembers(bucket);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <UserPlus className="h-5 w-5" />
          Bucket Members
        </CardTitle>
        <CardDescription>
          Manage who has access to this bucket and their permissions
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <AddMembers
          shareWith={shareWith}
          onShareWithChange={setShareWith}
          currentUserEmail={currentUserEmail}
          currentUserName={currentUserName}
          bucketId={bucket.id}
          onAllMembersChange={setAllMembers}
          showCurrentUser={true}
        />

        {hasChanges && (
          <div className="flex justify-end border-t pt-4">
            <Button onClick={handleUpdateMembers}>Update Members</Button>
          </div>
        )}
      </CardContent>
    </Card>
  );
};
