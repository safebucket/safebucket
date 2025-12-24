import type { FC } from "react";

import type { IMemberState } from "@/components/bucket-members/hooks/useBucketMembersData";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemMedia,
  ItemTitle,
} from "@/components/ui/item.tsx";
import { bucketGroups } from "@/types/bucket.ts";

interface IBucketMemberProps {
  member: IMemberState;
  isCurrentUser: boolean;
  isOwner: boolean;
  updateMemberRole: (email: string, newRole: string) => void;
}

export const BucketMember: FC<IBucketMemberProps> = ({
  member,
  isCurrentUser,
  isOwner,
  updateMemberRole,
}) => (
  <Item key={member.email} variant="outline">
    <ItemMedia>
      <Avatar className="size-10">
        <AvatarImage src="/avatars/01.png" />
        <AvatarFallback>{member.email.charAt(0).toUpperCase()}</AvatarFallback>
      </Avatar>
    </ItemMedia>
    <ItemContent>
      <ItemTitle>
        {member.first_name && member.last_name
          ? `${member.first_name} ${member.last_name}${isCurrentUser ? " (you)" : ""}`
          : member.email}
      </ItemTitle>
      <ItemDescription>{member.email}</ItemDescription>
      <div className="">
        {member.isNew && (
          <div className="text-xs text-green-500">New member</div>
        )}
      </div>
    </ItemContent>
    <ItemActions>
      <Select
        value={member.group}
        onValueChange={(value) => updateMemberRole(member.email, value)}
        disabled={isCurrentUser || !isOwner}
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
          {!isCurrentUser && isOwner && (
            <>
              <SelectSeparator />
              <SelectItem value="remove" className="text-red-600">
                Remove
              </SelectItem>
            </>
          )}
        </SelectContent>
      </Select>
    </ItemActions>
  </Item>
);
