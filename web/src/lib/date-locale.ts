import { de, enUS, fr } from "date-fns/locale";
import type { Locale } from "date-fns";

const localeMap: Record<string, Locale> = {
  en: enUS,
  fr,
  de,
};

export function dateFnsLocale(language: string): Locale {
  const code = language.split("-")[0] ?? "en";
  return localeMap[code] ?? enUS;
}
