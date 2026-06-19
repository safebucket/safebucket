import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";

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
import { useRemoveMFADeviceMutation } from "@/queries/mfa";
import { FormErrorAlert } from "@/components/common/FormErrorAlert";
import { MFAVerifyInput } from "@/components/mfa-view/components/MFAVerifyInput";
import { MFA_CODE_LENGTH } from "@/components/mfa-view/helpers/constants";
import { ProviderType } from "@/types/auth_providers.ts";

interface MFADeleteDialogProps {
  deviceId: string | null;
  onClose: () => void;
  providerType: string;
}

export function MFADeleteDialog({
  deviceId,
  onClose,
  providerType,
}: MFADeleteDialogProps) {
  const { t } = useTranslation();
  const removeMutation = useRemoveMFADeviceMutation();
  const isOIDC = providerType === ProviderType.OIDC;

  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!deviceId) {
      setPassword("");
      setCode("");
      setError(null);
    }
  }, [deviceId]);

  const canConfirm = isOIDC ? code.length === MFA_CODE_LENGTH : !!password;

  const handleConfirmDelete = async () => {
    if (!deviceId || !canConfirm) return;

    setError(null);
    try {
      await removeMutation.mutateAsync(
        isOIDC ? { deviceId, code } : { deviceId, password },
      );
      onClose();
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : "";
      if (errorMessage.includes("INVALID_PASSWORD")) {
        setError(t("auth.mfa.delete_invalid_password"));
      } else if (errorMessage.includes("INVALID_MFA_CODE")) {
        setError(t("auth.mfa.invalid_code"));
      } else {
        setError(t("auth.mfa.delete_error"));
      }
    }
  };

  return (
    <Dialog open={!!deviceId} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t("auth.mfa.delete_device_title")}</DialogTitle>
          <DialogDescription>
            {isOIDC
              ? t("auth.mfa.delete_code_instruction")
              : t("auth.mfa.delete_device_description")}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <FormErrorAlert error={error} />

          {isOIDC ? (
            <div className="space-y-2">
              <Label>{t("auth.mfa.delete_code_label")}</Label>
              <MFAVerifyInput
                value={code}
                onChange={setCode}
                disabled={removeMutation.isPending}
              />
            </div>
          ) : (
            <div className="space-y-2">
              <Label htmlFor="delete-password">
                {t("auth.mfa.delete_password_label")}
              </Label>
              <Input
                id="delete-password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder={t("auth.mfa.delete_password_placeholder")}
                disabled={removeMutation.isPending}
              />
            </div>
          )}
        </div>

        <DialogFooter className="sm:justify-between">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button
            variant="destructive"
            onClick={handleConfirmDelete}
            disabled={!canConfirm || removeMutation.isPending}
          >
            {removeMutation.isPending
              ? t("common.loading")
              : t("common.delete")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
