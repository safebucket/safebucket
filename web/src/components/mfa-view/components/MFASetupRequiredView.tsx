import { useEffect, useRef, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { Loader2, LogOut, Shield } from "lucide-react";
import { useMFASetup } from "../hooks/useMFASetup";
import {
  MFA_CODE_LENGTH,
  MFA_SUCCESS_REDIRECT_DELAY,
} from "../helpers/constants";
import { MFASetupSkeleton } from "./MFASetupSkeleton";
import { MFASuccessState } from "./MFASuccessState";
import { MFASetupErrorState } from "./MFASetupErrorState";
import { MFAQRCode } from "./MFAQRCode";
import { MFAVerifyInput } from "./MFAVerifyInput";
import { FormErrorAlert } from "@/components/common/FormErrorAlert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { useMFAAuth } from "@/context/MFAAuthContext";
import { useRefreshSession } from "@/hooks/useAuth";

type ViewMode = "loading" | "error" | "setup" | "success";

export interface IMFASetupRequiredViewProps {
  redirectPath?: string;
  onLogout: () => void;
}

export function MFASetupRequiredView({
  redirectPath,
  onLogout,
}: IMFASetupRequiredViewProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const refreshSession = useRefreshSession();
  const { restrictedToken } = useMFAAuth();
  const setupStarted = useRef(false);

  const [viewMode, setViewMode] = useState<ViewMode>("loading");

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
  } = useMFASetup(restrictedToken ?? undefined);

  useEffect(() => {
    if (viewMode === "loading" && !setupStarted.current) {
      if (restrictedToken) {
        setupStarted.current = true;
        setViewMode("setup");
      } else {
        navigate({ to: "/auth/login", search: { redirect: undefined } });
      }
    }
  }, [viewMode, navigate, restrictedToken]);

  useEffect(() => {
    if (step === "success") {
      setViewMode("success");
    }
  }, [step]);

  // Redirect after success
  useEffect(() => {
    if (viewMode === "success") {
      setTimeout(() => {
        refreshSession();
        navigate({ to: redirectPath || "/" });
      }, MFA_SUCCESS_REDIRECT_DELAY);
    }
  }, [viewMode, navigate, redirectPath, refreshSession]);

  const handleStartSetup = async () => {
    try {
      await startSetup();
    } catch {
      setViewMode("error");
    }
  };

  const handleRetry = () => {
    setViewMode("setup");
  };

  if (viewMode === "loading") {
    return <MFASetupSkeleton />;
  }

  if (viewMode === "error") {
    return <MFASetupErrorState onLogout={onLogout} onRetry={handleRetry} />;
  }

  if (viewMode === "success") {
    return <MFASuccessState title={t("auth.mfa.setup_success_title")} />;
  }

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
              <FormErrorAlert error={error} />

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
                {!restrictedToken && (
                  <div className="space-y-2">
                    <Label htmlFor="password">{t("auth.password")}</Label>
                    <Input
                      id="password"
                      type="password"
                      placeholder={t("auth.mfa.password_placeholder")}
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      disabled={isLoading}
                    />
                  </div>
                )}
                <Button
                  className="w-full"
                  onClick={handleStartSetup}
                  disabled={
                    isLoading || !deviceName || (!password && !restrictedToken)
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
              <FormErrorAlert error={error} />

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
                  onClick={verifyCode}
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
