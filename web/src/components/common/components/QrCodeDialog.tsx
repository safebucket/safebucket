import { Copy } from "lucide-react";
import { useTranslation } from "react-i18next";
import { QRCode } from "react-qr-code";
import type { FC } from "react";

import { Button } from "@/components/ui/button.tsx";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog.tsx";
import { Input } from "@/components/ui/input.tsx";
import { successToast } from "@/components/ui/hooks/use-toast.ts";

interface IQrCodeDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  value: string;
  title: string;
  description: string;
}

export const QrCodeDialog: FC<IQrCodeDialogProps> = ({
  open,
  onOpenChange,
  value,
  title,
  description,
}) => {
  const { t } = useTranslation();

  const handleCopy = () => {
    navigator.clipboard.writeText(value);
    successToast(t("common.copied"));
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <div className="flex flex-col items-center gap-4">
          <div className="mx-auto w-fit rounded-lg bg-white p-4">
            <QRCode value={value} size={200} />
          </div>
          <div className="flex w-full items-center gap-2">
            <Input
              readOnly
              value={value}
              className="bg-muted flex-1 font-mono text-xs"
            />
            <Button
              type="button"
              variant="outline"
              size="icon"
              onClick={handleCopy}
              aria-label={t("common.copy")}
              className="shrink-0"
            >
              <Copy className="size-4" />
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
};
