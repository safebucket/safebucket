import { useCallback, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";
import type { OnChangeFn, RowSelectionState } from "@tanstack/react-table";
import type { IDataTableColumn } from "@/components/common/components/DataTable/DataTable";
import type { IDataTableRowProps } from "@/components/common/components/DataTable/DataTableRow";
import type { BucketItem } from "@/types/bucket.ts";
import { DataTable } from "@/components/common/components/DataTable/DataTable";
import { BucketItemRow } from "@/components/bucket-view/components/BucketItemRow";
import { FileIconView } from "@/components/bucket-view/components/FileIconView";
import { FileStatusBadge } from "@/components/bucket-view/components/FileStatusBadge";
import { RowActionsMenu } from "@/components/bucket-view/components/RowActionsMenu";
import { isFolder } from "@/components/bucket-view/helpers/utils";
import { useOpenBucketItem } from "@/components/bucket-view/hooks/useOpenBucketItem";
import { formatDate, formatFileSize } from "@/lib/utils";
import { FileStatus } from "@/types/file.ts";

const isSelectable = (item: BucketItem): boolean =>
  isFolder(item) || item.status === FileStatus.uploaded;

interface IFilesTableProps {
  items: Array<BucketItem>;
  selected: BucketItem | null;
  setSelected: (item: BucketItem) => void;
  rowSelection: RowSelectionState;
  setRowSelection: OnChangeFn<RowSelectionState>;
  enabled: boolean;
  selectedIds: Array<string>;
}

export const FilesTable: FC<IFilesTableProps> = ({
  items,
  selected,
  setSelected,
  rowSelection,
  setRowSelection,
  enabled,
  selectedIds,
}: IFilesTableProps) => {
  const { t } = useTranslation();
  const openItem = useOpenBucketItem();

  const selectedIdsRef = useRef(selectedIds);
  selectedIdsRef.current = selectedIds;
  const getSelectedIds = useCallback(() => selectedIdsRef.current, []);

  const rowComponent = useMemo(
    () =>
      function BucketItemRowBound(props: IDataTableRowProps<BucketItem>) {
        return (
          <BucketItemRow
            {...props}
            enabled={enabled}
            getSelectedIds={getSelectedIds}
          />
        );
      },
    [enabled, getSelectedIds],
  );

  const columns = useMemo<Array<IDataTableColumn<BucketItem>>>(
    () => [
      {
        id: "name",
        header: t("bucket.list_view.name"),
        sortAccessor: (item) => item.name,
        cell: (item) => (
          <div className="flex items-center gap-2.5 overflow-hidden">
            <FileIconView
              className="text-primary h-5 w-5 shrink-0"
              isFolder={isFolder(item)}
              extension={!isFolder(item) ? item.extension : undefined}
            />
            <span className="truncate font-medium">{item.name}</span>
          </div>
        ),
      },
      {
        id: "size",
        header: t("bucket.list_view.size"),
        sortAccessor: (item) => (isFolder(item) ? 0 : item.size),
        responsive: "md",
        cellClassName: "text-muted-foreground",
        cell: (item) => (isFolder(item) ? "-" : formatFileSize(item.size)),
      },
      {
        id: "created_at",
        header: t("bucket.list_view.uploaded_at"),
        sortAccessor: (item) => new Date(item.created_at).getTime(),
        responsive: "lg",
        cellClassName: "text-muted-foreground",
        cell: (item) => formatDate(item.created_at),
      },
      {
        id: "status",
        header: t("bucket.list_view.status"),
        responsive: "sm",
        cell: (item) => <FileStatusBadge item={item} />,
      },
      {
        id: "actions",
        header: "",
        headerClassName: "w-13",
        cellClassName: "text-right",
        cell: (item) => <RowActionsMenu item={item} />,
      },
    ],
    [t],
  );

  return (
    <DataTable
      columns={columns}
      data={items}
      getRowId={(item) => item.id}
      emptyMessage={t("bucket.list_view.empty_folder")}
      groupOrder={(item) => (isFolder(item) ? 0 : 1)}
      defaultSortId="name"
      selection={{
        rowSelection,
        onRowSelectionChange: setRowSelection,
        getIsSelectable: isSelectable,
      }}
      onRowClick={setSelected}
      onRowDoubleClick={openItem}
      isRowActive={(item) => selected?.id === item.id}
      rowComponent={rowComponent}
    />
  );
};
