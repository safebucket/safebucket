import { useTranslation } from "react-i18next";

import { Calendar, Eye, EyeOff, Link, Lock, Upload } from "lucide-react";
import { useState } from "react";
import { Controller } from "react-hook-form";
import type { Control } from "react-hook-form";
import type { FC } from "react";

import type { IQuickShareForm } from "@/components/quick-share/QuickShareDialog";
import type { ShareScope } from "@/types/share.ts";
import { Datepicker } from "@/components/common/components/Datepicker";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";

interface IQuickShareOptionsStepProps {
  scope: ShareScope;
  control: Control<IQuickShareForm>;
  hasExpiry: boolean;
  hasCustomPath: boolean;
  limitViews: boolean;
  passwordProtected: boolean;
  allowUploads: boolean;
  customShareLinksEnabled: boolean;
}

export const QuickShareOptionsStep: FC<IQuickShareOptionsStepProps> = ({
  scope,
  control,
  hasExpiry,
  hasCustomPath,
  limitViews,
  passwordProtected,
  allowUploads,
  customShareLinksEnabled,
}) => {
  const { t } = useTranslation();
  const [showPassword, setShowPassword] = useState(false);

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
      </div>

      {customShareLinksEnabled && (
        <>
          <Separator />

          <div className="space-y-4">
            <div className="flex items-center justify-between gap-2">
              <div className="space-y-1">
                <div className="flex items-center gap-2">
                  <Link className="text-muted-foreground h-4 w-4" />
                  <Label htmlFor="custom-path-toggle">
                    {t("quick_share.custom_path")}
                  </Label>
                </div>
                <p className="text-muted-foreground text-xs">
                  {t("quick_share.custom_path_description")}
                </p>
              </div>
              <Controller
                name="hasCustomPath"
                control={control}
                render={({ field: { onChange, value } }) => (
                  <Switch
                    id="custom-path-toggle"
                    checked={value}
                    onCheckedChange={onChange}
                  />
                )}
              />
            </div>
            {hasCustomPath && (
              <Controller
                name="path"
                control={control}
                rules={{
                  validate: (value, values) =>
                    !values.hasCustomPath ||
                    value === "" ||
                    (value.length >= 3 &&
                      value.length <= 255 &&
                      /^[a-zA-Z0-9_-]+$/.test(value)),
                }}
                render={({ field: { onChange, value }, fieldState }) => (
                  <div className="space-y-1">
                    <div className="flex flex-nowrap items-center">
                      <span className="bg-muted text-muted-foreground flex h-9 shrink-0 items-center rounded-l-3xl px-3 text-sm whitespace-nowrap">
                        {t("quick_share.custom_path_prefix")}
                      </span>
                      <Input
                        type="text"
                        value={value}
                        onChange={onChange}
                        placeholder={t("quick_share.custom_path_placeholder")}
                        aria-label={t("quick_share.custom_path")}
                        aria-invalid={fieldState.invalid}
                        className="min-w-0 flex-1 rounded-l-none"
                      />
                    </div>
                    {fieldState.invalid && (
                      <p className="text-destructive text-xs">
                        {t("quick_share.custom_path_invalid")}
                      </p>
                    )}
                  </div>
                )}
              />
            )}
          </div>
        </>
      )}

      <Separator />

      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <Calendar className="text-muted-foreground h-4 w-4" />
              <Label>{t("quick_share.expiry_date")}</Label>
            </div>
            <p className="text-muted-foreground text-xs">
              {t("quick_share.expiry_date_description")}
            </p>
          </div>
          <Controller
            name="hasExpiry"
            control={control}
            render={({ field: { onChange, value } }) => (
              <Switch checked={value} onCheckedChange={onChange} />
            )}
          />
        </div>

        {hasExpiry && (
          <div className="space-y-2">
            <Label>{t("quick_share.expiry_date_label")}</Label>
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
        )}
      </div>

      <Separator />

      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <Eye className="text-muted-foreground h-4 w-4" />
              <Label>{t("quick_share.limit_views")}</Label>
            </div>
            <p className="text-muted-foreground text-xs">
              {t("quick_share.limit_views_description")}
            </p>
          </div>
          <Controller
            name="limitViews"
            control={control}
            render={({ field: { onChange, value } }) => (
              <Switch checked={value} onCheckedChange={onChange} />
            )}
          />
        </div>

        {limitViews && (
          <div className="space-y-2">
            <Label>{t("quick_share.max_views")}</Label>
            <Controller
              name="maxViews"
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
        )}
      </div>

      <Separator />

      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <Lock className="text-muted-foreground h-4 w-4" />
              <Label>{t("quick_share.password_protect")}</Label>
            </div>
            <p className="text-muted-foreground text-xs">
              {t("quick_share.password_protect_description")}
            </p>
          </div>
          <Controller
            name="passwordProtected"
            control={control}
            render={({ field: { onChange, value } }) => (
              <Switch checked={value} onCheckedChange={onChange} />
            )}
          />
        </div>

        {passwordProtected && (
          <div className="space-y-2">
            <Label>{t("quick_share.password")}</Label>
            <Controller
              name="password"
              control={control}
              rules={{ required: true, minLength: 8, maxLength: 72 }}
              render={({ field: { onChange, value } }) => (
                <div className="relative">
                  <Input
                    type={showPassword ? "text" : "password"}
                    value={value}
                    onChange={onChange}
                    minLength={8}
                    placeholder={t("quick_share.password_placeholder")}
                    className="pr-10"
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="absolute top-0 right-0 h-full px-3"
                    onClick={() => setShowPassword((prev) => !prev)}
                  >
                    {showPassword ? (
                      <EyeOff className="h-4 w-4" />
                    ) : (
                      <Eye className="h-4 w-4" />
                    )}
                  </Button>
                </div>
              )}
            />
          </div>
        )}
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
