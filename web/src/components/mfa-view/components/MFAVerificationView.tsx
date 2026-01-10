import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { Shield } from "lucide-react";
import { useVerificationFlow } from "../hooks/useVerificationFlow";
import { MFA_CODE_LENGTH } from "../helpers/constants";
import { MFADeviceSelector } from "./MFADeviceSelector";
import { MFASuccessState } from "./MFASuccessState";
import { MFAVerifyInput } from "./MFAVerifyInput";
import type { IMFADevice } from "../helpers/types";
import { FormErrorAlert } from "@/components/common/FormErrorAlert";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { useMFAAuth } from "@/context/MFAAuthContext";

export interface IMFAVerificationViewProps {
  mfaToken: string;
  devices: Array<IMFADevice>;
  redirectPath?: string;
}

export function MFAVerificationView({
  mfaToken,
  devices,
  redirectPath,
}: IMFAVerificationViewProps) {
  const { t } = useTranslation();
  const { clearMFAAuth } = useMFAAuth();

  const {
    code,
    setCode,
    selectedDeviceId,
    setSelectedDeviceId,
    error,
    isLoading,
    isVerified,
    handleSubmit,
    handleBackToLogin,
  } = useVerificationFlow({
    mfaToken,
    redirectPath,
    devices,
    onClearDevices: clearMFAAuth,
  });

  if (isVerified) {
    return <MFASuccessState />;
  }

  return (
    <div className="m-6 flex h-full items-center justify-center">
      <Card className="mx-auto w-full max-w-md">
        <CardHeader className="text-center">
          <div className="mx-auto mb-4 h-12 w-12 rounded-full bg-blue-100 p-3">
            <Shield className="h-6 w-6 text-blue-600" />
          </div>
          <CardTitle>{t("auth.mfa.verify_title")}</CardTitle>
          <CardDescription>{t("auth.mfa.verify_subtitle")}</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <FormErrorAlert error={error} />

            <MFADeviceSelector
              devices={devices}
              selectedDeviceId={selectedDeviceId}
              onSelectDevice={setSelectedDeviceId}
              disabled={isLoading}
            />

            <MFAVerifyInput
              value={code}
              onChange={setCode}
              disabled={isLoading}
            />

            <Button
              type="submit"
              className="w-full"
              disabled={isLoading || code.length !== MFA_CODE_LENGTH}
            >
              {isLoading
                ? t("auth.mfa.verifying")
                : t("auth.mfa.verify_button")}
            </Button>

            <div className="text-center">
              <Link
                to="/auth/login"
                search={{ redirect: undefined }}
                className="text-primary text-sm font-medium hover:underline"
                onClick={handleBackToLogin}
              >
                {t("auth.mfa.back_to_login")}
              </Link>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
