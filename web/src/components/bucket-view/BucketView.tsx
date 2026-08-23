import { PlusCircle } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";
import type { IBucket } from "@/types/bucket.ts";
import { NotificationPopover } from "@/components/bucket-view/components/NotificationPopover";
import { itemsToShow } from "@/components/bucket-view/helpers/utils";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { useTrashActions } from "@/components/bucket-view/hooks/useTrashActions";
import { ActivityTab } from "@/components/bucket-view/components/ActivityTab";
import { FilesTab } from "@/components/bucket-view/components/FilesTab";
import { MembersTab } from "@/components/bucket-view/components/MembersTab";
import { PageHeader } from "@/components/bucket-view/components/PageHeader";
import { SettingsTab } from "@/components/bucket-view/components/SettingsTab";
import { ShareLinksTab } from "@/components/bucket-view/components/ShareLinksTab";
import { TrashTab } from "@/components/bucket-view/components/TrashTab";
import { FormDialog } from "@/components/dialogs/components/FormDialog";
import { useDialog } from "@/components/dialogs/hooks/useDialog";
import { useFileActions } from "@/components/file-actions/hooks/useFileActions.ts";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useBucketPermissions } from "@/hooks/usePermissions";
import { UploadDialog } from "@/components/upload/components/UploadDialog";
import { useUploadDialog } from "@/components/upload/hooks/useUploadDialog";

interface IBucketViewProps {
  bucket: IBucket;
}

const tabTriggerClass =
  "relative rounded-none border-b-2 border-transparent bg-transparent px-3 pb-3 pt-1 text-muted-foreground shadow-none data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:text-foreground data-[state=active]:shadow-none";

export const BucketView: FC<IBucketViewProps> = ({
  bucket,
}: IBucketViewProps) => {
  const { t } = useTranslation();
  const { folderId } = useBucketViewContext();
  const { isContributor } = useBucketPermissions(bucket.id);
  const { createFolder } = useFileActions();
  const { trashedItems, restoreItem, purgeItem } = useTrashActions();
  const newFolderDialog = useDialog();
  const uploadDialog = useUploadDialog({ bucketId: bucket.id, folderId });
  const [tab, setTab] = useState("files");

  const items = useMemo(
    () => itemsToShow(bucket.files, bucket.folders, folderId),
    [bucket, folderId],
  );

  useEffect(() => {
    setTab("files");
  }, [folderId]);

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4 pt-2 pb-6">
      <PageHeader
        title={bucket.name}
        actions={
          <>
            <NotificationPopover bucketId={bucket.id} />
            {isContributor ? (
              <Button onClick={uploadDialog.open}>
                <PlusCircle className="h-4 w-4 md:mr-2" />
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
        <Tabs
          value={tab}
          onValueChange={setTab}
          className="flex min-h-0 flex-1 flex-col gap-0"
        >
          <TabsList className="border-border h-auto w-full justify-start gap-1 overflow-x-auto rounded-none border-b bg-transparent p-0 px-6 pt-2">
            <TabsTrigger value="files" className={tabTriggerClass}>
              {t("bucket.view.tabs.files")}
            </TabsTrigger>
            <TabsTrigger value="activity" className={tabTriggerClass}>
              {t("bucket.view.tabs.activity")}
            </TabsTrigger>
            <TabsTrigger value="trash" className={tabTriggerClass}>
              {t("bucket.header.trash")}
            </TabsTrigger>
            <TabsTrigger value="members" className={tabTriggerClass}>
              {t("bucket.view.tabs.members")}
            </TabsTrigger>
            <TabsTrigger value="shares" className={tabTriggerClass}>
              {t("bucket.view.tabs.shares")}
            </TabsTrigger>
            <TabsTrigger value="settings" className={tabTriggerClass}>
              {t("bucket.header.settings")}
            </TabsTrigger>
          </TabsList>

          <div className="bg-muted/40 min-h-0 flex-1 overflow-y-auto p-6">
            <TabsContent value="files" className="mt-0">
              <FilesTab
                bucket={bucket}
                items={items}
                onFilesDropped={uploadDialog.openWithFiles}
                onUpload={uploadDialog.open}
                onNewFolder={newFolderDialog.trigger}
                isContributor={isContributor}
              />
            </TabsContent>
            <TabsContent value="activity" className="mt-0">
              <ActivityTab />
            </TabsContent>
            <TabsContent value="trash" className="mt-0">
              <TrashTab
                items={trashedItems}
                bucket={bucket}
                onRestore={restoreItem}
                onPermanentDelete={purgeItem}
              />
            </TabsContent>
            <TabsContent value="members" className="mt-0">
              <MembersTab bucket={bucket} />
            </TabsContent>
            <TabsContent value="shares" className="mt-0">
              <ShareLinksTab bucket={bucket} />
            </TabsContent>
            <TabsContent value="settings" className="mt-0">
              <SettingsTab bucket={bucket} />
            </TabsContent>
          </div>
        </Tabs>
      </div>

      <UploadDialog
        {...uploadDialog.dialogProps}
        stagedFiles={uploadDialog.stagedFiles}
        onAddFiles={uploadDialog.addFiles}
        onRemoveFile={uploadDialog.removeFile}
        expiresAt={uploadDialog.expiresAt}
        onExpiresAtChange={uploadDialog.setExpiresAt}
        isAdvancedOpen={uploadDialog.isAdvancedOpen}
        onAdvancedOpenChange={uploadDialog.setIsAdvancedOpen}
        onUpload={uploadDialog.handleUpload}
      />

      <FormDialog
        {...newFolderDialog.props}
        title={t("file_actions.new_folder_dialog.title")}
        fields={[
          {
            id: "name",
            label: t("file_actions.new_folder_dialog.name_label"),
            type: "text",
            required: true,
          },
        ]}
        onSubmit={(data) => createFolder(data.name)}
        confirmLabel={t("file_actions.new_folder_dialog.create")}
      />
    </div>
  );
};
