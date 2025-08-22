import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import type { FC } from "react";

import type { IMembers } from "@/components/bucket-view/helpers/types";
import { AddMembersInput } from "@/components/add-members/components/AddMembersInput";
import { AddMembersSkeleton } from "@/components/add-members/components/AddMembersSkeleton";
import { PeopleWithAccess } from "@/components/add-members/components/PeopleWithAccess";
import { useAddMembers } from "@/components/add-members/hooks/useAddMembers";
import { Separator } from "@/components/ui/separator";
import { bucketMembersQueryOptions } from "@/queries/bucket.ts";

interface IAddMembersProps {
  shareWith: Array<IMembers>;
  onShareWithChange: (shareWith: Array<IMembers>) => void;
  currentUserEmail?: string;
  currentUserName?: string;
  showCurrentUser?: boolean;
  bucketId?: string;
  onAllMembersChange?: (allMembers: Array<IMembers>) => void;
}

export const AddMembers: FC<IAddMembersProps> = ({
  shareWith,
  onShareWithChange,
  currentUserEmail,
  currentUserName,
  showCurrentUser = true,
  bucketId,
  onAllMembersChange,
}) => {
  const { addEmail, setGroup, removeFromList } = useAddMembers(
    shareWith,
    onShareWithChange,
  );

  const { data: members, isLoading } = useQuery({
    ...bucketMembersQueryOptions(bucketId!),
    enabled: !!bucketId,
  });

  const [existingMemberChanges, setExistingMemberChanges] = useState<
    Record<string, string>
  >({});

  useEffect(() => {
    if (!members || !onAllMembersChange) return;

    const existingMembersAsInvites: Array<IMembers> = members
      .filter((member) => member.email !== currentUserEmail)
      .map((member) => ({
        email: member.email,
        group: existingMemberChanges[member.email] || member.group,
      }));

    const allMembers = [...existingMembersAsInvites, ...shareWith];
    onAllMembersChange(allMembers);
  }, [
    members,
    shareWith,
    existingMemberChanges,
    currentUserEmail,
    onAllMembersChange,
  ]);

  const handleExistingMemberGroupChange = (email: string, groupId: string) => {
    setExistingMemberChanges((prev) => ({
      ...prev,
      [email]: groupId,
    }));
  };

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
          existingMembers={members}
          onExistingMemberGroupChange={handleExistingMemberGroupChange}
        />
      )}
    </>
  );
};
