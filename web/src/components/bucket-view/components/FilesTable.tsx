import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";
import type { BucketItem } from "@/types/bucket.ts";
import { FileIconView } from "@/components/bucket-view/components/FileIconView";
import { FileStatusBadge } from "@/components/bucket-view/components/FileStatusBadge";
import { RowActionsMenu } from "@/components/bucket-view/components/RowActionsMenu";
import { SortableColumnHeader } from "@/components/bucket-view/components/SortableColumnHeader";
import { isFolder } from "@/components/bucket-view/helpers/utils";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { Checkbox } from "@/components/ui/checkbox";
import { cn, formatDate, formatFileSize } from "@/lib/utils";
import { FileStatus } from "@/types/file.ts";

type SortKey = "name" | "size" | "created_at";
type SortDir = "asc" | "desc";

const isSelectable = (item: BucketItem): boolean =>
  isFolder(item) || item.status === FileStatus.uploaded;

interface IFilesTableProps {
  items: Array<BucketItem>;
}

export const FilesTable: FC<IFilesTableProps> = ({
  items,
}: IFilesTableProps) => {
  const { t } = useTranslation();
  const { selected, setSelected, openItem, rowSelection, setRowSelection } =
    useBucketViewContext();
  const [sortKey, setSortKey] = useState<SortKey>("name");
  const [sortDir, setSortDir] = useState<SortDir>("asc");

  const sorted = useMemo(() => {
    const factor = sortDir === "asc" ? 1 : -1;
    return [...items].sort((a, b) => {
      if (isFolder(a) && !isFolder(b)) return -1;
      if (!isFolder(a) && isFolder(b)) return 1;
      let cmp = 0;
      if (sortKey === "size") {
        const sa = isFolder(a) ? 0 : a.size;
        const sb = isFolder(b) ? 0 : b.size;
        cmp = sa - sb;
      } else if (sortKey === "created_at") {
        cmp =
          new Date(a.created_at).getTime() - new Date(b.created_at).getTime();
      } else {
        cmp = a.name.localeCompare(b.name);
      }
      return cmp * factor;
    });
  }, [items, sortKey, sortDir]);

  const selectableIds = useMemo(
    () => sorted.filter(isSelectable).map((i) => i.id),
    [sorted],
  );
  const selectedIds = selectableIds.filter((id) => rowSelection[id]);
  const allSelected =
    selectableIds.length > 0 && selectedIds.length === selectableIds.length;
  const someSelected = selectedIds.length > 0 && !allSelected;

  const toggleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortKey(key);
      setSortDir("asc");
    }
  };

  const toggleAll = (value: boolean) => {
    setRowSelection(() => {
      const next: Record<string, boolean> = {};
      if (value) selectableIds.forEach((id) => (next[id] = true));
      return next;
    });
  };

  const toggleRow = (id: string, value: boolean) => {
    setRowSelection((prev) => {
      const next = { ...(prev as Record<string, boolean>) };
      if (value) next[id] = true;
      else delete next[id];
      return next;
    });
  };

  const cell = "px-4 py-3 align-middle";
  const headCell = "px-4 py-2.5 text-left";

  return (
    <div className="border-border bg-card overflow-hidden rounded-xl border">
      <table className="w-full border-collapse text-sm">
        <thead className="bg-muted">
          <tr>
            <th className={cn(headCell, "w-12")}>
              <Checkbox
                aria-label={t("bucket.bulk_download.select_all")}
                checked={allSelected || (someSelected && "indeterminate")}
                onCheckedChange={(v) => toggleAll(v === true)}
              />
            </th>
            <th className={headCell}>
              <SortableColumnHeader
                label={t("bucket.list_view.name")}
                active={sortKey === "name"}
                dir={sortDir}
                onClick={() => toggleSort("name")}
              />
            </th>
            <th className={cn(headCell, "hidden md:table-cell")}>
              <SortableColumnHeader
                label={t("bucket.list_view.size")}
                active={sortKey === "size"}
                dir={sortDir}
                onClick={() => toggleSort("size")}
              />
            </th>
            <th className={cn(headCell, "hidden lg:table-cell")}>
              <SortableColumnHeader
                label={t("bucket.list_view.uploaded_at")}
                active={sortKey === "created_at"}
                dir={sortDir}
                onClick={() => toggleSort("created_at")}
              />
            </th>
            <th className={cn(headCell, "hidden sm:table-cell")}>
              <span className="text-muted-foreground text-xs font-medium tracking-wide uppercase">
                {t("bucket.list_view.status")}
              </span>
            </th>
            <th className={cn(headCell, "w-13")} />
          </tr>
        </thead>
        <tbody>
          {sorted.length === 0 ? (
            <tr>
              <td
                colSpan={6}
                className="text-muted-foreground h-24 text-center"
              >
                {t("bucket.list_view.empty_folder")}
              </td>
            </tr>
          ) : (
            sorted.map((item) => {
              const itemIsFolder = isFolder(item);
              const isRowSelected = Boolean(rowSelection[item.id]);
              const isActive = selected?.id === item.id;
              return (
                <tr
                  key={item.id}
                  onClick={() => setSelected(item)}
                  onDoubleClick={() => openItem(item)}
                  className={cn(
                    "border-border hover:bg-muted/50 cursor-pointer border-t transition-colors",
                    isActive && "bg-accent/60",
                    isRowSelected && "bg-primary/5",
                  )}
                >
                  <td className={cell} onClick={(e) => e.stopPropagation()}>
                    <Checkbox
                      aria-label={t("bucket.bulk_download.select_row")}
                      checked={isRowSelected}
                      disabled={!isSelectable(item)}
                      onCheckedChange={(v) => toggleRow(item.id, v === true)}
                    />
                  </td>
                  <td className={cell}>
                    <div className="flex items-center gap-2.5 overflow-hidden">
                      <FileIconView
                        className="text-primary h-5 w-5 shrink-0"
                        isFolder={itemIsFolder}
                        extension={!itemIsFolder ? item.extension : undefined}
                      />
                      <span className="truncate font-medium">{item.name}</span>
                    </div>
                  </td>
                  <td
                    className={cn(
                      cell,
                      "text-muted-foreground hidden md:table-cell",
                    )}
                  >
                    {itemIsFolder ? "-" : formatFileSize(item.size)}
                  </td>
                  <td
                    className={cn(
                      cell,
                      "text-muted-foreground hidden lg:table-cell",
                    )}
                  >
                    {formatDate(item.created_at)}
                  </td>
                  <td className={cn(cell, "hidden sm:table-cell")}>
                    <FileStatusBadge item={item} />
                  </td>
                  <td
                    className={cn(cell, "text-right")}
                    onClick={(e) => e.stopPropagation()}
                  >
                    <RowActionsMenu item={item} />
                  </td>
                </tr>
              );
            })
          )}
        </tbody>
      </table>
    </div>
  );
};
