import { useTranslation } from "react-i18next";

import { AlertTriangle } from "lucide-react";
import type { FC } from "react";

import type { IBucket } from "@/components/bucket-view/helpers/types";
import { useBucketDeletion } from "@/components/bucket-view/hooks/useBucketDeletion";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

interface IBucketDeletionProps {
  bucket: IBucket;
}

export const BucketDeletion: FC<IBucketDeletionProps> = ({ bucket }) => {
  const { t } = useTranslation();
  const {
    confirmationText,
    setConfirmationText,
    expectedDeleteText,
    isConfirmationValid,
    handleDeleteBucket,
  } = useBucketDeletion(bucket);

  return (
    <Card className="border-red-200 bg-red-50/50">
      <CardContent className="p-4">
        <div className="space-y-3">
          <div>
            <h3 className="text-sm font-semibold text-red-700">
              {t("bucket.settings.deletion.title")}
            </h3>
            <p className="mt-1 text-xs text-red-600">
              {t("bucket.settings.deletion.description")}
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="confirmation" className="text-xs font-medium">
              {t("bucket.settings.deletion.type_to_confirm")}{" "}
              <span className="rounded bg-red-100 px-1 py-0.5 font-mono text-xs text-red-700">
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
                className="border-red-200 text-xs focus:border-red-300 focus:ring-red-200 mt-2"
              />
              <Button
                variant="destructive"
                size="sm"
                onClick={handleDeleteBucket}
                disabled={!isConfirmationValid}
                className="flex items-center gap-2"
              >
                <AlertTriangle className="h-3 w-3" />
                {t("bucket.settings.deletion.delete")}
              </Button>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
};
