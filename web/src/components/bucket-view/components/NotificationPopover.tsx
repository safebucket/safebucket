import { Bell } from "lucide-react";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import type { FC } from "react";

import type { INotificationPreferences } from "@/components/bucket-view/helpers/types.ts";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Switch } from "@/components/ui/switch";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useSession } from "@/hooks/useAuth";
import {
  bucketMembersQueryOptions,
  useUpdateNotificationPreferencesMutation,
} from "@/queries/bucket";

interface NotificationPopoverProps {
  bucketId: string;
}

export const NotificationPopover: FC<NotificationPopoverProps> = ({
  bucketId,
}) => {
  const { t } = useTranslation();
  const session = useSession();

  const { data: members } = useQuery(bucketMembersQueryOptions(bucketId));

  const membership = useMemo(() => {
    if (!session?.email || !members) return undefined;
    return members.find(
      (m) => m.email === session.email && m.status === "active",
    );
  }, [members, session?.email]);

  const mutation = useUpdateNotificationPreferencesMutation(bucketId);

  if (!membership) return null;

  const handleNotificationToggle = ({
    upload_notifications,
    download_notifications,
  }: INotificationPreferences) => {
    mutation.mutate({
      upload_notifications,
      download_notifications,
    });
  };

  return (
    <Popover>
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <PopoverTrigger asChild>
              <Button variant="secondary" size="icon">
                <Bell className="h-4 w-4" />
              </Button>
            </PopoverTrigger>
          </TooltipTrigger>
          <TooltipContent>
            <p>{t("bucket.header.notifications")}</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
      <PopoverContent align="end" className="w-74">
        <div className="space-y-4">
          <div>
            <h4 className="text-sm font-medium">
              {t("bucket.notifications.title")}
            </h4>
            <p className="text-muted-foreground text-xs">
              {t("bucket.notifications.description")}
            </p>
          </div>
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <Label
                htmlFor="upload-notifications"
                className="text-sm font-normal"
              >
                {t("bucket.notifications.uploads")}
              </Label>
              <Switch
                id="upload-notifications"
                checked={membership.upload_notifications}
                onCheckedChange={(val: boolean) =>
                  handleNotificationToggle({
                    upload_notifications: val,
                    download_notifications: membership.download_notifications,
                  })
                }
                disabled={mutation.isPending}
              />
            </div>
            <div className="flex items-center justify-between">
              <Label
                htmlFor="download-notifications"
                className="text-sm font-normal"
              >
                {t("bucket.notifications.downloads")}
              </Label>
              <Switch
                id="download-notifications"
                checked={membership.download_notifications}
                onCheckedChange={(val: boolean) =>
                  handleNotificationToggle({
                    upload_notifications: membership.upload_notifications,
                    download_notifications: val,
                  })
                }
                disabled={mutation.isPending}
              />
            </div>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
};
