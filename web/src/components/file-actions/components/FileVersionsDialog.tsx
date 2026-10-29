import { useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { Download, LoaderCircle, RotateCcw, Trash2 } from "lucide-react";
import type { IFile, IFileVersion } from "@/types/file";
import type { FileVersionAction } from "@/queries/file-versions";
import { getUserDisplayName } from "@/types/user";
import { CustomAlertDialog } from "@/components/dialogs/components/CustomAlertDialog";
import { downloadFromStorage } from "@/components/file-actions/helpers/api";
import { GridActionIcon } from "@/components/bucket-view/components/GridActionIcon";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { errorToast, resolveErrorMessage } from "@/lib/toast";
import { formatDate, formatFileSize } from "@/lib/utils";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
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
        <DialogContent className="flex max-h-[85dvh] flex-col sm:max-w-4xl">
          <DialogHeader className="pr-8">
            <DialogTitle>{t("file_versions.title")}</DialogTitle>
            <DialogDescription className="break-all">
              {file.name}
            </DialogDescription>
          </DialogHeader>
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
            <div className="min-h-0 overflow-y-auto">
              <Table className="min-w-3xl table-fixed">
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-48">
                      {t("file_versions.columns.version")}
                    </TableHead>
                    <TableHead className="w-52">
                      {t("file_versions.columns.date")}
                    </TableHead>
                    <TableHead className="w-24">
                      {t("file_versions.columns.size")}
                    </TableHead>
                    <TableHead>
                      {t("file_versions.columns.uploaded_by")}
                    </TableHead>
                    <TableHead className="w-32 text-right">
                      {t("file_versions.columns.actions")}
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {versions.data.map((version) => {
                    const uploader = version.uploaded_by
                      ? getUserDisplayName(
                          version.uploaded_by,
                          version.uploaded_by.email,
                        )
                      : t("file_versions.unknown_uploader");

                    return (
                      <TableRow
                        key={version.id}
                        role="group"
                        aria-label={t("file_versions.version", {
                          number: version.version,
                        })}
                      >
                        <TableCell className="font-medium">
                          {t("file_versions.version", {
                            number: version.version,
                          })}
                          {version.is_current && (
                            <Badge variant="secondary" className="ml-2">
                              {t("file_versions.current")}
                            </Badge>
                          )}
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          <time dateTime={version.created_at}>
                            {formatDate(version.created_at)}
                          </time>
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {formatFileSize(version.size)}
                        </TableCell>
                        <TableCell
                          className="truncate text-muted-foreground"
                          title={uploader}
                        >
                          {uploader}
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex justify-end gap-1">
                            <GridActionIcon
                              icon={<Download className="h-4 w-4" />}
                              label={t("common.download")}
                              disabled={busy}
                              onClick={() => download.mutate(version.id)}
                            />
                            {canManage && (
                              <>
                                <GridActionIcon
                                  icon={<RotateCcw className="h-4 w-4" />}
                                  label={t("file_versions.restore")}
                                  disabled={busy || version.is_current}
                                  onClick={() =>
                                    setConfirmation({
                                      action: "restore",
                                      version,
                                    })
                                  }
                                />
                                <GridActionIcon
                                  icon={<Trash2 className="h-4 w-4" />}
                                  label={t("common.delete")}
                                  destructive
                                  disabled={busy || version.is_current}
                                  onClick={() =>
                                    setConfirmation({
                                      action: "delete",
                                      version,
                                    })
                                  }
                                />
                              </>
                            )}
                          </div>
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </div>
          )}
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
