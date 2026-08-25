import { useState } from "react";
import { useTranslation } from "react-i18next";
import { QRCode } from "react-qr-code";

import { Check, Copy } from "lucide-react";
import { toast } from "sonner";
import type { FC } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

interface IQuickShareResultStepProps {
  generatedLink: string;
}

export const QuickShareResultStep: FC<IQuickShareResultStepProps> = ({
  generatedLink,
}) => {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(generatedLink).then(() => {
      setCopied(true);
      toast.success(t("quick_share.copied"));
      setTimeout(() => setCopied(false), 2000);
    });
  };

  return (
    <div className="flex flex-col items-center space-y-6 py-4">
      <div className="space-y-1 text-center">
        <p className="text-lg font-medium">{t("quick_share.link_ready")}</p>
        <p className="text-muted-foreground text-sm">
          {t("quick_share.link_ready_description")}
        </p>
      </div>

      <div className="rounded-lg bg-white p-4">
        <QRCode value={generatedLink} size={180} />
      </div>

      <div className="flex w-full items-center gap-2">
        <Input
          readOnly
          value={generatedLink}
          className="bg-muted flex-1 font-mono text-sm"
        />
        <Button
          type="button"
          variant={copied ? "default" : "outline"}
          onClick={handleCopy}
          className="shrink-0 gap-2"
        >
          {copied ? (
            <Check className="h-4 w-4" />
          ) : (
            <Copy className="h-4 w-4" />
          )}
          {t("quick_share.copy")}
        </Button>
      </div>
    </div>
  );
};
