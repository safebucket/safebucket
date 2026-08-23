import {
  ArchiveRestore,
  EllipsisVertical,
  LoaderCircle,
  Trash2,
} from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";
import type { TrashedItem } from "@/components/bucket-view/hooks/useTrashActions";
import type { IBucket } from "@/types/bucket.ts";
import { FileIconView } from "@/components/bucket-view/components/FileIconView";
import { SortableColumnHeader } from "@/components/bucket-view/components/SortableColumnHeader";
import { useBucketPermissions } from "@/hooks/usePermissions";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn, formatDate, formatFileSize } from "@/lib/utils";
import { FileStatus } from "@/types/file.ts";
import { FolderStatus } from "@/types/folder.ts";

type SortKey = "name" | "size" | "deleted_at";
type SortDir = "asc" | "desc";

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

export const TrashTab: FC<ITrashTabProps> = ({
  items,
  bucket,
  onRestore,
  onPermanentDelete,
}: ITrashTabProps) => {
  const { t } = useTranslation();
  const { isContributor } = useBucketPermissions(bucket.id);
  const [sortKey, setSortKey] = useState<SortKey>("deleted_at");
  const [sortDir, setSortDir] = useState<SortDir>("desc");
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedItem, setSelectedItem] = useState<ISelectedItem | null>(null);

  const sorted = useMemo(() => {
    const factor = sortDir === "asc" ? 1 : -1;
    return [...items].sort((a, b) => {
      if (a.itemType === "folder" && b.itemType !== "folder") return -1;
      if (a.itemType !== "folder" && b.itemType === "folder") return 1;
      let cmp = 0;
      if (sortKey === "size") {
        cmp = itemSize(a) - itemSize(b);
      } else if (sortKey === "deleted_at") {
        cmp =
          new Date(a.deleted_at ?? 0).getTime() -
          new Date(b.deleted_at ?? 0).getTime();
      } else {
        cmp = a.name.localeCompare(b.name);
      }
      return cmp * factor;
    });
  }, [items, sortKey, sortDir]);

  const toggleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortKey(key);
      setSortDir("asc");
    }
  };

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

  const cell = "px-4 py-3 align-middle";
  const headCell = "px-4 py-2.5 text-left";

  return (
    <div className="flex flex-col gap-4">
      <div className="border-border bg-card text-muted-foreground rounded-xl border px-4 py-3 text-sm">
        {t("bucket.trash_view.retention_notice")}
      </div>

      <div className="border-border bg-card overflow-hidden rounded-xl border">
        <table className="w-full border-collapse text-sm">
          <thead className="bg-muted">
            <tr>
              <th className={cn(headCell, "w-12")} />
              <th className={headCell}>
                <SortableColumnHeader
                  label={t("bucket.trash_view.name")}
                  active={sortKey === "name"}
                  dir={sortDir}
                  onClick={() => toggleSort("name")}
                />
              </th>
              <th className={cn(headCell, "hidden lg:table-cell")}>
                <span className="text-muted-foreground text-xs font-medium tracking-wide uppercase">
                  {t("bucket.trash_view.original_location")}
                </span>
              </th>
              <th className={cn(headCell, "hidden md:table-cell")}>
                <SortableColumnHeader
                  label={t("bucket.trash_view.size")}
                  active={sortKey === "size"}
                  dir={sortDir}
                  onClick={() => toggleSort("size")}
                />
              </th>
              <th className={cn(headCell, "hidden lg:table-cell")}>
                <SortableColumnHeader
                  label={t("bucket.trash_view.deleted_at")}
                  active={sortKey === "deleted_at"}
                  dir={sortDir}
                  onClick={() => toggleSort("deleted_at")}
                />
              </th>
              <th className={cn(headCell, "hidden sm:table-cell")}>
                <span className="text-muted-foreground text-xs font-medium tracking-wide uppercase">
                  {t("bucket.trash_view.status")}
                </span>
              </th>
              <th className={cn(headCell, "w-13")} />
            </tr>
          </thead>
          <tbody>
            {sorted.length === 0 ? (
              <tr>
                <td
                  colSpan={7}
                  className="text-muted-foreground h-24 text-center"
                >
                  {t("bucket.trash_view.empty")}
                </td>
              </tr>
            ) : (
              sorted.map((item) => {
                const isFolder = item.itemType === "folder";
                const path = item.original_path
                  ? item.original_path
                  : buildFolderPath(
                      "folder_id" in item ? item.folder_id : undefined,
                      bucket.folders,
                    );
                const restoring =
                  item.status === FileStatus.restoring ||
                  item.status === FolderStatus.restoring;
                return (
                  <tr
                    key={item.id}
                    className="border-border hover:bg-muted/50 border-t transition-colors"
                  >
                    <td className={cell}>
                      <FileIconView
                        className="text-primary h-5 w-5 shrink-0"
                        isFolder={isFolder}
                        extension={"extension" in item ? item.extension : ""}
                      />
                    </td>
                    <td className={cell}>
                      <span className="truncate font-medium">{item.name}</span>
                    </td>
                    <td
                      className={cn(
                        cell,
                        "text-muted-foreground hidden lg:table-cell",
                      )}
                    >
                      {path}
                    </td>
                    <td
                      className={cn(
                        cell,
                        "text-muted-foreground hidden md:table-cell",
                      )}
                    >
                      {isFolder ? "-" : formatFileSize(itemSize(item))}
                    </td>
                    <td
                      className={cn(
                        cell,
                        "text-muted-foreground hidden lg:table-cell",
                      )}
                    >
                      {formatDate(item.deleted_at)}
                    </td>
                    <td className={cn(cell, "hidden sm:table-cell")}>
                      {restoring ? (
                        <Badge className="rounded-full border-blue-200 bg-blue-100 text-blue-800 dark:border-blue-800 dark:bg-blue-900/20 dark:text-blue-300">
                          <LoaderCircle className="h-3 w-3 animate-spin" />
                          {t("bucket.trash_view.restoring")}
                        </Badge>
                      ) : (
                        <Badge className="rounded-full border-orange-200 bg-orange-100 text-orange-800 dark:border-orange-800 dark:bg-orange-900/20 dark:text-orange-300">
                          <Trash2 className="h-3 w-3" />
                          {t("bucket.trash_view.trashed")}
                        </Badge>
                      )}
                    </td>
                    <td className={cn(cell, "text-right")}>
                      {isContributor ? (
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button
                              variant="ghost"
                              size="icon"
                              className="h-7 w-7"
                            >
                              <EllipsisVertical className="h-4 w-4" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            <DropdownMenuItem
                              onClick={() =>
                                onRestore(item.id, item.name, item.itemType)
                              }
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
                      ) : null}
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
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
              onClick={confirmDelete}
              className="bg-destructive hover:bg-destructive/90 focus:ring-destructive text-white"
            >
              {t("bucket.trash_view.delete_permanently")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};
