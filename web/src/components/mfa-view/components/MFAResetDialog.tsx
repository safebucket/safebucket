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
import { useMFAReset } from "@/components/mfa-view/hooks/useMFAReset";
import { MFAVerifyInput } from "@/components/mfa-view/components/MFAVerifyInput";
import { FormErrorAlert } from "@/components/common/FormErrorAlert";
import { MFA_CODE_LENGTH } from "@/components/mfa-view/helpers/constants";

interface MFAResetDialogProps {
  userId: string;
  open: boolean;
  onClose: () => void;
}

export function MFAResetDialog({ userId, open, onClose }: MFAResetDialogProps) {
  const { t } = useTranslation();
  const {
    step,
    password,
    setPassword,
    code,
    setCode,
    error,
    isLoading,
    requestReset,
    verifyReset,
    reset,
  } = useMFAReset(userId);

  // Reset when dialog closes
  useEffect(() => {
    if (!open) {
      reset();
    }
  }, [open, reset]);

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-md">
        {step === "password" && (
          <>
            <DialogHeader>
              <DialogTitle>{t("auth.mfa.reset_title")}</DialogTitle>
              <DialogDescription>
                {t("auth.mfa.reset_password_instruction")}
              </DialogDescription>
            </DialogHeader>

            <div className="space-y-4">
              <FormErrorAlert error={error} />

              <div className="space-y-2">
                <Label htmlFor="reset-password">
                  {t("auth.mfa.reset_password_label")}
                </Label>
                <Input
                  id="reset-password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder={t("auth.mfa.reset_password_placeholder")}
                  disabled={isLoading}
                />
              </div>
            </div>

            <DialogFooter className="sm:justify-between">
              <Button variant="outline" onClick={onClose}>
                {t("common.cancel")}
              </Button>
              <Button onClick={requestReset} disabled={!password || isLoading}>
                {isLoading ? t("common.loading") : t("auth.continue")}
              </Button>
            </DialogFooter>
          </>
        )}

        {step === "email_sent" && (
          <>
            <DialogHeader>
              <DialogTitle>{t("auth.mfa.reset_email_sent_title")}</DialogTitle>
              <DialogDescription>
                {t("auth.mfa.reset_email_sent_description")}
              </DialogDescription>
            </DialogHeader>

            <div className="space-y-4">
              <FormErrorAlert error={error} />
              <MFAVerifyInput
                value={code}
                onChange={setCode}
                disabled={isLoading}
                uppercase
              />
            </div>

            <DialogFooter className="sm:justify-between">
              <Button variant="outline" onClick={onClose}>
                {t("common.cancel")}
              </Button>
              <Button
                onClick={verifyReset}
                disabled={code.length !== MFA_CODE_LENGTH || isLoading}
              >
                {isLoading
                  ? t("common.loading")
                  : t("auth.mfa.reset_verify_button")}
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
                  {t("auth.mfa.reset_success_title")}
                </h3>
                <p className="text-muted-foreground text-sm">
                  {t("auth.mfa.reset_success_message")}
                </p>
              </div>
            </div>

            <DialogFooter>
              <Button onClick={onClose} className="w-full">
                {t("common.close")}
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
