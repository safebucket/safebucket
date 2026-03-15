import { useState } from "react";
import { useTranslation } from "react-i18next";
import { X } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { CustomAlertDialog } from "@/components/dialogs/components/CustomAlertDialog";
import { authCookies } from "@/lib/auth-service";
import {
  useRevokeAllSessionsMutation,
  useRevokeSessionMutation,
  useSessionsQuery,
} from "@/queries/user";

interface SessionsTabProps {
  userId: string;
}

export function SessionsTab({ userId }: SessionsTabProps) {
  const { t } = useTranslation();
  const { data, isLoading } = useSessionsQuery(userId);
  const revokeSession = useRevokeSessionMutation(userId);
  const revokeAll = useRevokeAllSessionsMutation(userId);

  const [revokeTarget, setRevokeTarget] = useState<string | null>(null);
  const [showRevokeAll, setShowRevokeAll] = useState(false);

  if (isLoading || !data) {
    return null;
  }

  const sessions = data.sessions;
  const targetSession = sessions.find((s) => s.id === revokeTarget);
  const isRevokingCurrent = targetSession?.is_current ?? false;

  return (
    <>
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle className="text-base">
              {t("settings.sessions.title")}
            </CardTitle>
            <CardDescription>
              {t("settings.sessions.description")}
            </CardDescription>
          </div>
          {sessions.filter((s) => !s.is_current).length > 0 && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowRevokeAll(true)}
            >
              {t("settings.sessions.revoke_all")}
            </Button>
          )}
        </CardHeader>
        <CardContent>
          {sessions.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              {t("settings.sessions.no_sessions")}
            </p>
          ) : (
            <div className="space-y-3">
              {sessions.map((session) => (
                <div
                  key={session.id}
                  className="flex items-center justify-between rounded-md border p-3"
                >
                  <div className="flex items-center gap-3">
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-medium font-mono">
                          {session.id.slice(0, 8)}
                        </span>
                        {session.is_current && (
                          <Badge variant="secondary">
                            {t("settings.sessions.current")}
                          </Badge>
                        )}
                      </div>
                      <p className="text-xs text-muted-foreground">
                        {t("settings.sessions.created_at", {
                          date: new Date(session.created_at).toLocaleDateString(
                            undefined,
                            {
                              year: "numeric",
                              month: "short",
                              day: "numeric",
                              hour: "2-digit",
                              minute: "2-digit",
                            },
                          ),
                        })}
                      </p>
                    </div>
                  </div>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 text-destructive hover:text-destructive"
                    onClick={() => setRevokeTarget(session.id)}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <CustomAlertDialog
        open={!!revokeTarget}
        onOpenChange={(open) => {
          if (!open) setRevokeTarget(null);
        }}
        title={t(
          isRevokingCurrent
            ? "settings.sessions.revoke_current_confirm_title"
            : "settings.sessions.revoke_confirm_title",
        )}
        description={t(
          isRevokingCurrent
            ? "settings.sessions.revoke_current_confirm_description"
            : "settings.sessions.revoke_confirm_description",
        )}
        cancelLabel={t("common.cancel")}
        confirmLabel={t("settings.sessions.revoke")}
        onConfirm={() => {
          if (revokeTarget) {
            revokeSession.mutate(revokeTarget, {
              onSuccess: () => {
                if (isRevokingCurrent) {
                  authCookies.clearAll();
                  window.location.href = "/auth/login";
                }
              },
            });
            setRevokeTarget(null);
          }
        }}
      />

      <CustomAlertDialog
        open={showRevokeAll}
        onOpenChange={setShowRevokeAll}
        title={t("settings.sessions.revoke_all_confirm_title")}
        description={t("settings.sessions.revoke_all_confirm_description")}
        cancelLabel={t("common.cancel")}
        confirmLabel={t("settings.sessions.revoke_all")}
        onConfirm={() => {
          revokeAll.mutate();
          setShowRevokeAll(false);
        }}
      />
    </>
  );
}
