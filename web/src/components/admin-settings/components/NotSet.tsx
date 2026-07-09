import { useTranslation } from "react-i18next";

export function NotSet() {
  const { t } = useTranslation();
  return (
    <span className="font-normal italic text-muted-foreground">
      {t("admin.settings.values.not_set")}
    </span>
  );
}
