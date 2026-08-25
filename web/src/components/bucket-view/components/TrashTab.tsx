import {
  ArchiveRestore,
  EllipsisVertical,
  LoaderCircle,
  Trash2,
} from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";
import type { IDataTableColumn } from "@/components/common/components/DataTable/DataTable";
import type { TrashedItem } from "@/components/bucket-view/hooks/useTrashActions";
import type { IBucket } from "@/types/bucket.ts";
import { DataTable } from "@/components/common/components/DataTable/DataTable";
import { FileIconView } from "@/components/bucket-view/components/FileIconView";
import { useBucketPermissions } from "@/hooks/usePermissions";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { CustomAlertDialog } from "@/components/dialogs/components/CustomAlertDialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { formatDate, formatFileSize } from "@/lib/utils";
import { FileStatus } from "@/types/file.ts";
import { FolderStatus } from "@/types/folder.ts";

interface ISelectedItem {
  id: string;
  name: string;
  itemType: "file" | "folder";
}

interface ITrashTabProps {
  items: Array<TrashedItem>;
  bucket: IBucket;
  onRestore: (id: string, name: string, itemType: "file" | "folder") => void;
  onPermanentDelete: (
    id: string,
    name: string,
    itemType: "file" | "folder",
  ) => void;
}

const buildFolderPath = (
  folderId: string | undefined,
  folders: IBucket["folders"],
): string => {
  if (!folderId) return "/";
  const path: Array<string> = [];
  let currentId: string | undefined = folderId;
  while (currentId) {
    const folder = folders.find((f) => f.id === currentId);
    if (!folder) break;
    path.unshift(folder.name);
    currentId = folder.folder_id;
  }
  return "/" + path.join("/");
};

const itemSize = (item: TrashedItem): number =>
  item.itemType === "folder" || !("size" in item) ? 0 : item.size;

const isRestoring = (item: TrashedItem): boolean =>
  item.status === FileStatus.restoring ||
  item.status === FolderStatus.restoring;

export const TrashTab: FC<ITrashTabProps> = ({
  items,
  bucket,
  onRestore,
  onPermanentDelete,
}: ITrashTabProps) => {
  const { t } = useTranslation();
  const { isContributor } = useBucketPermissions(bucket.id);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedItem, setSelectedItem] = useState<ISelectedItem | null>(null);

  const openDeleteDialog = (item: ISelectedItem) => {
    setSelectedItem(item);
    setDeleteDialogOpen(true);
  };

  const confirmDelete = () => {
    if (selectedItem) {
      onPermanentDelete(
        selectedItem.id,
        selectedItem.name,
        selectedItem.itemType,
      );
    }
    setDeleteDialogOpen(false);
    setSelectedItem(null);
  };

  const columns = useMemo<Array<IDataTableColumn<TrashedItem>>>(
    () => [
      {
        id: "icon",
        header: "",
        headerClassName: "w-12",
        cell: (item) => (
          <FileIconView
            className="text-primary h-5 w-5 shrink-0"
            isFolder={item.itemType === "folder"}
            extension={"extension" in item ? item.extension : ""}
          />
        ),
      },
      {
        id: "name",
        header: t("bucket.trash_view.name"),
        sortAccessor: (item) => item.name,
        cell: (item) => (
          <span className="truncate font-medium">{item.name}</span>
        ),
      },
      {
        id: "location",
        header: t("bucket.trash_view.original_location"),
        responsive: "lg",
        cellClassName: "text-muted-foreground",
        cell: (item) =>
          item.original_path
            ? item.original_path
            : buildFolderPath(
                "folder_id" in item ? item.folder_id : undefined,
                bucket.folders,
              ),
      },
      {
        id: "size",
        header: t("bucket.trash_view.size"),
        sortAccessor: itemSize,
        responsive: "md",
        cellClassName: "text-muted-foreground",
        cell: (item) =>
          item.itemType === "folder" ? "-" : formatFileSize(itemSize(item)),
      },
      {
        id: "deleted_at",
        header: t("bucket.trash_view.deleted_at"),
        sortAccessor: (item) => new Date(item.deleted_at ?? 0).getTime(),
        responsive: "lg",
        cellClassName: "text-muted-foreground",
        cell: (item) => formatDate(item.deleted_at),
      },
      {
        id: "status",
        header: t("bucket.trash_view.status"),
        responsive: "sm",
        cell: (item) =>
          isRestoring(item) ? (
            <Badge className="rounded-full border-info/20 bg-info/10 text-info">
              <LoaderCircle className="h-3 w-3 animate-spin" />
              {t("bucket.trash_view.restoring")}
            </Badge>
          ) : (
            <Badge className="rounded-full border-warning/20 bg-warning/10 text-warning">
              <Trash2 className="h-3 w-3" />
              {t("bucket.trash_view.trashed")}
            </Badge>
          ),
      },
      {
        id: "actions",
        header: "",
        headerClassName: "w-13",
        cellClassName: "text-right",
        cell: (item) =>
          isContributor && !isRestoring(item) ? (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon" className="h-7 w-7">
                  <EllipsisVertical className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem
                  onClick={() => onRestore(item.id, item.name, item.itemType)}
                >
                  <ArchiveRestore className="mr-2 h-4 w-4" />
                  {t("bucket.trash_view.restore")}
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  className="text-destructive focus:text-destructive"
                  onClick={() =>
                    openDeleteDialog({
                      id: item.id,
                      name: item.name,
                      itemType: item.itemType,
                    })
                  }
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  {t("bucket.trash_view.delete_permanently")}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          ) : null,
      },
    ],
    [t, bucket.folders, isContributor, onRestore],
  );

  return (
    <div className="flex flex-col gap-4">
      <div className="border-border bg-card text-muted-foreground rounded-xl border px-4 py-3 text-sm">
        {t("bucket.trash_view.retention_notice")}
      </div>

      <DataTable
        columns={columns}
        data={items}
        getRowId={(item) => item.id}
        emptyMessage={t("bucket.trash_view.empty")}
        groupOrder={(item) => (item.itemType === "folder" ? 0 : 1)}
        defaultSortId="deleted_at"
        defaultSortDir="desc"
      />

      <CustomAlertDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        destructive
        title={t("bucket.trash_view.confirm_delete_title")}
        description={t("bucket.trash_view.confirm_delete_description", {
          fileName: selectedItem?.name,
        })}
        cancelLabel={t("bucket.trash_view.cancel")}
        confirmLabel={t("bucket.trash_view.delete_permanently")}
        onConfirm={confirmDelete}
      />
    </div>
  );
};
