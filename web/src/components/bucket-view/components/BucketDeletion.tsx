import { useTranslation } from "react-i18next";

import { AlertTriangle } from "lucide-react";
import type { FC } from "react";

import type { IBucket } from "@/types/bucket.ts";
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
    <Card className="border-destructive/20 bg-destructive/5">
      <CardContent>
        <div className="space-y-3 mt-4">
          <div>
            <h3 className="text-sm font-semibold text-destructive">
              {t("bucket.settings.deletion.title")}
            </h3>
            <p className="mt-1 text-xs text-destructive/80">
              {t("bucket.settings.deletion.description")}
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="confirmation" className="text-xs font-medium">
              {t("bucket.settings.deletion.type_to_confirm")}{" "}
              <span className="rounded bg-destructive/10 px-1 py-0.5 font-mono text-xs text-destructive">
                {expectedDeleteText}
              </span>{" "}
              {t("bucket.settings.deletion.to_confirm")}
            </Label>
            <div className="flex items-center gap-2 mt-2">
              <Input
                id="confirmation"
                value={confirmationText}
                onChange={(e) => setConfirmationText(e.target.value)}
                placeholder={expectedDeleteText}
                className="border-destructive/20 text-xs focus:border-destructive/30 focus:ring-destructive/20"
              />
              <Button
                variant="destructive"
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
