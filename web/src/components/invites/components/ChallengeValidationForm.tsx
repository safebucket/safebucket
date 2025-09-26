import { useState } from "react";

import Cookies from "js-cookie";
import { AlertCircle, CheckCircle, Shield } from "lucide-react";
import { useNavigate } from "@tanstack/react-router";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import type { FC } from "react";

import type { IChallengeValidationFormData } from "@/components/invites/helpers/types";
import { api_validateChallenge } from "@/components/invites/helpers/api";
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

export interface IChallengeValidationFormProps {
  invitationId: string;
  challengeId: string;
}

export const ChallengeValidationForm: FC<IChallengeValidationFormProps> = ({
  invitationId,
  challengeId,
}) => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [isValidated, setIsValidated] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [code, setCode] = useState("");

  const {
    formState: { isSubmitting },
  } = useForm<IChallengeValidationFormData>();

  const handleFormSubmit = () => {
    setError(null);

    if (code.length !== 6) {
      setError(t("invites.challenge_validation.code_length_error"));
      return;
    }

    api_validateChallenge(invitationId, challengeId, code)
      .then((res) => {
        Cookies.set("safebucket_access_token", res.access_token);
        Cookies.set("safebucket_refresh_token", res.refresh_token);
        Cookies.set("safebucket_auth_provider", "local");

        setIsValidated(true);

        navigate({ to: "/" });
      })
      .catch(() =>
        setError(t("invites.challenge_validation.code_invalid_error")),
      );
  };

  if (isValidated) {
    return (
      <Card className="mx-auto w-full max-w-md">
        <CardContent className="pt-6">
          <div className="space-y-4 text-center">
            <CheckCircle className="mx-auto h-12 w-12 text-green-500" />
            <h3 className="text-lg font-semibold">
              {t("invites.challenge_validation.success_title")}
            </h3>
            <p className="text-muted-foreground text-sm">
              {t("invites.challenge_validation.success_description")}
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
        <CardTitle>{t("invites.challenge_validation.title")}</CardTitle>
        <CardDescription>
          {t("invites.challenge_validation.description")}
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
            <Label className="flex justify-center" htmlFor="code">
              {t("invites.challenge_validation.code_label")}
            </Label>
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
            {isSubmitting
              ? t("invites.challenge_validation.validating_button")
              : t("invites.challenge_validation.validate_button")}
          </Button>

          <p className="text-muted-foreground mt-3 text-center text-xs">
            {t("invites.challenge_validation.footer_text")}
          </p>
        </div>
      </CardContent>
    </Card>
  );
};
