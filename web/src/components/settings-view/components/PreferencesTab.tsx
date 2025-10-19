import { useTranslation } from "react-i18next";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card.tsx";
import { Field, FieldContent, FieldLabel } from "@/components/ui/field.tsx";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select.tsx";
import { useTheme } from "@/components/theme/hooks/useTheme.ts";
import { ThemeSelector } from "@/components/theme/components/ThemeSelector.tsx";

export function PreferencesTab() {
  const { t, i18n } = useTranslation();
  const { mode, colorTheme, setMode, setColorTheme } = useTheme();

  const handleLanguageChange = (value: string) => {
    i18n.changeLanguage(value);
  };

  const languages = [
    { value: "en", label: t("settings.language.english") },
    { value: "fr", label: t("settings.language.french") },
  ];

  const themeModes = [
    { value: "light", label: t("settings.appearance.theme_mode.light") },
    { value: "dark", label: t("settings.appearance.theme_mode.dark") },
    { value: "system", label: t("settings.appearance.theme_mode.system") },
  ];

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>{t("settings.appearance.title")}</CardTitle>
          <CardDescription>
            {t("settings.appearance.description")}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <Field orientation="horizontal">
            <FieldLabel className="min-w-[200px]">
              {t("settings.appearance.theme_mode.label")}
            </FieldLabel>
            <FieldContent className="items-end">
              <Select value={mode} onValueChange={setMode}>
                <SelectTrigger className="w-[200px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {themeModes.map((themeMode) => (
                    <SelectItem key={themeMode.value} value={themeMode.value}>
                      {themeMode.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FieldContent>
          </Field>

          <div className="space-y-2">
            <label className="text-sm font-medium">
              {t("settings.appearance.color_theme.label")}
            </label>
            <ThemeSelector value={colorTheme} onChange={setColorTheme} />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("settings.language.label")}</CardTitle>
          <CardDescription>
            {t("settings.language.description")}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Field orientation="horizontal">
            <FieldLabel className="min-w-[200px]">
              {t("settings.language.label")}
            </FieldLabel>
            <FieldContent className="items-end">
              <Select
                value={i18n.language}
                onValueChange={handleLanguageChange}
              >
                <SelectTrigger className="w-[200px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {languages.map((lang) => (
                    <SelectItem key={lang.value} value={lang.value}>
                      {lang.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FieldContent>
          </Field>
        </CardContent>
      </Card>
    </div>
  );
}
