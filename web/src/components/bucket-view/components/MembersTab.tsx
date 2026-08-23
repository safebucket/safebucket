import { Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";
import type { IBucket } from "@/types/bucket.ts";
import { EMAIL_REGEX, bucketGroups } from "@/types/bucket.ts";
import { BucketMembersSkeleton } from "@/components/bucket-members/components/BucketMembersSkeleton";
import { useBucketMembersData } from "@/components/bucket-members/hooks/useBucketMembersData";
import { AddMemberDialog } from "@/components/bucket-view/components/AddMemberDialog";
import { GridActionIcon } from "@/components/bucket-view/components/GridActionIcon";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { cn } from "@/lib/utils";

interface IMembersTabProps {
  bucket: IBucket;
}

export const MembersTab: FC<IMembersTabProps> = ({
  bucket,
}: IMembersTabProps) => {
  const { t } = useTranslation();
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
  const [addOpen, setAddOpen] = useState(false);

  if (isLoading) {
    return <BucketMembersSkeleton />;
  }

  const cell = "px-4 py-3 align-middle";
  const headCell =
    "px-4 py-2.5 text-left text-xs font-medium tracking-wide text-muted-foreground uppercase";

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-end">
        <GridActionIcon
          icon={<Plus className="h-4 w-4" />}
          label={t("bucket.settings.members.add_member")}
          onClick={() => setAddOpen(true)}
        />
      </div>

      <div className="border-border bg-card overflow-hidden rounded-xl border">
        <table className="w-full border-collapse text-sm">
          <thead className="bg-muted">
            <tr>
              <th className={headCell}>{t("bucket.view.members.member")}</th>
              <th className={cn(headCell, "hidden sm:table-cell")}>
                {t("bucket.view.members.status")}
              </th>
              <th className={headCell}>
                {t("bucket.view.members.role_label")}
              </th>
              <th className={cn(headCell, "w-13")} />
            </tr>
          </thead>
          <tbody>
            {membersState.map((member) => {
              const isCurrentUser = member.email === currentUserEmail;
              const displayName =
                member.first_name && member.last_name
                  ? `${member.first_name} ${member.last_name}`
                  : member.email;
              return (
                <tr
                  key={member.email}
                  className="border-border hover:bg-muted/50 border-t transition-colors"
                >
                  <td className={cell}>
                    <div className="flex items-center gap-3 overflow-hidden">
                      <Avatar className="size-9 shrink-0">
                        <AvatarFallback>
                          {member.email.charAt(0).toUpperCase()}
                        </AvatarFallback>
                      </Avatar>
                      <div className="flex min-w-0 flex-col">
                        <span className="truncate font-medium">
                          {displayName}
                          {isCurrentUser ? (
                            <span className="text-muted-foreground">
                              {" "}
                              {t("bucket.view.members.you")}
                            </span>
                          ) : null}
                        </span>
                        <span className="text-muted-foreground truncate text-xs">
                          {member.email}
                        </span>
                      </div>
                    </div>
                  </td>
                  <td className={cn(cell, "hidden sm:table-cell")}>
                    <div className="flex items-center gap-2">
                      <Badge
                        variant={
                          member.status === "active" ? "secondary" : "outline"
                        }
                        className="rounded-full"
                      >
                        {member.status === "active"
                          ? t("bucket.view.members.active")
                          : t("bucket.view.members.invited")}
                      </Badge>
                      {member.isNew ? (
                        <Badge className="rounded-full border-green-200 bg-green-100 text-green-800 dark:border-green-800 dark:bg-green-900/20 dark:text-green-300">
                          {t("bucket.view.members.new")}
                        </Badge>
                      ) : null}
                    </div>
                  </td>
                  <td className={cell}>
                    <Select
                      value={member.group}
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
                      </SelectContent>
                    </Select>
                  </td>
                  <td className={cn(cell, "text-right")}>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="text-muted-foreground hover:text-destructive h-7 w-7"
                      disabled={isCurrentUser}
                      aria-label={t("bucket.view.members.remove")}
                      onClick={() => updateMemberRole(member.email, "remove")}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {hasChanges ? (
        <div className="bg-accent border-border flex items-center justify-between gap-4 rounded-lg border px-4 py-2.5">
          <span className="text-sm font-medium">
            {t("bucket.view.members.unsaved")}
          </span>
          <Button
            size="sm"
            onClick={handleUpdateMembers}
            disabled={isSubmitting}
          >
            {t("common.save")}
          </Button>
        </div>
      ) : null}

      <AddMemberDialog
        open={addOpen}
        onOpenChange={setAddOpen}
        email={newMemberEmail}
        onEmailChange={setNewMemberEmail}
        group={newMemberGroup}
        onGroupChange={setNewMemberGroup}
        canAdd={
          newMemberEmail.trim().length > 0 && EMAIL_REGEX.test(newMemberEmail)
        }
        onAdd={addMember}
      />
    </div>
  );
};
