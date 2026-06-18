import { formatDistanceToNow } from "date-fns";
import { useTranslation } from "react-i18next";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useTimeDisplay } from "@/components/time-display/hooks/useTimeDisplay";
import { formatAbsoluteTimestamp } from "@/components/time-display/helpers/format";

export function TimestampCell({ timestamp }: { timestamp: string }) {
  const { i18n } = useTranslation();
  const { mode } = useTimeDisplay();

  if (!timestamp) {
    return <span>-</span>;
  }

  const relative = formatDistanceToNow(new Date(Number(timestamp) / 1000000), {
    addSuffix: true,
  });

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className="cursor-default">
          {formatAbsoluteTimestamp(timestamp, mode, i18n.language)}
        </span>
      </TooltipTrigger>
      <TooltipContent>{relative}</TooltipContent>
    </Tooltip>
  );
}
