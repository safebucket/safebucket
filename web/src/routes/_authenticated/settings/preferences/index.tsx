import { createFileRoute } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { PreferencesTab } from "@/components/settings-view/components/PreferencesTab";

export const Route = createFileRoute("/_authenticated/settings/preferences/")({
  component: Preferences,
});

function Preferences() {
  const { t } = useTranslation();

  return (
    <div className="container mx-auto max-w-3xl p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold">
          {t("settings.preferences.title")}
        </h1>
        <p className="text-sm text-muted-foreground">
          {t("settings.preferences.description")}
        </p>
      </div>

      <PreferencesTab />
    </div>
  );
}
