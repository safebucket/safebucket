import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Calendar,
  Check,
  Copy,
  Download,
  Eye,
  Loader2,
  Lock,
  Upload,
} from "lucide-react";
import { toast } from "sonner";
import type { FC } from "react";

import type { IPublicShareResponse } from "@/types/share.ts";
import { Button } from "@/components/ui/button.tsx";
import { Avatar, AvatarFallback } from "@/components/ui/avatar.tsx";
import { formatFileSize } from "@/lib/utils.ts";

interface IShareHeaderProps {
  shareContent: IPublicShareResponse;
  totalSize: number;
  isDownloadingAll: boolean;
  onDownloadAll: () => void;
}

export const ShareHeader: FC<IShareHeaderProps> = ({
  shareContent,
  totalSize,
  isDownloadingAll,
  onDownloadAll,
}) => {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);

  const sharedByName = [
    shareContent.shared_by.first_name,
    shareContent.shared_by.last_name,
  ]
    .filter(Boolean)
    .join(" ");
  const sharedByDisplayName = sharedByName || shareContent.shared_by.email;
  const sharedByInitials =
    [shareContent.shared_by.first_name, shareContent.shared_by.last_name]
      .filter(Boolean)
      .map((name) => name.charAt(0).toUpperCase())
      .join("") || shareContent.shared_by.email.charAt(0).toUpperCase();
  const expiryText = shareContent.expires_at
    ? t("share_consumer.expires", {
        date: new Date(shareContent.expires_at).toLocaleDateString(),
      })
    : t("share_consumer.no_expiry");

  const handleCopy = async () => {
    await navigator.clipboard.writeText(window.location.href);
    setCopied(true);
    toast.success(t("common.copied"));
    window.setTimeout(() => setCopied(false), 2000);
  };

  return (
    <aside className="share-panel-gradient flex min-w-0 flex-col items-center p-5 text-center sm:min-h-78 sm:items-start sm:p-8 sm:text-left">
      <img
        src="/safebucket_banner.png"
        alt={t("share_consumer.app_logo")}
        className="h-10 w-auto max-w-48 object-contain object-left"
      />

      <div className="mt-10 w-full space-y-6 sm:mt-auto sm:pt-16">
        <div className="space-y-3">
          <div className="text-secondary-foreground flex items-center justify-center gap-2 text-sm sm:justify-start">
            <Avatar className="size-9">
              <AvatarFallback className="bg-background text-primary">
                {sharedByInitials || "?"}
              </AvatarFallback>
            </Avatar>
            <span>
              {t("share_consumer.shared_by", {
                name: sharedByDisplayName || t("share_consumer.user"),
              })}
            </span>
          </div>

          <h1 className="text-2xl font-bold tracking-tight sm:text-3xl">
            {shareContent.name}
          </h1>

          <div className="text-secondary-foreground flex flex-wrap justify-center gap-x-4 gap-y-2 text-sm sm:justify-start">
            <span className="flex items-center gap-1.5">
              <Calendar className="size-4" />
              {expiryText}
            </span>
            {shareContent.password_protected && (
              <span className="flex items-center gap-1.5">
                <Lock className="size-4" />
                {t("share_consumer.password_protected")}
              </span>
            )}
            {shareContent.max_views !== null && (
              <span className="flex items-center gap-1.5">
                <Eye className="size-4" />
                {t("share_consumer.views", {
                  current: shareContent.current_views,
                  max: shareContent.max_views,
                })}
              </span>
            )}
            {shareContent.allow_upload && (
              <span className="flex items-center gap-1.5">
                <Upload className="size-4" />
                {shareContent.max_uploads !== null
                  ? t("share_consumer.uploads", {
                      current: shareContent.current_uploads,
                      max: shareContent.max_uploads,
                    })
                  : t("share_consumer.uploads_allowed")}
              </span>
            )}
            {shareContent.allow_upload &&
              shareContent.max_upload_size !== null && (
                <span className="flex items-center gap-1.5">
                  <Upload className="size-4" />
                  {t("share_consumer.upload_size_limit", {
                    size: formatFileSize(shareContent.max_upload_size),
                  })}
                </span>
              )}
          </div>
        </div>

        <div className="grid gap-3">
          <Button
            type="button"
            size="lg"
            className="w-full"
            disabled={shareContent.files.length === 0 || isDownloadingAll}
            onClick={onDownloadAll}
          >
            {isDownloadingAll ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <Download className="size-4" />
            )}
            {t("share_consumer.download_all", {
              size: formatFileSize(totalSize),
            })}
          </Button>
          <Button
            type="button"
            variant="outline"
            size="lg"
            className="w-full"
            onClick={handleCopy}
          >
            {copied ? (
              <Check className="size-4" />
            ) : (
              <Copy className="size-4" />
            )}
            {t("share_consumer.copy_link")}
          </Button>
        </div>
      </div>
    </aside>
  );
};
