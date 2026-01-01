import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { AlertCircle, CheckCircle, LogOut, Shield } from "lucide-react";

import { useLogout } from "@/hooks/useAuth";
import { getCurrentSession } from "@/lib/auth-service";
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
import { useMFASetup } from "@/components/mfa-view/hooks/useMFASetup";
import { MFAQRCode } from "@/components/mfa-view/components/MFAQRCode";
import { MFAVerifyInput } from "@/components/mfa-view/components/MFAVerifyInput";
import { MFAErrorAlert } from "@/components/mfa-view/components/MFAErrorAlert";
import {
  MFA_CODE_LENGTH,
  MFA_SUCCESS_REDIRECT_DELAY,
} from "@/components/mfa-view/helpers/constants";

export const Route = createFileRoute("/auth/mfa/setup-required/")({
  validateSearch: (search: Record<string, unknown>) => {
    return {
      redirect: (search.redirect as string) || undefined,
    };
  },
  component: MFASetupRequired,
});

type PageStep = "loading" | "name" | "qr" | "verify" | "success" | "error";

function MFASetupRequired() {
  const navigate = useNavigate();
  const { redirect } = Route.useSearch();
  const logout = useLogout();

  const [pageStep, setPageStep] = useState<PageStep>("loading");
  const [userId, setUserId] = useState<string | null>(null);
  const setupStarted = useRef(false);

  // Check session on mount
  useEffect(() => {
    if (pageStep === "loading" && !setupStarted.current) {
      const session = getCurrentSession();
      if (session?.userId) {
        setupStarted.current = true;
        setUserId(session.userId);
        setPageStep("name");
      } else {
        navigate({ to: "/auth/login", search: { redirect: undefined } });
      }
    }
  }, [pageStep, navigate]);

  if (pageStep === "loading") {
    return <LoadingState />;
  }

  if (pageStep === "error") {
    return (
      <ErrorState
        onLogout={logout}
        onRetry={() => setPageStep("name")}
      />
    );
  }

  if (pageStep === "success") {
    return <SuccessState />;
  }

  if (!userId) {
    return <LoadingState />;
  }

  return (
    <SetupFlow
      userId={userId}
      redirect={redirect}
      onLogout={logout}
      onSuccess={() => setPageStep("success")}
      onError={() => setPageStep("error")}
    />
  );
}

function LoadingState() {
  const { t } = useTranslation();

  return (
    <div className="m-6 flex h-full items-center justify-center">
      <Card className="mx-auto w-full max-w-md">
        <CardContent className="pt-6">
          <div className="flex flex-col items-center space-y-4">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
            <p className="text-muted-foreground text-sm">
              {t("auth.mfa.setup_required_description")}
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function ErrorState({
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

function SuccessState() {
  const { t } = useTranslation();

  return (
    <div className="m-6 flex h-full items-center justify-center">
      <Card className="mx-auto w-full max-w-md">
        <CardContent className="pt-6">
          <div className="flex flex-col items-center space-y-4">
            <div className="flex h-16 w-16 items-center justify-center rounded-full bg-green-100">
              <CheckCircle className="h-8 w-8 text-green-600" />
            </div>
            <div className="text-center space-y-2">
              <h3 className="text-lg font-semibold">
                {t("auth.mfa.setup_success_title")}
              </h3>
              <p className="text-muted-foreground text-sm">
                {t("auth.mfa.success_message")}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function SetupFlow({
  userId,
  redirect,
  onLogout,
  onSuccess,
  onError,
}: {
  userId: string;
  redirect: string | undefined;
  onLogout: () => void;
  onSuccess: () => void;
  onError: () => void;
}) {
  const { t } = useTranslation();
  const navigate = useNavigate();

  const {
    step,
    deviceName,
    setDeviceName,
    setupData,
    code,
    setCode,
    error,
    isLoading,
    startSetup,
    goToVerify,
    goBack,
    verifyCode,
  } = useMFASetup(userId);

  // Handle step changes
  useEffect(() => {
    if (step === "success") {
      onSuccess();
      setTimeout(() => {
        navigate({ to: redirect || "/" });
      }, MFA_SUCCESS_REDIRECT_DELAY);
    }
  }, [step, onSuccess, navigate, redirect]);

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
              </div>

              <div className="flex gap-2">
                <Button
                  variant="outline"
                  onClick={onLogout}
                  className="flex-1"
                >
                  <LogOut className="mr-2 h-4 w-4" />
                  {t("common.logout")}
                </Button>
                <Button
                  onClick={handleStartSetup}
                  disabled={isLoading}
                  className="flex-1"
                >
                  {isLoading ? t("common.loading") : t("auth.continue")}
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
                <Button
                  variant="outline"
                  onClick={onLogout}
                  className="flex-1"
                >
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
