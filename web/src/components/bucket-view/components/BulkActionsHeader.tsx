import { Download, Loader2, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";
import type { IBucket } from "@/types/bucket.ts";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { useBulkDownload } from "@/components/bucket-view/hooks/useBulkDownload";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { formatFileSize } from "@/lib/utils";

interface IBulkActionsHeaderProps {
  bucket: IBucket;
}

export const BulkActionsHeader: FC<IBulkActionsHeaderProps> = ({
  bucket,
}: IBulkActionsHeaderProps) => {
  const { t } = useTranslation();
  const { rowSelection, clearRowSelection } = useBucketViewContext();

  const {
    start,
    dismissBlocked,
    blocked,
    isRunning,
    fileCount,
    totalBytes,
    maxBytes,
    maxFiles,
  } = useBulkDownload({ bucket, rowSelection, clearRowSelection });

  const selectedCount = Object.keys(rowSelection).filter(
    (k) => rowSelection[k],
  ).length;

  return (
    <div className="shrink-0 -my-2.25">
      <div className="bg-primary/5 dark:bg-primary/10 border-primary/20 flex items-center justify-between gap-4 rounded-md border px-3 py-2 shadow-sm">
        <div className="flex min-w-0 items-center gap-2">
          <Button
            variant="ghost"
            size="icon"
            onClick={clearRowSelection}
            aria-label={t("bucket.bulk_download.clear_selection")}
            disabled={isRunning}
          >
            <X className="h-4 w-4" />
          </Button>
          <span className="text-base font-medium md:text-lg">
            {t("bucket.bulk_download.selected", { count: selectedCount })}
          </span>
        </div>
        <div className="flex items-center gap-2 md:gap-4">
          <Button onClick={start} disabled={isRunning || fileCount === 0}>
            {isRunning ? (
              <Loader2 className="h-4 w-4 animate-spin md:mr-2" />
            ) : (
              <Download className="h-4 w-4 md:mr-2" />
            )}
            <span className="hidden md:inline">
              {t("bucket.bulk_download.download", {
                count: fileCount,
                size: formatFileSize(totalBytes),
              })}
            </span>
          </Button>
        </div>
      </div>

      <AlertDialog
        open={blocked !== null}
        onOpenChange={(isOpen) => {
          if (!isOpen) dismissBlocked();
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {t("bucket.bulk_download.over_limit_title")}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t("bucket.bulk_download.over_limit_body", {
                count: blocked?.count ?? 0,
                size: formatFileSize(blocked?.bytes ?? 0),
                maxCount: maxFiles,
                maxSize: formatFileSize(maxBytes),
              })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogAction onClick={dismissBlocked}>
              {t("common.ok")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};
