import React, { FC, useState } from "react";

import { AlertCircle, CheckCircle, Mail } from "lucide-react";
import { useForm } from "react-hook-form";

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

interface EmailConfirmationFormData {
  email: string;
}

interface EmailConfirmationFormProps {
  onSubmit?: (email: string) => void;
  invitationId?: string;
}

export const EmailConfirmationForm: FC<EmailConfirmationFormProps> = ({
  onSubmit,
  invitationId,
}) => {
  const [isSubmitted, setIsSubmitted] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<EmailConfirmationFormData>();

  const handleFormSubmit = async (data: EmailConfirmationFormData) => {
    setError(null);

    api_createChallenge(invitationId!, data.email)
      .then(() => {
        setIsSubmitted(true);
        onSubmit?.(data.email);
      })
      .catch(() =>
        setError("Failed to send verification code. Please try again."),
      );
  };

  if (isSubmitted) {
    return (
      <Card className="mx-auto w-full max-w-md">
        <CardContent className="pt-6">
          <div className="space-y-4 text-center">
            <CheckCircle className="mx-auto h-12 w-12 text-green-500" />
            <h3 className="text-lg font-semibold">Verification Code Sent</h3>
            <p className="text-sm text-muted-foreground">
              A verification code has been sent to your email address. Please
              check your inbox and enter the code to complete your account
              creation.
            </p>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="mx-auto w-full max-w-md">
      <CardHeader className="text-center">
        <div className="mx-auto mb-4 h-12 w-12 rounded-full bg-blue-100 p-3">
          <Mail className="h-6 w-6 text-blue-600" />
        </div>
        <CardTitle>Verify your email address</CardTitle>
        <CardDescription>
          Confirm your email address to receive a security code challenge that
          you&#39;ll need to create your account
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
            {isSubmitting ? "Sending code..." : "Send verification code"}
          </Button>

          <p className="mt-3 text-center text-xs text-muted-foreground">
            After confirming, check your email for a verification code to
            complete the process
          </p>
        </form>
      </CardContent>
    </Card>
  );
};
