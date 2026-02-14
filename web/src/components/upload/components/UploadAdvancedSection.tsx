import { ChevronRight } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";

import { Datepicker } from "@/components/common/components/Datepicker";
import { Label } from "@/components/ui/label";

interface UploadAdvancedSectionProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  expiresAt: Date | undefined;
  onExpiresAtChange: (date: Date | undefined) => void;
}

export const UploadAdvancedSection: FC<UploadAdvancedSectionProps> = ({
  isOpen,
  onOpenChange,
  expiresAt,
  onExpiresAtChange,
}) => {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col gap-3">
      <button
        type="button"
        className="text-muted-foreground hover:text-foreground flex w-full items-center gap-3 text-sm transition-colors"
        onClick={() => onOpenChange(!isOpen)}
      >
        <div className="bg-border h-px flex-1" />
        <span className="flex shrink-0 items-center gap-1.5">
          {t("upload.dialog.advanced")}
          <ChevronRight
            className={`h-3.5 w-3.5 transition-transform ${isOpen ? "rotate-90" : ""}`}
          />
        </span>
        <div className="bg-border h-px flex-1" />
      </button>

      {isOpen && (
        <div className="bg-muted/50 rounded-lg p-4">
          <div className="flex items-center justify-between">
            <Label>{t("upload.dialog.expiration_label")}</Label>
            <Datepicker value={expiresAt} onChange={onExpiresAtChange} />
          </div>
        </div>
      )}
    </div>
  );
};
