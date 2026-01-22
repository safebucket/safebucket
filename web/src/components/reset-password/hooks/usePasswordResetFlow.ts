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
} from "@/components/mfa-view/helpers/types";
import {
  api_completePasswordReset,
  api_validatePasswordReset,
  api_verifyMFAPasswordReset,
} from "@/components/auth-view/helpers/api";
import { useRefreshSession } from "@/hooks/useAuth";
import { fetchApi } from "@/lib/api";
import { authCookies, decodeToken } from "@/lib/auth-service";

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
  // Stage
  stage: PasswordResetStage;
  error: string | null;
  isLoading: boolean;

  // Code stage
  code: string;
  setCode: (code: string) => void;
  handleCodeSubmit: (e: React.FormEvent) => Promise<void>;

  // MFA stage
  mfaDevices: Array<IMFADevice>;
  mfaCode: string;
  setMfaCode: (code: string) => void;
  selectedDeviceId: string;
  setSelectedDeviceId: (id: string) => void;
  handleMFASubmit: (e: React.FormEvent) => Promise<void>;

  // Password stage (react-hook-form)
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

  // Stage management
  const [stage, setStage] = useState<PasswordResetStage>("code");
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  // Code stage state
  const [code, setCode] = useState("");

  // Restricted access token (from code validation, used for MFA and completion)
  const [restrictedToken, setRestrictedToken] = useState<string | null>(null);
  const [userId, setUserId] = useState<string | null>(null);

  // MFA stage state
  const [mfaDevices, setMfaDevices] = useState<Array<IMFADevice>>([]);
  const [mfaCode, setMfaCode] = useState("");
  const [selectedDeviceId, setSelectedDeviceId] = useState<string>("");

  // Password form (react-hook-form)
  const {
    register,
    handleSubmit,
    watch,
    formState: { errors },
  } = useForm<IPasswordResetPasswordFormData>();

  const newPassword = watch("newPassword");

  // Code submission handler
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

        setRestrictedToken(response.access_token);

        const decoded = decodeToken(response.access_token);
        if (decoded) {
          setUserId(decoded.payload.user_id);
        }

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

  // Fetch MFA devices when entering MFA stage
  useEffect(() => {
    if (
      stage === "mfa" &&
      userId &&
      restrictedToken &&
      mfaDevices.length === 0
    ) {
      const fetchDevices = async () => {
        try {
          const response = await fetchApi<IMFADevicesResponse>(`/mfa/devices`, {
            headers: { Authorization: `Bearer ${restrictedToken}` },
          });
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
  }, [stage, userId, restrictedToken, mfaDevices.length, t]);

  // MFA submission handler
  const handleMFASubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      setError(null);

      if (mfaCode.length !== 6) {
        setError(t("auth.mfa.error_code_length"));
        return;
      }

      if (!restrictedToken) {
        setError(t("auth.password_reset.validate.error_session_expired"));
        return;
      }

      setIsLoading(true);

      try {
        const response = await api_verifyMFAPasswordReset(
          restrictedToken,
          mfaCode,
          selectedDeviceId || undefined,
        );

        setRestrictedToken(response.access_token);
        setStage("password");
      } catch {
        setError(t("auth.mfa.error_verification_failed"));
      } finally {
        setIsLoading(false);
      }
    },
    [mfaCode, restrictedToken, selectedDeviceId, t],
  );

  // Password submission handler
  const handlePasswordSubmit = useCallback(
    async (data: IPasswordResetPasswordFormData) => {
      setError(null);

      if (data.newPassword !== data.confirmPassword) {
        setError(t("auth.password_reset.validate.error_password_mismatch"));
        return;
      }

      if (!restrictedToken) {
        setError(t("auth.password_reset.validate.error_session_expired"));
        return;
      }

      setIsLoading(true);

      try {
        const response = await api_completePasswordReset(
          challengeId,
          { new_password: data.newPassword },
          restrictedToken,
        );

        authCookies.setAll(
          response.access_token,
          response.refresh_token,
          "local",
        );

        setStage("success");

        setTimeout(() => {
          refreshSession();
          navigate({ to: "/" });
        }, PASSWORD_RESET_SUCCESS_DELAY);
      } catch {
        setError(t("auth.password_reset.validate.error_validation_failed"));
      } finally {
        setIsLoading(false);
      }
    },
    [challengeId, navigate, refreshSession, restrictedToken, t],
  );

  return {
    // Stage
    stage,
    error,
    isLoading,

    // Code stage
    code,
    setCode,
    handleCodeSubmit,

    // MFA stage
    mfaDevices,
    mfaCode,
    setMfaCode,
    selectedDeviceId,
    setSelectedDeviceId,
    handleMFASubmit,

    // Password stage
    passwordForm: {
      register,
      handleSubmit,
      errors,
      newPassword,
    },
    handlePasswordSubmit,
  };
}
