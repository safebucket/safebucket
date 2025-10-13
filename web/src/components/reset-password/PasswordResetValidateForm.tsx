import { useState } from "react";

import Cookies from "js-cookie";
import { AlertCircle, CheckCircle, Shield } from "lucide-react";
import { useNavigate } from "@tanstack/react-router";
import { useForm } from "react-hook-form";
import type { FC } from "react";

import type { IPasswordResetValidateFormData } from "@/components/auth-view/helpers/types.ts";
import { api_validatePasswordReset } from "@/components/auth-view/helpers/api.ts";
import { useSessionContext } from "@/components/auth-view/hooks/useSessionContext";
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
  const navigate = useNavigate();
  const { setAuthenticationState } = useSessionContext();
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
      setError("Verification code must be exactly 6 digits");
      return;
    }

    if (data.newPassword !== data.confirmPassword) {
      setError("Passwords do not match");
      return;
    }

    try {
      const response = await api_validatePasswordReset(challengeId, {
        code,
        new_password: data.newPassword,
      });

      Cookies.set("safebucket_access_token", response.access_token);
      Cookies.set("safebucket_refresh_token", response.refresh_token);
      Cookies.set("safebucket_auth_provider", "local");

      setIsValidated(true);

      // Navigate to home after a short delay
      setTimeout(() => {
        setAuthenticationState(response.access_token, response.refresh_token, "local");
        navigate({ to: "/" });
      }, 2000);
    } catch {
      setError(
        "Invalid verification code or failed to reset password. Please try again.",
      );
    }
  };

  if (isValidated) {
    return (
      <Card className="mx-auto w-full max-w-md">
        <CardContent className="pt-6">
          <div className="space-y-4 text-center">
            <CheckCircle className="mx-auto h-12 w-12 text-green-500" />
            <h3 className="text-lg font-semibold">Password Reset Successful</h3>
            <p className="text-muted-foreground text-sm">
              Your password has been reset successfully. You are now logged in
              and will be redirected to the homepage.
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
        <CardTitle>Reset your password</CardTitle>
        <CardDescription>
          Enter the 6-digit verification code from your email and choose a new
          password
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit(handleFormSubmit)} className="space-y-4">
          {error && (
            <div className="flex items-center space-x-2 rounded-md bg-red-50 p-3 text-red-600">
              <AlertCircle className="h-4 w-4" />
              <span className="text-sm">{error}</span>
            </div>
          )}

          <div className="space-y-2">
            <Label className="flex justify-center" htmlFor="code">
              6-digit verification code
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
            <Label htmlFor="newPassword">New password</Label>
            <Input
              id="newPassword"
              type="password"
              placeholder="Enter new password"
              {...register("newPassword", {
                required: "New password is required",
                minLength: {
                  value: 8,
                  message: "Password must be at least 8 characters",
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
            <Label htmlFor="confirmPassword">Confirm new password</Label>
            <Input
              id="confirmPassword"
              type="password"
              placeholder="Confirm new password"
              {...register("confirmPassword", {
                required: "Please confirm your password",
                validate: (value) =>
                  value === newPassword || "Passwords do not match",
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
            {isSubmitting ? "Resetting password..." : "Reset password"}
          </Button>

          <p className="text-muted-foreground mt-3 text-center text-xs">
            Didn&apos;t receive the code? Check your spam folder or request a
            new one
          </p>
        </form>
      </CardContent>
    </Card>
  );
};
