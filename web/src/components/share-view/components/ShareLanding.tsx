import { ArrowRight, Loader2, Share2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";

import { FormErrorAlert } from "@/components/common/FormErrorAlert.tsx";
import { Button } from "@/components/ui/button.tsx";
import { Card } from "@/components/ui/card.tsx";

interface IShareLandingProps {
  onAccess: () => void;
  isLoading: boolean;
  error: string | null;
}

export const ShareLanding: FC<IShareLandingProps> = ({
  onAccess,
  isLoading,
  error,
}) => {
  const { t } = useTranslation();

  return (
    <div className="flex min-h-svh items-center justify-center p-6">
      <Card className="flex w-full max-w-md flex-col items-center gap-6 p-8">
        <div className="bg-primary/10 flex h-16 w-16 items-center justify-center rounded-full">
          <Share2 className="text-primary h-8 w-8" />
        </div>

        <div className="space-y-2 text-center">
          <h1 className="text-xl font-semibold">
            {t("share_consumer.received_link")}
          </h1>
          <p className="text-muted-foreground text-sm">
            {t("share_consumer.received_link_description")}
          </p>
        </div>

        <FormErrorAlert error={error} />

        <Button
          onClick={onAccess}
          disabled={isLoading}
          className="w-full gap-2"
        >
          {isLoading ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <ArrowRight className="h-4 w-4" />
          )}
          {t("share_consumer.access_share")}
        </Button>
      </Card>
    </div>
  );
};
