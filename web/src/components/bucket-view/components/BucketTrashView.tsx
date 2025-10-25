import { useTranslation } from "react-i18next";
import { ArchiveRestore, LoaderCircle, Trash2 } from "lucide-react";
import { type FC, useState } from "react";
import type { ColumnDef } from "@tanstack/react-table";

import type { IFile } from "@/types/file.ts";
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
  files: Array<IFile>;
  onRestore: (fileId: string, fileName: string) => void;
  onPermanentDelete: (fileId: string, fileName: string) => void;
}

const createColumns = (
  t: (key: string) => string,
  onRestore: (fileId: string, fileName: string) => void,
  onOpenDeleteDialog: (fileId: string, fileName: string) => void,
): Array<ColumnDef<IFile>> => [
  {
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={t("bucket.trash_view.name")}
      />
    ),
    cell: ({ row }) => (
      <div className="flex w-[350px] items-center space-x-2">
        <FileIconView
          className="text-primary h-5 w-5 opacity-50"
          type={row.original.type}
          extension={row.original.extension}
        />
        <p className="opacity-70">{row.getValue("name")}</p>
      </div>
    ),
  },
  {
    accessorKey: "size",
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={t("bucket.trash_view.size")}
      />
    ),
    cell: ({ row }) => formatFileSize(row.getValue("size")),
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
    cell: ({ row }) => (
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
            onClick={() => onRestore(row.original.id, row.original.name)}
          >
            <ArchiveRestore className="mr-2 h-4 w-4" />
            {t("bucket.trash_view.restore")}
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            className="text-red-600"
            onClick={() =>
              onOpenDeleteDialog(row.original.id, row.original.name)
            }
          >
            <Trash2 className="mr-2 h-4 w-4" />
            {t("bucket.trash_view.delete_permanently")}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    ),
  },
];

export const BucketTrashView: FC<BucketTrashViewProps> = ({
  files,
  onRestore,
  onPermanentDelete,
}: BucketTrashViewProps) => {
  const { t } = useTranslation();
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedFile, setSelectedFile] = useState<{ id: string; name: string } | null>(null);

  const handleOpenDeleteDialog = (fileId: string, fileName: string) => {
    setSelectedFile({ id: fileId, name: fileName });
    setDeleteDialogOpen(true);
  };

  const handleConfirmDelete = () => {
    if (selectedFile) {
      onPermanentDelete(selectedFile.id, selectedFile.name);
      setDeleteDialogOpen(false);
      setSelectedFile(null);
    }
  };

  const columns = createColumns(t, onRestore, handleOpenDeleteDialog);

  if (files.length === 0) {
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
          data={files}
          selected={null}
          onRowClick={() => {}}
          onRowDoubleClick={() => {}}
          trashMode={true}
          onRestore={onRestore}
          onPermanentDelete={onPermanentDelete}
        />
      </div>

      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("bucket.trash_view.confirm_delete_title")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("bucket.trash_view.confirm_delete_description", { fileName: selectedFile?.name })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("bucket.trash_view.cancel")}</AlertDialogCancel>
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
