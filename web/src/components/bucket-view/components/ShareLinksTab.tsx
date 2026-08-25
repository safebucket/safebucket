import {
  Copy,
  Ellipsis,
  Lock,
  Plus,
  QrCode,
  Trash2,
  Upload,
} from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import type { FC } from "react";
import type { IDataTableColumn } from "@/components/common/components/DataTable/DataTable";
import type { IShare } from "@/types/share.ts";
import type { IBucket } from "@/types/bucket.ts";
import {
  bucketSharesQueryOptions,
  useDeleteShareMutation,
} from "@/queries/bucket.ts";
import { DataTable } from "@/components/common/components/DataTable/DataTable";
import { GridActionIcon } from "@/components/bucket-view/components/GridActionIcon";
import { QrCodeDialog } from "@/components/common/components/QrCodeDialog.tsx";
import { QuickShareDialog } from "@/components/quick-share/QuickShareDialog.tsx";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

interface IShareLinksTabProps {
  bucket: IBucket;
}

export const ShareLinksTab: FC<IShareLinksTabProps> = ({
  bucket,
}: IShareLinksTabProps) => {
  const { t } = useTranslation();
  const [createOpen, setCreateOpen] = useState(false);
  const [qrUrl, setQrUrl] = useState<string | null>(null);
  const deleteShare = useDeleteShareMutation(bucket.id);
  const { data: shares, isLoading } = useQuery(
    bucketSharesQueryOptions(bucket.id),
  );

  const shareUrl = (share: IShare) =>
    `${window.location.origin}/shares/${share.id}`;

  const copyLink = (share: IShare) => {
    navigator.clipboard.writeText(shareUrl(share));
    toast.success(t("bucket.settings.shares.link_copied"));
  };

  const expiryText = (share: IShare) => {
    if (!share.expires_at) return t("bucket.settings.shares.no_expiry");
    const date = new Date(share.expires_at);
    return date < new Date()
      ? t("bucket.settings.shares.expired")
      : t("bucket.settings.shares.expires", {
          date: date.toLocaleDateString(),
        });
  };

  const viewsText = (share: IShare) =>
    share.max_views
      ? t("bucket.settings.shares.views", {
          current: share.current_views,
          max: share.max_views,
        })
      : t("bucket.settings.shares.views_unlimited", {
          current: share.current_views,
        });

  const columns = useMemo<Array<IDataTableColumn<IShare>>>(
    () => [
      {
        id: "name",
        header: t("bucket.view.shares.name"),
        cell: (share) => (
          <div className="flex min-w-0 flex-col gap-1">
            <span className="truncate font-medium">{share.name}</span>
            <div className="flex flex-wrap items-center gap-1.5">
              {share.password_protected ? (
                <Badge variant="secondary" className="rounded-full">
                  <Lock className="size-3" />
                  {t("bucket.settings.shares.password_protected")}
                </Badge>
              ) : null}
              {share.allow_upload ? (
                <Badge variant="secondary" className="rounded-full">
                  <Upload className="size-3" />
                  {t("bucket.settings.shares.uploads_allowed")}
                </Badge>
              ) : null}
            </div>
          </div>
        ),
      },
      {
        id: "type",
        header: t("bucket.list_view.type"),
        cell: (share) => (
          <Badge variant="outline" className="rounded-full">
            {t(`bucket.settings.shares.type_${share.type}`)}
          </Badge>
        ),
      },
      {
        id: "link",
        header: t("bucket.view.shares.link"),
        responsive: "lg",
        cell: (share) => (
          <button
            type="button"
            onClick={() => copyLink(share)}
            className="hover:text-primary cursor-pointer font-mono text-xs"
          >
            {share.id}
          </button>
        ),
      },
      {
        id: "views",
        header: t("bucket.view.shares.views"),
        responsive: "md",
        cellClassName: "text-muted-foreground text-xs",
        cell: (share) => viewsText(share),
      },
      {
        id: "expiry",
        header: t("bucket.view.shares.expiry"),
        responsive: "md",
        cellClassName: "text-muted-foreground text-xs",
        cell: (share) => expiryText(share),
      },
      {
        id: "actions",
        header: "",
        headerClassName: "w-13",
        cellClassName: "text-right",
        cell: (share) => (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" className="h-7 w-7">
                <Ellipsis className="size-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => copyLink(share)}>
                <Copy className="size-4" />
                {t("bucket.settings.shares.copy_link")}
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => setQrUrl(shareUrl(share))}>
                <QrCode className="size-4" />
                {t("bucket.settings.shares.show_qr")}
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => deleteShare.mutate(share.id)}
                disabled={deleteShare.isPending}
                className="text-destructive focus:text-destructive"
              >
                <Trash2 className="size-4" />
                {t("bucket.settings.shares.delete")}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        ),
      },
    ],
    [t, deleteShare],
  );

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-end">
        <GridActionIcon
          icon={<Plus className="h-4 w-4" />}
          label={t("bucket.settings.shares.create")}
          onClick={() => setCreateOpen(true)}
        />
      </div>

      <DataTable
        columns={columns}
        data={shares ?? []}
        getRowId={(share) => share.id}
        emptyMessage={t("bucket.settings.shares.no_shares")}
        isLoading={isLoading}
      />

      <QuickShareDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        bucketId={bucket.id}
      />
      <QrCodeDialog
        open={qrUrl !== null}
        onOpenChange={(open) => {
          if (!open) setQrUrl(null);
        }}
        value={qrUrl ?? ""}
        title={t("bucket.settings.shares.qr_title")}
        description={t("bucket.settings.shares.qr_description")}
      />
    </div>
  );
};
