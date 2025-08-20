import type { FC } from "react";

import { UserX } from "lucide-react";

import { bucketGroups } from "@/components/add-members/helpers/constants";
import type {
  IBucketMember,
  IMembers,
} from "@/components/bucket-view/helpers/types";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

interface IPeopleWithAccessProps {
  shareWith: IMembers[];
  onGroupChange: (email: string, groupId: string) => void;
  onRemoveUser: (email: string) => void;
  currentUserEmail?: string;
  currentUserName?: string;
  showCurrentUser?: boolean;
  existingMembers?: IBucketMember[];
  onExistingMemberGroupChange: (email: string, groupId: string) => void;
}

export const PeopleWithAccess: FC<IPeopleWithAccessProps> = ({
  shareWith,
  onGroupChange,
  onRemoveUser,
  currentUserEmail,
  currentUserName,
  showCurrentUser = true,
  existingMembers = [],
  onExistingMemberGroupChange,
}) => {
  // Filter existing members to find current user (if exists)
  const currentUserMember = existingMembers.find(
    (member) => member.email === currentUserEmail,
  );
  // Filter out current user from existing members to avoid duplication
  const otherExistingMembers = existingMembers.filter(
    (member) => member.email !== currentUserEmail,
  );

  return (
    <div className="mb-2 space-y-4">
      <div className="text-sm font-medium">People with access</div>
      <div className="grid gap-2">
        {showCurrentUser && currentUserEmail && (
          <div className="mb-2 grid grid-cols-12 items-center">
            <div className="col-span-9 flex items-center space-x-4">
              <Avatar>
                <AvatarImage src="/avatars/03.png" />
                <AvatarFallback>
                  {currentUserEmail.charAt(0).toUpperCase()}
                </AvatarFallback>
              </Avatar>
              <div>
                <p className="text-sm leading-none font-medium">
                  {currentUserName ? `${currentUserName} (you)` : "You"}
                </p>
                <p className="text-muted-foreground text-sm">
                  {currentUserEmail}
                </p>
              </div>
            </div>

            <div className="col-span-2 mr-1 flex">
              <Button
                variant="outline"
                size="sm"
                disabled={true}
                className="w-full"
              >
                {currentUserMember
                  ? currentUserMember.role?.charAt(0).toUpperCase() +
                    currentUserMember.role?.slice(1)
                  : "Owner"}
              </Button>
            </div>

            <div className="col-span-1"></div>
          </div>
        )}
      </div>

      {/* Display existing members (excluding current user) */}
      {otherExistingMembers.map((member) => (
        <div key={member.email} className="mb-2 grid grid-cols-12 items-center">
          <div className="col-span-9 flex items-center space-x-4">
            <Avatar>
              <AvatarImage src="/avatars/01.png" alt="User avatar" />
              <AvatarFallback>
                {member.email.charAt(0).toUpperCase()}
              </AvatarFallback>
            </Avatar>
            <div>
              <div className="text-sm leading-none font-medium">
                {member.first_name && member.last_name
                  ? `${member.first_name} ${member.last_name}`
                  : member.email}
              </div>
              {member.first_name && member.last_name && (
                <div className="text-muted-foreground text-sm">
                  {member.email}
                </div>
              )}
              {member.status === "invited" && (
                <div className="text-xs text-orange-500">Invited</div>
              )}
            </div>
          </div>

          <div className="col-span-2 mr-1 flex">
            <Select
              value={member.role}
              onValueChange={(val) =>
                onExistingMemberGroupChange(member.email, val)
              }
            >
              <SelectTrigger className="w-full">
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
          </div>

          <div className="col-span-1">
            {/* Only show remove button for invited users */}
            {member.status === "invited" && (
              <Button
                variant="secondary"
                size="sm"
                onClick={(e) => {
                  e.preventDefault();
                  // TODO: Implement remove existing invite functionality
                  console.log("Remove invited user:", member.email);
                }}
              >
                <UserX className="h-4 w-4" />
              </Button>
            )}
          </div>
        </div>
      ))}

      {/* Display newly added members (from shareWith) */}
      {shareWith.map((user) => (
        <div key={user.email} className="mb-2 grid grid-cols-12 items-center">
          <div className="col-span-9 flex items-center space-x-4">
            <Avatar>
              <AvatarImage src="/avatars/01.png" alt="User avatar" />
              <AvatarFallback>
                {user.email.charAt(0).toUpperCase()}
              </AvatarFallback>
            </Avatar>
            <div className="text-sm leading-none font-medium">{user.email}</div>
          </div>

          <div className="col-span-2 mr-1 flex">
            <Select
              value={user.group}
              onValueChange={(val) => onGroupChange(user.email, val)}
            >
              <SelectTrigger className="w-full">
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
          </div>

          <div className="col-span-1">
            <Button
              variant="secondary"
              size="sm"
              onClick={(e) => {
                e.preventDefault();
                onRemoveUser(user.email);
              }}
            >
              <UserX className="h-4 w-4" />
            </Button>
          </div>
        </div>
      ))}
    </div>
  );
};
