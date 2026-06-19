import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";

import type { IMFADeviceSetupResponse } from "@/components/auth-view/types/session";
import type { SetupStep } from "@/components/mfa-view/helpers/types";

import {
  MFA_CODE_LENGTH,
  MFA_DEFAULT_DEVICE_NAME,
} from "@/components/mfa-view/helpers/constants";
import {
  useAddMFADeviceMutation,
  useVerifyMFADeviceMutation,
} from "@/queries/mfa";
import { ProviderType } from "@/types/auth_providers.ts";

export interface UseMFASetupOptions {
  isRestricted?: boolean;
  providerType?: string;
  hasExistingDevices?: boolean;
}

export interface UseMFASetupReturn {
  step: SetupStep;
  deviceName: string;
  setDeviceName: (name: string) => void;
  password: string;
  setPassword: (password: string) => void;
  stepUpCode: string;
  setStepUpCode: (code: string) => void;
  needsPassword: boolean;
  needsStepUpCode: boolean;
  setupData: IMFADeviceSetupResponse | null;
  code: string;
  setCode: (code: string) => void;
  error: string | null;
  isLoading: boolean;
  startSetup: () => Promise<void>;
  goToVerify: () => void;
  goBack: () => void;
  verifyCode: () => Promise<void>;
  reset: () => void;
}

export function useMFASetup(
  options: UseMFASetupOptions = {},
): UseMFASetupReturn {
  const {
    isRestricted = false,
    providerType,
    hasExistingDevices = false,
  } = options;
  const { t } = useTranslation();
  const addDeviceMutation = useAddMFADeviceMutation();
  const verifyDeviceMutation = useVerifyMFADeviceMutation();

  const isOIDC = providerType === ProviderType.OIDC;
  const needsPassword = !isRestricted && !isOIDC;
  const needsStepUpCode = !isRestricted && isOIDC && hasExistingDevices;

  const [step, setStep] = useState<SetupStep>("name");
  const [deviceName, setDeviceName] = useState(MFA_DEFAULT_DEVICE_NAME);
  const [setupData, setSetupData] = useState<IMFADeviceSetupResponse | null>(
    null,
  );
  const [code, setCode] = useState("");
  const [error, setError] = useState<string | null>(null);

  const [password, setPassword] = useState("");
  const [stepUpCode, setStepUpCode] = useState("");

  const reset = useCallback(() => {
    setStep("name");
    setDeviceName(MFA_DEFAULT_DEVICE_NAME);
    setPassword("");
    setStepUpCode("");
    setSetupData(null);
    setCode("");
    setError(null);
  }, []);

  const goToVerify = useCallback(() => {
    setStep("verify");
  }, []);

  const goBack = useCallback(() => {
    if (step === "verify") {
      setStep("qr");
      setCode("");
      setError(null);
    }
  }, [step]);

  const startSetup = useCallback(async () => {
    if (!deviceName.trim()) {
      setError(t("auth.mfa.error_device_name_required"));
      return;
    }

    if (needsPassword && !password) {
      setError(t("auth.mfa.error_password_required"));
      return;
    }

    if (needsStepUpCode && stepUpCode.length !== MFA_CODE_LENGTH) {
      setError(t("auth.mfa.error_stepup_required"));
      return;
    }

    setError(null);
    try {
      const response = await addDeviceMutation.mutateAsync({
        name: deviceName.trim(),
        password: needsPassword ? password : undefined,
        code: needsStepUpCode ? stepUpCode : undefined,
      });
      setSetupData(response);
      setStep("qr");
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : "";
      if (errorMessage.includes("MFA_DEVICE_NAME_EXISTS")) {
        setError(t("auth.mfa.error_device_name_exists"));
      } else if (errorMessage.includes("MAX_MFA_DEVICES_REACHED")) {
        setError(t("auth.mfa.error_max_devices"));
      } else if (errorMessage.includes("INVALID_PASSWORD")) {
        setError(t("auth.mfa.error_invalid_password"));
      } else if (errorMessage.includes("INVALID_MFA_CODE")) {
        setError(t("auth.mfa.invalid_code"));
      } else {
        setError(t("auth.mfa.setup_error"));
      }
    }
  }, [
    deviceName,
    password,
    stepUpCode,
    needsPassword,
    needsStepUpCode,
    addDeviceMutation,
    t,
  ]);

  const verifyCode = useCallback(async () => {
    if (code.length !== MFA_CODE_LENGTH) {
      setError(t("auth.mfa.error_code_length"));
      return;
    }

    if (!setupData?.device_id) {
      setError(t("auth.mfa.setup_error"));
      return;
    }

    setError(null);
    try {
      await verifyDeviceMutation.mutateAsync({
        deviceId: setupData.device_id,
        code,
      });

      setStep("success");
    } catch {
      setError(t("auth.mfa.verify_error"));
    }
  }, [code, setupData, verifyDeviceMutation, t]);

  return {
    step,
    deviceName,
    setDeviceName,
    password,
    setPassword,
    stepUpCode,
    setStepUpCode,
    needsPassword,
    needsStepUpCode,
    setupData,
    code,
    setCode,
    error,
    isLoading: addDeviceMutation.isPending || verifyDeviceMutation.isPending,
    startSetup,
    goToVerify,
    goBack,
    verifyCode,
    reset,
  };
}
