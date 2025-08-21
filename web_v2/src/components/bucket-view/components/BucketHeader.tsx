import type { FC } from "react";
import { useTranslation } from "react-i18next";

import { PlusCircle } from "lucide-react";

import { BucketViewOptions } from "@/components/bucket-view/components/BucketViewOptions";
import { shareFileFields } from "@/components/bucket-view/helpers/constants";
import type { IBucket } from "@/components/bucket-view/helpers/types";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { FormDialog } from "@/components/dialogs/components/FormDialog";
import { useDialog } from "@/components/dialogs/hooks/useDialog";
import { Button } from "@/components/ui/button";
import { UploadPopover } from "@/components/upload/components/UploadPopover";
import { useUploadContext } from "@/components/upload/hooks/useUploadContext";

interface IBucketHeaderProps {
  bucket: IBucket;
}

export const BucketHeader: FC<IBucketHeaderProps> = ({
  bucket,
}: IBucketHeaderProps) => {
  const { t } = useTranslation();
  const shareFileDialog = useDialog();

  const { path } = useBucketViewContext();
  const { startUpload } = useUploadContext();

  return (
    <div className="flex-1">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">{bucket.name}</h1>
        <div className="flex items-center gap-4">
          <BucketViewOptions />

          <UploadPopover />

          <Button onClick={shareFileDialog.trigger}>
            <PlusCircle className="mr-2 h-4 w-4" />
            {t("bucket.header.share_file")}
          </Button>

          <FormDialog
            {...shareFileDialog.props}
            title={t("bucket.header.share_file")}
            description={t("bucket.header.upload_and_share")}
            fields={shareFileFields}
            onSubmit={(data) => startUpload(data.files, path, bucket.id)}
            confirmLabel={t("bucket.header.share")}
          />
        </div>
      </div>
    </div>
  );
};
