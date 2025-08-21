import type { FC } from "react";

import type { IMemberState } from "@/components/bucket-members/hooks/useBucketMembersData";
import { bucketGroups } from "@/components/add-members";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

interface IBucketMemberProps {
  member: IMemberState;
  isCurrentUser: boolean;
  updateMemberRole: (email: string, newRole: string) => void;
}

export const BucketMember: FC<IBucketMemberProps> = ({
  member,
  isCurrentUser,
  updateMemberRole,
}) => (
  <div
    key={member.email}
    className="flex items-center justify-between rounded-lg border p-3"
  >
    <div className="flex items-center space-x-4">
      <Avatar>
        <AvatarImage src="/avatars/01.png" />
        <AvatarFallback>{member.email.charAt(0).toUpperCase()}</AvatarFallback>
      </Avatar>
      <div>
        <div className="text-sm font-medium">
          {member.first_name && member.last_name
            ? `${member.first_name} ${member.last_name}${isCurrentUser ? " (you)" : ""}`
            : member.email}
        </div>
        {member.first_name && member.last_name && (
          <div className="text-muted-foreground text-sm">{member.email}</div>
        )}
        {member.status === "invited" && (
          <div className="text-xs text-orange-500">Pending invitation</div>
        )}
        {member.isNew && (
          <div className="text-xs text-green-500">New member</div>
        )}
      </div>
    </div>

    <div className="flex items-center">
      <Select
        value={member.group}
        onValueChange={(value) => updateMemberRole(member.email, value)}
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
            <>
              <SelectSeparator />
              <SelectItem value="remove" className="text-red-600">
                Remove
              </SelectItem>
            </>
          )}
        </SelectContent>
      </Select>
    </div>
  </div>
);
