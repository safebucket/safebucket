import { useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import {
  Download,
  LoaderCircle,
  RefreshCw,
  RotateCcw,
  Trash2,
} from "lucide-react";
import type { IFile, IFileVersion } from "@/types/file";
import type { FileVersionAction } from "@/queries/file-versions";
import { CustomAlertDialog } from "@/components/dialogs/components/CustomAlertDialog";
import { downloadFromStorage } from "@/components/file-actions/helpers/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { errorToast, resolveErrorMessage } from "@/lib/toast";
import { formatDate, formatFileSize } from "@/lib/utils";
import {
  fileVersionsQueryOptions,
  getFileVersionDownload,
  useFileVersionMutation,
} from "@/queries/file-versions";

interface FileVersionsDialogProps {
  bucketId: string;
  file: Pick<IFile, "id" | "name">;
  canManage: boolean;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function FileVersionsDialog({
  bucketId,
  file,
  canManage,
  open,
  onOpenChange,
}: FileVersionsDialogProps) {
  const { t } = useTranslation();
  const [confirmation, setConfirmation] = useState<{
    action: FileVersionAction;
    version: IFileVersion;
  } | null>(null);
  const versions = useQuery({
    ...fileVersionsQueryOptions(bucketId, file.id),
    enabled: open,
  });
  const mutation = useFileVersionMutation(bucketId, file.id);
  const download = useMutation({
    mutationFn: (versionId: string) =>
      getFileVersionDownload(bucketId, file.id, versionId),
    onSuccess: ({ url }) => downloadFromStorage(url, file.name),
    onError: (error) => errorToast(error),
  });
  const busy = mutation.isPending || download.isPending;

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="flex max-h-[85dvh] flex-col sm:max-w-2xl">
          <DialogHeader className="pr-8">
            <DialogTitle>{t("file_versions.title")}</DialogTitle>
            <DialogDescription className="break-all">
              {file.name}
            </DialogDescription>
          </DialogHeader>
          <div className="flex justify-end">
            <Button
              variant="outline"
              size="sm"
              disabled={versions.isFetching || busy}
              onClick={() => void versions.refetch()}
            >
              <RefreshCw
                className={versions.isFetching ? "animate-spin" : undefined}
              />
              {t("file_versions.refresh")}
            </Button>
          </div>
          {versions.isPending && (
            <div
              role="status"
              className="flex items-center justify-center gap-2 py-8 text-muted-foreground"
            >
              <LoaderCircle className="size-4 animate-spin" />
              {t("common.loading")}
            </div>
          )}
          {versions.isError && (
            <div
              role="alert"
              className="space-y-3 rounded-xl bg-destructive/10 p-4 text-destructive"
            >
              <p>{resolveErrorMessage(versions.error)}</p>
              <Button
                variant="outline"
                disabled={versions.isFetching}
                onClick={() => void versions.refetch()}
              >
                {t("file_versions.retry")}
              </Button>
            </div>
          )}
          {versions.isSuccess && versions.data.length === 0 && (
            <p className="py-8 text-center text-muted-foreground">
              {t("file_versions.empty")}
            </p>
          )}
          {versions.isSuccess && versions.data.length > 0 && (
            <ul className="min-h-0 divide-y overflow-y-auto">
              {versions.data.map((version) => (
                <li key={version.id} className="py-4 first:pt-0 last:pb-0">
                  <div
                    role="group"
                    aria-label={t("file_versions.version", {
                      number: version.version,
                    })}
                    className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between"
                  >
                    <div className="space-y-1">
                      <div className="flex items-center gap-2 font-medium">
                        {t("file_versions.version", {
                          number: version.version,
                        })}
                        {version.is_current && (
                          <Badge variant="secondary">
                            {t("file_versions.current")}
                          </Badge>
                        )}
                      </div>
                      <div className="flex flex-wrap gap-x-3 text-xs text-muted-foreground">
                        <time dateTime={version.created_at}>
                          {formatDate(version.created_at)}
                        </time>
                        <span>{formatFileSize(version.size)}</span>
                      </div>
                    </div>
                    <div className="flex flex-wrap gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={busy}
                        onClick={() => download.mutate(version.id)}
                      >
                        <Download />
                        {t("common.download")}
                      </Button>
                      {canManage && (
                        <>
                          <Button
                            variant="outline"
                            size="sm"
                            disabled={busy || version.is_current}
                            onClick={() =>
                              setConfirmation({ action: "restore", version })
                            }
                          >
                            <RotateCcw />
                            {t("file_versions.restore")}
                          </Button>
                          <Button
                            variant="destructive"
                            size="sm"
                            disabled={busy || version.is_current}
                            onClick={() =>
                              setConfirmation({ action: "delete", version })
                            }
                          >
                            <Trash2 />
                            {t("common.delete")}
                          </Button>
                        </>
                      )}
                    </div>
                  </div>
                </li>
              ))}
            </ul>
          )}
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline">{t("common.close")}</Button>
            </DialogClose>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      {confirmation && (
        <CustomAlertDialog
          open={open}
          onOpenChange={(isOpen) => {
            if (!isOpen) setConfirmation(null);
          }}
          destructive={confirmation.action === "delete"}
          title={
            confirmation.action === "restore"
              ? t("file_versions.restore_title", {
                  number: confirmation.version.version,
                })
              : t("file_versions.delete_title", {
                  number: confirmation.version.version,
                })
          }
          description={
            confirmation.action === "restore"
              ? t("file_versions.restore_description", { filename: file.name })
              : t("file_versions.delete_description", { filename: file.name })
          }
          confirmLabel={
            confirmation.action === "restore"
              ? t("file_versions.restore")
              : t("common.delete")
          }
          cancelLabel={t("common.cancel")}
          onConfirm={() => {
            if (busy || !canManage) return;
            mutation.mutate({
              action: confirmation.action,
              versionId: confirmation.version.id,
            });
          }}
        />
      )}
    </>
  );
}
