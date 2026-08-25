import { Copy } from "lucide-react";
import { useTranslation } from "react-i18next";
import { NotSet } from "./NotSet";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { successToast } from "@/lib/toast";

export function CopyableValue({ value }: { value?: string | null }) {
  const { t } = useTranslation();
  if (value === undefined || value === null || value === "") {
    return <NotSet />;
  }
  const handleCopy = () => {
    navigator.clipboard.writeText(value);
    successToast(t("admin.settings.copy.copied"));
  };
  return (
    <span className="inline-flex items-center gap-1">
      <span>{value}</span>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            onClick={handleCopy}
            aria-label={t("admin.settings.copy.label")}
          >
            <Copy />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{t("admin.settings.copy.label")}</TooltipContent>
      </Tooltip>
    </span>
  );
}
