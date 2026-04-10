import { createFileRoute } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { ProfileForm } from "@/components/settings-view/components/ProfileForm";
import { SessionsTab } from "@/components/settings-view/components/SessionsTab";
import { useSession } from "@/hooks/useAuth";
import { useCurrentUser } from "@/queries/user";

export const Route = createFileRoute("/_authenticated/settings/profile/")({
  component: Profile,
});

function Profile() {
  const { t } = useTranslation();
  const { data: user, isLoading } = useCurrentUser();
  const session = useSession();

  if (isLoading || !user || !session) {
    return null;
  }

  return (
    <div className="min-h-0 flex-1 overflow-y-auto">
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
        <div className="mt-2">
          <SessionsTab userId={session.userId} />
        </div>
      </div>
    </div>
  );
}
