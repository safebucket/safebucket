import { useTranslation } from "react-i18next";
import { CheckCircle } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";

export interface IMFASuccessStateProps {
  title?: string;
  message?: string;
}

export function MFASuccessState({ title, message }: IMFASuccessStateProps) {
  const { t } = useTranslation();

  return (
    <div className="m-6 flex h-full items-center justify-center">
      <Card className="mx-auto w-full max-w-md">
        <CardContent className="pt-6">
          <div className="space-y-4 text-center">
            <CheckCircle className="mx-auto h-12 w-12 text-green-500" />
            <h3 className="text-lg font-semibold">
              {title || t("auth.mfa.success_title")}
            </h3>
            <p className="text-muted-foreground text-sm">
              {message || t("auth.mfa.success_message")}
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
