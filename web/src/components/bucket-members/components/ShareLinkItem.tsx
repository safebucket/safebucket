import { Copy, Ellipsis, Lock, Trash2, Upload } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";

import type { IShare } from "@/types/share.ts";
import { useDeleteShareMutation } from "@/queries/bucket.ts";
import { Badge } from "@/components/ui/badge.tsx";
import { Button } from "@/components/ui/button.tsx";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu.tsx";
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
  const deleteShare = useDeleteShareMutation(bucketId);

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
    navigator.clipboard.writeText(`${window.location.origin}/s/${share.id}`);
    successToast(t("bucket.settings.shares.link_copied"));
  };

  const handleCopyId = () => {
    navigator.clipboard.writeText(`${window.location.origin}/s/${share.id}`);
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
              onClick={handleCopyId}
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
    </Item>
  );
};
