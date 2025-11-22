import { useTranslation } from "react-i18next";
import { ArchiveRestore, LoaderCircle, Trash2, Folder } from "lucide-react";
import { type FC, useState, useMemo } from "react";
import type { ColumnDef } from "@tanstack/react-table";

import type { TrashedItem } from "@/components/bucket-view/hooks/useTrashActions";
import type { IBucket } from "@/types/bucket.ts";
import { FileStatus } from "@/types/file.ts";
import { FileIconView } from "@/components/bucket-view/components/FileIconView";
import { formatDate, formatFileSize } from "@/lib/utils";
import { DataTableColumnHeader } from "@/components/common/components/DataTable/DataColumnHeader";
import { DataTable } from "@/components/common/components/DataTable/DataTable";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { MoreHorizontal } from "lucide-react";

interface BucketTrashViewProps {
  items: Array<TrashedItem>;
  bucket: IBucket;
  onRestore: (
    itemId: string,
    itemName: string,
    itemType: "file" | "folder",
  ) => void;
  onPermanentDelete: (
    itemId: string,
    itemName: string,
    itemType: "file" | "folder",
  ) => void;
}

// Helper function to build the original folder path
const buildFolderPath = (
  folderId: string | undefined,
  folders: IBucket["folders"],
): string => {
  if (!folderId) return "/";

  const path: string[] = [];
  let currentId: string | undefined = folderId;

  while (currentId) {
    const folder = folders.find((f) => f.id === currentId);
    if (!folder) break;
    path.unshift(folder.name);
    currentId = folder.folder_id;
  }

  return "/" + path.join("/");
};

const createColumns = (
  t: (key: string) => string,
  bucket: IBucket,
  onRestore: (
    itemId: string,
    itemName: string,
    itemType: "file" | "folder",
  ) => void,
  onOpenDeleteDialog: (
    itemId: string,
    itemName: string,
    itemType: "file" | "folder",
  ) => void,
): Array<ColumnDef<TrashedItem>> => [
  {
    id: "type",
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={t("bucket.trash_view.type")}
      />
    ),
    cell: ({ row }) => {
      const item = row.original;
      if (item.itemType === "folder") {
        return <Folder className="h-5 w-5 text-primary opacity-50" />;
      } else {
        return (
          <FileIconView
            className="text-primary h-5 w-5 opacity-50"
            isFolder={false}
            extension={"extension" in item ? item.extension : ""}
          />
        );
      }
    },
  },
  {
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={t("bucket.trash_view.name")}
      />
    ),
    cell: ({ row }) => (
      <div className="flex w-[300px] items-center space-x-2">
        <p className="opacity-70">{row.getValue("name")}</p>
      </div>
    ),
  },
  {
    id: "original_location",
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={t("bucket.trash_view.original_location")}
      />
    ),
    cell: ({ row }) => {
      const item = row.original;
      const folderId = "folder_id" in item ? item.folder_id : undefined;
      const path = buildFolderPath(folderId, bucket.folders);
      return <span className="text-sm text-muted-foreground">{path}</span>;
    },
  },
  {
    accessorKey: "size",
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={t("bucket.trash_view.size")}
      />
    ),
    cell: ({ row }) => {
      const item = row.original;
      if (item.itemType === "folder") {
        return "-";
      }
      return "size" in item ? formatFileSize(item.size) : "-";
    },
  },
  {
    accessorKey: "trashed_at",
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={t("bucket.trash_view.deleted_at")}
      />
    ),
    cell: ({ row }) => formatDate(row.getValue("trashed_at")),
  },
  {
    accessorKey: "trashed_user",
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={t("bucket.trash_view.deleted_by")}
      />
    ),
    cell: ({ row }) => {
      const user = row.original.trashed_user;
      if (!user) return "-";
      return user.first_name && user.last_name
        ? `${user.first_name} ${user.last_name}`
        : user.email;
    },
  },
  {
    accessorKey: "status",
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={t("bucket.trash_view.status")}
      />
    ),
    cell: ({ row }) => {
      const status = row.getValue("status");

      switch (status) {
        case FileStatus.trashed:
          return (
            <Badge className="bg-orange-100 text-orange-800 border-orange-200 dark:bg-orange-900/20 dark:text-orange-300 dark:border-orange-800">
              <Trash2 className="h-3 w-3" />
              {t("bucket.trash_view.trashed")}
            </Badge>
          );
        case FileStatus.restoring:
          return (
            <Badge className="bg-blue-100 text-blue-800 border-blue-200 dark:bg-blue-900/20 dark:text-blue-300 dark:border-blue-800">
              <LoaderCircle className="h-3 w-3 animate-spin" />
              {t("bucket.trash_view.restoring")}
            </Badge>
          );
        default:
          return "-";
      }
    },
  },
  {
    id: "actions",
    cell: ({ row }) => {
      const item = row.original;
      return (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              className="flex h-8 w-8 p-0 data-[state=open]:bg-muted"
            >
              <MoreHorizontal className="h-4 w-4" />
              <span className="sr-only">Open menu</span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-[160px]">
            <DropdownMenuItem
              onClick={() => onRestore(item.id, item.name, item.itemType)}
            >
              <ArchiveRestore className="mr-2 h-4 w-4" />
              {t("bucket.trash_view.restore")}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              className="text-red-600"
              onClick={() =>
                onOpenDeleteDialog(item.id, item.name, item.itemType)
              }
            >
              <Trash2 className="mr-2 h-4 w-4" />
              {t("bucket.trash_view.delete_permanently")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      );
    },
  },
];

export const BucketTrashView: FC<BucketTrashViewProps> = ({
  items,
  bucket,
  onRestore,
  onPermanentDelete,
}: BucketTrashViewProps) => {
  const { t } = useTranslation();
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedItem, setSelectedItem] = useState<{
    id: string;
    name: string;
    itemType: "file" | "folder";
  } | null>(null);

  const handleOpenDeleteDialog = (
    itemId: string,
    itemName: string,
    itemType: "file" | "folder",
  ) => {
    setSelectedItem({ id: itemId, name: itemName, itemType });
    setDeleteDialogOpen(true);
  };

  const handleConfirmDelete = () => {
    if (selectedItem) {
      onPermanentDelete(
        selectedItem.id,
        selectedItem.name,
        selectedItem.itemType,
      );
      setDeleteDialogOpen(false);
      setSelectedItem(null);
    }
  };

  const columns = useMemo(
    () => createColumns(t, bucket, onRestore, handleOpenDeleteDialog),
    [t, bucket, onRestore],
  );

  if (items.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-[400px] text-muted-foreground">
        <Trash2 className="h-16 w-16 mb-4 opacity-20" />
        <p className="text-lg font-medium">{t("bucket.trash_view.empty")}</p>
        <p className="text-sm mt-2">
          {t("bucket.trash_view.empty_description")}
        </p>
      </div>
    );
  }

  return (
    <>
      <div className="space-y-4">
        <div className="bg-muted/50 p-4 rounded-lg border">
          <p className="text-sm text-muted-foreground">
            {t("bucket.trash_view.retention_notice")}
          </p>
        </div>
        <DataTable
          columns={columns}
          data={items}
          selected={null}
          onRowClick={() => {}}
          onRowDoubleClick={() => {}}
          trashMode={true}
          onRestore={(itemId: string, itemName: string) => {
            const item = items.find((i) => i.id === itemId);
            if (item) {
              onRestore(itemId, itemName, item.itemType);
            }
          }}
          onPermanentDelete={(itemId: string, itemName: string) => {
            const item = items.find((i) => i.id === itemId);
            if (item) {
              handleOpenDeleteDialog(itemId, itemName, item.itemType);
            }
          }}
        />
      </div>

      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {t("bucket.trash_view.confirm_delete_title")}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t("bucket.trash_view.confirm_delete_description", {
                fileName: selectedItem?.name,
              })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>
              {t("bucket.trash_view.cancel")}
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={handleConfirmDelete}
              className="bg-red-600 hover:bg-red-700 focus:ring-red-600"
            >
              {t("bucket.trash_view.delete_permanently")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
};
