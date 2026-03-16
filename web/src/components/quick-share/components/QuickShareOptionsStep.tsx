import { useTranslation } from "react-i18next";

import { Upload } from "lucide-react";
import { Controller } from "react-hook-form";
import type { Control } from "react-hook-form";
import type { FC } from "react";

import type { IQuickShareForm } from "@/components/quick-share/QuickShareDialog";
import type { ShareScope } from "@/types/share.ts";
import { Datepicker } from "@/components/common/components/Datepicker";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";

interface IQuickShareOptionsStepProps {
  scope: ShareScope;
  control: Control<IQuickShareForm>;
  allowUploads: boolean;
}

export const QuickShareOptionsStep: FC<IQuickShareOptionsStepProps> = ({
  scope,
  control,
  allowUploads,
}) => {
  const { t } = useTranslation();

  return (
    <div className="space-y-5">
      <div className="space-y-4">
        <div className="space-y-2">
          <Label>{t("quick_share.name")}</Label>
          <Controller
            name="name"
            control={control}
            rules={{ required: true, minLength: 1, maxLength: 255 }}
            render={({ field: { onChange, value } }) => (
              <Input type="text" value={value} onChange={onChange} />
            )}
          />
        </div>

        <div className="space-y-2">
          <Label>{t("quick_share.expiry_date")}</Label>
          <div>
            <Controller
              name="expiresAt"
              control={control}
              render={({ field: { onChange, value } }) => (
                <Datepicker value={value} onChange={onChange} />
              )}
            />
          </div>
        </div>

        <div className="space-y-2">
          <Label>{t("quick_share.max_views")}</Label>
          <p className="text-muted-foreground text-xs">
            {t("quick_share.max_views_description")}
          </p>
          <Controller
            name="maxViews"
            control={control}
            render={({ field: { onChange, value } }) => (
              <Input
                type="number"
                min={1}
                value={value}
                onChange={(e) => onChange(Number(e.target.value))}
              />
            )}
          />
        </div>
      </div>

      {scope !== "files" && (
        <>
          <Separator />

          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="space-y-1">
                <div className="flex items-center gap-2">
                  <Upload className="text-muted-foreground h-4 w-4" />
                  <Label>{t("quick_share.allow_uploads")}</Label>
                </div>
                <p className="text-muted-foreground text-xs">
                  {t("quick_share.allow_uploads_description")}
                </p>
              </div>
              <Controller
                name="allowUploads"
                control={control}
                render={({ field: { onChange, value } }) => (
                  <Switch checked={value} onCheckedChange={onChange} />
                )}
              />
            </div>

            {allowUploads && (
              <>
                <div className="space-y-2">
                  <Label>{t("quick_share.max_upload_size")}</Label>
                  <Controller
                    name="maxUploadSize"
                    control={control}
                    render={({ field: { onChange, value } }) => (
                      <Input
                        type="number"
                        min={1}
                        value={value}
                        onChange={(e) =>
                          onChange(
                            e.target.value === "" ? "" : Number(e.target.value),
                          )
                        }
                      />
                    )}
                  />
                </div>

                <div className="space-y-2">
                  <Label>{t("quick_share.max_uploads")}</Label>
                  <p className="text-muted-foreground text-xs">
                    {t("quick_share.max_uploads_description")}
                  </p>
                  <Controller
                    name="maxUploads"
                    control={control}
                    render={({ field: { onChange, value } }) => (
                      <Input
                        type="number"
                        min={1}
                        value={value}
                        onChange={(e) =>
                          onChange(
                            e.target.value === "" ? "" : Number(e.target.value),
                          )
                        }
                      />
                    )}
                  />
                </div>
              </>
            )}
          </div>
        </>
      )}
    </div>
  );
};
