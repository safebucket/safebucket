import { addDays, addHours, endOfDay, format } from "date-fns";
import { CalendarDays, Clock3 } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";

import { Datepicker } from "@/components/common/components/Datepicker";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { dateFnsLocale } from "@/lib/date-locale";
import { cn } from "@/lib/utils";

interface UploadAdvancedSectionProps {
  isEnabled: boolean;
  onEnabledChange: (enabled: boolean) => void;
  expiresAt: Date | undefined;
  onExpiresAtChange: (date: Date | undefined) => void;
}

export const UploadAdvancedSection: FC<UploadAdvancedSectionProps> = ({
  isEnabled,
  onEnabledChange,
  expiresAt,
  onExpiresAtChange,
}) => {
  const { t, i18n } = useTranslation();
  const dateLocale = dateFnsLocale(i18n.language);
  const [selectedPreset, setSelectedPreset] = useState<
    "24-hours" | "7-days" | "30-days" | "custom" | undefined
  >();

  const setExpiration = (
    date: Date,
    preset: Exclude<typeof selectedPreset, "custom" | undefined>,
    expiresAtEndOfDay = true,
  ) => {
    onEnabledChange(true);
    onExpiresAtChange(expiresAtEndOfDay ? endOfDay(date) : date);
    setSelectedPreset(preset);
  };

  const handleEnabledChange = (enabled: boolean) => {
    onEnabledChange(enabled);
    onExpiresAtChange(enabled ? endOfDay(addDays(new Date(), 7)) : undefined);
    setSelectedPreset(enabled ? "7-days" : undefined);
  };

  const handleDateChange = (date: Date | undefined) => {
    if (date) {
      onEnabledChange(true);
      onExpiresAtChange(endOfDay(date));
      setSelectedPreset("custom");
    } else {
      handleEnabledChange(false);
    }
  };

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-sm font-medium">
          <CalendarDays className="text-primary size-4" />
          {t("upload.dialog.expiration_label")}
        </div>
        <Switch checked={isEnabled} onCheckedChange={handleEnabledChange} />
      </div>

      {isEnabled && (
        <div className="bg-muted/50 flex flex-wrap items-center gap-2 rounded-xl border p-3">
          <Button
            type="button"
            variant="outline"
            size="sm"
            className={cn(
              selectedPreset === "24-hours" &&
                "border-primary/20 bg-primary/10 text-primary hover:bg-primary/15",
            )}
            onClick={() =>
              setExpiration(addHours(new Date(), 24), "24-hours", false)
            }
          >
            {t("upload.dialog.expiration_24_hours")}
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            className={cn(
              selectedPreset === "7-days" &&
                "border-primary/20 bg-primary/10 text-primary hover:bg-primary/15",
            )}
            onClick={() => setExpiration(addDays(new Date(), 7), "7-days")}
          >
            {t("upload.dialog.expiration_7_days")}
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            className={cn(
              selectedPreset === "30-days" &&
                "border-primary/20 bg-primary/10 text-primary hover:bg-primary/15",
            )}
            onClick={() => setExpiration(addDays(new Date(), 30), "30-days")}
          >
            {t("upload.dialog.expiration_30_days")}
          </Button>
          <Datepicker
            value={expiresAt}
            onChange={handleDateChange}
            className="w-auto"
            showValue={selectedPreset === "custom"}
          />
          {expiresAt && (
            <div className="text-muted-foreground flex w-full items-center gap-1.5 text-xs">
              <Clock3 className="text-primary size-3.5" />
              <span>{t("upload.dialog.expires")}</span>
              <span className="text-foreground font-medium">
                {format(expiresAt, "PPPp", { locale: dateLocale })}
              </span>
            </div>
          )}
        </div>
      )}
    </div>
  );
};
