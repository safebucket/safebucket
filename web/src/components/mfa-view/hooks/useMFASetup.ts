import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { useQueryClient } from "@tanstack/react-query";

import type {
  IMFADeviceSetupResponse,
  SetupStep,
} from "@/components/mfa-view/helpers/types";
import { authCookies } from "@/lib/auth-service";
import {
  MFA_CODE_LENGTH,
  MFA_DEFAULT_DEVICE_NAME,
} from "@/components/mfa-view/helpers/constants";
import { useMFADevices } from "@/components/mfa-view/hooks/useMFADevices";

export interface UseMFASetupReturn {
  step: SetupStep;
  deviceName: string;
  setDeviceName: (name: string) => void;
  password: string;
  setPassword: (password: string) => void;
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

export function useMFASetup(mfaToken?: string): UseMFASetupReturn {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { addDevice, verifyDevice, isAddingDevice, isVerifyingDevice } =
    useMFADevices(mfaToken);

  const [step, setStep] = useState<SetupStep>("name");
  const [deviceName, setDeviceName] = useState(MFA_DEFAULT_DEVICE_NAME);
  const [setupData, setSetupData] = useState<IMFADeviceSetupResponse | null>(
    null,
  );
  const [code, setCode] = useState("");
  const [error, setError] = useState<string | null>(null);

  const [password, setPassword] = useState("");

  const reset = useCallback(() => {
    setStep("name");
    setDeviceName(MFA_DEFAULT_DEVICE_NAME);
    setPassword("");
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

    if (!password && !mfaToken) {
      setError(t("auth.mfa.error_password_required"));
      return;
    }

    setError(null);
    try {
      const response = await addDevice(deviceName.trim(), password, mfaToken);
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
      } else {
        setError(t("auth.mfa.setup_error"));
      }
    }
  }, [deviceName, password, mfaToken, addDevice, t]);

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
      const response = (await verifyDevice(setupData.device_id, code)) as {
        access_token?: string;
        refresh_token?: string;
      };

      // Save the new tokens with updated MFA claim
      if (response.access_token && response.refresh_token) {
        authCookies.setAll(
          response.access_token,
          response.refresh_token,
          "local",
        );
      }

      setStep("success");
      queryClient.invalidateQueries({ queryKey: ["mfa", "devices"] });
    } catch {
      setError(t("auth.mfa.verify_error"));
    }
  }, [code, setupData, verifyDevice, queryClient, t]);

  return {
    step,
    deviceName,
    setDeviceName,
    password,
    setPassword,
    setupData,
    code,
    setCode,
    error,
    isLoading: isAddingDevice || isVerifyingDevice,
    startSetup,
    goToVerify,
    goBack,
    verifyCode,
    reset,
  };
}
