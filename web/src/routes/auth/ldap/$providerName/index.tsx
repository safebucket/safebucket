import { Link, createFileRoute, useNavigate } from "@tanstack/react-router";

import { useState } from "react";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { LogIn } from "lucide-react";
import { useForm } from "react-hook-form";
import type { SubmitHandler } from "react-hook-form";
import type { ILoginForm } from "@/components/auth-view/types/session";
import { useLogin } from "@/hooks/useAuth";
import { mfaPending } from "@/components/mfa-view/helpers/token";
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

export const Route = createFileRoute("/auth/ldap/$providerName/")({
  validateSearch: (search: Record<string, unknown>) => {
    return {
      redirect: (search.redirect as string) || undefined,
    };
  },
  loader: ({ context: { queryClient } }) =>
    queryClient.ensureQueryData(authProvidersQueryOptions()),
  component: LDAPLogin,
});

function LDAPLogin() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { providerName } = Route.useParams();
  const { redirect } = Route.useSearch();
  const providersQuery = useSuspenseQuery(authProvidersQueryOptions());
  const provider = providersQuery.data.find(
    (p) => p.id === providerName && p.type === ProviderType.LDAP,
  );

  const { loginLDAP } = useLogin();
  const { register, handleSubmit, watch } = useForm<ILoginForm>();
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  if (!provider) {
    void navigate({ to: "/auth/login", search: { redirect } });
    return null;
  }

  const handleLogin: SubmitHandler<ILoginForm> = async (data) => {
    setIsLoading(true);
    setError(null);

    const result = await loginLDAP(provider.id, data);

    if (result.mfaRequired) {
      mfaPending.set();
      navigate({
        to: "/auth/mfa",
        search: { redirect },
      });
    } else if (result.success) {
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
          <CardTitle>
            {t("auth.ldap_sign_in_title", { name: provider.name })}
          </CardTitle>
          <CardDescription>{t("auth.ldap_sign_in_subtitle")}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-2">
          <form onSubmit={handleSubmit(handleLogin)}>
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

            <div className="grid gap-2 mt-4">
              <Label htmlFor="password">{t("auth.password")}</Label>
              <Input
                id="password"
                type="password"
                {...register("password", { required: true })}
                disabled={isLoading}
              />
            </div>

            {error && <div className="text-sm text-red-600 mt-2">{error}</div>}

            <Button
              type="submit"
              className="w-full mt-4"
              disabled={isLoading || !watch("email")?.trim() || !watch("password")}
            >
              {isLoading ? t("auth.signing_in") : t("auth.sign_in")}
            </Button>
          </form>

          <div className="text-center">
            <Link
              to="/auth/login"
              search={{ redirect }}
              className="text-primary text-sm font-medium hover:underline"
            >
              {t("auth.back_to_login")}
            </Link>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
