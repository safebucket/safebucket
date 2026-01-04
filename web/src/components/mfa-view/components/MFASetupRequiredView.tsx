import { useEffect } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { AlertCircle, LogOut, Shield, Loader2 } from "lucide-react";
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
import { useMFASetup } from "../hooks/useMFASetup";
import { useSetupRequiredFlow } from "../hooks/useSetupRequiredFlow";
import { MFAQRCode } from "./MFAQRCode";
import { MFAVerifyInput } from "./MFAVerifyInput";
import { MFAErrorAlert } from "./MFAErrorAlert";
import { MFASetupSkeleton } from "./MFASetupSkeleton";
import { MFASuccessState } from "./MFASuccessState";
import {
  MFA_CODE_LENGTH,
  MFA_SUCCESS_REDIRECT_DELAY,
} from "../helpers/constants";

export interface IMFASetupRequiredViewProps {
  redirectPath?: string;
  onLogout: () => void;
}

export function MFASetupRequiredView({
  redirectPath,
  onLogout,
}: IMFASetupRequiredViewProps) {
  const navigate = useNavigate();
  const {
    viewMode,
    userId,
    mfaToken,
    handleSuccess,
    handleError,
    handleRetry,
  } = useSetupRequiredFlow();

  useEffect(() => {
    if (viewMode === "success") {
      setTimeout(() => {
        navigate({ to: redirectPath || "/" });
      }, MFA_SUCCESS_REDIRECT_DELAY);
    }
  }, [viewMode, navigate, redirectPath]);

  const viewComponents = {
    loading: <MFASetupSkeleton />,
    error: <ErrorStateView onLogout={onLogout} onRetry={handleRetry} />,
    success: (
      <MFASuccessState
        title={useTranslation().t("auth.mfa.setup_success_title")}
      />
    ),
    name: userId ? (
      <SetupFlowView
        userId={userId}
        mfaToken={mfaToken ?? undefined}
        onLogout={onLogout}
        onSuccess={handleSuccess}
        onError={handleError}
      />
    ) : (
      <MFASetupSkeleton />
    ),
    qr: userId ? (
      <SetupFlowView
        userId={userId}
        mfaToken={mfaToken ?? undefined}
        onLogout={onLogout}
        onSuccess={handleSuccess}
        onError={handleError}
      />
    ) : (
      <MFASetupSkeleton />
    ),
    verify: userId ? (
      <SetupFlowView
        userId={userId}
        mfaToken={mfaToken ?? undefined}
        onLogout={onLogout}
        onSuccess={handleSuccess}
        onError={handleError}
      />
    ) : (
      <MFASetupSkeleton />
    ),
  };

  return viewComponents[viewMode];
}

function ErrorStateView({
  onLogout,
  onRetry,
}: {
  onLogout: () => void;
  onRetry: () => void;
}) {
  const { t } = useTranslation();

  return (
    <div className="m-6 flex h-full items-center justify-center">
      <Card className="mx-auto w-full max-w-md">
        <CardHeader className="text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-red-100">
            <AlertCircle className="h-6 w-6 text-red-600" />
          </div>
          <CardTitle>{t("auth.mfa.setup_error")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-2">
            <Button variant="outline" onClick={onLogout} className="flex-1">
              <LogOut className="mr-2 h-4 w-4" />
              {t("common.logout")}
            </Button>
            <Button onClick={onRetry} className="flex-1">
              {t("auth.continue")}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function SetupFlowView({
  userId,
  mfaToken,
  onLogout,
  onSuccess,
  onError,
}: {
  userId: string;
  mfaToken?: string;
  onLogout: () => void;
  onSuccess: () => void;
  onError: () => void;
}) {
  const { t } = useTranslation();

  const {
    step,
    deviceName,
    setDeviceName,
    password,
    setPassword,
    setupData,
    code,
    setCode,
    error,
    isLoading,
    startSetup,
    goToVerify,
    goBack,
    verifyCode,
  } = useMFASetup(userId || "", mfaToken || undefined);

  useEffect(() => {
    if (step === "success") {
      onSuccess();
    }
  }, [step, onSuccess]);

  const handleStartSetup = async () => {
    try {
      await startSetup();
    } catch {
      onError();
    }
  };

  const handleVerifyCode = async () => {
    try {
      await verifyCode();
    } catch {
      // Error is handled by the hook
    }
  };

  return (
    <div className="m-6 flex h-full items-center justify-center">
      <Card className="mx-auto w-full max-w-md">
        <CardHeader className="text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-blue-100">
            <Shield className="h-6 w-6 text-blue-600" />
          </div>
          <CardTitle>{t("auth.mfa.setup_required_title")}</CardTitle>
          <CardDescription>
            {t("auth.mfa.setup_required_subtitle")}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          {step === "name" && (
            <>
              <MFAErrorAlert error={error} />

              <div className="space-y-4">
                <p className="text-muted-foreground text-center text-sm">
                  {t("auth.mfa.add_device_description")}
                </p>
                <div className="space-y-2">
                  <Label htmlFor="device-name">
                    {t("auth.mfa.device_name_label")}
                  </Label>
                  <Input
                    id="device-name"
                    value={deviceName}
                    onChange={(e) => setDeviceName(e.target.value)}
                    placeholder="Authenticator"
                    disabled={isLoading}
                  />
                </div>
                <div className="space-y-2">
                  {!mfaToken && (
                    <>
                      <Label htmlFor="password">{t("auth.password")}</Label>
                      <Input
                        id="password"
                        type="password"
                        placeholder={t("auth.mfa.password_placeholder")}
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        disabled={isLoading}
                      />
                    </>
                  )}
                </div>
                {error && <p className="text-sm text-destructive">{error}</p>}
                <Button
                  className="w-full"
                  onClick={handleStartSetup}
                  disabled={
                    isLoading || !deviceName || (!password && !mfaToken)
                  }
                >
                  {isLoading && (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  )}
                  {t("auth.continue")}
                </Button>
              </div>

              <div className="flex gap-2">
                <Button variant="outline" onClick={onLogout} className="flex-1">
                  <LogOut className="mr-2 h-4 w-4" />
                  {t("common.logout")}
                </Button>
              </div>
            </>
          )}

          {step === "qr" && setupData && (
            <>
              <p className="text-muted-foreground text-center text-sm">
                {t("auth.mfa.qr_code_instruction")}
              </p>

              <MFAQRCode
                qrCodeUri={setupData.qr_code_uri}
                secret={setupData.secret}
              />

              <div className="flex gap-2">
                <Button variant="outline" onClick={onLogout} className="flex-1">
                  <LogOut className="mr-2 h-4 w-4" />
                  {t("common.logout")}
                </Button>
                <Button onClick={goToVerify} className="flex-1">
                  {t("auth.continue")}
                </Button>
              </div>
            </>
          )}

          {step === "verify" && (
            <>
              <MFAErrorAlert error={error} />

              <div className="space-y-4">
                <p className="text-muted-foreground text-center text-sm">
                  {t("auth.mfa.verify_setup_instruction")}
                </p>
                <MFAVerifyInput
                  value={code}
                  onChange={setCode}
                  disabled={isLoading}
                />
              </div>

              <div className="flex gap-2">
                <Button variant="outline" onClick={goBack} className="flex-1">
                  {t("auth.mfa.back_to_login")}
                </Button>
                <Button
                  onClick={handleVerifyCode}
                  disabled={isLoading || code.length !== MFA_CODE_LENGTH}
                  className="flex-1"
                >
                  {isLoading
                    ? t("auth.mfa.enabling")
                    : t("auth.mfa.enable_button")}
                </Button>
              </div>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
