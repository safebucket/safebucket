import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { MFAVerifyInput } from "./MFAVerifyInput";
import { FormErrorAlert } from "@/components/common/FormErrorAlert";
import { MFA_CODE_LENGTH } from "../helpers/constants";

interface MFASetupVerifyStepProps {
  code: string;
  setCode: (value: string) => void;
  error: string | null;
  isLoading: boolean;
  onVerify: () => void;
  onBack: () => void;
}

export function MFASetupVerifyStep({
  code,
  setCode,
  error,
  isLoading,
  onVerify,
  onBack,
}: MFASetupVerifyStepProps) {
  const { t } = useTranslation();

  return (
    <>
      <FormErrorAlert error={error} />

      <div className="space-y-4">
        <p className="text-muted-foreground text-center text-sm">
          {t("auth.mfa.verify_setup_instruction")}
        </p>
        <MFAVerifyInput value={code} onChange={setCode} disabled={isLoading} />
      </div>

      <div className="flex gap-2">
        <Button variant="outline" onClick={onBack} className="flex-1">
          {t("auth.mfa.back_to_login")}
        </Button>
        <Button
          onClick={onVerify}
          disabled={isLoading || code.length !== MFA_CODE_LENGTH}
          className="flex-1"
        >
          {isLoading ? t("auth.mfa.enabling") : t("auth.mfa.enable_button")}
        </Button>
      </div>
    </>
  );
}
