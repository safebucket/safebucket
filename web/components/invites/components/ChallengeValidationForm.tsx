import React, { useState } from "react";

import { api } from "@/lib/api";
import Cookies from "js-cookie";
import { AlertCircle, CheckCircle, Shield } from "lucide-react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  InputOTP,
  InputOTPGroup,
  InputOTPSlot,
} from "@/components/ui/input-otp";
import { Label } from "@/components/ui/label";

import {
  ChallengeValidationFormData,
  ChallengeValidationFormProps,
  ChallengeValidationResponse,
} from "../helpers/types";

export const ChallengeValidationForm: React.FC<
  ChallengeValidationFormProps
> = ({ onSubmit, invitationId, challengeId }) => {
  const router = useRouter();
  const [isValidated, setIsValidated] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [code, setCode] = useState("");

  const {
    formState: { isSubmitting },
  } = useForm<ChallengeValidationFormData>();

  const handleFormSubmit = async () => {
    setError(null);

    if (code.length !== 6) {
      setError("Verification code must be exactly 6 digits");
      return;
    }

    try {
      const response = await api.post<ChallengeValidationResponse>(
        `/invites/${invitationId}/challenges/${challengeId}/validate`,
        {
          code: code,
        },
      );

      // Set authentication cookies (same pattern as localLogin)
      Cookies.set("safebucket_access_token", response.access_token);
      Cookies.set("safebucket_refresh_token", response.refresh_token);
      Cookies.set("safebucket_auth_provider", "local");

      setIsValidated(true);
      onSubmit?.(code);

      // Redirect to buckets page after successful login
      router.push("/buckets");
    } catch (err) {
      console.error("Failed to validate challenge code:", err);
      setError(
        "Invalid verification code. Please check your email and try again.",
      );
    }
  };

  if (isValidated) {
    return (
      <Card className="mx-auto w-full max-w-md">
        <CardContent className="pt-6">
          <div className="space-y-4 text-center">
            <CheckCircle className="mx-auto h-12 w-12 text-green-500" />
            <h3 className="text-lg font-semibold">
              Code Validated Successfully
            </h3>
            <p className="text-sm text-muted-foreground">
              Your verification code has been validated. Your account has been
              created and you can now access the invitation.
            </p>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="mx-auto w-full max-w-md">
      <CardHeader className="text-center">
        <div className="mx-auto mb-4 h-12 w-12 rounded-full bg-green-100 p-3">
          <Shield className="h-6 w-6 text-green-600" />
        </div>
        <CardTitle>Enter your verification code</CardTitle>
        <CardDescription>
          Enter the 6-digit verification code that was sent to your email
          address to complete your account creation
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          {error && (
            <div className="flex items-center space-x-2 rounded-md bg-red-50 p-3 text-red-600">
              <AlertCircle className="h-4 w-4" />
              <span className="text-sm">{error}</span>
            </div>
          )}

          <div className="space-y-2">
            <Label className="flex justify-center" htmlFor="code">6-digit verification code</Label>
            <div className="flex justify-center">
              <InputOTP
                maxLength={6}
                value={code}
                onChange={(value) => setCode(value)}
              >
                <InputOTPGroup>
                  <InputOTPSlot index={0} />
                  <InputOTPSlot index={1} />
                  <InputOTPSlot index={2} />
                  <InputOTPSlot index={3} />
                  <InputOTPSlot index={4} />
                  <InputOTPSlot index={5} />
                </InputOTPGroup>
              </InputOTP>
            </div>
          </div>

          <Button
            onClick={handleFormSubmit}
            className="w-full"
            disabled={isSubmitting || code.length !== 6}
          >
            {isSubmitting ? "Validating..." : "Validate code"}
          </Button>

          <p className="mt-3 text-center text-xs text-muted-foreground">
            Didn&apos;t receive the code? <br/> Check your spam folder or request a new one
          </p>
        </div>
      </CardContent>
    </Card>
  );
};
