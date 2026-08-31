"use client";

import { format } from "date-fns";
import { Calendar as CalendarIcon } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";
import { dateFnsLocale } from "@/lib/date-locale";

import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";

interface DatepickerProps {
  value?: Date;
  onChange?: (date: Date | undefined) => void;
  className?: string;
  showValue?: boolean;
}

export function Datepicker({
  value,
  onChange,
  className,
  showValue = true,
}: DatepickerProps) {
  const { t, i18n } = useTranslation();
  const dateLocale = dateFnsLocale(i18n.language);
  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button
          type="button"
          variant={"outline"}
          className={cn(
            "w-[280px]",
            !value && "text-muted-foreground",
            className,
          )}
        >
          <CalendarIcon className="mr-2 h-4 w-4" />
          {value && showValue ? (
            format(value, "PPP", { locale: dateLocale })
          ) : (
            <span>{t("upload.dialog.pick_a_date")}</span>
          )}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0">
        <Calendar
          className="rounded-lg border"
          mode="single"
          selected={value}
          onSelect={onChange}
          disabled={{ before: new Date() }}
        />
      </PopoverContent>
    </Popover>
  );
}
