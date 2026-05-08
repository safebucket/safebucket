import { useCallback, useEffect, useState } from "react";

import { useNavigate } from "@tanstack/react-router";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import type { FieldErrors, UseFormRegister } from "react-hook-form";

import type {
  IPasswordResetPasswordFormData,
  PasswordResetStage,
} from "@/components/auth-view/helpers/types";
import type {
  IMFADevice,
  IMFADevicesResponse,
} from "@/components/auth-view/types/session";
import {
  api_completePasswordReset,
  api_validatePasswordReset,
  api_verifyMFAPasswordReset,
} from "@/components/auth-view/helpers/api";
import { useRefreshSession } from "@/hooks/useAuth";
import { fetchApi } from "@/lib/api";

const PASSWORD_RESET_SUCCESS_DELAY = 2000;

export interface IPasswordFormState {
  register: UseFormRegister<IPasswordResetPasswordFormData>;
  handleSubmit: ReturnType<
    typeof useForm<IPasswordResetPasswordFormData>
  >["handleSubmit"];
  errors: FieldErrors<IPasswordResetPasswordFormData>;
  newPassword: string | undefined;
}

export interface IUsePasswordResetFlowReturn {
  stage: PasswordResetStage;
  error: string | null;
  isLoading: boolean;

  code: string;
  setCode: (code: string) => void;
  handleCodeSubmit: (e: React.FormEvent) => Promise<void>;

  mfaDevices: Array<IMFADevice>;
  mfaCode: string;
  setMfaCode: (code: string) => void;
  selectedDeviceId: string;
  setSelectedDeviceId: (id: string) => void;
  handleMFASubmit: (e: React.FormEvent) => Promise<void>;

  passwordForm: IPasswordFormState;
  handlePasswordSubmit: (data: IPasswordResetPasswordFormData) => Promise<void>;
}

export interface IUsePasswordResetFlowProps {
  challengeId: string;
}

export function usePasswordResetFlow({
  challengeId,
}: IUsePasswordResetFlowProps): IUsePasswordResetFlowReturn {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const refreshSession = useRefreshSession();

  const [stage, setStage] = useState<PasswordResetStage>("code");
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const [code, setCode] = useState("");

  const [mfaDevices, setMfaDevices] = useState<Array<IMFADevice>>([]);
  const [mfaCode, setMfaCode] = useState("");
  const [selectedDeviceId, setSelectedDeviceId] = useState<string>("");

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors },
  } = useForm<IPasswordResetPasswordFormData>();

  const newPassword = watch("newPassword");

  const handleCodeSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      setError(null);

      if (code.length !== 6) {
        setError(t("auth.password_reset.validate.error_code_length"));
        return;
      }

      setIsLoading(true);

      try {
        const response = await api_validatePasswordReset(challengeId, { code });

        if (response.mfa_required) {
          setStage("mfa");
        } else {
          setStage("password");
        }
      } catch {
        setError(t("auth.password_reset.validate.error_validation_failed"));
      } finally {
        setIsLoading(false);
      }
    },
    [challengeId, code, t],
  );

  useEffect(() => {
    if (stage === "mfa" && mfaDevices.length === 0) {
      const fetchDevices = async () => {
        try {
          const response = await fetchApi<IMFADevicesResponse>(`/mfa/devices`);
          setMfaDevices(response.devices);
          if (response.devices.length > 0) {
            const defaultDevice = response.devices.find((d) => d.is_default);
            setSelectedDeviceId(defaultDevice?.id ?? response.devices[0].id);
          }
        } catch {
          setError(t("auth.mfa.error_loading_devices"));
        }
      };
      fetchDevices();
    }
  }, [stage, mfaDevices.length, t]);

  const handleMFASubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      setError(null);

      if (mfaCode.length !== 6) {
        setError(t("auth.mfa.error_code_length"));
        return;
      }

      setIsLoading(true);

      try {
        await api_verifyMFAPasswordReset(
          mfaCode,
          selectedDeviceId || undefined,
        );
        setStage("password");
      } catch {
        setError(t("auth.mfa.error_verification_failed"));
      } finally {
        setIsLoading(false);
      }
    },
    [mfaCode, selectedDeviceId, t],
  );

  const handlePasswordSubmit = useCallback(
    async (data: IPasswordResetPasswordFormData) => {
      setError(null);

      if (data.newPassword !== data.confirmPassword) {
        setError(t("auth.password_reset.validate.error_password_mismatch"));
        return;
      }

      setIsLoading(true);

      try {
        await api_completePasswordReset(challengeId, {
          new_password: data.newPassword,
        });

        setStage("success");

        setTimeout(() => {
          void refreshSession();
          navigate({ to: "/" });
        }, PASSWORD_RESET_SUCCESS_DELAY);
      } catch {
        setError(t("auth.password_reset.validate.error_validation_failed"));
      } finally {
        setIsLoading(false);
      }
    },
    [challengeId, navigate, refreshSession, t],
  );

  return {
    stage,
    error,
    isLoading,

    code,
    setCode,
    handleCodeSubmit,

    mfaDevices,
    mfaCode,
    setMfaCode,
    selectedDeviceId,
    setSelectedDeviceId,
    handleMFASubmit,

    passwordForm: {
      register,
      handleSubmit,
      errors,
      newPassword,
    },
    handlePasswordSubmit,
  };
}
