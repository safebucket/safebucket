import type { FC } from "react";
import { useState } from "react";

import { CheckCircle, Mail } from "lucide-react";
import { FormErrorAlert } from "@/components/common/FormErrorAlert";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { useSuspenseQuery } from "@tanstack/react-query";

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
import { useLogin } from "@/hooks/useAuth";
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
  const { loginOAuth } = useLogin();
  const { t } = useTranslation();
  const [error, setError] = useState<string | null>(null);
  const [isSuccess, setIsSuccess] = useState(false);

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
      loginOAuth(provider.id);
    } else {
      api_createChallenge(invitationId, data.email)
        .then(() => setIsSuccess(true))
        .catch(() => setError(t("invites.smart_enrollment.send_error")));
    }
  };

  if (isSuccess) {
    return (
      <Card className="mx-auto w-full max-w-md">
        <CardHeader className="text-center">
          <div className="mx-auto mb-4 h-12 w-12 rounded-full bg-green-100 p-3">
            <CheckCircle className="h-6 w-6 text-green-600" />
          </div>
          <CardTitle>{t("invites.smart_enrollment.success_title")}</CardTitle>
          <CardDescription>
            {t("invites.smart_enrollment.success_message")}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-muted-foreground text-center text-sm">
            {t("invites.smart_enrollment.success_description")}
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="mx-auto w-full max-w-md">
      <CardHeader className="text-center">
        <div className="mx-auto mb-4 h-12 w-12 rounded-full bg-purple-100 p-3">
          <Mail className="h-6 w-6 text-purple-600" />
        </div>
        <CardTitle>{t("invites.smart_enrollment.title")}</CardTitle>
        <CardDescription>
          {t("invites.smart_enrollment.description")}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-2">
        <AuthProvidersButtons
          providers={providers.filter((p) => p.type === ProviderType.OIDC)}
        />

        {providers.find((p) => p.type === ProviderType.LOCAL) && (
          <>
            {providers.filter((p) => p.type === ProviderType.OIDC).length >
              0 && (
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
            )}

            <form onSubmit={handleSubmit(handleContinue)} className="space-y-4">
              <FormErrorAlert error={error} />

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
