import React, { useState } from "react";
import { useForm } from "react-hook-form";
import { useRouter } from "next/navigation";
import { Shield, CheckCircle, AlertCircle } from "lucide-react";
import Cookies from "js-cookie";

import { api } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

interface ChallengeValidationFormData {
  code: string;
}

interface ChallengeValidationResponse {
  access_token: string;
  refresh_token: string;
}

interface ChallengeValidationFormProps {
  onSubmit?: (code: string) => void;
  invitationId?: string;
  challengeId?: string;
}

export const ChallengeValidationForm: React.FC<ChallengeValidationFormProps> = ({
  onSubmit,
  invitationId,
  challengeId,
}) => {
  const router = useRouter();
  const [isValidated, setIsValidated] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<ChallengeValidationFormData>();

  const handleFormSubmit = async (data: ChallengeValidationFormData) => {
    setError(null);
    
    try {
      const response = await api.post<ChallengeValidationResponse>(`/invites/${invitationId}/challenge/${challengeId}/validate`, { 
        code: data.code 
      });
      
      // Set authentication cookies (same pattern as localLogin)
      Cookies.set("safebucket_access_token", response.access_token);
      Cookies.set("safebucket_refresh_token", response.refresh_token);
      Cookies.set("safebucket_auth_provider", "local");
      
      setIsValidated(true);
      onSubmit?.(data.code);
      
      // Redirect to buckets page after successful login
      router.push("/buckets");
    } catch (err) {
      console.error("Failed to validate challenge code:", err);
      setError("Invalid verification code. Please check your email and try again.");
    }
  };

  if (isValidated) {
    return (
      <Card className="w-full max-w-md mx-auto">
        <CardContent className="pt-6">
          <div className="text-center space-y-4">
            <CheckCircle className="mx-auto h-12 w-12 text-green-500" />
            <h3 className="text-lg font-semibold">Code Validated Successfully</h3>
            <p className="text-sm text-muted-foreground">
              Your verification code has been validated. Your account has been created and you can now access the invitation.
            </p>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="w-full max-w-md mx-auto">
      <CardHeader className="text-center">
        <div className="mx-auto mb-4 h-12 w-12 rounded-full bg-green-100 p-3">
          <Shield className="h-6 w-6 text-green-600" />
        </div>
        <CardTitle>Enter Verification Code</CardTitle>
        <CardDescription>
          Please enter the 6-digit verification code that was sent to your email address to complete your account creation.
          {invitationId && (
            <span className="block mt-2 text-xs font-mono text-muted-foreground/70">
              Invitation ID: {invitationId}
            </span>
          )}
          {challengeId && (
            <span className="block mt-1 text-xs font-mono text-muted-foreground/70">
              Challenge ID: {challengeId}
            </span>
          )}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit(handleFormSubmit)} className="space-y-4">
          {error && (
            <div className="flex items-center space-x-2 text-red-600 bg-red-50 p-3 rounded-md">
              <AlertCircle className="h-4 w-4" />
              <span className="text-sm">{error}</span>
            </div>
          )}
          
          <div className="space-y-2">
            <Label htmlFor="code">6-Digit Verification Code</Label>
            <Input
              id="code"
              type="text"
              placeholder="123456"
              {...register("code", {
                required: "Verification code is required",
                pattern: {
                  value: /^\d{6}$/,
                  message: "Verification code must be exactly 6 digits",
                },
              })}
              className={errors.code ? "border-red-500" : ""}
              maxLength={6}
            />
            {errors.code && (
              <p className="text-sm text-red-500">{errors.code.message}</p>
            )}
          </div>
          
          <Button
            type="submit"
            className="w-full"
            disabled={isSubmitting}
          >
            {isSubmitting ? "Validating..." : "Validate Code"}
          </Button>
          
          <p className="text-xs text-center text-muted-foreground mt-3">
            Didn't receive the code? Check your spam folder or go back to request a new one.
          </p>
        </form>
      </CardContent>
    </Card>
  );
};