import { useTranslation } from "react-i18next";
import { NotSet } from "./NotSet";

export function RateValue({ value }: { value?: number | null }) {
  const { t } = useTranslation();
  if (value === undefined || value === null) {
    return <NotSet />;
  }
  return <span>{t("admin.settings.values.per_minute", { value })}</span>;
}
