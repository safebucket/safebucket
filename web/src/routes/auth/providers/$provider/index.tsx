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
import { z } from "zod";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import type { SubmitHandler } from "react-hook-form";
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
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
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

  const loginSchema = z.object({
    email: z
      .string()
      .min(1, t("errors.FIELD_REQUIRED"))
      .pipe(z.email(t("errors.INVALID_EMAIL"))),
    password: z.string().min(1, t("errors.FIELD_REQUIRED")),
  });

  type LoginFormValues = z.infer<typeof loginSchema>;

  const { loginLDAP } = useLogin();
  const form = useForm<LoginFormValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: { email: "", password: "" },
  });
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const [emailValue, passwordValue] = form.watch(["email", "password"]);

  const handleLdapLogin: SubmitHandler<LoginFormValues> = async (data) => {
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
          <div className="mx-auto mb-4 h-12 w-12 rounded-full bg-info/10 p-3">
            <LogIn className="h-6 w-6 text-info" />
          </div>
          <CardTitle>{t("auth.ldap.title", { name: providerName })}</CardTitle>
          <CardDescription>
            {t("auth.ldap.description", { name: providerName })}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <Form {...form}>
            <form onSubmit={form.handleSubmit(handleLdapLogin)}>
              <FormField
                control={form.control}
                name="email"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("auth.email")}</FormLabel>
                    <FormControl>
                      <Input
                        type="email"
                        placeholder={t("auth.email_placeholder")}
                        disabled={isLoading}
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="password"
                render={({ field }) => (
                  <FormItem className="mt-4">
                    <FormLabel>{t("auth.password")}</FormLabel>
                    <FormControl>
                      <Input type="password" disabled={isLoading} {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {error && (
                <div className="text-sm text-destructive mt-2">{error}</div>
              )}

              <Button
                type="submit"
                className="w-full mt-4"
                disabled={isLoading || !emailValue.trim() || !passwordValue}
              >
                {isLoading ? t("auth.signing_in") : t("auth.sign_in")}
              </Button>
            </form>
          </Form>

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
