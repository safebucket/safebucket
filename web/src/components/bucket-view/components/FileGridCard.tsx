import { useTranslation } from "react-i18next";
import { CheckCircle, Clock, Download, LoaderCircle, Trash2 } from "lucide-react";
import type { FC } from "react";

import { FileStatus } from "@/types/file.ts";
import { isFolder } from "@/components/bucket-view/helpers/utils";
import { cn, formatDate, formatFileSize } from "@/lib/utils";
import { FileActions } from "@/components/FileActions/FileActions";
import { FileIconView } from "@/components/bucket-view/components/FileIconView";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import type { BucketItem } from "@/types/bucket.ts";

interface IFileGridCardProps {
  file: BucketItem;
  selected: BucketItem | null;
  setSelected: (item: BucketItem) => void;
  onDoubleClick: (item: BucketItem) => void;
}

export const FileGridCard: FC<IFileGridCardProps> = ({
  file,
  selected,
  setSelected,
  onDoubleClick,
}: IFileGridCardProps) => {
  const { t } = useTranslation();
  const isSelected = selected?.id === file.id;
  const itemIsFolder = isFolder(file);

  const renderStatusBadge = () => {
    if (!file.status) return null;

    switch (file.status) {
      case FileStatus.uploaded:
        return (
          <Badge className="gap-1 bg-green-100 text-green-800 border-green-200 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800">
            <CheckCircle className="h-3 w-3" />
            {t("bucket.grid_view.uploaded")}
          </Badge>
        );
      case FileStatus.uploading:
        return (
          <Badge className="gap-1 bg-blue-100 text-blue-800 border-blue-200 dark:bg-blue-900/20 dark:text-blue-300 dark:border-blue-800">
            <LoaderCircle className="h-3 w-3 animate-spin" />
            {t("bucket.grid_view.uploading")}
          </Badge>
        );
      case FileStatus.deleting:
        return (
          <Badge className="gap-1 bg-red-100 text-red-800 border-red-200 dark:bg-red-900/20 dark:text-red-300 dark:border-red-800">
            <LoaderCircle className="h-3 w-3 animate-spin" />
            {t("bucket.grid_view.deleting")}
          </Badge>
        );
      case FileStatus.deleted:
        return (
          <Badge className="gap-1 bg-orange-100 text-orange-800 border-orange-200 dark:bg-orange-900/20 dark:text-orange-300 dark:border-orange-800">
            <Trash2 className="h-3 w-3" />
            {t("bucket.trash_view.trashed")}
          </Badge>
        );
      case FileStatus.restoring:
        return (
          <Badge className="gap-1 bg-blue-100 text-blue-800 border-blue-200 dark:bg-blue-900/20 dark:text-blue-300 dark:border-blue-800">
            <LoaderCircle className="h-3 w-3 animate-spin" />
            {t("bucket.trash_view.restoring")}
          </Badge>
        );
      default:
        return null;
    }
  };

  const renderSharingIndicators = () => {
    // Only show for files (not folders) that have sharing options
    if (isFolder(file) || !("sharing_options" in file) || !file.sharing_options) {
      return null;
    }

    const sharingOptions = file.sharing_options;
    const indicators = [];

    // Expiration indicator
    if (sharingOptions.expires_at) {
      const expiresAt = new Date(sharingOptions.expires_at);
      const isExpired = expiresAt < new Date();

      indicators.push(
        <Tooltip key="expiration">
          <TooltipTrigger asChild>
            <Clock
              className={cn(
                "h-3.5 w-3.5",
                isExpired ? "text-red-500" : "text-blue-500",
                isSelected && "text-primary-foreground"
              )}
            />
          </TooltipTrigger>
          <TooltipContent>
            {isExpired
              ? t("file.sharing.expired")
              : t("file.sharing.expires_at", { date: formatDate(sharingOptions.expires_at) })}
          </TooltipContent>
        </Tooltip>
      );
    }

    // Download limit indicator
    if (sharingOptions.max_downloads) {
      const remaining = sharingOptions.max_downloads - sharingOptions.download_count;
      const isLimitReached = remaining <= 0;

      indicators.push(
        <Tooltip key="downloads">
          <TooltipTrigger asChild>
            <div className="flex items-center gap-0.5">
              <Download
                className={cn(
                  "h-3.5 w-3.5",
                  isLimitReached ? "text-red-500" : "text-purple-500",
                  isSelected && "text-primary-foreground"
                )}
              />
              <span
                className={cn(
                  "text-xs",
                  isLimitReached ? "text-red-500" : "text-purple-500",
                  isSelected && "text-primary-foreground"
                )}
              >
                {remaining}
              </span>
            </div>
          </TooltipTrigger>
          <TooltipContent>
            {t("file.sharing.downloads_remaining", {
              count: remaining,
              max: sharingOptions.max_downloads,
            })}
          </TooltipContent>
        </Tooltip>
      );
    }

    return indicators.length > 0 ? (
      <div className="flex items-center gap-2">{indicators}</div>
    ) : null;
  };

  return (
    <FileActions file={file} type="context">
      <Card
        className={cn(
          "relative flex cursor-pointer flex-col gap-4 p-5 transition-all hover:shadow-md min-h-[180px]",
          isSelected &&
            "bg-primary text-primary-foreground ring-2 ring-primary",
        )}
        onClick={() => setSelected(file)}
        onDoubleClick={() => onDoubleClick(file)}
      >
        <div className="flex items-start gap-4">
          <div
            className={cn(
              "bg-muted flex aspect-square w-16 flex-shrink-0 items-center justify-center rounded-lg",
              isSelected && "bg-primary-foreground text-primary",
            )}
          >
            <FileIconView
              className="h-8 w-8"
              isFolder={itemIsFolder}
              extension={!itemIsFolder ? file.extension : undefined}
            />
          </div>

          <div className="flex-1 min-w-0 flex flex-col gap-2">
            <h3
              className={cn(
                "font-medium text-sm leading-tight line-clamp-2",
                isSelected && "text-primary-foreground",
              )}
              title={file.name}
            >
              {file.name}
            </h3>
            <div className="flex flex-col gap-1">
              <p
                className={cn(
                  "text-xs",
                  isSelected
                    ? "text-primary-foreground/80"
                    : "text-muted-foreground",
                )}
              >
                {itemIsFolder ? "-" : formatFileSize(file.size)}
              </p>
              <div className="flex items-center gap-2">
                {renderStatusBadge()}
                {renderSharingIndicators()}
              </div>
            </div>
          </div>
        </div>

        <div className="flex items-center justify-between gap-2 pt-2 mt-auto border-t">
          <Badge
            variant="secondary"
            className={cn(
              "text-xs",
              isSelected && "bg-primary-foreground/20 text-primary-foreground",
            )}
          >
            {itemIsFolder ? "folder" : file.extension}
          </Badge>
          <span
            className={cn(
              "text-xs",
              isSelected
                ? "text-primary-foreground/70"
                : "text-muted-foreground",
            )}
            title={formatDate(file.created_at)}
          >
            {formatDate(file.created_at)}
          </span>
        </div>
      </Card>
    </FileActions>
  );
};
