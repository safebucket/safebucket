import { Link, createFileRoute, useNavigate } from "@tanstack/react-router";

import { useState } from "react";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { LogIn } from "lucide-react";
import { useForm } from "react-hook-form";
import type { SubmitHandler } from "react-hook-form";
import type { FormEvent } from "react";
import type { ILoginForm } from "@/components/auth-view/types/session";
import { useLogin } from "@/hooks/useAuth";
import { useMFAAuth } from "@/context/MFAAuthContext";
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
import { checkEmailDomain } from "@/components/reset-password/helpers/utils.ts";
import { AuthProvidersButtons } from "@/components/auth-providers-buttons/AuthProvidersButtons.tsx";

export const Route = createFileRoute("/auth/login/")({
  validateSearch: (search: Record<string, unknown>) => {
    return {
      redirect: (search.redirect as string) || undefined,
    };
  },
  loader: ({ context: { queryClient } }) =>
    queryClient.ensureQueryData(authProvidersQueryOptions()),
  component: Login,
});

function Login() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { redirect } = Route.useSearch();
  const providersQuery = useSuspenseQuery(authProvidersQueryOptions());
  const providers = providersQuery.data;

  const { loginOAuth, loginLocal } = useLogin();
  const { setMFAAuth } = useMFAAuth();
  const { register, handleSubmit, watch } = useForm<ILoginForm>();
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const emailValue = watch("email") || "";

  const handleContinue = (event: FormEvent) => {
    event.preventDefault();
    const formData = new FormData(event.target as HTMLFormElement);
    const email = formData.get("email") as string;

    if (!email) return;
    if (!email.includes("@")) return;

    const matchingProvider = checkEmailDomain(email, providers);
    if (matchingProvider) {
      loginOAuth(matchingProvider.id);
    } else {
      setShowPassword(true);
    }
  };

  const handleLocalLogin: SubmitHandler<ILoginForm> = async (data) => {
    setIsLoading(true);
    setError(null);

    const result = await loginLocal(data);

    if (result.mfaRequired && result.mfaToken && result.userId) {
      // Store MFA token, user ID, and devices in memory context
      setMFAAuth(result.mfaToken, result.userId, result.devices || []);
      // Redirect to MFA verification
      navigate({
        to: "/auth/mfa",
        search: { redirect },
      });
    } else if (result.mfaSetupRequired && result.mfaToken && result.userId) {
      // Store MFA token and user ID in memory context (used to skip password during setup)
      setMFAAuth(result.mfaToken, result.userId, []);
      // Redirect to MFA setup (admin requires MFA but not set up)
      navigate({ to: "/auth/mfa/setup-required", search: { redirect } });
    } else if (result.success) {
      // Navigate to redirect or home
      navigate({ to: redirect || "/" });
    } else {
      setError(result.error || t("auth.login_error"));
    }

    setIsLoading(false);
  };

  return (
    <div className="m-6 flex h-full items-center justify-center">
      <Card className="mx-auto w-full max-w-md">
        <CardHeader className="text-center">
          <div className="mx-auto mb-4 h-12 w-12 rounded-full bg-blue-100 p-3">
            <LogIn className="h-6 w-6 text-blue-600" />
          </div>
          <CardTitle>{t("auth.sign_in_title")}</CardTitle>
          <CardDescription>{t("auth.sign_in_subtitle")}</CardDescription>
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

              <form
                onSubmit={
                  showPassword ? handleSubmit(handleLocalLogin) : handleContinue
                }
              >
                <div className="grid gap-2">
                  <Label htmlFor="email">{t("auth.email")}</Label>
                  <Input
                    id="email"
                    type="email"
                    placeholder={t("auth.email_placeholder")}
                    {...register("email", { required: true })}
                    disabled={isLoading}
                  />
                </div>

                {showPassword && (
                  <div className="grid gap-2 mt-4">
                    <div className="flex items-center justify-between">
                      <Label htmlFor="password">{t("auth.password")}</Label>
                      <Link
                        to="/auth/reset-password"
                        className="text-primary text-sm font-medium hover:underline"
                      >
                        {t("auth.forgot_password")}
                      </Link>
                    </div>
                    <Input
                      id="password"
                      type="password"
                      {...register("password", {
                        required: showPassword,
                      })}
                      disabled={isLoading}
                    />
                  </div>
                )}

                {error && (
                  <div className="text-sm text-red-600 mt-2">{error}</div>
                )}

                <Button
                  type="submit"
                  className="w-full mt-4"
                  disabled={
                    isLoading ||
                    !emailValue.trim() ||
                    (showPassword && !watch("password"))
                  }
                >
                  {isLoading
                    ? t("auth.signing_in")
                    : showPassword
                      ? t("auth.sign_in")
                      : t("auth.continue")}
                </Button>
              </form>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
