import { useTranslation } from "react-i18next";
import { LogOut, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { FormErrorAlert } from "@/components/common/FormErrorAlert";

interface MFASetupNameStepProps {
  deviceName: string;
  setDeviceName: (value: string) => void;
  password: string;
  setPassword: (value: string) => void;
  mfaToken?: string;
  error: string | null;
  isLoading: boolean;
  onStartSetup: () => void;
  onLogout: () => void;
}

export function MFASetupNameStep({
  deviceName,
  setDeviceName,
  password,
  setPassword,
  mfaToken,
  error,
  isLoading,
  onStartSetup,
  onLogout,
}: MFASetupNameStepProps) {
  const { t } = useTranslation();

  return (
    <>
      <FormErrorAlert error={error} />

      <div className="space-y-4">
        <p className="text-muted-foreground text-center text-sm">
          {t("auth.mfa.add_device_description")}
        </p>
        <div className="space-y-2">
          <Label htmlFor="device-name">{t("auth.mfa.device_name_label")}</Label>
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
        <Button
          className="w-full"
          onClick={onStartSetup}
          disabled={isLoading || !deviceName || (!password && !mfaToken)}
        >
          {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
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
  );
}
