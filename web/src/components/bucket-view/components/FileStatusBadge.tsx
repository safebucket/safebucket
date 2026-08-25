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
          className={cn(pill, "border-success/20 bg-success/10 text-success")}
        >
          <CheckCircle className="h-3 w-3" />
          {t("bucket.list_view.uploaded")}
        </Badge>
      );
    case FileStatus.uploading:
      return (
        <Badge className={cn(pill, "border-info/20 bg-info/10 text-info")}>
          <LoaderCircle className="h-3 w-3 animate-spin" />
          {t("bucket.list_view.uploading")}
        </Badge>
      );
    case FileStatus.deleting:
      return (
        <Badge
          className={cn(
            pill,
            "border-destructive/20 bg-destructive/10 text-destructive",
          )}
        >
          <LoaderCircle className="h-3 w-3 animate-spin" />
          {t("bucket.list_view.deleting")}
        </Badge>
      );
    case FileStatus.deleted:
      return (
        <Badge
          className={cn(pill, "border-warning/20 bg-warning/10 text-warning")}
        >
          <Trash2 className="h-3 w-3" />
          {t("bucket.trash_view.trashed")}
        </Badge>
      );
    default:
      return <span className="text-muted-foreground">-</span>;
  }
};
