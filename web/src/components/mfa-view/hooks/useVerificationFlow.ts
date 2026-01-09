import { useState, useEffect } from "react";
import { useRouter } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import type { IMFADevice, IVerificationFlowState } from "../helpers/types";
import { useLogin } from "@/hooks/useAuth";
import { getDefaultDeviceId, isCodeValid } from "../helpers/utils";
import { MFA_VERIFICATION_SUCCESS_DELAY } from "../helpers/constants";

export interface IUseVerificationFlowProps {
  mfaToken: string;
  redirectPath?: string;
  devices: IMFADevice[];
  onClearDevices: () => void;
}

export function useVerificationFlow({
  mfaToken,
  redirectPath,
  devices,
  onClearDevices,
}: IUseVerificationFlowProps): IVerificationFlowState {
  const { t } = useTranslation();
  const router = useRouter();
  const { verifyMFA } = useLogin();

  const [code, setCode] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [isVerified, setIsVerified] = useState(false);

  const defaultDeviceId = getDefaultDeviceId(devices);
  const [selectedDeviceId, setSelectedDeviceId] =
    useState<string>(defaultDeviceId);

  useEffect(() => {
    if (defaultDeviceId && !selectedDeviceId) {
      setSelectedDeviceId(defaultDeviceId);
    }
  }, [defaultDeviceId, selectedDeviceId]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (!isCodeValid(code)) {
      setError(t("auth.mfa.error_code_length"));
      return;
    }

    setIsLoading(true);

    const deviceId = devices.length > 0 ? selectedDeviceId : undefined;
    const result = await verifyMFA(mfaToken, code, deviceId);

    if (result.success) {
      setIsVerified(true);
      setTimeout(async () => {
        onClearDevices();
        await router.invalidate();
        router.navigate({ to: redirectPath || "/", replace: true });
      }, MFA_VERIFICATION_SUCCESS_DELAY);
    } else {
      setError(result.error || t("auth.mfa.error_verification_failed"));
    }

    setIsLoading(false);
  };

  const handleBackToLogin = () => {
    onClearDevices();
  };

  return {
    code,
    setCode,
    selectedDeviceId,
    setSelectedDeviceId,
    error,
    isLoading,
    isVerified,
    handleSubmit,
    handleBackToLogin,
  };
}
