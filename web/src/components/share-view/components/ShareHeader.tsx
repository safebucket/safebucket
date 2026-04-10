import { useRef } from "react";
import { useTranslation } from "react-i18next";
import { Calendar, Eye, LayoutGrid, LayoutList, Upload } from "lucide-react";
import type { FC } from "react";

import type { IPublicShareResponse } from "@/types/share.ts";
import { Badge } from "@/components/ui/badge.tsx";
import { Button } from "@/components/ui/button.tsx";

type ViewMode = "list" | "grid";

interface IShareHeaderProps {
  shareContent: IPublicShareResponse;
  viewMode: ViewMode;
  onViewModeChange: (mode: ViewMode) => void;
  onUploadFiles: (files: Array<File>) => void;
}

export const ShareHeader: FC<IShareHeaderProps> = ({
  shareContent,
  viewMode,
  onViewModeChange,
  onUploadFiles,
}) => {
  const { t } = useTranslation();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const uploadsExhausted =
    shareContent.allow_upload &&
    shareContent.max_uploads !== null &&
    shareContent.current_uploads >= shareContent.max_uploads;

  const viewsText = shareContent.max_views
    ? t("share_consumer.views", {
        current: shareContent.current_views,
        max: shareContent.max_views,
      })
    : t("share_consumer.views_unlimited", {
        current: shareContent.current_views,
      });

  const expiryText = (() => {
    if (!shareContent.expires_at) return t("share_consumer.no_expiry");
    const date = new Date(shareContent.expires_at);
    return date < new Date()
      ? t("share_consumer.expired")
      : t("share_consumer.expires", { date: date.toLocaleDateString() });
  })();

  return (
    <div className="border-b bg-background">
      <div className="mx-auto max-w-6xl px-6 py-4">
        <div className="flex items-center justify-between">
          <div className="min-w-0">
            <h1 className="truncate text-xl font-bold md:text-2xl">
              {shareContent.name}
            </h1>
            <div className="text-muted-foreground mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
              <Badge variant="outline">
                {t(`share_consumer.type_${shareContent.type}`)}
              </Badge>
              <span className="flex items-center gap-1">
                <Eye className="h-3 w-3" />
                {viewsText}
              </span>
              <span className="flex items-center gap-1">
                <Calendar className="h-3 w-3" />
                {expiryText}
              </span>
              {shareContent.allow_upload && (
                <Badge variant="secondary" className="gap-1">
                  <Upload className="h-3 w-3" />
                  {t("share_consumer.uploads_allowed")}
                </Badge>
              )}
            </div>
          </div>

          <div className="flex items-center gap-2">
            {shareContent.allow_upload && !uploadsExhausted && (
              <>
                <input
                  ref={fileInputRef}
                  type="file"
                  multiple
                  className="hidden"
                  onChange={(e) => {
                    const files = e.target.files;
                    if (files && files.length > 0) {
                      onUploadFiles(Array.from(files));
                      e.target.value = "";
                    }
                  }}
                />
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => fileInputRef.current?.click()}
                >
                  <Upload className="mr-2 h-4 w-4" />
                  {t("common.upload")}
                </Button>
              </>
            )}
            <Button
              variant={viewMode === "list" ? "default" : "outline"}
              size="icon"
              onClick={() => onViewModeChange("list")}
              title={t("share_consumer.list_view")}
            >
              <LayoutList className="h-4 w-4" />
            </Button>
            <Button
              variant={viewMode === "grid" ? "default" : "outline"}
              size="icon"
              onClick={() => onViewModeChange("grid")}
              title={t("share_consumer.grid_view")}
            >
              <LayoutGrid className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
};
