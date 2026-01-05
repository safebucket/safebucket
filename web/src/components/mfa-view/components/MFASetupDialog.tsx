import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { CheckCircle } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useMFAViewContext } from "@/components/mfa-view/hooks/useMFAViewContext";
import { useMFASetup } from "@/components/mfa-view/hooks/useMFASetup";
import { MFAQRCode } from "@/components/mfa-view/components/MFAQRCode";
import { MFAVerifyInput } from "@/components/mfa-view/components/MFAVerifyInput";
import { FormErrorAlert } from "@/components/common/FormErrorAlert";
import { MFA_CODE_LENGTH } from "@/components/mfa-view/helpers/constants";

export function MFASetupDialog() {
  const { t } = useTranslation();
  const { userId, setupDialogOpen, closeAllDialogs } = useMFAViewContext();
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
    reset,
  } = useMFASetup(userId);

  // Reset when dialog closes
  useEffect(() => {
    if (!setupDialogOpen) {
      reset();
    }
  }, [setupDialogOpen, reset]);

  const handleClose = () => {
    closeAllDialogs();
  };

  return (
    <Dialog open={setupDialogOpen} onOpenChange={handleClose}>
      <DialogContent
        className="sm:max-w-md"
        showCloseButton={step !== "success"}
      >
        {step === "name" && (
          <>
            <DialogHeader>
              <DialogTitle>{t("auth.mfa.add_device_title")}</DialogTitle>
              <DialogDescription>
                {t("auth.mfa.add_device_description")}
              </DialogDescription>
            </DialogHeader>

            <div className="space-y-4">
              <FormErrorAlert error={error} />

              <div className="space-y-2">
                <Label htmlFor="device-name">
                  {t("auth.mfa.device_name_label")}
                </Label>
                <Input
                  id="device-name"
                  value={deviceName}
                  onChange={(e) => setDeviceName(e.target.value)}
                  placeholder={t("auth.mfa.device_name_placeholder")}
                  disabled={isLoading}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="password">{t("auth.password")}</Label>
                <Input
                  id="password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder={t("auth.mfa.password_placeholder")}
                  disabled={isLoading}
                />
              </div>
            </div>

            <DialogFooter className="sm:justify-between">
              <Button variant="outline" onClick={handleClose}>
                {t("common.cancel")}
              </Button>
              <Button
                onClick={startSetup}
                disabled={!deviceName.trim() || !password || isLoading}
              >
                {isLoading ? t("common.loading") : t("auth.continue")}
              </Button>
            </DialogFooter>
          </>
        )}

        {step === "qr" && setupData && (
          <>
            <DialogHeader>
              <DialogTitle>{t("auth.mfa.qr_code_title")}</DialogTitle>
              <DialogDescription>
                {t("auth.mfa.qr_code_instruction")}
              </DialogDescription>
            </DialogHeader>

            <MFAQRCode
              qrCodeUri={setupData.qr_code_uri}
              secret={setupData.secret}
            />

            <DialogFooter className="sm:justify-between">
              <Button variant="outline" onClick={handleClose}>
                {t("auth.mfa.cancel_setup")}
              </Button>
              <Button onClick={goToVerify}>{t("auth.continue")}</Button>
            </DialogFooter>
          </>
        )}

        {step === "verify" && (
          <>
            <DialogHeader>
              <DialogTitle>{t("auth.mfa.verify_setup_title")}</DialogTitle>
              <DialogDescription>
                {t("auth.mfa.verify_setup_instruction")}
              </DialogDescription>
            </DialogHeader>

            <div className="space-y-4">
              <FormErrorAlert error={error} />
              <MFAVerifyInput
                value={code}
                onChange={setCode}
                disabled={isLoading}
              />
            </div>

            <DialogFooter className="sm:justify-between">
              <Button variant="outline" onClick={goBack}>
                {t("auth.mfa.back_to_login")}
              </Button>
              <Button
                onClick={verifyCode}
                disabled={isLoading || code.length !== MFA_CODE_LENGTH}
              >
                {isLoading
                  ? t("auth.mfa.enabling")
                  : t("auth.mfa.enable_button")}
              </Button>
            </DialogFooter>
          </>
        )}

        {step === "success" && (
          <>
            <div className="flex flex-col items-center space-y-4 py-4">
              <div className="flex h-16 w-16 items-center justify-center rounded-full bg-green-100">
                <CheckCircle className="h-8 w-8 text-green-600" />
              </div>
              <div className="space-y-2 text-center">
                <h3 className="text-lg font-semibold">
                  {t("auth.mfa.device_added_title")}
                </h3>
                <p className="text-muted-foreground text-sm">
                  {t("auth.mfa.device_added_message")}
                </p>
              </div>
            </div>

            <DialogFooter>
              <Button onClick={handleClose} className="w-full">
                {t("common.close")}
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
