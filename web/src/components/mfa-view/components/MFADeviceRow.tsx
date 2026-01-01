import { useTranslation } from "react-i18next";
import { format } from "date-fns";
import { enUS, fr } from "date-fns/locale";
import { Smartphone, Star, Trash2 } from "lucide-react";

import type { IMFADevice } from "@/components/mfa-view/helpers/types";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { useMFAViewContext } from "@/components/mfa-view/hooks/useMFAViewContext";

interface MFADeviceRowProps {
  device: IMFADevice;
}

export function MFADeviceRow({ device }: MFADeviceRowProps) {
  const { t, i18n } = useTranslation();
  const { setDeviceDefault, openDeleteDialog } = useMFAViewContext();
  const dateLocale = i18n.language === "fr" ? fr : enUS;

  return (
    <div className="flex items-center justify-between rounded-lg border p-4">
      <div className="flex items-center gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-full bg-muted">
          <Smartphone className="h-5 w-5 text-muted-foreground" />
        </div>
        <div>
          <div className="flex items-center gap-2">
            <span className="font-medium">{device.name}</span>
            {device.is_default && (
              <Badge variant="secondary" className="text-xs">
                <Star className="mr-1 h-3 w-3" />
                {t("auth.mfa.default")}
              </Badge>
            )}
          </div>
          <div className="text-muted-foreground text-xs">
            {t("auth.mfa.added_on", {
              date: format(new Date(device.created_at), "PPP", {
                locale: dateLocale,
              }),
            })}
          </div>
        </div>
      </div>
      <div className="flex items-center gap-2">
        {!device.is_default && device.is_verified && (
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setDeviceDefault(device.id)}
          >
            <Star className="mr-1 h-4 w-4" />
            {t("auth.mfa.set_default")}
          </Button>
        )}
        <Button
          variant="ghost"
          size="sm"
          className="text-destructive hover:text-destructive"
          onClick={() => openDeleteDialog(device.id)}
        >
          <Trash2 className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}
