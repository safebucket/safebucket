import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";

interface MFAEmptyStateProps {
  onSetup: () => void;
}

export function MFAEmptyState({ onSetup }: MFAEmptyStateProps) {
  const { t } = useTranslation();

  return (
    <div className="flex items-center justify-between">
      <div className="space-y-1">
        <div className="flex items-center gap-2">
          <Badge variant="secondary">{t("auth.mfa.not_enabled")}</Badge>
        </div>
        <p className="text-muted-foreground text-sm">
          {t("auth.mfa.not_enabled_description")}
        </p>
      </div>
      <Button onClick={onSetup}>{t("auth.mfa.setup_button")}</Button>
    </div>
  );
}
