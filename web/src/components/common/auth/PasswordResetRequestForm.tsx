import type { FC } from "react";
import { useState } from "react";

import { AlertCircle, Key, Mail } from "lucide-react";
import { Link } from "@tanstack/react-router";
import { useForm } from "react-hook-form";

import { api_requestPasswordReset } from "@/components/auth-view/helpers/api";
import type { IPasswordResetRequestFormData } from "@/components/auth-view/helpers/types";
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

export const PasswordResetRequestForm: FC = () => {
  const [isSuccess, setIsSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<IPasswordResetRequestFormData>();

  const handleFormSubmit = async (data: IPasswordResetRequestFormData) => {
    setError(null);

    try {
      await api_requestPasswordReset(data);
      setIsSuccess(true);
    } catch {
      setError("Failed to send password reset email. Please try again.");
    }
  };

  if (isSuccess) {
    return (
      <Card className="mx-auto w-full max-w-md">
        <CardHeader className="text-center">
          <div className="mx-auto mb-4 h-12 w-12 rounded-full bg-green-100 p-3">
            <Mail className="h-6 w-6 text-green-600" />
          </div>
          <CardTitle>Check your email</CardTitle>
          <CardDescription>
            If an account with this email exists, we've sent a password reset
            code to your email address
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <p className="text-muted-foreground text-center text-sm">
              The email contains a 6-digit code and a link to reset your
              password. If you don't see it, check your spam folder.
            </p>
            <Link to="/auth/login">
              <Button variant="outline" className="w-full">
                Back to login
              </Button>
            </Link>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="mx-auto w-full max-w-md">
      <CardHeader className="text-center">
        <div className="mx-auto mb-4 h-12 w-12 rounded-full bg-blue-100 p-3">
          <Key className="h-6 w-6 text-blue-600" />
        </div>
        <CardTitle>Reset your password</CardTitle>
        <CardDescription>
          Enter your email address and we'll send you a verification code to
          reset your password
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit(handleFormSubmit)} className="space-y-4">
          {error && (
            <div className="flex items-center space-x-2 rounded-md bg-red-50 p-3 text-red-600">
              <AlertCircle className="h-4 w-4" />
              <span className="text-sm">{error}</span>
            </div>
          )}

          <div className="space-y-2">
            <Label htmlFor="email">Email address</Label>
            <Input
              id="email"
              type="email"
              placeholder="name@example.com"
              {...register("email", {
                required: "Email is required",
                pattern: {
                  value: /^[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}$/i,
                  message: "Please enter a valid email address",
                },
              })}
              className={errors.email ? "border-red-500" : ""}
            />
            {errors.email && (
              <p className="text-sm text-red-500">{errors.email.message}</p>
            )}
          </div>

          <Button type="submit" className="w-full" disabled={isSubmitting}>
            {isSubmitting ? "Sending..." : "Send reset code"}
          </Button>

          <div className="text-center">
            <Link
              to="/auth/login"
              className="text-muted-foreground hover:text-primary text-sm underline"
            >
              Back to login
            </Link>
          </div>

          <p className="text-muted-foreground mt-3 text-center text-xs">
            You'll receive an email with a verification code to reset your
            password
          </p>
        </form>
      </CardContent>
    </Card>
  );
};
