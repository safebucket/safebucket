import { createFileRoute } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { ProfileForm } from "@/components/settings-view/components/ProfileForm";
import { useCurrentUser } from "@/queries/user";

export const Route = createFileRoute("/_authenticated/settings/profile/")({
  component: Profile,
});

function Profile() {
  const { t } = useTranslation();
  const { data: user, isLoading } = useCurrentUser();

  if (isLoading || !user) {
    return null;
  }

  return (
    <div className="container mx-auto max-w-3xl p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold">
          {t("settings.profile.title")}
        </h1>
        <p className="text-sm text-muted-foreground">
          {t("settings.profile.description")}
        </p>
      </div>

      <ProfileForm user={user} />
    </div>
  );
}
