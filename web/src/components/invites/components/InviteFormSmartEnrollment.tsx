import { useState } from "react";

import { AlertCircle, Mail } from "lucide-react";
import { useNavigate } from "@tanstack/react-router";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { useSuspenseQuery } from "@tanstack/react-query";
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
import { authProvidersQueryOptions } from "@/queries/auth_providers.ts";
import { ProviderType } from "@/types/auth_providers.ts";
import { useSessionContext } from "@/components/auth-view/hooks/useSessionContext.ts";
import { checkEmailDomain } from "@/components/reset-password/helpers/utils.ts";
import { AuthProvidersButtons } from "@/components/auth-providers-buttons/AuthProvidersButtons.tsx";

interface ISmartInviteEnrollmentData {
  email: string;
}

interface ISmartInviteEnrollmentProps {
  invitationId: string;
}

export const InviteFormSmartEnrollment: FC<ISmartInviteEnrollmentProps> = ({
  invitationId,
}) => {
  const { login } = useSessionContext();
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [error, setError] = useState<string | null>(null);

  const providersQuery = useSuspenseQuery(authProvidersQueryOptions());
  const providers = providersQuery.data;

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<ISmartInviteEnrollmentData>();

  const emailValue = watch("email") || "";

  const handleContinue = (data: ISmartInviteEnrollmentData) => {
    setError(null);
    const provider = checkEmailDomain(data.email, providers);
    if (provider) {
      login(provider.id);
    } else {
      api_createChallenge(invitationId, data.email)
        .then((res) =>
          navigate({ to: `/invites/${invitationId}/challenges/${res.id}` }),
        )
        .catch(() => setError(t("invites.smart_enrollment.send_error")));
    }
  };

  return (
    <Card className="mx-auto w-full max-w-md">
      <CardHeader className="text-center">
        <div className="mx-auto mb-4 h-12 w-12 rounded-full bg-blue-100 p-3">
          <Mail className="h-6 w-6 text-blue-600" />
        </div>
        <CardTitle>{t("invites.smart_enrollment.title")}</CardTitle>
        <CardDescription>
          {t("invites.smart_enrollment.description")}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <AuthProvidersButtons
          providers={providers.filter((p) => p.type === ProviderType.OIDC)}
        />

        {providers.find((p) => p.type === ProviderType.LOCAL) && (
          <>
            <div className="relative">
              <div className="absolute inset-0 flex items-center">
                <span className="w-full border-t" />
              </div>
              <div className="relative flex justify-center text-xs uppercase">
                <span className="bg-primary-foreground text-muted-foreground px-2">
                  {t("auth.or_continue_with")}
                </span>
              </div>
            </div>
            <form onSubmit={handleSubmit(handleContinue)} className="space-y-4">
              {error && (
                <div className="flex items-center space-x-2 rounded-md bg-red-50 p-3 text-red-600">
                  <AlertCircle className="h-4 w-4" />
                  <span className="text-sm">{error}</span>
                </div>
              )}

              <div className="space-y-2">
                <Label htmlFor="email">
                  {t("invites.smart_enrollment.email_label")}
                </Label>
                <Input
                  id="email"
                  type="email"
                  placeholder={t("invites.smart_enrollment.email_placeholder")}
                  {...register("email", {
                    required: t("invites.smart_enrollment.email_required"),
                    pattern: {
                      value: /^[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}$/i,
                      message: t("invites.smart_enrollment.email_invalid"),
                    },
                  })}
                  className={errors.email ? "border-red-500" : ""}
                />
                {errors.email && (
                  <p className="text-sm text-red-500">{errors.email.message}</p>
                )}
              </div>

              <Button
                type="submit"
                className="w-full"
                disabled={isSubmitting || !emailValue.trim()}
              >
                {isSubmitting
                  ? t("invites.smart_enrollment.sending_button")
                  : t("auth.continue")}
              </Button>

              <p className="text-muted-foreground text-center text-xs">
                {t("invites.smart_enrollment.footer_text")}
              </p>
            </form>
          </>
        )}
      </CardContent>
    </Card>
  );
};
