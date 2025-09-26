import { useState } from "react";

import { AlertCircle, Mail } from "lucide-react";
import { useNavigate } from "@tanstack/react-router";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import type { FC } from "react";

import { api_createChallenge } from "@/components/invites/helpers/api";
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

interface IEmailConfirmationFormData {
  email: string;
}

interface IEmailConfirmationFormProps {
  invitationId: string;
}

export const EmailConfirmationForm: FC<IEmailConfirmationFormProps> = ({
  invitationId,
}) => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [error, setError] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<IEmailConfirmationFormData>();

  const handleFormSubmit = async (data: IEmailConfirmationFormData) => {
    setError(null);

    api_createChallenge(invitationId, data.email)
      .then((res) =>
        navigate({ to: `/invites/${invitationId}/challenges/${res.id}` }),
      )
      .catch(() =>
        setError(t("invites.email_confirmation.send_error")),
      );
  };

  return (
    <Card className="mx-auto w-full max-w-md">
      <CardHeader className="text-center">
        <div className="mx-auto mb-4 h-12 w-12 rounded-full bg-blue-100 p-3">
          <Mail className="h-6 w-6 text-blue-600" />
        </div>
        <CardTitle>{t("invites.email_confirmation.title")}</CardTitle>
        <CardDescription>
          {t("invites.email_confirmation.description")}
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
            <Label htmlFor="email">{t("invites.email_confirmation.email_label")}</Label>
            <Input
              id="email"
              type="email"
              placeholder={t("invites.email_confirmation.email_placeholder")}
              {...register("email", {
                required: t("invites.email_confirmation.email_required"),
                pattern: {
                  value: /^[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}$/i,
                  message: t("invites.email_confirmation.email_invalid"),
                },
              })}
              className={errors.email ? "border-red-500" : ""}
            />
            {errors.email && (
              <p className="text-sm text-red-500">{errors.email.message}</p>
            )}
          </div>

          <Button type="submit" className="w-full" disabled={isSubmitting}>
            {isSubmitting ? t("invites.email_confirmation.sending_button") : t("invites.email_confirmation.send_button")}
          </Button>

          <p className="text-muted-foreground mt-3 text-center text-xs">
            {t("invites.email_confirmation.footer_text")}
          </p>
        </form>
      </CardContent>
    </Card>
  );
};
