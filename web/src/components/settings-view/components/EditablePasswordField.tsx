import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Check, Edit2, X } from "lucide-react";
import { Input } from "@/components/ui/input.tsx";
import { Button } from "@/components/ui/button.tsx";

interface EditablePasswordFieldProps {
  label: string;
  onSave: (oldPassword: string, newPassword: string) => void;
  isLoading?: boolean;
}

export function EditablePasswordField({
  label,
  onSave,
  isLoading = false,
}: EditablePasswordFieldProps) {
  const { t } = useTranslation();
  const [isEditing, setIsEditing] = useState(false);
  const [oldPassword, setOldPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState("");

  const handleSave = () => {
    setError("");

    if (!oldPassword) {
      setError(t("settings.profile.old_password_required"));
      return;
    }

    if (newPassword.length < 8) {
      setError(t("settings.profile.password_min_length"));
      return;
    }

    if (newPassword !== confirmPassword) {
      setError(t("settings.profile.password_mismatch"));
      return;
    }

    onSave(oldPassword, newPassword);
    handleCancel();
  };

  const handleCancel = () => {
    setOldPassword("");
    setNewPassword("");
    setConfirmPassword("");
    setError("");
    setIsEditing(false);
  };

  return (
    <div className="space-y-2">
      <label className="text-sm font-medium">{label}</label>
      {isEditing ? (
        <div className="space-y-2">
          <div className="flex items-start gap-2">
            <div className="flex-1 space-y-2">
              <Input
                type="password"
                value={oldPassword}
                onChange={(e) => setOldPassword(e.target.value)}
                placeholder={t("settings.profile.old_password_placeholder")}
                className="text-sm"
                autoFocus
                disabled={isLoading}
              />
              <Input
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                placeholder={t("settings.profile.password_placeholder")}
                className="text-sm"
                disabled={isLoading}
              />
              <Input
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                placeholder={t("settings.profile.confirm_password_placeholder")}
                className="text-sm"
                disabled={isLoading}
              />
            </div>
            <div className="flex gap-2 pt-1">
              <Button
                size="sm"
                onClick={handleSave}
                disabled={
                  isLoading || !oldPassword || !newPassword || !confirmPassword
                }
              >
                <Check className="h-3 w-3" />
              </Button>
              <Button
                size="sm"
                variant="outline"
                onClick={handleCancel}
                disabled={isLoading}
              >
                <X className="h-3 w-3" />
              </Button>
            </div>
          </div>
          {error && <p className="text-sm text-destructive">{error}</p>}
        </div>
      ) : (
        <div className="flex items-center gap-2">
          <Input
            type="password"
            value="••••••••"
            disabled
            className="text-sm"
          />
          <Button
            size="sm"
            variant="outline"
            onClick={() => setIsEditing(true)}
          >
            <Edit2 className="h-3 w-3" />
          </Button>
        </div>
      )}
    </div>
  );
}
