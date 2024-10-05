"use client";

import React from "react";
import Link from "next/link";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ProvidersButton } from "@/app/auth/providers/providers-button";
import { useSession } from "@/app/auth/hooks/useSession";

export default function Login() {
  const { register, handleSubmit, localLogin } = useSession();

  return (
    <div className="m-6 flex-1">
      <div className="grid grid-cols-1 gap-8">
        <div className="items-center justify-center px-4 py-12 sm:px-6 lg:px-8">
          <div className="mx-auto max-w-md space-y-4">
            <div className="text-center">
              <h1 className="text-3xl font-bold tracking-tight text-foreground">
                Sign in to your account
              </h1>
              <p className="mt-2 text-muted-foreground">
                Unable to sign in? Contact your administrator.
              </p>
            </div>
            <Card>
              <CardContent className="space-y-4">
                <ProvidersButton />
                <div className="relative">
                  <div className="absolute inset-0 flex items-center">
                    <span className="w-full border-t" />
                  </div>
                  <div
                    className="relative flex justify-center text-xs uppercase">
                    <span className="bg-background px-2 text-muted-foreground">
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
                      href="#"
                      className="text-sm font-medium text-primary hover:underline"
                      prefetch={false}
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
                  onClick={handleSubmit(localLogin)}
                >
                  Sign in
                </Button>
              </CardFooter>
            </Card>
          </div>
        </div>
      </div>
    </div>
  );
}
