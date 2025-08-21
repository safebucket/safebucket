import { Link, createFileRoute } from "@tanstack/react-router";

import { AuthProvidersButtons } from "@/components/auth-view/components/AuthProvidersButtons";
import { useSessionContext } from "@/components/auth-view/hooks/useSessionContext";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export const Route = createFileRoute("/auth/login/")({
  component: Login,
});

function Login() {
  const { register, handleSubmit, localLogin } = useSessionContext();

  return (
    <div className="m-6 flex-1">
      <div className="grid grid-cols-1 gap-8">
        <div className="items-center justify-center px-4 py-12 sm:px-6 lg:px-8">
          <div className="mx-auto max-w-md space-y-4">
            <div className="text-center">
              <h1 className="text-foreground text-3xl font-bold tracking-tight">
                Sign in to your account
              </h1>
              <p className="text-muted-foreground mt-2">
                Unable to sign in? Contact your administrator.
              </p>
            </div>
            <Card>
              <form onSubmit={handleSubmit(localLogin)}>
                <CardContent className="space-y-4">
                  <AuthProvidersButtons />

                  <div className="relative">
                    <div className="absolute inset-0 flex items-center">
                      <span className="w-full border-t" />
                    </div>
                    <div className="relative flex justify-center text-xs uppercase">
                      <span className="bg-background text-muted-foreground px-2">
                        Or continue with
                      </span>
                    </div>
                  </div>
                  <div className="grid gap-2">
                    <Label htmlFor="email">Email</Label>
                    <Input
                      id="email"
                      type="email"
                      placeholder="name@example.com"
                      {...register("email", { required: true })}
                    />
                  </div>
                  <div className="grid gap-2">
                    <div className="flex items-center justify-between">
                      <Label htmlFor="password">Password</Label>
                      <Link
                        to="/"
                        className="text-primary text-sm font-medium hover:underline"
                      >
                        Forgot password?
                      </Link>
                    </div>
                    <Input
                      id="password"
                      type="password"
                      {...register("password", { required: true })}
                    />
                  </div>
                </CardContent>
                <CardFooter>
                  <Button
                    type="submit"
                    className="w-full"
                  >
                    Sign in
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
