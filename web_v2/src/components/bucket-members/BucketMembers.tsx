import { UserPlus, Users } from "lucide-react";
import type { FC } from "react";

import type { IBucket } from "@/components/bucket-view/helpers/types";
import {
  EMAIL_REGEX,
  bucketGroups,
} from "@/components/add-members/helpers/constants";
import { BucketMember } from "@/components/bucket-members/components/BucketMember";
import { BucketMembersSkeleton } from "@/components/bucket-members/components/BucketMembersSkeleton";
import { useBucketMembersData } from "@/components/bucket-members/hooks/useBucketMembersData";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

interface IBucketMembersProps {
  bucket: IBucket;
}

export const BucketMembers: FC<IBucketMembersProps> = ({ bucket }) => {
  const {
    isLoading,
    membersState,
    newMemberEmail,
    setNewMemberEmail,
    newMemberGroup,
    setNewMemberGroup,
    currentUserEmail,
    hasChanges,
    isSubmitting,
    addMember,
    updateMemberRole,
    handleUpdateMembers,
  } = useBucketMembersData(bucket);

  if (isLoading) {
    return <BucketMembersSkeleton />;
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Users className="h-5 w-5" />
          Bucket Members
        </CardTitle>
        <CardDescription>
          Manage who has access to this bucket and their permissions
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="space-y-4">
          <div className="text-sm font-medium">Add Member</div>
          <div className="flex gap-3">
            <Input
              type="email"
              placeholder="Enter email address"
              value={newMemberEmail}
              onChange={(e) => setNewMemberEmail(e.target.value)}
              className="flex-1"
            />
            <Select value={newMemberGroup} onValueChange={setNewMemberGroup}>
              <SelectTrigger className="w-32">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {bucketGroups.map((group) => (
                  <SelectItem key={group.id} value={group.id}>
                    {group.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button
              onClick={addMember}
              disabled={
                !newMemberEmail.trim() || !EMAIL_REGEX.test(newMemberEmail)
              }
              size="sm"
            >
              <UserPlus className="h-4 w-4" />
            </Button>
          </div>
        </div>

        <div className="space-y-4">
          <div className="text-sm font-medium">Members</div>
          <div className="space-y-3">
            {membersState.map((member) => (
              <BucketMember
                key={member.email}
                member={member}
                isCurrentUser={member.email === currentUserEmail}
                updateMemberRole={updateMemberRole}
              />
            ))}
          </div>
        </div>

        <div className="flex justify-end border-t pt-4">
          <Button
            onClick={handleUpdateMembers}
            disabled={!hasChanges || isSubmitting}
          >
            {isSubmitting ? "Updating..." : "Update Members"}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
};
