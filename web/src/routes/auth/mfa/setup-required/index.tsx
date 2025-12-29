import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState, useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import QRCode from "react-qr-code";
import {
  Shield,
  Copy,
  CheckCircle,
  AlertCircle,
  LogOut,
} from "lucide-react";

import { useLogout } from "@/hooks/useAuth";
import { api } from "@/lib/api";
import { authCookies, getCurrentSession } from "@/lib/auth-service";
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

export const Route = createFileRoute("/auth/mfa/setup-required/")({
  validateSearch: (search: Record<string, unknown>) => {
    return {
      redirect: (search.redirect as string) || undefined,
    };
  },
  component: MFASetupRequired,
});

interface MFASetupResponse {
  secret: string;
  qr_code_uri: string;
  issuer: string;
}

type SetupStep = "loading" | "qr" | "verify" | "success" | "error";

function MFASetupRequired() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { redirect } = Route.useSearch();
  const logout = useLogout();

  const [setupStep, setSetupStep] = useState<SetupStep>("loading");
  const [setupData, setSetupData] = useState<MFASetupResponse | null>(null);
  const [code, setCode] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [secretCopied, setSecretCopied] = useState(false);
  const [userId, setUserId] = useState<string | null>(null);
  const setupStarted = useRef(false);

  // Start MFA setup automatically when component mounts
  const startSetup = async (currentUserId: string) => {
    setIsLoading(true);
    setError(null);

    try {
      const response = await api.post<MFASetupResponse>(
        `/users/${currentUserId}/mfa/setup`
      );
      setSetupData(response);
      setSetupStep("qr");
    } catch (err) {
      setError(t("auth.mfa.setup_error"));
      setSetupStep("error");
    } finally {
      setIsLoading(false);
    }
  };

  // Start setup on mount (only once)
  // Read session directly from cookies to avoid router context timing issues
  useEffect(() => {
    if (setupStep === "loading" && !setupStarted.current) {
      const session = getCurrentSession();
      if (session?.userId) {
        setupStarted.current = true;
        setUserId(session.userId);
        startSetup(session.userId);
      } else {
        // No session in cookies, redirect to login
        navigate({ to: "/auth/login", search: { redirect: undefined } });
      }
    }
  }, [setupStep, navigate]);

  const handleVerifyCode = async () => {
    if (!userId) return;

    if (code.length !== 6) {
      setError(t("auth.mfa.error_code_length"));
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      const response = await api.post<{
        access_token: string;
        refresh_token: string;
      }>(`/users/${userId}/mfa/verify`, { code });

      // Save the new tokens with updated MFA claim
      if (response.access_token) {
        authCookies.setAccessToken(response.access_token);
      }
      if (response.refresh_token) {
        authCookies.setRefreshToken(response.refresh_token);
      }

      setSetupStep("success");
      // Redirect after a brief delay
      setTimeout(() => {
        navigate({ to: redirect || "/" });
      }, 2000);
    } catch (err) {
      setError(t("auth.mfa.verify_error"));
    } finally {
      setIsLoading(false);
    }
  };

  const handleCopySecret = async () => {
    if (setupData?.secret) {
      await navigator.clipboard.writeText(setupData.secret);
      setSecretCopied(true);
      setTimeout(() => setSecretCopied(false), 2000);
    }
  };

  const handleLogout = () => {
    logout();
  };

  const handleRetrySetup = () => {
    if (userId) {
      startSetup(userId);
    }
  };

  if (setupStep === "loading") {
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

  if (setupStep === "error") {
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
              <Button variant="outline" onClick={handleLogout} className="flex-1">
                <LogOut className="mr-2 h-4 w-4" />
                {t("common.logout")}
              </Button>
              <Button onClick={handleRetrySetup} className="flex-1">
                {t("auth.continue")}
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  if (setupStep === "success") {
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
          {setupStep === "qr" && setupData && (
            <>
              <div className="flex flex-col items-center space-y-4">
                <p className="text-muted-foreground text-center text-sm">
                  {t("auth.mfa.qr_code_instruction")}
                </p>
                <div className="rounded-lg bg-white p-4">
                  <QRCode value={setupData.qr_code_uri} size={180} />
                </div>

                <div className="w-full space-y-2">
                  <p className="text-sm font-medium">
                    {t("auth.mfa.manual_entry_title")}
                  </p>
                  <p className="text-muted-foreground text-xs">
                    {t("auth.mfa.manual_entry_instruction")}
                  </p>
                  <div className="flex items-center gap-2">
                    <code className="bg-muted flex-1 rounded px-3 py-2 text-sm font-mono break-all">
                      {setupData.secret}
                    </code>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={handleCopySecret}
                    >
                      {secretCopied ? (
                        <CheckCircle className="h-4 w-4 text-green-600" />
                      ) : (
                        <Copy className="h-4 w-4" />
                      )}
                    </Button>
                  </div>
                </div>
              </div>

              <div className="flex gap-2">
                <Button variant="outline" onClick={handleLogout} className="flex-1">
                  <LogOut className="mr-2 h-4 w-4" />
                  {t("common.logout")}
                </Button>
                <Button onClick={() => setSetupStep("verify")} className="flex-1">
                  {t("auth.continue")}
                </Button>
              </div>
            </>
          )}

          {setupStep === "verify" && (
            <>
              {error && (
                <div className="flex items-center gap-2 rounded-md bg-red-50 p-3 text-sm text-red-600">
                  <AlertCircle className="h-4 w-4" />
                  {error}
                </div>
              )}

              <div className="space-y-4">
                <p className="text-muted-foreground text-center text-sm">
                  {t("auth.mfa.verify_setup_instruction")}
                </p>
                <div className="flex justify-center">
                  <InputOTP
                    maxLength={6}
                    value={code}
                    onChange={(value) => setCode(value)}
                    disabled={isLoading}
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

              <div className="flex gap-2">
                <Button variant="outline" onClick={() => setSetupStep("qr")} className="flex-1">
                  {t("auth.mfa.back_to_login")}
                </Button>
                <Button
                  onClick={handleVerifyCode}
                  disabled={isLoading || code.length !== 6}
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
