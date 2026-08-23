import { Check, Copy, Pencil, QrCode, X } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";
import type { IBucket } from "@/types/bucket.ts";
import { useBucketDeletion } from "@/components/bucket-view/hooks/useBucketDeletion";
import { useBucketInformation } from "@/components/bucket-view/hooks/useBucketInformation";
import { QrCodeDialog } from "@/components/common/components/QrCodeDialog.tsx";
import { useBucketPermissions } from "@/hooks/usePermissions";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

interface ISettingsTabProps {
  bucket: IBucket;
}

export const SettingsTab: FC<ISettingsTabProps> = ({
  bucket,
}: ISettingsTabProps) => {
  const { t } = useTranslation();
  const { isOwner } = useBucketPermissions(bucket.id);
  const [qrOpen, setQrOpen] = useState(false);
  const {
    isEditingName,
    setIsEditingName,
    bucketName,
    setBucketName,
    copiedField,
    bucketUrl,
    handleCopy,
    handleSaveName,
    handleCancelName,
  } = useBucketInformation(bucket);
  const {
    confirmationText,
    setConfirmationText,
    expectedDeleteText,
    isConfirmationValid,
    handleDeleteBucket,
  } = useBucketDeletion(bucket);

  const sectionHeader = "border-border border-b px-6 py-4";
  const sectionBody = "flex flex-col gap-5 px-6 py-5";

  return (
    <div className="flex flex-col gap-6">
      <div className="bg-card border-border overflow-hidden rounded-xl border">
        <div className={sectionHeader}>
          <h3 className="text-base font-semibold tracking-tight">
            {t("bucket.settings.information.title")}
          </h3>
        </div>
        <div className={sectionBody}>
          <div className="flex flex-col gap-2">
            <Label className="text-muted-foreground text-xs">
              {t("bucket.settings.information.bucket_url")}
            </Label>
            <div className="flex items-center gap-2">
              <Input value={bucketUrl} readOnly className="flex-1 font-mono" />
              <Button
                size="icon"
                variant="outline"
                onClick={() => handleCopy(bucketUrl, "Bucket URL")}
                aria-label={t("common.copy")}
              >
                {copiedField === "Bucket URL" ? (
                  <Check className="h-4 w-4" />
                ) : (
                  <Copy className="h-4 w-4" />
                )}
              </Button>
              <Button
                size="icon"
                variant="outline"
                onClick={() => setQrOpen(true)}
                aria-label={t("bucket.settings.information.show_qr")}
              >
                <QrCode className="h-4 w-4" />
              </Button>
            </div>
          </div>

          <div className="flex flex-col gap-2">
            <Label className="text-muted-foreground text-xs">
              {t("bucket.settings.information.bucket_name")}
            </Label>
            <div className="flex items-center gap-2">
              {isEditingName ? (
                <>
                  <Input
                    value={bucketName}
                    onChange={(e) => setBucketName(e.target.value)}
                    placeholder={t(
                      "bucket.settings.information.enter_bucket_name",
                    )}
                    className="flex-1"
                  />
                  <Button
                    size="icon"
                    onClick={handleSaveName}
                    aria-label={t("common.save")}
                  >
                    <Check className="h-4 w-4" />
                  </Button>
                  <Button
                    size="icon"
                    variant="outline"
                    onClick={handleCancelName}
                    aria-label={t("common.cancel")}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </>
              ) : (
                <>
                  <Input value={bucket.name} disabled className="flex-1" />
                  {isOwner ? (
                    <Button
                      size="icon"
                      variant="outline"
                      onClick={() => setIsEditingName(true)}
                      aria-label={t("common.edit")}
                    >
                      <Pencil className="h-4 w-4" />
                    </Button>
                  ) : null}
                </>
              )}
            </div>
          </div>
        </div>
      </div>

      {isOwner ? (
        <div className="bg-card border-destructive/30 overflow-hidden rounded-xl border">
          <div className="border-destructive/20 border-b px-6 py-4">
            <h3 className="text-destructive text-base font-semibold tracking-tight">
              {t("bucket.settings.deletion.title")}
            </h3>
            <p className="text-muted-foreground mt-1 text-sm">
              {t("bucket.settings.deletion.description")}
            </p>
          </div>
          <div className="flex flex-col gap-3 px-6 py-5">
            <Label
              htmlFor="confirmation"
              className="text-muted-foreground text-xs"
            >
              {t("bucket.settings.deletion.type_to_confirm")}{" "}
              <span className="bg-destructive/10 text-destructive rounded px-1 py-0.5 font-mono">
                {expectedDeleteText}
              </span>{" "}
              {t("bucket.settings.deletion.to_confirm")}
            </Label>
            <div className="flex items-center gap-2">
              <Input
                id="confirmation"
                value={confirmationText}
                onChange={(e) => setConfirmationText(e.target.value)}
                placeholder={expectedDeleteText}
                className="flex-1"
              />
              <Button
                variant="destructive"
                onClick={handleDeleteBucket}
                disabled={!isConfirmationValid}
              >
                {t("bucket.settings.deletion.delete")}
              </Button>
            </div>
          </div>
        </div>
      ) : null}

      <QrCodeDialog
        open={qrOpen}
        onOpenChange={setQrOpen}
        value={bucketUrl}
        title={t("bucket.settings.information.qr_title")}
        description={t("bucket.settings.information.qr_description")}
      />
    </div>
  );
};
