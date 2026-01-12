import type { FC } from "react";
import { useState } from "react";

import { CheckCircle, Shield, Smartphone } from "lucide-react";
import { FormErrorAlert } from "@/components/common/FormErrorAlert";
import { useNavigate } from "@tanstack/react-router";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";

import type {
  IPasswordResetPasswordFormData,
  PasswordResetStage,
} from "@/components/auth-view/helpers/types";
import type { IMFADevice } from "@/components/mfa-view/helpers/types";
import {
  api_validatePasswordReset,
  api_completePasswordReset,
  api_verifyMFAPasswordReset,
} from "@/components/auth-view/helpers/api";
import { authCookies } from "@/lib/auth-service";
import { useRefreshSession } from "@/hooks/useAuth";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  InputOTP,
  InputOTPGroup,
  InputOTPSlot,
} from "@/components/ui/input-otp";
import { Label } from "@/components/ui/label";
import { MFADeviceSelector } from "@/components/mfa-view/components/MFADeviceSelector";

export interface IPasswordResetValidateFormProps {
  challengeId: string;
}

export const PasswordResetValidateForm: FC<IPasswordResetValidateFormProps> = ({
  challengeId,
}) => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const refreshSession = useRefreshSession();

  // Flow state
  const [stage, setStage] = useState<PasswordResetStage>("code");
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  // Code verification state
  const [code, setCode] = useState("");

  // MFA state
  const [mfaToken, setMfaToken] = useState<string | null>(null);
  const [mfaDevices, setMfaDevices] = useState<IMFADevice[]>([]);
  const [mfaCode, setMfaCode] = useState("");
  const [selectedDeviceId, setSelectedDeviceId] = useState<string>("");

  // Completion token state
  const [completionToken, setCompletionToken] = useState<string | null>(null);

  // Password form
  const {
    register,
    handleSubmit,
    watch,
    formState: { errors },
  } = useForm<IPasswordResetPasswordFormData>();

  const newPassword = watch("newPassword");

  // Stage 1: Code verification
  const handleCodeSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (code.length !== 6) {
      setError(t("auth.password_reset.validate.error_code_length"));
      return;
    }

    setIsLoading(true);

    try {
      const response = await api_validatePasswordReset(challengeId, { code });

      if (response.mfa_required && response.mfa_token) {
        // MFA required - move to MFA stage
        setMfaToken(response.mfa_token);
        setMfaDevices(response.devices || []);
        if (response.devices && response.devices.length > 0) {
          const defaultDevice = response.devices.find((d) => d.is_default);
          setSelectedDeviceId(defaultDevice?.id || response.devices[0].id);
        }
        setStage("mfa");
      } else if (response.completion_token) {
        // No MFA - move directly to password stage
        setCompletionToken(response.completion_token);
        setStage("password");
      }
    } catch {
      setError(t("auth.password_reset.validate.error_validation_failed"));
    } finally {
      setIsLoading(false);
    }
  };

  // Stage 2: MFA verification
  const handleMFASubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (mfaCode.length !== 6) {
      setError(t("auth.mfa.error_code_length"));
      return;
    }

    if (!mfaToken) {
      setError(t("auth.password_reset.validate.error_session_expired"));
      return;
    }

    setIsLoading(true);

    try {
      const response = await api_verifyMFAPasswordReset(
        mfaToken,
        mfaCode,
        selectedDeviceId || undefined,
      );

      if (response.password_reset && response.completion_token) {
        // MFA verified - move to password stage
        setCompletionToken(response.completion_token);
        setStage("password");
      } else {
        setError(t("auth.mfa.error_verification_failed"));
      }
    } catch {
      setError(t("auth.mfa.error_verification_failed"));
    } finally {
      setIsLoading(false);
    }
  };

  // Stage 3: Password submission
  const handlePasswordSubmit = async (data: IPasswordResetPasswordFormData) => {
    setError(null);

    if (data.newPassword !== data.confirmPassword) {
      setError(t("auth.password_reset.validate.error_password_mismatch"));
      return;
    }

    if (!completionToken) {
      setError(t("auth.password_reset.validate.error_session_expired"));
      return;
    }

    setIsLoading(true);

    try {
      const response = await api_completePasswordReset(challengeId, {
        completion_token: completionToken,
        new_password: data.newPassword,
      });

      // Set authentication state
      authCookies.setAll(
        response.access_token,
        response.refresh_token,
        "local",
      );

      setStage("success");

      // Navigate to home after a short delay
      setTimeout(() => {
        refreshSession();
        navigate({ to: "/" });
      }, 2000);
    } catch {
      setError(t("auth.password_reset.validate.error_validation_failed"));
    } finally {
      setIsLoading(false);
    }
  };

  // Success state
  if (stage === "success") {
    return (
      <Card className="mx-auto w-full max-w-md">
        <CardContent className="pt-6">
          <div className="space-y-4 text-center">
            <CheckCircle className="mx-auto h-12 w-12 text-green-500" />
            <h3 className="text-lg font-semibold">
              {t("auth.password_reset.validate.success_title")}
            </h3>
            <p className="text-muted-foreground text-sm">
              {t("auth.password_reset.validate.success_message")}
            </p>
          </div>
        </CardContent>
      </Card>
    );
  }

  // MFA verification stage
  if (stage === "mfa") {
    return (
      <Card className="mx-auto w-full max-w-md">
        <CardHeader className="text-center">
          <div className="mx-auto mb-4 h-12 w-12 rounded-full bg-blue-100 p-3">
            <Smartphone className="h-6 w-6 text-blue-600" />
          </div>
          <CardTitle>{t("auth.password_reset.mfa.title")}</CardTitle>
          <CardDescription>
            {t("auth.password_reset.mfa.subtitle")}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleMFASubmit} className="space-y-4">
            <FormErrorAlert error={error} />

            {mfaDevices.length > 1 && (
              <MFADeviceSelector
                devices={mfaDevices}
                selectedDeviceId={selectedDeviceId}
                onSelectDevice={setSelectedDeviceId}
                disabled={isLoading}
              />
            )}

            <div className="space-y-2">
              <p className="text-muted-foreground text-center text-sm">
                {t("auth.mfa.code_instruction")}
              </p>
              <div className="flex justify-center">
                <InputOTP
                  maxLength={6}
                  value={mfaCode}
                  onChange={(value) => setMfaCode(value)}
                  disabled={isLoading}
                >
                  <InputOTPGroup>
                    <InputOTPSlot index={0} />
                    <InputOTPSlot index={1} />
                    <InputOTPSlot index={2} />
                    <InputOTPSlot index={3} />
                    <InputOTPSlot index={4} />
                    <InputOTPSlot index={5} />
                  </InputOTPGroup>
                </InputOTP>
              </div>
            </div>

            <Button
              type="submit"
              className="w-full"
              disabled={isLoading || mfaCode.length !== 6}
            >
              {isLoading
                ? t("auth.mfa.verifying")
                : t("auth.mfa.verify_button")}
            </Button>
          </form>
        </CardContent>
      </Card>
    );
  }

  // Password entry stage
  if (stage === "password") {
    return (
      <Card className="mx-auto w-full max-w-md">
        <CardHeader className="text-center">
          <div className="mx-auto mb-4 h-12 w-12 rounded-full bg-green-100 p-3">
            <Shield className="h-6 w-6 text-green-600" />
          </div>
          <CardTitle>{t("auth.password_reset.password.title")}</CardTitle>
          <CardDescription>
            {t("auth.password_reset.password.subtitle")}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form
            onSubmit={handleSubmit(handlePasswordSubmit)}
            className="space-y-4"
          >
            <FormErrorAlert error={error} />

            <div className="space-y-2">
              <Label htmlFor="newPassword">
                {t("auth.password_reset.validate.new_password_label")}
              </Label>
              <Input
                id="newPassword"
                type="password"
                placeholder={t(
                  "auth.password_reset.validate.new_password_placeholder",
                )}
                {...register("newPassword", {
                  required: t(
                    "auth.password_reset.validate.error_new_password_required",
                  ),
                  minLength: {
                    value: 8,
                    message: t(
                      "auth.password_reset.validate.error_new_password_min_length",
                    ),
                  },
                })}
                className={errors.newPassword ? "border-red-500" : ""}
                disabled={isLoading}
              />
              {errors.newPassword && (
                <p className="text-sm text-red-500">
                  {errors.newPassword.message}
                </p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="confirmPassword">
                {t("auth.password_reset.validate.confirm_password_label")}
              </Label>
              <Input
                id="confirmPassword"
                type="password"
                placeholder={t(
                  "auth.password_reset.validate.confirm_password_placeholder",
                )}
                {...register("confirmPassword", {
                  required: t(
                    "auth.password_reset.validate.error_confirm_password_required",
                  ),
                  validate: (value) =>
                    value === newPassword ||
                    t(
                      "auth.password_reset.validate.error_confirm_password_mismatch",
                    ),
                })}
                className={errors.confirmPassword ? "border-red-500" : ""}
                disabled={isLoading}
              />
              {errors.confirmPassword && (
                <p className="text-sm text-red-500">
                  {errors.confirmPassword.message}
                </p>
              )}
            </div>

            <Button type="submit" className="w-full" disabled={isLoading}>
              {isLoading
                ? t("auth.password_reset.validate.resetting")
                : t("auth.password_reset.validate.reset_button")}
            </Button>
          </form>
        </CardContent>
      </Card>
    );
  }

  // Code verification stage (default)
  return (
    <Card className="mx-auto w-full max-w-md">
      <CardHeader className="text-center">
        <div className="mx-auto mb-4 h-12 w-12 rounded-full bg-red-100 p-3">
          <Shield className="h-6 w-6 text-red-600" />
        </div>
        <CardTitle>{t("auth.password_reset.code.title")}</CardTitle>
        <CardDescription>
          {t("auth.password_reset.code.subtitle")}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleCodeSubmit} className="space-y-4">
          <FormErrorAlert error={error} />

          <div className="space-y-2">
            <Label className="flex justify-center" htmlFor="code">
              {t("auth.password_reset.validate.code_label")}
            </Label>
            <div className="flex justify-center">
              <InputOTP
                maxLength={6}
                value={code}
                onChange={(value) => setCode(value)}
                disabled={isLoading}
              >
                <InputOTPGroup>
                  <InputOTPSlot index={0} />
                  <InputOTPSlot index={1} />
                  <InputOTPSlot index={2} />
                  <InputOTPSlot index={3} />
                  <InputOTPSlot index={4} />
                  <InputOTPSlot index={5} />
                </InputOTPGroup>
              </InputOTP>
            </div>
          </div>

          <Button
            type="submit"
            className="w-full"
            disabled={isLoading || code.length !== 6}
          >
            {isLoading
              ? t("auth.password_reset.code.verifying")
              : t("auth.password_reset.code.verify_button")}
          </Button>

          <p className="text-muted-foreground mt-3 text-center text-xs">
            {t("auth.password_reset.validate.footer_text")}
          </p>
        </form>
      </CardContent>
    </Card>
  );
};
