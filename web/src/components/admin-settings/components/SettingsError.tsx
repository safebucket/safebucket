import { TriangleAlert } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";

export function SettingsError({ onRetry }: { onRetry: () => void }) {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed p-10 text-center">
      <TriangleAlert className="h-8 w-8 text-muted-foreground" />
      <p className="text-sm text-muted-foreground">
        {t("admin.settings.error.title")}
      </p>
      <Button variant="outline" size="sm" onClick={onRetry}>
        {t("admin.settings.error.retry")}
      </Button>
    </div>
  );
}
