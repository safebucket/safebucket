import React, { FC } from "react";

import { AddMembersInput } from "@/components/add-members/components/AddMembersInput";
import { AddMembersSkeleton } from "@/components/add-members/components/AddMembersSkeleton";
import { PeopleWithAccess } from "@/components/add-members/components/PeopleWithAccess";
import { useAddMembers } from "@/components/add-members/hooks/useAddMembers";
import { IInvites } from "@/components/bucket-view/helpers/types";
import { useBucketMembersData } from "@/components/bucket-view/hooks/useBucketMembersData";
import { Separator } from "@/components/ui/separator";

interface IAddMembersProps {
  shareWith: IInvites[];
  onShareWithChange: (shareWith: IInvites[]) => void;
  currentUserEmail?: string;
  currentUserName?: string;
  showCurrentUser?: boolean;
  bucketId?: string;
}

export const AddMembers: FC<IAddMembersProps> = ({
  shareWith,
  onShareWithChange,
  currentUserEmail,
  currentUserName,
  showCurrentUser = true,
  bucketId,
}) => {
  const { addEmail, setGroup, removeFromList } = useAddMembers(
    shareWith,
    onShareWithChange,
  );

  const { members, isLoading } = useBucketMembersData(
    bucketId || null,
  );

  return (
    <>
      <AddMembersInput onAddEmail={addEmail} />

      <Separator className="my-4" />

      {isLoading ? (
        <AddMembersSkeleton />
      ) : (
        <PeopleWithAccess
          shareWith={shareWith}
          onGroupChange={setGroup}
          onRemoveUser={removeFromList}
          currentUserEmail={currentUserEmail}
          currentUserName={currentUserName}
          showCurrentUser={showCurrentUser}
          existingMembers={members || []}
        />
      )}
    </>
  );
};
