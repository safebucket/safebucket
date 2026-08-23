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
import type { FC } from "react";
import type { IShare } from "@/types/share.ts";
import type { IBucket } from "@/types/bucket.ts";
import {
  bucketSharesQueryOptions,
  useDeleteShareMutation,
} from "@/queries/bucket.ts";
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
import { Skeleton } from "@/components/ui/skeleton";
import { successToast } from "@/components/ui/hooks/use-toast";
import { cn } from "@/lib/utils";

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

  const shareList = useMemo(() => shares ?? [], [shares]);

  const cell = "px-4 py-3 align-middle";
  const headCell =
    "px-4 py-2.5 text-left text-xs font-medium tracking-wide text-muted-foreground uppercase";

  const shareUrl = (share: IShare) =>
    `${window.location.origin}/shares/${share.id}`;

  const copyLink = (share: IShare) => {
    navigator.clipboard.writeText(shareUrl(share));
    successToast(t("bucket.settings.shares.link_copied"));
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

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-end">
        <GridActionIcon
          icon={<Plus className="h-4 w-4" />}
          label={t("bucket.settings.shares.create")}
          onClick={() => setCreateOpen(true)}
        />
      </div>

      <div className="border-border bg-card overflow-hidden rounded-xl border">
        <table className="w-full border-collapse text-sm">
          <thead className="bg-muted">
            <tr>
              <th className={headCell}>{t("bucket.view.shares.name")}</th>
              <th className={headCell}>{t("bucket.list_view.type")}</th>
              <th className={cn(headCell, "hidden lg:table-cell")}>
                {t("bucket.view.shares.link")}
              </th>
              <th className={cn(headCell, "hidden md:table-cell")}>
                {t("bucket.view.shares.views")}
              </th>
              <th className={cn(headCell, "hidden md:table-cell")}>
                {t("bucket.view.shares.expiry")}
              </th>
              <th className={cn(headCell, "w-13")} />
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              [1, 2, 3].map((i) => (
                <tr key={i} className="border-border border-t">
                  <td className={cell} colSpan={6}>
                    <Skeleton className="h-6 w-full" />
                  </td>
                </tr>
              ))
            ) : shareList.length > 0 ? (
              shareList.map((share) => (
                <tr
                  key={share.id}
                  className="border-border hover:bg-muted/50 border-t transition-colors"
                >
                  <td className={cell}>
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
                  </td>
                  <td className={cell}>
                    <Badge variant="outline" className="rounded-full">
                      {t(`bucket.settings.shares.type_${share.type}`)}
                    </Badge>
                  </td>
                  <td className={cn(cell, "hidden lg:table-cell")}>
                    <button
                      type="button"
                      onClick={() => copyLink(share)}
                      className="hover:text-primary cursor-pointer font-mono text-xs"
                    >
                      {share.id}
                    </button>
                  </td>
                  <td
                    className={cn(
                      cell,
                      "text-muted-foreground hidden text-xs md:table-cell",
                    )}
                  >
                    {viewsText(share)}
                  </td>
                  <td
                    className={cn(
                      cell,
                      "text-muted-foreground hidden text-xs md:table-cell",
                    )}
                  >
                    {expiryText(share)}
                  </td>
                  <td className={cn(cell, "text-right")}>
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
                        <DropdownMenuItem
                          onClick={() => setQrUrl(shareUrl(share))}
                        >
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
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td
                  colSpan={6}
                  className="text-muted-foreground h-24 text-center"
                >
                  {t("bucket.settings.shares.no_shares")}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

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
