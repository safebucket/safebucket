import type { FC } from "react";
import { useTranslation } from "react-i18next";

import { Check, Copy, Edit2, Info, X } from "lucide-react";

import type { IBucket } from "@/components/bucket-view/helpers/types";
import { useBucketInformation } from "@/components/bucket-view/hooks/useBucketInformation";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

interface IBucketInformationProps {
  bucket: IBucket;
}

export const BucketInformation: FC<IBucketInformationProps> = ({ bucket }) => {
  const { t } = useTranslation();
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

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Info className="h-5 w-5" />
          {t("bucket.settings.information.title")}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <Label className="text-sm font-medium">{t("bucket.settings.information.bucket_url")}</Label>
          <div className="flex items-center gap-2">
            <Input value={bucketUrl} disabled className="font-mono text-xs" />
            <Button
              size="sm"
              variant="outline"
              onClick={() => handleCopy(bucketUrl, "Bucket URL")}
            >
              {copiedField === "Bucket URL" ? (
                <Check className="h-3 w-3" />
              ) : (
                <Copy className="h-3 w-3" />
              )}
            </Button>
          </div>
        </div>

        <div className="space-y-2">
          <Label className="text-sm font-medium">{t("bucket.settings.information.bucket_name")}</Label>
          <div className="flex items-center gap-2">
            {isEditingName ? (
              <>
                <Input
                  value={bucketName}
                  onChange={(e) => setBucketName(e.target.value)}
                  placeholder={t("bucket.settings.information.enter_bucket_name")}
                  className="text-sm"
                />
                <Button size="sm" onClick={handleSaveName}>
                  <Check className="h-3 w-3" />
                </Button>
                <Button size="sm" variant="outline" onClick={handleCancelName}>
                  <X className="h-3 w-3" />
                </Button>
              </>
            ) : (
              <>
                <Input value={bucket.name} disabled className="text-sm" />
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => setIsEditingName(true)}
                >
                  <Edit2 className="h-3 w-3" />
                </Button>
              </>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
};
