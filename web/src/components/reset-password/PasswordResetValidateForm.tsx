import { CheckCircle, Shield, Smartphone } from "lucide-react";
import { useTranslation } from "react-i18next";
import { usePasswordResetFlow } from "./hooks/usePasswordResetFlow";
import type { FC } from "react";

import { FormErrorAlert } from "@/components/common/FormErrorAlert";
import { MFADeviceSelector } from "@/components/mfa-view/components/MFADeviceSelector";
import { MFAVerifyInput } from "@/components/mfa-view/components/MFAVerifyInput";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export interface IPasswordResetValidateFormProps {
  challengeId: string;
}

export const PasswordResetValidateForm: FC<IPasswordResetValidateFormProps> = ({
  challengeId,
}) => {
  const { t } = useTranslation();

  const {
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
    passwordForm,
    handlePasswordSubmit,
  } = usePasswordResetFlow({ challengeId });

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
              <MFAVerifyInput
                value={mfaCode}
                onChange={setMfaCode}
                disabled={isLoading}
              />
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
            onSubmit={passwordForm.handleSubmit(handlePasswordSubmit)}
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
                {...passwordForm.register("newPassword", {
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
                className={
                  passwordForm.errors.newPassword ? "border-red-500" : ""
                }
                disabled={isLoading}
              />
              {passwordForm.errors.newPassword && (
                <p className="text-sm text-red-500">
                  {passwordForm.errors.newPassword.message}
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
                {...passwordForm.register("confirmPassword", {
                  required: t(
                    "auth.password_reset.validate.error_confirm_password_required",
                  ),
                  validate: (value) =>
                    value === passwordForm.newPassword ||
                    t(
                      "auth.password_reset.validate.error_confirm_password_mismatch",
                    ),
                })}
                className={
                  passwordForm.errors.confirmPassword ? "border-red-500" : ""
                }
                disabled={isLoading}
              />
              {passwordForm.errors.confirmPassword && (
                <p className="text-sm text-red-500">
                  {passwordForm.errors.confirmPassword.message}
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
            <MFAVerifyInput
              value={code}
              onChange={setCode}
              disabled={isLoading}
              uppercase
            />
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
