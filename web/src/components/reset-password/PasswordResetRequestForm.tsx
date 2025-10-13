import { useState } from "react";

import { AlertCircle, Key, Mail } from "lucide-react";
import { Link } from "@tanstack/react-router";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import type { FC } from "react";

import type { IPasswordResetRequestFormData } from "@/components/auth-view/helpers/types.ts";
import { api_requestPasswordReset } from "@/components/auth-view/helpers/api.ts";
import { Button } from "@/components/ui/button.tsx";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card.tsx";
import { Input } from "@/components/ui/input.tsx";
import { Label } from "@/components/ui/label.tsx";

export const PasswordResetRequestForm: FC = () => {
  const { t } = useTranslation();
  const [isSuccess, setIsSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<IPasswordResetRequestFormData>();

  const handleFormSubmit = async (data: IPasswordResetRequestFormData) => {
    setError(null);

    try {
      await api_requestPasswordReset(data);
      setIsSuccess(true);
    } catch {
      setError(t("auth.password_reset.error_message"));
    }
  };

  if (isSuccess) {
    return (
      <Card className="mx-auto w-full max-w-md">
        <CardHeader className="text-center">
          <div className="mx-auto mb-4 h-12 w-12 rounded-full bg-green-100 p-3">
            <Mail className="h-6 w-6 text-green-600" />
          </div>
          <CardTitle>{t("auth.password_reset.success_title")}</CardTitle>
          <CardDescription>
            {t("auth.password_reset.success_message")}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <p className="text-muted-foreground text-center text-sm">
              {t("auth.password_reset.success_description")}
            </p>
            <Link to="/auth/login">
              <Button variant="outline" className="w-full">
                {t("auth.password_reset.back_to_login")}
              </Button>
            </Link>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="mx-auto w-full max-w-md">
      <CardHeader className="text-center">
        <div className="mx-auto mb-4 h-12 w-12 rounded-full bg-blue-100 p-3">
          <Key className="h-6 w-6 text-blue-600" />
        </div>
        <CardTitle>{t("auth.password_reset.title")}</CardTitle>
        <CardDescription>{t("auth.password_reset.subtitle")}</CardDescription>
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
            <Label htmlFor="email">{t("auth.email")}</Label>
            <Input
              id="email"
              type="email"
              placeholder={t("auth.email_placeholder")}
              {...register("email", {
                required: "Email is required",
                pattern: {
                  value: /^[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}$/i,
                  message: "Please enter a valid email address",
                },
              })}
              className={errors.email ? "border-red-500" : ""}
            />
            {errors.email && (
              <p className="text-sm text-red-500">{errors.email.message}</p>
            )}
          </div>

          <Button type="submit" className="w-full" disabled={isSubmitting}>
            {isSubmitting
              ? t("auth.password_reset.sending")
              : t("auth.password_reset.send_reset_code")}
          </Button>

          <div className="text-center">
            <Link
              to="/auth/login"
              className="text-muted-foreground hover:text-primary text-sm underline"
            >
              {t("auth.password_reset.back_to_login")}
            </Link>
          </div>
        </form>
      </CardContent>
    </Card>
  );
};
