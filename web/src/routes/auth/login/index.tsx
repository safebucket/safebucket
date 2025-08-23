import { Link, createFileRoute } from "@tanstack/react-router";

import { useSuspenseQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { AuthProvidersButtons } from "@/components/auth-view/components/AuthProvidersButtons";
import { useSessionContext } from "@/components/auth-view/hooks/useSessionContext";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { authProvidersQueryOptions } from "@/queries/auth_providers.ts";

export const Route = createFileRoute("/auth/login/")({
  loader: ({ context: { queryClient } }) =>
    queryClient.ensureQueryData(authProvidersQueryOptions()),
  component: Login,
});

function Login() {
  const { t } = useTranslation();
  const providersQuery = useSuspenseQuery(authProvidersQueryOptions());
  const providers = providersQuery.data;

  const { register, handleSubmit, localLogin } = useSessionContext();

  return (
    <div className="m-6 flex-1">
      <div className="grid grid-cols-1 gap-8">
        <div className="items-center justify-center px-4 py-12 sm:px-6 lg:px-8">
          <div className="mx-auto max-w-md space-y-4">
            <div className="text-center">
              <h1 className="text-foreground text-3xl font-bold tracking-tight">
                {t("auth.sign_in_title")}
              </h1>
              <p className="text-muted-foreground mt-2">
                {t("auth.sign_in_subtitle")}
              </p>
            </div>
            <Card className="pt-0">
              <form onSubmit={handleSubmit(localLogin)}>
                <CardContent className="space-y-4">
                  <AuthProvidersButtons
                    providers={providers.filter((p) => p.id != "local")}
                  />

                  {providers.find((p) => p.id == "local") && (
                    <>
                      <div className="relative">
                        <div className="absolute inset-0 flex items-center">
                          <span className="w-full border-t" />
                        </div>
                        <div className="relative flex justify-center text-xs uppercase">
                          <span className="bg-background text-muted-foreground px-2">
                            {t("auth.or_continue_with")}
                          </span>
                        </div>
                      </div>
                      <div className="grid gap-2">
                        <Label htmlFor="email">{t("auth.email")}</Label>
                        <Input
                          id="email"
                          type="email"
                          placeholder={t("auth.email_placeholder")}
                          {...register("email", { required: true })}
                        />
                      </div>
                      <div className="grid gap-2">
                        <div className="flex items-center justify-between">
                          <Label htmlFor="password">{t("auth.password")}</Label>
                          <Link
                            to="/"
                            className="text-primary text-sm font-medium hover:underline"
                          >
                            {t("auth.forgot_password")}
                          </Link>
                        </div>
                        <Input
                          id="password"
                          type="password"
                          {...register("password", { required: true })}
                        />
                      </div>
                    </>
                  )}
                </CardContent>
                <CardFooter>
                  <Button type="submit" className="w-full mt-4">
                    {t("auth.sign_in")}
                  </Button>
                </CardFooter>
              </form>
            </Card>
          </div>
        </div>
      </div>
    </div>
  );
}
