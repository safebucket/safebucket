import { useState } from "react";
import { useTranslation } from "react-i18next";
import { EllipsisVertical, Pencil, Trash2 } from "lucide-react";
import type { FC } from "react";

import type { IFileComment } from "@/types/comment";
import { getUserDisplayName, getUserInitials } from "@/types/user";
import { timeAgo } from "@/components/activity-view/helpers/utils";
import { CustomAlertDialog } from "@/components/dialogs/components/CustomAlertDialog";
import { useDialog } from "@/components/dialogs/hooks/useDialog";
import { CommentForm } from "@/components/file-comments/components/CommentForm";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { formatDate } from "@/lib/utils";

interface ICommentItemProps {
  comment: IFileComment;
  canEdit: boolean;
  canDelete: boolean;
  isUpdating: boolean;
  onUpdate: (body: string) => Promise<unknown>;
  onDelete: () => void;
}

export const CommentItem: FC<ICommentItemProps> = ({
  comment,
  canEdit,
  canDelete,
  isUpdating,
  onUpdate,
  onDelete,
}: ICommentItemProps) => {
  const { t } = useTranslation();
  const [isEditing, setIsEditing] = useState(false);
  const deleteDialog = useDialog();
  const authorName = comment.user
    ? getUserDisplayName(comment.user, comment.user.email)
    : t("file_comments.deleted_user");
  const createdAtNano = String(Date.parse(comment.created_at) * 1_000_000);

  return (
    <li className="flex gap-3">
      <Avatar size="sm">
        <AvatarFallback>
          {comment.user ? getUserInitials(comment.user) : "?"}
        </AvatarFallback>
      </Avatar>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="truncate text-sm font-medium">{authorName}</span>
          <time
            dateTime={comment.created_at}
            title={formatDate(comment.created_at)}
            className="shrink-0 text-xs text-muted-foreground"
          >
            {timeAgo(createdAtNano, t)}
          </time>
          {(canEdit || canDelete) && !isEditing && (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon-xs"
                  className="ml-auto"
                  aria-label={t("file_comments.options")}
                >
                  <EllipsisVertical />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                {canEdit && (
                  <DropdownMenuItem onClick={() => setIsEditing(true)}>
                    <Pencil className="mr-2 h-4 w-4" />
                    {t("common.edit")}
                  </DropdownMenuItem>
                )}
                {canDelete && (
                  <DropdownMenuItem
                    className="text-destructive"
                    onClick={deleteDialog.trigger}
                  >
                    <Trash2 className="mr-2 h-4 w-4" />
                    {t("common.delete")}
                  </DropdownMenuItem>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </div>
        {isEditing ? (
          <div className="mt-1">
            <CommentForm
              initialValue={comment.body}
              submitLabel={t("common.save")}
              isPending={isUpdating}
              onSubmit={(body) =>
                onUpdate(body).then(() => setIsEditing(false))
              }
              onCancel={() => setIsEditing(false)}
            />
          </div>
        ) : (
          <p className="text-sm break-words whitespace-pre-wrap">
            {comment.body}
          </p>
        )}
      </div>
      {canDelete && (
        <CustomAlertDialog
          {...deleteDialog.props}
          destructive
          title={t("file_comments.delete_title")}
          description={t("file_comments.delete_description")}
          confirmLabel={t("common.delete")}
          cancelLabel={t("common.cancel")}
          onConfirm={onDelete}
        />
      )}
    </li>
  );
};
