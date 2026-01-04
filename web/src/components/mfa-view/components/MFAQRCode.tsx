import { useState } from "react";
import { useTranslation } from "react-i18next";
import QRCode from "react-qr-code";
import { CheckCircle, Copy } from "lucide-react";

import { Button } from "@/components/ui/button";
import { MFA_SECRET_COPY_TIMEOUT } from "@/components/mfa-view/helpers/constants";

interface MFAQRCodeProps {
  qrCodeUri: string;
  secret: string;
}

export function MFAQRCode({ qrCodeUri, secret }: MFAQRCodeProps) {
  const { t } = useTranslation();
  const [secretCopied, setSecretCopied] = useState(false);

  const handleCopySecret = async () => {
    await navigator.clipboard.writeText(secret);
    setSecretCopied(true);
    setTimeout(() => setSecretCopied(false), MFA_SECRET_COPY_TIMEOUT);
  };

  return (
    <div className="flex flex-col items-center space-y-4">
      <div className="rounded-lg bg-white p-4">
        <QRCode value={qrCodeUri} size={180} />
      </div>

      <div className="w-full space-y-2">
        <p className="text-sm font-medium">
          {t("auth.mfa.manual_entry_title")}
        </p>
        <p className="text-muted-foreground text-xs">
          {t("auth.mfa.manual_entry_instruction")}
        </p>
        <div className="flex items-center gap-2">
          <code className="bg-muted flex-1 rounded px-3 py-2 font-mono text-sm break-all">
            {secret}
          </code>
          <Button variant="outline" size="sm" onClick={handleCopySecret}>
            {secretCopied ? (
              <CheckCircle className="h-4 w-4 text-green-600" />
            ) : (
              <Copy className="h-4 w-4" />
            )}
          </Button>
        </div>
      </div>
    </div>
  );
}
