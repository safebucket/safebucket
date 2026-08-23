import { CheckCircle, LoaderCircle, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";
import type { BucketItem } from "@/types/bucket.ts";
import { isFolder } from "@/components/bucket-view/helpers/utils";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { FileStatus } from "@/types/file.ts";

interface IFileStatusBadgeProps {
  item: BucketItem;
}

const pill = "rounded-full border";

export const FileStatusBadge: FC<IFileStatusBadgeProps> = ({
  item,
}: IFileStatusBadgeProps) => {
  const { t } = useTranslation();

  if (isFolder(item)) {
    return (
      <Badge variant="outline" className="rounded-full">
        {t("bucket.view.folder")}
      </Badge>
    );
  }

  switch (item.status) {
    case FileStatus.uploaded:
      return (
        <Badge
          className={cn(
            pill,
            "border-green-200 bg-green-100 text-green-800 dark:border-green-800 dark:bg-green-900/20 dark:text-green-300",
          )}
        >
          <CheckCircle className="h-3 w-3" />
          {t("bucket.list_view.uploaded")}
        </Badge>
      );
    case FileStatus.uploading:
      return (
        <Badge
          className={cn(
            pill,
            "border-blue-200 bg-blue-100 text-blue-800 dark:border-blue-800 dark:bg-blue-900/20 dark:text-blue-300",
          )}
        >
          <LoaderCircle className="h-3 w-3 animate-spin" />
          {t("bucket.list_view.uploading")}
        </Badge>
      );
    case FileStatus.deleting:
      return (
        <Badge
          className={cn(
            pill,
            "border-red-200 bg-red-100 text-red-800 dark:border-red-800 dark:bg-red-900/20 dark:text-red-300",
          )}
        >
          <LoaderCircle className="h-3 w-3 animate-spin" />
          {t("bucket.list_view.deleting")}
        </Badge>
      );
    case FileStatus.deleted:
      return (
        <Badge
          className={cn(
            pill,
            "border-orange-200 bg-orange-100 text-orange-800 dark:border-orange-800 dark:bg-orange-900/20 dark:text-orange-300",
          )}
        >
          <Trash2 className="h-3 w-3" />
          {t("bucket.trash_view.trashed")}
        </Badge>
      );
    default:
      return <span className="text-muted-foreground">-</span>;
  }
};
