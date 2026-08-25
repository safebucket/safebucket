import { toast } from "sonner";
import i18n from "@/lib/i18n";

export function resolveErrorMessage(error: Error): string {
  const translated = i18n.t(`errors.${error.message}`, { defaultValue: "" });
  return translated || i18n.t("errors.default");
}

export function errorToast(error: Error | string, description?: string) {
  if (typeof error === "string") {
    if (description === undefined) {
      toast.error(error);
    } else {
      toast.error(error, { description });
    }
    return;
  }
  toast.error(i18n.t("common.error"), {
    description: resolveErrorMessage(error),
  });
}
