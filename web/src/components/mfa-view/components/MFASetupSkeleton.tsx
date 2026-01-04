import { useTranslation } from "react-i18next";
import { Card, CardContent } from "@/components/ui/card";

export function MFASetupSkeleton() {
  const { t } = useTranslation();

  return (
    <div className="m-6 flex h-full items-center justify-center">
      <Card className="mx-auto w-full max-w-md">
        <CardContent className="pt-6">
          <div className="flex flex-col items-center space-y-4">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
            <p className="text-muted-foreground text-sm">
              {t("auth.mfa.setup_required_description")}
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
