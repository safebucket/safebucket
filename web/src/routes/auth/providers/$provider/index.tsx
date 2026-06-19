import {
  Link,
  createFileRoute,
  redirect,
  useNavigate,
} from "@tanstack/react-router";

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

export const Route = createFileRoute("/auth/providers/$provider/")({
  validateSearch: (search: Record<string, unknown>) => {
    return {
      redirect: (search.redirect as string) || undefined,
    };
  },
  beforeLoad: async ({ context: { queryClient }, params }) => {
    const { data: providers } = await queryClient.ensureQueryData(
      authProvidersQueryOptions(),
    );
    const isLdapProvider = providers.some(
      (p) => p.id === params.provider && p.type === ProviderType.LDAP,
    );
    if (!isLdapProvider) {
      throw redirect({ to: "/auth/login", search: { redirect: undefined } });
    }
  },
  component: LdapLogin,
});

function LdapLogin() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { provider: providerId } = Route.useParams();
  const { redirect: redirectTo } = Route.useSearch();
  const providersQuery = useSuspenseQuery(authProvidersQueryOptions());
  const provider = providersQuery.data.find((p) => p.id === providerId);
  const providerName = provider?.name ?? providerId;

  const { loginLDAP } = useLogin();
  const { register, handleSubmit, watch } = useForm<ILoginForm>();
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const emailValue = watch("email") || "";
  const passwordValue = watch("password") || "";

  const handleLdapLogin: SubmitHandler<ILoginForm> = async (data) => {
    setIsLoading(true);
    setError(null);

    const result = await loginLDAP(providerId, data);

    if (result.mfaRequired) {
      mfaPending.set();
      navigate({
        to: "/auth/mfa",
        search: { redirect: redirectTo },
      });
    } else if (result.success) {
      navigate({ to: redirectTo || "/" });
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
          <CardTitle>{t("auth.ldap.title", { name: providerName })}</CardTitle>
          <CardDescription>
            {t("auth.ldap.description", { name: providerName })}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <form onSubmit={handleSubmit(handleLdapLogin)}>
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

            <div className="mt-4 grid gap-2">
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
              disabled={isLoading || !emailValue.trim() || !passwordValue}
            >
              {isLoading ? t("auth.signing_in") : t("auth.sign_in")}
            </Button>
          </form>

          <div className="text-center">
            <Link
              to="/auth/login"
              search={{ redirect: undefined }}
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
