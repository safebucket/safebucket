import { Link2Off, Plus } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import type { FC } from "react";

import type { IBucket } from "@/types/bucket.ts";
import { bucketSharesQueryOptions } from "@/queries/bucket.ts";
import { ShareLinkItem } from "@/components/bucket-members/components/ShareLinkItem.tsx";
import { QuickShareDialog } from "@/components/quick-share/QuickShareDialog.tsx";
import { Skeleton } from "@/components/ui/skeleton.tsx";

interface IShareLinksListProps {
  bucket: IBucket;
}

export const ShareLinksList: FC<IShareLinksListProps> = ({ bucket }) => {
  const { t } = useTranslation();
  const [dialogOpen, setDialogOpen] = useState(false);
  const { data: shares, isLoading } = useQuery(
    bucketSharesQueryOptions(bucket.id),
  );

  if (isLoading) {
    return (
      <div className="space-y-3">
        {[1, 2, 3].map((i) => (
          <div
            key={i}
            className="flex items-center gap-4 rounded-md border p-4"
          >
            <Skeleton className="size-10 rounded-md" />
            <div className="flex-1 space-y-2">
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-3 w-48" />
            </div>
            <Skeleton className="size-8" />
          </div>
        ))}
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {shares &&
        shares.length > 0 &&
        shares.map((share) => (
          <ShareLinkItem key={share.id} share={share} bucketId={bucket.id} />
        ))}
      {(!shares || shares.length === 0) && (
        <div className="flex flex-col items-center justify-center py-8 text-center text-muted-foreground">
          <Link2Off className="mb-3 size-10 opacity-50" />
          <p className="text-sm">{t("bucket.settings.shares.no_shares")}</p>
        </div>
      )}
      <button
        type="button"
        onClick={() => setDialogOpen(true)}
        className="flex w-full cursor-pointer items-center justify-center gap-2 rounded-md border border-dashed border-border p-4 text-sm text-muted-foreground transition-colors hover:border-primary hover:text-primary"
      >
        <Plus className="size-4" />
        {t("bucket.settings.shares.create")}
      </button>
      <QuickShareDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        bucketId={bucket.id}
      />
    </div>
  );
};
