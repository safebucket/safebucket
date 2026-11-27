import { useEffect, useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { LoaderCircle, MessagesSquare } from "lucide-react";
import type { FC } from "react";

import { useSession } from "@/hooks/useAuth";
import { useBucketPermissions } from "@/hooks/usePermissions";
import { CommentForm } from "@/components/file-comments/components/CommentForm";
import { CommentItem } from "@/components/file-comments/components/CommentItem";
import { resolveErrorMessage } from "@/lib/toast";
import {
  fileCommentsQueryOptions,
  useFileCommentMutations,
} from "@/queries/file-comments";

interface IFileCommentsPanelProps {
  bucketId: string;
  fileId: string;
}

export const FileCommentsPanel: FC<IFileCommentsPanelProps> = ({
  bucketId,
  fileId,
}: IFileCommentsPanelProps) => {
  const { t } = useTranslation();
  const session = useSession();
  const { isOwner } = useBucketPermissions(bucketId);
  const canModerate = isOwner || session?.role === "admin";
  const comments = useQuery(fileCommentsQueryOptions(bucketId, fileId));
  const { create, update, remove } = useFileCommentMutations(bucketId, fileId);
  const scrollRef = useRef<HTMLDivElement>(null);
  const commentCount = comments.data?.length;

  useEffect(() => {
    const el = scrollRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [commentCount]);

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div
        ref={scrollRef}
        className="flex min-h-0 flex-1 flex-col overflow-y-auto p-4"
      >
        {comments.isPending && (
          <div className="flex flex-1 items-center justify-center">
            <LoaderCircle className="size-5 animate-spin text-muted-foreground" />
          </div>
        )}
        {comments.isError && (
          <p className="text-sm text-destructive">
            {resolveErrorMessage(comments.error)}
          </p>
        )}
        {comments.isSuccess && comments.data.length === 0 && (
          <div className="flex flex-1 flex-col items-center justify-center gap-2 text-muted-foreground">
            <MessagesSquare className="size-6" />
            <p className="text-sm">{t("file_comments.empty")}</p>
          </div>
        )}
        {comments.isSuccess && comments.data.length > 0 && (
          <ul className="mt-auto flex flex-col gap-5">
            {comments.data.map((comment) => {
              const isAuthor = comment.user?.id === session?.userId;
              return (
                <CommentItem
                  key={comment.id}
                  comment={comment}
                  canEdit={isAuthor}
                  canDelete={isAuthor || canModerate}
                  isUpdating={update.isPending}
                  onUpdate={(body) =>
                    update.mutateAsync({ commentId: comment.id, body })
                  }
                  onDelete={() => remove.mutate(comment.id)}
                />
              );
            })}
          </ul>
        )}
      </div>
      <div className="border-t p-4">
        <CommentForm
          placeholder={t("file_comments.placeholder")}
          submitLabel={t("file_comments.submit")}
          isPending={create.isPending}
          onSubmit={(body) => create.mutateAsync(body)}
        />
      </div>
    </div>
  );
};
