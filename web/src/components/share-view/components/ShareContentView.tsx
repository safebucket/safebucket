import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  ArrowLeft,
  Calendar,
  Eye,
  FolderOpen,
  LayoutGrid,
  LayoutList,
  Lock,
  Upload,
} from "lucide-react";
import type { ColumnDef, VisibilityState } from "@tanstack/react-table";
import type { FC } from "react";

import type { IPublicShareContent, IShare } from "@/types/share.ts";
import type { BucketItem } from "@/types/bucket.ts";
import { isFolder } from "@/components/bucket-view/helpers/utils.ts";
import { FileIconView } from "@/components/bucket-view/components/FileIconView.tsx";
import { DataTableColumnHeader } from "@/components/common/components/DataTable/DataColumnHeader.tsx";
import { ShareListView } from "@/components/share-view/components/ShareListView.tsx";
import { ShareFileGridCard } from "@/components/share-view/components/ShareFileGridCard.tsx";
import { Badge } from "@/components/ui/badge.tsx";
import { Button } from "@/components/ui/button.tsx";
import { useIsMobile } from "@/components/ui/hooks/use-mobile.tsx";
import { formatDate, formatFileSize } from "@/lib/utils.ts";

type ViewMode = "list" | "grid";

interface IShareContentViewProps {
  share: IShare;
  content: IPublicShareContent;
}

const createColumns = (
  t: (key: string) => string,
): Array<ColumnDef<BucketItem>> => [
  {
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t("share_consumer.name")} />
    ),
    size: 350,
    cell: ({ row }) => {
      const item = row.original;
      const itemIsFolder = isFolder(item);
      return (
        <div className="flex items-center space-x-2 overflow-hidden max-w-[calc(100vw-8rem)] md:max-w-87.5">
          <FileIconView
            className="text-primary h-5 w-5 shrink-0"
            isFolder={itemIsFolder}
            extension={!itemIsFolder ? item.extension : undefined}
          />
          <p className="truncate">{row.getValue("name")}</p>
        </div>
      );
    },
  },
  {
    accessorKey: "size",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t("share_consumer.size")} />
    ),
    cell: ({ row }) => {
      const item = row.original;
      return isFolder(item) ? "-" : formatFileSize(item.size);
    },
  },
  {
    id: "type",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t("share_consumer.type")} />
    ),
    cell: ({ row }) => {
      const item = row.original;
      const itemType = isFolder(item) ? "folder" : item.extension;
      return <Badge variant="secondary">{itemType}</Badge>;
    },
  },
  {
    accessorKey: "created_at",
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={t("share_consumer.uploaded_at")}
      />
    ),
    cell: ({ row }) => formatDate(row.getValue("created_at")),
  },
];

export const ShareContentView: FC<IShareContentViewProps> = ({
  share,
  content,
}) => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [viewMode, setViewMode] = useState<ViewMode>("list");
  const [currentFolderId, setCurrentFolderId] = useState<string | undefined>(
    share.type === "folder" ? (share.folder_id ?? undefined) : undefined,
  );
  const [folderHistory, setFolderHistory] = useState<Array<string>>([]);

  const columns = useMemo(() => createColumns(t), [t]);

  const columnVisibility = useMemo(
    (): VisibilityState =>
      isMobile
        ? { size: false, type: false, created_at: false }
        : ({} as VisibilityState),
    [isMobile],
  );

  const items = useMemo((): Array<BucketItem> => {
    const folderItems = content.folders.filter(
      (folder) =>
        (!currentFolderId && !folder.folder_id) ||
        folder.folder_id === currentFolderId,
    );
    const fileItems = content.files.filter(
      (file) =>
        (!currentFolderId && !file.folder_id) ||
        file.folder_id === currentFolderId,
    );
    return [...folderItems, ...fileItems];
  }, [content, currentFolderId]);

  const openFolder = (item: BucketItem) => {
    if (isFolder(item)) {
      setFolderHistory((prev) => [...prev, currentFolderId ?? ""]);
      setCurrentFolderId(item.id);
    }
  };

  const goBack = () => {
    const prev = folderHistory[folderHistory.length - 1];
    setFolderHistory((h) => h.slice(0, -1));
    setCurrentFolderId(prev || undefined);
  };

  const currentFolderName = currentFolderId
    ? content.folders.find((f) => f.id === currentFolderId)?.name
    : null;

  const viewsText = share.max_views
    ? t("share_consumer.views", {
        current: share.current_views,
        max: share.max_views,
      })
    : t("share_consumer.views_unlimited", {
        current: share.current_views,
      });

  let expiryText: string;
  if (!share.expires_at) {
    expiryText = t("share_consumer.no_expiry");
  } else {
    const date = new Date(share.expires_at);
    expiryText =
      date < new Date()
        ? t("share_consumer.expired")
        : t("share_consumer.expires", { date: date.toLocaleDateString() });
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="border-b bg-background">
        <div className="mx-auto max-w-6xl px-6 py-4">
          <div className="flex items-center justify-between">
            <div className="min-w-0">
              <h1 className="truncate text-xl font-bold md:text-2xl">
                {share.name}
              </h1>
              <div className="text-muted-foreground mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
                <Badge variant="outline">
                  {t(`share_consumer.type_${share.type}`)}
                </Badge>
                <span className="flex items-center gap-1">
                  <Eye className="h-3 w-3" />
                  {viewsText}
                </span>
                <span className="flex items-center gap-1">
                  <Calendar className="h-3 w-3" />
                  {expiryText}
                </span>
                {share.password_protected && (
                  <Badge variant="secondary" className="gap-1">
                    <Lock className="h-3 w-3" />
                    {t("share_consumer.password_protected")}
                  </Badge>
                )}
                {share.allow_upload && (
                  <Badge variant="secondary" className="gap-1">
                    <Upload className="h-3 w-3" />
                    {t("share_consumer.uploads_allowed")}
                  </Badge>
                )}
              </div>
            </div>

            <div className="flex items-center gap-2">
              <Button
                variant={viewMode === "list" ? "default" : "outline"}
                size="icon"
                onClick={() => setViewMode("list")}
                title={t("share_consumer.list_view")}
              >
                <LayoutList className="h-4 w-4" />
              </Button>
              <Button
                variant={viewMode === "grid" ? "default" : "outline"}
                size="icon"
                onClick={() => setViewMode("grid")}
                title={t("share_consumer.grid_view")}
              >
                <LayoutGrid className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </div>
      </div>

      <div className="mx-auto w-full max-w-6xl flex-1 overflow-y-auto px-6 py-6">
        {folderHistory.length > 0 && (
          <button
            type="button"
            onClick={goBack}
            className="mb-4 flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground"
          >
            <ArrowLeft className="h-4 w-4" />
            {currentFolderName ?? t("share_consumer.back")}
          </button>
        )}

        {items.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-center">
            <FolderOpen className="text-muted-foreground mb-4 h-16 w-16" />
            <p className="text-muted-foreground text-lg">
              {t("share_consumer.empty")}
            </p>
          </div>
        ) : viewMode === "list" ? (
          <ShareListView
            columns={columns}
            data={items}
            onRowDoubleClick={openFolder}
            defaultColumnVisibility={columnVisibility}
          />
        ) : (
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5">
            {items.map((item) => (
              <ShareFileGridCard
                key={item.id}
                file={item}
                onDoubleClick={openFolder}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
};
