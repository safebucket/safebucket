import { UserX } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";

import type {
  IBucketMember,
  IMembers,
} from "@/components/bucket-view/helpers/types";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemMedia,
  ItemTitle,
} from "@/components/ui/item";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { bucketGroups } from "@/types/bucket.ts";

interface IAddMembersListProps {
  shareWith: Array<IMembers>;
  onGroupChange: (email: string, groupId: string) => void;
  onRemoveUser: (email: string) => void;
  currentUserEmail?: string;
  currentUserName?: string;
  showCurrentUser?: boolean;
  existingMembers?: Array<IBucketMember>;
  onExistingMemberGroupChange: (email: string, groupId: string) => void;
}

export const AddMembersList: FC<IAddMembersListProps> = ({
  shareWith,
  onGroupChange,
  onRemoveUser,
  currentUserEmail,
  currentUserName,
  showCurrentUser = true,
  existingMembers = [],
  onExistingMemberGroupChange,
}) => {
  const { t } = useTranslation();

  // Filter existing members to find current user (if exists)
  const currentUserMember = existingMembers.find(
    (member) => member.email === currentUserEmail,
  );
  // Filter out current user from existing members to avoid duplication
  const otherExistingMembers = existingMembers.filter(
    (member) => member.email !== currentUserEmail,
  );

  return (
    <div className="space-y-4">
      <div className="text-sm font-medium">
        {t("bucket.settings.members.people_with_access")}
      </div>
      <div className="space-y-3">
        {/* Current User */}
        {showCurrentUser && currentUserEmail && (
          <Item variant="outline">
            <ItemMedia>
              <Avatar className="size-10">
                <AvatarImage src="/avatars/03.png" />
                <AvatarFallback>
                  {currentUserEmail.charAt(0).toUpperCase()}
                </AvatarFallback>
              </Avatar>
            </ItemMedia>
            <ItemContent>
              <ItemTitle>
                {currentUserName
                  ? `${currentUserName} (${t("bucket.settings.members.you")})`
                  : t("bucket.settings.members.you")}
              </ItemTitle>
              <ItemDescription>{currentUserEmail}</ItemDescription>
            </ItemContent>
            <ItemActions>
              <Select
                value={currentUserMember ? currentUserMember.group : "owner"}
                disabled={true}
              >
                <SelectTrigger className="w-32">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="owner">
                    {t("bucket.settings.members.owner")}
                  </SelectItem>
                </SelectContent>
              </Select>
            </ItemActions>
          </Item>
        )}

        {/* Existing Members (excluding current user) */}
        {otherExistingMembers.map((member) => (
          <Item key={member.email} variant="outline">
            <ItemMedia>
              <Avatar className="size-10">
                <AvatarImage src="/avatars/01.png" />
                <AvatarFallback>
                  {member.email.charAt(0).toUpperCase()}
                </AvatarFallback>
              </Avatar>
            </ItemMedia>
            <ItemContent>
              <ItemTitle>
                {member.first_name && member.last_name
                  ? `${member.first_name} ${member.last_name}`
                  : member.email}
              </ItemTitle>
              <ItemDescription>{member.email}</ItemDescription>
              {member.status === "invited" && (
                <div className="text-xs text-orange-500">
                  {t("bucket.settings.members.invited")}
                </div>
              )}
            </ItemContent>
            <ItemActions>
              <Select
                value={member.group}
                onValueChange={(value) =>
                  onExistingMemberGroupChange(member.email, value)
                }
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
                </SelectContent>
              </Select>
            </ItemActions>
          </Item>
        ))}

        {/* Newly Added Members */}
        {shareWith.map((user) => (
          <Item key={user.email} variant="outline">
            <ItemMedia>
              <Avatar className="size-10">
                <AvatarImage src="/avatars/01.png" />
                <AvatarFallback>
                  {user.email.charAt(0).toUpperCase()}
                </AvatarFallback>
              </Avatar>
            </ItemMedia>
            <ItemContent>
              <ItemTitle>{user.email}</ItemTitle>
            </ItemContent>
            <ItemActions>
              <Select
                value={user.group}
                onValueChange={(value) => onGroupChange(user.email, value)}
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
                  <SelectSeparator />
                  <SelectItem value="remove" className="text-red-600">
                    Remove
                  </SelectItem>
                </SelectContent>
              </Select>
              <Button
                variant="outline"
                size="icon"
                onClick={(e) => {
                  e.preventDefault();
                  onRemoveUser(user.email);
                }}
              >
                <UserX />
              </Button>
            </ItemActions>
          </Item>
        ))}
      </div>
    </div>
  );
};
