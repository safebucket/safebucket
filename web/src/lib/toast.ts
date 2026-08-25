import { toast } from "sonner";
import i18n from "@/lib/i18n";

export function successToast(message: string) {
  toast.success(i18n.t("common.success"), { description: message });
}

export function resolveErrorMessage(error: Error): string {
  const translated = i18n.t(`errors.${error.message}`, { defaultValue: "" });
  return translated || i18n.t("errors.default");
}

export function errorToast(error: Error) {
  toast.error(i18n.t("common.error"), {
    description: resolveErrorMessage(error),
  });
}
