import { useTranslation } from "react-i18next";
import { AlertCircle, LogOut } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface MFASetupErrorStateProps {
  onLogout: () => void;
  onRetry: () => void;
}

export function MFASetupErrorState({
  onLogout,
  onRetry,
}: MFASetupErrorStateProps) {
  const { t } = useTranslation();

  return (
    <div className="m-6 flex h-full items-center justify-center">
      <Card className="mx-auto w-full max-w-md">
        <CardHeader className="text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-red-100">
            <AlertCircle className="h-6 w-6 text-red-600" />
          </div>
          <CardTitle>{t("auth.mfa.setup_error")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-2">
            <Button variant="outline" onClick={onLogout} className="flex-1">
              <LogOut className="mr-2 h-4 w-4" />
              {t("common.logout")}
            </Button>
            <Button onClick={onRetry} className="flex-1">
              {t("auth.continue")}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
