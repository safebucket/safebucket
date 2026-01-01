import { useTranslation } from "react-i18next";
import { Info } from "lucide-react";
import type { IUser } from "@/components/auth-view/types/session.ts";
import { ProviderType } from "@/types/auth_providers.ts";
import { useUpdateUserMutation } from "@/queries/user.ts";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card.tsx";
import { EditableField } from "@/components/settings-view/components/EditableField.tsx";
import { EditablePasswordField } from "@/components/settings-view/components/EditablePasswordField.tsx";
import { MFAView } from "@/components/mfa-view/MFAView";

interface ProfileFormProps {
  user: IUser;
}

export function ProfileForm({ user }: ProfileFormProps) {
  const { t } = useTranslation();
  const updateUserMutation = useUpdateUserMutation(user.id);

  const isLocalProvider = user.provider_type === ProviderType.LOCAL;

  const handleUpdateField = (field: string, value: string) => {
    updateUserMutation.mutate({ [field]: value });
  };

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Info className="h-4 w-4" />
            {t("settings.profile.card_title")}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <EditableField
            label={t("settings.profile.email")}
            value={user.email}
            onSave={() => {}}
            disabled
            type="email"
          />

          <EditableField
            label={t("settings.profile.first_name")}
            value={user.first_name || ""}
            onSave={(value) => handleUpdateField("first_name", value)}
            placeholder={t("settings.profile.first_name_placeholder")}
            isLoading={updateUserMutation.isPending}
          />

          <EditableField
            label={t("settings.profile.last_name")}
            value={user.last_name || ""}
            onSave={(value) => handleUpdateField("last_name", value)}
            placeholder={t("settings.profile.last_name_placeholder")}
            isLoading={updateUserMutation.isPending}
          />

          {isLocalProvider && (
            <EditablePasswordField
              label={t("settings.profile.password")}
              onSave={(oldPassword, newPassword) => {
                updateUserMutation.mutate({
                  old_password: oldPassword,
                  new_password: newPassword,
                });
              }}
              isLoading={updateUserMutation.isPending}
            />
          )}
        </CardContent>
      </Card>

      {isLocalProvider && <MFAView userId={user.id} className="mt-2" />}
    </>
  );
}
