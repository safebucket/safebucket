import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import type { FC } from "react";

import type { IMembers } from "@/components/bucket-view/helpers/types";
import { AddMembersInput } from "@/components/app-sidebar/components/AddMembersInput";
import { AddMembersList } from "@/components/app-sidebar/components/AddMembersList";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { bucketMembersQueryOptions } from "@/queries/bucket.ts";
import { useAddMembers } from "@/components/app-sidebar/hooks/useAddMembers.ts";

interface IAddMembersCardProps {
  shareWith: Array<IMembers>;
  onShareWithChange: (shareWith: Array<IMembers>) => void;
  currentUserEmail?: string;
  currentUserName?: string;
  showCurrentUser?: boolean;
  bucketId?: string;
  onAllMembersChange?: (allMembers: Array<IMembers>) => void;
}

export const AddMembersCard: FC<IAddMembersCardProps> = ({
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
        <div className="space-y-3">
          <Skeleton className="h-4 w-32" />
          <Skeleton className="h-20 w-full" />
          <Skeleton className="h-20 w-full" />
        </div>
      ) : (
        <AddMembersList
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
