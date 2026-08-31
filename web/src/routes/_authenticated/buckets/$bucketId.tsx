import { Upload } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useSuspenseQuery } from "@tanstack/react-query";
import { Outlet, createFileRoute, useParams } from "@tanstack/react-router";

import { NotificationPopover } from "@/components/bucket-view/components/NotificationPopover";
import { PageHeader } from "@/components/bucket-view/components/PageHeader";
import { TabLink } from "@/components/bucket-view/components/TabLink";
import { Button } from "@/components/ui/button";
import { useBucketPermissions } from "@/hooks/usePermissions";
import { DragDropZone } from "@/components/upload/components/DragDropZone";
import { UploadDialog } from "@/components/upload/components/UploadDialog";
import { useUploadDialog } from "@/components/upload/hooks/useUploadDialog";
import { bucketDataQueryOptions } from "@/queries/bucket.ts";

export const Route = createFileRoute("/_authenticated/buckets/$bucketId")({
  loader: ({ context: { queryClient }, params: { bucketId } }) =>
    queryClient.ensureQueryData(bucketDataQueryOptions(bucketId)),
  component: BucketLayout,
});

const tabs = [
  {
    to: "/buckets/$bucketId/files/{-$folderId}",
    labelKey: "bucket.view.tabs.files",
  },
  { to: "/buckets/$bucketId/activity", labelKey: "bucket.view.tabs.activity" },
  { to: "/buckets/$bucketId/trash", labelKey: "bucket.header.trash" },
  { to: "/buckets/$bucketId/members", labelKey: "bucket.view.tabs.members" },
  { to: "/buckets/$bucketId/shares", labelKey: "bucket.view.tabs.shares" },
  { to: "/buckets/$bucketId/settings", labelKey: "bucket.header.settings" },
] as const;

function BucketLayout() {
  const { t } = useTranslation();
  const { bucketId } = Route.useParams();
  const { folderId } = useParams({ strict: false });
  const bucket = useSuspenseQuery(bucketDataQueryOptions(bucketId)).data;
  const { isContributor } = useBucketPermissions(bucketId);
  const uploadDialog = useUploadDialog({ bucketId, folderId });

  return (
    <div className="flex w-full min-h-0 flex-1 flex-col">
      <div className="mx-6 flex min-h-0 flex-1 flex-col gap-4 pt-2 pb-6">
        <PageHeader
          title={bucket.name}
          actions={
            <>
              <NotificationPopover bucketId={bucketId} />
              {isContributor ? (
                <Button onClick={uploadDialog.open}>
                  <Upload className="h-4 w-4 md:mr-2" />
                  <span className="hidden md:inline">
                    {t("bucket.header.upload_file")}
                  </span>
                </Button>
              ) : (
                <span className="text-muted-foreground text-sm">
                  {t("bucket.header.view_only")}
                </span>
              )}
            </>
          }
        />

        <div className="bg-card ring-foreground/10 flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl shadow-sm ring-1">
          <div className="border-border flex w-full justify-start gap-1 overflow-x-auto border-b px-6 pt-2">
            {tabs.map((tab) => (
              <TabLink key={tab.to} to={tab.to} params={{ bucketId }}>
                {t(tab.labelKey)}
              </TabLink>
            ))}
          </div>

          <DragDropZone
            bucketId={bucketId}
            className="bg-muted/40 min-h-0 flex-1 overflow-y-auto p-6"
            onFilesDropped={uploadDialog.openWithFiles}
          >
            <Outlet />
          </DragDropZone>
        </div>
      </div>

      <UploadDialog
        {...uploadDialog.dialogProps}
        stagedFiles={uploadDialog.stagedFiles}
        onAddFiles={uploadDialog.addFiles}
        onRemoveFile={uploadDialog.removeFile}
        expiresAt={uploadDialog.expiresAt}
        onExpiresAtChange={uploadDialog.setExpiresAt}
        isExpirationEnabled={uploadDialog.isExpirationEnabled}
        onExpirationEnabledChange={uploadDialog.setIsExpirationEnabled}
        onUpload={uploadDialog.handleUpload}
      />
    </div>
  );
}
