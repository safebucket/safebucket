import { Copy, Ellipsis, Lock, QrCode, Trash2, Upload } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { QRCode } from "react-qr-code";
import type { FC } from "react";

import type { IShare } from "@/types/share.ts";
import { useDeleteShareMutation } from "@/queries/bucket.ts";
import { Badge } from "@/components/ui/badge.tsx";
import { Button } from "@/components/ui/button.tsx";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog.tsx";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu.tsx";
import { Input } from "@/components/ui/input.tsx";
import {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemFooter,
  ItemHeader,
  ItemTitle,
} from "@/components/ui/item.tsx";
import { successToast } from "@/components/ui/hooks/use-toast.ts";

interface IShareLinkItemProps {
  share: IShare;
  bucketId: string;
}

export const ShareLinkItem: FC<IShareLinkItemProps> = ({ share, bucketId }) => {
  const { t } = useTranslation();
  const [qrOpen, setQrOpen] = useState(false);
  const deleteShare = useDeleteShareMutation(bucketId);

  const shareUrl = `${window.location.origin}/shares/${share.id}`;

  const viewsText = share.max_views
    ? t("bucket.settings.shares.views", {
        current: share.current_views,
        max: share.max_views,
      })
    : t("bucket.settings.shares.views_unlimited", {
        current: share.current_views,
      });

  let expiryText: string;
  if (!share.expires_at) {
    expiryText = t("bucket.settings.shares.no_expiry");
  } else {
    const date = new Date(share.expires_at);
    expiryText =
      date < new Date()
        ? t("bucket.settings.shares.expired")
        : t("bucket.settings.shares.expires", {
            date: date.toLocaleDateString(),
          });
  }

  const createdText = new Date(share.created_at).toLocaleDateString();

  const handleCopyLink = () => {
    navigator.clipboard.writeText(shareUrl);
    successToast(t("bucket.settings.shares.link_copied"));
  };

  return (
    <Item variant="outline">
      <ItemHeader>
        <ItemContent>
          <ItemTitle>
            {share.name}
            <Badge variant="outline">
              {t(`bucket.settings.shares.type_${share.type}`)}
            </Badge>
          </ItemTitle>
          <ItemDescription>
            <button
              type="button"
              onClick={handleCopyLink}
              className="cursor-pointer font-mono text-xs hover:text-primary"
            >
              {share.id}
            </button>
          </ItemDescription>
        </ItemContent>
        <ItemActions>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon">
                <Ellipsis className="size-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={handleCopyLink}>
                <Copy className="size-4" />
                {t("bucket.settings.shares.copy_link")}
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => setQrOpen(true)}>
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
        </ItemActions>
      </ItemHeader>
      <ItemFooter>
        <div className="text-muted-foreground flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
          <span>{viewsText}</span>
          <span>·</span>
          <span>{expiryText}</span>
          <span>·</span>
          <span>{createdText}</span>
          {share.password_protected && (
            <Badge variant="secondary">
              <Lock className="size-3" />
              {t("bucket.settings.shares.password_protected")}
            </Badge>
          )}
          {share.allow_upload && (
            <Badge variant="secondary">
              <Upload className="size-3" />
              {t("bucket.settings.shares.uploads_allowed")}
            </Badge>
          )}
        </div>
      </ItemFooter>
      <Dialog open={qrOpen} onOpenChange={setQrOpen}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>{t("bucket.settings.shares.qr_title")}</DialogTitle>
            <DialogDescription>
              {t("bucket.settings.shares.qr_description")}
            </DialogDescription>
          </DialogHeader>
          <div className="flex flex-col items-center gap-4">
            <div className="mx-auto w-fit rounded-lg bg-white p-4">
              <QRCode value={shareUrl} size={200} />
            </div>
            <div className="flex w-full items-center gap-2">
              <Input
                readOnly
                value={shareUrl}
                className="bg-muted flex-1 font-mono text-xs"
              />
              <Button
                type="button"
                variant="outline"
                size="icon"
                onClick={handleCopyLink}
                aria-label={t("bucket.settings.shares.copy_link")}
                className="shrink-0"
              >
                <Copy className="size-4" />
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </Item>
  );
};
