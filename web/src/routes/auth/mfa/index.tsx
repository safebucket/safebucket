import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { AlertCircle, CheckCircle, Shield, Smartphone, Star } from "lucide-react";

import type { IMFADevice } from "@/components/auth-view/types/session";
import { useLogin } from "@/hooks/useAuth";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Label } from "@/components/ui/label";

export const Route = createFileRoute("/auth/mfa/")({
  validateSearch: (search: Record<string, unknown>) => {
    return {
      mfa_token: (search.mfa_token as string) || "",
      redirect: (search.redirect as string) || undefined,
    };
  },
  component: MFAVerification,
});

function MFAVerification() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { mfa_token, redirect } = Route.useSearch();
  const { verifyMFA } = useLogin();

  const [code, setCode] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [isVerified, setIsVerified] = useState(false);

  const devices = useMemo<IMFADevice[]>(() => {
    try {
      const stored = sessionStorage.getItem("mfa_devices");
      if (stored) {
        return JSON.parse(stored) as IMFADevice[];
      }
    } catch {}
    return [];
  }, []);

  const defaultDeviceId = useMemo(() => {
    const primary = devices.find((d) => d.is_primary);
    return primary?.id || devices[0]?.id || "";
  }, [devices]);

  const [selectedDeviceId, setSelectedDeviceId] = useState<string>(defaultDeviceId);

  useEffect(() => {
    if (defaultDeviceId && !selectedDeviceId) {
      setSelectedDeviceId(defaultDeviceId);
    }
  }, [defaultDeviceId, selectedDeviceId]);

  if (!mfa_token) {
    navigate({ to: "/auth/login", search: { redirect: undefined } });
    return null;
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (code.length !== 6) {
      setError(t("auth.mfa.error_code_length"));
      return;
    }

    setIsLoading(true);

    const deviceId = devices.length > 0 ? selectedDeviceId : undefined;
    const result = await verifyMFA(mfa_token, code, deviceId);

    if (result.success) {
      sessionStorage.removeItem("mfa_devices");
      setIsVerified(true);
      setTimeout(() => {
        navigate({ to: redirect || "/" });
      }, 1500);
    } else {
      setError(result.error || t("auth.mfa.error_verification_failed"));
    }

    setIsLoading(false);
  };

  const handleBackToLogin = () => {
    sessionStorage.removeItem("mfa_devices");
  };

  if (isVerified) {
    return (
      <div className="m-6 flex h-full items-center justify-center">
        <Card className="mx-auto w-full max-w-md">
          <CardContent className="pt-6">
            <div className="space-y-4 text-center">
              <CheckCircle className="mx-auto h-12 w-12 text-green-500" />
              <h3 className="text-lg font-semibold">
                {t("auth.mfa.success_title")}
              </h3>
              <p className="text-muted-foreground text-sm">
                {t("auth.mfa.success_message")}
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  const showDeviceSelector = devices.length > 1;

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
            {error && (
              <div className="flex items-center space-x-2 rounded-md bg-red-50 p-3 text-red-600">
                <AlertCircle className="h-4 w-4" />
                <span className="text-sm">{error}</span>
              </div>
            )}

            {showDeviceSelector && (
              <div className="space-y-2">
                <Label htmlFor="device-select">
                  {t("auth.mfa.select_device")}
                </Label>
                <Select
                  value={selectedDeviceId}
                  onValueChange={setSelectedDeviceId}
                  disabled={isLoading}
                >
                  <SelectTrigger id="device-select" className="w-full">
                    <SelectValue placeholder={t("auth.mfa.select_device")} />
                  </SelectTrigger>
                  <SelectContent>
                    {devices.map((device) => (
                      <SelectItem key={device.id} value={device.id}>
                        <div className="flex items-center gap-2">
                          <Smartphone className="h-4 w-4" />
                          <span>{device.name}</span>
                          {device.is_primary && (
                            <span className="ml-1 flex items-center gap-1 text-xs text-muted-foreground">
                              <Star className="h-3 w-3 fill-current" />
                              {t("auth.mfa.device_primary")}
                            </span>
                          )}
                        </div>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}

            <div className="space-y-2">
              <p className="text-muted-foreground text-center text-sm">
                {t("auth.mfa.code_instruction")}
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

            <Button
              type="submit"
              className="w-full"
              disabled={isLoading || code.length !== 6}
            >
              {isLoading ? t("auth.mfa.verifying") : t("auth.mfa.verify_button")}
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
