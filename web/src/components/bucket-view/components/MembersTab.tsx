import { Plus, Trash2 } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";
import type { IDataTableColumn } from "@/components/common/components/DataTable/DataTable";
import type { IMemberState } from "@/components/bucket-members/hooks/useBucketMembersData";
import type { IBucket } from "@/types/bucket.ts";
import { EMAIL_REGEX, bucketGroups } from "@/types/bucket.ts";
import { DataTable } from "@/components/common/components/DataTable/DataTable";
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

  const columns = useMemo<Array<IDataTableColumn<IMemberState>>>(
    () => [
      {
        id: "member",
        header: t("bucket.view.members.member"),
        cell: (member) => {
          const isCurrentUser = member.email === currentUserEmail;
          const displayName =
            member.first_name && member.last_name
              ? `${member.first_name} ${member.last_name}`
              : member.email;
          return (
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
          );
        },
      },
      {
        id: "status",
        header: t("bucket.view.members.status"),
        responsive: "sm",
        cell: (member) => (
          <div className="flex items-center gap-2">
            <Badge
              variant={member.status === "active" ? "secondary" : "outline"}
              className="rounded-full"
            >
              {member.status === "active"
                ? t("bucket.view.members.active")
                : t("bucket.view.members.invited")}
            </Badge>
            {member.isNew ? (
              <Badge className="rounded-full border-success/20 bg-success/10 text-success">
                {t("bucket.view.members.new")}
              </Badge>
            ) : null}
          </div>
        ),
      },
      {
        id: "role",
        header: t("bucket.view.members.role_label"),
        cell: (member) => (
          <Select
            value={member.group}
            onValueChange={(value) => updateMemberRole(member.email, value)}
            disabled={member.email === currentUserEmail}
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
        ),
      },
      {
        id: "actions",
        header: "",
        headerClassName: "w-13",
        cellClassName: "text-right",
        cell: (member) => (
          <Button
            variant="ghost"
            size="icon"
            className="text-muted-foreground hover:text-destructive h-7 w-7"
            disabled={member.email === currentUserEmail}
            aria-label={t("bucket.view.members.remove")}
            onClick={() => updateMemberRole(member.email, "remove")}
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        ),
      },
    ],
    [t, currentUserEmail, updateMemberRole],
  );

  if (isLoading) {
    return <BucketMembersSkeleton />;
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-end">
        <GridActionIcon
          icon={<Plus className="h-4 w-4" />}
          label={t("bucket.settings.members.add_member")}
          onClick={() => setAddOpen(true)}
        />
      </div>

      <DataTable
        columns={columns}
        data={membersState}
        getRowId={(member) => member.email}
      />

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
