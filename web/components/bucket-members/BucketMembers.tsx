import React, { FC } from "react";

import { UserPlus, Users } from "lucide-react";

import {
  EMAIL_REGEX,
  bucketGroups,
} from "@/components/add-members/helpers/constants";
import { BucketMembersSkeleton } from "@/components/bucket-members/components/BucketMembersSkeleton";
import { useBucketMembersData } from "@/components/bucket-members/hooks/useBucketMembersData";
import { IBucket } from "@/components/bucket-view/helpers/types";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
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
    newMemberRole,
    setNewMemberRole,
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
            <Select value={newMemberRole} onValueChange={setNewMemberRole}>
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
            {membersState.map((member) => {
              const isCurrentUser = member.email === currentUserEmail;

              return (
                <div
                  key={member.email}
                  className="flex items-center justify-between rounded-lg border p-3"
                >
                  <div className="flex items-center space-x-4">
                    <Avatar>
                      <AvatarImage src="/avatars/01.png" />
                      <AvatarFallback>
                        {member.email.charAt(0).toUpperCase()}
                      </AvatarFallback>
                    </Avatar>
                    <div>
                      <div className="text-sm font-medium">
                        {member.first_name && member.last_name
                          ? `${member.first_name} ${member.last_name}${isCurrentUser ? " (you)" : ""}`
                          : member.email}
                      </div>
                      {member.first_name && member.last_name && (
                        <div className="text-sm text-muted-foreground">
                          {member.email}
                        </div>
                      )}
                      {member.status === "invited" && (
                        <div className="text-xs text-orange-500">
                          Pending invitation
                        </div>
                      )}
                      {member.isNew && (
                        <div className="text-xs text-green-500">New member</div>
                      )}
                    </div>
                  </div>

                  <div className="flex items-center">
                    <Select
                      value={member.role}
                      onValueChange={(value) =>
                        updateMemberRole(member.email, value)
                      }
                      disabled={isCurrentUser}
                    >
                      <SelectTrigger className="w-32">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {bucketGroups.map((group) => (
                          <SelectItem key={group.id} value={group.id}>
                            {group.name}
                          </SelectItem>
                        ))}
                        {!isCurrentUser && (
                          <SelectItem value="remove" className="text-red-600">
                            Remove
                          </SelectItem>
                        )}
                      </SelectContent>
                    </Select>
                  </div>
                </div>
              );
            })}
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
