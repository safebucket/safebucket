"use client";

import React from "react";

import Image from "next/image";
import Link from "next/link";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export default function Login() {
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
                Don&apos;t have an account?{" "}
                <Link
                  href="/register"
                  className="font-medium text-primary hover:underline"
                  prefetch={false}
                >
                  Register
                </Link>
              </p>
            </div>
            <Card>
              <CardContent className="space-y-4">
                <div className="mt-4 grid grid-cols-2 gap-4">
                  <Button variant="outline">
                    <Image
                      width={15}
                      height={15}
                      alt="Google logo"
                      src="/google.svg"
                      className="mr-2 h-4 w-4"
                    />
                    Sign in with Google
                  </Button>
                  <Button variant="outline">
                    <Image
                      width={25}
                      height={25}
                      alt="Apple logo"
                      src="/apple.svg"
                      className="mr-2"
                    />
                    Sign in with Apple
                  </Button>
                </div>
                <div className="relative">
                  <div className="absolute inset-0 flex items-center">
                    <span className="w-full border-t" />
                  </div>
                  <div className="relative flex justify-center text-xs uppercase">
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
                    required
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
                  <Input id="password" type="password" required />
                </div>
              </CardContent>
              <CardFooter>
                <Button type="submit" className="w-full">
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
