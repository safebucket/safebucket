import { useState } from "react";
import { CalendarIcon, X } from "lucide-react";
import { endOfDay, format, startOfDay } from "date-fns";
import { useTranslation } from "react-i18next";
import type { DateRange } from "react-day-picker";
import type { FC } from "react";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";

export const MAX_RANGE_DAYS = 90;

export interface ActivityRange {
  from?: string;
  to?: string;
}

export const dateRangeToQuery = (
  range: DateRange | undefined,
): ActivityRange => ({
  from: range?.from ? startOfDay(range.from).toISOString() : undefined,
  to: range?.to ? endOfDay(range.to).toISOString() : undefined,
});

interface ActivityDateRangePickerProps {
  value: DateRange | undefined;
  onChange: (range: DateRange | undefined) => void;
}

export const ActivityDateRangePicker: FC<ActivityDateRangePickerProps> = ({
  value,
  onChange,
}) => {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);

  const label =
    value?.from && value.to
      ? `${format(value.from, "PP")} - ${format(value.to, "PP")}`
      : value?.from
        ? format(value.from, "PP")
        : t("activity.date_range.placeholder");

  return (
    <div className="flex items-center gap-1">
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button variant="outline" size="sm" className="gap-2">
            <CalendarIcon className="size-4 opacity-50" />
            {label}
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-auto p-0" align="end">
          <Calendar
            mode="range"
            max={MAX_RANGE_DAYS - 1}
            numberOfMonths={2}
            selected={value}
            onSelect={onChange}
            disabled={{ after: new Date() }}
          />
        </PopoverContent>
      </Popover>
      {value?.from && (
        <Button
          type="button"
          variant="ghost"
          size="icon"
          aria-label={t("activity.date_range.clear")}
          onClick={() => onChange(undefined)}
        >
          <X className="size-4" />
        </Button>
      )}
    </div>
  );
};
