import { useEffect } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { Shield } from "lucide-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { useMFASetup } from "../hooks/useMFASetup";
import { useSetupRequiredFlow } from "../hooks/useSetupRequiredFlow";
import { MFASetupSkeleton } from "./MFASetupSkeleton";
import { MFASuccessState } from "./MFASuccessState";
import { MFASetupErrorState } from "./MFASetupErrorState";
import { MFASetupNameStep } from "./MFASetupNameStep";
import { MFASetupQRStep } from "./MFASetupQRStep";
import { MFASetupVerifyStep } from "./MFASetupVerifyStep";
import { MFA_SUCCESS_REDIRECT_DELAY } from "../helpers/constants";

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

  if (viewMode === "loading") {
    return <MFASetupSkeleton />;
  }

  if (viewMode === "error") {
    return <MFASetupErrorState onLogout={onLogout} onRetry={handleRetry} />;
  }

  if (viewMode === "success") {
    return <MFASuccessState title={t("auth.mfa.setup_success_title")} />;
  }

  if (!userId) {
    return <MFASetupSkeleton />;
  }

  return (
    <SetupFlowView
      userId={userId}
      mfaToken={mfaToken ?? undefined}
      onLogout={onLogout}
      onSuccess={handleSuccess}
      onError={handleError}
    />
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
  } = useMFASetup(userId, mfaToken);

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

  const renderStepContent = () => {
    switch (step) {
      case "name":
        return (
          <MFASetupNameStep
            deviceName={deviceName}
            setDeviceName={setDeviceName}
            password={password}
            setPassword={setPassword}
            mfaToken={mfaToken}
            error={error}
            isLoading={isLoading}
            onStartSetup={handleStartSetup}
            onLogout={onLogout}
          />
        );
      case "qr":
        return setupData ? (
          <MFASetupQRStep
            qrCodeUri={setupData.qr_code_uri}
            secret={setupData.secret}
            onContinue={goToVerify}
            onLogout={onLogout}
          />
        ) : null;
      case "verify":
        return (
          <MFASetupVerifyStep
            code={code}
            setCode={setCode}
            error={error}
            isLoading={isLoading}
            onVerify={handleVerifyCode}
            onBack={goBack}
          />
        );
      default:
        return null;
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
        <CardContent className="space-y-6">{renderStepContent()}</CardContent>
      </Card>
    </div>
  );
}
