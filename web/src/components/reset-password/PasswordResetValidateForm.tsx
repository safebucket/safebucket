import { useState } from "react";

import { CheckCircle, Shield } from "lucide-react";
import { useNavigate } from "@tanstack/react-router";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import type { FC } from "react";

import type { IPasswordResetValidateFormData } from "@/components/auth-view/helpers/types.ts";
import { FormErrorAlert } from "@/components/common/FormErrorAlert";
import { api_validatePasswordReset } from "@/components/auth-view/helpers/api.ts";
import { authCookies } from "@/lib/auth-service";
import { useRefreshSession } from "@/hooks/useAuth";
import { Button } from "@/components/ui/button.tsx";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card.tsx";
import { Input } from "@/components/ui/input.tsx";
import {
  InputOTP,
  InputOTPGroup,
  InputOTPSlot,
} from "@/components/ui/input-otp.tsx";
import { Label } from "@/components/ui/label.tsx";

export interface IPasswordResetValidateFormProps {
  challengeId: string;
}

export const PasswordResetValidateForm: FC<IPasswordResetValidateFormProps> = ({
  challengeId,
}) => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const refreshSession = useRefreshSession();
  const [isValidated, setIsValidated] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [code, setCode] = useState("");

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<IPasswordResetValidateFormData>();

  const newPassword = watch("newPassword");

  const handleFormSubmit = async (data: IPasswordResetValidateFormData) => {
    setError(null);

    if (code.length !== 6) {
      setError(t("auth.password_reset.validate.error_code_length"));
      return;
    }

    if (data.newPassword !== data.confirmPassword) {
      setError(t("auth.password_reset.validate.error_password_mismatch"));
      return;
    }

    try {
      const response = await api_validatePasswordReset(challengeId, {
        code,
        new_password: data.newPassword,
      });

      // Set authentication state via auth service
      authCookies.setAll(
        response.access_token,
        response.refresh_token,
        "local",
      );

      setIsValidated(true);

      // Navigate to home after a short delay
      setTimeout(() => {
        refreshSession();
        navigate({ to: "/" });
      }, 2000);
    } catch {
      setError(t("auth.password_reset.validate.error_validation_failed"));
    }
  };

  if (isValidated) {
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

  return (
    <Card className="mx-auto w-full max-w-md">
      <CardHeader className="text-center">
        <div className="mx-auto mb-4 h-12 w-12 rounded-full bg-red-100 p-3">
          <Shield className="h-6 w-6 text-red-600" />
        </div>
        <CardTitle>{t("auth.password_reset.validate.title")}</CardTitle>
        <CardDescription>
          {t("auth.password_reset.validate.subtitle")}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit(handleFormSubmit)} className="space-y-4">
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
            />
            {errors.confirmPassword && (
              <p className="text-sm text-red-500">
                {errors.confirmPassword.message}
              </p>
            )}
          </div>

          <Button
            type="submit"
            className="w-full"
            disabled={isSubmitting || code.length !== 6}
          >
            {isSubmitting
              ? t("auth.password_reset.validate.resetting")
              : t("auth.password_reset.validate.reset_button")}
          </Button>

          <p className="text-muted-foreground mt-3 text-center text-xs">
            {t("auth.password_reset.validate.footer_text")}
          </p>
        </form>
      </CardContent>
    </Card>
  );
};
