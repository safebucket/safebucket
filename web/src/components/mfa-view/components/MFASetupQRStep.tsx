import { useTranslation } from "react-i18next";
import { LogOut } from "lucide-react";
import { Button } from "@/components/ui/button";
import { MFAQRCode } from "./MFAQRCode";

interface MFASetupQRStepProps {
  qrCodeUri: string;
  secret: string;
  onContinue: () => void;
  onLogout: () => void;
}

export function MFASetupQRStep({
  qrCodeUri,
  secret,
  onContinue,
  onLogout,
}: MFASetupQRStepProps) {
  const { t } = useTranslation();

  return (
    <>
      <p className="text-muted-foreground text-center text-sm">
        {t("auth.mfa.qr_code_instruction")}
      </p>

      <MFAQRCode qrCodeUri={qrCodeUri} secret={secret} />

      <div className="flex gap-2">
        <Button variant="outline" onClick={onLogout} className="flex-1">
          <LogOut className="mr-2 h-4 w-4" />
          {t("common.logout")}
        </Button>
        <Button onClick={onContinue} className="flex-1">
          {t("auth.continue")}
        </Button>
      </div>
    </>
  );
}
