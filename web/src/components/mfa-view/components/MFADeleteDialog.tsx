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
import { useMFADevices } from "@/components/mfa-view/hooks/useMFADevices";
import { FormErrorAlert } from "@/components/common/FormErrorAlert";

interface MFADeleteDialogProps {
  deviceId: string | null;
  onClose: () => void;
}

export function MFADeleteDialog({ deviceId, onClose }: MFADeleteDialogProps) {
  const { t } = useTranslation();
  const { removeDevice, isRemovingDevice } = useMFADevices();

  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);

  // Reset when dialog closes
  useEffect(() => {
    if (!deviceId) {
      setPassword("");
      setError(null);
    }
  }, [deviceId]);

  const handleConfirmDelete = async () => {
    if (!deviceId || !password) return;

    setError(null);
    try {
      await removeDevice(deviceId, password);
      onClose();
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : "";
      if (errorMessage.includes("INVALID_PASSWORD")) {
        setError(t("auth.mfa.delete_invalid_password"));
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
            {t("auth.mfa.delete_device_description")}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <FormErrorAlert error={error} />

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
              disabled={isRemovingDevice}
            />
          </div>
        </div>

        <DialogFooter className="sm:justify-between">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button
            variant="destructive"
            onClick={handleConfirmDelete}
            disabled={!password || isRemovingDevice}
          >
            {isRemovingDevice ? t("common.loading") : t("common.delete")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
