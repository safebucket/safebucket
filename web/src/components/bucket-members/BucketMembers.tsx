import { UserPlus, Users } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";

import type { IBucket } from "@/types/bucket.ts";
import { EMAIL_REGEX, bucketGroups } from "@/types/bucket.ts";
import { BucketMember } from "@/components/bucket-members/components/BucketMember";
import { BucketMembersSkeleton } from "@/components/bucket-members/components/BucketMembersSkeleton";
import { useBucketMembersData } from "@/components/bucket-members/hooks/useBucketMembersData";
import { useBucketPermissions } from "@/hooks/usePermissions";
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
import { ButtonGroup } from "@/components/ui/button-group.tsx";

interface IBucketMembersProps {
  bucket: IBucket;
}

export const BucketMembers: FC<IBucketMembersProps> = ({ bucket }) => {
  const { t } = useTranslation();
  const { isOwner } = useBucketPermissions(bucket.id);
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
          {t("bucket.settings.members.title")}
        </CardTitle>
        <CardDescription>
          {t("bucket.settings.members.description")}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        {isOwner && (
          <div className="space-y-4">
            <div className="text-sm font-medium">
              {t("bucket.settings.members.add_member")}
            </div>
            <div className="flex gap-3">
              <ButtonGroup className="w-full">
                <Input
                  id="email"
                  type="email"
                  placeholder={t("bucket.settings.members.enter_email")}
                  value={newMemberEmail}
                  onChange={(e) => setNewMemberEmail(e.target.value)}
                  className="flex-1 w-full"
                />
                <Select
                  value={newMemberGroup}
                  onValueChange={setNewMemberGroup}
                >
                  <SelectTrigger className="w-32">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent className="min-w-24">
                    {bucketGroups.map((group) => (
                      <SelectItem key={group.id} value={group.id}>
                        {group.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </ButtonGroup>
              <ButtonGroup>
                <Button
                  aria-label="Add member"
                  onClick={addMember}
                  disabled={
                    !newMemberEmail.trim() || !EMAIL_REGEX.test(newMemberEmail)
                  }
                  variant="outline"
                  size="icon"
                >
                  <UserPlus />
                </Button>
              </ButtonGroup>
            </div>
          </div>
        )}

        <div className="space-y-4">
          <div className="text-sm font-medium">
            {t("bucket.settings.members.members")}
          </div>
          <div className="space-y-3">
            {membersState.map((member) => (
              <BucketMember
                key={member.email}
                member={member}
                isCurrentUser={member.email === currentUserEmail}
                isOwner={isOwner}
                updateMemberRole={updateMemberRole}
              />
            ))}
          </div>
        </div>

        {isOwner && (
          <div className="flex justify-end border-t pt-4">
            <Button
              onClick={handleUpdateMembers}
              disabled={!hasChanges || isSubmitting}
            >
              {t("common.save")}
            </Button>
          </div>
        )}
      </CardContent>
    </Card>
  );
};
